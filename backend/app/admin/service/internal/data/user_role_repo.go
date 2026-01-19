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

// CleanRelationsByUserID 删除会员的所有角色关联
func (r *UserRoleRepo) CleanRelationsByUserID(ctx context.Context, tx *ent.Tx, userID uint32) error {
	if userID == 0 {
		return nil
	}

	if _, err := tx.UserRole.Delete().
		Where(
			userrole.UserIDEQ(userID),
		).
		Exec(ctx); err != nil {
		r.log.Errorf("delete old user roles failed: %s", err.Error())
		return userV1.ErrorInternalServerError("delete old user roles failed")
	}
	return nil
}

// CleanRelationsByUserIDs 删除多个会员的所有角色关联
func (r *UserRoleRepo) CleanRelationsByUserIDs(ctx context.Context, tx *ent.Tx, userIDs []uint32) error {
	if len(userIDs) == 0 {
		return nil
	}

	if _, err := tx.UserRole.Delete().
		Where(
			userrole.UserIDIn(userIDs...),
		).
		Exec(ctx); err != nil {
		r.log.Errorf("delete old user roles by user ids failed: %s", err.Error())
		return userV1.ErrorInternalServerError("delete old user roles by user ids failed")
	}
	return nil
}

// CleanRelationsByRoleID 删除角色的所有用户关联
func (r *UserRoleRepo) CleanRelationsByRoleID(ctx context.Context, tx *ent.Tx, roleID uint32) error {
	if roleID == 0 {
		return nil
	}

	if _, err := tx.UserRole.Delete().
		Where(
			userrole.RoleIDEQ(roleID),
		).
		Exec(ctx); err != nil {
		r.log.Errorf("delete old user roles by role id failed: %s", err.Error())
		return userV1.ErrorInternalServerError("delete old user roles by role id failed")
	}
	return nil
}

// CleanRelationsByRoleIDs 删除多个角色的所有用户关联
func (r *UserRoleRepo) CleanRelationsByRoleIDs(ctx context.Context, tx *ent.Tx, roleIDs []uint32) error {
	if len(roleIDs) == 0 {
		return nil
	}

	if _, err := tx.UserRole.Delete().
		Where(
			userrole.RoleIDIn(roleIDs...),
		).
		Exec(ctx); err != nil {
		r.log.Errorf("delete old user roles by role ids failed: %s", err.Error())
		return userV1.ErrorInternalServerError("delete old user roles by role ids failed")
	}
	return nil
}

// RemoveRolesFromUser 从用户移除角色
func (r *UserRoleRepo) RemoveRolesFromUser(ctx context.Context, userID uint32, roleIDs []uint32) error {
	if len(roleIDs) == 0 || userID == 0 {
		return nil
	}

	_, err := r.entClient.Client().UserRole.Delete().
		Where(
			userrole.And(
				userrole.UserIDEQ(userID),
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

// AssignUserRole 分配角色
func (r *UserRoleRepo) AssignUserRole(ctx context.Context, tx *ent.Tx, data *userV1.UserRole) error {
	if data == nil {
		return nil
	}

	now := time.Now()

	_, err := tx.UserRole.
		Create().
		SetUserID(data.GetUserId()).
		SetRoleID(data.GetRoleId()).
		SetNillableStatus(r.statusConverter.ToEntity(data.Status)).
		SetNillableAssignedBy(data.AssignedBy).
		SetNillableAssignedAt(timeutil.TimestamppbToTime(data.AssignedAt)).
		SetNillableIsPrimary(data.IsPrimary).
		SetNillableStartAt(timeutil.TimestamppbToTime(data.StartAt)).
		SetNillableEndAt(timeutil.TimestamppbToTime(data.EndAt)).
		SetCreatedAt(now).
		SetNillableCreatedBy(data.CreatedBy).
		Save(ctx)
	if err != nil {
		r.log.Errorf("assign role to user failed: %s", err.Error())
		return userV1.ErrorInternalServerError("assign role to user failed")
	}

	return nil
}

// AssignUserRoles 分配角色
func (r *UserRoleRepo) AssignUserRoles(ctx context.Context, tx *ent.Tx, userID uint32, datas []*userV1.UserRole) error {
	if len(datas) == 0 || userID == 0 {
		return nil
	}

	var err error

	// 删除该用户的所有旧关联
	if err = r.CleanRelationsByUserID(ctx, tx, userID); err != nil {
		return userV1.ErrorInternalServerError("clean old user roles failed")
	}

	now := time.Now()

	var userRoleCreates []*ent.UserRoleCreate
	for _, data := range datas {
		if data.StartAt == nil {
			data.StartAt = timeutil.TimeToTimestamppb(&now)
		}

		rm := tx.UserRole.
			Create().
			SetNillableTenantID(data.TenantId).
			SetUserID(userID).
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
	if userID == 0 {
		return []uint32{}, nil
	}

	q := r.entClient.Client().Debug().UserRole.Query().
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

// ListUserIDs 获取角色关联的用户ID列表
func (r *UserRoleRepo) ListUserIDs(ctx context.Context, roleID uint32, excludeExpired bool) ([]uint32, error) {
	if roleID == 0 {
		return []uint32{}, nil
	}

	q := r.entClient.Client().UserRole.Query().
		Where(
			userrole.RoleIDEQ(roleID),
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
		Select(userrole.FieldUserID).
		Ints(ctx)
	if err != nil {
		r.log.Errorf("query user ids by role id failed: %s", err.Error())
		return nil, userV1.ErrorInternalServerError("query user ids by role id failed")
	}
	ids := make([]uint32, len(intIDs))
	for i, v := range intIDs {
		ids[i] = uint32(v)
	}
	return ids, nil
}

// ListUserIDsByRoleIDs 获取多个角色关联的用户ID列表
func (r *UserRoleRepo) ListUserIDsByRoleIDs(ctx context.Context, roleIDs []uint32, excludeExpired bool) ([]uint32, error) {
	if len(roleIDs) == 0 {
		return nil, nil
	}

	q := r.entClient.Client().UserRole.Query().
		Where(
			userrole.RoleIDIn(roleIDs...),
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
		Select(userrole.FieldUserID).
		Ints(ctx)
	if err != nil {
		r.log.Errorf("query user ids by role ids failed: %s", err.Error())
		return nil, userV1.ErrorInternalServerError("query user ids by role ids failed")
	}
	ids := make([]uint32, len(intIDs))
	for i, v := range intIDs {
		ids[i] = uint32(v)
	}
	return ids, nil
}
