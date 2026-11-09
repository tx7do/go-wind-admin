package data

import (
	"context"
	"fmt"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"github.com/tx7do/go-crud/viewer"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/aidoc"
	"go-wind-admin/app/admin/service/internal/data/ent/aiknowledgebase"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	"google.golang.org/protobuf/types/known/timestamppb"

	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"
)

type AiKnowledgeRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *bLogger.Helper

	mapper *mapper.CopierMapper[aiV1.AiKnowledgeBase, ent.AiKnowledgeBase]

	repository *entCrud.Repository[
		ent.AiKnowledgeBaseQuery, ent.AiKnowledgeBaseSelect,
		ent.AiKnowledgeBaseCreate, ent.AiKnowledgeBaseCreateBulk,
		ent.AiKnowledgeBaseUpdate, ent.AiKnowledgeBaseUpdateOne,
		ent.AiKnowledgeBaseDelete,
		predicate.AiKnowledgeBase,
		aiV1.AiKnowledgeBase, ent.AiKnowledgeBase,
	]
}

func NewAiKnowledgeRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *AiKnowledgeRepo {
	repo := &AiKnowledgeRepo{
		log:       ctx.NewLoggerHelper("ai_knowledge/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[aiV1.AiKnowledgeBase, ent.AiKnowledgeBase](),
	}

	repo.init()

	return repo
}

func (r *AiKnowledgeRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.AiKnowledgeBaseQuery, ent.AiKnowledgeBaseSelect,
		ent.AiKnowledgeBaseCreate, ent.AiKnowledgeBaseCreateBulk,
		ent.AiKnowledgeBaseUpdate, ent.AiKnowledgeBaseUpdateOne,
		ent.AiKnowledgeBaseDelete,
		predicate.AiKnowledgeBase,
		aiV1.AiKnowledgeBase, ent.AiKnowledgeBase,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

// ── 知识库 CRUD（泛型仓库标准形态） ─────────────────────────────────

func (r *AiKnowledgeRepo) Count(ctx context.Context, req *paginationV1.PagingRequest) (uint64, error) {
	builder := r.entClient.Client().AiKnowledgeBase.Query()

	whereSelectors, _, _ := r.repository.BuildListSelectorWithPaging(builder, req)
	if len(whereSelectors) != 0 {
		builder.Modify(whereSelectors...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query ai knowledge base count failed: %s", err.Error())
		return 0, aiV1.ErrorInternalServerError("query ai knowledge base count failed")
	}
	return uint64(count), nil
}

func (r *AiKnowledgeRepo) List(ctx context.Context, req *paginationV1.PagingRequest) ([]*aiV1.AiKnowledgeBase, uint64, error) {
	if req == nil {
		return nil, 0, aiV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().AiKnowledgeBase.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, 0, err
	}

	return ret.Items, ret.Total, nil
}

func (r *AiKnowledgeRepo) Get(ctx context.Context, req *aiV1.GetAiKnowledgeBaseRequest) (*aiV1.AiKnowledgeBase, error) {
	if req == nil {
		return nil, aiV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().AiKnowledgeBase.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *aiV1.GetAiKnowledgeBaseRequest_Id:
		whereCond = append(whereCond, aiknowledgebase.IDEQ(req.GetId()))
	}

	return r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
}

func (r *AiKnowledgeRepo) Create(ctx context.Context, data *aiV1.AiKnowledgeBase) (*ent.AiKnowledgeBase, error) {
	if data == nil {
		return nil, aiV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().AiKnowledgeBase.Create().
		SetNillableName(data.Name).
		SetNillableDescription(data.Description).
		SetNillableProviderID(data.ProviderId).
		SetNillableEmbeddingModel(data.EmbeddingModel).
		SetNillableUserID(data.UserId).
		SetNillableTenantID(data.TenantId).
		SetNillableCreatedBy(data.CreatedBy).
		SetCreatedAt(time.Now())

	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf(ctx, "insert ai knowledge base failed: %s", err.Error())
		return nil, aiV1.ErrorInternalServerError("insert ai knowledge base failed")
	}
	return entity, nil
}

func (r *AiKnowledgeRepo) Update(ctx context.Context, req *aiV1.UpdateAiKnowledgeBaseRequest) error {
	if req == nil || req.Data == nil {
		return aiV1.ErrorBadRequest("invalid parameter")
	}
	if req.GetId() == 0 {
		return aiV1.ErrorBadRequest("id is required")
	}

	builder := r.entClient.Client().AiKnowledgeBase.Update()

	if err := r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *aiV1.AiKnowledgeBase) {
			builder.
				SetNillableName(req.Data.Name).
				SetNillableDescription(req.Data.Description).
				SetNillableProviderID(req.Data.ProviderId).
				SetNillableEmbeddingModel(req.Data.EmbeddingModel).
				SetNillableUpdatedBy(req.Data.UpdatedBy).
				SetUpdatedAt(time.Now())
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(aiknowledgebase.FieldID, req.GetId()))
		},
	); err != nil {
		r.log.Errorf(ctx, "update ai knowledge base failed: %s", err.Error())
		return err
	}
	return nil
}

// Delete 删除知识库；docs 边 Cascade 级联删文档行，文档的 chunks 由 doc 侧 Cascade 删除。
func (r *AiKnowledgeRepo) Delete(ctx context.Context, id uint64) error {
	if _, err := r.entClient.Client().AiKnowledgeBase.Delete().
		Where(aiknowledgebase.IDEQ(id)).
		Exec(ctx); err != nil {
		r.log.Errorf(ctx, "delete ai knowledge base failed: %s", err.Error())
		return aiV1.ErrorInternalServerError("delete ai knowledge base failed")
	}
	return nil
}

func (r *AiKnowledgeRepo) GetEntityByID(ctx context.Context, id uint64) (*ent.AiKnowledgeBase, error) {
	entity, err := r.entClient.Client().AiKnowledgeBase.Query().
		Where(aiknowledgebase.IDEQ(id)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, aiV1.ErrorNotFound("ai knowledge base not found")
		}
		r.log.Errorf(ctx, "query ai knowledge base failed: %s", err.Error())
		return nil, aiV1.ErrorInternalServerError("query ai knowledge base failed")
	}
	return entity, nil
}

func (r *AiKnowledgeRepo) ToDTO(entity *ent.AiKnowledgeBase) *aiV1.AiKnowledgeBase {
	if entity == nil {
		return nil
	}
	return r.mapper.ToDTO(entity)
}

// ── 文档（手写 ent 调用：固定 base_id 查询，不走通用 filter） ────────

func (r *AiKnowledgeRepo) CreateDoc(ctx context.Context, data *aiV1.AiDoc) (*ent.AiDoc, error) {
	if data == nil {
		return nil, aiV1.ErrorBadRequest("invalid parameter")
	}
	entity, err := r.entClient.Client().AiDoc.Create().
		SetNillableBaseID(data.BaseId).
		SetNillableName(data.Name).
		SetNillableChunkCount(data.ChunkCount).
		SetNillableStatus(data.Status).
		SetNillableErrorMessage(data.ErrorMessage).
		SetNillableUserID(data.UserId).
		SetNillableTenantID(data.TenantId).
		SetCreatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		r.log.Errorf(ctx, "insert ai doc failed: %s", err.Error())
		return nil, aiV1.ErrorInternalServerError("insert ai doc failed")
	}
	return entity, nil
}

func (r *AiKnowledgeRepo) UpdateDocStatus(ctx context.Context, id uint64, status string, chunkCount uint32, errMsg string) error {
	builder := r.entClient.Client().AiDoc.UpdateOneID(id).
		SetNillableStatus(&status).
		SetChunkCount(chunkCount)
	if errMsg != "" {
		builder.SetErrorMessage(errMsg)
	}
	if err := builder.Exec(ctx); err != nil {
		r.log.Errorf(ctx, "update ai doc status failed: %s", err.Error())
		return aiV1.ErrorInternalServerError("update ai doc failed")
	}
	return nil
}

func (r *AiKnowledgeRepo) ListDocs(ctx context.Context, baseId uint64) ([]*ent.AiDoc, error) {
	return r.entClient.Client().AiDoc.Query().
		Where(aidoc.BaseIDEQ(baseId)).
		Order(ent.Asc(aidoc.FieldID)).
		All(ctx)
}

func (r *AiKnowledgeRepo) GetDocByID(ctx context.Context, id uint64) (*ent.AiDoc, error) {
	entity, err := r.entClient.Client().AiDoc.Query().
		Where(aidoc.IDEQ(id)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, aiV1.ErrorNotFound("ai doc not found")
		}
		r.log.Errorf(ctx, "query ai doc failed: %s", err.Error())
		return nil, aiV1.ErrorInternalServerError("query ai doc failed")
	}
	return entity, nil
}

func (r *AiKnowledgeRepo) DeleteDoc(ctx context.Context, id uint64) error {
	if _, err := r.entClient.Client().AiDoc.Delete().
		Where(aidoc.IDEQ(id)).
		Exec(ctx); err != nil {
		r.log.Errorf(ctx, "delete ai doc failed: %s", err.Error())
		return aiV1.ErrorInternalServerError("delete ai doc failed")
	}
	return nil
}

func (r *AiKnowledgeRepo) ToDocDTO(entity *ent.AiDoc) *aiV1.AiDoc {
	if entity == nil {
		return nil
	}
	return &aiV1.AiDoc{
		Id:           &entity.ID,
		BaseId:       entity.BaseID,
		Name:         entity.Name,
		ChunkCount:   entity.ChunkCount,
		Status:       entity.Status,
		ErrorMessage: entity.ErrorMessage,
		UserId:       entity.UserID,
		CreatedAt:    timestamppbPtr(entity.CreatedAt),
	}
}

// ── 切片（原生 SQL：pgvector 列不在 ent schema 内） ─────────────────

// InsertChunks 批量写入切片与向量：整文档单事务、分批多行 INSERT——
// 任一片失败全文档回滚，不再留半截切片（文档行随后由调用方标 FAILED）。
// vecLiteral 为 pgvector 字面量 '[0.1,0.2,...]'；列定维 1536 由
// EnsureVectorColumnDim 保证，异维向量在此 fail-fast。
func (r *AiKnowledgeRepo) InsertChunks(ctx context.Context, tenantId uint64, docId uint64, chunks []string, vectors []string) error {
	if len(chunks) != len(vectors) {
		return aiV1.ErrorInternalServerError("chunk/vector length mismatch")
	}
	if len(chunks) == 0 {
		return nil
	}
	db := r.entClient.DB()
	tx, txErr := db.BeginTx(ctx, nil)
	if txErr != nil {
		r.log.Errorf(ctx, "begin tx for ai chunks failed: %s", txErr.Error())
		return aiV1.ErrorInternalServerError("insert ai chunk failed")
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	const rowsPerStmt = 100
	for start := 0; start < len(chunks); start += rowsPerStmt {
		end := start + rowsPerStmt
		if end > len(chunks) {
			end = len(chunks)
		}
		var sb strings.Builder
		var args []any
		sb.WriteString("INSERT INTO sys_ai_chunks (created_at, tenant_id, doc_id, content, chunk_index, embedding) VALUES ")
		for j := start; j < end; j++ {
			if j > start {
				sb.WriteByte(',')
			}
			base := (j - start) * 5
			_, _ = fmt.Fprintf(&sb, "(NOW(),$%d,$%d,$%d,$%d,$%d::vector)", base+1, base+2, base+3, base+4, base+5)
			args = append(args, tenantId, docId, chunks[j], uint64(j), vectors[j])
		}
		if _, execErr := tx.ExecContext(ctx, sb.String(), args...); execErr != nil {
			r.log.Errorf(ctx, "insert ai chunks batch failed: %s", execErr.Error())
			return aiV1.ErrorInternalServerError("insert ai chunk failed")
		}
	}

	if cErr := tx.Commit(); cErr != nil {
		r.log.Errorf(ctx, "commit ai chunks failed: %s", cErr.Error())
		return aiV1.ErrorInternalServerError("insert ai chunk failed")
	}
	committed = true
	return nil
}

// ChunkHit 检索命中。
type ChunkHit struct {
	Content    string  `json:"content"`
	ChunkIndex uint32  `json:"chunk_index"`
	DocID      uint64  `json:"doc_id"`
	Score      float64 `json:"score"`
}

// SearchChunks 余弦相似度检索：限定租户 + 知识库，取 topK。
func (r *AiKnowledgeRepo) SearchChunks(ctx context.Context, tenantId, baseId uint64, vecLiteral string, topK int) ([]*ChunkHit, error) {
	if topK <= 0 {
		topK = 3
	}
	// P3 加固：上限钳制（pgvector 全表距离排序 + 大结果集物化的资源面）。
	if topK > 50 {
		topK = 50
	}
	db := r.entClient.DB()
	rows, err := db.QueryContext(ctx,
		`SELECT c.content, c.chunk_index, c.doc_id, 1 - (c.embedding <=> $1::vector) AS score
		 FROM sys_ai_chunks c
		 JOIN sys_ai_docs d ON d.id = c.doc_id
		 WHERE d.base_id = $2 AND ($3 = 0 OR c.tenant_id = $3)
		 ORDER BY c.embedding <=> $1::vector
		 LIMIT $4`,
		vecLiteral, baseId, tenantId, topK,
	)
	if err != nil {
		r.log.Errorf(ctx, "search ai chunks failed: %s", err.Error())
		return nil, aiV1.ErrorInternalServerError("search ai chunks failed")
	}
	defer func() { _ = rows.Close() }()

	hits := make([]*ChunkHit, 0, topK)
	for rows.Next() {
		var hit ChunkHit
		if err := rows.Scan(&hit.Content, &hit.ChunkIndex, &hit.DocID, &hit.Score); err != nil {
			r.log.Errorf(ctx, "scan ai chunk hit failed: %s", err.Error())
			return nil, aiV1.ErrorInternalServerError("scan ai chunk failed")
		}
		hits = append(hits, &hit)
	}
	return hits, rows.Err()
}

// ── 向量列启动迁移（幂等） ─────────────────────────────────────────

// MigrateVectorColumn 启动期补建 pgvector 扩展、定维 embedding 列，并按表规模
// 自动落 HNSW 门槛（见 data.EnsureHnswIndexIfLarge）。
// 扩展需要超级用户；失败只记日志不阻断服务（RAG 功能降级，其余功能不受影响）。
// 历史非定维/异维度列会被整列重建（向量清空），此时须重跑 ai_doc_reindex 任务重索引。
//
// HNSW 门槛只接 sys_ai_chunks：菜单语义搜索表行数随菜单数有界（几十行），
// 永不越线且其查询本就无过滤形态，不接。
func (r *AiKnowledgeRepo) MigrateVectorColumn(ctx context.Context) error {
	db := r.entClient.DB()
	rebuilt, err := EnsureVectorColumnDim(ctx, db, "sys_ai_chunks")
	if err != nil {
		r.log.Errorf(ctx, "ai chunks vector column migration failed (rag disabled): %s", err.Error())
		return err
	}
	if rebuilt {
		r.log.Warnf(ctx,
			"sys_ai_chunks.embedding 与当前 embedding 模型维度不符，已整列重建、旧向量清空：请在任务管理创建 ai_doc_reindex 任务全量重索引")
	}
	rows, cntErr := CountTableRows(ctx, db, "sys_ai_chunks")
	if cntErr != nil {
		r.log.Errorf(ctx, "ai chunks row count for hnsw gate failed: %s", cntErr.Error())
		r.log.Infof(ctx, "pgvector ready: sys_ai_chunks.embedding vector(%d)", AiEmbeddingDimensions)
		return nil
	}
	created, idxErr := EnsureHnswIndexIfLarge(ctx, db, "sys_ai_chunks", rows)
	if idxErr != nil {
		r.log.Errorf(ctx, "hnsw gate for sys_ai_chunks failed: %s", idxErr.Error())
		r.log.Infof(ctx, "pgvector ready: sys_ai_chunks.embedding vector(%d)", AiEmbeddingDimensions)
		return nil
	}
	if created {
		if gErr := SetHnswIterativeScanDefault(ctx, db); gErr != nil {
			r.log.Errorf(ctx, "set hnsw.iterative_scan default failed: %s", gErr.Error())
		}
		r.log.Warnf(ctx,
			"sys_ai_chunks 行数(%d)已越过 HNSW 门槛(%d)，已自动建 HNSW cosine 索引并将本库 hnsw.iterative_scan 默认设为 relaxed_order。"+
				"此后 ANN 检索为近似召回（top-K 可能漏配）、距离序可能不严格、写入承担索引维护代价（实测 ~3×/行）；"+
				"建议尽快评估把向量负载迁出主库（docs/ai_module.md 部署要求）",
			rows, HnswIndexRowThreshold)
	}
	r.log.Infof(ctx, "pgvector ready: sys_ai_chunks.embedding vector(%d)", AiEmbeddingDimensions)
	return nil
}

// timestamppbPtr time → timestamppb（nil 安全）。
func timestamppbPtr(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

// FillDocCounts 一次聚合填充各知识库的文档数（List 读视图）。
func (r *AiKnowledgeRepo) FillDocCounts(ctx context.Context, bases []*aiV1.AiKnowledgeBase) {
	if len(bases) == 0 {
		return
	}
	ids := make([]uint64, 0, len(bases))
	for _, b := range bases {
		if b.Id != nil {
			ids = append(ids, *b.Id)
		}
	}
	if len(ids) == 0 {
		return
	}

	var rows []struct {
		BaseID uint64 `sql:"base_id"`
		Count  uint64 `sql:"count"`
	}
	sysCtx := viewer.WithSystemContext(ctx)
	if err := r.entClient.Client().AiDoc.Query().
		Where(aidoc.BaseIDIn(ids...)).
		GroupBy(aidoc.FieldBaseID).
		Aggregate(ent.As(ent.Count(), "count")).
		Scan(sysCtx, &rows); err != nil {
		r.log.Errorf(ctx, "count ai docs failed: %s", err.Error())
		return
	}
	counts := make(map[uint64]uint64, len(rows))
	for _, row := range rows {
		counts[row.BaseID] = uint64(row.Count)
	}
	for _, b := range bases {
		if b.Id != nil {
			c := uint32(counts[*b.Id])
			b.DocCount = &c
		}
	}
}

// ── 重索引（批量重算 embedding） ─────────────────────────────────────

// ChunkRef 重索引用的切片引用。
type ChunkRef struct {
	ID      uint64
	Content string
}

// ListBases 全量知识库（系统查看器；重索引任务无请求上下文）。
func (r *AiKnowledgeRepo) ListBases(ctx context.Context) ([]*ent.AiKnowledgeBase, error) {
	sysCtx := viewer.WithSystemContext(ctx)
	return r.entClient.Client().AiKnowledgeBase.Query().All(sysCtx)
}

// ListChunkRefsByBase 取知识库下全部切片的 id+content（原生 SQL：chunks 表不在泛型仓库域）。
func (r *AiKnowledgeRepo) ListChunkRefsByBase(ctx context.Context, baseId uint64) ([]ChunkRef, error) {
	rows, err := r.entClient.DB().QueryContext(ctx,
		`SELECT c.id, c.content FROM sys_ai_chunks c
		 JOIN sys_ai_docs d ON d.id = c.doc_id
		 WHERE d.base_id = $1
		 ORDER BY c.id`,
		baseId,
	)
	if err != nil {
		r.log.Errorf(ctx, "list chunks for reindex failed: %s", err.Error())
		return nil, aiV1.ErrorInternalServerError("list chunks failed")
	}
	defer func() { _ = rows.Close() }()

	refs := make([]ChunkRef, 0, 64)
	for rows.Next() {
		var ref ChunkRef
		var content []byte
		if err := rows.Scan(&ref.ID, &content); err != nil {
			r.log.Errorf(ctx, "scan chunk ref failed: %s", err.Error())
			return nil, aiV1.ErrorInternalServerError("scan chunks failed")
		}
		ref.Content = string(content)
		refs = append(refs, ref)
	}
	return refs, rows.Err()
}

// UpdateChunkEmbedding 就地更新切片向量。
func (r *AiKnowledgeRepo) UpdateChunkEmbedding(ctx context.Context, chunkId uint64, vecLiteral string) error {
	_, err := r.entClient.DB().ExecContext(ctx,
		`UPDATE sys_ai_chunks SET embedding = $1::vector WHERE id = $2`,
		vecLiteral, chunkId,
	)
	if err != nil {
		r.log.Errorf(ctx, "update chunk embedding failed: chunk=%d: %s", chunkId, err.Error())
		return aiV1.ErrorInternalServerError("update chunk embedding failed")
	}
	return nil
}
