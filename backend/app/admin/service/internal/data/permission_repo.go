package data

import (
	"context"
	"slices"
	"strconv"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/permission"
	"go-wind-admin/app/admin/service/internal/data/ent/permissionpolicy"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"

	"github.com/tx7do/go-utils/sliceutil"
	"github.com/tx7do/go-utils/trans"
	"go-wind-admin/pkg/constants"
)

type PermissionRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *bLogger.Helper

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
	rolePermissionRepo *RolePermissionRepo
}

func NewPermissionRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
	permissionApiRepo *PermissionApiRepo,
	permissionMenuRepo *PermissionMenuRepo,
	rolePermissionRepo *RolePermissionRepo,
) *PermissionRepo {
	repo := &PermissionRepo{
		log:       ctx.NewLoggerHelper("permission/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[permissionV1.Permission, ent.Permission](),
		statusConverter: mapper.NewEnumTypeConverter[permissionV1.Permission_Status, permission.Status](
			permissionV1.Permission_Status_name, permissionV1.Permission_Status_value,
		),
		permissionApiRepo:  permissionApiRepo,
		permissionMenuRepo: permissionMenuRepo,
		rolePermissionRepo: rolePermissionRepo,
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

func (r *PermissionRepo) Count(ctx context.Context, req *paginationV1.PagingRequest) (*permissionV1.CountPermissionResponse, error) {
	builder := r.entClient.Client().Permission.Query()

	whereSelectors, _, _ := r.repository.BuildListSelectorWithPaging(builder, req)
	if len(whereSelectors) != 0 {
		builder.Modify(whereSelectors...)
	}


	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query permission count failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query permission count failed")
	}

	return &permissionV1.CountPermissionResponse{
		Count: uint64(count),
	}, nil
}

func clearFilterExprByFieldNames(expr *paginationV1.FilterExpr, fieldName string) {
	if expr == nil {
		return
	}

	for i := len(expr.GetConditions()) - 1; i >= 0; i-- {
		cond := expr.GetConditions()[i]
		if cond.GetField() == fieldName {
			expr.Conditions = append(expr.Conditions[:i], expr.Conditions[i+1:]...)
		}
	}

	for _, subExpr := range expr.GetGroups() {
		clearFilterExprByFieldNames(subExpr, fieldName)
	}
}

func (r *PermissionRepo) List(ctx context.Context, req *paginationV1.PagingRequest, limitPermissionIDs []uint32) (*permissionV1.ListPermissionResponse, error) {
	if req == nil {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Permission.Query()

	if len(limitPermissionIDs) > 0 {
		filterExpr, err := r.repository.ConvertFilterByPagingRequest(req)
		if err != nil {
			return nil, err
		}
		if filterExpr != nil {
			clearFilterExprByFieldNames(filterExpr, "id")
			req.FilteringType = &paginationV1.PagingRequest_FilterExpr{FilterExpr: filterExpr}
		}

		var strIDs []string
		for _, id := range limitPermissionIDs {
			strIDs = append(strIDs, strconv.FormatUint(uint64(id), 10))
		}

		condition := &paginationV1.FilterCondition{
			Field:  "id",
			Op:     paginationV1.Operator_IN,
			Values: strIDs,
		}

		if req.FilteringType == nil {
			req.FilteringType = &paginationV1.PagingRequest_FilterExpr{
				FilterExpr: &paginationV1.FilterExpr{
					Type: paginationV1.ExprType_AND,
					Conditions: []*paginationV1.FilterCondition{
						condition,
					},
				},
			}
		} else {
			req.GetFilterExpr().Conditions = append(req.GetFilterExpr().GetConditions(), condition)
		}
	}

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
		r.log.Errorf(ctx, "query permission codes by ids failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query permission codes by ids failed")
	}
	return codes, nil
}

// GetPermissionIDsByCodes 通过权限代码列表获取权限ID列表
func (r *PermissionRepo) GetPermissionIDsByCodes(ctx context.Context, codes []string) ([]uint32, error) {
	q := r.entClient.Client().Permission.Query().
		Where(
			permission.CodeIn(codes...),
		)

	intIDs, err := q.
		Select(permission.FieldID).
		Ints(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query permission ids by codes failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query permission ids by codes failed")
	}
	ids := make([]uint32, len(intIDs))
	for i, v := range intIDs {
		ids[i] = uint32(v)
	}
	return ids, nil
}

// GetPermissionIDsByCodesWithTx 通过权限代码列表获取权限ID列表
func (r *PermissionRepo) GetPermissionIDsByCodesWithTx(ctx context.Context, tx *ent.Tx, codes []string) ([]uint32, error) {
	q := tx.Permission.Query().
		Where(
			permission.CodeIn(codes...),
		)

	intIDs, err := q.
		Select(permission.FieldID).
		Ints(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query permission ids by codes failed: %s", err.Error())
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
		r.log.Errorf(ctx, "query exist failed: %s", err.Error())
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
		r.log.Errorf(ctx, "insert permission failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("insert permission failed")
	}

	if len(req.Data.ApiIds) > 0 {
		if err = r.permissionApiRepo.AssignApis(ctx, entity.ID, req.Data.GetApiIds()); err != nil {
			return err
		}
	}
	if len(req.Data.MenuIds) > 0 {
		if err = r.permissionMenuRepo.AssignMenus(ctx, entity.ID, req.Data.GetMenuIds()); err != nil {
			return err
		}
	}

	return nil
}

// BatchCreate 批量创建 Permission
func (r *PermissionRepo) BatchCreate(ctx context.Context, permissions []*permissionV1.Permission) (err error) {
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
		r.log.Errorf(ctx, "batch insert permissions failed: %s", err.Error())
		// 带出底层原因（唯一约束/必填缺失各有排障方向），不吞成同文案
		return permissionV1.ErrorInternalServerError("batch insert permissions failed: %s", err.Error())
	}

	for i, perm := range permissions {
		entity := entities[i]

		if err = r.permissionApiRepo.AssignApis(ctx, entity.ID, perm.GetApiIds()); err != nil {
			return err
		}

		if err = r.permissionMenuRepo.AssignMenus(ctx, entity.ID, perm.GetMenuIds()); err != nil {
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
		SetNillableDescription(permission.Description).
		SetNillableGroupID(permission.GroupId).
		SetNillableCreatedBy(permission.CreatedBy).
		SetCreatedAt(time.Now())

	if permission.Id != nil {
		builder.SetID(permission.GetId())
	}

	return builder
}

// Update 更新 Permission
func (r *PermissionRepo) Update(ctx context.Context, req *permissionV1.UpdatePermissionRequest) error {
	if req == nil || req.Data == nil {
		return permissionV1.ErrorBadRequest("invalid parameter")
	}
	if req.GetId() == 0 {
		return permissionV1.ErrorBadRequest("id is required")
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

	// 关联载荷先快照：UpdateOne 内部的 FilterByFieldMask 会把不在 mask 里的 Data 字段清零，
	// 而 api_ids / menu_ids 恰被下面的黑名单移出 mask——不先取就走不到 Assign（编辑权限会静默
	// 清空它的全部菜单与接口授权）。语义照 role_repo / user_repo 的快照模式：提交了该项（含清空）
	// 才整体替换，没提交则维持原关联。
	updateApis := hasPath("api_ids", req.UpdateMask) || hasPath("apiIds", req.UpdateMask)
	updateMenus := hasPath("menu_ids", req.UpdateMask) || hasPath("menuIds", req.UpdateMask)
	var wantApiIds, wantMenuIds []uint32
	if updateApis {
		wantApiIds = slices.Clone(req.Data.GetApiIds())
	}
	if updateMenus {
		wantMenuIds = slices.Clone(req.Data.GetMenuIds())
	}

	// 剔除关联字段：api_ids / menu_ids 是关联表字段，并非 sys_permissions 表的列。
	// 若保留在 updateMask 中，当其值为空时会被当作 nil 字段生成 SET api_ids=NULL 的 SQL，触发列不存在错误。
	// 关联关系由下方的 AssignApis / AssignMenus 单独维护。
	if req.UpdateMask != nil {
		req.UpdateMask.Paths = sliceutil.FilterBlacklist(req.UpdateMask.GetPaths(), []string{
			"api_ids", "menu_ids",
		})
	}

	builder := r.entClient.Client().Permission.UpdateOneID(req.GetId())
	perm, err := r.repository.UpdateOne(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *permissionV1.Permission) {
			builder.
				SetNillableName(req.Data.Name).
				SetNillableCode(req.Data.Code).
				SetNillableGroupID(req.Data.GroupId).
				SetNillableStatus(r.statusConverter.ToEntity(req.Data.Status)).
				SetNillableDescription(req.Data.Description).
				SetNillableUpdatedBy(req.Data.UpdatedBy).
				SetUpdatedAt(time.Now())
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(permission.FieldID, req.GetId()))
		},
	)
	if err != nil {
		return err
	}

	if updateApis {
		if err = r.permissionApiRepo.AssignApis(ctx, perm.GetId(), wantApiIds); err != nil {
			return err
		}
	}

	if updateMenus {
		if err = r.permissionMenuRepo.AssignMenus(ctx, perm.GetId(), wantMenuIds); err != nil {
			return err
		}
	}

	return nil
}

// Delete 删除 Permission
func (r *PermissionRepo) Delete(ctx context.Context, req *permissionV1.DeletePermissionRequest) (err error) {
	if req == nil {
		return permissionV1.ErrorBadRequest("invalid parameter")
	}

	// 行删除与级联清理同一事务：删了行清不掉绑定会留悬空授权，反之亦然；
	// id 基线与行删除同事务内取，按 code/组删除时不存在两查询间的竞态窗口
	var tx *ent.Tx
	tx, err = r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf(ctx, "start transaction failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("start transaction failed")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.log.Errorf(ctx, "transaction rollback failed: %s", rollbackErr.Error())
			}
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			r.log.Errorf(ctx, "transaction commit failed: %s", commitErr.Error())
			err = permissionV1.ErrorInternalServerError("transaction commit failed")
		}
	}()

	var permissionIDs []uint32
	switch req.QueryBy.(type) {
	default:
	case *permissionV1.DeletePermissionRequest_Id:
		permissionIDs = append(permissionIDs, req.GetId())

	case *permissionV1.DeletePermissionRequest_Code:
		// 注意：Query.IDs/Ints 内部已追加字段选择，外层再 Select 会重复列导致 scan 失败。
		permissionIDs, err = tx.Permission.Query().
			Where(permission.CodeEQ(req.GetCode())).
			IDs(ctx)
		if err != nil {
			r.log.Errorf(ctx, "get permission ids by code failed: %s", err.Error())
			return permissionV1.ErrorInternalServerError("get permission ids by code failed")
		}

	case *permissionV1.DeletePermissionRequest_GroupId:
		permissionIDs, err = tx.Permission.Query().
			Where(permission.GroupIDEQ(req.GetGroupId())).
			IDs(ctx)
		if err != nil {
			r.log.Errorf(ctx, "get permission ids by group id failed: %s", err.Error())
			return permissionV1.ErrorInternalServerError("get permission ids by group id failed")
		}
	}

	builder := tx.Permission.Delete()
	switch req.QueryBy.(type) {
	default:
	case *permissionV1.DeletePermissionRequest_Id:
		builder.Where(permission.IDEQ(req.GetId()))
	case *permissionV1.DeletePermissionRequest_Code:
		builder.Where(permission.CodeEQ(req.GetCode()))
	case *permissionV1.DeletePermissionRequest_GroupId:
		builder.Where(permission.GroupIDEQ(req.GetGroupId()))
	}

	_, err = builder.Exec(ctx)
	if err != nil {
		r.log.Errorf(ctx, "delete permission failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete permission failed")
	}

	// 级联清理全部从属绑定：接口/菜单快照、角色授权（sys_role_permissions，
	// 此前漏清会留悬空授权）、策略联动行
	if err = r.cascadeCleanByPermissionIDsWithClient(ctx, tx.Client(), permissionIDs); err != nil {
		return err
	}

	return nil
}

// ListBizPermissions 列出全部业务权限行（code 非 sys: 前缀）的最小视图
//（id+code+name），权限同步 MERGE 的 diff 基线用；规模有界，无分页必要。
func (r *PermissionRepo) ListBizPermissions(ctx context.Context) ([]*permissionV1.Permission, error) {
	rows, err := r.entClient.Client().Permission.Query().
		Where(
			permission.CodeNEQ(""),
			permission.Not(permission.CodeHasPrefix(constants.SystemPermissionCodePrefix)),
		).
		All(ctx)
	if err != nil {
		r.log.Errorf(ctx, "list biz permissions failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("list biz permissions failed")
	}

	dtos := make([]*permissionV1.Permission, 0, len(rows))
	for _, row := range rows {
		dtos = append(dtos, &permissionV1.Permission{
			Id:   trans.Ptr(row.ID),
			Code: row.Code,
			Name: row.Name,
		})
	}
	return dtos, nil
}

// DeleteByIDs 按 id 集合删除权限行并级联清理全部从属绑定（权限同步 MERGE 的
// 删除段用）。行删除与级联清理同一事务，中途失败整体回滚不留半删状态。
func (r *PermissionRepo) DeleteByIDs(ctx context.Context, permissionIDs []uint32) (err error) {
	if len(permissionIDs) == 0 {
		return nil
	}

	var tx *ent.Tx
	tx, err = r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf(ctx, "start transaction failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("start transaction failed")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.log.Errorf(ctx, "transaction rollback failed: %s", rollbackErr.Error())
			}
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			r.log.Errorf(ctx, "transaction commit failed: %s", commitErr.Error())
			err = permissionV1.ErrorInternalServerError("transaction commit failed")
		}
	}()

	if _, err = tx.Permission.Delete().
		Where(permission.IDIn(permissionIDs...)).
		Exec(ctx); err != nil {
		r.log.Errorf(ctx, "delete permissions by ids failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete permissions by ids failed")
	}

	return r.cascadeCleanByPermissionIDsWithClient(ctx, tx.Client(), permissionIDs)
}

// UpdateForSync 权限同步 MERGE 的更新段：原位更新派生集字段（名称/分组），
// **不触碰 status**——管理端手工停用的权限不被同步重新启用（与菜单同步 MERGE
// 同口径）。
func (r *PermissionRepo) UpdateForSync(ctx context.Context, id uint32, derived *permissionV1.Permission) error {
	builder := r.entClient.Client().Permission.UpdateOneID(id).
		SetNillableName(derived.Name).
		SetNillableGroupID(derived.GroupId).
		SetUpdatedAt(time.Now())
	if err := builder.Exec(ctx); err != nil {
		r.log.Errorf(ctx, "sync update permission failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("sync update permission failed")
	}
	return nil
}

// AssignSyncBindings 权限同步 MERGE 的绑定替换段：按派生集整体替换端点/菜单
// 快照绑定（AssignApis/AssignMenus 自带替换语义，空派生 = 清空该权限的绑定）。
func (r *PermissionRepo) AssignSyncBindings(ctx context.Context, id uint32, apiIDs, menuIDs []uint32) error {
	if err := r.permissionApiRepo.AssignApis(ctx, id, apiIDs); err != nil {
		return err
	}
	return r.permissionMenuRepo.AssignMenus(ctx, id, menuIDs)
}

// cascadeCleanByPermissionIDsWithClient 权限行删除后的级联清理：接口/菜单快照
// 绑定、角色授权绑定（sys_role_permissions）、策略联动行（sys_permission_policies）。
// 授权链上不允许残留指向已删权限行的绑定——"权限行重建换 id 后旧绑定悬空"
// 的最后残留就此终结。client 形参使级联与行删除可同事务执行。
func (r *PermissionRepo) cascadeCleanByPermissionIDsWithClient(ctx context.Context, client *ent.Client, permissionIDs []uint32) error {
	if len(permissionIDs) == 0 {
		return nil
	}

	if err := r.permissionApiRepo.DeleteByPermissionIDsOnClient(ctx, client, permissionIDs); err != nil {
		return err
	}
	if err := r.permissionMenuRepo.DeleteByPermissionIDsOnClient(ctx, client, permissionIDs); err != nil {
		return err
	}
	if err := r.rolePermissionRepo.DeleteByPermissionIDsOnClient(ctx, client, permissionIDs); err != nil {
		return err
	}

	// 策略联动行（PermissionPolicyRepo 为空壳，策略行走裸 ent client）
	if _, err := client.PermissionPolicy.Delete().
		Where(permissionpolicy.PermissionIDIn(permissionIDs...)).
		Exec(ctx); err != nil {
		r.log.Errorf(ctx, "delete permission policies by permission ids failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete permission policies failed")
	}

	return nil
}

// listSystemPermissionIDs 按 code 前缀解析系统种子权限 id 集合（绑定表清理的
// 保留集）。权限行自身由权限同步 MERGE 按 code 前缀保护，绑定保护集与之同源，
// 不再依赖"种子权限=1/2/3"的硬编码 id 假设。
func (r *PermissionRepo) listSystemPermissionIDs(ctx context.Context) ([]uint32, error) {
	return r.entClient.Client().Permission.Query().
		Where(permission.CodeHasPrefix(constants.SystemPermissionCodePrefix)).
		IDs(ctx)
}

// Truncate 清空表数据
func (r *PermissionRepo) Truncate(ctx context.Context) error {
	if _, err := r.entClient.Client().Permission.Delete().Exec(ctx); err != nil {
		r.log.Errorf(ctx, "failed to truncate permission table: %s", err.Error())
		return permissionV1.ErrorInternalServerError("truncate failed")
	}

	// 权限行已全清，绑定全清（不再保留任何权限的绑定）
	if err := r.permissionApiRepo.TruncateExcept(ctx, nil); err != nil {
		return err
	}

	if err := r.permissionMenuRepo.TruncateExcept(ctx, nil); err != nil {
		return err
	}

	return nil
}

// CleanDataPermissions 清理数据权限
func (r *PermissionRepo) CleanDataPermissions(ctx context.Context) error {
	return nil
}

// ListApiEndpointsByPermissionIDs 列出权限关联的端点快照（(path, method)，
// 授权执行键；绑定表即真相，不再回查 sys_apis）
func (r *PermissionRepo) ListApiEndpointsByPermissionIDs(ctx context.Context, permissionIDs []uint32) ([]Endpoint, error) {
	endpoints, err := r.permissionApiRepo.ListEndpoints(ctx, permissionIDs)
	if err != nil {
		return nil, err
	}

	return endpoints, nil
}

// ListMenuFullPathsByPermissionIDs 列出权限关联的菜单全路径快照（菜单身份键；
// 绑定表即真相，导航/回显链路自行反解回 id）
func (r *PermissionRepo) ListMenuFullPathsByPermissionIDs(ctx context.Context, permissionIDs []uint32) ([]string, error) {
	paths, err := r.permissionMenuRepo.ListMenuFullPaths(ctx, permissionIDs)
	if err != nil {
		return nil, err
	}

	return paths, nil
}
