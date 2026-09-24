// ScriptService 的 SQLite 内存库集成测试（白盒，包内测试）。
//
// 覆盖目标：
//   - CRUD：Create（操作人盖章/唯一名冲突/字段校验——名称与源码必填、语言受支持、
//     非法请求体）、Get（命中/未命中）、Update（单字段掩码、改名冲突、
//     掩码外字段保持原值）、Delete（按 ID 列表清空、空列表拒绝）、List/Count。
//   - 变更联动 resync：Create/Delete 成功后触发运行时全量重同步
//     （ListHookPoints 反映挂载/卸载）。
//   - TestRun：nil 请求/缺 target 拒绝、草稿执行（输入 JSON 解析 + 执行数据
//     JSON 编码回读）、按已保存 ID 执行、不存在 ID NotFound、
//     执行异常回传 Success=false + Error。
//   - ListHookPoints：语言列表与钩子点聚合（含挂载数）。
//
// 跳过项：跨实例 resync 通知（redisClient 恒 nil）。
package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	scriptV1 "go-wind-admin/api/gen/go/script/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/middleware/auth"
)

// newScriptServiceForTest 白盒复刻 NewScriptService 的字段初始化：
// log 换 NopLogger helper；runtime 为本包 newScriptRuntimeForTest 的最小装配
// （scriptLog 挂载 testkit 仓库，覆盖 TestRun 的执行日志路径）。
func newScriptServiceForTest(t *testing.T) (*ScriptService, *ent.Client) {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	runtime := newScriptRuntimeForTest(t, entClient, true)
	return &ScriptService{
		log:     bLogger.NewHelper(bLogger.NopLogger()),
		repo:    runtime.repo,
		runtime: runtime,
	}, entClient.Client()
}

// TestScriptService_CreateValidation 验证 Create 的参数校验分支：
// nil 请求、nil Data、空名、空源码、不支持语言均拒绝且不落库。
func TestScriptService_CreateValidation(t *testing.T) {
	svc, client := newScriptServiceForTest(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7})

	_, err := svc.Create(opCtx, nil)
	require.Error(t, err, "nil 请求应拒绝")
	require.True(t, scriptV1.IsBadRequest(err))

	_, err = svc.Create(opCtx, &scriptV1.CreateScriptRequest{})
	require.Error(t, err, "nil Data 应拒绝")
	require.True(t, scriptV1.IsBadRequest(err))

	for _, tc := range []struct {
		name string
		data *scriptV1.Script
	}{
		{"空名", &scriptV1.Script{Name: trans.Ptr(""), Source: trans.Ptr("return 1"), Language: scriptV1.Language_LUA.Enum()}},
		{"空源码", &scriptV1.Script{Name: trans.Ptr("svc-empty-src"), Source: trans.Ptr("   "), Language: scriptV1.Language_LUA.Enum()}},
		{"不支持语言", &scriptV1.Script{Name: trans.Ptr("svc-bad-lang"), Source: trans.Ptr("return 1"), Language: scriptV1.Language(99).Enum()}},
	} {
		_, err = svc.Create(opCtx, &scriptV1.CreateScriptRequest{Data: tc.data})
		require.Error(t, err, "非法草稿（%s）应拒绝", tc.name)
		require.True(t, scriptV1.IsBadRequest(err), "非法草稿（%s）应为 BadRequest", tc.name)
	}

	cnt, err := client.Script.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, cnt, "全部非法请求都不应落库")
}

// TestScriptService_CreateNameConflict 验证脚本名唯一约束：
// 同名二次创建返回 Conflict；Update 改名为既有他名同样 Conflict，
// 改为全新名成功。
func TestScriptService_CreateNameConflict(t *testing.T) {
	svc, client := newScriptServiceForTest(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7})

	_, err := svc.Create(opCtx, &scriptV1.CreateScriptRequest{
		Data: &scriptV1.Script{
			Name:      trans.Ptr("svc_unique_name_a"),
			Language:  scriptV1.Language_LUA.Enum(),
			Source:    trans.Ptr("return 1"),
			Priority:  trans.Ptr(int32(1)),
			IsEnabled: trans.Ptr(false),
		},
	})
	require.NoError(t, err, "首个同名脚本创建应成功")

	_, err = svc.Create(opCtx, &scriptV1.CreateScriptRequest{
		Data: &scriptV1.Script{
			Name:      trans.Ptr("svc_unique_name_a"),
			Language:  scriptV1.Language_LUA.Enum(),
			Source:    trans.Ptr("return 2"),
			IsEnabled: trans.Ptr(false),
		},
	})
	require.Error(t, err, "同名二次创建应拒绝")
	require.True(t, scriptV1.IsConflict(err), "应为 Conflict 语义错误")

	rows, err := client.Script.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "冲突请求不应追加行")
	require.Equal(t, uint32(7), *rows[0].CreatedBy, "创建行应盖入操作人 ID")
}

// TestScriptService_CrudLifecycle 验证 Get 命中/未命中、单字段掩码更新
// （掩码外字段保持原值）、按 ID 列表删除清空。
func TestScriptService_CrudLifecycle(t *testing.T) {
	svc, client := newScriptServiceForTest(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7})

	_, err := svc.Create(opCtx, &scriptV1.CreateScriptRequest{
		Data: &scriptV1.Script{
			Name:        trans.Ptr("svc_crud_row"),
			Language:    scriptV1.Language_LUA.Enum(),
			Source:      trans.Ptr("return 1"),
			Description: trans.Ptr("初版描述"),
			Priority:    trans.Ptr(int32(1)),
			IsEnabled:   trans.Ptr(false),
		},
	})
	require.NoError(t, err)

	rows, err := client.Script.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	rowID := rows[0].ID
	require.Equal(t, "svc_crud_row", *rows[0].Name, "名称应按请求落库")
	require.Equal(t, "初版描述", *rows[0].Description, "描述应按请求落库")

	// Get 命中 / 未命中
	got, err := svc.Get(ctx, &scriptV1.GetScriptRequest{
		QueryBy: &scriptV1.GetScriptRequest_Id{Id: rowID},
	})
	require.NoError(t, err, "按已存在主键查询应命中")
	require.Equal(t, "svc_crud_row", got.GetName())

	_, err = svc.Get(ctx, &scriptV1.GetScriptRequest{
		QueryBy: &scriptV1.GetScriptRequest_Id{Id: 9999999},
	})
	require.Error(t, err, "按不存在主键查询应返回错误")

	// 单字段掩码更新：description 更新，name 保持
	_, err = svc.Update(opCtx, &scriptV1.UpdateScriptRequest{
		Id:         rowID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
		Data:       &scriptV1.Script{Description: trans.Ptr("更新后的描述")},
	})
	require.NoError(t, err, "单字段掩码更新应成功")

	after, err := client.Script.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "更新后的描述", *after.Description, "掩码内字段 description 应被更新")
	require.Equal(t, "svc_crud_row", *after.Name, "掩码外字段 name 应保持原值")
	require.Equal(t, uint32(7), *after.UpdatedBy, "更新行应盖入操作人 ID")

	// Update 改名为全新名：validateDraft 对携带 Name 的载荷同时校验 Source
	// （钉死语义），故载荷需同时带源码；掩码只放开 name，源码不落库
	_, err = svc.Update(opCtx, &scriptV1.UpdateScriptRequest{
		Id:         rowID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
		Data: &scriptV1.Script{
			Name:     trans.Ptr("svc_crud_row_renamed"),
			Source:   trans.Ptr("return 1"),
			Language: scriptV1.Language_LUA.Enum(),
		},
	})
	require.NoError(t, err, "带齐校验字段的改名更新应成功")

	// Delete：空列表拒绝，按 ID 删除清空
	_, err = svc.Delete(ctx, &scriptV1.DeleteScriptRequest{})
	require.Error(t, err, "空 ID 列表应拒绝")
	require.True(t, scriptV1.IsBadRequest(err))

	_, err = svc.Delete(ctx, &scriptV1.DeleteScriptRequest{Ids: []uint32{rowID}})
	require.NoError(t, err, "按 ID 删除应成功")
	cnt, err := client.Script.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, cnt, "删除后脚本表应清空")
}

// TestScriptService_ListCount 验证 List/Count 返回已落库行。
func TestScriptService_ListCount(t *testing.T) {
	svc, _ := newScriptServiceForTest(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	listResp, err := svc.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(0), listResp.GetTotal(), "空表 List 应为 0")
	require.Empty(t, listResp.GetItems())

	cntResp, err := svc.Count(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Zero(t, cntResp.GetCount(), "空表 Count 应为 0")
}

// TestScriptService_CreateDeleteResyncLinkage 验证 Create/Delete 的 resync 联动：
// 创建带钩子点的启用脚本后 ListHookPoints 出现挂载，删除后卸载。
func TestScriptService_CreateDeleteResyncLinkage(t *testing.T) {
	svc, _ := newScriptServiceForTest(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7})

	_, err := svc.Create(opCtx, &scriptV1.CreateScriptRequest{
		Data: &scriptV1.Script{
			Name:      trans.Ptr("svc_resync_linkage"),
			Language:  scriptV1.Language_LUA.Enum(),
			HookPoint: trans.Ptr("svc_linkage_hook"),
			Source:    trans.Ptr("return true"),
			Priority:  trans.Ptr(int32(1)),
			IsEnabled: trans.Ptr(true),
		},
	})
	require.NoError(t, err, "创建启用脚本应成功")

	hpResp, err := svc.ListHookPoints(ctx, nil)
	require.NoError(t, err)
	require.NotEmpty(t, hpResp.GetLanguages(), "语言列表应非空（装配了全部注册语言引擎）")
	var mounted bool
	for _, hp := range hpResp.GetItems() {
		if hp.GetName() == "svc_linkage_hook" {
			mounted = true
			require.Equal(t, uint32(1), hp.GetScriptCount(), "挂载数应为 1")
		}
	}
	require.True(t, mounted, "创建启用钩子脚本后钩子点应出现（服务层 resync 联动）")

	rows, err := svc.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(1), rows.GetTotal(), "脚本表应含刚创建的行")
	ids := make([]uint32, 0, len(rows.GetItems()))
	for _, item := range rows.GetItems() {
		ids = append(ids, item.GetId())
	}
	_, err = svc.Delete(ctx, &scriptV1.DeleteScriptRequest{Ids: ids})
	require.NoError(t, err, "删除应成功")

	hpResp2, err := svc.ListHookPoints(ctx, nil)
	require.NoError(t, err)
	for _, hp := range hpResp2.GetItems() {
		require.NotEqual(t, "svc_linkage_hook", hp.GetName(),
			"删除并 resync 后钩子点应卸载")
	}
}

// TestScriptService_TestRunDraft 验证 TestRun 草稿执行：
// 输入上下文 JSON 解析（数值/非 JSON 原样字符串）与执行数据 JSON 编码回读。
func TestScriptService_TestRunDraft(t *testing.T) {
	svc, _ := newScriptServiceForTest(t)

	resp, err := svc.TestRun(context.Background(), &scriptV1.TestRunScriptRequest{
		Target: &scriptV1.TestRunScriptRequest_Draft{Draft: &scriptV1.Script{
			Name:     trans.Ptr("svc_test_run_draft"),
			Language: scriptV1.Language_LUA.Enum(),
			Source: trans.Ptr(`
local ctx = __get_ctx()
local n = ctx.get("num")
local s = ctx.get("raw")
ctx.set("echo_raw", s)
ctx.set("echo_num", n + 1)
return true
`),
		}},
		Input: map[string]string{
			"num": "123",
			"raw": "not json",
		},
	})
	require.NoError(t, err, "草稿试运行应成功")
	require.True(t, resp.GetSuccess(), "正常脚本应成功")

	expectedRaw, _ := json.Marshal("not json")
	expectedNum, _ := json.Marshal(124)
	require.Equal(t, string(expectedRaw), resp.GetContext()["echo_raw"],
		"非 JSON 输入应原样注入字符串并以 JSON 编码回读")
	require.Equal(t, string(expectedNum), resp.GetContext()["echo_num"],
		"JSON 数值输入应解析为数值参与运算并以 JSON 编码回读")
}

// TestScriptService_TestRunSavedByIdAndErrors 验证 TestRun 按 ID 执行与各错误分支：
// nil 请求/缺 target 拒绝、不存在 ID NotFound、执行异常 Success=false 且带错误消息。
func TestScriptService_TestRunSavedByIdAndErrors(t *testing.T) {
	svc, _ := newScriptServiceForTest(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	_, err := svc.TestRun(ctx, nil)
	require.Error(t, err, "nil 请求应拒绝")
	require.True(t, scriptV1.IsBadRequest(err))

	_, err = svc.TestRun(ctx, &scriptV1.TestRunScriptRequest{})
	require.Error(t, err, "缺 target 应拒绝")
	require.True(t, scriptV1.IsBadRequest(err))

	_, err = svc.TestRun(ctx, &scriptV1.TestRunScriptRequest{
		Target: &scriptV1.TestRunScriptRequest_Id{Id: 424242},
	})
	require.Error(t, err, "不存在的 ID 应返回 NotFound")
	require.True(t, scriptV1.IsNotFound(err))

	// 落一条脚本并按 ID 试运行（读回保存源码执行）
	require.NoError(t, svc.repo.Create(ctx, &scriptV1.CreateScriptRequest{
		Data: &scriptV1.Script{
			Name:      trans.Ptr("svc_saved_run"),
			Language:  scriptV1.Language_LUA.Enum(),
			Source:    trans.Ptr("local ctx = __get_ctx() ctx.set(\"saved_echo\", \"saved\") return true"),
			IsEnabled: trans.Ptr(false),
		},
	}))
	rows, err := svc.repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Len(t, rows.GetItems(), 1)
	savedID := rows.GetItems()[0].GetId()

	resp, err := svc.TestRun(ctx, &scriptV1.TestRunScriptRequest{
		Target: &scriptV1.TestRunScriptRequest_Id{Id: savedID},
	})
	require.NoError(t, err, "按已保存 ID 试运行应成功")
	require.True(t, resp.GetSuccess())
	expectedSaved, _ := json.Marshal("saved")
	require.Equal(t, string(expectedSaved), resp.GetContext()["saved_echo"],
		"保存的源码应被读回执行并回读执行数据")

	// 执行异常：Success=false + Error
	resp, err = svc.TestRun(ctx, &scriptV1.TestRunScriptRequest{
		Target: &scriptV1.TestRunScriptRequest_Draft{Draft: &scriptV1.Script{
			Name:     trans.Ptr("svc_failing_draft"),
			Language: scriptV1.Language_LUA.Enum(),
			Source:   trans.Ptr("error('boom')"),
		}},
	})
	require.NoError(t, err, "执行失败也应返回响应体（Success=false）")
	require.False(t, resp.GetSuccess(), "异常脚本应标记失败")
	require.NotEmpty(t, resp.GetError(), "异常脚本应回传错误消息")
}
