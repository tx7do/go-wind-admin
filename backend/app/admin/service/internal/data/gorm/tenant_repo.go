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
	gormCrudFilter "github.com/tx7do/go-crud/gorm/filter"
	paginationFilter "github.com/tx7do/go-crud/pagination/filter"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/admin/service/internal/data/gorm/models"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
)

type TenantRepo struct {
	client                *gormCrud.Client
	log                   *log.Helper
	mapper                *mapper.CopierMapper[identityV1.Tenant, models.Tenant]
	repository            *gormCrud.Repository[identityV1.Tenant, models.Tenant]
	structuredFilter      *gormCrudFilter.StructuredFilter
	statusConverter       *mapper.EnumTypeConverter[identityV1.Tenant_Status, string]
	typeConverter         *mapper.EnumTypeConverter[identityV1.Tenant_Type, string]
	auditStatusConverter  *mapper.EnumTypeConverter[identityV1.Tenant_AuditStatus, string]
}

func NewTenantRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *TenantRepo {
	repo := &TenantRepo{
		log:                   ctx.NewLoggerHelper("tenant/gorm-repo/admin-service"),
		client:                client,
		mapper:                mapper.NewCopierMapper[identityV1.Tenant, models.Tenant](),
		structuredFilter:      gormCrudFilter.NewStructuredFilter(),
		statusConverter:       mapper.NewEnumTypeConverter[identityV1.Tenant_Status, string](identityV1.Tenant_Status_name, identityV1.Tenant_Status_value),
		typeConverter:         mapper.NewEnumTypeConverter[identityV1.Tenant_Type, string](identityV1.Tenant_Type_name, identityV1.Tenant_Type_value),
		auditStatusConverter:  mapper.NewEnumTypeConverter[identityV1.Tenant_AuditStatus, string](identityV1.Tenant_AuditStatus_name, identityV1.Tenant_AuditStatus_value),
	}

	repo.init()

	return repo
}

func (r *TenantRepo) init() {
	r.repository = gormCrud.NewRepository[identityV1.Tenant, models.Tenant](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
	r.mapper.AppendConverters(r.typeConverter.NewConverterPair())
	r.mapper.AppendConverters(r.auditStatusConverter.NewConverterPair())
}

func (r *TenantRepo) Count(ctx context.Context, req *paginationV1.PagingRequest) (int, error) {
	filterExpr, err := paginationFilter.ConvertFilterByPagingRequest(req)
	if err != nil {
		r.log.Errorf("parse count param error [%s]", err.Error())
		return 0, identityV1.ErrorBadRequest("invalid query parameter")
	}

	scopes, err := r.structuredFilter.BuildSelectors(filterExpr)
	if err != nil {
		r.log.Errorf("parse count param error [%s]", err.Error())
		return 0, identityV1.ErrorBadRequest("invalid query parameter")
	}

	count, err := r.repository.Count(ctx, r.client.DB, scopes)
	if err != nil {
		r.log.Errorf("query tenant count failed: %s", err.Error())
		return 0, identityV1.ErrorInternalServerError("query count failed")
	}

	return int(count), nil
}

func (r *TenantRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*identityV1.ListTenantResponse, error) {
	if req == nil {
		return nil, identityV1.ErrorBadRequest("invalid parameter")
	}

	ret, err := r.repository.ListWithPaging(ctx, r.client.DB, req)
	if err != nil {
		return nil, identityV1.ErrorInternalServerError("query list failed")
	}
	if ret == nil {
		return &identityV1.ListTenantResponse{Total: 0, Items: nil}, nil
	}

	return &identityV1.ListTenantResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *TenantRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.repository.ExistsWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", id) },
	})
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, identityV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *TenantRepo) Get(ctx context.Context, req *identityV1.GetTenantRequest) (*identityV1.Tenant, error) {
	if req == nil {
		return nil, identityV1.ErrorBadRequest("invalid parameter")
	}

	var scopes []func(*gormDB.DB) *gormDB.DB
	switch req.QueryBy.(type) {
	default:
	case *identityV1.GetTenantRequest_Id:
		scopes = append(scopes, func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) })
	case *identityV1.GetTenantRequest_Code:
		scopes = append(scopes, func(db *gormDB.DB) *gormDB.DB { return db.Where("code = ?", req.GetCode()) })
	case *identityV1.GetTenantRequest_Name:
		scopes = append(scopes, func(db *gormDB.DB) *gormDB.DB { return db.Where("name = ?", req.GetName()) })
	}

	dto, err := r.repository.GetWithFilters(ctx, r.client.DB, scopes, req.GetViewMask())
	if err != nil {
		if errors.Is(err, gormDB.ErrRecordNotFound) {
			return nil, identityV1.ErrorNotFound("tenant not found")
		}
		r.log.Errorf("query tenant failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query tenant failed")
	}

	return dto, nil
}

func (r *TenantRepo) Create(ctx context.Context, data *identityV1.Tenant) (*identityV1.Tenant, error) {
	if data == nil {
		return nil, identityV1.ErrorBadRequest("invalid parameter")
	}

	dto, err := r.repository.Create(ctx, r.client.DB, data, nil)
	if err != nil {
		r.log.Errorf("insert tenant failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("insert data failed")
	}

	return dto, nil
}

func (r *TenantRepo) Update(ctx context.Context, req *identityV1.UpdateTenantRequest) error {
	if req == nil || req.Data == nil {
		return identityV1.ErrorBadRequest("invalid parameter")
	}
	if req.GetId() == 0 {
		return identityV1.ErrorBadRequest("id is required")
	}

	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return err
		}
		if !exist {
			createReq := &identityV1.CreateTenantRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			_, err = r.Create(ctx, createReq.Data)
			return err
		}
	}

	if _, err := r.repository.UpdateWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) },
	}, req.Data, req.GetUpdateMask()); err != nil {
		r.log.Errorf("update tenant failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("update tenant failed")
	}

	return nil
}

func (r *TenantRepo) Delete(ctx context.Context, req *identityV1.DeleteTenantRequest) error {
	if req == nil {
		return identityV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) },
	}); err != nil {
		if errors.Is(err, gormDB.ErrRecordNotFound) {
			return identityV1.ErrorNotFound("tenant not found")
		}
		r.log.Errorf("delete tenant failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("delete failed")
	}

	return nil
}

func (r *TenantRepo) TenantExists(ctx context.Context, req *identityV1.TenantExistsRequest) (*identityV1.TenantExistsResponse, error) {
	var scopes []func(*gormDB.DB) *gormDB.DB
	if code := req.GetCode(); code != "" {
		scopes = append(scopes, func(db *gormDB.DB) *gormDB.DB { return db.Where("code = ?", code) })
	}
	if name := req.GetName(); name != "" {
		scopes = append(scopes, func(db *gormDB.DB) *gormDB.DB { return db.Where("name = ?", name) })
	}

	exist, err := r.repository.ExistsWithFilters(ctx, r.client.DB, scopes)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query exist failed")
	}

	return &identityV1.TenantExistsResponse{
		Exist: exist,
	}, nil
}

func (r *TenantRepo) ListTenantsByIds(ctx context.Context, ids []uint32) ([]*identityV1.Tenant, error) {
	if len(ids) == 0 {
		return []*identityV1.Tenant{}, nil
	}

	var entities []*models.Tenant
	if err := r.client.DB.WithContext(ctx).Model(&models.Tenant{}).Where("id IN ?", ids).Find(&entities).Error; err != nil {
		r.log.Errorf("query tenant by ids failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query tenant by ids failed")
	}

	dtos := make([]*identityV1.Tenant, 0, len(entities))
	for _, entity := range entities {
		dtos = append(dtos, r.mapper.ToDTO(entity))
	}

	return dtos, nil
}

// BeginTx — gorm scaffold stub: ent *ent.Tx 跨仓储事务发起（tenant-onboarding 链的起点），
// go-crud/gorm 无对应原语。见 data/tenant_repo.go。
func (r *TenantRepo) BeginTx(ctx context.Context) (any, func(), error) {
	return nil, nil, identityV1.ErrorInternalServerError("gorm scaffold: BeginTx not implemented — ent *ent.Tx cross-repo transaction has no go-crud/gorm primitive; see data/tenant_repo.go")
}

// CreateWithTx — gorm scaffold stub: 跨仓储事务内的 tenant 创建，见 data/tenant_repo.go。
func (r *TenantRepo) CreateWithTx(ctx context.Context, tx any, data *identityV1.Tenant) (*identityV1.Tenant, error) {
	return nil, identityV1.ErrorInternalServerError("gorm scaffold: CreateWithTx not implemented — ent *ent.Tx cross-repo transaction has no go-crud/gorm primitive; see data/tenant_repo.go")
}

// AssignTenantAdmin — gorm scaffold stub: 跨仓储事务内的 admin_user_id 设置，见 data/tenant_repo.go。
func (r *TenantRepo) AssignTenantAdmin(ctx context.Context, tx any, tenantId uint32, userId uint32) error {
	return identityV1.ErrorInternalServerError("gorm scaffold: AssignTenantAdmin not implemented — ent *ent.Tx cross-repo transaction has no go-crud/gorm primitive; see data/tenant_repo.go")
}
