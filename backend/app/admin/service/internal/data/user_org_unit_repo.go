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

	userV1 "go-wind-admin/api/gen/go/user/service/v1"
)

type UserOrgUnitRepo struct {
	log *log.Helper

	entClient       *entCrud.EntClient[*ent.Client]
	statusConverter *mapper.EnumTypeConverter[userV1.UserOrgUnit_Status, userorgunit.Status]
}

func NewUserOrgUnitRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *UserOrgUnitRepo {
	return &UserOrgUnitRepo{
		log:       ctx.NewLoggerHelper("user-org-unit/repo/admin-service"),
		entClient: entClient,
		statusConverter: mapper.NewEnumTypeConverter[userV1.UserOrgUnit_Status, userorgunit.Status](
			userV1.UserOrgUnit_Status_name,
			userV1.UserOrgUnit_Status_value,
		),
	}
}

// CleanOrgUnits 清理会员组织单元关联
func (r *UserOrgUnitRepo) CleanOrgUnits(
	ctx context.Context,
	tx *ent.Tx,
	userID, tenantID uint32,
) error {
	if _, err := tx.UserOrgUnit.Delete().
		Where(
			userorgunit.UserIDEQ(userID),
			userorgunit.TenantIDEQ(tenantID),
		).
		Exec(ctx); err != nil {
		r.log.Errorf("delete old user orgUnits failed: %s", err.Error())
		return userV1.ErrorInternalServerError("delete old user orgUnits failed")
	}
	return nil
}

// AssignOrgUnits 分配组织单元给会员
func (r *UserOrgUnitRepo) AssignOrgUnits(
	ctx context.Context,
	tx *ent.Tx,
	userID, tenantID uint32,
	datas []*userV1.UserOrgUnit,
) error {
	var err error

	// 删除该角色的所有旧关联
	if err = r.CleanOrgUnits(ctx, tx, userID, tenantID); err != nil {
		return userV1.ErrorInternalServerError("clean old user orgUnits failed")
	}

	// 如果没有分配任何，则直接提交事务返回
	if len(datas) == 0 {
		return nil
	}

	now := time.Now()

	var userOrgUnitCreates []*ent.UserOrgUnitCreate
	for _, data := range datas {
		if data.StartAt == nil {
			data.StartAt = timeutil.TimeToTimestamppb(&now)
		}
		rm := tx.UserOrgUnit.
			Create().
			SetTenantID(data.GetTenantId()).
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
		return userV1.ErrorInternalServerError("assign orgUnit to user failed")
	}

	return nil
}

// ListOrgUnitIDs 列出角色关联的组织单元ID列表
func (r *UserOrgUnitRepo) ListOrgUnitIDs(ctx context.Context, userID uint32, excludeExpired bool) ([]uint32, error) {
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
		return nil, userV1.ErrorInternalServerError("query orgUnit ids by user id failed")
	}
	ids := make([]uint32, len(intIDs))
	for i, v := range intIDs {
		ids[i] = uint32(v)
	}
	return ids, nil
}

// RemoveOrgUnits 删除会员的组织单元关联
func (r *UserOrgUnitRepo) RemoveOrgUnits(ctx context.Context, userID, tenantID uint32, ids []uint32) error {
	_, err := r.entClient.Client().UserOrgUnit.Delete().
		Where(
			userorgunit.And(
				userorgunit.UserIDEQ(userID),
				userorgunit.TenantIDEQ(tenantID),
				userorgunit.OrgUnitIDIn(ids...),
			),
		).
		Exec(ctx)
	if err != nil {
		r.log.Errorf("remove user orgUnits failed: %s", err.Error())
		return userV1.ErrorInternalServerError("remove user orgUnits failed")
	}
	return nil
}
