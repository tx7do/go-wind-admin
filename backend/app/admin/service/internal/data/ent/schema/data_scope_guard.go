package schema

import (
	"context"

	"entgo.io/ent"

	"github.com/tx7do/go-crud/entgo/rule"

	entPrivacy "go-wind-admin/app/admin/service/internal/data/ent/privacy"
)

// dataScopeFilterRule 数据范围查询过滤规则适配器：把 ent 查询的
// privacy.Filter 交给库 rule.PermissionRule 处理。
// 全部语义在库规则内实现（rule/system.go PermissionRule）：
//   - 平台/系统上下文放行；缺 Viewer 拒绝（fail-closed）
//   - 无 scope / 显式 None 拒绝
//   - ALL 放行；SELF 注入 created_by = uid；UNIT 注入 org_unit_id IN targets
//   - 多 scope 以 OR 并集注入
var dataScopeFilterRule = entPrivacy.FilterFunc(func(ctx context.Context, f entPrivacy.Filter) error {
	return rule.PermissionRule(ctx, f)
})

// DataScopeQueryPolicy 数据范围查询过滤策略（V1 仅查询侧）。
//
// 挂载前置条件：实体表必须具备 created_by 与 org_unit_id 两列
// （分别承载 SELF / UNIT 档谓词），缺列会在谓词注入时引发运行时错误。
// V1 试点：position（唯一同时具备两列的表）。其余表接入照此 opt-in：
// schema 的 Policy() 改返 TenantAndDataScopePolicy（组合策略），
// 并先行核对两列齐备。
type DataScopeQueryPolicy struct{}

func (DataScopeQueryPolicy) EvalQuery(ctx context.Context, q ent.Query) error {
	return dataScopeFilterRule.EvalQuery(ctx, q)
}

func (DataScopeQueryPolicy) EvalMutation(_ context.Context, _ ent.Mutation) error {
	// V1 仅查询侧过滤；变更侧行级过滤（R2）另行接入。
	return nil
}

// TenantAndDataScopePolicy 组合策略：链式评估租户变更防护与数据范围查询过滤。
// 语义与 ent 隐私规则链一致：任一子策略返回错误即整体拒绝，全 nil 放行。
// 查询侧 = TenantMutationGuard（透传，查询隔离由库 TenantPrivacy 负责）
//   - DataScopeQuery（行级数据范围过滤）；
//
// 变更侧 = TenantMutationGuard（跨租户篡改/删除防护）。
type TenantAndDataScopePolicy struct{}

func (TenantAndDataScopePolicy) EvalQuery(ctx context.Context, q ent.Query) error {
	if err := (TenantMutationGuardPolicy{}).EvalQuery(ctx, q); err != nil {
		return err
	}
	return (DataScopeQueryPolicy{}).EvalQuery(ctx, q)
}

func (TenantAndDataScopePolicy) EvalMutation(ctx context.Context, m ent.Mutation) error {
	return (TenantMutationGuardPolicy{}).EvalMutation(ctx, m)
}
