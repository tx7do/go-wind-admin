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

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
)

type UserOrgUnitRepo struct {
	client          *gormCrud.Client
	log             *log.Helper
	mapper          *mapper.CopierMapper[identityV1.UserOrgUnit, models.UserOrgUnit]
	repository      *gormCrud.Repository[identityV1.UserOrgUnit, models.UserOrgUnit]
	statusConverter *mapper.EnumTypeConverter[identityV1.UserOrgUnit_Status, string]
}

func NewUserOrgUnitRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *UserOrgUnitRepo {
	repo := &UserOrgUnitRepo{
		log:       ctx.NewLoggerHelper("user-org-unit/gorm-repo/admin-service"),
		client:    client,
		mapper:    mapper.NewCopierMapper[identityV1.UserOrgUnit, models.UserOrgUnit](),
		statusConverter: mapper.NewEnumTypeConverter[identityV1.UserOrgUnit_Status, string](
			identityV1.UserOrgUnit_Status_name,
			identityV1.UserOrgUnit_Status_value,
		),
	}

	repo.init()

	return repo
}

func (r *UserOrgUnitRepo) init() {
	r.repository = gormCrud.NewRepository[identityV1.UserOrgUnit, models.UserOrgUnit](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
}

// CleanRelationsByUserID 清理用户组织单元关联
func (r *UserOrgUnitRepo) CleanRelationsByUserID(ctx context.Context, userID uint32) error {
	if userID == 0 {
		return nil
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("user_id = ?", userID) },
	}); err != nil {
		r.log.Errorf("delete old user orgUnits failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("delete old user orgUnits failed")
	}
	return nil
}

// CleanRelationsByUserIDs 清理多个用户组织单元关联
func (r *UserOrgUnitRepo) CleanRelationsByUserIDs(ctx context.Context, userIDs []uint32) error {
	if len(userIDs) == 0 {
		return nil
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("user_id IN ?", userIDs) },
	}); err != nil {
		r.log.Errorf("delete old user orgUnits by user ids failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("delete old user orgUnits by user ids failed")
	}
	return nil
}

// CleanRelationsByOrgUnitID 清理组织单元的用户关联
func (r *UserOrgUnitRepo) CleanRelationsByOrgUnitID(ctx context.Context, orgUnitID uint32) error {
	if orgUnitID == 0 {
		return nil
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("org_unit_id = ?", orgUnitID) },
	}); err != nil {
		r.log.Errorf("delete old user orgUnits by orgUnit id failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("delete old user orgUnits by orgUnit id failed")
	}
	return nil
}

// CleanRelationsByOrgUnitIDs 清理组织单元的用户关联
func (r *UserOrgUnitRepo) CleanRelationsByOrgUnitIDs(ctx context.Context, orgUnitIDs []uint32) error {
	if len(orgUnitIDs) == 0 {
		return nil
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("org_unit_id IN ?", orgUnitIDs) },
	}); err != nil {
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

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB {
			return db.Where("user_id = ?", userID).Where("org_unit_id IN ?", orgUnitIDs)
		},
	}); err != nil {
		r.log.Errorf("remove user orgUnits failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("remove user orgUnits failed")
	}
	return nil
}

// AssignUserOrgUnit 分配组织单元给用户
func (r *UserOrgUnitRepo) AssignUserOrgUnit(
	ctx context.Context,
	data *identityV1.UserOrgUnit,
) error {
	if data == nil {
		return nil
	}

	if _, err := r.repository.Create(ctx, r.client.DB, data, nil); err != nil {
		r.log.Errorf("assign orgUnit to user failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("assign orgUnit to user failed")
	}
	return nil
}

// AssignUserOrgUnits 分配组织单元给用户
func (r *UserOrgUnitRepo) AssignUserOrgUnits(
	ctx context.Context, userID uint32,
	datas []*identityV1.UserOrgUnit,
) error {
	if len(datas) == 0 || userID == 0 {
		return nil
	}

	// 删除该角色的所有旧关联
	if err := r.CleanRelationsByUserID(ctx, userID); err != nil {
		return identityV1.ErrorInternalServerError("clean old user orgUnits failed")
	}

	if _, err := r.repository.BatchCreate(ctx, r.client.DB, datas, nil); err != nil {
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

	db := r.client.DB.WithContext(ctx).Model(&models.UserOrgUnit{}).Where("user_id = ?", userID)

	if excludeExpired {
		now := time.Now()
		db = db.Where("end_at IS NULL OR end_at > ?", now)
	}

	var ids []uint32
	if err := db.Pluck("org_unit_id", &ids).Error; err != nil {
		r.log.Errorf("query orgUnit ids by user id failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query orgUnit ids by user id failed")
	}
	return ids, nil
}

// ListUserIDs 列出组织单元关联的用户ID列表
func (r *UserOrgUnitRepo) ListUserIDs(ctx context.Context, orgUnitID uint32, excludeExpired bool) ([]uint32, error) {
	if orgUnitID == 0 {
		return []uint32{}, nil
	}

	db := r.client.DB.WithContext(ctx).Model(&models.UserOrgUnit{}).Where("org_unit_id = ?", orgUnitID)

	if excludeExpired {
		now := time.Now()
		db = db.Where("end_at IS NULL OR end_at > ?", now)
	}

	var ids []uint32
	if err := db.Pluck("user_id", &ids).Error; err != nil {
		r.log.Errorf("query user ids by orgUnit id failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query user ids by orgUnit id failed")
	}
	return ids, nil
}

// ListUserIDsByOrgUnitIDs 列出多个组织单元关联的用户ID列表
func (r *UserOrgUnitRepo) ListUserIDsByOrgUnitIDs(ctx context.Context, orgUnitIDs []uint32, excludeExpired bool) ([]uint32, error) {
	if len(orgUnitIDs) == 0 {
		return nil, nil
	}

	db := r.client.DB.WithContext(ctx).Model(&models.UserOrgUnit{}).Where("org_unit_id IN ?", orgUnitIDs)

	if excludeExpired {
		now := time.Now()
		db = db.Where("end_at IS NULL OR end_at > ?", now)
	}

	var ids []uint32
	if err := db.Pluck("user_id", &ids).Error; err != nil {
		r.log.Errorf("query user ids by orgUnit ids failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query user ids by orgUnit ids failed")
	}
	return ids, nil
}
