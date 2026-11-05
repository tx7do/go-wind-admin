package service

import "testing"

// TestAiQueryRejectDisjunctionOrComments 租户路径 SQL 护栏加固回归（P1-5）：
// 析取（OR）与注释一律拒绝——"tenant_id = 5 OR 1=1" 绕过出现性检查、
// "-- tenant_id = 5" 注释骗过存在性正则而谓词实际不生效。
// 字符串字面量里的 "or" 与注释不算结构析取；ORDER BY 不受影响。
func TestAiQueryRejectDisjunctionOrComments(t *testing.T) {
	cases := []struct {
		name    string
		sql     string
		blocked bool
	}{
		{"合取谓词放行", "SELECT * FROM sys_users WHERE tenant_id = 5 AND status = 1", false},
		{"析取绕过拒绝", "SELECT * FROM sys_users WHERE tenant_id = 5 OR 1=1", true},
		{"字符串内的or放行", "SELECT * FROM sys_users WHERE tenant_id = 5 AND name = 'or x'", false},
		{"行内注释拒绝", "SELECT * FROM sys_users -- tenant_id = 5", true},
		{"块注释拒绝", "SELECT * /* tenant_id = 5 */ FROM sys_users", true},
		{"order by不受影响", "SELECT * FROM sys_users WHERE tenant_id = 5 ORDER BY id LIMIT 10", false},
		{"in列表放行", "SELECT * FROM sys_users WHERE tenant_id = 5 AND status IN (1, 2)", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := rejectDisjunctionOrComments(tc.sql)
			if tc.blocked && err == nil {
				t.Fatalf("应被拒绝: %s", tc.sql)
			}
			if !tc.blocked && err != nil {
				t.Fatalf("应放行: %s, got %v", tc.sql, err)
			}
		})
	}
}
