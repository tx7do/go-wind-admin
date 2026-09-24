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
	entUserRole "go-wind-admin/app/admin/service/internal/data/ent/userrole"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newUserRoleRepoSqlite 用 enttest helper 构造一个可直接做关联 CRUD 的 UserRoleRepo。
// 白盒构造逐字段复刻 NewUserRoleRepo 的初始化（仅 log 换 NopLogger、entClient 换测试 client）。
func newUserRoleRepoSqlite(t *testing.T) *UserRoleRepo {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	return &UserRoleRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		statusConverter: mapper.NewEnumTypeConverter[permissionV1.UserRole_Status, entUserRole.Status](
			permissionV1.UserRole_Status_name,
			permissionV1.UserRole_Status_value,
		),
	}
}

// TestUserRoleRepoSqlite_AssignListAndClean 覆盖 UserRoleRepo 的关联生命周期：
// AssignUserRoles（写入）→ ent client 直查确认落库 → ListRoleIDs/ListUserIDs（正反向查询）
// → CleanRelationsByUserID（清理）→ 计数归零。
func TestUserRoleRepoSqlite_AssignListAndClean(t *testing.T) {
	repo := newUserRoleRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const (
		testUserID = uint32(1101)
		testRoleA  = uint32(2201)
		testRoleB  = uint32(2202)
	)

	tx, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	// 无论断言成败都释放事务连接：Commit 成功后 Rollback 返回 ErrTxDone 被忽略；
	// 断言失败（Goexit）时回滚，避免残留写锁阻塞后续测试。
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.AssignUserRoles(ctx, tx, testUserID, []*permissionV1.UserRole{
		{
			UserId: trans.Ptr(testUserID),
			RoleId: trans.Ptr(testRoleA),
			Status: trans.Ptr(permissionV1.UserRole_ACTIVE),
		},
		{
			UserId: trans.Ptr(testUserID),
			RoleId: trans.Ptr(testRoleB),
			Status: trans.Ptr(permissionV1.UserRole_ACTIVE),
		},
	}), "批量分配两条用户-角色关联应成功")
	require.NoError(t, tx.Commit())

	// ent client 直查：两条关联确实落库
	rowCount, err := repo.entClient.Client().UserRole.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, rowCount, "sys_user_roles 应有 2 条关联记录")

	// 正向：按用户列出角色 ID
	roleIDs, err := repo.ListRoleIDs(ctx, testUserID, false)
	require.NoError(t, err)
	require.ElementsMatch(t, []uint32{testRoleA, testRoleB}, roleIDs, "按用户查询应返回两条关联的角色 ID")

	// 反向：按角色列出用户 ID（含多角色批量变体）
	userIDs, err := repo.ListUserIDs(ctx, testRoleA, false)
	require.NoError(t, err)
	require.Equal(t, []uint32{testUserID}, userIDs, "按角色查询应返回关联的用户 ID")
	userIDsByRoles, err := repo.ListUserIDsByRoleIDs(ctx, []uint32{testRoleA, testRoleB}, false)
	require.NoError(t, err)
	// 该仓库实现按角色逐个收集、不去重：两个角色各自关联同一用户，返回两份该用户 ID
	require.Len(t, userIDsByRoles, 2, "按角色列表查询应返回两份关联（每角色一份）")
	require.Equal(t, []uint32{testUserID, testUserID}, userIDsByRoles, "两份关联均应指向该用户")

	// 清理：按用户删除全部关联
	tx2, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	// 无论断言成败都释放事务连接：Commit 成功后 Rollback 返回 ErrTxDone 被忽略；
	// 断言失败（Goexit）时回滚，避免残留写锁阻塞后续测试。
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.CleanRelationsByUserID(ctx, tx2, testUserID))
	require.NoError(t, tx2.Commit())

	rowCount, err = repo.entClient.Client().UserRole.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, rowCount, "清理后 sys_user_roles 计数应归零")

	empty, err := repo.ListRoleIDs(ctx, testUserID, false)
	require.NoError(t, err)
	require.Empty(t, empty, "清理后按用户查询应无角色 ID")
}

// TestUserRoleRepoSqlite_ReplaceSemantics 验证 AssignUserRoles 的整体替换语义：
// 再次分配会先清空该用户既有关联，再写入新集合。
func TestUserRoleRepoSqlite_ReplaceSemantics(t *testing.T) {
	repo := newUserRoleRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const (
		testUserID = uint32(1102)
		testRoleA  = uint32(2203)
		testRoleC  = uint32(2204)
	)

	tx, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	// 无论断言成败都释放事务连接：Commit 成功后 Rollback 返回 ErrTxDone 被忽略；
	// 断言失败（Goexit）时回滚，避免残留写锁阻塞后续测试。
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.AssignUserRoles(ctx, tx, testUserID, []*permissionV1.UserRole{
		{UserId: trans.Ptr(testUserID), RoleId: trans.Ptr(testRoleA), Status: trans.Ptr(permissionV1.UserRole_ACTIVE)},
	}))
	require.NoError(t, tx.Commit())

	// 第二次分配：既有 roleA 应被替换为 roleC
	tx2, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	// 无论断言成败都释放事务连接：Commit 成功后 Rollback 返回 ErrTxDone 被忽略；
	// 断言失败（Goexit）时回滚，避免残留写锁阻塞后续测试。
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.AssignUserRoles(ctx, tx2, testUserID, []*permissionV1.UserRole{
		{UserId: trans.Ptr(testUserID), RoleId: trans.Ptr(testRoleC), Status: trans.Ptr(permissionV1.UserRole_ACTIVE)},
	}))
	require.NoError(t, tx2.Commit())

	roleIDs, err := repo.ListRoleIDs(ctx, testUserID, false)
	require.NoError(t, err)
	require.Equal(t, []uint32{testRoleC}, roleIDs, "再次分配后仅应存在新集合中的角色 ID")

	rowCount, err := repo.entClient.Client().UserRole.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, rowCount, "替换后表内应只有 1 条关联")

	// 收尾清理
	tx3, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	// 无论断言成败都释放事务连接：Commit 成功后 Rollback 返回 ErrTxDone 被忽略；
	// 断言失败（Goexit）时回滚，避免残留写锁阻塞后续测试。
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.CleanRelationsByUserID(ctx, tx3, testUserID))
	require.NoError(t, tx3.Commit())
}

// TestUserRoleRepoSqlite_RemoveRolesFromUser 验证 RemoveRolesFromUser 的
// 单向解除语义：只移除指定角色关联，其余关联保留。
func TestUserRoleRepoSqlite_RemoveRolesFromUser(t *testing.T) {
	repo := newUserRoleRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const (
		testUserID = uint32(1103)
		testRoleA  = uint32(2205)
		testRoleB  = uint32(2206)
	)

	tx, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	// 无论断言成败都释放事务连接：Commit 成功后 Rollback 返回 ErrTxDone 被忽略；
	// 断言失败（Goexit）时回滚，避免残留写锁阻塞后续测试。
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.AssignUserRoles(ctx, tx, testUserID, []*permissionV1.UserRole{
		{UserId: trans.Ptr(testUserID), RoleId: trans.Ptr(testRoleA), Status: trans.Ptr(permissionV1.UserRole_ACTIVE)},
		{UserId: trans.Ptr(testUserID), RoleId: trans.Ptr(testRoleB), Status: trans.Ptr(permissionV1.UserRole_ACTIVE)},
	}))
	require.NoError(t, tx.Commit())

	// 单向解除 roleA：仅剩 roleB
	require.NoError(t, repo.RemoveRolesFromUser(ctx, testUserID, []uint32{testRoleA}))

	roleIDs, err := repo.ListRoleIDs(ctx, testUserID, false)
	require.NoError(t, err)
	require.Equal(t, []uint32{testRoleB}, roleIDs, "解除后应仅保留未被移除的关联")

	// 收尾清理
	tx2, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	// 无论断言成败都释放事务连接：Commit 成功后 Rollback 返回 ErrTxDone 被忽略；
	// 断言失败（Goexit）时回滚，避免残留写锁阻塞后续测试。
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.CleanRelationsByUserID(ctx, tx2, testUserID))
	require.NoError(t, tx2.Commit())

	rowCount, err := repo.entClient.Client().UserRole.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, rowCount, "清理后 sys_user_roles 计数应归零")
}

// TestUserRoleRepoSqlite_ExcludeExpired 验证 ListRoleIDs/ListUserIDs 的
// excludeExpired 语义：过期（end_at 早于当前时刻）的关联在过滤后被排除。
func TestUserRoleRepoSqlite_ExcludeExpired(t *testing.T) {
	repo := newUserRoleRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const (
		testUserID  = uint32(1104)
		testRoleOK  = uint32(2207)
		testRoleExp = uint32(2208)
	)

	past := timestamppb.New(time.Now().Add(-1 * time.Hour))

	tx, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	// 无论断言成败都释放事务连接：Commit 成功后 Rollback 返回 ErrTxDone 被忽略；
	// 断言失败（Goexit）时回滚，避免残留写锁阻塞后续测试。
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.AssignUserRoles(ctx, tx, testUserID, []*permissionV1.UserRole{
		{UserId: trans.Ptr(testUserID), RoleId: trans.Ptr(testRoleOK), Status: trans.Ptr(permissionV1.UserRole_ACTIVE)},
		{UserId: trans.Ptr(testUserID), RoleId: trans.Ptr(testRoleExp), Status: trans.Ptr(permissionV1.UserRole_ACTIVE), EndAt: past},
	}))
	require.NoError(t, tx.Commit())

	// 不过滤：两条都可见
	all, err := repo.ListRoleIDs(ctx, testUserID, false)
	require.NoError(t, err)
	require.Len(t, all, 2, "不过滤时应返回全部 2 条关联")

	// 过滤过期：仅剩未过期那条
	active, err := repo.ListRoleIDs(ctx, testUserID, true)
	require.NoError(t, err)
	require.Equal(t, []uint32{testRoleOK}, active, "过滤过期后应仅返回未过期关联")

	// 反向过滤同理
	activeUsers, err := repo.ListUserIDs(ctx, testRoleExp, true)
	require.NoError(t, err)
	require.Empty(t, activeUsers, "过期角色在过滤后的反向查询中不应出现")

	// 收尾清理
	tx2, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	// 无论断言成败都释放事务连接：Commit 成功后 Rollback 返回 ErrTxDone 被忽略；
	// 断言失败（Goexit）时回滚，避免残留写锁阻塞后续测试。
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.CleanRelationsByUserIDs(ctx, tx2, []uint32{testUserID}),
		"按用户列表清理应成功")
	require.NoError(t, tx2.Commit())

	rowCount, err := repo.entClient.Client().UserRole.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, rowCount, "清理后 sys_user_roles 计数应归零")
}
