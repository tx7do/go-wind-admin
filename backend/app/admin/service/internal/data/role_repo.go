package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	pagination "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/timeutil"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"
	"go-wind-admin/app/admin/service/internal/data/ent/role"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
	userV1 "go-wind-admin/api/gen/go/user/service/v1"
)

type RoleRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper          *mapper.CopierMapper[userV1.Role, ent.Role]
	statusConverter *mapper.EnumTypeConverter[userV1.Role_Status, role.Status]

	repository *entCrud.Repository[
		ent.RoleQuery, ent.RoleSelect,
		ent.RoleCreate, ent.RoleCreateBulk,
		ent.RoleUpdate, ent.RoleUpdateOne,
		ent.RoleDelete,
		predicate.Role,
		userV1.Role, ent.Role,
	]

	rolePermissionRepo *RolePermissionRepo
	permissionRepo     *PermissionRepo
}

func NewRoleRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
	rolePermissionRepo *RolePermissionRepo,
	permissionRepo *PermissionRepo,
) *RoleRepo {
	repo := &RoleRepo{
		log:       ctx.NewLoggerHelper("role/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[userV1.Role, ent.Role](),
		statusConverter: mapper.NewEnumTypeConverter[userV1.Role_Status, role.Status](
			userV1.Role_Status_name,
			userV1.Role_Status_value,
		),
		permissionRepo:     permissionRepo,
		rolePermissionRepo: rolePermissionRepo,
	}

	repo.init()

	return repo
}

func (r *RoleRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.RoleQuery, ent.RoleSelect,
		ent.RoleCreate, ent.RoleCreateBulk,
		ent.RoleUpdate, ent.RoleUpdateOne,
		ent.RoleDelete,
		predicate.Role,
		userV1.Role, ent.Role,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
}

func (r *RoleRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().Role.Query()
	if len(whereCond) != 0 {
		builder.Modify(whereCond...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, userV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *RoleRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().Role.Query().
		Where(role.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, userV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *RoleRepo) List(ctx context.Context, req *pagination.PagingRequest) (*userV1.ListRoleResponse, error) {
	if req == nil {
		return nil, userV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Role.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &userV1.ListRoleResponse{Total: 0, Items: nil}, nil
	}

	for _, item := range ret.Items {
		_ = r.fillPermissionIDs(ctx, item)
	}

	return &userV1.ListRoleResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *RoleRepo) fillPermissionIDs(ctx context.Context, dto *userV1.Role) error {
	permissionIDs, err := r.rolePermissionRepo.ListPermissionIDs(ctx, dto.GetId())
	if err != nil {
		r.log.Errorf("list permission ids failed: %s", err.Error())
		return err
	}
	dto.Permissions = permissionIDs
	return nil
}

// ListRolesByRoleCodes 通过角色编码列表获取角色列表
func (r *RoleRepo) ListRolesByRoleCodes(ctx context.Context, codes []string) ([]*userV1.Role, error) {
	if len(codes) == 0 {
		return []*userV1.Role{}, nil
	}

	entities, err := r.entClient.Client().Role.Query().
		Where(role.CodeIn(codes...)).
		All(ctx)
	if err != nil {
		r.log.Errorf("query roles by codes failed: %s", err.Error())
		return nil, userV1.ErrorInternalServerError("query roles by codes failed")
	}

	dtos := make([]*userV1.Role, 0, len(entities))
	for _, entity := range entities {
		dto := r.mapper.ToDTO(entity)
		dtos = append(dtos, dto)
	}

	for _, item := range dtos {
		_ = r.fillPermissionIDs(ctx, item)
	}

	return dtos, nil
}

// ListRolesByRoleIds 通过角色ID列表获取角色列表
func (r *RoleRepo) ListRolesByRoleIds(ctx context.Context, ids []uint32) ([]*userV1.Role, error) {
	if len(ids) == 0 {
		return []*userV1.Role{}, nil
	}

	entities, err := r.entClient.Client().Role.Query().
		Where(role.IDIn(ids...)).
		All(ctx)
	if err != nil {
		r.log.Errorf("query roles by ids failed: %s", err.Error())
		return nil, userV1.ErrorInternalServerError("query roles by ids failed")
	}

	dtos := make([]*userV1.Role, 0, len(entities))
	for _, entity := range entities {
		dto := r.mapper.ToDTO(entity)
		dtos = append(dtos, dto)
	}

	for _, item := range dtos {
		_ = r.fillPermissionIDs(ctx, item)
	}

	return dtos, nil
}

// ListRoleCodesByRoleIds 通过角色ID列表获取角色编码列表
func (r *RoleRepo) ListRoleCodesByRoleIds(ctx context.Context, ids []uint32) ([]string, error) {
	if len(ids) == 0 {
		return []string{}, nil
	}

	entities, err := r.entClient.Client().Role.Query().
		Where(role.IDIn(ids...)).
		All(ctx)
	if err != nil {
		r.log.Errorf("query role codes failed: %s", err.Error())
		return nil, userV1.ErrorInternalServerError("query role codes failed")
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

	entities, err := r.entClient.Client().Role.Query().
		Where(role.CodeIn(codes...)).
		Select(role.FieldID).
		All(ctx)
	if err != nil {
		r.log.Errorf("query role ids failed: %s", err.Error())
		return nil, userV1.ErrorInternalServerError("query role ids failed")
	}

	ids := make([]uint32, 0, len(entities))
	for _, entity := range entities {
		ids = append(ids, entity.ID)
	}

	return ids, nil
}

// Get 获取角色信息
func (r *RoleRepo) Get(ctx context.Context, req *userV1.GetRoleRequest) (*userV1.Role, error) {
	if req == nil {
		return nil, userV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Role.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *userV1.GetRoleRequest_Id:
		whereCond = append(whereCond, role.IDEQ(req.GetId()))
	case *userV1.GetRoleRequest_Name:
		whereCond = append(whereCond, role.NameEQ(req.GetName()))
	case *userV1.GetRoleRequest_Code:
		whereCond = append(whereCond, role.CodeEQ(req.GetCode()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	_ = r.fillPermissionIDs(ctx, dto)

	return dto, err
}

// Create 创建角色
func (r *RoleRepo) Create(ctx context.Context, req *userV1.CreateRoleRequest) (err error) {
	if req == nil || req.Data == nil {
		return userV1.ErrorBadRequest("invalid parameter")
	}

	var tx *ent.Tx
	tx, err = r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf("start transaction failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("start transaction failed")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.log.Errorf("transaction rollback failed: %s", rollbackErr.Error())
			}
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			r.log.Errorf("transaction commit failed: %s", commitErr.Error())
			err = permissionV1.ErrorInternalServerError("transaction commit failed")
		}
	}()

	builder := tx.Role.Create().
		SetNillableName(req.Data.Name).
		SetNillableCode(req.Data.Code).
		SetNillableTenantID(req.Data.TenantId).
		SetNillableSortOrder(req.Data.SortOrder).
		SetNillableIsProtected(req.Data.IsProtected).
		SetNillableStatus(r.statusConverter.ToEntity(req.Data.Status)).
		SetNillableDescription(req.Data.Description).
		SetNillableCreatedBy(req.Data.CreatedBy).
		SetNillableCreatedAt(timeutil.TimestamppbToTime(req.Data.CreatedAt))

	if req.Data.TenantId == nil {
		builder.SetTenantID(req.Data.GetTenantId())
	}
	if req.Data.CreatedAt == nil {
		builder.SetCreatedAt(time.Now())
	}

	if req.Data.Id != nil {
		builder.SetID(req.GetData().GetId())
	}

	var ret *ent.Role
	if ret, err = builder.Save(ctx); err != nil {
		r.log.Errorf("insert one data failed: %s", err.Error())
		return userV1.ErrorInternalServerError("insert data failed")
	}

	if req.Data.Permissions != nil {
		if err = r.assignPermissionsToRole(ctx, tx, req.Data.GetTenantId(), ret.ID, req.Data.GetCreatedBy(), req.Data.Permissions); err != nil {
			r.log.Errorf("assign permissions to role failed: %s", err.Error())
			return userV1.ErrorInternalServerError("assign permissions to role failed")
		}
	}

	return nil
}

// Update 更新角色信息
func (r *RoleRepo) Update(ctx context.Context, req *userV1.UpdateRoleRequest) (err error) {
	if req == nil || req.Data == nil {
		return userV1.ErrorBadRequest("invalid parameter")
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return err
		}
		if !exist {
			createReq := &userV1.CreateRoleRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	var tx *ent.Tx
	tx, err = r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf("start transaction failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("start transaction failed")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.log.Errorf("transaction rollback failed: %s", rollbackErr.Error())
			}
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			r.log.Errorf("transaction commit failed: %s", commitErr.Error())
			err = permissionV1.ErrorInternalServerError("transaction commit failed")
		}
	}()

	builder := tx.Role.Update()
	err = r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *userV1.Role) {
			builder.
				SetNillableName(req.Data.Name).
				SetNillableCode(req.Data.Code).
				SetNillableSortOrder(req.Data.SortOrder).
				SetNillableIsProtected(req.Data.IsProtected).
				SetNillableStatus(r.statusConverter.ToEntity(req.Data.Status)).
				SetNillableDescription(req.Data.Description).
				SetNillableUpdatedBy(req.Data.UpdatedBy).
				SetNillableUpdatedAt(timeutil.TimestamppbToTime(req.Data.UpdatedAt))

			if req.Data.UpdatedAt == nil {
				builder.SetUpdatedAt(time.Now())
			}

		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(role.FieldID, req.GetId()))
		},
	)
	if err != nil {
		r.log.Errorf("update role failed: %s", err.Error())
		return userV1.ErrorInternalServerError("update role failed")
	}

	if req.Data.Permissions != nil {
		if err = r.assignPermissionsToRole(ctx, tx, req.Data.GetTenantId(), req.GetId(), req.Data.GetUpdatedBy(), req.Data.Permissions); err != nil {
			r.log.Errorf("assign permissions to role failed: %s", err.Error())
			return userV1.ErrorInternalServerError("assign permissions to role failed")
		}
	}

	return nil
}

// Delete 删除角色
func (r *RoleRepo) Delete(ctx context.Context, req *userV1.DeleteRoleRequest) (err error) {
	if req == nil {
		return userV1.ErrorBadRequest("invalid parameter")
	}

	var tx *ent.Tx
	tx, err = r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf("start transaction failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("start transaction failed")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.log.Errorf("transaction rollback failed: %s", rollbackErr.Error())
			}
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			r.log.Errorf("transaction commit failed: %s", commitErr.Error())
			err = permissionV1.ErrorInternalServerError("transaction commit failed")
		}
	}()

	if _, err = tx.Role.Delete().
		Where(role.IDEQ(req.GetId())).
		Exec(ctx); err != nil {
		r.log.Errorf("delete role failed: %s", err.Error())
		return userV1.ErrorInternalServerError("delete role failed")
	}

	if err = r.rolePermissionRepo.CleanPermissions(ctx, tx, req.GetId()); err != nil {
		return err
	}

	return nil
}

// GetPermissionsByRoleIDs 通过角色ID列表获取权限ID列表
func (r *RoleRepo) GetPermissionsByRoleIDs(ctx context.Context, roleIDs []uint32) ([]uint32, error) {
	return r.rolePermissionRepo.GetPermissionsByRoleIDs(ctx, roleIDs)
}

// assignPermissionCodesToRole 分配权限编码给角色
func (r *RoleRepo) assignPermissionCodesToRole(ctx context.Context, tx *ent.Tx, tenantID, roleID, operatorID uint32, codes []string) error {
	ids, err := r.permissionRepo.GetPermissionIDsByCodesWithTx(ctx, tx, tenantID, codes)
	if err != nil {
		return err
	}

	return r.rolePermissionRepo.AssignPermissions(ctx, tx, tenantID, roleID, operatorID, ids)
}

// assignPermissionsToRole 分配权限给角色
func (r *RoleRepo) assignPermissionsToRole(ctx context.Context, tx *ent.Tx, tenantID, roleID, operatorID uint32, permissionIDs []uint32) error {
	return r.rolePermissionRepo.AssignPermissions(ctx, tx, tenantID, roleID, operatorID, permissionIDs)
}

// GetRolePermissionApiIDs 获取角色关联的权限API资源ID列表
func (r *RoleRepo) GetRolePermissionApiIDs(ctx context.Context, roleID uint32) ([]uint32, error) {
	permissionIDs, err := r.rolePermissionRepo.ListPermissionIDs(ctx, roleID)
	if err != nil {
		return nil, err
	}

	apiIDs, err := r.permissionRepo.ListApiIDsByPermissionIDs(ctx, permissionIDs)
	if err != nil {
		return nil, err
	}

	return apiIDs, nil
}

// GetRolePermissionMenuIDs 获取角色关联的权限菜单ID列表
func (r *RoleRepo) GetRolePermissionMenuIDs(ctx context.Context, roleID uint32) ([]uint32, error) {
	permissionIDs, err := r.rolePermissionRepo.ListPermissionIDs(ctx, roleID)
	if err != nil {
		return nil, err
	}

	menuIDs, err := r.permissionRepo.ListMenuIDsByPermissionIDs(ctx, permissionIDs)
	if err != nil {
		return nil, err
	}

	return menuIDs, nil
}

func (r *RoleRepo) GetRolesPermissionMenuIDs(ctx context.Context, roleIDs []uint32) ([]uint32, error) {
	permissionIDs, err := r.rolePermissionRepo.GetPermissionsByRoleIDs(ctx, roleIDs)
	if err != nil {
		return nil, err
	}

	menuIDs, err := r.permissionRepo.ListMenuIDsByPermissionIDs(ctx, permissionIDs)
	if err != nil {
		return nil, err
	}

	return menuIDs, nil
}
