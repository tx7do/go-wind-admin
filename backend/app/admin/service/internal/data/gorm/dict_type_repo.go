//go:build gorm_backend
// +build gorm_backend

// Package gorm 中的仓储是 ent 仓储的平行 gorm 镜像，作为"ent 为主力、gorm 为备选"脚手架的完整代码。
// 这些仓储为死代码：未接入 cmd/server/wiring.go、不被 service 引用；采用者需要时自行装配。
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

	dictV1 "go-wind-admin/api/gen/go/dict/service/v1"
)

type DictTypeRepo struct {
	client     *gormCrud.Client
	log        *log.Helper
	mapper     *mapper.CopierMapper[dictV1.DictType, models.DictType]
	repository *gormCrud.Repository[dictV1.DictType, models.DictType]
}

func NewDictTypeRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *DictTypeRepo {
	repo := &DictTypeRepo{
		log:    ctx.NewLoggerHelper("dict-type/gorm-repo/admin-service"),
		client: client,
		mapper: mapper.NewCopierMapper[dictV1.DictType, models.DictType](),
	}

	repo.init()

	return repo
}

func (r *DictTypeRepo) init() {
	r.repository = gormCrud.NewRepository[dictV1.DictType, models.DictType](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

func (r *DictTypeRepo) Count(ctx context.Context, scopes []func(*gormDB.DB) *gormDB.DB) (int, error) {
	count, err := r.repository.Count(ctx, r.client.DB, scopes)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, dictV1.ErrorInternalServerError("query count failed")
	}

	return int(count), nil
}

func (r *DictTypeRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*dictV1.ListDictTypeResponse, error) {
	if req == nil {
		return nil, dictV1.ErrorBadRequest("invalid parameter")
	}

	ret, err := r.repository.ListWithPaging(ctx, r.client.DB, req)
	if err != nil {
		return nil, dictV1.ErrorInternalServerError("query list failed")
	}
	if ret == nil {
		return &dictV1.ListDictTypeResponse{Total: 0, Items: nil}, nil
	}

	return &dictV1.ListDictTypeResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *DictTypeRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.repository.ExistsWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", id) },
	})
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

	var scopes []func(*gormDB.DB) *gormDB.DB
	switch req.QueryBy.(type) {
	default:
	case *dictV1.GetDictTypeRequest_Id:
		scopes = append(scopes, func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) })
	case *dictV1.GetDictTypeRequest_Code:
		scopes = append(scopes, func(db *gormDB.DB) *gormDB.DB { return db.Where("type_code = ?", req.GetCode()) })
	}

	dto, err := r.repository.GetWithFilters(ctx, r.client.DB, scopes, req.GetViewMask())
	if err != nil {
		if errors.Is(err, gormDB.ErrRecordNotFound) {
			return nil, dictV1.ErrorNotFound("dict type not found")
		}
		r.log.Errorf("query dict type failed: %s", err.Error())
		return nil, dictV1.ErrorInternalServerError("query dict type failed")
	}

	return dto, nil
}

func (r *DictTypeRepo) Create(ctx context.Context, req *dictV1.CreateDictTypeRequest) error {
	if req == nil || req.Data == nil {
		return dictV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := r.repository.Create(ctx, r.client.DB, req.Data, nil); err != nil {
		r.log.Errorf("insert dict type failed: %s", err.Error())
		return dictV1.ErrorInternalServerError("insert data failed")
	}

	return nil
}

func (r *DictTypeRepo) Update(ctx context.Context, req *dictV1.UpdateDictTypeRequest) error {
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
			createReq := &dictV1.CreateDictTypeRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	if _, err := r.repository.UpdateWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) },
	}, req.Data, req.GetUpdateMask()); err != nil {
		r.log.Errorf("update dict type failed: %s", err.Error())
		return dictV1.ErrorInternalServerError("update dict type failed")
	}

	return nil
}

func (r *DictTypeRepo) Delete(ctx context.Context, id uint32) error {
	if id == 0 {
		return dictV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", id) },
	}); err != nil {
		r.log.Errorf("delete dict type failed: %s", err.Error())
		return dictV1.ErrorInternalServerError("delete dict type failed")
	}

	return nil
}

func (r *DictTypeRepo) BatchDelete(ctx context.Context, ids []uint32) error {
	if len(ids) == 0 {
		return dictV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id IN ?", ids) },
	}); err != nil {
		r.log.Errorf("delete dict type failed: %s", err.Error())
		return dictV1.ErrorInternalServerError("delete dict type failed")
	}

	return nil
}
