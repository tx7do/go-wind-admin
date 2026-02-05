package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/sliceutil"
	"github.com/tx7do/go-utils/timeutil"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/membership"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

type MembershipRepo struct {
	log *log.Helper

	entClient       *entCrud.EntClient[*ent.Client]
	mapper          *mapper.CopierMapper[identityV1.Membership, ent.Membership]
	statusConverter *mapper.EnumTypeConverter[identityV1.Membership_Status, membership.Status]

	membershipRoleRepo     *MembershipRoleRepo
	membershipPositionRepo *MembershipPositionRepo
	membershipOrgUnitRepo  *MembershipOrgUnitRepo
}

func NewMembershipRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
	membershipRoleRepo *MembershipRoleRepo,
	membershipPositionRepo *MembershipPositionRepo,
	membershipOrgUnitRepo *MembershipOrgUnitRepo,
) *MembershipRepo {
	repo := &MembershipRepo{
		log:       ctx.NewLoggerHelper("membership/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[identityV1.Membership, ent.Membership](),
		statusConverter: mapper.NewEnumTypeConverter[identityV1.Membership_Status, membership.Status](
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

func (r *MembershipRepo) init() {
	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
}

func (r *MembershipRepo) AssignTenantMembershipWith(ctx context.Context, data *identityV1.Membership) (err error) {
	var tx *ent.Tx
	tx, err = r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf("start transaction failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("start transaction failed")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.log.Errorf("transaction rollback failed: %s", rollbackErr.Error())
			}
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			r.log.Errorf("transaction commit failed: %s", commitErr.Error())
			err = identityV1.ErrorInternalServerError("transaction commit failed")
		}
	}()

	return r.AssignTenantMembershipWithTx(ctx, tx, data)
}

// AssignTenantMembershipWithTx 使用 Membership 数据为用户分配租户
func (r *MembershipRepo) AssignTenantMembershipWithTx(ctx context.Context, tx *ent.Tx, data *identityV1.Membership) (err error) {
	var entity *ent.Membership
	entity, err = r.upsertMembership(ctx, tx, data)
	if err != nil {
		return err
	}

	var roleIDs []uint32
	if data.RoleId != nil {
		roleIDs = append(roleIDs, data.GetRoleId())
	}
	if len(data.RoleIds) > 0 {
		roleIDs = append(roleIDs, data.RoleIds...)
	}
	roleIDs = sliceutil.Unique(roleIDs)

	var orgUnitIDs []uint32
	if data.OrgUnitId != nil {
		orgUnitIDs = append(orgUnitIDs, data.GetOrgUnitId())
	}
	if len(data.OrgUnitIds) > 0 {
		orgUnitIDs = append(orgUnitIDs, data.OrgUnitIds...)
	}
	orgUnitIDs = sliceutil.Unique(orgUnitIDs)

	var positionIDs []uint32
	if data.PositionId != nil {
		positionIDs = append(positionIDs, data.GetPositionId())
	}
	if len(data.PositionIds) > 0 {
		positionIDs = append(positionIDs, data.PositionIds...)
	}
	positionIDs = sliceutil.Unique(positionIDs)

	if len(roleIDs) > 0 {
		var roles []*permissionV1.MembershipRole
		for _, roleID := range roleIDs {
			role := &permissionV1.MembershipRole{
				MembershipId: trans.Ptr(entity.ID),
				TenantId:     data.TenantId,
				RoleId:       trans.Ptr(roleID),
				Status:       trans.Ptr(permissionV1.MembershipRole_ACTIVE),
				CreatedBy:    data.CreatedBy,
				AssignedBy:   data.AssignedBy,
				AssignedAt:   data.AssignedAt,
				StartAt:      data.StartAt,
				EndAt:        data.EndAt,
				//IsPrimary:    data.IsPrimary,
			}
			roles = append(roles, role)
		}

		if err = r.membershipRoleRepo.AssignMembershipRoles(ctx, tx, entity.ID, roles); err != nil {
			return err
		}
	}

	if len(orgUnitIDs) > 0 {
		var orgUnits []*identityV1.MembershipOrgUnit
		for _, orgUnitID := range orgUnitIDs {
			orgUnit := &identityV1.MembershipOrgUnit{
				MembershipId: trans.Ptr(entity.ID),
				TenantId:     data.TenantId,
				OrgUnitId:    trans.Ptr(orgUnitID),
				Status:       trans.Ptr(identityV1.MembershipOrgUnit_ACTIVE),
				CreatedBy:    data.CreatedBy,
				AssignedBy:   data.AssignedBy,
				AssignedAt:   data.AssignedAt,
				StartAt:      data.StartAt,
				EndAt:        data.EndAt,
				//IsPrimary:    data.IsPrimary,
			}
			orgUnits = append(orgUnits, orgUnit)
		}

		if err = r.membershipOrgUnitRepo.AssignMembershipOrgUnits(ctx, tx, entity.ID, orgUnits); err != nil {
			return err
		}
	}

	if len(positionIDs) > 0 {
		var positions []*identityV1.MembershipPosition
		for _, positionID := range positionIDs {
			position := &identityV1.MembershipPosition{
				MembershipId: trans.Ptr(entity.ID),
				TenantId:     data.TenantId,
				PositionId:   trans.Ptr(positionID),
				Status:       trans.Ptr(identityV1.MembershipPosition_ACTIVE),
				CreatedBy:    data.CreatedBy,
				AssignedBy:   data.AssignedBy,
				AssignedAt:   data.AssignedAt,
				StartAt:      data.StartAt,
				EndAt:        data.EndAt,
				//IsPrimary:    data.IsPrimary,
			}
			positions = append(positions, position)
		}

		if err = r.membershipPositionRepo.AssignMembershipPositions(ctx, tx, entity.ID, positions); err != nil {
			return err
		}
	}

	return nil
}

func (r *MembershipRepo) AssignMembershipRoles(ctx context.Context,
	userID uint32,
	datas []*permissionV1.MembershipRole,
) (err error) {
	var tx *ent.Tx
	tx, err = r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf("start transaction failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("start transaction failed")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.log.Errorf("transaction rollback failed: %s", rollbackErr.Error())
			}
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			r.log.Errorf("transaction commit failed: %s", commitErr.Error())
			err = identityV1.ErrorInternalServerError("transaction commit failed")
		}
	}()

	var membershipID uint32
	membershipID, err = r.queryMembershipID(ctx, tx, userID)
	if err != nil {
		r.log.Errorf("get membership id failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("get membership id failed")
	}

	if err = r.membershipRoleRepo.AssignMembershipRoles(ctx, tx, membershipID, datas); err != nil {
		return err
	}

	return nil
}

// AssignMembershipPositions 分配职位给用户
func (r *MembershipRepo) AssignMembershipPositions(
	ctx context.Context,
	userID uint32,
	datas []*identityV1.MembershipPosition,
) (err error) {
	var tx *ent.Tx
	tx, err = r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf("start transaction failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("start transaction failed")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.log.Errorf("transaction rollback failed: %s", rollbackErr.Error())
			}
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			r.log.Errorf("transaction commit failed: %s", commitErr.Error())
			err = identityV1.ErrorInternalServerError("transaction commit failed")
		}
	}()

	var membershipID uint32
	membershipID, err = r.queryMembershipID(ctx, tx, userID)
	if err != nil {
		r.log.Errorf("get membership id failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("get membership id failed")
	}

	if err = r.membershipPositionRepo.AssignMembershipPositions(ctx, tx, membershipID, datas); err != nil {
		return err
	}

	return nil
}

// AssignMembershipOrgUnits 分配组织单元给用户
func (r *MembershipRepo) AssignMembershipOrgUnits(ctx context.Context,
	userID uint32,
	datas []*identityV1.MembershipOrgUnit,
) (err error) {
	var tx *ent.Tx
	tx, err = r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf("start transaction failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("start transaction failed")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.log.Errorf("transaction rollback failed: %s", rollbackErr.Error())
			}
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			r.log.Errorf("transaction commit failed: %s", commitErr.Error())
			err = identityV1.ErrorInternalServerError("transaction commit failed")
		}
	}()

	var membershipID uint32
	membershipID, err = r.queryMembershipID(ctx, tx, userID)
	if err != nil {
		r.log.Errorf("get membership id failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("get membership id failed")
	}

	if err = r.membershipOrgUnitRepo.AssignMembershipOrgUnits(ctx, tx, membershipID, datas); err != nil {
		return err
	}

	return nil
}

// SetUserOrgUnitID 设置用户的组织单元 ID
// 如果只是单一一个组织，可以直接在 Membership 表中设置 org_unit_id 字段。
func (r *MembershipRepo) SetUserOrgUnitID(ctx context.Context, userID uint32, orgUnitID uint32) error {
	up := r.entClient.Client().Membership.
		Update().
		Where(
			membership.UserIDEQ(userID),
		)

	if orgUnitID == 0 {
		if _, err := up.ClearOrgUnitID().Save(ctx); err != nil {
			r.log.Errorf("update membership org_unit_id failed: %s", err.Error())
			return identityV1.ErrorInternalServerError("update membership org_unit_id failed")
		}
		return nil
	}

	if _, err := up.SetOrgUnitID(orgUnitID).Save(ctx); err != nil {
		r.log.Errorf("update membership org_unit_id failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("update membership org_unit_id failed")
	}
	return nil
}

// SetUserRoleID 设置用户的角色 ID
// 如果只是单一一个角色，可以直接在 Membership 表中设置 role_id 字段。
func (r *MembershipRepo) SetUserRoleID(ctx context.Context, userID uint32, roleID uint32) error {
	up := r.entClient.Client().Membership.
		Update().
		Where(
			membership.UserIDEQ(userID),
		)

	if roleID == 0 {
		if _, err := up.ClearRoleID().Save(ctx); err != nil {
			r.log.Errorf("update membership role_id failed: %s", err.Error())
			return identityV1.ErrorInternalServerError("update membership role_id failed")
		}
		return nil
	}

	if _, err := up.SetRoleID(roleID).Save(ctx); err != nil {
		r.log.Errorf("update membership role_id failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("update membership role_id failed")
	}
	return nil
}

// SetUserPositionID 设置用户的职位 ID
// 如果只是单一个职位，可以直接在 Membership 表中设置 position_id 字段。
func (r *MembershipRepo) SetUserPositionID(ctx context.Context, userID uint32, positionID uint32) error {
	up := r.entClient.Client().Membership.
		Update().
		Where(
			membership.UserIDEQ(userID),
		)

	if positionID == 0 {
		if _, err := up.ClearPositionID().Save(ctx); err != nil {
			r.log.Errorf("update membership position_id failed: %s", err.Error())
			return identityV1.ErrorInternalServerError("update membership position_id failed")
		}
		return nil
	}

	if _, err := up.SetPositionID(positionID).Save(ctx); err != nil {
		r.log.Errorf("update membership position_id failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("update membership position_id failed")
	}
	return nil
}

// SetUserStatus 设置用户的状态
func (r *MembershipRepo) SetUserStatus(ctx context.Context, userID uint32, status *identityV1.Membership_Status) error {
	up := r.entClient.Client().Membership.
		Update().
		Where(
			membership.UserIDEQ(userID),
		)

	if status == nil {
		if _, err := up.ClearStatus().Save(ctx); err != nil {
			r.log.Errorf("update membership status failed: %s", err.Error())
			return identityV1.ErrorInternalServerError("update membership status failed")
		}
		return nil
	}

	if _, err := up.SetStatus(*r.statusConverter.ToEntity(status)).Save(ctx); err != nil {
		r.log.Errorf("update membership status failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("update membership status failed")
	}
	return nil
}

// SetUserEndAt 设置用户的结束时间
func (r *MembershipRepo) SetUserEndAt(ctx context.Context, userID uint32, endAt *time.Time) error {
	up := r.entClient.Client().Membership.
		Update().
		Where(
			membership.UserIDEQ(userID),
		)

	if endAt == nil {
		if _, err := up.ClearEndAt().Save(ctx); err != nil {
			r.log.Errorf("update membership end_at failed: %s", err.Error())
			return identityV1.ErrorInternalServerError("update membership end_at failed")
		}
		return nil
	}

	if _, err := up.SetEndAt(*endAt).Save(ctx); err != nil {
		r.log.Errorf("update membership end_at failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("update membership end_at failed")
	}
	return nil
}

// GetMembershipID 获取 Membership ID
func (r *MembershipRepo) GetMembershipID(ctx context.Context, userID uint32) (membershipID uint32, err error) {
	var tx *ent.Tx
	tx, err = r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf("start transaction failed: %s", err.Error())
		return uint32(0), identityV1.ErrorInternalServerError("start transaction failed")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.log.Errorf("transaction rollback failed: %s", rollbackErr.Error())
			}
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			r.log.Errorf("transaction commit failed: %s", commitErr.Error())
			err = identityV1.ErrorInternalServerError("transaction commit failed")
		}
	}()

	return r.queryMembershipID(ctx, tx, userID)
}

// GetMembershipByUserTenant 获取用户在租户下的 Membership 记录
func (r *MembershipRepo) GetMembershipByUserTenant(ctx context.Context, userID uint32) (*identityV1.Membership, error) {
	now := time.Now()
	builder := r.entClient.Client().Membership.Query()
	builder.Where(
		membership.UserIDEQ(userID),
		membership.Or(
			membership.EndAtIsNil(),
			membership.EndAtGT(now),
		),
	)

	entity, err := builder.Only(ctx)
	if err != nil {
		r.log.Errorf("get membership failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("get membership failed")
	}

	dto := r.mapper.ToDTO(entity)

	return dto, nil
}

// GetUserActiveMemberships 获取用户所有有效的 Membership 列表
func (r *MembershipRepo) GetUserActiveMemberships(ctx context.Context, userID uint32) ([]*identityV1.Membership, error) {
	now := time.Now()
	builder := r.entClient.Client().Membership.Query()
	builder.Where(
		membership.UserIDEQ(userID),
		membership.Or(
			membership.EndAtIsNil(),
			membership.EndAtGT(now),
		),
	)
	entities, err := builder.All(ctx)
	if err != nil {
		r.log.Errorf("get user active memberships failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("get user active memberships failed")
	}

	dtos := make([]*identityV1.Membership, 0, len(entities))
	for _, entity := range entities {
		dto := r.mapper.ToDTO(entity)
		dtos = append(dtos, dto)
	}

	return dtos, nil
}

// ListMembershipRoleIDs 获取 Membership 关联的角色 ID 列表
func (r *MembershipRepo) ListMembershipRoleIDs(ctx context.Context, userID uint32) (roleIDs []uint32, err error) {
	var tx *ent.Tx
	tx, err = r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf("start transaction failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("start transaction failed")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.log.Errorf("transaction rollback failed: %s", rollbackErr.Error())
			}
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			r.log.Errorf("transaction commit failed: %s", commitErr.Error())
			err = identityV1.ErrorInternalServerError("transaction commit failed")
		}
	}()

	var membershipID uint32
	membershipID, err = r.queryMembershipID(ctx, tx, userID)
	if err != nil {
		r.log.Errorf("get membership id failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("get membership id failed")
	}

	return r.membershipRoleRepo.ListRoleIDs(ctx, membershipID, false)
}

// GetRoleIDsByMembership 根据 Membership ID 获取关联的角色 ID 列表
func (r *MembershipRepo) GetRoleIDsByMembership(ctx context.Context, membershipID uint32) (roleIDs []uint32, err error) {
	return r.membershipRoleRepo.ListRoleIDs(ctx, membershipID, false)
}

// ListMembershipOrgUnitIDs 获取 Membership 关联的组织单元 ID 列表
func (r *MembershipRepo) ListMembershipOrgUnitIDs(ctx context.Context, userID uint32) (orgUnitIDs []uint32, err error) {
	var tx *ent.Tx
	tx, err = r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf("start transaction failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("start transaction failed")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.log.Errorf("transaction rollback failed: %s", rollbackErr.Error())
			}
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			r.log.Errorf("transaction commit failed: %s", commitErr.Error())
			err = identityV1.ErrorInternalServerError("transaction commit failed")
		}
	}()

	membershipID, err := r.queryMembershipID(ctx, tx, userID)
	if err != nil {
		r.log.Errorf("get membership id failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("get membership id failed")
	}

	return r.membershipOrgUnitRepo.ListOrgUnitIDs(ctx, membershipID, false)
}

// ListMembershipPositionIDs 获取 Membership 关联的职位 ID 列表
func (r *MembershipRepo) ListMembershipPositionIDs(ctx context.Context, userID uint32) (positionIDs []uint32, err error) {
	var tx *ent.Tx
	tx, err = r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf("start transaction failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("start transaction failed")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.log.Errorf("transaction rollback failed: %s", rollbackErr.Error())
			}
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			r.log.Errorf("transaction commit failed: %s", commitErr.Error())
			err = identityV1.ErrorInternalServerError("transaction commit failed")
		}
	}()

	membershipID, err := r.queryMembershipID(ctx, tx, userID)
	if err != nil {
		r.log.Errorf("get membership id failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("get membership id failed")
	}

	return r.membershipPositionRepo.ListPositionIDs(ctx, membershipID, false)
}

// GetMembershipIDByUserID 获取 Membership ID By User ID
func (r *MembershipRepo) GetMembershipIDByUserID(ctx context.Context, userID uint32) (membershipID uint32, err error) {
	now := time.Now()

	builder := r.entClient.Client().Membership.Query()
	builder.Where(
		membership.UserIDEQ(userID),
		membership.Or(
			membership.EndAtIsNil(),
			membership.EndAtGT(now),
		),
	)
	builder.Select(
		membership.FieldID,
	)

	ms, err := builder.Only(ctx)
	if err != nil {
		r.log.Errorf("get membership failed: %s", err.Error())
		return 0, identityV1.ErrorInternalServerError("get membership failed")
	}
	if ms == nil {
		r.log.Errorf("membership not found for user %d", userID)
		return 0, identityV1.ErrorNotFound("membership not found")
	}

	membershipID = ms.ID
	return
}

// ListMembershipRelationIDs 获取 Membership 关联的角色、职位、组织单元 ID 列表
func (r *MembershipRepo) ListMembershipRelationIDs(ctx context.Context, userID uint32) (roleIDs []uint32, positionIDs []uint32, orgUnitIDs []uint32, err error) {
	// 获取 Membership ID
	var membershipID uint32
	if membershipID, err = r.GetMembershipIDByUserID(ctx, userID); err != nil {
		return nil, nil, nil, err
	}

	// 获取关联的角色、职位、组织单元 ID 列表
	roleIDs, err = r.membershipRoleRepo.ListRoleIDs(ctx, membershipID, false)
	if err != nil {
		return nil, nil, nil, err
	}

	// 获取关联的组织单元 ID 列表
	orgUnitIDs, err = r.membershipOrgUnitRepo.ListOrgUnitIDs(ctx, membershipID, false)
	if err != nil {
		return nil, nil, nil, err
	}

	// 获取关联的职位 ID 列表
	positionIDs, err = r.membershipPositionRepo.ListPositionIDs(ctx, membershipID, false)
	if err != nil {
		return nil, nil, nil, err
	}

	return
}

// createMembership 创建 Membership 记录
func (r *MembershipRepo) createMembership(ctx context.Context, tx *ent.Tx, data *identityV1.Membership) (*ent.Membership, error) {
	return r.upsertMembership(ctx, tx, data)
}

// upsertMembership 更新或插入 Membership 记录
func (r *MembershipRepo) upsertMembership(ctx context.Context, tx *ent.Tx, data *identityV1.Membership) (*ent.Membership, error) {
	now := time.Now()

	if data.StartAt == nil {
		data.StartAt = timeutil.TimeToTimestamppb(&now)
	}

	builder := tx.Membership.Create()

	builder.
		SetNillableTenantID(data.TenantId).
		SetUserID(data.GetUserId()).
		SetNillableRoleID(data.RoleId).
		SetNillablePositionID(data.PositionId).
		SetNillableOrgUnitID(data.OrgUnitId).
		SetNillableStatus(r.statusConverter.ToEntity(data.Status)).
		SetNillableAssignedBy(data.AssignedBy).
		SetNillableAssignedAt(timeutil.TimestamppbToTime(data.AssignedAt)).
		SetNillableJoinedAt(timeutil.TimestamppbToTime(data.JoinedAt)).
		SetNillableIsPrimary(data.IsPrimary).
		SetNillableStartAt(timeutil.TimestamppbToTime(data.StartAt)).
		SetNillableEndAt(timeutil.TimestamppbToTime(data.EndAt)).
		SetNillableCreatedBy(data.CreatedBy).
		SetCreatedAt(now).
		OnConflictColumns(
			membership.FieldTenantID,
			membership.FieldUserID,
		).
		UpdateNewValues().
		SetUpdatedAt(now).
		SetUpdatedBy(data.GetUpdatedBy())

	if entity, err := builder.Save(ctx); err != nil {
		r.log.Errorf("upsert membership failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("upsert membership failed")
	} else {
		return entity, err
	}
}

// queryMembershipID 查询 Membership ID
func (r *MembershipRepo) queryMembershipID(
	ctx context.Context, tx *ent.Tx,
	userID uint32,
) (uint32, error) {
	now := time.Now()
	membershipID, err := tx.Membership.Query().
		Where(
			membership.UserIDEQ(userID),
			membership.Or(
				membership.EndAtIsNil(),
				membership.EndAtGT(now),
			),
		).
		OnlyID(ctx)
	if err != nil {
		r.log.Errorf("get membership id failed: %s", err.Error())
		return 0, identityV1.ErrorInternalServerError("get membership id failed")
	}
	return membershipID, nil
}

// ListUserIDs 获取 Membership ID 关联的用户 ID 列表
func (r *MembershipRepo) ListUserIDs(ctx context.Context, membershipID uint32, excludeExpired bool) ([]uint32, error) {
	q := r.entClient.Client().Membership.Query().
		Where(
			membership.IDEQ(membershipID),
		)

	if excludeExpired {
		now := time.Now()
		q = q.Where(
			membership.Or(
				membership.EndAtIsNil(),
				membership.EndAtGT(now),
			),
		)
	}

	intIDs, err := q.
		Select(membership.FieldUserID).
		Ints(ctx)
	if err != nil {
		r.log.Errorf("query user ids by membership id failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query user ids by membership id failed")
	}
	ids := make([]uint32, len(intIDs))
	for i, v := range intIDs {
		ids[i] = uint32(v)
	}
	return ids, nil

}

// ListUserIDsByMembershipIDs 获取 Membership ID 列表关联的用户 ID 列表
func (r *MembershipRepo) ListUserIDsByMembershipIDs(ctx context.Context, membershipIDs []uint32, excludeExpired bool) ([]uint32, error) {
	q := r.entClient.Client().Membership.Query().
		Where(
			membership.IDIn(membershipIDs...),
		)
	if excludeExpired {
		now := time.Now()
		q = q.Where(
			membership.Or(
				membership.EndAtIsNil(),
				membership.EndAtGT(now),
			),
		)
	}

	intIDs, err := q.
		Select(membership.FieldUserID).
		Ints(ctx)
	if err != nil {
		r.log.Errorf("query user ids by membership ids failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query user ids by membership ids failed")
	}
	ids := make([]uint32, len(intIDs))
	for i, v := range intIDs {
		ids[i] = uint32(v)
	}
	return ids, nil
}

// ListUserIDsByOrgUnitID 获取组织单元关联的用户 ID 列表
func (r *MembershipRepo) ListUserIDsByOrgUnitID(ctx context.Context, orgUnitID uint32, excludeExpired bool) ([]uint32, error) {
	membershipIDs, err := r.membershipOrgUnitRepo.ListMembershipIDs(ctx, orgUnitID, excludeExpired)
	if err != nil {
		return nil, err
	}

	return r.ListUserIDsByMembershipIDs(ctx, membershipIDs, excludeExpired)
}

// ListUserIDsByOrgUnitIDs 获取多个组织单元关联的用户 ID 列表
func (r *MembershipRepo) ListUserIDsByOrgUnitIDs(ctx context.Context, orgUnitIDs []uint32, excludeExpired bool) ([]uint32, error) {
	membershipIDs, err := r.membershipOrgUnitRepo.ListMembershipIDsByOrgUnitIDs(ctx, orgUnitIDs, excludeExpired)
	if err != nil {
		return nil, err
	}

	return r.ListUserIDsByMembershipIDs(ctx, membershipIDs, excludeExpired)
}

// ListUserIDsByPositionID 获取职位关联的用户 ID 列表
func (r *MembershipRepo) ListUserIDsByPositionID(ctx context.Context, positionID uint32, excludeExpired bool) ([]uint32, error) {
	membershipIDs, err := r.membershipPositionRepo.ListMembershipIDs(ctx, positionID, excludeExpired)
	if err != nil {
		return nil, err
	}

	return r.ListUserIDsByMembershipIDs(ctx, membershipIDs, excludeExpired)
}

// ListUserIDsByPositionIDs 获取多个职位关联的用户 ID 列表
func (r *MembershipRepo) ListUserIDsByPositionIDs(ctx context.Context, positionIDs []uint32, excludeExpired bool) ([]uint32, error) {
	membershipIDs, err := r.membershipPositionRepo.ListMembershipIDsByPositionIDs(ctx, positionIDs, excludeExpired)
	if err != nil {
		return nil, err
	}

	return r.ListUserIDsByMembershipIDs(ctx, membershipIDs, excludeExpired)
}

// ListUserIDsByRoleID 获取角色关联的用户 ID 列表
func (r *MembershipRepo) ListUserIDsByRoleID(ctx context.Context, roleID uint32, excludeExpired bool) ([]uint32, error) {
	membershipIDs, err := r.membershipRoleRepo.ListMembershipIDs(ctx, roleID, excludeExpired)
	if err != nil {
		return nil, err
	}

	return r.ListUserIDsByMembershipIDs(ctx, membershipIDs, excludeExpired)
}

// ListUserIDsByRoleIDs 获取多个角色关联的用户 ID 列表
func (r *MembershipRepo) ListUserIDsByRoleIDs(ctx context.Context, roleIDs []uint32, excludeExpired bool) ([]uint32, error) {
	membershipIDs, err := r.membershipRoleRepo.ListMembershipIDsByRoleIDs(ctx, roleIDs, excludeExpired)
	if err != nil {
		return nil, err
	}

	return r.ListUserIDsByMembershipIDs(ctx, membershipIDs, excludeExpired)
}

// CleanRelationsByUserID 清理用户关联的所有关系数据
func (r *MembershipRepo) CleanRelationsByUserID(ctx context.Context, tx *ent.Tx, userID uint32) (err error) {
	if userID == 0 {
		return nil
	}

	var membershipID uint32
	membershipID, err = r.queryMembershipID(ctx, tx, userID)
	if err != nil {
		return
	}

	if err = r.membershipRoleRepo.CleanRelationsByMembershipID(ctx, tx, membershipID); err != nil {
		r.log.Errorf("clean membership roles by membership id failed: %s", err.Error())
	}

	if err = r.membershipPositionRepo.CleanRelationsByMembershipID(ctx, tx, membershipID); err != nil {
		r.log.Errorf("clean membership positions by membership id failed: %s", err.Error())
	}

	if err = r.membershipOrgUnitRepo.CleanRelationsByMembershipID(ctx, tx, membershipID); err != nil {
		r.log.Errorf("clean membership org units by membership id failed: %s", err.Error())
	}

	if err = tx.Membership.DeleteOneID(membershipID).Exec(ctx); err != nil {
		r.log.Errorf("delete membership by id failed: %s", err.Error())
	}

	return
}
