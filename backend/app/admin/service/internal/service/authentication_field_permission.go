package service

import (
	"context"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
)

// aggregateHiddenFields 聚合用户各角色的字段权限配置为令牌承载的隐藏字段集。
//
// 语义（黑名单式字段权限）：
//   - 平台上下文（tenantID==0）：不聚合（平台管理员字段全可见）。
//   - 各角色配置的 "资源.字段" 条目按并集聚合；未配置即无隐藏字段。
//   - 聚合失败时 fail-open（仅告警、不加条目）：字段权限是纵深防御手段，
//     与数据范围 "配置异常即拒绝登录" 的取向不同——登录可用性优先，
//     且条目写入侧已有上限与校验。
//
// 条目格式为 "资源.字段"（字段为 proto json_name）；refresh 流程同样走本函数，
// 配置变更最迟随下次刷新令牌生效。
func (s *AuthenticationService) aggregateHiddenFields(
	ctx context.Context,
	tenantID uint64,
	roleIDs []uint64,
	tokenPayload *authenticationV1.UserTokenPayload,
) {
	if tenantID == 0 {
		return
	}

	groups, err := s.roleFieldPermissionRepo.ListHiddenFieldGroupsByRoleIDs(ctx, roleIDs)
	if err != nil {
		s.log.Errorf(ctx, "aggregate hidden fields: list by roles [%v] failed: %s", roleIDs, err.Error())
		return
	}

	seen := make(map[string]struct{})
	for _, entries := range groups {
		for _, entry := range entries {
			resource := entry.GetResource()
			if resource == "" {
				continue
			}
			for _, f := range entry.GetHiddenFields() {
				if f == "" {
					continue
				}
				key := resource + "." + f
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				tokenPayload.HiddenFields = append(tokenPayload.HiddenFields, key)
			}
		}
	}
}
