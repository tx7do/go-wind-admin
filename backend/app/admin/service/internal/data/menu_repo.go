package data

import (
	"context"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"
	entgoUpdate "github.com/tx7do/go-crud/entgo/update"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/menu"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	resourceV1 "go-wind-admin/api/gen/go/resource/service/v1"
)

type MenuRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper          *mapper.CopierMapper[resourceV1.Menu, ent.Menu]
	statusConverter *mapper.EnumTypeConverter[resourceV1.Menu_Status, menu.Status]
	typeConverter   *mapper.EnumTypeConverter[resourceV1.Menu_Type, menu.Type]

	repository *entCrud.Repository[
		ent.MenuQuery, ent.MenuSelect,
		ent.MenuCreate, ent.MenuCreateBulk,
		ent.MenuUpdate, ent.MenuUpdateOne,
		ent.MenuDelete,
		predicate.Menu,
		resourceV1.Menu, ent.Menu,
	]
}

func NewMenuRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *MenuRepo {
	repo := &MenuRepo{
		log:             ctx.NewLoggerHelper("menu/repo/admin-service"),
		entClient:       entClient,
		mapper:          mapper.NewCopierMapper[resourceV1.Menu, ent.Menu](),
		statusConverter: mapper.NewEnumTypeConverter[resourceV1.Menu_Status, menu.Status](resourceV1.Menu_Status_name, resourceV1.Menu_Status_value),
		typeConverter:   mapper.NewEnumTypeConverter[resourceV1.Menu_Type, menu.Type](resourceV1.Menu_Type_name, resourceV1.Menu_Type_value),
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
		resourceV1.Menu, ent.Menu,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
	r.mapper.AppendConverters(r.typeConverter.NewConverterPair())
}

func (r *MenuRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().Menu.Query()
	if len(whereCond) != 0 {
		builder.Modify(whereCond...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, resourceV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *MenuRepo) List(ctx context.Context, req *paginationV1.PagingRequest, treeTravel bool) (*resourceV1.ListMenuResponse, error) {
	if req == nil {
		return nil, resourceV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Menu.Query()

	whereSelectors, _, err := r.repository.BuildListSelectorWithPaging(builder, req)
	if err != nil {
		r.log.Errorf("parse list param error [%s]", err.Error())
		return nil, resourceV1.ErrorBadRequest("invalid query parameter")
	}

	entities, err := builder.All(ctx)
	if err != nil {
		r.log.Errorf("query menu list failed: %s", err.Error())
		return nil, resourceV1.ErrorInternalServerError("query menu list failed")
	}

	dtos := make([]*resourceV1.Menu, 0, len(entities))
	if treeTravel {
		for _, entity := range entities {
			if entity.ParentID == nil {
				dto := r.mapper.ToDTO(entity)
				dtos = append(dtos, dto)
			}
		}
		for _, entity := range entities {
			if entity.ParentID != nil {
				dto := r.mapper.ToDTO(entity)

				if entCrud.TravelChild(&dtos, dto, func(parent *resourceV1.Menu, node *resourceV1.Menu) {
					parent.Children = append(parent.Children, node)
				}) {
					continue
				}

				dtos = append(dtos, dto)
			}
		}
	} else {
		for _, entity := range entities {
			dto := r.mapper.ToDTO(entity)
			dtos = append(dtos, dto)
		}
	}

	count, err := r.Count(ctx, whereSelectors)
	if err != nil {
		return nil, err
	}

	return &resourceV1.ListMenuResponse{
		Total: uint64(count),
		Items: dtos,
	}, nil
}

func (r *MenuRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().Menu.Query().
		Where(menu.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, resourceV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *MenuRepo) Get(ctx context.Context, req *resourceV1.GetMenuRequest) (*resourceV1.Menu, error) {
	if req == nil {
		return nil, resourceV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Menu.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *resourceV1.GetMenuRequest_Id:
		whereCond = append(whereCond, menu.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

func (r *MenuRepo) Create(ctx context.Context, req *resourceV1.CreateMenuRequest) error {
	if req == nil || req.Data == nil {
		return resourceV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Menu.Create().
		SetNillableParentID(req.Data.ParentId).
		SetNillableType(r.typeConverter.ToEntity(req.Data.Type)).
		SetNillablePath(req.Data.Path).
		SetNillableRedirect(req.Data.Redirect).
		SetNillableAlias(req.Data.Alias).
		SetNillableName(req.Data.Name).
		SetNillableComponent(req.Data.Component).
		SetNillableStatus(r.statusConverter.ToEntity(req.Data.Status)).
		SetNillableCreatedBy(req.Data.CreatedBy).
		SetCreatedAt(time.Now())

	if req.Data.Meta != nil {
		builder.SetMeta(req.Data.Meta)
	}

	if req.Data.Id != nil {
		builder.SetID(req.GetData().GetId())
	}

	if err := builder.Exec(ctx); err != nil {
		r.log.Errorf("insert menu failed: %s", err.Error())
		return resourceV1.ErrorInternalServerError("insert menu failed")
	}

	return nil
}

func (r *MenuRepo) Update(ctx context.Context, req *resourceV1.UpdateMenuRequest) error {
	if req == nil || req.Data == nil {
		return resourceV1.ErrorBadRequest("invalid parameter")
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return err
		}
		if !exist {
			createReq := &resourceV1.CreateMenuRequest{Data: req.Data}
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

	builder := r.entClient.Client().Debug().Menu.Update()
	err := r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *resourceV1.Menu) {
			builder.
				SetNillableParentID(req.Data.ParentId).
				SetNillableType(r.typeConverter.ToEntity(req.Data.Type)).
				SetNillablePath(req.Data.Path).
				SetNillableRedirect(req.Data.Redirect).
				SetNillableAlias(req.Data.Alias).
				SetNillableName(req.Data.Name).
				SetNillableComponent(req.Data.Component).
				SetNillableStatus(r.statusConverter.ToEntity(req.Data.Status)).
				SetNillableUpdatedBy(req.Data.UpdatedBy).
				SetUpdatedAt(time.Now())

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

func (r *MenuRepo) updateMetaField(builder *ent.MenuUpdate, meta *resourceV1.MenuMeta, metaPaths []string) {
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

func (r *MenuRepo) Delete(ctx context.Context, req *resourceV1.DeleteMenuRequest) error {
	if req == nil {
		return resourceV1.ErrorBadRequest("invalid parameter")
	}

	ids, err := entCrud.QueryAllChildrenIds(ctx, r.entClient, "sys_menus", req.GetId())
	if err != nil {
		r.log.Errorf("query child menus failed: %s", err.Error())
		return resourceV1.ErrorInternalServerError("query child menus failed")
	}
	ids = append(ids, req.GetId())

	//r.log.Info("menu ids to delete: ", ids)

	builder := r.entClient.Client().Debug().Menu.Delete()

	_, err = r.repository.Delete(ctx, builder, func(s *sql.Selector) {
		s.Where(sql.In(menu.FieldID, ids))
	})
	if err != nil {
		r.log.Errorf("delete menu failed: %s", err.Error())
		return resourceV1.ErrorInternalServerError("delete menu failed")
	}

	return nil
}
