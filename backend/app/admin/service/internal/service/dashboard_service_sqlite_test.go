// DashboardService 的 SQLite 内存库测试（白盒，包内测试）。
//
// 首页统计是"把库里的数字原样端给老板"的链路，测试盯两件事：
//   - 聚合口径：哪些行该计入（今日 LOGIN、近 N 天窗口）哪些不该（LOGOUT、窗口外、
//     昨日登录），缺日补零与日期升序——口径错一天，老板看到的报表就错一天；
//   - AI 洞察的权限闸：无操作人上下文拒绝、租户用户拒绝（明细是全平台数据且会
//     外发模型端点，只有平台管理员可看）。
//
// 跳过项：GetAiInsights 平台管理员主路径——其依赖的三个 repo 查询
// （UserBehavior24h/LoginFailAccounts/SensitiveOps24h）是 PG 专属原生 SQL
// （AT TIME ZONE / $1 占位符），SQLite 内存库跑不了，属已知的测试基建边界。
package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	entCrud "github.com/tx7do/go-crud/entgo"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/loginauditlog"
	"go-wind-admin/app/admin/service/internal/data/ent/operationauditlog"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/middleware/auth"
)

// newDashboardServiceForTest 白盒装配 DashboardService：log 换 NopLogger，
// repo 走 SQLite 内存库；scriptRuntime 用零值（GetAiInsights 的守卫分支到不了 LLM 调用）。
func newDashboardServiceForTest(t *testing.T) (*DashboardService, *entCrud.EntClient[*ent.Client]) {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	return &DashboardService{
		log:           bLogger.NewHelper(bLogger.NopLogger()),
		dashboardRepo: data.NewDashboardRepoForTest(entClient),
		scriptRuntime: &ScriptRuntime{},
	}, entClient
}

// dashboardTodayZero 返回本地时区今日零点（与 data.startOfToday 同口径）。
func dashboardTodayZero() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

// seedDashboardUser 造一个最小用户（username 是唯一必填列）。
func seedDashboardUser(t *testing.T, client *ent.Client, ctx context.Context, username string) {
	t.Helper()
	require.NoError(t, client.User.Create().SetUsername(username).Exec(ctx),
		"播种用户 %s 应成功", username)
}

// seedDashboardLogin 造一行登录审计（action_type/status 显式给，created_at 决定口径归属）。
func seedDashboardLogin(t *testing.T, client *ent.Client, ctx context.Context, at time.Time, actionType loginauditlog.ActionType, status loginauditlog.Status, username string) {
	t.Helper()
	require.NoError(t, client.LoginAuditLog.Create().
		SetCreatedAt(at).
		SetActionType(actionType).
		SetStatus(status).
		SetUsername(username).
		Exec(ctx), "播种登录审计应成功")
}

// seedDashboardOperation 造一行操作审计。
func seedDashboardOperation(t *testing.T, client *ent.Client, ctx context.Context, at time.Time, action operationauditlog.Action, username string, success bool) {
	t.Helper()
	require.NoError(t, client.OperationAuditLog.Create().
		SetCreatedAt(at).
		SetAction(action).
		SetUsername(username).
		SetSuccess(success).
		Exec(ctx), "播种操作审计应成功")
}

// TestDashboardServiceSqlite_GetOverview 验证四张概览卡的计数口径：
// 用户/角色全量计数；今日登录只数今天的 LOGIN（LOGOUT 不算、昨天的 LOGIN 不算）；
// 今日操作按 created_at >= 今日零点。
func TestDashboardServiceSqlite_GetOverview(t *testing.T) {
	svc, entClient := newDashboardServiceForTest(t)
	ctx := enttest.NewSystemContext(context.Background())

	seedDashboardUser(t, entClient.Client(), ctx, "用户甲")
	seedDashboardUser(t, entClient.Client(), ctx, "用户乙")
	require.NoError(t, entClient.Client().Role.Create().
		SetName("角色甲").SetCode("role-a").Exec(ctx), "播种角色应成功")

	today := dashboardTodayZero()
	// 今日 LOGIN ×2（计入）、今日 LOGOUT（不计入）、昨日 LOGIN（不计入）
	seedDashboardLogin(t, entClient.Client(), ctx, today.Add(time.Hour), loginauditlog.ActionTypeLogin, loginauditlog.StatusSuccess, "用户甲")
	seedDashboardLogin(t, entClient.Client(), ctx, today.Add(2*time.Hour), loginauditlog.ActionTypeLogin, loginauditlog.StatusFailed, "用户乙")
	seedDashboardLogin(t, entClient.Client(), ctx, today.Add(3*time.Hour), loginauditlog.ActionTypeLogout, loginauditlog.StatusSuccess, "用户甲")
	seedDashboardLogin(t, entClient.Client(), ctx, today.Add(-24*time.Hour), loginauditlog.ActionTypeLogin, loginauditlog.StatusSuccess, "用户甲")
	// 今日操作 ×1、昨日操作 ×1（不计入）
	seedDashboardOperation(t, entClient.Client(), ctx, today.Add(time.Hour), operationauditlog.ActionCreate, "用户甲", true)
	seedDashboardOperation(t, entClient.Client(), ctx, today.Add(-24*time.Hour), operationauditlog.ActionDelete, "用户甲", true)

	resp, err := svc.GetOverview(ctx, nil)
	require.NoError(t, err, "空库外的常规聚合应成功")
	require.Equal(t, uint32(2), resp.GetUserCount(), "用户计数应为 2")
	require.Equal(t, uint32(1), resp.GetRoleCount(), "角色计数应为 1")
	require.Equal(t, uint32(2), resp.GetTodayLoginCount(), "今日登录只数 LOGIN 两行：LOGOUT 与昨日 LOGIN 不算")
	require.Equal(t, uint32(1), resp.GetTodayOperationCount(), "今日操作只数今天的一行")
}

// TestDashboardServiceSqlite_GetLoginTrend 验证近 N 天趋势：
// 按日期升序、无记录日期补零、只数 LOGIN（LOGOUT 不算）、窗口外不算、days<=0 落 7 天默认。
func TestDashboardServiceSqlite_GetLoginTrend(t *testing.T) {
	svc, entClient := newDashboardServiceForTest(t)
	ctx := enttest.NewSystemContext(context.Background())

	today := dashboardTodayZero()
	seed := func(offsetDays int, count int, actionType loginauditlog.ActionType) {
		for i := 0; i < count; i++ {
			seedDashboardLogin(t, entClient.Client(), ctx, today.AddDate(0, 0, -offsetDays).Add(time.Duration(i)*time.Minute),
				actionType, loginauditlog.StatusSuccess, "用户甲")
		}
	}
	seed(0, 2, loginauditlog.ActionTypeLogin)  // 今天 ×2
	seed(1, 1, loginauditlog.ActionTypeLogin)  // 昨天 ×1
	seed(3, 3, loginauditlog.ActionTypeLogin)  // 3 天前 ×3
	seed(0, 1, loginauditlog.ActionTypeLogout) // 今天 LOGOUT（不计入）
	seed(8, 4, loginauditlog.ActionTypeLogin)  // 8 天前（窗口外）

	resp, err := svc.GetLoginTrend(ctx, &adminV1.GetLoginTrendRequest{Days: int32Ptr(7)})
	require.NoError(t, err, "趋势查询应成功")
	points := resp.GetPoints()
	require.Len(t, points, 7, "应返回 7 个点（缺日补零）")
	// 升序：points[0] 最旧（6 天前），points[6] 今天
	require.Equal(t, today.AddDate(0, 0, -6).Format("2006-01-02"), points[0].GetDate(), "首点应为 6 天前")
	require.Equal(t, today.Format("2006-01-02"), points[6].GetDate(), "末点应为今天")
	require.Equal(t, uint32(3), points[3].GetCount(), "3 天前应有 3 次")
	require.Equal(t, uint32(1), points[5].GetCount(), "昨天应有 1 次")
	require.Equal(t, uint32(2), points[6].GetCount(), "今天应有 2 次（LOGOUT 不算）")
	sum := uint32(0)
	for _, p := range points {
		sum += p.GetCount()
	}
	require.Equal(t, uint32(6), sum, "窗口外（8 天前 ×4）与 LOGOUT 都不该计入")

	// days<=0 落默认 7 天
	resp, err = svc.GetLoginTrend(ctx, &adminV1.GetLoginTrendRequest{Days: int32Ptr(0)})
	require.NoError(t, err)
	require.Len(t, resp.GetPoints(), 7, "days=0 应落默认 7 天窗口")
}

// TestDashboardServiceSqlite_Distributions 验证两个分布接口：
// 按枚举值分组计数、service 层把分组列透传成 Label。
func TestDashboardServiceSqlite_Distributions(t *testing.T) {
	svc, entClient := newDashboardServiceForTest(t)
	ctx := enttest.NewSystemContext(context.Background())

	now := time.Now()
	seedDashboardOperation(t, entClient.Client(), ctx, now, operationauditlog.ActionCreate, "用户甲", true)
	seedDashboardOperation(t, entClient.Client(), ctx, now, operationauditlog.ActionCreate, "用户乙", false)
	seedDashboardOperation(t, entClient.Client(), ctx, now, operationauditlog.ActionDelete, "用户甲", true)

	actionDist, err := svc.GetOperationActionDistribution(ctx, nil)
	require.NoError(t, err)
	actionByLabel := map[string]uint32{}
	for _, it := range actionDist.GetItems() {
		actionByLabel[it.GetLabel()] = it.GetCount()
	}
	require.Len(t, actionByLabel, 2, "只有 CREATE 与 DELETE 两种动作")
	require.Equal(t, uint32(2), actionByLabel["CREATE"])
	require.Equal(t, uint32(1), actionByLabel["DELETE"])

	seedDashboardLogin(t, entClient.Client(), ctx, now, loginauditlog.ActionTypeLogin, loginauditlog.StatusSuccess, "用户甲")
	seedDashboardLogin(t, entClient.Client(), ctx, now, loginauditlog.ActionTypeLogin, loginauditlog.StatusSuccess, "用户乙")
	seedDashboardLogin(t, entClient.Client(), ctx, now, loginauditlog.ActionTypeLogin, loginauditlog.StatusFailed, "用户丙")

	statusDist, err := svc.GetLoginStatusDistribution(ctx, nil)
	require.NoError(t, err)
	statusByLabel := map[string]uint32{}
	for _, it := range statusDist.GetItems() {
		statusByLabel[it.GetLabel()] = it.GetCount()
	}
	require.Len(t, statusByLabel, 2)
	require.Equal(t, uint32(2), statusByLabel["SUCCESS"])
	require.Equal(t, uint32(1), statusByLabel["FAILED"])
}

// TestDashboardServiceSqlite_GetAiInsightsGuards 验证 AI 洞察的权限闸：
// 无操作人上下文拒绝；租户用户拒绝（明细是全平台数据且会外发模型端点）。
// 平台管理员主路径依赖 PG 专属原生 SQL，SQLite 内存库不覆盖（见文件头说明）。
func TestDashboardServiceSqlite_GetAiInsightsGuards(t *testing.T) {
	svc, _ := newDashboardServiceForTest(t)
	ctx := enttest.NewSystemContext(context.Background())

	// 无操作人上下文
	_, err := svc.GetAiInsights(ctx, &adminV1.AiInsightsRequest{})
	require.Error(t, err, "缺操作人上下文应被拒绝")

	// 租户用户（tenant != 0）
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 9, TenantId: int32PtrU32(7)})
	_, err = svc.GetAiInsights(opCtx, &adminV1.AiInsightsRequest{})
	require.Error(t, err, "租户用户应被平台闸拒绝")
	require.Contains(t, err.Error(), "platform-only", "拒绝文案要指明平台专属")
}

// TestDashboardService_SplitInsightPoints 验证模型要点文本的结构化拆分：
// 三种列表前缀与数字序号剥离、单行按句号拆分、最多 5 条、空文本回空。
func TestDashboardService_SplitInsightPoints(t *testing.T) {
	// 破折号列表
	require.Equal(t, []string{"第一条", "第二条"}, splitInsightPoints("- 第一条\n- 第二条"))
	// 星号与圆点前缀
	require.Equal(t, []string{"甲", "乙"}, splitInsightPoints("* 甲\n• 乙"))
	// 数字序号（半角点与顿号）
	require.Equal(t, []string{"first", "second"}, splitInsightPoints("1. first\n2、second"))
	// 序号两位数也能剥；单行时句号兜底接管且不得把同一内容重复出两份
	require.Equal(t, []string{"tenth。"}, splitInsightPoints("10. tenth"))
	// 单条列表项：兜底后只剩句号形态一份
	require.Equal(t, []string{"第一条。"}, splitInsightPoints("- 第一条"))
	// 无列表前缀的单行：按句号拆分并保留句号
	require.Equal(t, []string{"句子一。", "句子二。"}, splitInsightPoints("句子一。句子二。"))
	// 多行时不再做句号拆分（行即条目）
	require.Equal(t, []string{"行一，没句号", "行二。"}, splitInsightPoints("行一，没句号\n行二。"))
	// 超过 5 条截断
	require.Len(t, splitInsightPoints("1\n2\n3\n4\n5\n6\n7"), 5, "最多保留 5 条")
	// 空文本
	require.Empty(t, splitInsightPoints(""))
}

// int32Ptr 造 *int32（GetLoginTrendRequest.Days 是可选标量）。
func int32Ptr(v int32) *int32 {
	return &v
}

// int32PtrU32 造 *int64（UserTokenPayload.TenantId 是可选标量）。
func int32PtrU32(v int64) *int64 {
	return &v
}
