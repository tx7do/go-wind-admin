//go:build gorm_backend
// +build gorm_backend

package gorm

import (
	"context"

	gormDB "gorm.io/gorm"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	gormCrud "github.com/tx7do/go-crud/gorm"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/admin/service/internal/data/gorm/models"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

type RolePermissionRepo struct {
	client          *gormCrud.Client
	log             *bLogger.Helper
	mapper          *mapper.CopierMapper[permissionV1.RolePermission, models.RolePermission]
	repository      *gormCrud.Repository[permissionV1.RolePermission, models.RolePermission]
	statusConverter *mapper.EnumTypeConverter[permissionV1.RolePermission_Status, string]
	effectConverter *mapper.EnumTypeConverter[permissionV1.RolePermission_EffectiveStatus, string]
}

func NewRolePermissionRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *RolePermissionRepo {
	repo := &RolePermissionRepo{
		log:    ctx.NewLoggerHelper("role-permission/gorm-repo/admin-service"),
		client: client,
		mapper: mapper.NewCopierMapper[permissionV1.RolePermission, models.RolePermission](),
		statusConverter: mapper.NewEnumTypeConverter[permissionV1.RolePermission_Status, string](
			permissionV1.RolePermission_Status_name,
			permissionV1.RolePermission_Status_value,
		),
		effectConverter: mapper.NewEnumTypeConverter[permissionV1.RolePermission_EffectiveStatus, string](
			permissionV1.RolePermission_EffectiveStatus_name,
			permissionV1.RolePermission_EffectiveStatus_value,
		),
	}

	repo.init()

	return repo
}

func (r *RolePermissionRepo) init() {
	r.repository = gormCrud.NewRepository[permissionV1.RolePermission, models.RolePermission](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
	r.mapper.AppendConverters(r.effectConverter.NewConverterPair())
}

// CleanPermissions 清理角色的所有权限
func (r *RolePermissionRepo) CleanPermissions(
	ctx context.Context,
	roleID uint32,
) error {
	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("role_id = ?", roleID) },
	}); err != nil {
		r.log.Errorf(ctx, "delete old role [%d] permissions failed: %s", roleID, err.Error())
		return permissionV1.ErrorInternalServerError("delete old role permissions failed")
	}
	return nil
}

func (r *RolePermissionRepo) BatchCreate(ctx context.Context, datas []*permissionV1.RolePermission) error {
	if len(datas) == 0 {
		return nil
	}

	if _, err := r.repository.BatchCreate(ctx, r.client.DB, datas, nil); err != nil {
		r.log.Errorf(ctx, "batch create role permissions failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("batch create role permissions failed")
	}

	return nil
}

// Upsert 创建或更新角色权限关联
func (r *RolePermissionRepo) Upsert(ctx context.Context, data *permissionV1.RolePermission) error {
	if data == nil {
		return nil
	}

	if _, err := r.repository.Create(ctx, r.client.DB, data, nil); err != nil {
		r.log.Errorf(ctx, "create role permission failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("create role permission failed")
	}

	return nil
}

// ReplacePermissions 整体替换角色的权限：先清空该角色现有权限，再插入新的权限集合。
// 与 AssignPermissions（仅 upsert，不删除）不同，此方法用于更新场景，确保取消勾选的权限被正确移除。
func (r *RolePermissionRepo) ReplacePermissions(ctx context.Context,
	tenantID, operatorID uint32,
	roleID uint32, permissionIDs []uint32,
) error {
	if err := r.CleanPermissions(ctx, roleID); err != nil {
		return err
	}

	if len(permissionIDs) == 0 {
		return nil
	}

	return r.AssignPermissions(ctx, tenantID, operatorID, roleID, permissionIDs)
}

// AssignPermissions 给角色分配权限
func (r *RolePermissionRepo) AssignPermissions(ctx context.Context,
	tenantID, operatorID uint32,
	roleID uint32, permissionIDs []uint32,
) error {
	if len(permissionIDs) == 0 {
		return nil
	}

	datas := make([]*permissionV1.RolePermission, 0, len(permissionIDs))
	for _, permissionID := range permissionIDs {
		permID := permissionID
		role := roleID
		tenant := tenantID
		operator := operatorID
		dto := &permissionV1.RolePermission{}
		dto.PermissionId = &permID
		dto.RoleId = &role
		dto.TenantId = &tenant
		dto.CreatedBy = &operator
		datas = append(datas, dto)
	}

	if _, err := r.repository.BatchCreate(ctx, r.client.DB, datas, nil); err != nil {
		r.log.Errorf(ctx, "assign permission to role failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("assign permission to role failed")
	}

	return nil
}

// ListPermissionIDs 列出角色的权限ID列表
func (r *RolePermissionRepo) ListPermissionIDs(ctx context.Context, roleID uint32) ([]uint32, error) {
	var ids []uint32
	if err := r.client.DB.WithContext(ctx).
		Model(&models.RolePermission{}).
		Where("role_id = ?", roleID).
		Pluck("permission_id", &ids).Error; err != nil {
		r.log.Errorf(ctx, "query permission ids by role id failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query permission ids by role id failed")
	}
	return ids, nil
}

// ListPermissionIDsByRoleIDs 根据角色ID列表获取权限ID列表
func (r *RolePermissionRepo) ListPermissionIDsByRoleIDs(ctx context.Context, roleIDs []uint32) ([]uint32, error) {
	var ids []uint32
	if err := r.client.DB.WithContext(ctx).
		Model(&models.RolePermission{}).
		Where("role_id IN ?", roleIDs).
		Pluck("permission_id", &ids).Error; err != nil {
		r.log.Errorf(ctx, "query permission ids by role ids failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query permission ids by role ids failed")
	}
	return ids, nil
}

// RemovePermissions 移除角色的部分权限
func (r *RolePermissionRepo) RemovePermissions(ctx context.Context, tenantID, roleID uint32, permissionIDs []uint32) error {
	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB {
			return db.Where("role_id = ?", roleID).
				Where("tenant_id = ?", tenantID).
				Where("permission_id IN ?", permissionIDs)
		},
	}); err != nil {
		r.log.Errorf(ctx, "remove roles by role id failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("remove roles by role id failed")
	}
	return nil
}

func (r *RolePermissionRepo) ListPermissionsByRoleID(ctx context.Context, roleID uint32) ([]*permissionV1.RolePermission, error) {
	var entities []*models.RolePermission
	if err := r.client.DB.WithContext(ctx).
		Model(&models.RolePermission{}).
		Where("role_id = ?", roleID).
		Find(&entities).Error; err != nil {
		r.log.Errorf(ctx, "list role permissions by role id failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("list role permissions by role id failed")
	}

	results := make([]*permissionV1.RolePermission, 0, len(entities))
	for _, entity := range entities {
		results = append(results, r.mapper.ToDTO(entity))
	}

	return results, nil
}
