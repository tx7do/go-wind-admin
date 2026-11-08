// PermissionMenuRepo 的 SQLite 内存库测试（白盒，包内测试）。
//
// 绑定存菜单全路径快照后，测试盯的是授权不漂移的三件事：
//   - 分配即解析：入参是菜单引用（id），落库的是全路径快照；引用不存在整体
//     400（fail-closed，不吞错）且零落库；
//   - 集合语义：重新分配 = 先清掉不在新集合里的旧关联再补新行；空集合 =
//     回收该权限的全部菜单授权（钉住它防止将来被当成 bug"修"反）；
//   - 身份免疫：菜单行删除后同路径重建（新 id），绑定快照原样存活且反解回
//     新 id——这正是旧 menu_id 绑定做不到的事（REPLACE 重建曾把授权整批漂没）。
package data

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	entCrud "github.com/tx7do/go-crud/entgo"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/menu"
	"go-wind-admin/app/admin/service/internal/data/ent/permissionmenu"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newPermissionMenuRepoForCase 建独立 SQLite 库并构造 repo。
func newPermissionMenuRepoForCase(t *testing.T) (*PermissionMenuRepo, *entCrud.EntClient[*ent.Client]) {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	return NewPermissionMenuRepoForTest(entClient), entClient
}

// createFullPathMenu 直建一条全路径身份完整的菜单行（根节点，path 即全路径），
// 返回其实体。
func createFullPathMenu(t *testing.T, entClient *entCrud.EntClient[*ent.Client], ctx context.Context, name, fullPath string) *ent.Menu {
	t.Helper()
	row, err := entClient.Client().Menu.Create().
		SetName(name).
		SetPath(fullPath).
		SetType(menu.TypeMenu).
		Save(ctx)
	require.NoError(t, err, "直建菜单应成功")
	return row
}

// countPermissionMenus 统计某权限的关联行数。
func countPermissionMenus(t *testing.T, client *ent.Client, ctx context.Context, permissionID uint32) int {
	t.Helper()
	cnt, err := client.PermissionMenu.Query().
		Where(permissionmenu.PermissionIDEQ(permissionID)).
		Count(ctx)
	require.NoError(t, err)
	return cnt
}

// menuPathSet 取权限关联的全路径快照集合（无序比较用）。
func menuPathSet(t *testing.T, client *ent.Client, ctx context.Context, permissionID uint32) map[string]struct{} {
	t.Helper()
	rows, err := client.PermissionMenu.Query().
		Where(permissionmenu.PermissionIDEQ(permissionID)).
		All(ctx)
	require.NoError(t, err)
	set := make(map[string]struct{}, len(rows))
	for _, r := range rows {
		require.NotNil(t, r.FullPath)
		set[*r.FullPath] = struct{}{}
	}
	return set
}

// TestPermissionMenuRepoSqlite_AssignResolvesToFullPathSnapshots 验证分配主路径：
// id 引用解析成全路径快照落库、重复分配 upsert 幂等、集合替换语义、
// 空集合回收全部、失效引用整体 400 且零落库、ListMenuIDs 反解回当前 id。
func TestPermissionMenuRepoSqlite_AssignResolvesToFullPathSnapshots(t *testing.T) {
	repo, entClient := newPermissionMenuRepoForCase(t)
	client := entClient.Client()
	ctx := enttest.NewSystemContext(context.Background())

	menuA := createFullPathMenu(t, entClient, ctx, "MenuA", "/m/a")
	menuB := createFullPathMenu(t, entClient, ctx, "MenuB", "/m/b")
	menuC := createFullPathMenu(t, entClient, ctx, "MenuC", "/m/c")

	// 首次分配：权限 100 → [A, B] —— 落库的是全路径快照
	require.NoError(t, repo.AssignMenus(ctx, 100, []uint32{menuA.ID, menuB.ID}), "首次分配应成功")
	got := menuPathSet(t, client, ctx, 100)
	require.Len(t, got, 2, "两个菜单应各落一行")
	require.Contains(t, got, "/m/a", "绑定应落全路径快照而非 id")
	require.Contains(t, got, "/m/b")

	// 行级断言：created_at 显式写入（NULL 的 created_at 列表页会出一排空行）
	rows, err := client.PermissionMenu.Query().
		Where(permissionmenu.PermissionIDEQ(100)).
		All(ctx)
	require.NoError(t, err)
	for _, r := range rows {
		require.NotNil(t, r.CreatedAt, "created_at 必须显式写入")
		require.Equal(t, uint32(100), *r.PermissionID, "permission_id 应按入参落库")
	}

	// ListMenuIDs：快照反解回当前 id
	ids, err := repo.ListMenuIDs(ctx, []uint32{100})
	require.NoError(t, err)
	require.ElementsMatch(t, []uint32{menuA.ID, menuB.ID}, ids, "ListMenuIDs 应反解回当前菜单 id")

	// 重新分配 [B, C]：A 被清掉，B 复用原行
	require.NoError(t, repo.AssignMenus(ctx, 100, []uint32{menuB.ID, menuC.ID}), "重新分配应成功")
	require.Len(t, menuPathSet(t, client, ctx, 100), 2, "重新分配后仍是 2 行：被移除的清掉、保留的复用原行")
	got = menuPathSet(t, client, ctx, 100)
	require.Contains(t, got, "/m/b")
	require.Contains(t, got, "/m/c")
	require.NotContains(t, got, "/m/a", "被移出新集合的菜单关联必须清掉，残留即越权")

	// 同集合重复分配：upsert 幂等，行数不涨
	require.NoError(t, repo.AssignMenus(ctx, 100, []uint32{menuB.ID, menuC.ID}), "重复分配同集合应幂等成功")
	require.Equal(t, 2, countPermissionMenus(t, client, ctx, 100), "重复分配不得新增行（唯一索引冲突应走更新分支）")

	// 回收全部：空列表 = CleanNotExist 的 NOT IN 空集合恒真 → 关联全部清掉
	require.NoError(t, repo.AssignMenus(ctx, 100, nil), "传空集合的分配应成功")
	require.Empty(t, menuPathSet(t, client, ctx, 100), "空列表的净效果应是回收该权限的全部菜单授权")

	// 失效引用：提交了不存在的菜单 id → 整体 400 带缺失明细，库内零落库（不吞错铁律）
	require.NoError(t, repo.AssignMenus(ctx, 101, []uint32{menuA.ID}), "先落一条作对照")
	err = repo.AssignMenus(ctx, 101, []uint32{menuA.ID, 987654321})
	require.Error(t, err, "含失效引用的分配应整体失败")
	require.ErrorContains(t, err, "987654321", "错误应带缺失 id 明细")
	require.Equal(t, 1, countPermissionMenus(t, client, ctx, 101), "失败分支不得多落任何绑定行")

	// 多权限查询并集 + 未授权/空入参返回空集
	require.NoError(t, repo.AssignMenus(ctx, 102, []uint32{menuC.ID}))
	ids, err = repo.ListMenuIDs(ctx, []uint32{101, 102})
	require.NoError(t, err)
	require.ElementsMatch(t, []uint32{menuA.ID, menuC.ID}, ids, "多权限查询应返回并集")
	ids, err = repo.ListMenuIDs(ctx, []uint32{999})
	require.NoError(t, err)
	require.Empty(t, ids, "未授权权限应返回空集而不是错误")
	ids, err = repo.ListMenuIDs(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, ids, "空权限列表应返回空集而不是错误")
}

// TestPermissionMenuRepoSqlite_BindingSurvivesMenuRecreate 零漂移核心用例：
// 菜单行删除后同路径重建（新 id），绑定快照原样存活且反解回新 id——
// 旧 menu_id 绑定在这里会整批漂没（菜单同步 REPLACE 曾如此）。
func TestPermissionMenuRepoSqlite_BindingSurvivesMenuRecreate(t *testing.T) {
	repo, entClient := newPermissionMenuRepoForCase(t)
	client := entClient.Client()
	ctx := enttest.NewSystemContext(context.Background())

	menuRow := createFullPathMenu(t, entClient, ctx, "DriftMenu", "/m/drift")
	bindingPerm := uint32(200)
	require.NoError(t, repo.AssignMenus(ctx, bindingPerm, []uint32{menuRow.ID}))
	bindingID, err := client.PermissionMenu.Query().
		Where(permissionmenu.PermissionIDEQ(bindingPerm)).
		FirstID(ctx)
	require.NoError(t, err)

	// 模拟菜单同步 REPLACE 的删了再加：同路径、id 重新生成
	require.NoError(t, client.Menu.DeleteOneID(menuRow.ID).Exec(ctx))
	reborn := createFullPathMenu(t, entClient, ctx, "DriftMenu", "/m/drift")
	require.NotEqual(t, menuRow.ID, reborn.ID, "重建行的 id 应不同于原行")

	// 绑定快照原样存活：行未动、full_path 不变
	binding, err := client.PermissionMenu.Query().
		Where(permissionmenu.PermissionIDEQ(bindingPerm)).
		Only(ctx)
	require.NoError(t, err)
	require.Equal(t, bindingID, binding.ID, "绑定行应原样存活")
	require.Equal(t, "/m/drift", *binding.FullPath)

	// 反解回的是重建后的新 id：导航链路照常命中
	ids, err := repo.ListMenuIDs(ctx, []uint32{bindingPerm})
	require.NoError(t, err)
	require.Equal(t, []uint32{reborn.ID}, ids, "ListMenuIDs 应反解到重建后的新 id")
}

// TestPermissionMenuRepoSqlite_DeleteByFullPaths 验证全路径级联清理：
// 菜单行删除流程按 full_path 清绑定；未删菜单的绑定不受牵连。
func TestPermissionMenuRepoSqlite_DeleteByFullPaths(t *testing.T) {
	repo, entClient := newPermissionMenuRepoForCase(t)
	client := entClient.Client()
	ctx := enttest.NewSystemContext(context.Background())

	doomed := createFullPathMenu(t, entClient, ctx, "Doomed", "/m/doomed")
	kept := createFullPathMenu(t, entClient, ctx, "Kept", "/m/kept")

	require.NoError(t, repo.AssignMenus(ctx, 300, []uint32{doomed.ID, kept.ID}))
	require.Equal(t, 2, countPermissionMenus(t, client, ctx, 300))

	require.NoError(t, repo.DeleteByFullPaths(ctx, client, []string{"/m/doomed"}), "DeleteByFullPaths 应成功")

	got := menuPathSet(t, client, ctx, 300)
	require.Len(t, got, 1, "被删菜单的绑定应被级联清理")
	require.Contains(t, got, "/m/kept", "未删菜单的绑定不得被牵连")
}

// TestPermissionMenuRepoSqlite_AssignMenusWithTx 验证事务内分配：
// 空集合早退不触碰行；提交后可见；回滚后不残留。
// WithTx 直接收全路径快照（不做解析），故无需预置菜单行。
func TestPermissionMenuRepoSqlite_AssignMenusWithTx(t *testing.T) {
	repo, entClient := newPermissionMenuRepoForCase(t)
	client := entClient.Client()
	ctx := enttest.NewSystemContext(context.Background())

	// 空集合：直接成功返回，不开任何写
	tx, err := client.Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.AssignMenusWithTx(ctx, tx, 200, nil), "空 menuIDs 应早退成功")
	require.Equal(t, 0, countPermissionMenus(t, client, ctx, 200), "空 menuIDs 不得落行")

	// 事务内分配，提交后可见
	require.NoError(t, repo.AssignMenusWithTx(ctx, tx, 200, []string{"/tx/a", "/tx/b"}), "事务内分配应成功")
	require.NoError(t, tx.Commit())
	require.Len(t, menuPathSet(t, client, ctx, 200), 2, "提交后分配应可见")

	// 回滚：事务内的分配不残留（半次授权比没有授权更危险）
	tx2, err := client.Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback() }()
	require.NoError(t, repo.AssignMenusWithTx(ctx, tx2, 200, []string{"/tx/c"}), "回滚事务内分配应成功")
	require.NoError(t, tx2.Rollback())
	require.Len(t, menuPathSet(t, client, ctx, 200), 2, "回滚后不得残留新分配的行")
}

// TestPermissionMenuRepoSqlite_TruncateAndDelete 验证批量删除入口：
// TruncateExcept 保留指定权限集合的绑定（由调用方按 code 前缀解析，不再依赖
// "种子权限=1/2/3"）、清掉其余；Delete/DeleteByPermissionIDs 按权限精确删除，
// 目标不存在时静默成功。
func TestPermissionMenuRepoSqlite_TruncateAndDelete(t *testing.T) {
	repo, entClient := newPermissionMenuRepoForCase(t)
	client := entClient.Client()
	ctx := enttest.NewSystemContext(context.Background())

	m1 := createFullPathMenu(t, entClient, ctx, "T1", "/t/1")
	m2 := createFullPathMenu(t, entClient, ctx, "T2", "/t/2")
	m3 := createFullPathMenu(t, entClient, ctx, "T3", "/t/3")
	m4 := createFullPathMenu(t, entClient, ctx, "T4", "/t/4")

	// 播一批：受保护集合 700/701 各一行，普通权限 500/501 各两行
	require.NoError(t, repo.AssignMenus(ctx, 700, []uint32{m1.ID}))
	require.NoError(t, repo.AssignMenus(ctx, 701, []uint32{m2.ID}))
	require.NoError(t, repo.AssignMenus(ctx, 500, []uint32{m3.ID, m4.ID}))

	require.NoError(t, repo.TruncateExcept(ctx, []uint32{700, 701}), "TruncateExcept 应成功")

	require.Equal(t, 1, countPermissionMenus(t, client, ctx, 700), "保留集内权限的关联应保留")
	require.Equal(t, 1, countPermissionMenus(t, client, ctx, 701), "保留集内权限的关联应保留")
	require.Equal(t, 0, countPermissionMenus(t, client, ctx, 500), "保留集外权限的关联应被清空")

	// TruncateExcept(nil) = 全清
	require.NoError(t, repo.AssignMenus(ctx, 700, []uint32{m1.ID}))
	require.NoError(t, repo.TruncateExcept(ctx, nil), "全清应成功")
	require.Equal(t, 0, countPermissionMenus(t, client, ctx, 700), "全清后不得残留")

	// 重新播数据测 Delete / DeleteByPermissionIDs
	m5 := createFullPathMenu(t, entClient, ctx, "T5", "/t/5")
	m6 := createFullPathMenu(t, entClient, ctx, "T6", "/t/6")
	require.NoError(t, repo.AssignMenus(ctx, 600, []uint32{m5.ID}))
	require.NoError(t, repo.AssignMenus(ctx, 601, []uint32{m6.ID}))

	require.NoError(t, repo.Delete(ctx, 600), "按权限删除应成功")
	require.Equal(t, 0, countPermissionMenus(t, client, ctx, 600), "删除后该权限应无关联")
	require.Equal(t, 1, countPermissionMenus(t, client, ctx, 601), "其他权限不得被牵连")

	require.NoError(t, repo.DeleteByPermissionIDs(ctx, []uint32{601}), "按权限集合批量删除应成功")
	require.Equal(t, 0, countPermissionMenus(t, client, ctx, 601), "批量删除应覆盖权限 601")

	// 目标不存在：静默成功（幂等删除语义，重试与并发清理依赖它）
	require.NoError(t, repo.Delete(ctx, 999999), "删除无关联的权限应静默成功")
	require.NoError(t, repo.DeleteByPermissionIDs(ctx, nil), "空集合批量删除应静默成功")
}
