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

type PermissionApiRepo struct {
	client     *gormCrud.Client
	log        *bLogger.Helper
	mapper     *mapper.CopierMapper[permissionV1.PermissionApi, models.PermissionApi]
	repository *gormCrud.Repository[permissionV1.PermissionApi, models.PermissionApi]
}

func NewPermissionApiRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *PermissionApiRepo {
	repo := &PermissionApiRepo{
		log:    ctx.NewLoggerHelper("permission-api/gorm-repo/admin-service"),
		client: client,
		mapper: mapper.NewCopierMapper[permissionV1.PermissionApi, models.PermissionApi](),
	}

	repo.init()

	return repo
}

func (r *PermissionApiRepo) init() {
	r.repository = gormCrud.NewRepository[permissionV1.PermissionApi, models.PermissionApi](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

// CleanApis 清理权限的所有API资源
func (r *PermissionApiRepo) CleanApis(
	ctx context.Context,
	permissionIDs []uint32,
) error {
	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("permission_id IN ?", permissionIDs) },
	}); err != nil {
		r.log.Errorf(ctx, "delete old permission apis failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete old permission apis failed")
	}
	return nil
}

// CleanNotExistApis 清理权限中不存在的API资源
func (r *PermissionApiRepo) CleanNotExistApis(
	ctx context.Context,
	permissionID uint32,
	apiIDs []uint32,
) error {
	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB {
			return db.Where("api_id NOT IN ?", apiIDs).Where("permission_id = ?", permissionID)
		},
	}); err != nil {
		r.log.Errorf(ctx, "clean not exists permission apis failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("clean not exists permission apis failed")
	}
	return nil
}

// AssignApis 给权限分配API资源
func (r *PermissionApiRepo) AssignApis(
	ctx context.Context,
	permissionID uint32,
	apiIDs []uint32,
) error {
	if err := r.CleanNotExistApis(ctx, permissionID, apiIDs); err != nil {
		return err
	}

	return r.AssignApisWithTx(ctx, permissionID, apiIDs)
}

// AssignApisWithTx 给权限分配API资源
func (r *PermissionApiRepo) AssignApisWithTx(
	ctx context.Context,
	permissionID uint32,
	apis []uint32,
) error {
	if len(apis) == 0 {
		return nil
	}

	datas := make([]*permissionV1.PermissionApi, 0, len(apis))
	for _, api := range apis {
		apiID := api
		permID := permissionID
		dto := &permissionV1.PermissionApi{}
		dto.ApiId = &apiID
		dto.PermissionId = &permID
		datas = append(datas, dto)
	}

	if _, err := r.repository.BatchCreate(ctx, r.client.DB, datas, nil); err != nil {
		r.log.Errorf(ctx, "assign permission apis failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("assign permission apis failed")
	}

	return nil
}

// ListApiIDs 列出权限关联的API资源ID列表
func (r *PermissionApiRepo) ListApiIDs(ctx context.Context, permissionIDs []uint32) ([]uint32, error) {
	var ids []uint32
	if err := r.client.DB.WithContext(ctx).
		Model(&models.PermissionApi{}).
		Where("permission_id IN ?", permissionIDs).
		Pluck("api_id", &ids).Error; err != nil {
		r.log.Errorf(ctx, "list permission apis by permission id failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("list permission apis by permission id failed")
	}

	return ids, nil
}

// Truncate 清空表数据
func (r *PermissionApiRepo) Truncate(ctx context.Context) error {
	if err := r.client.DB.WithContext(ctx).Where("1 = 1").Delete(&models.PermissionApi{}).Error; err != nil {
		r.log.Errorf(ctx, "failed to truncate permission api table: %s", err.Error())
		return permissionV1.ErrorInternalServerError("truncate failed")
	}

	return nil
}

// Delete 删除权限关联的API资源
func (r *PermissionApiRepo) Delete(ctx context.Context, permissionID uint32) error {
	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("permission_id = ?", permissionID) },
	}); err != nil {
		r.log.Errorf(ctx, "delete permission apis by permission id failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete permission apis by permission id failed")
	}
	return nil
}

func (r *PermissionApiRepo) DeleteByPermissionIDs(ctx context.Context, permissionIDs []uint32) error {
	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("permission_id IN ?", permissionIDs) },
	}); err != nil {
		r.log.Errorf(ctx, "delete permission apis by permission ids failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete permission apis by permission ids failed")
	}
	return nil
}

// AssignApi 给权限分配API资源
func (r *PermissionApiRepo) AssignApi(ctx context.Context, permissionID uint32, apiID uint32) error {
	api := apiID
	permID := permissionID
	dto := &permissionV1.PermissionApi{}
	dto.ApiId = &api
	dto.PermissionId = &permID

	if _, err := r.repository.Create(ctx, r.client.DB, dto, nil); err != nil {
		return permissionV1.ErrorInternalServerError("assign permission api failed")
	}

	return nil
}
