//go:build gorm_backend
// +build gorm_backend

package gorm

import (
	"context"

	gormDB "gorm.io/gorm"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	gormCrud "github.com/tx7do/go-crud/gorm"
	gormCrudFilter "github.com/tx7do/go-crud/gorm/filter"
	paginationFilter "github.com/tx7do/go-crud/pagination/filter"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/admin/service/internal/data/gorm/models"
	"go-wind-admin/pkg/constants"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

type PermissionRepo struct {
	client           *gormCrud.Client
	log              *bLogger.Helper
	mapper           *mapper.CopierMapper[permissionV1.Permission, models.Permission]
	repository       *gormCrud.Repository[permissionV1.Permission, models.Permission]
	structuredFilter *gormCrudFilter.StructuredFilter
	statusConverter  *mapper.EnumTypeConverter[permissionV1.Permission_Status, string]
}

func NewPermissionRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *PermissionRepo {
	repo := &PermissionRepo{
		log:              ctx.NewLoggerHelper("permission/gorm-repo/admin-service"),
		client:           client,
		mapper:           mapper.NewCopierMapper[permissionV1.Permission, models.Permission](),
		structuredFilter: gormCrudFilter.NewStructuredFilter(),
		statusConverter:  mapper.NewEnumTypeConverter[permissionV1.Permission_Status, string](permissionV1.Permission_Status_name, permissionV1.Permission_Status_value),
	}

	repo.init()

	return repo
}

func (r *PermissionRepo) init() {
	r.repository = gormCrud.NewRepository[permissionV1.Permission, models.Permission](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
}

func (r *PermissionRepo) Count(ctx context.Context, req *paginationV1.PagingRequest) (*permissionV1.CountPermissionResponse, error) {
	filterExpr, err := paginationFilter.ConvertFilterByPagingRequest(req)
	if err != nil {
		r.log.Errorf(ctx, "parse count param error [%s]", err.Error())
		return nil, permissionV1.ErrorBadRequest("invalid query parameter")
	}

	scopes, err := r.structuredFilter.BuildSelectors(filterExpr)
	if err != nil {
		r.log.Errorf(ctx, "parse count param error [%s]", err.Error())
		return nil, permissionV1.ErrorBadRequest("invalid query parameter")
	}

	count, err := r.repository.Count(ctx, r.client.DB, scopes)
	if err != nil {
		r.log.Errorf(ctx, "query permission count failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query count failed")
	}

	return &permissionV1.CountPermissionResponse{
		Count: uint64(count),
	}, nil
}

func (r *PermissionRepo) GetPermissionCodesByIDs(ctx context.Context, ids []uint32) ([]string, error) {
	var codes []string
	if err := r.client.DB.WithContext(ctx).Model(&models.Permission{}).
		Where("id IN ?", ids).
		Pluck("code", &codes).Error; err != nil {
		r.log.Errorf(ctx, "query permission codes by ids failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query permission codes by ids failed")
	}
	return codes, nil
}

func (r *PermissionRepo) GetPermissionIDsByCodes(ctx context.Context, codes []string) ([]uint32, error) {
	var ids []uint32
	if err := r.client.DB.WithContext(ctx).Model(&models.Permission{}).
		Where("code IN ?", codes).
		Pluck("id", &ids).Error; err != nil {
		r.log.Errorf(ctx, "query permission ids by codes failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query permission ids by codes failed")
	}
	return ids, nil
}

func (r *PermissionRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.repository.ExistsWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", id) },
	})
	if err != nil {
		r.log.Errorf(ctx, "query exist failed: %s", err.Error())
		return false, permissionV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *PermissionRepo) CleanPermissionsByCodes(ctx context.Context, codes []string) error {
	if err := r.client.DB.WithContext(ctx).Where("code IN ?", codes).
		Delete(&models.Permission{}).Error; err != nil {
		r.log.Errorf(ctx, "delete permissions by codes failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete permissions by codes failed")
	}
	return nil
}

func (r *PermissionRepo) CleanDataPermissions(ctx context.Context) error {
	return nil
}

func (r *PermissionRepo) TruncateBizPermissions(ctx context.Context) error {
	if err := r.client.DB.WithContext(ctx).
		Where("code NOT LIKE ?", constants.SystemPermissionCodePrefix+"%").
		Delete(&models.Permission{}).Error; err != nil {
		r.log.Errorf(ctx, "truncate biz permissions failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("truncate biz permissions failed")
	}
	return nil
}

// === 以下方法为 gorm scaffold 桩：依赖 ent 跨仓储事务/委托/edge，go-crud/gorm 无对应原语 ===

func (r *PermissionRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*permissionV1.ListPermissionResponse, error) {
	return nil, permissionV1.ErrorInternalServerError("gorm scaffold: List not implemented — ent cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/permission_repo.go")
}

func (r *PermissionRepo) Get(ctx context.Context, req *permissionV1.GetPermissionRequest) (*permissionV1.Permission, error) {
	return nil, permissionV1.ErrorInternalServerError("gorm scaffold: Get not implemented — ent cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/permission_repo.go")
}

func (r *PermissionRepo) Create(ctx context.Context, req *permissionV1.CreatePermissionRequest) error {
	return permissionV1.ErrorInternalServerError("gorm scaffold: Create not implemented — ent cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/permission_repo.go")
}

func (r *PermissionRepo) BatchCreate(ctx context.Context, permissions []*permissionV1.Permission) error {
	return permissionV1.ErrorInternalServerError("gorm scaffold: BatchCreate not implemented — ent cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/permission_repo.go")
}

func (r *PermissionRepo) Update(ctx context.Context, req *permissionV1.UpdatePermissionRequest) error {
	return permissionV1.ErrorInternalServerError("gorm scaffold: Update not implemented — ent cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/permission_repo.go")
}

func (r *PermissionRepo) Delete(ctx context.Context, req *permissionV1.DeletePermissionRequest) error {
	return permissionV1.ErrorInternalServerError("gorm scaffold: Delete not implemented — ent cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/permission_repo.go")
}

func (r *PermissionRepo) Truncate(ctx context.Context) error {
	return permissionV1.ErrorInternalServerError("gorm scaffold: Truncate not implemented — ent cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/permission_repo.go")
}

func (r *PermissionRepo) CleanApiPermissions(ctx context.Context) error {
	return permissionV1.ErrorInternalServerError("gorm scaffold: CleanApiPermissions not implemented — ent cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/permission_repo.go")
}

func (r *PermissionRepo) CleanMenuPermissions(ctx context.Context) error {
	return permissionV1.ErrorInternalServerError("gorm scaffold: CleanMenuPermissions not implemented — ent cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/permission_repo.go")
}

func (r *PermissionRepo) ListApiIDsByPermissionIDs(ctx context.Context, permissionIDs []uint32) ([]uint32, error) {
	return nil, permissionV1.ErrorInternalServerError("gorm scaffold: ListApiIDsByPermissionIDs not implemented — ent cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/permission_repo.go")
}

func (r *PermissionRepo) ListMenuIDsByPermissionIDs(ctx context.Context, permissionIDs []uint32) ([]uint32, error) {
	return nil, permissionV1.ErrorInternalServerError("gorm scaffold: ListMenuIDsByPermissionIDs not implemented — ent cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/permission_repo.go")
}
