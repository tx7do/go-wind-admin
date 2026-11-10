package data

import (
	"context"
	"time"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"

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
	log       *bLogger.Helper

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

// Upsert 新增或更新字典类型多语言数据。
// (entry_id, language_code) 建不了唯一索引——ent DSL 不支持在从表 Indexes()
// 中引用边外键列，而 ON CONFLICT 的冲突目标必须精确匹配某个唯一索引，
// 之前的 OnConflictColumns 写法在 SQLite/PG 下整条被拒、从未生效。
// 改为查得则更新、查无则插入（经 schema Policy 的 Update/UpdateOne 租户过滤）。
func (r *DictEntryI18nRepo) Upsert(ctx context.Context,
	tenantID, operatorID, entryID int64,
	langCode string, data *dictV1.DictEntryI18N,
) error {
	now := time.Now()

	entity, err := r.entClient.Client().DictEntryI18n.Query().
		Where(
			dictentryi18n.HasDictEntryWith(dictentry.IDEQ(entryID)),
			dictentryi18n.LanguageCodeEQ(langCode),
		).
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return err
	}

	if ent.IsNotFound(err) {
		return r.entClient.Client().DictEntryI18n.Create().
			SetTenantID(tenantID).
			SetLanguageCode(langCode).
			SetDictEntryID(entryID).
			SetEntryLabel(data.GetEntryLabel()).
			SetDescription(data.GetDescription()).
			SetCreatedBy(operatorID).
			SetCreatedAt(now).
			Exec(ctx)
	}

	return r.entClient.Client().DictEntryI18n.UpdateOne(entity).
		SetEntryLabel(data.GetEntryLabel()).
		SetDescription(data.GetDescription()).
		SetUpdatedAt(now).
		SetUpdatedBy(operatorID).
		Exec(ctx)
}

// ListByEntryID 根据字典项ID查询多语言数据列表
func (r *DictEntryI18nRepo) ListByEntryID(ctx context.Context, entryID int64) (map[string]*dictV1.DictEntryI18N, error) {

	entities, err := r.entClient.Client().DictEntryI18n.Query().
		WithDictEntry(func(query *ent.DictEntryQuery) {
			query.Where(dictentry.IDEQ(entryID))
		}).
		Where(dictentryi18n.HasDictEntryWith(dictentry.IDEQ(entryID))).
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

// GetByEntryIDAndLangCode 根据字典项ID和语言代码查询多语言数据
func (r *DictEntryI18nRepo) GetByEntryIDAndLangCode(ctx context.Context, entryID int64, langCode string) (*dictV1.DictEntryI18N, error) {
	entity, err := r.entClient.Client().DictEntryI18n.Query().
		WithDictEntry(func(query *ent.DictEntryQuery) {
			query.Where(dictentry.IDEQ(entryID))
		}).
		Where(
			dictentryi18n.HasDictEntryWith(dictentry.IDEQ(entryID)),
			dictentryi18n.LanguageCodeEQ(langCode),
		).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	dto := r.mapper.ToDTO(entity)
	return dto, nil
}

// Truncate 清理字典类型多语言数据
func (r *DictEntryI18nRepo) Truncate(ctx context.Context) error {
	_, err := r.entClient.Client().DictEntryI18n.Delete().Exec(ctx)
	return err
}

// CleanByEntryID 根据字典项ID清理多语言数据
func (r *DictEntryI18nRepo) CleanByEntryID(ctx context.Context, tx *ent.Tx, entryID int64) error {
	_, err := tx.DictEntryI18n.Delete().
		Where(dictentryi18n.HasDictEntryWith(dictentry.IDEQ(entryID))).
		Exec(ctx)
	return err
}

// CleanByEntryIDs 根据字典项ID清理多语言数据
func (r *DictEntryI18nRepo) CleanByEntryIDs(ctx context.Context, entryIDs []int64) error {
	_, err := r.entClient.Client().DictEntryI18n.Delete().
		Where(dictentryi18n.HasDictEntryWith(dictentry.IDIn(entryIDs...))).
		Exec(ctx)
	return err
}

// ReplaceByEntryID 根据字典类型ID替换多语言数据
func (r *DictEntryI18nRepo) ReplaceByEntryID(
	ctx context.Context,
	tx *ent.Tx,
	tenantID, operatorID int64,
	entryID int64, items map[string]*dictV1.DictEntryI18N,
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
			SetDictEntryID(entryID).
			SetEntryLabel(item.GetEntryLabel()).
			SetDescription(item.GetDescription()).
			SetCreatedBy(operatorID).
			SetCreatedAt(now)
		dictEntryI18nCreates = append(dictEntryI18nCreates, dictEntryI18nCreate)
	}

	if err = tx.DictEntryI18n.CreateBulk(dictEntryI18nCreates...).Exec(ctx); err != nil {
		r.log.Errorf(ctx, "bulk insert dict entry i18n failed: %s", err.Error())
		return dictV1.ErrorInternalServerError("bulk insert dict entry i18n failed")
	}

	return err
}
