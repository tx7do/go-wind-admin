//go:build gorm_backend
// +build gorm_backend

package gorm

import (
	"context"

	gormDB "gorm.io/gorm"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	gormCrud "github.com/tx7do/go-crud/gorm"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/admin/service/internal/data/gorm/models"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

type PermissionMenuRepo struct {
	client     *gormCrud.Client
	log        *log.Helper
	mapper     *mapper.CopierMapper[permissionV1.PermissionMenu, models.PermissionMenu]
	repository *gormCrud.Repository[permissionV1.PermissionMenu, models.PermissionMenu]
}

func NewPermissionMenuRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *PermissionMenuRepo {
	repo := &PermissionMenuRepo{
		log:    ctx.NewLoggerHelper("permission-menu/gorm-repo/admin-service"),
		client: client,
		mapper: mapper.NewCopierMapper[permissionV1.PermissionMenu, models.PermissionMenu](),
	}

	repo.init()

	return repo
}

func (r *PermissionMenuRepo) init() {
	r.repository = gormCrud.NewRepository[permissionV1.PermissionMenu, models.PermissionMenu](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

// CleanMenus 清理权限的所有菜单
func (r *PermissionMenuRepo) CleanMenus(
	ctx context.Context,
	permissionIDs []uint32,
) error {
	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("permission_id IN ?", permissionIDs) },
	}); err != nil {
		r.log.Errorf("delete old permission menus failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete old permission menus failed")
	}
	return nil
}

// CleanNotExistMenus 清理权限中不存在的菜单
func (r *PermissionMenuRepo) CleanNotExistMenus(
	ctx context.Context,
	permissionID uint32,
	menuIDs []uint32,
) error {
	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB {
			return db.Where("menu_id NOT IN ?", menuIDs).Where("permission_id = ?", permissionID)
		},
	}); err != nil {
		r.log.Errorf("clean not exists permission menus failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("clean not exists permission menus failed")
	}
	return nil
}

// AssignMenus 给权限分配菜单
func (r *PermissionMenuRepo) AssignMenus(ctx context.Context, permissionID uint32, menuIDs []uint32) error {
	if err := r.CleanNotExistMenus(ctx, permissionID, menuIDs); err != nil {
		return err
	}

	return r.AssignMenusWithTx(ctx, permissionID, menuIDs)
}

// AssignMenusWithTx 给权限分配菜单
func (r *PermissionMenuRepo) AssignMenusWithTx(ctx context.Context, permissionID uint32, menuIDs []uint32) error {
	if len(menuIDs) == 0 {
		return nil
	}

	datas := make([]*permissionV1.PermissionMenu, 0, len(menuIDs))
	for _, menu := range menuIDs {
		menuID := menu
		permID := permissionID
		dto := &permissionV1.PermissionMenu{}
		dto.MenuId = &menuID
		dto.PermissionId = &permID
		datas = append(datas, dto)
	}

	if _, err := r.repository.BatchCreate(ctx, r.client.DB, datas, nil); err != nil {
		r.log.Errorf("assign permission menuIDs failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("assign permission menuIDs failed")
	}

	return nil
}

// ListMenuIDs 列出权限关联的菜单ID列表
func (r *PermissionMenuRepo) ListMenuIDs(ctx context.Context, permissionIDs []uint32) ([]uint32, error) {
	var ids []uint32
	if err := r.client.DB.WithContext(ctx).
		Model(&models.PermissionMenu{}).
		Where("permission_id IN ?", permissionIDs).
		Pluck("menu_id", &ids).Error; err != nil {
		r.log.Errorf("list permission menus by permission id failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("list permission menus by permission id failed")
	}

	return ids, nil
}

// Truncate 清空表数据
func (r *PermissionMenuRepo) Truncate(ctx context.Context) error {
	if err := r.client.DB.WithContext(ctx).Where("1 = 1").Delete(&models.PermissionMenu{}).Error; err != nil {
		r.log.Errorf("failed to truncate permission menu table: %s", err.Error())
		return permissionV1.ErrorInternalServerError("truncate failed")
	}

	return nil
}

// Delete 删除权限关联的菜单
func (r *PermissionMenuRepo) Delete(ctx context.Context, permissionID uint32) error {
	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("permission_id = ?", permissionID) },
	}); err != nil {
		r.log.Errorf("failed to delete permission menu by permission id: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete failed")
	}

	return nil
}

func (r *PermissionMenuRepo) DeleteByPermissionIDs(ctx context.Context, permissionIDs []uint32) error {
	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("permission_id IN ?", permissionIDs) },
	}); err != nil {
		r.log.Errorf("delete permission menus by permission ids failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete permission menus by permission ids failed")
	}
	return nil
}

// AssignMenu 给权限分配菜单
func (r *PermissionMenuRepo) AssignMenu(ctx context.Context, permissionID uint32, menuID uint32) error {
	menu := menuID
	permID := permissionID
	dto := &permissionV1.PermissionMenu{}
	dto.MenuId = &menu
	dto.PermissionId = &permID

	if _, err := r.repository.Create(ctx, r.client.DB, dto, nil); err != nil {
		return permissionV1.ErrorInternalServerError("assign permission menu failed")
	}

	return nil
}
