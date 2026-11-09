// AiUsageLogService 的 SQLite 内存库测试（白盒，包内测试）。
//
// 覆盖目标：
//   - GetUsageSummary 的两种口径：租户用户只算本租户的当月用量并回套餐配额；
//     平台用户（tenant 0）算全量且配额恒"未配置"（平台不限量）；
//   - 月度窗口：上月老账不得计入本月（summary 与流水列表同口径）；
//   - List / Count 的直通查询；入参守卫（nil 请求 / 无身份）。
package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/emptypb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	entCrud "github.com/tx7do/go-crud/entgo"
	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/planquota"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/middleware/auth"
)

// newAiUsageLogServiceForTest 白盒复刻 NewAiUsageLogService 的字段初始化。
func newAiUsageLogServiceForTest(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) *AiUsageLogService {
	t.Helper()
	return &AiUsageLogService{
		log:  bLogger.NewHelper(bLogger.NopLogger()),
		repo: data.NewAiUsageLogRepoForTest(entClient),
	}
}

// aiUsageOperator 构造指定用户/租户的操作者上下文。
func aiUsageOperator(ctx context.Context, userID uint64, tenantID uint64) context.Context {
	return auth.NewContext(ctx, &authenticationV1.UserTokenPayload{
		UserId:   userID,
		TenantId: trans.Ptr(tenantID),
	})
}

// aiUsageSeedRow 直插一条指定租户/总量/时间的用量行（服务侧 Create 只写 now，
// 钉"上月老账不进本月"必须直插历史时间）。
func aiUsageSeedRow(t *testing.T, entClient *entCrud.EntClient[*ent.Client], ctx context.Context, tenantID uint64, totalTokens uint32, createdAt time.Time) {
	t.Helper()
	_, err := entClient.Client().AiUsageLog.Create().
		SetTenantID(tenantID).
		SetTotalTokens(totalTokens).
		SetModelName("seed-model").
		SetCreatedAt(createdAt).
		Save(ctx)
	require.NoError(t, err, "种子用量行应直插成功")
}

// aiUsageSeedTenantWithQuota 建租户并挂配了 AI_TOKENS 上限的套餐，返回租户 ID。
func aiUsageSeedTenantWithQuota(t *testing.T, entClient *entCrud.EntClient[*ent.Client], ctx context.Context, name string, aiTokenLimit uint64) uint64 {
	t.Helper()
	plan, err := entClient.Client().Plan.Create().SetNillableName(trans.Ptr(name + "套餐")).Save(ctx)
	require.NoError(t, err)
	_, err = entClient.Client().PlanQuota.Create().
		SetPlan(plan).
		SetQuotaType(planquota.QuotaTypeAiTokens).
		SetQuotaValue(aiTokenLimit).
		Save(ctx)
	require.NoError(t, err)
	tenant, err := entClient.Client().Tenant.Create().
		SetNillableName(trans.Ptr(name)).
		SetPlan(plan).
		Save(ctx)
	require.NoError(t, err)
	return tenant.ID
}

// TestAiUsageLogService_GetUsageSummary 验证月度摘要：租户口径只算本租户的当月流水
// 并带回套餐配额；平台口径算全量且配额恒未配置；上月老账不进本月。
func TestAiUsageLogService_GetUsageSummary(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newAiUsageLogServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	tenantA := aiUsageSeedTenantWithQuota(t, entClient, ctx, "摘要租户甲", 10000)
	tenantB := aiUsageSeedTenantWithQuota(t, entClient, ctx, "摘要租户乙", 500)

	// 租户甲本月：300 + 200（两笔调用）；租户乙本月：300（一笔）
	now := time.Now()
	aiUsageSeedRow(t, entClient, ctx, tenantA, 300, now.Add(-time.Hour))
	aiUsageSeedRow(t, entClient, ctx, tenantA, 200, now.Add(-2*time.Hour))
	aiUsageSeedRow(t, entClient, ctx, tenantB, 300, now.Add(-3*time.Hour))
	// 租户甲的上月老账 9000：不得计入本月摘要
	aiUsageSeedRow(t, entClient, ctx, tenantA, 9000, now.AddDate(0, -1, 0))

	// 租户甲视角：只算本租户，配额来自套餐
	resp, err := svc.GetUsageSummary(aiUsageOperator(ctx, 51, tenantA), &emptypb.Empty{})
	require.NoError(t, err, "租户用户的月度摘要应成功")
	require.Equal(t, uint32(500), resp.GetMonthTokens(), "租户甲本月 tokens 应只含窗口内两行（老账 9000 不算）")
	require.Equal(t, uint32(2), resp.GetMonthCalls(), "调用次数按行数计")
	require.True(t, resp.GetQuotaConfigured(), "套餐配了 AI_TOKENS 应回已配置")
	require.Equal(t, uint64(10000), resp.GetQuotaLimit(), "配额上限应取套餐配值")

	// 租户乙视角：只算自己的
	resp, err = svc.GetUsageSummary(aiUsageOperator(ctx, 52, tenantB), &emptypb.Empty{})
	require.NoError(t, err)
	require.Equal(t, uint32(300), resp.GetMonthTokens(), "租户乙的账不能串进租户甲的摘要，反之亦然")
	require.Equal(t, uint32(1), resp.GetMonthCalls())
	require.Equal(t, uint64(500), resp.GetQuotaLimit())

	// 平台用户（tenant 0）：全量口径 + 配额恒未配置（平台不限量）
	resp, err = svc.GetUsageSummary(aiUsageOperator(ctx, 53, 0), &emptypb.Empty{})
	require.NoError(t, err)
	require.Equal(t, uint32(800), resp.GetMonthTokens(), "平台视角应统计全租户的当月流水")
	require.Equal(t, uint32(3), resp.GetMonthCalls())
	require.False(t, resp.GetQuotaConfigured(), "平台用户配额恒未配置（不限量语义）")
	require.Equal(t, uint64(0), resp.GetQuotaLimit())

	// 无身份
	_, err = svc.GetUsageSummary(ctx, &emptypb.Empty{})
	require.Error(t, err, "无操作者身份的摘要必须被拒绝")
}

// TestAiUsageLogService_ListAndCount 验证流水列表与计数的直通查询与 nil 守卫。
func TestAiUsageLogService_ListAndCount(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newAiUsageLogServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := aiUsageOperator(ctx, 61, 0)

	aiUsageSeedRow(t, entClient, ctx, 7, 500, time.Now().Add(-time.Hour))
	aiUsageSeedRow(t, entClient, ctx, 8, 300, time.Now())

	listResp, err := svc.List(opCtx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), listResp.GetTotal(), "列表 total 应与行数一致")
	require.Len(t, listResp.GetItems(), 2)

	countResp, err := svc.Count(opCtx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), countResp.GetCount(), "计数应与行数一致")

	_, err = svc.List(opCtx, nil)
	require.Error(t, err, "nil 分页请求应返回 BadRequest")
}
