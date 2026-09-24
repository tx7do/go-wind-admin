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
	entMembershipPosition "go-wind-admin/app/admin/service/internal/data/ent/membershipposition"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newMembershipPositionRepoSqlite 用 enttest helper 构造一个可直接做关联 CRUD 的 MembershipPositionRepo。
// 白盒构造逐字段复刻 NewMembershipPositionRepo 的初始化（仅 log 换 NopLogger、entClient 换测试 client）。
func newMembershipPositionRepoSqlite(t *testing.T) *MembershipPositionRepo {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	return &MembershipPositionRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		statusConverter: mapper.NewEnumTypeConverter[identityV1.MembershipPosition_Status, entMembershipPosition.Status](
			identityV1.MembershipPosition_Status_name,
			identityV1.MembershipPosition_Status_value,
		),
	}
}

// TestMembershipPositionRepoSqlite_AssignListAndClean 覆盖 MembershipPositionRepo 的关联生命周期：
// AssignMembershipPositions（写入）→ ent client 直查确认落库 →
// ListPositionIDs/ListMembershipIDs（正反向查询）→ CleanRelationsByMembershipID（清理）→ 计数归零。
func TestMembershipPositionRepoSqlite_AssignListAndClean(t *testing.T) {
	repo := newMembershipPositionRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const (
		testMembershipID = uint32(8301)
		testPosA         = uint32(9301)
		testPosB         = uint32(9302)
	)

	tx, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	// 无论断言成败都释放事务连接：Commit 成功后 Rollback 返回 ErrTxDone 被忽略；
	// 断言失败（Goexit）时回滚，避免残留写锁阻塞后续测试。
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.AssignMembershipPositions(ctx, tx, testMembershipID, []*identityV1.MembershipPosition{
		{
			MembershipId: trans.Ptr(testMembershipID),
			PositionId:   trans.Ptr(testPosA),
			Status:       trans.Ptr(identityV1.MembershipPosition_ACTIVE),
		},
		{
			MembershipId: trans.Ptr(testMembershipID),
			PositionId:   trans.Ptr(testPosB),
			Status:       trans.Ptr(identityV1.MembershipPosition_ACTIVE),
		},
	}), "批量分配两条成员-岗位关联应成功")
	require.NoError(t, tx.Commit())

	// ent client 直查：两条关联确实落库
	rowCount, err := repo.entClient.Client().MembershipPosition.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, rowCount, "sys_membership_positions 应有 2 条关联记录")

	// 正向：按成员列出岗位 ID
	positionIDs, err := repo.ListPositionIDs(ctx, testMembershipID, false)
	require.NoError(t, err)
	require.ElementsMatch(t, []uint32{testPosA, testPosB}, positionIDs, "按成员查询应返回两条关联的岗位 ID")

	// 反向：按岗位列出成员 ID（含多岗位批量变体，按岗位逐个收集不去重）
	membershipIDs, err := repo.ListMembershipIDs(ctx, testPosA, false)
	require.NoError(t, err)
	require.Equal(t, []uint32{testMembershipID}, membershipIDs, "按岗位查询应返回关联的成员 ID")
	membershipIDsByPositions, err := repo.ListMembershipIDsByPositionIDs(ctx, []uint32{testPosA, testPosB}, false)
	require.NoError(t, err)
	require.Equal(t, []uint32{testMembershipID, testMembershipID}, membershipIDsByPositions, "按岗位列表查询应每岗位各返回一份关联")

	// 清理：按成员删除全部关联
	tx2, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback() }()
	require.NoError(t, repo.CleanRelationsByMembershipID(ctx, tx2, testMembershipID))
	require.NoError(t, tx2.Commit())

	rowCount, err = repo.entClient.Client().MembershipPosition.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, rowCount, "清理后 sys_membership_positions 计数应归零")

	empty, err := repo.ListPositionIDs(ctx, testMembershipID, false)
	require.NoError(t, err)
	require.Empty(t, empty, "清理后按成员查询应无岗位 ID")
}

// TestMembershipPositionRepoSqlite_RemoveAndCleanByPosition 验证
// RemovePositionsFromMembership 的单向解除语义与按岗位清理路径。
func TestMembershipPositionRepoSqlite_RemoveAndCleanByPosition(t *testing.T) {
	repo := newMembershipPositionRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const (
		testMembershipID = uint32(8302)
		testPosA         = uint32(9303)
		testPosB         = uint32(9304)
	)

	tx, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.AssignMembershipPositions(ctx, tx, testMembershipID, []*identityV1.MembershipPosition{
		{MembershipId: trans.Ptr(testMembershipID), PositionId: trans.Ptr(testPosA), Status: trans.Ptr(identityV1.MembershipPosition_ACTIVE)},
		{MembershipId: trans.Ptr(testMembershipID), PositionId: trans.Ptr(testPosB), Status: trans.Ptr(identityV1.MembershipPosition_ACTIVE)},
	}))
	require.NoError(t, tx.Commit())

	// 单向解除 posA：仅剩 posB
	require.NoError(t, repo.RemovePositionsFromMembership(ctx, testMembershipID, []uint32{testPosA}))

	positionIDs, err := repo.ListPositionIDs(ctx, testMembershipID, false)
	require.NoError(t, err)
	require.Equal(t, []uint32{testPosB}, positionIDs, "解除后应仅保留未被移除的关联")

	// 按岗位清理：posB 的关联被整体删除
	tx2, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback() }()
	require.NoError(t, repo.CleanRelationsByPositionIDs(ctx, tx2, []uint32{testPosB}))
	require.NoError(t, tx2.Commit())

	rowCount, err := repo.entClient.Client().MembershipPosition.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, rowCount, "按岗位清理后 sys_membership_positions 计数应归零")
}

// TestMembershipPositionRepoSqlite_ExcludeExpired 验证 ListPositionIDs/ListMembershipIDs 的
// excludeExpired 语义：过期（end_at 早于当前时刻）的关联在过滤后被排除。
func TestMembershipPositionRepoSqlite_ExcludeExpired(t *testing.T) {
	repo := newMembershipPositionRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const (
		testMembershipID = uint32(8303)
		testPosOK        = uint32(9305)
		testPosExp       = uint32(9306)
	)

	past := timestamppb.New(time.Now().Add(-1 * time.Hour))

	tx, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.AssignMembershipPositions(ctx, tx, testMembershipID, []*identityV1.MembershipPosition{
		{MembershipId: trans.Ptr(testMembershipID), PositionId: trans.Ptr(testPosOK), Status: trans.Ptr(identityV1.MembershipPosition_ACTIVE)},
		{MembershipId: trans.Ptr(testMembershipID), PositionId: trans.Ptr(testPosExp), Status: trans.Ptr(identityV1.MembershipPosition_ACTIVE), EndAt: past},
	}))
	require.NoError(t, tx.Commit())

	// 不过滤：两条都可见
	all, err := repo.ListPositionIDs(ctx, testMembershipID, false)
	require.NoError(t, err)
	require.Len(t, all, 2, "不过滤时应返回全部 2 条关联")

	// 过滤过期：仅剩未过期那条
	active, err := repo.ListPositionIDs(ctx, testMembershipID, true)
	require.NoError(t, err)
	require.Equal(t, []uint32{testPosOK}, active, "过滤过期后应仅返回未过期关联")

	// 反向过滤同理
	activeMemberships, err := repo.ListMembershipIDs(ctx, testPosExp, true)
	require.NoError(t, err)
	require.Empty(t, activeMemberships, "过期岗位在过滤后的反向查询中不应出现")

	// 收尾清理
	tx2, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback() }()
	require.NoError(t, repo.CleanRelationsByMembershipIDs(ctx, tx2, []uint32{testMembershipID}))
	require.NoError(t, tx2.Commit())

	rowCount, err := repo.entClient.Client().MembershipPosition.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, rowCount, "清理后 sys_membership_positions 计数应归零")
}
