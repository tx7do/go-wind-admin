package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	pagination "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/timeutil"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/dictentry"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	dictV1 "go-wind-admin/api/gen/go/dict/service/v1"
)

type DictEntryRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper *mapper.CopierMapper[dictV1.DictEntry, ent.DictEntry]

	repository *entCrud.Repository[
		ent.DictEntryQuery, ent.DictEntrySelect,
		ent.DictEntryCreate, ent.DictEntryCreateBulk,
		ent.DictEntryUpdate, ent.DictEntryUpdateOne,
		ent.DictEntryDelete,
		predicate.DictEntry,
		dictV1.DictEntry, ent.DictEntry,
	]
}

func NewDictEntryRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *DictEntryRepo {
	repo := &DictEntryRepo{
		log:       ctx.NewLoggerHelper("dict-item/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[dictV1.DictEntry, ent.DictEntry](),
	}

	repo.init()

	return repo
}

func (r *DictEntryRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.DictEntryQuery, ent.DictEntrySelect,
		ent.DictEntryCreate, ent.DictEntryCreateBulk,
		ent.DictEntryUpdate, ent.DictEntryUpdateOne,
		ent.DictEntryDelete,
		predicate.DictEntry,
		dictV1.DictEntry, ent.DictEntry,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

func (r *DictEntryRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().DictEntry.Query()
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

func (r *DictEntryRepo) List(ctx context.Context, req *pagination.PagingRequest) (*dictV1.ListDictEntryResponse, error) {
	if req == nil {
		return nil, dictV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().DictEntry.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &dictV1.ListDictEntryResponse{Total: 0, Items: nil}, nil
	}

	return &dictV1.ListDictEntryResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *DictEntryRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().DictEntry.Query().
		Where(dictentry.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, dictV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *DictEntryRepo) Create(ctx context.Context, req *dictV1.CreateDictEntryRequest) error {
	if req == nil || req.Data == nil {
		return dictV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().DictEntry.Create().
		SetNillableEntryLabel(req.Data.EntryLabel).
		SetNillableEntryValue(req.Data.EntryValue).
		SetNillableNumericValue(req.Data.NumericValue).
		SetNillableLanguageCode(req.Data.LanguageCode).
		SetNillableIsEnabled(req.Data.IsEnabled).
		SetNillableSortOrder(req.Data.SortOrder).
		SetNillableDescription(req.Data.Description).
		SetNillableCreatedBy(req.Data.CreatedBy).
		SetNillableCreatedAt(timeutil.TimestamppbToTime(req.Data.CreatedAt))

	if req.Data.TypeId == nil {
		builder.SetSysDictTypesID(req.Data.GetTypeId())
	}
	if req.Data.CreatedAt == nil {
		builder.SetCreatedAt(time.Now())
	}

	if req.Data.Id != nil {
		builder.SetID(req.GetData().GetId())
	}

	if err := builder.Exec(ctx); err != nil {
		r.log.Errorf("insert dict entry failed: %s", err.Error())
		return dictV1.ErrorInternalServerError("insert dict entry failed")
	}

	return nil
}

func (r *DictEntryRepo) Update(ctx context.Context, req *dictV1.UpdateDictEntryRequest) error {
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
			createReq := &dictV1.CreateDictEntryRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	builder := r.entClient.Client().Debug().DictEntry.Update()
	err := r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *dictV1.DictEntry) {
			builder.
				SetNillableEntryLabel(req.Data.EntryLabel).
				SetNillableEntryValue(req.Data.EntryValue).
				SetNillableNumericValue(req.Data.NumericValue).
				SetNillableLanguageCode(req.Data.LanguageCode).
				SetNillableIsEnabled(req.Data.IsEnabled).
				SetNillableSortOrder(req.Data.SortOrder).
				SetNillableDescription(req.Data.Description).
				SetNillableUpdatedBy(req.Data.UpdatedBy).
				SetNillableUpdatedAt(timeutil.TimestamppbToTime(req.Data.UpdatedAt))

			if req.Data.UpdatedAt == nil {
				builder.SetUpdatedAt(time.Now())
			}
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(dictentry.FieldID, req.GetId()))
		},
	)

	return err
}

func (r *DictEntryRepo) Delete(ctx context.Context, id uint32) error {
	if id == 0 {
		return dictV1.ErrorBadRequest("invalid parameter")
	}

	if err := r.entClient.Client().DictEntry.DeleteOneID(id).Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return dictV1.ErrorNotFound("dict not found")
		}

		r.log.Errorf("delete one data failed: %s", err.Error())

		return dictV1.ErrorInternalServerError("delete failed")
	}

	return nil
}

func (r *DictEntryRepo) BatchDelete(ctx context.Context, ids []uint32) error {
	if len(ids) == 0 {
		return dictV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := r.entClient.Client().DictEntry.Delete().
		Where(dictentry.IDIn(ids...)).
		Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return dictV1.ErrorNotFound("dict not found")
		}

		r.log.Errorf("delete one data failed: %s", err.Error())

		return dictV1.ErrorInternalServerError("delete failed")
	}

	return nil
}
