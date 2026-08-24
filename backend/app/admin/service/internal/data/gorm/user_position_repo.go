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

type UserPositionRepo struct {
	client     *gormCrud.Client
	log        *log.Helper
	mapper     *mapper.CopierMapper[identityV1.UserPosition, models.UserPosition]
	repository *gormCrud.Repository[identityV1.UserPosition, models.UserPosition]
}

func NewUserPositionRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *UserPositionRepo {
	repo := &UserPositionRepo{
		log:    ctx.NewLoggerHelper("user-position/gorm-repo/admin-service"),
		client: client,
		mapper: mapper.NewCopierMapper[identityV1.UserPosition, models.UserPosition](),
	}

	repo.init()

	return repo
}

func (r *UserPositionRepo) init() {
	r.repository = gormCrud.NewRepository[identityV1.UserPosition, models.UserPosition](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

// CleanRelationsByUserID 删除用户的所有岗位关联
func (r *UserPositionRepo) CleanRelationsByUserID(ctx context.Context, userID uint32) error {
	if userID == 0 {
		return nil
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("user_id = ?", userID) },
	}); err != nil {
		r.log.Errorf("delete old user positions failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("delete old user positions failed")
	}
	return nil
}

// CleanRelationsByUserIDs 删除多个用户的所有岗位关联
func (r *UserPositionRepo) CleanRelationsByUserIDs(ctx context.Context, userIDs []uint32) error {
	if len(userIDs) == 0 {
		return nil
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("user_id IN ?", userIDs) },
	}); err != nil {
		r.log.Errorf("delete old user positions by user ids failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("delete old user positions by user ids failed")
	}
	return nil
}

// CleanRelationsByPositionID 删除岗位的所有用户关联
func (r *UserPositionRepo) CleanRelationsByPositionID(ctx context.Context, positionID uint32) error {
	if positionID == 0 {
		return nil
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("position_id = ?", positionID) },
	}); err != nil {
		r.log.Errorf("delete old user positions by position id failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("delete old user positions by position id failed")
	}
	return nil
}

// CleanRelationsByPositionIDs 删除多个岗位的所有用户关联
func (r *UserPositionRepo) CleanRelationsByPositionIDs(ctx context.Context, positionIDs []uint32) error {
	if len(positionIDs) == 0 {
		return nil
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("position_id IN ?", positionIDs) },
	}); err != nil {
		r.log.Errorf("delete old user positions by position ids failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("delete old user positions by position ids failed")
	}
	return nil
}

// RemovePositionsFromUser 从用户移除岗位
func (r *UserPositionRepo) RemovePositionsFromUser(ctx context.Context, userID uint32, positionIDs []uint32) error {
	if len(positionIDs) == 0 || userID == 0 {
		return nil
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB {
			return db.Where("user_id = ?", userID).Where("position_id IN ?", positionIDs)
		},
	}); err != nil {
		r.log.Errorf("remove positions from user failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("remove positions from user failed")
	}
	return nil
}

func (r *UserPositionRepo) AssignUserPosition(
	ctx context.Context,
	data *identityV1.UserPosition,
) error {
	if data == nil {
		return nil
	}

	if _, err := r.repository.Create(ctx, r.client.DB, data, nil); err != nil {
		r.log.Errorf("assign position to user failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("assign position to user failed")
	}

	return nil
}

// AssignUserPositions 分配岗位给用户
func (r *UserPositionRepo) AssignUserPositions(
	ctx context.Context, userID uint32,
	datas []*identityV1.UserPosition,
) error {
	if len(datas) == 0 || userID == 0 {
		return nil
	}

	// 删除该用户的所有旧关联
	if err := r.CleanRelationsByUserID(ctx, userID); err != nil {
		return identityV1.ErrorInternalServerError("clean old user positions failed")
	}

	if _, err := r.repository.BatchCreate(ctx, r.client.DB, datas, nil); err != nil {
		r.log.Errorf("assign positions to user failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("assign positions to user failed")
	}

	return nil
}

// ListPositionIDs 获取用户的岗位ID列表
func (r *UserPositionRepo) ListPositionIDs(ctx context.Context, userID uint32, excludeExpired bool) ([]uint32, error) {
	if userID == 0 {
		return []uint32{}, nil
	}

	db := r.client.DB.WithContext(ctx).Model(&models.UserPosition{}).Where("user_id = ?", userID)

	if excludeExpired {
		now := time.Now()
		db = db.Where("end_at IS NULL OR end_at > ?", now)
	}

	var ids []uint32
	if err := db.Pluck("position_id", &ids).Error; err != nil {
		r.log.Errorf("query position ids by user id failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query position ids by user id failed")
	}
	return ids, nil
}

// ListUserIDs 获取岗位关联的用户ID列表
func (r *UserPositionRepo) ListUserIDs(ctx context.Context, positionID uint32, excludeExpired bool) ([]uint32, error) {
	if positionID == 0 {
		return []uint32{}, nil
	}

	db := r.client.DB.WithContext(ctx).Model(&models.UserPosition{}).Where("position_id = ?", positionID)

	if excludeExpired {
		now := time.Now()
		db = db.Where("end_at IS NULL OR end_at > ?", now)
	}

	var ids []uint32
	if err := db.Pluck("user_id", &ids).Error; err != nil {
		r.log.Errorf("query user ids by position id failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query user ids by position id failed")
	}
	return ids, nil
}

// ListUserIDsByPositionIDs 获取多个岗位关联的用户ID列表
func (r *UserPositionRepo) ListUserIDsByPositionIDs(ctx context.Context, positionIDs []uint32, excludeExpired bool) ([]uint32, error) {
	if len(positionIDs) == 0 {
		return nil, nil
	}

	db := r.client.DB.WithContext(ctx).Model(&models.UserPosition{}).Where("position_id IN ?", positionIDs)

	if excludeExpired {
		now := time.Now()
		db = db.Where("end_at IS NULL OR end_at > ?", now)
	}

	var ids []uint32
	if err := db.Pluck("user_id", &ids).Error; err != nil {
		r.log.Errorf("query user ids by position ids failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query user ids by position ids failed")
	}
	return ids, nil
}
