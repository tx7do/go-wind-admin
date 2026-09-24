package data

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/timestamppb"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
	entMembershipRole "go-wind-admin/app/admin/service/internal/data/ent/membershiprole"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newMembershipRoleRepoSqlite 用 enttest helper 构造一个可直接做关联 CRUD 的 MembershipRoleRepo。
// 白盒构造逐字段复刻 NewMembershipRoleRepo 的初始化（仅 log 换 NopLogger、entClient 换测试 client）。
func newMembershipRoleRepoSqlite(t *testing.T) *MembershipRoleRepo {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	return &MembershipRoleRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		statusConverter: mapper.NewEnumTypeConverter[permissionV1.MembershipRole_Status, entMembershipRole.Status](
			permissionV1.MembershipRole_Status_name,
			permissionV1.MembershipRole_Status_value,
		),
	}
}

// TestMembershipRoleRepoSqlite_AssignListAndClean 覆盖 MembershipRoleRepo 的关联生命周期：
// AssignMembershipRoles（写入）→ ent client 直查确认落库 →
// ListRoleIDs/ListMembershipIDs（正反向查询）→ CleanRelationsByMembershipID（清理）→ 计数归零。
func TestMembershipRoleRepoSqlite_AssignListAndClean(t *testing.T) {
	repo := newMembershipRoleRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const (
		testMembershipID = uint32(8101)
		testRoleA        = uint32(9101)
		testRoleB        = uint32(9102)
	)

	tx, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	// 无论断言成败都释放事务连接：Commit 成功后 Rollback 返回 ErrTxDone 被忽略；
	// 断言失败（Goexit）时回滚，避免残留写锁阻塞后续测试。
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.AssignMembershipRoles(ctx, tx, testMembershipID, []*permissionV1.MembershipRole{
		{
			MembershipId: trans.Ptr(testMembershipID),
			RoleId:       trans.Ptr(testRoleA),
			Status:       trans.Ptr(permissionV1.MembershipRole_ACTIVE),
		},
		{
			MembershipId: trans.Ptr(testMembershipID),
			RoleId:       trans.Ptr(testRoleB),
			Status:       trans.Ptr(permissionV1.MembershipRole_ACTIVE),
		},
	}), "批量分配两条成员-角色关联应成功")
	require.NoError(t, tx.Commit())

	// ent client 直查：两条关联确实落库
	rowCount, err := repo.entClient.Client().MembershipRole.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, rowCount, "sys_membership_roles 应有 2 条关联记录")

	// 正向：按成员列出角色 ID
	roleIDs, err := repo.ListRoleIDs(ctx, testMembershipID, false)
	require.NoError(t, err)
	require.ElementsMatch(t, []uint32{testRoleA, testRoleB}, roleIDs, "按成员查询应返回两条关联的角色 ID")

	// 反向：按角色列出成员 ID（含多角色批量变体，按角色逐个收集不去重）
	membershipIDs, err := repo.ListMembershipIDs(ctx, testRoleA, false)
	require.NoError(t, err)
	require.Equal(t, []uint32{testMembershipID}, membershipIDs, "按角色查询应返回关联的成员 ID")
	membershipIDsByRoles, err := repo.ListMembershipIDsByRoleIDs(ctx, []uint32{testRoleA, testRoleB}, false)
	require.NoError(t, err)
	require.Equal(t, []uint32{testMembershipID, testMembershipID}, membershipIDsByRoles, "按角色列表查询应每角色各返回一份关联")

	// 清理：按成员删除全部关联
	tx2, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback() }()
	require.NoError(t, repo.CleanRelationsByMembershipID(ctx, tx2, testMembershipID))
	require.NoError(t, tx2.Commit())

	rowCount, err = repo.entClient.Client().MembershipRole.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, rowCount, "清理后 sys_membership_roles 计数应归零")

	empty, err := repo.ListRoleIDs(ctx, testMembershipID, false)
	require.NoError(t, err)
	require.Empty(t, empty, "清理后按成员查询应无角色 ID")
}

// TestMembershipRoleRepoSqlite_RemoveRolesFromMembership 验证 RemoveRolesFromMembership
// 的单向解除语义与按角色清理路径。
func TestMembershipRoleRepoSqlite_RemoveRolesFromMembership(t *testing.T) {
	repo := newMembershipRoleRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const (
		testMembershipID = uint32(8102)
		testRoleA        = uint32(9103)
		testRoleB        = uint32(9104)
	)

	tx, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.AssignMembershipRoles(ctx, tx, testMembershipID, []*permissionV1.MembershipRole{
		{MembershipId: trans.Ptr(testMembershipID), RoleId: trans.Ptr(testRoleA), Status: trans.Ptr(permissionV1.MembershipRole_ACTIVE)},
		{MembershipId: trans.Ptr(testMembershipID), RoleId: trans.Ptr(testRoleB), Status: trans.Ptr(permissionV1.MembershipRole_ACTIVE)},
	}))
	require.NoError(t, tx.Commit())

	// 单向解除 roleA：仅剩 roleB
	require.NoError(t, repo.RemoveRolesFromMembership(ctx, testMembershipID, []uint32{testRoleA}))

	roleIDs, err := repo.ListRoleIDs(ctx, testMembershipID, false)
	require.NoError(t, err)
	require.Equal(t, []uint32{testRoleB}, roleIDs, "解除后应仅保留未被移除的关联")

	// 按角色清理：roleB 的关联被整体删除
	tx2, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback() }()
	require.NoError(t, repo.CleanRelationsByRoleIDs(ctx, tx2, []uint32{testRoleB}))
	require.NoError(t, tx2.Commit())

	rowCount, err := repo.entClient.Client().MembershipRole.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, rowCount, "按角色清理后 sys_membership_roles 计数应归零")
}

// TestMembershipRoleRepoSqlite_ExcludeExpired 验证 ListRoleIDs/ListMembershipIDs 的
// excludeExpired 语义：过期（end_at 早于当前时刻）的关联在过滤后被排除。
func TestMembershipRoleRepoSqlite_ExcludeExpired(t *testing.T) {
	repo := newMembershipRoleRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const (
		testMembershipID = uint32(8103)
		testRoleOK       = uint32(9105)
		testRoleExp      = uint32(9106)
	)

	past := timestamppb.New(time.Now().Add(-1 * time.Hour))

	tx, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.AssignMembershipRoles(ctx, tx, testMembershipID, []*permissionV1.MembershipRole{
		{MembershipId: trans.Ptr(testMembershipID), RoleId: trans.Ptr(testRoleOK), Status: trans.Ptr(permissionV1.MembershipRole_ACTIVE)},
		{MembershipId: trans.Ptr(testMembershipID), RoleId: trans.Ptr(testRoleExp), Status: trans.Ptr(permissionV1.MembershipRole_ACTIVE), EndAt: past},
	}))
	require.NoError(t, tx.Commit())

	// 不过滤：两条都可见
	all, err := repo.ListRoleIDs(ctx, testMembershipID, false)
	require.NoError(t, err)
	require.Len(t, all, 2, "不过滤时应返回全部 2 条关联")

	// 过滤过期：仅剩未过期那条
	active, err := repo.ListRoleIDs(ctx, testMembershipID, true)
	require.NoError(t, err)
	require.Equal(t, []uint32{testRoleOK}, active, "过滤过期后应仅返回未过期关联")

	// 反向过滤同理
	activeMemberships, err := repo.ListMembershipIDs(ctx, testRoleExp, true)
	require.NoError(t, err)
	require.Empty(t, activeMemberships, "过期角色在过滤后的反向查询中不应出现")

	// 收尾清理
	tx2, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback() }()
	require.NoError(t, repo.CleanRelationsByMembershipIDs(ctx, tx2, []uint32{testMembershipID}))
	require.NoError(t, tx2.Commit())

	rowCount, err := repo.entClient.Client().MembershipRole.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, rowCount, "清理后 sys_membership_roles 计数应归零")
}
