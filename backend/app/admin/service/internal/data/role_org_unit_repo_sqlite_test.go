package data

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newRoleOrgUnitRepoSqlite 用 enttest helper 构造一个可直接做关联 CRUD 的 RoleOrgUnitRepo。
// 白盒构造逐字段复刻 NewRoleOrgUnitRepo 的初始化（该仓库只有 log 与 entClient 两个字段）。
func newRoleOrgUnitRepoSqlite(t *testing.T) *RoleOrgUnitRepo {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	return &RoleOrgUnitRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
	}
}

// TestRoleOrgUnitRepoSqlite_AssignListReplaceClean 覆盖 RoleOrgUnitRepo 的关联生命周期：
// AssignOrgUnits（写入，含租户一致性校验）→ ent client 直查确认落库 →
// ListOrgUnitIDs/ListOrgUnitIDsByRoleIDs（查询）→ ReplaceOrgUnits（整体替换）→
// CleanOrgUnits（清理）→ 计数归零。
func TestRoleOrgUnitRepoSqlite_AssignListReplaceClean(t *testing.T) {
	repo := newRoleOrgUnitRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	// 局部闭包直插 org_unit 父行：AssignOrgUnits 的租户一致性校验要求
	// 单元真实存在且归属目标租户。包内不引入通用 helper，避免与并行测试文件的包级符号撞名。
	createOrgUnitRowFor := func(tenantID uint32, name string) uint32 {
		row, err := repo.entClient.Client().OrgUnit.Create().
			SetName(name).
			SetTenantID(tenantID).
			Save(ctx)
		require.NoError(t, err, "直插 org_unit 父行应成功")
		return row.ID
	}

	const (
		tenantID = uint32(7001)
		testRole = uint32(7101)
	)
	unitA := createOrgUnitRowFor(tenantID, fmt.Sprintf("sqlite单元A-%d", 7001))
	unitB := createOrgUnitRowFor(tenantID, fmt.Sprintf("sqlite单元B-%d", 7002))

	tx, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	// 无论断言成败都释放事务连接：Commit 成功后 Rollback 返回 ErrTxDone 被忽略；
	// 断言失败（Goexit）时回滚，避免残留写锁阻塞后续测试。
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.AssignOrgUnits(ctx, tx, tenantID, 0, testRole, []uint32{unitA, unitB}),
		"给角色分配两个同租户单元应通过租户一致性校验")
	require.NoError(t, tx.Commit())

	// ent client 直查：两条关联确实落库
	rowCount, err := repo.entClient.Client().RoleOrgUnit.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, rowCount, "sys_role_org_units 应有 2 条关联记录")

	// 按角色列出单元 ID
	unitIDs, err := repo.ListOrgUnitIDs(ctx, testRole)
	require.NoError(t, err)
	require.ElementsMatch(t, []uint32{unitA, unitB}, unitIDs, "按角色查询应返回两条关联的单元 ID")

	// 按角色列表分组查询
	grouped, err := repo.ListOrgUnitIDsByRoleIDs(ctx, []uint32{testRole})
	require.NoError(t, err)
	require.ElementsMatch(t, []uint32{unitA, unitB}, grouped[testRole], "分组查询应返回该角色关联的单元 ID")

	// 整体替换：仅保留 unitA
	tx2, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback() }()
	require.NoError(t, repo.ReplaceOrgUnits(ctx, tx2, tenantID, 0, testRole, []uint32{unitA}))
	require.NoError(t, tx2.Commit())

	afterReplace, err := repo.ListOrgUnitIDs(ctx, testRole)
	require.NoError(t, err)
	require.Equal(t, []uint32{unitA}, afterReplace, "整体替换后应仅保留新集合中的单元 ID")

	rowCount, err = repo.entClient.Client().RoleOrgUnit.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, rowCount, "替换后表内应只有 1 条关联")

	// 清理：按角色删除全部关联
	tx3, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx3.Rollback() }()
	require.NoError(t, repo.CleanOrgUnits(ctx, tx3, testRole))
	require.NoError(t, tx3.Commit())

	rowCount, err = repo.entClient.Client().RoleOrgUnit.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, rowCount, "清理后 sys_role_org_units 计数应归零")

	empty, err := repo.ListOrgUnitIDs(ctx, testRole)
	require.NoError(t, err)
	require.Empty(t, empty, "清理后按角色查询应无单元 ID")
}

// TestRoleOrgUnitRepoSqlite_CrossTenantRejected 验证 AssignOrgUnits 的
// 租户一致性校验：跨租户（或不属于目标租户）的单元集应被整体拒绝，不落任何行。
func TestRoleOrgUnitRepoSqlite_CrossTenantRejected(t *testing.T) {
	repo := newRoleOrgUnitRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const (
		tenantA  = uint32(7003)
		tenantB  = uint32(7004)
		testRole = uint32(7102)
	)
	// 局部闭包直插指定租户的 org_unit 父行（理由同上一测试）
	createOrgUnitRowFor := func(tenantID uint32, name string) uint32 {
		row, err := repo.entClient.Client().OrgUnit.Create().
			SetName(name).
			SetTenantID(tenantID).
			Save(ctx)
		require.NoError(t, err, "直插 org_unit 父行应成功")
		return row.ID
	}
	unitInA := createOrgUnitRowFor(tenantA, fmt.Sprintf("sqlite单元A-%d", 7003))
	unitInB := createOrgUnitRowFor(tenantB, fmt.Sprintf("sqlite单元B-%d", 7004))

	// 单个跨租户单元 → 拒绝
	tx, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	require.Error(t, repo.AssignOrgUnits(ctx, tx, tenantA, 0, testRole, []uint32{unitInB}),
		"分配不属于目标租户的单元应被拒绝")
	_ = tx.Rollback()

	// 混合集（含一个跨租户单元）→ 整体拒绝
	tx2, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback() }()
	require.Error(t, repo.AssignOrgUnits(ctx, tx2, tenantA, 0, testRole, []uint32{unitInA, unitInB}),
		"混合跨租户的单元集应被整体拒绝")
	_ = tx2.Rollback()

	rowCount, err := repo.entClient.Client().RoleOrgUnit.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, rowCount, "被拒绝的分配不应落任何关联行")

	empty, err := repo.ListOrgUnitIDs(ctx, testRole)
	require.NoError(t, err)
	require.Empty(t, empty, "被拒绝的角色应无关联单元 ID")
}
