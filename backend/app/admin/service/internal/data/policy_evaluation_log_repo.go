package data

import (
	"context"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/api"
	"go-wind-admin/app/admin/service/internal/data/ent/permissionapi"
	"go-wind-admin/app/admin/service/internal/data/ent/permissionpolicy"
	"go-wind-admin/app/admin/service/internal/data/ent/policyevaluationlog"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"
	"go-wind-admin/app/admin/service/internal/data/ent/role"
	"go-wind-admin/app/admin/service/internal/data/ent/rolepermission"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
)

type PolicyEvaluationLogRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *bLogger.Helper

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
		log:       ctx.NewLoggerHelper("policy-evaluation-log/repo/admin-service"),
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
		r.log.Errorf(ctx, "query count failed: %s", err.Error())
		return 0, permissionV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *PolicyEvaluationLogRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*permissionV1.ListPolicyEvaluationLogResponse, error) {
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
		r.log.Errorf(ctx, "query exist failed: %s", err.Error())
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
		SetNillableTenantID(req.Data.TenantId).
		SetUserID(req.Data.GetUserId()).
		SetMembershipID(req.Data.GetMembershipId()).
		SetPermissionID(req.Data.GetPermissionId()).
		SetNillablePolicyID(req.Data.PolicyId).
		SetNillableRequestPath(req.Data.RequestPath).
		SetNillableRequestMethod(req.Data.RequestMethod).
		SetNillableResult(req.Data.Result).
		SetNillableEffectDetails(req.Data.EffectDetails).
		SetNillableScopeSQL(req.Data.ScopeSql).
		SetIPAddress(req.Data.GetIpAddress()).
		SetNillableTraceID(req.Data.TraceId).
		SetNillableEvaluationContext(req.Data.EvaluationContext).
		SetNillableLogHash(req.Data.LogHash).
		SetSignature(req.Data.Signature).
		SetCreatedAt(time.Now())

	err := builder.Exec(ctx)
	if err != nil {
		r.log.Errorf(ctx, "insert policy evaluation log failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("insert policy evaluation log failed")
	}

	return err
}

// ResolvePermissionPolicyByRoute 反查评估命中的权限点与挂接策略：
// (角色码, 路径模板, HTTP方法) → sys_apis → sys_permission_apis → sys_permissions
// （角色经 sys_role_permissions 持有该权限）→ sys_permission_policies 按 eval_order 首条。
// 任一环缺数据（API 未登记 / 角色未授权 / 无策略行）对应 ID 返回 0，不报错——
// 评估埋点尽力而为，绝不影响鉴权主流程。调用方需自行做 TTL 缓存。
// 四步各走索引的单表查询（曾试过一条 Modify 联表，pgx 参数类型推断 42P18）。
func (r *PolicyEvaluationLogRepo) ResolvePermissionPolicyByRoute(ctx context.Context, roleCode, path, method string) (permissionID uint32, policyID uint32) {
	if roleCode == "" || path == "" || method == "" {
		return 0, 0
	}
	client := r.entClient.Client()

	apiID, err := client.Api.Query().
		Where(api.PathEQ(path), api.MethodEQ(strings.ToUpper(method))).
		FirstID(ctx)
	if err != nil {
		return 0, 0
	}

	pa, err := client.PermissionApi.Query().
		Where(permissionapi.APIIDEQ(apiID)).
		First(ctx)
	if err != nil {
		return 0, 0
	}
	if pa.PermissionID != nil {
		permissionID = *pa.PermissionID
	}

	// 校验该角色确实持有此权限（经 sys_role_permissions）
	roleID, err := client.Role.Query().
		Where(role.CodeEQ(roleCode)).
		FirstID(ctx)
	if err != nil {
		return 0, 0
	}
	held, err := client.RolePermission.Query().
		Where(rolepermission.RoleIDEQ(roleID), rolepermission.PermissionIDEQ(permissionID)).
		Exist(ctx)
	if err != nil || !held {
		return 0, 0
	}

	policyID, err = client.PermissionPolicy.Query().
		Where(permissionpolicy.PermissionIDEQ(permissionID)).
		Order(ent.Asc(permissionpolicy.FieldEvalOrder)).
		FirstID(ctx)
	if err != nil {
		return 0, permissionID
	}
	return permissionID, policyID
}
