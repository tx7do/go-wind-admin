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

type DataAccessAuditLogRepo struct {
	client     *gormCrud.Client
	log        *log.Helper
	mapper     *mapper.CopierMapper[auditV1.DataAccessAuditLog, models.DataAccessAuditLog]
	repository *gormCrud.Repository[auditV1.DataAccessAuditLog, models.DataAccessAuditLog]

	accessTypeConverter     *mapper.EnumTypeConverter[auditV1.DataAccessAuditLog_AccessType, string]
	sensitiveLevelConverter *mapper.EnumTypeConverter[auditV1.SensitiveLevel, string]
}

func NewDataAccessAuditLogRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *DataAccessAuditLogRepo {
	repo := &DataAccessAuditLogRepo{
		log:    ctx.NewLoggerHelper("data-access-audit-log/gorm-repo/admin-service"),
		client: client,
		mapper: mapper.NewCopierMapper[auditV1.DataAccessAuditLog, models.DataAccessAuditLog](),

		accessTypeConverter:     mapper.NewEnumTypeConverter[auditV1.DataAccessAuditLog_AccessType, string](auditV1.DataAccessAuditLog_AccessType_name, auditV1.DataAccessAuditLog_AccessType_value),
		sensitiveLevelConverter: mapper.NewEnumTypeConverter[auditV1.SensitiveLevel, string](auditV1.SensitiveLevel_name, auditV1.SensitiveLevel_value),
	}

	repo.init()

	return repo
}

func (r *DataAccessAuditLogRepo) init() {
	r.repository = gormCrud.NewRepository[auditV1.DataAccessAuditLog, models.DataAccessAuditLog](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.accessTypeConverter.NewConverterPair())
	r.mapper.AppendConverters(r.sensitiveLevelConverter.NewConverterPair())
}

func (r *DataAccessAuditLogRepo) Count(ctx context.Context, scopes []func(*gormDB.DB) *gormDB.DB) (int, error) {
	count, err := r.repository.Count(ctx, r.client.DB, scopes)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, adminV1.ErrorInternalServerError("query count failed")
	}

	return int(count), nil
}

func (r *DataAccessAuditLogRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*auditV1.ListDataAccessAuditLogResponse, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	ret, err := r.repository.ListWithPaging(ctx, r.client.DB, req)
	if err != nil {
		return nil, adminV1.ErrorInternalServerError("query list failed")
	}
	if ret == nil {
		return &auditV1.ListDataAccessAuditLogResponse{Total: 0, Items: nil}, nil
	}

	return &auditV1.ListDataAccessAuditLogResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *DataAccessAuditLogRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.repository.ExistsWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", id) },
	})
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, adminV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *DataAccessAuditLogRepo) Get(ctx context.Context, req *auditV1.GetDataAccessAuditLogRequest) (*auditV1.DataAccessAuditLog, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	var scopes []func(*gormDB.DB) *gormDB.DB
	switch req.QueryBy.(type) {
	default:
	case *auditV1.GetDataAccessAuditLogRequest_Id:
		scopes = append(scopes, func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) })
	}

	dto, err := r.repository.GetWithFilters(ctx, r.client.DB, scopes, req.GetViewMask())
	if err != nil {
		if errors.Is(err, gormDB.ErrRecordNotFound) {
			return nil, adminV1.ErrorNotFound("data access audit log not found")
		}
		r.log.Errorf("query data access audit log failed: %s", err.Error())
		return nil, adminV1.ErrorInternalServerError("query data access audit log failed")
	}

	return dto, nil
}

func (r *DataAccessAuditLogRepo) Create(ctx context.Context, req *auditV1.CreateDataAccessAuditLogRequest) error {
	if req == nil || req.Data == nil {
		return adminV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := r.repository.Create(ctx, r.client.DB, req.Data, nil); err != nil {
		r.log.Errorf("insert data access audit log failed: %s", err.Error())
		return adminV1.ErrorInternalServerError("insert data access audit log failed")
	}

	return nil
}
