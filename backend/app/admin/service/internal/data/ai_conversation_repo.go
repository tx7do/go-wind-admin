package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/aiconversation"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"
)

type AiConversationRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *bLogger.Helper

	mapper *mapper.CopierMapper[aiV1.AiConversation, ent.AiConversation]

	repository *entCrud.Repository[
		ent.AiConversationQuery, ent.AiConversationSelect,
		ent.AiConversationCreate, ent.AiConversationCreateBulk,
		ent.AiConversationUpdate, ent.AiConversationUpdateOne,
		ent.AiConversationDelete,
		predicate.AiConversation,
		aiV1.AiConversation, ent.AiConversation,
	]
}

func NewAiConversationRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *AiConversationRepo {
	repo := &AiConversationRepo{
		log:       ctx.NewLoggerHelper("ai_conversation/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[aiV1.AiConversation, ent.AiConversation](),
	}

	repo.init()

	return repo
}

func (r *AiConversationRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.AiConversationQuery, ent.AiConversationSelect,
		ent.AiConversationCreate, ent.AiConversationCreateBulk,
		ent.AiConversationUpdate, ent.AiConversationUpdateOne,
		ent.AiConversationDelete,
		predicate.AiConversation,
		aiV1.AiConversation, ent.AiConversation,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

func (r *AiConversationRepo) Count(ctx context.Context, req *paginationV1.PagingRequest) (*aiV1.CountAiConversationResponse, error) {
	builder := r.entClient.Client().AiConversation.Query()

	whereSelectors, _, _ := r.repository.BuildListSelectorWithPaging(builder, req)
	if len(whereSelectors) != 0 {
		builder.Modify(whereSelectors...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query ai conversation count failed: %s", err.Error())
		return nil, aiV1.ErrorInternalServerError("query ai conversation count failed")
	}

	return &aiV1.CountAiConversationResponse{Count: int64(count)}, nil
}

// List 按分页条件查询；ownerUserID > 0 时强制只返回该用户的会话（聊天场景不可信前端过滤）。
func (r *AiConversationRepo) List(ctx context.Context, req *paginationV1.PagingRequest, ownerUserID int64) (*aiV1.ListAiConversationResponse, error) {
	if req == nil {
		return nil, aiV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().AiConversation.Query()
	if ownerUserID > 0 {
		builder.Where(aiconversation.UserIDEQ(ownerUserID))
	}

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &aiV1.ListAiConversationResponse{Total: 0, Items: nil}, nil
	}

	return &aiV1.ListAiConversationResponse{Total: ret.Total, Items: ret.Items}, nil
}

func (r *AiConversationRepo) Get(ctx context.Context, req *aiV1.GetAiConversationRequest) (*aiV1.AiConversation, error) {
	if req == nil {
		return nil, aiV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().AiConversation.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *aiV1.GetAiConversationRequest_Id:
		whereCond = append(whereCond, aiconversation.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, nil
}

// GetEntityByID 取实体（service 层做归属校验用）。
func (r *AiConversationRepo) GetEntityByID(ctx context.Context, id int64) (*ent.AiConversation, error) {
	entity, err := r.entClient.Client().AiConversation.Query().
		Where(aiconversation.IDEQ(id)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, aiV1.ErrorNotFound("ai conversation not found")
		}
		r.log.Errorf(ctx, "query ai conversation failed: %s", err.Error())
		return nil, aiV1.ErrorInternalServerError("query ai conversation failed")
	}
	return entity, nil
}

func (r *AiConversationRepo) Create(ctx context.Context, data *aiV1.AiConversation) (*ent.AiConversation, error) {
	if data == nil {
		return nil, aiV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().AiConversation.Create().
		SetNillableTitle(data.Title).
		SetNillableProviderID(data.ProviderId).
		SetNillableUserID(data.UserId).
		SetNillableTenantID(data.TenantId).
		SetNillableCreatedBy(data.CreatedBy).
		SetCreatedAt(time.Now())

	if data.Id != nil {
		builder.SetID(data.GetId())
	}

	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf(ctx, "insert ai conversation failed: %s", err.Error())
		return nil, aiV1.ErrorInternalServerError("insert ai conversation failed")
	}

	return entity, nil
}

func (r *AiConversationRepo) Update(ctx context.Context, req *aiV1.UpdateAiConversationRequest) error {
	if req == nil || req.Data == nil {
		return aiV1.ErrorBadRequest("invalid parameter")
	}
	if req.GetId() == 0 {
		return aiV1.ErrorBadRequest("id is required")
	}

	builder := r.entClient.Client().AiConversation.Update()

	if err := r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *aiV1.AiConversation) {
			builder.
				SetNillableTitle(req.Data.Title).
				SetNillableProviderID(req.Data.ProviderId).
				SetNillableLastMessageAt(tsOf(req.Data.LastMessageAt)).
				SetNillableUpdatedBy(req.Data.UpdatedBy).
				SetUpdatedAt(time.Now())
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(aiconversation.FieldID, req.GetId()))
		},
	); err != nil {
		r.log.Errorf(ctx, "update ai conversation failed: %s", err.Error())
		return err
	}

	return nil
}

// TouchLastMessage 更新最近消息时间（chat 主链路高频小写）。
func (r *AiConversationRepo) TouchLastMessage(ctx context.Context, id int64) error {
	err := r.entClient.Client().AiConversation.UpdateOneID(id).
		SetLastMessageAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Exec(ctx)
	if err != nil {
		r.log.Errorf(ctx, "touch ai conversation last message failed: %s", err.Error())
		return aiV1.ErrorInternalServerError("update ai conversation failed")
	}
	return nil
}

func (r *AiConversationRepo) Delete(ctx context.Context, req *aiV1.DeleteAiConversationRequest) error {
	if req == nil {
		return aiV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().AiConversation.Delete()

	var predicates []predicate.AiConversation
	switch req.QueryBy.(type) {
	default:
	case *aiV1.DeleteAiConversationRequest_Id:
		predicates = append(predicates, aiconversation.IDEQ(req.GetId()))
	}

	// messages 边是 Cascade：会话删除时消息行随删（schema 级声明）。
	if _, err := r.repository.Delete(ctx, builder, predicates...); err != nil {
		r.log.Errorf(ctx, "delete ai conversation failed: %s", err.Error())
		return err
	}

	return nil
}

// ToDTO 实体转 DTO（chat 主链路在 service 层组装响应时使用）。
func (r *AiConversationRepo) ToDTO(entity *ent.AiConversation) *aiV1.AiConversation {
	if entity == nil {
		return nil
	}
	return r.mapper.ToDTO(entity)
}
