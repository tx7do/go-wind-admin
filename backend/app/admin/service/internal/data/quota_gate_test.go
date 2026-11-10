package data

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/api"
	"go-wind-admin/app/admin/service/internal/data/ent/planmodule"
	"go-wind-admin/app/admin/service/internal/data/ent/planquota"
	"go-wind-admin/app/admin/service/internal/data/ent/tenant"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

func ptrOf[T any](v T) *T { return &v }

// TestApiCallLimitFromPlanQuotas 上限抽取：只认 API_CALL 条目；
// nil/空/无该类条目/值缺失一律 (0,false)，混合列表里挑出 API_CALL 条目。
func TestApiCallLimitFromPlanQuotas(t *testing.T) {
	// nil / 空 → 未限售
	_, ok := apiCallLimitFromPlanQuotas(nil)
	require.False(t, ok)
	_, ok = apiCallLimitFromPlanQuotas([]*ent.PlanQuota{})
	require.False(t, ok)

	// 只有非 API_CALL 条目 → 未限售
	_, ok = apiCallLimitFromPlanQuotas([]*ent.PlanQuota{
		{QuotaType: ptrOf(planquota.QuotaTypeUserLimit), QuotaValue: ptrOf(uint64(50))},
		{QuotaType: ptrOf(planquota.QuotaTypeStorage), QuotaValue: ptrOf(uint64(1024))},
	})
	require.False(t, ok)

	// API_CALL 条目但值缺失 → 未限售
	_, ok = apiCallLimitFromPlanQuotas([]*ent.PlanQuota{
		{QuotaType: ptrOf(planquota.QuotaTypeApiCall)},
	})
	require.False(t, ok)
	_, ok = apiCallLimitFromPlanQuotas([]*ent.PlanQuota{
		{QuotaValue: ptrOf(uint64(7))},
	})
	require.False(t, ok)

	// 混合列表里挑出 API_CALL 条目
	limit, ok := apiCallLimitFromPlanQuotas([]*ent.PlanQuota{
		{QuotaType: ptrOf(planquota.QuotaTypeStorage), QuotaValue: ptrOf(uint64(1024))},
		{QuotaType: ptrOf(planquota.QuotaTypeAiTokens), QuotaValue: ptrOf(uint64(999))},
		{QuotaType: ptrOf(planquota.QuotaTypeApiCall), QuotaValue: ptrOf(uint64(7))},
	})
	require.True(t, ok)
	require.Equal(t, int64(7), limit)
}

// TestCheckApiCallQuota 纯判定：未限售放行 / 未满放行 / 已满拒绝（消息钉类型与数值）。
func TestCheckApiCallQuota(t *testing.T) {
	// 未限售（无配额条目形态）：任意计数放行
	require.NoError(t, checkApiCallQuota(0, false, 999999))

	// 未满
	require.NoError(t, checkApiCallQuota(10, true, 9))

	// 已满（current >= limit）→ 拒绝，消息含类型与数值
	err := checkApiCallQuota(10, true, 10)
	require.Error(t, err)
	require.Contains(t, err.Error(), "API_CALL")
	require.Contains(t, err.Error(), "limit 10")

	// 超限 → 拒绝
	require.Error(t, checkApiCallQuota(10, true, 11))
}

// TestCheckTenantAccessApiCallQuotaGate 闸门接线（SQLite 内存库，白盒）：
// 套餐挂 API_CALL 配额（上限 2）+ 模块白名单放行同一路由模板 + 租户 ON 且挂套餐——
// 步骤 1–3 全放行；计数未满（0<2）放行、已满（2=2）403 且消息钉类型与数值。
// 覆盖纯函数测不到的接线层：WithPlan(WithQuotas) 预载真的把配额边装进 Edges、
// 计数查询真的按租户 COUNT。
func TestCheckTenantAccessApiCallQuotaGate(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	ctx := enttest.NewSystemContext(context.Background())
	client := entClient.Client()

	// 套餐（全字段默认）+ API_CALL 上限 2 + SYSTEM 模块白名单行
	planRow, err := client.Plan.Create().Save(ctx)
	require.NoError(t, err)
	require.NoError(t, client.PlanQuota.Create().
		SetPlanID(planRow.ID).
		SetQuotaType(planquota.QuotaTypeApiCall).
		SetQuotaValue(2).
		Exec(ctx))
	require.NoError(t, client.PlanModule.Create().
		SetPlanID(planRow.ID).
		SetModule(planmodule.ModuleSystem).
		Exec(ctx))

	// 租户挂套餐、status=ON
	tenantRow, err := client.Tenant.Create().
		SetName("配额闸门接线租户").
		SetCode("QUOTA_GATE_T").
		SetStatus(tenant.StatusOn).
		SetPlanID(planRow.ID).
		Save(ctx)
	require.NoError(t, err)

	// Api 行：路由模板与调用参数一致、business_module=SYSTEM（第 3 段白名单命中）
	require.NoError(t, client.Api.Create().
		SetNillablePath(ptrOf("/admin/v1/quota-gate/probe")).
		SetNillableMethod(ptrOf("GET")).
		SetBusinessModule(api.BusinessModuleSystem).
		Exec(ctx))

	checker := &TenantAccessCheckerImpl{
		entClient: client,
		log:       bLogger.NewHelper(bLogger.NopLogger()),
	}

	// 计数 0 < 2 → 放行（含步骤 1–3 的放行路径）
	require.NoError(t, checker.CheckTenantAccess(ctx, tenantRow.ID, "/admin/v1/quota-gate/probe", "GET"))

	// 两行审计行 → 计数 2 = 上限 2 → 403，消息钉类型与数值
	require.NoError(t, client.ApiAuditLog.Create().
		SetTenantID(tenantRow.ID).
		Exec(ctx))
	require.NoError(t, client.ApiAuditLog.Create().
		SetTenantID(tenantRow.ID).
		Exec(ctx))
	err = checker.CheckTenantAccess(ctx, tenantRow.ID, "/admin/v1/quota-gate/probe", "GET")
	require.Error(t, err)
	require.Contains(t, err.Error(), "API_CALL")
	require.Contains(t, err.Error(), "limit 2")
}
