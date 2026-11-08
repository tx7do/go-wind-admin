package constants

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tx7do/go-utils/password"
	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

// 种子自洽校验：种子引用全面语义化后（去硬编码 id），这些静态不变量是
// "全新安装播种不悬空"的第一道防线——引用键必须在被引用集合里真实存在。

// TestDefaultPasswordMeetsPolicy 种子密码必须通过等保复杂度策略。
// 回归背景：默认管理员密码曾是 "admin"，prepareCredential 入库前的复杂度校验
// 必然拒绝，导致空库全新部署时凭证行创建失败且错误被吞，admin 永久无法登录
// （GitHub issue #58）。
func TestDefaultPasswordMeetsPolicy(t *testing.T) {
	assert.NoError(t, password.ValidateComplexity(DefaultUserPassword, DefaultPasswordMinLen),
		"DefaultUserPassword 必须满足口令复杂度策略，否则默认数据初始化会半途失败")
}

// TestDefaultUserCredentialsUsePolicyCompliantPassword 防漂移：
// 哈希类凭证的种子密码必须是 DefaultUserPassword（策略合规的唯一默认密码），
// 不允许再引入独立的明文密码常量绕过复杂度校验。
func TestDefaultUserCredentialsUsePolicyCompliantPassword(t *testing.T) {
	for _, credential := range DefaultUserCredentials {
		if credential.GetCredentialType() != authenticationV1.UserCredential_PASSWORD_HASH {
			continue
		}
		assert.Equal(t, DefaultUserPassword, credential.GetCredential(),
			"PASSWORD_HASH 种子凭证的 Credential 必须引用 DefaultUserPassword")
		assert.NoError(t, password.ValidateComplexity(credential.GetCredential(), DefaultPasswordMinLen))
	}
}

// TestDefaultMenusHaveExplicitIds 防漂移：菜单播种按显式 Id 落库（全路径身份
// 键父链拼接、权限种子引用都依赖它），漏 Id 的菜单行会拿自增 id——引用全部悬空。
func TestDefaultMenusHaveExplicitIds(t *testing.T) {
	seen := make(map[uint32]string, len(DefaultMenus))
	for _, m := range DefaultMenus {
		require.NotNil(t, m.Id, "菜单 [%s] 必须显式声明 Id", m.GetName())
		require.NotContains(t, seen, m.GetId(), "菜单 Id %d 重复声明（[%s] 与 [%s]）",
			m.GetId(), seen[m.GetId()], m.GetName())
		seen[m.GetId()] = m.GetName()
	}
}

// TestDefaultMenusTreeReferencesAreClosed 防漂移：菜单的 ParentId 必须引用
// DefaultMenus 内已声明的菜单 id（父链断裂会让全路径拼接把子菜单当根节点，
// 绑定快照与导航反查两侧错位）。
func TestDefaultMenusTreeReferencesAreClosed(t *testing.T) {
	byID := make(map[uint32]*permissionV1.Menu, len(DefaultMenus))
	for _, m := range DefaultMenus {
		byID[m.GetId()] = m
	}
	for _, m := range DefaultMenus {
		if m.ParentId == nil {
			continue
		}
		parent, ok := byID[m.GetParentId()]
		assert.True(t, ok, "菜单 [%s] 的 ParentId %d 在 DefaultMenus 中不存在",
			m.GetName(), m.GetParentId())
		if ok {
			assert.NotEqual(t, m.GetId(), parent.GetId(), "菜单 [%s] 不得自引用为父节点", m.GetName())
		}
	}
}

// TestDefaultPermissionMenuIdsReferenceExistingMenus 防漂移：模板权限 MenuIds
// 引用的菜单 id 必须都在 DefaultMenus 里——菜单播种以 DefaultMenus 为源，
// 引用了不存在的 id 即播种期收敛剔除（软降级 WARN），属于种子笔误。
func TestDefaultPermissionMenuIdsReferenceExistingMenus(t *testing.T) {
	menuIDs := make(map[uint32]struct{}, len(DefaultMenus))
	for _, m := range DefaultMenus {
		menuIDs[m.GetId()] = struct{}{}
	}

	for _, p := range DefaultPermissions {
		for _, mid := range p.GetMenuIds() {
			assert.Contains(t, menuIDs, mid,
				"权限 [%s] 的 MenuIds 引用了 DefaultMenus 中不存在的菜单 id %d", p.GetCode(), mid)
		}
	}
}

// TestDefaultPermissionGroupNamesReferenceDeclaredGroups 防漂移：权限种子的
// GroupName 必须在 DefaultPermissionGroups 中有同名组——GroupName 是播种期
// 分组解析键，解析不到软降级为"未分组"，属种子笔误。
func TestDefaultPermissionGroupNamesReferenceDeclaredGroups(t *testing.T) {
	groupNames := make(map[string]struct{}, len(DefaultPermissionGroups))
	for _, g := range DefaultPermissionGroups {
		require.NotNil(t, g.Name, "DefaultPermissionGroups 每行必须声明 Name（分组解析键）")
		groupNames[g.GetName()] = struct{}{}
	}

	for _, p := range DefaultPermissions {
		if p.GroupName == nil {
			continue
		}
		assert.Contains(t, groupNames, p.GetGroupName(),
			"权限 [%s] 的 GroupName [%s] 在 DefaultPermissionGroups 中不存在", p.GetCode(), p.GetGroupName())
	}
}

// TestDefaultRolePermissionCodesConsistent 防漂移：DefaultRolePermissionCodes
// 与 DefaultRoles 双向对齐——map 里出现的角色 code 必须是种子角色，系统/模板
// 角色也应声明权限码集合（否则角色播种后无任何权限绑定，admin 无法越权链断头）。
func TestDefaultRolePermissionCodesConsistent(t *testing.T) {
	roleCodes := make(map[string]*permissionV1.Role, len(DefaultRoles))
	for _, r := range DefaultRoles {
		require.NotNil(t, r.Code, "DefaultRoles 每行必须声明 Code")
		roleCodes[r.GetCode()] = r
	}

	for roleCode, codes := range DefaultRolePermissionCodes {
		assert.Contains(t, roleCodes, roleCode,
			"DefaultRolePermissionCodes 引用了 DefaultRoles 中不存在的角色 code [%s]", roleCode)
		assert.NotEmpty(t, codes, "角色 [%s] 的权限码集合不应为空", roleCode)
		for _, pc := range codes {
			assert.True(t, IsReservedPermissionCode(pc),
				"角色 [%s] 引用的权限码 [%s] 应为系统权限码（种子权限均为 sys: 前缀）", roleCode, pc)
		}
	}

	for _, r := range DefaultRoles {
		switch r.GetType() {
		case permissionV1.Role_SYSTEM, permissionV1.Role_TEMPLATE:
			assert.Contains(t, DefaultRolePermissionCodes, r.GetCode(),
				"系统/模板角色 [%s] 未在 DefaultRolePermissionCodes 中声明权限集", r.GetCode())
		}
	}
}

// TestDefaultPermissionSystemCodesArePrefixProtected 防漂移：种子权限的 code
// 必须全部带系统前缀——权限同步（MERGE）按前缀保护系统权限行不被重建吞掉，
// 漏前缀 = 播种权限会被权限同步删除重建（id 换新、引用悬空）。
func TestDefaultPermissionSystemCodesArePrefixProtected(t *testing.T) {
	for _, p := range DefaultPermissions {
		assert.True(t, IsReservedPermissionCode(p.GetCode()),
			"种子权限 [%s] 的 code 必须带系统前缀 [%s]", p.GetCode(), SystemPermissionCodePrefix)
	}
}

// TestIsTenantManagerDeniedApiPath 黑名单匹配行为：前缀命中、租户可用端点不误伤。
func TestIsTenantManagerDeniedApiPath(t *testing.T) {
	assert.NotEmpty(t, TenantManagerDeniedApiPrefixes, "黑名单不应为空（空=租户管理员模板获得全部端点）")

	denied := []string{
		"/admin/v1/tenants",            // 精确等于前缀
		"/admin/v1/tenants/42",         // 前缀下的子路径
		"/admin/v1/tenants:exists",     // 前缀 + 动作后缀（真机形态）
		"/admin/v1/ai/providers",       // AI 平台页
		"/admin/v1/ai/providers/x",     // AI 平台页子路径
		"/admin/v1/api-audit-logs",     // 审计族
		"/admin/v1/notification-rules", // 通知域管理页
	}
	for _, p := range denied {
		assert.True(t, IsTenantManagerDeniedApiPath(p), "[%s] 应命中黑名单", p)
	}

	allowed := []string{
		"/admin/v1/roles",        // 租户管理员可管理角色（黑名单刻意不含）
		"/admin/v1/users",        // 租户内用户管理
		"/admin/v1/ai/chat",      // AI 对话（登录即可用）
		"/admin/v1/ai/knowledge", // 知识库
		"/admin/v1/tasks",        // 任务管理（菜单声明租户管理员可见）
		"/admin/v1/me",           // 个人资料
		"/admin/v1/login",        // 认证端点
	}
	for _, p := range allowed {
		assert.False(t, IsTenantManagerDeniedApiPath(p), "[%s] 不应命中黑名单（租户管理员模板应可见）", p)
	}
}
