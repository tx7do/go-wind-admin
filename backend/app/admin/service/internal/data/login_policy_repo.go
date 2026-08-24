package data

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/loginpolicy"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
)

type LoginPolicyRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *bLogger.Helper

	mapper          *mapper.CopierMapper[authenticationV1.LoginPolicy, ent.LoginPolicy]
	typeConverter   *mapper.EnumTypeConverter[authenticationV1.LoginPolicy_Type, loginpolicy.Type]
	methodConverter *mapper.EnumTypeConverter[authenticationV1.LoginPolicy_Method, loginpolicy.Method]

	repository *entCrud.Repository[
		ent.LoginPolicyQuery, ent.LoginPolicySelect,
		ent.LoginPolicyCreate, ent.LoginPolicyCreateBulk,
		ent.LoginPolicyUpdate, ent.LoginPolicyUpdateOne,
		ent.LoginPolicyDelete,
		predicate.LoginPolicy,
		authenticationV1.LoginPolicy, ent.LoginPolicy,
	]
}

func NewLoginPolicyRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *LoginPolicyRepo {
	repo := &LoginPolicyRepo{
		log:       ctx.NewLoggerHelper("login-policy/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[authenticationV1.LoginPolicy, ent.LoginPolicy](),
		typeConverter: mapper.NewEnumTypeConverter[authenticationV1.LoginPolicy_Type, loginpolicy.Type](
			authenticationV1.LoginPolicy_Type_name, authenticationV1.LoginPolicy_Type_value,
		),
		methodConverter: mapper.NewEnumTypeConverter[authenticationV1.LoginPolicy_Method, loginpolicy.Method](
			authenticationV1.LoginPolicy_Method_name, authenticationV1.LoginPolicy_Method_value,
		),
	}

	repo.init()

	return repo
}

func (r *LoginPolicyRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.LoginPolicyQuery, ent.LoginPolicySelect,
		ent.LoginPolicyCreate, ent.LoginPolicyCreateBulk,
		ent.LoginPolicyUpdate, ent.LoginPolicyUpdateOne,
		ent.LoginPolicyDelete,
		predicate.LoginPolicy,
		authenticationV1.LoginPolicy, ent.LoginPolicy,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.typeConverter.NewConverterPair())
	r.mapper.AppendConverters(r.methodConverter.NewConverterPair())
}

// EffectivePolicy 登录闸门用的策略条目（ent 实体的精简视图，避免服务层依赖生成代码）。
type EffectivePolicy struct {
	TargetID uint32 // 0 表示全局策略（不限定用户）
	Value    string
	Type     string // BLACK_LIST / WHITE_LIST
	Method   string // IP / MAC / REGION / TIME / DEVICE
	Reason   string
}

// ListForLogin 拉取租户内全部登录策略，供登录闸门在内存中按
// 全局（target_id 为空）/ 用户定向（target_id = userId）两批匹配。
// 策略量级小（管理配置项），全量拉取 + 内存过滤即可，无需按用户建索引。
func (r *LoginPolicyRepo) ListForLogin(ctx context.Context, tenantID uint32) ([]EffectivePolicy, error) {
	entities, err := r.entClient.Client().LoginPolicy.Query().
		Where(loginpolicy.TenantIDEQ(tenantID)).
		All(ctx)
	if err != nil {
		r.log.Errorf(ctx, "list login policies for login failed: %s", err.Error())
		return nil, fmt.Errorf("list login policies failed")
	}
	policies := make([]EffectivePolicy, 0, len(entities))
	for _, e := range entities {
		policies = append(policies, EffectivePolicy{
			TargetID: derefUint32(e.TargetID),
			Value:    derefStr(e.Value),
			Type:     derefStrP(e.Type),
			Method:   derefStrP(e.Method),
			Reason:   derefStr(e.Reason),
		})
	}
	return policies, nil
}

func derefUint32(p *uint32) uint32 {
	if p == nil {
		return 0
	}
	return *p
}

func derefStrP[T ~string](p *T) string {
	if p == nil {
		return ""
	}
	return string(*p)
}

func (r *LoginPolicyRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().LoginPolicy.Query()
	if len(whereCond) != 0 {
		builder.Modify(whereCond...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query count failed: %s", err.Error())
		return 0, adminV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *LoginPolicyRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*authenticationV1.ListLoginPolicyResponse, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().LoginPolicy.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &authenticationV1.ListLoginPolicyResponse{Total: 0, Items: nil}, nil
	}

	return &authenticationV1.ListLoginPolicyResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *LoginPolicyRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().LoginPolicy.Query().
		Where(loginpolicy.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query exist failed: %s", err.Error())
		return false, adminV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *LoginPolicyRepo) Get(ctx context.Context, req *authenticationV1.GetLoginPolicyRequest) (*authenticationV1.LoginPolicy, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().LoginPolicy.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *authenticationV1.GetLoginPolicyRequest_Id:
		whereCond = append(whereCond, loginpolicy.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

func (r *LoginPolicyRepo) Create(ctx context.Context, req *authenticationV1.CreateLoginPolicyRequest) error {
	if req == nil || req.Data == nil {
		return adminV1.ErrorBadRequest("invalid request")
	}

	builder := r.entClient.Client().LoginPolicy.Create().
		SetNillableTenantID(req.Data.TenantId).
		SetNillableTargetID(req.Data.TargetId).
		SetNillableType(r.typeConverter.ToEntity(req.Data.Type)).
		SetNillableMethod(r.methodConverter.ToEntity(req.Data.Method)).
		SetNillableValue(req.Data.Value).
		SetNillableReason(req.Data.Reason).
		SetNillableCreatedBy(req.Data.CreatedBy).
		SetCreatedAt(time.Now())

	if err := builder.Exec(ctx); err != nil {
		r.log.Errorf(ctx, "insert admin login restriction failed: %s", err.Error())
		return adminV1.ErrorInternalServerError("insert admin login restriction failed")
	}

	return nil
}

func (r *LoginPolicyRepo) Update(ctx context.Context, req *authenticationV1.UpdateLoginPolicyRequest) error {
	if req == nil || req.Data == nil {
		return adminV1.ErrorBadRequest("invalid request")
	}
	if req.GetId() == 0 {
		return adminV1.ErrorBadRequest("id is required")
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return err
		}
		if !exist {
			createReq := &authenticationV1.CreateLoginPolicyRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	builder := r.entClient.Client().LoginPolicy.Update()
	err := r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *authenticationV1.LoginPolicy) {
			builder.
				SetNillableTargetID(req.Data.TargetId).
				SetNillableType(r.typeConverter.ToEntity(req.Data.Type)).
				SetNillableMethod(r.methodConverter.ToEntity(req.Data.Method)).
				SetNillableValue(req.Data.Value).
				SetNillableReason(req.Data.Reason).
				SetNillableUpdatedBy(req.Data.UpdatedBy).
				SetUpdatedAt(time.Now())
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(loginpolicy.FieldID, req.GetId()))
		},
	)

	return err
}

func (r *LoginPolicyRepo) Delete(ctx context.Context, req *authenticationV1.DeleteLoginPolicyRequest) error {
	if req == nil {
		return adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().LoginPolicy.Delete()
	_, err := r.repository.Delete(ctx, builder, func(s *sql.Selector) {
		s.Where(sql.EQ(loginpolicy.FieldID, req.GetId()))
	})
	if err != nil {
		r.log.Errorf(ctx, "delete internal message categories failed: %s", err.Error())
		return adminV1.ErrorInternalServerError("delete admin login restriction failed")
	}

	return nil
}
