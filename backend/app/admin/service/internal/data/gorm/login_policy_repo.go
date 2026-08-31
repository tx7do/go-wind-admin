//go:build gorm_backend
// +build gorm_backend

// Package gorm 中的仓储是 ent 仓储的平行 gorm 镜像，作为"ent 为主力、gorm 为备选"脚手架的完整代码。
// 这些仓储仅由 cmd/server/wiring_gorm.go(gorm_backend 构建,ORM 切换 Phase 4 占位)装配,服务层尚未接入。
//
// gorm 仓储不做租户隔离（ent 侧靠编译进生成代码的 privacy 策略自动注入，gorm 侧无此机制）。
// 直接切换 gorm 后端会有跨租户数据泄露风险，采用者须自行加 scope/plugin。
package gorm

import (
	"context"
	"errors"

	gormDB "gorm.io/gorm"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	gormCrud "github.com/tx7do/go-crud/gorm"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/admin/service/internal/data/gorm/models"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
)

type LoginPolicyRepo struct {
	client     *gormCrud.Client
	log        *bLogger.Helper
	mapper     *mapper.CopierMapper[authenticationV1.LoginPolicy, models.LoginPolicy]
	repository *gormCrud.Repository[authenticationV1.LoginPolicy, models.LoginPolicy]

	typeConverter   *mapper.EnumTypeConverter[authenticationV1.LoginPolicy_Type, string]
	methodConverter *mapper.EnumTypeConverter[authenticationV1.LoginPolicy_Method, string]
}

func NewLoginPolicyRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *LoginPolicyRepo {
	repo := &LoginPolicyRepo{
		log:    ctx.NewLoggerHelper("login-policy/gorm-repo/admin-service"),
		client: client,
		mapper: mapper.NewCopierMapper[authenticationV1.LoginPolicy, models.LoginPolicy](),

		typeConverter:   mapper.NewEnumTypeConverter[authenticationV1.LoginPolicy_Type, string](authenticationV1.LoginPolicy_Type_name, authenticationV1.LoginPolicy_Type_value),
		methodConverter: mapper.NewEnumTypeConverter[authenticationV1.LoginPolicy_Method, string](authenticationV1.LoginPolicy_Method_name, authenticationV1.LoginPolicy_Method_value),
	}

	repo.init()

	return repo
}

func (r *LoginPolicyRepo) init() {
	r.repository = gormCrud.NewRepository[authenticationV1.LoginPolicy, models.LoginPolicy](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.typeConverter.NewConverterPair())
	r.mapper.AppendConverters(r.methodConverter.NewConverterPair())
}

func (r *LoginPolicyRepo) Count(ctx context.Context, scopes []func(*gormDB.DB) *gormDB.DB) (int, error) {
	count, err := r.repository.Count(ctx, r.client.DB, scopes)
	if err != nil {
		r.log.Errorf(ctx, "query count failed: %s", err.Error())
		return 0, adminV1.ErrorInternalServerError("query count failed")
	}

	return int(count), nil
}

func (r *LoginPolicyRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*authenticationV1.ListLoginPolicyResponse, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	ret, err := r.repository.ListWithPaging(ctx, r.client.DB, req)
	if err != nil {
		return nil, adminV1.ErrorInternalServerError("query list failed")
	}
	if ret == nil {
		return &authenticationV1.ListLoginPolicyResponse{Total: 0, Items: nil}, nil
	}

	return &authenticationV1.ListLoginPolicyResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *LoginPolicyRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.repository.ExistsWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", id) },
	})
	if err != nil {
		r.log.Errorf(ctx, "query exist failed: %s", err.Error())
		return false, adminV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *LoginPolicyRepo) Get(ctx context.Context, req *authenticationV1.GetLoginPolicyRequest) (*authenticationV1.LoginPolicy, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	var scopes []func(*gormDB.DB) *gormDB.DB
	switch req.QueryBy.(type) {
	default:
	case *authenticationV1.GetLoginPolicyRequest_Id:
		scopes = append(scopes, func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) })
	}

	dto, err := r.repository.GetWithFilters(ctx, r.client.DB, scopes, req.GetViewMask())
	if err != nil {
		if errors.Is(err, gormDB.ErrRecordNotFound) {
			return nil, adminV1.ErrorNotFound("login policy not found")
		}
		r.log.Errorf(ctx, "query login policy failed: %s", err.Error())
		return nil, adminV1.ErrorInternalServerError("query login policy failed")
	}

	return dto, nil
}

func (r *LoginPolicyRepo) Create(ctx context.Context, req *authenticationV1.CreateLoginPolicyRequest) error {
	if req == nil || req.Data == nil {
		return adminV1.ErrorBadRequest("invalid request")
	}

	if _, err := r.repository.Create(ctx, r.client.DB, req.Data, nil); err != nil {
		r.log.Errorf(ctx, "insert admin login restriction failed: %s", err.Error())
		return adminV1.ErrorInternalServerError("insert admin login restriction failed")
	}

	return nil
}

func (r *LoginPolicyRepo) Update(ctx context.Context, req *authenticationV1.UpdateLoginPolicyRequest) error {
	if req == nil || req.Data == nil {
		return adminV1.ErrorBadRequest("invalid request")
	}
	if req.GetId() == 0 {
		return adminV1.ErrorBadRequest("id is required")
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return err
		}
		if !exist {
			createReq := &authenticationV1.CreateLoginPolicyRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	if _, err := r.repository.UpdateWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) },
	}, req.Data, req.GetUpdateMask()); err != nil {
		r.log.Errorf(ctx, "update admin login restriction failed: %s", err.Error())
		return adminV1.ErrorInternalServerError("update admin login restriction failed")
	}

	return nil
}

func (r *LoginPolicyRepo) Delete(ctx context.Context, req *authenticationV1.DeleteLoginPolicyRequest) error {
	if req == nil {
		return adminV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) },
	}); err != nil {
		r.log.Errorf(ctx, "delete admin login restriction failed: %s", err.Error())
		return adminV1.ErrorInternalServerError("delete admin login restriction failed")
	}

	return nil
}
