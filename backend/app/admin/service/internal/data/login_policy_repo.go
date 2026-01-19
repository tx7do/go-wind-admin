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
	"go-wind-admin/app/admin/service/internal/data/ent/loginpolicy"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
)

type LoginPolicyRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper          *mapper.CopierMapper[adminV1.LoginPolicy, ent.LoginPolicy]
	typeConverter   *mapper.EnumTypeConverter[adminV1.LoginPolicy_Type, loginpolicy.Type]
	methodConverter *mapper.EnumTypeConverter[adminV1.LoginPolicy_Method, loginpolicy.Method]

	repository *entCrud.Repository[
		ent.LoginPolicyQuery, ent.LoginPolicySelect,
		ent.LoginPolicyCreate, ent.LoginPolicyCreateBulk,
		ent.LoginPolicyUpdate, ent.LoginPolicyUpdateOne,
		ent.LoginPolicyDelete,
		predicate.LoginPolicy,
		adminV1.LoginPolicy, ent.LoginPolicy,
	]
}

func NewLoginPolicyRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *LoginPolicyRepo {
	repo := &LoginPolicyRepo{
		log:             ctx.NewLoggerHelper("login-policy/repo/admin-service"),
		entClient:       entClient,
		mapper:          mapper.NewCopierMapper[adminV1.LoginPolicy, ent.LoginPolicy](),
		typeConverter:   mapper.NewEnumTypeConverter[adminV1.LoginPolicy_Type, loginpolicy.Type](adminV1.LoginPolicy_Type_name, adminV1.LoginPolicy_Type_value),
		methodConverter: mapper.NewEnumTypeConverter[adminV1.LoginPolicy_Method, loginpolicy.Method](adminV1.LoginPolicy_Method_name, adminV1.LoginPolicy_Method_value),
	}

	repo.init()

	return repo
}

func (r *LoginPolicyRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.LoginPolicyQuery, ent.LoginPolicySelect,
		ent.LoginPolicyCreate, ent.LoginPolicyCreateBulk,
		ent.LoginPolicyUpdate, ent.LoginPolicyUpdateOne,
		ent.LoginPolicyDelete,
		predicate.LoginPolicy,
		adminV1.LoginPolicy, ent.LoginPolicy,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.typeConverter.NewConverterPair())
	r.mapper.AppendConverters(r.methodConverter.NewConverterPair())
}

func (r *LoginPolicyRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().LoginPolicy.Query()
	if len(whereCond) != 0 {
		builder.Modify(whereCond...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, adminV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *LoginPolicyRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*adminV1.ListLoginPolicyResponse, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().LoginPolicy.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &adminV1.ListLoginPolicyResponse{Total: 0, Items: nil}, nil
	}

	return &adminV1.ListLoginPolicyResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *LoginPolicyRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().LoginPolicy.Query().
		Where(loginpolicy.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, adminV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *LoginPolicyRepo) Get(ctx context.Context, req *adminV1.GetLoginPolicyRequest) (*adminV1.LoginPolicy, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().LoginPolicy.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *adminV1.GetLoginPolicyRequest_Id:
		whereCond = append(whereCond, loginpolicy.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

func (r *LoginPolicyRepo) Create(ctx context.Context, req *adminV1.CreateLoginPolicyRequest) error {
	if req == nil || req.Data == nil {
		return adminV1.ErrorBadRequest("invalid request")
	}

	builder := r.entClient.Client().LoginPolicy.Create().
		SetNillableTenantID(req.Data.TenantId).
		SetNillableTargetID(req.Data.TargetId).
		SetNillableType(r.typeConverter.ToEntity(req.Data.Type)).
		SetNillableMethod(r.methodConverter.ToEntity(req.Data.Method)).
		SetNillableValue(req.Data.Value).
		SetNillableReason(req.Data.Reason).
		SetNillableCreatedBy(req.Data.CreatedBy).
		SetNillableCreatedAt(timeutil.TimestamppbToTime(req.Data.CreatedAt))

	if req.Data.CreatedAt == nil {
		builder.SetCreatedAt(time.Now())
	}

	if err := builder.Exec(ctx); err != nil {
		r.log.Errorf("insert admin login restriction failed: %s", err.Error())
		return adminV1.ErrorInternalServerError("insert admin login restriction failed")
	}

	return nil
}

func (r *LoginPolicyRepo) Update(ctx context.Context, req *adminV1.UpdateLoginPolicyRequest) error {
	if req == nil || req.Data == nil {
		return adminV1.ErrorBadRequest("invalid request")
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return err
		}
		if !exist {
			createReq := &adminV1.CreateLoginPolicyRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	builder := r.entClient.Client().Debug().LoginPolicy.Update()
	err := r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *adminV1.LoginPolicy) {
			builder.
				SetNillableTargetID(req.Data.TargetId).
				SetNillableType(r.typeConverter.ToEntity(req.Data.Type)).
				SetNillableMethod(r.methodConverter.ToEntity(req.Data.Method)).
				SetNillableValue(req.Data.Value).
				SetNillableReason(req.Data.Reason).
				SetNillableUpdatedBy(req.Data.UpdatedBy).
				SetNillableUpdatedAt(timeutil.TimestamppbToTime(req.Data.UpdatedAt))

			if req.Data.UpdatedAt == nil {
				builder.SetUpdatedAt(time.Now())
			}
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(loginpolicy.FieldID, req.GetId()))
		},
	)

	return err
}

func (r *LoginPolicyRepo) Delete(ctx context.Context, req *adminV1.DeleteLoginPolicyRequest) error {
	if req == nil {
		return adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Debug().LoginPolicy.Delete()
	_, err := r.repository.Delete(ctx, builder, func(s *sql.Selector) {
		s.Where(sql.EQ(loginpolicy.FieldID, req.GetId()))
	})
	if err != nil {
		r.log.Errorf("delete internal message categories failed: %s", err.Error())
		return adminV1.ErrorInternalServerError("delete admin login restriction failed")
	}

	return nil
}
