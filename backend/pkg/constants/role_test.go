package constants

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIsPlatformRoleCode 命中 platform: 前缀的角色码应判定为平台角色。
func TestIsPlatformRoleCode(t *testing.T) {
	cases := []struct {
		name string
		code string
		want bool
	}{
		{"exact admin", PlatformAdminRoleCode, true},
		{"prefix only", PlatformRoleCodePrefix, true},
		{"platform other", "platform:foo", true},
		{"tenant code", TenantAdminRoleCode, false},
		{"template code", TenantAdminTemplateRoleCode, false},
		{"no prefix", "admin", false},
		{"empty", "", false},
		{"case sensitive", "Platform:admin", false},
		{"substring not prefix", "xplatform:admin", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsPlatformRoleCode(tc.code))
		})
	}
}

// TestIsTenantRoleCode 命中 tenant: 前缀的角色码应判定为租户角色。
func TestIsTenantRoleCode(t *testing.T) {
	cases := []struct {
		name string
		code string
		want bool
	}{
		{"exact manager", TenantAdminRoleCode, true},
		{"prefix only", TenantRoleCodePrefix, true},
		{"tenant other", "tenant:foo", true},
		{"platform code", PlatformAdminRoleCode, false},
		{"template code", TenantAdminTemplateRoleCode, false},
		{"no prefix", "manager", false},
		{"empty", "", false},
		{"case sensitive", "Tenant:manager", false},
		{"substring not prefix", "xtenant:manager", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsTenantRoleCode(tc.code))
		})
	}
}

// TestIsTemplateRoleCode 命中 template: 前缀的角色码应判定为模板角色。
func TestIsTemplateRoleCode(t *testing.T) {
	cases := []struct {
		name string
		code string
		want bool
	}{
		{"exact template", TenantAdminTemplateRoleCode, true},
		{"prefix only", TemplateRoleCodePrefix, true},
		{"template other", "template:foo", true},
		{"platform code", PlatformAdminRoleCode, false},
		{"tenant code", TenantAdminRoleCode, false},
		{"no prefix", "admin", false},
		{"empty", "", false},
		{"case sensitive", "Template:foo", false},
		{"substring not prefix", "xtemplate:foo", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsTemplateRoleCode(tc.code))
		})
	}
}

// TestExtractRoleCodeFromTemplate 模板角色码应剥离 template: 前缀;
// 非模板角色码原样返回。
func TestExtractRoleCodeFromTemplate(t *testing.T) {
	cases := []struct {
		name string
		code string
		want string
	}{
		{"template stripped", TenantAdminTemplateRoleCode, TenantAdminRoleCode},
		{"template other stripped", "template:foo", "foo"},
		{"prefix only to empty", TemplateRoleCodePrefix, ""},
		{"non-template passthrough", PlatformAdminRoleCode, PlatformAdminRoleCode},
		{"tenant passthrough", TenantAdminRoleCode, TenantAdminRoleCode},
		{"plain code passthrough", "admin", "admin"},
		{"empty passthrough", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ExtractRoleCodeFromTemplate(tc.code))
		})
	}
}

// TestHasRoleCodePrefix 通用前缀判定函数的边界行为。
func TestHasRoleCodePrefix(t *testing.T) {
	cases := []struct {
		name string
		code string
		pfx  string
		want bool
	}{
		{"matches", "abc:def", "abc:", true},
		{"exact equal", "abc", "abc", true},
		{"prefix longer than code", "ab", "abc", false},
		{"empty code nonempty prefix", "", "abc", false},
		{"both empty", "", "", true},
		{"empty prefix nonempty code", "abc", "", true},
		{"no match", "abc:def", "xyz:", false},
		{"case sensitive", "Abc:def", "abc:", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, HasRoleCodePrefix(tc.code, tc.pfx))
		})
	}
}

// TestRoleCodeConstants 钉死角色码常量的确切值,防止误改影响 RBAC 分流。
func TestRoleCodeConstants(t *testing.T) {
	assert.Equal(t, ":", RoleCodeSpilt)
	assert.Equal(t, "platform:", PlatformRoleCodePrefix)
	assert.Equal(t, "tenant:", TenantRoleCodePrefix)
	assert.Equal(t, "template:", TemplateRoleCodePrefix)
	assert.Equal(t, "platform:admin", PlatformAdminRoleCode)
	assert.Equal(t, "tenant:manager", TenantAdminRoleCode)
	assert.Equal(t, "template:tenant:manager", TenantAdminTemplateRoleCode)
	assert.Equal(t, "平台管理员", DefaultPlatformAdminRoleName)
	assert.Equal(t, "租户管理员", DefaultTenantManagerRoleName)
}

// TestIsReservedRoleCode 保留角色码判定：内置身份码与保留命名空间全数命中，
// 普通码（含非内置的 tenant: 前缀值）放行。该判定是角色 CRUD 的越权闸门，
// 误放行会让自建角色伪装成内置身份（登录态管理员标志按码值精确匹配）。
func TestIsReservedRoleCode(t *testing.T) {
	cases := []struct {
		name string
		code string
		want bool
	}{
		{"exact platform admin", PlatformAdminRoleCode, true},
		{"exact tenant manager", TenantAdminRoleCode, true},
		{"exact tenant admin template", TenantAdminTemplateRoleCode, true},
		{"platform namespace", "platform:foo", true},
		{"platform namespace bare prefix", PlatformRoleCodePrefix, true},
		{"template namespace", "template:foo", true},
		{"template namespace bare prefix", TemplateRoleCodePrefix, true},
		{"tenant non-builtin allowed", "tenant:sales", false},
		{"tenant namespace bare prefix", TenantRoleCodePrefix, false},
		{"tenant prefix lookalike", "xtenant:manager", false},
		{"plain code", "admin", false},
		{"case sensitive", "Platform:admin", false},
		{"case sensitive builtin", "Tenant:Manager", false},
		{"empty", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsReservedRoleCode(tc.code))
		})
	}
}
