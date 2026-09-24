package data

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	entPlan "go-wind-admin/app/admin/service/internal/data/ent/plan"
	entPlanQuota "go-wind-admin/app/admin/service/internal/data/ent/planquota"
	entTenant "go-wind-admin/app/admin/service/internal/data/ent/tenant"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newTenantUsageRepoSqlite 用 enttest helper 构造 TenantUsageRepo，
// 逐字段复刻 NewTenantUsageRepo（log 换 NopLogger；authenticator 按生产签名保留，
// 测试不覆盖吊销令牌链路传 nil，GetUsage/CleanupTenantData/EnforceExpiryPolicies
// 对 nil authenticator 均有守卫）。
func newTenantUsageRepoSqlite(t *testing.T) *TenantUsageRepo {
	t.Helper()
	return &TenantUsageRepo{
		entClient:     enttest.NewEntClientForTest(t),
		authenticator: nil,
		log:           bLogger.NewHelper(bLogger.NopLogger()),
	}
}

// TestTenantUsageRepoSqlite_GetUsageTenantNotFound 验证租户不存在时 GetUsage 报错。
func TestTenantUsageRepoSqlite_GetUsageTenantNotFound(t *testing.T) {
	repo := newTenantUsageRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	_, err := repo.GetUsage(ctx, 424242)
	require.Error(t, err, "查询不存在的租户应返回错误")
}

// TestTenantUsageRepoSqlite_GetUsageEmptyTenant 验证无套餐、无数据的租户：
// 计数全零、无套餐上限回填。
func TestTenantUsageRepoSqlite_GetUsageEmptyTenant(t *testing.T) {
	repo := newTenantUsageRepoSqlite(t)
	client := repo.entClient.Client()
	ctx := enttest.NewSystemViewerCtx(context.Background())

	tenantRow, err := client.Tenant.Create().
		SetName("sqlite_usage_tenant_empty").
		SetCode("SQLITE_USAGE_TENANT_EMPTY").
		Save(ctx)
	require.NoError(t, err, "直插空租户行应成功")

	usage, err := repo.GetUsage(ctx, tenantRow.ID)
	require.NoError(t, err, "空租户 GetUsage 应成功")
	require.Equal(t, tenantRow.ID, usage.GetTenantId(), "TenantId 应为请求的租户 ID")
	require.Zero(t, usage.GetUserCount(), "无用户时 UserCount 应为 0")
	require.Zero(t, usage.GetStorageUsedBytes(), "无文件时 StorageUsedBytes 应为 0")
	require.Zero(t, usage.GetApiCallCount(), "无 API 审计时 ApiCallCount 应为 0")
	require.Zero(t, usage.GetPlanId(), "无套餐关联时 PlanId 不应回填")
	require.Empty(t, usage.GetPlanName(), "无套餐关联时 PlanName 不应回填")
	require.Empty(t, usage.GetQuotas(), "无套餐关联时 Quotas 应为空")
}

// TestTenantUsageRepoSqlite_GetUsageWithPlanAndData 验证带套餐与用量数据的租户：
// 套餐名与三类配额上限回填（含 quota_type 映射），用户数 / 存储字节数 / API 调用量
// 按租户聚合（另一租户的数据不计入本租户）。
func TestTenantUsageRepoSqlite_GetUsageWithPlanAndData(t *testing.T) {
	repo := newTenantUsageRepoSqlite(t)
	client := repo.entClient.Client()
	ctx := enttest.NewSystemViewerCtx(context.Background())

	now := time.Now()

	// 租户 A：挂套餐（三类配额），带 2 用户 / 2 文件(100+200) / 2 条 API 审计
	tenantA, err := client.Tenant.Create().
		SetName("sqlite_usage_tenant_a").
		SetCode("SQLITE_USAGE_TENANT_A").
		Save(ctx)
	require.NoError(t, err)
	planA, err := client.Plan.Create().
		SetNillableName(trans.Ptr("sqlite_usage_plan_a")).
		AddTenantIDs(tenantA.ID).
		Save(ctx)
	require.NoError(t, err, "建套餐并挂租户 A 应成功")
	require.NoError(t, client.PlanQuota.Create().SetPlanID(planA.ID).
		SetQuotaType(entPlanQuota.QuotaTypeUserLimit).SetQuotaValue(uint64(10)).Exec(ctx),
		"建 USER_LIMIT 配额应成功")
	require.NoError(t, client.PlanQuota.Create().SetPlanID(planA.ID).
		SetQuotaType(entPlanQuota.QuotaTypeStorage).SetQuotaValue(uint64(20)).Exec(ctx),
		"建 STORAGE 配额应成功")
	require.NoError(t, client.PlanQuota.Create().SetPlanID(planA.ID).
		SetQuotaType(entPlanQuota.QuotaTypeApiCall).SetQuotaValue(uint64(30)).Exec(ctx),
		"建 API_CALL 配额应成功")
	require.NoError(t, client.User.Create().SetUsername("sqlite_usage_user_a1").SetTenantID(tenantA.ID).Exec(ctx))
	require.NoError(t, client.User.Create().SetUsername("sqlite_usage_user_a2").SetTenantID(tenantA.ID).Exec(ctx))
	require.NoError(t, client.File.Create().SetCreatedAt(now).SetTenantID(tenantA.ID).SetSize(uint64(100)).Exec(ctx))
	require.NoError(t, client.File.Create().SetCreatedAt(now).SetTenantID(tenantA.ID).SetSize(uint64(200)).Exec(ctx))
	require.NoError(t, client.ApiAuditLog.Create().SetCreatedAt(now).SetTenantID(tenantA.ID).Exec(ctx))
	require.NoError(t, client.ApiAuditLog.Create().SetCreatedAt(now).SetTenantID(tenantA.ID).Exec(ctx))

	// 租户 B：无套餐，带 1 用户 / 1 文件(50) / 1 条 API 审计（验证聚合按租户隔离）
	tenantB, err := client.Tenant.Create().
		SetName("sqlite_usage_tenant_b").
		SetCode("SQLITE_USAGE_TENANT_B").
		Save(ctx)
	require.NoError(t, err)
	require.NoError(t, client.User.Create().SetUsername("sqlite_usage_user_b1").SetTenantID(tenantB.ID).Exec(ctx))
	require.NoError(t, client.File.Create().SetCreatedAt(now).SetTenantID(tenantB.ID).SetSize(uint64(50)).Exec(ctx))
	require.NoError(t, client.ApiAuditLog.Create().SetCreatedAt(now).SetTenantID(tenantB.ID).Exec(ctx))

	// 租户 A 的用量：套餐与配额回填 + 本租户聚合计数
	usageA, err := repo.GetUsage(ctx, tenantA.ID)
	require.NoError(t, err)
	require.Equal(t, tenantA.ID, usageA.GetTenantId())
	require.Equal(t, planA.ID, usageA.GetPlanId(), "PlanId 应回填关联套餐 ID")
	require.Equal(t, "sqlite_usage_plan_a", usageA.GetPlanName(), "PlanName 应回填套餐名")
	require.Len(t, usageA.GetQuotas(), 3, "应回填三类配额上限")
	quotaByType := map[int32]uint64{}
	for _, q := range usageA.GetQuotas() {
		quotaByType[int32(q.GetQuotaType())] = q.GetQuotaValue()
	}
	require.Equal(t, map[int32]uint64{
		int32(identityV1.PlanQuota_USER_LIMIT): 10,
		int32(identityV1.PlanQuota_STORAGE):    20,
		int32(identityV1.PlanQuota_API_CALL):   30,
	}, quotaByType, "配额上限应按 quota_type 映射回填（USER_LIMIT/STORAGE/API_CALL）")
	require.Equal(t, uint64(2), usageA.GetUserCount(), "租户 A 的用户计数应为 2")
	require.Equal(t, uint64(300), usageA.GetStorageUsedBytes(), "租户 A 的存储字节数应为 100+200=300")
	require.Equal(t, uint64(2), usageA.GetApiCallCount(), "租户 A 的 API 调用计数应为 2")

	// 租户 B 的用量：无套餐回填，聚合只含本租户数据（不串入 A 的数据）
	usageB, err := repo.GetUsage(ctx, tenantB.ID)
	require.NoError(t, err)
	require.Equal(t, tenantB.ID, usageB.GetTenantId())
	require.Zero(t, usageB.GetPlanId(), "租户 B 无套餐，PlanId 不应回填")
	require.Empty(t, usageB.GetQuotas(), "租户 B 无套餐，Quotas 应为空")
	require.Equal(t, uint64(1), usageB.GetUserCount(), "租户 B 的用户计数应只统计本租户的 1 个")
	require.Equal(t, uint64(50), usageB.GetStorageUsedBytes(), "租户 B 的存储字节数应只统计本租户的 50")
	require.Equal(t, uint64(1), usageB.GetApiCallCount(), "租户 B 的 API 调用计数应只统计本租户的 1 条")
}

// TestTenantUsageRepoSqlite_CleanupTenantData 验证清理：
// 租户的各表数据全部删除、租户记录保留且状态置 OFF。
func TestTenantUsageRepoSqlite_CleanupTenantData(t *testing.T) {
	repo := newTenantUsageRepoSqlite(t)
	client := repo.entClient.Client()
	ctx := enttest.NewSystemViewerCtx(context.Background())

	now := time.Now()
	tenantRow, err := client.Tenant.Create().
		SetName("sqlite_usage_tenant_cleanup").
		SetCode("SQLITE_USAGE_TENANT_CLEANUP").
		SetStatus(entTenant.StatusOn).
		Save(ctx)
	require.NoError(t, err)
	tid := tenantRow.ID

	// 给该租户播种带 tenant_id 的数据（跨文件/审计/用户表）
	require.NoError(t, client.File.Create().SetCreatedAt(now).SetTenantID(tid).SetSize(uint64(11)).Exec(ctx))
	require.NoError(t, client.ApiAuditLog.Create().SetCreatedAt(now).SetTenantID(tid).Exec(ctx))
	require.NoError(t, client.OperationAuditLog.Create().SetCreatedAt(now).SetTenantID(tid).Exec(ctx))
	require.NoError(t, client.User.Create().SetUsername("sqlite_usage_user_cleanup").SetTenantID(tid).Exec(ctx))

	// 无关租户的数据应保留
	otherTenant, err := client.Tenant.Create().
		SetName("sqlite_usage_tenant_keep").
		SetCode("SQLITE_USAGE_TENANT_KEEP").
		Save(ctx)
	require.NoError(t, err)
	require.NoError(t, client.File.Create().SetCreatedAt(now).SetTenantID(otherTenant.ID).SetSize(uint64(7)).Exec(ctx))

	require.NoError(t, repo.CleanupTenantData(ctx, tid), "清理租户数据应成功")

	fileCountByTenant := map[uint32]int{}
	fileRows, err := client.File.Query().All(ctx)
	require.NoError(t, err)
	for _, f := range fileRows {
		if f.TenantID != nil {
			fileCountByTenant[*f.TenantID]++
		}
	}
	require.Zero(t, fileCountByTenant[tid], "被清理租户的文件应全部删除")
	require.Equal(t, 1, fileCountByTenant[otherTenant.ID], "其它租户的文件应保留")

	apiCount, err := client.ApiAuditLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, apiCount, "API 审计表应只剩无租户归属的 0 行")
	opCount, err := client.OperationAuditLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, opCount, "操作审计表中该租户的行应被删除")
	userCount, err := client.User.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, userCount, "该租户的用户行应被删除")

	after, err := client.Tenant.Query().Where(entTenant.IDEQ(tid)).Only(ctx)
	require.NoError(t, err)
	require.NotNil(t, after.Status, "租户状态应非空")
	require.Equal(t, entTenant.StatusOff, *after.Status, "清理后租户状态应被置为 OFF（记录保留）")
}

// TestTenantUsageRepoSqlite_EnforceExpiryPolicies 验证到期策略扫描的各分支：
// BLOCK_LOGIN→EXPIRED、FREEZE→FREEZE（计入返回值）；
// READONLY / 无套餐 / 套餐未声明策略（默认 READONLY）保持 ON；
// 未到期与已停用租户不进入扫描。返回值为被改状态的租户数。
func TestTenantUsageRepoSqlite_EnforceExpiryPolicies(t *testing.T) {
	repo := newTenantUsageRepoSqlite(t)
	client := repo.entClient.Client()
	ctx := enttest.NewSystemViewerCtx(context.Background())

	past := time.Now().Add(-48 * time.Hour)
	future := time.Now().Add(48 * time.Hour)

	makeTenant := func(name, code string, status entTenant.Status, expiredAt time.Time) *ent.Tenant {
		row, err := client.Tenant.Create().
			SetName(name).
			SetCode(code).
			SetStatus(status).
			SetExpiredAt(expiredAt).
			Save(ctx)
		require.NoError(t, err, "建租户 %s 应成功", name)
		return row
	}
	linkPlan := func(tid uint32, policy *entPlan.ExpiryPolicy) {
		builder := client.Plan.Create().SetNillableName(
			trans.Ptr("sqlite_expiry_plan_" + strconv.FormatUint(uint64(tid), 10)))
		if policy != nil {
			builder = builder.SetExpiryPolicy(*policy)
		}
		_, err := builder.AddTenantIDs(tid).Save(ctx)
		require.NoError(t, err, "建套餐并挂租户 %d 应成功", tid)
	}

	blockLogin := entPlan.ExpiryPolicyBlockLogin
	freeze := entPlan.ExpiryPolicyFreeze
	readonly := entPlan.ExpiryPolicyReadonly

	ta := makeTenant("sqlite_expiry_a", "SQLITE_EXPIRY_A", entTenant.StatusOn, past)
	linkPlan(ta.ID, &blockLogin)
	tb := makeTenant("sqlite_expiry_b", "SQLITE_EXPIRY_B", entTenant.StatusOn, past)
	linkPlan(tb.ID, &freeze)
	tc := makeTenant("sqlite_expiry_c", "SQLITE_EXPIRY_C", entTenant.StatusOn, past)
	linkPlan(tc.ID, &readonly)
	td := makeTenant("sqlite_expiry_d", "SQLITE_EXPIRY_D", entTenant.StatusOn, past) // 无套餐
	te := makeTenant("sqlite_expiry_e", "SQLITE_EXPIRY_E", entTenant.StatusOn, future)
	linkPlan(te.ID, &blockLogin)
	tf := makeTenant("sqlite_expiry_f", "SQLITE_EXPIRY_F", entTenant.StatusOff, past)
	linkPlan(tf.ID, &blockLogin)
	tg := makeTenant("sqlite_expiry_g", "SQLITE_EXPIRY_G", entTenant.StatusOn, past) // 套餐未声明策略
	linkPlan(tg.ID, nil)

	enforced, err := repo.EnforceExpiryPolicies(ctx)
	require.NoError(t, err, "到期扫描应成功")
	require.Equal(t, 2, enforced, "仅 BLOCK_LOGIN 与 FREEZE 两个租户被改状态")

	statusOf := func(id uint32) entTenant.Status {
		row, err := client.Tenant.Query().Where(entTenant.IDEQ(id)).Only(ctx)
		require.NoError(t, err)
		require.NotNil(t, row.Status)
		return *row.Status
	}
	require.Equal(t, entTenant.StatusExpired, statusOf(ta.ID), "BLOCK_LOGIN 到期租户应被置为 EXPIRED")
	require.Equal(t, entTenant.StatusFreeze, statusOf(tb.ID), "FREEZE 到期租户应被置为 FREEZE")
	require.Equal(t, entTenant.StatusOn, statusOf(tc.ID), "READONLY 到期租户应保持 ON")
	require.Equal(t, entTenant.StatusOn, statusOf(td.ID), "无套餐到期租户应保持 ON")
	require.Equal(t, entTenant.StatusOn, statusOf(te.ID), "未到期租户应保持 ON")
	require.Equal(t, entTenant.StatusOff, statusOf(tf.ID), "已停用租户不进入扫描，应保持 OFF")
	require.Equal(t, entTenant.StatusOn, statusOf(tg.ID), "套餐未声明策略（默认 READONLY）应保持 ON")
}
