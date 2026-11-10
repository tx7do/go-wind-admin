package data

import (
	"context"
	"math"
	"time"

	"github.com/tx7do/go-crud/viewer"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/planquota"
	"go-wind-admin/app/admin/service/internal/data/ent/tenant"
)

// 套餐配额水位扫描（plan_billing.md §7.3、task_system.md §5.7）。
//
// 与 §7.2 的硬执行（超限拒绝）互补：这里是只读扫描 + 提前告警——在用量到达
// 上限之前把水位事实通知租户管理员，让租户有机会提前扩容。扫描与计量同源：
// USER_LIMIT / STORAGE / API_CALL 复用 CountUsersInTenant / SumStorageInTenant /
// CountApiCallsInTenant（与 GetUsage 同一查询），AI_TOKENS 为月度口径
//（SumTokensByTenantSince，本月 1 日起）。
//
// 阈值是运维口径常量（非套餐数据、当前无配置面）：用量/上限达到 80% 即命中。
// 单维度计量查询失败记日志后跳过该维度——扫描不因计量抖动中断，也不因
// 某维度缺数据而误报其余维度。待扫描租户仅 status==ON（停用/到期冻结的租户
// 已被闸门拦截，告警是噪音）。

// QuotaWatermarkThreshold 水位阈值（用量/上限达到该比例即命中）。
const QuotaWatermarkThreshold = 0.8

// QuotaWatermarkHit 单租户单配额类型的水位命中（只读扫描结果，无任何拒绝语义）。
type QuotaWatermarkHit struct {
	TenantID   int64
	TenantName string
	// AdminUserID 租户管理员（sys_tenants.admin_user_id；0 = 未设置，投递侧跳过）。
	AdminUserID int64
	// QuotaType 配额类型技术名（proto 枚举名，与用量页/闸门报错口径一致，不做翻译）。
	QuotaType string
	Used      int64
	Limit     int64
	// RatioPct 用量百分数（四舍五入；limit==0 且 used>0 视为 100）。
	RatioPct int64
}

// quotaWatermarkExceeded 纯判定（可测）：limit==0 且 used>0 视为命中（配 0 上限
// 且已有用量 = 事实超限）；否则 used >= 阈值×limit 即命中。
func quotaWatermarkExceeded(used, limit int64) bool {
	if limit == 0 {
		return used > 0
	}
	return float64(used) >= QuotaWatermarkThreshold*float64(limit)
}

// ratioPct 纯函数（可测）：用量百分数。
func ratioPct(used, limit int64) int64 {
	if limit == 0 {
		if used > 0 {
			return 100
		}
		return 0
	}
	return int64(math.Round(float64(used) / float64(limit) * 100))
}

// ScanQuotaWatermarks 扫描全部 ON 租户的配额水位，返回命中清单。
// 只读：不拒绝任何请求、不写任何业务行；投递由服务层（站内信）完成。
func (r *TenantUsageRepo) ScanQuotaWatermarks(ctx context.Context, aiRepo *AiUsageLogRepo, monthStart time.Time) ([]QuotaWatermarkHit, error) {
	sysCtx := viewer.WithSystemContext(ctx)

	tenants, err := r.entClient.Client().Tenant.Query().
		Where(tenant.StatusEQ(tenant.StatusOn)).
		WithPlan(func(q *ent.PlanQuery) { q.WithQuotas() }).
		All(sysCtx)
	if err != nil {
		r.log.Errorf(ctx, "quota watermark scan: query tenants failed: %v", err)
		return nil, err
	}

	var hits []QuotaWatermarkHit
	for _, t := range tenants {
		if t.Edges.Plan == nil || len(t.Edges.Plan.Edges.Quotas) == 0 {
			continue
		}
		tenantName := ""
		if t.Name != nil {
			tenantName = *t.Name
		}
		adminUserID := int64(0)
		if t.AdminUserID != nil {
			adminUserID = *t.AdminUserID
		}

		// 各维度用量。单维度计量失败记日志（带原始错误）后跳过该维度：
		// 扫描继续、该租户该维度不产生命中。
		usage := map[planquota.QuotaType]int64{}
		if n, uerr := r.CountUsersInTenant(sysCtx, t.ID); uerr != nil {
			r.log.Errorf(ctx, "quota watermark scan: count users for tenant %d failed: %v", t.ID, uerr)
		} else {
			usage[planquota.QuotaTypeUserLimit] = n
		}
		if n, serr := r.SumStorageInTenant(sysCtx, t.ID); serr != nil {
			r.log.Errorf(ctx, "quota watermark scan: sum storage for tenant %d failed: %v", t.ID, serr)
		} else {
			usage[planquota.QuotaTypeStorage] = n
		}
		if n, aerr := r.CountApiCallsInTenant(sysCtx, t.ID); aerr != nil {
			r.log.Errorf(ctx, "quota watermark scan: count api calls for tenant %d failed: %v", t.ID, aerr)
		} else {
			usage[planquota.QuotaTypeApiCall] = n
		}
		if aiRepo != nil {
			if n, terr := aiRepo.SumTokensByTenantSince(sysCtx, t.ID, monthStart); terr != nil {
				r.log.Errorf(ctx, "quota watermark scan: sum ai tokens for tenant %d failed: %v", t.ID, terr)
			} else {
				usage[planquota.QuotaTypeAiTokens] = n
			}
		}

		for _, q := range t.Edges.Plan.Edges.Quotas {
			if q.QuotaType == nil || q.QuotaValue == nil {
				continue
			}
			used, ok := usage[*q.QuotaType]
			if !ok {
				// 该维度计量失败被跳过，或该配额类型没有用量源：不判定。
				continue
			}
			if quotaWatermarkExceeded(used, int64(*q.QuotaValue)) {
				hits = append(hits, QuotaWatermarkHit{
					TenantID:    t.ID,
					TenantName:  tenantName,
					AdminUserID: adminUserID,
					QuotaType:   mapEntQuotaTypeToProto(*q.QuotaType).String(),
					Used:        used,
					Limit:       int64(*q.QuotaValue),
					RatioPct:    ratioPct(used, int64(*q.QuotaValue)),
				})
			}
		}
	}

	r.log.Infof(ctx, "quota watermark scan: %d tenants scanned, %d watermark hits", len(tenants), len(hits))
	return hits, nil
}
