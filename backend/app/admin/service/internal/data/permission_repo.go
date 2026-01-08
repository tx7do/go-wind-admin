package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	pagination "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/timeutil"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/permission"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

type PermissionRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper          *mapper.CopierMapper[permissionV1.Permission, ent.Permission]
	statusConverter *mapper.EnumTypeConverter[permissionV1.Permission_Status, permission.Status]

	repository *entCrud.Repository[
		ent.PermissionQuery, ent.PermissionSelect,
		ent.PermissionCreate, ent.PermissionCreateBulk,
		ent.PermissionUpdate, ent.PermissionUpdateOne,
		ent.PermissionDelete,
		predicate.Permission,
		permissionV1.Permission, ent.Permission,
	]

	permissionApiRepo  *PermissionApiRepo
	permissionMenuRepo *PermissionMenuRepo
}

func NewPermissionRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
	permissionApiRepo *PermissionApiRepo,
	permissionMenuRepo *PermissionMenuRepo,
) *PermissionRepo {
	repo := &PermissionRepo{
		log:                ctx.NewLoggerHelper("permission/repo/admin-service"),
		entClient:          entClient,
		mapper:             mapper.NewCopierMapper[permissionV1.Permission, ent.Permission](),
		statusConverter:    mapper.NewEnumTypeConverter[permissionV1.Permission_Status, permission.Status](permissionV1.Permission_Status_name, permissionV1.Permission_Status_value),
		permissionApiRepo:  permissionApiRepo,
		permissionMenuRepo: permissionMenuRepo,
	}

	repo.init()

	return repo
}

func (r *PermissionRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.PermissionQuery, ent.PermissionSelect,
		ent.PermissionCreate, ent.PermissionCreateBulk,
		ent.PermissionUpdate, ent.PermissionUpdateOne,
		ent.PermissionDelete,
		predicate.Permission,
		permissionV1.Permission, ent.Permission,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
}

func (r *PermissionRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().Permission.Query()
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

func (r *PermissionRepo) List(ctx context.Context, req *pagination.PagingRequest) (*permissionV1.ListPermissionResponse, error) {
	if req == nil {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Permission.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &permissionV1.ListPermissionResponse{Total: 0, Items: nil}, nil
	}

	hasMenuID := hasPath("menu_id", req.GetFieldMask())
	hasApiID := hasPath("api_id", req.GetFieldMask())

	for _, dto := range ret.Items {
		if hasMenuID {
			menuIDs, err := r.permissionMenuRepo.ListMenuIDs(ctx, []uint32{dto.GetId()})
			if err != nil {
				return nil, err
			}
			dto.MenuIds = menuIDs
		}
		if hasApiID {
			apiIDs, err := r.permissionApiRepo.ListApiIDs(ctx, []uint32{dto.GetId()})
			if err != nil {
				return nil, err
			}
			dto.ApiIds = apiIDs
		}
	}

	return &permissionV1.ListPermissionResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

// GetPermissionCodesByIDs 通过权限ID列表获取权限代码列表
func (r *PermissionRepo) GetPermissionCodesByIDs(ctx context.Context, ids []uint32) ([]string, error) {
	q := r.entClient.Client().Permission.Query().
		Where(
			permission.IDIn(ids...),
		)

	codes, err := q.
		Select(permission.FieldCode).
		Strings(ctx)
	if err != nil {
		r.log.Errorf("query permission codes by ids failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query permission codes by ids failed")
	}
	return codes, nil
}

// GetPermissionIDsByCodes 通过权限代码列表获取权限ID列表
func (r *PermissionRepo) GetPermissionIDsByCodes(ctx context.Context, tenantID uint32, codes []string) ([]uint32, error) {
	q := r.entClient.Client().Permission.Query().
		Where(
			permission.TenantIDEQ(tenantID),
			permission.CodeIn(codes...),
		)

	intIDs, err := q.
		Select(permission.FieldID).
		Ints(ctx)
	if err != nil {
		r.log.Errorf("query permission ids by codes failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query permission ids by codes failed")
	}
	ids := make([]uint32, len(intIDs))
	for i, v := range intIDs {
		ids[i] = uint32(v)
	}
	return ids, nil
}

// GetPermissionIDsByCodesWithTx 通过权限代码列表获取权限ID列表
func (r *PermissionRepo) GetPermissionIDsByCodesWithTx(ctx context.Context, tx *ent.Tx, tenantID uint32, codes []string) ([]uint32, error) {
	q := tx.Permission.Query().
		Where(
			permission.TenantIDEQ(tenantID),
			permission.CodeIn(codes...),
		)

	intIDs, err := q.
		Select(permission.FieldID).
		Ints(ctx)
	if err != nil {
		r.log.Errorf("query permission ids by codes failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query permission ids by codes failed")
	}
	ids := make([]uint32, len(intIDs))
	for i, v := range intIDs {
		ids[i] = uint32(v)
	}
	return ids, nil
}

// IsExist 检查 Permission 是否存在
func (r *PermissionRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().Permission.Query().
		Where(permission.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, permissionV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

// Get 获取 Permission
func (r *PermissionRepo) Get(ctx context.Context, req *permissionV1.GetPermissionRequest) (*permissionV1.Permission, error) {
	if req == nil {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Permission.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *permissionV1.GetPermissionRequest_Id:
		whereCond = append(whereCond, permission.IDEQ(req.GetId()))
	case *permissionV1.GetPermissionRequest_Code:
		whereCond = append(whereCond, permission.CodeEQ(req.GetCode()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	if hasPath("api_id", req.GetViewMask()) {
		apiIDs, err := r.permissionApiRepo.ListApiIDs(ctx, []uint32{dto.GetId()})
		if err != nil {
			return nil, err
		}
		dto.ApiIds = apiIDs
	}
	if hasPath("menu_id", req.GetViewMask()) {
		menuIDs, err := r.permissionMenuRepo.ListMenuIDs(ctx, []uint32{dto.GetId()})
		if err != nil {
			return nil, err
		}
		dto.MenuIds = menuIDs
	}

	return dto, err
}

func hasPath(path string, fieldMask *fieldmaskpb.FieldMask) bool {
	if fieldMask == nil {
		return true
	}
	for _, p := range fieldMask.GetPaths() {
		if path == p {
			return true
		}
	}
	return false
}

// Create 创建 Permission
func (r *PermissionRepo) Create(ctx context.Context, req *permissionV1.CreatePermissionRequest) error {
	if req == nil || req.Data == nil {
		return permissionV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.newPermissionCreate(req.Data)

	var entity *ent.Permission
	var err error
	if entity, err = builder.Save(ctx); err != nil {
		r.log.Errorf("insert one data failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("insert data failed")
	}

	if len(req.Data.ApiIds) > 0 {
		if err = r.permissionApiRepo.AssignApis(ctx, req.Data.GetTenantId(), entity.ID, req.Data.GetApiIds()); err != nil {
			return err
		}
	}
	if len(req.Data.MenuIds) > 0 {
		if err = r.permissionMenuRepo.AssignMenus(ctx, req.Data.GetTenantId(), entity.ID, req.Data.GetMenuIds()); err != nil {
			return err
		}
	}

	return nil
}

// BatchCreate 批量创建 Permission
func (r *PermissionRepo) BatchCreate(ctx context.Context, tenantID uint32, permissions []*permissionV1.Permission) (err error) {
	if len(permissions) == 0 {
		return permissionV1.ErrorBadRequest("invalid parameter")
	}

	var permissionCreates []*ent.PermissionCreate
	for _, perm := range permissions {
		pc := r.newPermissionCreate(perm)
		permissionCreates = append(permissionCreates, pc)
	}

	builder := r.entClient.Client().Permission.CreateBulk(permissionCreates...)

	var entities []*ent.Permission
	if entities, err = builder.Save(ctx); err != nil {
		r.log.Errorf("batch insert data failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("batch insert data failed")
	}

	for i, perm := range permissions {
		entity := entities[i]

		if err = r.permissionApiRepo.AssignApis(ctx, tenantID, entity.ID, perm.GetApiIds()); err != nil {
			return err
		}

		if err = r.permissionMenuRepo.AssignMenus(ctx, tenantID, entity.ID, perm.GetMenuIds()); err != nil {
			return err
		}
	}

	return nil
}

// newPermissionCreate 创建 Permission Create 构造器
func (r *PermissionRepo) newPermissionCreate(permission *permissionV1.Permission) *ent.PermissionCreate {
	builder := r.entClient.Client().Permission.Create().
		SetName(permission.GetName()).
		SetCode(permission.GetCode()).
		SetNillableStatus(r.statusConverter.ToEntity(permission.Status)).
		SetNillableGroupID(permission.GroupId).
		SetNillableTenantID(permission.TenantId).
		SetNillableCreatedBy(permission.CreatedBy).
		SetNillableCreatedAt(timeutil.TimestamppbToTime(permission.CreatedAt))

	if permission.TenantId == nil {
		builder.SetTenantID(permission.GetTenantId())
	}
	if permission.CreatedAt == nil {
		builder.SetCreatedAt(time.Now())
	}

	if permission.Id != nil {
		builder.SetID(permission.GetId())
	}

	return builder
}

// Update 更新 Permission
func (r *PermissionRepo) Update(ctx context.Context, tenantID uint32, req *permissionV1.UpdatePermissionRequest) error {
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
			createReq := &permissionV1.CreatePermissionRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	builder := r.entClient.Client().Debug().Permission.UpdateOneID(req.GetId())
	perm, err := r.repository.UpdateOne(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *permissionV1.Permission) {
			builder.
				SetNillableName(req.Data.Name).
				SetNillableCode(req.Data.Code).
				SetNillableGroupID(req.Data.GroupId).
				SetNillableStatus(r.statusConverter.ToEntity(req.Data.Status)).
				SetNillableUpdatedBy(req.Data.UpdatedBy).
				SetNillableUpdatedAt(timeutil.TimestamppbToTime(req.Data.UpdatedAt))

			if req.Data.UpdatedAt == nil {
				builder.SetUpdatedAt(time.Now())
			}
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(permission.FieldID, req.GetId()))
		},
	)
	if err != nil {
		return err
	}

	if err = r.permissionApiRepo.AssignApis(ctx, tenantID, perm.GetId(), req.Data.GetApiIds()); err != nil {
		return err
	}

	if err = r.permissionMenuRepo.AssignMenus(ctx, tenantID, perm.GetId(), req.Data.GetMenuIds()); err != nil {
		return err
	}

	return nil
}

// Delete 删除 Permission
func (r *PermissionRepo) Delete(ctx context.Context, req *permissionV1.DeletePermissionRequest) error {
	if req == nil {
		return permissionV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Permission.Delete()

	_, err := r.repository.Delete(ctx, builder, func(s *sql.Selector) {
		s.Where(sql.EQ(permission.FieldID, req.GetId()))
	})
	if err != nil {
		r.log.Errorf("delete permission failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete permission failed")
	}

	if err = r.permissionApiRepo.Delete(ctx, req.GetId()); err != nil {
		return err
	}

	if err = r.permissionMenuRepo.Delete(ctx, req.GetId()); err != nil {
		return err
	}

	return nil
}

// Truncate 清空表数据
func (r *PermissionRepo) Truncate(ctx context.Context) error {
	if _, err := r.entClient.Client().Permission.Delete().Exec(ctx); err != nil {
		r.log.Errorf("failed to truncate permission table: %s", err.Error())
		return permissionV1.ErrorInternalServerError("truncate failed")
	}

	if err := r.permissionApiRepo.Truncate(ctx); err != nil {
		return err
	}

	if err := r.permissionMenuRepo.Truncate(ctx); err != nil {
		return err
	}

	return nil
}

// CleanPermissionsByCodes 清理指定权限代码的权限
func (r *PermissionRepo) CleanPermissionsByCodes(ctx context.Context, codes []string) error {
	builder := r.entClient.Client().Permission.Delete().
		Where(
			permission.CodeIn(codes...),
		)

	_, err := builder.Exec(ctx)
	if err != nil {
		r.log.Errorf("delete permissions by codes failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete permissions by codes failed")
	}

	return nil
}

// CleanApiPermissions 清理API相关权限
func (r *PermissionRepo) CleanApiPermissions(ctx context.Context) error {
	return r.permissionApiRepo.Truncate(ctx)
}

// CleanDataPermissions 清理数据权限
func (r *PermissionRepo) CleanDataPermissions(ctx context.Context) error {
	return nil
}

// CleanMenuPermissions 清理菜单相关权限
func (r *PermissionRepo) CleanMenuPermissions(ctx context.Context) error {
	return r.permissionMenuRepo.Truncate(ctx)
}

// ListApiIDsByPermissionIDs 列出权限关联的API资源ID列表
func (r *PermissionRepo) ListApiIDsByPermissionIDs(ctx context.Context, permissionIDs []uint32) ([]uint32, error) {
	apiIDs, err := r.permissionApiRepo.ListApiIDs(ctx, permissionIDs)
	if err != nil {
		return nil, err
	}

	return apiIDs, nil
}

// ListMenuIDsByPermissionIDs 列出权限关联的菜单ID列表
func (r *PermissionRepo) ListMenuIDsByPermissionIDs(ctx context.Context, permissionIDs []uint32) ([]uint32, error) {
	apiIDs, err := r.permissionMenuRepo.ListMenuIDs(ctx, permissionIDs)
	if err != nil {
		return nil, err
	}

	return apiIDs, nil
}
