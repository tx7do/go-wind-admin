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

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
)

type PlanRepo struct {
	client           *gormCrud.Client
	log              *log.Helper
	mapper           *mapper.CopierMapper[identityV1.Plan, models.Plan]
	versionConverter *mapper.EnumTypeConverter[identityV1.Plan_Version, string]
	expiryPolicyConv *mapper.EnumTypeConverter[identityV1.Plan_ExpiryPolicy, string]
	repository       *gormCrud.Repository[identityV1.Plan, models.Plan]
}

func NewPlanRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *PlanRepo {
	repo := &PlanRepo{
		log:    ctx.NewLoggerHelper("plan/gorm-repo/admin-service"),
		client: client,
		mapper: mapper.NewCopierMapper[identityV1.Plan, models.Plan](),
		versionConverter: mapper.NewEnumTypeConverter[identityV1.Plan_Version, string](
			identityV1.Plan_Version_name, identityV1.Plan_Version_value,
		),
		expiryPolicyConv: mapper.NewEnumTypeConverter[identityV1.Plan_ExpiryPolicy, string](
			identityV1.Plan_ExpiryPolicy_name, identityV1.Plan_ExpiryPolicy_value,
		),
	}

	repo.init()

	return repo
}

func (r *PlanRepo) init() {
	r.repository = gormCrud.NewRepository[identityV1.Plan, models.Plan](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.versionConverter.NewConverterPair())
	r.mapper.AppendConverters(r.expiryPolicyConv.NewConverterPair())
}

func (r *PlanRepo) Count(ctx context.Context, scopes []func(*gormDB.DB) *gormDB.DB) (int, error) {
	count, err := r.repository.Count(ctx, r.client.DB, scopes)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, identityV1.ErrorInternalServerError("query count failed")
	}

	return int(count), nil
}

func (r *PlanRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*identityV1.ListPlanResponse, error) {
	if req == nil {
		return nil, identityV1.ErrorBadRequest("invalid parameter")
	}

	ret, err := r.repository.ListWithPaging(ctx, r.client.DB, req)
	if err != nil {
		return nil, identityV1.ErrorInternalServerError("query list failed")
	}
	if ret == nil {
		return &identityV1.ListPlanResponse{Total: 0, Items: nil}, nil
	}

	return &identityV1.ListPlanResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *PlanRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.repository.ExistsWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", id) },
	})
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, identityV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *PlanRepo) Get(ctx context.Context, req *identityV1.GetPlanRequest) (*identityV1.Plan, error) {
	if req == nil {
		return nil, identityV1.ErrorBadRequest("invalid parameter")
	}

	var scopes []func(*gormDB.DB) *gormDB.DB
	switch req.QueryBy.(type) {
	default:
	case *identityV1.GetPlanRequest_Id:
		scopes = append(scopes, func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) })
	}

	dto, err := r.repository.GetWithFilters(ctx, r.client.DB, scopes, req.GetViewMask())
	if err != nil {
		if errors.Is(err, gormDB.ErrRecordNotFound) {
			return nil, identityV1.ErrorNotFound("plan not found")
		}
		r.log.Errorf("query plan failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query plan failed")
	}

	return dto, nil
}

func (r *PlanRepo) Create(ctx context.Context, req *identityV1.CreatePlanRequest) error {
	if req == nil || req.Data == nil {
		return identityV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := r.repository.Create(ctx, r.client.DB, req.Data, nil); err != nil {
		r.log.Errorf("insert plan failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("insert data failed")
	}

	return nil
}

func (r *PlanRepo) Update(ctx context.Context, req *identityV1.UpdatePlanRequest) error {
	if req == nil || req.Data == nil {
		return identityV1.ErrorBadRequest("invalid parameter")
	}
	if req.GetId() == 0 {
		return identityV1.ErrorBadRequest("id is required")
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return err
		}
		if !exist {
			createReq := &identityV1.CreatePlanRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	if _, err := r.repository.UpdateWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) },
	}, req.Data, req.GetUpdateMask()); err != nil {
		r.log.Errorf("update plan failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("update plan failed")
	}

	return nil
}

func (r *PlanRepo) Delete(ctx context.Context, id uint32) error {
	if id == 0 {
		return identityV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", id) },
	}); err != nil {
		r.log.Errorf("delete plan failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("delete plan failed")
	}

	return nil
}
