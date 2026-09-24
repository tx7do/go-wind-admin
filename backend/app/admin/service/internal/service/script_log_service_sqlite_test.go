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

	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/timestamppb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	scriptV1 "go-wind-admin/api/gen/go/script/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/enttest"
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
	ctx := enttest.NewSystemViewerCtx(context.Background())

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

// TestScriptLogService_Purge 验证 Purge 的三分支：
// nil 请求拒绝、未来时间点全量清理、过去时间点与默认 90 天阈值不清理近期行。
func TestScriptLogService_Purge(t *testing.T) {
	svc, _ := newScriptLogServiceForTest(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	_, err := svc.Purge(ctx, nil)
	require.Error(t, err, "nil 请求应拒绝")
	require.True(t, scriptV1.IsBadRequest(err))

	// 过去时间点：近期行不受影响
	recordLogRows(t, svc, ctx)
	resp, err := svc.Purge(ctx, &scriptV1.PurgeScriptLogsRequest{
		Before: timestamppb.New(time.Now().Add(-24 * time.Hour)),
	})
	require.NoError(t, err)
	require.Zero(t, resp.GetDeleted(), "过去时间点不应清理近期行")

	// 省略 Before：默认 90 天前阈值，同样不清理近期行
	resp, err = svc.Purge(ctx, &scriptV1.PurgeScriptLogsRequest{})
	require.NoError(t, err)
	require.Zero(t, resp.GetDeleted(), "默认 90 天阈值不应清理近期行")
	cnt, err := svc.repo.CountLog(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(2), cnt.GetCount(), "两次 Purge 后行数应保持 2")

	// 未来时间点：近期行全量清理。
	// 注：取 +10 年而非 +1 小时——SQLite 的时间列按各时区格式化成字符串做
	// 词法比较，timestamppb.AsTime 归一为 UTC 后小时段在不同时区下词法序不稳定，
	// 年份段在任何时区下词法序稳定（2036… > 2026…），跨时区可复现。
	resp, err = svc.Purge(ctx, &scriptV1.PurgeScriptLogsRequest{
		Before: timestamppb.New(time.Now().AddDate(10, 0, 0)),
	})
	require.NoError(t, err)
	require.Equal(t, uint64(2), resp.GetDeleted(), "远未来时间点应清理全部近期行")
	cnt, err = svc.repo.CountLog(ctx)
	require.NoError(t, err)
	require.Zero(t, cnt.GetCount(), "清理后日志表应清空")
}
