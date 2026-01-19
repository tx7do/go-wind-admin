package data

import (
	"context"
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
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"
	"go-wind-admin/app/admin/service/internal/data/ent/task"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
)

type TaskRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper        *mapper.CopierMapper[adminV1.Task, ent.Task]
	typeConverter *mapper.EnumTypeConverter[adminV1.Task_Type, task.Type]

	repository *entCrud.Repository[
		ent.TaskQuery, ent.TaskSelect,
		ent.TaskCreate, ent.TaskCreateBulk,
		ent.TaskUpdate, ent.TaskUpdateOne,
		ent.TaskDelete,
		predicate.Task,
		adminV1.Task, ent.Task,
	]
}

func NewTaskRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *TaskRepo {
	repo := &TaskRepo{
		log:           ctx.NewLoggerHelper("task/repo/admin-service"),
		entClient:     entClient,
		mapper:        mapper.NewCopierMapper[adminV1.Task, ent.Task](),
		typeConverter: mapper.NewEnumTypeConverter[adminV1.Task_Type, task.Type](adminV1.Task_Type_name, adminV1.Task_Type_value),
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
		adminV1.Task, ent.Task,
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
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, adminV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *TaskRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*adminV1.ListTaskResponse, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Task.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &adminV1.ListTaskResponse{Total: 0, Items: nil}, nil
	}

	return &adminV1.ListTaskResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *TaskRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().Task.Query().
		Where(task.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, adminV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *TaskRepo) Get(ctx context.Context, req *adminV1.GetTaskRequest) (*adminV1.Task, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Task.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *adminV1.GetTaskRequest_Id:
		whereCond = append(whereCond, task.IDEQ(req.GetId()))

	case *adminV1.GetTaskRequest_TypeName:
		whereCond = append(whereCond, task.TypeNameEQ(req.GetTypeName()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

func (r *TaskRepo) Create(ctx context.Context, req *adminV1.CreateTaskRequest) (*adminV1.Task, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
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
		SetNillableCreatedAt(timeutil.TimestamppbToTime(req.Data.CreatedAt))

	if req.Data.CreatedAt == nil {
		builder.SetCreatedAt(time.Now())
	}

	if req.Data.TaskOptions != nil {
		builder.SetTaskOptions(req.Data.TaskOptions)
	}

	if req.Data.Id != nil {
		builder.SetID(req.GetData().GetId())
	}

	t, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert task failed: %s", err.Error())
		return nil, adminV1.ErrorInternalServerError("insert task failed")
	}

	return r.mapper.ToDTO(t), nil
}

func (r *TaskRepo) Update(ctx context.Context, req *adminV1.UpdateTaskRequest) (*adminV1.Task, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return nil, err
		}
		if !exist {
			createReq := &adminV1.CreateTaskRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	builder := r.entClient.Client().Debug().Task.UpdateOneID(req.GetId())
	result, err := r.repository.UpdateOne(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *adminV1.Task) {
			builder.
				SetNillableType(r.typeConverter.ToEntity(req.Data.Type)).
				SetNillableTypeName(req.Data.TypeName).
				SetNillableTaskPayload(req.Data.TaskPayload).
				SetNillableCronSpec(req.Data.CronSpec).
				SetNillableEnable(req.Data.Enable).
				SetNillableRemark(req.Data.Remark).
				SetNillableUpdatedBy(req.Data.UpdatedBy).
				SetNillableUpdatedAt(timeutil.TimestamppbToTime(req.Data.UpdatedAt))

			if req.Data.TaskOptions != nil {
				builder.SetTaskOptions(req.Data.TaskOptions)
			}

			if req.Data.UpdatedAt == nil {
				builder.SetUpdatedAt(time.Now())
			}
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(task.FieldID, req.GetId()))
		},
	)

	return result, err
}

func (r *TaskRepo) Delete(ctx context.Context, req *adminV1.DeleteTaskRequest) error {
	if req == nil {
		return adminV1.ErrorBadRequest("invalid parameter")
	}

	if err := r.entClient.Client().Task.DeleteOneID(req.GetId()).Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return adminV1.ErrorNotFound("task not found")
		}

		r.log.Errorf("delete one data failed: %s", err.Error())

		return adminV1.ErrorInternalServerError("delete failed")
	}

	return nil
}
