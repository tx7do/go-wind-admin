package data

import (
	"context"

	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	pagination "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/timeutil"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/apiresource"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

type ApiResourceRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper         *mapper.CopierMapper[permissionV1.ApiResource, ent.ApiResource]
	scopeConverter *mapper.EnumTypeConverter[permissionV1.ApiResource_Scope, apiresource.Scope]

	repository *entCrud.Repository[
		ent.ApiResourceQuery, ent.ApiResourceSelect,
		ent.ApiResourceCreate, ent.ApiResourceCreateBulk,
		ent.ApiResourceUpdate, ent.ApiResourceUpdateOne,
		ent.ApiResourceDelete,
		predicate.ApiResource,
		permissionV1.ApiResource, ent.ApiResource,
	]
}

func NewApiResourceRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *ApiResourceRepo {
	repo := &ApiResourceRepo{
		log:       ctx.NewLoggerHelper("api-resource/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[permissionV1.ApiResource, ent.ApiResource](),
		scopeConverter: mapper.NewEnumTypeConverter[permissionV1.ApiResource_Scope, apiresource.Scope](
			permissionV1.ApiResource_Scope_name, permissionV1.ApiResource_Scope_value,
		),
	}

	repo.init()

	return repo
}

func (r *ApiResourceRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.ApiResourceQuery, ent.ApiResourceSelect,
		ent.ApiResourceCreate, ent.ApiResourceCreateBulk,
		ent.ApiResourceUpdate, ent.ApiResourceUpdateOne,
		ent.ApiResourceDelete,
		predicate.ApiResource,
		permissionV1.ApiResource, ent.ApiResource,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.scopeConverter.NewConverterPair())
}

func (r *ApiResourceRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().ApiResource.Query()
	if len(whereCond) != 0 {
		builder.Modify(whereCond...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, permissionV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *ApiResourceRepo) List(ctx context.Context, req *pagination.PagingRequest) (*permissionV1.ListApiResourceResponse, error) {
	if req == nil {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().ApiResource.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &permissionV1.ListApiResourceResponse{Total: 0, Items: nil}, nil
	}

	return &permissionV1.ListApiResourceResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *ApiResourceRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().ApiResource.Query().
		Where(apiresource.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, permissionV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *ApiResourceRepo) Get(ctx context.Context, req *permissionV1.GetApiResourceRequest) (*permissionV1.ApiResource, error) {
	if req == nil {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().ApiResource.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *permissionV1.GetApiResourceRequest_Id:
		whereCond = append(whereCond, apiresource.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

// GetApiResourceByEndpoint 根据路径和方法获取API资源
func (r *ApiResourceRepo) GetApiResourceByEndpoint(ctx context.Context, path, method string) (*permissionV1.ApiResource, error) {
	if path == "" || method == "" {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	entity, err := r.entClient.Client().ApiResource.Query().
		Where(
			apiresource.PathEQ(path),
			apiresource.MethodEQ(method),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, permissionV1.ErrorNotFound("api resource not found")
		}

		r.log.Errorf("query one data failed: %s", err.Error())

		return nil, permissionV1.ErrorInternalServerError("query data failed")
	}

	return r.mapper.ToDTO(entity), nil
}

// GetApiResourceByIDs 根据ID列表获取API资源
func (r *ApiResourceRepo) GetApiResourceByIDs(ctx context.Context, ids []uint32) ([]*permissionV1.ApiResource, error) {
	if len(ids) == 0 {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	entities, err := r.entClient.Client().ApiResource.Query().
		Where(
			apiresource.IDIn(ids...),
		).
		All(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, permissionV1.ErrorNotFound("api resource not found")
		}

		r.log.Errorf("query one data failed: %s", err.Error())

		return nil, permissionV1.ErrorInternalServerError("query data failed")
	}

	dtos := make([]*permissionV1.ApiResource, 0, len(entities))
	for _, entity := range entities {
		dto := r.mapper.ToDTO(entity)
		dtos = append(dtos, dto)
	}

	return dtos, nil
}

func (r *ApiResourceRepo) Create(ctx context.Context, req *permissionV1.CreateApiResourceRequest) error {
	if req == nil || req.Data == nil {
		return permissionV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.newApiResourceCreate(req.Data)

	if err := builder.Exec(ctx); err != nil {
		r.log.Errorf("insert one data failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("insert data failed")
	}

	return nil
}

func (r *ApiResourceRepo) newApiResourceCreate(apiResource *permissionV1.ApiResource) *ent.ApiResourceCreate {
	builder := r.entClient.Client().ApiResource.Create().
		SetNillableDescription(apiResource.Description).
		SetNillableModule(apiResource.Module).
		SetNillableModuleDescription(apiResource.ModuleDescription).
		SetNillableOperation(apiResource.Operation).
		SetNillablePath(apiResource.Path).
		SetNillableMethod(apiResource.Method).
		SetNillableScope(r.scopeConverter.ToEntity(apiResource.Scope)).
		SetNillableCreatedBy(apiResource.CreatedBy).
		SetNillableCreatedAt(timeutil.TimestamppbToTime(apiResource.CreatedAt))

	if apiResource.CreatedAt == nil {
		builder.SetCreatedAt(time.Now())
	}

	if apiResource.Id != nil {
		builder.SetID(apiResource.GetId())
	}

	return builder
}

func (r *ApiResourceRepo) BatchCreate(ctx context.Context, apiResources []*permissionV1.ApiResource) error {
	if len(apiResources) == 0 {
		return nil
	}

	bulk := make([]*ent.ApiResourceCreate, 0, len(apiResources))
	for _, dto := range apiResources {
		builder := r.newApiResourceCreate(dto)
		bulk = append(bulk, builder)
	}

	bulkBuilder := r.entClient.Client().ApiResource.CreateBulk(bulk...)

	if err := bulkBuilder.Exec(ctx); err != nil {
		r.log.Errorf("batch insert data failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("batch insert data failed")
	}

	return nil
}

func (r *ApiResourceRepo) Update(ctx context.Context, req *permissionV1.UpdateApiResourceRequest) error {
	if req == nil || req.Data == nil {
		return permissionV1.ErrorBadRequest("invalid parameter")
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return err
		}
		if !exist {
			createReq := &permissionV1.CreateApiResourceRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	builder := r.entClient.Client().Debug().ApiResource.Update()
	err := r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *permissionV1.ApiResource) {
			builder.
				SetNillableDescription(req.Data.Description).
				SetNillableModule(req.Data.Module).
				SetNillableModuleDescription(req.Data.ModuleDescription).
				SetNillableOperation(req.Data.Operation).
				SetNillablePath(req.Data.Path).
				SetNillableMethod(req.Data.Method).
				SetNillableScope(r.scopeConverter.ToEntity(req.Data.Scope)).
				SetNillableUpdatedBy(req.Data.UpdatedBy).
				SetNillableUpdatedAt(timeutil.TimestamppbToTime(req.Data.UpdatedAt))

			if req.Data.UpdatedAt == nil {
				builder.SetUpdatedAt(time.Now())
			}
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(apiresource.FieldID, req.GetId()))
		},
	)

	return err
}

func (r *ApiResourceRepo) Delete(ctx context.Context, req *permissionV1.DeleteApiResourceRequest) error {
	if req == nil {
		return permissionV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Debug().ApiResource.Delete()

	_, err := r.repository.Delete(ctx, builder, func(s *sql.Selector) {
		s.Where(sql.EQ(apiresource.FieldID, req.GetId()))
	})
	if err != nil {
		r.log.Errorf("delete api resource failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete api resource failed")
	}

	return nil
}

// Truncate 清空表数据
func (r *ApiResourceRepo) Truncate(ctx context.Context) error {
	if _, err := r.entClient.Client().ApiResource.Delete().Exec(ctx); err != nil {
		r.log.Errorf("failed to truncate api_resources table: %s", err.Error())
		return permissionV1.ErrorInternalServerError("truncate failed")
	}
	return nil
}
