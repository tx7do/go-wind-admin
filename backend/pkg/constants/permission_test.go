package constants

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPermissionCodeConstants 钉死系统权限代码常量的确切值。
// 这些代码由 RBAC 拦截器按字符串匹配（sys:platform_admin 等），
// 值漂移会导致默认角色静默丢权限。
func TestPermissionCodeConstants(t *testing.T) {
	assert.Equal(t, "sys:", SystemPermissionCodePrefix)
	assert.Equal(t, "sys:access_backend", SystemAccessBackendPermissionCode)
	assert.Equal(t, "sys:manage_tenants", SystemManageTenantsPermissionCode)
	assert.Equal(t, "sys:audit_logs", SystemAuditLogsPermissionCode)
	assert.Equal(t, "sys:platform_admin", SystemPlatformAdminPermissionCode)
	assert.Equal(t, "sys:tenant_manager", SystemTenantManagerPermissionCode)

	// 系统权限代码必须由统一前缀拼出，防止引入脱离前缀体系的平行常量
	assert.Equal(t, SystemPermissionCodePrefix+"access_backend", SystemAccessBackendPermissionCode)
	assert.Equal(t, SystemPermissionCodePrefix+"manage_tenants", SystemManageTenantsPermissionCode)
	assert.Equal(t, SystemPermissionCodePrefix+"audit_logs", SystemAuditLogsPermissionCode)
	assert.Equal(t, SystemPermissionCodePrefix+"platform_admin", SystemPlatformAdminPermissionCode)
	assert.Equal(t, SystemPermissionCodePrefix+"tenant_manager", SystemTenantManagerPermissionCode)

	assert.Equal(t, "sys", SystemPermissionModule)
	assert.Equal(t, "biz", DefaultBizPermissionModule)
	assert.Equal(t, "uncategorized", UncategorizedPermissionGroup)
}

// TestProtectedPermissionCodes 受保护权限代码清单必须恰好覆盖全部
// 系统权限代码，且全部带 sys: 前缀。删除保护标记的条目会让这些
// 权限变成可删状态，默认管理员角色会随种子数据漂移而失能。
func TestProtectedPermissionCodes(t *testing.T) {
	expected := []string{
		SystemAccessBackendPermissionCode,
		SystemManageTenantsPermissionCode,
		SystemAuditLogsPermissionCode,
		SystemPlatformAdminPermissionCode,
		SystemTenantManagerPermissionCode,
	}
	assert.ElementsMatch(t, expected, ProtectedPermissionCodes,
		"受保护权限代码清单漂移；如有意变更请同步更新本测试")
	for _, code := range ProtectedPermissionCodes {
		assert.True(t, HasRoleCodePrefix(code, SystemPermissionCodePrefix),
			"受保护权限代码 %q 必须位于 sys: 前缀体系内", code)
	}
}

// TestIsReservedPermissionCode 保留权限码判定：sys: 命名空间全数命中，
// 业务码放行。该判定是权限 CRUD 的越权闸门——sys:access_backend 在登录
// 链路按码值精确匹配，误放行可让自建权限伪造登录资格。
func TestIsReservedPermissionCode(t *testing.T) {
	cases := []struct {
		name string
		code string
		want bool
	}{
		{"access backend", SystemAccessBackendPermissionCode, true},
		{"manage tenants", SystemManageTenantsPermissionCode, true},
		{"platform admin", SystemPlatformAdminPermissionCode, true},
		{"sys namespace other", "sys:foo", true},
		{"sys namespace bare prefix", SystemPermissionCodePrefix, true},
		{"biz code allowed", "biz:user:create", false},
		{"plain code allowed", "user:create", false},
		{"sys lookalike not prefix", "xsys:foo", false},
		{"case sensitive", "Sys:foo", false},
		{"empty", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsReservedPermissionCode(tc.code))
		})
	}
}
