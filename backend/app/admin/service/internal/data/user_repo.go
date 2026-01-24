package data

import (
	"context"

	"strconv"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/timeutil"
	"github.com/tx7do/go-utils/trans"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"
	"go-wind-admin/app/admin/service/internal/data/ent/user"

	userV1 "go-wind-admin/api/gen/go/user/service/v1"

	"go-wind-admin/pkg/constants"
	"go-wind-admin/pkg/utils/slice"
)

type UserRepo interface {
	List(ctx context.Context, req *paginationV1.PagingRequest) (*userV1.ListUserResponse, error)

	Get(ctx context.Context, req *userV1.GetUserRequest) (*userV1.User, error)

	Create(ctx context.Context, req *userV1.CreateUserRequest) (*userV1.User, error)
	CreateWithTx(ctx context.Context, tx *ent.Tx, data *userV1.User) (*userV1.User, error)

	Update(ctx context.Context, req *userV1.UpdateUserRequest) error

	Delete(ctx context.Context, req *userV1.DeleteUserRequest) error

	Count(ctx context.Context) (int, error)

	UserExists(ctx context.Context, req *userV1.UserExistsRequest) (*userV1.UserExistsResponse, error)

	AssignUserRole(ctx context.Context, data *userV1.UserRole) error
	AssignUserRoles(ctx context.Context, userID uint32, datas []*userV1.UserRole) error

	AssignUserOrgUnit(ctx context.Context, data *userV1.UserOrgUnit) error
	AssignUserOrgUnits(ctx context.Context, userID uint32, datas []*userV1.UserOrgUnit) error

	AssignUserPosition(ctx context.Context, data *userV1.UserPosition) error
	AssignUserPositions(ctx context.Context, userID uint32, datas []*userV1.UserPosition) error

	ListUsersByIds(ctx context.Context, ids []uint32) ([]*userV1.User, error)

	ListRoleIDsByUserID(ctx context.Context, userID uint32) ([]uint32, error)

	ListPositionIDsByUserID(ctx context.Context, userID uint32) ([]uint32, error)

	ListOrgUnitIDsByUserID(ctx context.Context, userID uint32) ([]uint32, error)

	ListUserRelationIDs(ctx context.Context, userID uint32) (roleIDs []uint32, positionIDs []uint32, orgUnitIDs []uint32, err error)
}

type userRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper          *mapper.CopierMapper[userV1.User, ent.User]
	genderConverter *mapper.EnumTypeConverter[userV1.User_Gender, user.Gender]
	statusConverter *mapper.EnumTypeConverter[userV1.User_Status, user.Status]

	repository *entCrud.Repository[
		ent.UserQuery, ent.UserSelect,
		ent.UserCreate, ent.UserCreateBulk,
		ent.UserUpdate, ent.UserUpdateOne,
		ent.UserDelete,
		predicate.User,
		userV1.User, ent.User,
	]

	userRoleRepo     *UserRoleRepo
	userOrgUnitRepo  *UserOrgUnitRepo
	userPositionRepo *UserPositionRepo

	membershipRepo *MembershipRepo
}

func NewUserRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
	userRoleRepo *UserRoleRepo,
	userOrgUnitRepo *UserOrgUnitRepo,
	userPositionRepo *UserPositionRepo,
	membershipRepo *MembershipRepo,
) UserRepo {
	repo := &userRepo{
		log:              ctx.NewLoggerHelper("user/repo/admin-service"),
		entClient:        entClient,
		mapper:           mapper.NewCopierMapper[userV1.User, ent.User](),
		genderConverter:  mapper.NewEnumTypeConverter[userV1.User_Gender, user.Gender](userV1.User_Gender_name, userV1.User_Gender_value),
		statusConverter:  mapper.NewEnumTypeConverter[userV1.User_Status, user.Status](userV1.User_Status_name, userV1.User_Status_value),
		userRoleRepo:     userRoleRepo,
		userOrgUnitRepo:  userOrgUnitRepo,
		userPositionRepo: userPositionRepo,
		membershipRepo:   membershipRepo,
	}

	repo.init()

	return repo
}

func (r *userRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.UserQuery, ent.UserSelect,
		ent.UserCreate, ent.UserCreateBulk,
		ent.UserUpdate, ent.UserUpdateOne,
		ent.UserDelete,
		predicate.User,
		userV1.User, ent.User,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.genderConverter.NewConverterPair())
	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
}

// Count 统计用户数量
func (r *userRepo) Count(ctx context.Context) (int, error) {
	builder := r.entClient.Client().User.Query()

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, userV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

// FilterFields 过滤掉不需要的字段条件
func FilterFields(filterExpr *paginationV1.FilterExpr, excludeFields []string) []*paginationV1.FilterCondition {
	if filterExpr == nil || len(filterExpr.Conditions) == 0 {
		return []*paginationV1.FilterCondition{}
	}

	exclude := make(map[string]struct{}, len(excludeFields))
	for _, f := range excludeFields {
		if f == "" {
			continue
		}
		exclude[f] = struct{}{}
	}

	includeConditions := make([]*paginationV1.FilterCondition, 0, len(filterExpr.Conditions))
	excludeConditions := make([]*paginationV1.FilterCondition, 0, len(filterExpr.Conditions))
	for _, cond := range filterExpr.Conditions {
		if cond == nil || cond.Field == "" {
			continue
		}
		if _, skip := exclude[cond.Field]; skip {
			excludeConditions = append(excludeConditions, cond)
			continue
		}
		includeConditions = append(includeConditions, cond)
	}

	filterExpr.Conditions = includeConditions

	return excludeConditions
}

// queryUserIDsByRelationIDs 根据关联关系查询用户ID列表
func (r *userRepo) queryUserIDsByRelationIDs(ctx context.Context, roleIDs []uint32, orgUnitIDs []uint32, positionIDs []uint32) ([]uint32, error) {
	if len(roleIDs) == 0 && len(orgUnitIDs) == 0 && len(positionIDs) == 0 {
		return nil, nil
	}

	switch constants.DefaultUserTenantRelationType {
	default:
		fallthrough
	case constants.UserTenantRelationOneToOne:
		return r.queryUserIDsByRelationIDsUserTenantRelationOneToOne(ctx, roleIDs, orgUnitIDs, positionIDs)
	case constants.UserTenantRelationOneToMany:
		return r.queryUserIDsByRelationIDsUserTenantRelationOneToMany(ctx, roleIDs, orgUnitIDs, positionIDs)
	}
}

// queryUserIDsByRelationIDsUserTenantRelationOneToMany 根据关联关系一对多查询用户ID列表
func (r *userRepo) queryUserIDsByRelationIDsUserTenantRelationOneToMany(ctx context.Context, roleIDs []uint32, orgUnitIDs []uint32, positionIDs []uint32) ([]uint32, error) {
	if len(roleIDs) == 0 && len(orgUnitIDs) == 0 && len(positionIDs) == 0 {
		return nil, nil
	}

	var err error

	var orgUnitUserIDs []uint32
	var positionUserIDs []uint32
	var roleUserIDs []uint32

	if len(orgUnitIDs) > 0 {
		orgUnitUserIDs, err = r.membershipRepo.ListUserIDsByOrgUnitIDs(ctx, orgUnitIDs, false)
		if err != nil {
			return nil, err
		}
	}
	if len(positionIDs) > 0 {
		positionUserIDs, err = r.membershipRepo.ListUserIDsByPositionIDs(ctx, positionIDs, false)
		if err != nil {
			return nil, err
		}
	}
	if len(roleIDs) > 0 {
		roleUserIDs, err = r.membershipRepo.ListUserIDsByRoleIDs(ctx, roleIDs, false)
		if err != nil {
			return nil, err
		}
	}

	// 收集所有非空列表用于求交集
	lists := make([][]uint32, 0, 3)
	if orgUnitUserIDs != nil {
		lists = append(lists, orgUnitUserIDs)
	}
	if positionUserIDs != nil {
		lists = append(lists, positionUserIDs)
	}
	if roleUserIDs != nil {
		lists = append(lists, roleUserIDs)
	}

	// 如果没有任何实际列表（例如对应 ids 为空导致查询未执行），返回空
	if len(lists) == 0 {
		return []uint32{}, nil
	}

	// 逐步求交集
	result := lists[0]
	for i := 1; i < len(lists); i++ {
		result = slice.Intersect(result, lists[i])
		if len(result) == 0 {
			break
		}
	}

	return result, nil
}

// queryUserIDsByRelationIDsUserTenantRelationOneToOne 根据关联关系一对一查询用户ID列表
func (r *userRepo) queryUserIDsByRelationIDsUserTenantRelationOneToOne(ctx context.Context, roleIDs []uint32, orgUnitIDs []uint32, positionIDs []uint32) ([]uint32, error) {
	if len(roleIDs) == 0 && len(orgUnitIDs) == 0 && len(positionIDs) == 0 {
		return nil, nil
	}

	var err error

	var orgUnitUserIDs []uint32
	var positionUserIDs []uint32
	var roleUserIDs []uint32
	if len(orgUnitIDs) > 0 {
		orgUnitUserIDs, err = r.userOrgUnitRepo.ListUserIDsByOrgUnitIDs(ctx, orgUnitIDs, false)
		if err != nil {
			return nil, err
		}
	}
	if len(positionIDs) > 0 {
		positionUserIDs, err = r.userPositionRepo.ListUserIDsByPositionIDs(ctx, positionIDs, false)
		if err != nil {
			return nil, err
		}
	}
	if len(roleIDs) > 0 {
		roleUserIDs, err = r.userRoleRepo.ListUserIDsByRoleIDs(ctx, roleIDs, false)
		if err != nil {
			return nil, err
		}
	}

	// 收集所有非空列表用于求交集
	lists := make([][]uint32, 0, 3)
	if orgUnitUserIDs != nil {
		lists = append(lists, orgUnitUserIDs)
	}
	if positionUserIDs != nil {
		lists = append(lists, positionUserIDs)
	}
	if roleUserIDs != nil {
		lists = append(lists, roleUserIDs)
	}

	// 如果没有任何实际列表（例如对应 ids 为空导致查询未执行），返回空
	if len(lists) == 0 {
		return []uint32{}, nil
	}

	// 逐步求交集
	result := lists[0]
	for i := 1; i < len(lists); i++ {
		result = slice.Intersect(result, lists[i])
		if len(result) == 0 {
			break
		}
	}

	return result, nil
}

// List 列出用户
func (r *userRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*userV1.ListUserResponse, error) {
	if req == nil {
		return nil, userV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Debug().User.Query()

	filterExpr, err := r.repository.ConvertFilterByPagingRequest(req)
	if err != nil {
		r.log.Errorf("convert filter by paging request failed: %s", err.Error())
		return nil, err
	}

	excludeConditions := FilterFields(filterExpr, []string{
		"org_unit_id", "org_unit_ids",
		"position_id", "position_ids",
		"role_id", "role_ids",
	})

	var orgUnitIDs []uint32
	var positionIDs []uint32
	var roleIDs []uint32
	for _, cond := range excludeConditions {
		//r.log.Debugf("excluding filter condition: field=%s operator=%s value=%v", cond.GetField(), cond.GetOp(), cond.GetValue())

		var val uint64
		switch cond.GetField() {
		case "org_unit_id":
			if val, err = strconv.ParseUint(cond.GetValue(), 10, 64); err == nil {
				orgUnitIDs = append(orgUnitIDs, uint32(val))
			} else {
				r.log.Errorf("parse org_unit_id value failed: %s", err.Error())
			}
		case "org_unit_ids":
			for _, v := range cond.GetValues() {
				if val, err = strconv.ParseUint(v, 10, 64); err == nil {
					orgUnitIDs = append(orgUnitIDs, uint32(val))
				} else {
					r.log.Errorf("parse org_unit_ids value failed: %s", err.Error())
				}
			}

		case "position_id":
			if val, err = strconv.ParseUint(cond.GetValue(), 10, 64); err == nil {
				positionIDs = append(positionIDs, uint32(val))
			} else {
				r.log.Errorf("parse position_id value failed: %s", err.Error())
			}
		case "position_ids":
			for _, v := range cond.GetValues() {
				if val, err = strconv.ParseUint(v, 10, 64); err == nil {
					positionIDs = append(positionIDs, uint32(val))
				} else {
					r.log.Errorf("parse position_ids value failed: %s", err.Error())
				}
			}

		case "role_id":
			if val, err = strconv.ParseUint(cond.GetValue(), 10, 64); err == nil {
				roleIDs = append(roleIDs, uint32(val))
			} else {
				r.log.Errorf("parse role_id value failed: %s", err.Error())
			}
		case "role_ids":
			for _, v := range cond.GetValues() {
				if val, err = strconv.ParseUint(v, 10, 64); err == nil {
					roleIDs = append(roleIDs, uint32(val))
				} else {
					r.log.Errorf("parse role_ids value failed: %s", err.Error())
				}
			}
		}
	}

	var mergedUserIDs []uint32
	mergedUserIDs, err = r.queryUserIDsByRelationIDs(ctx, roleIDs, orgUnitIDs, positionIDs)
	if err != nil {
		r.log.Errorf("query user ids by relation ids failed: %s", err.Error())
		return nil, err
	}

	//r.log.Debugf("filtered user ids by relation ids: [%v] [%v] [%v] [%v]", roleIDs, orgUnitIDs, positionIDs, mergedUserIDs)

	hasRelationFilter := len(roleIDs) > 0 || len(orgUnitIDs) > 0 || len(positionIDs) > 0
	if hasRelationFilter && len(mergedUserIDs) == 0 {
		// 如果有关系过滤条件但没有匹配的用户ID，直接返回空结果
		return &userV1.ListUserResponse{Total: 0, Items: nil}, nil
	}

	if len(mergedUserIDs) > 0 {
		filterExpr.Conditions = append(filterExpr.Conditions, &paginationV1.FilterCondition{
			Field: "id",
			Op:    paginationV1.Operator_IN,
			Values: func() []string {
				values := make([]string, 0, len(mergedUserIDs))
				for _, id := range mergedUserIDs {
					values = append(values, strconv.FormatUint(uint64(id), 10))
				}
				return values
			}(),
		})
	}

	req.FilteringType = &paginationV1.PagingRequest_FilterExpr{FilterExpr: filterExpr}

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &userV1.ListUserResponse{Total: 0, Items: nil}, nil
	}

	resp := &userV1.ListUserResponse{
		Total: ret.Total,
		Items: ret.Items,
	}

	for _, item := range resp.Items {
		roleIDs, positionIDs, orgUnitIDs, err = r.ListUserRelationIDs(ctx, item.GetId())
		if err != nil {
			r.log.Errorf("list user relation ids failed: %s", err.Error())
			continue
		}
		item.RoleIds = roleIDs
		item.PositionIds = positionIDs
		item.OrgUnitIds = orgUnitIDs

		//r.log.Debugf("user id=%d role_ids=%v position_ids=%v org_unit_ids=%v", item.GetId(), roleIDs, positionIDs, orgUnitIDs)
	}

	return resp, nil
}

// Get 获取用户
func (r *userRepo) Get(ctx context.Context, req *userV1.GetUserRequest) (*userV1.User, error) {
	if req == nil {
		return nil, userV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().User.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	case *userV1.GetUserRequest_Id:
		whereCond = append(whereCond, user.IDEQ(req.GetId()))
	case *userV1.GetUserRequest_Username:
		whereCond = append(whereCond, user.UsernameEQ(req.GetUsername()))
	default:
		whereCond = append(whereCond, user.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	roleIDs, positionIDs, orgUnitIDs, err := r.ListUserRelationIDs(ctx, dto.GetId())
	if err != nil {
		r.log.Errorf("list user relation ids failed: %s", err.Error())
	}
	dto.RoleIds = roleIDs
	dto.PositionIds = positionIDs
	dto.OrgUnitIds = orgUnitIDs

	return dto, err
}

// Create 创建用户
func (r *userRepo) Create(ctx context.Context, req *userV1.CreateUserRequest) (dto *userV1.User, err error) {
	if req == nil || req.Data == nil {
		return nil, userV1.ErrorBadRequest("invalid parameter")
	}

	var tx *ent.Tx
	tx, err = r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf("start transaction failed: %s", err.Error())
		return nil, userV1.ErrorInternalServerError("start transaction failed")
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

	return r.CreateWithTx(ctx, tx, req.GetData())
}

// CreateWithTx 在事务中创建用户
func (r *userRepo) CreateWithTx(ctx context.Context, tx *ent.Tx, data *userV1.User) (dto *userV1.User, err error) {
	if data == nil {
		return nil, userV1.ErrorBadRequest("invalid parameter")
	}

	builder := tx.User.Create().
		SetNillableTenantID(data.TenantId).
		SetNillableUsername(data.Username).
		SetNillableNickname(data.Nickname).
		SetNillableRealname(data.Realname).
		SetNillableAvatar(data.Avatar).
		SetNillableEmail(data.Email).
		SetNillableMobile(data.Mobile).
		SetNillableTelephone(data.Telephone).
		SetNillableRegion(data.Region).
		SetNillableAddress(data.Address).
		SetNillableDescription(data.Description).
		SetNillableRemark(data.Remark).
		SetNillableLastLoginAt(timeutil.TimestamppbToTime(data.LastLoginAt)).
		SetNillableLockedUntil(timeutil.TimestamppbToTime(data.LockedUntil)).
		SetNillableLastLoginIP(data.LastLoginIp).
		SetNillableGender(r.genderConverter.ToEntity(data.Gender)).
		SetNillableStatus(r.statusConverter.ToEntity(data.Status)).
		SetNillableCreatedBy(data.CreatedBy).
		SetCreatedAt(time.Now())

	if data.Id != nil {
		builder.SetID(data.GetId())
	}

	var entity *ent.User
	if entity, err = builder.Save(ctx); err != nil {
		r.log.Errorf("insert user failed: %s", err.Error())
		return nil, userV1.ErrorInternalServerError("insert user failed")
	}

	if data.RoleId != nil {
		data.RoleIds = append(data.RoleIds, data.GetRoleId())
	}
	if data.OrgUnitId != nil {
		data.OrgUnitIds = append(data.OrgUnitIds, data.GetOrgUnitId())
	}
	if data.PositionId != nil {
		data.PositionIds = append(data.PositionIds, data.GetPositionId())
	}

	switch constants.DefaultUserTenantRelationType {
	case constants.UserTenantRelationNone, constants.UserTenantRelationOneToOne:
		if err = r.assignUserRelations(ctx, tx,
			data.GetTenantId(), entity.ID,
			data.GetRoleIds(),
			data.GetOrgUnitIds(),
			data.GetPositionIds()); err != nil {
			return nil, err
		}
	case constants.UserTenantRelationOneToMany:
		if err = r.assignMembershipRelations(ctx, tx,
			data.GetTenantId(), entity.ID,
			data.GetRoleIds(),
			data.GetOrgUnitIds(),
			data.GetPositionIds()); err != nil {
			return nil, err
		}
	}

	return r.mapper.ToDTO(entity), nil
}

// Update 更新用户
func (r *userRepo) Update(ctx context.Context, req *userV1.UpdateUserRequest) (err error) {
	if req == nil || req.Data == nil {
		return userV1.ErrorBadRequest("invalid parameter")
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		var existResp *userV1.UserExistsResponse
		existResp, err = r.UserExists(ctx, &userV1.UserExistsRequest{
			QueryBy: &userV1.UserExistsRequest_Id{Id: req.GetData().GetId()},
		})
		if err != nil {
			return err
		}
		if !existResp.Exist {
			createReq := &userV1.CreateUserRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			_, err = r.Create(ctx, createReq)
			return err
		}
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

	var entitiy *userV1.User
	builder := tx.User.UpdateOneID(req.GetId())
	entitiy, err = r.repository.UpdateOne(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *userV1.User) {
			builder.
				SetNillableNickname(req.Data.Nickname).
				SetNillableRealname(req.Data.Realname).
				SetNillableAvatar(req.Data.Avatar).
				SetNillableEmail(req.Data.Email).
				SetNillableMobile(req.Data.Mobile).
				SetNillableTelephone(req.Data.Telephone).
				SetNillableRegion(req.Data.Region).
				SetNillableAddress(req.Data.Address).
				SetNillableDescription(req.Data.Description).
				SetNillableRemark(req.Data.Remark).
				SetNillableLastLoginAt(timeutil.TimestamppbToTime(req.Data.LastLoginAt)).
				SetNillableLockedUntil(timeutil.TimestamppbToTime(req.Data.LockedUntil)).
				SetNillableLastLoginIP(req.Data.LastLoginIp).
				SetNillableGender(r.genderConverter.ToEntity(req.Data.Gender)).
				SetNillableStatus(r.statusConverter.ToEntity(req.Data.Status)).
				SetNillableUpdatedBy(req.Data.UpdatedBy).
				SetUpdatedAt(time.Now())

			if req.Data.RoleIds != nil {
				var roleIds []int
				for _, roleId := range req.Data.GetRoleIds() {
					roleIds = append(roleIds, int(roleId))
				}
				//builder.SetRoleIds(roleIds)
			}
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(user.FieldID, req.GetId()))
		},
	)

	switch constants.DefaultUserTenantRelationType {
	case constants.UserTenantRelationNone, constants.UserTenantRelationOneToOne:
		if err = r.assignUserRelations(ctx, tx,
			entitiy.GetTenantId(),
			req.GetId(),
			req.GetData().GetRoleIds(),
			req.GetData().GetOrgUnitIds(),
			req.GetData().GetPositionIds()); err != nil {
			return err
		}
	case constants.UserTenantRelationOneToMany:
		if err = r.assignMembershipRelations(ctx, tx,
			entitiy.GetTenantId(),
			req.GetId(),
			req.GetData().GetRoleIds(),
			req.GetData().GetOrgUnitIds(),
			req.GetData().GetPositionIds()); err != nil {
			return err
		}
	}

	return err
}

// Delete 删除用户
func (r *userRepo) Delete(ctx context.Context, req *userV1.DeleteUserRequest) (err error) {
	var existResp *userV1.UserExistsResponse
	existReq := &userV1.UserExistsRequest{}
	switch req.DeleteBy.(type) {
	default:
	case *userV1.DeleteUserRequest_Id:
		existReq.QueryBy = &userV1.UserExistsRequest_Id{Id: req.GetId()}
	case *userV1.DeleteUserRequest_Username:
		existReq.QueryBy = &userV1.UserExistsRequest_Username{Username: req.GetUsername()}
	}
	existResp, err = r.UserExists(ctx, existReq)
	if err != nil {
		return err
	}
	if !existResp.Exist {
		return userV1.ErrorNotFound("user not found")
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

	builder := tx.User.Delete()

	switch req.DeleteBy.(type) {
	case *userV1.DeleteUserRequest_Id:
		builder.Where(user.IDEQ(req.GetId()))
	case *userV1.DeleteUserRequest_Username:
		builder.Where(user.UsernameEQ(req.GetUsername()))
	default:
		builder.Where(user.IDEQ(req.GetId()))
	}

	if _, err = builder.Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return userV1.ErrorNotFound("user not found")
		}

		r.log.Errorf("delete one data failed: %s", err.Error())

		return userV1.ErrorInternalServerError("delete failed")
	}

	switch constants.DefaultUserTenantRelationType {
	case constants.UserTenantRelationNone, constants.UserTenantRelationOneToOne:
		if err = r.removeUserRelations(ctx, tx, req.GetId()); err != nil {
			return err
		}
	case constants.UserTenantRelationOneToMany:
		if err = r.removeMembershipRelations(ctx, tx, req.GetId()); err != nil {
			return err
		}
	}

	return nil
}

// removeUserRelations 移除用户关联关系
func (r *userRepo) removeUserRelations(ctx context.Context, tx *ent.Tx, userID uint32) (err error) {
	if err = r.userRoleRepo.CleanRelationsByUserID(ctx, tx, userID); err != nil {
		r.log.Errorf("clean user role relations failed: %s", err.Error())
	}
	if err = r.userOrgUnitRepo.CleanRelationsByUserID(ctx, tx, userID); err != nil {
		r.log.Errorf("clean user org unit relations failed: %s", err.Error())
	}
	if err = r.userPositionRepo.CleanRelationsByUserID(ctx, tx, userID); err != nil {
		r.log.Errorf("clean user position relations failed: %s", err.Error())
	}

	return
}

// removeMembershipRelations 移除用户关联关系（通过 Membership 关联）
func (r *userRepo) removeMembershipRelations(ctx context.Context, tx *ent.Tx, userID uint32) error {
	if err := r.membershipRepo.CleanRelationsByUserID(ctx, tx, userID); err != nil {
		r.log.Errorf("clean user membership relations failed: %s", err.Error())
		return err
	}
	return nil
}

// UserExists 检查用户是否存在
func (r *userRepo) UserExists(ctx context.Context, req *userV1.UserExistsRequest) (*userV1.UserExistsResponse, error) {
	builder := r.entClient.Client().User.Query()

	switch req.QueryBy.(type) {
	case *userV1.UserExistsRequest_Id:
		builder.Where(user.IDEQ(req.GetId()))
	case *userV1.UserExistsRequest_Username:
		builder.Where(user.UsernameEQ(req.GetUsername()))
	default:
		return &userV1.UserExistsResponse{
			Exist: false,
		}, userV1.ErrorBadRequest("invalid query by type")
	}

	exist, err := builder.Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return &userV1.UserExistsResponse{
			Exist: false,
		}, userV1.ErrorInternalServerError("query exist failed")
	}

	return &userV1.UserExistsResponse{
		Exist: exist,
	}, nil
}

// assignMembershipRelations 分配用户关联关系（通过 Membership 关联）
func (r *userRepo) assignMembershipRelations(ctx context.Context, tx *ent.Tx,
	tenantID, userID uint32,
	roleIDs, orgUnitIDs, positionIDs []uint32,
) error {
	now := time.Now()
	err := r.membershipRepo.AssignTenantMembershipWithTx(ctx, tx, &userV1.Membership{
		TenantId:   trans.Ptr(tenantID),
		UserId:     trans.Ptr(userID),
		Status:     userV1.Membership_ACTIVE.Enum(),
		IsPrimary:  trans.Ptr(true),
		AssignedAt: timeutil.TimeToTimestamppb(&now),

		RoleIds:     roleIDs,
		OrgUnitIds:  orgUnitIDs,
		PositionIds: positionIDs,
	})
	if err != nil {
		return err
	}

	return nil
}

// assignUserRelations 分配用户关联关系
func (r *userRepo) assignUserRelations(ctx context.Context, tx *ent.Tx,
	tenantID, userID uint32,
	roleIDs, orgUnitIDs, positionIDs []uint32,
) (err error) {
	if len(roleIDs) == 0 && len(orgUnitIDs) == 0 && len(positionIDs) == 0 {
		return nil
	}

	now := time.Now()

	if len(roleIDs) > 0 {
		roleIDs = slice.Unique(roleIDs)
		var userRoles []*userV1.UserRole
		for _, roleID := range roleIDs {
			userRoles = append(userRoles, &userV1.UserRole{
				TenantId:   trans.Ptr(tenantID),
				UserId:     trans.Ptr(userID),
				RoleId:     trans.Ptr(roleID),
				Status:     userV1.UserRole_ACTIVE.Enum(),
				IsPrimary:  trans.Ptr(true),
				AssignedAt: timeutil.TimeToTimestamppb(&now),
			})
		}
		if err = r.userRoleRepo.AssignUserRoles(ctx, tx, userID, userRoles); err != nil {
			return err
		}
	}
	if len(orgUnitIDs) > 0 {
		orgUnitIDs = slice.Unique(orgUnitIDs)
		//r.log.Debugf("assigning org unit ids: %v", orgUnitIDs)
		var userOrgUnits []*userV1.UserOrgUnit
		for _, orgUnitID := range orgUnitIDs {
			userOrgUnits = append(userOrgUnits, &userV1.UserOrgUnit{
				TenantId:   trans.Ptr(tenantID),
				UserId:     trans.Ptr(userID),
				OrgUnitId:  trans.Ptr(orgUnitID),
				Status:     userV1.UserOrgUnit_ACTIVE.Enum(),
				IsPrimary:  trans.Ptr(true),
				AssignedAt: timeutil.TimeToTimestamppb(&now),
			})
		}
		if err = r.userOrgUnitRepo.AssignUserOrgUnits(ctx, tx, userID, userOrgUnits); err != nil {
			return err
		}
	}
	if len(positionIDs) > 0 {
		positionIDs = slice.Unique(positionIDs)
		//r.log.Debugf("assigning position ids: %v", positionIDs)
		var userPositions []*userV1.UserPosition
		for _, positionID := range positionIDs {
			userPositions = append(userPositions, &userV1.UserPosition{
				TenantId:   trans.Ptr(tenantID),
				UserId:     trans.Ptr(userID),
				PositionId: trans.Ptr(positionID),
				Status:     userV1.UserPosition_ACTIVE.Enum(),
				IsPrimary:  trans.Ptr(true),
				AssignedAt: timeutil.TimeToTimestamppb(&now),
			})
		}
		if err = r.userPositionRepo.AssignUserPositions(ctx, tx, userID, userPositions); err != nil {
			return err
		}
	}

	return nil
}

// AssignUserRole 分配角色
func (r *userRepo) AssignUserRole(ctx context.Context, data *userV1.UserRole) error {
	var tx *ent.Tx
	tx, err := r.entClient.Client().Tx(ctx)
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

	return r.userRoleRepo.AssignUserRole(ctx, tx, data)
}

// AssignUserRoles 分配角色
func (r *userRepo) AssignUserRoles(ctx context.Context, userID uint32, datas []*userV1.UserRole) error {
	var tx *ent.Tx
	tx, err := r.entClient.Client().Tx(ctx)
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

	return r.userRoleRepo.AssignUserRoles(ctx, tx, userID, datas)
}

// AssignUserOrgUnit 分配组织单元给用户
func (r *userRepo) AssignUserOrgUnit(ctx context.Context, data *userV1.UserOrgUnit) error {
	var tx *ent.Tx
	tx, err := r.entClient.Client().Tx(ctx)
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

	return r.userOrgUnitRepo.AssignUserOrgUnit(ctx, tx, data)
}

// AssignUserOrgUnits 分配组织单元给用户
func (r *userRepo) AssignUserOrgUnits(ctx context.Context, userID uint32, datas []*userV1.UserOrgUnit) error {
	var tx *ent.Tx
	tx, err := r.entClient.Client().Tx(ctx)
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

	return r.userOrgUnitRepo.AssignUserOrgUnits(ctx, tx, userID, datas)
}

// AssignUserPosition 分配岗位给用户
func (r *userRepo) AssignUserPosition(ctx context.Context, data *userV1.UserPosition) error {
	var tx *ent.Tx
	tx, err := r.entClient.Client().Tx(ctx)
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

	return r.userPositionRepo.AssignUserPosition(ctx, tx, data)
}

// AssignUserPositions 分配岗位给用户
func (r *userRepo) AssignUserPositions(ctx context.Context, userID uint32, datas []*userV1.UserPosition) error {
	var tx *ent.Tx
	tx, err := r.entClient.Client().Tx(ctx)
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

	return r.userPositionRepo.AssignUserPositions(ctx, tx, userID, datas)
}

// ListUsersByIds 根据ID列表获取用户列表
func (r *userRepo) ListUsersByIds(ctx context.Context, ids []uint32) ([]*userV1.User, error) {
	if len(ids) == 0 {
		return []*userV1.User{}, nil
	}

	entities, err := r.entClient.Client().User.Query().
		Where(user.IDIn(ids...)).
		All(ctx)
	if err != nil {
		r.log.Errorf("query user by ids failed: %s", err.Error())
		return nil, userV1.ErrorInternalServerError("query user by ids failed")
	}

	dtos := make([]*userV1.User, 0, len(entities))
	for _, entity := range entities {
		dto := r.mapper.ToDTO(entity)
		dtos = append(dtos, dto)
	}

	return dtos, nil
}

func (r *userRepo) ListRoleIDsByUserID(ctx context.Context, userID uint32) ([]uint32, error) {
	return r.userRoleRepo.ListRoleIDs(ctx, userID, false)
}

func (r *userRepo) ListPositionIDsByUserID(ctx context.Context, userID uint32) ([]uint32, error) {
	return r.userPositionRepo.ListPositionIDs(ctx, userID, false)
}

func (r *userRepo) ListOrgUnitIDsByUserID(ctx context.Context, userID uint32) ([]uint32, error) {
	return r.userOrgUnitRepo.ListOrgUnitIDs(ctx, userID, false)
}

// ListUserRelationIDs 列出用户关联的角色、岗位、组织单元ID列表
func (r *userRepo) ListUserRelationIDs(ctx context.Context, userID uint32) (roleIDs []uint32, positionIDs []uint32, orgUnitIDs []uint32, err error) {
	if userID == 0 {
		return
	}

	switch constants.DefaultUserTenantRelationType {
	default:
		fallthrough
	case constants.UserTenantRelationOneToOne:
		return r.listUserRelationIDs(ctx, userID)
	case constants.UserTenantRelationOneToMany:
		return r.membershipRepo.ListMembershipRelationIDs(ctx, userID)
	}
}

// listUserRelationIDsOneToOne 列出用户关联的角色、岗位、组织单元ID列表（一对一关系）
func (r *userRepo) listUserRelationIDs(ctx context.Context, userID uint32) (roleIDs []uint32, positionIDs []uint32, orgUnitIDs []uint32, err error) {
	if userID == 0 {
		return
	}

	if roleIDs, err = r.userRoleRepo.ListRoleIDs(ctx, userID, false); err != nil {
		return
	}

	if positionIDs, err = r.userPositionRepo.ListPositionIDs(ctx, userID, false); err != nil {
		return
	}

	if orgUnitIDs, err = r.userOrgUnitRepo.ListOrgUnitIDs(ctx, userID, false); err != nil {
		return
	}

	return
}
