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
	"go-wind-admin/app/admin/service/internal/data/ent/monitoralertrule"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	monitorAlertV1 "go-wind-admin/api/gen/go/monitor_alert/service/v1"
	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"
)

// MonitorAlertRuleRepo 监控告警规则仓储：指标阈值 → 触发通知，平台全局。
type MonitorAlertRuleRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *bLogger.Helper

	mapper           *mapper.CopierMapper[monitorAlertV1.MonitorAlertRule, ent.MonitorAlertRule]
	metricConverter  *mapper.EnumTypeConverter[monitorAlertV1.MonitorMetric, monitoralertrule.Metric]
	opConverter      *mapper.EnumTypeConverter[monitorAlertV1.AlertOp, monitoralertrule.Op]
	channelConverter *mapper.EnumTypeConverter[notificationV1.Channel, monitoralertrule.Channel]

	repository *entCrud.Repository[
		ent.MonitorAlertRuleQuery, ent.MonitorAlertRuleSelect,
		ent.MonitorAlertRuleCreate, ent.MonitorAlertRuleCreateBulk,
		ent.MonitorAlertRuleUpdate, ent.MonitorAlertRuleUpdateOne,
		ent.MonitorAlertRuleDelete,
		predicate.MonitorAlertRule,
		monitorAlertV1.MonitorAlertRule, ent.MonitorAlertRule,
	]
}

func NewMonitorAlertRuleRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *MonitorAlertRuleRepo {
	return newMonitorAlertRuleRepo(ctx.NewLoggerHelper("monitor-alert/repo/admin-service"), entClient)
}

func NewMonitorAlertRuleRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *MonitorAlertRuleRepo {
	return newMonitorAlertRuleRepo(bLogger.NewHelper(bLogger.NopLogger()), entClient)
}

func newMonitorAlertRuleRepo(log *bLogger.Helper, entClient *entCrud.EntClient[*ent.Client]) *MonitorAlertRuleRepo {
	repo := &MonitorAlertRuleRepo{
		log:              log,
		entClient:        entClient,
		mapper:           mapper.NewCopierMapper[monitorAlertV1.MonitorAlertRule, ent.MonitorAlertRule](),
		metricConverter:  mapper.NewEnumTypeConverter[monitorAlertV1.MonitorMetric, monitoralertrule.Metric](monitorAlertV1.MonitorMetric_name, monitorAlertV1.MonitorMetric_value),
		opConverter:      mapper.NewEnumTypeConverter[monitorAlertV1.AlertOp, monitoralertrule.Op](monitorAlertV1.AlertOp_name, monitorAlertV1.AlertOp_value),
		channelConverter: mapper.NewEnumTypeConverter[notificationV1.Channel, monitoralertrule.Channel](notificationV1.Channel_name, notificationV1.Channel_value),
	}

	repo.init()

	return repo
}

func (r *MonitorAlertRuleRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.MonitorAlertRuleQuery, ent.MonitorAlertRuleSelect,
		ent.MonitorAlertRuleCreate, ent.MonitorAlertRuleCreateBulk,
		ent.MonitorAlertRuleUpdate, ent.MonitorAlertRuleUpdateOne,
		ent.MonitorAlertRuleDelete,
		predicate.MonitorAlertRule,
		monitorAlertV1.MonitorAlertRule, ent.MonitorAlertRule,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.metricConverter.NewConverterPair())
	r.mapper.AppendConverters(r.opConverter.NewConverterPair())
	r.mapper.AppendConverters(r.channelConverter.NewConverterPair())
}

func (r *MonitorAlertRuleRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*monitorAlertV1.ListMonitorAlertRuleResponse, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().MonitorAlertRule.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &monitorAlertV1.ListMonitorAlertRuleResponse{Total: 0, Items: nil}, nil
	}

	return &monitorAlertV1.ListMonitorAlertRuleResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

// ListEnabled 取全部启用规则（评估器每轮用，量小不分页）。
func (r *MonitorAlertRuleRepo) ListEnabled(ctx context.Context) ([]*monitorAlertV1.MonitorAlertRule, error) {
	entities, err := r.entClient.Client().MonitorAlertRule.Query().
		Where(monitoralertrule.IsEnabledEQ(true)).
		All(ctx)
	if err != nil {
		r.log.Errorf(ctx, "list enabled monitor alert rules failed: %s", err.Error())
		return nil, adminV1.ErrorInternalServerError("list monitor alert rules failed")
	}

	dtos := make([]*monitorAlertV1.MonitorAlertRule, 0, len(entities))
	for _, entity := range entities {
		dtos = append(dtos, r.mapper.ToDTO(entity))
	}
	return dtos, nil
}

func (r *MonitorAlertRuleRepo) Get(ctx context.Context, id uint64) (*monitorAlertV1.MonitorAlertRule, error) {
	if id == 0 {
		return nil, adminV1.ErrorBadRequest("id is required")
	}

	entity, err := r.entClient.Client().MonitorAlertRule.Get(ctx, id)
	if err != nil {
		r.log.Errorf(ctx, "get monitor alert rule [%d] failed: %s", id, err.Error())
		return nil, adminV1.ErrorNotFound("monitor alert rule not found")
	}

	return r.mapper.ToDTO(entity), nil
}

// Create 建一条规则；created_at 显式写入（mixin 列没有 DB 默认值，理由同规则表）。
func (r *MonitorAlertRuleRepo) Create(ctx context.Context, req *monitorAlertV1.MonitorAlertRule, operatorID uint64) (uint64, error) {
	if req == nil {
		return 0, adminV1.ErrorBadRequest("invalid request")
	}

	builder := r.entClient.Client().MonitorAlertRule.Create().
		SetNillableName(req.Name).
		SetNillableMetric(r.metricConverter.ToEntity(req.Metric)).
		SetNillableOp(r.opConverter.ToEntity(req.Op)).
		SetNillableThreshold(req.Threshold).
		SetNillableCooldownMinutes(req.CooldownMinutes).
		SetNillableChannel(r.channelConverter.ToEntity(req.Channel)).
		SetNillableTarget(req.Target).
		SetNillableIsEnabled(req.IsEnabled).
		SetNillableRemark(req.Remark).
		SetCreatedAt(time.Now())

	if operatorID != 0 {
		builder.SetCreatedBy(operatorID)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf(ctx, "insert monitor alert rule failed: %s", err.Error())
		return 0, adminV1.ErrorInternalServerError("insert monitor alert rule failed")
	}

	return created.ID, nil
}

func (r *MonitorAlertRuleRepo) Update(ctx context.Context, req *monitorAlertV1.UpdateMonitorAlertRuleRequest, operatorID uint64) error {
	if req == nil || req.Data == nil {
		return adminV1.ErrorBadRequest("invalid request")
	}
	if req.GetId() == 0 {
		return adminV1.ErrorBadRequest("id is required")
	}

	builder := r.entClient.Client().MonitorAlertRule.Update()
	return r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *monitorAlertV1.MonitorAlertRule) {
			builder.
				SetNillableName(req.Data.Name).
				SetNillableMetric(r.metricConverter.ToEntity(req.Data.Metric)).
				SetNillableOp(r.opConverter.ToEntity(req.Data.Op)).
				SetNillableThreshold(req.Data.Threshold).
				SetNillableCooldownMinutes(req.Data.CooldownMinutes).
				SetNillableChannel(r.channelConverter.ToEntity(req.Data.Channel)).
				SetNillableTarget(req.Data.Target).
				SetNillableIsEnabled(req.Data.IsEnabled).
				SetNillableRemark(req.Data.Remark).
				SetNillableUpdatedBy(trans.Ptr(operatorID)).
				SetUpdatedAt(time.Now())
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(monitoralertrule.FieldID, req.GetId()))
		},
	)
}

// MarkFiring 评估器回写规则状态：firing 状态、本次指标值、告警时间。
// 独立成一个写路径（不走 Update 的掩码语义）：评估回写是系统行为，
// 与管理员的编辑互不覆盖——掩码更新永不触碰这三列，本方法只写这三列。
func (r *MonitorAlertRuleRepo) MarkFiring(ctx context.Context, id uint64, firing bool, value *float64, alertedAt *time.Time) error {
	builder := r.entClient.Client().MonitorAlertRule.UpdateOneID(id).
		SetLastFiring(firing).
		SetNillableLastValue(value)
	builder.SetNillableLastAlertedAt(alertedAt)
	if _, err := builder.Save(ctx); err != nil {
		r.log.Errorf(ctx, "mark monitor alert rule [%d] firing=%v failed: %s", id, firing, err.Error())
		return adminV1.ErrorInternalServerError("update monitor alert rule failed")
	}
	return nil
}

func (r *MonitorAlertRuleRepo) Delete(ctx context.Context, id uint64) error {
	if id == 0 {
		return adminV1.ErrorBadRequest("id is required")
	}

	if err := r.entClient.Client().MonitorAlertRule.DeleteOneID(id).Exec(ctx); err != nil {
		r.log.Errorf(ctx, "delete monitor alert rule [%d] failed: %s", id, err.Error())
		return adminV1.ErrorInternalServerError("delete monitor alert rule failed")
	}

	return nil
}
