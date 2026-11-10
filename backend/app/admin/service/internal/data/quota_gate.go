package data

import (
	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/planquota"
)

// API_CALL 配额闸门（plan_billing.md §7.2）。
//
// 与 USER_LIMIT / STORAGE（service 层 quota_enforcer.go，资源创建入口）不同，
// API_CALL 的"调用入口"在网关侧：每个请求都经过租户闸门
// （pkg/middleware/auth 的 auth.Server → TenantAccessChecker.CheckTenantAccess），
// 闸门实现在本包，因此判定函数也落在本包，与 quota_enforcer 同一套语义：
//
// fail-open：租户无套餐 / 套餐无 API_CALL 配额条目 / 计量查询失败 → 放行；
// 拒绝只在"配额明确存在且已用满"时发生。配额缺失 = 未限售，
// 与登录策略的 fail-open 容错取向一致——计费基础设施抖动不打死租户业务。
//
// 计量口径与 §7.1 GetUsage 的 ApiCallCount 完全同源：sys_api_audit_logs 按租户
// COUNT，展示与拦截读同一张表、同一个查询，两边永远一致。两个口径事实：
//   - 审计中间件位于中间件链最外层，被本闸门拒绝的请求同样落行、同样计入；
//   - 审计归档（ArchiveExpired 导出后删行）会让计数回落——活表现存行数即口径。

// apiCallLimitFromPlanQuotas 从套餐预载的配额边里取 API_CALL 上限；无条目返回 ok=false。
func apiCallLimitFromPlanQuotas(quotas []*ent.PlanQuota) (int64, bool) {
	for _, q := range quotas {
		if q.QuotaType != nil && *q.QuotaType == planquota.QuotaTypeApiCall && q.QuotaValue != nil {
			return int64(*q.QuotaValue), true
		}
	}
	return 0, false
}

// checkApiCallQuota 纯判定（可测）：API_CALL 上限存在且当前计数已达上限 → 403。
// 入口接线见 tenant_access_checker.go 的 CheckTenantAccess 第 4 步。
func checkApiCallQuota(limit int64, hasLimit bool, currentCount int) error {
	if !hasLimit {
		return nil // 套餐未设 API_CALL 配额 = 未限售
	}
	if int64(currentCount) >= limit {
		return adminV1.ErrorForbidden(
			"plan quota exceeded: %s limit %d reached (current %d)",
			"API_CALL", limit, int64(currentCount))
	}
	return nil
}
