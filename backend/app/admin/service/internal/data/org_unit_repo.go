package data

import (
	"context"
	"sort"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/timeutil"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/orgunit"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	userV1 "go-wind-admin/api/gen/go/user/service/v1"
)

type OrgUnitRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper          *mapper.CopierMapper[userV1.OrgUnit, ent.OrgUnit]
	typeConverter   *mapper.EnumTypeConverter[userV1.OrgUnit_Type, orgunit.Type]
	statusConverter *mapper.EnumTypeConverter[userV1.OrgUnit_Status, orgunit.Status]

	repository *entCrud.Repository[
		ent.OrgUnitQuery, ent.OrgUnitSelect,
		ent.OrgUnitCreate, ent.OrgUnitCreateBulk,
		ent.OrgUnitUpdate, ent.OrgUnitUpdateOne,
		ent.OrgUnitDelete,
		predicate.OrgUnit,
		userV1.OrgUnit, ent.OrgUnit,
	]
}

func NewOrgUnitRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *OrgUnitRepo {
	repo := &OrgUnitRepo{
		log:             ctx.NewLoggerHelper("org-unit/repo/admin-service"),
		entClient:       entClient,
		mapper:          mapper.NewCopierMapper[userV1.OrgUnit, ent.OrgUnit](),
		typeConverter:   mapper.NewEnumTypeConverter[userV1.OrgUnit_Type, orgunit.Type](userV1.OrgUnit_Type_name, userV1.OrgUnit_Type_value),
		statusConverter: mapper.NewEnumTypeConverter[userV1.OrgUnit_Status, orgunit.Status](userV1.OrgUnit_Status_name, userV1.OrgUnit_Status_value),
	}

	repo.init()

	return repo
}

func (r *OrgUnitRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.OrgUnitQuery, ent.OrgUnitSelect,
		ent.OrgUnitCreate, ent.OrgUnitCreateBulk,
		ent.OrgUnitUpdate, ent.OrgUnitUpdateOne,
		ent.OrgUnitDelete,
		predicate.OrgUnit,
		userV1.OrgUnit, ent.OrgUnit,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.typeConverter.NewConverterPair())
	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
}

func (r *OrgUnitRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().OrgUnit.Query()
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

func (r *OrgUnitRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*userV1.ListOrgUnitResponse, error) {
	if req == nil {
		return nil, userV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().OrgUnit.Query()

	whereSelectors, _, err := r.repository.BuildListSelectorWithPaging(builder, req)
	if err != nil {
		r.log.Errorf("parse list param error [%s]", err.Error())
		return nil, userV1.ErrorBadRequest("invalid query parameter")
	}

	entities, err := builder.All(ctx)
	if err != nil {
		r.log.Errorf("query org unit list failed: %s", err.Error())
		return nil, userV1.ErrorInternalServerError("query org unit list failed")
	}

	sort.SliceStable(entities, func(i, j int) bool {
		var sortI, sortJ uint32
		if entities[i].SortOrder != nil {
			sortI = *entities[i].SortOrder
		}
		if entities[j].SortOrder != nil {
			sortJ = *entities[j].SortOrder
		}
		return sortI < sortJ
	})

	dtos := make([]*userV1.OrgUnit, 0, len(entities))
	for _, entity := range entities {
		if entity.ParentID == nil {
			dto := r.mapper.ToDTO(entity)
			dtos = append(dtos, dto)
		}
	}
	for _, entity := range entities {
		if entity.ParentID != nil {
			dto := r.mapper.ToDTO(entity)

			if entCrud.TravelChild(&dtos, dto, func(parent *userV1.OrgUnit, node *userV1.OrgUnit) {
				parent.Children = append(parent.Children, node)
			}) {
				continue
			}

			dtos = append(dtos, dto)
		}
	}

	count, err := r.Count(ctx, whereSelectors)
	if err != nil {
		return nil, err
	}

	return &userV1.ListOrgUnitResponse{
		Total: uint64(count),
		Items: dtos,
	}, err
}

func (r *OrgUnitRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().OrgUnit.Query().
		Where(orgunit.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, userV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *OrgUnitRepo) Get(ctx context.Context, req *userV1.GetOrgUnitRequest) (*userV1.OrgUnit, error) {
	if req == nil {
		return nil, userV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().OrgUnit.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *userV1.GetOrgUnitRequest_Id:
		whereCond = append(whereCond, orgunit.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

// ListOrgUnitsByIds 通过多个ID获取组织列表
func (r *OrgUnitRepo) ListOrgUnitsByIds(ctx context.Context, ids []uint32) ([]*userV1.OrgUnit, error) {
	if len(ids) == 0 {
		return []*userV1.OrgUnit{}, nil
	}

	entities, err := r.entClient.Client().OrgUnit.Query().
		Where(orgunit.IDIn(ids...)).
		All(ctx)
	if err != nil {
		r.log.Errorf("query orgUnit by ids failed: %s", err.Error())
		return nil, userV1.ErrorInternalServerError("query orgUnit by ids failed")
	}

	dtos := make([]*userV1.OrgUnit, 0, len(entities))
	for _, entity := range entities {
		dto := r.mapper.ToDTO(entity)
		dtos = append(dtos, dto)
	}

	return dtos, nil
}

func (r *OrgUnitRepo) Create(ctx context.Context, req *userV1.CreateOrgUnitRequest) (err error) {
	if req == nil || req.Data == nil {
		return userV1.ErrorBadRequest("invalid parameter")
	}

	var tx *ent.Tx
	tx, err = r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf("start transaction failed: %s", err.Error())
		return userV1.ErrorInternalServerError("start transaction failed")
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
			err = userV1.ErrorInternalServerError("transaction commit failed")
		}
	}()

	builder := tx.OrgUnit.Create().
		SetNillableTenantID(req.Data.TenantId).
		SetName(req.Data.GetName()).
		SetNillableCode(req.Data.Code).
		SetNillableStatus(r.statusConverter.ToEntity(req.Data.Status)).
		SetNillableType(r.typeConverter.ToEntity(req.Data.Type)).
		SetNillablePath(req.Data.Path).
		SetNillableParentID(req.Data.ParentId).
		SetNillableSortOrder(req.Data.SortOrder).
		SetNillableLeaderID(req.Data.LeaderId).
		SetNillableDescription(req.Data.Description).
		SetNillableRemark(req.Data.Remark).
		SetNillableExternalID(req.Data.ExternalId).
		SetNillableIsLegalEntity(req.Data.IsLegalEntity).
		SetNillableRegistrationNumber(req.Data.RegistrationNumber).
		SetNillableTaxID(req.Data.TaxId).
		SetNillableLegalEntityOrgID(req.Data.LegalEntityOrgId).
		SetNillableAddress(req.Data.Address).
		SetNillablePhone(req.Data.Phone).
		SetNillableEmail(req.Data.Email).
		SetNillableTimezone(req.Data.Timezone).
		SetNillableCountry(req.Data.Country).
		SetNillableLatitude(req.Data.Latitude).
		SetNillableLongitude(req.Data.Longitude).
		SetNillableStartAt(timeutil.TimestamppbToTime(req.Data.StartAt)).
		SetNillableEndAt(timeutil.TimestamppbToTime(req.Data.EndAt)).
		SetNillableContactUserID(req.Data.ContactUserId).
		SetNillableCreatedBy(req.Data.CreatedBy).
		SetCreatedAt(time.Now())

	if req.Data.BusinessScopes == nil {
		builder.SetBusinessScopes(req.Data.GetBusinessScopes())
	}
	if req.Data.PermissionTags == nil {
		builder.SetPermissionTags(req.Data.GetPermissionTags())
	}

	if req.Data.Id != nil {
		builder.SetID(req.GetData().GetId())
	}

	var entity *ent.OrgUnit
	if entity, err = builder.Save(ctx); err != nil {
		r.log.Errorf("insert org unit failed: %s", err.Error())
		return userV1.ErrorInternalServerError("insert org unit failed")
	}

	if err = r.setTreePath(ctx, tx, entity); err != nil {
		return err
	}

	return nil
}

func (r *OrgUnitRepo) Update(ctx context.Context, req *userV1.UpdateOrgUnitRequest) error {
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
			createReq := &userV1.CreateOrgUnitRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	builder := r.entClient.Client().Debug().OrgUnit.Update()
	err := r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *userV1.OrgUnit) {
			builder.
				SetNillableName(req.Data.Name).
				SetNillableCode(req.Data.Code).
				SetNillableStatus(r.statusConverter.ToEntity(req.Data.Status)).
				SetNillableType(r.typeConverter.ToEntity(req.Data.Type)).
				SetNillablePath(req.Data.Path).
				SetNillableParentID(req.Data.ParentId).
				SetNillableSortOrder(req.Data.SortOrder).
				SetNillableLeaderID(req.Data.LeaderId).
				SetNillableDescription(req.Data.Description).
				SetNillableRemark(req.Data.Remark).
				SetNillableExternalID(req.Data.ExternalId).
				SetNillableIsLegalEntity(req.Data.IsLegalEntity).
				SetNillableRegistrationNumber(req.Data.RegistrationNumber).
				SetNillableTaxID(req.Data.TaxId).
				SetNillableLegalEntityOrgID(req.Data.LegalEntityOrgId).
				SetNillableAddress(req.Data.Address).
				SetNillablePhone(req.Data.Phone).
				SetNillableEmail(req.Data.Email).
				SetNillableTimezone(req.Data.Timezone).
				SetNillableCountry(req.Data.Country).
				SetNillableLatitude(req.Data.Latitude).
				SetNillableLongitude(req.Data.Longitude).
				SetNillableStartAt(timeutil.TimestamppbToTime(req.Data.StartAt)).
				SetNillableEndAt(timeutil.TimestamppbToTime(req.Data.EndAt)).
				SetNillableContactUserID(req.Data.ContactUserId).
				SetNillableUpdatedBy(req.Data.UpdatedBy).
				SetUpdatedAt(time.Now())

			if req.Data.BusinessScopes == nil {
				builder.SetBusinessScopes(req.Data.GetBusinessScopes())
			}
			if req.Data.PermissionTags == nil {
				builder.SetPermissionTags(req.Data.GetPermissionTags())
			}
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(orgunit.FieldID, req.GetId()))
		},
	)

	return err
}

func (r *OrgUnitRepo) Delete(ctx context.Context, req *userV1.DeleteOrgUnitRequest) error {
	if req == nil {
		return userV1.ErrorBadRequest("invalid parameter")
	}

	ids, err := entCrud.QueryAllChildrenIds(ctx, r.entClient, "sys_org_units", req.GetId())
	if err != nil {
		r.log.Errorf("query child orgUnits failed: %s", err.Error())
		return userV1.ErrorInternalServerError("query child orgUnits failed")
	}
	ids = append(ids, req.GetId())

	//r.log.Info("orgunits ids to delete: ", ids)

	builder := r.entClient.Client().Debug().OrgUnit.Delete()

	_, err = r.repository.Delete(ctx, builder, func(s *sql.Selector) {
		s.Where(sql.In(orgunit.FieldID, ids))
	})
	if err != nil {
		r.log.Errorf("delete orgUnit failed: %s", err.Error())
		return userV1.ErrorInternalServerError("delete orgUnit failed")
	}

	return nil
}

func (r *OrgUnitRepo) setTreePath(ctx context.Context, tx *ent.Tx, entity *ent.OrgUnit) (err error) {
	var parentPath string
	if entity.ParentID != nil {
		var parentEntity *ent.OrgUnit
		parentEntity, err = tx.OrgUnit.Query().
			Where(
				orgunit.IDEQ(*entity.ParentID),
			).
			Select(orgunit.FieldPath).
			Only(ctx)
		if err != nil {
			return err
		} else {
			if parentEntity.Path != nil {
				parentPath = *parentEntity.Path
			}
		}
	}
	err = tx.OrgUnit.UpdateOneID(entity.ID).
		SetPath(entCrud.ComputeTreePath(parentPath, entity.ID)).
		Exec(ctx)

	return err
}
