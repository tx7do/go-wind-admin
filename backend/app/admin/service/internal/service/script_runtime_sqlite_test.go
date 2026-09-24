// ScriptRuntime 的 SQLite 内存库集成测试（白盒，包内测试）。
//
// 覆盖目标：
//   - 引擎装配与查询：EngineFor 的大小写不敏感命中/未命中、Languages、HookPoints
//     （初始为空 / Resync 后反映挂载 / disabled 不挂载）、ScriptTaskHandlers
//     （注册表副本）、Close 幂等清空。
//   - 任务处理器桥：AttachScriptTaskRegistrar + RegisterScriptTaskSubscriber 的
//     固定分发类型注册、nil registrar 跳过；RunScriptTaskHandler 的
//     未注册 NotFound / 无引擎 InternalServerError / 经 Resync 注册的真实执行闭环
//     （含必填参数校验失败）。
//   - Resync：挂载钩子点脚本（enabled/disabled）、自注册脚本（task.register_handler）
//     的加载与处理器归属、坏脚本（语法错误）计入失败不中断整体。
//   - TestRun：一次性沙箱执行——上下文注入与执行数据回读（脚本经 __get_ctx()
//     返回的上下文表 get/set）、不支持的语言拒绝、__stop 中止回传错误。
//   - logExecution 审计：scriptLog 仓库挂载时执行日志落库（trigger=task/test_run）。
//   - NotifyResync / StartResyncListener / StopResyncListener 的 redisClient=nil no-op 分支。
//
// 跳过项：跨实例 Redis pub/sub 真链路（redisClient 恒 nil，含订阅循环重建退避）、
// OSS/Redis 模块注入（SetRedis/SetOSS 传 nil 与生产一致但无真连接）。
package service

import (
	"context"
	"encoding/json"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	entCrud "github.com/tx7do/go-crud/entgo"

	gsEngine "github.com/tx7do/go-scripts"

	scriptV1 "go-wind-admin/api/gen/go/script/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/scripting"
	"go-wind-admin/pkg/task"
)

// newScriptRuntimeForTest 白盒复刻 NewScriptRuntime 的字段初始化与引擎装配循环：
// log/logger 换 NopLogger、redisClient/ossClient 恒 nil（SetRedis/SetOSS(nil) 与
// 生产对齐），scriptLog 按需挂载 testkit 仓库；Close 注册 t.Cleanup 关停引擎。
func newScriptRuntimeForTest(t *testing.T, entClient *entCrud.EntClient[*ent.Client], withLogRepo bool) *ScriptRuntime {
	t.Helper()
	r := &ScriptRuntime{
		engines:          make(map[gsEngine.Type]*scripting.Engine),
		cfg:              scripting.DefaultConfig(),
		repo:             data.NewScriptRepoForTest(entClient),
		log:              bLogger.NewHelper(bLogger.NopLogger()),
		logger:           bLogger.NopLogger(),
		taskHandlerOwner: make(map[string]gsEngine.Type),
		scriptLog:        nil,
		redisClient:      nil,
	}
	if withLogRepo {
		r.scriptLog = data.NewScriptLogRepoForTest(entClient)
	}
	r.cfg.HTTPOptions = scripting.HTTPAllowlistFromEnv()
	for _, typ := range scripting.SupportedTypes() {
		cfg := *r.cfg
		cfg.EngineType = typ
		cfg.ScriptDir = ""
		eng := scripting.NewEngine(&cfg, r.logger)
		eng.SetRedis(nil)
		eng.SetOSS(nil)
		r.engines[typ] = eng
	}
	t.Cleanup(r.Close)
	return r
}

// createScriptRow 经 repo 落一条 lua 脚本记录（测试内唯一命名）。
func createScriptRow(t *testing.T, r *ScriptRuntime, ctx context.Context, name, hookPoint, source string, enabled bool) {
	t.Helper()
	err := r.repo.Create(ctx, &scriptV1.CreateScriptRequest{
		Data: &scriptV1.Script{
			Name:      trans.Ptr(name),
			Language:  scriptV1.Language_LUA.Enum(),
			HookPoint: trans.Ptr(hookPoint),
			Source:    trans.Ptr(source),
			Priority:  trans.Ptr(int32(1)),
			IsEnabled: trans.Ptr(enabled),
		},
	})
	require.NoError(t, err, "写入脚本 %s 应成功", name)
}

// TestScriptRuntime_EngineFor_CaseInsensitive 验证 EngineFor 的大小写不敏感命中
// 与未命中（lua/javascript 两种注册语言均命中，未知语言返回 nil）。
func TestScriptRuntime_EngineFor_CaseInsensitive(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	r := newScriptRuntimeForTest(t, entClient, false)

	lower, upper := r.EngineFor("lua"), r.EngineFor("LUA")
	require.NotNil(t, lower, "小写 lua 应命中引擎")
	require.NotNil(t, upper, "大写 LUA 应命中引擎")
	require.Same(t, lower, upper, "大小写应命中同一引擎实例")

	require.NotNil(t, r.EngineFor("JavaScript"), "javascript 应命中引擎")
	require.Nil(t, r.EngineFor("python"), "未注册语言应返回 nil")
	require.Nil(t, r.EngineFor(""), "空语言应返回 nil")
}

// TestScriptRuntime_Languages 验证 Languages 返回全部已装配引擎的语言小写字符串
// （与 scripting.SupportedTypes 集合一致；顺序不稳定故按集合比较）。
func TestScriptRuntime_Languages(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	r := newScriptRuntimeForTest(t, entClient, false)

	got := r.Languages()
	want := scripting.SupportedTypes()
	sort.Strings(got)
	wantStrs := make([]string, 0, len(want))
	for _, typ := range want {
		wantStrs = append(wantStrs, string(typ))
	}
	sort.Strings(wantStrs)
	require.Equal(t, wantStrs, got, "Languages 应逐一无遗漏返回已装配引擎的语言标识")
}

// TestScriptRuntime_HookPoints_EmptyThenMounted 验证 HookPoints 聚合：
// 初始无脚本为空；Resync 挂载钩子脚本后出现对应钩子点且 ScriptCount=1。
func TestScriptRuntime_HookPoints_EmptyThenMounted(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	r := newScriptRuntimeForTest(t, entClient, false)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.Empty(t, r.HookPoints(), "初始状态无任何钩子点挂载")

	createScriptRow(t, r, ctx, "rt_hook_mounted", "rt_test_hook", "return true", true)
	require.NoError(t, r.Resync(ctx), "Resync 应成功")

	found := false
	for _, hp := range r.HookPoints() {
		if hp.Name == "rt_test_hook" {
			found = true
			require.Equal(t, 1, hp.ScriptCount, "挂载脚本数应为 1")
			require.Equal(t, 0, hp.CallbackCount, "无 hook.register 自注册回调")
		}
	}
	require.True(t, found, "Resync 后 rt_test_hook 应出现在 HookPoints 中")
}

// TestScriptRuntime_Resync_SkipsDisabled 验证 disabled 脚本不进 HookPoints。
func TestScriptRuntime_Resync_SkipsDisabled(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	r := newScriptRuntimeForTest(t, entClient, false)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	createScriptRow(t, r, ctx, "rt_hook_disabled", "rt_disabled_hook", "return true", false)
	require.NoError(t, r.Resync(ctx), "Resync 应成功（disabled 行被 ListEnabledScripts 排除）")
	require.Empty(t, r.HookPoints(), "disabled 脚本不应挂载任何钩子点")
}

// TestScriptRuntime_Resync_BadScriptCountedFailed 验证语法错误的自注册脚本
// 计入失败但不中断整体同步（好脚本仍挂载成功）。
func TestScriptRuntime_Resync_BadScriptCountedFailed(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	r := newScriptRuntimeForTest(t, entClient, false)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	createScriptRow(t, r, ctx, "rt_good_mounted", "rt_good_hook", "return true", true)
	createScriptRow(t, r, ctx, "rt_bad_syntax", "", "this is (not lua", true)
	require.NoError(t, r.Resync(ctx), "单个坏脚本不应让整体 Resync 失败")

	seen := false
	for _, hp := range r.HookPoints() {
		if hp.Name == "rt_good_hook" {
			seen = true
		}
	}
	require.True(t, seen, "坏脚本之外的正常脚本应照常挂载")
}

// TestScriptRuntime_TaskRegistrar_And_OwnerlessHandler 验证：
//   - nil registrar 时 RegisterScriptTaskSubscriber 为 no-op；
//   - AttachScriptTaskRegistrar 后 RegisterScriptTaskSubscriber 以固定分发类型
//     task.ScriptTaskDispatchType 注册回调；
//   - RunScriptTaskHandler 对未注册处理器名返回 NotFound；
//   - taskHandlerOwner 指向无引擎语言时返回 InternalServerError。
func TestScriptRuntime_TaskRegistrar_And_OwnerlessHandler(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	r := newScriptRuntimeForTest(t, entClient, false)

	// nil registrar：no-op，不 panic
	r.RegisterScriptTaskSubscriber()

	var gotType string
	require.NoError(t, r.AttachScriptTaskRegistrar(func(taskType string, fn func(taskType string, data *task.ScriptTaskData) error) error {
		gotType = taskType
		// 模拟 asynq 派发到未注册处理器名：RunScriptTaskHandler 应 NotFound
		if err := fn(taskType, &task.ScriptTaskData{Handler: "rt_nonexistent_handler", Params: map[string]any{}}); err != nil {
			require.True(t, scriptV1.IsNotFound(err), "未注册处理器应映射为 NotFound")
		} else {
			t.Fatal("未注册处理器名应返回错误")
		}
		return nil
	}), "AttachScriptTaskRegistrar 应成功")
	r.RegisterScriptTaskSubscriber()
	require.Equal(t, task.ScriptTaskDispatchType, gotType,
		"订阅注册必须使用固定分发类型 task.ScriptTaskDispatchType")

	// 归属指向已装配引擎之外：InternalServerError
	r.mu.Lock()
	r.taskHandlerOwner["rt_orphan_handler"] = gsEngine.Type("python")
	r.mu.Unlock()
	err := r.RunScriptTaskHandler(context.Background(), "rt_orphan_handler", nil)
	require.Error(t, err, "无引擎可用的处理器应返回错误")
	require.True(t, scriptV1.IsInternalServerError(err), "应为 InternalServerError 语义错误")
}

// TestScriptRuntime_TaskHandler_FullRoundTrip 经 Resync 自注册（task.register_handler）
// 的处理器闭环：注册表副本可见、归属引擎记录、RunScriptTaskHandler 缺必填参数报错、
// 参数齐全执行成功、scriptLog 挂载时执行日志落库（trigger=task）。
func TestScriptRuntime_TaskHandler_FullRoundTrip(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	r := newScriptRuntimeForTest(t, entClient, true)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	createScriptRow(t, r, ctx, "rt_task_reg", "", `
local task = require "task"
task.register_handler("rt_demo_handler", "runtime test handler", function(params)
	return true
end, {
	required = { "bucket" },
	optional = {}
})
`, true)
	require.NoError(t, r.Resync(ctx), "Resync 应成功")

	// 注册表副本可见
	handled := false
	for _, h := range r.ScriptTaskHandlers() {
		if h.Name == "rt_demo_handler" {
			handled = true
		}
	}
	require.True(t, handled, "ScriptTaskHandlers 应包含经 Resync 注册的处理器条目")

	// 归属引擎已记录
	r.mu.RLock()
	owner, ok := r.taskHandlerOwner["rt_demo_handler"]
	r.mu.RUnlock()
	require.True(t, ok, "Resync 后处理器归属应被记录")
	require.Equal(t, gsEngine.LuaType, owner, "lua 脚本注册的处理器归属 lua 引擎")

	// 缺必填参数：报错
	err := r.RunScriptTaskHandler(context.Background(), "rt_demo_handler", map[string]any{})
	require.Error(t, err, "缺必填参数 bucket 应报错")

	// 参数齐全：执行成功，且执行日志落库（trigger=task）
	require.NoError(t, r.RunScriptTaskHandler(context.Background(), "rt_demo_handler", map[string]any{"bucket": "bkt"}),
		"参数齐全时处理器应执行成功")

	rows, err := entClient.Client().ScriptLog.Query().All(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, rows, "scriptLog 挂载时任务处理器执行应落执行日志")
	for _, row := range rows {
		require.Equal(t, "task", *row.TriggerType, "执行日志的触发方式应为 task")
		require.Equal(t, task.ScriptTaskDispatchType, *row.HookPoint, "执行日志的钩子点列应为固定分发类型名")
		require.Equal(t, "lua", *row.Language, "执行日志语言列应为 lua")
	}
}

// TestScriptRuntime_TestRun_SandboxRoundTrip 验证 TestRun 一次性沙箱：
// 上下文注入（数值/字符串）、脚本经上下文表 get/set 的执行数据回读、
// 不支持的语言拒绝、__stop 中止回传错误。
func TestScriptRuntime_TestRun_SandboxRoundTrip(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	r := newScriptRuntimeForTest(t, entClient, true)

	// 注入数值与字符串上下文，脚本读出并回写派生数据
	data, err := r.TestRun(context.Background(), "lua", "", `
local ctx = __get_ctx()
local n = ctx.get("num")
local s = ctx.get("raw")
ctx.set("echo_raw", s)
ctx.set("echo_num", n + 1)
return true
`, map[string]any{
		"num": 123,
		"raw": "not json",
	})
	require.NoError(t, err, "正常草稿脚本应执行成功")

	require.Equal(t, "not json", data["echo_raw"], "字符串上下文应原样注入并回读")
	require.Equal(t, "124", mustJSON(t, data["echo_num"]),
		"数值上下文应参与运算后回读（JSON 编码）")

	// 不支持的语言
	_, err = r.TestRun(context.Background(), "python", "x", "return 1", nil)
	require.Error(t, err, "不支持的应返回错误")
	require.True(t, scriptV1.IsBadRequest(err), "不支持的应为 BadRequest 语义")

	// 异常脚本：错误回传（含停止原因）
	_, err = r.TestRun(context.Background(), "LUA", "rt_bad", "__stop('sandbox stop')", nil)
	require.Error(t, err, "__stop 应中止执行并回传错误")
	require.ErrorContains(t, err, "sandbox stop", "错误应透传停止原因")

	// 两次 TestRun（成功一次+中止一次）各落一条执行日志（trigger=test_run）
	ctx := enttest.NewSystemViewerCtx(context.Background())
	rows, err := entClient.Client().ScriptLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2, "两次 TestRun 各应落一条执行日志")
	sawSuccess, sawFailure := false, false
	for _, row := range rows {
		require.Equal(t, "test_run", *row.TriggerType, "TestRun 的日志触发方式应为 test_run")
		if *row.Success {
			sawSuccess = true
		} else {
			sawFailure = true
			require.Contains(t, *row.Error, "sandbox stop", "失败日志应记录停止原因")
		}
	}
	require.True(t, sawSuccess && sawFailure, "成功与中止两次执行都应被记录")
}

// mustJSON 把执行数据值编成 JSON 字符串（与生产回读路径同一编码器）。
func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}

// TestScriptRuntime_Close_Idempotent 验证 Close 关停引擎并清空注册表、幂等。
func TestScriptRuntime_Close_Idempotent(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	r := newScriptRuntimeForTest(t, entClient, false)
	require.NotEmpty(t, r.Languages(), "关停前应有已装配引擎")

	r.Close()
	require.Empty(t, r.Languages(), "Close 后引擎表应清空")
	r.Close() // 幂等：第二次不 panic
}

// TestScriptRuntime_NotifyResync_NilRedisNoop 验证 redisClient 恒 nil 时
// NotifyResync / StartResyncListener / StopResyncListener 的 no-op 分支不 panic。
func TestScriptRuntime_NotifyResync_NilRedisNoop(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	r := newScriptRuntimeForTest(t, entClient, false)

	require.NotPanics(t, func() {
		r.NotifyResync(context.Background())
		r.StartResyncListener(context.Background())
		r.StopResyncListener()
		r.StopResyncListener() // 幂等
	}, "redisClient 为 nil 时监听/通知路径应为 no-op")
	require.Nil(t, r.redisClient, "测试装配恒为 nil redis")
}
