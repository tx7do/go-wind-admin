package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	pagination "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/timeutil"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"
	_ "go-wind-admin/app/admin/service/internal/data/ent/runtime"
	"go-wind-admin/app/admin/service/internal/data/ent/user"

	userV1 "go-wind-admin/api/gen/go/user/service/v1"
)

type UserRepo interface {
	List(ctx context.Context, req *pagination.PagingRequest) (*userV1.ListUserResponse, error)

	Get(ctx context.Context, req *userV1.GetUserRequest) (*userV1.User, error)

	Create(ctx context.Context, req *userV1.CreateUserRequest) (*userV1.User, error)

	Update(ctx context.Context, req *userV1.UpdateUserRequest) error

	Delete(ctx context.Context, req *userV1.DeleteUserRequest) error

	Count(ctx context.Context) (int, error)

	ListUsersByIds(ctx context.Context, ids []uint32) ([]*userV1.User, error)

	UserExists(ctx context.Context, req *userV1.UserExistsRequest) (*userV1.UserExistsResponse, error)
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
}

func NewUserRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) UserRepo {
	repo := &userRepo{
		log:             ctx.NewLoggerHelper("user/repo/admin-service"),
		entClient:       entClient,
		mapper:          mapper.NewCopierMapper[userV1.User, ent.User](),
		genderConverter: mapper.NewEnumTypeConverter[userV1.User_Gender, user.Gender](userV1.User_Gender_name, userV1.User_Gender_value),
		statusConverter: mapper.NewEnumTypeConverter[userV1.User_Status, user.Status](userV1.User_Status_name, userV1.User_Status_value),
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

func (r *userRepo) Count(ctx context.Context) (int, error) {
	builder := r.entClient.Client().User.Query()

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, userV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *userRepo) List(ctx context.Context, req *pagination.PagingRequest) (*userV1.ListUserResponse, error) {
	if req == nil {
		return nil, userV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().User.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &userV1.ListUserResponse{Total: 0, Items: nil}, nil
	}

	return &userV1.ListUserResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

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

	return dto, err
}

func (r *userRepo) Create(ctx context.Context, req *userV1.CreateUserRequest) (*userV1.User, error) {
	if req == nil || req.Data == nil {
		return nil, userV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().User.Create().
		SetNillableUsername(req.Data.Username).
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
		SetNillableCreatedBy(req.Data.CreatedBy).
		SetNillableCreatedAt(timeutil.TimestamppbToTime(req.Data.CreatedAt))

	if req.Data.TenantId == nil {
		builder.SetTenantID(req.Data.GetTenantId())
	}
	if req.Data.CreatedAt == nil {
		builder.SetCreatedAt(time.Now())
	}

	if req.Data.Id != nil {
		builder.SetID(req.GetData().GetId())
	}

	//if req.Data.Roles != nil {
	//	builder.SetRoles(req.Data.GetRoles())
	//}

	if req.Data.RoleIds != nil {
		var roleIds []int
		for _, roleId := range req.Data.GetRoleIds() {
			roleIds = append(roleIds, int(roleId))
		}
		//builder.SetRoleIds(roleIds)
	}

	if ret, err := builder.Save(ctx); err != nil {
		r.log.Errorf("insert one data failed: %s", err.Error())
		return nil, userV1.ErrorInternalServerError("insert data failed")
	} else {
		return r.mapper.ToDTO(ret), nil
	}
}

func (r *userRepo) Update(ctx context.Context, req *userV1.UpdateUserRequest) error {
	if req == nil || req.Data == nil {
		return userV1.ErrorBadRequest("invalid parameter")
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		exist, err := r.UserExists(ctx, &userV1.UserExistsRequest{
			QueryBy: &userV1.UserExistsRequest_Id{Id: req.GetData().GetId()},
		})
		if err != nil {
			return err
		}
		if !exist.Exist {
			createReq := &userV1.CreateUserRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			_, err = r.Create(ctx, createReq)
			return err
		}
	}

	builder := r.entClient.Client().Debug().User.Update()
	err := r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(),
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
				SetNillableUpdatedAt(timeutil.TimestamppbToTime(req.Data.UpdatedAt))

			if req.Data.UpdatedAt == nil {
				builder.SetUpdatedAt(time.Now())
			}

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

	return err
}

func (r *userRepo) Delete(ctx context.Context, req *userV1.DeleteUserRequest) error {
	builder := r.entClient.Client().User.Delete()

	switch req.DeleteBy.(type) {
	case *userV1.DeleteUserRequest_Id:
		builder.Where(user.IDEQ(req.GetId()))
	case *userV1.DeleteUserRequest_Username:
		builder.Where(user.UsernameEQ(req.GetUsername()))
	default:
		builder.Where(user.IDEQ(req.GetId()))
	}

	if _, err := builder.Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return userV1.ErrorNotFound("user not found")
		}

		r.log.Errorf("delete one data failed: %s", err.Error())

		return userV1.ErrorInternalServerError("delete failed")
	}

	return nil
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
