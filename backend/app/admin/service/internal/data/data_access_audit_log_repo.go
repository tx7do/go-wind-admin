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
	"go-wind-admin/app/admin/service/internal/data/ent/dataaccessauditlog"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
)

type DataAccessAuditLogRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper *mapper.CopierMapper[auditV1.DataAccessAuditLog, ent.DataAccessAuditLog]

	accessTypeConverter     *mapper.EnumTypeConverter[auditV1.DataAccessAuditLog_AccessType, dataaccessauditlog.AccessType]
	sensitiveLevelConverter *mapper.EnumTypeConverter[auditV1.SensitiveLevel, dataaccessauditlog.SensitiveLevel]

	repository *entCrud.Repository[
		ent.DataAccessAuditLogQuery, ent.DataAccessAuditLogSelect,
		ent.DataAccessAuditLogCreate, ent.DataAccessAuditLogCreateBulk,
		ent.DataAccessAuditLogUpdate, ent.DataAccessAuditLogUpdateOne,
		ent.DataAccessAuditLogDelete,
		predicate.DataAccessAuditLog, auditV1.DataAccessAuditLog, ent.DataAccessAuditLog,
	]
}

func NewDataAccessAuditLogRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *DataAccessAuditLogRepo {
	repo := &DataAccessAuditLogRepo{
		log:       ctx.NewLoggerHelper("data-access-audit-log/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[auditV1.DataAccessAuditLog, ent.DataAccessAuditLog](),
		accessTypeConverter: mapper.NewEnumTypeConverter[auditV1.DataAccessAuditLog_AccessType, dataaccessauditlog.AccessType](
			auditV1.DataAccessAuditLog_AccessType_name, auditV1.DataAccessAuditLog_AccessType_value,
		),
		sensitiveLevelConverter: mapper.NewEnumTypeConverter[auditV1.SensitiveLevel, dataaccessauditlog.SensitiveLevel](
			auditV1.SensitiveLevel_name, auditV1.SensitiveLevel_value,
		),
	}

	repo.init()

	return repo
}

func (r *DataAccessAuditLogRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.DataAccessAuditLogQuery, ent.DataAccessAuditLogSelect,
		ent.DataAccessAuditLogCreate, ent.DataAccessAuditLogCreateBulk,
		ent.DataAccessAuditLogUpdate, ent.DataAccessAuditLogUpdateOne,
		ent.DataAccessAuditLogDelete,
		predicate.DataAccessAuditLog, auditV1.DataAccessAuditLog, ent.DataAccessAuditLog,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.accessTypeConverter.NewConverterPair())
	r.mapper.AppendConverters(r.sensitiveLevelConverter.NewConverterPair())
}

func (r *DataAccessAuditLogRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().DataAccessAuditLog.Query()
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

func (r *DataAccessAuditLogRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*auditV1.ListDataAccessAuditLogResponse, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().DataAccessAuditLog.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &auditV1.ListDataAccessAuditLogResponse{Total: 0, Items: nil}, nil
	}

	return &auditV1.ListDataAccessAuditLogResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *DataAccessAuditLogRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().DataAccessAuditLog.Query().
		Where(dataaccessauditlog.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, adminV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *DataAccessAuditLogRepo) Get(ctx context.Context, req *auditV1.GetDataAccessAuditLogRequest) (*auditV1.DataAccessAuditLog, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Debug().DataAccessAuditLog.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *auditV1.GetDataAccessAuditLogRequest_Id:
		whereCond = append(whereCond, dataaccessauditlog.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

func (r *DataAccessAuditLogRepo) Create(ctx context.Context, req *auditV1.CreateDataAccessAuditLogRequest) error {
	if req == nil || req.Data == nil {
		return adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().DataAccessAuditLog.Create().
		SetNillableTenantID(req.Data.TenantId).
		SetNillableUserID(req.Data.UserId).
		SetNillableUsername(req.Data.Username).
		SetNillableIPAddress(req.Data.IpAddress).
		SetNillableRequestID(req.Data.RequestId).
		SetNillableDataSource(req.Data.DataSource).
		SetNillableTableName(req.Data.TableName).
		SetNillableDataID(req.Data.DataId).
		SetNillableAccessType(r.accessTypeConverter.ToEntity(req.Data.AccessType)).
		SetNillableSQLDigest(req.Data.SqlDigest).
		SetNillableSQLText(req.Data.SqlText).
		SetNillableAffectedRows(req.Data.AffectedRows).
		SetNillableLatencyMs(req.Data.LatencyMs).
		SetNillableSuccess(req.Data.Success).
		SetNillableSensitiveLevel(r.sensitiveLevelConverter.ToEntity(req.Data.SensitiveLevel)).
		SetNillableDataMasked(req.Data.DataMasked).
		SetNillableMaskingRules(req.Data.MaskingRules).
		SetNillableBusinessPurpose(req.Data.BusinessPurpose).
		SetNillableDataCategory(req.Data.DataCategory).
		SetNillableDbUser(req.Data.DbUser).
		SetNillableLogHash(req.Data.LogHash).
		SetSignature(req.Data.Signature).
		SetCreatedAt(time.Now())

	if err := builder.Exec(ctx); err != nil {
		r.log.Errorf("insert data access audit log failed: %s", err.Error())
		return adminV1.ErrorInternalServerError("insert data access audit log failed")
	}

	return nil
}
