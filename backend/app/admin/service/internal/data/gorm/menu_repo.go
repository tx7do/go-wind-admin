//go:build gorm_backend
// +build gorm_backend

package gorm

import (
	"context"
	"errors"

	gormDB "gorm.io/gorm"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	gormCrud "github.com/tx7do/go-crud/gorm"
	pagination "github.com/tx7do/go-crud/pagination"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/admin/service/internal/data/gorm/models"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

type MenuRepo struct {
	client          *gormCrud.Client
	log             *log.Helper
	mapper          *mapper.CopierMapper[permissionV1.Menu, models.Menu]
	statusConverter *mapper.EnumTypeConverter[permissionV1.Menu_Status, string]
	typeConverter   *mapper.EnumTypeConverter[permissionV1.Menu_Type, string]
	moduleConverter *mapper.EnumTypeConverter[identityV1.Module, string]

	repository *gormCrud.Repository[permissionV1.Menu, models.Menu]
}

func NewMenuRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *MenuRepo {
	repo := &MenuRepo{
		log:             ctx.NewLoggerHelper("menu/gorm-repo/admin-service"),
		client:          client,
		mapper:          mapper.NewCopierMapper[permissionV1.Menu, models.Menu](),
		statusConverter: mapper.NewEnumTypeConverter[permissionV1.Menu_Status, string](permissionV1.Menu_Status_name, permissionV1.Menu_Status_value),
		typeConverter:   mapper.NewEnumTypeConverter[permissionV1.Menu_Type, string](permissionV1.Menu_Type_name, permissionV1.Menu_Type_value),
		moduleConverter: mapper.NewEnumTypeConverter[identityV1.Module, string](identityV1.Module_name, identityV1.Module_value),
	}

	repo.init()

	return repo
}

func (r *MenuRepo) init() {
	r.repository = gormCrud.NewRepository[permissionV1.Menu, models.Menu](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
	r.mapper.AppendConverters(r.typeConverter.NewConverterPair())
	r.mapper.AppendConverters(r.moduleConverter.NewConverterPair())
}

func (r *MenuRepo) Count(ctx context.Context, scopes []func(*gormDB.DB) *gormDB.DB) (int, error) {
	count, err := r.repository.Count(ctx, r.client.DB, scopes)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, permissionV1.ErrorInternalServerError("query count failed")
	}

	return int(count), nil
}

func (r *MenuRepo) List(ctx context.Context, req *paginationV1.PagingRequest, treeTravel bool) (*permissionV1.ListMenuResponse, error) {
	if req == nil {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	var entities []*models.Menu
	if err := r.client.DB.WithContext(ctx).Model(&models.Menu{}).Find(&entities).Error; err != nil {
		r.log.Errorf("query menu list failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query menu list failed")
	}

	// 转换所有实体为 DTO
	dtos := make([]*permissionV1.Menu, 0, len(entities))
	for _, entity := range entities {
		dto := r.mapper.ToDTO(entity)
		dtos = append(dtos, dto)
	}

	// 构建树形结构
	if treeTravel {
		dtos = pagination.BuildTree(
			dtos,
			func(node *permissionV1.Menu) *uint32 { return node.Id },
			func(node *permissionV1.Menu) *uint32 { return node.ParentId },
			func(node *permissionV1.Menu) *[]*permissionV1.Menu { return &node.Children },
		)
	}

	count, err := r.Count(ctx, nil)
	if err != nil {
		return nil, err
	}

	return &permissionV1.ListMenuResponse{
		Total: uint64(count),
		Items: dtos,
	}, nil
}

func (r *MenuRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.repository.ExistsWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", id) },
	})
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, permissionV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *MenuRepo) Get(ctx context.Context, req *permissionV1.GetMenuRequest) (*permissionV1.Menu, error) {
	if req == nil {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	var scopes []func(*gormDB.DB) *gormDB.DB
	switch req.QueryBy.(type) {
	default:
	case *permissionV1.GetMenuRequest_Id:
		scopes = append(scopes, func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) })
	}

	dto, err := r.repository.GetWithFilters(ctx, r.client.DB, scopes, req.GetViewMask())
	if err != nil {
		if errors.Is(err, gormDB.ErrRecordNotFound) {
			return nil, permissionV1.ErrorNotFound("menu not found")
		}
		r.log.Errorf("query menu failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query menu failed")
	}

	return dto, nil
}

func (r *MenuRepo) Create(ctx context.Context, req *permissionV1.CreateMenuRequest) error {
	if req == nil || req.Data == nil {
		return permissionV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := r.repository.Create(ctx, r.client.DB, req.Data, nil); err != nil {
		r.log.Errorf("insert menu failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("insert menu failed")
	}

	return nil
}

// CreateReturn 创建菜单并返回包含数据库生成 ID 的实体
// CreateReturn creates a menu and returns the entity with the database-generated ID
func (r *MenuRepo) CreateReturn(ctx context.Context, req *permissionV1.CreateMenuRequest) (*permissionV1.Menu, error) {
	if req == nil || req.Data == nil {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	dto, err := r.repository.Create(ctx, r.client.DB, req.Data, nil)
	if err != nil {
		r.log.Errorf("insert menu failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("insert menu failed")
	}

	return dto, nil
}

func (r *MenuRepo) Update(ctx context.Context, req *permissionV1.UpdateMenuRequest) error {
	if req == nil || req.Data == nil {
		return permissionV1.ErrorBadRequest("invalid parameter")
	}
	if req.GetId() == 0 {
		return permissionV1.ErrorBadRequest("id is required")
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return err
		}
		if !exist {
			createReq := &permissionV1.CreateMenuRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	if _, err := r.repository.UpdateWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) },
	}, req.Data, req.GetUpdateMask()); err != nil {
		r.log.Errorf("update menu failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("update menu failed")
	}

	return nil
}

// Truncate 清空菜单表数据
// Clear all menu data from the table
func (r *MenuRepo) Truncate(ctx context.Context) error {
	if err := r.client.DB.WithContext(ctx).Where("1 = 1").Delete(&models.Menu{}).Error; err != nil {
		r.log.Errorf("failed to truncate menus table: %s", err.Error())
		return permissionV1.ErrorInternalServerError("truncate menus failed")
	}
	return nil
}

func (r *MenuRepo) Delete(ctx context.Context, req *permissionV1.DeleteMenuRequest) error {
	return permissionV1.ErrorInternalServerError("gorm scaffold: Delete not implemented — ent QueryAllChildrenIds tree-cascade has no go-crud/gorm primitive; see data/menu_repo.go")
}
