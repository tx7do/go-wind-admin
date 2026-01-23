package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"

	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/dicttype"
	"go-wind-admin/app/admin/service/internal/data/ent/dicttypei18n"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	dictV1 "go-wind-admin/api/gen/go/dict/service/v1"
)

type DictTypeI18nRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper *mapper.CopierMapper[dictV1.DictTypeI18N, ent.DictTypeI18n]

	repository *entCrud.Repository[
		ent.DictTypeI18nQuery, ent.DictTypeI18nSelect,
		ent.DictTypeI18nCreate, ent.DictTypeI18nCreateBulk,
		ent.DictTypeI18nUpdate, ent.DictTypeI18nUpdateOne,
		ent.DictTypeI18nDelete,
		predicate.DictTypeI18n,
		dictV1.DictTypeI18N, ent.DictTypeI18n,
	]
}

// NewDictTypeI18nRepo creates a new DictTypeI18nRepo
func NewDictTypeI18nRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *DictTypeI18nRepo {
	repo := &DictTypeI18nRepo{
		log:       ctx.NewLoggerHelper("dict-type-i18n/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[dictV1.DictTypeI18N, ent.DictTypeI18n](),
	}

	repo.init()

	return repo
}

func (r *DictTypeI18nRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.DictTypeI18nQuery, ent.DictTypeI18nSelect,
		ent.DictTypeI18nCreate, ent.DictTypeI18nCreateBulk,
		ent.DictTypeI18nUpdate, ent.DictTypeI18nUpdateOne,
		ent.DictTypeI18nDelete,
		predicate.DictTypeI18n,
		dictV1.DictTypeI18N, ent.DictTypeI18n,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

// Upsert 新增或更新字典类型多语言数据
func (r *DictTypeI18nRepo) Upsert(ctx context.Context,
	tenantID, operatorID, typeID uint32,
	langCode string, data *dictV1.DictTypeI18N,
) error {
	now := time.Now()
	var err error
	err = r.entClient.Client().DictTypeI18n.Create().
		SetTenantID(tenantID).
		SetLanguageCode(langCode).
		SetSysDictTypesID(typeID).
		SetTypeName(data.GetTypeName()).
		SetDescription(data.GetDescription()).
		SetCreatedBy(operatorID).
		SetCreatedAt(now).
		OnConflictColumns(
			dicttypei18n.FieldLanguageCode,
			dicttypei18n.SysDictTypesColumn,
		).
		SetTypeName(data.GetTypeName()).
		SetDescription(data.GetDescription()).
		SetUpdatedBy(operatorID).
		SetUpdatedAt(now).
		Exec(ctx)
	return err
}

// Get 获取字典类型多语言数据
func (r *DictTypeI18nRepo) Get(ctx context.Context, typeID uint32) (map[string]*dictV1.DictTypeI18N, error) {
	entities, err := r.entClient.Client().DictTypeI18n.Query().
		Where(dicttypei18n.HasSysDictTypesWith(dicttype.IDEQ(typeID))).
		All(ctx)
	if err != nil {
		r.log.Errorf("query dict type i18n failed: %s", err.Error())
		return nil, err
	}

	result := make(map[string]*dictV1.DictTypeI18N)
	for _, entity := range entities {
		if entity.LanguageCode == nil {
			continue
		}

		dto := r.mapper.ToDTO(entity)
		result[*entity.LanguageCode] = dto
	}

	return result, nil
}

// Truncate 清理所有多语言数据
func (r *DictTypeI18nRepo) Truncate(ctx context.Context) error {
	_, err := r.entClient.Client().DictTypeI18n.Delete().Exec(ctx)
	return err
}

// CleanByTypeID 根据字典类型ID清理多语言数据
func (r *DictTypeI18nRepo) CleanByTypeID(ctx context.Context, tx *ent.Tx, typeID uint32) error {
	_, err := tx.DictTypeI18n.Delete().
		Where(dicttypei18n.HasSysDictTypesWith(dicttype.IDEQ(typeID))).
		Exec(ctx)
	return err
}

// CleanByTypeIDs 根据字典类型ID列表清理多语言数据
func (r *DictTypeI18nRepo) CleanByTypeIDs(ctx context.Context, typeIDs []uint32) error {
	_, err := r.entClient.Client().DictTypeI18n.Delete().
		Where(dicttypei18n.HasSysDictTypesWith(dicttype.IDIn(typeIDs...))).
		Exec(ctx)
	return err
}

// ReplaceByTypeID 根据字典类型ID替换多语言数据
func (r *DictTypeI18nRepo) ReplaceByTypeID(
	ctx context.Context,
	tx *ent.Tx,
	tenantID, operatorID uint32,
	typeID uint32, items map[string]*dictV1.DictTypeI18N,
) (err error) {
	if err = r.CleanByTypeID(ctx, tx, typeID); err != nil {
		return err
	}

	if len(items) == 0 {
		return nil
	}

	now := time.Now()
	var dictTypeI18nCreates []*ent.DictTypeI18nCreate
	for langCode, item := range items {
		dictTypeI18nCreate := tx.DictTypeI18n.Create().
			SetTenantID(tenantID).
			SetLanguageCode(langCode).
			SetSysDictTypesID(typeID).
			SetTypeName(item.GetTypeName()).
			SetDescription(item.GetDescription()).
			SetCreatedBy(operatorID).
			SetCreatedAt(now)
		dictTypeI18nCreates = append(dictTypeI18nCreates, dictTypeI18nCreate)
	}

	if err = tx.DictTypeI18n.CreateBulk(dictTypeI18nCreates...).Exec(ctx); err != nil {
		r.log.Errorf("bulk insert dict type i18n failed: %s", err.Error())
		return dictV1.ErrorInternalServerError("bulk insert dict type i18n failed")
	}

	return err
}
