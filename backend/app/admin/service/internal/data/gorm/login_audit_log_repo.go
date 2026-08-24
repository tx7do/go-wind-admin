//go:build gorm_backend
// +build gorm_backend

// Package gorm 中的仓储是 ent 仓储的平行 gorm 镜像，作为"ent 为主力、gorm 为备选"脚手架的完整代码。
// 这些仓储为死代码：不接入 wire、不被 service 引用；采用者需要时自行装配。
//
// gorm 仓储不做租户隔离（ent 侧靠编译进生成代码的 privacy 策略自动注入，gorm 侧无此机制）。
// 直接切换 gorm 后端会有跨租户数据泄露风险，采用者须自行加 scope/plugin。
package gorm

import (
	"context"
	"errors"

	gormDB "gorm.io/gorm"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	gormCrud "github.com/tx7do/go-crud/gorm"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/admin/service/internal/data/gorm/models"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
)

type LoginAuditLogRepo struct {
	client     *gormCrud.Client
	log        *log.Helper
	mapper     *mapper.CopierMapper[auditV1.LoginAuditLog, models.LoginAuditLog]
	repository *gormCrud.Repository[auditV1.LoginAuditLog, models.LoginAuditLog]

	statusConverter      *mapper.EnumTypeConverter[auditV1.LoginAuditLog_Status, string]
	actionTypeConverter  *mapper.EnumTypeConverter[auditV1.LoginAuditLog_ActionType, string]
	riskLevelConverter   *mapper.EnumTypeConverter[auditV1.LoginAuditLog_RiskLevel, string]
	loginMethodConverter *mapper.EnumTypeConverter[auditV1.LoginAuditLog_LoginMethod, string]
}

func NewLoginAuditLogRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *LoginAuditLogRepo {
	repo := &LoginAuditLogRepo{
		log:    ctx.NewLoggerHelper("login-audit-log/gorm-repo/admin-service"),
		client: client,
		mapper: mapper.NewCopierMapper[auditV1.LoginAuditLog, models.LoginAuditLog](),

		statusConverter:      mapper.NewEnumTypeConverter[auditV1.LoginAuditLog_Status, string](auditV1.LoginAuditLog_Status_name, auditV1.LoginAuditLog_Status_value),
		actionTypeConverter:  mapper.NewEnumTypeConverter[auditV1.LoginAuditLog_ActionType, string](auditV1.LoginAuditLog_ActionType_name, auditV1.LoginAuditLog_ActionType_value),
		riskLevelConverter:   mapper.NewEnumTypeConverter[auditV1.LoginAuditLog_RiskLevel, string](auditV1.LoginAuditLog_RiskLevel_name, auditV1.LoginAuditLog_RiskLevel_value),
		loginMethodConverter: mapper.NewEnumTypeConverter[auditV1.LoginAuditLog_LoginMethod, string](auditV1.LoginAuditLog_LoginMethod_name, auditV1.LoginAuditLog_LoginMethod_value),
	}

	repo.init()

	return repo
}

func (r *LoginAuditLogRepo) init() {
	r.repository = gormCrud.NewRepository[auditV1.LoginAuditLog, models.LoginAuditLog](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
	r.mapper.AppendConverters(r.actionTypeConverter.NewConverterPair())
	r.mapper.AppendConverters(r.riskLevelConverter.NewConverterPair())
	// 此前漏注册 loginMethodConverter：写入时直接调 converter 故正常，
	// 但读回（Get/List）走共享 mapper，login_method 字段恒为零值。补注册。
	r.mapper.AppendConverters(r.loginMethodConverter.NewConverterPair())
}

func (r *LoginAuditLogRepo) Count(ctx context.Context, scopes []func(*gormDB.DB) *gormDB.DB) (int, error) {
	count, err := r.repository.Count(ctx, r.client.DB, scopes)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, adminV1.ErrorInternalServerError("query count failed")
	}

	return int(count), nil
}

func (r *LoginAuditLogRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*auditV1.ListLoginAuditLogResponse, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	ret, err := r.repository.ListWithPaging(ctx, r.client.DB, req)
	if err != nil {
		return nil, adminV1.ErrorInternalServerError("query list failed")
	}
	if ret == nil {
		return &auditV1.ListLoginAuditLogResponse{Total: 0, Items: nil}, nil
	}

	return &auditV1.ListLoginAuditLogResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *LoginAuditLogRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.repository.ExistsWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", id) },
	})
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, adminV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *LoginAuditLogRepo) Get(ctx context.Context, req *auditV1.GetLoginAuditLogRequest) (*auditV1.LoginAuditLog, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	var scopes []func(*gormDB.DB) *gormDB.DB
	switch req.QueryBy.(type) {
	default:
	case *auditV1.GetLoginAuditLogRequest_Id:
		scopes = append(scopes, func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) })
	}

	dto, err := r.repository.GetWithFilters(ctx, r.client.DB, scopes, req.GetViewMask())
	if err != nil {
		if errors.Is(err, gormDB.ErrRecordNotFound) {
			return nil, adminV1.ErrorNotFound("login audit log not found")
		}
		r.log.Errorf("query login audit log failed: %s", err.Error())
		return nil, adminV1.ErrorInternalServerError("query login audit log failed")
	}

	return dto, nil
}

func (r *LoginAuditLogRepo) Create(ctx context.Context, req *auditV1.CreateLoginAuditLogRequest) error {
	if req == nil || req.Data == nil {
		return adminV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := r.repository.Create(ctx, r.client.DB, req.Data, nil); err != nil {
		r.log.Errorf("insert login audit log failed: %s", err.Error())
		return adminV1.ErrorInternalServerError("insert login audit log failed")
	}

	return nil
}
