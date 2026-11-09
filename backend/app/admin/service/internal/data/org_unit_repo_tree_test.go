// OrgUnitRepo 树形行为的 SQLite 内存库白盒测试：
//   - parent_id 显式 0 与未设置同义（挂根，不再触发自引用 FK 违约 500）；
//   - 显式 0 更新 = 移到根级，物化路径重算为 "/<自身ID>/"；
//   - 挂到自己后代下拒绝；
//   - 存量环脏数据下 relocateSubtree 以 visited 收敛不死循环；
//   - 删除子树级联清角色↔单元授权绑定（role_org_units），方言无关（不再依赖
//     递归 CTE 的子树收集）。
package data

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	entCrud "github.com/tx7do/go-crud/entgo"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/orgunit"
	"go-wind-admin/app/admin/service/internal/data/ent/roleorgunit"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

func newOrgUnitRepoTreeSqlite(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) (*OrgUnitRepo, *ent.Client) {
	t.Helper()
	return NewOrgUnitRepoForTest(entClient), entClient.Client()
}

// createOrgUnitRow 经 repo.Create 直建组织单元（parentID nil = 根；显式 0 同根）。
// 走 repo 路径让 setTreePath 物化路径照常落库；id 经唯一 name 反查（repo.Create
// 不回传产物）。
func createOrgUnitRow(t *testing.T, ctx context.Context, repo *OrgUnitRepo, name string, parentID *uint64) uint64 {
	t.Helper()
	data := &identityV1.OrgUnit{
		Name:     trans.Ptr(name),
		TenantId: trans.Ptr(uint64(42)),
	}
	if parentID != nil {
		data.ParentId = parentID
	}
	require.NoError(t, repo.Create(ctx, &identityV1.CreateOrgUnitRequest{Data: data}),
		"直建组织单元 [%s] 应成功", name)
	row, err := repo.entClient.Client().OrgUnit.Query().Where(orgunit.NameEQ(name)).Only(ctx)
	require.NoError(t, err, "反查直建单元 [%s] 应命中", name)
	return row.ID
}

// TestOrgUnitRepoSqlite_CreateExplicitZeroParentAsRoot parent_id 显式 0 创建 =
// 挂根（此前 0 直接写库触发自引用 FK 违约 500）。
func TestOrgUnitRepoSqlite_CreateExplicitZeroParentAsRoot(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo, client := newOrgUnitRepoTreeSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	require.NoError(t, repo.Create(ctx, &identityV1.CreateOrgUnitRequest{
		Data: &identityV1.OrgUnit{
			Name:     trans.Ptr("零父根单元"),
			TenantId: trans.Ptr(uint64(42)),
			ParentId: trans.Ptr(uint64(0)),
		},
	}), "显式 parent_id=0 的创建应按挂根处理而非 FK 违约")

	row, err := client.OrgUnit.Query().Only(ctx)
	require.NoError(t, err)
	require.Nil(t, row.ParentID, "parent_id=0 应落 NULL（根）")
	require.NotNil(t, row.Path)
	require.Equal(t, "/"+strconv.FormatUint(uint64(row.ID), 10)+"/", *row.Path, "根节点物化路径应为 /<自身ID>/")
}

// TestOrgUnitRepoSqlite_UpdateExplicitZeroParentMovesToRoot 显式 0 更新 = 移到
// 根级，物化路径从父链前缀重算为根路径。
func TestOrgUnitRepoSqlite_UpdateExplicitZeroParentMovesToRoot(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo, client := newOrgUnitRepoTreeSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	root := createOrgUnitRow(t, ctx, repo, "移根父单元", nil)
	child := createOrgUnitRow(t, ctx, repo, "移根子单元", &root)

	// 前置确认子节点路径带父前缀
	rowChild, err := client.OrgUnit.Get(ctx, child)
	require.NoError(t, err)
	require.NotNil(t, rowChild.Path)
	require.Contains(t, *rowChild.Path, "/"+strconv.FormatUint(uint64(root), 10)+"/",
		"子节点初始路径应带父前缀")

	zero := uint64(0)
	require.NoError(t, repo.Update(ctx, &identityV1.UpdateOrgUnitRequest{
		Id:   child,
		Data: &identityV1.OrgUnit{ParentId: &zero},
	}), "显式 parent_id=0 的更新应把单元移到根级")

	rowChild, err = client.OrgUnit.Get(ctx, child)
	require.NoError(t, err)
	require.Nil(t, rowChild.ParentID, "移根后 parent_id 应落 NULL")
	require.NotNil(t, rowChild.Path)
	require.Equal(t, "/"+strconv.FormatUint(uint64(child), 10)+"/", *rowChild.Path,
		"移根后物化路径应重算为根路径 /<自身ID>/")
}

// TestOrgUnitRepoSqlite_MoveUnderDescendantRejected 挂到自己后代下拒绝
//（relocate 前置的路径前缀环检测）。
func TestOrgUnitRepoSqlite_MoveUnderDescendantRejected(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo, client := newOrgUnitRepoTreeSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	a := createOrgUnitRow(t, ctx, repo, "拒环甲", nil)
	bID := createOrgUnitRow(t, ctx, repo, "拒环乙", &a)
	c := createOrgUnitRow(t, ctx, repo, "拒环丙", &bID)

	err := repo.Update(ctx, &identityV1.UpdateOrgUnitRequest{
		Id:   a,
		Data: &identityV1.OrgUnit{ParentId: &c},
	})
	require.Error(t, err, "挂到自己后代下必须拒绝")
	require.True(t, identityV1.IsBadRequest(err), "应报 400 而非 500")

	rowA, err := client.OrgUnit.Get(ctx, a)
	require.NoError(t, err)
	require.Nil(t, rowA.ParentID, "被拒更新不得改动挂载关系")
}

// TestOrgUnitRepoSqlite_RelocateToleratesDirtyCycle 存量环脏数据下
// relocateSubtree 以 visited 收敛返回（无防护版本会 BFS 死循环挂死 goroutine）。
func TestOrgUnitRepoSqlite_RelocateToleratesDirtyCycle(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo, client := newOrgUnitRepoTreeSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	a := createOrgUnitRow(t, ctx, repo, "环甲", nil)
	b := createOrgUnitRow(t, ctx, repo, "环乙", &a)

	// 直改库构造 A↔B 互为父子的环（绕过一切校验）
	require.NoError(t, client.OrgUnit.UpdateOneID(a).SetParentID(b).Exec(ctx))
	require.NoError(t, client.OrgUnit.UpdateOneID(b).SetParentID(a).Exec(ctx))

	// 从乙起 relocate：父链前缀环检测不命中（甲路径不含 /乙ID/），BFS 沿
	// 甲↔乙环展开，visited 断路后必须收敛返回
	require.NoError(t, repo.relocateSubtree(ctx, b),
		"环脏数据下 relocate 必须收敛返回而非挂死")

	rowA, err := client.OrgUnit.Get(ctx, a)
	require.NoError(t, err)
	require.NotNil(t, rowA.Path, "环上游走应落定确定性的路径而非死循环")
}

// TestOrgUnitRepoSqlite_DeleteCascadesRoleOrgUnits 删除子树时角色↔单元授权
// 绑定随子树级联清理，子树外的绑定不动；子树收集内存化，方言无关。
func TestOrgUnitRepoSqlite_DeleteCascadesRoleOrgUnits(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo, client := newOrgUnitRepoTreeSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	root := createOrgUnitRow(t, ctx, repo, "级联根单元", nil)
	child := createOrgUnitRow(t, ctx, repo, "级联子单元", &root)
	side := createOrgUnitRow(t, ctx, repo, "旁路单元", nil)

	roleHit, err := client.Role.Create().SetName("级联命中角色").SetCode("CASCADE_HIT_ROLE").Save(ctx)
	require.NoError(t, err)
	roleSide, err := client.Role.Create().SetName("旁路角色").SetCode("CASCADE_SIDE_ROLE").Save(ctx)
	require.NoError(t, err)

	now := time.Now()
	// 命中子树：根与子单元各一条
	require.NoError(t, client.RoleOrgUnit.Create().
		SetTenantID(42).SetRoleID(roleHit.ID).SetOrgUnitID(root).SetCreatedAt(now).
		Exec(ctx), "直建角色-根单元绑定应成功")
	require.NoError(t, client.RoleOrgUnit.Create().
		SetTenantID(42).SetRoleID(roleHit.ID).SetOrgUnitID(child).SetCreatedAt(now).
		Exec(ctx), "直建角色-子单元绑定应成功")
	// 子树外：旁路单元
	require.NoError(t, client.RoleOrgUnit.Create().
		SetTenantID(42).SetRoleID(roleSide.ID).SetOrgUnitID(side).SetCreatedAt(now).
		Exec(ctx), "直建旁路绑定应成功")

	require.NoError(t, repo.Delete(ctx, &identityV1.DeleteOrgUnitRequest{
		QueryBy: &identityV1.DeleteOrgUnitRequest_Id{Id: root},
	}), "删除子树应成功")

	unitCnt, err := client.OrgUnit.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, unitCnt, "子树两行应删净，只剩旁路单元")

	hitCnt, err := client.RoleOrgUnit.Query().
		Where(roleorgunit.RoleIDEQ(roleHit.ID)).
		Count(ctx)
	require.NoError(t, err)
	require.Zero(t, hitCnt, "子树内单元的角色授权绑定必须级联清掉")
	sideCnt, err := client.RoleOrgUnit.Query().
		Where(roleorgunit.OrgUnitIDEQ(side)).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, sideCnt, "子树外绑定必须存活")
}

// TestOrgUnitRepoSqlite_CreateChildPath 常规父子创建的物化路径回归：
// 显式 >0 父级照旧生效（守卫只拦 0，不误伤正常挂载）。
func TestOrgUnitRepoSqlite_CreateChildPath(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo, client := newOrgUnitRepoTreeSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	root := createOrgUnitRow(t, ctx, repo, "常规根", nil)

	require.NoError(t, repo.Create(ctx, &identityV1.CreateOrgUnitRequest{
		Data: &identityV1.OrgUnit{
			Name:     trans.Ptr("常规子单元"),
			TenantId: trans.Ptr(uint64(42)),
			ParentId: trans.Ptr(root),
		},
	}), "显式 >0 父级的创建应照常挂载")

	row, err := client.OrgUnit.Query().Where(orgunit.NameEQ("常规子单元")).Only(ctx)
	require.NoError(t, err)
	require.NotNil(t, row.ParentID)
	require.Equal(t, root, *row.ParentID, "显式父级应照常落库")
	require.NotNil(t, row.Path)
	require.Contains(t, *row.Path, "/"+strconv.FormatUint(uint64(root), 10)+"/",
		"子节点物化路径应带父前缀")
}
