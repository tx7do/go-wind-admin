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
	"go-wind-admin/app/admin/service/internal/data/ent/policyevaluationlog"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/timeutil"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

type PolicyEvaluationLogRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper *mapper.CopierMapper[permissionV1.PolicyEvaluationLog, ent.PolicyEvaluationLog]

	repository *entCrud.Repository[
		ent.PolicyEvaluationLogQuery, ent.PolicyEvaluationLogSelect,
		ent.PolicyEvaluationLogCreate, ent.PolicyEvaluationLogCreateBulk,
		ent.PolicyEvaluationLogUpdate, ent.PolicyEvaluationLogUpdateOne,
		ent.PolicyEvaluationLogDelete,
		predicate.PolicyEvaluationLog,
		permissionV1.PolicyEvaluationLog, ent.PolicyEvaluationLog,
	]
}

func NewPolicyEvaluationLogRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *PolicyEvaluationLogRepo {
	repo := &PolicyEvaluationLogRepo{
		log:       ctx.NewLoggerHelper("admin-operation-log/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[permissionV1.PolicyEvaluationLog, ent.PolicyEvaluationLog](),
	}

	repo.init()

	return repo
}

func (r *PolicyEvaluationLogRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.PolicyEvaluationLogQuery, ent.PolicyEvaluationLogSelect,
		ent.PolicyEvaluationLogCreate, ent.PolicyEvaluationLogCreateBulk,
		ent.PolicyEvaluationLogUpdate, ent.PolicyEvaluationLogUpdateOne,
		ent.PolicyEvaluationLogDelete,
		predicate.PolicyEvaluationLog,
		permissionV1.PolicyEvaluationLog, ent.PolicyEvaluationLog,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

func (r *PolicyEvaluationLogRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().PolicyEvaluationLog.Query()
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

func (r *PolicyEvaluationLogRepo) List(ctx context.Context, req *pagination.PagingRequest) (*permissionV1.ListPolicyEvaluationLogResponse, error) {
	if req == nil {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().PolicyEvaluationLog.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &permissionV1.ListPolicyEvaluationLogResponse{Total: 0, Items: nil}, nil
	}

	return &permissionV1.ListPolicyEvaluationLogResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *PolicyEvaluationLogRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().PolicyEvaluationLog.Query().
		Where(policyevaluationlog.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, permissionV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *PolicyEvaluationLogRepo) Get(ctx context.Context, req *permissionV1.GetPolicyEvaluationLogRequest) (*permissionV1.PolicyEvaluationLog, error) {
	if req == nil {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().PolicyEvaluationLog.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *permissionV1.GetPolicyEvaluationLogRequest_Id:
		whereCond = append(whereCond, policyevaluationlog.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

func (r *PolicyEvaluationLogRepo) Create(ctx context.Context, req *permissionV1.CreatePolicyEvaluationLogRequest) error {
	if req == nil || req.Data == nil {
		return permissionV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().PolicyEvaluationLog.
		Create().
		SetUserID(req.Data.GetUserId()).
		SetMembershipID(req.Data.GetMembershipId()).
		SetPermissionID(req.Data.GetPermissionId()).
		SetNillablePolicyID(req.Data.PolicyId).
		SetNillableResult(req.Data.Result).
		SetNillableScopeSQL(req.Data.ScopeSql).
		SetNillableRequestPath(req.Data.RequestPath).
		SetIPAddress(req.Data.GetIpAddress()).
		SetNillableTenantID(req.Data.TenantId).
		SetNillableCreatedAt(timeutil.TimestamppbToTime(req.Data.CreatedAt))

	if req.Data.CreatedAt == nil {
		builder.SetCreatedAt(time.Now())
	}

	err := builder.Exec(ctx)
	if err != nil {
		r.log.Errorf("insert one data failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("insert data failed")
	}

	return err
}
