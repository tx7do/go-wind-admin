// PermissionMenuRepo 的 SQLite 内存库测试（白盒，包内测试）。
//
// 关联表仓库没有复杂映射，测试盯的是"授权错一行就是越权/漏权"的两件事：
//   - upsert 幂等：同一 (permission, menu) 重复分配必须落在同一行上，唯一索引
//     (permission_id, menu_id) 命中 ON CONFLICT 走更新分支，绝不产生重复行；
//   - 集合语义：重新分配 = 先清掉不在新集合里的旧关联再补新行（CleanNotExistMenus
//     用 NOT IN）；ent 对空集合的 In/NotIn 有明确约定（In→恒假、NotIn→恒真），
//     因此 AssignMenus 传空 menuID 列表的净效果是"回收该权限的全部菜单"——
//     这正是回收授权该走的路，钉住它防止将来被当成 bug"修"反。
package data

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	entCrud "github.com/tx7do/go-crud/entgo"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/permissionmenu"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newPermissionMenuRepoForCase 建独立 SQLite 库并构造 repo。
func newPermissionMenuRepoForCase(t *testing.T) (*PermissionMenuRepo, *entCrud.EntClient[*ent.Client]) {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	return NewPermissionMenuRepoForTest(entClient), entClient
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

// menuIDSet 取权限关联的菜单 ID 集合（无序比较用）。
func menuIDSet(t *testing.T, client *ent.Client, ctx context.Context, permissionID uint32) map[uint32]struct{} {
	t.Helper()
	rows, err := client.PermissionMenu.Query().
		Where(permissionmenu.PermissionIDEQ(permissionID)).
		All(ctx)
	require.NoError(t, err)
	set := make(map[uint32]struct{}, len(rows))
	for _, r := range rows {
		require.NotNil(t, r.MenuID)
		set[*r.MenuID] = struct{}{}
	}
	return set
}

// TestPermissionMenuRepoSqlite_AssignAndList 验证分配与查询主路径：
// 分配按行落库（created_at 显式写入）、重复分配走 upsert 不产生重复行、
// 重新分配 = 集合替换（旧集合里被移除的菜单关联被清掉）、
// ListMenuIDs 按权限并集返回、未授权的权限返回空集。
func TestPermissionMenuRepoSqlite_AssignAndList(t *testing.T) {
	repo, entClient := newPermissionMenuRepoForCase(t)
	client := entClient.Client()
	ctx := enttest.NewSystemContext(context.Background())

	// 首次分配：权限 100 → 菜单 [11,12,13]
	require.NoError(t, repo.AssignMenus(ctx, 100, []uint32{11, 12, 13}), "首次分配应成功")
	require.Len(t, menuIDSet(t, client, ctx, 100), 3, "三个菜单应各落一行")

	// 行级断言：外键两列都落对，created_at 显式写入（NULL 的 created_at 列表页会出一排空行）
	rows, err := client.PermissionMenu.Query().
		Where(permissionmenu.PermissionIDEQ(100)).
		All(ctx)
	require.NoError(t, err)
	for _, r := range rows {
		require.NotNil(t, r.CreatedAt, "created_at 必须显式写入")
	}
	single := rows[0]
	require.Equal(t, uint32(100), *single.PermissionID, "permission_id 应按入参落库")

	// 重新分配 [12,13,14]：旧集合里的 11 被清掉，12/13 复用原行（不重复）
	require.NoError(t, repo.AssignMenus(ctx, 100, []uint32{12, 13, 14}), "重新分配应成功")
	require.Len(t, menuIDSet(t, client, ctx, 100), 3, "重新分配后仍是 3 行：被移除的清掉、保留的复用原行")
	got := menuIDSet(t, client, ctx, 100)
	for _, want := range []uint32{12, 13, 14} {
		require.Contains(t, got, want, "新集合内的菜单关联应在场")
	}
	require.NotContains(t, got, uint32(11), "被移出新集合的菜单关联必须清掉，残留即越权")

	// 同集合重复分配：upsert 幂等，行数不涨
	require.NoError(t, repo.AssignMenus(ctx, 100, []uint32{12, 13, 14}), "重复分配同集合应幂等成功")
	require.Len(t, menuIDSet(t, client, ctx, 100), 3, "重复分配不得新增行（唯一索引冲突应走更新分支）")

	// 回收全部：空 menuID 列表 → CleanNotExistMenus 的 NOT IN 空集合恒真 → 该权限关联全部清掉
	require.NoError(t, repo.AssignMenus(ctx, 100, nil), "传空集合的分配应成功")
	require.Empty(t, menuIDSet(t, client, ctx, 100), "空 menuID 列表的净效果应是回收该权限的全部菜单授权")

	// 单条分配 + 幂等：AssignMenu 对同一 (permission, menu) 重复调用不产生第二行
	require.NoError(t, repo.AssignMenu(ctx, 101, 21), "单条分配应成功")
	require.NoError(t, repo.AssignMenu(ctx, 101, 21), "重复单条分配应幂等成功")
	require.Equal(t, 1, countPermissionMenus(t, client, ctx, 101), "重复单条分配必须仍是 1 行")

	// 多权限的查询：并集语义
	require.NoError(t, repo.AssignMenus(ctx, 102, []uint32{31, 32}))
	ids, err := repo.ListMenuIDs(ctx, []uint32{101, 102})
	require.NoError(t, err)
	require.ElementsMatch(t, []uint32{21, 31, 32}, ids, "多权限查询应返回并集")

	// 未授权权限与空入参：空集而非错误
	ids, err = repo.ListMenuIDs(ctx, []uint32{999})
	require.NoError(t, err)
	require.Empty(t, ids, "未授权权限应返回空集而不是错误")
	ids, err = repo.ListMenuIDs(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, ids, "空权限列表应返回空集而不是错误（ent 空 IN 恒假）")
}

// TestPermissionMenuRepoSqlite_AssignMenusWithTx 验证事务内分配：
// 空集合早退不触碰行；提交后可见；回滚后不残留。
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
	require.NoError(t, repo.AssignMenusWithTx(ctx, tx, 200, []uint32{41, 42}), "事务内分配应成功")
	require.NoError(t, tx.Commit())
	require.Len(t, menuIDSet(t, client, ctx, 200), 2, "提交后分配应可见")

	// 回滚：事务内的分配不残留（半次授权比没有授权更危险）
	tx2, err := client.Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback() }()
	require.NoError(t, repo.AssignMenusWithTx(ctx, tx2, 200, []uint32{43}), "回滚事务内分配应成功")
	require.NoError(t, tx2.Rollback())
	require.Len(t, menuIDSet(t, client, ctx, 200), 2, "回滚后不得残留新分配的行")
}

// TestPermissionMenuRepoSqlite_Clean 验证两个清理入口：
// CleanMenus 批量清掉权限集合的全部关联；CleanNotExistMenus 只保留集合内的菜单关联。
// 两者都是事务内接口（角色保存流程在同一个事务里先清后写）。
func TestPermissionMenuRepoSqlite_Clean(t *testing.T) {
	repo, entClient := newPermissionMenuRepoForCase(t)
	client := entClient.Client()
	ctx := enttest.NewSystemContext(context.Background())

	require.NoError(t, repo.AssignMenus(ctx, 300, []uint32{51, 52}))
	require.NoError(t, repo.AssignMenus(ctx, 301, []uint32{53, 54}))

	// CleanMenus：权限 300 的关联全清，权限 301 不受牵连
	tx, err := client.Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.CleanMenus(ctx, tx, []uint32{300}), "批量清理应成功")
	require.NoError(t, tx.Commit())
	require.Equal(t, 0, countPermissionMenus(t, client, ctx, 300), "CleanMenus 应清掉权限 300 的全部关联")
	require.Equal(t, 2, countPermissionMenus(t, client, ctx, 301), "权限 301 的关联不得被牵连")

	// CleanNotExistMenus：权限 301 只保留集合内的 53，集合外的 54 被清
	tx2, err := client.Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback() }()
	require.NoError(t, repo.CleanNotExistMenus(ctx, tx2, 301, []uint32{53}), "清理集合外关联应成功")
	require.NoError(t, tx2.Commit())
	remain := menuIDSet(t, client, ctx, 301)
	require.Contains(t, remain, uint32(53), "集合内的关联应保留")
	require.NotContains(t, remain, uint32(54), "集合外的关联应被清掉：这是重新分配的替换语义")

	// 事务回滚：清理不生效（调用方失败回滚时不能把授权清没了）
	tx3, err := client.Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx3.Rollback() }()
	require.NoError(t, repo.CleanMenus(ctx, tx3, []uint32{301}), "回滚事务内清理应成功")
	require.NoError(t, tx3.Rollback())
	require.Equal(t, 1, countPermissionMenus(t, client, ctx, 301), "回滚后清理不得生效")
}

// TestPermissionMenuRepoSqlite_TruncateAndDelete 验证批量删除入口：
// Truncate 保留受保护的权限 1/2/3（系统级权限的关联不可清）、清掉其余；
// Delete/DeleteByPermissionIDs 按权限精确删除，目标不存在时静默成功。
func TestPermissionMenuRepoSqlite_TruncateAndDelete(t *testing.T) {
	repo, entClient := newPermissionMenuRepoForCase(t)
	client := entClient.Client()
	ctx := enttest.NewSystemContext(context.Background())

	// 播一批：受保护的 1/2/3 各一行，普通权限 500/501 各两行
	require.NoError(t, repo.AssignMenus(ctx, 1, []uint32{61}))
	require.NoError(t, repo.AssignMenus(ctx, 2, []uint32{62}))
	require.NoError(t, repo.AssignMenus(ctx, 3, []uint32{63}))
	require.NoError(t, repo.AssignMenus(ctx, 500, []uint32{71, 72}))
	require.NoError(t, repo.AssignMenus(ctx, 501, []uint32{73, 74}))

	require.NoError(t, repo.Truncate(ctx), "Truncate 应成功")

	// 受保护的权限 1/2/3 关联保留
	require.Equal(t, 1, countPermissionMenus(t, client, ctx, 1), "权限 1 的关联应保留（Truncate 的白名单）")
	require.Equal(t, 1, countPermissionMenus(t, client, ctx, 2), "权限 2 的关联应保留")
	require.Equal(t, 1, countPermissionMenus(t, client, ctx, 3), "权限 3 的关联应保留")
	// 其余全清
	require.Equal(t, 0, countPermissionMenus(t, client, ctx, 500), "权限 500 的关联应被清空")
	require.Equal(t, 0, countPermissionMenus(t, client, ctx, 501), "权限 501 的关联应被清空")

	// 重新播数据测 Delete / DeleteByPermissionIDs
	require.NoError(t, repo.AssignMenus(ctx, 600, []uint32{81, 82}))
	require.NoError(t, repo.AssignMenus(ctx, 601, []uint32{83}))
	require.NoError(t, repo.AssignMenus(ctx, 602, []uint32{84}))

	require.NoError(t, repo.Delete(ctx, 600), "按权限删除应成功")
	require.Equal(t, 0, countPermissionMenus(t, client, ctx, 600), "删除后该权限应无关联")
	require.Equal(t, 1, countPermissionMenus(t, client, ctx, 601), "其他权限不得被牵连")

	require.NoError(t, repo.DeleteByPermissionIDs(ctx, []uint32{601, 602}), "按权限集合批量删除应成功")
	require.Equal(t, 0, countPermissionMenus(t, client, ctx, 601), "批量删除应覆盖权限 601")
	require.Equal(t, 0, countPermissionMenus(t, client, ctx, 602), "批量删除应覆盖权限 602")

	// 目标不存在：静默成功（幂等删除语义，重试与并发清理依赖它）
	require.NoError(t, repo.Delete(ctx, 999999), "删除无关联的权限应静默成功")
	require.NoError(t, repo.DeleteByPermissionIDs(ctx, nil), "空集合批量删除应静默成功")
}
