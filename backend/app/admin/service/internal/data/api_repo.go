package data

import (
	"context"

	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/api"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

type ApiRepo struct {
	entClient  *entCrud.EntClient[*ent.Client]
	driverName string
	log        *bLogger.Helper

	// permissionApi 绑定 repo：接口同步/删除按 (path, method) 级联清理绑定
	// 快照用；测试场景可经 NewApiRepoWithoutBindings 留空（级联能力不可用）。
	permissionApi *PermissionApiRepo

	mapper                  *mapper.CopierMapper[permissionV1.Api, ent.Api]
	statusConverter         *mapper.EnumTypeConverter[permissionV1.Api_Status, api.Status]
	scopeConverter          *mapper.EnumTypeConverter[permissionV1.Api_Scope, api.Scope]
	businessModuleConverter *mapper.EnumTypeConverter[identityV1.Module, api.BusinessModule]

	repository *entCrud.Repository[
		ent.APIQuery, ent.APISelect,
		ent.APICreate, ent.APICreateBulk,
		ent.APIUpdate, ent.APIUpdateOne,
		ent.APIDelete,
		predicate.Api,
		permissionV1.Api, ent.Api,
	]
}

func NewApiRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client], permissionApi *PermissionApiRepo) *ApiRepo {
	repo := &ApiRepo{
		log:           ctx.NewLoggerHelper("api/repo/admin-service"),
		entClient:     entClient,
		driverName:    driverNameOf(ctx),
		permissionApi: permissionApi,
		mapper:        mapper.NewCopierMapper[permissionV1.Api, ent.Api](),
		statusConverter: mapper.NewEnumTypeConverter[permissionV1.Api_Status, api.Status](
			permissionV1.Api_Status_name, permissionV1.Api_Status_value,
		),
		scopeConverter: mapper.NewEnumTypeConverter[permissionV1.Api_Scope, api.Scope](
			permissionV1.Api_Scope_name, permissionV1.Api_Scope_value,
		),
		businessModuleConverter: mapper.NewEnumTypeConverter[identityV1.Module, api.BusinessModule](
			identityV1.Module_name, identityV1.Module_value,
		),
	}

	repo.init()

	return repo
}

// NewApiRepoWithoutBindings 测试/装配兜底：无绑定 repo 依赖的构造器。
// 仅可做不涉及绑定级联的操作（CRUD），SyncMerge/级联删除不可用。
func NewApiRepoWithoutBindings(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *ApiRepo {
	return NewApiRepo(ctx, entClient, nil)
}

func (r *ApiRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.APIQuery, ent.APISelect,
		ent.APICreate, ent.APICreateBulk,
		ent.APIUpdate, ent.APIUpdateOne,
		ent.APIDelete,
		predicate.Api,
		permissionV1.Api, ent.Api,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	// status 转换器补注册：本仓 status 实体列为可空指针枚举（SwitchStatus
	// mixin、列默认 ON），但本仓此前从未注册 statusConverter——scope/
	// business_module 均有注册而 status 独缺。mapper 经转换对（指针↔指针）
	// 查表转换，缺注册即无转换对、copier 对该字段直接跳过，读视图恒零值
	//（与 scope 域同表同列形态却长期不可见）。补注册后读路径经 mapper 如实
	// 流通；写路径维持"未指定即走列默认"现状不变。
	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
	r.mapper.AppendConverters(r.scopeConverter.NewConverterPair())
	r.mapper.AppendConverters(r.businessModuleConverter.NewConverterPair())
}

func (r *ApiRepo) Count(ctx context.Context, req *paginationV1.PagingRequest) (*permissionV1.CountApiResponse, error) {
	builder := r.entClient.Client().Api.Query()

	whereSelectors, _, _ := r.repository.BuildListSelectorWithPaging(builder, req)
	if len(whereSelectors) != 0 {
		builder.Modify(whereSelectors...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query api count failed: %s", err.Error())
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

	builder := r.entClient.Client().Api.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
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
	exist, err := r.entClient.Client().Api.Query().
		Where(api.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query exist failed: %s", err.Error())
		return false, permissionV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *ApiRepo) Get(ctx context.Context, req *permissionV1.GetApiRequest) (*permissionV1.Api, error) {
	if req == nil {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Api.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *permissionV1.GetApiRequest_Id:
		whereCond = append(whereCond, api.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

// GetApiByEndpoint 根据路径和方法获取API资源
func (r *ApiRepo) GetApiByEndpoint(ctx context.Context, path, method string) (*permissionV1.Api, error) {
	if path == "" || method == "" {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	entity, err := r.entClient.Client().Api.Query().
		Where(
			api.PathEQ(path),
			api.MethodEQ(method),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, permissionV1.ErrorNotFound("api not found")
		}

		r.log.Errorf(ctx, "query one data failed: %s", err.Error())

		return nil, permissionV1.ErrorInternalServerError("query data failed")
	}

	return r.mapper.ToDTO(entity), nil
}

// GetApiByIDs 根据ID列表获取API资源
func (r *ApiRepo) GetApiByIDs(ctx context.Context, ids []uint32) ([]*permissionV1.Api, error) {
	if len(ids) == 0 {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	entities, err := r.entClient.Client().Api.Query().
		Where(
			api.IDIn(ids...),
		).
		All(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, permissionV1.ErrorNotFound("api not found")
		}

		r.log.Errorf(ctx, "query one data failed: %s", err.Error())

		return nil, permissionV1.ErrorInternalServerError("query data failed")
	}

	dtos := make([]*permissionV1.Api, 0, len(entities))
	for _, entity := range entities {
		dto := r.mapper.ToDTO(entity)
		dtos = append(dtos, dto)
	}

	return dtos, nil
}

func (r *ApiRepo) Create(ctx context.Context, req *permissionV1.CreateApiRequest) error {
	if req == nil || req.Data == nil {
		return permissionV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.newApiCreate(req.Data)

	if err := builder.Exec(ctx); err != nil {
		r.log.Errorf(ctx, "insert api failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("insert api failed")
	}

	return nil
}

func (r *ApiRepo) newApiCreate(api *permissionV1.Api) *ent.APICreate {
	builder := r.entClient.Client().Api.Create().
		SetNillableDescription(api.Description).
		SetNillableModule(api.Module).
		SetNillableModuleDescription(api.ModuleDescription).
		SetNillableOperation(api.Operation).
		SetNillablePath(api.Path).
		SetNillableMethod(api.Method).
		SetNillableScope(r.scopeConverter.ToEntity(api.Scope)).
		SetNillableCreatedBy(api.CreatedBy).
		SetCreatedAt(time.Now())

	// business_module 为 proto 零值（MODULE_UNSPECIFIED）时跳过：ent schema 未声明该值，
	// SetNillableBusinessModule 会触发 BusinessModuleValidator 失败。未指定即留空。
	if api.BusinessModule != nil && *api.BusinessModule != identityV1.Module_MODULE_UNSPECIFIED {
		builder.SetNillableBusinessModule(r.businessModuleConverter.ToEntity(api.BusinessModule))
	}

	if api.Id != nil {
		builder.SetID(api.GetId())
	}

	return builder
}

func (r *ApiRepo) BatchCreate(ctx context.Context, apis []*permissionV1.Api) error {
	if len(apis) == 0 {
		return nil
	}

	bulk := make([]*ent.APICreate, 0, len(apis))
	for _, dto := range apis {
		builder := r.newApiCreate(dto)
		bulk = append(bulk, builder)
	}

	bulkBuilder := r.entClient.Client().Api.CreateBulk(bulk...)

	if err := bulkBuilder.Exec(ctx); err != nil {
		r.log.Errorf(ctx, "batch insert apis failed: %s", err.Error())
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

	// (path, method) 是 API 的身份键（权限绑定快照/租户闸门/authorizer 策略都
	// 以它为裁决键），不接受改身份：要变更 = 删旧建新（或走接口同步 MERGE）。
	// 放行改身份会让既有绑定与策略悬空，授权静默失效。
	existing, err := r.entClient.Client().Api.Query().
		Where(api.IDEQ(req.GetId())).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return permissionV1.ErrorNotFound("api not found")
		}
		r.log.Errorf(ctx, "query api for identity guard failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("query api failed")
	}
	if req.Data.Path != nil && existing.Path != nil && *req.Data.Path != *existing.Path {
		return permissionV1.ErrorBadRequest("api path is immutable: delete and recreate instead")
	}
	if req.Data.Method != nil && existing.Method != nil && *req.Data.Method != *existing.Method {
		return permissionV1.ErrorBadRequest("api method is immutable: delete and recreate instead")
	}

	builder := r.entClient.Client().Api.Update()
	err = r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *permissionV1.Api) {
			builder.
				SetNillableDescription(req.Data.Description).
				SetNillableModule(req.Data.Module).
				SetNillableModuleDescription(req.Data.ModuleDescription).
				SetNillableOperation(req.Data.Operation).
				SetNillablePath(req.Data.Path).
				SetNillableMethod(req.Data.Method).
				SetNillableScope(r.scopeConverter.ToEntity(req.Data.Scope)).
				SetNillableUpdatedBy(req.Data.UpdatedBy).
				SetUpdatedAt(time.Now())

			// business_module 为 proto 零值（MODULE_UNSPECIFIED）时跳过：ent schema 未声明该值，
			// SetNillableBusinessModule 会触发 BusinessModuleValidator 失败。未指定即不更新该字段。
			if req.Data.BusinessModule != nil && *req.Data.BusinessModule != identityV1.Module_MODULE_UNSPECIFIED {
				builder.SetNillableBusinessModule(r.businessModuleConverter.ToEntity(req.Data.BusinessModule))
			}
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(api.FieldID, req.GetId()))
		},
	)

	return err
}

func (r *ApiRepo) Delete(ctx context.Context, req *permissionV1.DeleteApiRequest) error {
	if req == nil {
		return permissionV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Api.Delete()

	_, err := r.repository.Delete(ctx, builder, func(s *sql.Selector) {
		s.Where(sql.EQ(api.FieldID, req.GetId()))
	})
	if err != nil {
		r.log.Errorf(ctx, "delete api failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete api failed")
	}

	return nil
}

// ListAllIDs 列出全部 API 行 id（模板权限播种期收敛引用用；接口规模有界，
// 全量拉取无分页必要）。
func (r *ApiRepo) ListAllIDs(ctx context.Context) ([]uint32, error) {
	intIDs, err := r.entClient.Client().Api.Query().
		Select(api.FieldID).
		Ints(ctx)
	if err != nil {
		r.log.Errorf(ctx, "list all api ids failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("list all api ids failed")
	}
	ids := make([]uint32, len(intIDs))
	for i, v := range intIDs {
		ids[i] = uint32(v)
	}
	return ids, nil
}

// CascadeDeleteBindings 按 (path, method) 级联清理权限绑定快照：API 行删除后
// 调用，消灭孤儿绑定。供 service 层删除流程使用。
func (r *ApiRepo) CascadeDeleteBindings(ctx context.Context, path, method string) error {
	if r.permissionApi == nil {
		// 测试装配未注入绑定 repo：级联能力不可用，如实报错而非静默跳过
		return permissionV1.ErrorInternalServerError("permission api repo not wired")
	}
	return r.permissionApi.DeleteByEndpoints(ctx, r.entClient.Client(), path, method)
}

// SyncMerge 接口同步 MERGE：以 (path, method) 自然键对齐 spec 与库存，全程单事务。
//
//   - 新增：spec 有库无 → 插入（自增 id，永不写显式 id）；
//   - 更新：键相同但元数据（module/description/operation/business_module）变化才 UPDATE；
//   - 删除：库有 spec 无 → 删行，并按端点级联清理权限绑定快照（消灭孤儿绑定）。
//
// 与旧 Truncate + 顺序重编号的两点本质差别：同步窗口不再全表清空（租户闸门
// fail-closed 的全租户 403 窗口缩到毫秒级事务）；id 与键永不漂移（授权零漂移）。
// operatorId 为 0 表示系统身份（启动期播种）。
func (r *ApiRepo) SyncMerge(ctx context.Context, apis []*permissionV1.Api, operatorID uint32) (added, updated, removed int, err error) {
	if r.permissionApi == nil {
		return 0, 0, 0, permissionV1.ErrorInternalServerError("permission api repo not wired")
	}

	var tx *ent.Tx
	tx, err = r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf(ctx, "start transaction failed: %s", err.Error())
		return 0, 0, 0, permissionV1.ErrorInternalServerError("start transaction failed")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.log.Errorf(context.Background(), "transaction rollback failed: %s", rollbackErr.Error())
			}
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			r.log.Errorf(context.Background(), "transaction commit failed: %s", commitErr.Error())
			err = permissionV1.ErrorInternalServerError("transaction commit failed")
		}
	}()

	client := tx.Client()

	existing, err := client.Api.Query().All(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query existing apis for sync failed: %s", err.Error())
		return 0, 0, 0, permissionV1.ErrorInternalServerError("query apis failed")
	}

	existingByKey := make(map[string]*ent.Api, len(existing))
	for _, row := range existing {
		if row == nil || row.Path == nil || row.Method == nil {
			continue
		}
		key := (Endpoint{Path: *row.Path, Method: *row.Method}).key()
		existingByKey[key] = row
	}

	now := time.Now()
	specKeys := make(map[string]struct{}, len(apis))
	for _, dto := range apis {
		if dto == nil || dto.GetPath() == "" || dto.GetMethod() == "" {
			continue
		}
		key := (Endpoint{Path: dto.GetPath(), Method: dto.GetMethod()}).key()
		if _, dup := specKeys[key]; dup {
			// spec 内重复键不可能是合法 OpenAPI，防御性跳过
			continue
		}
		specKeys[key] = struct{}{}

		if exist, ok := existingByKey[key]; ok {
			if r.apiMetadataEqual(exist, dto) {
				continue
			}
			builder := client.Api.UpdateOneID(exist.ID).
				SetNillableModule(dto.Module).
				SetNillableModuleDescription(dto.ModuleDescription).
				SetNillableDescription(dto.Description).
				SetNillableOperation(dto.Operation).
				SetNillableUpdatedBy(nillableOperator(operatorID)).
				SetUpdatedAt(now)
			// business_module 为 proto 零值（MODULE_UNSPECIFIED）时跳过：ent schema
			// 未声明该值，SetNillableBusinessModule 会触发 BusinessModuleValidator 失败
			if dto.BusinessModule != nil && *dto.BusinessModule != identityV1.Module_MODULE_UNSPECIFIED {
				builder.SetNillableBusinessModule(r.businessModuleConverter.ToEntity(dto.BusinessModule))
			}
			if err = builder.Exec(ctx); err != nil {
				r.log.Errorf(ctx, "sync merge update api [%s %s] failed: %s", dto.GetMethod(), dto.GetPath(), err.Error())
				return added, updated, removed, permissionV1.ErrorInternalServerError("sync merge update api failed")
			}
			updated++
			continue
		}

		builder := client.Api.Create().
			SetNillablePath(dto.Path).
			SetNillableMethod(dto.Method).
			SetNillableModule(dto.Module).
			SetNillableModuleDescription(dto.ModuleDescription).
			SetNillableDescription(dto.Description).
			SetNillableOperation(dto.Operation).
			SetNillableCreatedBy(nillableOperator(operatorID)).
			SetCreatedAt(now)
		if dto.BusinessModule != nil && *dto.BusinessModule != identityV1.Module_MODULE_UNSPECIFIED {
			builder.SetNillableBusinessModule(r.businessModuleConverter.ToEntity(dto.BusinessModule))
		}
		if err = builder.Exec(ctx); err != nil {
			r.log.Errorf(ctx, "sync merge create api [%s %s] failed: %s", dto.GetMethod(), dto.GetPath(), err.Error())
			return added, updated, removed, permissionV1.ErrorInternalServerError("sync merge create api failed")
		}
		added++
	}

	// 删除段：库有 spec 无 → 删行 + 级联清绑定（同事务，防孤儿绑定）
	for key, row := range existingByKey {
		if _, ok := specKeys[key]; ok {
			continue
		}
		if err = client.Api.DeleteOneID(row.ID).Exec(ctx); err != nil {
			r.log.Errorf(ctx, "sync merge delete api [%s %s] failed: %s", *row.Method, *row.Path, err.Error())
			return added, updated, removed, permissionV1.ErrorInternalServerError("sync merge delete api failed")
		}
		if err = r.permissionApi.DeleteByEndpoints(ctx, client, *row.Path, *row.Method); err != nil {
			return added, updated, removed, err
		}
		removed++
	}

	return added, updated, removed, nil
}

// apiMetadataEqual 判断 MERGE 更新段是否需要落库：仅比较同步会写入的元数据列，
// 未变化则跳过 UPDATE（避免每次同步把全表 updated_at 刷一遍，审计列表全是噪音）。
func (r *ApiRepo) apiMetadataEqual(entity *ent.Api, dto *permissionV1.Api) bool {
	if entity == nil || dto == nil {
		return false
	}
	strEq := func(a, b *string) bool {
		if a == nil && b == nil {
			return true
		}
		if a == nil || b == nil {
			return false
		}
		return *a == *b
	}

	// business_module 经转换器对齐两侧枚举（proto Module ↔ ent BusinessModule）再比
	wantBM := identityV1.Module_MODULE_UNSPECIFIED
	if dto.BusinessModule != nil {
		wantBM = *dto.BusinessModule
	}
	if wantBM == identityV1.Module_MODULE_UNSPECIFIED {
		if entity.BusinessModule != nil {
			return false
		}
	} else {
		if entity.BusinessModule == nil {
			return false
		}
		if got := r.businessModuleConverter.ToDTO(entity.BusinessModule); got == nil || *got != wantBM {
			return false
		}
	}

	return strEq(entity.Module, dto.Module) &&
		strEq(entity.ModuleDescription, dto.ModuleDescription) &&
		strEq(entity.Description, dto.Description) &&
		strEq(entity.Operation, dto.Operation)
}

// nillableOperator operatorID 为 0（系统身份）时不落 updated_by/created_by。
func nillableOperator(operatorID uint32) *uint32 {
	if operatorID == 0 {
		return nil
	}
	return &operatorID
}
