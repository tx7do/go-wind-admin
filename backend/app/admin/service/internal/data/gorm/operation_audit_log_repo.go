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
	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
)

type OperationAuditLogRepo struct {
	client     *gormCrud.Client
	log        *bLogger.Helper
	mapper     *mapper.CopierMapper[auditV1.OperationAuditLog, models.OperationAuditLog]
	repository *gormCrud.Repository[auditV1.OperationAuditLog, models.OperationAuditLog]

	actionTypeConverter     *mapper.EnumTypeConverter[auditV1.OperationAuditLog_ActionType, string]
	sensitiveLevelConverter *mapper.EnumTypeConverter[auditV1.SensitiveLevel, string]
}

func NewOperationAuditLogRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *OperationAuditLogRepo {
	repo := &OperationAuditLogRepo{
		log:    ctx.NewLoggerHelper("operation-audit-log/gorm-repo/admin-service"),
		client: client,
		mapper: mapper.NewCopierMapper[auditV1.OperationAuditLog, models.OperationAuditLog](),

		actionTypeConverter:     mapper.NewEnumTypeConverter[auditV1.OperationAuditLog_ActionType, string](auditV1.OperationAuditLog_ActionType_name, auditV1.OperationAuditLog_ActionType_value),
		sensitiveLevelConverter: mapper.NewEnumTypeConverter[auditV1.SensitiveLevel, string](auditV1.SensitiveLevel_name, auditV1.SensitiveLevel_value),
	}

	repo.init()

	return repo
}

func (r *OperationAuditLogRepo) init() {
	r.repository = gormCrud.NewRepository[auditV1.OperationAuditLog, models.OperationAuditLog](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.actionTypeConverter.NewConverterPair())
	r.mapper.AppendConverters(r.sensitiveLevelConverter.NewConverterPair())
}

func (r *OperationAuditLogRepo) Count(ctx context.Context, scopes []func(*gormDB.DB) *gormDB.DB) (int, error) {
	count, err := r.repository.Count(ctx, r.client.DB, scopes)
	if err != nil {
		r.log.Errorf(ctx, "query count failed: %s", err.Error())
		return 0, adminV1.ErrorInternalServerError("query count failed")
	}

	return int(count), nil
}

func (r *OperationAuditLogRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*auditV1.ListOperationAuditLogResponse, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	ret, err := r.repository.ListWithPaging(ctx, r.client.DB, req)
	if err != nil {
		return nil, adminV1.ErrorInternalServerError("query list failed")
	}
	if ret == nil {
		return &auditV1.ListOperationAuditLogResponse{Total: 0, Items: nil}, nil
	}

	return &auditV1.ListOperationAuditLogResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *OperationAuditLogRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.repository.ExistsWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", id) },
	})
	if err != nil {
		r.log.Errorf(ctx, "query exist failed: %s", err.Error())
		return false, adminV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *OperationAuditLogRepo) Get(ctx context.Context, req *auditV1.GetOperationAuditLogRequest) (*auditV1.OperationAuditLog, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	var scopes []func(*gormDB.DB) *gormDB.DB
	switch req.QueryBy.(type) {
	default:
	case *auditV1.GetOperationAuditLogRequest_Id:
		scopes = append(scopes, func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) })
	}

	dto, err := r.repository.GetWithFilters(ctx, r.client.DB, scopes, req.GetViewMask())
	if err != nil {
		if errors.Is(err, gormDB.ErrRecordNotFound) {
			return nil, adminV1.ErrorNotFound("operation audit log not found")
		}
		r.log.Errorf(ctx, "query operation audit log failed: %s", err.Error())
		return nil, adminV1.ErrorInternalServerError("query operation audit log failed")
	}

	return dto, nil
}

func (r *OperationAuditLogRepo) Create(ctx context.Context, req *auditV1.CreateOperationAuditLogRequest) error {
	if req == nil || req.Data == nil {
		return adminV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := r.repository.Create(ctx, r.client.DB, req.Data, nil); err != nil {
		r.log.Errorf(ctx, "insert operation audit log failed: %s", err.Error())
		return adminV1.ErrorInternalServerError("insert operation audit log failed")
	}

	return nil
}
