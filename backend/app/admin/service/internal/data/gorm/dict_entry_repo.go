//go:build gorm_backend
// +build gorm_backend

package gorm

import (
	"context"
	"errors"

	gormDB "gorm.io/gorm"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	gormCrud "github.com/tx7do/go-crud/gorm"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/admin/service/internal/data/gorm/models"

	dictV1 "go-wind-admin/api/gen/go/dict/service/v1"
)

type DictEntryRepo struct {
	client     *gormCrud.Client
	log        *bLogger.Helper
	mapper     *mapper.CopierMapper[dictV1.DictEntry, models.DictEntry]
	repository *gormCrud.Repository[dictV1.DictEntry, models.DictEntry]
}

func NewDictEntryRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *DictEntryRepo {
	repo := &DictEntryRepo{
		log:    ctx.NewLoggerHelper("dict-entry/gorm-repo/admin-service"),
		client: client,
		mapper: mapper.NewCopierMapper[dictV1.DictEntry, models.DictEntry](),
	}

	repo.init()

	return repo
}

func (r *DictEntryRepo) init() {
	r.repository = gormCrud.NewRepository[dictV1.DictEntry, models.DictEntry](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

func (r *DictEntryRepo) Count(ctx context.Context, scopes []func(*gormDB.DB) *gormDB.DB) (int, error) {
	count, err := r.repository.Count(ctx, r.client.DB, scopes)
	if err != nil {
		r.log.Errorf(ctx, "query count failed: %s", err.Error())
		return 0, dictV1.ErrorInternalServerError("query count failed")
	}

	return int(count), nil
}

func (r *DictEntryRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*dictV1.ListDictEntryResponse, error) {
	return nil, dictV1.ErrorInternalServerError("gorm scaffold: List not implemented — ent .WithDictType/HasDictTypeWith/.Edges edge predicate has no go-crud/gorm primitive; see data/dict_entry_repo.go")
}

func (r *DictEntryRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.repository.ExistsWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", id) },
	})
	if err != nil {
		r.log.Errorf(ctx, "query exist failed: %s", err.Error())
		return false, dictV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *DictEntryRepo) Get(ctx context.Context, req *dictV1.GetDictEntryRequest) (*dictV1.DictEntry, error) {
	if req == nil {
		return nil, dictV1.ErrorBadRequest("invalid parameter")
	}

	var scopes []func(*gormDB.DB) *gormDB.DB
	switch req.QueryBy.(type) {
	default:
	case *dictV1.GetDictEntryRequest_Id:
		scopes = append(scopes, func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) })
	case *dictV1.GetDictEntryRequest_Value:
		scopes = append(scopes, func(db *gormDB.DB) *gormDB.DB { return db.Where("entry_value = ?", req.GetValue()) })
	}

	dto, err := r.repository.GetWithFilters(ctx, r.client.DB, scopes, req.GetViewMask())
	if err != nil {
		if errors.Is(err, gormDB.ErrRecordNotFound) {
			return nil, dictV1.ErrorNotFound("dict not found")
		}
		r.log.Errorf(ctx, "query dict entry failed: %s", err.Error())
		return nil, dictV1.ErrorInternalServerError("query dict entry failed")
	}

	return dto, nil
}

func (r *DictEntryRepo) Create(ctx context.Context, req *dictV1.CreateDictEntryRequest) error {
	return dictV1.ErrorInternalServerError("gorm scaffold: Create not implemented — ent tx/.Edges/ReplaceByEntryID edge write has no go-crud/gorm primitive; see data/dict_entry_repo.go")
}

func (r *DictEntryRepo) Update(ctx context.Context, req *dictV1.UpdateDictEntryRequest) error {
	return dictV1.ErrorInternalServerError("gorm scaffold: Update not implemented — ent tx/.Edges/ReplaceByEntryID edge write has no go-crud/gorm primitive; see data/dict_entry_repo.go")
}

func (r *DictEntryRepo) Delete(ctx context.Context, id uint32) error {
	if id == 0 {
		return dictV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", id) },
	}); err != nil {
		r.log.Errorf(ctx, "delete dict entry failed: %s", err.Error())
		return dictV1.ErrorInternalServerError("delete failed")
	}

	return nil
}

func (r *DictEntryRepo) BatchDelete(ctx context.Context, ids []uint32) error {
	if len(ids) == 0 {
		return dictV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id IN ?", ids) },
	}); err != nil {
		r.log.Errorf(ctx, "delete dict entry failed: %s", err.Error())
		return dictV1.ErrorInternalServerError("delete failed")
	}

	return nil
}

func (r *DictEntryRepo) ListByTypeCode(ctx context.Context, req *dictV1.ListDictEntryByTypeCodeRequest) (*dictV1.ListDictEntryByTypeCodeResponse, error) {
	return nil, dictV1.ErrorInternalServerError("gorm scaffold: ListByTypeCode not implemented — ent HasDictTypeWith edge predicate has no go-crud/gorm primitive; see data/dict_entry_repo.go")
}
