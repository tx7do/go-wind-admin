package audit

import (
	"reflect"
	"testing"
)

func TestExtractTables(t *testing.T) {
	cases := []struct {
		name string
		sql  string
		want []string
	}{
		{
			name: "ent select quoted",
			sql:  `SELECT "sys_users"."id" FROM "sys_users" WHERE "sys_users"."id" = $1 LIMIT *** OFFSET ***`,
			want: []string{"sys_users"},
		},
		{
			name: "select count",
			sql:  `SELECT COUNT("sys_policy_evaluation_logs"."id") FROM "sys_policy_evaluation_logs"`,
			want: []string{"sys_policy_evaluation_logs"},
		},
		{
			name: "insert into",
			sql:  `INSERT INTO "sys_users" ("name", "age") VALUES ($1, $2) RETURNING "id"`,
			want: []string{"sys_users"},
		},
		{
			name: "upsert on conflict do update",
			sql:  `INSERT INTO "sys_users" ("id", "name") VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET name = $3`,
			want: []string{"sys_users"},
		},
		{
			name: "update set",
			sql:  `UPDATE "sys_users" SET "name" = $1 WHERE "id" = $2`,
			want: []string{"sys_users"},
		},
		{
			name: "delete from",
			sql:  `DELETE FROM "sys_users" WHERE "id" = $1`,
			want: []string{"sys_users"},
		},
		{
			name: "joins",
			sql:  `SELECT * FROM "sys_user_org_units" AS u0 JOIN "sys_org_units" AS u1 ON u0."org" = u1."id" LEFT JOIN "sys_positions" p ON u1."id" = p."org"`,
			want: []string{"sys_user_org_units", "sys_org_units", "sys_positions"},
		},
		{
			name: "from comma list",
			sql:  `SELECT * FROM "a", "b" WHERE x = y`,
			want: []string{"a", "b"},
		},
		{
			name: "subquery paren skipped",
			sql:  `SELECT * FROM (SELECT "id" FROM "sys_users") AS t WHERE t."id" = ***`,
			want: []string{"sys_users"},
		},
		{
			name: "dedupe",
			sql:  `SELECT * FROM "sys_users" JOIN "sys_roles" ON 1=1 JOIN "sys_users" ON 2=2`,
			want: []string{"sys_users", "sys_roles"},
		},
		{
			name: "schema qualified keeps full name",
			sql: `SELECT * FROM public."sys_users"`,
			want: []string{"public.sys_users"},
		},
		{
			name: "bare unquoted",
			sql:  `select * from sys_users where id = 1`,
			want: []string{"sys_users"},
		},
		{
			name: "no from ddl-ish",
			sql:  `SHOW server_version`,
			want: nil,
		},
		{
			name: "masked literals present",
			sql:  `SELECT "t"."a" FROM "t" WHERE "t"."name" = *** AND "t"."n" > ***`,
			want: []string{"t"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// 提取应同时工作于原始与脱敏文本
			for _, input := range []string{tc.sql, MaskSQL(tc.sql)} {
				got := ExtractTables(input)
				if len(got) == 0 && len(tc.want) == 0 {
					continue
				}
				if !reflect.DeepEqual(got, tc.want) {
					t.Errorf("ExtractTables(%q) = %v, want %v", input, got, tc.want)
				}
			}
		})
	}
}

func TestClassifyTable(t *testing.T) {
	cases := map[string]string{
		"sys_users":                  CategoryUserData,
		`"sys_user_credentials"`:     CategoryUserData,
		"sys_user_mfa_factors":       CategoryUserData,
		"sys_org_units":              CategoryOrgData,
		"sys_positions":              CategoryOrgData,
		"sys_membership_roles":       CategoryOrgData,
		"sys_roles":                  CategoryAccessControl,
		"sys_role_permissions":       CategoryAccessControl,
		"sys_permissions":            CategoryAccessControl,
		"sys_permission_groups":      CategoryAccessControl,
		"sys_menus":                  CategoryAccessControl,
		"sys_apis":                   CategoryAccessControl,
		"sys_login_policies":         CategoryAccessControl,
		"sys_tenants":                CategoryTenantData,
		"sys_plans":                  CategoryTenantData,
		"sys_plan_quotas":            CategoryTenantData,
		"sys_internal_messages":      CategoryMessage,
		"sys_internal_message_recipients": CategoryMessage,
		"sys_data_access_audit_logs": CategoryAuditLog,
		"sys_permission_audit_logs":  CategoryAuditLog,
		"sys_policy_evaluation_logs": CategoryAuditLog,
		"sys_dict_entries":           CategorySystemConfig,
		"sys_languages":              CategorySystemConfig,
		"sys_files":                  CategorySystemConfig,
		"sys_tasks":                  CategorySystemConfig,
		"order_info":                 CategoryUnknown,
	}
	for table, want := range cases {
		if got := ClassifyTable(table); got != want {
			t.Errorf("ClassifyTable(%q) = %v, want %v", table, got, want)
		}
	}
}
