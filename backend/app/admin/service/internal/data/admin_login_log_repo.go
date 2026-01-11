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
	"go-wind-admin/app/admin/service/internal/data/ent/adminloginlog"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
)

type AdminLoginLogRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper *mapper.CopierMapper[adminV1.AdminLoginLog, ent.AdminLoginLog]

	repository *entCrud.Repository[
		ent.AdminLoginLogQuery, ent.AdminLoginLogSelect,
		ent.AdminLoginLogCreate, ent.AdminLoginLogCreateBulk,
		ent.AdminLoginLogUpdate, ent.AdminLoginLogUpdateOne,
		ent.AdminLoginLogDelete,
		predicate.AdminLoginLog, adminV1.AdminLoginLog, ent.AdminLoginLog,
	]
}

func NewAdminLoginLogRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *AdminLoginLogRepo {
	repo := &AdminLoginLogRepo{
		log:       ctx.NewLoggerHelper("admin-login-log/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[adminV1.AdminLoginLog, ent.AdminLoginLog](),
	}

	repo.init()

	return repo
}

func (r *AdminLoginLogRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.AdminLoginLogQuery, ent.AdminLoginLogSelect,
		ent.AdminLoginLogCreate, ent.AdminLoginLogCreateBulk,
		ent.AdminLoginLogUpdate, ent.AdminLoginLogUpdateOne,
		ent.AdminLoginLogDelete,
		predicate.AdminLoginLog, adminV1.AdminLoginLog, ent.AdminLoginLog,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

func (r *AdminLoginLogRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().AdminLoginLog.Query()
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

func (r *AdminLoginLogRepo) List(ctx context.Context, req *pagination.PagingRequest) (*adminV1.ListAdminLoginLogResponse, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().AdminLoginLog.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &adminV1.ListAdminLoginLogResponse{Total: 0, Items: nil}, nil
	}

	return &adminV1.ListAdminLoginLogResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *AdminLoginLogRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().AdminLoginLog.Query().
		Where(adminloginlog.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, adminV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *AdminLoginLogRepo) Get(ctx context.Context, req *adminV1.GetAdminLoginLogRequest) (*adminV1.AdminLoginLog, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Debug().AdminLoginLog.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *adminV1.GetAdminLoginLogRequest_Id:
		whereCond = append(whereCond, adminloginlog.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

func (r *AdminLoginLogRepo) Create(ctx context.Context, req *adminV1.CreateAdminLoginLogRequest) error {
	if req == nil || req.Data == nil {
		return adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().AdminLoginLog.
		Create().
		SetNillableLoginIP(req.Data.LoginIp).
		SetNillableLoginMAC(req.Data.LoginMac).
		SetNillableUserAgent(req.Data.UserAgent).
		SetNillableBrowserName(req.Data.BrowserName).
		SetNillableBrowserVersion(req.Data.BrowserVersion).
		SetNillableClientID(req.Data.ClientId).
		SetNillableClientName(req.Data.ClientName).
		SetNillableOsName(req.Data.OsName).
		SetNillableOsVersion(req.Data.OsVersion).
		SetNillableUserID(req.Data.UserId).
		SetNillableUsername(req.Data.Username).
		SetNillableStatusCode(req.Data.StatusCode).
		SetNillableSuccess(req.Data.Success).
		SetNillableReason(req.Data.Reason).
		SetNillableLocation(req.Data.Location).
		SetNillableLoginTime(timeutil.TimestamppbToTime(req.Data.LoginTime)).
		SetNillableCreatedAt(timeutil.TimestamppbToTime(req.Data.CreatedAt))

	if req.Data.LoginTime == nil {
		builder.SetLoginTime(time.Now())
	}

	if req.Data.CreatedAt == nil {
		builder.SetCreatedAt(time.Now())
	}

	if err := builder.Exec(ctx); err != nil {
		r.log.Errorf("insert admin login log failed: %s", err.Error())
		return adminV1.ErrorInternalServerError("insert admin login log failed")
	}

	return nil
}
