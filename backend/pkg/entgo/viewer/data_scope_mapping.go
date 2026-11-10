package viewer

import (
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"

	"github.com/tx7do/go-crud/viewer"
)

// 本文件是 identityV1 数据范围声明到 go-crud/viewer 库层执行结构的唯一适配层；
// 用户/系统上下文的标准实现（UserContext/SystemContext）由库直接提供。

// BuildDataScopes 把令牌承载的聚合数据范围转换为库层执行结构。
//
// 旧单值回退：过渡期令牌可能只带单值 ds 声明（无 dss），按单元素处理，
// 与旧 convertDataScope 行为对齐（UNIT 类目标集为空时由库规则按
// "无有效谓词" fail-closed 拒绝，而非放行）。
//
// UNSPECIFIED 一律剔除：空集不做兜底放行，交由库 rule.PermissionRule
// 以 "no data scope defined" 拒绝。
func BuildDataScopes(
	scopes []identityV1.DataScope,
	unitIDs []int64,
	legacy identityV1.DataScope,
) []viewer.DataScope {
	if len(scopes) == 0 && legacy != identityV1.DataScope_DATA_SCOPE_UNSPECIFIED {
		scopes = []identityV1.DataScope{legacy}
	}

	result := make([]viewer.DataScope, 0, len(scopes))
	for _, s := range scopes {
		switch s {
		case identityV1.DataScope_ALL:
			result = append(result, viewer.DataScope{
				ScopeType: viewer.ScopeTypeAll,
			})

		case identityV1.DataScope_SELF:
			result = append(result, viewer.DataScope{
				ScopeType: viewer.ScopeTypeSelf,
			})

		case identityV1.DataScope_UNIT_ONLY,
			identityV1.DataScope_UNIT_AND_CHILD,
			identityV1.DataScope_SELECTED_UNITS:
			result = append(result, viewer.DataScope{
				ScopeType: viewer.ScopeTypeUnit,
				TargetIDs: unitIDs,
			})

		default:
			// UNSPECIFIED / 未知值剔除
		}
	}
	return result
}
