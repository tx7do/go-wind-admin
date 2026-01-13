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
	"go-wind-admin/app/admin/service/internal/data/ent/apiauditlog"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
)

type ApiAuditLogRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper *mapper.CopierMapper[auditV1.ApiAuditLog, ent.ApiAuditLog]

	repository *entCrud.Repository[
		ent.ApiAuditLogQuery, ent.ApiAuditLogSelect,
		ent.ApiAuditLogCreate, ent.ApiAuditLogCreateBulk,
		ent.ApiAuditLogUpdate, ent.ApiAuditLogUpdateOne,
		ent.ApiAuditLogDelete,
		predicate.ApiAuditLog,
		auditV1.ApiAuditLog, ent.ApiAuditLog,
	]
}

func NewApiAuditLogRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *ApiAuditLogRepo {
	repo := &ApiAuditLogRepo{
		log:       ctx.NewLoggerHelper("api-audit-log/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[auditV1.ApiAuditLog, ent.ApiAuditLog](),
	}

	repo.init()

	return repo
}

func (r *ApiAuditLogRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.ApiAuditLogQuery, ent.ApiAuditLogSelect,
		ent.ApiAuditLogCreate, ent.ApiAuditLogCreateBulk,
		ent.ApiAuditLogUpdate, ent.ApiAuditLogUpdateOne,
		ent.ApiAuditLogDelete,
		predicate.ApiAuditLog,
		auditV1.ApiAuditLog, ent.ApiAuditLog,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.NewFloatSecondConverterPair())
}

func (r *ApiAuditLogRepo) NewFloatSecondConverterPair() []copier.TypeConverter {
	srcType := durationpb.New(0)
	dstType := trans.Ptr(float64(0))

	fromFn := timeutil.DurationpbToSecond
	toFn := timeutil.SecondToDurationpb

	return copierutil.NewGenericTypeConverterPair(srcType, dstType, fromFn, toFn)
}

func (r *ApiAuditLogRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().ApiAuditLog.Query()
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

func (r *ApiAuditLogRepo) List(ctx context.Context, req *pagination.PagingRequest) (*auditV1.ListApiAuditLogResponse, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().ApiAuditLog.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &auditV1.ListApiAuditLogResponse{Total: 0, Items: nil}, nil
	}

	return &auditV1.ListApiAuditLogResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *ApiAuditLogRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().ApiAuditLog.Query().
		Where(apiauditlog.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, adminV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *ApiAuditLogRepo) Get(ctx context.Context, req *auditV1.GetApiAuditLogRequest) (*auditV1.ApiAuditLog, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().ApiAuditLog.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *auditV1.GetApiAuditLogRequest_Id:
		whereCond = append(whereCond, apiauditlog.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

func (r *ApiAuditLogRepo) Create(ctx context.Context, req *auditV1.CreateApiAuditLogRequest) error {
	if req == nil || req.Data == nil {
		return adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().ApiAuditLog.
		Create().
		SetNillableTenantID(req.Data.TenantId).
		SetNillableUserID(req.Data.UserId).
		SetNillableUsername(req.Data.Username).
		SetNillableIPAddress(req.Data.IpAddress).
		SetGeoLocation(req.Data.GeoLocation).
		SetDeviceInfo(req.Data.DeviceInfo).
		SetNillableReferer(req.Data.Referer).
		SetNillableAppVersion(req.Data.AppVersion).
		SetNillableHTTPMethod(req.Data.HttpMethod).
		SetNillablePath(req.Data.Path).
		SetNillableRequestURI(req.Data.RequestUri).
		SetNillableAPIModule(req.Data.ApiModule).
		SetNillableAPIOperation(req.Data.ApiOperation).
		SetNillableAPIDescription(req.Data.ApiDescription).
		SetNillableRequestID(req.Data.RequestId).
		SetNillableTraceID(req.Data.TraceId).
		SetNillableSpanID(req.Data.SpanId).
		SetNillableLatencyMs(req.Data.LatencyMs).
		SetNillableSuccess(req.Data.Success).
		SetNillableStatusCode(req.Data.StatusCode).
		SetNillableReason(req.Data.Reason).
		SetNillableRequestHeader(req.Data.RequestHeader).
		SetNillableRequestBody(req.Data.RequestBody).
		SetNillableResponse(req.Data.Response).
		SetNillableLogHash(req.Data.LogHash).
		SetSignature(req.Data.Signature).
		SetNillableCreatedAt(timeutil.TimestamppbToTime(req.Data.CreatedAt))

	if req.Data.CreatedAt == nil {
		builder.SetCreatedAt(time.Now())
	}

	err := builder.Exec(ctx)
	if err != nil {
		r.log.Errorf("insert api audit log failed: %s", err.Error())
		return adminV1.ErrorInternalServerError("insert api audit log failed")
	}

	return err
}
