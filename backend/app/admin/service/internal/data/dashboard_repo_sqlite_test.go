package data

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	entLoginAuditLog "go-wind-admin/app/admin/service/internal/data/ent/loginauditlog"
	entOperationAuditLog "go-wind-admin/app/admin/service/internal/data/ent/operationauditlog"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newDashboardRepoSqlite 用 enttest helper 构造 DashboardRepo，
// 逐字段复刻 NewDashboardRepo（log 换 NopLogger；该 repo 无 init()）。
func newDashboardRepoSqlite(t *testing.T) *DashboardRepo {
	t.Helper()
	return &DashboardRepo{
		entClient: enttest.NewEntClientForTest(t),
		log:       bLogger.NewHelper(bLogger.NopLogger()),
	}
}

// TestDashboardRepoSqlite_CountEmptyTables 验证空库时
// CountActiveUsers / CountRoles / CountTodayLogins / CountTodayOperations 均为 0。
func TestDashboardRepoSqlite_CountEmptyTables(t *testing.T) {
	repo := newDashboardRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	n, err := repo.CountActiveUsers(ctx)
	require.NoError(t, err)
	require.Zero(t, n, "空库 CountActiveUsers 应为 0")

	n, err = repo.CountRoles(ctx)
	require.NoError(t, err)
	require.Zero(t, n, "空库 CountRoles 应为 0")

	n, err = repo.CountTodayLogins(ctx)
	require.NoError(t, err)
	require.Zero(t, n, "空库 CountTodayLogins 应为 0")

	n, err = repo.CountTodayOperations(ctx)
	require.NoError(t, err)
	require.Zero(t, n, "空库 CountTodayOperations 应为 0")
}

// TestDashboardRepoSqlite_CountUsersAndRoles 验证
// CountActiveUsers / CountRoles 的全表计数语义。
func TestDashboardRepoSqlite_CountUsersAndRoles(t *testing.T) {
	repo := newDashboardRepoSqlite(t)
	client := repo.entClient.Client()
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, client.User.Create().SetUsername("sqlite_dash_user_a").Exec(ctx))
	require.NoError(t, client.User.Create().SetUsername("sqlite_dash_user_b").Exec(ctx))
	require.NoError(t, client.Role.Create().SetName("sqlite_dash_role").SetCode("SQLITE_DASH_ROLE").Exec(ctx))

	n, err := repo.CountActiveUsers(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, n, "CountActiveUsers 应统计全部 2 个用户")

	n, err = repo.CountRoles(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, n, "CountRoles 应统计全部 1 个角色")
}

// TestDashboardRepoSqlite_CountTodayLoginsAndOperations 验证当日计数：
// 只统计 action_type=LOGIN 的登录审计行（LOGOUT 行不计入）与全部操作审计行。
func TestDashboardRepoSqlite_CountTodayLoginsAndOperations(t *testing.T) {
	repo := newDashboardRepoSqlite(t)
	client := repo.entClient.Client()
	ctx := enttest.NewSystemViewerCtx(context.Background())

	now := time.Now()
	for i := 0; i < 2; i++ {
		require.NoError(t, client.LoginAuditLog.Create().
			SetCreatedAt(now).
			SetActionType(entLoginAuditLog.ActionTypeLogin).
			Exec(ctx), "直插 LOGIN 登录审计行应成功")
	}
	require.NoError(t, client.LoginAuditLog.Create().
		SetCreatedAt(now).
		SetActionType(entLoginAuditLog.ActionTypeLogout).
		Exec(ctx), "直插 LOGOUT 登录审计行应成功")
	require.NoError(t, client.OperationAuditLog.Create().SetCreatedAt(now).Exec(ctx), "直插操作审计行应成功")
	require.NoError(t, client.OperationAuditLog.Create().SetCreatedAt(now).Exec(ctx), "直插操作审计行应成功")

	n, err := repo.CountTodayLogins(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, n, "今日登录计数应只统计 action_type=LOGIN 的 2 行（LOGOUT 不计入）")

	n, err = repo.CountTodayOperations(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, n, "今日操作计数应为全部 2 行操作审计")
}

// TestDashboardRepoSqlite_LoginTrend 验证按日分桶：
// 桶数恒等于 days（days<=0 时默认 7）、日期自最早到今日升序、无记录日期为 0，
// 且只统计 action_type=LOGIN 的行（LOGOUT 不计入）。
func TestDashboardRepoSqlite_LoginTrend(t *testing.T) {
	repo := newDashboardRepoSqlite(t)
	client := repo.entClient.Client()
	ctx := enttest.NewSystemViewerCtx(context.Background())

	now := time.Now()
	for i := 0; i < 2; i++ {
		require.NoError(t, client.LoginAuditLog.Create().
			SetCreatedAt(now).
			SetActionType(entLoginAuditLog.ActionTypeLogin).
			Exec(ctx), "直插 LOGIN 登录审计行应成功")
	}
	require.NoError(t, client.LoginAuditLog.Create().
		SetCreatedAt(now).
		SetActionType(entLoginAuditLog.ActionTypeLogout).
		Exec(ctx), "直插 LOGOUT 登录审计行应成功")

	// days=3：3 个桶，前两日为 0，今日为 2（仅 LOGIN 行）
	trend, err := repo.LoginTrend(ctx, 3)
	require.NoError(t, err)
	require.Len(t, trend, 3, "days=3 应返回 3 个桶")
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	for i, row := range trend {
		require.Equal(t, today.AddDate(0, 0, -(2-i)).Format("2006-01-02"), row.Date,
			"第 %d 个桶的日期应自最早日升序到今日", i)
		expected := 0
		if i == 2 {
			expected = 2
		}
		require.Equal(t, expected, row.Count, "第 %d 个桶的计数应为 %d", i, expected)
	}

	// days=0：默认 7 天窗口，今日桶为 2、其余为 0
	trend, err = repo.LoginTrend(ctx, 0)
	require.NoError(t, err)
	require.Len(t, trend, 7, "days<=0 应默认 7 天窗口")
	for i, row := range trend {
		require.Equal(t, today.AddDate(0, 0, -(6-i)).Format("2006-01-02"), row.Date,
			"第 %d 个桶的日期应自最早日升序到今日", i)
		expected := 0
		if i == 6 {
			expected = 2
		}
		require.Equal(t, expected, row.Count, "第 %d 个桶的计数应为 %d", i, expected)
	}
}

// TestDashboardRepoSqlite_LoginTrendNoData 验证空数据时全部桶为 0。
func TestDashboardRepoSqlite_LoginTrendNoData(t *testing.T) {
	repo := newDashboardRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	trend, err := repo.LoginTrend(ctx, 4)
	require.NoError(t, err)
	require.Len(t, trend, 4)
	for _, row := range trend {
		require.Zero(t, row.Count, "无数据时所有桶应为 0")
	}
}

// TestDashboardRepoSqlite_OperationActionDistribution 验证操作审计按 action
// 分组的聚合计数。
func TestDashboardRepoSqlite_OperationActionDistribution(t *testing.T) {
	repo := newDashboardRepoSqlite(t)
	client := repo.entClient.Client()
	ctx := enttest.NewSystemViewerCtx(context.Background())

	now := time.Now()
	for i := 0; i < 2; i++ {
		require.NoError(t, client.OperationAuditLog.Create().
			SetCreatedAt(now).
			SetAction(entOperationAuditLog.ActionCreate).
			Exec(ctx), "直插 action=CREATE 行应成功")
	}
	require.NoError(t, client.OperationAuditLog.Create().
		SetCreatedAt(now).
		SetAction(entOperationAuditLog.ActionRead).
		Exec(ctx), "直插 action=READ 行应成功")

	rows, err := repo.OperationActionDistribution(ctx)
	require.NoError(t, err)
	actual := map[string]int{}
	for _, row := range rows {
		actual[row.Action] += row.Count
	}
	require.Equal(t, map[string]int{"CREATE": 2, "READ": 1}, actual,
		"action 分布应按落库值聚合计数")
}

// TestDashboardRepoSqlite_LoginStatusDistribution 验证登录审计按 status
// 分组的聚合计数。
func TestDashboardRepoSqlite_LoginStatusDistribution(t *testing.T) {
	repo := newDashboardRepoSqlite(t)
	client := repo.entClient.Client()
	ctx := enttest.NewSystemViewerCtx(context.Background())

	now := time.Now()
	for i := 0; i < 2; i++ {
		require.NoError(t, client.LoginAuditLog.Create().
			SetCreatedAt(now).
			SetStatus(entLoginAuditLog.StatusSuccess).
			Exec(ctx), "直插 status=SUCCESS 行应成功")
	}
	require.NoError(t, client.LoginAuditLog.Create().
		SetCreatedAt(now).
		SetStatus(entLoginAuditLog.StatusFailed).
		Exec(ctx), "直插 status=FAILED 行应成功")

	rows, err := repo.LoginStatusDistribution(ctx)
	require.NoError(t, err)
	actual := map[string]int{}
	for _, row := range rows {
		actual[row.Status] += row.Count
	}
	require.Equal(t, map[string]int{"SUCCESS": 2, "FAILED": 1}, actual,
		"status 分布应按落库值聚合计数")
}
