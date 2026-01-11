package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	pagination "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/permissionauditlog"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/timeutil"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

type PermissionAuditLogRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper *mapper.CopierMapper[permissionV1.PermissionAuditLog, ent.PermissionAuditLog]

	repository *entCrud.Repository[
		ent.PermissionAuditLogQuery, ent.PermissionAuditLogSelect,
		ent.PermissionAuditLogCreate, ent.PermissionAuditLogCreateBulk,
		ent.PermissionAuditLogUpdate, ent.PermissionAuditLogUpdateOne,
		ent.PermissionAuditLogDelete,
		predicate.PermissionAuditLog,
		permissionV1.PermissionAuditLog, ent.PermissionAuditLog,
	]
}

func NewPermissionAuditLogRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *PermissionAuditLogRepo {
	repo := &PermissionAuditLogRepo{
		log:       ctx.NewLoggerHelper("admin-operation-log/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[permissionV1.PermissionAuditLog, ent.PermissionAuditLog](),
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
		permissionV1.PermissionAuditLog, ent.PermissionAuditLog,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

func (r *PermissionAuditLogRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().PermissionAuditLog.Query()
	if len(whereCond) != 0 {
		builder.Modify(whereCond...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, permissionV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *PermissionAuditLogRepo) List(ctx context.Context, req *pagination.PagingRequest) (*permissionV1.ListPermissionAuditLogResponse, error) {
	if req == nil {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().PermissionAuditLog.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &permissionV1.ListPermissionAuditLogResponse{Total: 0, Items: nil}, nil
	}

	return &permissionV1.ListPermissionAuditLogResponse{
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
		return false, permissionV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *PermissionAuditLogRepo) Get(ctx context.Context, req *permissionV1.GetPermissionAuditLogRequest) (*permissionV1.PermissionAuditLog, error) {
	if req == nil {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().PermissionAuditLog.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *permissionV1.GetPermissionAuditLogRequest_Id:
		whereCond = append(whereCond, permissionauditlog.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

func (r *PermissionAuditLogRepo) Create(ctx context.Context, req *permissionV1.CreatePermissionAuditLogRequest) error {
	if req == nil || req.Data == nil {
		return permissionV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().PermissionAuditLog.
		Create().
		SetNillableOperatorID(req.Data.OperatorId).
		SetNillableTargetID(req.Data.TargetId).
		SetNillableTargetType(req.Data.TargetType).
		SetNillableAction(req.Data.Action).
		SetNillableOldValue(req.Data.OldValue).
		SetNillableNewValue(req.Data.NewValue).
		SetIPAddress(req.Data.GetIpAddress()).
		SetNillableTenantID(req.Data.TenantId).
		SetNillableCreatedAt(timeutil.TimestamppbToTime(req.Data.CreatedAt))

	if req.Data.CreatedAt == nil {
		builder.SetCreatedAt(time.Now())
	}

	err := builder.Exec(ctx)
	if err != nil {
		r.log.Errorf("insert permission audit log failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("insert permission audit log failed")
	}

	return err
}
