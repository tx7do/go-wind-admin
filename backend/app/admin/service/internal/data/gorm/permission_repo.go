//go:build gorm_backend
// +build gorm_backend

package gorm

import (
	"context"
	"time"

	gormDB "gorm.io/gorm"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	gormCrud "github.com/tx7do/go-crud/gorm"
	gormCrudFilter "github.com/tx7do/go-crud/gorm/filter"
	paginationFilter "github.com/tx7do/go-crud/pagination/filter"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/trans"
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

	// permissionApiRepo / permissionMenuRepo / rolePermissionRepo 绑定子 repo：
	// 端点与菜单全路径快照反查、删除级联用（与 data 栈同构）。
	permissionApiRepo  *PermissionApiRepo
	permissionMenuRepo *PermissionMenuRepo
	rolePermissionRepo *RolePermissionRepo
}

func NewPermissionRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
	permissionApiRepo *PermissionApiRepo,
	permissionMenuRepo *PermissionMenuRepo,
	rolePermissionRepo *RolePermissionRepo,
) *PermissionRepo {
	repo := &PermissionRepo{
		log:                ctx.NewLoggerHelper("permission/gorm-repo/admin-service"),
		client:             client,
		mapper:             mapper.NewCopierMapper[permissionV1.Permission, models.Permission](),
		structuredFilter:   gormCrudFilter.NewStructuredFilter(),
		statusConverter:    mapper.NewEnumTypeConverter[permissionV1.Permission_Status, string](permissionV1.Permission_Status_name, permissionV1.Permission_Status_value),
		permissionApiRepo:  permissionApiRepo,
		permissionMenuRepo: permissionMenuRepo,
		rolePermissionRepo: rolePermissionRepo,
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
		Count: int64(count),
	}, nil
}

func (r *PermissionRepo) GetPermissionCodesByIDs(ctx context.Context, ids []int64) ([]string, error) {
	var codes []string
	if err := r.client.DB.WithContext(ctx).Model(&models.Permission{}).
		Where("id IN ?", ids).
		Pluck("code", &codes).Error; err != nil {
		r.log.Errorf(ctx, "query permission codes by ids failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query permission codes by ids failed")
	}
	return codes, nil
}

func (r *PermissionRepo) GetPermissionIDsByCodes(ctx context.Context, codes []string) ([]int64, error) {
	var ids []int64
	if err := r.client.DB.WithContext(ctx).Model(&models.Permission{}).
		Where("code IN ?", codes).
		Pluck("id", &ids).Error; err != nil {
		r.log.Errorf(ctx, "query permission ids by codes failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query permission ids by codes failed")
	}
	return ids, nil
}

func (r *PermissionRepo) IsExist(ctx context.Context, id int64) (bool, error) {
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

// ListBizPermissions 列出全部业务权限行（code 非 sys: 前缀）的最小视图。
// 与 data 栈同名方法语义一致。
func (r *PermissionRepo) ListBizPermissions(ctx context.Context) ([]*permissionV1.Permission, error) {
	var rows []*models.Permission
	if err := r.client.DB.WithContext(ctx).Model(&models.Permission{}).
		Where("code IS NOT NULL AND code <> '' AND code NOT LIKE ?", "sys:%").
		Find(&rows).Error; err != nil {
		r.log.Errorf(ctx, "list biz permissions failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("list biz permissions failed")
	}
	dtos := make([]*permissionV1.Permission, 0, len(rows))
	for _, row := range rows {
		dtos = append(dtos, &permissionV1.Permission{Id: trans.Ptr(row.ID), Code: row.Code, Name: row.Name})
	}
	return dtos, nil
}

// UpdateForSync 权限同步 MERGE 的更新段：原位更新名称/分组，不触碰 status。
// 与 data 栈同名方法语义一致。
func (r *PermissionRepo) UpdateForSync(ctx context.Context, id int64, derived *permissionV1.Permission) error {
	updates := map[string]any{"updated_at": time.Now()}
	if derived.Name != nil {
		updates["name"] = *derived.Name
	}
	if derived.GroupId != nil {
		updates["group_id"] = *derived.GroupId
	}
	if err := r.client.DB.WithContext(ctx).Model(&models.Permission{}).
		Where("id = ?", id).Updates(updates).Error; err != nil {
		r.log.Errorf(ctx, "sync update permission failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("sync update permission failed")
	}
	return nil
}

// AssignSyncBindings 权限同步 MERGE 的绑定替换段：按派生集整体替换端点/菜单
// 快照绑定。与 data 栈同名方法语义一致。
func (r *PermissionRepo) AssignSyncBindings(ctx context.Context, id int64, apiIDs, menuIDs []int64) error {
	if err := r.permissionApiRepo.AssignApis(ctx, id, apiIDs); err != nil {
		return err
	}
	return r.permissionMenuRepo.AssignMenus(ctx, id, menuIDs)
}

// DeleteByIDs 按 id 集合删除权限行并级联清理全部从属绑定。与 data 栈同名方法
// 语义一致。
func (r *PermissionRepo) DeleteByIDs(ctx context.Context, permissionIDs []int64) error {
	if len(permissionIDs) == 0 {
		return nil
	}
	if err := r.client.DB.WithContext(ctx).Where("id IN ?", permissionIDs).
		Delete(&models.Permission{}).Error; err != nil {
		r.log.Errorf(ctx, "delete permissions by ids failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete permissions by ids failed")
	}
	return r.cascadeCleanByPermissionIDs(ctx, permissionIDs)
}

// cascadeCleanByPermissionIDs 权限行删除后的级联清理：接口/菜单快照绑定、
// 角色授权绑定、策略联动行（sys_permission_policies）。与 data 栈同构。
func (r *PermissionRepo) cascadeCleanByPermissionIDs(ctx context.Context, permissionIDs []int64) error {
	if len(permissionIDs) == 0 {
		return nil
	}
	if err := r.permissionApiRepo.DeleteByPermissionIDs(ctx, permissionIDs); err != nil {
		return err
	}
	if err := r.permissionMenuRepo.DeleteByPermissionIDs(ctx, permissionIDs); err != nil {
		return err
	}
	if err := r.rolePermissionRepo.DeleteByPermissionIDs(ctx, permissionIDs); err != nil {
		return err
	}
	if err := r.client.DB.WithContext(ctx).Exec(
		"DELETE FROM sys_permission_policies WHERE permission_id IN ?", permissionIDs,
	).Error; err != nil {
		r.log.Errorf(ctx, "delete permission policies failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete permission policies failed")
	}
	return nil
}

// ListApiEndpointsByPermissionIDs 列出权限关联的端点快照（(path, method)，
// 授权执行键；绑定表即真相，不回查 sys_apis）。与 data 栈同名方法语义一致。
func (r *PermissionRepo) ListApiEndpointsByPermissionIDs(ctx context.Context, permissionIDs []int64) ([]Endpoint, error) {
	return r.permissionApiRepo.ListEndpoints(ctx, permissionIDs)
}

// ListMenuFullPathsByPermissionIDs 列出权限关联的菜单全路径快照（菜单身份键；
// 绑定表即真相，导航/回显链路自行反解回 id）。与 data 栈同名方法语义一致。
func (r *PermissionRepo) ListMenuFullPathsByPermissionIDs(ctx context.Context, permissionIDs []int64) ([]string, error) {
	return r.permissionMenuRepo.ListMenuFullPaths(ctx, permissionIDs)
}
