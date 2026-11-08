package data

import (
	"context"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"github.com/tx7do/go-utils/timeutil"
	"github.com/tx7do/go-utils/trans"

	entCrud "github.com/tx7do/go-crud/entgo"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"
	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/aiagentrevision"
)

// AiAgentRevisionRepo AI Agent 应用版本快照薄仓储：只增不改不删（删除随主表级联），
// 只需 快照落库 / 按应用分页拉取 / 取单行 三个动作（形态同 BI 的版本快照仓储）。
type AiAgentRevisionRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *bLogger.Helper
}

func NewAiAgentRevisionRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *AiAgentRevisionRepo {
	return &AiAgentRevisionRepo{
		entClient: entClient,
		log:       ctx.NewLoggerHelper("ai-agent-revision/repo/admin-service"),
	}
}

// Create 落一行快照。调用方（service 层）已取好旧值；失败由调用方决定是否阻断。
func (r *AiAgentRevisionRepo) Create(ctx context.Context, revision *aiV1.AiAgentRevision) error {
	if revision == nil || revision.GetAppId() == 0 {
		return adminV1.ErrorBadRequest("invalid revision parameter")
	}
	builder := r.entClient.Client().AiAgentRevision.Create().
		SetNillableAppID(trans.Ptr(revision.GetAppId())).
		SetNillableGraph(revision.Graph).
		SetNillableCreatedBy(revision.CreatedBy)
	if src := revision.GetSource(); src != "" {
		builder = builder.SetSource(aiagentrevision.Source(src))
	}
	if _, err := builder.Save(ctx); err != nil {
		r.log.Errorf(ctx, "create ai agent revision failed: %s", err.Error())
		return adminV1.ErrorInternalServerError("create ai agent revision failed")
	}
	return nil
}

// List 按应用拉取快照分页（新快照在前）。
func (r *AiAgentRevisionRepo) List(ctx context.Context, appID uint32, page *paginationV1.PagingRequest) ([]*aiV1.AiAgentRevision, uint32, error) {
	query := r.entClient.Client().AiAgentRevision.Query().
		Where(aiagentrevision.AppIDEQ(appID))

	total, err := query.Count(ctx)
	if err != nil {
		r.log.Errorf(ctx, "count ai agent revisions failed: %s", err.Error())
		return nil, 0, adminV1.ErrorInternalServerError("count ai agent revisions failed")
	}
	if total == 0 {
		return []*aiV1.AiAgentRevision{}, 0, nil
	}

	pageNo, pageSize := uint32(1), uint32(20)
	if page != nil {
		if page.GetPage() > 0 {
			pageNo = page.GetPage()
		}
		if page.GetPageSize() > 0 {
			pageSize = page.GetPageSize()
		}
	}

	rows, err := query.
		Order(ent.Desc(aiagentrevision.FieldCreatedAt, aiagentrevision.FieldID)).
		Offset(int((pageNo - 1) * pageSize)).
		Limit(int(pageSize)).
		All(ctx)
	if err != nil {
		r.log.Errorf(ctx, "list ai agent revisions failed: %s", err.Error())
		return nil, 0, adminV1.ErrorInternalServerError("list ai agent revisions failed")
	}

	items := make([]*aiV1.AiAgentRevision, 0, len(rows))
	for _, row := range rows {
		items = append(items, aiAgentRevisionEntityToDTO(row))
	}
	return items, uint32(total), nil
}

// Get 取单个快照（恢复时校验归属）。
func (r *AiAgentRevisionRepo) Get(ctx context.Context, id uint32) (*aiV1.AiAgentRevision, error) {
	row, err := r.entClient.Client().AiAgentRevision.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, adminV1.ErrorNotFound("ai agent revision not found")
		}
		r.log.Errorf(ctx, "get ai agent revision failed: %s", err.Error())
		return nil, adminV1.ErrorInternalServerError("get ai agent revision failed")
	}
	return aiAgentRevisionEntityToDTO(row), nil
}

func aiAgentRevisionEntityToDTO(row *ent.AiAgentRevision) *aiV1.AiAgentRevision {
	dto := &aiV1.AiAgentRevision{
		Id: trans.Ptr(row.ID),
	}
	if row.AppID != nil {
		dto.AppId = trans.Ptr(*row.AppID)
	}
	if row.Graph != nil {
		dto.Graph = trans.Ptr(*row.Graph)
	}
	if row.Source != nil {
		dto.Source = trans.Ptr(string(*row.Source))
	}
	if row.CreatedBy != nil {
		dto.CreatedBy = trans.Ptr(*row.CreatedBy)
	}
	if row.CreatedAt != nil {
		dto.CreatedAt = timeutil.TimeToTimestamppb(row.CreatedAt)
	}
	return dto
}
