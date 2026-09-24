package service

import (
	"context"

	"github.com/tx7do/go-utils/trans"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
)

// maxDataScopeUnitIds UNIT 类数据范围单元目标集上限：
// 防止超大授权集撑爆令牌体积；超限按配置异常退化整体拒绝。
const maxDataScopeUnitIds = 256

// aggregateDataScopes 聚合用户各角色的数据范围配置为令牌承载的扁平结构。
//
// 语义（若依式角色级数据范围）：
//   - 平台上下文（tenantID==0）：整体 [ALL]（平台侧本就全放行，执行规则跳过）。
//   - 任一角色 ALL → 整体 [ALL]。
//   - 否则活动类型取并集（SELF 与 UNIT_* 可共存），UNIT_* 的单元目标集为
//     各角色贡献的并集，全部按令牌租户过滤（登录上下文为 privacy.Allow，
//     绕过租户隐私层，显式租户谓词是唯一防线）：
//     UNIT_ONLY      → 用户所属单元（限本租户）
//     UNIT_AND_CHILD → 用户所属单元及其全部后代（path 前缀展开，限本租户）
//     SELECTED_UNITS → 角色配置的自定义单元集（纵深防御：二次租户过滤）
//   - 退化（目标集超上限 / 无任何有效活动类型）→ [UNSPECIFIED]：
//     viewer 构建侧将其剔除为空集，库规则按 "no data scope defined"
//     fail-closed 拒绝，而非放行。
//
// 令牌单值镜像：聚合结果恰为 [ALL] 或 [SELF] 时同步写旧 ds 单值字段，
// 供过渡期旧中间件按旧轨道解析（UNIT_* 无法用单值无损表达，不镜像）。
// refresh 流程同样走本函数，配置变更最迟随下次刷新令牌生效。
func (s *AuthenticationService) aggregateDataScopes(
	ctx context.Context,
	tenantID uint32,
	userID uint32,
	roleIDs []uint32,
	tokenPayload *authenticationV1.UserTokenPayload,
) {
	// 平台上下文：无租户隔离维度，整体全量。
	if tenantID == 0 {
		tokenPayload.DataScopes = []identityV1.DataScope{identityV1.DataScope_ALL}
		tokenPayload.DataScope = trans.Ptr(identityV1.DataScope_ALL)
		return
	}

	roles, err := s.roleRepo.ListRolesByRoleIds(ctx, roleIDs)
	if err != nil {
		s.log.Errorf(ctx, "aggregate data scopes: list roles by ids [%v] failed: %s", roleIDs, err.Error())
		return
	}

	// 阶段一：扫描角色配置，识别活动类型与待取目标集。
	hasSelf := false
	unitTypes := make(map[identityV1.DataScope]struct{})
	needOwnUnits := false
	needDescendants := false
	var selectedUnitRoleIDs []uint32

	for _, role := range roles {
		switch role.GetDataScope() {
		case identityV1.DataScope_ALL:
			tokenPayload.DataScopes = []identityV1.DataScope{identityV1.DataScope_ALL}
			tokenPayload.DataScope = trans.Ptr(identityV1.DataScope_ALL)
			return

		case identityV1.DataScope_SELF:
			hasSelf = true

		case identityV1.DataScope_UNIT_ONLY:
			unitTypes[identityV1.DataScope_UNIT_ONLY] = struct{}{}
			needOwnUnits = true

		case identityV1.DataScope_UNIT_AND_CHILD:
			unitTypes[identityV1.DataScope_UNIT_AND_CHILD] = struct{}{}
			needOwnUnits = true
			needDescendants = true

		case identityV1.DataScope_SELECTED_UNITS:
			unitTypes[identityV1.DataScope_SELECTED_UNITS] = struct{}{}
			if role.GetId() != 0 {
				selectedUnitRoleIDs = append(selectedUnitRoleIDs, role.GetId())
			}
		}
	}

	// 阶段二：收集 UNIT 类目标集（全部经本租户过滤）。
	unitTargets := make(map[uint64]struct{})

	if needOwnUnits {
		ownUnits, err := s.userRepo.ListOrgUnitIDsByUserID(ctx, userID)
		if err != nil {
			s.log.Errorf(ctx, "aggregate data scopes: list own org units for user [%d] failed: %s", userID, err.Error())
			return
		}
		tenantOwn, err := s.orgUnitRepo.ListOrgUnitIDsInTenant(ctx, tenantID, ownUnits)
		if err != nil {
			s.log.Errorf(ctx, "aggregate data scopes: filter own org units for user [%d] failed: %s", userID, err.Error())
			return
		}
		if needDescendants {
			expanded, err := s.orgUnitRepo.ListSelfAndDescendantOrgUnitIds(ctx, tenantID, tenantOwn)
			if err != nil {
				s.log.Errorf(ctx, "aggregate data scopes: expand descendants for user [%d] failed: %s", userID, err.Error())
				return
			}
			tenantOwn = expanded
		}
		for _, id := range tenantOwn {
			unitTargets[uint64(id)] = struct{}{}
		}
	}

	if len(selectedUnitRoleIDs) > 0 {
		configuredByRole, err := s.roleOrgUnitRepo.ListOrgUnitIDsByRoleIDs(ctx, selectedUnitRoleIDs)
		if err != nil {
			s.log.Errorf(ctx, "aggregate data scopes: list configured org units by roles [%v] failed: %s", selectedUnitRoleIDs, err.Error())
			return
		}
		for _, ids := range configuredByRole {
			// 纵深防御：配置集写入时已校验同租户，此处按令牌租户二次过滤。
			inTenant, err := s.orgUnitRepo.ListOrgUnitIDsInTenant(ctx, tenantID, ids)
			if err != nil {
				s.log.Errorf(ctx, "aggregate data scopes: filter configured org units failed: %s", err.Error())
				return
			}
			for _, id := range inTenant {
				unitTargets[uint64(id)] = struct{}{}
			}
		}
	}

	// 阶段三：装配。
	var activeTypes []identityV1.DataScope
	if hasSelf {
		activeTypes = append(activeTypes, identityV1.DataScope_SELF)
	}
	if len(unitTypes) > 0 && len(unitTargets) > 0 {
		for t := range unitTypes {
			activeTypes = append(activeTypes, t)
		}
		for id := range unitTargets {
			tokenPayload.DataScopeUnitIds = append(tokenPayload.DataScopeUnitIds, id)
		}
	}

	// 退化：无有效活动类型（含 UNIT 类目标集为空的情形）。
	if len(activeTypes) == 0 {
		s.log.Errorf(ctx, "aggregate data scopes: user [%d] has no effective data scope (degenerate config, denying)",
			userID)
		tokenPayload.DataScopes = []identityV1.DataScope{identityV1.DataScope_DATA_SCOPE_UNSPECIFIED}
		return
	}

	// 目标集超上限：配置异常，整体拒绝。
	if len(tokenPayload.DataScopeUnitIds) > maxDataScopeUnitIds {
		s.log.Errorf(ctx, "aggregate data scopes: user [%d] unit targets [%d] exceed cap [%d], denying",
			userID, len(tokenPayload.DataScopeUnitIds), maxDataScopeUnitIds)
		tokenPayload.DataScopes = []identityV1.DataScope{identityV1.DataScope_DATA_SCOPE_UNSPECIFIED}
		tokenPayload.DataScopeUnitIds = nil
		return
	}

	tokenPayload.DataScopes = activeTypes

	// 单值镜像：恰为 [ALL] / [SELF] 时同步旧 ds 轨道（过渡期兼容）。
	if len(activeTypes) == 1 {
		switch activeTypes[0] {
		case identityV1.DataScope_ALL:
			tokenPayload.DataScope = trans.Ptr(identityV1.DataScope_ALL)
		case identityV1.DataScope_SELF:
			tokenPayload.DataScope = trans.Ptr(identityV1.DataScope_SELF)
		}
	}
}
