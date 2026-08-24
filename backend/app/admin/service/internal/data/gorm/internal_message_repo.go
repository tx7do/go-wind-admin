//go:build gorm_backend
// +build gorm_backend

// Package gorm 中的仓储是 ent 仓储的平行 gorm 镜像，作为"ent 为主力、gorm 为备选"脚手架的完整代码。
// 这些仓储为死代码：不接入 wire、不被 service 引用；采用者需要时自行装配。
//
// gorm 仓储不做租户隔离（ent 侧靠编译进生成代码的 privacy 策略自动注入，gorm 侧无此机制）。
// 直接切换 gorm 后端会有跨租户数据泄露风险，采用者须自行加 scope/plugin。
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

type InternalMessageRepo struct {
	client     *gormCrud.Client
	log        *log.Helper
	mapper     *mapper.CopierMapper[internalMessageV1.InternalMessage, models.InternalMessage]
	repository *gormCrud.Repository[internalMessageV1.InternalMessage, models.InternalMessage]

	statusConverter *mapper.EnumTypeConverter[internalMessageV1.InternalMessage_Status, string]
	typeConverter   *mapper.EnumTypeConverter[internalMessageV1.InternalMessage_Type, string]
}

func NewInternalMessageRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *InternalMessageRepo {
	repo := &InternalMessageRepo{
		log:    ctx.NewLoggerHelper("internal-message/gorm-repo/admin-service"),
		client: client,
		mapper: mapper.NewCopierMapper[internalMessageV1.InternalMessage, models.InternalMessage](),

		statusConverter: mapper.NewEnumTypeConverter[internalMessageV1.InternalMessage_Status, string](internalMessageV1.InternalMessage_Status_name, internalMessageV1.InternalMessage_Status_value),
		typeConverter:   mapper.NewEnumTypeConverter[internalMessageV1.InternalMessage_Type, string](internalMessageV1.InternalMessage_Type_name, internalMessageV1.InternalMessage_Type_value),
	}

	repo.init()

	return repo
}

func (r *InternalMessageRepo) init() {
	r.repository = gormCrud.NewRepository[internalMessageV1.InternalMessage, models.InternalMessage](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
	r.mapper.AppendConverters(r.typeConverter.NewConverterPair())
}

func (r *InternalMessageRepo) Count(ctx context.Context, scopes []func(*gormDB.DB) *gormDB.DB) (int, error) {
	count, err := r.repository.Count(ctx, r.client.DB, scopes)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, internalMessageV1.ErrorInternalServerError("query count failed")
	}

	return int(count), nil
}

func (r *InternalMessageRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*internalMessageV1.ListInternalMessageResponse, error) {
	if req == nil {
		return nil, internalMessageV1.ErrorBadRequest("invalid parameter")
	}

	ret, err := r.repository.ListWithPaging(ctx, r.client.DB, req)
	if err != nil {
		return nil, internalMessageV1.ErrorInternalServerError("query list failed")
	}
	if ret == nil {
		return &internalMessageV1.ListInternalMessageResponse{Total: 0, Items: nil}, nil
	}

	return &internalMessageV1.ListInternalMessageResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *InternalMessageRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.repository.ExistsWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", id) },
	})
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, internalMessageV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *InternalMessageRepo) Get(ctx context.Context, req *internalMessageV1.GetInternalMessageRequest) (*internalMessageV1.InternalMessage, error) {
	if req == nil {
		return nil, internalMessageV1.ErrorBadRequest("invalid parameter")
	}

	var scopes []func(*gormDB.DB) *gormDB.DB
	switch req.QueryBy.(type) {
	default:
	case *internalMessageV1.GetInternalMessageRequest_Id:
		scopes = append(scopes, func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) })
	}

	dto, err := r.repository.GetWithFilters(ctx, r.client.DB, scopes, req.GetViewMask())
	if err != nil {
		if errors.Is(err, gormDB.ErrRecordNotFound) {
			return nil, internalMessageV1.ErrorNotFound("internal message not found")
		}
		r.log.Errorf("query internal message failed: %s", err.Error())
		return nil, internalMessageV1.ErrorInternalServerError("query internal message failed")
	}

	return dto, nil
}

func (r *InternalMessageRepo) Create(ctx context.Context, req *internalMessageV1.CreateInternalMessageRequest) (*internalMessageV1.InternalMessage, error) {
	if req == nil || req.Data == nil {
		return nil, internalMessageV1.ErrorBadRequest("invalid parameter")
	}

	dto, err := r.repository.Create(ctx, r.client.DB, req.Data, nil)
	if err != nil {
		r.log.Errorf("insert internal message failed: %s", err.Error())
		return nil, internalMessageV1.ErrorInternalServerError("insert internal message failed")
	}

	return dto, nil
}

func (r *InternalMessageRepo) Update(ctx context.Context, req *internalMessageV1.UpdateInternalMessageRequest) error {
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
			createReq := &internalMessageV1.CreateInternalMessageRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			_, err = r.Create(ctx, createReq)
			return err
		}
	}

	if _, err := r.repository.UpdateWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) },
	}, req.Data, req.GetUpdateMask()); err != nil {
		r.log.Errorf("update internal message failed: %s", err.Error())
		return internalMessageV1.ErrorInternalServerError("update internal message failed")
	}

	return nil
}

func (r *InternalMessageRepo) Delete(ctx context.Context, id uint32) error {
	if id == 0 {
		return internalMessageV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", id) },
	}); err != nil {
		r.log.Errorf("delete internal message failed: %s", err.Error())
		return internalMessageV1.ErrorInternalServerError("delete failed")
	}

	return nil
}
