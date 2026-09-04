package data

import (
	"context"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"
	entgoUpdate "github.com/tx7do/go-crud/entgo/update"
	"github.com/tx7do/go-crud/pagination"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/menu"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
)

type MenuRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *bLogger.Helper

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

func NewMenuRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *MenuRepo {
	repo := &MenuRepo{
		log:             ctx.NewLoggerHelper("menu/repo/admin-service"),
		entClient:       entClient,
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

	entity, err := builder.Save(ctx)
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

	var metaPaths []string
	if req.UpdateMask != nil {
		for _, v := range req.UpdateMask.GetPaths() {
			if strings.HasPrefix(v, "meta.") {
				metaPaths = append(metaPaths, strings.SplitAfter(v, "meta.")[1])
			}
		}
	}

	builder := r.entClient.Client().Menu.Update()
	err := r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(),
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

			// parent_id=0 同 Create：按无父级处理，避免外键违约
			if req.Data.ParentId != nil && *req.Data.ParentId > 0 {
				builder.SetParentID(*req.Data.ParentId)
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

	return err
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

func (r *MenuRepo) Delete(ctx context.Context, req *permissionV1.DeleteMenuRequest) error {
	if req == nil {
		return permissionV1.ErrorBadRequest("invalid parameter")
	}

	childrenIds, err := entCrud.QueryAllChildrenIds(ctx, r.entClient, "sys_menus", req.GetId())
	if err != nil {
		r.log.Errorf(ctx, "query child menus failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("query child menus failed")
	}
	childrenIds = append(childrenIds, req.GetId())

	//r.log.Info(ctx, "menu childrenIds to delete: ", childrenIds)

	var ids []any
	for _, id := range childrenIds {
		ids = append(ids, id)
	}

	builder := r.entClient.Client().Menu.Delete()

	_, err = r.repository.Delete(ctx, builder, func(s *sql.Selector) {
		s.Where(sql.In(menu.FieldID, ids...))
	})
	if err != nil {
		r.log.Errorf(ctx, "delete menu failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete menu failed")
	}

	return nil
}
