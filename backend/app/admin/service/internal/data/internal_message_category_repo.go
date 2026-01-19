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
	"go-wind-admin/app/admin/service/internal/data/ent/internalmessagecategory"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	internalMessageV1 "go-wind-admin/api/gen/go/internal_message/service/v1"
)

type InternalMessageCategoryRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper *mapper.CopierMapper[internalMessageV1.InternalMessageCategory, ent.InternalMessageCategory]

	repository *entCrud.Repository[
		ent.InternalMessageCategoryQuery, ent.InternalMessageCategorySelect,
		ent.InternalMessageCategoryCreate, ent.InternalMessageCategoryCreateBulk,
		ent.InternalMessageCategoryUpdate, ent.InternalMessageCategoryUpdateOne,
		ent.InternalMessageCategoryDelete,
		predicate.InternalMessageCategory,
		internalMessageV1.InternalMessageCategory, ent.InternalMessageCategory,
	]
}

func NewInternalMessageCategoryRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *InternalMessageCategoryRepo {
	repo := &InternalMessageCategoryRepo{
		log:       ctx.NewLoggerHelper("internal-message-category/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[internalMessageV1.InternalMessageCategory, ent.InternalMessageCategory](),
	}

	repo.init()

	return repo
}

func (r *InternalMessageCategoryRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.InternalMessageCategoryQuery, ent.InternalMessageCategorySelect,
		ent.InternalMessageCategoryCreate, ent.InternalMessageCategoryCreateBulk,
		ent.InternalMessageCategoryUpdate, ent.InternalMessageCategoryUpdateOne,
		ent.InternalMessageCategoryDelete,
		predicate.InternalMessageCategory,
		internalMessageV1.InternalMessageCategory, ent.InternalMessageCategory,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

func (r *InternalMessageCategoryRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().InternalMessageCategory.Query()
	if len(whereCond) != 0 {
		builder.Modify(whereCond...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, internalMessageV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *InternalMessageCategoryRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*internalMessageV1.ListInternalMessageCategoryResponse, error) {
	if req == nil {
		return nil, internalMessageV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().InternalMessageCategory.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &internalMessageV1.ListInternalMessageCategoryResponse{Total: 0, Items: nil}, nil
	}

	return &internalMessageV1.ListInternalMessageCategoryResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *InternalMessageCategoryRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().InternalMessageCategory.Query().
		Where(internalmessagecategory.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, internalMessageV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *InternalMessageCategoryRepo) Get(ctx context.Context, req *internalMessageV1.GetInternalMessageCategoryRequest) (*internalMessageV1.InternalMessageCategory, error) {
	if req == nil {
		return nil, internalMessageV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().InternalMessageCategory.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *internalMessageV1.GetInternalMessageCategoryRequest_Id:
		whereCond = append(whereCond, internalmessagecategory.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

// ListCategoriesByIds 根据ID列表获取分类列表
func (r *InternalMessageCategoryRepo) ListCategoriesByIds(ctx context.Context, ids []uint32) ([]*internalMessageV1.InternalMessageCategory, error) {
	if len(ids) == 0 {
		return []*internalMessageV1.InternalMessageCategory{}, nil
	}

	entities, err := r.entClient.Client().InternalMessageCategory.Query().
		Where(internalmessagecategory.IDIn(ids...)).
		All(ctx)
	if err != nil {
		r.log.Errorf("query internal message category by ids failed: %s", err.Error())
		return nil, internalMessageV1.ErrorInternalServerError("query internal message category by ids failed")
	}

	dtos := make([]*internalMessageV1.InternalMessageCategory, 0, len(entities))
	for _, entity := range entities {
		dto := r.mapper.ToDTO(entity)
		dtos = append(dtos, dto)
	}

	return dtos, nil
}

func (r *InternalMessageCategoryRepo) Create(ctx context.Context, req *internalMessageV1.CreateInternalMessageCategoryRequest) error {
	if req == nil || req.Data == nil {
		return internalMessageV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().InternalMessageCategory.Create().
		SetNillableTenantID(req.Data.TenantId).
		SetNillableName(req.Data.Name).
		SetNillableCode(req.Data.Code).
		SetNillableIconURL(req.Data.IconUrl).
		SetNillableSortOrder(req.Data.SortOrder).
		SetNillableIsEnabled(req.Data.IsEnabled).
		SetNillableCreatedBy(req.Data.CreatedBy).
		SetNillableCreatedAt(timeutil.TimestamppbToTime(req.Data.CreatedAt))

	if req.Data.CreatedAt == nil {
		builder.SetCreatedAt(time.Now())
	}

	if req.Data.Id != nil {
		builder.SetID(req.GetData().GetId())
	}

	if err := builder.Exec(ctx); err != nil {
		r.log.Errorf("insert internal message category failed: %s", err.Error())
		return internalMessageV1.ErrorInternalServerError("insert internal message category failed")
	}

	return nil
}

func (r *InternalMessageCategoryRepo) Update(ctx context.Context, req *internalMessageV1.UpdateInternalMessageCategoryRequest) error {
	if req == nil || req.Data == nil {
		return internalMessageV1.ErrorBadRequest("invalid parameter")
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return err
		}
		if !exist {
			createReq := &internalMessageV1.CreateInternalMessageCategoryRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	builder := r.entClient.Client().Debug().InternalMessageCategory.Update()
	err := r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *internalMessageV1.InternalMessageCategory) {
			builder.
				SetNillableName(req.Data.Name).
				SetNillableCode(req.Data.Code).
				SetNillableIconURL(req.Data.IconUrl).
				SetNillableSortOrder(req.Data.SortOrder).
				SetNillableIsEnabled(req.Data.IsEnabled).
				SetNillableUpdatedBy(req.Data.UpdatedBy).
				SetNillableUpdatedAt(timeutil.TimestamppbToTime(req.Data.UpdatedAt))

			if req.Data.UpdatedAt == nil {
				builder.SetUpdatedAt(time.Now())
			}
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(internalmessagecategory.FieldID, req.GetId()))
		},
	)

	return err
}

func (r *InternalMessageCategoryRepo) Delete(ctx context.Context, req *internalMessageV1.DeleteInternalMessageCategoryRequest) error {
	if req == nil {
		return internalMessageV1.ErrorBadRequest("invalid parameter")
	}

	ids, err := entCrud.QueryAllChildrenIds(ctx, r.entClient, "internal_message_categories", req.GetId())
	if err != nil {
		r.log.Errorf("query child internal message categories failed: %s", err.Error())
		return internalMessageV1.ErrorInternalServerError("query child internal message categories failed")
	}
	ids = append(ids, req.GetId())

	//r.log.Info("internal message category ids to delete: ", ids)

	builder := r.entClient.Client().Debug().InternalMessageCategory.Delete()

	_, err = r.repository.Delete(ctx, builder, func(s *sql.Selector) {
		s.Where(sql.In(internalmessagecategory.FieldID, ids))
	})
	if err != nil {
		r.log.Errorf("delete internal message categories failed: %s", err.Error())
		return internalMessageV1.ErrorInternalServerError("delete internal message categories failed")
	}

	return nil
}
