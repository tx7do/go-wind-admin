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
	"go-wind-admin/app/admin/service/internal/data/ent/dicttype"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	dictV1 "go-wind-admin/api/gen/go/dict/service/v1"
)

type DictTypeRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper *mapper.CopierMapper[dictV1.DictType, ent.DictType]

	repository *entCrud.Repository[
		ent.DictTypeQuery, ent.DictTypeSelect,
		ent.DictTypeCreate, ent.DictTypeCreateBulk,
		ent.DictTypeUpdate, ent.DictTypeUpdateOne,
		ent.DictTypeDelete,
		predicate.DictType,
		dictV1.DictType, ent.DictType,
	]

	i18n *DictTypeI18nRepo
}

func NewDictTypeRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
	i18n *DictTypeI18nRepo,
) *DictTypeRepo {
	repo := &DictTypeRepo{
		log:       ctx.NewLoggerHelper("dict-type/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[dictV1.DictType, ent.DictType](),
		i18n:      i18n,
	}

	repo.init()

	return repo
}

func (r *DictTypeRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.DictTypeQuery, ent.DictTypeSelect,
		ent.DictTypeCreate, ent.DictTypeCreateBulk,
		ent.DictTypeUpdate, ent.DictTypeUpdateOne,
		ent.DictTypeDelete,
		predicate.DictType,
		dictV1.DictType, ent.DictType,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

func (r *DictTypeRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().DictType.Query()
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

func (r *DictTypeRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*dictV1.ListDictTypeResponse, error) {
	if req == nil {
		return nil, dictV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().DictType.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &dictV1.ListDictTypeResponse{Total: 0, Items: nil}, nil
	}

	for _, item := range ret.Items {
		i18ns, err := r.i18n.Get(ctx, item.GetId())
		if err != nil {
			return nil, err
		}
		item.I18N = i18ns
	}

	return &dictV1.ListDictTypeResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *DictTypeRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().DictType.Query().
		Where(dicttype.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, dictV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *DictTypeRepo) Get(ctx context.Context, req *dictV1.GetDictTypeRequest) (*dictV1.DictType, error) {
	if req == nil {
		return nil, dictV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().DictType.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *dictV1.GetDictTypeRequest_Id:
		whereCond = append(whereCond, dicttype.IDEQ(req.GetId()))
	case *dictV1.GetDictTypeRequest_Code:
		builder.Where(dicttype.TypeCodeEQ(req.GetCode()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	i18ns, err := r.i18n.Get(ctx, dto.GetId())
	if err != nil {
		return nil, err
	}
	dto.I18N = i18ns

	return dto, err
}

func (r *DictTypeRepo) Create(ctx context.Context, req *dictV1.CreateDictTypeRequest) (err error) {
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

	builder := tx.DictType.Create().
		SetNillableTenantID(req.Data.TenantId).
		SetNillableTypeCode(req.Data.TypeCode).
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

	var entity *ent.DictType
	if entity, err = builder.Save(ctx); err != nil {
		r.log.Errorf("insert dict type failed: %s", err.Error())
		return dictV1.ErrorInternalServerError("insert dict type failed")
	}

	if req.Data.I18N != nil {
		if err = r.i18n.ReplaceByTypeID(
			ctx,
			tx,
			req.Data.GetTenantId(),
			req.Data.GetCreatedBy(),
			entity.ID,
			req.Data.I18N,
		); err != nil {
			return err
		}
	}

	return nil
}

func (r *DictTypeRepo) Update(ctx context.Context, req *dictV1.UpdateDictTypeRequest) (err error) {
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
			createReq := &dictV1.CreateDictTypeRequest{Data: req.Data}
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

	builder := tx.DictType.UpdateOneID(req.GetId())
	dto, err := r.repository.UpdateOne(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *dictV1.DictType) {
			builder.
				//SetNillableTypeCode(req.Data.TypeCode).
				SetNillableSortOrder(req.Data.SortOrder).
				SetNillableIsEnabled(req.Data.IsEnabled).
				SetNillableUpdatedBy(req.Data.UpdatedBy).
				SetNillableUpdatedAt(timeutil.TimestamppbToTime(req.Data.UpdatedAt))

			if req.Data.UpdatedAt == nil {
				builder.SetUpdatedAt(time.Now())
			}
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(dicttype.FieldID, req.GetId()))
		},
	)
	if err != nil {
		r.log.Errorf("update dict type failed: %s", err.Error())
		return dictV1.ErrorInternalServerError("update dict type failed")
	}

	if err = r.i18n.ReplaceByTypeID(
		ctx,
		tx,
		req.Data.GetTenantId(),
		req.Data.GetCreatedBy(),
		dto.GetId(),
		req.Data.I18N,
	); err != nil {
		return err
	}

	return err
}

func (r *DictTypeRepo) Delete(ctx context.Context, id uint32) error {
	if id == 0 {
		return dictV1.ErrorBadRequest("invalid parameter")
	}

	if err := r.entClient.Client().DictType.DeleteOneID(id).Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return dictV1.ErrorNotFound("dict not found")
		}

		r.log.Errorf("delete one data failed: %s", err.Error())

		return dictV1.ErrorInternalServerError("delete failed")
	}

	return nil
}

func (r *DictTypeRepo) BatchDelete(ctx context.Context, ids []uint32) error {
	if len(ids) == 0 {
		return dictV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := r.entClient.Client().DictType.Delete().
		Where(dicttype.IDIn(ids...)).
		Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return dictV1.ErrorNotFound("dict not found")
		}

		r.log.Errorf("delete one data failed: %s", err.Error())

		return dictV1.ErrorInternalServerError("delete failed")
	}

	return nil
}
