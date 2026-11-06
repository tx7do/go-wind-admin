package service

import "testing"

// enforcePerBlockTenantFilter 结构化逐块强制的穿透面封堵回归（P1-5 深层）。
// 场景全部按租户 5 + 缩减白名单（两张表）构造。
func TestEnforcePerBlockTenantFilter(t *testing.T) {
	wl := map[string]string{
		"sys_users": "用户表",
		"sys_roles": "角色表",
	}

	cases := []struct {
		name   string
		sql    string
		blocks bool // true = 应拒绝
	}{
		{"单表带谓词放行",
			"SELECT * FROM sys_users WHERE tenant_id = 5", false},
		{"别名前缀谓词放行",
			"SELECT u.* FROM sys_users u WHERE u.tenant_id = 5", false},
		{"两表两谓词放行",
			"SELECT * FROM sys_users u JOIN sys_roles r ON r.id = u.role_id WHERE u.tenant_id = 5 AND r.tenant_id = 5", false},
		{"JOIN主表漏滤拒绝(此前可穿透)",
			"SELECT * FROM sys_users u JOIN sys_roles r ON r.tenant_id = 5 AND r.id = u.role_id", true},
		{"JOIN一表无谓词拒绝",
			"SELECT * FROM sys_users u JOIN sys_roles r ON r.id = u.role_id WHERE u.tenant_id = 5", true},
		{"子查询内表无谓词拒绝(此前可穿透)",
			"SELECT * FROM sys_users WHERE tenant_id = 5 AND id IN (SELECT user_id FROM sys_roles WHERE status = 1)", true},
		{"子查询自带谓词放行",
			"SELECT * FROM sys_users WHERE tenant_id = 5 AND id IN (SELECT user_id FROM sys_roles WHERE tenant_id = 5 AND status = 1)", false},
		{"派生表无谓词拒绝",
			"SELECT * FROM (SELECT * FROM sys_users) t WHERE t.tenant_id = 5", true},
		{"派生表内谓词放行",
			"SELECT * FROM (SELECT * FROM sys_users WHERE tenant_id = 5) t", false},
		{"CASE里的谓词不算覆盖",
			"SELECT CASE WHEN tenant_id = 5 THEN 1 ELSE 0 END AS c FROM sys_users", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stripped, _ := stripSQLStringsAndComments(tc.sql)
			err := enforcePerBlockTenantFilter(stripped, wl, 5)
			if tc.blocks && err == nil {
				t.Fatalf("应拒绝: %s", tc.sql)
			}
			if !tc.blocks && err != nil {
				t.Fatalf("应放行: %s, got %v", tc.sql, err)
			}
		})
	}
}

// TestSanitizeSQLForTenantIntegration 集成：sanitizeSQLForTenant 对
// 多表 JOIN 漏滤形态拒绝、合规形态放行并完成 LIMIT 钳制。
func TestSanitizeSQLForTenantIntegration(t *testing.T) {
	wl := map[string]string{
		"sys_users": "用户表",
		"sys_roles": "角色表",
	}
	pat := tenantIDPatternFor(5)

	// 穿透形态：JOIN 带谓词但主表无
	leak := "SELECT u.username FROM sys_users u JOIN sys_roles r ON r.tenant_id = 5 AND r.id = u.role_id LIMIT 10"
	if _, err := sanitizeSQLForTenant(leak, wl, pat, 5); err == nil {
		t.Fatal("JOIN 漏滤主表的穿透形态应被拒绝")
	}

	// 合规形态：两表两谓词 → 放行并钳 LIMIT
	ok := "SELECT u.username FROM sys_users u JOIN sys_roles r ON r.id = u.role_id WHERE u.tenant_id = 5 AND r.tenant_id = 5"
	cleaned, err := sanitizeSQLForTenant(ok, wl, pat, 5)
	if err != nil {
		t.Fatalf("合规形态应放行: %v", err)
	}
	if !aiQuerySelectPattern.MatchString(cleaned) {
		t.Fatalf("放行产物应为合法 SELECT: %s", cleaned)
	}
}
