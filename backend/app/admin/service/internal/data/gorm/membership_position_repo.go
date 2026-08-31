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

type MembershipPositionRepo struct {
	client          *gormCrud.Client
	log             *bLogger.Helper
	mapper          *mapper.CopierMapper[identityV1.MembershipPosition, models.MembershipPosition]
	repository      *gormCrud.Repository[identityV1.MembershipPosition, models.MembershipPosition]
	statusConverter *mapper.EnumTypeConverter[identityV1.MembershipPosition_Status, string]
}

func NewMembershipPositionRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *MembershipPositionRepo {
	repo := &MembershipPositionRepo{
		log:    ctx.NewLoggerHelper("membership-position/gorm-repo/admin-service"),
		client: client,
		mapper: mapper.NewCopierMapper[identityV1.MembershipPosition, models.MembershipPosition](),
		statusConverter: mapper.NewEnumTypeConverter[identityV1.MembershipPosition_Status, string](
			identityV1.MembershipPosition_Status_name,
			identityV1.MembershipPosition_Status_value,
		),
	}

	repo.init()

	return repo
}

func (r *MembershipPositionRepo) init() {
	r.repository = gormCrud.NewRepository[identityV1.MembershipPosition, models.MembershipPosition](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
}

// CleanRelationsByMembershipID 删除会员在某租户下的所有职位关联
func (r *MembershipPositionRepo) CleanRelationsByMembershipID(ctx context.Context, membershipID uint32) error {
	if membershipID == 0 {
		return nil
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("membership_id = ?", membershipID) },
	}); err != nil {
		r.log.Errorf(ctx, "delete old membership positions failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("delete old membership positions failed")
	}
	return nil
}

// CleanRelationsByMembershipIDs 删除多个会员的所有职位关联
func (r *MembershipPositionRepo) CleanRelationsByMembershipIDs(ctx context.Context, membershipIDs []uint32) error {
	if len(membershipIDs) == 0 {
		return nil
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("membership_id IN ?", membershipIDs) },
	}); err != nil {
		r.log.Errorf(ctx, "delete old membership positions by membership ids failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("delete old membership positions by membership ids failed")
	}
	return nil
}

// CleanRelationsByPositionID 删除岗位的所有会员关联
func (r *MembershipPositionRepo) CleanRelationsByPositionID(ctx context.Context, positionID uint32) error {
	if positionID == 0 {
		return nil
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("position_id = ?", positionID) },
	}); err != nil {
		r.log.Errorf(ctx, "delete old membership positions by position id failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("delete old membership positions by position id failed")
	}
	return nil
}

// CleanRelationsByPositionIDs 删除多个岗位的所有会员关联
func (r *MembershipPositionRepo) CleanRelationsByPositionIDs(ctx context.Context, positionIDs []uint32) error {
	if len(positionIDs) == 0 {
		return nil
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("position_id IN ?", positionIDs) },
	}); err != nil {
		r.log.Errorf(ctx, "delete old membership positions by position ids failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("delete old membership positions by position ids failed")
	}
	return nil
}

// RemovePositionsFromMembership 从用户移除岗位
func (r *MembershipPositionRepo) RemovePositionsFromMembership(ctx context.Context, membershipID uint32, positionIDs []uint32) error {
	if membershipID == 0 || len(positionIDs) == 0 {
		return nil
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB {
			return db.Where("membership_id = ?", membershipID).Where("position_id IN ?", positionIDs)
		},
	}); err != nil {
		r.log.Errorf(ctx, "remove positions from membership failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("remove positions from membership failed")
	}
	return nil
}

// AssignMembershipPositions 分配岗位给用户
func (r *MembershipPositionRepo) AssignMembershipPositions(
	ctx context.Context,
	membershipID uint32,
	datas []*identityV1.MembershipPosition,
) error {
	if len(datas) == 0 || membershipID == 0 {
		return nil
	}

	// 删除该用户的所有旧关联
	if err := r.CleanRelationsByMembershipID(ctx, membershipID); err != nil {
		return identityV1.ErrorInternalServerError("clean old membership positions failed")
	}

	if _, err := r.repository.BatchCreate(ctx, r.client.DB, datas, nil); err != nil {
		r.log.Errorf(ctx, "assign positions to membership failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("assign positions to membership failed")
	}

	return nil
}

// ListPositionIDs 获取用户的岗位ID列表
func (r *MembershipPositionRepo) ListPositionIDs(ctx context.Context, membershipID uint32, excludeExpired bool) ([]uint32, error) {
	if membershipID == 0 {
		return []uint32{}, nil
	}

	db := r.client.DB.WithContext(ctx).Model(&models.MembershipPosition{}).Where("membership_id = ?", membershipID)

	if excludeExpired {
		now := time.Now()
		db = db.Where("end_at IS NULL OR end_at > ?", now)
	}

	var ids []uint32
	if err := db.Pluck("position_id", &ids).Error; err != nil {
		r.log.Errorf(ctx, "query position ids by membership id failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query position ids by membership id failed")
	}
	return ids, nil
}

// ListMembershipIDs 获取岗位关联的会员ID列表
func (r *MembershipPositionRepo) ListMembershipIDs(ctx context.Context, positionID uint32, excludeExpired bool) ([]uint32, error) {
	if positionID == 0 {
		return []uint32{}, nil
	}

	db := r.client.DB.WithContext(ctx).Model(&models.MembershipPosition{}).Where("position_id = ?", positionID)

	if excludeExpired {
		now := time.Now()
		db = db.Where("end_at IS NULL OR end_at > ?", now)
	}

	var ids []uint32
	if err := db.Pluck("membership_id", &ids).Error; err != nil {
		r.log.Errorf(ctx, "query membership ids by position id failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query membership ids by position id failed")
	}
	return ids, nil
}

// ListMembershipIDsByPositionIDs 获取多个岗位关联的会员ID列表
func (r *MembershipPositionRepo) ListMembershipIDsByPositionIDs(ctx context.Context, positionIDs []uint32, excludeExpired bool) ([]uint32, error) {
	if len(positionIDs) == 0 {
		return nil, nil
	}

	db := r.client.DB.WithContext(ctx).Model(&models.MembershipPosition{}).Where("position_id IN ?", positionIDs)

	if excludeExpired {
		now := time.Now()
		db = db.Where("end_at IS NULL OR end_at > ?", now)
	}

	var ids []uint32
	if err := db.Pluck("membership_id", &ids).Error; err != nil {
		r.log.Errorf(ctx, "query membership ids by position ids failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query membership ids by position ids failed")
	}
	return ids, nil
}
