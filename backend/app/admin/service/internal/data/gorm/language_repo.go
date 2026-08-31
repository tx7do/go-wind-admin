//go:build gorm_backend
// +build gorm_backend

// Package gorm 中的仓储是 ent 仓储的平行 gorm 镜像，作为"ent 为主力、gorm 为备选"脚手架的完整代码。
// 这些仓储仅由 cmd/server/wiring_gorm.go(gorm_backend 构建,ORM 切换 Phase 4 占位)装配,服务层尚未接入。
//
// gorm 仓储不做租户隔离（ent 侧靠编译进生成代码的 privacy 策略自动注入，gorm 侧无此机制）。
// 直接切换 gorm 后端会有跨租户数据泄露风险，采用者须自行加 scope/plugin。
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

type LanguageRepo struct {
	client     *gormCrud.Client
	log        *bLogger.Helper
	mapper     *mapper.CopierMapper[dictV1.Language, models.Language]
	repository *gormCrud.Repository[dictV1.Language, models.Language]
}

func NewLanguageRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *LanguageRepo {
	repo := &LanguageRepo{
		log:    ctx.NewLoggerHelper("language/gorm-repo/admin-service"),
		client: client,
		mapper: mapper.NewCopierMapper[dictV1.Language, models.Language](),
	}

	repo.init()

	return repo
}

func (r *LanguageRepo) init() {
	r.repository = gormCrud.NewRepository[dictV1.Language, models.Language](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

func (r *LanguageRepo) Count(ctx context.Context, scopes []func(*gormDB.DB) *gormDB.DB) (int, error) {
	count, err := r.repository.Count(ctx, r.client.DB, scopes)
	if err != nil {
		r.log.Errorf(ctx, "query count failed: %s", err.Error())
		return 0, dictV1.ErrorInternalServerError("query count failed")
	}

	return int(count), nil
}

func (r *LanguageRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*dictV1.ListLanguageResponse, error) {
	if req == nil {
		return nil, dictV1.ErrorBadRequest("invalid parameter")
	}

	ret, err := r.repository.ListWithPaging(ctx, r.client.DB, req)
	if err != nil {
		return nil, dictV1.ErrorInternalServerError("query list failed")
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
	exist, err := r.repository.ExistsWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", id) },
	})
	if err != nil {
		r.log.Errorf(ctx, "query exist failed: %s", err.Error())
		return false, dictV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *LanguageRepo) Get(ctx context.Context, req *dictV1.GetLanguageRequest) (*dictV1.Language, error) {
	if req == nil {
		return nil, dictV1.ErrorBadRequest("invalid parameter")
	}

	var scopes []func(*gormDB.DB) *gormDB.DB
	switch req.QueryBy.(type) {
	default:
	case *dictV1.GetLanguageRequest_Id:
		scopes = append(scopes, func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) })
	case *dictV1.GetLanguageRequest_Code:
		scopes = append(scopes, func(db *gormDB.DB) *gormDB.DB { return db.Where("language_code = ?", req.GetCode()) })
	}

	dto, err := r.repository.GetWithFilters(ctx, r.client.DB, scopes, req.GetViewMask())
	if err != nil {
		if errors.Is(err, gormDB.ErrRecordNotFound) {
			return nil, dictV1.ErrorNotFound("language not found")
		}
		r.log.Errorf(ctx, "query language failed: %s", err.Error())
		return nil, dictV1.ErrorInternalServerError("query language failed")
	}

	return dto, nil
}

// ListLanguageByIds 通过多个ID获取语言信息
func (r *LanguageRepo) ListLanguageByIds(ctx context.Context, ids []uint32) ([]*dictV1.Language, error) {
	if len(ids) == 0 {
		return []*dictV1.Language{}, nil
	}

	var entities []*models.Language
	if err := r.client.DB.WithContext(ctx).Model(&models.Language{}).Where("id IN ?", ids).Find(&entities).Error; err != nil {
		r.log.Errorf(ctx, "query language by ids failed: %s", err.Error())
		return nil, dictV1.ErrorInternalServerError("query language by ids failed")
	}

	dtos := make([]*dictV1.Language, 0, len(entities))
	for _, entity := range entities {
		dtos = append(dtos, r.mapper.ToDTO(entity))
	}

	return dtos, nil
}

func (r *LanguageRepo) Create(ctx context.Context, req *dictV1.CreateLanguageRequest) error {
	if req == nil || req.Data == nil {
		return dictV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := r.repository.Create(ctx, r.client.DB, req.Data, nil); err != nil {
		r.log.Errorf(ctx, "insert language failed: %s", err.Error())
		return dictV1.ErrorInternalServerError("insert data failed")
	}

	return nil
}

func (r *LanguageRepo) Update(ctx context.Context, req *dictV1.UpdateLanguageRequest) error {
	if req == nil || req.Data == nil {
		return dictV1.ErrorBadRequest("invalid parameter")
	}
	if req.GetId() == 0 {
		return dictV1.ErrorBadRequest("id is required")
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

	if _, err := r.repository.UpdateWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) },
	}, req.Data, req.GetUpdateMask()); err != nil {
		r.log.Errorf(ctx, "update language failed: %s", err.Error())
		return dictV1.ErrorInternalServerError("update language failed")
	}

	return nil
}

func (r *LanguageRepo) Delete(ctx context.Context, req *dictV1.DeleteLanguageRequest) error {
	if req == nil {
		return dictV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) },
	}); err != nil {
		r.log.Errorf(ctx, "delete language failed: %s", err.Error())
		return dictV1.ErrorInternalServerError("delete language failed")
	}

	return nil
}
