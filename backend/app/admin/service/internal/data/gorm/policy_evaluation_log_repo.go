//go:build gorm_backend
// +build gorm_backend

// Package gorm 中的仓储是 ent 仓储的平行 gorm 镜像，作为"ent 为主力、gorm 为备选"脚手架的完整代码。
// 这些仓储为死代码：未接入 cmd/server/wiring.go、不被 service 引用；采用者需要时自行装配。
//
// gorm 仓储不做租户隔离（ent 侧靠编译进生成代码的 privacy 策略自动注入，gorm 侧无此机制）。
// 直接切换 gorm 后端会有跨租户数据泄露风险，采用者须自行加 scope/plugin。
package gorm

import (
	"context"
	"errors"

	gormDB "gorm.io/gorm"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	gormCrud "github.com/tx7do/go-crud/gorm"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/admin/service/internal/data/gorm/models"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

type PolicyEvaluationLogRepo struct {
	client     *gormCrud.Client
	log        *log.Helper
	mapper     *mapper.CopierMapper[permissionV1.PolicyEvaluationLog, models.PolicyEvaluationLog]
	repository *gormCrud.Repository[permissionV1.PolicyEvaluationLog, models.PolicyEvaluationLog]
}

func NewPolicyEvaluationLogRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *PolicyEvaluationLogRepo {
	repo := &PolicyEvaluationLogRepo{
		log:    ctx.NewLoggerHelper("policy-evaluation-log/gorm-repo/admin-service"),
		client: client,
		mapper: mapper.NewCopierMapper[permissionV1.PolicyEvaluationLog, models.PolicyEvaluationLog](),
	}

	repo.init()

	return repo
}

func (r *PolicyEvaluationLogRepo) init() {
	r.repository = gormCrud.NewRepository[permissionV1.PolicyEvaluationLog, models.PolicyEvaluationLog](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

func (r *PolicyEvaluationLogRepo) Count(ctx context.Context, scopes []func(*gormDB.DB) *gormDB.DB) (int, error) {
	count, err := r.repository.Count(ctx, r.client.DB, scopes)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, permissionV1.ErrorInternalServerError("query count failed")
	}

	return int(count), nil
}

func (r *PolicyEvaluationLogRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*permissionV1.ListPolicyEvaluationLogResponse, error) {
	if req == nil {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	ret, err := r.repository.ListWithPaging(ctx, r.client.DB, req)
	if err != nil {
		return nil, permissionV1.ErrorInternalServerError("query list failed")
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
	exist, err := r.repository.ExistsWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", id) },
	})
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

	var scopes []func(*gormDB.DB) *gormDB.DB
	switch req.QueryBy.(type) {
	default:
	case *permissionV1.GetPolicyEvaluationLogRequest_Id:
		scopes = append(scopes, func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) })
	}

	dto, err := r.repository.GetWithFilters(ctx, r.client.DB, scopes, req.GetViewMask())
	if err != nil {
		if errors.Is(err, gormDB.ErrRecordNotFound) {
			return nil, permissionV1.ErrorNotFound("policy evaluation log not found")
		}
		r.log.Errorf("query policy evaluation log failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query policy evaluation log failed")
	}

	return dto, nil
}

func (r *PolicyEvaluationLogRepo) Create(ctx context.Context, req *permissionV1.CreatePolicyEvaluationLogRequest) error {
	if req == nil || req.Data == nil {
		return permissionV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := r.repository.Create(ctx, r.client.DB, req.Data, nil); err != nil {
		r.log.Errorf("insert policy evaluation log failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("insert policy evaluation log failed")
	}

	return nil
}
