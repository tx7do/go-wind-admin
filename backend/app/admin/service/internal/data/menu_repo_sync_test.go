package data

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/menu"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newMenuRepoSqlite 用 enttest helper 白盒构造一个可直接做 CRUD 的 MenuRepo
// （同 position_repo_sqlite_test.go 的套路）。
func newMenuRepoSqlite(t *testing.T) *MenuRepo {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	repo := &MenuRepo{
		entClient:       entClient,
		log:             bLogger.NewHelper(bLogger.NopLogger()),
		mapper:          mapper.NewCopierMapper[permissionV1.Menu, ent.Menu](),
		statusConverter: mapper.NewEnumTypeConverter[permissionV1.Menu_Status, menu.Status](permissionV1.Menu_Status_name, permissionV1.Menu_Status_value),
		typeConverter:   mapper.NewEnumTypeConverter[permissionV1.Menu_Type, menu.Type](permissionV1.Menu_Type_name, permissionV1.Menu_Type_value),
		moduleConverter: mapper.NewEnumTypeConverter[identityV1.Module, menu.Module](identityV1.Module_name, identityV1.Module_value),
	}
	repo.init()
	return repo
}

// TestMenuRepoSyncMenus_MergePreservesIDs 增量合并：按全路径匹配，已存在菜单原位更新
// （ID 与 status 保留），缺失才新增——这是"同步不废角色-菜单授权"的关键语义。
func TestMenuRepoSyncMenus_MergePreservesIDs(t *testing.T) {
	repo := newMenuRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	// 种子：/system 目录（手工停用）+ 子菜单 dict
	require.NoError(t, repo.Create(ctx, &permissionV1.CreateMenuRequest{Data: &permissionV1.Menu{
		Name:   trans.Ptr("System"),
		Path:   trans.Ptr("/system"),
		Type:   permissionV1.Menu_CATALOG.Enum(),
		Status: permissionV1.Menu_OFF.Enum(),
	}}))
	systemID := mustMenuIDByName(t, repo, ctx, "System")
	require.NoError(t, repo.Create(ctx, &permissionV1.CreateMenuRequest{Data: &permissionV1.Menu{
		Name:     trans.Ptr("DictManagement"),
		Path:     trans.Ptr("dict"),
		Type:     permissionV1.Menu_MENU.Enum(),
		ParentId: trans.Ptr(systemID),
	}}))
	dictID := mustMenuIDByName(t, repo, ctx, "DictManagement")

	// MERGE：/system 原位更新（补 component/meta）、dict 原位更新、configs 新增
	payload := []*permissionV1.Menu{{
		Name:      trans.Ptr("System"),
		Path:      trans.Ptr("/system"),
		Type:      permissionV1.Menu_CATALOG.Enum(),
		Component: trans.Ptr("BasicLayout"),
		Meta:      &permissionV1.MenuMeta{Title: trans.Ptr("系统管理"), Order: trans.Ptr(int32(2005))},
		Children: []*permissionV1.Menu{
			{
				Name:      trans.Ptr("DictManagement"),
				Path:      trans.Ptr("dict"),
				Type:      permissionV1.Menu_MENU.Enum(),
				Component: trans.Ptr("app/system/dict/index.vue"),
			},
			{
				Name:      trans.Ptr("ConfigManagement"),
				Path:      trans.Ptr("configs"),
				Type:      permissionV1.Menu_MENU.Enum(),
				Component: trans.Ptr("app/system/config/index.vue"),
			},
		},
	}}

	count, err := repo.SyncMenus(ctx, payload, permissionV1.SyncMenusRequest_MERGE, 9)
	require.NoError(t, err)
	require.Equal(t, 3, count, "合并应更新 2 条 + 新增 1 条")

	// 已存在菜单 ID 不变（角色-菜单授权得以保留）
	require.Equal(t, systemID, mustMenuIDByName(t, repo, ctx, "System"), "/system 应原位更新保留 ID")
	require.Equal(t, dictID, mustMenuIDByName(t, repo, ctx, "DictManagement"), "dict 应原位更新保留 ID")

	// 更新不触碰 status：手工停用的菜单不应被同步重新启用
	off, err := repo.entClient.Client().Menu.Query().Where(menu.IDEQ(systemID)).Only(ctx)
	require.NoError(t, err)
	require.NotNil(t, off.Status)
	require.Equal(t, menu.StatusOff, *off.Status, "MERGE 更新不应改写 status")

	// 新增菜单挂到正确的父节点下
	configs, err := repo.entClient.Client().Menu.Query().Where(menu.NameEQ("ConfigManagement")).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, systemID, *configs.ParentID, "configs 应挂在 /system 下")
	require.Equal(t, "app/system/config/index.vue", derefStrP(configs.Component))
	require.NotNil(t, configs.Module)
	require.Equal(t, menu.ModuleSystem, *configs.Module, "component 应归类到 SYSTEM 模块")
}

// TestMenuRepoSyncMenus_RebuildReplace 全量重建：清空后重建，旧 ID 全部变化（兼容旧行为）。
func TestMenuRepoSyncMenus_RebuildReplace(t *testing.T) {
	repo := newMenuRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &permissionV1.CreateMenuRequest{Data: &permissionV1.Menu{
		Name: trans.Ptr("Old"),
		Path: trans.Ptr("/old"),
		Type: permissionV1.Menu_CATALOG.Enum(),
	}}))
	oldID := mustMenuIDByName(t, repo, ctx, "Old")

	payload := []*permissionV1.Menu{{
		Name: trans.Ptr("Fresh"),
		Path: trans.Ptr("/fresh"),
		Type: permissionV1.Menu_CATALOG.Enum(),
	}}
	count, err := repo.SyncMenus(ctx, payload, permissionV1.SyncMenusRequest_REPLACE, 9)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	_, err = repo.entClient.Client().Menu.Query().Where(menu.IDEQ(oldID)).Only(ctx)
	require.Error(t, err, "REPLACE 后旧菜单行应不存在")

	fresh, err := repo.entClient.Client().Menu.Query().Where(menu.NameEQ("Fresh")).Only(ctx)
	require.NoError(t, err)
	require.NotEqual(t, oldID, fresh.ID)
}

func mustMenuIDByName(t *testing.T, repo *MenuRepo, ctx context.Context, name string) uint32 {
	t.Helper()
	row, err := repo.entClient.Client().Menu.Query().Where(menu.NameEQ(name)).Only(ctx)
	require.NoError(t, err)
	return row.ID
}
