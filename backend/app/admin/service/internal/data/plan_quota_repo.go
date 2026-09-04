package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/planquota"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
)

type PlanQuotaRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *bLogger.Helper

	mapper           *mapper.CopierMapper[identityV1.PlanQuota, ent.PlanQuota]
	quotaTypeConv    *mapper.EnumTypeConverter[identityV1.PlanQuota_QuotaType, planquota.QuotaType]

	repository *entCrud.Repository[
		ent.PlanQuotaQuery, ent.PlanQuotaSelect,
		ent.PlanQuotaCreate, ent.PlanQuotaCreateBulk,
		ent.PlanQuotaUpdate, ent.PlanQuotaUpdateOne,
		ent.PlanQuotaDelete,
		predicate.PlanQuota,
		identityV1.PlanQuota, ent.PlanQuota,
	]
}

func NewPlanQuotaRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) *PlanQuotaRepo {
	repo := &PlanQuotaRepo{
		log:       ctx.NewLoggerHelper("plan-quota/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[identityV1.PlanQuota, ent.PlanQuota](),
		quotaTypeConv: mapper.NewEnumTypeConverter[identityV1.PlanQuota_QuotaType, planquota.QuotaType](
			identityV1.PlanQuota_QuotaType_name, identityV1.PlanQuota_QuotaType_value,
		),
	}

	repo.init()

	return repo
}

func (r *PlanQuotaRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.PlanQuotaQuery, ent.PlanQuotaSelect,
		ent.PlanQuotaCreate, ent.PlanQuotaCreateBulk,
		ent.PlanQuotaUpdate, ent.PlanQuotaUpdateOne,
		ent.PlanQuotaDelete,
		predicate.PlanQuota,
		identityV1.PlanQuota, ent.PlanQuota,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.quotaTypeConv.NewConverterPair())
}

func (r *PlanQuotaRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().PlanQuota.Query()
	if len(whereCond) != 0 {
		builder.Modify(whereCond...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query count failed: %s", err.Error())
		return 0, identityV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *PlanQuotaRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*identityV1.ListPlanQuotaResponse, error) {
	if req == nil {
		return nil, identityV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().PlanQuota.Query().WithPlan()

	whereSelectors, _, err := r.repository.BuildListSelectorWithPaging(builder, req)
	if err != nil {
		r.log.Errorf(ctx, "parse list param error [%s]", err.Error())
		return nil, identityV1.ErrorBadRequest("invalid query parameter")
	}

	entities, err := builder.All(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query plan quota list failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query plan quota list failed")
	}

	dtos := make([]*identityV1.PlanQuota, 0, len(entities))
	for _, entity := range entities {
		dto := r.mapper.ToDTO(entity)
		if entity.Edges.Plan != nil {
			dto.PlanId = &entity.Edges.Plan.ID
		}
		dtos = append(dtos, dto)
	}

	count, err := r.Count(ctx, whereSelectors)
	if err != nil {
		return nil, err
	}

	return &identityV1.ListPlanQuotaResponse{
		Total: uint64(count),
		Items: dtos,
	}, nil
}

func (r *PlanQuotaRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().PlanQuota.Query().
		Where(planquota.IDEQ(id)).
		Exist(ctx)
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

	builder := r.entClient.Client().PlanQuota.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *identityV1.GetPlanQuotaRequest_Id:
		whereCond = append(whereCond, planquota.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

func (r *PlanQuotaRepo) Create(ctx context.Context, req *identityV1.CreatePlanQuotaRequest) (err error) {
	if req == nil || req.Data == nil {
		return identityV1.ErrorBadRequest("invalid parameter")
	}

	var tx *ent.Tx
	tx, err = r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf(ctx, "start transaction failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("start transaction failed")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.log.Errorf(ctx, "transaction rollback failed: %s", rollbackErr.Error())
			}
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			r.log.Errorf(ctx, "transaction commit failed: %s", commitErr.Error())
			err = identityV1.ErrorInternalServerError("transaction commit failed")
		}
	}()

	builder := tx.PlanQuota.Create().
		SetNillableQuotaType(r.quotaTypeConv.ToEntity(req.Data.QuotaType)).
		SetNillableQuotaValue(req.Data.QuotaValue).
		SetNillableCreatedBy(req.Data.CreatedBy).
		SetCreatedAt(time.Now())

	// 原条件写反（== nil 时才 Set），导致请求带 planId 也落库为 NULL
	if req.Data.PlanId != nil {
		builder.SetPlanID(*req.Data.PlanId)
	}

	if req.Data.Id != nil {
		builder.SetID(req.GetData().GetId())
	}

	if _, err = builder.Save(ctx); err != nil {
		r.log.Errorf(ctx, "insert plan quota failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("insert plan quota failed")
	}

	return nil
}

func (r *PlanQuotaRepo) Update(ctx context.Context, req *identityV1.UpdatePlanQuotaRequest) (err error) {
	if req == nil || req.Data == nil {
		return identityV1.ErrorBadRequest("invalid parameter")
	}
	if req.GetId() == 0 {
		return identityV1.ErrorBadRequest("id is required")
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		var exist bool
		exist, err = r.IsExist(ctx, req.GetId())
		if err != nil {
			return err
		}
		if !exist {
			createReq := &identityV1.CreatePlanQuotaRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	var tx *ent.Tx
	tx, err = r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf(ctx, "start transaction failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("start transaction failed")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.log.Errorf(ctx, "transaction rollback failed: %s", rollbackErr.Error())
			}
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			r.log.Errorf(ctx, "transaction commit failed: %s", commitErr.Error())
			err = identityV1.ErrorInternalServerError("transaction commit failed")
		}
	}()

	builder := tx.PlanQuota.UpdateOneID(req.GetId())
	_, err = r.repository.UpdateOne(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *identityV1.PlanQuota) {
			builder.
				SetNillableQuotaType(r.quotaTypeConv.ToEntity(req.Data.QuotaType)).
				SetNillableQuotaValue(req.Data.QuotaValue).
				SetNillableUpdatedBy(req.Data.UpdatedBy).
				SetUpdatedAt(time.Now())
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(planquota.FieldID, req.GetId()))
		},
	)
	if err != nil {
		r.log.Errorf(ctx, "update plan quota failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("update plan quota failed")
	}

	return err
}

func (r *PlanQuotaRepo) Delete(ctx context.Context, id uint32) error {
	if id == 0 {
		return identityV1.ErrorBadRequest("invalid parameter")
	}

	if err := r.entClient.Client().PlanQuota.DeleteOneID(id).Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return identityV1.ErrorNotFound("plan quota not found")
		}

		r.log.Errorf(ctx, "delete one data failed: %s", err.Error())

		return identityV1.ErrorInternalServerError("delete failed")
	}

	return nil
}
