package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entSql "entgo.io/ent/dialect/sql"
	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-crud/viewer"
	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"
	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/aiprovider"
	"go-wind-admin/pkg/middleware/auth"
)

// ── 门控客户端（与 data 包向量集成测试同一门控：PGVECTOR_TEST_DSN 覆盖，
// 默认本地专用测试库；连不上 / 无 pgvector 一律 skip 保持缺环境套件绿） ──

func openContentEntClient(t *testing.T) *entCrud.EntClient[*ent.Client] {
	t.Helper()
	dsn := os.Getenv("PGVECTOR_TEST_DSN")
	if dsn == "" {
		dsn = "host=127.0.0.1 port=5432 user=postgres password=*Abcd123456 dbname=gwa_guard_test sslmode=disable"
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
	t.Cleanup(func() {
		_ = client.Close()
		_ = drv.Close()
	})
	return entCrud.NewEntClient(client, drv)
}

// ── mock 嵌入服务 ─────────────────────────────────────────────────

// contentTestUserID 测试操作者（用量流水归属断言与清理键）。
const contentTestUserID = int64(81)

// signatureOf 文本签名（字节和取模）。mock 向量为确定性正交编码：
// vec[sig(text)] = 1、其余 0（L2 范数 1）——同签名余弦相似度 1，
// 异签名正交（相似度 0），排序断言因此确定。
func signatureOf(text string) int {
	sum := 0
	for _, b := range []byte(text) {
		sum += int(b)
	}
	return sum % data.AiEmbeddingDimensions
}

// startMockEmbedServer 起本地 OpenAI 兼容 /v1/embeddings 服务：
// 按输入文本签名出 1536 维确定性向量，usage 恒 1 token（计量断言用）。
func startMockEmbedServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/embeddings") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body struct {
			Input []string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		dataArr := make([]map[string]any, 0, len(body.Input))
		for i, text := range body.Input {
			vec := make([]float32, data.AiEmbeddingDimensions)
			vec[signatureOf(text)] = 1
			dataArr = append(dataArr, map[string]any{
				"object": "embedding", "index": i, "embedding": vec,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list", "data": dataArr, "model": "mock",
			"usage": map[string]any{"prompt_tokens": 1, "total_tokens": 1},
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestAiContentServiceMenuSearchLoop 集成（PG 门控）全回路：
// 重建索引（菜单收集→mock 向量化→事务批量落库→用量计量）与
// 语义搜索（查询向量化→余弦检索→用量计量）端到端。
func TestAiContentServiceMenuSearchLoop(t *testing.T) {
	entCrudClient := openContentEntClient(t)
	db := entCrudClient.DB()
	client := entCrudClient.Client()

	sysCtx := viewer.WithSystemContext(context.Background())
	opCtx := auth.NewContext(viewer.WithSystemContext(context.Background()),
		&authenticationV1.UserTokenPayload{UserId: contentTestUserID})

	titleA := "甲组仪表盘"
	titleB := "乙级审计台账"
	require.NotEqual(t, signatureOf(titleA), signatureOf(titleB),
		"两标题签名须互异，排序断言才确定")

	srv := startMockEmbedServer(t)

	// 造数：启用默认 provider 指向 mock；两条根菜单（标题在 meta）。
	provider := client.AiProvider.Create().
		SetIsEnabled(true).
		SetIsDefault(true).
		SetModelType(aiprovider.ModelTypeCLOUD).
		SetBaseURL(srv.URL).
		SetAPIKey("mock-key").
		SaveX(sysCtx)
	menuA := client.Menu.Create().
		SetName("m-a").SetPath("/test/a").
		SetMeta(&permissionV1.MenuMeta{Title: trans.Ptr(titleA)}).
		SaveX(sysCtx)
	menuB := client.Menu.Create().
		SetName("m-b").SetPath("/test/b").
		SetMeta(&permissionV1.MenuMeta{Title: trans.Ptr(titleB)}).
		SaveX(sysCtx)
	t.Cleanup(func() {
		// 硬删（绕开 ent 软删）：菜单 (parent_id,name) 唯一索引会拦住软删后的同名重种。
		_, _ = db.ExecContext(context.Background(), "DELETE FROM sys_menus WHERE id = $1", menuA.ID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM sys_menus WHERE id = $1", menuB.ID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM sys_ai_providers WHERE id = $1", provider.ID)
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM sys_ai_search_index WHERE route IN ('/test/a','/test/b')")
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM sys_ai_usage_logs WHERE user_id = $1", contentTestUserID)
	})

	svc := &AiContentService{
		log:          bLogger.NewHelper(bLogger.NopLogger()),
		providerRepo: data.NewAiProviderRepoForTest(entCrudClient),
		usageRepo:    data.NewAiUsageLogRepoForTest(entCrudClient),
		menuRepo:     data.NewMenuRepoForTest(entCrudClient),
		entClient:    entCrudClient,
	}

	usageCount := func() int {
		var n int
		require.NoError(t, db.QueryRowContext(context.Background(),
			"SELECT count(*) FROM sys_ai_usage_logs WHERE user_id = $1", contentTestUserID).Scan(&n))
		return n
	}

	// ── 全回路①：重建索引 ──────────────────────────────────────
	cnt, err := svc.BuildMenuSearchIndex(opCtx)
	require.NoError(t, err)
	require.Equal(t, int64(2), cnt, "两条可索引菜单应全部入库")

	// 索引表断言：恰好两行、item_id 为真实菜单 id、title/route 对位、向量定维。
	rows, err := db.QueryContext(context.Background(),
		"SELECT item_id, title, route, vector_dims(embedding) FROM sys_ai_search_index WHERE item_type = 'menu'")
	require.NoError(t, err)
	type idxRow struct {
		itemID int64
		title  string
		dim    int
	}
	got := map[string]idxRow{}
	for rows.Next() {
		var r idxRow
		var route string
		require.NoError(t, rows.Scan(&r.itemID, &r.title, &route, &r.dim))
		got[route] = r
	}
	require.NoError(t, rows.Err())
	_ = rows.Close()
	require.Len(t, got, 2)
	want := map[string]struct {
		id    int64
		title string
	}{
		"/test/a": {menuA.ID, titleA},
		"/test/b": {menuB.ID, titleB},
	}
	for route, w := range want {
		g := got[route]
		require.Equal(t, w.id, g.itemID, "item_id 须为真实菜单 id：%s", route)
		require.Equal(t, w.title, g.title, "title 须与菜单标题对位：%s", route)
		require.Equal(t, data.AiEmbeddingDimensions, g.dim, "向量须为定维：%s", route)
	}

	// 重建为单批（2 输入 < 32）→ 用量流水恰好 1 行，归属测试操作者。
	require.Equal(t, 1, usageCount(), "重建的嵌入批调用应记 1 行用量")

	// ── 全回路②：语义搜索 ──────────────────────────────────────
	resp, err := svc.SemanticSearch(opCtx, &aiV1.SemanticSearchRequest{
		Query: titleA, Limit: trans.Ptr(uint32(2)),
	})
	require.NoError(t, err)
	require.Len(t, resp.Items, 2, "索引共两行应全部返回")
	// 查询与 menuA 同签名 → 余弦相似度 1 居首；menuB 正交 → 相似度 0 殿后。
	require.Equal(t, "/test/a", resp.Items[0].Route)
	require.InDelta(t, float32(1.0), resp.Items[0].Score, 1e-5)
	require.Equal(t, "/test/b", resp.Items[1].Route)
	require.InDelta(t, float32(0.0), resp.Items[1].Score, 1e-5)

	// 查询向量化一次 → 用量流水递增到 2。
	require.Equal(t, 2, usageCount(), "查询向量化应再记 1 行用量")
}
