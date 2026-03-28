package data

import (
	"context"
	resourceV1 "go-wind-admin/api/gen/go/resource/service/v1"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/dictentry"
	"go-wind-admin/app/admin/service/internal/data/ent/dicttype"
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

	i18n *DictEntryI18nRepo
}

func NewDictEntryRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
	i18n *DictEntryI18nRepo,
) *DictEntryRepo {
	repo := &DictEntryRepo{
		log:       ctx.NewLoggerHelper("dict-entry/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[dictV1.DictEntry, ent.DictEntry](),
		i18n:      i18n,
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

func (r *DictEntryRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*dictV1.ListDictEntryResponse, error) {
	if req == nil {
		return nil, dictV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().DictEntry.Query().WithDictType()

	whereSelectors, _, err := r.repository.BuildListSelectorWithPaging(builder, req)
	if err != nil {
		r.log.Errorf("parse list param error [%s]", err.Error())
		return nil, resourceV1.ErrorBadRequest("invalid query parameter")
	}

	entities, err := builder.All(ctx)
	if err != nil {
		r.log.Errorf("query dict entry list failed: %s", err.Error())
		return nil, resourceV1.ErrorInternalServerError("query dict entry list failed")
	}

	dtos := make([]*dictV1.DictEntry, 0, len(entities))
	for _, entity := range entities {
		dto := r.mapper.ToDTO(entity)
		if entity.Edges.DictType != nil {
			dto.TypeId = &entity.Edges.DictType.ID
		}
		dtos = append(dtos, dto)
		r.log.Debugf("dict entry entity ID: %v", entity.Edges.DictType.ID)
	}

	count, err := r.Count(ctx, whereSelectors)
	if err != nil {
		return nil, err
	}

	var i18ns map[string]*dictV1.DictEntryI18N
	for _, item := range dtos {
		i18ns, err = r.i18n.ListByEntryID(ctx, item.GetId())
		if err != nil {
			return nil, err
		}
		item.I18N = i18ns

		r.log.Debugf("dict entry: %v", item)
	}

	return &dictV1.ListDictEntryResponse{
		Total: uint64(count),
		Items: dtos,
	}, nil
}

func (r *DictEntryRepo) Get(ctx context.Context, req *dictV1.GetDictEntryRequest) (*dictV1.DictEntry, error) {
	if req == nil {
		return nil, dictV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().DictEntry.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *dictV1.GetDictEntryRequest_Id:
		whereCond = append(whereCond, dictentry.IDEQ(req.GetId()))
	case *dictV1.GetDictEntryRequest_Value:
		builder.Where(dictentry.EntryValueEQ(req.GetValue()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	i18ns, err := r.i18n.ListByEntryID(ctx, dto.GetId())
	if err != nil {
		return nil, err
	}
	dto.I18N = i18ns

	return dto, err
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

func (r *DictEntryRepo) Create(ctx context.Context, req *dictV1.CreateDictEntryRequest) (err error) {
	if req == nil || req.Data == nil {
		return dictV1.ErrorBadRequest("invalid parameter")
	}

	var tx *ent.Tx
	tx, err = r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf("start transaction failed: %s", err.Error())
		return dictV1.ErrorInternalServerError("start transaction failed")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.log.Errorf("transaction rollback failed: %s", rollbackErr.Error())
			}
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			r.log.Errorf("transaction commit failed: %s", commitErr.Error())
			err = dictV1.ErrorInternalServerError("transaction commit failed")
		}
	}()

	builder := tx.DictEntry.Create().
		SetNillableTenantID(req.Data.TenantId).
		SetEntryValue(req.Data.GetEntryValue()).
		SetNillableNumericValue(req.Data.NumericValue).
		SetNillableIsEnabled(req.Data.IsEnabled).
		SetNillableSortOrder(req.Data.SortOrder).
		SetNillableCreatedBy(req.Data.CreatedBy).
		SetCreatedAt(time.Now())

	if req.Data.TypeId == nil {
		builder.SetDictTypeID(req.Data.GetTypeId())
	}

	if req.Data.Id != nil {
		builder.SetID(req.GetData().GetId())
	}

	var entity *ent.DictEntry
	if entity, err = builder.Save(ctx); err != nil {
		r.log.Errorf("insert dict entry failed: %s", err.Error())
		return dictV1.ErrorInternalServerError("insert dict entry failed")
	}

	if len(req.Data.I18N) > 0 {
		if err = r.i18n.ReplaceByEntryID(
			ctx,
			tx,
			req.Data.GetTenantId(), req.Data.GetCreatedBy(),
			entity.ID,
			req.Data.I18N,
		); err != nil {
			return err
		}
	}

	return nil
}

func (r *DictEntryRepo) Update(ctx context.Context, req *dictV1.UpdateDictEntryRequest) (err error) {
	if req == nil || req.Data == nil {
		return dictV1.ErrorBadRequest("invalid parameter")
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		var exist bool
		exist, err = r.IsExist(ctx, req.GetId())
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

	var tx *ent.Tx
	tx, err = r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf("start transaction failed: %s", err.Error())
		return dictV1.ErrorInternalServerError("start transaction failed")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.log.Errorf("transaction rollback failed: %s", rollbackErr.Error())
			}
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			r.log.Errorf("transaction commit failed: %s", commitErr.Error())
			err = dictV1.ErrorInternalServerError("transaction commit failed")
		}
	}()

	var hasI18n bool
	var i18n map[string]*dictV1.DictEntryI18N
	for n, p := range req.GetUpdateMask().GetPaths() {
		if strings.ToLower(p) == "i18n" {
			hasI18n = true
			req.GetUpdateMask().Paths = append(req.GetUpdateMask().GetPaths()[:n], req.GetUpdateMask().GetPaths()[n+1:]...)
			i18n = req.Data.I18N
			break
		}
	}

	builder := tx.DictEntry.UpdateOneID(req.GetId())
	dto, err := r.repository.UpdateOne(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *dictV1.DictEntry) {
			builder.
				SetNillableEntryValue(req.Data.EntryValue).
				SetNillableNumericValue(req.Data.NumericValue).
				SetNillableIsEnabled(req.Data.IsEnabled).
				SetNillableSortOrder(req.Data.SortOrder).
				SetNillableUpdatedBy(req.Data.UpdatedBy).
				SetUpdatedAt(time.Now())
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(dictentry.FieldID, req.GetId()))
		},
	)
	if err != nil {
		r.log.Errorf("update dict entry failed: %s", err.Error())
		return dictV1.ErrorInternalServerError("update dict entry failed")
	}

	if hasI18n && len(i18n) > 0 {
		if err = r.i18n.ReplaceByEntryID(
			ctx,
			tx,
			req.Data.GetTenantId(),
			req.Data.GetUpdatedBy(),
			dto.GetId(),
			i18n,
		); err != nil {
			return err
		}
	}

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

func (r *DictEntryRepo) ListByTypeCode(ctx context.Context, req *dictV1.ListDictEntryByTypeCodeRequest) (*dictV1.ListDictEntryByTypeCodeResponse, error) {
	if req == nil {
		return nil, dictV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().DictEntry.Query().
		Where(
			dictentry.HasDictTypeWith(
				dicttype.TypeCodeEQ(req.GetTypeCode()),
			),
			dictentry.IsEnabledEQ(true),
		).
		Order(ent.Asc(dictentry.FieldSortOrder))

	entities, err := builder.All(ctx)
	if err != nil {
		r.log.Errorf("query dict entry by type code failed: %s", err.Error())
		return nil, dictV1.ErrorInternalServerError("query dict entry by type code failed")
	}

	var dtos []*dictV1.DictEntry
	for _, entity := range entities {
		dtos = append(dtos, r.mapper.ToDTO(entity))
	}

	if req.GetLocal() != "" {
		var i18n *dictV1.DictEntryI18N
		for _, item := range dtos {
			i18n, err = r.i18n.GetByEntryIDAndLangCode(ctx, item.GetId(), req.GetLocal())
			if err != nil {
				return nil, err
			}
			item.I18N = map[string]*dictV1.DictEntryI18N{
				req.GetLocal(): i18n,
			}
		}
	} else {
		var i18ns map[string]*dictV1.DictEntryI18N
		for _, item := range dtos {
			i18ns, err = r.i18n.ListByEntryID(ctx, item.GetId())
			if err != nil {
				return nil, err
			}
			item.I18N = i18ns
		}
	}

	return &dictV1.ListDictEntryByTypeCodeResponse{
		Items: dtos,
	}, nil
}
