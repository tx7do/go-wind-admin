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
	pagination "github.com/tx7do/go-crud/pagination"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/admin/service/internal/data/gorm/models"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

type PermissionGroupRepo struct {
	client          *gormCrud.Client
	log             *log.Helper
	mapper          *mapper.CopierMapper[permissionV1.PermissionGroup, models.PermissionGroup]
	statusConverter *mapper.EnumTypeConverter[permissionV1.PermissionGroup_Status, string]

	repository *gormCrud.Repository[permissionV1.PermissionGroup, models.PermissionGroup]
}

func NewPermissionGroupRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *PermissionGroupRepo {
	repo := &PermissionGroupRepo{
		log:             ctx.NewLoggerHelper("permission-group/gorm-repo/admin-service"),
		client:          client,
		mapper:          mapper.NewCopierMapper[permissionV1.PermissionGroup, models.PermissionGroup](),
		statusConverter: mapper.NewEnumTypeConverter[permissionV1.PermissionGroup_Status, string](permissionV1.PermissionGroup_Status_name, permissionV1.PermissionGroup_Status_value),
	}

	repo.init()

	return repo
}

func (r *PermissionGroupRepo) init() {
	r.repository = gormCrud.NewRepository[permissionV1.PermissionGroup, models.PermissionGroup](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
}

func (r *PermissionGroupRepo) Count(ctx context.Context, scopes []func(*gormDB.DB) *gormDB.DB) (int, error) {
	count, err := r.repository.Count(ctx, r.client.DB, scopes)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, permissionV1.ErrorInternalServerError("query count failed")
	}

	return int(count), nil
}

func (r *PermissionGroupRepo) List(ctx context.Context, req *paginationV1.PagingRequest, treeTravel bool) (*permissionV1.ListPermissionGroupResponse, error) {
	if req == nil {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	var entities []*models.PermissionGroup
	if err := r.client.DB.WithContext(ctx).Model(&models.PermissionGroup{}).Find(&entities).Error; err != nil {
		r.log.Errorf("query permission group list failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query permission group list failed")
	}

	// 转换所有实体为 DTO
	dtos := make([]*permissionV1.PermissionGroup, 0, len(entities))
	for _, entity := range entities {
		dto := r.mapper.ToDTO(entity)
		dtos = append(dtos, dto)
	}

	// 构建树形结构
	if treeTravel {
		dtos = pagination.BuildTree(
			dtos,
			func(node *permissionV1.PermissionGroup) *uint32 { return node.Id },
			func(node *permissionV1.PermissionGroup) *uint32 { return node.ParentId },
			func(node *permissionV1.PermissionGroup) *[]*permissionV1.PermissionGroup { return &node.Children },
		)
	}

	count, err := r.Count(ctx, nil)
	if err != nil {
		return nil, err
	}

	return &permissionV1.ListPermissionGroupResponse{
		Total: uint64(count),
		Items: dtos,
	}, nil
}

func (r *PermissionGroupRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.repository.ExistsWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", id) },
	})
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, permissionV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *PermissionGroupRepo) Get(ctx context.Context, req *permissionV1.GetPermissionGroupRequest) (*permissionV1.PermissionGroup, error) {
	if req == nil {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	var scopes []func(*gormDB.DB) *gormDB.DB
	switch req.QueryBy.(type) {
	default:
	case *permissionV1.GetPermissionGroupRequest_Id:
		scopes = append(scopes, func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) })
	}

	dto, err := r.repository.GetWithFilters(ctx, r.client.DB, scopes, req.GetViewMask())
	if err != nil {
		if errors.Is(err, gormDB.ErrRecordNotFound) {
			return nil, permissionV1.ErrorNotFound("permission group not found")
		}
		r.log.Errorf("query permission group failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query permission group failed")
	}

	return dto, nil
}

// Truncate 清空表数据
func (r *PermissionGroupRepo) Truncate(ctx context.Context) error {
	if err := r.client.DB.WithContext(ctx).Where("1 = 1").Delete(&models.PermissionGroup{}).Error; err != nil {
		r.log.Errorf("failed to truncate permission group table: %s", err.Error())
		return permissionV1.ErrorInternalServerError("truncate failed")
	}

	return nil
}

func (r *PermissionGroupRepo) ListByIDs(ctx context.Context, ids []uint32) ([]*permissionV1.PermissionGroup, error) {
	if len(ids) == 0 {
		return []*permissionV1.PermissionGroup{}, nil
	}

	var entities []*models.PermissionGroup
	if err := r.client.DB.WithContext(ctx).Model(&models.PermissionGroup{}).Where("id IN ?", ids).Find(&entities).Error; err != nil {
		r.log.Errorf("query list by ids failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query list by ids failed")
	}

	dtos := make([]*permissionV1.PermissionGroup, 0, len(entities))
	for _, entity := range entities {
		dto := r.mapper.ToDTO(entity)
		dtos = append(dtos, dto)
	}

	return dtos, nil
}

// Create 创建 Permission
func (r *PermissionGroupRepo) Create(ctx context.Context, req *permissionV1.CreatePermissionGroupRequest) (*permissionV1.PermissionGroup, error) {
	return nil, permissionV1.ErrorInternalServerError("gorm scaffold: Create not implemented — ent setTreePath materialized-path write has no go-crud/gorm primitive; see data/permission_group_repo.go")
}

// BatchCreate 批量创建 Permission
func (r *PermissionGroupRepo) BatchCreate(ctx context.Context, permissionGroups []*permissionV1.PermissionGroup) ([]*permissionV1.PermissionGroup, error) {
	return nil, permissionV1.ErrorInternalServerError("gorm scaffold: BatchCreate not implemented — ent setTreePath materialized-path write has no go-crud/gorm primitive; see data/permission_group_repo.go")
}

// Update 更新 Permission
func (r *PermissionGroupRepo) Update(ctx context.Context, req *permissionV1.UpdatePermissionGroupRequest) error {
	return permissionV1.ErrorInternalServerError("gorm scaffold: Update not implemented — ent setTreePath materialized-path write has no go-crud/gorm primitive; see data/permission_group_repo.go")
}

// UpdateParentIDs 更新 Permission ParentID
func (r *PermissionGroupRepo) UpdateParentIDs(ctx context.Context, parentIDs map[uint32]uint32) error {
	return permissionV1.ErrorInternalServerError("gorm scaffold: UpdateParentIDs not implemented — ent setTreePath materialized-path write has no go-crud/gorm primitive; see data/permission_group_repo.go")
}

// Delete 删除 Permission
func (r *PermissionGroupRepo) Delete(ctx context.Context, req *permissionV1.DeletePermissionGroupRequest) error {
	return permissionV1.ErrorInternalServerError("gorm scaffold: Delete not implemented — ent setTreePath materialized-path write has no go-crud/gorm primitive; see data/permission_group_repo.go")
}

// TruncateBizGroup 清空业务表数据，保留系统内置数据
func (r *PermissionGroupRepo) TruncateBizGroup(ctx context.Context) error {
	return permissionV1.ErrorInternalServerError("gorm scaffold: TruncateBizGroup not implemented — ent topology mutation has no go-crud/gorm primitive; see data/permission_group_repo.go")
}
