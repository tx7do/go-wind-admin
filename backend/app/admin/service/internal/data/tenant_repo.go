package data

import (
	"context"
	"errors"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/go-crud/entgo/field"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/timeutil"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"
	"go-wind-admin/app/admin/service/internal/data/ent/schema"
	"go-wind-admin/app/admin/service/internal/data/ent/tenant"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
)

type TenantRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *bLogger.Helper

	mapper               *mapper.CopierMapper[identityV1.Tenant, ent.Tenant]
	statusConverter      *mapper.EnumTypeConverter[identityV1.Tenant_Status, tenant.Status]
	typeConverter        *mapper.EnumTypeConverter[identityV1.Tenant_Type, tenant.Type]
	auditStatusConverter *mapper.EnumTypeConverter[identityV1.Tenant_AuditStatus, tenant.AuditStatus]

	repository *entCrud.Repository[
		ent.TenantQuery, ent.TenantSelect,
		ent.TenantCreate, ent.TenantCreateBulk,
		ent.TenantUpdate, ent.TenantUpdateOne,
		ent.TenantDelete,
		predicate.Tenant,
		identityV1.Tenant, ent.Tenant,
	]
}

func NewTenantRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *TenantRepo {
	repo := &TenantRepo{
		log:                  ctx.NewLoggerHelper("tenant/repo/admin-service"),
		entClient:            entClient,
		mapper:               mapper.NewCopierMapper[identityV1.Tenant, ent.Tenant](),
		statusConverter:      mapper.NewEnumTypeConverter[identityV1.Tenant_Status, tenant.Status](identityV1.Tenant_Status_name, identityV1.Tenant_Status_value),
		typeConverter:        mapper.NewEnumTypeConverter[identityV1.Tenant_Type, tenant.Type](identityV1.Tenant_Type_name, identityV1.Tenant_Type_value),
		auditStatusConverter: mapper.NewEnumTypeConverter[identityV1.Tenant_AuditStatus, tenant.AuditStatus](identityV1.Tenant_AuditStatus_name, identityV1.Tenant_AuditStatus_value),
	}

	repo.init()

	return repo
}

func (r *TenantRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.TenantQuery, ent.TenantSelect,
		ent.TenantCreate, ent.TenantCreateBulk,
		ent.TenantUpdate, ent.TenantUpdateOne,
		ent.TenantDelete,
		predicate.Tenant,
		identityV1.Tenant, ent.Tenant,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
	r.mapper.AppendConverters(r.typeConverter.NewConverterPair())
	r.mapper.AppendConverters(r.auditStatusConverter.NewConverterPair())
}

func (r *TenantRepo) Count(ctx context.Context, req *paginationV1.PagingRequest) (int, error) {
	builder := r.entClient.Client().Tenant.Query()

	whereSelectors, _, _ := r.repository.BuildListSelectorWithPaging(builder, req)
	if len(whereSelectors) != 0 {
		builder.Modify(whereSelectors...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query tenant count failed: %s", err.Error())
		return 0, identityV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *TenantRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*identityV1.ListTenantResponse, error) {
	if req == nil {
		return nil, identityV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Tenant.Query().WithPlan()
	countBuilder := r.entClient.Client().Tenant.Query()

	whereSelectors, _, err := r.repository.BuildListSelectorWithPaging(builder, req)
	if err != nil {
		r.log.Errorf(ctx, "parse list param error [%s]", err.Error())
		return nil, identityV1.ErrorBadRequest("invalid query parameter")
	}

	entities, err := builder.All(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query tenant list failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query tenant list failed")
	}

	// plan_id 是边外键（实体上是非导出字段），CopierMapper 拷不进 DTO，
	// 须 WithPlan 预载后从 Edges 回填，否则消费方拿到的 PlanId 恒为 nil。
	dtos := make([]*identityV1.Tenant, 0, len(entities))
	for _, entity := range entities {
		dto := r.mapper.ToDTO(entity)
		if entity.Edges.Plan != nil {
			dto.PlanId = &entity.Edges.Plan.ID
		}
		dtos = append(dtos, dto)
	}

	if len(whereSelectors) != 0 {
		countBuilder.Modify(whereSelectors...)
	}
	count, err := countBuilder.Count(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query tenant count failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query count failed")
	}

	return &identityV1.ListTenantResponse{
		Total: int64(count),
		Items: dtos,
	}, nil
}

func (r *TenantRepo) IsExist(ctx context.Context, id int64) (bool, error) {
	exist, err := r.entClient.Client().Tenant.Query().
		Where(tenant.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query exist failed: %s", err.Error())
		return false, identityV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *TenantRepo) Get(ctx context.Context, req *identityV1.GetTenantRequest) (*identityV1.Tenant, error) {
	if req == nil {
		return nil, identityV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Tenant.Query().WithPlan()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *identityV1.GetTenantRequest_Id:
		whereCond = append(whereCond, tenant.IDEQ(req.GetId()))

	case *identityV1.GetTenantRequest_Code:
		whereCond = append(whereCond, tenant.CodeEQ(req.GetCode()))

	case *identityV1.GetTenantRequest_Name:
		whereCond = append(whereCond, tenant.NameEQ(req.GetName()))
	}

	if len(whereCond) > 0 {
		builder.Modify(whereCond...)
	}

	// repository.Get 泛型实现拿不到 entity 无法回填边，这里按其语义展开
	//（mask 归一化 + 列裁剪 + Only），并保留 entity 供 Edges 回填。
	viewMask := req.GetViewMask()
	field.NormalizeFieldMaskPaths(viewMask)
	if viewMask != nil && len(viewMask.GetPaths()) > 0 {
		builder.Select(viewMask.GetPaths()...)
	}

	entity, err := builder.Only(ctx)
	if err != nil {
		return nil, err
	}

	dto := r.mapper.ToDTO(entity)
	if entity.Edges.Plan != nil {
		dto.PlanId = &entity.Edges.Plan.ID
	}

	return dto, nil
}

func (r *TenantRepo) BeginTx(ctx context.Context) (tx *ent.Tx, cleanup func(), err error) {
	tx, err = r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf(ctx, "start transaction failed: %s", err.Error())
		return nil, nil, identityV1.ErrorInternalServerError("start transaction failed")
	}

	cleanup = func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.log.Errorf(ctx, "transaction rollback failed: %s", rollbackErr.Error())
			}
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			r.log.Errorf(ctx, "transaction commit failed: %s", commitErr.Error())
			err = identityV1.ErrorInternalServerError("transaction commit failed")
		}
	}

	return tx, cleanup, nil
}

func (r *TenantRepo) Create(ctx context.Context, data *identityV1.Tenant) (tenant *identityV1.Tenant, err error) {
	if data == nil {
		return nil, identityV1.ErrorBadRequest("invalid parameter")
	}

	var tx *ent.Tx
	tx, err = r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf(ctx, "start transaction failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("start transaction failed")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.log.Errorf(ctx, "transaction rollback failed: %s", rollbackErr.Error())
			}
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			r.log.Errorf(ctx, "transaction commit failed: %s", commitErr.Error())
			err = identityV1.ErrorInternalServerError("transaction commit failed")
		}
	}()

	return r.CreateWithTx(ctx, tx, data)
}

func (r *TenantRepo) CreateWithTx(ctx context.Context, tx *ent.Tx, data *identityV1.Tenant) (*identityV1.Tenant, error) {
	if data == nil {
		return nil, identityV1.ErrorBadRequest("invalid parameter")
	}

	builder := tx.Tenant.Create().
		SetNillableName(data.Name).
		SetNillableCode(data.Code).
		SetNillableDomain(data.Domain).
		SetNillableLogoURL(data.LogoUrl).
		SetNillableRemark(data.Remark).
		SetNillableIndustry(data.Industry).
		SetNillableAdminUserID(data.AdminUserId).
		SetNillableStatus(r.statusConverter.ToEntity(data.Status)).
		SetNillableType(r.typeConverter.ToEntity(data.Type)).
		SetNillableAuditStatus(r.auditStatusConverter.ToEntity(data.AuditStatus)).
		SetNillableSubscriptionPlan(data.SubscriptionPlan).
		SetNillableExpiredAt(timeutil.TimestamppbToTime(data.ExpiredAt)).
		SetNillableSubscriptionAt(timeutil.TimestamppbToTime(data.SubscriptionAt)).
		SetNillableUnsubscribeAt(timeutil.TimestamppbToTime(data.UnsubscribeAt)).
		SetNillablePlanID(data.PlanId).
		SetNillableCreatedBy(data.CreatedBy).
		SetCreatedAt(time.Now())

	if data.Id != nil {
		builder.SetID(data.GetId())
	}

	if ret, err := builder.Save(ctx); err != nil {
		r.log.Errorf(ctx, "insert tenant failed: %s", err.Error())
		// 唯一索引不含软删位：回收站停车行的 name/code 仍占用唯一键，重名/重码
		// 会撞 DB 约束——映射成明确提示（可从回收站恢复或彻底删除后重试）。
		var constraintErr *ent.ConstraintError
		if errors.As(err, &constraintErr) {
			return nil, identityV1.ErrorBadRequest(
				"tenant name or code already exists (possibly parked in recycle bin)")
		}
		return nil, identityV1.ErrorInternalServerError("insert tenant failed")
	} else {
		return r.mapper.ToDTO(ret), nil
	}
}

func (r *TenantRepo) Update(ctx context.Context, req *identityV1.UpdateTenantRequest) error {
	if req == nil || req.Data == nil {
		return identityV1.ErrorBadRequest("invalid parameter")
	}
	if req.GetId() == 0 {
		return identityV1.ErrorBadRequest("id is required")
	}

	// 如果不存在则创建
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

	builder := r.entClient.Client().Tenant.Update()
	err := r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *identityV1.Tenant) {
			builder.
				SetNillableName(req.Data.Name).
				SetNillableCode(req.Data.Code).
				SetNillableDomain(req.Data.Domain).
				SetNillableLogoURL(req.Data.LogoUrl).
				SetNillableRemark(req.Data.Remark).
				SetNillableIndustry(req.Data.Industry).
				SetNillableAdminUserID(req.Data.AdminUserId).
				SetNillableStatus(r.statusConverter.ToEntity(req.Data.Status)).
				SetNillableType(r.typeConverter.ToEntity(req.Data.Type)).
				SetNillableAuditStatus(r.auditStatusConverter.ToEntity(req.Data.AuditStatus)).
				SetNillableSubscriptionPlan(req.Data.SubscriptionPlan).
				SetNillableExpiredAt(timeutil.TimestamppbToTime(req.Data.ExpiredAt)).
				SetNillableSubscriptionAt(timeutil.TimestamppbToTime(req.Data.SubscriptionAt)).
				SetNillableUnsubscribeAt(timeutil.TimestamppbToTime(req.Data.UnsubscribeAt)).
				SetNillablePlanID(req.Data.PlanId).
				SetNillableUpdatedBy(req.Data.UpdatedBy).
				SetUpdatedAt(time.Now())
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(tenant.FieldID, req.GetId()))
		},
	)

	return err
}

// AssignTenantAdmin assigns an admin user to a tenant within a transaction.
func (r *TenantRepo) AssignTenantAdmin(ctx context.Context, tx *ent.Tx, tenantId int64, userId int64) error {
	_, err := tx.Tenant.Update().
		Where(tenant.IDEQ(tenantId)).
		SetAdminUserID(userId).
		Save(ctx)
	if err != nil {
		r.log.Errorf(ctx, "assign tenant admin failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("assign tenant admin failed")
	}

	return nil
}

// Delete 回收站语义（issue #3）：删除=停车——置 deleted_at 并把状态拨 OFF，
// 行保留在库；在营查询经 SoftDelete 拦截器自然过滤，恢复走 Restore，
// 彻底删除走 Purge。此前此处的 DeleteOneID 是硬删，租户直接蒸发。
func (r *TenantRepo) Delete(ctx context.Context, req *identityV1.DeleteTenantRequest) error {
	if req == nil {
		return identityV1.ErrorBadRequest("invalid parameter")
	}

	err := r.entClient.Client().Tenant.UpdateOneID(req.GetId()).
		SetDeletedAt(time.Now()).
		SetStatus(tenant.StatusOff).
		Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return identityV1.ErrorNotFound("tenant not found")
		}

		r.log.Errorf(ctx, "soft delete (park) tenant failed: %s", err.Error())

		return identityV1.ErrorInternalServerError("delete failed")
	}

	return nil
}

// ListDeleted 回收站列表：分页查询已停车（deleted_at 非空）租户。
// 带 SkipSoftDelete 旁路绕过软删过滤，再显式用 DeletedAtNotNil 收窄范围。
func (r *TenantRepo) ListDeleted(ctx context.Context, req *paginationV1.PagingRequest) (*identityV1.ListTenantResponse, error) {
	if req == nil {
		return nil, identityV1.ErrorBadRequest("invalid parameter")
	}

	ctx = schema.SkipSoftDelete(ctx)

	builder := r.entClient.Client().Tenant.Query().WithPlan().Where(tenant.DeletedAtNotNil())
	countBuilder := r.entClient.Client().Tenant.Query().Where(tenant.DeletedAtNotNil())

	whereSelectors, _, err := r.repository.BuildListSelectorWithPaging(builder, req)
	if err != nil {
		r.log.Errorf(ctx, "parse list param error [%s]", err.Error())
		return nil, identityV1.ErrorBadRequest("invalid query parameter")
	}

	entities, err := builder.All(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query deleted tenant list failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query tenant trash list failed")
	}

	dtos := make([]*identityV1.Tenant, 0, len(entities))
	for _, entity := range entities {
		dto := r.mapper.ToDTO(entity)
		if entity.Edges.Plan != nil {
			dto.PlanId = &entity.Edges.Plan.ID
		}
		dtos = append(dtos, dto)
	}

	if len(whereSelectors) != 0 {
		countBuilder.Modify(whereSelectors...)
	}
	count, err := countBuilder.Count(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query deleted tenant count failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query count failed")
	}

	return &identityV1.ListTenantResponse{
		Total: int64(count),
		Items: dtos,
	}, nil
}

// Restore 回收站：恢复停车租户——清 deleted_at 并把状态拨回 ON。
func (r *TenantRepo) Restore(ctx context.Context, id int64) error {
	err := r.entClient.Client().Tenant.UpdateOneID(id).
		ClearDeletedAt().
		SetStatus(tenant.StatusOn).
		Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return identityV1.ErrorNotFound("tenant not found")
		}

		r.log.Errorf(ctx, "restore parked tenant failed: %s", err.Error())

		return identityV1.ErrorInternalServerError("restore tenant failed")
	}
	return nil
}

// Purge 回收站：彻底删除停车租户（硬删行，不可逆）。
// 前置校验目标确属停车行——在营租户不允许走此路径，防误操作直接蒸发。
func (r *TenantRepo) Purge(ctx context.Context, id int64) error {
	checkCtx := schema.SkipSoftDelete(ctx)
	exist, err := r.entClient.Client().Tenant.Query().
		Where(tenant.ID(id), tenant.DeletedAtNotNil()).
		Exist(checkCtx)
	if err != nil {
		r.log.Errorf(ctx, "check parked tenant exist failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("purge tenant check failed")
	}
	if !exist {
		return identityV1.ErrorNotFound("parked tenant not found")
	}

	if err := r.entClient.Client().Tenant.DeleteOneID(id).Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return identityV1.ErrorNotFound("tenant not found")
		}

		r.log.Errorf(ctx, "purge tenant failed: %s", err.Error())

		return identityV1.ErrorInternalServerError("purge tenant failed")
	}
	return nil
}

// TenantExists checks if a tenant with the given code or name exists.
func (r *TenantRepo) TenantExists(ctx context.Context, req *identityV1.TenantExistsRequest) (*identityV1.TenantExistsResponse, error) {
	// code 和 name 各自唯一（见 schema 索引），冲突检测应取 OR：
	// 此前用 AND，仅当 code 和 name 同时命中同一行才返回存在，单字段冲突被漏判。
	query := r.entClient.Client().Tenant.Query()
	predicates := []predicate.Tenant{}
	if code := req.GetCode(); code != "" {
		predicates = append(predicates, tenant.CodeEQ(code))
	}
	if name := req.GetName(); name != "" {
		predicates = append(predicates, tenant.NameEQ(name))
	}
	if len(predicates) > 0 {
		query = query.Where(tenant.Or(predicates...))
	}

	exist, err := query.Exist(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query exist failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query exist failed")
	}

	return &identityV1.TenantExistsResponse{
		Exist: exist,
	}, nil
}

// ListTenantsByIds gets tenants by a list of IDs.
func (r *TenantRepo) ListTenantsByIds(ctx context.Context, ids []int64) ([]*identityV1.Tenant, error) {
	if len(ids) == 0 {
		return []*identityV1.Tenant{}, nil
	}

	entities, err := r.entClient.Client().Tenant.Query().
		Where(tenant.IDIn(ids...)).
		All(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query tenant by ids failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query tenant by ids failed")
	}

	dtos := make([]*identityV1.Tenant, 0, len(entities))
	for _, entity := range entities {
		dto := r.mapper.ToDTO(entity)
		dtos = append(dtos, dto)
	}

	return dtos, nil
}
