package data

import (
	"context"
	"time"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"github.com/tx7do/go-crud/viewer"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/aiusagelog"
	"go-wind-admin/app/admin/service/internal/data/ent/planquota"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"
	"go-wind-admin/app/admin/service/internal/data/ent/tenant"

	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"
)

type AiUsageLogRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *bLogger.Helper

	mapper *mapper.CopierMapper[aiV1.AiUsageLog, ent.AiUsageLog]

	repository *entCrud.Repository[
		ent.AiUsageLogQuery, ent.AiUsageLogSelect,
		ent.AiUsageLogCreate, ent.AiUsageLogCreateBulk,
		ent.AiUsageLogUpdate, ent.AiUsageLogUpdateOne,
		ent.AiUsageLogDelete,
		predicate.AiUsageLog,
		aiV1.AiUsageLog, ent.AiUsageLog,
	]
}

func NewAiUsageLogRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *AiUsageLogRepo {
	repo := &AiUsageLogRepo{
		log:       ctx.NewLoggerHelper("ai_usage_log/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[aiV1.AiUsageLog, ent.AiUsageLog](),
	}

	repo.init()

	return repo
}

func (r *AiUsageLogRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.AiUsageLogQuery, ent.AiUsageLogSelect,
		ent.AiUsageLogCreate, ent.AiUsageLogCreateBulk,
		ent.AiUsageLogUpdate, ent.AiUsageLogUpdateOne,
		ent.AiUsageLogDelete,
		predicate.AiUsageLog,
		aiV1.AiUsageLog, ent.AiUsageLog,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

func (r *AiUsageLogRepo) Count(ctx context.Context, req *paginationV1.PagingRequest) (*aiV1.CountAiUsageLogResponse, error) {
	builder := r.entClient.Client().AiUsageLog.Query()

	whereSelectors, _, _ := r.repository.BuildListSelectorWithPaging(builder, req)
	if len(whereSelectors) != 0 {
		builder.Modify(whereSelectors...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query ai usage log count failed: %s", err.Error())
		return nil, aiV1.ErrorInternalServerError("query ai usage log count failed")
	}

	return &aiV1.CountAiUsageLogResponse{Count: int64(count)}, nil
}

func (r *AiUsageLogRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*aiV1.ListAiUsageLogResponse, error) {
	if req == nil {
		return nil, aiV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().AiUsageLog.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &aiV1.ListAiUsageLogResponse{Total: 0, Items: nil}, nil
	}

	return &aiV1.ListAiUsageLogResponse{Total: ret.Total, Items: ret.Items}, nil
}

// Create 记一笔成功调用的用量流水（只增不改）。
func (r *AiUsageLogRepo) Create(ctx context.Context, data *aiV1.AiUsageLog) error {
	if data == nil {
		return aiV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().AiUsageLog.Create().
		SetNillableProviderID(data.ProviderId).
		SetNillableConversationID(data.ConversationId).
		SetNillableUserID(data.UserId).
		SetNillableTenantID(data.TenantId).
		SetNillableModelName(data.ModelName).
		SetNillablePromptTokens(data.PromptTokens).
		SetNillableCompletionTokens(data.CompletionTokens).
		SetNillableTotalTokens(data.TotalTokens).
		SetNillableDurationMs(data.DurationMs).
		SetCreatedAt(time.Now())

	if err := builder.Exec(ctx); err != nil {
		r.log.Errorf(ctx, "insert ai usage log failed: %s", err.Error())
		return aiV1.ErrorInternalServerError("insert ai usage log failed")
	}

	return nil
}

// SumTokensByTenantSince 统计租户自某时刻起消耗的总 token 数（配额检查用）。
func (r *AiUsageLogRepo) SumTokensByTenantSince(ctx context.Context, tenantId int64, since time.Time) (int64, error) {
	sum := int64(0)
	var rows []struct {
		Total int64 `sql:"total"`
	}
	err := r.entClient.Client().AiUsageLog.Query().
		Where(
			aiusagelog.TenantIDEQ(tenantId),
			aiusagelog.CreatedAtGTE(since),
		).
		Aggregate(ent.As(ent.Sum(aiusagelog.FieldTotalTokens), "total")).
		Scan(ctx, &rows)
	if err != nil {
		r.log.Errorf(ctx, "sum ai usage tokens failed: %s", err.Error())
		return 0, aiV1.ErrorInternalServerError("sum ai usage tokens failed")
	}
	if len(rows) > 0 {
		sum = rows[0].Total
	}
	return sum, nil
}

// FetchTenantTokenQuotaLimit 读租户套餐上的 AI_TOKENS 配额上限。
// 返回 (上限, 是否配置, 错误)：未配置=false 时调用方视为不限量（配额缺省语义与
// 其他维度一致——没配的维度不设限）。
func (r *AiUsageLogRepo) FetchTenantTokenQuotaLimit(ctx context.Context, tenantId int64) (int64, bool, error) {
	if tenantId == 0 {
		return 0, false, nil
	}

	// 系统查看器绕租户谓词读套餐链（与 tenant_usage_repo.GetUsage 同型）。
	sysCtx := viewer.WithSystemContext(ctx)

	t, err := r.entClient.Client().Tenant.Query().
		Where(tenant.IDEQ(tenantId)).
		WithPlan(func(q *ent.PlanQuery) { q.WithQuotas() }).
		Only(sysCtx)
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, false, nil
		}
		r.log.Errorf(ctx, "fetch tenant plan for ai quota failed: %s", err.Error())
		return 0, false, aiV1.ErrorInternalServerError("fetch tenant plan failed")
	}
	if t.Edges.Plan == nil {
		return 0, false, nil
	}
	for _, q := range t.Edges.Plan.Edges.Quotas {
		if q.QuotaType != nil && *q.QuotaType == planquota.QuotaTypeAiTokens && q.QuotaValue != nil {
			return int64(*q.QuotaValue), true, nil
		}
	}
	return 0, false, nil
}

// MonthStats 聚合本月（自 monthStart 起）的用量：tokens 总和与调用次数。
// 口径与本表列表一致：租户管理员只算本租户，平台管理员（tenantId=0）算全量。
// 这里换 SystemContext 是为了绕开 privacy 做聚合，谓词就得自己按 viewer 补——原先无条件
// TenantIDEQ(tenantId) 让平台管理员的摘要只统计 tenant_id=0 的行，与它下方全量的流水列表对不上。
func (r *AiUsageLogRepo) MonthStats(ctx context.Context, tenantId int64, monthStart time.Time) (int64, int64, error) {
	sysCtx := viewer.WithSystemContext(ctx)
	var rows []struct {
		Total int64 `sql:"total"`
		Cnt   int64 `sql:"cnt"`
	}
	q := r.entClient.Client().AiUsageLog.Query().Where(aiusagelog.CreatedAtGTE(monthStart))
	if tenantId > 0 {
		q = q.Where(aiusagelog.TenantIDEQ(tenantId))
	}
	err := q.Aggregate(
		ent.As(ent.Sum(aiusagelog.FieldTotalTokens), "total"),
		ent.As(ent.Count(), "cnt"),
	).
		Scan(sysCtx, &rows)
	if err != nil {
		r.log.Errorf(ctx, "month stats failed: %s", err.Error())
		return 0, 0, aiV1.ErrorInternalServerError("month stats failed")
	}
	if len(rows) > 0 {
		return rows[0].Total, rows[0].Cnt, nil
	}
	return 0, 0, nil
}
