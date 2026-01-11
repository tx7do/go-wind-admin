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
	"go-wind-admin/app/admin/service/internal/data/ent/adminloginrestriction"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
)

type AdminLoginRestrictionRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper          *mapper.CopierMapper[adminV1.AdminLoginRestriction, ent.AdminLoginRestriction]
	typeConverter   *mapper.EnumTypeConverter[adminV1.AdminLoginRestriction_Type, adminloginrestriction.Type]
	methodConverter *mapper.EnumTypeConverter[adminV1.AdminLoginRestriction_Method, adminloginrestriction.Method]

	repository *entCrud.Repository[
		ent.AdminLoginRestrictionQuery, ent.AdminLoginRestrictionSelect,
		ent.AdminLoginRestrictionCreate, ent.AdminLoginRestrictionCreateBulk,
		ent.AdminLoginRestrictionUpdate, ent.AdminLoginRestrictionUpdateOne,
		ent.AdminLoginRestrictionDelete,
		predicate.AdminLoginRestriction,
		adminV1.AdminLoginRestriction, ent.AdminLoginRestriction,
	]
}

func NewAdminLoginRestrictionRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *AdminLoginRestrictionRepo {
	repo := &AdminLoginRestrictionRepo{
		log:             ctx.NewLoggerHelper("admin-login-restriction/repo/admin-service"),
		entClient:       entClient,
		mapper:          mapper.NewCopierMapper[adminV1.AdminLoginRestriction, ent.AdminLoginRestriction](),
		typeConverter:   mapper.NewEnumTypeConverter[adminV1.AdminLoginRestriction_Type, adminloginrestriction.Type](adminV1.AdminLoginRestriction_Type_name, adminV1.AdminLoginRestriction_Type_value),
		methodConverter: mapper.NewEnumTypeConverter[adminV1.AdminLoginRestriction_Method, adminloginrestriction.Method](adminV1.AdminLoginRestriction_Method_name, adminV1.AdminLoginRestriction_Method_value),
	}

	repo.init()

	return repo
}

func (r *AdminLoginRestrictionRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.AdminLoginRestrictionQuery, ent.AdminLoginRestrictionSelect,
		ent.AdminLoginRestrictionCreate, ent.AdminLoginRestrictionCreateBulk,
		ent.AdminLoginRestrictionUpdate, ent.AdminLoginRestrictionUpdateOne,
		ent.AdminLoginRestrictionDelete,
		predicate.AdminLoginRestriction,
		adminV1.AdminLoginRestriction, ent.AdminLoginRestriction,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.typeConverter.NewConverterPair())
	r.mapper.AppendConverters(r.methodConverter.NewConverterPair())
}

func (r *AdminLoginRestrictionRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().AdminLoginRestriction.Query()
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

func (r *AdminLoginRestrictionRepo) List(ctx context.Context, req *pagination.PagingRequest) (*adminV1.ListAdminLoginRestrictionResponse, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().AdminLoginRestriction.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &adminV1.ListAdminLoginRestrictionResponse{Total: 0, Items: nil}, nil
	}

	return &adminV1.ListAdminLoginRestrictionResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *AdminLoginRestrictionRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().AdminLoginRestriction.Query().
		Where(adminloginrestriction.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, adminV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *AdminLoginRestrictionRepo) Get(ctx context.Context, req *adminV1.GetAdminLoginRestrictionRequest) (*adminV1.AdminLoginRestriction, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().AdminLoginRestriction.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *adminV1.GetAdminLoginRestrictionRequest_Id:
		whereCond = append(whereCond, adminloginrestriction.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

func (r *AdminLoginRestrictionRepo) Create(ctx context.Context, req *adminV1.CreateAdminLoginRestrictionRequest) error {
	if req == nil || req.Data == nil {
		return adminV1.ErrorBadRequest("invalid request")
	}

	builder := r.entClient.Client().AdminLoginRestriction.Create().
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

func (r *AdminLoginRestrictionRepo) Update(ctx context.Context, req *adminV1.UpdateAdminLoginRestrictionRequest) error {
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
			createReq := &adminV1.CreateAdminLoginRestrictionRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	builder := r.entClient.Client().Debug().AdminLoginRestriction.Update()
	err := r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *adminV1.AdminLoginRestriction) {
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
			s.Where(sql.EQ(adminloginrestriction.FieldID, req.GetId()))
		},
	)

	return err
}

func (r *AdminLoginRestrictionRepo) Delete(ctx context.Context, req *adminV1.DeleteAdminLoginRestrictionRequest) error {
	if req == nil {
		return adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Debug().AdminLoginRestriction.Delete()
	_, err := r.repository.Delete(ctx, builder, func(s *sql.Selector) {
		s.Where(sql.EQ(adminloginrestriction.FieldID, req.GetId()))
	})
	if err != nil {
		r.log.Errorf("delete internal message categories failed: %s", err.Error())
		return adminV1.ErrorInternalServerError("delete admin login restriction failed")
	}

	return nil
}
