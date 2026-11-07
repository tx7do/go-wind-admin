package data

import (
	"context"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	entCrud "github.com/tx7do/go-crud/entgo"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"

	biV1 "go-wind-admin/api/gen/go/bi/service/v1"
	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/bidashboardrevision"
)

// BiDashboardRevisionRepo BI 仪表盘版本快照薄仓储：只增不改不删（删除随主表级联），
// 只需 快照落库 / 按仪表盘分页拉取 / 取单行 三个动作，不走通用 CRUD 定式。
type BiDashboardRevisionRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *bLogger.Helper
}

func NewBiDashboardRevisionRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *BiDashboardRevisionRepo {
	return &BiDashboardRevisionRepo{
		entClient: entClient,
		log:       ctx.NewLoggerHelper("bi-dashboard-revision/repo/admin-service"),
	}
}

// Create 落一行快照。调用方（service 层）已取好旧值；失败由调用方决定是否阻断。
func (r *BiDashboardRevisionRepo) Create(ctx context.Context, revision *biV1.BiDashboardRevision) error {
	if revision == nil || revision.GetDashboardId() == 0 {
		return adminV1.ErrorBadRequest("invalid revision parameter")
	}
	builder := r.entClient.Client().BiDashboardRevision.Create().
		SetNillableDashboardID(trans.Ptr(revision.GetDashboardId())).
		SetNillableVersion(trans.Ptr(revision.GetVersion())).
		SetNillableName(revision.Name).
		SetNillablePayload(revision.Payload).
		SetNillableCreatedBy(revision.CreatedBy)
	if src := revision.GetSource(); src != "" {
		builder = builder.SetSource(bidashboardrevision.Source(src))
	}
	if _, err := builder.Save(ctx); err != nil {
		r.log.Errorf(ctx, "create bi dashboard revision failed: %s", err.Error())
		return adminV1.ErrorInternalServerError("create bi dashboard revision failed")
	}
	return nil
}

// List 按仪表盘拉取快照分页（新快照在前）。
func (r *BiDashboardRevisionRepo) List(ctx context.Context, dashboardId uint32, page *paginationV1.PagingRequest) ([]*biV1.BiDashboardRevision, uint32, error) {
	query := r.entClient.Client().BiDashboardRevision.Query().
		Where(bidashboardrevision.DashboardIDEQ(dashboardId))

	total, err := query.Count(ctx)
	if err != nil {
		r.log.Errorf(ctx, "count bi dashboard revisions failed: %s", err.Error())
		return nil, 0, adminV1.ErrorInternalServerError("count bi dashboard revisions failed")
	}
	if total == 0 {
		return []*biV1.BiDashboardRevision{}, 0, nil
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
		Order(ent.Desc(bidashboardrevision.FieldVersion, bidashboardrevision.FieldID)).
		Offset(int((pageNo - 1) * pageSize)).
		Limit(int(pageSize)).
		All(ctx)
	if err != nil {
		r.log.Errorf(ctx, "list bi dashboard revisions failed: %s", err.Error())
		return nil, 0, adminV1.ErrorInternalServerError("list bi dashboard revisions failed")
	}

	items := make([]*biV1.BiDashboardRevision, 0, len(rows))
	for _, row := range rows {
		items = append(items, revisionEntityToDTO(row))
	}
	return items, uint32(total), nil
}

// Get 取单个快照（恢复时校验归属）。
func (r *BiDashboardRevisionRepo) Get(ctx context.Context, id uint32) (*biV1.BiDashboardRevision, error) {
	row, err := r.entClient.Client().BiDashboardRevision.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, adminV1.ErrorNotFound("bi dashboard revision not found")
		}
		r.log.Errorf(ctx, "get bi dashboard revision failed: %s", err.Error())
		return nil, adminV1.ErrorInternalServerError("get bi dashboard revision failed")
	}
	return revisionEntityToDTO(row), nil
}

func revisionEntityToDTO(row *ent.BiDashboardRevision) *biV1.BiDashboardRevision {
	dto := &biV1.BiDashboardRevision{
		Id: row.ID,
	}
	if row.DashboardID != nil {
		dto.DashboardId = *row.DashboardID
	}
	if row.Version != nil {
		dto.Version = *row.Version
	}
	if row.Name != nil {
		dto.Name = trans.Ptr(*row.Name)
	}
	if row.Payload != nil {
		dto.Payload = trans.Ptr(*row.Payload)
	}
	if row.Source != nil {
		dto.Source = trans.Ptr(string(*row.Source))
	}
	if row.CreatedBy != nil {
		dto.CreatedBy = trans.Ptr(*row.CreatedBy)
	}
	if row.CreatedAt != nil {
		dto.CreatedAt = trans.Ptr(row.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	return dto
}
