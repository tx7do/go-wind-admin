// ScriptLogService 的 SQLite 内存库集成测试（白盒，包内测试）。
//
// 覆盖目标：
//   - List / Count 对已落库执行日志的返回（经 ScriptLogRepo.Record 落行的
//     字段回读：脚本名/触发方式/成败）。
//   - Purge：nil 请求拒绝；显式未来时间点清理近期行；显式过去时间点与
//     省略 Before（默认 90 天前）不清理近期行。
package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/timestamppb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	scriptV1 "go-wind-admin/api/gen/go/script/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/middleware/auth"
)

// newScriptLogServiceForTest 白盒复刻 NewScriptLogService 的字段初始化
// （log 换 NopLogger helper；repo 用 testkit 构造器）。
func newScriptLogServiceForTest(t *testing.T) (*ScriptLogService, *ent.Client) {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	svc := &ScriptLogService{
		log:  bLogger.NewHelper(bLogger.NopLogger()),
		repo: data.NewScriptLogRepoForTest(entClient),
	}
	return svc, entClient.Client()
}

// recordLogRows 经 repo 落两条执行日志（一成一败）。
func recordLogRows(t *testing.T, svc *ScriptLogService, ctx context.Context) {
	t.Helper()
	require.NotPanics(t, func() {
		svc.repo.Record(ctx, data.ScriptLogRecord{
			ScriptName: "logsvc_success_row",
			Language:   "lua",
			Trigger:    "test_run",
			Version:    1,
			Success:    true,
			DurationMS: 5,
		})
		svc.repo.Record(ctx, data.ScriptLogRecord{
			ScriptName: "logsvc_failure_row",
			Language:   "lua",
			Trigger:    "test_run",
			Version:    1,
			Success:    false,
			DurationMS: 7,
			Error:      "boom",
		})
	}, "Record 为尽力而为落库，不应 panic")
}

// TestScriptLogService_ListAndCount 验证 List/Count 返回已落库的执行日志
// 且关键字段回读正确。
func TestScriptLogService_ListAndCount(t *testing.T) {
	svc, _ := newScriptLogServiceForTest(t)
	ctx := enttest.NewSystemContext(context.Background())

	recordLogRows(t, svc, ctx)

	listResp, err := svc.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), listResp.GetTotal(), "应统计全部 2 条日志")
	require.Len(t, listResp.GetItems(), 2)
	nameSet := map[string]bool{}
	for _, item := range listResp.GetItems() {
		require.Equal(t, "lua", item.GetLanguage(), "语言字段应回读")
		require.Equal(t, "test_run", item.GetTriggerType(), "触发方式字段应回读")
		nameSet[item.GetScriptName()] = true
	}
	require.True(t, nameSet["logsvc_success_row"] && nameSet["logsvc_failure_row"],
		"两条日志的脚本名都应回读")

	cntResp, err := svc.Count(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), cntResp.GetCount(), "计数应等于落库行数")
}

// TestScriptLogService_Purge 验证 Purge 的守卫与钳制（P2 修复后口径）：
// nil 请求拒绝、非平台管理员拒绝（sys_script_logs 全平台共表，Purge 是审计
// 清理动作）、未来时间点被钳制为"至少保留最近 1 天"（近期行不可一键清空）、
// 过去时间点与默认 90 天阈值不清理近期行。
func TestScriptLogService_Purge(t *testing.T) {
	svc, _ := newScriptLogServiceForTest(t)
	ctx := enttest.NewSystemContext(context.Background())
	adminCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{
		UserId:          1,
		IsPlatformAdmin: trans.Ptr(true),
	})

	_, err := svc.Purge(ctx, nil)
	require.Error(t, err, "nil 请求应拒绝")
	require.True(t, scriptV1.IsBadRequest(err))

	// 平台闸门：无操作者上下文拒绝
	_, err = svc.Purge(ctx, &scriptV1.PurgeScriptLogsRequest{})
	require.Error(t, err, "无操作者上下文的清理应被平台闸门拒绝")

	// 平台闸门：非平台管理员拒绝
	plainCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 2})
	_, err = svc.Purge(plainCtx, &scriptV1.PurgeScriptLogsRequest{})
	require.Error(t, err, "非平台管理员的清理应被平台闸门拒绝")

	// 过去时间点：近期行不受影响
	recordLogRows(t, svc, ctx)
	resp, err := svc.Purge(adminCtx, &scriptV1.PurgeScriptLogsRequest{
		Before: timestamppb.New(time.Now().Add(-24 * time.Hour)),
	})
	require.NoError(t, err)
	require.Zero(t, resp.GetDeleted(), "过去时间点不应清理近期行")

	// 省略 Before：默认 90 天前阈值，同样不清理近期行
	resp, err = svc.Purge(adminCtx, &scriptV1.PurgeScriptLogsRequest{})
	require.NoError(t, err)
	require.Zero(t, resp.GetDeleted(), "默认 90 天阈值不应清理近期行")
	cnt, err := svc.repo.CountLog(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(2), cnt.GetCount(), "两次 Purge 后行数应保持 2")

	// 未来时间点：被钳制为"至少保留最近 1 天"，近期行不可一键清空（P2 修复）。
	// 注：此前口径是未来 before 全量清理——sys_script_logs 是钩子/任务审计痕迹，
	// 不允许一键清空全平台台账。
	resp, err = svc.Purge(adminCtx, &scriptV1.PurgeScriptLogsRequest{
		Before: timestamppb.New(time.Now().AddDate(10, 0, 0)),
	})
	require.NoError(t, err)
	require.Zero(t, resp.GetDeleted(), "未来时间点应被钳制，近期行保留")
	cnt, err = svc.repo.CountLog(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(2), cnt.GetCount(), "钳制后近期行应全部保留")
}
