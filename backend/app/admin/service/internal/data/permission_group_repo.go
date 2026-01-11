package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	pagination "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/timeutil"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/permissiongroup"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"

	"go-wind-admin/pkg/constants"
)

type PermissionGroupRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper          *mapper.CopierMapper[permissionV1.PermissionGroup, ent.PermissionGroup]
	statusConverter *mapper.EnumTypeConverter[permissionV1.PermissionGroup_Status, permissiongroup.Status]

	repository *entCrud.Repository[
		ent.PermissionGroupQuery, ent.PermissionGroupSelect,
		ent.PermissionGroupCreate, ent.PermissionGroupCreateBulk,
		ent.PermissionGroupUpdate, ent.PermissionGroupUpdateOne,
		ent.PermissionGroupDelete,
		predicate.PermissionGroup,
		permissionV1.PermissionGroup, ent.PermissionGroup,
	]
}

func NewPermissionGroupRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) *PermissionGroupRepo {
	repo := &PermissionGroupRepo{
		log:       ctx.NewLoggerHelper("permission-group/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[permissionV1.PermissionGroup, ent.PermissionGroup](),
		statusConverter: mapper.NewEnumTypeConverter[permissionV1.PermissionGroup_Status, permissiongroup.Status](
			permissionV1.PermissionGroup_Status_name, permissionV1.PermissionGroup_Status_value,
		),
	}

	repo.init()

	return repo
}

func (r *PermissionGroupRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.PermissionGroupQuery, ent.PermissionGroupSelect,
		ent.PermissionGroupCreate, ent.PermissionGroupCreateBulk,
		ent.PermissionGroupUpdate, ent.PermissionGroupUpdateOne,
		ent.PermissionGroupDelete,
		predicate.PermissionGroup,
		permissionV1.PermissionGroup, ent.PermissionGroup,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
}

func (r *PermissionGroupRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().PermissionGroup.Query()
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

func (r *PermissionGroupRepo) List(ctx context.Context, req *pagination.PagingRequest, treeTravel bool) (*permissionV1.ListPermissionGroupResponse, error) {
	if req == nil {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().PermissionGroup.Query()

	whereSelectors, _, err := r.repository.BuildListSelectorWithPaging(builder, req)
	if err != nil {
		r.log.Errorf("parse list param error [%s]", err.Error())
		return nil, permissionV1.ErrorBadRequest("invalid query parameter")
	}

	entities, err := builder.All(ctx)
	if err != nil {
		r.log.Errorf("query list failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query list failed")
	}

	dtos := make([]*permissionV1.PermissionGroup, 0, len(entities))
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

				if entCrud.TravelChild(&dtos, dto, func(parent *permissionV1.PermissionGroup, node *permissionV1.PermissionGroup) {
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

	return &permissionV1.ListPermissionGroupResponse{
		Total: uint64(count),
		Items: dtos,
	}, nil
}

func (r *PermissionGroupRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().PermissionGroup.Query().
		Where(permissiongroup.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, permissionV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *PermissionGroupRepo) Get(ctx context.Context, req *permissionV1.GetPermissionGroupRequest) (*permissionV1.PermissionGroup, error) {
	if req == nil {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().PermissionGroup.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *permissionV1.GetPermissionGroupRequest_Id:
		whereCond = append(whereCond, permissiongroup.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

// Create 创建 Permission
func (r *PermissionGroupRepo) Create(ctx context.Context, req *permissionV1.CreatePermissionGroupRequest) (*permissionV1.PermissionGroup, error) {
	if req == nil || req.Data == nil {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.newPermissionCreate(req.Data)

	var err error
	var entity *ent.PermissionGroup
	if entity, err = builder.Save(ctx); err != nil {
		r.log.Errorf("insert permission group failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("insert permission group failed")
	}

	dto := r.mapper.ToDTO(entity)

	return dto, nil
}

// BatchCreate 批量创建 Permission
func (r *PermissionGroupRepo) BatchCreate(ctx context.Context, permissionGroups []*permissionV1.PermissionGroup) (dtos []*permissionV1.PermissionGroup, err error) {
	if len(permissionGroups) == 0 {
		return nil, permissionV1.ErrorBadRequest("invalid parameter")
	}

	var permissionGroupCreates []*ent.PermissionGroupCreate
	for _, perm := range permissionGroups {
		pc := r.newPermissionCreate(perm)
		permissionGroupCreates = append(permissionGroupCreates, pc)
	}

	builder := r.entClient.Client().PermissionGroup.CreateBulk(permissionGroupCreates...)

	var entities []*ent.PermissionGroup
	if entities, err = builder.Save(ctx); err != nil {
		r.log.Errorf("batch insert permission groups failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("batch insert permission groups failed")
	}

	for _, entity := range entities {
		dto := r.mapper.ToDTO(entity)
		dtos = append(dtos, dto)
	}

	return dtos, nil
}

// newPermissionCreate 创建 Permission Create 构造器
func (r *PermissionGroupRepo) newPermissionCreate(permissionGroup *permissionV1.PermissionGroup) *ent.PermissionGroupCreate {
	builder := r.entClient.Client().PermissionGroup.Create().
		SetName(permissionGroup.GetName()).
		SetNillableStatus(r.statusConverter.ToEntity(permissionGroup.Status)).
		SetNillableModule(permissionGroup.Module).
		SetNillableSortOrder(permissionGroup.SortOrder).
		SetNillableDescription(permissionGroup.Description).
		SetNillableParentID(permissionGroup.ParentId).
		SetNillableCreatedBy(permissionGroup.CreatedBy).
		SetNillableCreatedAt(timeutil.TimestamppbToTime(permissionGroup.CreatedAt))

	if permissionGroup.CreatedAt == nil {
		builder.SetCreatedAt(time.Now())
	}

	if permissionGroup.Id != nil {
		builder.SetID(permissionGroup.GetId())
	}

	return builder
}

// Update 更新 Permission
func (r *PermissionGroupRepo) Update(ctx context.Context, req *permissionV1.UpdatePermissionGroupRequest) error {
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
			createReq := &permissionV1.CreatePermissionGroupRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			_, err = r.Create(ctx, createReq)
			return err
		}
	}

	builder := r.entClient.Client().PermissionGroup.UpdateOneID(req.GetId())
	_, err := r.repository.UpdateOne(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *permissionV1.PermissionGroup) {
			builder.
				SetNillableName(req.Data.Name).
				SetNillableStatus(r.statusConverter.ToEntity(req.Data.Status)).
				SetNillableModule(req.Data.Module).
				SetNillableSortOrder(req.Data.SortOrder).
				SetNillableDescription(req.Data.Description).
				SetNillableParentID(req.Data.ParentId).
				SetNillableUpdatedBy(req.Data.UpdatedBy).
				SetNillableUpdatedAt(timeutil.TimestamppbToTime(req.Data.UpdatedAt))

			if req.Data.UpdatedAt == nil {
				builder.SetUpdatedAt(time.Now())
			}
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(permissiongroup.FieldID, req.GetId()))
		},
	)
	if err != nil {
		return err
	}

	return nil
}

// UpdateParentIDs 更新 Permission ParentID
func (r *PermissionGroupRepo) UpdateParentIDs(ctx context.Context, parentIDs map[uint32]uint32) (err error) {
	if len(parentIDs) == 0 {
		return nil
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

	for permID, parentID := range parentIDs {
		builder := tx.PermissionGroup.Update().
			SetParentID(parentID).
			Where(permissiongroup.IDEQ(permID))

		if err = builder.Exec(ctx); err != nil {
			r.log.Errorf("update permission parent_id failed: %s", err.Error())
			return permissionV1.ErrorInternalServerError("update permission parent_id failed")
		}
	}

	return nil
}

// Delete 删除 Permission
func (r *PermissionGroupRepo) Delete(ctx context.Context, req *permissionV1.DeletePermissionGroupRequest) error {
	if req == nil {
		return permissionV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().PermissionGroup.Delete()

	_, err := r.repository.Delete(ctx, builder, func(s *sql.Selector) {
		s.Where(sql.EQ(permissiongroup.FieldID, req.GetId()))
	})
	if err != nil {
		r.log.Errorf("delete permission group failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete permission group failed")
	}

	return nil
}

// Truncate 清空表数据
func (r *PermissionGroupRepo) Truncate(ctx context.Context) error {
	if _, err := r.entClient.Client().PermissionGroup.Delete().Exec(ctx); err != nil {
		r.log.Errorf("failed to truncate permission group table: %s", err.Error())
		return permissionV1.ErrorInternalServerError("truncate failed")
	}

	return nil
}

// TruncateBizGroup 清空业务表数据，保留系统内置数据
func (r *PermissionGroupRepo) TruncateBizGroup(ctx context.Context) error {
	builder := r.entClient.Client().PermissionGroup.Delete().
		Where(
			permissiongroup.ModuleNotIn(constants.SystemPermissionModule),
		)

	if _, err := builder.Exec(ctx); err != nil {
		r.log.Errorf("failed to truncate permission group table: %s", err.Error())
		return permissionV1.ErrorInternalServerError("truncate failed")
	}

	return nil
}

func (r *PermissionGroupRepo) ListByIDs(ctx context.Context, ids []uint32) ([]*permissionV1.PermissionGroup, error) {
	if len(ids) == 0 {
		return []*permissionV1.PermissionGroup{}, nil
	}

	builder := r.entClient.Client().PermissionGroup.Query().
		Where(permissiongroup.IDIn(ids...))

	entities, err := builder.All(ctx)
	if err != nil {
		r.log.Errorf("query list by ids failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query list by ids failed")
	}

	dtos := make([]*permissionV1.PermissionGroup, 0, len(entities))
	for _, entity := range entities {
		dto := r.mapper.ToDTO(entity)
		dtos = append(dtos, dto)
	}

	return dtos, nil
}
