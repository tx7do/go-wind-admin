package data

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entSql "entgo.io/ent/dialect/sql"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"github.com/stretchr/testify/require"

	entCrud "github.com/tx7do/go-crud/entgo"

	"go-wind-admin/app/admin/service/internal/data/ent"
)

// openKnowledgeEntClient 打开向量集成测试库并按生产同款路径就绪表结构：
// ent 迁移建 sys_ai_* 表 → EnsureVectorColumnDim 给 sys_ai_chunks 补建定维
// embedding 列。门控同 openVectorTestDB（缺环境一律 skip）。
func openKnowledgeEntClient(t *testing.T) (*ent.Client, *entCrud.EntClient[*ent.Client]) {
	t.Helper()
	dsn := os.Getenv("PGVECTOR_TEST_DSN")
	if dsn == "" {
		dsn = vectorTestDSN
	}
	drv, err := entSql.Open(dialect.Postgres, dsn)
	if err != nil {
		t.Skipf("open vector test postgres: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err = drv.DB().PingContext(ctx); err != nil {
		_ = drv.Close()
		t.Skipf("vector test postgres unavailable (set PGVECTOR_TEST_DSN to override): %v", err)
	}
	if _, err = drv.DB().ExecContext(ctx, `CREATE EXTENSION IF NOT EXISTS vector`); err != nil {
		_ = drv.Close()
		t.Skipf("pgvector extension unavailable: %v", err)
	}
	client := ent.NewClient(ent.Driver(drv))
	if err = client.Schema.Create(ctx); err != nil {
		_ = client.Close()
		_ = drv.Close()
		t.Skipf("migrate schema: %v", err)
	}
	if _, err = EnsureVectorColumnDim(ctx, drv.DB(), "sys_ai_chunks"); err != nil {
		_ = client.Close()
		_ = drv.Close()
		t.Skipf("ensure vector column: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
		_ = drv.Close()
	})
	return client, entCrud.NewEntClient(client, drv)
}

// negVecLiteralOfDim n 维全 -0.5 的 pgvector 字面量（与查询向量正反相对，
// 用于余弦排序断言）。
func negVecLiteralOfDim(n int) string {
	return "[" + strings.TrimSuffix(strings.Repeat("-0.5,", n), ",") + "]"
}

// TestAiKnowledgeRepoChunksRoundTrip 集成（PG 门控）：批量写入 → topK 检索的
// 命中排序与得分、租户过滤（租户互不可见、平台全见）、topK 截断、异维向量
// 整批 fail-fast 且事务不留半截（原有行数不变）。
func TestAiKnowledgeRepoChunksRoundTrip(t *testing.T) {
	_, entCrudClient := openKnowledgeEntClient(t)
	repo := &AiKnowledgeRepo{
		entClient: entCrudClient,
		log:       bLogger.NewHelper(bLogger.NopLogger()),
	}
	db := repo.entClient.DB()
	ctx := context.Background()

	// 造数：一库一文档（列全可空，走原始 SQL 最小造行），切片经 repo 写入。
	var baseID, docID int64
	err := db.QueryRowContext(ctx, "INSERT INTO sys_ai_knowledge_bases DEFAULT VALUES RETURNING id").Scan(&baseID)
	require.NoError(t, err, "seed base row")
	err = db.QueryRowContext(ctx, "INSERT INTO sys_ai_docs (base_id) VALUES ($1) RETURNING id", baseID).Scan(&docID)
	require.NoError(t, err, "seed doc row")
	t.Cleanup(func() {
		// LIFO：先删子表（原始 SQL 写入的切片），再删 doc/base（FK 顺序）。
		_, _ = db.ExecContext(context.Background(), "DELETE FROM sys_ai_chunks WHERE doc_id = $1", docID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM sys_ai_docs WHERE id = $1", docID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM sys_ai_knowledge_bases WHERE id = $1", baseID)
	})

	pos := vecLiteralOfDim(1536)
	neg := negVecLiteralOfDim(1536)
	require.NoError(t, repo.InsertChunks(ctx, 7, docID,
		[]string{"c0", "c1", "c2"}, []string{neg, pos, neg}),
		"batch insert of 3 chunks should succeed")

	// topK 检索：与查询同向的 c1 必居首、余弦得分 1、chunk_index 对位；其余按序返回。
	hits, err := repo.SearchChunks(ctx, 7, baseID, pos, 3)
	require.NoError(t, err)
	require.Len(t, hits, 3)
	require.Equal(t, "c1", hits[0].Content, "identical vector must rank first")
	require.InDelta(t, 1.0, hits[0].Score, 1e-6)
	require.Equal(t, uint32(1), hits[0].ChunkIndex)

	// topK 截断
	hits1, err := repo.SearchChunks(ctx, 7, baseID, pos, 1)
	require.NoError(t, err)
	require.Len(t, hits1, 1)

	// 租户隔离：tenant 8 检索不到 tenant 7 的切片；平台（tenantId=0）全见。
	hits8, err := repo.SearchChunks(ctx, 8, baseID, pos, 3)
	require.NoError(t, err)
	require.Empty(t, hits8, "tenant 8 must not see tenant 7 chunks")
	hits0, err := repo.SearchChunks(ctx, 0, baseID, pos, 3)
	require.NoError(t, err)
	require.Len(t, hits0, 3, "platform viewer must see all chunks")

	// 异维 fail-fast + 事务原子性：混入 384 维向量整批失败，新行零落库、旧行原样。
	err = repo.InsertChunks(ctx, 7, docID, []string{"x", "y"}, []string{pos, vecLiteralOfDim(384)})
	require.Error(t, err, "mismatched-dim batch must fail")
	var total, xy int
	require.NoError(t, db.QueryRowContext(ctx,
		"SELECT count(*) FROM sys_ai_chunks WHERE doc_id = $1", docID).Scan(&total))
	require.Equal(t, 3, total, "original rows must be untouched after failed batch")
	require.NoError(t, db.QueryRowContext(ctx,
		"SELECT count(*) FROM sys_ai_chunks WHERE doc_id = $1 AND content IN ('x','y')", docID).Scan(&xy))
	require.Zero(t, xy, "failed batch must leave zero rows")
}
