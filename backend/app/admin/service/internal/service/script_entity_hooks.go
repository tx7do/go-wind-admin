package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	scriptV1 "go-wind-admin/api/gen/go/script/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/pkg/scripting"
)

// 实体生命周期钩子桥：
//
// 经 ent 的 client.Use() 挂载全局 mutation hook，提供两类钩子点：
//
//   - <entity>.before_<op>（同步，可否决）：变更前执行，脚本返回 false 或
//     ctx.stop(reason) 时业务写入被拒绝。同步执行有请求路径耗时成本，
//     且脚本引擎故障可能影响写入——因此设计为 fail-open：钩子自身执行出错
//     不阻断业务（显式否决除外），且仅对明确登记的实体启用。
//
//   - <entity>.after_<op>（异步，只读旁路）：变更成功后独立 goroutine 执行，
//     单脚本失败仅记日志/审计，绝不影响业务结果。
//
// 执行审计统一落 sys_script_logs（logExecution）。

// EntityHooksMapping ent 实体类型名（m.Type()，PascalCase）→ 钩子点前缀（小写）。
// 只收录有明确业务意义的实体，避免钩子风暴；新增实体在此登记即可生效。
var EntityHooksMapping = map[string]string{
	"User":                "user",
	"Tenant":              "tenant",
	"Role":                "role",
	"InternalMessage":     "internal_message",
	"NotificationChannel": "notification_channel",
}

// ScriptHookInvoker after 类钩子：由 app 层注入（包装 ScriptRuntime 异步执行），
// 结果只记日志/审计不影响业务。
type ScriptHookInvoker func(ctx context.Context, hookPoint string, data map[string]any)

// ScriptHookVetoInvoker before 类钩子：同步执行，返回 error 即否决业务写入
// （脚本返回 false 或 ctx.stop）。nil 阶段跳过。
type ScriptHookVetoInvoker func(ctx context.Context, hookPoint string, data map[string]any) error

// AttachEntityHooks 在 ent client 上挂载全局生命周期钩子。
// vetoInvoker 为 nil 时 before 类钩子跳过（仅 after 生效）。
func AttachEntityHooks(client *ent.Client, invoker ScriptHookInvoker, vetoInvoker ScriptHookVetoInvoker) {
	client.Use(newEntityHook(invoker, vetoInvoker))
}

// newEntityHook 构造 ent 全局 hook：
// 变更前同步触发 <entity>.before_<op>（可否决），变更成功后异步触发 <entity>.after_<op>。
func newEntityHook(invoker ScriptHookInvoker, vetoInvoker ScriptHookVetoInvoker) ent.Hook {
	h := func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			var (
				typ    = m.Type()
				prefix string
				mapped bool
				op     string
			)

			if prefix, mapped = EntityHooksMapping[typ]; mapped {
				op = mutationOpName(m.Op())
			}

			// ── before 阶段：同步、可否决 ──
			// fail-open：vetoInvoker 自身 panic 不阻断业务（显式否决返回 error 才拒绝）
			if mapped && vetoInvoker != nil {
				beforePoint := prefix + ".before_" + op
				vetoErr := func() (vetoErr error) {
					defer func() {
						if r := recover(); r != nil {
							fmt.Printf("[script-hook] before hook %s panic (fail-open): %v\n", beforePoint, r)
						}
					}()
					return vetoInvoker(ctx, beforePoint, map[string]any{
						"entity": typ,
						"op":     op,
						"id":     mutationID(m),
					})
				}()
				if vetoErr != nil {
					return nil, fmt.Errorf("%w: hook=%s reason=%v", scripting.ErrScriptVetoed, beforePoint, vetoErr)
				}
			}

			value, err := next.Mutate(ctx, m)

			// ── after 阶段：变更成功后异步触发 ──
			if err == nil && mapped && invoker != nil {
				hookPoint := prefix + ".after_" + op

				payload := map[string]any{
					"entity": typ,
					"op":     op,
					"id":     mutationID(m),
				}

				go func() {
					defer func() {
						// 钩子 goroutine 的兜底防护：脚本 panic 不能带走业务进程
						if r := recover(); r != nil {
							fmt.Printf("[script-hook] panic in %s: %v\n", hookPoint, r)
						}
					}()

					// 带超时的独立上下文：不继承请求取消信号（业务返回后钩子仍应跑完）
					hookCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
					defer cancel()
					invoker(hookCtx, hookPoint, payload)
				}()
			}

			return value, err
		})
	}
	return ent.Hook(h)
}

// mutationOpName 把 ent Op 映射为钩子点命名中的操作名。
func mutationOpName(op ent.Op) string {
	switch op {
	case ent.OpCreate:
		return "create"
	case ent.OpUpdate | ent.OpUpdateOne:
		return "update"
	case ent.OpDelete:
		return "delete"
	case ent.OpDeleteOne:
		return "delete"
	default:
		return strings.ToLower(op.String())
	}
}

// mutationID 从 mutation 尽力提取实体 ID（取不到为 0）。
func mutationID(m ent.Mutation) uint32 {
	type idGetter interface{ ID() (uint32, bool) }
	if g, ok := m.(idGetter); ok {
		if id, exists := g.ID(); exists {
			return id
		}
	}
	return 0
}

// InvokeEntityHook 异步钩子（after）的执行入口：组装执行上下文并触发钩子点。
// 与 before 侧同形：「该钩子点上没有脚本」是常态（登记的实体 × 三类操作远多于挂过脚本的钩子点），
// 不是失败，返回 nil——否则每次实体变更都白记一条 ERROR。脚本真出错仍上抛，
// 加载失败另有 Resync 阶段按脚本名逐条 ERROR，可发现性不在这条路径上丢。
func (r *ScriptRuntime) InvokeEntityHook(hookPoint string, payload map[string]any) error {
	eng := r.engineForHookPoint(hookPoint)
	if eng == nil {
		return nil
	}

	execCtx := scripting.NewContext(hookPoint)
	for k, v := range payload {
		execCtx.Set(k, v)
	}

	started := time.Now()
	err := eng.ExecuteHook(context.Background(), hookPoint, execCtx)
	r.logExecution("hook", hookPoint, hookPoint, "", 0, started, err)
	return err
}

// InvokeEntityHookVeto before 钩子的同步执行入口：返回 error 即否决业务写入。
// ctx.stop(reason) 的脚本经 ExecuteHook 的 Stopped 信号转为错误；
// 「无脚本挂载」不视为否决（返回 nil）。
func (r *ScriptRuntime) InvokeEntityHookVeto(hookPoint string, payload map[string]any) error {
	eng := r.engineForHookPoint(hookPoint)
	if eng == nil {
		return nil
	}

	execCtx := scripting.NewContext(hookPoint)
	for k, v := range payload {
		execCtx.Set(k, v)
	}

	err := eng.ExecuteHook(context.Background(), hookPoint, execCtx)
	if err != nil {
		return err
	}
	if execCtx.Stopped {
		return scriptV1.ErrorBadRequest("script veto: %s", execCtx.StopReason)
	}
	return nil
}

// engineForHookPoint 返回挂载了指定钩子点脚本的引擎（任一语言命中即可）。
// 无挂载时返回 nil（业务侧据此完全跳过执行开销）。
func (r *ScriptRuntime) engineForHookPoint(hookPoint string) *scripting.Engine {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, eng := range r.engines {
		for _, hp := range eng.HookPoints() {
			if hp.Name == hookPoint && (hp.ScriptCount > 0 || hp.CallbackCount > 0) {
				return eng
			}
		}
	}
	return nil
}
