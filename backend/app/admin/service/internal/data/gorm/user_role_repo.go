//go:build gorm_backend
// +build gorm_backend

package gorm

import (
	"context"
	"time"

	gormDB "gorm.io/gorm"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	gormCrud "github.com/tx7do/go-crud/gorm"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/admin/service/internal/data/gorm/models"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

type UserRoleRepo struct {
	client     *gormCrud.Client
	log        *bLogger.Helper
	mapper     *mapper.CopierMapper[permissionV1.UserRole, models.UserRole]
	repository *gormCrud.Repository[permissionV1.UserRole, models.UserRole]
}

func NewUserRoleRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *UserRoleRepo {
	repo := &UserRoleRepo{
		log:    ctx.NewLoggerHelper("user-role/gorm-repo/admin-service"),
		client: client,
		mapper: mapper.NewCopierMapper[permissionV1.UserRole, models.UserRole](),
	}

	repo.init()

	return repo
}

func (r *UserRoleRepo) init() {
	r.repository = gormCrud.NewRepository[permissionV1.UserRole, models.UserRole](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

// CleanRelationsByUserID 删除会员的所有角色关联
func (r *UserRoleRepo) CleanRelationsByUserID(ctx context.Context, userID uint32) error {
	if userID == 0 {
		return nil
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("user_id = ?", userID) },
	}); err != nil {
		r.log.Errorf(ctx, "delete old user roles failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete old user roles failed")
	}
	return nil
}

// CleanRelationsByUserIDs 删除多个会员的所有角色关联
func (r *UserRoleRepo) CleanRelationsByUserIDs(ctx context.Context, userIDs []uint32) error {
	if len(userIDs) == 0 {
		return nil
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("user_id IN ?", userIDs) },
	}); err != nil {
		r.log.Errorf(ctx, "delete old user roles by user ids failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete old user roles by user ids failed")
	}
	return nil
}

// CleanRelationsByRoleID 删除角色的所有用户关联
func (r *UserRoleRepo) CleanRelationsByRoleID(ctx context.Context, roleID uint32) error {
	if roleID == 0 {
		return nil
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("role_id = ?", roleID) },
	}); err != nil {
		r.log.Errorf(ctx, "delete old user roles by role id failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete old user roles by role id failed")
	}
	return nil
}

// CleanRelationsByRoleIDs 删除多个角色的所有用户关联
func (r *UserRoleRepo) CleanRelationsByRoleIDs(ctx context.Context, roleIDs []uint32) error {
	if len(roleIDs) == 0 {
		return nil
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("role_id IN ?", roleIDs) },
	}); err != nil {
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

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB {
			return db.Where("user_id = ?", userID).Where("role_id IN ?", roleIDs)
		},
	}); err != nil {
		r.log.Errorf(ctx, "remove roles from user failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("remove roles from user failed")
	}
	return nil
}

// AssignUserRole 分配角色
func (r *UserRoleRepo) AssignUserRole(ctx context.Context, data *permissionV1.UserRole) error {
	if data == nil {
		return nil
	}

	if _, err := r.repository.Create(ctx, r.client.DB, data, nil); err != nil {
		r.log.Errorf(ctx, "assign role to user failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("assign role to user failed")
	}

	return nil
}

// AssignUserRoles 分配角色
func (r *UserRoleRepo) AssignUserRoles(ctx context.Context, userID uint32, datas []*permissionV1.UserRole) error {
	if len(datas) == 0 || userID == 0 {
		return nil
	}

	// 删除该用户的所有旧关联
	if err := r.CleanRelationsByUserID(ctx, userID); err != nil {
		return permissionV1.ErrorInternalServerError("clean old user roles failed")
	}

	if _, err := r.repository.BatchCreate(ctx, r.client.DB, datas, nil); err != nil {
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

	db := r.client.DB.WithContext(ctx).Model(&models.UserRole{}).Where("user_id = ?", userID)

	if excludeExpired {
		now := time.Now()
		db = db.Where("end_at IS NULL OR end_at > ?", now)
	}

	var ids []uint32
	if err := db.Pluck("role_id", &ids).Error; err != nil {
		r.log.Errorf(ctx, "query role ids by user id failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query role ids by user id failed")
	}
	return ids, nil
}

// ListUserIDs 获取角色关联的用户ID列表
func (r *UserRoleRepo) ListUserIDs(ctx context.Context, roleID uint32, excludeExpired bool) ([]uint32, error) {
	if roleID == 0 {
		return []uint32{}, nil
	}

	db := r.client.DB.WithContext(ctx).Model(&models.UserRole{}).Where("role_id = ?", roleID)

	if excludeExpired {
		now := time.Now()
		db = db.Where("end_at IS NULL OR end_at > ?", now)
	}

	var ids []uint32
	if err := db.Pluck("user_id", &ids).Error; err != nil {
		r.log.Errorf(ctx, "query user ids by role id failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query user ids by role id failed")
	}
	return ids, nil
}

// ListUserIDsByRoleIDs 获取多个角色关联的用户ID列表
func (r *UserRoleRepo) ListUserIDsByRoleIDs(ctx context.Context, roleIDs []uint32, excludeExpired bool) ([]uint32, error) {
	if len(roleIDs) == 0 {
		return []uint32{}, nil
	}

	db := r.client.DB.WithContext(ctx).Model(&models.UserRole{}).Where("role_id IN ?", roleIDs)

	if excludeExpired {
		now := time.Now()
		db = db.Where("end_at IS NULL OR end_at > ?", now)
	}

	var ids []uint32
	if err := db.Pluck("user_id", &ids).Error; err != nil {
		r.log.Errorf(ctx, "query user ids by role ids failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query user ids by role ids failed")
	}
	return ids, nil
}
