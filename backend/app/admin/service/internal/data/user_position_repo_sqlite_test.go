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
	entUserPosition "go-wind-admin/app/admin/service/internal/data/ent/userposition"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newUserPositionRepoSqlite 用 enttest helper 构造一个可直接做关联 CRUD 的 UserPositionRepo。
// 白盒构造逐字段复刻 NewUserPositionRepo 的初始化（仅 log 换 NopLogger、entClient 换测试 client）。
func newUserPositionRepoSqlite(t *testing.T) *UserPositionRepo {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	return &UserPositionRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		statusConverter: mapper.NewEnumTypeConverter[identityV1.UserPosition_Status, entUserPosition.Status](
			identityV1.UserPosition_Status_name,
			identityV1.UserPosition_Status_value,
		),
	}
}

// TestUserPositionRepoSqlite_AssignListAndClean 覆盖 UserPositionRepo 的关联生命周期：
// AssignUserPosition(s)（写入）→ ent client 直查确认落库 →
// ListPositionIDs/ListUserIDs（正反向查询）→ CleanRelationsByXxx（清理）→ 计数归零。
func TestUserPositionRepoSqlite_AssignListAndClean(t *testing.T) {
	repo := newUserPositionRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const (
		testUserID = uint32(5101)
		testPosA   = uint32(6101)
		testPosB   = uint32(6102)
	)

	// 单条分配（无预清理语义）
	tx, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	// 无论断言成败都释放事务连接：Commit 成功后 Rollback 返回 ErrTxDone 被忽略；
	// 断言失败（Goexit）时回滚，避免残留写锁阻塞后续测试。
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.AssignUserPosition(ctx, tx, &identityV1.UserPosition{
		UserId:     trans.Ptr(testUserID),
		PositionId: trans.Ptr(testPosA),
		Status:     trans.Ptr(identityV1.UserPosition_ACTIVE),
	}))
	require.NoError(t, tx.Commit())

	// 批量分配：整体替换为两条
	tx2, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	// 无论断言成败都释放事务连接：Commit 成功后 Rollback 返回 ErrTxDone 被忽略；
	// 断言失败（Goexit）时回滚，避免残留写锁阻塞后续测试。
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.AssignUserPositions(ctx, tx2, testUserID, []*identityV1.UserPosition{
		{UserId: trans.Ptr(testUserID), PositionId: trans.Ptr(testPosA), Status: trans.Ptr(identityV1.UserPosition_ACTIVE)},
		{UserId: trans.Ptr(testUserID), PositionId: trans.Ptr(testPosB), Status: trans.Ptr(identityV1.UserPosition_ACTIVE)},
	}))
	require.NoError(t, tx2.Commit())

	// ent client 直查：两条关联确实落库
	rowCount, err := repo.entClient.Client().UserPosition.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, rowCount, "sys_user_positions 应有 2 条关联记录")

	// 正向：按用户列出岗位 ID
	positionIDs, err := repo.ListPositionIDs(ctx, testUserID, false)
	require.NoError(t, err)
	require.ElementsMatch(t, []uint32{testPosA, testPosB}, positionIDs, "按用户查询应返回两条关联的岗位 ID")

	// 反向：按岗位列出用户 ID（含多岗位批量变体，按岗位逐个收集不去重）
	userIDs, err := repo.ListUserIDs(ctx, testPosA, false)
	require.NoError(t, err)
	require.Equal(t, []uint32{testUserID}, userIDs, "按岗位查询应返回关联的用户 ID")
	userIDsByPositions, err := repo.ListUserIDsByPositionIDs(ctx, []uint32{testPosA, testPosB}, false)
	require.NoError(t, err)
	require.Equal(t, []uint32{testUserID, testUserID}, userIDsByPositions, "按岗位列表查询应每岗位各返回一份关联")

	// 清理：按用户删除全部关联
	tx3, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	// 无论断言成败都释放事务连接：Commit 成功后 Rollback 返回 ErrTxDone 被忽略；
	// 断言失败（Goexit）时回滚，避免残留写锁阻塞后续测试。
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.CleanRelationsByUserID(ctx, tx3, testUserID))
	require.NoError(t, tx3.Commit())

	rowCount, err = repo.entClient.Client().UserPosition.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, rowCount, "清理后 sys_user_positions 计数应归零")

	empty, err := repo.ListPositionIDs(ctx, testUserID, false)
	require.NoError(t, err)
	require.Empty(t, empty, "清理后按用户查询应无岗位 ID")
}

// TestUserPositionRepoSqlite_RemoveAndCleanByPosition 验证
// RemovePositionsFromUser 的单向解除语义与按岗位清理路径。
func TestUserPositionRepoSqlite_RemoveAndCleanByPosition(t *testing.T) {
	repo := newUserPositionRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const (
		testUserID = uint32(5102)
		testPosA   = uint32(6103)
		testPosB   = uint32(6104)
	)

	tx, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	// 无论断言成败都释放事务连接：Commit 成功后 Rollback 返回 ErrTxDone 被忽略；
	// 断言失败（Goexit）时回滚，避免残留写锁阻塞后续测试。
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.AssignUserPositions(ctx, tx, testUserID, []*identityV1.UserPosition{
		{UserId: trans.Ptr(testUserID), PositionId: trans.Ptr(testPosA), Status: trans.Ptr(identityV1.UserPosition_ACTIVE)},
		{UserId: trans.Ptr(testUserID), PositionId: trans.Ptr(testPosB), Status: trans.Ptr(identityV1.UserPosition_ACTIVE)},
	}))
	require.NoError(t, tx.Commit())

	// 单向解除 posA：仅剩 posB
	require.NoError(t, repo.RemovePositionsFromUser(ctx, testUserID, []uint32{testPosA}))

	positionIDs, err := repo.ListPositionIDs(ctx, testUserID, false)
	require.NoError(t, err)
	require.Equal(t, []uint32{testPosB}, positionIDs, "解除后应仅保留未被移除的关联")

	// 按岗位清理：posB 的关联被整体删除
	tx2, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	// 无论断言成败都释放事务连接：Commit 成功后 Rollback 返回 ErrTxDone 被忽略；
	// 断言失败（Goexit）时回滚，避免残留写锁阻塞后续测试。
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.CleanRelationsByPositionIDs(ctx, tx2, []uint32{testPosB}))
	require.NoError(t, tx2.Commit())

	rowCount, err := repo.entClient.Client().UserPosition.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, rowCount, "按岗位清理后 sys_user_positions 计数应归零")
}

// TestUserPositionRepoSqlite_ExcludeExpired 验证 ListPositionIDs/ListUserIDs 的
// excludeExpired 语义：过期（end_at 早于当前时刻）的关联在过滤后被排除。
func TestUserPositionRepoSqlite_ExcludeExpired(t *testing.T) {
	repo := newUserPositionRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const (
		testUserID = uint32(5103)
		testPosOK  = uint32(6105)
		testPosEx  = uint32(6106)
	)

	past := timestamppb.New(time.Now().Add(-1 * time.Hour))

	tx, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	// 无论断言成败都释放事务连接：Commit 成功后 Rollback 返回 ErrTxDone 被忽略；
	// 断言失败（Goexit）时回滚，避免残留写锁阻塞后续测试。
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.AssignUserPositions(ctx, tx, testUserID, []*identityV1.UserPosition{
		{UserId: trans.Ptr(testUserID), PositionId: trans.Ptr(testPosOK), Status: trans.Ptr(identityV1.UserPosition_ACTIVE)},
		{UserId: trans.Ptr(testUserID), PositionId: trans.Ptr(testPosEx), Status: trans.Ptr(identityV1.UserPosition_ACTIVE), EndAt: past},
	}))
	require.NoError(t, tx.Commit())

	// 不过滤：两条都可见
	all, err := repo.ListPositionIDs(ctx, testUserID, false)
	require.NoError(t, err)
	require.Len(t, all, 2, "不过滤时应返回全部 2 条关联")

	// 过滤过期：仅剩未过期那条
	active, err := repo.ListPositionIDs(ctx, testUserID, true)
	require.NoError(t, err)
	require.Equal(t, []uint32{testPosOK}, active, "过滤过期后应仅返回未过期关联")

	// 反向过滤同理
	activeUsers, err := repo.ListUserIDs(ctx, testPosEx, true)
	require.NoError(t, err)
	require.Empty(t, activeUsers, "过期岗位在过滤后的反向查询中不应出现")

	// 收尾清理
	tx2, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	// 无论断言成败都释放事务连接：Commit 成功后 Rollback 返回 ErrTxDone 被忽略；
	// 断言失败（Goexit）时回滚，避免残留写锁阻塞后续测试。
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.CleanRelationsByUserIDs(ctx, tx2, []uint32{testUserID}))
	require.NoError(t, tx2.Commit())

	rowCount, err := repo.entClient.Client().UserPosition.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, rowCount, "清理后 sys_user_positions 计数应归零")
}
