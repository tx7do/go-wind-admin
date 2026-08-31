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

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

type RoleRepo struct {
	client           *gormCrud.Client
	log              *bLogger.Helper
	mapper           *mapper.CopierMapper[permissionV1.Role, models.Role]
	repository       *gormCrud.Repository[permissionV1.Role, models.Role]
	structuredFilter *gormCrudFilter.StructuredFilter
	statusConverter  *mapper.EnumTypeConverter[permissionV1.Role_Status, string]
	typeConverter    *mapper.EnumTypeConverter[permissionV1.Role_Type, string]
}

func NewRoleRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *RoleRepo {
	repo := &RoleRepo{
		log:              ctx.NewLoggerHelper("role/gorm-repo/admin-service"),
		client:           client,
		mapper:           mapper.NewCopierMapper[permissionV1.Role, models.Role](),
		structuredFilter: gormCrudFilter.NewStructuredFilter(),
		statusConverter:  mapper.NewEnumTypeConverter[permissionV1.Role_Status, string](permissionV1.Role_Status_name, permissionV1.Role_Status_value),
		typeConverter:    mapper.NewEnumTypeConverter[permissionV1.Role_Type, string](permissionV1.Role_Type_name, permissionV1.Role_Type_value),
	}

	repo.init()

	return repo
}

func (r *RoleRepo) init() {
	r.repository = gormCrud.NewRepository[permissionV1.Role, models.Role](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
	r.mapper.AppendConverters(r.typeConverter.NewConverterPair())
}

func (r *RoleRepo) Count(ctx context.Context, req *paginationV1.PagingRequest) (int, error) {
	filterExpr, err := paginationFilter.ConvertFilterByPagingRequest(req)
	if err != nil {
		r.log.Errorf(ctx, "parse count param error [%s]", err.Error())
		return 0, permissionV1.ErrorBadRequest("invalid query parameter")
	}

	scopes, err := r.structuredFilter.BuildSelectors(filterExpr)
	if err != nil {
		r.log.Errorf(ctx, "parse count param error [%s]", err.Error())
		return 0, permissionV1.ErrorBadRequest("invalid query parameter")
	}

	count, err := r.repository.Count(ctx, r.client.DB, scopes)
	if err != nil {
		r.log.Errorf(ctx, "query role count failed: %s", err.Error())
		return 0, permissionV1.ErrorInternalServerError("query count failed")
	}

	return int(count), nil
}

func (r *RoleRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.repository.ExistsWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", id) },
	})
	if err != nil {
		r.log.Errorf(ctx, "query exist failed: %s", err.Error())
		return false, permissionV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *RoleRepo) ListRoleCodesByRoleIds(ctx context.Context, ids []uint32) ([]string, error) {
	if len(ids) == 0 {
		return []string{}, nil
	}

	var entities []*models.Role
	if err := r.client.DB.WithContext(ctx).Model(&models.Role{}).Where("id IN ?", ids).Find(&entities).Error; err != nil {
		r.log.Errorf(ctx, "query role codes failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query role codes failed")
	}

	codes := make([]string, 0, len(entities))
	for _, entity := range entities {
		if entity.Code != nil {
			codes = append(codes, *entity.Code)
		}
	}

	return codes, nil
}

func (r *RoleRepo) ListRoleIDsByRoleCodes(ctx context.Context, codes []string) ([]uint32, error) {
	if len(codes) == 0 {
		return []uint32{}, nil
	}

	var entities []*models.Role
	if err := r.client.DB.WithContext(ctx).Model(&models.Role{}).Where("code IN ?", codes).Find(&entities).Error; err != nil {
		r.log.Errorf(ctx, "query role ids failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query role ids failed")
	}

	ids := make([]uint32, 0, len(entities))
	for _, entity := range entities {
		ids = append(ids, entity.ID)
	}

	return ids, nil
}

// === 以下方法为 gorm scaffold 桩：依赖 ent 跨仓储事务/委托/edge，go-crud/gorm 无对应原语 ===

func (r *RoleRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*permissionV1.ListRoleResponse, error) {
	return nil, permissionV1.ErrorInternalServerError("gorm scaffold: List not implemented — ent fillPermissionIDs cross-repo delegation has no go-crud/gorm primitive; see data/role_repo.go")
}

func (r *RoleRepo) ListRolesByRoleCodes(ctx context.Context, codes []string) ([]*permissionV1.Role, error) {
	return nil, permissionV1.ErrorInternalServerError("gorm scaffold: ListRolesByRoleCodes not implemented — ent fillPermissionIDs cross-repo delegation has no go-crud/gorm primitive; see data/role_repo.go")
}

func (r *RoleRepo) ListRolesByRoleIds(ctx context.Context, ids []uint32) ([]*permissionV1.Role, error) {
	return nil, permissionV1.ErrorInternalServerError("gorm scaffold: ListRolesByRoleIds not implemented — ent fillPermissionIDs cross-repo delegation has no go-crud/gorm primitive; see data/role_repo.go")
}

func (r *RoleRepo) Get(ctx context.Context, req *permissionV1.GetRoleRequest) (*permissionV1.Role, error) {
	return nil, permissionV1.ErrorInternalServerError("gorm scaffold: Get not implemented — ent fillPermissionIDs cross-repo delegation has no go-crud/gorm primitive; see data/role_repo.go")
}

func (r *RoleRepo) GetTemplateRole(ctx context.Context, templateCode string) (*permissionV1.Role, error) {
	return nil, permissionV1.ErrorInternalServerError("gorm scaffold: GetTemplateRole not implemented — ent fillPermissionIDs cross-repo delegation has no go-crud/gorm primitive; see data/role_repo.go")
}

func (r *RoleRepo) CreateTenantRoleFromTemplate(ctx context.Context, tx any, tenantID uint32, operatorID uint32) error {
	return permissionV1.ErrorInternalServerError("gorm scaffold: CreateTenantRoleFromTemplate not implemented — ent *ent.Tx cross-repo transaction has no go-crud/gorm primitive; see data/role_repo.go")
}

func (r *RoleRepo) Create(ctx context.Context, req *permissionV1.CreateRoleRequest) error {
	return permissionV1.ErrorInternalServerError("gorm scaffold: Create not implemented — ent *ent.Tx cross-repo transaction has no go-crud/gorm primitive; see data/role_repo.go")
}

func (r *RoleRepo) CreateWithTx(ctx context.Context, tx any, data *permissionV1.Role) (*permissionV1.Role, error) {
	return nil, permissionV1.ErrorInternalServerError("gorm scaffold: CreateWithTx not implemented — ent *ent.Tx cross-repo transaction has no go-crud/gorm primitive; see data/role_repo.go")
}

func (r *RoleRepo) Update(ctx context.Context, req *permissionV1.UpdateRoleRequest) error {
	return permissionV1.ErrorInternalServerError("gorm scaffold: Update not implemented — ent *ent.Tx cross-repo transaction has no go-crud/gorm primitive; see data/role_repo.go")
}

func (r *RoleRepo) Delete(ctx context.Context, req *permissionV1.DeleteRoleRequest) error {
	return permissionV1.ErrorInternalServerError("gorm scaffold: Delete not implemented — ent *ent.Tx cross-repo transaction has no go-crud/gorm primitive; see data/role_repo.go")
}

func (r *RoleRepo) ListPermissionIDsByRoleIDs(ctx context.Context, roleIDs []uint32) ([]uint32, error) {
	return nil, permissionV1.ErrorInternalServerError("gorm scaffold: ListPermissionIDsByRoleIDs not implemented — ent cross-repo delegation has no go-crud/gorm primitive; see data/role_repo.go")
}

func (r *RoleRepo) ListPermissionIDsByRoleCodes(ctx context.Context, codes []string) ([]uint32, error) {
	return nil, permissionV1.ErrorInternalServerError("gorm scaffold: ListPermissionIDsByRoleCodes not implemented — ent cross-repo delegation has no go-crud/gorm primitive; see data/role_repo.go")
}

func (r *RoleRepo) GetRolePermissionApiIDs(ctx context.Context, roleID uint32) ([]uint32, error) {
	return nil, permissionV1.ErrorInternalServerError("gorm scaffold: GetRolePermissionApiIDs not implemented — ent cross-repo delegation has no go-crud/gorm primitive; see data/role_repo.go")
}

func (r *RoleRepo) GetRolePermissionMenuIDs(ctx context.Context, roleID uint32) ([]uint32, error) {
	return nil, permissionV1.ErrorInternalServerError("gorm scaffold: GetRolePermissionMenuIDs not implemented — ent cross-repo delegation has no go-crud/gorm primitive; see data/role_repo.go")
}

func (r *RoleRepo) GetRolesPermissionMenuIDs(ctx context.Context, roleIDs []uint32) ([]uint32, error) {
	return nil, permissionV1.ErrorInternalServerError("gorm scaffold: GetRolesPermissionMenuIDs not implemented — ent cross-repo delegation has no go-crud/gorm primitive; see data/role_repo.go")
}

func (r *RoleRepo) CanAssignRole(ctx context.Context, roleID uint32) (bool, error) {
	return false, permissionV1.ErrorInternalServerError("gorm scaffold: CanAssignRole not implemented — ent cross-repo delegation has no go-crud/gorm primitive; see data/role_repo.go")
}
