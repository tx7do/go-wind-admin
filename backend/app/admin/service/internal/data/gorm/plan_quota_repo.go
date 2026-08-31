//go:build gorm_backend
// +build gorm_backend

package gorm

import (
	"context"
	"errors"

	gormDB "gorm.io/gorm"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	gormCrud "github.com/tx7do/go-crud/gorm"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/admin/service/internal/data/gorm/models"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
)

type PlanQuotaRepo struct {
	client        *gormCrud.Client
	log           *bLogger.Helper
	mapper        *mapper.CopierMapper[identityV1.PlanQuota, models.PlanQuota]
	quotaTypeConv *mapper.EnumTypeConverter[identityV1.PlanQuota_QuotaType, string]

	repository *gormCrud.Repository[identityV1.PlanQuota, models.PlanQuota]
}

func NewPlanQuotaRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *PlanQuotaRepo {
	repo := &PlanQuotaRepo{
		log:           ctx.NewLoggerHelper("plan-quota/gorm-repo/admin-service"),
		client:        client,
		mapper:        mapper.NewCopierMapper[identityV1.PlanQuota, models.PlanQuota](),
		quotaTypeConv: mapper.NewEnumTypeConverter[identityV1.PlanQuota_QuotaType, string](identityV1.PlanQuota_QuotaType_name, identityV1.PlanQuota_QuotaType_value),
	}

	repo.init()

	return repo
}

func (r *PlanQuotaRepo) init() {
	r.repository = gormCrud.NewRepository[identityV1.PlanQuota, models.PlanQuota](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.quotaTypeConv.NewConverterPair())
}

func (r *PlanQuotaRepo) Count(ctx context.Context, scopes []func(*gormDB.DB) *gormDB.DB) (int, error) {
	count, err := r.repository.Count(ctx, r.client.DB, scopes)
	if err != nil {
		r.log.Errorf(ctx, "query count failed: %s", err.Error())
		return 0, identityV1.ErrorInternalServerError("query count failed")
	}

	return int(count), nil
}

func (r *PlanQuotaRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*identityV1.ListPlanQuotaResponse, error) {
	return nil, identityV1.ErrorInternalServerError("gorm scaffold: List not implemented — ent .WithPlan/HasPlanWith/.Edges edge predicate has no go-crud/gorm primitive; see data/plan_quota_repo.go")
}

func (r *PlanQuotaRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.repository.ExistsWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", id) },
	})
	if err != nil {
		r.log.Errorf(ctx, "query exist failed: %s", err.Error())
		return false, identityV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *PlanQuotaRepo) Get(ctx context.Context, req *identityV1.GetPlanQuotaRequest) (*identityV1.PlanQuota, error) {
	if req == nil {
		return nil, identityV1.ErrorBadRequest("invalid parameter")
	}

	var scopes []func(*gormDB.DB) *gormDB.DB
	switch req.QueryBy.(type) {
	default:
	case *identityV1.GetPlanQuotaRequest_Id:
		scopes = append(scopes, func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) })
	}

	dto, err := r.repository.GetWithFilters(ctx, r.client.DB, scopes, req.GetViewMask())
	if err != nil {
		if errors.Is(err, gormDB.ErrRecordNotFound) {
			return nil, identityV1.ErrorNotFound("plan quota not found")
		}
		r.log.Errorf(ctx, "query plan quota failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query plan quota failed")
	}

	return dto, nil
}

func (r *PlanQuotaRepo) Create(ctx context.Context, req *identityV1.CreatePlanQuotaRequest) error {
	return identityV1.ErrorInternalServerError("gorm scaffold: Create not implemented — ent tx edge write has no go-crud/gorm primitive; see data/plan_quota_repo.go")
}

func (r *PlanQuotaRepo) Update(ctx context.Context, req *identityV1.UpdatePlanQuotaRequest) error {
	return identityV1.ErrorInternalServerError("gorm scaffold: Update not implemented — ent tx edge write has no go-crud/gorm primitive; see data/plan_quota_repo.go")
}

func (r *PlanQuotaRepo) Delete(ctx context.Context, id uint32) error {
	if id == 0 {
		return identityV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", id) },
	}); err != nil {
		r.log.Errorf(ctx, "delete plan quota failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("delete failed")
	}

	return nil
}
