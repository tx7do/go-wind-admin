package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/timeutil"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/position"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
)

type PositionRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper          *mapper.CopierMapper[identityV1.Position, ent.Position]
	statusConverter *mapper.EnumTypeConverter[identityV1.Position_Status, position.Status]
	typeConverter   *mapper.EnumTypeConverter[identityV1.Position_Type, position.Type]

	repository *entCrud.Repository[
		ent.PositionQuery, ent.PositionSelect,
		ent.PositionCreate, ent.PositionCreateBulk,
		ent.PositionUpdate, ent.PositionUpdateOne,
		ent.PositionDelete,
		predicate.Position,
		identityV1.Position, ent.Position,
	]
}

func NewPositionRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *PositionRepo {
	repo := &PositionRepo{
		log:       ctx.NewLoggerHelper("position/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[identityV1.Position, ent.Position](),
		statusConverter: mapper.NewEnumTypeConverter[identityV1.Position_Status, position.Status](
			identityV1.Position_Status_name, identityV1.Position_Status_value,
		),
		typeConverter: mapper.NewEnumTypeConverter[identityV1.Position_Type, position.Type](
			identityV1.Position_Type_name, identityV1.Position_Type_value,
		),
	}

	repo.init()

	return repo
}

func (r *PositionRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.PositionQuery, ent.PositionSelect,
		ent.PositionCreate, ent.PositionCreateBulk,
		ent.PositionUpdate, ent.PositionUpdateOne,
		ent.PositionDelete,
		predicate.Position,
		identityV1.Position, ent.Position,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
	r.mapper.AppendConverters(r.typeConverter.NewConverterPair())
}

func (r *PositionRepo) count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().Position.Query()
	if len(whereCond) != 0 {
		builder.Modify(whereCond...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, identityV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *PositionRepo) Count(ctx context.Context, req *paginationV1.PagingRequest) (int, error) {
	builder := r.entClient.Client().Position.Query()

	whereSelectors, _, err := r.repository.BuildListSelectorWithPaging(builder, req)
	if len(whereSelectors) != 0 {
		builder.Modify(whereSelectors...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf("query position count failed: %s", err.Error())
		return 0, identityV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *PositionRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*identityV1.ListPositionResponse, error) {
	if req == nil {
		return nil, identityV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Debug().Position.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &identityV1.ListPositionResponse{Total: 0, Items: nil}, nil
	}

	return &identityV1.ListPositionResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *PositionRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().Position.Query().
		Where(position.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, identityV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *PositionRepo) Get(ctx context.Context, req *identityV1.GetPositionRequest) (*identityV1.Position, error) {
	if req == nil {
		return nil, identityV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Position.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *identityV1.GetPositionRequest_Id:
		whereCond = append(whereCond, position.IDEQ(req.GetId()))
	case *identityV1.GetPositionRequest_Name:
		whereCond = append(whereCond, position.NameEQ(req.GetName()))
	case *identityV1.GetPositionRequest_Code:
		whereCond = append(whereCond, position.CodeEQ(req.GetCode()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

// ListPositionByIds 通过多个ID获取职位信息
func (r *PositionRepo) ListPositionByIds(ctx context.Context, ids []uint32) ([]*identityV1.Position, error) {
	if len(ids) == 0 {
		return []*identityV1.Position{}, nil
	}

	entities, err := r.entClient.Client().Position.Query().
		Where(position.IDIn(ids...)).
		All(ctx)
	if err != nil {
		r.log.Errorf("query position by ids failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query position by ids failed")
	}

	dtos := make([]*identityV1.Position, 0, len(entities))
	for _, entity := range entities {
		dto := r.mapper.ToDTO(entity)
		dtos = append(dtos, dto)
	}

	return dtos, nil
}

func (r *PositionRepo) Create(ctx context.Context, req *identityV1.CreatePositionRequest) error {
	if req == nil || req.Data == nil {
		return identityV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Position.Create().
		SetNillableTenantID(req.Data.TenantId).
		SetName(req.Data.GetName()).
		SetCode(req.Data.GetCode()).
		SetOrgUnitID(req.Data.GetOrgUnitId()).
		SetReportsToPositionID(req.Data.GetReportsToPositionId()).
		SetNillableSortOrder(req.Data.SortOrder).
		SetNillableStatus(r.statusConverter.ToEntity(req.Data.Status)).
		SetNillableType(r.typeConverter.ToEntity(req.Data.Type)).
		SetNillableJobFamily(req.Data.JobFamily).
		SetNillableJobGrade(req.Data.JobGrade).
		SetNillableLevel(req.Data.Level).
		SetNillableIsKeyPosition(req.Data.IsKeyPosition).
		SetNillableHeadcount(req.Data.Headcount).
		SetNillableDescription(req.Data.Description).
		SetNillableRemark(req.Data.Remark).
		SetNillableStartAt(timeutil.TimestamppbToTime(req.Data.StartAt)).
		SetNillableEndAt(timeutil.TimestamppbToTime(req.Data.EndAt)).
		SetNillableCreatedBy(req.Data.CreatedBy).
		SetCreatedAt(time.Now())

	if req.Data.Id != nil {
		builder.SetID(req.GetData().GetId())
	}

	if err := builder.Exec(ctx); err != nil {
		r.log.Errorf("insert position failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("insert data failed")
	}

	return nil
}

func (r *PositionRepo) Update(ctx context.Context, req *identityV1.UpdatePositionRequest) error {
	if req == nil || req.Data == nil {
		return identityV1.ErrorBadRequest("invalid parameter")
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return err
		}
		if !exist {
			createReq := &identityV1.CreatePositionRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	builder := r.entClient.Client().Debug().Position.Update()
	err := r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *identityV1.Position) {
			builder.
				SetNillableName(req.Data.Name).
				SetNillableCode(req.Data.Code).
				SetNillableOrgUnitID(req.Data.OrgUnitId).
				SetNillableReportsToPositionID(req.Data.ReportsToPositionId).
				SetNillableSortOrder(req.Data.SortOrder).
				SetNillableStatus(r.statusConverter.ToEntity(req.Data.Status)).
				SetNillableType(r.typeConverter.ToEntity(req.Data.Type)).
				SetNillableJobFamily(req.Data.JobFamily).
				SetNillableJobGrade(req.Data.JobGrade).
				SetNillableLevel(req.Data.Level).
				SetNillableIsKeyPosition(req.Data.IsKeyPosition).
				SetNillableHeadcount(req.Data.Headcount).
				SetNillableDescription(req.Data.Description).
				SetNillableRemark(req.Data.Remark).
				SetNillableStartAt(timeutil.TimestamppbToTime(req.Data.StartAt)).
				SetNillableEndAt(timeutil.TimestamppbToTime(req.Data.EndAt)).
				SetNillableUpdatedBy(req.Data.UpdatedBy).
				SetUpdatedAt(time.Now())
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(position.FieldID, req.GetId()))
		},
	)

	return err
}

func (r *PositionRepo) Delete(ctx context.Context, req *identityV1.DeletePositionRequest) error {
	if req == nil {
		return identityV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Debug().Position.Delete()

	var err error
	_, err = r.repository.Delete(ctx, builder, func(s *sql.Selector) {
		s.Where(sql.EQ(position.FieldID, req.GetId()))
	})
	if err != nil {
		r.log.Errorf("delete position failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("delete position failed")
	}

	return nil
}
