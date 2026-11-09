// MenuRepo 树形行为的 SQLite 内存库白盒测试：
//   - 换父环检测：挂到自己/自己后代下必须拒绝（菜单无物化路径列，环会让
//     子树从根不可达——全路径身份键整块失明）；
//   - parent_id 显式 0 = 移到根级（落 NULL），不再被静默忽略；
//   - 移动/改名后子树授权绑定快照（sys_permission_menus.full_path）随行改写；
//   - 删除子树级联清绑定且事务化；
//   - 存量环脏数据下删除/换父校验正常收敛不死循环（visited 防护）。
package data

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	entCrud "github.com/tx7do/go-crud/entgo"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/menu"
	"go-wind-admin/app/admin/service/internal/data/ent/permissionmenu"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newMenuRepoTreeSqlite 在给定 enttest client 上构造 MenuRepo（经
// NewMenuRepoForTest：已装配同库 PermissionMenuRepo，删除级联与快照改写生效）。
func newMenuRepoTreeSqlite(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) (*MenuRepo, *ent.Client) {
	t.Helper()
	return NewMenuRepoForTest(entClient), entClient.Client()
}

// createMenuRow 直建一条菜单行（绕过 service 层校验），parentID nil = 根。
func createMenuRow(t *testing.T, ctx context.Context, client *ent.Client, name, path string, parentID *uint64) uint64 {
	t.Helper()
	builder := client.Menu.Create().SetName(name).SetPath(path).SetType(menu.TypeMenu)
	if parentID != nil {
		builder.SetParentID(*parentID)
	}
	row, err := builder.Save(ctx)
	require.NoError(t, err, "直建菜单行 [%s] 应成功", name)
	return row.ID
}

// createBindingRow 直建一条权限-菜单绑定（全路径快照）。
func createBindingRow(t *testing.T, ctx context.Context, client *ent.Client, permissionID uint64, fullPath string) {
	t.Helper()
	require.NoError(t, client.PermissionMenu.Create().
		SetPermissionID(permissionID).
		SetFullPath(fullPath).
		Exec(ctx), "直建绑定 [%s] 应成功", fullPath)
}

// bindingPaths 列出指定权限的全部绑定全路径（集合化便于断言）。
func bindingPaths(t *testing.T, ctx context.Context, client *ent.Client, permissionID uint64) map[string]struct{} {
	t.Helper()
	rows, err := client.PermissionMenu.Query().
		Where(permissionmenu.PermissionIDEQ(permissionID)).
		All(ctx)
	require.NoError(t, err)
	out := make(map[string]struct{}, len(rows))
	for _, r := range rows {
		if r.FullPath != nil {
			out[*r.FullPath] = struct{}{}
		}
	}
	return out
}

// TestMenuRepoSqlite_MoveUnderDescendantRejected 换父环检测：挂到自己后代或
// 自己下拒绝，且库内挂载关系不被改动。
func TestMenuRepoSqlite_MoveUnderDescendantRejected(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo, client := newMenuRepoTreeSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	a := createMenuRow(t, ctx, client, "链A", "/chain-a", nil)
	bID := createMenuRow(t, ctx, client, "链B", "chain-b", &a)
	c := createMenuRow(t, ctx, client, "链C", "chain-c", &bID)

	// A 挂到自己的后代 C 下 → 拒绝
	err := repo.Update(ctx, &permissionV1.UpdateMenuRequest{
		Id:   a,
		Data: &permissionV1.Menu{ParentId: &c},
	})
	require.Error(t, err, "挂到自己后代下必须拒绝")
	require.True(t, permissionV1.IsBadRequest(err), "应报 400 而非 500")

	// A 挂到自己下 → 拒绝
	err = repo.Update(ctx, &permissionV1.UpdateMenuRequest{
		Id:   a,
		Data: &permissionV1.Menu{ParentId: &a},
	})
	require.Error(t, err, "自挂必须拒绝")

	// 拒绝路径不得改动库内挂载关系
	rowA, err := client.Menu.Get(ctx, a)
	require.NoError(t, err)
	require.Nil(t, rowA.ParentID, "被拒更新不得改动 A 的根级挂载")
	rowB, err := client.Menu.Get(ctx, bID)
	require.NoError(t, err)
	require.NotNil(t, rowB.ParentID)
	require.Equal(t, a, *rowB.ParentID, "B 的父级应仍为 A")
}

// TestMenuRepoSqlite_ExplicitZeroParentMovesToRoot parent_id 显式 0 = 移到根级
// （落 NULL），不再被静默忽略；未设置（nil）= 不改挂载关系。
func TestMenuRepoSqlite_ExplicitZeroParentMovesToRoot(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo, client := newMenuRepoTreeSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	root := createMenuRow(t, ctx, client, "挂根根", "/to-root", nil)
	child := createMenuRow(t, ctx, client, "挂根子", "to-root-child", &root)

	// 未设置 parent_id：不改挂载
	require.NoError(t, repo.Update(ctx, &permissionV1.UpdateMenuRequest{
		Id:   child,
		Data: &permissionV1.Menu{Name: trans.Ptr("挂根子-改名")},
	}))
	rowChild, err := client.Menu.Get(ctx, child)
	require.NoError(t, err)
	require.NotNil(t, rowChild.ParentID, "未设置 parent_id 的更新不得改挂载关系")

	// 显式 0：移回根级（此前被静默忽略，"移到根"做不到）
	zero := uint64(0)
	require.NoError(t, repo.Update(ctx, &permissionV1.UpdateMenuRequest{
		Id:   child,
		Data: &permissionV1.Menu{ParentId: &zero},
	}))
	rowChild, err = client.Menu.Get(ctx, child)
	require.NoError(t, err)
	require.Nil(t, rowChild.ParentID, "显式 parent_id=0 应把子菜单移到根级（parent_id 落 NULL）")
}

// TestMenuRepoSqlite_RenameRefreshesBindingFullPaths 改名后本节点及子树的
// 授权绑定快照按旧全路径→新全路径改写。
func TestMenuRepoSqlite_RenameRefreshesBindingFullPaths(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo, client := newMenuRepoTreeSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	root := createMenuRow(t, ctx, client, "改名根", "/rename-root", nil)
	createMenuRow(t, ctx, client, "改名子", "rename-child", &root)

	perm, err := client.Permission.Create().SetName("改名绑定权限").SetCode("rename_binding_perm").Save(ctx)
	require.NoError(t, err)
	createBindingRow(t, ctx, client, perm.ID, "/rename-root")
	createBindingRow(t, ctx, client, perm.ID, "/rename-root/rename-child")
	createBindingRow(t, ctx, client, perm.ID, "/untouched")

	require.NoError(t, repo.Update(ctx, &permissionV1.UpdateMenuRequest{
		Id:   root,
		Data: &permissionV1.Menu{Path: trans.Ptr("/rename-root-moved")},
	}), "根菜单改名应成功")

	got := bindingPaths(t, ctx, client, perm.ID)
	require.Len(t, got, 3, "绑定总数不变（改写而非重建）")
	require.Contains(t, got, "/rename-root-moved", "根全路径快照应随改名改写")
	require.Contains(t, got, "/rename-root-moved/rename-child", "子树全路径快照应随改名改写")
	require.Contains(t, got, "/untouched", "无关绑定不得被牵连")
}

// TestMenuRepoSqlite_MoveRefreshesSubtreeBindingFullPaths 整棵子树移到新父下，
// 子树全部节点的绑定快照随新挂载点改写。注意子段必须是相对路径（不以 / 开头）：
// 绝对路径段按路由约定自成全路径、不随父链拼接（joinMenuFullPath 语义）。
func TestMenuRepoSqlite_MoveRefreshesSubtreeBindingFullPaths(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo, client := newMenuRepoTreeSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	src := createMenuRow(t, ctx, client, "搬移源", "move-src", nil)
	createMenuRow(t, ctx, client, "搬移子", "move-child", &src)
	dst := createMenuRow(t, ctx, client, "搬移目标", "/move-dst", nil)

	perm, err := client.Permission.Create().SetName("搬移绑定权限").SetCode("move_binding_perm").Save(ctx)
	require.NoError(t, err)
	createBindingRow(t, ctx, client, perm.ID, "/move-src")
	createBindingRow(t, ctx, client, perm.ID, "/move-src/move-child")

	require.NoError(t, repo.Update(ctx, &permissionV1.UpdateMenuRequest{
		Id:   src,
		Data: &permissionV1.Menu{ParentId: &dst},
	}), "子树移到新父下应成功")

	got := bindingPaths(t, ctx, client, perm.ID)
	require.Len(t, got, 2)
	require.Contains(t, got, "/move-dst/move-src", "移动后的根快照应带新父前缀")
	require.Contains(t, got, "/move-dst/move-src/move-child", "子树快照应整体随新挂载点改写")
}

// TestMenuRepoSqlite_DeleteCascadesBindings 事务化删除：子树行与授权绑定
// 同生共死，无关绑定不动。
func TestMenuRepoSqlite_DeleteCascadesBindings(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo, client := newMenuRepoTreeSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	root := createMenuRow(t, ctx, client, "级联根", "/cascade-root", nil)
	createMenuRow(t, ctx, client, "级联子", "cascade-child", &root)
	createMenuRow(t, ctx, client, "旁路菜单", "/side-menu", nil)

	perm, err := client.Permission.Create().SetName("级联绑定权限").SetCode("cascade_binding_perm").Save(ctx)
	require.NoError(t, err)
	createBindingRow(t, ctx, client, perm.ID, "/cascade-root")
	createBindingRow(t, ctx, client, perm.ID, "/cascade-root/cascade-child")
	createBindingRow(t, ctx, client, perm.ID, "/side-menu")

	require.NoError(t, repo.Delete(ctx, &permissionV1.DeleteMenuRequest{
		QueryBy: &permissionV1.DeleteMenuRequest_Id{Id: root},
	}), "删除子树应成功")

	menuCnt, err := client.Menu.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, menuCnt, "子树两行应删净，只剩旁路菜单")
	got := bindingPaths(t, ctx, client, perm.ID)
	require.Len(t, got, 1, "子树路径的绑定应级联清掉")
	require.Contains(t, got, "/side-menu", "无关绑定必须存活")
}

// TestMenuRepoSqlite_DeleteToleratesDirtyCycle 存量环脏数据（外部导入/直改库）
// 下删除正常收敛：子树按 parent 链内存收集 + visited 防护，不会像 UNION ALL
// 递归 CTE 那样无限递归挂死查询。
func TestMenuRepoSqlite_DeleteToleratesDirtyCycle(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo, client := newMenuRepoTreeSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	a := createMenuRow(t, ctx, client, "环A", "/cycle-a", nil)
	b := createMenuRow(t, ctx, client, "环B", "/cycle-b", nil)

	// 直改库构造 A↔B 互为父子的环（绕过一切校验）
	require.NoError(t, client.Menu.UpdateOneID(a).SetParentID(b).Exec(ctx))
	require.NoError(t, client.Menu.UpdateOneID(b).SetParentID(a).Exec(ctx))

	require.NoError(t, repo.Delete(ctx, &permissionV1.DeleteMenuRequest{
		QueryBy: &permissionV1.DeleteMenuRequest_Id{Id: a},
	}), "环脏数据下删除必须收敛返回而非挂死")

	menuCnt, err := client.Menu.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, menuCnt, "环上两节点应全部删除")
}

// TestMenuRepoSqlite_UpdateToleratesDirtyCycle 换父校验的 parent 链上行在
// 存量环上必须 visited 收敛：与本节点无关的环不放行也不死循环。
func TestMenuRepoSqlite_UpdateToleratesDirtyCycle(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo, client := newMenuRepoTreeSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	a := createMenuRow(t, ctx, client, "环A", "/cycle-a", nil)
	b := createMenuRow(t, ctx, client, "环B", "/cycle-b", nil)
	outsider := createMenuRow(t, ctx, client, "环外C", "/cycle-c", nil)

	require.NoError(t, client.Menu.UpdateOneID(a).SetParentID(b).Exec(ctx))
	require.NoError(t, client.Menu.UpdateOneID(b).SetParentID(a).Exec(ctx))

	// 环外节点挂到环上节点下：校验沿环上行，visited 断路后放行
	require.NoError(t, repo.Update(ctx, &permissionV1.UpdateMenuRequest{
		Id:   outsider,
		Data: &permissionV1.Menu{ParentId: &a},
	}), "与本节点无关的存量环不应阻塞正常换父，也不得死循环")

	row, err := client.Menu.Get(ctx, outsider)
	require.NoError(t, err)
	require.NotNil(t, row.ParentID)
	require.Equal(t, a, *row.ParentID, "换父应生效")
}
