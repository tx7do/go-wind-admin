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
	"go-wind-admin/app/admin/service/internal/data/ent/operationauditlog"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
)

type OperationAuditLogRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper *mapper.CopierMapper[auditV1.OperationAuditLog, ent.OperationAuditLog]

	actionTypeConverter     *mapper.EnumTypeConverter[auditV1.OperationAuditLog_ActionType, operationauditlog.Action]
	sensitiveLevelConverter *mapper.EnumTypeConverter[auditV1.SensitiveLevel, operationauditlog.SensitiveLevel]

	repository *entCrud.Repository[
		ent.OperationAuditLogQuery, ent.OperationAuditLogSelect,
		ent.OperationAuditLogCreate, ent.OperationAuditLogCreateBulk,
		ent.OperationAuditLogUpdate, ent.OperationAuditLogUpdateOne,
		ent.OperationAuditLogDelete,
		predicate.OperationAuditLog, auditV1.OperationAuditLog, ent.OperationAuditLog,
	]
}

func NewOperationAuditLogRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *OperationAuditLogRepo {
	repo := &OperationAuditLogRepo{
		log:       ctx.NewLoggerHelper("operation-audit-log/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[auditV1.OperationAuditLog, ent.OperationAuditLog](),
		actionTypeConverter: mapper.NewEnumTypeConverter[auditV1.OperationAuditLog_ActionType, operationauditlog.Action](
			auditV1.OperationAuditLog_ActionType_name, auditV1.OperationAuditLog_ActionType_value,
		),
		sensitiveLevelConverter: mapper.NewEnumTypeConverter[auditV1.SensitiveLevel, operationauditlog.SensitiveLevel](
			auditV1.SensitiveLevel_name, auditV1.SensitiveLevel_value,
		),
	}

	repo.init()

	return repo
}

func (r *OperationAuditLogRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.OperationAuditLogQuery, ent.OperationAuditLogSelect,
		ent.OperationAuditLogCreate, ent.OperationAuditLogCreateBulk,
		ent.OperationAuditLogUpdate, ent.OperationAuditLogUpdateOne,
		ent.OperationAuditLogDelete,
		predicate.OperationAuditLog, auditV1.OperationAuditLog, ent.OperationAuditLog,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.actionTypeConverter.NewConverterPair())
	r.mapper.AppendConverters(r.sensitiveLevelConverter.NewConverterPair())
}

func (r *OperationAuditLogRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().OperationAuditLog.Query()
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

func (r *OperationAuditLogRepo) List(ctx context.Context, req *pagination.PagingRequest) (*auditV1.ListOperationAuditLogResponse, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().OperationAuditLog.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &auditV1.ListOperationAuditLogResponse{Total: 0, Items: nil}, nil
	}

	return &auditV1.ListOperationAuditLogResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *OperationAuditLogRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().OperationAuditLog.Query().
		Where(operationauditlog.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, adminV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *OperationAuditLogRepo) Get(ctx context.Context, req *auditV1.GetOperationAuditLogRequest) (*auditV1.OperationAuditLog, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Debug().OperationAuditLog.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *auditV1.GetOperationAuditLogRequest_Id:
		whereCond = append(whereCond, operationauditlog.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

func (r *OperationAuditLogRepo) Create(ctx context.Context, req *auditV1.CreateOperationAuditLogRequest) error {
	if req == nil || req.Data == nil {
		return adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().OperationAuditLog.
		Create().
		SetNillableTenantID(req.Data.TenantId).
		SetNillableUserID(req.Data.UserId).
		SetNillableUsername(req.Data.Username).
		SetNillableResourceType(req.Data.ResourceType).
		SetNillableResourceID(req.Data.RequestId).
		SetNillableAction(r.actionTypeConverter.ToEntity(req.Data.Action)).
		SetNillableBeforeData(req.Data.BeforeData).
		SetNillableAfterData(req.Data.AfterData).
		SetNillableSensitiveLevel(r.sensitiveLevelConverter.ToEntity(req.Data.SensitiveLevel)).
		SetNillableRequestID(req.Data.RequestId).
		SetNillableTraceID(req.Data.TraceId).
		SetNillableSuccess(req.Data.Success).
		SetNillableFailureReason(req.Data.FailureReason).
		SetNillableIPAddress(req.Data.IpAddress).
		SetGeoLocation(req.Data.GeoLocation).
		SetNillableLogHash(req.Data.LogHash).
		SetSignature(req.Data.Signature).
		SetNillableCreatedAt(timeutil.TimestamppbToTime(req.Data.CreatedAt))

	if req.Data.CreatedAt == nil {
		builder.SetCreatedAt(time.Now())
	}

	if err := builder.Exec(ctx); err != nil {
		r.log.Errorf("insert operation audit log failed: %s", err.Error())
		return adminV1.ErrorInternalServerError("insert operation audit log failed")
	}

	return nil
}
