package data

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	"go-wind-admin/app/admin/service/internal/data/ent/planquota"
	"go-wind-admin/app/admin/service/internal/data/ent/tenant"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// TestQuotaWatermarkExceeded 纯判定：达到阈值命中、未达到不命中、恰好等于命中
// （阈值是 >=）；limit==0 且 used>0 视为命中，0/0 不命中。
func TestQuotaWatermarkExceeded(t *testing.T) {
	cases := []struct {
		used, limit int64
		want        bool
	}{
		{8, 10, true},
		{7, 10, false},
		{10, 10, true},
		{0, 10, false},
		{80, 100, true},
		{79, 100, false},
		{1, 0, true},
		{0, 0, false},
	}
	for _, c := range cases {
		require.Equal(t, c.want, quotaWatermarkExceeded(c.used, c.limit), "used=%d limit=%d", c.used, c.limit)
	}
}

// TestRatioPct 百分数：四舍五入；limit==0 且 used>0 为 100，0/0 为 0。
func TestRatioPct(t *testing.T) {
	cases := []struct {
		used, limit int64
		want        int64
	}{
		{8, 10, 80},
		{7, 10, 70},
		{10, 10, 100},
		{0, 10, 0},
		{80, 100, 80},
		{1, 0, 100},
		{0, 0, 0},
	}
	for _, c := range cases {
		require.Equal(t, c.want, ratioPct(c.used, c.limit), "used=%d limit=%d", c.used, c.limit)
	}
}

// TestScanQuotaWatermarksSqlite 接线级（SQLite 内存库，白盒）：
// 命中租户（计数=上限=2 → 100%）产出一条命中并逐字段传播（含 admin_user_id 与
// 租户名）；未达水位租户（1/2）、OFF 租户（状态过滤）、无配额套餐租户均不命中。
// aiRepo 传 nil（AI 维度跳过，其余三维度照常）。
func TestScanQuotaWatermarksSqlite(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	ctx := enttest.NewSystemContext(context.Background())
	client := entClient.Client()

	planWithQuota, err := client.Plan.Create().Save(ctx)
	require.NoError(t, err)
	require.NoError(t, client.PlanQuota.Create().
		SetPlanID(planWithQuota.ID).
		SetQuotaType(planquota.QuotaTypeApiCall).
		SetQuotaValue(2).
		Exec(ctx))
	planNoQuota, err := client.Plan.Create().Save(ctx)
	require.NoError(t, err)

	// 命中租户：ON + 带配额套餐 + admin_user_id + 2 行审计行（= 上限 2）
	tenantHit, err := client.Tenant.Create().
		SetName("水位租户甲").
		SetCode("WM_HIT").
		SetStatus(tenant.StatusOn).
		SetAdminUserID(77).
		SetPlanID(planWithQuota.ID).
		Save(ctx)
	require.NoError(t, err)
	require.NoError(t, client.ApiAuditLog.Create().SetTenantID(tenantHit.ID).Exec(ctx))
	require.NoError(t, client.ApiAuditLog.Create().SetTenantID(tenantHit.ID).Exec(ctx))

	// 未达水位：1 行（1/2 = 50% < 80%）
	tenantHalf, err := client.Tenant.Create().
		SetName("水位租户乙").
		SetCode("WM_HALF").
		SetStatus(tenant.StatusOn).
		SetAdminUserID(88).
		SetPlanID(planWithQuota.ID).
		Save(ctx)
	require.NoError(t, err)
	require.NoError(t, client.ApiAuditLog.Create().SetTenantID(tenantHalf.ID).Exec(ctx))

	// OFF 租户：状态过滤，不进扫描
	tenantOff, err := client.Tenant.Create().
		SetName("水位租户丙").
		SetCode("WM_OFF").
		SetStatus(tenant.StatusOff).
		SetAdminUserID(99).
		SetPlanID(planWithQuota.ID).
		Save(ctx)
	require.NoError(t, err)
	require.NoError(t, client.ApiAuditLog.Create().SetTenantID(tenantOff.ID).Exec(ctx))
	require.NoError(t, client.ApiAuditLog.Create().SetTenantID(tenantOff.ID).Exec(ctx))

	// 无配额套餐租户：套餐无边，跳过
	tenantNoQuota, err := client.Tenant.Create().
		SetName("水位租户丁").
		SetCode("WM_NOQUOTA").
		SetStatus(tenant.StatusOn).
		SetPlanID(planNoQuota.ID).
		Save(ctx)
	require.NoError(t, err)
	require.NoError(t, client.ApiAuditLog.Create().SetTenantID(tenantNoQuota.ID).Exec(ctx))

	repo := &TenantUsageRepo{
		entClient:     entClient,
		authenticator: nil,
		log:           bLogger.NewHelper(bLogger.NopLogger()),
	}
	// aiRepo 传 nil：AI 维度跳过；monthStart 仅喂 aiRepo，传零值即可。
	hits, err := repo.ScanQuotaWatermarks(ctx, nil, time.Time{})
	require.NoError(t, err)
	require.Len(t, hits, 1, "只应命中水位租户甲（乙未达阈值、丙 OFF、丁无配额）")
	h := hits[0]
	require.Equal(t, tenantHit.ID, h.TenantID)
	require.Equal(t, "水位租户甲", h.TenantName)
	require.Equal(t, int64(77), h.AdminUserID)
	require.Equal(t, "API_CALL", h.QuotaType)
	require.Equal(t, int64(2), h.Used)
	require.Equal(t, int64(2), h.Limit)
	require.Equal(t, int64(100), h.RatioPct)
}
