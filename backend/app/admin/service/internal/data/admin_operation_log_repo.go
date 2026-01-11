package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/jinzhu/copier"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/durationpb"

	pagination "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/timeutil"
	"github.com/tx7do/go-utils/trans"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/adminoperationlog"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
)

type AdminOperationLogRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper *mapper.CopierMapper[adminV1.AdminOperationLog, ent.AdminOperationLog]

	repository *entCrud.Repository[
		ent.AdminOperationLogQuery, ent.AdminOperationLogSelect,
		ent.AdminOperationLogCreate, ent.AdminOperationLogCreateBulk,
		ent.AdminOperationLogUpdate, ent.AdminOperationLogUpdateOne,
		ent.AdminOperationLogDelete,
		predicate.AdminOperationLog,
		adminV1.AdminOperationLog, ent.AdminOperationLog,
	]
}

func NewAdminOperationLogRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *AdminOperationLogRepo {
	repo := &AdminOperationLogRepo{
		log:       ctx.NewLoggerHelper("admin-operation-log/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[adminV1.AdminOperationLog, ent.AdminOperationLog](),
	}

	repo.init()

	return repo
}

func (r *AdminOperationLogRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.AdminOperationLogQuery, ent.AdminOperationLogSelect,
		ent.AdminOperationLogCreate, ent.AdminOperationLogCreateBulk,
		ent.AdminOperationLogUpdate, ent.AdminOperationLogUpdateOne,
		ent.AdminOperationLogDelete,
		predicate.AdminOperationLog,
		adminV1.AdminOperationLog, ent.AdminOperationLog,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.NewFloatSecondConverterPair())
}

func (r *AdminOperationLogRepo) NewFloatSecondConverterPair() []copier.TypeConverter {
	srcType := durationpb.New(0)
	dstType := trans.Ptr(float64(0))

	fromFn := timeutil.DurationpbToSecond
	toFn := timeutil.SecondToDurationpb

	return copierutil.NewGenericTypeConverterPair(srcType, dstType, fromFn, toFn)
}

func (r *AdminOperationLogRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().AdminOperationLog.Query()
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

func (r *AdminOperationLogRepo) List(ctx context.Context, req *pagination.PagingRequest) (*adminV1.ListAdminOperationLogResponse, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().AdminOperationLog.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &adminV1.ListAdminOperationLogResponse{Total: 0, Items: nil}, nil
	}

	return &adminV1.ListAdminOperationLogResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *AdminOperationLogRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().AdminOperationLog.Query().
		Where(adminoperationlog.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, adminV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *AdminOperationLogRepo) Get(ctx context.Context, req *adminV1.GetAdminOperationLogRequest) (*adminV1.AdminOperationLog, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().AdminOperationLog.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *adminV1.GetAdminOperationLogRequest_Id:
		whereCond = append(whereCond, adminoperationlog.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

func (r *AdminOperationLogRepo) Create(ctx context.Context, req *adminV1.CreateAdminOperationLogRequest) error {
	if req == nil || req.Data == nil {
		return adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().AdminOperationLog.
		Create().
		SetNillableRequestID(req.Data.RequestId).
		SetNillableMethod(req.Data.Method).
		SetNillableOperation(req.Data.Operation).
		SetNillablePath(req.Data.Path).
		SetNillableReferer(req.Data.Referer).
		SetNillableRequestURI(req.Data.RequestUri).
		SetNillableRequestBody(req.Data.RequestBody).
		SetNillableRequestHeader(req.Data.RequestHeader).
		SetNillableResponse(req.Data.Response).
		SetNillableCostTime(timeutil.DurationpbToSecond(req.Data.CostTime)).
		SetNillableUserID(req.Data.UserId).
		SetNillableUsername(req.Data.Username).
		SetNillableClientIP(req.Data.ClientIp).
		SetNillableUserAgent(req.Data.UserAgent).
		SetNillableBrowserName(req.Data.BrowserName).
		SetNillableBrowserVersion(req.Data.BrowserVersion).
		SetNillableClientID(req.Data.ClientId).
		SetNillableClientName(req.Data.ClientName).
		SetNillableOsName(req.Data.OsName).
		SetNillableOsVersion(req.Data.OsVersion).
		SetNillableStatusCode(req.Data.StatusCode).
		SetNillableSuccess(req.Data.Success).
		SetNillableReason(req.Data.Reason).
		SetNillableLocation(req.Data.Location).
		SetNillableCreatedAt(timeutil.TimestamppbToTime(req.Data.CreatedAt))

	if req.Data.CreatedAt == nil {
		builder.SetCreatedAt(time.Now())
	}

	err := builder.Exec(ctx)
	if err != nil {
		r.log.Errorf("insert admin operation log failed: %s", err.Error())
		return adminV1.ErrorInternalServerError("insert admin operation log failed")
	}

	return err
}
