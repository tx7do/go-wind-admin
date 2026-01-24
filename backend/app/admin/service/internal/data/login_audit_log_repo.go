package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/loginauditlog"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
)

type LoginAuditLogRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper *mapper.CopierMapper[auditV1.LoginAuditLog, ent.LoginAuditLog]

	statusConverter      *mapper.EnumTypeConverter[auditV1.LoginAuditLog_Status, loginauditlog.Status]
	actionTypeConverter  *mapper.EnumTypeConverter[auditV1.LoginAuditLog_ActionType, loginauditlog.ActionType]
	riskLevelConverter   *mapper.EnumTypeConverter[auditV1.LoginAuditLog_RiskLevel, loginauditlog.RiskLevel]
	loginMethodConverter *mapper.EnumTypeConverter[auditV1.LoginAuditLog_LoginMethod, loginauditlog.LoginMethod]

	repository *entCrud.Repository[
		ent.LoginAuditLogQuery, ent.LoginAuditLogSelect,
		ent.LoginAuditLogCreate, ent.LoginAuditLogCreateBulk,
		ent.LoginAuditLogUpdate, ent.LoginAuditLogUpdateOne,
		ent.LoginAuditLogDelete,
		predicate.LoginAuditLog, auditV1.LoginAuditLog, ent.LoginAuditLog,
	]
}

func NewLoginAuditLogRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *LoginAuditLogRepo {
	repo := &LoginAuditLogRepo{
		log:       ctx.NewLoggerHelper("login-audit-log/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[auditV1.LoginAuditLog, ent.LoginAuditLog](),
		statusConverter: mapper.NewEnumTypeConverter[auditV1.LoginAuditLog_Status, loginauditlog.Status](
			auditV1.LoginAuditLog_Status_name, auditV1.LoginAuditLog_Status_value,
		),
		actionTypeConverter: mapper.NewEnumTypeConverter[auditV1.LoginAuditLog_ActionType, loginauditlog.ActionType](
			auditV1.LoginAuditLog_ActionType_name, auditV1.LoginAuditLog_ActionType_value,
		),
		riskLevelConverter: mapper.NewEnumTypeConverter[auditV1.LoginAuditLog_RiskLevel, loginauditlog.RiskLevel](
			auditV1.LoginAuditLog_RiskLevel_name, auditV1.LoginAuditLog_RiskLevel_value,
		),
		loginMethodConverter: mapper.NewEnumTypeConverter[auditV1.LoginAuditLog_LoginMethod, loginauditlog.LoginMethod](
			auditV1.LoginAuditLog_LoginMethod_name, auditV1.LoginAuditLog_LoginMethod_value,
		),
	}

	repo.init()

	return repo
}

func (r *LoginAuditLogRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.LoginAuditLogQuery, ent.LoginAuditLogSelect,
		ent.LoginAuditLogCreate, ent.LoginAuditLogCreateBulk,
		ent.LoginAuditLogUpdate, ent.LoginAuditLogUpdateOne,
		ent.LoginAuditLogDelete,
		predicate.LoginAuditLog, auditV1.LoginAuditLog, ent.LoginAuditLog,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
	r.mapper.AppendConverters(r.actionTypeConverter.NewConverterPair())
	r.mapper.AppendConverters(r.riskLevelConverter.NewConverterPair())
}

func (r *LoginAuditLogRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().LoginAuditLog.Query()
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

func (r *LoginAuditLogRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*auditV1.ListLoginAuditLogResponse, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().LoginAuditLog.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &auditV1.ListLoginAuditLogResponse{Total: 0, Items: nil}, nil
	}

	return &auditV1.ListLoginAuditLogResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *LoginAuditLogRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().LoginAuditLog.Query().
		Where(loginauditlog.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, adminV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *LoginAuditLogRepo) Get(ctx context.Context, req *auditV1.GetLoginAuditLogRequest) (*auditV1.LoginAuditLog, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Debug().LoginAuditLog.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *auditV1.GetLoginAuditLogRequest_Id:
		whereCond = append(whereCond, loginauditlog.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

func (r *LoginAuditLogRepo) Create(ctx context.Context, req *auditV1.CreateLoginAuditLogRequest) error {
	if req == nil || req.Data == nil {
		return adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().LoginAuditLog.Create().
		SetNillableTenantID(req.Data.TenantId).
		SetNillableUserID(req.Data.UserId).
		SetNillableUsername(req.Data.Username).
		SetNillableIPAddress(req.Data.IpAddress).
		SetGeoLocation(req.Data.GeoLocation).
		SetNillableSessionID(req.Data.SessionId).
		SetDeviceInfo(req.Data.DeviceInfo).
		SetNillableRequestID(req.Data.RequestId).
		SetNillableTraceID(req.Data.TraceId).
		SetNillableActionType(r.actionTypeConverter.ToEntity(req.Data.ActionType)).
		SetNillableStatus(r.statusConverter.ToEntity(req.Data.Status)).
		SetNillableLoginMethod(r.loginMethodConverter.ToEntity(req.Data.LoginMethod)).
		SetNillableFailureReason(req.Data.FailureReason).
		SetNillableMfaStatus(req.Data.MfaStatus).
		SetNillableRiskScore(req.Data.RiskScore).
		SetNillableRiskLevel(r.riskLevelConverter.ToEntity(req.Data.RiskLevel)).
		SetRiskFactors(req.Data.RiskFactors).
		SetNillableLogHash(req.Data.LogHash).
		SetSignature(req.Data.Signature).
		SetCreatedAt(time.Now())

	if err := builder.Exec(ctx); err != nil {
		r.log.Errorf("insert login audit log failed: %s", err.Error())
		return adminV1.ErrorInternalServerError("insert login audit log failed")
	}

	return nil
}
