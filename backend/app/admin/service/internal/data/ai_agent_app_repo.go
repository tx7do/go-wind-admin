package data

import (
	"context"
	"time"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/aiagentapp"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"
)

type AiAgentAppRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *bLogger.Helper

	mapper *mapper.CopierMapper[aiV1.AiAgentApp, ent.AiAgentApp]

	repository *entCrud.Repository[
		ent.AiAgentAppQuery, ent.AiAgentAppSelect,
		ent.AiAgentAppCreate, ent.AiAgentAppCreateBulk,
		ent.AiAgentAppUpdate, ent.AiAgentAppUpdateOne,
		ent.AiAgentAppDelete,
		predicate.AiAgentApp,
		aiV1.AiAgentApp, ent.AiAgentApp,
	]
}

func NewAiAgentAppRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *AiAgentAppRepo {
	repo := &AiAgentAppRepo{
		log:       ctx.NewLoggerHelper("ai-agent-app/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[aiV1.AiAgentApp, ent.AiAgentApp](),
	}
	repo.init()
	return repo
}

func (r *AiAgentAppRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.AiAgentAppQuery, ent.AiAgentAppSelect,
		ent.AiAgentAppCreate, ent.AiAgentAppCreateBulk,
		ent.AiAgentAppUpdate, ent.AiAgentAppUpdateOne,
		ent.AiAgentAppDelete,
		predicate.AiAgentApp,
		aiV1.AiAgentApp, ent.AiAgentApp,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

// List 分页查询应用（通用过滤机制走 entCrud；enabled 由读路径显式回填）。
func (r *AiAgentAppRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*aiV1.ListAiAgentAppsResponse, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().AiAgentApp.Query()
	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		r.log.Errorf(ctx, "list ai agent apps failed: %s", err.Error())
		return nil, adminV1.ErrorInternalServerError("list ai agent apps failed")
	}
	if ret == nil {
		return &aiV1.ListAiAgentAppsResponse{Total: 0}, nil
	}

	// 实体 enabled 是值型 bool、DTO 是可选指针：copier 对这一形态失配（同渠道 Type 的教训），
	// 读路径批量回填。
	entities, err := r.entClient.Client().AiAgentApp.Query().
		Select(aiagentapp.FieldID, aiagentapp.FieldEnabled).
		All(ctx)
	if err != nil {
		r.log.Errorf(ctx, "backfill enabled failed: %s", err.Error())
	} else {
		enabledByID := make(map[int64]bool, len(entities))
		for _, e := range entities {
			enabledByID[e.ID] = e.Enabled
		}
		for _, it := range ret.Items {
			it.Enabled = trans.Ptr(enabledByID[it.GetId()])
		}
	}

	return &aiV1.ListAiAgentAppsResponse{Total: ret.Total, Items: ret.Items}, nil
}

// Get 按 ID 查询应用。
func (r *AiAgentAppRepo) Get(ctx context.Context, id int64) (*aiV1.AiAgentApp, error) {
	entity, err := r.entClient.Client().AiAgentApp.Get(ctx, id)
	if err != nil {
		r.log.Errorf(ctx, "get ai agent app [%d] failed: %s", id, err.Error())
		return nil, adminV1.ErrorNotFound("ai agent app not found")
	}
	return r.toDTO(entity), nil
}

// GetEntityByID 取应用实体（Run 流程要 graph 原文与 enabled，DTO 不够用）。
func (r *AiAgentAppRepo) GetEntityByID(ctx context.Context, id int64) (*ent.AiAgentApp, error) {
	entity, err := r.entClient.Client().AiAgentApp.Get(ctx, id)
	if err != nil {
		r.log.Errorf(ctx, "get ai agent app entity [%d] failed: %s", id, err.Error())
		return nil, adminV1.ErrorNotFound("ai agent app not found")
	}
	return entity, nil
}

func (r *AiAgentAppRepo) toDTO(entity *ent.AiAgentApp) *aiV1.AiAgentApp {
	dto := r.mapper.ToDTO(entity)
	dto.Enabled = trans.Ptr(entity.Enabled)
	return dto
}

// Create 创建应用；租户内 name 唯一约束冲突转成可读的 400。
func (r *AiAgentAppRepo) Create(ctx context.Context, data *aiV1.AiAgentApp, operatorID int64) (int64, error) {
	if data == nil {
		return 0, adminV1.ErrorBadRequest("invalid parameter")
	}
	if data.GetName() == "" {
		return 0, adminV1.ErrorBadRequest("name is required")
	}

	builder := r.entClient.Client().AiAgentApp.Create().
		SetName(data.GetName()).
		SetNillableDescription(data.Description).
		SetNillableGraph(data.Graph).
		SetEnabled(data.GetEnabled()).
		SetNillableTenantID(data.TenantId).
		SetCreatedBy(operatorID).
		SetCreatedAt(time.Now())

	created, err := builder.Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return 0, adminV1.ErrorBadRequest("agent app name already exists")
		}
		r.log.Errorf(ctx, "insert ai agent app failed: %s", err.Error())
		return 0, adminV1.ErrorInternalServerError("insert ai agent app failed")
	}
	return created.ID, nil
}

// Update 更新应用；updateMask 点名要改的字段，留空 = 全量按入参。
func (r *AiAgentAppRepo) Update(ctx context.Context, id int64, data *aiV1.AiAgentApp, updateMask []string, operatorID int64) error {
	if data == nil || id == 0 {
		return adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().AiAgentApp.Update().
		Where(aiagentapp.IDEQ(id))

	apply := func() *ent.AiAgentAppUpdate {
		return builder.
			SetNillableDescription(data.Description).
			SetNillableGraph(data.Graph).
			SetEnabled(data.GetEnabled()).
			SetNillableName(data.Name).
			SetNillableUpdatedBy(trans.Ptr(operatorID)).
			SetUpdatedAt(time.Now())
	}

	mask := map[string]bool{}
	for _, m := range updateMask {
		mask[m] = true
	}
	if len(mask) > 0 {
		b := builder
		if mask["name"] {
			b = b.SetNillableName(data.Name)
		}
		if mask["description"] {
			b = b.SetNillableDescription(data.Description)
		}
		if mask["graph"] {
			b = b.SetNillableGraph(data.Graph)
		}
		if mask["enabled"] {
			b = b.SetEnabled(data.GetEnabled())
		}
		b = b.SetNillableUpdatedBy(trans.Ptr(operatorID)).SetUpdatedAt(time.Now())
		if _, err := b.Save(ctx); err != nil {
			r.log.Errorf(ctx, "update ai agent app [%d] failed: %s", id, err.Error())
			return adminV1.ErrorInternalServerError("update ai agent app failed")
		}
		return nil
	}

	if _, err := apply().Save(ctx); err != nil {
		r.log.Errorf(ctx, "update ai agent app [%d] failed: %s", id, err.Error())
		return adminV1.ErrorInternalServerError("update ai agent app failed")
	}
	return nil
}

func (r *AiAgentAppRepo) Delete(ctx context.Context, id int64) error {
	if id == 0 {
		return adminV1.ErrorBadRequest("id is required")
	}
	if err := r.entClient.Client().AiAgentApp.DeleteOneID(id).Exec(ctx); err != nil {
		r.log.Errorf(ctx, "delete ai agent app [%d] failed: %s", id, err.Error())
		return adminV1.ErrorInternalServerError("delete ai agent app failed")
	}
	return nil
}
