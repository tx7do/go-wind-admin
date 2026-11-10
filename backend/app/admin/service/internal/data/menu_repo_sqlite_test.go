package data

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// TestMenuRepoSqlite_EnumReadViewBackfill 验证菜单的 status/type/module 三个
// 枚举字段在 Get（按主键）与 List 读视图中的如实回显：
//   - 显式指定的枚举值（行 A/B）按写入值回显；
//   - 未显式指定的 status/type（行 C）按列默认（ON/MENU）落库并回显；
//   - module 列无可默认值，行 C 未指定即 NULL，读视图保持缺省（无可回填值）。
//
// 历史缺陷：DTO 侧三者均为可选指针字段，mapper 的枚举转换对（值↔值）无法
// 赋入指针字段而直接丢弃，读视图恒呈零值；仓内现经 queryEnumsAndBackfill /
// backfillEnumsFrom 统一回填。仓构造器复用 menu_repo_sync_test.go 的
// newMenuRepoSqlite。
func TestMenuRepoSqlite_EnumReadViewBackfill(t *testing.T) {
	repo := newMenuRepoSqlite(t)
	ctx := enttest.NewSystemContext(context.Background())

	// 行 A：status/type/module 全部显式指定
	require.NoError(t, repo.Create(ctx, &permissionV1.CreateMenuRequest{
		Data: &permissionV1.Menu{
			Path:   trans.Ptr("/sqlite/menu/enum/a"),
			Status: permissionV1.Menu_ON.Enum(),
			Type:   permissionV1.Menu_CATALOG.Enum(),
			Module: identityV1.Module_DASHBOARD.Enum(),
		},
	}), "行 A 创建应成功")
	// 行 B：另一组显式枚举值
	require.NoError(t, repo.Create(ctx, &permissionV1.CreateMenuRequest{
		Data: &permissionV1.Menu{
			Path:   trans.Ptr("/sqlite/menu/enum/b"),
			Status: permissionV1.Menu_OFF.Enum(),
			Type:   permissionV1.Menu_LINK.Enum(),
			Module: identityV1.Module_TASK.Enum(),
		},
	}), "行 B 创建应成功")
	// 行 C：枚举字段全部缺省（status/type 走列默认，module 为 NULL）
	require.NoError(t, repo.Create(ctx, &permissionV1.CreateMenuRequest{
		Data: &permissionV1.Menu{
			Path: trans.Ptr("/sqlite/menu/enum/c"),
		},
	}), "行 C 创建应成功")

	rows, err := repo.entClient.Client().Menu.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 3, "应有 3 条菜单记录")
	// 期望的读视图按行路径区分：A/B 为写入值，C 为列默认（module 保持缺省）。
	expect := map[string]struct {
		status permissionV1.Menu_Status
		typ    permissionV1.Menu_Type
		module identityV1.Module
	}{
		"/sqlite/menu/enum/a": {permissionV1.Menu_ON, permissionV1.Menu_CATALOG, identityV1.Module_DASHBOARD},
		"/sqlite/menu/enum/b": {permissionV1.Menu_OFF, permissionV1.Menu_LINK, identityV1.Module_TASK},
		"/sqlite/menu/enum/c": {permissionV1.Menu_ON, permissionV1.Menu_MENU, identityV1.Module_MODULE_UNSPECIFIED},
	}
	idByPath := map[string]int64{}
	for _, row := range rows {
		idByPath[*row.Path] = int64(row.ID)
	}
	require.Len(t, idByPath, 3, "三行应有互异的路径")

	// Get（按主键）读视图：枚举字段经回填如实呈现。
	for path, id := range idByPath {
		got, getErr := repo.Get(ctx, &permissionV1.GetMenuRequest{
			QueryBy: &permissionV1.GetMenuRequest_Id{Id: id},
		})
		require.NoError(t, getErr, "按主键读取行 %s 应命中", path)
		exp := expect[path]
		require.Equal(t, exp.status, got.GetStatus(), "行 %s 读视图应回显 status", path)
		require.Equal(t, exp.typ, got.GetType(), "行 %s 读视图应回显 type", path)
		if exp.module == identityV1.Module_MODULE_UNSPECIFIED {
			require.Equal(t, identityV1.Module_MODULE_UNSPECIFIED, got.GetModule(),
				"行 %s 的 module 列为 NULL，读视图应保持缺省（无可回填值）", path)
		} else {
			require.Equal(t, exp.module, got.GetModule(), "行 %s 读视图应回显写入的 module", path)
		}
	}

	// List（扁平，不树化）读视图：同上，全量行经回填如实呈现。
	listed, err := repo.List(ctx, &paginationV1.PagingRequest{}, false)
	require.NoError(t, err)
	require.Equal(t, int64(3), listed.Total, "无过滤时应统计全部 3 行")
	require.Len(t, listed.Items, 3, "无过滤时应返回 3 行")
	viewByID := map[int64]*permissionV1.Menu{}
	for _, item := range listed.Items {
		viewByID[item.GetId()] = item
	}
	require.Len(t, viewByID, 3, "读视图应覆盖全部 3 行")
	for path, id := range idByPath {
		item := viewByID[id]
		require.NotNil(t, item, "读视图应包含行 %s", path)
		exp := expect[path]
		require.Equal(t, exp.status, item.GetStatus(), "行 %s 列表读视图应回显 status", path)
		require.Equal(t, exp.typ, item.GetType(), "行 %s 列表读视图应回显 type", path)
		if exp.module == identityV1.Module_MODULE_UNSPECIFIED {
			require.Equal(t, identityV1.Module_MODULE_UNSPECIFIED, item.GetModule(),
				"行 %s 的 module 列为 NULL，列表读视图应保持缺省", path)
		} else {
			require.Equal(t, exp.module, item.GetModule(), "行 %s 列表读视图应回显写入的 module", path)
		}
	}
}
