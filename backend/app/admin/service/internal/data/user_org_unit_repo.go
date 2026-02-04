package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/timeutil"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/userorgunit"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
)

type UserOrgUnitRepo struct {
	log *log.Helper

	entClient       *entCrud.EntClient[*ent.Client]
	statusConverter *mapper.EnumTypeConverter[identityV1.UserOrgUnit_Status, userorgunit.Status]
}

func NewUserOrgUnitRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *UserOrgUnitRepo {
	return &UserOrgUnitRepo{
		log:       ctx.NewLoggerHelper("user-org-unit/repo/admin-service"),
		entClient: entClient,
		statusConverter: mapper.NewEnumTypeConverter[identityV1.UserOrgUnit_Status, userorgunit.Status](
			identityV1.UserOrgUnit_Status_name,
			identityV1.UserOrgUnit_Status_value,
		),
	}
}

// CleanRelationsByUserID 清理用户组织单元关联
func (r *UserOrgUnitRepo) CleanRelationsByUserID(ctx context.Context, tx *ent.Tx, userID uint32) error {
	if userID == 0 {
		return nil
	}

	if _, err := tx.UserOrgUnit.Delete().
		Where(
			userorgunit.UserIDEQ(userID),
		).
		Exec(ctx); err != nil {
		r.log.Errorf("delete old user orgUnits failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("delete old user orgUnits failed")
	}
	return nil
}

// CleanRelationsByUserIDs 清理多个用户组织单元关联
func (r *UserOrgUnitRepo) CleanRelationsByUserIDs(ctx context.Context, tx *ent.Tx, userIDs []uint32) error {
	if len(userIDs) == 0 {
		return nil
	}

	if _, err := tx.UserOrgUnit.Delete().
		Where(
			userorgunit.UserIDIn(userIDs...),
		).
		Exec(ctx); err != nil {
		r.log.Errorf("delete old user orgUnits by user ids failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("delete old user orgUnits by user ids failed")
	}
	return nil
}

// CleanRelationsByOrgUnitID 清理组织单元的用户关联
func (r *UserOrgUnitRepo) CleanRelationsByOrgUnitID(ctx context.Context, tx *ent.Tx, orgUnitID uint32) error {
	if orgUnitID == 0 {
		return nil
	}

	if _, err := tx.UserOrgUnit.Delete().
		Where(
			userorgunit.OrgUnitIDEQ(orgUnitID),
		).
		Exec(ctx); err != nil {
		r.log.Errorf("delete old user orgUnits by orgUnit id failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("delete old user orgUnits by orgUnit id failed")
	}
	return nil
}

// CleanRelationsByOrgUnitIDs 清理组织单元的用户关联
func (r *UserOrgUnitRepo) CleanRelationsByOrgUnitIDs(ctx context.Context, tx *ent.Tx, orgUnitIDs []uint32) error {
	if len(orgUnitIDs) == 0 {
		return nil
	}

	if _, err := tx.UserOrgUnit.Delete().
		Where(
			userorgunit.OrgUnitIDIn(orgUnitIDs...),
		).
		Exec(ctx); err != nil {
		r.log.Errorf("delete old user orgUnits by orgUnit ids failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("delete old user orgUnits by orgUnit ids failed")
	}
	return nil
}

// RemoveOrgUnitsFromUser 从用户移除组织单元
func (r *UserOrgUnitRepo) RemoveOrgUnitsFromUser(ctx context.Context, userID uint32, orgUnitIDs []uint32) error {
	if len(orgUnitIDs) == 0 || userID == 0 {
		return nil
	}

	_, err := r.entClient.Client().UserOrgUnit.Delete().
		Where(
			userorgunit.And(
				userorgunit.UserIDEQ(userID),
				userorgunit.OrgUnitIDIn(orgUnitIDs...),
			),
		).
		Exec(ctx)
	if err != nil {
		r.log.Errorf("remove user orgUnits failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("remove user orgUnits failed")
	}
	return nil
}

// AssignUserOrgUnit 分配组织单元给用户
func (r *UserOrgUnitRepo) AssignUserOrgUnit(
	ctx context.Context,
	tx *ent.Tx,
	data *identityV1.UserOrgUnit,
) error {
	if data == nil {
		return nil
	}

	now := time.Now()
	if data.StartAt == nil {
		data.StartAt = timeutil.TimeToTimestamppb(&now)
	}
	_, err := tx.UserOrgUnit.
		Create().
		SetUserID(data.GetUserId()).
		SetOrgUnitID(data.GetOrgUnitId()).
		SetNillableStatus(r.statusConverter.ToEntity(data.Status)).
		SetNillableAssignedBy(data.AssignedBy).
		SetNillableAssignedAt(timeutil.TimestamppbToTime(data.AssignedAt)).
		SetNillableIsPrimary(data.IsPrimary).
		SetNillableStartAt(timeutil.TimestamppbToTime(data.StartAt)).
		SetNillableEndAt(timeutil.TimestamppbToTime(data.EndAt)).
		SetNillableCreatedBy(data.CreatedBy).
		SetCreatedAt(now).
		Save(ctx)
	if err != nil {
		r.log.Errorf("assign orgUnit to user failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("assign orgUnit to user failed")
	}
	return nil
}

// AssignUserOrgUnits 分配组织单元给用户
func (r *UserOrgUnitRepo) AssignUserOrgUnits(
	ctx context.Context, tx *ent.Tx,
	userID uint32,
	datas []*identityV1.UserOrgUnit,
) error {
	if len(datas) == 0 || userID == 0 {
		return nil
	}

	var err error

	// 删除该角色的所有旧关联
	if err = r.CleanRelationsByUserID(ctx, tx, userID); err != nil {
		return identityV1.ErrorInternalServerError("clean old user orgUnits failed")
	}

	now := time.Now()

	var userOrgUnitCreates []*ent.UserOrgUnitCreate
	for _, data := range datas {
		if data.StartAt == nil {
			data.StartAt = timeutil.TimeToTimestamppb(&now)
		}
		rm := tx.UserOrgUnit.
			Create().
			SetNillableTenantID(data.TenantId).
			SetUserID(data.GetUserId()).
			SetOrgUnitID(data.GetOrgUnitId()).
			SetNillableStatus(r.statusConverter.ToEntity(data.Status)).
			SetNillableAssignedBy(data.AssignedBy).
			SetNillableAssignedAt(timeutil.TimestamppbToTime(data.AssignedAt)).
			SetNillableIsPrimary(data.IsPrimary).
			SetNillableStartAt(timeutil.TimestamppbToTime(data.StartAt)).
			SetNillableEndAt(timeutil.TimestamppbToTime(data.EndAt)).
			SetNillableCreatedBy(data.CreatedBy).
			SetCreatedAt(now)
		userOrgUnitCreates = append(userOrgUnitCreates, rm)
	}

	_, err = tx.UserOrgUnit.CreateBulk(userOrgUnitCreates...).Save(ctx)
	if err != nil {
		r.log.Errorf("assign orgUnit to user failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("assign orgUnit to user failed")
	}

	return nil
}

// ListOrgUnitIDs 列出角色关联的组织单元ID列表
func (r *UserOrgUnitRepo) ListOrgUnitIDs(ctx context.Context, userID uint32, excludeExpired bool) ([]uint32, error) {
	if userID == 0 {
		return []uint32{}, nil
	}

	q := r.entClient.Client().UserOrgUnit.Query().
		Where(
			userorgunit.UserIDEQ(userID),
		)

	if excludeExpired {
		now := time.Now()
		q = q.Where(
			userorgunit.Or(
				userorgunit.EndAtIsNil(),
				userorgunit.EndAtGT(now),
			),
		)
	}

	intIDs, err := q.
		Select(userorgunit.FieldOrgUnitID).
		Ints(ctx)
	if err != nil {
		r.log.Errorf("query orgUnit ids by user id failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query orgUnit ids by user id failed")
	}
	ids := make([]uint32, len(intIDs))
	for i, v := range intIDs {
		ids[i] = uint32(v)
	}
	return ids, nil
}

// ListUserIDs 列出组织单元关联的用户ID列表
func (r *UserOrgUnitRepo) ListUserIDs(ctx context.Context, orgUnitID uint32, excludeExpired bool) ([]uint32, error) {
	if orgUnitID == 0 {
		return []uint32{}, nil
	}

	q := r.entClient.Client().UserOrgUnit.Query().
		Where(
			userorgunit.OrgUnitIDEQ(orgUnitID),
		)

	if excludeExpired {
		now := time.Now()
		q = q.Where(
			userorgunit.Or(
				userorgunit.EndAtIsNil(),
				userorgunit.EndAtGT(now),
			),
		)
	}

	intIDs, err := q.
		Select(userorgunit.FieldUserID).
		Ints(ctx)
	if err != nil {
		r.log.Errorf("query user ids by orgUnit id failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query user ids by orgUnit id failed")
	}
	ids := make([]uint32, len(intIDs))
	for i, v := range intIDs {
		ids[i] = uint32(v)
	}
	return ids, nil
}

// ListUserIDsByOrgUnitIDs 列出多个组织单元关联的用户ID列表
func (r *UserOrgUnitRepo) ListUserIDsByOrgUnitIDs(ctx context.Context, orgUnitIDs []uint32, excludeExpired bool) ([]uint32, error) {
	if len(orgUnitIDs) == 0 {
		return nil, nil
	}

	q := r.entClient.Client().UserOrgUnit.Query().
		Where(
			userorgunit.OrgUnitIDIn(orgUnitIDs...),
		)

	if excludeExpired {
		now := time.Now()
		q = q.Where(
			userorgunit.Or(
				userorgunit.EndAtIsNil(),
				userorgunit.EndAtGT(now),
			),
		)
	}

	intIDs, err := q.
		Select(userorgunit.FieldUserID).
		Ints(ctx)
	if err != nil {
		r.log.Errorf("query user ids by orgUnit ids failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query user ids by orgUnit ids failed")
	}
	ids := make([]uint32, len(intIDs))
	for i, v := range intIDs {
		ids[i] = uint32(v)
	}
	return ids, nil
}
