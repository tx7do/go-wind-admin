package service

import (
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
)

// 套餐配额硬限制（plan_billing.md §7.2 落地）。
//
// 与计量（GetUsage，只读展示）不同，检查器在**资源创建入口**拒绝超限操作：
//   - USER_LIMIT：创建用户前，当前租户用户数 >= 上限 → 拒绝；
//   - STORAGE：文件上传前，当前占用 + 本次大小 > 上限 → 拒绝；
//   - API_CALL：调用入口在网关侧（租户闸门中间件 → data 层
//     TenantAccessCheckerImpl.CheckTenantAccess 第 4 步），判定函数在 data 包
//     quota_gate.go（闸门实现所在包）——与本文件同一套 fail-open 语义。
//
// fail-open 语义：租户无套餐 / 无该类型配额条目 / 计量失败 → 放行（仅告警）。
// 配额缺失 = 未限售该资源，与登录策略的 fail-open 容错取向一致；
// 拒绝只在"配额明确存在且已用满"时发生，避免计费基础设施抖动把租户业务打死。

// quotaExceededError 构造配额超限错误（403，消息带类型与数值供前端展示）。
func quotaExceededError(quotaType string, limit, current int64) error {
	return identityV1.ErrorForbidden(
		"plan quota exceeded: %s limit %d reached (current %d)",
		quotaType, limit, current)
}

// quotaLimitFromUsages 从配额条目里找指定类型的上限；无条目返回 ok=false。
func quotaLimitFromUsages(quotas []*identityV1.QuotaUsage, quotaType identityV1.PlanQuota_QuotaType) (int64, bool) {
	for _, q := range quotas {
		if q.GetQuotaType() == quotaType {
			return int64(q.GetQuotaValue()), true
		}
	}
	return 0, false
}

// checkUserQuotaWith 纯判定（可测）：从配额条目找 USER_LIMIT 上限，已满即超限。
// 入口接线见 user_service.go 的 Create（GetUsage 计数后调用）。
func checkUserQuotaWith(quotas []*identityV1.QuotaUsage, currentUsers int64) error {
	limit, ok := quotaLimitFromUsages(quotas, identityV1.PlanQuota_USER_LIMIT)
	if !ok {
		return nil // 套餐未设用户上限 = 未限售
	}
	if currentUsers >= limit {
		return quotaExceededError("USER_LIMIT", limit, currentUsers)
	}
	return nil
}

// checkStorageQuotaWith 纯判定（可测）：STORAGE 上限，当前占用 + 本次上传 > 上限即超。
// 入口接线见 file_transfer_service.go 的 directUploadFile。
func checkStorageQuotaWith(quotas []*identityV1.QuotaUsage, currentBytes, uploadBytes int64) error {
	limit, ok := quotaLimitFromUsages(quotas, identityV1.PlanQuota_STORAGE)
	if !ok {
		return nil
	}
	if currentBytes+uploadBytes > limit {
		return quotaExceededError("STORAGE", limit, currentBytes)
	}
	return nil
}
