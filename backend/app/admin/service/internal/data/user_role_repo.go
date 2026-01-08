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
	"go-wind-admin/app/admin/service/internal/data/ent/userrole"

	userV1 "go-wind-admin/api/gen/go/user/service/v1"
)

type UserRoleRepo struct {
	log             *log.Helper
	entClient       *entCrud.EntClient[*ent.Client]
	statusConverter *mapper.EnumTypeConverter[userV1.UserRole_Status, userrole.Status]
}

func NewUserRoleRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *UserRoleRepo {
	return &UserRoleRepo{
		log:       ctx.NewLoggerHelper("user-role/repo/admin-service"),
		entClient: entClient,
		statusConverter: mapper.NewEnumTypeConverter[userV1.UserRole_Status, userrole.Status](
			userV1.UserRole_Status_name,
			userV1.UserRole_Status_value,
		),
	}
}

func (r *UserRoleRepo) CleanRoles(
	ctx context.Context,
	tx *ent.Tx,
	userID, tenantID uint32,
) error {
	if _, err := tx.UserRole.Delete().
		Where(
			userrole.UserIDEQ(userID),
			userrole.TenantIDEQ(tenantID),
		).
		Exec(ctx); err != nil {
		r.log.Errorf("delete old user roles failed: %s", err.Error())
		return userV1.ErrorInternalServerError("delete old user roles failed")
	}
	return nil
}

// AssignRoles 分配角色
func (r *UserRoleRepo) AssignRoles(ctx context.Context,
	tx *ent.Tx,
	userID, tenantID uint32,
	datas []*userV1.UserRole,
) error {
	var err error

	// 删除该用户的所有旧关联
	if err = r.CleanRoles(ctx, tx, userID, tenantID); err != nil {
		return userV1.ErrorInternalServerError("clean old user roles failed")
	}

	if len(datas) == 0 {
		return nil
	}

	now := time.Now()

	var userRoleCreates []*ent.UserRoleCreate
	for _, data := range datas {
		if data.StartAt == nil {
			data.StartAt = timeutil.TimeToTimestamppb(&now)
		}

		rm := tx.UserRole.
			Create().
			SetTenantID(tenantID).
			SetUserID(data.GetUserId()).
			SetRoleID(data.GetRoleId()).
			SetNillableStatus(r.statusConverter.ToEntity(data.Status)).
			SetNillableAssignedBy(data.AssignedBy).
			SetNillableAssignedAt(timeutil.TimestamppbToTime(data.AssignedAt)).
			SetNillableIsPrimary(data.IsPrimary).
			SetNillableStartAt(timeutil.TimestamppbToTime(data.StartAt)).
			SetNillableEndAt(timeutil.TimestamppbToTime(data.EndAt)).
			SetCreatedAt(now).
			SetNillableCreatedBy(data.CreatedBy)
		userRoleCreates = append(userRoleCreates, rm)
	}

	_, err = tx.UserRole.CreateBulk(userRoleCreates...).Save(ctx)
	if err != nil {
		r.log.Errorf("assign roles to user failed: %s", err.Error())
		return userV1.ErrorInternalServerError("assign roles to user failed")
	}

	return nil
}

// ListRoleIDs 获取用户关联的角色ID列表
func (r *UserRoleRepo) ListRoleIDs(ctx context.Context, userID uint32, excludeExpired bool) ([]uint32, error) {
	q := r.entClient.Client().UserRole.Query().
		Where(
			userrole.UserIDEQ(userID),
		)

	if excludeExpired {
		now := time.Now()
		q = q.Where(
			userrole.Or(
				userrole.EndAtIsNil(),
				userrole.EndAtGT(now),
			),
		)
	}

	intIDs, err := q.
		Select(userrole.FieldRoleID).
		Ints(ctx)
	if err != nil {
		r.log.Errorf("query role ids by user id failed: %s", err.Error())
		return nil, userV1.ErrorInternalServerError("query role ids by user id failed")
	}
	ids := make([]uint32, len(intIDs))
	for i, v := range intIDs {
		ids[i] = uint32(v)
	}
	return ids, nil
}

// RemoveRoles 移除角色
func (r *UserRoleRepo) RemoveRoles(ctx context.Context, userID, tenantID uint32, roleIDs []uint32) error {
	_, err := r.entClient.Client().UserRole.Delete().
		Where(
			userrole.And(
				userrole.UserIDEQ(userID),
				userrole.TenantIDEQ(tenantID),
				userrole.RoleIDIn(roleIDs...),
			),
		).
		Exec(ctx)
	if err != nil {
		r.log.Errorf("remove roles from user failed: %s", err.Error())
		return userV1.ErrorInternalServerError("remove roles from user failed")
	}
	return nil
}
