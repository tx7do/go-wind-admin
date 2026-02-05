package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/trans"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"
	"go-wind-admin/app/admin/service/internal/data/ent/role"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"

	"go-wind-admin/pkg/constants"
)

type RoleRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper          *mapper.CopierMapper[permissionV1.Role, ent.Role]
	statusConverter *mapper.EnumTypeConverter[permissionV1.Role_Status, role.Status]

	repository *entCrud.Repository[
		ent.RoleQuery, ent.RoleSelect,
		ent.RoleCreate, ent.RoleCreateBulk,
		ent.RoleUpdate, ent.RoleUpdateOne,
		ent.RoleDelete,
		predicate.Role,
		permissionV1.Role, ent.Role,
	]

	rolePermissionRepo *RolePermissionRepo
	permissionRepo     *PermissionRepo
	roleMetadataRepo   *RoleMetadataRepo
}

func NewRoleRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
	rolePermissionRepo *RolePermissionRepo,
	permissionRepo *PermissionRepo,
	roleMetadataRepo *RoleMetadataRepo,
) *RoleRepo {
	repo := &RoleRepo{
		log:       ctx.NewLoggerHelper("role/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[permissionV1.Role, ent.Role](),
		statusConverter: mapper.NewEnumTypeConverter[permissionV1.Role_Status, role.Status](
			permissionV1.Role_Status_name,
			permissionV1.Role_Status_value,
		),
		permissionRepo:     permissionRepo,
		rolePermissionRepo: rolePermissionRepo,
		roleMetadataRepo:   roleMetadataRepo,
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
		permissionV1.Role, ent.Role,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
}

// Count 统计角色数量
func (r *RoleRepo) count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().Role.Query()
	if len(whereCond) != 0 {
		builder.Modify(whereCond...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, permissionV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *RoleRepo) Count(ctx context.Context, req *paginationV1.PagingRequest) (int, error) {
	builder := r.entClient.Client().Role.Query()

	whereSelectors, _, err := r.repository.BuildListSelectorWithPaging(builder, req)
	if len(whereSelectors) != 0 {
		builder.Modify(whereSelectors...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf("query role count failed: %s", err.Error())
		return 0, permissionV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

// IsExist 判断角色是否存在
func (r *RoleRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().Role.Query().
		Where(role.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, permissionV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

// List 列表角色信息
func (r *RoleRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*permissionV1.ListRoleResponse, error) {
	if req == nil {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Role.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &permissionV1.ListRoleResponse{Total: 0, Items: nil}, nil
	}

	for _, item := range ret.Items {
		_ = r.fillPermissionIDs(ctx, item)
	}

	return &permissionV1.ListRoleResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

// fillPermissionIDs 填充角色权限ID列表
func (r *RoleRepo) fillPermissionIDs(ctx context.Context, dto *permissionV1.Role) error {
	permissionIDs, err := r.rolePermissionRepo.ListPermissionIDs(ctx, dto.GetId())
	if err != nil {
		r.log.Errorf("list permission ids failed: %s", err.Error())
		return err
	}
	dto.Permissions = permissionIDs
	return nil
}

// ListRolesByRoleCodes 通过角色编码列表获取角色列表
func (r *RoleRepo) ListRolesByRoleCodes(ctx context.Context, codes []string) ([]*permissionV1.Role, error) {
	if len(codes) == 0 {
		return []*permissionV1.Role{}, nil
	}

	entities, err := r.entClient.Client().Role.Query().
		Where(role.CodeIn(codes...)).
		All(ctx)
	if err != nil {
		r.log.Errorf("query roles by codes failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query roles by codes failed")
	}

	dtos := make([]*permissionV1.Role, 0, len(entities))
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
func (r *RoleRepo) ListRolesByRoleIds(ctx context.Context, ids []uint32) ([]*permissionV1.Role, error) {
	if len(ids) == 0 {
		return []*permissionV1.Role{}, nil
	}

	entities, err := r.entClient.Client().Role.Query().
		Where(role.IDIn(ids...)).
		All(ctx)
	if err != nil {
		r.log.Errorf("query roles by ids failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query roles by ids failed")
	}

	dtos := make([]*permissionV1.Role, 0, len(entities))
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

// ListRoleIDsByRoleCodes 通过角色编码列表获取角色ID列表
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
		return nil, permissionV1.ErrorInternalServerError("query role ids failed")
	}

	ids := make([]uint32, 0, len(entities))
	for _, entity := range entities {
		ids = append(ids, entity.ID)
	}

	return ids, nil
}

// Get 获取角色信息
func (r *RoleRepo) Get(ctx context.Context, req *permissionV1.GetRoleRequest) (*permissionV1.Role, error) {
	if req == nil {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Role.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *permissionV1.GetRoleRequest_Id:
		whereCond = append(whereCond, role.IDEQ(req.GetId()))
	case *permissionV1.GetRoleRequest_Name:
		whereCond = append(whereCond, role.NameEQ(req.GetName()))
	case *permissionV1.GetRoleRequest_Code:
		whereCond = append(whereCond, role.CodeEQ(req.GetCode()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	_ = r.fillPermissionIDs(ctx, dto)

	return dto, err
}

// GetTemplateRole 获取角色模板信息
func (r *RoleRepo) GetTemplateRole(ctx context.Context, templateCode string) (*permissionV1.Role, error) {
	if templateCode == "" {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	code := constants.TemplateRoleCodePrefix + templateCode
	builder := r.entClient.Client().Role.Query().
		Where(
			role.CodeEQ(code),
			role.StatusEQ(role.StatusOn),
			role.IsProtectedEQ(true),
			role.Or(
				role.TenantIDIsNil(),
				role.TenantIDEQ(0),
			),
		)

	dto, err := r.repository.Get(ctx, builder, nil)
	if err != nil {
		return nil, err
	}

	_ = r.fillPermissionIDs(ctx, dto)

	return dto, err
}

// CreateTenantRoleFromTemplate 从模版创建租户角色
func (r *RoleRepo) CreateTenantRoleFromTemplate(ctx context.Context, tx *ent.Tx, tenantID, operatorID uint32) (dto *permissionV1.Role, err error) {
	roleTemplate, err := r.Get(ctx, &permissionV1.GetRoleRequest{
		QueryBy: &permissionV1.GetRoleRequest_Code{
			Code: constants.TenantAdminTemplateRoleCode,
		}},
	)
	if err != nil {
		return nil, err
	}

	roleTemplate.Id = nil
	roleTemplate.Name = trans.Ptr(constants.DefaultTenantManagerRoleName)
	roleTemplate.Code = trans.Ptr(constants.ExtractRoleCodeFromTemplate(roleTemplate.GetCode()))
	roleTemplate.TenantId = trans.Ptr(tenantID)
	roleTemplate.CreatedBy = trans.Ptr(operatorID)
	roleTemplate.CreatedAt = nil
	roleTemplate.UpdatedBy = nil
	roleTemplate.UpdatedAt = nil

	//r.log.Infof("Creating tenant role from template: %+v", roleTemplate)

	dto, err = r.CreateWithTx(ctx, tx, roleTemplate)
	if err != nil {
		return nil, err
	}

	return dto, nil
}

// Create 创建角色
func (r *RoleRepo) Create(ctx context.Context, req *permissionV1.CreateRoleRequest) (err error) {
	if req == nil || req.Data == nil {
		return permissionV1.ErrorBadRequest("invalid parameter")
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

	_, err = r.CreateWithTx(ctx, tx, req.GetData())
	return err
}

// CreateWithTx 创建角色
func (r *RoleRepo) CreateWithTx(ctx context.Context, tx *ent.Tx, data *permissionV1.Role) (dto *permissionV1.Role, err error) {
	if data == nil {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	builder := tx.Role.Create().
		SetNillableTenantID(data.TenantId).
		SetNillableName(data.Name).
		SetNillableCode(data.Code).
		SetNillableSortOrder(data.SortOrder).
		SetNillableIsProtected(data.IsProtected).
		SetNillableIsSystem(data.IsSystem).
		SetNillableStatus(r.statusConverter.ToEntity(data.Status)).
		SetNillableDescription(data.Description).
		SetNillableCreatedBy(data.CreatedBy).
		SetCreatedAt(time.Now())

	if data.Id != nil {
		builder.SetID(data.GetId())
	}

	var ret *ent.Role
	if ret, err = builder.Save(ctx); err != nil {
		r.log.Errorf("insert role failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("insert role failed")
	}

	// 创建角色元数据
	var scope *permissionV1.RoleMetadata_Scope
	if data.GetIsSystem() {
		scope = permissionV1.RoleMetadata_PLATFORM.Enum()
	} else {
		scope = permissionV1.RoleMetadata_TENANT.Enum()
	}
	isTemplate := constants.IsTemplateRoleCode(data.GetCode())
	var templateFor string
	if isTemplate {
		templateFor = constants.ExtractRoleCodeFromTemplate(data.GetCode())
	}
	if err = r.roleMetadataRepo.Create(ctx, tx, &permissionV1.RoleMetadata{
		RoleId:      trans.Ptr(ret.ID),
		TenantId:    data.TenantId,
		CreatedBy:   data.CreatedBy,
		IsTemplate:  trans.Ptr(isTemplate),
		TemplateFor: trans.Ptr(templateFor),
		SyncPolicy:  permissionV1.RoleMetadata_AUTO.Enum(),
		Scope:       scope,
	}); err != nil {
		r.log.Errorf("create role metadata failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("create role metadata failed")
	}

	// 分配权限到角色
	if len(data.Permissions) > 0 {
		if err = r.assignPermissionsToRole(ctx, tx,
			data.GetTenantId(), data.GetCreatedBy(),
			ret.ID, data.Permissions); err != nil {
			r.log.Errorf("assign permissions to role failed: %s", err.Error())
			return nil, permissionV1.ErrorInternalServerError("assign permissions to role failed")
		}
	}

	return r.mapper.ToDTO(ret), nil
}

// Update 更新角色信息
func (r *RoleRepo) Update(ctx context.Context, req *permissionV1.UpdateRoleRequest) (err error) {
	if req == nil || req.Data == nil {
		return permissionV1.ErrorBadRequest("invalid parameter")
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return err
		}
		if !exist {
			createReq := &permissionV1.CreateRoleRequest{Data: req.Data}
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

	var entity *permissionV1.Role
	builder := tx.Role.UpdateOneID(req.GetId())
	entity, err = r.repository.UpdateOne(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *permissionV1.Role) {
			builder.
				SetNillableName(req.Data.Name).
				SetNillableCode(req.Data.Code).
				SetNillableSortOrder(req.Data.SortOrder).
				SetNillableIsProtected(req.Data.IsProtected).
				SetNillableIsSystem(req.Data.IsSystem).
				SetNillableStatus(r.statusConverter.ToEntity(req.Data.Status)).
				SetNillableDescription(req.Data.Description).
				SetNillableUpdatedBy(req.Data.UpdatedBy).
				SetUpdatedAt(time.Now())
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(role.FieldID, req.GetId()))
		},
	)
	if err != nil {
		r.log.Errorf("update role failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("update role failed")
	}

	// 升级角色元数据模板版本
	if err = r.roleMetadataRepo.UpgradeTemplateVersion(ctx, tx, req.GetId()); err != nil {
		r.log.Errorf("upgrade role metadata template version failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("upgrade role metadata template version failed")
	}

	if len(req.Data.Permissions) > 0 {
		if err = r.assignPermissionsToRole(ctx, tx,
			*entity.TenantId, req.Data.GetUpdatedBy(),
			req.GetId(), req.Data.Permissions); err != nil {
			r.log.Errorf("assign permissions to role failed: %s", err.Error())
			return permissionV1.ErrorInternalServerError("assign permissions to role failed")
		}
	}

	return nil
}

// Delete 删除角色
func (r *RoleRepo) Delete(ctx context.Context, req *permissionV1.DeleteRoleRequest) (err error) {
	if req == nil {
		return permissionV1.ErrorBadRequest("invalid parameter")
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

	ret, err := tx.Role.Query().Where(role.IDEQ(req.GetId())).Only(ctx)
	if err != nil {
		r.log.Errorf("get role failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("get role failed")
	}

	// 保护角色禁止删除
	if ret.IsProtected != nil && *ret.IsProtected {
		return permissionV1.ErrorForbidden("protected role cannot be deleted")
	}
	if ret.IsSystem != nil && *ret.IsSystem {
		return permissionV1.ErrorForbidden("protected role cannot be deleted")
	}

	// 删除角色记录
	if _, err = tx.Role.Delete().
		Where(role.IDEQ(req.GetId())).
		Exec(ctx); err != nil {
		r.log.Errorf("delete role failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete role failed")
	}

	if err = r.rolePermissionRepo.CleanPermissions(ctx, tx, req.GetId()); err != nil {
		return err
	}

	return nil
}

// ListPermissionIDsByRoleIDs 通过角色ID列表获取权限ID列表
func (r *RoleRepo) ListPermissionIDsByRoleIDs(ctx context.Context, roleIDs []uint32) ([]uint32, error) {
	return r.rolePermissionRepo.ListPermissionIDsByRoleIDs(ctx, roleIDs)
}

// ListPermissionIDsByRoleCodes 通过角色编码列表获取权限ID列表
func (r *RoleRepo) ListPermissionIDsByRoleCodes(ctx context.Context, roleCodes []string) ([]uint32, error) {
	roleIDs, err := r.ListRoleIDsByRoleCodes(ctx, roleCodes)
	if err != nil {
		return nil, err
	}

	return r.rolePermissionRepo.ListPermissionIDsByRoleIDs(ctx, roleIDs)
}

// assignPermissionCodesToRole 分配权限编码给角色
func (r *RoleRepo) assignPermissionCodesToRole(ctx context.Context, tx *ent.Tx,
	tenantID, operatorID uint32,
	roleID uint32,
	codes []string,
) error {
	ids, err := r.permissionRepo.GetPermissionIDsByCodesWithTx(ctx, tx, codes)
	if err != nil {
		return err
	}

	return r.rolePermissionRepo.AssignPermissions(ctx, tx, tenantID, operatorID, roleID, ids)
}

// assignPermissionsToRole 分配权限给角色
func (r *RoleRepo) assignPermissionsToRole(ctx context.Context, tx *ent.Tx,
	tenantID, operatorID uint32,
	roleID uint32,
	permissionIDs []uint32,
) error {
	return r.rolePermissionRepo.AssignPermissions(ctx, tx, tenantID, operatorID, roleID, permissionIDs)
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

// GetRolesPermissionMenuIDs 获取多个角色关联的权限菜单ID列表
func (r *RoleRepo) GetRolesPermissionMenuIDs(ctx context.Context, roleIDs []uint32) ([]uint32, error) {
	permissionIDs, err := r.rolePermissionRepo.ListPermissionIDsByRoleIDs(ctx, roleIDs)
	if err != nil {
		return nil, err
	}

	menuIDs, err := r.permissionRepo.ListMenuIDsByPermissionIDs(ctx, permissionIDs)
	if err != nil {
		return nil, err
	}

	return menuIDs, nil
}

// CanAssignRole 判断角色是否可以被分配（非保护且启用状态）
func (r *RoleRepo) CanAssignRole(ctx context.Context, roleID uint32) (bool, error) {
	aRole, err := r.Get(ctx, &permissionV1.GetRoleRequest{QueryBy: &permissionV1.GetRoleRequest_Id{Id: roleID}})
	if err != nil {
		return false, err
	}

	// 确认角色是否为开启状态
	if aRole.GetStatus() != permissionV1.Role_ON {
		return false, permissionV1.ErrorForbidden("角色未启用，禁止分配")
	}

	metadata, err := r.roleMetadataRepo.Get(ctx, roleID)
	if err != nil {
		return false, err
	}

	// 确认角色是否为模板角色
	if metadata.GetIsTemplate() {
		return false, permissionV1.ErrorForbidden("角色模板不可分配")
	}

	// 确认角色是否为阻断同步角色
	if metadata.GetSyncPolicy() == permissionV1.RoleMetadata_BLOCKED {
		return false, permissionV1.ErrorForbidden("该角色为阻断同步角色，禁止分配")
	}

	// 确认角色是否为平台管理员模板角色
	if metadata.GetTemplateFor() == constants.PlatformAdminRoleCode {
		return false, permissionV1.ErrorForbidden("平台管理员模板角色禁止分配")
	}

	return true, nil
}
