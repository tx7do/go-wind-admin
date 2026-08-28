package audit

import "testing"

func TestMaskSQL(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "字符串字面量脱敏",
			in:   "SELECT id FROM t WHERE email = 'a@b.c'",
			want: "SELECT id FROM t WHERE email = ***",
		},
		{
			name: "双单引号转义整体脱敏",
			in:   "WHERE n = 'O''Brien'",
			want: "WHERE n = ***",
		},
		{
			name: "反斜杠转义整体脱敏",
			in:   `WHERE n = 'O\'Brien'`,
			want: "WHERE n = ***",
		},
		{
			name: "E 转义串",
			in:   `WHERE a = E'x\ny'`,
			want: "WHERE a = ***",
		},
		{
			name: "数值字面量脱敏",
			in:   "WHERE age > 25 AND score = 1.5e-3",
			want: "WHERE age > *** AND score = ***",
		},
		{
			name: "占位符保留",
			in:   `WHERE id = $1 AND t."uid" = $12`,
			want: `WHERE id = $1 AND t."uid" = $12`,
		},
		{
			name: "双引号标识符保留",
			in:   `SELECT "sys_users"."password" FROM "sys_users"`,
			want: `SELECT "sys_users"."password" FROM "sys_users"`,
		},
		{
			name: "行注释原样且引号不误判",
			in:   "SELECT 1 -- don't leak 'x'\nFROM t",
			want: "SELECT *** -- don't leak 'x'\nFROM t",
		},
		{
			name: "嵌套块注释原样",
			in:   `/* 'a' /* 'b' */ 'c' */ SELECT 'd'`,
			want: `/* 'a' /* 'b' */ 'c' */ SELECT ***`,
		},
		{
			name: "美元引用空标签整体脱敏",
			in:   "DO $$ BEGIN RAISE 'hi'; END $$",
			want: "DO ***",
		},
		{
			name: "美元引用带标签整体脱敏",
			in:   "$fn$ body 'q' $fn$",
			want: "***",
		},
		{
			name: "INSERT VALUES 与 RETURNING",
			in:   "INSERT INTO t (a,b) VALUES ('x', 42) RETURNING id",
			want: "INSERT INTO t (a,b) VALUES (***, ***) RETURNING id",
		},
		{
			name: "类型转换保留",
			in:   "WHERE x = 'a'::text AND n = 5::int",
			want: "WHERE x = ***::text AND n = ***::int",
		},
		{
			name: "空串脱敏",
			in:   "SET a = ''",
			want: "SET a = ***",
		},
		{
			name: "十六进制串",
			in:   "INSERT INTO t VALUES (X'deadbeef')",
			want: "INSERT INTO t VALUES (X***)",
		},
		{
			name: "LIMIT/OFFSET 数值",
			in:   "SELECT * FROM t LIMIT 10 OFFSET 0",
			want: "SELECT * FROM t LIMIT *** OFFSET ***",
		},
		{
			name: "未闭合串脱敏到末尾",
			in:   "WHERE a = 'abc",
			want: "WHERE a = ***",
		},
		{
			name: "UPDATE SET",
			in:   "UPDATE users SET nickname = '喵', age = 30 WHERE id = 5",
			want: "UPDATE users SET nickname = ***, age = *** WHERE id = ***",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := MaskSQL(tc.in); got != tc.want {
				t.Errorf("MaskSQL(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// 指纹稳定性：字面量不同、结构相同的 SQL 脱敏后应一致（digest 分组语义）。
func TestMaskSQLFingerprint(t *testing.T) {
	a := MaskSQL("SELECT * FROM users WHERE email = 'alice@x.com' AND age > 20")
	b := MaskSQL("SELECT * FROM users WHERE email = 'bob@y.org' AND age > 35")
	if a != b {
		t.Errorf("同构 SQL 脱敏后不一致: %q vs %q", a, b)
	}
}
