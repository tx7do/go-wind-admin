package data

// api_endpoint_migrate 的纯函数与迁移端到端测试。
//
// 迁移是数据破坏性操作（回填快照、删行、摘列摘索引），必须钉死两类不变量：
//   - 全路径拼接与 menu_repo.joinMenuFullPath **逐字一致**（身份键两侧同源，
//     规则漂移 = 绑定反查错位）——特别是**绝对路径分支**：path 以 "/" 开头时
//     脱离父链（/dashboard 下的 /analytics 是顶层路由），漏掉会拼出
//     //parent//child 错位路径（真机 e2e 抓到过的坑）；
//   - 迁移幂等：重复执行零操作；孤儿/重复组处理方向正确（保最小绑定 id）。

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJoinMenuFullPathCanonical(t *testing.T) {
	cases := []struct {
		name   string
		parent string
		path   string
		want   string
	}{
		{"根节点绝对路径", "", "/dashboard", "/dashboard"},
		{"根下相对段", "", "members", "/members"},
		{"父为斜杠", "/", "members", "/members"},
		{"父链拼接", "/system", "dict", "/system/dict"},
		{"父链尾斜杠修剪", "/system/", "dict", "/system/dict"},
		{"子菜单绝对路径脱离父链", "/dashboard", "/analytics", "/analytics"},
		{"深链绝对路径脱离父链", "/a/b/c", "/x", "/x"},
		{"空 path 返回父全路径", "/system", "", "/system"},
		{"两侧空白修剪", "/system", "  dict  ", "/system/dict"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, joinMenuFullPathCanonical(tc.parent, tc.path))
		})
	}
}

func TestWalkMenuFullPaths(t *testing.T) {
	rows := []migrateMenuRow{
		{id: 1, parentID: 0, path: "/dashboard"},
		{id: 2, parentID: 1, path: "/analytics"}, // 绝对路径子菜单：full path 脱离父链
		{id: 3, parentID: 1, path: "workspace"},  // 相对段：正常拼接
		{id: 4, parentID: 2, path: "deep"},       // 绝对路径子菜单的子级：挂在脱离后的链上
		{id: 5, parentID: 0, path: "/system"},
		{id: 6, parentID: 5, path: ""},
	}

	byID := walkMenuFullPaths(rows)

	require.Equal(t, "/dashboard", byID[1])
	require.Equal(t, "/analytics", byID[2], "绝对路径子菜单的 full path 必须脱离父链（真机坑回归）")
	require.Equal(t, "/dashboard/workspace", byID[3], "相对段按父链拼接")
	require.Equal(t, "/analytics/deep", byID[4], "绝对路径子菜单的后代沿脱离后的链拼接")
	require.Equal(t, "/system", byID[5])
	require.Equal(t, "/system", byID[6], "空 path 行返回父全路径")
}

func TestWalkMenuFullPathsCycleGuard(t *testing.T) {
	// 成环脏数据：2 -> 3 -> 2，防护 = visited 集合，遍历终止不死循环
	rows := []migrateMenuRow{
		{id: 1, parentID: 0, path: "/root"},
		{id: 2, parentID: 3, path: "b"},
		{id: 3, parentID: 2, path: "c"},
	}
	require.NotPanics(t, func() {
		byID := walkMenuFullPaths(rows)
		require.Contains(t, byID, int64(1))
	})
}

// newMigrationSQLiteDB 起一个独立 modernc SQLite 内存库（不经 enttest，模拟
// "旧 schema 形态的存量库"——迁移函数的输入就是这种形态）。
func newMigrationSQLiteDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:enttest_migrate_probe_"+t.Name()+"?mode=memory&cache=shared&_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	// 旧形态 schema：sys_permission_menus 带 menu_id、sys_menus 带父链列
	_, err = db.ExecContext(context.Background(), `
CREATE TABLE sys_menus (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    parent_id INTEGER,
    path TEXT
);
CREATE TABLE sys_apis (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT,
    method TEXT
);
CREATE TABLE sys_permission_apis (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    permission_id INTEGER,
    api_id INTEGER,
    path TEXT,
    method TEXT
);
CREATE TABLE sys_permission_menus (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    permission_id INTEGER,
    menu_id INTEGER,
    full_path TEXT
);`)
	require.NoError(t, err)
	return db
}

func TestMigratePermissionMenuFullPaths_BackfillAndDrop(t *testing.T) {
	db := newMigrationSQLiteDB(t)
	ctx := context.Background()

	// 旧形态数据：3 级菜单树 + 一条绝对路径子菜单 + 一条孤儿绑定（menu_id 无主）
	seed := []struct {
		id       int
		parentID int
		path     string
	}{
		{1, 0, "/dashboard"},
		{2, 1, "/analytics"}, // 绝对路径：full path = /analytics
		{3, 1, "workspace"},  // 相对段：full path = /dashboard/workspace
		{4, 3, "detail"},     // 相对段：full path = /dashboard/workspace/detail
	}
	for _, m := range seed {
		_, err := db.ExecContext(ctx, `INSERT INTO sys_menus (id, parent_id, path) VALUES (?, ?, ?)`,
			m.id, m.parentID, m.path)
		require.NoError(t, err)
	}
	bindings := []struct {
		permID int
		menuID int
	}{
		{100, 1}, // -> /dashboard
		{100, 2}, // -> /analytics（绝对路径形态回归）
		{101, 3}, // -> /dashboard/workspace
		{102, 4}, // -> /dashboard/workspace/detail
		{103, 999},
	}
	for _, b := range bindings {
		_, err := db.ExecContext(ctx,
			`INSERT INTO sys_permission_menus (permission_id, menu_id) VALUES (?, ?)`,
			b.permID, b.menuID)
		require.NoError(t, err)
	}

	require.NoError(t, MigratePermissionMenuFullPaths(ctx, db, "sqlite3"))

	// 回填结果：全路径快照按身份键规则落位（(permission, full_path) 对集合断言，
	// 同一 permission 可绑多条菜单）
	want := map[int]map[string]struct{}{
		100: {"/dashboard": {}, "/analytics": {}}, // menu 2 是绝对路径子菜单，full path 脱离父链
		101: {"/dashboard/workspace": {}},
		102: {"/dashboard/workspace/detail": {}},
	}
	rows, err := db.QueryContext(ctx, `SELECT permission_id, full_path FROM sys_permission_menus ORDER BY permission_id`)
	require.NoError(t, err)
	got := map[int]map[string]struct{}{}
	for rows.Next() {
		var pid int
		var fp string
		require.NoError(t, rows.Scan(&pid, &fp))
		if got[pid] == nil {
			got[pid] = map[string]struct{}{}
		}
		got[pid][fp] = struct{}{}
	}
	require.NoError(t, rows.Close())
	require.Equal(t, want, got, "绑定快照应按身份键规则回填")

	// 孤儿（menu_id 999 无主）被删除：剩余行 = 回填行总数。want 按 permission
	// 分组，同一 permission 可绑多条菜单（perm 100 两条），总行数要按路径数累计
	wantRows := 0
	for _, paths := range want {
		wantRows += len(paths)
	}
	left, err := db.QueryContext(ctx, `SELECT count(*) FROM sys_permission_menus`)
	require.NoError(t, err)
	for left.Next() {
		var n int
		require.NoError(t, left.Scan(&n))
		require.Equal(t, wantRows, n, "孤儿绑定应被删除，其余绑定原样保留")
	}
	require.NoError(t, left.Close())

	// 旧列摘除、新列保留
	var menuIDCol int
	require.NoError(t, db.QueryRowContext(ctx,
		`SELECT count(*) FROM pragma_table_info('sys_permission_menus') WHERE name='menu_id'`).Scan(&menuIDCol))
	require.Zero(t, menuIDCol, "旧 menu_id 列应被摘除")
	var fullPathCol int
	require.NoError(t, db.QueryRowContext(ctx,
		`SELECT count(*) FROM pragma_table_info('sys_permission_menus') WHERE name='full_path'`).Scan(&fullPathCol))
	require.Equal(t, 1, fullPathCol, "full_path 列应保留")

	// 幂等：二次执行零操作、不报错
	require.NoError(t, MigratePermissionMenuFullPaths(ctx, db, "sqlite3"))
}

func TestMigratePermissionMenuFullPaths_DedupesSamePathBindings(t *testing.T) {
	db := newMigrationSQLiteDB(t)
	ctx := context.Background()

	// 两组菜单行同路径（脏数据）+ 同权限绑定两者 → (perm, full_path) 唯一键冲突，
	// 迁移必须保最小绑定 id、删重复组
	_, err := db.ExecContext(ctx, `
INSERT INTO sys_menus (id, parent_id, path) VALUES (1, 0, '/dup');
INSERT INTO sys_menus (id, parent_id, path) VALUES (2, 0, '/dup');
INSERT INTO sys_permission_menus (id, permission_id, menu_id) VALUES (10, 200, 1);
INSERT INTO sys_permission_menus (id, permission_id, menu_id) VALUES (11, 200, 2);`)
	require.NoError(t, err)

	require.NoError(t, MigratePermissionMenuFullPaths(ctx, db, "sqlite3"))

	var cnt int
	require.NoError(t, db.QueryRowContext(ctx,
		`SELECT count(*) FROM sys_permission_menus WHERE permission_id = 200 AND full_path = '/dup'`).Scan(&cnt))
	require.Equal(t, 1, cnt, "同 (permission, full_path) 的重复组只留最小绑定 id")

	var minID int
	require.NoError(t, db.QueryRowContext(ctx,
		`SELECT min(id) FROM sys_permission_menus WHERE permission_id = 200`).Scan(&minID))
	require.Equal(t, 10, minID, "保留的应是绑定 id 较小的一行")
}

func TestMigratePermissionApiEndpoints_Backfill(t *testing.T) {
	db := newMigrationSQLiteDB(t)
	ctx := context.Background()

	// 旧形态：api_id 绑定 + api 行（含 method 大小写原样）+ 一条孤儿绑定
	_, err := db.ExecContext(ctx, `
INSERT INTO sys_apis (id, path, method) VALUES (1, '/admin/v1/orders', 'GET');
INSERT INTO sys_apis (id, path, method) VALUES (2, '/admin/v1/orders', 'POST');
INSERT INTO sys_permission_apis (permission_id, api_id) VALUES (300, 1);
INSERT INTO sys_permission_apis (permission_id, api_id) VALUES (301, 2);
INSERT INTO sys_permission_apis (permission_id, api_id) VALUES (302, 999);`)
	require.NoError(t, err)

	require.NoError(t, MigratePermissionApiEndpoints(ctx, db, "sqlite3"))

	var pairA, pairB, orphans int
	require.NoError(t, db.QueryRowContext(ctx,
		`SELECT count(*) FROM sys_permission_apis WHERE permission_id=300 AND path='/admin/v1/orders' AND method='GET'`).Scan(&pairA))
	require.NoError(t, db.QueryRowContext(ctx,
		`SELECT count(*) FROM sys_permission_apis WHERE permission_id=301 AND path='/admin/v1/orders' AND method='POST'`).Scan(&pairB))
	require.NoError(t, db.QueryRowContext(ctx,
		`SELECT count(*) FROM sys_permission_apis WHERE permission_id=302`).Scan(&orphans))
	require.Equal(t, 1, pairA, "api_id 绑定应回填为端点快照")
	require.Equal(t, 1, pairB)
	require.Zero(t, orphans, "api_id 无主的孤儿绑定应被删除")

	// 幂等：二次执行零操作
	require.NoError(t, MigratePermissionApiEndpoints(ctx, db, "sqlite3"))
}

func TestDedupeSysApisByIdentity(t *testing.T) {
	db := newMigrationSQLiteDB(t)
	ctx := context.Background()

	// 同 (path, method) 重复 + NULL 身份行（不参与去重）
	_, err := db.ExecContext(ctx, `
INSERT INTO sys_apis (id, path, method) VALUES (1, '/a', 'GET');
INSERT INTO sys_apis (id, path, method) VALUES (2, '/a', 'GET');
INSERT INTO sys_apis (id, path, method) VALUES (3, '/a', 'GET');
INSERT INTO sys_apis (id, path, method) VALUES (4, '/b', 'POST');
INSERT INTO sys_apis (id, path, method) VALUES (5, NULL, NULL);`)
	require.NoError(t, err)

	removed, err := DedupeSysApisByIdentity(ctx, db, "sqlite3")
	require.NoError(t, err)
	require.Equal(t, int64(2), removed, "同键 3 行应去重掉 2 行（保留最小 id）")

	var ids []int
	got, err := db.QueryContext(ctx, `SELECT id FROM sys_apis ORDER BY id`)
	require.NoError(t, err)
	for got.Next() {
		var id int
		require.NoError(t, got.Scan(&id))
		ids = append(ids, id)
	}
	require.NoError(t, got.Close())
	require.Equal(t, []int{1, 4, 5}, ids, "保留每组最小 id；NULL 身份行不参与去重")
}
