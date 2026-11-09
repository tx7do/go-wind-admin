package data

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/go-crud/pagination"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/timeutil"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/orgunit"
	"go-wind-admin/app/admin/service/internal/data/ent/position"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
)

type OrgUnitRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *bLogger.Helper

	userOrgUnitRepo *UserOrgUnitRepo
	roleOrgUnitRepo *RoleOrgUnitRepo

	mapper          *mapper.CopierMapper[identityV1.OrgUnit, ent.OrgUnit]
	typeConverter   *mapper.EnumTypeConverter[identityV1.OrgUnit_Type, orgunit.Type]
	statusConverter *mapper.EnumTypeConverter[identityV1.OrgUnit_Status, orgunit.Status]

	repository *entCrud.Repository[
		ent.OrgUnitQuery, ent.OrgUnitSelect,
		ent.OrgUnitCreate, ent.OrgUnitCreateBulk,
		ent.OrgUnitUpdate, ent.OrgUnitUpdateOne,
		ent.OrgUnitDelete,
		predicate.OrgUnit,
		identityV1.OrgUnit, ent.OrgUnit,
	]
}

func NewOrgUnitRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client], userOrgUnitRepo *UserOrgUnitRepo, roleOrgUnitRepo *RoleOrgUnitRepo) *OrgUnitRepo {
	repo := &OrgUnitRepo{
		log:             ctx.NewLoggerHelper("org-unit/repo/admin-service"),
		entClient:       entClient,
		userOrgUnitRepo: userOrgUnitRepo,
		roleOrgUnitRepo: roleOrgUnitRepo,
		mapper:          mapper.NewCopierMapper[identityV1.OrgUnit, ent.OrgUnit](),
		typeConverter:   mapper.NewEnumTypeConverter[identityV1.OrgUnit_Type, orgunit.Type](identityV1.OrgUnit_Type_name, identityV1.OrgUnit_Type_value),
		statusConverter: mapper.NewEnumTypeConverter[identityV1.OrgUnit_Status, orgunit.Status](identityV1.OrgUnit_Status_name, identityV1.OrgUnit_Status_value),
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
		identityV1.OrgUnit, ent.OrgUnit,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.typeConverter.NewConverterPair())
	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
}

func (r *OrgUnitRepo) count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().OrgUnit.Query()
	if len(whereCond) != 0 {
		builder.Modify(whereCond...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query count failed: %s", err.Error())
		return 0, identityV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *OrgUnitRepo) Count(ctx context.Context, req *paginationV1.PagingRequest) (int, error) {
	builder := r.entClient.Client().OrgUnit.Query()

	whereSelectors, _, _ := r.repository.BuildListSelectorWithPaging(builder, req)
	if len(whereSelectors) != 0 {
		builder.Modify(whereSelectors...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query org-unit count failed: %s", err.Error())
		return 0, identityV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *OrgUnitRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*identityV1.ListOrgUnitResponse, error) {
	if req == nil {
		return nil, identityV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().OrgUnit.Query()

	whereSelectors, _, err := r.repository.BuildListSelectorWithPaging(builder, req)
	if err != nil {
		r.log.Errorf(ctx, "parse list param error [%s]", err.Error())
		return nil, identityV1.ErrorBadRequest("invalid query parameter")
	}

	entities, err := builder.All(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query org unit list failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query org unit list failed")
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

	// 转换所有实体为 DTO
	dtos := make([]*identityV1.OrgUnit, 0, len(entities))
	for _, entity := range entities {
		dto := r.mapper.ToDTO(entity)
		dtos = append(dtos, dto)
	}

	// 构建树形结构
	dtos = pagination.BuildTree(
		dtos,
		func(node *identityV1.OrgUnit) *uint64 { return node.Id },
		func(node *identityV1.OrgUnit) *uint64 { return node.ParentId },
		func(node *identityV1.OrgUnit) *[]*identityV1.OrgUnit { return &node.Children },
	)

	count, err := r.count(ctx, whereSelectors)
	if err != nil {
		return nil, err
	}

	return &identityV1.ListOrgUnitResponse{
		Total: uint64(count),
		Items: dtos,
	}, err
}

func (r *OrgUnitRepo) IsExist(ctx context.Context, id uint64) (bool, error) {
	exist, err := r.entClient.Client().OrgUnit.Query().
		Where(orgunit.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query exist failed: %s", err.Error())
		return false, identityV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *OrgUnitRepo) Get(ctx context.Context, req *identityV1.GetOrgUnitRequest) (*identityV1.OrgUnit, error) {
	if req == nil {
		return nil, identityV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().OrgUnit.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *identityV1.GetOrgUnitRequest_Id:
		whereCond = append(whereCond, orgunit.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

// ListOrgUnitsByIds 通过多个ID获取组织列表
func (r *OrgUnitRepo) ListOrgUnitsByIds(ctx context.Context, ids []uint64) ([]*identityV1.OrgUnit, error) {
	if len(ids) == 0 {
		return []*identityV1.OrgUnit{}, nil
	}

	entities, err := r.entClient.Client().OrgUnit.Query().
		Where(orgunit.IDIn(ids...)).
		All(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query orgUnit by ids failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query orgUnit by ids failed")
	}

	dtos := make([]*identityV1.OrgUnit, 0, len(entities))
	for _, entity := range entities {
		dto := r.mapper.ToDTO(entity)
		dtos = append(dtos, dto)
	}

	return dtos, nil
}

func (r *OrgUnitRepo) Create(ctx context.Context, req *identityV1.CreateOrgUnitRequest) (err error) {
	if req == nil || req.Data == nil {
		return identityV1.ErrorBadRequest("invalid parameter")
	}

	var tx *ent.Tx
	tx, err = r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf(ctx, "start transaction failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("start transaction failed")
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
			err = identityV1.ErrorInternalServerError("transaction commit failed")
		}
	}()

	builder := tx.OrgUnit.Create().
		SetNillableTenantID(req.Data.TenantId).
		SetName(req.Data.GetName()).
		SetNillableCode(req.Data.Code).
		SetNillableStatus(r.statusConverter.ToEntity(req.Data.Status)).
		SetNillableType(r.typeConverter.ToEntity(req.Data.Type)).
		SetNillablePath(req.Data.Path).
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

	// parent_id=0 与未设置同义（挂根）：自引用 FK 不接受 0，必须跳过不设
	if p := req.Data.ParentId; p != nil && *p > 0 {
		builder.SetParentID(*p)
	}

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
		r.log.Errorf(ctx, "insert org unit failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("insert org unit failed")
	}

	if err = r.setTreePath(ctx, tx, entity); err != nil {
		return err
	}

	return nil
}

func (r *OrgUnitRepo) Update(ctx context.Context, req *identityV1.UpdateOrgUnitRequest) error {
	if req == nil || req.Data == nil {
		return identityV1.ErrorBadRequest("invalid parameter")
	}
	if req.GetId() == 0 {
		return identityV1.ErrorBadRequest("id is required")
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return err
		}
		if !exist {
			createReq := &identityV1.CreateOrgUnitRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	// 换父前置校验：目标父不得是自身或自身后代（成环）。环检测必须发生在
	// 更新落库之前——此前检测在 relocateSubtree（更新后）才做，拒得了请求
	// 却拦不住落库，环真实存在。
	if req.Data.ParentId != nil && *req.Data.ParentId > 0 {
		if err := r.ensureUnitAcyclicParent(ctx, req.GetId(), *req.Data.ParentId); err != nil {
			return err
		}
	}

	builder := r.entClient.Client().OrgUnit.Update()
	err := r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *identityV1.OrgUnit) {
			builder.
				SetNillableName(req.Data.Name).
				SetNillableCode(req.Data.Code).
				SetNillableStatus(r.statusConverter.ToEntity(req.Data.Status)).
				SetNillableType(r.typeConverter.ToEntity(req.Data.Type)).
				SetNillablePath(req.Data.Path).
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

			// 挂载关系三态：未设置（nil）= 不改；显式 >0 = 挂到该父级；显式 0 =
			// 移到根级，落 NULL（自引用 FK 不接受 0）。此前 0 直接写库触发 FK
			// 违约报 500，"移回根级"做不到。
			if p := req.Data.ParentId; p != nil {
				if *p > 0 {
					builder.SetParentID(*p)
				} else {
					builder.ClearParentID()
				}
			}

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
	if err != nil {
		return err
	}

	// parent_id 变更后重算本节点及全部后代的物化路径（path 前缀是数据范围
	// UNIT_AND_CHILD 展开的依据，挂载关系变了路径必须跟着走）。
	if req.Data.ParentId != nil {
		if err = r.relocateSubtree(ctx, req.GetId()); err != nil {
			return err
		}
	}

	return err
}

// ensureUnitAcyclicParent 校验把 nodeID 挂到 newParentID 下不会成环：从新父沿
// parent 链上行，命中 nodeID 即拒绝。visited 防护让存量环脏数据也不死循环。
func (r *OrgUnitRepo) ensureUnitAcyclicParent(ctx context.Context, nodeID, newParentID uint64) error {
	if newParentID == nodeID {
		return identityV1.ErrorBadRequest("cannot move org unit under itself")
	}
	rows, err := r.entClient.Client().OrgUnit.Query().
		Select(orgunit.FieldID, orgunit.FieldParentID).
		All(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query org units for parent acyclicity check failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("query org units failed")
	}
	parentOf := make(map[uint64]uint64, len(rows))
	for _, row := range rows {
		if row.ParentID != nil && *row.ParentID != 0 {
			parentOf[row.ID] = *row.ParentID
		}
	}
	visited := make(map[uint64]struct{}, len(rows))
	cur := newParentID
	for cur != 0 {
		if cur == nodeID {
			return identityV1.ErrorBadRequest("cannot move org unit under its own descendant")
		}
		if _, seen := visited[cur]; seen {
			break // 存量环（与本节点无关）：放行但不死循环
		}
		visited[cur] = struct{}{}
		cur = parentOf[cur]
	}
	return nil
}

// relocateSubtree 以 parent_id 链为准，BFS 重算 node 及其全部后代的物化路径。
// 不按旧 path 前缀扫描：历史数据可能存着脏路径（如全表都是 "/"），按 parent 链
// 走可顺带自愈；移动到自身后代下会成环，检测到即拒绝。
func (r *OrgUnitRepo) relocateSubtree(ctx context.Context, nodeID uint64) error {
	client := r.entClient.Client()

	node, err := client.OrgUnit.Query().
		Where(orgunit.IDEQ(nodeID)).
		Select(orgunit.FieldPath, orgunit.FieldParentID).
		Only(ctx)
	if err != nil {
		r.log.Errorf(ctx, "relocate subtree: query org unit [%d] failed: %s", nodeID, err.Error())
		return identityV1.ErrorInternalServerError("query org unit failed")
	}

	var parentPath string
	if node.ParentID != nil && *node.ParentID != 0 {
		var parent *ent.OrgUnit
		parent, err = client.OrgUnit.Query().
			Where(orgunit.IDEQ(*node.ParentID)).
			Select(orgunit.FieldPath).
			Only(ctx)
		if err != nil {
			r.log.Errorf(ctx, "relocate subtree: query parent org unit [%d] failed: %s", *node.ParentID, err.Error())
			return identityV1.ErrorInternalServerError("query parent org unit failed")
		}
		if parent.Path != nil {
			parentPath = *parent.Path
		}
		if parentPath != "" && strings.Contains(parentPath, "/"+strconv.FormatUint(uint64(nodeID), 10)+"/") {
			return identityV1.ErrorBadRequest("cannot move org unit under its own descendant")
		}
	}

	type queueItem struct {
		id         uint64
		parentPath string
	}
	// visited 防护：存量环脏数据（外部导入/直改库）下 BFS 正常收敛，不会
	// 死循环挂死 goroutine（permission_group 版本同款防线）
	visited := map[uint64]bool{nodeID: true}
	queue := []queueItem{{id: nodeID, parentPath: parentPath}}
	for len(queue) > 0 {
		it := queue[0]
		queue = queue[1:]

		newPath := r.computeUnitTreePath(it.parentPath, it.id)
		cur, cerr := client.OrgUnit.Query().
			Where(orgunit.IDEQ(it.id)).
			Select(orgunit.FieldPath).
			Only(ctx)
		if cerr != nil {
			r.log.Errorf(ctx, "relocate subtree: query org unit [%d] failed: %s", it.id, cerr.Error())
			return identityV1.ErrorInternalServerError("query org unit failed")
		}
		if cur.Path == nil || *cur.Path != newPath {
			if _, uerr := client.OrgUnit.UpdateOneID(it.id).SetPath(newPath).Save(ctx); uerr != nil {
				r.log.Errorf(ctx, "relocate subtree: update org unit [%d] path failed: %s", it.id, uerr.Error())
				return identityV1.ErrorInternalServerError("update org unit path failed")
			}
		}

		children, cerr := client.OrgUnit.Query().
			Where(orgunit.ParentIDEQ(it.id)).
			Select(orgunit.FieldID).
			All(ctx)
		if cerr != nil {
			r.log.Errorf(ctx, "relocate subtree: query children of [%d] failed: %s", it.id, cerr.Error())
			return identityV1.ErrorInternalServerError("query org unit children failed")
		}
		for _, ch := range children {
			if visited[ch.ID] {
				continue
			}
			visited[ch.ID] = true
			queue = append(queue, queueItem{id: ch.ID, parentPath: newPath})
		}
	}

	return nil
}

func (r *OrgUnitRepo) Delete(ctx context.Context, req *identityV1.DeleteOrgUnitRequest) error {
	if req == nil {
		return identityV1.ErrorBadRequest("invalid parameter")
	}

	// 子树 id 按 parent 链内存收集（visited 防护，存量环脏数据正常收敛）。
	// 不走 QueryAllChildrenIds：其递归 CTE 只有 MySQL/Postgres 分支，其余方言
	// 静默返回空集会让删除只删根、整条子树变孤儿。
	rows, err := r.entClient.Client().OrgUnit.Query().
		Select(orgunit.FieldID, orgunit.FieldParentID).
		All(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query org units for subtree collection failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("query org units failed")
	}
	childrenOf := make(map[uint64][]uint64, len(rows))
	for _, row := range rows {
		if row.ParentID != nil && *row.ParentID != 0 {
			childrenOf[*row.ParentID] = append(childrenOf[*row.ParentID], row.ID)
		}
	}
	childrenIds := collectTreeIDs(childrenOf, func(id uint64) uint64 { return id }, req.GetId())

	//r.log.Info(ctx, "orgunits childrenIds to delete: ", childrenIds)

	// 岗位硬性挂在单元上（org_unit_id NOT NULL，且无 DB 外键兜底）：子树内
	// 还有岗位时拒绝删除，避免岗位随级联静默消失或产生悬挂引用。
	// 先删除/转移子树内的岗位，再删单元。
	posCnt, err := r.entClient.Client().Position.Query().
		Where(position.OrgUnitIDIn(childrenIds...)).
		Count(ctx)
	if err != nil {
		r.log.Errorf(ctx, "count positions under org units failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("count positions under org units failed")
	}
	if posCnt > 0 {
		return identityV1.ErrorBadRequest("exist %d positions under the org unit subtree, delete or move them first", posCnt)
	}

	var ids []any
	for _, id := range childrenIds {
		ids = append(ids, id)
	}

	var tx *ent.Tx
	tx, err = r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf(ctx, "start transaction failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("start transaction failed")
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
			err = identityV1.ErrorInternalServerError("transaction commit failed")
		}
	}()

	// 用户↔单元是纯绑定关系：随子树删除清理绑定行，用户本身不受影响
	if err = r.userOrgUnitRepo.CleanRelationsByOrgUnitIDs(ctx, tx, childrenIds); err != nil {
		return err
	}

	// 角色↔单元授权绑定（数据范围 SELECTED_UNITS 档）随子树清理，避免角色
	// 配置里残留指向已删单元的 stale 行（与权限删除级联清角色授权同一教义）
	if err = r.roleOrgUnitRepo.DeleteByOrgUnitIDs(ctx, tx, childrenIds); err != nil {
		return err
	}

	builder := tx.OrgUnit.Delete()

	_, err = r.repository.Delete(ctx, builder, func(s *sql.Selector) {
		s.Where(sql.In(orgunit.FieldID, ids...))
	})
	if err != nil {
		r.log.Errorf(ctx, "delete orgUnit failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("delete orgUnit failed")
	}

	return nil
}

// setTreePath 计算并落库节点的物化路径：根节点 "/ID/"，子孙节点 "/父路径/ID/"。
// 不能直接用 entCrud.ComputeTreePath 落库：它对空父路径返回 "/"，会让所有根节点
// 共享同一前缀，数据范围 UNIT_AND_CHILD 的 path 前缀展开会因此误匹配全表。
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
		SetPath(r.computeUnitTreePath(parentPath, entity.ID)).
		Exec(ctx)

	return err
}

// computeUnitTreePath 物化路径：根节点 "/ID/"（各根前缀互不相同），
// 子孙节点沿用库函数的 "/父路径/ID/" 拼接。
func (r *OrgUnitRepo) computeUnitTreePath(parentPath string, nodeID uint64) string {
	if parentPath == "" {
		return "/" + strconv.FormatUint(uint64(nodeID), 10) + "/"
	}
	return entCrud.ComputeTreePath64(parentPath, nodeID)
}

// ListOrgUnitIDsInTenant 返回 ids 中属于指定租户的子集（跨租户剔除）。
// 登录聚合上下文为 privacy.Allow（绕过租户隐私层），本方法内的显式租户谓词
// 是该路径上唯一的租户防线。
func (r *OrgUnitRepo) ListOrgUnitIDsInTenant(ctx context.Context, tenantID uint64, ids []uint64) ([]uint64, error) {
	if len(ids) == 0 {
		return []uint64{}, nil
	}

	intIDs, err := r.entClient.Client().OrgUnit.Query().
		Where(
			orgunit.IDIn(ids...),
			orgunit.TenantIDEQ(tenantID),
		).
		IDs(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query org unit ids in tenant failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query org unit ids in tenant failed")
	}

	result := make([]uint64, len(intIDs))
	for i, v := range intIDs {
		result[i] = uint64(v)
	}
	return result, nil
}

// ListSelfAndDescendantOrgUnitIds 返回种子单元集及其全部后代的并集
// （含种子自身），以 path 前缀匹配展开，全程限定指定租户。
// UNIT_AND_CHILD 数据范围的登录期展开专用；登录上下文为 privacy.Allow，
// 显式租户谓词是该路径上唯一的租户防线。
func (r *OrgUnitRepo) ListSelfAndDescendantOrgUnitIds(ctx context.Context, tenantID uint64, seeds []uint64) ([]uint64, error) {
	if len(seeds) == 0 {
		return []uint64{}, nil
	}

	seedPaths, err := r.entClient.Client().OrgUnit.Query().
		Where(
			orgunit.IDIn(seeds...),
			orgunit.TenantIDEQ(tenantID),
		).
		Select(orgunit.FieldPath).
		Strings(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query seed org unit paths failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query seed org unit paths failed")
	}

	var pathPreds []predicate.OrgUnit
	for _, p := range seedPaths {
		if p == "" {
			continue
		}
		pathPreds = append(pathPreds, orgunit.PathHasPrefix(p))
	}
	if len(pathPreds) == 0 {
		return []uint64{}, nil
	}

	intIDs, err := r.entClient.Client().OrgUnit.Query().
		Where(
			orgunit.TenantIDEQ(tenantID),
			orgunit.Or(pathPreds...),
		).
		IDs(ctx)
	if err != nil {
		r.log.Errorf(ctx, "expand org unit descendants failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("expand org unit descendants failed")
	}

	result := make([]uint64, len(intIDs))
	for i, v := range intIDs {
		result[i] = uint64(v)
	}
	return result, nil
}
