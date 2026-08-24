package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"
	"go-wind-admin/app/admin/service/internal/data/ent/task"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	taskV1 "go-wind-admin/api/gen/go/task/service/v1"
)

type TaskRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *bLogger.Helper

	mapper        *mapper.CopierMapper[taskV1.Task, ent.Task]
	typeConverter *mapper.EnumTypeConverter[taskV1.Task_Type, task.Type]

	repository *entCrud.Repository[
		ent.TaskQuery, ent.TaskSelect,
		ent.TaskCreate, ent.TaskCreateBulk,
		ent.TaskUpdate, ent.TaskUpdateOne,
		ent.TaskDelete,
		predicate.Task,
		taskV1.Task, ent.Task,
	]
}

func NewTaskRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *TaskRepo {
	repo := &TaskRepo{
		log:           ctx.NewLoggerHelper("task/repo/admin-service"),
		entClient:     entClient,
		mapper:        mapper.NewCopierMapper[taskV1.Task, ent.Task](),
		typeConverter: mapper.NewEnumTypeConverter[taskV1.Task_Type, task.Type](taskV1.Task_Type_name, taskV1.Task_Type_value),
	}

	repo.init()

	return repo
}

func (r *TaskRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.TaskQuery, ent.TaskSelect,
		ent.TaskCreate, ent.TaskCreateBulk,
		ent.TaskUpdate, ent.TaskUpdateOne,
		ent.TaskDelete,
		predicate.Task,
		taskV1.Task, ent.Task,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.typeConverter.NewConverterPair())
}

func (r *TaskRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().Task.Query()
	if len(whereCond) != 0 {
		builder.Modify(whereCond...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query count failed: %s", err.Error())
		return 0, taskV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *TaskRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*taskV1.ListTaskResponse, error) {
	if req == nil {
		return nil, taskV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Task.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
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
	exist, err := r.entClient.Client().Task.Query().
		Where(task.IDEQ(id)).
		Exist(ctx)
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

	builder := r.entClient.Client().Task.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *taskV1.GetTaskRequest_Id:
		whereCond = append(whereCond, task.IDEQ(req.GetId()))

	case *taskV1.GetTaskRequest_TypeName:
		// type_name 仅在 (tenant_id, type_name) 维度唯一，平台上下文(tid=0)下按 type_name 查询
		// 会跨租户匹配多行导致 .Only() 报 not singular，且会跨租户操控其他租户定时任务。
		// 仅允许在具名租户上下文(tid>0)下按 type_name 查询。
		if _, hasTenant := maybeTenantFromViewer(ctx); !hasTenant {
			return nil, taskV1.ErrorBadRequest("tenant scope required to query task by type name")
		}
		whereCond = append(whereCond, task.TypeNameEQ(req.GetTypeName()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

func (r *TaskRepo) Create(ctx context.Context, req *taskV1.CreateTaskRequest) (*taskV1.Task, error) {
	if req == nil || req.Data == nil {
		return nil, taskV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Task.Create().
		SetNillableTenantID(req.Data.TenantId).
		SetNillableType(r.typeConverter.ToEntity(req.Data.Type)).
		SetNillableTypeName(req.Data.TypeName).
		SetNillableTaskPayload(req.Data.TaskPayload).
		SetNillableCronSpec(req.Data.CronSpec).
		SetNillableEnable(req.Data.Enable).
		SetNillableRemark(req.Data.Remark).
		SetNillableCreatedBy(req.Data.CreatedBy).
		SetCreatedAt(time.Now())

	if req.Data.TaskOptions != nil {
		builder.SetTaskOptions(req.Data.TaskOptions)
	}

	if req.Data.Id != nil {
		builder.SetID(req.GetData().GetId())
	}

	t, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf(ctx, "insert task failed: %s", err.Error())
		return nil, taskV1.ErrorInternalServerError("insert task failed")
	}

	return r.mapper.ToDTO(t), nil
}

func (r *TaskRepo) Update(ctx context.Context, req *taskV1.UpdateTaskRequest) (*taskV1.Task, error) {
	if req == nil || req.Data == nil {
		return nil, taskV1.ErrorBadRequest("invalid parameter")
	}
	if req.GetId() == 0 {
		return nil, taskV1.ErrorBadRequest("id is required")
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return nil, err
		}
		if !exist {
			createReq := &taskV1.CreateTaskRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	builder := r.entClient.Client().Task.UpdateOneID(req.GetId())
	result, err := r.repository.UpdateOne(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *taskV1.Task) {
			builder.
				SetNillableType(r.typeConverter.ToEntity(req.Data.Type)).
				SetNillableTypeName(req.Data.TypeName).
				SetNillableTaskPayload(req.Data.TaskPayload).
				SetNillableCronSpec(req.Data.CronSpec).
				SetNillableEnable(req.Data.Enable).
				SetNillableRemark(req.Data.Remark).
				SetNillableUpdatedBy(req.Data.UpdatedBy).
				SetUpdatedAt(time.Now())

			if req.Data.TaskOptions != nil {
				builder.SetTaskOptions(req.Data.TaskOptions)
			}
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(task.FieldID, req.GetId()))
		},
	)

	return result, err
}

func (r *TaskRepo) Delete(ctx context.Context, req *taskV1.DeleteTaskRequest) error {
	if req == nil {
		return taskV1.ErrorBadRequest("invalid parameter")
	}

	if err := r.entClient.Client().Task.DeleteOneID(req.GetId()).Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return taskV1.ErrorNotFound("task not found")
		}

		r.log.Errorf(ctx, "delete one data failed: %s", err.Error())

		return taskV1.ErrorInternalServerError("delete failed")
	}

	return nil
}
