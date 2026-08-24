//go:build gorm_backend
// +build gorm_backend

package gorm

import (
	"context"
	"errors"

	gormDB "gorm.io/gorm"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	gormCrud "github.com/tx7do/go-crud/gorm"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/admin/service/internal/data/gorm/models"

	internalMessageV1 "go-wind-admin/api/gen/go/internal_message/service/v1"
)

type InternalMessageCategoryRepo struct {
	client     *gormCrud.Client
	log        *log.Helper
	mapper     *mapper.CopierMapper[internalMessageV1.InternalMessageCategory, models.InternalMessageCategory]
	repository *gormCrud.Repository[internalMessageV1.InternalMessageCategory, models.InternalMessageCategory]
}

func NewInternalMessageCategoryRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *InternalMessageCategoryRepo {
	repo := &InternalMessageCategoryRepo{
		log:       ctx.NewLoggerHelper("internal-message-category/gorm-repo/admin-service"),
		client:    client,
		mapper:    mapper.NewCopierMapper[internalMessageV1.InternalMessageCategory, models.InternalMessageCategory](),
	}

	repo.init()

	return repo
}

func (r *InternalMessageCategoryRepo) init() {
	r.repository = gormCrud.NewRepository[internalMessageV1.InternalMessageCategory, models.InternalMessageCategory](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

func (r *InternalMessageCategoryRepo) Count(ctx context.Context, scopes []func(*gormDB.DB) *gormDB.DB) (int, error) {
	count, err := r.repository.Count(ctx, r.client.DB, scopes)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, internalMessageV1.ErrorInternalServerError("query count failed")
	}

	return int(count), nil
}

func (r *InternalMessageCategoryRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*internalMessageV1.ListInternalMessageCategoryResponse, error) {
	if req == nil {
		return nil, internalMessageV1.ErrorBadRequest("invalid parameter")
	}

	ret, err := r.repository.ListWithPaging(ctx, r.client.DB, req)
	if err != nil {
		return nil, internalMessageV1.ErrorInternalServerError("query list failed")
	}
	if ret == nil {
		return &internalMessageV1.ListInternalMessageCategoryResponse{Total: 0, Items: nil}, nil
	}

	return &internalMessageV1.ListInternalMessageCategoryResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *InternalMessageCategoryRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.repository.ExistsWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", id) },
	})
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, internalMessageV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *InternalMessageCategoryRepo) Get(ctx context.Context, req *internalMessageV1.GetInternalMessageCategoryRequest) (*internalMessageV1.InternalMessageCategory, error) {
	if req == nil {
		return nil, internalMessageV1.ErrorBadRequest("invalid parameter")
	}

	var scopes []func(*gormDB.DB) *gormDB.DB
	switch req.QueryBy.(type) {
	default:
	case *internalMessageV1.GetInternalMessageCategoryRequest_Id:
		scopes = append(scopes, func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) })
	}

	dto, err := r.repository.GetWithFilters(ctx, r.client.DB, scopes, req.GetViewMask())
	if err != nil {
		if errors.Is(err, gormDB.ErrRecordNotFound) {
			return nil, internalMessageV1.ErrorNotFound("internal message category not found")
		}
		r.log.Errorf("query internal message category failed: %s", err.Error())
		return nil, internalMessageV1.ErrorInternalServerError("query internal message category failed")
	}

	return dto, nil
}

// ListCategoriesByIds 根据ID列表获取分类列表
func (r *InternalMessageCategoryRepo) ListCategoriesByIds(ctx context.Context, ids []uint32) ([]*internalMessageV1.InternalMessageCategory, error) {
	if len(ids) == 0 {
		return []*internalMessageV1.InternalMessageCategory{}, nil
	}

	var entities []*models.InternalMessageCategory
	if err := r.client.DB.WithContext(ctx).Model(&models.InternalMessageCategory{}).Where("id IN ?", ids).Find(&entities).Error; err != nil {
		r.log.Errorf("query internal message category by ids failed: %s", err.Error())
		return nil, internalMessageV1.ErrorInternalServerError("query internal message category by ids failed")
	}

	dtos := make([]*internalMessageV1.InternalMessageCategory, 0, len(entities))
	for _, entity := range entities {
		dtos = append(dtos, r.mapper.ToDTO(entity))
	}

	return dtos, nil
}

func (r *InternalMessageCategoryRepo) Create(ctx context.Context, req *internalMessageV1.CreateInternalMessageCategoryRequest) error {
	if req == nil || req.Data == nil {
		return internalMessageV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := r.repository.Create(ctx, r.client.DB, req.Data, nil); err != nil {
		r.log.Errorf("insert internal message category failed: %s", err.Error())
		return internalMessageV1.ErrorInternalServerError("insert internal message category failed")
	}

	return nil
}

func (r *InternalMessageCategoryRepo) Update(ctx context.Context, req *internalMessageV1.UpdateInternalMessageCategoryRequest) error {
	if req == nil || req.Data == nil {
		return internalMessageV1.ErrorBadRequest("invalid parameter")
	}
	if req.GetId() == 0 {
		return internalMessageV1.ErrorBadRequest("id is required")
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return err
		}
		if !exist {
			createReq := &internalMessageV1.CreateInternalMessageCategoryRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	if _, err := r.repository.UpdateWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) },
	}, req.Data, req.GetUpdateMask()); err != nil {
		r.log.Errorf("update internal message category failed: %s", err.Error())
		return internalMessageV1.ErrorInternalServerError("update internal message category failed")
	}

	return nil
}

func (r *InternalMessageCategoryRepo) Delete(ctx context.Context, req *internalMessageV1.DeleteInternalMessageCategoryRequest) error {
	return internalMessageV1.ErrorInternalServerError("gorm scaffold: Delete not implemented — ent QueryAllChildrenIds tree-cascade has no go-crud/gorm primitive; see data/internal_message_category_repo.go")
}
