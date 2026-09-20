// script_entity_hooks.go 的集成测试（白盒，包内测试）：
// ent 全局 mutation 钩子桥（before 同步可否决 / after 异步只读旁路）
// 与 ScriptRuntime 的实体钩子执行入口（InvokeEntityHook / InvokeEntityHookVeto）。
//
// 覆盖目标：
//   - mutationOpName 的 Op→操作名映射（create / update 组合 / delete / deleteOne /
//     default 分支的 ToLower(String()) 语义钉死）。
//   - mutationID 的尽力提取（带 ID() 的 mutation 取值，无 ID() 的取 0）。
//   - AttachEntityHooks 桥：
//       · 映射实体（Tenant）before 钩子同步触发，vetoInvoker 返回错误时
//         写入被拒且错误包裹 scripting.ErrScriptVetoed（errors.Is 可判）；
//       · vetoInvoker panic 时 fail-open（写入放行）；
//       · 写入成功后 after 钩子异步触发（独立 goroutine + 超时上下文），
//         payload 携带 entity/op/id；
//       · 非映射实体（Language）两类钩子均不触发；
//       · 双 nil invoker 时全部跳过，写入正常。
//   - 生产接线形态（vetoInvoker=InvokeEntityHookVeto / after=InvokeEntityHook）：
//       · 挂载 __stop 脚本的 before 钩子否决写入（ErrScriptVetoed）；
//       · 挂载良性脚本的 after 钩子执行并落 script_log（trigger=hook）。
//   - InvokeEntityHook/InvokeEntityHookVeto 的无挂载返回 nil（after/before 两侧），
//     以及"挂载脚本执行失败仍上抛 + 落 hook 执行审计"。
//
// 跳过项：after 钩子 invoker 自身 panic 的兜底恢复（仅进程存活可观察，断言不成立）。
package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"

	dictV1 "go-wind-admin/api/gen/go/dict/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	entTenant "go-wind-admin/app/admin/service/internal/data/ent/tenant"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/scripting"
)

// ── 纯函数直测 ──────────────────────────────────────────────────────────────

// TestMutationOpName 钉死 mutationOpName 的 Op→操作名映射。
// 注：OpUpdate/OpUpdateOne 单独值走 default 分支（ToLower(String())），
// OpUpdate|OpUpdateOne 组合值命中 "update"——生产对单值与组合值不同形，
// 此处按现状钉死（OpUpdateOne 单值产出 "updateone"）。
func TestMutationOpName(t *testing.T) {
	// 注：entgo 的 Op 位值为 OpCreate=1、OpUpdate=2、OpUpdateOne=4、OpDelete=8、
	// OpDeleteOne=16。生产 switch 只命中 OpCreate / (OpUpdate|OpUpdateOne)=6 /
	// OpDelete / OpDeleteOne 四个值；单值 OpUpdate/OpUpdateOne 与一切其他组合
	// 走 default（ToLower(String())）——单值得到 "opupdate"/"opupdateone"，
	// 未知组合得到数值名 "op(N)"。本表按现状钉死该不对称语义。
	cases := []struct {
		op   ent.Op
		want string
	}{
		{ent.OpCreate, "create"},
		{ent.OpUpdate | ent.OpUpdateOne, "update"},
		{ent.OpDelete, "delete"},
		{ent.OpDeleteOne, "delete"},
		{ent.OpUpdate, "opupdate"},               // default 分支：单值 OpUpdate 不命中组合 case
		{ent.OpUpdateOne, "opupdateone"},         // default 分支：单值 OpUpdateOne 不命中组合 case
		{ent.OpCreate | ent.OpDelete, "op(9)"},   // default 分支：未知组合的数值名
	}
	for _, c := range cases {
		require.Equal(t, c.want, mutationOpName(c.op), "Op(%d) 的映射", c.op)
	}
}

// idMutationStub 携带 ID() 的假 mutation（嵌入接口满足 ent.Mutation，
// 未覆写方法一旦被调用即 nil 接口 panic，本用例只调用 ID）。
type idMutationStub struct {
	ent.Mutation
	id  uint32
	has bool
}

func (s *idMutationStub) ID() (uint32, bool) { return s.id, s.has }

// noIDMutationStub 无 ID() 实现的假 mutation。
type noIDMutationStub struct {
	ent.Mutation
}

// TestMutationID 验证 mutationID 的尽力提取：带 ID() 的取其值，否则 0。
func TestMutationID(t *testing.T) {
	require.Equal(t, uint32(77), mutationID(&idMutationStub{id: 77, has: true}),
		"带 ID() 的 mutation 应提取其 ID")
	require.Equal(t, uint32(0), mutationID(&idMutationStub{has: false}),
		"ID() 不存在值时提取 0")
	require.Equal(t, uint32(0), mutationID(&noIDMutationStub{}),
		"未实现 ID() 的 mutation 提取 0")
}

// ── 钩子桥行为（stub invoker） ─────────────────────────────────────────────

// hookProbe 记录钩子调用的探针。
type hookProbe struct {
	before chan string
	after  chan map[string]any
}

func newHookProbe() *hookProbe {
	return &hookProbe{
		before: make(chan string, 8),
		after:  make(chan map[string]any, 8),
	}
}

// drainAfterNonBlocking 在限定窗口内收取一次 after 探针事件；
// timeoutMS 内无事件返回 nil（用于非映射实体的阴性断言）。
func (p *hookProbe) drainAfterNonBlocking(timeoutMS int) map[string]any {
	select {
	case ev := <-p.after:
		return ev
	case <-time.After(time.Duration(timeoutMS) * time.Millisecond):
		return nil
	}
}

// TestEntityHooks_VetoBlocksMappedEntity 验证映射实体（Tenant）的 before 钩子：
// vetoInvoker 返回错误时写入被拒，错误包裹 scripting.ErrScriptVetoed，且无行落库。
func TestEntityHooks_VetoBlocksMappedEntity(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	probe := newHookProbe()
	AttachEntityHooks(entClient.Client(),
		func(_ context.Context, _ string, _ map[string]any) {},
		func(_ context.Context, hookPoint string, _ map[string]any) error {
			probe.before <- hookPoint
			return errors.New("stub veto")
		})
	ctx := enttest.NewSystemViewerCtx(context.Background())

	_, err := entClient.Client().Tenant.Create().
		SetName("hook-veto-tenant").
		SetCode("HOOK_VETO_TENANT").
		SetStatus(entTenant.StatusOn).
		SetType(entTenant.TypeTrial).
		SetAuditStatus(entTenant.AuditStatusApproved).
		Save(ctx)
	require.Error(t, err, "veto 应拒绝写入")
	require.ErrorIs(t, err, scripting.ErrScriptVetoed, "拒绝错误必须可经 errors.Is 判定为脚本否决")
	select {
	case pt := <-probe.before:
		require.Equal(t, "tenant.before_create", pt, "before 钩子点应为 tenant.before_create")
	default:
		t.Fatal("before 钩子应同步触发")
	}
	cnt, err := entClient.Client().Tenant.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, cnt, "被否决的写入不应落库")
}

// TestEntityHooks_VetoPanicFailOpen 验证 vetoInvoker panic 时 fail-open：
// 写入放行、钩子自身异常不阻断业务。
func TestEntityHooks_VetoPanicFailOpen(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	AttachEntityHooks(entClient.Client(), nil, func(_ context.Context, _ string, _ map[string]any) error {
		panic("veto invoker panic")
	})
	ctx := enttest.NewSystemViewerCtx(context.Background())

	created, err := entClient.Client().Tenant.Create().
		SetName("hook-failopen-tenant").
		SetCode("HOOK_FAILOPEN_TENANT").
		SetStatus(entTenant.StatusOn).
		SetType(entTenant.TypeTrial).
		SetAuditStatus(entTenant.AuditStatusApproved).
		Save(ctx)
	require.NoError(t, err, "veto invoker panic 应 fail-open 放行写入")
	require.NotNil(t, created, "写入应成功返回实体")
}

// TestEntityHooks_AfterFiresAsyncOnSuccess 验证写入成功后 after 钩子异步触发：
// payload 携带 entity/op，且在写入调用返回后才到达（goroutine 旁路）。
func TestEntityHooks_AfterFiresAsyncOnSuccess(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	probe := newHookProbe()
	AttachEntityHooks(entClient.Client(),
		func(_ context.Context, _ string, payload map[string]any) {
			probe.after <- payload
		}, nil)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	_, err := entClient.Client().Tenant.Create().
		SetName("hook-after-tenant").
		SetCode("HOOK_AFTER_TENANT").
		SetStatus(entTenant.StatusOn).
		SetType(entTenant.TypeTrial).
		SetAuditStatus(entTenant.AuditStatusApproved).
		Save(ctx)
	require.NoError(t, err, "无 veto 的写入应成功")

	deadline := time.Now().Add(5 * time.Second)
	var payload map[string]any
	for payload == nil && time.Now().Before(deadline) {
		payload = probe.drainAfterNonBlocking(100)
	}
	require.NotNil(t, payload, "after 钩子应在写入成功后异步触发")
	require.Equal(t, "Tenant", payload["entity"], "payload 应携带映射实体名")
	require.Equal(t, "create", payload["op"], "payload 应携带操作名")
	require.Contains(t, payload, "id", "payload 应携带尽力提取的实体 ID 键")
}

// TestEntityHooks_UnmappedEntityNoHooks 验证非映射实体（Language）：
// 写入成功但两类钩子均不触发。
func TestEntityHooks_UnmappedEntityNoHooks(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	probe := newHookProbe()
	AttachEntityHooks(entClient.Client(),
		func(_ context.Context, _ string, payload map[string]any) {
			probe.after <- payload
		},
		func(_ context.Context, _ string, _ map[string]any) error {
			return nil
		})
	ctx := enttest.NewSystemViewerCtx(context.Background())

	languageRepo := data.NewLanguageRepoForTest(entClient)
	err := languageRepo.Create(ctx, &dictV1.CreateLanguageRequest{
		Data: &dictV1.Language{
			LanguageName: trans.Ptr("钩子桥未映射语言"),
			LanguageCode: trans.Ptr("HOOKBRIDGE_LANG"),
			NativeName:   trans.Ptr("native"),
		},
	})
	require.NoError(t, err, "非映射实体写入应成功")

	require.Nil(t, probe.drainAfterNonBlocking(300), "非映射实体不应触发 after 钩子")
	rows, err := entClient.Client().Language.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, rows, "非映射实体写入应正常落库")
}

// TestEntityHooks_NilInvokersNoop 验证双 nil invoker：全部跳过，写入正常。
func TestEntityHooks_NilInvokersNoop(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	AttachEntityHooks(entClient.Client(), nil, nil)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	created, err := entClient.Client().Tenant.Create().
		SetName("hook-nilinv-tenant").
		SetCode("HOOK_NILINV_TENANT").
		SetStatus(entTenant.StatusOn).
		SetType(entTenant.TypeTrial).
		SetAuditStatus(entTenant.AuditStatusApproved).
		Save(ctx)
	require.NoError(t, err, "nil invoker 不应阻断写入")
	require.NotNil(t, created, "写入应成功返回实体")
}

// ── 生产接线形态（runtime 注入） ────────────────────────────────────────────

// TestEntityHooks_WiringVetoEndToEnd 生产接线形态的端到端否决：
// 挂载 __stop 脚本的 before 钩子经 InvokeEntityHookVeto 否决写入。
func TestEntityHooks_WiringVetoEndToEnd(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	r := newScriptRuntimeForTest(t, entClient, false)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	// 挂载否决脚本（before 钩子点）并重同步
	createScriptRow(t, r, ctx, "wiring_veto_script", "tenant.before_create", "__stop('wiring veto')", true)
	require.NoError(t, r.Resync(ctx), "Resync 应成功")

	AttachEntityHooks(entClient.Client(),
		func(hookCtx context.Context, hookPoint string, payload map[string]any) {
			_ = r.InvokeEntityHook(hookPoint, payload)
		},
		func(hookCtx context.Context, hookPoint string, payload map[string]any) error {
			return r.InvokeEntityHookVeto(hookPoint, payload)
		})

	_, err := entClient.Client().Tenant.Create().
		SetName("wiring-veto-tenant").
		SetCode("WIRING_VETO_TENANT").
		SetStatus(entTenant.StatusOn).
		SetType(entTenant.TypeTrial).
		SetAuditStatus(entTenant.AuditStatusApproved).
		Save(ctx)
	require.Error(t, err, "挂载否决脚本的实体钩子应拒绝写入")
	require.ErrorIs(t, err, scripting.ErrScriptVetoed, "应为脚本否决语义错误")
	cnt, qErr := entClient.Client().Tenant.Query().Count(ctx)
	require.NoError(t, qErr)
	require.Zero(t, cnt, "否决的写入不应落库")
}

// TestEntityHooks_WiringAfterLogsExecution 生产接线形态的 after 钩子：
// 良性挂载脚本异步执行并落 script_log（trigger=hook）。
func TestEntityHooks_WiringAfterLogsExecution(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	r := newScriptRuntimeForTest(t, entClient, true)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	createScriptRow(t, r, ctx, "wiring_after_script", "tenant.after_create", "return true", true)
	require.NoError(t, r.Resync(ctx), "Resync 应成功")

	AttachEntityHooks(entClient.Client(),
		func(_ context.Context, hookPoint string, payload map[string]any) {
			_ = r.InvokeEntityHook(hookPoint, payload)
		},
		func(_ context.Context, hookPoint string, payload map[string]any) error {
			return r.InvokeEntityHookVeto(hookPoint, payload)
		})

	_, err := entClient.Client().Tenant.Create().
		SetName("wiring-after-tenant").
		SetCode("WIRING_AFTER_TENANT").
		SetStatus(entTenant.StatusOn).
		SetType(entTenant.TypeTrial).
		SetAuditStatus(entTenant.AuditStatusApproved).
		Save(ctx)
	require.NoError(t, err, "未挂否决脚本的写入应成功")

	// after 钩子经 goroutine 异步执行并落审计日志，轮询等待
	require.Eventually(t, func() bool {
		rows, qerr := entClient.Client().ScriptLog.Query().All(ctx)
		if qerr != nil {
			return false
		}
		for _, row := range rows {
			if row.TriggerType != nil && *row.TriggerType == "hook" &&
				row.HookPoint != nil && *row.HookPoint == "tenant.after_create" {
				return true
			}
		}
		return false
	}, 10*time.Second, 100*time.Millisecond, "after 钩子执行应落 hook 触发方式的执行日志")
}

// TestScriptRuntime_InvokeEntityHook_NoMount 验证无挂载时的两个入口都不算失败：
// after 侧「这个钩子点上没脚本」是常态（每次实体变更都会问一遍），返回 nil；
// before 侧同理不视为否决（nil）。两条一起钉住，是为了让"不报错"成为契约而不是一次偶然。
func TestScriptRuntime_InvokeEntityHook_NoMount(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	r := newScriptRuntimeForTest(t, entClient, false)

	require.NoError(t, r.InvokeEntityHook("tenant.after_create", map[string]any{}),
		"无挂载的 after 钩子点应视为无事可做（nil），不再返回 NotFound")
	require.NoError(t, r.InvokeEntityHookVeto("tenant.before_create", map[string]any{}),
		"无挂载的 before 钩子点不视为否决（返回 nil）")
}

// TestScriptRuntime_InvokeEntityHook_MountedFailureStillErrors 钉住"只消噪、不消错"：
// 真挂载了脚本且脚本执行失败时，InvokeEntityHook 必须把错误上抛（调用方才有 ERROR 日志/审计）。
// 脚本用运行时 error 而非语法错误——语法错误会在 Resync 阶段加载失败被跳过，
// 走到的是"无挂载"分支，测不到这里想测的那一支。
func TestScriptRuntime_InvokeEntityHook_MountedFailureStillErrors(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	r := newScriptRuntimeForTest(t, entClient, true)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	createScriptRow(t, r, ctx, "s2_failing_after_script", "tenant.after_create", "error('s2 boom')", true)
	require.NoError(t, r.Resync(ctx), "Resync 应成功")

	err := r.InvokeEntityHook("tenant.after_create", map[string]any{"entity": "Tenant"})
	require.Error(t, err, "挂载脚本执行失败必须上抛，不能被'无挂载返回 nil'一并吞掉")
	require.ErrorContains(t, err, "s2_failing_after_script", "错误应指名失败的脚本")

	// 失败一次也要落一条执行审计（logExecution 的 err 分支），否则"消噪"顺带消掉了可追查痕迹
	rows, qErr := entClient.Client().ScriptLog.Query().All(ctx)
	require.NoError(t, qErr)
	var logged bool
	for _, row := range rows {
		if row.TriggerType != nil && *row.TriggerType == "hook" &&
			row.HookPoint != nil && *row.HookPoint == "tenant.after_create" {
			require.NotNil(t, row.Error, "失败执行应记下错误信息")
			require.NotEmpty(t, *row.Error)
			logged = true
		}
	}
	require.True(t, logged, "钩子失败应落 hook 触发方式的执行日志")
}
