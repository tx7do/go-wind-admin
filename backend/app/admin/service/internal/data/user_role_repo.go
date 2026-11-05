package data

import (
	"context"
	"time"

	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/go-crud/viewer"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/sliceutil"
	"github.com/tx7do/go-utils/timeutil"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/role"
	"go-wind-admin/app/admin/service/internal/data/ent/userrole"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

type UserRoleRepo struct {
	log             *bLogger.Helper
	entClient       *entCrud.EntClient[*ent.Client]
	statusConverter *mapper.EnumTypeConverter[permissionV1.UserRole_Status, userrole.Status]
}

func NewUserRoleRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *UserRoleRepo {
	return &UserRoleRepo{
		log:       ctx.NewLoggerHelper("user-role/repo/admin-service"),
		entClient: entClient,
		statusConverter: mapper.NewEnumTypeConverter[permissionV1.UserRole_Status, userrole.Status](
			permissionV1.UserRole_Status_name,
			permissionV1.UserRole_Status_value,
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
		r.log.Errorf(ctx, "delete old user roles failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete old user roles failed")
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
		r.log.Errorf(ctx, "delete old user roles by user ids failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete old user roles by user ids failed")
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
		r.log.Errorf(ctx, "delete old user roles by role id failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete old user roles by role id failed")
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
		r.log.Errorf(ctx, "delete old user roles by role ids failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete old user roles by role ids failed")
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
		r.log.Errorf(ctx, "remove roles from user failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("remove roles from user failed")
	}
	return nil
}

// validateRoleBindingOwnership 角色归属校验（纵深防御）。
//
// user_role join 行只落调用者租户的 tenant_id，但 role_id 此前不做任何校验——本租户行
// 可以挂任意角色的 id（platform:admin 种子数据恒为 id=1），配合 /me 这类无字段白名单
// 的入口即可自绑平台管理员，刷新令牌后所有平台闸门失效。这里对常规租户上下文强制
// 校验每个角色真实存在且可见：TenantPrivacy 按 viewer 租户过滤，租户 viewer 查不到
// platform/SYSTEM 角色与其他租户的角色（0 行即拒绝）。平台视图/系统上下文（种子/
// 引导/后台任务）按 TenantPrivacy 同一信任边界跳过。
func (r *UserRoleRepo) validateRoleBindingOwnership(ctx context.Context, tx *ent.Tx, roleIds []uint32) error {
	if len(roleIds) == 0 {
		return nil
	}

	if vc, ok := viewer.FromContext(ctx); ok && (vc.IsPlatformContext() || vc.IsSystemContext()) {
		return nil
	}

	ids := sliceutil.Unique(roleIds)

	roles, err := tx.Role.Query().Where(role.IDIn(ids...)).All(ctx)
	if err != nil {
		r.log.Errorf(ctx, "validate role bindings query failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("validate role bindings failed")
	}
	if len(roles) != len(ids) {
		r.log.Errorf(ctx, "some role bindings not found or not accessible for operator, requested [%v] visible [%d]",
			ids, len(roles))
		return permissionV1.ErrorBadRequest("some roles not found or not accessible")
	}

	return nil
}

// AssignUserRole 分配角色
func (r *UserRoleRepo) AssignUserRole(ctx context.Context, tx *ent.Tx, data *permissionV1.UserRole) error {
	if data == nil {
		return nil
	}

	if err := r.validateRoleBindingOwnership(ctx, tx, []uint32{data.GetRoleId()}); err != nil {
		return err
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
		r.log.Errorf(ctx, "assign role to user failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("assign role to user failed")
	}

	return nil
}

// AssignUserRoles 分配角色
func (r *UserRoleRepo) AssignUserRoles(ctx context.Context, tx *ent.Tx, userID uint32, datas []*permissionV1.UserRole) error {
	if len(datas) == 0 || userID == 0 {
		return nil
	}

	var err error

	// 先校验再清理：角色归属校验不过时不动现有关联
	roleIds := make([]uint32, 0, len(datas))
	for _, data := range datas {
		roleIds = append(roleIds, data.GetRoleId())
	}
	if err = r.validateRoleBindingOwnership(ctx, tx, roleIds); err != nil {
		return err
	}

	// 删除该用户的所有旧关联
	if err = r.CleanRelationsByUserID(ctx, tx, userID); err != nil {
		return permissionV1.ErrorInternalServerError("clean old user roles failed")
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
		r.log.Errorf(ctx, "assign roles to user failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("assign roles to user failed")
	}

	return nil
}

// ListRoleIDs 获取用户关联的角色ID列表
func (r *UserRoleRepo) ListRoleIDs(ctx context.Context, userID uint32, excludeExpired bool) ([]uint32, error) {
	if userID == 0 {
		return []uint32{}, nil
	}

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
		r.log.Errorf(ctx, "query role ids by user id failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query role ids by user id failed")
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
		r.log.Errorf(ctx, "query user ids by role id failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query user ids by role id failed")
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
		r.log.Errorf(ctx, "query user ids by role ids failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query user ids by role ids failed")
	}
	ids := make([]uint32, len(intIDs))
	for i, v := range intIDs {
		ids[i] = uint32(v)
	}
	return ids, nil
}
