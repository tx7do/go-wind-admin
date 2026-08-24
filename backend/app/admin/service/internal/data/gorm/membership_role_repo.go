//go:build gorm_backend
// +build gorm_backend

package gorm

import (
	"context"
	"time"

	gormDB "gorm.io/gorm"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	gormCrud "github.com/tx7do/go-crud/gorm"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/admin/service/internal/data/gorm/models"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

type MembershipRoleRepo struct {
	client          *gormCrud.Client
	log             *log.Helper
	mapper          *mapper.CopierMapper[permissionV1.MembershipRole, models.MembershipRole]
	repository      *gormCrud.Repository[permissionV1.MembershipRole, models.MembershipRole]
	statusConverter *mapper.EnumTypeConverter[permissionV1.MembershipRole_Status, string]
}

func NewMembershipRoleRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *MembershipRoleRepo {
	repo := &MembershipRoleRepo{
		log:       ctx.NewLoggerHelper("membership-role/gorm-repo/admin-service"),
		client:    client,
		mapper:    mapper.NewCopierMapper[permissionV1.MembershipRole, models.MembershipRole](),
		statusConverter: mapper.NewEnumTypeConverter[permissionV1.MembershipRole_Status, string](
			permissionV1.MembershipRole_Status_name,
			permissionV1.MembershipRole_Status_value,
		),
	}

	repo.init()

	return repo
}

func (r *MembershipRoleRepo) init() {
	r.repository = gormCrud.NewRepository[permissionV1.MembershipRole, models.MembershipRole](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
}

// CleanRelationsByMembershipID 删除会员的所有角色关联
func (r *MembershipRoleRepo) CleanRelationsByMembershipID(ctx context.Context, membershipID uint32) error {
	if membershipID == 0 {
		return nil
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("membership_id = ?", membershipID) },
	}); err != nil {
		r.log.Errorf("delete old membership roles failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete old membership roles failed")
	}
	return nil
}

// CleanRelationsByMembershipIDs 删除多个会员的所有角色关联
func (r *MembershipRoleRepo) CleanRelationsByMembershipIDs(ctx context.Context, membershipIDs []uint32) error {
	if len(membershipIDs) == 0 {
		return nil
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("membership_id IN ?", membershipIDs) },
	}); err != nil {
		r.log.Errorf("delete old membership roles by membership ids failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete old membership roles by membership ids failed")
	}
	return nil
}

// CleanRelationsByRoleID 删除角色的所有会员关联
func (r *MembershipRoleRepo) CleanRelationsByRoleID(ctx context.Context, roleID uint32) error {
	if roleID == 0 {
		return nil
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("role_id = ?", roleID) },
	}); err != nil {
		r.log.Errorf("delete old membership roles by role id failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete old membership roles by role id failed")
	}
	return nil
}

// CleanRelationsByRoleIDs 删除多个角色的所有会员关联
func (r *MembershipRoleRepo) CleanRelationsByRoleIDs(ctx context.Context, roleIDs []uint32) error {
	if len(roleIDs) == 0 {
		return nil
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("role_id IN ?", roleIDs) },
	}); err != nil {
		r.log.Errorf("delete old membership roles by role ids failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete old membership roles by role ids failed")
	}
	return nil
}

// RemoveRolesFromMembership 移除角色
func (r *MembershipRoleRepo) RemoveRolesFromMembership(ctx context.Context, membershipID uint32, roleIDs []uint32) error {
	if membershipID == 0 || len(roleIDs) == 0 {
		return nil
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB {
			return db.Where("membership_id = ?", membershipID).Where("role_id IN ?", roleIDs)
		},
	}); err != nil {
		r.log.Errorf("remove roles from membership failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("remove roles from membership failed")
	}
	return nil
}

// AssignMembershipRoles 分配角色
func (r *MembershipRoleRepo) AssignMembershipRoles(ctx context.Context,
	membershipID uint32,
	datas []*permissionV1.MembershipRole,
) error {
	if membershipID == 0 || len(datas) == 0 {
		return nil
	}

	// 删除该用户的所有旧关联
	if err := r.CleanRelationsByMembershipID(ctx, membershipID); err != nil {
		return permissionV1.ErrorInternalServerError("clean old membership roles failed")
	}

	if _, err := r.repository.BatchCreate(ctx, r.client.DB, datas, nil); err != nil {
		r.log.Errorf("assign roles to membership failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("assign roles to membership failed")
	}

	return nil
}

// ListRoleIDs 获取用户关联的角色ID列表
func (r *MembershipRoleRepo) ListRoleIDs(ctx context.Context, membershipID uint32, excludeExpired bool) ([]uint32, error) {
	if membershipID == 0 {
		return []uint32{}, nil
	}

	db := r.client.DB.WithContext(ctx).Model(&models.MembershipRole{}).Where("membership_id = ?", membershipID)

	if excludeExpired {
		now := time.Now()
		db = db.Where("end_at IS NULL OR end_at > ?", now)
	}

	var ids []uint32
	if err := db.Pluck("role_id", &ids).Error; err != nil {
		r.log.Errorf("query role ids by membership id failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query role ids by membership id failed")
	}
	return ids, nil
}

// ListMembershipIDs 获取角色关联的会员ID列表
func (r *MembershipRoleRepo) ListMembershipIDs(ctx context.Context, roleID uint32, excludeExpired bool) ([]uint32, error) {
	if roleID == 0 {
		return []uint32{}, nil
	}

	db := r.client.DB.WithContext(ctx).Model(&models.MembershipRole{}).Where("role_id = ?", roleID)

	if excludeExpired {
		now := time.Now()
		db = db.Where("end_at IS NULL OR end_at > ?", now)
	}

	var ids []uint32
	if err := db.Pluck("membership_id", &ids).Error; err != nil {
		r.log.Errorf("query membership ids by role id failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query membership ids by role id failed")
	}
	return ids, nil
}

// ListMembershipIDsByRoleIDs 获取多个角色关联的会员ID列表
func (r *MembershipRoleRepo) ListMembershipIDsByRoleIDs(ctx context.Context, roleIDs []uint32, excludeExpired bool) ([]uint32, error) {
	if len(roleIDs) == 0 {
		return nil, nil
	}

	db := r.client.DB.WithContext(ctx).Model(&models.MembershipRole{}).Where("role_id IN ?", roleIDs)

	if excludeExpired {
		now := time.Now()
		db = db.Where("end_at IS NULL OR end_at > ?", now)
	}

	var ids []uint32
	if err := db.Pluck("membership_id", &ids).Error; err != nil {
		r.log.Errorf("query membership ids by role ids failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query membership ids by role ids failed")
	}
	return ids, nil
}
