package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/permissionauditlog"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
)

type PermissionAuditLogRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper              *mapper.CopierMapper[auditV1.PermissionAuditLog, ent.PermissionAuditLog]
	actionTypeConverter *mapper.EnumTypeConverter[auditV1.PermissionAuditLog_ActionType, permissionauditlog.Action]

	repository *entCrud.Repository[
		ent.PermissionAuditLogQuery, ent.PermissionAuditLogSelect,
		ent.PermissionAuditLogCreate, ent.PermissionAuditLogCreateBulk,
		ent.PermissionAuditLogUpdate, ent.PermissionAuditLogUpdateOne,
		ent.PermissionAuditLogDelete,
		predicate.PermissionAuditLog,
		auditV1.PermissionAuditLog, ent.PermissionAuditLog,
	]
}

func NewPermissionAuditLogRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *PermissionAuditLogRepo {
	repo := &PermissionAuditLogRepo{
		log:       ctx.NewLoggerHelper("permission-audit-log/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[auditV1.PermissionAuditLog, ent.PermissionAuditLog](),
		actionTypeConverter: mapper.NewEnumTypeConverter[auditV1.PermissionAuditLog_ActionType, permissionauditlog.Action](
			auditV1.PermissionAuditLog_ActionType_name, auditV1.PermissionAuditLog_ActionType_value,
		),
	}

	repo.init()

	return repo
}

func (r *PermissionAuditLogRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.PermissionAuditLogQuery, ent.PermissionAuditLogSelect,
		ent.PermissionAuditLogCreate, ent.PermissionAuditLogCreateBulk,
		ent.PermissionAuditLogUpdate, ent.PermissionAuditLogUpdateOne,
		ent.PermissionAuditLogDelete,
		predicate.PermissionAuditLog,
		auditV1.PermissionAuditLog, ent.PermissionAuditLog,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.actionTypeConverter.NewConverterPair())
}

func (r *PermissionAuditLogRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().PermissionAuditLog.Query()
	if len(whereCond) != 0 {
		builder.Modify(whereCond...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, auditV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *PermissionAuditLogRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*auditV1.ListPermissionAuditLogResponse, error) {
	if req == nil {
		return nil, auditV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().PermissionAuditLog.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &auditV1.ListPermissionAuditLogResponse{Total: 0, Items: nil}, nil
	}

	return &auditV1.ListPermissionAuditLogResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *PermissionAuditLogRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().PermissionAuditLog.Query().
		Where(permissionauditlog.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, auditV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *PermissionAuditLogRepo) Get(ctx context.Context, req *auditV1.GetPermissionAuditLogRequest) (*auditV1.PermissionAuditLog, error) {
	if req == nil {
		return nil, auditV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().PermissionAuditLog.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *auditV1.GetPermissionAuditLogRequest_Id:
		whereCond = append(whereCond, permissionauditlog.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

func (r *PermissionAuditLogRepo) Create(ctx context.Context, req *auditV1.CreatePermissionAuditLogRequest) error {
	if req == nil || req.Data == nil {
		return auditV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().PermissionAuditLog.Create().
		SetNillableTenantID(req.Data.TenantId).
		SetNillableOperatorID(req.Data.OperatorId).
		SetNillableTargetID(req.Data.TargetId).
		SetNillableTargetType(req.Data.TargetType).
		SetNillableAction(r.actionTypeConverter.ToEntity(req.Data.Action)).
		SetNillableOldValue(req.Data.OldValue).
		SetNillableNewValue(req.Data.NewValue).
		SetIPAddress(req.Data.GetIpAddress()).
		SetRequestID(req.Data.GetRequestId()).
		SetReason(req.Data.GetReason()).
		SetNillableLogHash(req.Data.LogHash).
		SetSignature(req.Data.Signature).
		SetCreatedAt(time.Now())

	err := builder.Exec(ctx)
	if err != nil {
		r.log.Errorf("insert permission audit log failed: %s", err.Error())
		return auditV1.ErrorInternalServerError("insert permission audit log failed")
	}

	return err
}
