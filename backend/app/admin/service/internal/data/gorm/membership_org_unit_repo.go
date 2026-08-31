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

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
)

type MembershipOrgUnitRepo struct {
	client          *gormCrud.Client
	log             *bLogger.Helper
	mapper          *mapper.CopierMapper[identityV1.MembershipOrgUnit, models.MembershipOrgUnit]
	repository      *gormCrud.Repository[identityV1.MembershipOrgUnit, models.MembershipOrgUnit]
	statusConverter *mapper.EnumTypeConverter[identityV1.MembershipOrgUnit_Status, string]
}

func NewMembershipOrgUnitRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *MembershipOrgUnitRepo {
	repo := &MembershipOrgUnitRepo{
		log:    ctx.NewLoggerHelper("membership-org-unit/gorm-repo/admin-service"),
		client: client,
		mapper: mapper.NewCopierMapper[identityV1.MembershipOrgUnit, models.MembershipOrgUnit](),
		statusConverter: mapper.NewEnumTypeConverter[identityV1.MembershipOrgUnit_Status, string](
			identityV1.MembershipOrgUnit_Status_name,
			identityV1.MembershipOrgUnit_Status_value,
		),
	}

	repo.init()

	return repo
}

func (r *MembershipOrgUnitRepo) init() {
	r.repository = gormCrud.NewRepository[identityV1.MembershipOrgUnit, models.MembershipOrgUnit](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
}

// CleanRelationsByMembershipID 清理会员组织单元关联
func (r *MembershipOrgUnitRepo) CleanRelationsByMembershipID(ctx context.Context, membershipID uint32) error {
	if membershipID == 0 {
		return nil
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("membership_id = ?", membershipID) },
	}); err != nil {
		r.log.Errorf(ctx, "delete old membership orgUnits failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("delete old membership orgUnits failed")
	}
	return nil
}

// CleanRelationsByMembershipIDs 清理多个会员组织单元关联
func (r *MembershipOrgUnitRepo) CleanRelationsByMembershipIDs(ctx context.Context, membershipIDs []uint32) error {
	if len(membershipIDs) == 0 {
		return nil
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("membership_id IN ?", membershipIDs) },
	}); err != nil {
		r.log.Errorf(ctx, "delete old membership orgUnits by membership ids failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("delete old membership orgUnits by membership ids failed")
	}
	return nil
}

// CleanRelationsByOrgUnitID 清理组织单元的会员关联
func (r *MembershipOrgUnitRepo) CleanRelationsByOrgUnitID(ctx context.Context, orgUnitID uint32) error {
	if orgUnitID == 0 {
		return nil
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("org_unit_id = ?", orgUnitID) },
	}); err != nil {
		r.log.Errorf(ctx, "delete old membership orgUnits by orgUnit id failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("delete old membership orgUnits by orgUnit id failed")
	}
	return nil
}

// CleanRelationsByOrgUnitIDs 清理多个组织单元的会员关联
func (r *MembershipOrgUnitRepo) CleanRelationsByOrgUnitIDs(ctx context.Context, orgUnitIDs []uint32) error {
	if len(orgUnitIDs) == 0 {
		return nil
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("org_unit_id IN ?", orgUnitIDs) },
	}); err != nil {
		r.log.Errorf(ctx, "delete old membership orgUnits by orgUnit ids failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("delete old membership orgUnits by orgUnit ids failed")
	}
	return nil
}

// RemoveOrgUnitsFromMembership 删除会员的组织单元关联
func (r *MembershipOrgUnitRepo) RemoveOrgUnitsFromMembership(ctx context.Context, membershipID uint32, ids []uint32) error {
	if membershipID == 0 || len(ids) == 0 {
		return nil
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB {
			return db.Where("membership_id = ?", membershipID).Where("org_unit_id IN ?", ids)
		},
	}); err != nil {
		r.log.Errorf(ctx, "remove orgUnits failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("remove orgUnits failed")
	}
	return nil
}

// AssignMembershipOrgUnits 分配组织单元给会员
func (r *MembershipOrgUnitRepo) AssignMembershipOrgUnits(
	ctx context.Context,
	membershipID uint32,
	datas []*identityV1.MembershipOrgUnit,
) error {
	if len(datas) == 0 || membershipID == 0 {
		return nil
	}

	// 删除该角色的所有旧关联
	if err := r.CleanRelationsByMembershipID(ctx, membershipID); err != nil {
		return identityV1.ErrorInternalServerError("clean old membership orgUnits failed")
	}

	if _, err := r.repository.BatchCreate(ctx, r.client.DB, datas, nil); err != nil {
		r.log.Errorf(ctx, "assign orgUnit to membership failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("assign orgUnit to membership failed")
	}

	return nil
}

// ListOrgUnitIDs 列出角色关联的组织单元ID列表
func (r *MembershipOrgUnitRepo) ListOrgUnitIDs(ctx context.Context, membershipID uint32, excludeExpired bool) ([]uint32, error) {
	if membershipID == 0 {
		return []uint32{}, nil
	}

	db := r.client.DB.WithContext(ctx).Model(&models.MembershipOrgUnit{}).Where("membership_id = ?", membershipID)

	if excludeExpired {
		now := time.Now()
		db = db.Where("end_at IS NULL OR end_at > ?", now)
	}

	var ids []uint32
	if err := db.Pluck("org_unit_id", &ids).Error; err != nil {
		r.log.Errorf(ctx, "query orgUnit ids by membership id failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query orgUnit ids by membership id failed")
	}
	return ids, nil
}

// ListMembershipIDs 获取组织单元关联的会员ID列表
func (r *MembershipOrgUnitRepo) ListMembershipIDs(ctx context.Context, orgUnitID uint32, excludeExpired bool) ([]uint32, error) {
	if orgUnitID == 0 {
		return []uint32{}, nil
	}

	db := r.client.DB.WithContext(ctx).Model(&models.MembershipOrgUnit{}).Where("org_unit_id = ?", orgUnitID)

	if excludeExpired {
		now := time.Now()
		db = db.Where("end_at IS NULL OR end_at > ?", now)
	}

	var ids []uint32
	if err := db.Pluck("membership_id", &ids).Error; err != nil {
		r.log.Errorf(ctx, "query membership ids by orgUnit id failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query membership ids by orgUnit id failed")
	}
	return ids, nil
}

// ListMembershipIDsByOrgUnitIDs 获取多个组织单元关联的会员ID列表
func (r *MembershipOrgUnitRepo) ListMembershipIDsByOrgUnitIDs(ctx context.Context, orgUnitIDs []uint32, excludeExpired bool) ([]uint32, error) {
	if len(orgUnitIDs) == 0 {
		return nil, nil
	}

	db := r.client.DB.WithContext(ctx).Model(&models.MembershipOrgUnit{}).Where("org_unit_id IN ?", orgUnitIDs)

	if excludeExpired {
		now := time.Now()
		db = db.Where("end_at IS NULL OR end_at > ?", now)
	}

	var ids []uint32
	if err := db.Pluck("membership_id", &ids).Error; err != nil {
		r.log.Errorf(ctx, "query membership ids by orgUnit ids failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query membership ids by orgUnit ids failed")
	}
	return ids, nil
}
