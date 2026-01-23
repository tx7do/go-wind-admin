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
	"go-wind-admin/app/admin/service/internal/data/ent/dictentry"
	"go-wind-admin/app/admin/service/internal/data/ent/dictentryi18n"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	dictV1 "go-wind-admin/api/gen/go/dict/service/v1"
)

type DictEntryI18nRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper *mapper.CopierMapper[dictV1.DictEntryI18N, ent.DictEntryI18n]

	repository *entCrud.Repository[
		ent.DictEntryI18nQuery, ent.DictEntryI18nSelect,
		ent.DictEntryI18nCreate, ent.DictEntryI18nCreateBulk,
		ent.DictEntryI18nUpdate, ent.DictEntryI18nUpdateOne,
		ent.DictEntryI18nDelete,
		predicate.DictEntryI18n,
		dictV1.DictEntryI18N, ent.DictEntryI18n,
	]
}

// NewDictEntryI18nRepo creates a new DictEntryI18nRepo
func NewDictEntryI18nRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *DictEntryI18nRepo {
	repo := &DictEntryI18nRepo{
		log:       ctx.NewLoggerHelper("dict-entry-i18n/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[dictV1.DictEntryI18N, ent.DictEntryI18n](),
	}

	repo.init()

	return repo
}

func (r *DictEntryI18nRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.DictEntryI18nQuery, ent.DictEntryI18nSelect,
		ent.DictEntryI18nCreate, ent.DictEntryI18nCreateBulk,
		ent.DictEntryI18nUpdate, ent.DictEntryI18nUpdateOne,
		ent.DictEntryI18nDelete,
		predicate.DictEntryI18n,
		dictV1.DictEntryI18N, ent.DictEntryI18n,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

// Upsert 新增或更新字典类型多语言数据
func (r *DictEntryI18nRepo) Upsert(ctx context.Context,
	tenantID, operatorID, entryID uint32,
	langCode string, data *dictV1.DictEntryI18N,
) error {
	now := time.Now()
	var err error
	err = r.entClient.Client().DictEntryI18n.Create().
		SetTenantID(tenantID).
		SetLanguageCode(langCode).
		SetSysDictEntriesID(entryID).
		SetEntryLabel(data.GetEntryLabel()).
		SetDescription(data.GetDescription()).
		SetCreatedBy(operatorID).
		SetCreatedAt(now).
		OnConflictColumns(
			dictentryi18n.FieldLanguageCode,
			dictentryi18n.SysDictEntriesColumn,
		).
		SetEntryLabel(data.GetEntryLabel()).
		SetDescription(data.GetDescription()).
		SetUpdatedBy(operatorID).
		SetUpdatedAt(now).
		Exec(ctx)
	return err
}

// Get 获取字典项多语言数据
func (r *DictEntryI18nRepo) Get(ctx context.Context, entryID uint32) (map[string]*dictV1.DictEntryI18N, error) {

	entities, err := r.entClient.Client().DictEntryI18n.Query().
		WithSysDictEntries(func(query *ent.DictEntryQuery) {
			query.Where(dictentry.IDEQ(entryID))
		}).
		Where(dictentryi18n.HasSysDictEntriesWith(dictentry.IDEQ(entryID))).
		All(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string]*dictV1.DictEntryI18N)
	for _, entity := range entities {
		if entity.LanguageCode == nil {
			continue
		}

		dto := r.mapper.ToDTO(entity)
		result[*entity.LanguageCode] = dto
	}

	return result, nil
}

// Truncate 清理字典类型多语言数据
func (r *DictEntryI18nRepo) Truncate(ctx context.Context) error {
	_, err := r.entClient.Client().DictEntryI18n.Delete().Exec(ctx)
	return err
}

// CleanByEntryID 根据字典项ID清理多语言数据
func (r *DictEntryI18nRepo) CleanByEntryID(ctx context.Context, tx *ent.Tx, entryID uint32) error {
	_, err := tx.DictEntryI18n.Delete().
		Where(dictentryi18n.HasSysDictEntriesWith(dictentry.IDEQ(entryID))).
		Exec(ctx)
	return err
}

// CleanByEntryIDs 根据字典项ID清理多语言数据
func (r *DictEntryI18nRepo) CleanByEntryIDs(ctx context.Context, entryIDs []uint32) error {
	_, err := r.entClient.Client().DictEntryI18n.Delete().
		Where(dictentryi18n.HasSysDictEntriesWith(dictentry.IDIn(entryIDs...))).
		Exec(ctx)
	return err
}

// ReplaceByEntryID 根据字典类型ID替换多语言数据
func (r *DictEntryI18nRepo) ReplaceByEntryID(
	ctx context.Context,
	tx *ent.Tx,
	tenantID, operatorID uint32,
	entryID uint32, items map[string]*dictV1.DictEntryI18N,
) (err error) {
	if err = r.CleanByEntryID(ctx, tx, entryID); err != nil {
		return err
	}

	if len(items) == 0 {
		return nil
	}

	now := time.Now()
	var dictEntryI18nCreates []*ent.DictEntryI18nCreate
	for langCode, item := range items {
		dictEntryI18nCreate := tx.DictEntryI18n.Create().
			SetTenantID(tenantID).
			SetLanguageCode(langCode).
			SetSysDictEntriesID(entryID).
			SetEntryLabel(item.GetEntryLabel()).
			SetDescription(item.GetDescription()).
			SetCreatedBy(operatorID).
			SetCreatedAt(now)
		dictEntryI18nCreates = append(dictEntryI18nCreates, dictEntryI18nCreate)
	}

	if err = tx.DictEntryI18n.CreateBulk(dictEntryI18nCreates...).Exec(ctx); err != nil {
		r.log.Errorf("bulk insert dict entry i18n failed: %s", err.Error())
		return dictV1.ErrorInternalServerError("bulk insert dict entry i18n failed")
	}

	return err
}
