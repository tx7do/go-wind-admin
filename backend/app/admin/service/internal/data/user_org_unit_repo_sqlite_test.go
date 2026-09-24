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

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	entUserOrgUnit "go-wind-admin/app/admin/service/internal/data/ent/userorgunit"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newUserOrgUnitRepoSqlite 用 enttest helper 构造一个可直接做关联 CRUD 的 UserOrgUnitRepo。
// 白盒构造逐字段复刻 NewUserOrgUnitRepo 的初始化（仅 log 换 NopLogger、entClient 换测试 client）。
func newUserOrgUnitRepoSqlite(t *testing.T) *UserOrgUnitRepo {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	return &UserOrgUnitRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		statusConverter: mapper.NewEnumTypeConverter[identityV1.UserOrgUnit_Status, entUserOrgUnit.Status](
			identityV1.UserOrgUnit_Status_name,
			identityV1.UserOrgUnit_Status_value,
		),
	}
}

// TestUserOrgUnitRepoSqlite_AssignListAndClean 覆盖 UserOrgUnitRepo 的关联生命周期：
// AssignUserOrgUnit(s)（写入）→ ent client 直查确认落库 →
// ListOrgUnitIDs/ListUserIDs（正反向查询）→ CleanRelationsByXxx（清理）→ 计数归零。
func TestUserOrgUnitRepoSqlite_AssignListAndClean(t *testing.T) {
	repo := newUserOrgUnitRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const (
		testUserID = uint32(3101)
		testUnitA  = uint32(4101)
		testUnitB  = uint32(4102)
	)

	// 单条分配（无预清理语义）
	tx, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	// 无论断言成败都释放事务连接：Commit 成功后 Rollback 返回 ErrTxDone 被忽略；
	// 断言失败（Goexit）时回滚，避免残留写锁阻塞后续测试。
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.AssignUserOrgUnit(ctx, tx, &identityV1.UserOrgUnit{
		UserId:    trans.Ptr(testUserID),
		OrgUnitId: trans.Ptr(testUnitA),
		Status:    trans.Ptr(identityV1.UserOrgUnit_ACTIVE),
	}))
	require.NoError(t, tx.Commit())

	// 批量分配：整体替换为两条
	tx2, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	// 无论断言成败都释放事务连接：Commit 成功后 Rollback 返回 ErrTxDone 被忽略；
	// 断言失败（Goexit）时回滚，避免残留写锁阻塞后续测试。
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.AssignUserOrgUnits(ctx, tx2, testUserID, []*identityV1.UserOrgUnit{
		// 两行 is_primary 取不同值：sys_user_org_units 上 (tenant_id, user_id, is_primary)
		// 是唯一索引，同主键的两行会被唯一约束拒绝。
		{UserId: trans.Ptr(testUserID), OrgUnitId: trans.Ptr(testUnitA), IsPrimary: trans.Ptr(false), Status: trans.Ptr(identityV1.UserOrgUnit_ACTIVE)},
		{UserId: trans.Ptr(testUserID), OrgUnitId: trans.Ptr(testUnitB), IsPrimary: trans.Ptr(true), Status: trans.Ptr(identityV1.UserOrgUnit_ACTIVE)},
	}))
	require.NoError(t, tx2.Commit())

	// ent client 直查：两条关联确实落库
	rowCount, err := repo.entClient.Client().UserOrgUnit.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, rowCount, "sys_user_org_units 应有 2 条关联记录")

	// 正向：按用户列出单元 ID
	unitIDs, err := repo.ListOrgUnitIDs(ctx, testUserID, false)
	require.NoError(t, err)
	require.ElementsMatch(t, []uint32{testUnitA, testUnitB}, unitIDs, "按用户查询应返回两条关联的单元 ID")

	// 反向：按单元列出用户 ID（含多单元批量变体，按单元逐个收集不去重）
	userIDs, err := repo.ListUserIDs(ctx, testUnitA, false)
	require.NoError(t, err)
	require.Equal(t, []uint32{testUserID}, userIDs, "按单元查询应返回关联的用户 ID")
	userIDsByUnits, err := repo.ListUserIDsByOrgUnitIDs(ctx, []uint32{testUnitA, testUnitB}, false)
	require.NoError(t, err)
	require.Equal(t, []uint32{testUserID, testUserID}, userIDsByUnits, "按单元列表查询应每单元各返回一份关联")

	// 清理：按用户删除全部关联
	tx3, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	// 无论断言成败都释放事务连接：Commit 成功后 Rollback 返回 ErrTxDone 被忽略；
	// 断言失败（Goexit）时回滚，避免残留写锁阻塞后续测试。
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.CleanRelationsByUserID(ctx, tx3, testUserID))
	require.NoError(t, tx3.Commit())

	rowCount, err = repo.entClient.Client().UserOrgUnit.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, rowCount, "清理后 sys_user_org_units 计数应归零")

	empty, err := repo.ListOrgUnitIDs(ctx, testUserID, false)
	require.NoError(t, err)
	require.Empty(t, empty, "清理后按用户查询应无单元 ID")
}

// TestUserOrgUnitRepoSqlite_RemoveAndCleanByOrgUnit 验证
// RemoveOrgUnitsFromUser 的单向解除语义与按单元清理路径。
func TestUserOrgUnitRepoSqlite_RemoveAndCleanByOrgUnit(t *testing.T) {
	repo := newUserOrgUnitRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const (
		testUserID = uint32(3102)
		testUnitA  = uint32(4103)
		testUnitB  = uint32(4104)
	)

	tx, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	// 无论断言成败都释放事务连接：Commit 成功后 Rollback 返回 ErrTxDone 被忽略；
	// 断言失败（Goexit）时回滚，避免残留写锁阻塞后续测试。
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.AssignUserOrgUnits(ctx, tx, testUserID, []*identityV1.UserOrgUnit{
		// 两行 is_primary 取不同值：sys_user_org_units 上 (tenant_id, user_id, is_primary)
		// 是唯一索引，同主键的两行会被唯一约束拒绝。
		{UserId: trans.Ptr(testUserID), OrgUnitId: trans.Ptr(testUnitA), IsPrimary: trans.Ptr(false), Status: trans.Ptr(identityV1.UserOrgUnit_ACTIVE)},
		{UserId: trans.Ptr(testUserID), OrgUnitId: trans.Ptr(testUnitB), IsPrimary: trans.Ptr(true), Status: trans.Ptr(identityV1.UserOrgUnit_ACTIVE)},
	}))
	require.NoError(t, tx.Commit())

	// 单向解除 unitA：仅剩 unitB
	require.NoError(t, repo.RemoveOrgUnitsFromUser(ctx, testUserID, []uint32{testUnitA}))

	unitIDs, err := repo.ListOrgUnitIDs(ctx, testUserID, false)
	require.NoError(t, err)
	require.Equal(t, []uint32{testUnitB}, unitIDs, "解除后应仅保留未被移除的关联")

	// 按单元清理：unitB 的关联被整体删除
	tx2, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	// 无论断言成败都释放事务连接：Commit 成功后 Rollback 返回 ErrTxDone 被忽略；
	// 断言失败（Goexit）时回滚，避免残留写锁阻塞后续测试。
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.CleanRelationsByOrgUnitIDs(ctx, tx2, []uint32{testUnitB}))
	require.NoError(t, tx2.Commit())

	rowCount, err := repo.entClient.Client().UserOrgUnit.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, rowCount, "按单元清理后 sys_user_org_units 计数应归零")
}

// TestUserOrgUnitRepoSqlite_ExcludeExpired 验证 ListOrgUnitIDs/ListUserIDs 的
// excludeExpired 语义：过期（end_at 早于当前时刻）的关联在过滤后被排除。
func TestUserOrgUnitRepoSqlite_ExcludeExpired(t *testing.T) {
	repo := newUserOrgUnitRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const (
		testUserID = uint32(3103)
		testUnitOK = uint32(4105)
		testUnitEx = uint32(4106)
	)

	past := timestamppb.New(time.Now().Add(-1 * time.Hour))

	tx, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	// 无论断言成败都释放事务连接：Commit 成功后 Rollback 返回 ErrTxDone 被忽略；
	// 断言失败（Goexit）时回滚，避免残留写锁阻塞后续测试。
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.AssignUserOrgUnits(ctx, tx, testUserID, []*identityV1.UserOrgUnit{
		// 同上：is_primary 取不同值以满足唯一索引。
		{UserId: trans.Ptr(testUserID), OrgUnitId: trans.Ptr(testUnitOK), IsPrimary: trans.Ptr(false), Status: trans.Ptr(identityV1.UserOrgUnit_ACTIVE)},
		{UserId: trans.Ptr(testUserID), OrgUnitId: trans.Ptr(testUnitEx), IsPrimary: trans.Ptr(true), Status: trans.Ptr(identityV1.UserOrgUnit_ACTIVE), EndAt: past},
	}))
	require.NoError(t, tx.Commit())

	// 不过滤：两条都可见
	all, err := repo.ListOrgUnitIDs(ctx, testUserID, false)
	require.NoError(t, err)
	require.Len(t, all, 2, "不过滤时应返回全部 2 条关联")

	// 过滤过期：仅剩未过期那条
	active, err := repo.ListOrgUnitIDs(ctx, testUserID, true)
	require.NoError(t, err)
	require.Equal(t, []uint32{testUnitOK}, active, "过滤过期后应仅返回未过期关联")

	// 反向过滤同理
	activeUsers, err := repo.ListUserIDs(ctx, testUnitEx, true)
	require.NoError(t, err)
	require.Empty(t, activeUsers, "过期单元在过滤后的反向查询中不应出现")

	// 收尾清理
	tx2, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	// 无论断言成败都释放事务连接：Commit 成功后 Rollback 返回 ErrTxDone 被忽略；
	// 断言失败（Goexit）时回滚，避免残留写锁阻塞后续测试。
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.CleanRelationsByUserIDs(ctx, tx2, []uint32{testUserID}))
	require.NoError(t, tx2.Commit())

	rowCount, err := repo.entClient.Client().UserOrgUnit.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, rowCount, "清理后 sys_user_org_units 计数应归零")
}
