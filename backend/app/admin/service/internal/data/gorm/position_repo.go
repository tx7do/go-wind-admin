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

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
)

type PositionRepo struct {
	client          *gormCrud.Client
	log             *log.Helper
	mapper          *mapper.CopierMapper[identityV1.Position, models.Position]
	statusConverter *mapper.EnumTypeConverter[identityV1.Position_Status, string]
	typeConverter   *mapper.EnumTypeConverter[identityV1.Position_Type, string]
	repository      *gormCrud.Repository[identityV1.Position, models.Position]
}

func NewPositionRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *PositionRepo {
	repo := &PositionRepo{
		log:    ctx.NewLoggerHelper("position/gorm-repo/admin-service"),
		client: client,
		mapper: mapper.NewCopierMapper[identityV1.Position, models.Position](),
		statusConverter: mapper.NewEnumTypeConverter[identityV1.Position_Status, string](
			identityV1.Position_Status_name, identityV1.Position_Status_value,
		),
		typeConverter: mapper.NewEnumTypeConverter[identityV1.Position_Type, string](
			identityV1.Position_Type_name, identityV1.Position_Type_value,
		),
	}

	repo.init()

	return repo
}

func (r *PositionRepo) init() {
	r.repository = gormCrud.NewRepository[identityV1.Position, models.Position](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
	r.mapper.AppendConverters(r.typeConverter.NewConverterPair())
}

func (r *PositionRepo) Count(ctx context.Context, scopes []func(*gormDB.DB) *gormDB.DB) (int, error) {
	count, err := r.repository.Count(ctx, r.client.DB, scopes)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, identityV1.ErrorInternalServerError("query count failed")
	}

	return int(count), nil
}

func (r *PositionRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*identityV1.ListPositionResponse, error) {
	if req == nil {
		return nil, identityV1.ErrorBadRequest("invalid parameter")
	}

	ret, err := r.repository.ListWithPaging(ctx, r.client.DB, req)
	if err != nil {
		return nil, identityV1.ErrorInternalServerError("query list failed")
	}
	if ret == nil {
		return &identityV1.ListPositionResponse{Total: 0, Items: nil}, nil
	}

	return &identityV1.ListPositionResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *PositionRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.repository.ExistsWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", id) },
	})
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, identityV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *PositionRepo) Get(ctx context.Context, req *identityV1.GetPositionRequest) (*identityV1.Position, error) {
	if req == nil {
		return nil, identityV1.ErrorBadRequest("invalid parameter")
	}

	var scopes []func(*gormDB.DB) *gormDB.DB
	switch req.QueryBy.(type) {
	default:
	case *identityV1.GetPositionRequest_Id:
		scopes = append(scopes, func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) })
	case *identityV1.GetPositionRequest_Name:
		scopes = append(scopes, func(db *gormDB.DB) *gormDB.DB { return db.Where("name = ?", req.GetName()) })
	case *identityV1.GetPositionRequest_Code:
		scopes = append(scopes, func(db *gormDB.DB) *gormDB.DB { return db.Where("code = ?", req.GetCode()) })
	}

	dto, err := r.repository.GetWithFilters(ctx, r.client.DB, scopes, req.GetViewMask())
	if err != nil {
		if errors.Is(err, gormDB.ErrRecordNotFound) {
			return nil, identityV1.ErrorNotFound("position not found")
		}
		r.log.Errorf("query position failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query position failed")
	}

	return dto, nil
}

// ListPositionByIds 通过多个ID获取职位信息
func (r *PositionRepo) ListPositionByIds(ctx context.Context, ids []uint32) ([]*identityV1.Position, error) {
	if len(ids) == 0 {
		return []*identityV1.Position{}, nil
	}

	var entities []*models.Position
	if err := r.client.DB.WithContext(ctx).Model(&models.Position{}).Where("id IN ?", ids).Find(&entities).Error; err != nil {
		r.log.Errorf("query position by ids failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query position by ids failed")
	}

	dtos := make([]*identityV1.Position, 0, len(entities))
	for _, entity := range entities {
		dto := r.mapper.ToDTO(entity)
		dtos = append(dtos, dto)
	}

	return dtos, nil
}

func (r *PositionRepo) Create(ctx context.Context, req *identityV1.CreatePositionRequest) error {
	if req == nil || req.Data == nil {
		return identityV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := r.repository.Create(ctx, r.client.DB, req.Data, nil); err != nil {
		r.log.Errorf("insert position failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("insert data failed")
	}

	return nil
}

func (r *PositionRepo) Update(ctx context.Context, req *identityV1.UpdatePositionRequest) error {
	if req == nil || req.Data == nil {
		return identityV1.ErrorBadRequest("invalid parameter")
	}
	if req.GetId() == 0 {
		return identityV1.ErrorBadRequest("id is required")
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return err
		}
		if !exist {
			createReq := &identityV1.CreatePositionRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	if _, err := r.repository.UpdateWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) },
	}, req.Data, req.GetUpdateMask()); err != nil {
		r.log.Errorf("update position failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("update position failed")
	}

	return nil
}

func (r *PositionRepo) Delete(ctx context.Context, req *identityV1.DeletePositionRequest) error {
	if req == nil {
		return identityV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) },
	}); err != nil {
		r.log.Errorf("delete position failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("delete position failed")
	}

	return nil
}
