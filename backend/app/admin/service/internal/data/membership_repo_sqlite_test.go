package data

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	entMembership "go-wind-admin/app/admin/service/internal/data/ent/membership"
	entMembershipOrgUnit "go-wind-admin/app/admin/service/internal/data/ent/membershiporgunit"
	entMembershipPosition "go-wind-admin/app/admin/service/internal/data/ent/membershipposition"
	entMembershipRole "go-wind-admin/app/admin/service/internal/data/ent/membershiprole"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newMembershipRepoSqlite 用 enttest helper 构造一个可直接做 CRUD 的 MembershipRepo。
// 白盒构造逐字段复刻 NewMembershipRepo 的初始化；其三个依赖仓库在生产构造器中
// 由 DI 注入，这里在同一 entclient 上内联构造（各自逐字段复刻生产构造器），
// 保证全部共享同一个 SQLite 内存库。
func newMembershipRepoSqlite(t *testing.T) *MembershipRepo {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)

	// 内联依赖 1/3：MembershipRoleRepo（复刻 NewMembershipRoleRepo）
	membershipRoleRepo := &MembershipRoleRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		statusConverter: mapper.NewEnumTypeConverter[permissionV1.MembershipRole_Status, entMembershipRole.Status](
			permissionV1.MembershipRole_Status_name,
			permissionV1.MembershipRole_Status_value,
		),
	}

	// 内联依赖 2/3：MembershipPositionRepo（复刻 NewMembershipPositionRepo）
	membershipPositionRepo := &MembershipPositionRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		statusConverter: mapper.NewEnumTypeConverter[identityV1.MembershipPosition_Status, entMembershipPosition.Status](
			identityV1.MembershipPosition_Status_name,
			identityV1.MembershipPosition_Status_value,
		),
	}

	// 内联依赖 3/3：MembershipOrgUnitRepo（复刻 NewMembershipOrgUnitRepo）
	membershipOrgUnitRepo := &MembershipOrgUnitRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		statusConverter: mapper.NewEnumTypeConverter[identityV1.MembershipOrgUnit_Status, entMembershipOrgUnit.Status](
			identityV1.MembershipOrgUnit_Status_name,
			identityV1.MembershipOrgUnit_Status_value,
		),
	}

	repo := &MembershipRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[identityV1.Membership, ent.Membership](),
		statusConverter: mapper.NewEnumTypeConverter[identityV1.Membership_Status, entMembership.Status](
			identityV1.Membership_Status_name,
			identityV1.Membership_Status_value,
		),
		membershipRoleRepo:     membershipRoleRepo,
		membershipPositionRepo: membershipPositionRepo,
		membershipOrgUnitRepo:  membershipOrgUnitRepo,
	}

	repo.init()

	return repo
}

// TestMembershipRepoSqlite_AssignAndRelationPropagation 覆盖
// AssignTenantMembershipWith 的完整写入链路：membership 行落库（含状态枚举转换）
// 与三张关联表（角色/单元/岗位）的联动写入，随后经各 List 查询路径交叉验证。
func TestMembershipRepoSqlite_AssignAndRelationPropagation(t *testing.T) {
	repo := newMembershipRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const (
		testUserID = uint32(17001)
		testRoleA  = uint32(18001)
		testRoleB  = uint32(18002)
		testUnitA  = uint32(19001)
		testPosA   = uint32(20001)
	)

	require.NoError(t, repo.AssignTenantMembershipWith(ctx, &identityV1.Membership{
		UserId:      trans.Ptr(testUserID),
		RoleIds:     []uint32{testRoleA, testRoleB},
		OrgUnitIds:  []uint32{testUnitA},
		PositionIds: []uint32{testPosA},
		Status:      trans.Ptr(identityV1.Membership_ACTIVE),
	}), "分配租户成员关系（含角色/单元/岗位关联）应成功")

	// membership 主表：1 行，user_id 与状态枚举按载荷落库
	membershipRows, err := repo.entClient.Client().Membership.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, membershipRows, 1, "sys_memberships 应有 1 条记录")
	membershipID := uint32(membershipRows[0].ID)
	require.Equal(t, testUserID, *membershipRows[0].UserID, "user_id 应按载荷落库")
	require.NotNil(t, membershipRows[0].Status, "status 应经转换器落库")
	require.Equal(t, entMembership.StatusActive, *membershipRows[0].Status, "proto Membership_ACTIVE 应映射为 ent StatusActive")

	// 三张关联表：角色 2 行、单元 1 行、岗位 1 行
	roleLinkCount, err := repo.entClient.Client().MembershipRole.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, roleLinkCount, "sys_membership_roles 应联动写入 2 条")
	unitLinkCount, err := repo.entClient.Client().MembershipOrgUnit.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, unitLinkCount, "sys_membership_org_units 应联动写入 1 条")
	posLinkCount, err := repo.entClient.Client().MembershipPosition.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, posLinkCount, "sys_membership_positions 应联动写入 1 条")

	// 交叉验证：按关联键回查用户 ID（经 membershipRepo 的聚合查询路径）
	usersByRole, err := repo.ListUserIDsByRoleID(ctx, testRoleA, false)
	require.NoError(t, err)
	require.Equal(t, []uint32{testUserID}, usersByRole, "按角色 A 回查应命中该成员的用户 ID")
	// 聚合路径按 membership 行取 user_id：两条角色关联都指向同一条 membership，
	// 最终按 membership 行去重后只返回一份该用户的 ID。
	usersByRoles, err := repo.ListUserIDsByRoleIDs(ctx, []uint32{testRoleA, testRoleB}, false)
	require.NoError(t, err)
	require.Equal(t, []uint32{testUserID}, usersByRoles, "按角色列表聚合回查应返回关联用户 ID（按 membership 行去重）")
	usersByUnit, err := repo.ListUserIDsByOrgUnitID(ctx, testUnitA, false)
	require.NoError(t, err)
	require.Equal(t, []uint32{testUserID}, usersByUnit, "按单元回查应命中该成员的用户 ID")
	usersByPosition, err := repo.ListUserIDsByPositionID(ctx, testPosA, false)
	require.NoError(t, err)
	require.Equal(t, []uint32{testUserID}, usersByPosition, "按岗位回查应命中该成员的用户 ID")

	// 按 membershipID 的直查路径
	usersByMembership, err := repo.ListUserIDs(ctx, membershipID, false)
	require.NoError(t, err)
	require.Equal(t, []uint32{testUserID}, usersByMembership, "按 membershipID 查询应返回关联用户 ID")
	usersByMemberships, err := repo.ListUserIDsByMembershipIDs(ctx, []uint32{membershipID}, false)
	require.NoError(t, err)
	require.Equal(t, []uint32{testUserID}, usersByMemberships, "按 membershipID 列表查询应返回关联用户 ID")

	// 非成员查询路径：按不存在的关联键查询应返回空
	noneUsers, err := repo.ListUserIDsByRoleID(ctx, 99999, false)
	require.NoError(t, err)
	require.Empty(t, noneUsers, "按不存在的角色查询应返回空")
}

// TestMembershipRepoSqlite_GetMembershipByUserTenantAndActive 验证按用户+租户
// 的单条查询与活跃成员列表查询。
func TestMembershipRepoSqlite_GetMembershipByUserTenantAndActive(t *testing.T) {
	repo := newMembershipRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const testUserID = uint32(17002)

	require.NoError(t, repo.AssignTenantMembershipWith(ctx, &identityV1.Membership{
		UserId: trans.Ptr(testUserID),
		Status: trans.Ptr(identityV1.Membership_ACTIVE),
	}), "写入一条无关联的成员关系应成功")

	rows, err := repo.entClient.Client().Membership.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	membershipID := uint32(rows[0].ID)

	// 命中：按用户 + 租户（0）
	got, err := repo.GetMembershipByUserTenant(ctx, testUserID, 0)
	require.NoError(t, err, "按用户+租户查询已存在成员关系应命中")
	require.Equal(t, testUserID, got.GetUserId(), "命中记录的 user_id 应与写入一致")
	require.Equal(t, membershipID, got.GetId(), "命中记录的主键应与写入时一致")
	// 读视图：status 经 backfillEnumsFrom 回填如实呈现（写入为 ACTIVE）。
	require.Equal(t, identityV1.Membership_ACTIVE, got.GetStatus(), "命中记录的读视图应回填 status")

	// 活跃成员列表：该用户一条
	actives, err := repo.GetUserActiveMemberships(ctx, testUserID)
	require.NoError(t, err)
	require.Len(t, actives, 1, "无失效时间的成员关系应计入活跃列表")
	require.Equal(t, testUserID, actives[0].GetUserId(), "活跃列表中的记录应为该用户")
	// 读视图：列表路径的 status 同样经回填如实呈现（写入为 ACTIVE）。
	require.Equal(t, identityV1.Membership_ACTIVE, actives[0].GetStatus(), "活跃列表的读视图应回填 status")

	// 未命中：不存在的用户
	_, err = repo.GetMembershipByUserTenant(ctx, 99999, 0)
	require.Error(t, err, "查询不存在的用户成员关系应返回错误")
	noneActives, err := repo.GetUserActiveMemberships(ctx, 99999)
	require.NoError(t, err)
	require.Empty(t, noneActives, "不存在用户的活跃列表应为空")
}

// TestMembershipRepoSqlite_UpsertIdempotent 验证 AssignTenantMembershipWith 的
// upsert 语义：同一 (tenant_id, user_id) 重复分配只更新既有行而不新增。
func TestMembershipRepoSqlite_UpsertIdempotent(t *testing.T) {
	repo := newMembershipRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const testUserID = uint32(17003)

	require.NoError(t, repo.AssignTenantMembershipWith(ctx, &identityV1.Membership{
		UserId: trans.Ptr(testUserID),
		Status: trans.Ptr(identityV1.Membership_ACTIVE),
	}), "首次分配应成功")
	require.NoError(t, repo.AssignTenantMembershipWith(ctx, &identityV1.Membership{
		UserId: trans.Ptr(testUserID),
		Status: trans.Ptr(identityV1.Membership_ACTIVE),
	}), "重复分配应按 upsert 语义成功")

	count, err := repo.entClient.Client().Membership.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count, "重复分配同一 (tenant, user) 不应新增 membership 行")
}

// TestMembershipRepoSqlite_SetUserFields 验证 Membership 表单值冗余字段与
// 状态/失效时间的按用户更新路径。
func TestMembershipRepoSqlite_SetUserFields(t *testing.T) {
	repo := newMembershipRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const testUserID = uint32(17004)

	require.NoError(t, repo.AssignTenantMembershipWith(ctx, &identityV1.Membership{
		UserId: trans.Ptr(testUserID),
		Status: trans.Ptr(identityV1.Membership_ACTIVE),
	}), "写入初始成员关系应成功")

	// 单值冗余字段更新
	require.NoError(t, repo.SetUserRoleID(ctx, testUserID, 42), "更新 role_id 应成功")
	require.NoError(t, repo.SetUserPositionID(ctx, testUserID, 43), "更新 position_id 应成功")
	require.NoError(t, repo.SetUserOrgUnitID(ctx, testUserID, 44), "更新 org_unit_id 应成功")

	// 状态更新（ACTIVE → DISABLED，经转换器）
	require.NoError(t, repo.SetUserStatus(ctx, testUserID, trans.Ptr(identityV1.Membership_DISABLED)),
		"更新状态应成功")

	// 失效时间更新
	endAt := time.Now().Add(24 * time.Hour)
	require.NoError(t, repo.SetUserEndAt(ctx, testUserID, &endAt), "更新失效时间应成功")

	row, err := repo.entClient.Client().Membership.Query().Only(ctx)
	require.NoError(t, err)
	require.NotNil(t, row.RoleID, "role_id 应已被更新")
	require.Equal(t, uint32(42), *row.RoleID, "role_id 应更新为 42")
	require.NotNil(t, row.PositionID, "position_id 应已被更新")
	require.Equal(t, uint32(43), *row.PositionID, "position_id 应更新为 43")
	require.NotNil(t, row.OrgUnitID, "org_unit_id 应已被更新")
	require.Equal(t, uint32(44), *row.OrgUnitID, "org_unit_id 应更新为 44")
	require.NotNil(t, row.Status, "status 应已被更新")
	require.Equal(t, entMembership.StatusDisabled, *row.Status, "proto Membership_DISABLED 应映射为 ent StatusDisabled")

	// 读视图：更新为 DISABLED 后，GetMembershipByUserTenant 的读视图应经回填
	// 如实呈现 DISABLED（而非零值缺省）。
	updated, err := repo.GetMembershipByUserTenant(ctx, testUserID, 0)
	require.NoError(t, err, "更新后的成员关系应仍可按用户+租户命中")
	require.Equal(t, identityV1.Membership_DISABLED, updated.GetStatus(), "更新后的读视图应回填 status 为 DISABLED")
	require.NotNil(t, row.EndAt, "end_at 应已被更新")
	require.True(t, row.EndAt.After(time.Now().Add(23*time.Hour)), "end_at 应更新为 24 小时后的时刻")

	// 清除路径：把冗余字段清空
	require.NoError(t, repo.SetUserRoleID(ctx, testUserID, 0), "清空 role_id 应成功")
	row, err = repo.entClient.Client().Membership.Query().Only(ctx)
	require.NoError(t, err)
	require.Nil(t, row.RoleID, "role_id 应已被清空")
}

// TestMembershipRepoSqlite_TenantScopeGuard 验证依赖 queryMembershipID 的
// 各查询/分配入口在平台上下文（System viewer，无租户范围）下的闸门行为。
func TestMembershipRepoSqlite_TenantScopeGuard(t *testing.T) {
	repo := newMembershipRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const testUserID = uint32(17005)

	// GetMembershipIDByUserID：需要租户范围
	_, err := repo.GetMembershipIDByUserID(ctx, testUserID)
	require.Error(t, err, "平台上下文按用户查 membershipID 应被拒绝")

	// GetMembershipID：需要租户范围
	_, err = repo.GetMembershipID(ctx, testUserID)
	require.Error(t, err, "平台上下文 GetMembershipID 应被拒绝")

	// Assign/Query 关联入口：均经 queryMembershipID，同样被拒
	require.Error(t, repo.AssignMembershipRoles(ctx, testUserID, nil),
		"平台上下文 AssignMembershipRoles 应被拒绝")
	require.Error(t, repo.AssignMembershipPositions(ctx, testUserID, nil),
		"平台上下文 AssignMembershipPositions 应被拒绝")
	require.Error(t, repo.AssignMembershipOrgUnits(ctx, testUserID, nil),
		"平台上下文 AssignMembershipOrgUnits 应被拒绝")
	_, err = repo.ListMembershipRoleIDs(ctx, testUserID)
	require.Error(t, err, "平台上下文 ListMembershipRoleIDs 应被拒绝")
	_, err = repo.ListMembershipOrgUnitIDs(ctx, testUserID)
	require.Error(t, err, "平台上下文 ListMembershipOrgUnitIDs 应被拒绝")
	_, err = repo.ListMembershipPositionIDs(ctx, testUserID)
	require.Error(t, err, "平台上下文 ListMembershipPositionIDs 应被拒绝")
	_, _, _, err = repo.ListMembershipRelationIDs(ctx, testUserID)
	require.Error(t, err, "平台上下文 ListMembershipRelationIDs 应被拒绝")

	// CleanRelationsByUserID：同样经 queryMembershipID 被拒
	tx, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	require.Error(t, repo.CleanRelationsByUserID(ctx, tx, testUserID),
		"平台上下文 CleanRelationsByUserID 应被拒绝")
	_ = tx.Rollback()
}
