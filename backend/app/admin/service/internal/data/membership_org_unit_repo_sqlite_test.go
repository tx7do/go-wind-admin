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
	entMembershipOrgUnit "go-wind-admin/app/admin/service/internal/data/ent/membershiporgunit"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newMembershipOrgUnitRepoSqlite 用 enttest helper 构造一个可直接做关联 CRUD 的 MembershipOrgUnitRepo。
// 白盒构造逐字段复刻 NewMembershipOrgUnitRepo 的初始化（仅 log 换 NopLogger、entClient 换测试 client）。
func newMembershipOrgUnitRepoSqlite(t *testing.T) *MembershipOrgUnitRepo {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	return &MembershipOrgUnitRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		statusConverter: mapper.NewEnumTypeConverter[identityV1.MembershipOrgUnit_Status, entMembershipOrgUnit.Status](
			identityV1.MembershipOrgUnit_Status_name,
			identityV1.MembershipOrgUnit_Status_value,
		),
	}
}

// TestMembershipOrgUnitRepoSqlite_AssignListAndClean 覆盖 MembershipOrgUnitRepo 的关联生命周期：
// AssignMembershipOrgUnits（写入）→ ent client 直查确认落库 →
// ListOrgUnitIDs/ListMembershipIDs（正反向查询）→ CleanRelationsByMembershipID（清理）→ 计数归零。
func TestMembershipOrgUnitRepoSqlite_AssignListAndClean(t *testing.T) {
	repo := newMembershipOrgUnitRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const (
		testMembershipID = uint32(8201)
		testUnitA        = uint32(9201)
		testUnitB        = uint32(9202)
	)

	tx, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	// 无论断言成败都释放事务连接：Commit 成功后 Rollback 返回 ErrTxDone 被忽略；
	// 断言失败（Goexit）时回滚，避免残留写锁阻塞后续测试。
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.AssignMembershipOrgUnits(ctx, tx, testMembershipID, []*identityV1.MembershipOrgUnit{
		{
			MembershipId: trans.Ptr(testMembershipID),
			OrgUnitId:    trans.Ptr(testUnitA),
			IsPrimary:    trans.Ptr(false),
			Status:       trans.Ptr(identityV1.MembershipOrgUnit_ACTIVE),
		},
		{
			MembershipId: trans.Ptr(testMembershipID),
			OrgUnitId:    trans.Ptr(testUnitB),
			IsPrimary:    trans.Ptr(true),
			Status:       trans.Ptr(identityV1.MembershipOrgUnit_ACTIVE),
		},
	}), "批量分配两条成员-单元关联应成功")
	require.NoError(t, tx.Commit())

	// ent client 直查：两条关联确实落库
	rowCount, err := repo.entClient.Client().MembershipOrgUnit.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, rowCount, "sys_membership_org_units 应有 2 条关联记录")

	// 正向：按成员列出单元 ID
	unitIDs, err := repo.ListOrgUnitIDs(ctx, testMembershipID, false)
	require.NoError(t, err)
	require.ElementsMatch(t, []uint32{testUnitA, testUnitB}, unitIDs, "按成员查询应返回两条关联的单元 ID")

	// 反向：按单元列出成员 ID（含多单元批量变体，按单元逐个收集不去重）
	membershipIDs, err := repo.ListMembershipIDs(ctx, testUnitA, false)
	require.NoError(t, err)
	require.Equal(t, []uint32{testMembershipID}, membershipIDs, "按单元查询应返回关联的成员 ID")
	membershipIDsByUnits, err := repo.ListMembershipIDsByOrgUnitIDs(ctx, []uint32{testUnitA, testUnitB}, false)
	require.NoError(t, err)
	require.Equal(t, []uint32{testMembershipID, testMembershipID}, membershipIDsByUnits, "按单元列表查询应每单元各返回一份关联")

	// 清理：按成员删除全部关联
	tx2, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback() }()
	require.NoError(t, repo.CleanRelationsByMembershipID(ctx, tx2, testMembershipID))
	require.NoError(t, tx2.Commit())

	rowCount, err = repo.entClient.Client().MembershipOrgUnit.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, rowCount, "清理后 sys_membership_org_units 计数应归零")

	empty, err := repo.ListOrgUnitIDs(ctx, testMembershipID, false)
	require.NoError(t, err)
	require.Empty(t, empty, "清理后按成员查询应无单元 ID")
}

// TestMembershipOrgUnitRepoSqlite_RemoveAndCleanByOrgUnit 验证
// RemoveOrgUnitsFromMembership 的单向解除语义与按单元清理路径。
func TestMembershipOrgUnitRepoSqlite_RemoveAndCleanByOrgUnit(t *testing.T) {
	repo := newMembershipOrgUnitRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const (
		testMembershipID = uint32(8202)
		testUnitA        = uint32(9203)
		testUnitB        = uint32(9204)
	)

	tx, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.AssignMembershipOrgUnits(ctx, tx, testMembershipID, []*identityV1.MembershipOrgUnit{
		// 两行 is_primary 取不同值：sys_membership_org_units 上 (tenant_id, membership_id, is_primary)
		// 是唯一索引，同主键的两行会被唯一约束拒绝。
		{MembershipId: trans.Ptr(testMembershipID), OrgUnitId: trans.Ptr(testUnitA), IsPrimary: trans.Ptr(false), Status: trans.Ptr(identityV1.MembershipOrgUnit_ACTIVE)},
		{MembershipId: trans.Ptr(testMembershipID), OrgUnitId: trans.Ptr(testUnitB), IsPrimary: trans.Ptr(true), Status: trans.Ptr(identityV1.MembershipOrgUnit_ACTIVE)},
	}))
	require.NoError(t, tx.Commit())

	// 单向解除 unitA：仅剩 unitB
	require.NoError(t, repo.RemoveOrgUnitsFromMembership(ctx, testMembershipID, []uint32{testUnitA}))

	unitIDs, err := repo.ListOrgUnitIDs(ctx, testMembershipID, false)
	require.NoError(t, err)
	require.Equal(t, []uint32{testUnitB}, unitIDs, "解除后应仅保留未被移除的关联")

	// 按单元清理：unitB 的关联被整体删除
	tx2, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback() }()
	require.NoError(t, repo.CleanRelationsByOrgUnitIDs(ctx, tx2, []uint32{testUnitB}))
	require.NoError(t, tx2.Commit())

	rowCount, err := repo.entClient.Client().MembershipOrgUnit.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, rowCount, "按单元清理后 sys_membership_org_units 计数应归零")
}

// TestMembershipOrgUnitRepoSqlite_ExcludeExpired 验证 ListOrgUnitIDs/ListMembershipIDs 的
// excludeExpired 语义：过期（end_at 早于当前时刻）的关联在过滤后被排除。
func TestMembershipOrgUnitRepoSqlite_ExcludeExpired(t *testing.T) {
	repo := newMembershipOrgUnitRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const (
		testMembershipID = uint32(8203)
		testUnitOK       = uint32(9205)
		testUnitExp      = uint32(9206)
	)

	past := timestamppb.New(time.Now().Add(-1 * time.Hour))

	tx, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.AssignMembershipOrgUnits(ctx, tx, testMembershipID, []*identityV1.MembershipOrgUnit{
		// 同上：is_primary 取不同值以满足唯一索引。
		{MembershipId: trans.Ptr(testMembershipID), OrgUnitId: trans.Ptr(testUnitOK), IsPrimary: trans.Ptr(false), Status: trans.Ptr(identityV1.MembershipOrgUnit_ACTIVE)},
		{MembershipId: trans.Ptr(testMembershipID), OrgUnitId: trans.Ptr(testUnitExp), IsPrimary: trans.Ptr(true), Status: trans.Ptr(identityV1.MembershipOrgUnit_ACTIVE), EndAt: past},
	}))
	require.NoError(t, tx.Commit())

	// 不过滤：两条都可见
	all, err := repo.ListOrgUnitIDs(ctx, testMembershipID, false)
	require.NoError(t, err)
	require.Len(t, all, 2, "不过滤时应返回全部 2 条关联")

	// 过滤过期：仅剩未过期那条
	active, err := repo.ListOrgUnitIDs(ctx, testMembershipID, true)
	require.NoError(t, err)
	require.Equal(t, []uint32{testUnitOK}, active, "过滤过期后应仅返回未过期关联")

	// 反向过滤同理
	activeMemberships, err := repo.ListMembershipIDs(ctx, testUnitExp, true)
	require.NoError(t, err)
	require.Empty(t, activeMemberships, "过期单元在过滤后的反向查询中不应出现")

	// 收尾清理
	tx2, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback() }()
	require.NoError(t, repo.CleanRelationsByMembershipIDs(ctx, tx2, []uint32{testMembershipID}))
	require.NoError(t, tx2.Commit())

	rowCount, err := repo.entClient.Client().MembershipOrgUnit.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, rowCount, "清理后 sys_membership_org_units 计数应归零")
}
