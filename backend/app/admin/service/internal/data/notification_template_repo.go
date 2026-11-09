package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/notificationtemplate"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"
)

// NotificationTemplateRepo 通知模板仓储：可复用的标题/正文占位模板，平台全局，
// code 全局唯一（SendDirect 与测试渲染的引用锚）。
type NotificationTemplateRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *bLogger.Helper

	mapper *mapper.CopierMapper[notificationV1.NotificationTemplate, ent.NotificationTemplate]

	repository *entCrud.Repository[
		ent.NotificationTemplateQuery, ent.NotificationTemplateSelect,
		ent.NotificationTemplateCreate, ent.NotificationTemplateCreateBulk,
		ent.NotificationTemplateUpdate, ent.NotificationTemplateUpdateOne,
		ent.NotificationTemplateDelete,
		predicate.NotificationTemplate,
		notificationV1.NotificationTemplate, ent.NotificationTemplate,
	]
}

func NewNotificationTemplateRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *NotificationTemplateRepo {
	return newNotificationTemplateRepo(ctx.NewLoggerHelper("notification-template/repo/admin-service"), entClient)
}

func NewNotificationTemplateRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *NotificationTemplateRepo {
	return newNotificationTemplateRepo(bLogger.NewHelper(bLogger.NopLogger()), entClient)
}

func newNotificationTemplateRepo(log *bLogger.Helper, entClient *entCrud.EntClient[*ent.Client]) *NotificationTemplateRepo {
	repo := &NotificationTemplateRepo{
		log:       log,
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[notificationV1.NotificationTemplate, ent.NotificationTemplate](),
	}

	repo.init()

	return repo
}

func (r *NotificationTemplateRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.NotificationTemplateQuery, ent.NotificationTemplateSelect,
		ent.NotificationTemplateCreate, ent.NotificationTemplateCreateBulk,
		ent.NotificationTemplateUpdate, ent.NotificationTemplateUpdateOne,
		ent.NotificationTemplateDelete,
		predicate.NotificationTemplate,
		notificationV1.NotificationTemplate, ent.NotificationTemplate,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

func (r *NotificationTemplateRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*notificationV1.ListNotificationTemplateResponse, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().NotificationTemplate.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &notificationV1.ListNotificationTemplateResponse{Total: 0, Items: nil}, nil
	}

	return &notificationV1.ListNotificationTemplateResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *NotificationTemplateRepo) Get(ctx context.Context, id uint64) (*notificationV1.NotificationTemplate, error) {
	if id == 0 {
		return nil, adminV1.ErrorBadRequest("id is required")
	}

	entity, err := r.entClient.Client().NotificationTemplate.Get(ctx, id)
	if err != nil {
		r.log.Errorf(ctx, "get notification template [%d] failed: %s", id, err.Error())
		return nil, adminV1.ErrorNotFound("notification template not found")
	}

	return r.mapper.ToDTO(entity), nil
}

// GetByCode 按编码取模板；没有该行时返回 (nil, nil)——
// 与 GetByEventType 同一区分："没配这个模板"是配置事实，"查库失败"是基础设施故障，
// 两者压成同一种 error 会把 SendDirect 的排障方向引反。
func (r *NotificationTemplateRepo) GetByCode(ctx context.Context, code string) (*notificationV1.NotificationTemplate, error) {
	if code == "" {
		return nil, nil
	}

	entity, err := r.entClient.Client().NotificationTemplate.Query().
		Where(notificationtemplate.CodeEQ(code)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		r.log.Errorf(ctx, "query notification template by code [%s] failed: %s", code, err.Error())
		return nil, adminV1.ErrorInternalServerError("query notification template failed")
	}

	return r.mapper.ToDTO(entity), nil
}

// Create 建一条模板；code 全局唯一，撞码在写入前显式拒掉（唯一索引兜底）。
// created_at 显式写入，理由同规则表（mixin 列没有 DB 默认值）。
func (r *NotificationTemplateRepo) Create(ctx context.Context, req *notificationV1.NotificationTemplate, operatorID uint64) (uint64, error) {
	if req == nil {
		return 0, adminV1.ErrorBadRequest("invalid request")
	}
	if req.GetCode() == "" {
		return 0, adminV1.ErrorBadRequest("code is required")
	}

	exist, err := r.entClient.Client().NotificationTemplate.Query().
		Where(notificationtemplate.CodeEQ(req.GetCode())).
		Exist(ctx)
	if err != nil {
		r.log.Errorf(ctx, "check duplicated template code failed: %s", err.Error())
		return 0, adminV1.ErrorInternalServerError("check notification template failed")
	}
	if exist {
		return 0, adminV1.ErrorBadRequest("template code %s already exists", req.GetCode())
	}

	builder := r.entClient.Client().NotificationTemplate.Create().
		SetNillableName(req.Name).
		SetNillableCode(req.Code).
		SetNillableTitleTemplate(req.TitleTemplate).
		SetNillableContentTemplate(req.ContentTemplate).
		SetNillableIsEnabled(req.IsEnabled).
		SetNillableRemark(req.Remark).
		SetCreatedAt(time.Now())

	if operatorID != 0 {
		builder.SetCreatedBy(operatorID)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf(ctx, "insert notification template failed: %s", err.Error())
		return 0, adminV1.ErrorInternalServerError("insert notification template failed")
	}

	return created.ID, nil
}

func (r *NotificationTemplateRepo) Update(ctx context.Context, req *notificationV1.UpdateNotificationTemplateRequest, operatorID uint64) error {
	if req == nil || req.Data == nil {
		return adminV1.ErrorBadRequest("invalid request")
	}
	if req.GetId() == 0 {
		return adminV1.ErrorBadRequest("id is required")
	}

	// code 若在更新里出现且与当前行不同，查重（唯一索引兜底并发窗口）
	if req.Data.Code != nil {
		existing, err := r.entClient.Client().NotificationTemplate.Get(ctx, req.GetId())
		if err != nil {
			r.log.Errorf(ctx, "get notification template [%d] for code check failed: %s", req.GetId(), err.Error())
			return adminV1.ErrorNotFound("notification template not found")
		}
		if existing.Code != nil && *existing.Code != req.Data.GetCode() {
			exist, err := r.entClient.Client().NotificationTemplate.Query().
				Where(notificationtemplate.CodeEQ(req.Data.GetCode())).
				Exist(ctx)
			if err != nil {
				r.log.Errorf(ctx, "check duplicated template code failed: %s", err.Error())
				return adminV1.ErrorInternalServerError("check notification template failed")
			}
			if exist {
				return adminV1.ErrorBadRequest("template code %s already exists", req.Data.GetCode())
			}
		}
	}

	builder := r.entClient.Client().NotificationTemplate.Update()
	return r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *notificationV1.NotificationTemplate) {
			builder.
				SetNillableName(req.Data.Name).
				SetNillableCode(req.Data.Code).
				SetNillableTitleTemplate(req.Data.TitleTemplate).
				SetNillableContentTemplate(req.Data.ContentTemplate).
				SetNillableIsEnabled(req.Data.IsEnabled).
				SetNillableRemark(req.Data.Remark).
				SetNillableUpdatedBy(trans.Ptr(operatorID)).
				SetUpdatedAt(time.Now())
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(notificationtemplate.FieldID, req.GetId()))
		},
	)
}

func (r *NotificationTemplateRepo) Delete(ctx context.Context, id uint64) error {
	if id == 0 {
		return adminV1.ErrorBadRequest("id is required")
	}

	if err := r.entClient.Client().NotificationTemplate.DeleteOneID(id).Exec(ctx); err != nil {
		r.log.Errorf(ctx, "delete notification template [%d] failed: %s", id, err.Error())
		return adminV1.ErrorInternalServerError("delete notification template failed")
	}

	return nil
}
