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
	"go-wind-admin/app/admin/service/internal/data/ent/language"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	dictV1 "go-wind-admin/api/gen/go/dict/service/v1"
)

type LanguageRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper *mapper.CopierMapper[dictV1.Language, ent.Language]

	repository *entCrud.Repository[
		ent.LanguageQuery, ent.LanguageSelect,
		ent.LanguageCreate, ent.LanguageCreateBulk,
		ent.LanguageUpdate, ent.LanguageUpdateOne,
		ent.LanguageDelete,
		predicate.Language,
		dictV1.Language, ent.Language,
	]
}

func NewLanguageRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *LanguageRepo {
	repo := &LanguageRepo{
		log:       ctx.NewLoggerHelper("language/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[dictV1.Language, ent.Language](),
	}

	repo.init()

	return repo
}

func (r *LanguageRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.LanguageQuery, ent.LanguageSelect,
		ent.LanguageCreate, ent.LanguageCreateBulk,
		ent.LanguageUpdate, ent.LanguageUpdateOne,
		ent.LanguageDelete,
		predicate.Language,
		dictV1.Language, ent.Language,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

func (r *LanguageRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().Language.Query()
	if len(whereCond) != 0 {
		builder.Modify(whereCond...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, dictV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *LanguageRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*dictV1.ListLanguageResponse, error) {
	if req == nil {
		return nil, dictV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Debug().Language.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &dictV1.ListLanguageResponse{Total: 0, Items: nil}, nil
	}

	return &dictV1.ListLanguageResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *LanguageRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().Language.Query().
		Where(language.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, dictV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *LanguageRepo) Get(ctx context.Context, req *dictV1.GetLanguageRequest) (*dictV1.Language, error) {
	if req == nil {
		return nil, dictV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Language.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *dictV1.GetLanguageRequest_Id:
		whereCond = append(whereCond, language.IDEQ(req.GetId()))
	case *dictV1.GetLanguageRequest_Code:
		whereCond = append(whereCond, language.LanguageCodeEQ(req.GetCode()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

// ListLanguageByIds 通过多个ID获取职位信息
func (r *LanguageRepo) ListLanguageByIds(ctx context.Context, ids []uint32) ([]*dictV1.Language, error) {
	if len(ids) == 0 {
		return []*dictV1.Language{}, nil
	}

	entities, err := r.entClient.Client().Language.Query().
		Where(language.IDIn(ids...)).
		All(ctx)
	if err != nil {
		r.log.Errorf("query language by ids failed: %s", err.Error())
		return nil, dictV1.ErrorInternalServerError("query language by ids failed")
	}

	dtos := make([]*dictV1.Language, 0, len(entities))
	for _, entity := range entities {
		dto := r.mapper.ToDTO(entity)
		dtos = append(dtos, dto)
	}

	return dtos, nil
}

func (r *LanguageRepo) Create(ctx context.Context, req *dictV1.CreateLanguageRequest) error {
	if req == nil || req.Data == nil {
		return dictV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Language.Create().
		SetLanguageName(req.Data.GetLanguageName()).
		SetLanguageCode(req.Data.GetLanguageCode()).
		SetNativeName(req.Data.GetNativeName()).
		SetNillableIsEnabled(req.Data.IsEnabled).
		SetNillableIsDefault(req.Data.IsDefault).
		SetNillableSortOrder(req.Data.SortOrder).
		SetNillableCreatedBy(req.Data.CreatedBy).
		SetCreatedAt(time.Now())

	if req.Data.Id != nil {
		builder.SetID(req.GetData().GetId())
	}

	if err := builder.Exec(ctx); err != nil {
		r.log.Errorf("insert language failed: %s", err.Error())
		return dictV1.ErrorInternalServerError("insert data failed")
	}

	return nil
}

func (r *LanguageRepo) Update(ctx context.Context, req *dictV1.UpdateLanguageRequest) error {
	if req == nil || req.Data == nil {
		return dictV1.ErrorBadRequest("invalid parameter")
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return err
		}
		if !exist {
			createReq := &dictV1.CreateLanguageRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	builder := r.entClient.Client().Debug().Language.Update()
	err := r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *dictV1.Language) {
			builder.
				SetLanguageName(req.Data.GetLanguageName()).
				//SetLanguageCode(req.Data.GetLanguageCode()).
				SetNativeName(req.Data.GetNativeName()).
				SetNillableIsEnabled(req.Data.IsEnabled).
				SetNillableIsDefault(req.Data.IsDefault).
				SetNillableSortOrder(req.Data.SortOrder).
				SetNillableUpdatedBy(req.Data.UpdatedBy).
				SetUpdatedAt(time.Now())
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(language.FieldID, req.GetId()))
		},
	)

	return err
}

func (r *LanguageRepo) Delete(ctx context.Context, req *dictV1.DeleteLanguageRequest) error {
	if req == nil {
		return dictV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Debug().Language.Delete()

	var err error
	_, err = r.repository.Delete(ctx, builder, func(s *sql.Selector) {
		s.Where(sql.EQ(language.FieldID, req.GetId()))
	})
	if err != nil {
		r.log.Errorf("delete language failed: %s", err.Error())
		return dictV1.ErrorInternalServerError("delete language failed")
	}

	return nil
}
