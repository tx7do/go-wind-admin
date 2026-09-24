package data

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newBackupRepoSqlite 用 enttest helper 构造 BackupRepo，
// 逐字段复刻 NewBackupRepo（log 换 NopLogger；该 repo 无 init()）。
func newBackupRepoSqlite(t *testing.T) *BackupRepo {
	t.Helper()
	return &BackupRepo{
		entClient: enttest.NewEntClientForTest(t),
		log:       bLogger.NewHelper(bLogger.NopLogger()),
	}
}

// backupTableNames ExportCoreTables 导出的核心表键集合。
var backupTableNames = []string{
	"tenants", "users", "roles", "permissions", "memberships", "org_units", "positions", "menus",
}

// lenOfExported 导出值的长度（切片经 any 装载，用反射取 len）。
func lenOfExported(t *testing.T, v any) int {
	t.Helper()
	rv := reflect.ValueOf(v)
	require.Equal(t, reflect.Slice, rv.Kind(), "导出值应为切片")
	return rv.Len()
}

// TestBackupRepoSqlite_IsConfigured 验证 IsConfigured：
// 正常构造（entClient 就绪）为 true；零值与 nil 接收者为 false。
func TestBackupRepoSqlite_IsConfigured(t *testing.T) {
	repo := newBackupRepoSqlite(t)
	require.True(t, repo.IsConfigured(), "entClient 就绪时 IsConfigured 应为 true")

	var zero BackupRepo
	require.False(t, zero.IsConfigured(), "零值 BackupRepo（无 entClient）应为 false")

	var nilRepo *BackupRepo
	require.False(t, nilRepo.IsConfigured(), "nil 接收者应为 false")
}

// TestBackupRepoSqlite_ExportEmptyTables 验证空库导出：
// 八张核心表键全部存在，各导出切片长度为 0，无错误（空表不是失败）。
func TestBackupRepoSqlite_ExportEmptyTables(t *testing.T) {
	repo := newBackupRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	exported, err := repo.ExportCoreTables(ctx)
	require.NoError(t, err, "空库导出应成功")
	require.Len(t, exported, len(backupTableNames), "导出 map 应包含全部 8 张核心表键")
	for _, name := range backupTableNames {
		require.Contains(t, exported, name, "导出 map 应含键 %q", name)
		require.Zero(t, lenOfExported(t, exported[name]), "空表 %q 的导出切片应为空", name)
	}
}

// TestBackupRepoSqlite_ExportSeededTables 验证含数据导出：
// 已播种表导出全部行，未播种表导出空切片，八键齐全。
func TestBackupRepoSqlite_ExportSeededTables(t *testing.T) {
	repo := newBackupRepoSqlite(t)
	client := repo.entClient.Client()
	ctx := enttest.NewSystemViewerCtx(context.Background())

	// 播种六张可直接最小化建行的表（membership/menu 留空，
	// 用于验证空表在含数据导出中仍产出空切片键）。
	require.NoError(t, client.Tenant.Create().
		SetName("sqlite_backup_tenant").
		SetCode("SQLITE_BACKUP_TENANT").
		Exec(ctx), "直插 tenant 行应成功")
	require.NoError(t, client.User.Create().
		SetUsername("sqlite_backup_user_a").
		Exec(ctx), "直插 user 行应成功")
	require.NoError(t, client.User.Create().
		SetUsername("sqlite_backup_user_b").
		Exec(ctx), "直插 user 行应成功")
	require.NoError(t, client.Role.Create().
		SetName("sqlite_backup_role").
		SetCode("SQLITE_BACKUP_ROLE").
		Exec(ctx), "直插 role 行应成功")
	require.NoError(t, client.Permission.Create().
		SetName("sqlite_backup_perm").
		SetCode("SQLITE_BACKUP_PERM").
		Exec(ctx), "直插 permission 行应成功")
	orgUnit, err := client.OrgUnit.Create().
		SetName(fmt.Sprintf("sqlite_backup_org_unit_%d", 8001)).
		Save(ctx)
	require.NoError(t, err, "直插 org_unit 行应成功")
	// position 的 org_unit_id 为必填外键：挂到刚建的 org_unit 上
	require.NoError(t, client.Position.Create().
		SetName("sqlite_backup_position").
		SetCode(fmt.Sprintf("SQLITE_BACKUP_POS_%d", 8001)).
		SetOrgUnitID(orgUnit.ID).
		Exec(ctx), "直插 position 行应成功")

	expected := map[string]int{
		"tenants":     1,
		"users":       2,
		"roles":       1,
		"permissions": 1,
		"memberships": 0,
		"org_units":   1,
		"positions":   1,
		"menus":       0,
	}

	exported, err := repo.ExportCoreTables(ctx)
	require.NoError(t, err, "含数据导出应成功")
	require.Len(t, exported, len(backupTableNames), "导出 map 应包含全部 8 张核心表键")
	for _, name := range backupTableNames {
		require.Contains(t, exported, name, "导出 map 应含键 %q", name)
		require.Equal(t, expected[name], lenOfExported(t, exported[name]),
			"表 %q 的导出行数应与库内行数一致", name)
	}
}
