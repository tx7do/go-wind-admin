//go:build gorm_backend
// +build gorm_backend

package gorm

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	gormCrud "github.com/tx7do/go-crud/gorm"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/admin/service/internal/data/gorm/models"

	dictV1 "go-wind-admin/api/gen/go/dict/service/v1"
)

type DictEntryI18nRepo struct {
	client     *gormCrud.Client
	log        *log.Helper
	mapper     *mapper.CopierMapper[dictV1.DictEntryI18N, models.DictEntryI18n]
	repository *gormCrud.Repository[dictV1.DictEntryI18N, models.DictEntryI18n]
}

// NewDictEntryI18nRepo creates a new DictEntryI18nRepo
func NewDictEntryI18nRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *DictEntryI18nRepo {
	repo := &DictEntryI18nRepo{
		log:       ctx.NewLoggerHelper("dict-entry-i18n/gorm-repo/admin-service"),
		client:    client,
		mapper:    mapper.NewCopierMapper[dictV1.DictEntryI18N, models.DictEntryI18n](),
	}

	repo.init()

	return repo
}

func (r *DictEntryI18nRepo) init() {
	r.repository = gormCrud.NewRepository[dictV1.DictEntryI18N, models.DictEntryI18n](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

// Upsert 新增或更新字典类型多语言数据
func (r *DictEntryI18nRepo) Upsert(ctx context.Context,
	tenantID, operatorID, entryID uint32,
	langCode string, data *dictV1.DictEntryI18N,
) error {
	if data == nil {
		return dictV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := r.repository.UpsertWithFilters(ctx, r.client.DB, nil, data, nil); err != nil {
		r.log.Errorf("upsert dict entry i18n failed: %s", err.Error())
		return dictV1.ErrorInternalServerError("upsert dict entry i18n failed")
	}

	return nil
}

// ListByEntryID 根据字典项ID查询多语言数据列表
func (r *DictEntryI18nRepo) ListByEntryID(ctx context.Context, entryID uint32) (map[string]*dictV1.DictEntryI18N, error) {
	return nil, dictV1.ErrorInternalServerError("gorm scaffold: ListByEntryID not implemented — ent .WithDictEntry/HasDictEntryWith edge predicate has no go-crud/gorm primitive; see data/dict_entry_i18n_repo.go")
}

// GetByEntryIDAndLangCode 根据字典项ID和语言代码查询多语言数据
func (r *DictEntryI18nRepo) GetByEntryIDAndLangCode(ctx context.Context, entryID uint32, langCode string) (*dictV1.DictEntryI18N, error) {
	return nil, dictV1.ErrorInternalServerError("gorm scaffold: GetByEntryIDAndLangCode not implemented — ent .WithDictEntry/HasDictEntryWith edge predicate has no go-crud/gorm primitive; see data/dict_entry_i18n_repo.go")
}

// Truncate 清理字典类型多语言数据
func (r *DictEntryI18nRepo) Truncate(ctx context.Context) error {
	if err := r.client.DB.WithContext(ctx).Where("1 = 1").Delete(&models.DictEntryI18n{}).Error; err != nil {
		r.log.Errorf("truncate dict entry i18n failed: %s", err.Error())
		return dictV1.ErrorInternalServerError("truncate dict entry i18n failed")
	}
	return nil
}

// CleanByEntryID 根据字典项ID清理多语言数据
func (r *DictEntryI18nRepo) CleanByEntryID(ctx context.Context, tx interface{}, entryID uint32) error {
	return dictV1.ErrorInternalServerError("gorm scaffold: CleanByEntryID not implemented — ent HasDictEntryWith edge predicate + tx has no go-crud/gorm primitive; see data/dict_entry_i18n_repo.go")
}

// CleanByEntryIDs 根据字典项ID清理多语言数据
func (r *DictEntryI18nRepo) CleanByEntryIDs(ctx context.Context, entryIDs []uint32) error {
	return dictV1.ErrorInternalServerError("gorm scaffold: CleanByEntryIDs not implemented — ent HasDictEntryWith edge predicate has no go-crud/gorm primitive; see data/dict_entry_i18n_repo.go")
}

// ReplaceByEntryID 根据字典类型ID替换多语言数据
func (r *DictEntryI18nRepo) ReplaceByEntryID(
	ctx context.Context,
	tx interface{},
	tenantID, operatorID uint32,
	entryID uint32, items map[string]*dictV1.DictEntryI18N,
) error {
	return dictV1.ErrorInternalServerError("gorm scaffold: ReplaceByEntryID not implemented — ent HasDictEntryWith edge predicate + tx has no go-crud/gorm primitive; see data/dict_entry_i18n_repo.go")
}
