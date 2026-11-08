package data

import (
	"context"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"
	entgoUpdate "github.com/tx7do/go-crud/entgo/update"
	"github.com/tx7do/go-crud/pagination"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/menu"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	"github.com/tx7do/go-utils/trans"
	"google.golang.org/protobuf/proto"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"

	"go-wind-admin/pkg/constants"
)

type MenuRepo struct {
	entClient  *entCrud.EntClient[*ent.Client]
	driverName string
	log        *bLogger.Helper

	// permissionMenu 菜单绑定 repo：菜单删除时按全路径快照级联清理授权绑定
	//（绑定存 full_path 后，菜单行删除而绑定残留会让授权反查悬空）。
	permissionMenu *PermissionMenuRepo

	mapper          *mapper.CopierMapper[permissionV1.Menu, ent.Menu]
	statusConverter *mapper.EnumTypeConverter[permissionV1.Menu_Status, menu.Status]
	typeConverter   *mapper.EnumTypeConverter[permissionV1.Menu_Type, menu.Type]
	moduleConverter *mapper.EnumTypeConverter[identityV1.Module, menu.Module]

	repository *entCrud.Repository[
		ent.MenuQuery, ent.MenuSelect,
		ent.MenuCreate, ent.MenuCreateBulk,
		ent.MenuUpdate, ent.MenuUpdateOne,
		ent.MenuDelete,
		predicate.Menu,
		permissionV1.Menu, ent.Menu,
	]
}

func NewMenuRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client], permissionMenu *PermissionMenuRepo) *MenuRepo {
	repo := &MenuRepo{
		log:             ctx.NewLoggerHelper("menu/repo/admin-service"),
		entClient:       entClient,
		driverName:      driverNameOf(ctx),
		permissionMenu:  permissionMenu,
		mapper:          mapper.NewCopierMapper[permissionV1.Menu, ent.Menu](),
		statusConverter: mapper.NewEnumTypeConverter[permissionV1.Menu_Status, menu.Status](permissionV1.Menu_Status_name, permissionV1.Menu_Status_value),
		typeConverter:   mapper.NewEnumTypeConverter[permissionV1.Menu_Type, menu.Type](permissionV1.Menu_Type_name, permissionV1.Menu_Type_value),
		moduleConverter: mapper.NewEnumTypeConverter[identityV1.Module, menu.Module](identityV1.Module_name, identityV1.Module_value),
	}

	repo.init()

	return repo
}

func (r *MenuRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.MenuQuery, ent.MenuSelect,
		ent.MenuCreate, ent.MenuCreateBulk,
		ent.MenuUpdate, ent.MenuUpdateOne,
		ent.MenuDelete,
		predicate.Menu,
		permissionV1.Menu, ent.Menu,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
	r.mapper.AppendConverters(r.typeConverter.NewConverterPair())
	r.mapper.AppendConverters(r.moduleConverter.NewConverterPair())
}

func (r *MenuRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().Menu.Query()
	if len(whereCond) != 0 {
		builder.Modify(whereCond...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query count failed: %s", err.Error())
		return 0, permissionV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *MenuRepo) List(ctx context.Context, req *paginationV1.PagingRequest, treeTravel bool) (*permissionV1.ListMenuResponse, error) {
	if req == nil {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Menu.Query()

	whereSelectors, _, err := r.repository.BuildListSelectorWithPaging(builder, req)
	if err != nil {
		r.log.Errorf(ctx, "parse list param error [%s]", err.Error())
		return nil, permissionV1.ErrorBadRequest("invalid query parameter")
	}

	entities, err := builder.All(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query menu list failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query menu list failed")
	}

	// 转换所有实体为 DTO
	dtos := make([]*permissionV1.Menu, 0, len(entities))
	for _, entity := range entities {
		dto := r.mapper.ToDTO(entity)
		dtos = append(dtos, dto)
	}

	// 构建树形结构
	if treeTravel {
		dtos = pagination.BuildTree(
			dtos,
			func(node *permissionV1.Menu) *uint32 { return node.Id },
			func(node *permissionV1.Menu) *uint32 { return node.ParentId },
			func(node *permissionV1.Menu) *[]*permissionV1.Menu { return &node.Children },
		)
	}

	count, err := r.Count(ctx, whereSelectors)
	if err != nil {
		return nil, err
	}

	return &permissionV1.ListMenuResponse{
		Total: uint64(count),
		Items: dtos,
	}, nil
}

func (r *MenuRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().Menu.Query().
		Where(menu.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query exist failed: %s", err.Error())
		return false, permissionV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *MenuRepo) Get(ctx context.Context, req *permissionV1.GetMenuRequest) (*permissionV1.Menu, error) {
	if req == nil {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Menu.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *permissionV1.GetMenuRequest_Id:
		whereCond = append(whereCond, menu.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

func (r *MenuRepo) Create(ctx context.Context, req *permissionV1.CreateMenuRequest) error {
	if req == nil || req.Data == nil {
		return permissionV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Menu.Create().
		SetNillableType(r.typeConverter.ToEntity(req.Data.Type)).
		SetNillablePath(req.Data.Path).
		SetNillableRedirect(req.Data.Redirect).
		SetNillableAlias(req.Data.Alias).
		SetNillableName(req.Data.Name).
		SetNillableComponent(req.Data.Component).
		SetNillableStatus(r.statusConverter.ToEntity(req.Data.Status)).
		SetNillableCreatedBy(req.Data.CreatedBy).
		SetCreatedAt(time.Now())

	// parent_id=0 表示挂根节点；proto optional 会把显式 0 当作"已设置"，
	// SetParentID(0) 指向不存在的行触发自引用外键违约，必须按无父级处理。
	if req.Data.ParentId != nil && *req.Data.ParentId > 0 {
		builder.SetParentID(*req.Data.ParentId)
	}

	// module 为 proto 零值（MODULE_UNSPECIFIED）时跳过：ent schema 未声明该值，
	// SetNillableModule 会触发 ModuleValidator 失败。未指定即留空，等价不写。
	if req.Data.Module != nil && *req.Data.Module != identityV1.Module_MODULE_UNSPECIFIED {
		builder.SetNillableModule(r.moduleConverter.ToEntity(req.Data.Module))
	}

	if req.Data.Meta != nil {
		builder.SetMeta(req.Data.Meta)
	}

	if req.Data.Id != nil {
		builder.SetID(req.GetData().GetId())
	}

	if err := builder.Exec(ctx); err != nil {
		r.log.Errorf(ctx, "insert menu failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("insert menu failed")
	}

	return nil
}

// CreateReturn 创建菜单并返回包含数据库生成 ID 的实体
// CreateReturn creates a menu and returns the entity with the database-generated ID
func (r *MenuRepo) CreateReturn(ctx context.Context, req *permissionV1.CreateMenuRequest) (*permissionV1.Menu, error) {
	if req == nil || req.Data == nil {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	return r.createMenuReturn(ctx, r.entClient.Client(), req.Data)
}

// newMenuCreate 在指定 client（事务或直连）上构建菜单插入 builder
func (r *MenuRepo) newMenuCreate(client *ent.Client, data *permissionV1.Menu) *ent.MenuCreate {
	builder := client.Menu.Create().
		SetNillableType(r.typeConverter.ToEntity(data.Type)).
		SetNillablePath(data.Path).
		SetNillableRedirect(data.Redirect).
		SetNillableAlias(data.Alias).
		SetNillableName(data.Name).
		SetNillableComponent(data.Component).
		SetNillableStatus(r.statusConverter.ToEntity(data.Status)).
		SetNillableCreatedBy(data.CreatedBy).
		SetCreatedAt(time.Now())

	// parent_id=0 表示挂根节点；proto optional 会把显式 0 当作"已设置"，
	// SetParentID(0) 指向不存在的行触发自引用外键违约，必须按无父级处理。
	if data.ParentId != nil && *data.ParentId > 0 {
		builder.SetParentID(*data.ParentId)
	}

	// module 为 proto 零值（MODULE_UNSPECIFIED）时跳过：ent schema 未声明该值，
	// SetNillableModule 会触发 ModuleValidator 失败。未指定即留空，等价不写。
	if data.Module != nil && *data.Module != identityV1.Module_MODULE_UNSPECIFIED {
		builder.SetNillableModule(r.moduleConverter.ToEntity(data.Module))
	}

	if data.Meta != nil {
		builder.SetMeta(data.Meta)
	}

	if data.Id != nil {
		builder.SetID(data.GetId())
	}

	return builder
}

func (r *MenuRepo) createMenuReturn(ctx context.Context, client *ent.Client, data *permissionV1.Menu) (*permissionV1.Menu, error) {
	entity, err := r.newMenuCreate(client, data).Save(ctx)
	if err != nil {
		r.log.Errorf(ctx, "insert menu failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("insert menu failed")
	}

	return r.mapper.ToDTO(entity), nil
}

func (r *MenuRepo) Update(ctx context.Context, req *permissionV1.UpdateMenuRequest) error {
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
			createReq := &permissionV1.CreateMenuRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	// 换父前置校验：目标父不得是自身或自身后代（成环让子树从根不可达，
	// 全路径身份键整块失明——授权绑定反查与同步 MERGE 全部匹配不到）。
	if req.Data.ParentId != nil && *req.Data.ParentId > 0 {
		if err := r.ensureAcyclicParent(ctx, req.GetId(), *req.Data.ParentId); err != nil {
			return err
		}
	}

	// 变更前基线：本节点旧全路径与树形态。移动/改名会让本节点及其子树的
	// 身份键（全路径）整体变化，授权绑定快照按此改写（见函数尾部）。
	_, byIDBefore, childrenOf, err := r.buildMenuFullPathIndex(ctx, r.entClient.Client())
	if err != nil {
		return err
	}
	selfOldPath := byIDBefore[req.GetId()]

	var metaPaths []string
	if req.UpdateMask != nil {
		for _, v := range req.UpdateMask.GetPaths() {
			if strings.HasPrefix(v, "meta.") {
				metaPaths = append(metaPaths, strings.SplitAfter(v, "meta.")[1])
			}
		}
	}

	builder := r.entClient.Client().Menu.Update()
	err = r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *permissionV1.Menu) {
			builder.
				SetNillableType(r.typeConverter.ToEntity(req.Data.Type)).
				SetNillablePath(req.Data.Path).
				SetNillableRedirect(req.Data.Redirect).
				SetNillableAlias(req.Data.Alias).
				SetNillableName(req.Data.Name).
				SetNillableComponent(req.Data.Component).
				SetNillableStatus(r.statusConverter.ToEntity(req.Data.Status)).
				SetNillableUpdatedBy(req.Data.UpdatedBy).
				SetUpdatedAt(time.Now())

			// 挂载关系三态：未设置（nil）= 不改；显式 >0 = 挂到该父级；
			// 显式 0 = 移到根级，落 NULL（自引用 FK 不接受 0，与 Create
			// 的 >0 守卫同因）。此前 0 被静默忽略，"移回根级"做不到。
			if req.Data.ParentId != nil {
				if *req.Data.ParentId > 0 {
					builder.SetParentID(*req.Data.ParentId)
				} else {
					builder.ClearParentID()
				}
			}

			// module 为 proto 零值（MODULE_UNSPECIFIED）时跳过：ent schema 未声明该值，
			// SetNillableModule 会触发 ModuleValidator 失败。未指定即不更新该字段。
			if req.Data.Module != nil && *req.Data.Module != identityV1.Module_MODULE_UNSPECIFIED {
				builder.SetNillableModule(r.moduleConverter.ToEntity(req.Data.Module))
			}

			if req.Data.Meta != nil {
				r.updateMetaField(builder, req.Data.Meta, metaPaths)
			}
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(menu.FieldID, req.GetId()))
		},
	)
	if err != nil {
		return err
	}

	// 移动/改名后按"旧全路径 → 新全路径"改写授权绑定快照：绑定以全路径为
	// 身份键，菜单行的全路径变了绑定必须随行，否则角色授权静默失效（删除
	// 有级联清理，这里是更新侧的对称语义）。子树整体一起改写。
	if selfOldPath != "" {
		if err = r.refreshBindingFullPaths(ctx, req.GetId(), byIDBefore, childrenOf); err != nil {
			return err
		}
	}

	return nil
}

// ensureAcyclicParent 校验把 nodeID 挂到 newParentID 下不会成环：从新父沿
// parent 链上行，命中 nodeID 即拒绝。visited 防护让存量环脏数据也不会死循环。
func (r *MenuRepo) ensureAcyclicParent(ctx context.Context, nodeID, newParentID uint32) error {
	if newParentID == nodeID {
		return permissionV1.ErrorBadRequest("cannot move menu under itself")
	}
	rows, err := r.entClient.Client().Menu.Query().
		Select(menu.FieldID, menu.FieldParentID).
		All(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query menus for parent acyclicity check failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("query menus failed")
	}
	parentOf := make(map[uint32]uint32, len(rows))
	for _, row := range rows {
		if row.ParentID != nil && *row.ParentID > 0 {
			parentOf[row.ID] = *row.ParentID
		}
	}
	visited := make(map[uint32]struct{}, len(rows))
	cur := newParentID
	for cur != 0 {
		if cur == nodeID {
			return permissionV1.ErrorBadRequest("cannot move menu under its own descendant")
		}
		if _, seen := visited[cur]; seen {
			// 存量环（与本节点无关）：放行但不死循环，修复交给菜单同步/运维
			break
		}
		visited[cur] = struct{}{}
		cur = parentOf[cur]
	}
	return nil
}

// refreshBindingFullPaths 移动/改名后把本节点及其子树的授权绑定快照从旧全路径
// 改写为新全路径。子树按 parent 链内存收集（visited 防护，存量环不死循环）。
func (r *MenuRepo) refreshBindingFullPaths(ctx context.Context, nodeID uint32, byIDBefore map[uint32]string, childrenOf map[uint32][]*ent.Menu) error {
	_, byIDAfter, _, err := r.buildMenuFullPathIndex(ctx, r.entClient.Client())
	if err != nil {
		return err
	}

	rewrites := make(map[string]string, len(byIDAfter))
	for _, id := range collectMenuSubtreeIDs(childrenOf, nodeID) {
		oldPath, newPath := byIDBefore[id], byIDAfter[id]
		if oldPath == "" || newPath == "" || oldPath == newPath {
			continue
		}
		rewrites[oldPath] = newPath
	}
	if len(rewrites) == 0 {
		return nil
	}
	return r.permissionMenu.UpdateFullPaths(ctx, rewrites)
}

func (r *MenuRepo) updateMetaField(builder *ent.MenuUpdate, meta *permissionV1.MenuMeta, metaPaths []string) {
	//builder.SetMeta(meta)

	// 删除空值
	nullUpdater := entgoUpdate.SetJsonFieldValueUpdateBuilder(menu.FieldMeta, meta, metaPaths, false)
	if nullUpdater != nil {
		builder.Modify(nullUpdater)
	}
	// 更新字段
	setUpdater := entgoUpdate.SetJsonNullFieldUpdateBuilder(menu.FieldMeta, meta, metaPaths)
	if setUpdater != nil {
		builder.Modify(setUpdater)
	}
}

// Truncate 清空菜单表数据
// Clear all menu data from the table
func (r *MenuRepo) Truncate(ctx context.Context) error {
	if _, err := r.entClient.Client().Menu.Delete().Exec(ctx); err != nil {
		r.log.Errorf(ctx, "failed to truncate menus table: %s", err.Error())
		return permissionV1.ErrorInternalServerError("truncate menus failed")
	}
	return nil
}

// SyncMenus 事务化菜单同步，两种模式：
//   - REPLACE（缺省，兼容旧客户端）：清空现有菜单后按传入树重建，菜单 ID 全部变化，角色-菜单授权失效；
//   - MERGE：按全路径匹配，已存在则原位更新（保留 ID 与角色授权），缺失才新增；数据库多出的菜单保留不动。
//     更新不触碰 status：管理端手工停用的菜单不会被同步重新启用。
//
// 整个同步在单个事务内执行，任一步失败整体回滚，不会把菜单表留在半空状态。
func (r *MenuRepo) SyncMenus(ctx context.Context, items []*permissionV1.Menu, mode permissionV1.SyncMenusRequest_Mode, operatorId uint32) (count int, err error) {
	tx, err := r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf(ctx, "start transaction failed: %s", err.Error())
		return 0, permissionV1.ErrorInternalServerError("start transaction failed")
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

	client := tx.Client()

	if mode == permissionV1.SyncMenusRequest_MERGE {
		existing, mapErr := r.buildMenuFullPathMap(ctx, client)
		if mapErr != nil {
			return 0, mapErr
		}
		return r.syncMergeTree(ctx, client, items, nil, "", existing, operatorId)
	}

	if _, err = client.Menu.Delete().Exec(ctx); err != nil {
		r.log.Errorf(ctx, "truncate menus failed: %s", err.Error())
		return 0, permissionV1.ErrorInternalServerError("truncate menus failed")
	}
	return r.syncInsertTree(ctx, client, items, nil, operatorId)
}

// syncInsertTree 递归插入菜单树：先插父节点拿 ID，再递归子节点（REPLACE 模式）
func (r *MenuRepo) syncInsertTree(ctx context.Context, client *ent.Client, items []*permissionV1.Menu, parentID *uint32, operatorId uint32) (int, error) {
	count := 0
	for _, m := range items {
		if m == nil {
			continue
		}

		dto := proto.Clone(m).(*permissionV1.Menu)
		dto.Children = nil
		dto.Id = nil
		if parentID != nil && *parentID > 0 {
			dto.ParentId = parentID
		} else {
			dto.ParentId = nil
		}
		dto.Module = moduleForComponent(m.GetComponent())
		dto.CreatedBy = trans.Ptr(operatorId)
		dto.UpdatedBy = nil

		created, err := r.createMenuReturn(ctx, client, dto)
		if err != nil {
			return count, err
		}
		count++

		if len(m.Children) > 0 {
			childCount, err := r.syncInsertTree(ctx, client, m.Children, created.Id, operatorId)
			if err != nil {
				return count, err
			}
			count += childCount
		}
	}
	return count, nil
}

// syncMergeTree 增量合并：按全路径匹配，已存在则原位更新（保 ID、不动 status），缺失才新增
func (r *MenuRepo) syncMergeTree(
	ctx context.Context,
	client *ent.Client,
	items []*permissionV1.Menu,
	parentID *uint32,
	parentFullPath string,
	existing map[string]*ent.Menu,
	operatorId uint32,
) (int, error) {
	count := 0
	for _, m := range items {
		if m == nil {
			continue
		}

		fullPath := joinMenuFullPath(parentFullPath, m.GetPath())
		if fullPath == "" {
			continue
		}

		exist := existing[fullPath]
		var rowID uint32
		if exist != nil {
			builder := client.Menu.Update().
				Where(menu.IDEQ(exist.ID)).
				SetNillableType(r.typeConverter.ToEntity(m.Type)).
				SetNillablePath(m.Path).
				SetNillableRedirect(m.Redirect).
				SetNillableComponent(m.Component).
				SetNillableUpdatedBy(trans.Ptr(operatorId)).
				SetUpdatedAt(time.Now())
			if parentID != nil && *parentID > 0 {
				builder.SetParentID(*parentID)
			} else {
				builder.ClearParentID()
			}
			if module := moduleForComponent(m.GetComponent()); module != nil {
				builder.SetNillableModule(r.moduleConverter.ToEntity(module))
			} else {
				builder.ClearModule()
			}
			if m.Meta != nil {
				builder.SetMeta(m.Meta)
			}
			if err := builder.Exec(ctx); err != nil {
				r.log.Errorf(ctx, "merge update menu failed, path: %s, err: %s", fullPath, err.Error())
				return count, permissionV1.ErrorInternalServerError("merge update menu failed")
			}
			rowID = exist.ID
		} else {
			dto := proto.Clone(m).(*permissionV1.Menu)
			dto.Children = nil
			dto.Id = nil
			if parentID != nil && *parentID > 0 {
				dto.ParentId = parentID
			} else {
				dto.ParentId = nil
			}
			dto.Module = moduleForComponent(m.GetComponent())
			dto.CreatedBy = trans.Ptr(operatorId)
			dto.UpdatedBy = nil

			created, err := r.createMenuReturn(ctx, client, dto)
			if err != nil {
				return count, err
			}
			rowID = created.GetId()
		}
		count++

		if len(m.Children) > 0 {
			childCount, err := r.syncMergeTree(ctx, client, m.Children, &rowID, fullPath, existing, operatorId)
			if err != nil {
				return count, err
			}
			count += childCount
		}
	}
	return count, nil
}

// joinMenuFullPath 计算菜单全路径：子节点相对路径拼在父全路径后，绝对路径子节点按绝对处理
func joinMenuFullPath(parentFullPath, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return parentFullPath
	}
	if strings.HasPrefix(path, "/") {
		return path
	}
	if parentFullPath == "" || parentFullPath == "/" {
		return "/" + path
	}
	return strings.TrimRight(parentFullPath, "/") + "/" + path
}

// buildMenuFullPathMap 加载现有菜单并建立 全路径 → 实体 映射（MERGE 匹配用）
func (r *MenuRepo) buildMenuFullPathMap(ctx context.Context, client *ent.Client) (map[string]*ent.Menu, error) {
	byPath, _, _, err := r.buildMenuFullPathIndex(ctx, client)
	return byPath, err
}

// buildMenuFullPathIndex 加载现有菜单做一次树遍历，同时产出身份键索引与树形态：
// 全路径 → 实体（MERGE 匹配/反查菜单行用）、id → 全路径（绑定快照解析/改写用）、
// parent 链 → 子节点表（子树内存收集用——代替 UNION ALL 递归 CTE，存量环脏数据
// 下 visited 防护正常收敛，不会打转挂死）。
// 全路径是菜单的身份键：菜单同步 MERGE 按它保号，权限-菜单授权绑定存它的
// 快照（sys_permission_menus.full_path）。
func (r *MenuRepo) buildMenuFullPathIndex(ctx context.Context, client *ent.Client) (byPath map[string]*ent.Menu, byID map[uint32]string, childrenOf map[uint32][]*ent.Menu, err error) {
	rows, err := client.Menu.Query().All(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query existing menus failed: %s", err.Error())
		return nil, nil, nil, permissionV1.ErrorInternalServerError("query existing menus failed")
	}

	childrenOf = make(map[uint32][]*ent.Menu, len(rows))
	var roots []*ent.Menu
	for _, row := range rows {
		if row.ParentID == nil || *row.ParentID == 0 {
			roots = append(roots, row)
		} else {
			childrenOf[*row.ParentID] = append(childrenOf[*row.ParentID], row)
		}
	}

	byPath = make(map[string]*ent.Menu, len(rows))
	byID = make(map[uint32]string, len(rows))
	visited := make(map[uint32]bool, len(rows))
	var walk func(rows []*ent.Menu, parentFullPath string)
	walk = func(rows []*ent.Menu, parentFullPath string) {
		for _, row := range rows {
			if visited[row.ID] {
				continue // 脏数据成环防护
			}
			visited[row.ID] = true
			fullPath := joinMenuFullPath(parentFullPath, derefStrP(row.Path))
			byPath[fullPath] = row
			byID[row.ID] = fullPath
			walk(childrenOf[row.ID], fullPath)
		}
	}
	walk(roots, "")

	return byPath, byID, childrenOf, nil
}

// collectMenuSubtreeIDs 按 parent 链内存收集 rootID 及其全部后代（含自身）。
func collectMenuSubtreeIDs(childrenOf map[uint32][]*ent.Menu, rootID uint32) []uint32 {
	return collectTreeIDs(childrenOf, func(m *ent.Menu) uint32 { return m.ID }, rootID)
}

// ResolveMenuIDsByFullPaths 把授权绑定的菜单全路径快照反解回当前 sys_menus 的 id，
// 供导航/回显链路使用。路径已不存在的快照（菜单被删除且无同路径重建）解析不出
// id，防御性跳过。
func (r *MenuRepo) ResolveMenuIDsByFullPaths(ctx context.Context, fullPaths []string) ([]uint32, error) {
	if len(fullPaths) == 0 {
		return nil, nil
	}
	_, byID, _, err := r.buildMenuFullPathIndex(ctx, r.entClient.Client())
	if err != nil {
		return nil, err
	}
	// 索引是 id→path，反着用：path→id
	pathToID := make(map[string]uint32, len(byID))
	for id, p := range byID {
		pathToID[p] = id
	}
	ids := make([]uint32, 0, len(fullPaths))
	seen := make(map[uint32]struct{}, len(fullPaths))
	for _, p := range fullPaths {
		if id, ok := pathToID[p]; ok {
			if _, dup := seen[id]; dup {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// ListAllIDs 列出全部菜单行 id（模板权限播种期收敛引用用；菜单规模有界，
// 全量拉取无分页必要）。
func (r *MenuRepo) ListAllIDs(ctx context.Context) ([]uint32, error) {
	intIDs, err := r.entClient.Client().Menu.Query().
		Select(menu.FieldID).
		Ints(ctx)
	if err != nil {
		r.log.Errorf(ctx, "list all menu ids failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("list all menu ids failed")
	}
	ids := make([]uint32, len(intIDs))
	for i, v := range intIDs {
		ids[i] = uint32(v)
	}
	return ids, nil
}

// moduleForComponent 由组件路径归类业务模块；未声明组件或无法归类时返回 nil（等价不写）
func moduleForComponent(component string) *identityV1.Module {
	if component == "" {
		return nil
	}
	mod := constants.ComponentToModule(component)
	if mod == identityV1.Module_MODULE_UNSPECIFIED {
		return nil
	}
	return &mod
}

func (r *MenuRepo) Delete(ctx context.Context, req *permissionV1.DeleteMenuRequest) error {
	if req == nil {
		return permissionV1.ErrorBadRequest("invalid parameter")
	}

	// 删除前取整棵子树：id 按 parent 链内存收集（visited 防护，存量环脏数据
	// 下正常收敛——QueryAllChildrenIds 的 UNION ALL 递归 CTE 遇环数据会无限
	// 递归挂死查询，不再走它），全路径快照用于级联清理授权绑定。
	_, byID, childrenOf, err := r.buildMenuFullPathIndex(ctx, r.entClient.Client())
	if err != nil {
		return err
	}
	childrenIds := collectMenuSubtreeIDs(childrenOf, req.GetId())

	deadPaths := make([]string, 0, len(childrenIds))
	for _, id := range childrenIds {
		if p, ok := byID[id]; ok && p != "" {
			deadPaths = append(deadPaths, p)
		}
	}

	var ids []any
	for _, id := range childrenIds {
		ids = append(ids, id)
	}

	// 行删除与绑定级联同一事务：删了行清不掉绑定会留悬空授权，反之亦然
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

	builder := tx.Menu.Delete()

	_, err = r.repository.Delete(ctx, builder, func(s *sql.Selector) {
		s.Where(sql.In(menu.FieldID, ids...))
	})
	if err != nil {
		r.log.Errorf(ctx, "delete menu failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete menu failed")
	}

	// 级联清绑定（路径没了 = 授权该跟着没；同路径重建走重新授权，明账）
	if len(deadPaths) > 0 && r.permissionMenu != nil {
		if err = r.permissionMenu.DeleteByFullPaths(ctx, tx.Client(), deadPaths); err != nil {
			return err
		}
	}

	return nil
}
