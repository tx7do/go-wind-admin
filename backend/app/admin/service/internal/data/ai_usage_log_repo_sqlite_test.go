// AiUsageLogRepo 的 SQLite 内存库测试（白盒，包内测试）。
//
// 这张表是计费与配额的流水真相，测试盯两件"算错一次就多扣或少放行一个租户"的事：
//   - 聚合口径：SumTokensByTenantSince / MonthStats 的租户过滤与时间下界——平台管理员
//     （tenantId=0）必须算全量、租户必须只算自己，口径歪了配额闸门会放错人；
//   - 套餐配额读取：FetchTenantTokenQuotaLimit 的四态（无租户/无套餐/有套餐无该维度/
//     有 AI_TOKENS 配额）——"没配 = 不限量"是和其他维度一致的缺省语义，误读成 0 会
//     把不限量租户直接饿死。
package data

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/planquota"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// seedUsageLog 直插一条指定租户/总量/时间的用量行：repo 的 Create 只写 now，
// 而时间下界聚合的窗口前老账需要钉死的历史时间（Update builder 保护 created_at
// 不可改，只能建行时显式 SetCreatedAt）。
func seedUsageLog(t *testing.T, entClient *entCrud.EntClient[*ent.Client], ctx context.Context, tenantID uint32, totalTokens uint32, createdAt time.Time) uint32 {
	t.Helper()
	entity, err := entClient.Client().AiUsageLog.Create().
		SetTenantID(tenantID).
		SetTotalTokens(totalTokens).
		SetModelName("seed-model").
		SetCreatedAt(createdAt).
		Save(ctx)
	require.NoError(t, err, "种子用量行应直插成功")
	return entity.ID
}

// newUsageLog 构造一条用量流水载荷，字段可按需覆盖。
func newUsageLog(tenantID uint32, totalTokens uint32) *aiV1.AiUsageLog {
	return &aiV1.AiUsageLog{
		ProviderId:       trans.Ptr(uint32(11)),
		ConversationId:   trans.Ptr(uint32(22)),
		UserId:           trans.Ptr(uint32(33)),
		TenantId:         trans.Ptr(tenantID),
		ModelName:        trans.Ptr("qwen-max"),
		PromptTokens:     trans.Ptr(totalTokens / 2),
		CompletionTokens: trans.Ptr(totalTokens - totalTokens/2),
		TotalTokens:      trans.Ptr(totalTokens),
		DurationMs:       trans.Ptr(uint32(1200)),
	}
}

// TestAiUsageLogRepoSqlite_Create 验证字段级落库：token 三件套、租户/用户归属、
// created_at 显式写入；nil 载荷 BadRequest；List 的 total 与 Count 同口径。
func TestAiUsageLogRepoSqlite_Create(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiUsageLogRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	require.NoError(t, repo.Create(ctx, newUsageLog(7, 500)), "完整载荷的 Create 应成功")
	require.NoError(t, repo.Create(ctx, newUsageLog(7, 300)))
	require.NoError(t, repo.Create(ctx, newUsageLog(8, 100)))

	row, err := entClient.Client().AiUsageLog.Query().First(ctx)
	require.NoError(t, err)
	require.Equal(t, uint32(11), *row.ProviderID, "provider_id 应按载荷落库")
	require.Equal(t, uint32(22), *row.ConversationID)
	require.Equal(t, uint32(33), *row.UserID)
	require.Equal(t, uint32(7), *row.TenantID, "租户归属必须落库：配额聚合按它分组")
	require.Equal(t, "qwen-max", *row.ModelName)
	require.Equal(t, uint32(250), *row.PromptTokens)
	require.Equal(t, uint32(250), *row.CompletionTokens)
	require.Equal(t, uint32(500), *row.TotalTokens, "total_tokens 是配额聚合的累加列，必须保值")
	require.Equal(t, uint32(1200), *row.DurationMs)
	require.NotNil(t, row.CreatedAt, "created_at 必须显式写入：时间下界聚合全靠它")

	cnt, err := repo.Count(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, uint64(3), cnt.GetCount())

	require.Error(t, repo.Create(ctx, nil), "nil 载荷应返回 BadRequest")

	// List：total 口径与 Count 一致
	all, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(3), all.GetTotal())
	require.Len(t, all.GetItems(), 3)

	filtered, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_Query{
			Query: `{"model_name__contains":"qwen"}`,
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(3), filtered.GetTotal(), "contains 过滤应命中全部 qwen 行")
	require.Len(t, filtered.GetItems(), 3)

	filtered, err = repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_Query{
			Query: `{"model_name__contains":"glm"}`,
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(0), filtered.GetTotal(), "不含关键词的 contains 过滤应为 0 行")

	_, err = repo.List(ctx, nil)
	require.Error(t, err, "nil 分页请求应返回 BadRequest")
}

// TestAiUsageLogRepoSqlite_SumTokensByTenantSince 验证配额检查的求和口径：
// 只算该租户的行、只算 since 之后的行（把别家租户或历史账算进来都会放错配额闸门）。
func TestAiUsageLogRepoSqlite_SumTokensByTenantSince(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiUsageLogRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	// 一行落在窗口前（老账 500）、一行在窗口内（300）、别家租户 100
	cutoff := time.Now().Add(-time.Hour)
	seedUsageLog(t, entClient, ctx, 7, 500, cutoff.Add(-2*time.Hour))
	err := repo.Create(ctx, newUsageLog(7, 300))
	require.NoError(t, err)
	err = repo.Create(ctx, newUsageLog(8, 100))
	require.NoError(t, err)

	sum, err := repo.SumTokensByTenantSince(ctx, 7, cutoff)
	require.NoError(t, err)
	require.Equal(t, uint64(300), sum, "只应累加租户 7 在 since 之后的 300，窗口前的 500 不算")

	sum, err = repo.SumTokensByTenantSince(ctx, 8, cutoff)
	require.NoError(t, err)
	require.Equal(t, uint64(100), sum, "租户 8 的账不能串到租户 7 头上")

	sum, err = repo.SumTokensByTenantSince(ctx, 999, cutoff)
	require.NoError(t, err)
	require.Equal(t, uint64(0), sum, "没有流水的租户求和应为 0 而不是报错")
}

// TestAiUsageLogRepoSqlite_MonthStats 验证月度摘要的两种口径：租户管理员只算本租户、
// 平台管理员（tenantId=0）算全量；返回 (tokens 总和, 调用次数)。
func TestAiUsageLogRepoSqlite_MonthStats(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiUsageLogRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	err := repo.Create(ctx, newUsageLog(7, 500))
	require.NoError(t, err)
	err = repo.Create(ctx, newUsageLog(7, 300))
	require.NoError(t, err)
	err = repo.Create(ctx, newUsageLog(8, 100))
	require.NoError(t, err)

	// 一个月之前的老账：不得计入本月
	monthStart := time.Now().Add(-15 * 24 * time.Hour)
	seedUsageLog(t, entClient, ctx, 7, 9000, monthStart.Add(-24*time.Hour))

	tokens, calls, err := repo.MonthStats(ctx, 7, monthStart)
	require.NoError(t, err)
	require.Equal(t, uint64(800), tokens, "租户 7 本月 tokens 只含窗口内两行（老账 9000 不算）")
	require.Equal(t, uint64(2), calls, "调用次数按行数计，窗口外不计")

	tokens, calls, err = repo.MonthStats(ctx, 0, monthStart)
	require.NoError(t, err)
	require.Equal(t, uint64(900), tokens, "平台管理员（tenantId=0）应统计全租户的窗口内行")
	require.Equal(t, uint64(3), calls)

	tokens, calls, err = repo.MonthStats(ctx, 999, monthStart)
	require.NoError(t, err)
	require.Equal(t, uint64(0), tokens, "无流水租户的月度摘要应为 0")
	require.Equal(t, uint64(0), calls)
}

// TestAiUsageLogRepoSqlite_FetchTenantTokenQuotaLimit 验证四态：tenantId=0 直接放行、
// 无套餐视为未配置、套餐没配 AI_TOKENS 维度视为未配置、配上了返回上限
// （"没配 = 不限量"，误读成 0 会饿死不限量租户）。
func TestAiUsageLogRepoSqlite_FetchTenantTokenQuotaLimit(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiUsageLogRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	// tenantId=0：平台自身不走租户配额
	limit, configured, err := repo.FetchTenantTokenQuotaLimit(ctx, 0)
	require.NoError(t, err)
	require.False(t, configured, "平台身份（tenantId=0）应视为不限量")
	require.Equal(t, uint64(0), limit)

	// 无套餐租户：未配置
	tenantNoPlan, err := entClient.Client().Tenant.Create().
		SetNillableName(trans.Ptr("裸租户")).
		Save(ctx)
	require.NoError(t, err)

	limit, configured, err = repo.FetchTenantTokenQuotaLimit(ctx, tenantNoPlan.ID)
	require.NoError(t, err)
	require.False(t, configured, "无套餐租户应视为不限量")
	require.Equal(t, uint64(0), limit)

	// 有套餐但没配 AI_TOKENS 维度：未配置
	planNoTokens, err := entClient.Client().Plan.Create().
		SetNillableName(trans.Ptr("无AI额度套餐")).
		Save(ctx)
	require.NoError(t, err)
	_, err = entClient.Client().PlanQuota.Create().
		SetPlan(planNoTokens).
		SetQuotaType(planquota.QuotaTypeApiCall).
		SetQuotaValue(1000).
		Save(ctx)
	require.NoError(t, err)
	tenantOtherQuota, err := entClient.Client().Tenant.Create().
		SetNillableName(trans.Ptr("别维度配额租户")).
		SetPlan(planNoTokens).
		Save(ctx)
	require.NoError(t, err)

	limit, configured, err = repo.FetchTenantTokenQuotaLimit(ctx, tenantOtherQuota.ID)
	require.NoError(t, err)
	require.False(t, configured, "套餐只配了别的维度时 AI_TOKENS 应视为不限量")

	// 有套餐且配了 AI_TOKENS：返回上限
	planWithTokens, err := entClient.Client().Plan.Create().
		SetNillableName(trans.Ptr("含AI额度套餐")).
		Save(ctx)
	require.NoError(t, err)
	_, err = entClient.Client().PlanQuota.Create().
		SetPlan(planWithTokens).
		SetQuotaType(planquota.QuotaTypeAiTokens).
		SetQuotaValue(123456).
		Save(ctx)
	require.NoError(t, err)
	tenantWithQuota, err := entClient.Client().Tenant.Create().
		SetNillableName(trans.Ptr("限额租户")).
		SetPlan(planWithTokens).
		Save(ctx)
	require.NoError(t, err)

	limit, configured, err = repo.FetchTenantTokenQuotaLimit(ctx, tenantWithQuota.ID)
	require.NoError(t, err)
	require.True(t, configured, "配了 AI_TOKENS 配额应返回已配置")
	require.Equal(t, uint64(123456), limit, "上限应取套餐配额的配值")

	// 不存在的租户：视同未配置（NotFound 被吞成 (0,false,nil)，让配额检查按不限量放行而非报错）
	limit, configured, err = repo.FetchTenantTokenQuotaLimit(ctx, 987654)
	require.NoError(t, err)
	require.False(t, configured, "租户行不存在应视为未配置而不是错误")
	require.Equal(t, uint64(0), limit)
}
