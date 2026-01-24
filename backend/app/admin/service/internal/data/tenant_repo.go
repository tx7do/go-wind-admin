package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/timeutil"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"
	"go-wind-admin/app/admin/service/internal/data/ent/tenant"

	userV1 "go-wind-admin/api/gen/go/user/service/v1"
)

type TenantRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper               *mapper.CopierMapper[userV1.Tenant, ent.Tenant]
	statusConverter      *mapper.EnumTypeConverter[userV1.Tenant_Status, tenant.Status]
	typeConverter        *mapper.EnumTypeConverter[userV1.Tenant_Type, tenant.Type]
	auditStatusConverter *mapper.EnumTypeConverter[userV1.Tenant_AuditStatus, tenant.AuditStatus]

	repository *entCrud.Repository[
		ent.TenantQuery, ent.TenantSelect,
		ent.TenantCreate, ent.TenantCreateBulk,
		ent.TenantUpdate, ent.TenantUpdateOne,
		ent.TenantDelete,
		predicate.Tenant,
		userV1.Tenant, ent.Tenant,
	]
}

func NewTenantRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *TenantRepo {
	repo := &TenantRepo{
		log:                  ctx.NewLoggerHelper("tenant/repo/admin-service"),
		entClient:            entClient,
		mapper:               mapper.NewCopierMapper[userV1.Tenant, ent.Tenant](),
		statusConverter:      mapper.NewEnumTypeConverter[userV1.Tenant_Status, tenant.Status](userV1.Tenant_Status_name, userV1.Tenant_Status_value),
		typeConverter:        mapper.NewEnumTypeConverter[userV1.Tenant_Type, tenant.Type](userV1.Tenant_Type_name, userV1.Tenant_Type_value),
		auditStatusConverter: mapper.NewEnumTypeConverter[userV1.Tenant_AuditStatus, tenant.AuditStatus](userV1.Tenant_AuditStatus_name, userV1.Tenant_AuditStatus_value),
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
		userV1.Tenant, ent.Tenant,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
	r.mapper.AppendConverters(r.typeConverter.NewConverterPair())
	r.mapper.AppendConverters(r.auditStatusConverter.NewConverterPair())
}

func (r *TenantRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().Tenant.Query()
	if len(whereCond) != 0 {
		builder.Modify(whereCond...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, userV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *TenantRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*userV1.ListTenantResponse, error) {
	if req == nil {
		return nil, userV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Tenant.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &userV1.ListTenantResponse{Total: 0, Items: nil}, nil
	}

	return &userV1.ListTenantResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *TenantRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().Tenant.Query().
		Where(tenant.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, userV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *TenantRepo) Get(ctx context.Context, req *userV1.GetTenantRequest) (*userV1.Tenant, error) {
	if req == nil {
		return nil, userV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Tenant.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *userV1.GetTenantRequest_Id:
		whereCond = append(whereCond, tenant.IDEQ(req.GetId()))

	case *userV1.GetTenantRequest_Code:
		whereCond = append(whereCond, tenant.CodeEQ(req.GetCode()))

	case *userV1.GetTenantRequest_Name:
		whereCond = append(whereCond, tenant.NameEQ(req.GetName()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

func (r *TenantRepo) BeginTx(ctx context.Context) (tx *ent.Tx, cleanup func(), err error) {
	tx, err = r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf("start transaction failed: %s", err.Error())
		return nil, nil, userV1.ErrorInternalServerError("start transaction failed")
	}

	cleanup = func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.log.Errorf("transaction rollback failed: %s", rollbackErr.Error())
			}
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			r.log.Errorf("transaction commit failed: %s", commitErr.Error())
			err = userV1.ErrorInternalServerError("transaction commit failed")
		}
	}

	return tx, cleanup, nil
}

func (r *TenantRepo) Create(ctx context.Context, data *userV1.Tenant) (tenant *userV1.Tenant, err error) {
	if data == nil {
		return nil, userV1.ErrorBadRequest("invalid parameter")
	}

	var tx *ent.Tx
	tx, err = r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf("start transaction failed: %s", err.Error())
		return nil, userV1.ErrorInternalServerError("start transaction failed")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.log.Errorf("transaction rollback failed: %s", rollbackErr.Error())
			}
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			r.log.Errorf("transaction commit failed: %s", commitErr.Error())
			err = userV1.ErrorInternalServerError("transaction commit failed")
		}
	}()

	return r.CreateWithTx(ctx, tx, data)
}

func (r *TenantRepo) CreateWithTx(ctx context.Context, tx *ent.Tx, data *userV1.Tenant) (*userV1.Tenant, error) {
	if data == nil {
		return nil, userV1.ErrorBadRequest("invalid parameter")
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
		SetNillableCreatedBy(data.CreatedBy).
		SetCreatedAt(time.Now())

	if data.Id != nil {
		builder.SetID(data.GetId())
	}

	if ret, err := builder.Save(ctx); err != nil {
		r.log.Errorf("insert tenant failed: %s", err.Error())
		return nil, userV1.ErrorInternalServerError("insert tenant failed")
	} else {
		return r.mapper.ToDTO(ret), nil
	}
}

func (r *TenantRepo) Update(ctx context.Context, req *userV1.UpdateTenantRequest) error {
	if req == nil || req.Data == nil {
		return userV1.ErrorBadRequest("invalid parameter")
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return err
		}
		if !exist {
			createReq := &userV1.CreateTenantRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			_, err = r.Create(ctx, createReq.Data)
			return err
		}
	}

	builder := r.entClient.Client().Debug().Tenant.Update()
	err := r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *userV1.Tenant) {
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
func (r *TenantRepo) AssignTenantAdmin(ctx context.Context, tx *ent.Tx, tenantId uint32, userId uint32) error {
	_, err := tx.Tenant.Update().
		Where(tenant.IDEQ(tenantId)).
		SetAdminUserID(userId).
		Save(ctx)
	if err != nil {
		r.log.Errorf("assign tenant admin failed: %s", err.Error())
		return userV1.ErrorInternalServerError("assign tenant admin failed")
	}

	return nil
}

func (r *TenantRepo) Delete(ctx context.Context, req *userV1.DeleteTenantRequest) error {
	if req == nil {
		return userV1.ErrorBadRequest("invalid parameter")
	}

	if err := r.entClient.Client().Tenant.DeleteOneID(req.GetId()).Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return userV1.ErrorNotFound("tenant not found")
		}

		r.log.Errorf("delete one data failed: %s", err.Error())

		return userV1.ErrorInternalServerError("delete failed")
	}

	return nil
}

// TenantExists checks if a tenant with the given username exists.
func (r *TenantRepo) TenantExists(ctx context.Context, req *userV1.TenantExistsRequest) (*userV1.TenantExistsResponse, error) {
	exist, err := r.entClient.Client().Tenant.Query().
		Where(
			tenant.CodeEQ(req.GetCode()),
			tenant.NameEQ(req.GetName()),
		).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return nil, userV1.ErrorInternalServerError("query exist failed")
	}

	return &userV1.TenantExistsResponse{
		Exist: exist,
	}, nil
}

// ListTenantsByIds gets tenants by a list of IDs.
func (r *TenantRepo) ListTenantsByIds(ctx context.Context, ids []uint32) ([]*userV1.Tenant, error) {
	if len(ids) == 0 {
		return []*userV1.Tenant{}, nil
	}

	entities, err := r.entClient.Client().Tenant.Query().
		Where(tenant.IDIn(ids...)).
		All(ctx)
	if err != nil {
		r.log.Errorf("query tenant by ids failed: %s", err.Error())
		return nil, userV1.ErrorInternalServerError("query tenant by ids failed")
	}

	dtos := make([]*userV1.Tenant, 0, len(entities))
	for _, entity := range entities {
		dto := r.mapper.ToDTO(entity)
		dtos = append(dtos, dto)
	}

	return dtos, nil
}
