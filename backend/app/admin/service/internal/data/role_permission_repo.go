package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/rolepermission"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
)

type RolePermissionRepo struct {
	log       *log.Helper
	entClient *entCrud.EntClient[*ent.Client]
}

func NewRolePermissionRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *RolePermissionRepo {
	return &RolePermissionRepo{
		log:       ctx.NewLoggerHelper("role-permission/repo/admin-service"),
		entClient: entClient,
	}
}

// CleanPermissions 清理角色的所有权限
func (r *RolePermissionRepo) CleanPermissions(
	ctx context.Context,
	tx *ent.Tx,
	roleID uint32,
) error {
	if _, err := tx.RolePermission.Delete().
		Where(
			rolepermission.RoleIDEQ(roleID),
		).
		Exec(ctx); err != nil {
		r.log.Errorf("delete old role [%d] permissions failed: %s", roleID, err.Error())
		return adminV1.ErrorInternalServerError("delete old role permissions failed")
	}
	return nil
}

// AssignPermissions 给角色分配权限
func (r *RolePermissionRepo) AssignPermissions(ctx context.Context, tx *ent.Tx, tenantID, roleID, operatorID uint32, permissions []uint32) error {
	if len(permissions) == 0 {
		return nil
	}

	now := time.Now()

	for _, permissionID := range permissions {
		rp := tx.RolePermission.
			Create().
			SetTenantID(tenantID).
			SetPermissionID(permissionID).
			SetRoleID(roleID).
			SetCreatedBy(operatorID).
			SetCreatedAt(now).
			OnConflictColumns(
				rolepermission.FieldTenantID,
				rolepermission.FieldPermissionID,
				rolepermission.FieldRoleID,
			).
			UpdateNewValues().
			SetUpdatedBy(operatorID).
			SetUpdatedAt(now)

		if err := rp.Exec(ctx); err != nil {
			r.log.Errorf("assign permission to role failed: %s", err.Error())
			return adminV1.ErrorInternalServerError("assign permission to role failed")
		}
	}

	return nil
}

// ListPermissionIDs 列出角色的权限ID列表
func (r *RolePermissionRepo) ListPermissionIDs(ctx context.Context, roleID uint32) ([]uint32, error) {
	q := r.entClient.Client().RolePermission.Query().
		Where(
			rolepermission.RoleIDEQ(roleID),
		)

	intIDs, err := q.
		Select(rolepermission.FieldPermissionID).
		Ints(ctx)
	if err != nil {
		r.log.Errorf("query permission ids by role id failed: %s", err.Error())
		return nil, adminV1.ErrorInternalServerError("query permission ids by role id failed")
	}
	ids := make([]uint32, len(intIDs))
	for i, v := range intIDs {
		ids[i] = uint32(v)
	}
	return ids, nil
}

// GetPermissionsByRoleIDs 根据角色ID列表获取权限ID列表
func (r *RolePermissionRepo) GetPermissionsByRoleIDs(ctx context.Context, roleIDs []uint32) ([]uint32, error) {
	q := r.entClient.Client().RolePermission.Query().
		Where(
			rolepermission.RoleIDIn(roleIDs...),
		)

	intIDs, err := q.
		Select(rolepermission.FieldPermissionID).
		Ints(ctx)
	if err != nil {
		r.log.Errorf("query permission ids by role ids failed: %s", err.Error())
		return nil, adminV1.ErrorInternalServerError("query permission ids by role ids failed")
	}
	ids := make([]uint32, len(intIDs))
	for i, v := range intIDs {
		ids[i] = uint32(v)
	}
	return ids, nil
}

// RemovePermissions 移除角色的部分权限
func (r *RolePermissionRepo) RemovePermissions(ctx context.Context, tenantID, roleID uint32, permissionIDs []uint32) error {
	_, err := r.entClient.Client().RolePermission.Delete().
		Where(
			rolepermission.And(
				rolepermission.RoleIDEQ(roleID),
				rolepermission.TenantIDEQ(tenantID),
				rolepermission.PermissionIDIn(permissionIDs...),
			),
		).
		Exec(ctx)
	if err != nil {
		r.log.Errorf("remove roles by role id failed: %s", err.Error())
		return adminV1.ErrorInternalServerError("remove roles by role id failed")
	}
	return nil
}
