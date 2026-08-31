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

	taskV1 "go-wind-admin/api/gen/go/task/service/v1"
)

type TaskRepo struct {
	client        *gormCrud.Client
	log           *bLogger.Helper
	mapper        *mapper.CopierMapper[taskV1.Task, models.Task]
	typeConverter *mapper.EnumTypeConverter[taskV1.Task_Type, string]
	repository    *gormCrud.Repository[taskV1.Task, models.Task]
}

func NewTaskRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *TaskRepo {
	repo := &TaskRepo{
		log:           ctx.NewLoggerHelper("task/gorm-repo/admin-service"),
		client:        client,
		mapper:        mapper.NewCopierMapper[taskV1.Task, models.Task](),
		typeConverter: mapper.NewEnumTypeConverter[taskV1.Task_Type, string](taskV1.Task_Type_name, taskV1.Task_Type_value),
	}

	repo.init()

	return repo
}

func (r *TaskRepo) init() {
	r.repository = gormCrud.NewRepository[taskV1.Task, models.Task](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.typeConverter.NewConverterPair())
}

func (r *TaskRepo) Count(ctx context.Context, scopes []func(*gormDB.DB) *gormDB.DB) (int, error) {
	count, err := r.repository.Count(ctx, r.client.DB, scopes)
	if err != nil {
		r.log.Errorf(ctx, "query count failed: %s", err.Error())
		return 0, taskV1.ErrorInternalServerError("query count failed")
	}

	return int(count), nil
}

func (r *TaskRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*taskV1.ListTaskResponse, error) {
	if req == nil {
		return nil, taskV1.ErrorBadRequest("invalid parameter")
	}

	ret, err := r.repository.ListWithPaging(ctx, r.client.DB, req)
	if err != nil {
		return nil, taskV1.ErrorInternalServerError("query list failed")
	}
	if ret == nil {
		return &taskV1.ListTaskResponse{Total: 0, Items: nil}, nil
	}

	return &taskV1.ListTaskResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *TaskRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.repository.ExistsWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", id) },
	})
	if err != nil {
		r.log.Errorf(ctx, "query exist failed: %s", err.Error())
		return false, taskV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *TaskRepo) Get(ctx context.Context, req *taskV1.GetTaskRequest) (*taskV1.Task, error) {
	if req == nil {
		return nil, taskV1.ErrorBadRequest("invalid parameter")
	}

	var scopes []func(*gormDB.DB) *gormDB.DB
	switch req.QueryBy.(type) {
	default:
	case *taskV1.GetTaskRequest_Id:
		scopes = append(scopes, func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) })
	case *taskV1.GetTaskRequest_TypeName:
		scopes = append(scopes, func(db *gormDB.DB) *gormDB.DB { return db.Where("type_name = ?", req.GetTypeName()) })
	}

	dto, err := r.repository.GetWithFilters(ctx, r.client.DB, scopes, req.GetViewMask())
	if err != nil {
		if errors.Is(err, gormDB.ErrRecordNotFound) {
			return nil, taskV1.ErrorNotFound("task not found")
		}
		r.log.Errorf(ctx, "query task failed: %s", err.Error())
		return nil, taskV1.ErrorInternalServerError("query task failed")
	}

	return dto, nil
}

func (r *TaskRepo) Create(ctx context.Context, req *taskV1.CreateTaskRequest) error {
	if req == nil || req.Data == nil {
		return taskV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := r.repository.Create(ctx, r.client.DB, req.Data, nil); err != nil {
		r.log.Errorf(ctx, "insert task failed: %s", err.Error())
		return taskV1.ErrorInternalServerError("insert data failed")
	}

	return nil
}

func (r *TaskRepo) Update(ctx context.Context, req *taskV1.UpdateTaskRequest) error {
	if req == nil || req.Data == nil {
		return taskV1.ErrorBadRequest("invalid parameter")
	}
	if req.GetId() == 0 {
		return taskV1.ErrorBadRequest("id is required")
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return err
		}
		if !exist {
			createReq := &taskV1.CreateTaskRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	if _, err := r.repository.UpdateWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) },
	}, req.Data, req.GetUpdateMask()); err != nil {
		r.log.Errorf(ctx, "update task failed: %s", err.Error())
		return taskV1.ErrorInternalServerError("update task failed")
	}

	return nil
}

func (r *TaskRepo) Delete(ctx context.Context, req *taskV1.DeleteTaskRequest) error {
	if req == nil {
		return taskV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) },
	}); err != nil {
		r.log.Errorf(ctx, "delete task failed: %s", err.Error())
		return taskV1.ErrorInternalServerError("delete task failed")
	}

	return nil
}
