package data

import (
	"context"
	"time"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/timeutil"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/notificationdelivery"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"
)

// NotificationDeliveryRepo 通知投递台账仓储。
//
// 台账是"投递事实"的记录，生命周期只有两步：投递前 Create(SENDING)，投递后 MarkResult。
// 因此本仓不提供 Update/Delete——改一条已发生的投递记录等于伪造事实。
type NotificationDeliveryRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *bLogger.Helper

	mapper *mapper.CopierMapper[notificationV1.NotificationDelivery, ent.NotificationDelivery]

	channelConverter   *mapper.EnumTypeConverter[notificationV1.Channel, notificationdelivery.Channel]
	eventTypeConverter *mapper.EnumTypeConverter[notificationV1.EventType, notificationdelivery.EventType]
	statusConverter    *mapper.EnumTypeConverter[notificationV1.DeliveryStatus, notificationdelivery.Status]

	repository *entCrud.Repository[
		ent.NotificationDeliveryQuery, ent.NotificationDeliverySelect,
		ent.NotificationDeliveryCreate, ent.NotificationDeliveryCreateBulk,
		ent.NotificationDeliveryUpdate, ent.NotificationDeliveryUpdateOne,
		ent.NotificationDeliveryDelete,
		predicate.NotificationDelivery,
		notificationV1.NotificationDelivery, ent.NotificationDelivery,
	]
}

func NewNotificationDeliveryRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *NotificationDeliveryRepo {
	return newNotificationDeliveryRepo(ctx.NewLoggerHelper("notification-delivery/repo/admin-service"), entClient)
}

// NewNotificationDeliveryRepoForTest 供服务层 sqlite 测试构造真实仓储。
//
// 与既有 repo_testkit*.go 的差别：那些文件把生产构造器逐字段抄一遍再换 logger，
// 加字段时容易漏抄；这里让两者共用 newNotificationDeliveryRepo，构造路径不可能漂移。
func NewNotificationDeliveryRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *NotificationDeliveryRepo {
	return newNotificationDeliveryRepo(bLogger.NewHelper(bLogger.NopLogger()), entClient)
}

func newNotificationDeliveryRepo(log *bLogger.Helper, entClient *entCrud.EntClient[*ent.Client]) *NotificationDeliveryRepo {
	repo := &NotificationDeliveryRepo{
		log:       log,
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[notificationV1.NotificationDelivery, ent.NotificationDelivery](),
		channelConverter: mapper.NewEnumTypeConverter[notificationV1.Channel, notificationdelivery.Channel](
			notificationV1.Channel_name, notificationV1.Channel_value,
		),
		eventTypeConverter: mapper.NewEnumTypeConverter[notificationV1.EventType, notificationdelivery.EventType](
			notificationV1.EventType_name, notificationV1.EventType_value,
		),
		statusConverter: mapper.NewEnumTypeConverter[notificationV1.DeliveryStatus, notificationdelivery.Status](
			notificationV1.DeliveryStatus_name, notificationV1.DeliveryStatus_value,
		),
	}

	repo.init()

	return repo
}

func (r *NotificationDeliveryRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.NotificationDeliveryQuery, ent.NotificationDeliverySelect,
		ent.NotificationDeliveryCreate, ent.NotificationDeliveryCreateBulk,
		ent.NotificationDeliveryUpdate, ent.NotificationDeliveryUpdateOne,
		ent.NotificationDeliveryDelete,
		predicate.NotificationDelivery,
		notificationV1.NotificationDelivery, ent.NotificationDelivery,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.channelConverter.NewConverterPair())
	r.mapper.AppendConverters(r.eventTypeConverter.NewConverterPair())
	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
}

func (r *NotificationDeliveryRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*notificationV1.ListNotificationDeliveryResponse, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().NotificationDelivery.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &notificationV1.ListNotificationDeliveryResponse{Total: 0, Items: nil}, nil
	}

	return &notificationV1.ListNotificationDeliveryResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *NotificationDeliveryRepo) Get(ctx context.Context, id uint32) (*notificationV1.NotificationDelivery, error) {
	if id == 0 {
		return nil, adminV1.ErrorBadRequest("id is required")
	}

	entity, err := r.entClient.Client().NotificationDelivery.Get(ctx, id)
	if err != nil {
		r.log.Errorf(ctx, "get notification delivery [%d] failed: %s", id, err.Error())
		return nil, adminV1.ErrorNotFound("notification delivery not found")
	}

	return r.mapper.ToDTO(entity), nil
}

// Create 落一条投递台账（status 由 req 决定，未给时列默认 SENDING）。
func (r *NotificationDeliveryRepo) Create(ctx context.Context, req *notificationV1.NotificationDelivery) (*notificationV1.NotificationDelivery, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().NotificationDelivery.Create().
		SetNillableEventType(r.eventTypeConverter.ToEntity(req.EventType)).
		SetNillableChannel(r.channelConverter.ToEntity(req.Channel)).
		SetNillableChannelID(req.ChannelId).
		SetNillableRecipientUserID(req.RecipientUserId).
		SetNillableRelatedID(req.RelatedId).
		SetNillableTarget(req.Target).
		SetNillableStatus(r.statusConverter.ToEntity(req.Status)).
		SetNillableSentAt(timeutil.TimestamppbToTime(req.SentAt)).
		SetNillableCreatedBy(req.CreatedBy).
		SetNillableCreatedAt(timeutil.TimestamppbToTime(req.CreatedAt))

	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf(ctx, "insert notification delivery failed: %s", err.Error())
		return nil, adminV1.ErrorInternalServerError("insert notification delivery failed")
	}

	return r.mapper.ToDTO(entity), nil
}

// DeliveryOutcome 投递结果回写的内容。
type DeliveryOutcome struct {
	Status    notificationV1.DeliveryStatus
	LastError string
	// ChannelID 实际选中的渠道配置；nil 表示不修改（自选渠道时 Create 阶段还不知道）。
	ChannelID *uint32
	// SentAt 投递完成时间；FAILED/SKIPPED 传 nil，列保持空。
	SentAt *time.Time
}

// MarkResult 回写投递结果。
//
// 只按主键定位、只覆盖结果字段：台账其余部分是投递意图的快照，不应被结果回写篡改。
func (r *NotificationDeliveryRepo) MarkResult(ctx context.Context, id uint32, outcome *DeliveryOutcome) error {
	if id == 0 || outcome == nil {
		return adminV1.ErrorBadRequest("id and outcome are required")
	}

	update := r.entClient.Client().NotificationDelivery.UpdateOneID(id).
		SetStatus(*r.statusConverter.ToEntity(trans.Ptr(outcome.Status)))
	if outcome.LastError != "" {
		update.SetLastError(outcome.LastError)
	}
	if outcome.ChannelID != nil {
		update.SetChannelID(*outcome.ChannelID)
	}
	if outcome.SentAt != nil {
		update.SetSentAt(*outcome.SentAt)
	}

	if err := update.Exec(ctx); err != nil {
		r.log.Errorf(ctx, "mark notification delivery [%d] result failed: %s", id, err.Error())
		return adminV1.ErrorInternalServerError("mark notification delivery result failed")
	}

	return nil
}
