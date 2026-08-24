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
)

type OrgUnitRepo struct {
	client          *gormCrud.Client
	log             *log.Helper
	mapper          *mapper.CopierMapper[identityV1.OrgUnit, models.OrgUnit]
	typeConverter   *mapper.EnumTypeConverter[identityV1.OrgUnit_Type, string]
	statusConverter *mapper.EnumTypeConverter[identityV1.OrgUnit_Status, string]

	repository *gormCrud.Repository[identityV1.OrgUnit, models.OrgUnit]
}

func NewOrgUnitRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *OrgUnitRepo {
	repo := &OrgUnitRepo{
		log:             ctx.NewLoggerHelper("org-unit/gorm-repo/admin-service"),
		client:          client,
		mapper:          mapper.NewCopierMapper[identityV1.OrgUnit, models.OrgUnit](),
		typeConverter:   mapper.NewEnumTypeConverter[identityV1.OrgUnit_Type, string](identityV1.OrgUnit_Type_name, identityV1.OrgUnit_Type_value),
		statusConverter: mapper.NewEnumTypeConverter[identityV1.OrgUnit_Status, string](identityV1.OrgUnit_Status_name, identityV1.OrgUnit_Status_value),
	}

	repo.init()

	return repo
}

func (r *OrgUnitRepo) init() {
	r.repository = gormCrud.NewRepository[identityV1.OrgUnit, models.OrgUnit](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.typeConverter.NewConverterPair())
	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
}

func (r *OrgUnitRepo) Count(ctx context.Context, scopes []func(*gormDB.DB) *gormDB.DB) (int, error) {
	count, err := r.repository.Count(ctx, r.client.DB, scopes)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, identityV1.ErrorInternalServerError("query count failed")
	}

	return int(count), nil
}

func (r *OrgUnitRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*identityV1.ListOrgUnitResponse, error) {
	if req == nil {
		return nil, identityV1.ErrorBadRequest("invalid parameter")
	}

	var entities []*models.OrgUnit
	if err := r.client.DB.WithContext(ctx).Model(&models.OrgUnit{}).Find(&entities).Error; err != nil {
		r.log.Errorf("query org unit list failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query org unit list failed")
	}

	// 转换所有实体为 DTO
	dtos := make([]*identityV1.OrgUnit, 0, len(entities))
	for _, entity := range entities {
		dto := r.mapper.ToDTO(entity)
		dtos = append(dtos, dto)
	}

	// 构建树形结构
	dtos = pagination.BuildTree(
		dtos,
		func(node *identityV1.OrgUnit) *uint32 { return node.Id },
		func(node *identityV1.OrgUnit) *uint32 { return node.ParentId },
		func(node *identityV1.OrgUnit) *[]*identityV1.OrgUnit { return &node.Children },
	)

	count, err := r.Count(ctx, nil)
	if err != nil {
		return nil, err
	}

	return &identityV1.ListOrgUnitResponse{
		Total: uint64(count),
		Items: dtos,
	}, err
}

func (r *OrgUnitRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.repository.ExistsWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", id) },
	})
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, identityV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *OrgUnitRepo) Get(ctx context.Context, req *identityV1.GetOrgUnitRequest) (*identityV1.OrgUnit, error) {
	if req == nil {
		return nil, identityV1.ErrorBadRequest("invalid parameter")
	}

	var scopes []func(*gormDB.DB) *gormDB.DB
	switch req.QueryBy.(type) {
	default:
	case *identityV1.GetOrgUnitRequest_Id:
		scopes = append(scopes, func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) })
	}

	dto, err := r.repository.GetWithFilters(ctx, r.client.DB, scopes, req.GetViewMask())
	if err != nil {
		if errors.Is(err, gormDB.ErrRecordNotFound) {
			return nil, identityV1.ErrorNotFound("org unit not found")
		}
		r.log.Errorf("query org unit failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query org unit failed")
	}

	return dto, nil
}

// ListOrgUnitsByIds 通过多个ID获取组织列表
func (r *OrgUnitRepo) ListOrgUnitsByIds(ctx context.Context, ids []uint32) ([]*identityV1.OrgUnit, error) {
	if len(ids) == 0 {
		return []*identityV1.OrgUnit{}, nil
	}

	var entities []*models.OrgUnit
	if err := r.client.DB.WithContext(ctx).Model(&models.OrgUnit{}).Where("id IN ?", ids).Find(&entities).Error; err != nil {
		r.log.Errorf("query orgUnit by ids failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query orgUnit by ids failed")
	}

	dtos := make([]*identityV1.OrgUnit, 0, len(entities))
	for _, entity := range entities {
		dtos = append(dtos, r.mapper.ToDTO(entity))
	}

	return dtos, nil
}

func (r *OrgUnitRepo) Create(ctx context.Context, req *identityV1.CreateOrgUnitRequest) error {
	return identityV1.ErrorInternalServerError("gorm scaffold: Create not implemented — ent setTreePath materialized-path write has no go-crud/gorm primitive; see data/org_unit_repo.go")
}

func (r *OrgUnitRepo) Update(ctx context.Context, req *identityV1.UpdateOrgUnitRequest) error {
	return identityV1.ErrorInternalServerError("gorm scaffold: Update not implemented — ent setTreePath materialized-path write has no go-crud/gorm primitive; see data/org_unit_repo.go")
}

func (r *OrgUnitRepo) Delete(ctx context.Context, req *identityV1.DeleteOrgUnitRequest) error {
	return identityV1.ErrorInternalServerError("gorm scaffold: Delete not implemented — ent QueryAllChildrenIds tree-cascade has no go-crud/gorm primitive; see data/org_unit_repo.go")
}
