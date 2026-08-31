//go:build gorm_backend
// +build gorm_backend

// Package gorm 中的仓储是 ent 仓储的平行 gorm 镜像，作为"ent 为主力、gorm 为备选"脚手架的完整代码。
// 这些仓储为死代码：未接入 cmd/server/wiring.go、不被 service 引用；采用者需要时自行装配。
//
// gorm 仓储不做租户隔离（ent 侧靠编译进生成代码的 privacy 策略自动注入，gorm 侧无此机制）。
// 直接切换 gorm 后端会有跨租户数据泄露风险，采用者须自行加 scope/plugin。
package gorm

import (
	"context"
	"errors"

	gormDB "gorm.io/gorm"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	gormCrud "github.com/tx7do/go-crud/gorm"
	gormCrudFilter "github.com/tx7do/go-crud/gorm/filter"
	paginationFilter "github.com/tx7do/go-crud/pagination/filter"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/admin/service/internal/data/gorm/models"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

type ApiRepo struct {
	client           *gormCrud.Client
	log              *log.Helper
	mapper           *mapper.CopierMapper[permissionV1.Api, models.Api]
	repository       *gormCrud.Repository[permissionV1.Api, models.Api]
	structuredFilter *gormCrudFilter.StructuredFilter

	scopeConverter          *mapper.EnumTypeConverter[permissionV1.Api_Scope, string]
	businessModuleConverter *mapper.EnumTypeConverter[identityV1.Module, string]
}

func NewApiRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *ApiRepo {
	repo := &ApiRepo{
		log:              ctx.NewLoggerHelper("api/gorm-repo/admin-service"),
		client:           client,
		mapper:           mapper.NewCopierMapper[permissionV1.Api, models.Api](),
		structuredFilter: gormCrudFilter.NewStructuredFilter(),

		scopeConverter:          mapper.NewEnumTypeConverter[permissionV1.Api_Scope, string](permissionV1.Api_Scope_name, permissionV1.Api_Scope_value),
		businessModuleConverter: mapper.NewEnumTypeConverter[identityV1.Module, string](identityV1.Module_name, identityV1.Module_value),
	}

	repo.init()

	return repo
}

func (r *ApiRepo) init() {
	r.repository = gormCrud.NewRepository[permissionV1.Api, models.Api](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.scopeConverter.NewConverterPair())
	r.mapper.AppendConverters(r.businessModuleConverter.NewConverterPair())
}

func (r *ApiRepo) Count(ctx context.Context, req *paginationV1.PagingRequest) (*permissionV1.CountApiResponse, error) {
	if req == nil {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	filterExpr, err := paginationFilter.ConvertFilterByPagingRequest(req)
	if err != nil {
		r.log.Errorf("convert filter failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query api count failed")
	}

	scopes, err := r.structuredFilter.BuildSelectors(filterExpr)
	if err != nil {
		r.log.Errorf("build selectors failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query api count failed")
	}

	count, err := r.repository.Count(ctx, r.client.DB, scopes)
	if err != nil {
		r.log.Errorf("query api count failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query api count failed")
	}

	return &permissionV1.CountApiResponse{
		Count: uint64(count),
	}, nil
}

func (r *ApiRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*permissionV1.ListApiResponse, error) {
	if req == nil {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	ret, err := r.repository.ListWithPaging(ctx, r.client.DB, req)
	if err != nil {
		return nil, permissionV1.ErrorInternalServerError("query list failed")
	}
	if ret == nil {
		return &permissionV1.ListApiResponse{Total: 0, Items: nil}, nil
	}

	return &permissionV1.ListApiResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *ApiRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.repository.ExistsWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", id) },
	})
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, permissionV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *ApiRepo) Get(ctx context.Context, req *permissionV1.GetApiRequest) (*permissionV1.Api, error) {
	if req == nil {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	var scopes []func(*gormDB.DB) *gormDB.DB
	switch req.QueryBy.(type) {
	default:
	case *permissionV1.GetApiRequest_Id:
		scopes = append(scopes, func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) })
	}

	dto, err := r.repository.GetWithFilters(ctx, r.client.DB, scopes, req.GetViewMask())
	if err != nil {
		if errors.Is(err, gormDB.ErrRecordNotFound) {
			return nil, permissionV1.ErrorNotFound("api not found")
		}
		r.log.Errorf("query api failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query api failed")
	}

	return dto, nil
}

// GetApiByEndpoint 根据路径和方法获取API资源
func (r *ApiRepo) GetApiByEndpoint(ctx context.Context, path, method string) (*permissionV1.Api, error) {
	if path == "" || method == "" {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	var entity models.Api
	if err := r.client.DB.WithContext(ctx).Model(&models.Api{}).Where("path = ? AND method = ?", path, method).First(&entity).Error; err != nil {
		if errors.Is(err, gormDB.ErrRecordNotFound) {
			return nil, permissionV1.ErrorNotFound("api not found")
		}
		r.log.Errorf("query data failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query data failed")
	}

	return r.mapper.ToDTO(&entity), nil
}

// GetApiByIDs 根据ID列表获取API资源
func (r *ApiRepo) GetApiByIDs(ctx context.Context, ids []uint32) ([]*permissionV1.Api, error) {
	if len(ids) == 0 {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	var entities []*models.Api
	if err := r.client.DB.WithContext(ctx).Model(&models.Api{}).Where("id IN ?", ids).Find(&entities).Error; err != nil {
		r.log.Errorf("query data failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query data failed")
	}

	dtos := make([]*permissionV1.Api, 0, len(entities))
	for _, entity := range entities {
		dtos = append(dtos, r.mapper.ToDTO(entity))
	}

	return dtos, nil
}

func (r *ApiRepo) Create(ctx context.Context, req *permissionV1.CreateApiRequest) error {
	if req == nil || req.Data == nil {
		return permissionV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := r.repository.Create(ctx, r.client.DB, req.Data, nil); err != nil {
		r.log.Errorf("insert api failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("insert api failed")
	}

	return nil
}

func (r *ApiRepo) BatchCreate(ctx context.Context, apis []*permissionV1.Api) error {
	if len(apis) == 0 {
		return nil
	}

	if _, err := r.repository.BatchCreate(ctx, r.client.DB, apis, nil); err != nil {
		r.log.Errorf("batch insert apis failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("batch insert apis failed")
	}

	return nil
}

func (r *ApiRepo) Update(ctx context.Context, req *permissionV1.UpdateApiRequest) error {
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
			createReq := &permissionV1.CreateApiRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	if _, err := r.repository.UpdateWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) },
	}, req.Data, req.GetUpdateMask()); err != nil {
		r.log.Errorf("update api failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("update api failed")
	}

	return nil
}

func (r *ApiRepo) Delete(ctx context.Context, req *permissionV1.DeleteApiRequest) error {
	if req == nil {
		return permissionV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) },
	}); err != nil {
		r.log.Errorf("delete api failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete api failed")
	}

	return nil
}

// Truncate 清空表数据
func (r *ApiRepo) Truncate(ctx context.Context) error {
	if err := r.client.DB.WithContext(ctx).Where("1 = 1").Delete(&models.Api{}).Error; err != nil {
		r.log.Errorf("failed to truncate apis table: %s", err.Error())
		return permissionV1.ErrorInternalServerError("truncate failed")
	}
	return nil
}
