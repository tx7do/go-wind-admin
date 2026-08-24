//go:build gorm_backend
// +build gorm_backend

package gorm

import (
	"context"

	gormDB "gorm.io/gorm"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	gormCrud "github.com/tx7do/go-crud/gorm"
	gormCrudFilter "github.com/tx7do/go-crud/gorm/filter"
	paginationFilter "github.com/tx7do/go-crud/pagination/filter"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/admin/service/internal/data/gorm/models"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
)

type UserRepo struct {
	client           *gormCrud.Client
	log              *log.Helper
	mapper           *mapper.CopierMapper[identityV1.User, models.User]
	repository       *gormCrud.Repository[identityV1.User, models.User]
	structuredFilter *gormCrudFilter.StructuredFilter
	genderConverter  *mapper.EnumTypeConverter[identityV1.User_Gender, string]
	statusConverter  *mapper.EnumTypeConverter[identityV1.User_Status, string]
}

func NewUserRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *UserRepo {
	repo := &UserRepo{
		log:              ctx.NewLoggerHelper("user/gorm-repo/admin-service"),
		client:           client,
		mapper:           mapper.NewCopierMapper[identityV1.User, models.User](),
		structuredFilter: gormCrudFilter.NewStructuredFilter(),
		genderConverter:  mapper.NewEnumTypeConverter[identityV1.User_Gender, string](identityV1.User_Gender_name, identityV1.User_Gender_value),
		statusConverter:  mapper.NewEnumTypeConverter[identityV1.User_Status, string](identityV1.User_Status_name, identityV1.User_Status_value),
	}

	repo.init()

	return repo
}

func (r *UserRepo) init() {
	r.repository = gormCrud.NewRepository[identityV1.User, models.User](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.genderConverter.NewConverterPair())
	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
}

func (r *UserRepo) Count(ctx context.Context, req *paginationV1.PagingRequest) (int, error) {
	filterExpr, err := paginationFilter.ConvertFilterByPagingRequest(req)
	if err != nil {
		r.log.Errorf("parse count param error [%s]", err.Error())
		return 0, identityV1.ErrorBadRequest("invalid query parameter")
	}

	scopes, err := r.structuredFilter.BuildSelectors(filterExpr)
	if err != nil {
		r.log.Errorf("parse count param error [%s]", err.Error())
		return 0, identityV1.ErrorBadRequest("invalid query parameter")
	}

	count, err := r.repository.Count(ctx, r.client.DB, scopes)
	if err != nil {
		r.log.Errorf("query user count failed: %s", err.Error())
		return 0, identityV1.ErrorInternalServerError("query count failed")
	}

	return int(count), nil
}

func (r *UserRepo) UserExists(ctx context.Context, req *identityV1.UserExistsRequest) (*identityV1.UserExistsResponse, error) {
	var scopes []func(*gormDB.DB) *gormDB.DB
	switch req.QueryBy.(type) {
	case *identityV1.UserExistsRequest_Id:
		scopes = append(scopes, func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) })
	case *identityV1.UserExistsRequest_Username:
		scopes = append(scopes, func(db *gormDB.DB) *gormDB.DB { return db.Where("username = ?", req.GetUsername()) })
	default:
		return &identityV1.UserExistsResponse{Exist: false}, identityV1.ErrorBadRequest("invalid query by type")
	}

	exist, err := r.repository.ExistsWithFilters(ctx, r.client.DB, scopes)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return &identityV1.UserExistsResponse{Exist: false}, identityV1.ErrorInternalServerError("query exist failed")
	}

	return &identityV1.UserExistsResponse{
		Exist: exist,
	}, nil
}

func (r *UserRepo) ListUsersByIds(ctx context.Context, ids []uint32) ([]*identityV1.User, error) {
	if len(ids) == 0 {
		return []*identityV1.User{}, nil
	}

	var entities []*models.User
	if err := r.client.DB.WithContext(ctx).Model(&models.User{}).Where("id IN ?", ids).Find(&entities).Error; err != nil {
		r.log.Errorf("query user by ids failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query user by ids failed")
	}

	dtos := make([]*identityV1.User, 0, len(entities))
	for _, entity := range entities {
		dtos = append(dtos, r.mapper.ToDTO(entity))
	}

	return dtos, nil
}

func (r *UserRepo) CountByTenantIDs(ctx context.Context, tenantIDs []uint32) (map[uint32]int, error) {
	result := make(map[uint32]int, len(tenantIDs))
	if len(tenantIDs) == 0 {
		return result, nil
	}
	for _, tid := range tenantIDs {
		count, err := r.repository.Count(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
			func(db *gormDB.DB) *gormDB.DB { return db.Where("tenant_id = ?", tid) },
		})
		if err != nil {
			r.log.Errorf("count users by tenant %d failed: %s", tid, err.Error())
			return nil, identityV1.ErrorInternalServerError("count users by tenant failed")
		}
		result[tid] = int(count)
	}
	return result, nil
}

// === 以下方法为 gorm scaffold 桩：依赖 ent 跨仓储事务/委托/edge，go-crud/gorm 无对应原语 ===

func (r *UserRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*identityV1.ListUserResponse, error) {
	return nil, identityV1.ErrorInternalServerError("gorm scaffold: List not implemented — ent cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/user_repo.go")
}

func (r *UserRepo) Get(ctx context.Context, req *identityV1.GetUserRequest) (*identityV1.User, error) {
	return nil, identityV1.ErrorInternalServerError("gorm scaffold: Get not implemented — ent ListUserRelationIDs cross-repo delegation has no go-crud/gorm primitive; see data/user_repo.go")
}

func (r *UserRepo) Create(ctx context.Context, req *identityV1.CreateUserRequest) (*identityV1.User, error) {
	return nil, identityV1.ErrorInternalServerError("gorm scaffold: Create not implemented — ent *ent.Tx cross-repo transaction has no go-crud/gorm primitive; see data/user_repo.go")
}

func (r *UserRepo) CreateWithTx(ctx context.Context, tx any, data *identityV1.User) (*identityV1.User, error) {
	return nil, identityV1.ErrorInternalServerError("gorm scaffold: CreateWithTx not implemented — ent *ent.Tx cross-repo transaction has no go-crud/gorm primitive; see data/user_repo.go")
}

func (r *UserRepo) Update(ctx context.Context, req *identityV1.UpdateUserRequest) error {
	return identityV1.ErrorInternalServerError("gorm scaffold: Update not implemented — ent *ent.Tx cross-repo transaction has no go-crud/gorm primitive; see data/user_repo.go")
}

func (r *UserRepo) Delete(ctx context.Context, req *identityV1.DeleteUserRequest) error {
	return identityV1.ErrorInternalServerError("gorm scaffold: Delete not implemented — ent *ent.Tx cross-repo transaction has no go-crud/gorm primitive; see data/user_repo.go")
}

func (r *UserRepo) AssignUserRole(ctx context.Context, userID uint32, roleID uint32) error {
	return identityV1.ErrorInternalServerError("gorm scaffold: AssignUserRole not implemented — ent *ent.Tx cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/user_repo.go")
}

func (r *UserRepo) AssignUserRoles(ctx context.Context, userID uint32, roleIDs []uint32) error {
	return identityV1.ErrorInternalServerError("gorm scaffold: AssignUserRoles not implemented — ent *ent.Tx cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/user_repo.go")
}

func (r *UserRepo) AssignUserOrgUnit(ctx context.Context, userID uint32, orgUnitID uint32) error {
	return identityV1.ErrorInternalServerError("gorm scaffold: AssignUserOrgUnit not implemented — ent *ent.Tx cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/user_repo.go")
}

func (r *UserRepo) AssignUserOrgUnits(ctx context.Context, userID uint32, orgUnitIDs []uint32) error {
	return identityV1.ErrorInternalServerError("gorm scaffold: AssignUserOrgUnits not implemented — ent *ent.Tx cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/user_repo.go")
}

func (r *UserRepo) AssignUserPosition(ctx context.Context, userID uint32, positionID uint32) error {
	return identityV1.ErrorInternalServerError("gorm scaffold: AssignUserPosition not implemented — ent *ent.Tx cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/user_repo.go")
}

func (r *UserRepo) AssignUserPositions(ctx context.Context, userID uint32, positionIDs []uint32) error {
	return identityV1.ErrorInternalServerError("gorm scaffold: AssignUserPositions not implemented — ent *ent.Tx cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/user_repo.go")
}

func (r *UserRepo) ListRoleIDsByUserID(ctx context.Context, userID uint32) ([]uint32, error) {
	return nil, identityV1.ErrorInternalServerError("gorm scaffold: ListRoleIDsByUserID not implemented — ent cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/user_repo.go")
}

func (r *UserRepo) ListPositionIDsByUserID(ctx context.Context, userID uint32) ([]uint32, error) {
	return nil, identityV1.ErrorInternalServerError("gorm scaffold: ListPositionIDsByUserID not implemented — ent cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/user_repo.go")
}

func (r *UserRepo) ListOrgUnitIDsByUserID(ctx context.Context, userID uint32) ([]uint32, error) {
	return nil, identityV1.ErrorInternalServerError("gorm scaffold: ListOrgUnitIDsByUserID not implemented — ent cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/user_repo.go")
}

func (r *UserRepo) ListUserRelationIDs(ctx context.Context, userID uint32) (roleIDs []uint32, positionIDs []uint32, orgUnitIDs []uint32, err error) {
	return nil, nil, nil, identityV1.ErrorInternalServerError("gorm scaffold: ListUserRelationIDs not implemented — ent cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/user_repo.go")
}
