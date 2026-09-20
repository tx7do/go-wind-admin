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
// 台账是"投递事实"的记录，生命周期只有三步：投递前 Create(SENDING)，投递后 MarkResult，
// 外加 SweepStaleSending 给"再也不会有结论"的行补一个终态。
// 因此本仓不提供 Update/Delete——改一条已发生的投递记录等于伪造事实；
// 清扫之所以不算伪造，见它的注释。
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
//
// attempts 不从 req 取：它是结果列，只有 MarkAttempted / MarkResult 写得动
// （新建时靠列默认 0），否则"台账不提供 Update"这条约束就从旁边漏了个洞。
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
		SetNillableRequestID(req.RequestId).
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
	// Attempts 已尝试次数；nil 表示不修改（异步路径在拨号前用 MarkAttempted 写过，回写时就不再碰）。
	Attempts *uint32
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
	if outcome.Attempts != nil {
		update.SetAttempts(*outcome.Attempts)
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

// MarkAttempted 在拨号之前把"第 attempts 次尝试已经开始"写进台账。
//
// 为什么不跟结果一起写：进程在 SMTP 握手中途被杀，这一行会永远停在 SENDING，
// 而 attempts 是那次"发了一半"唯一留得下来的痕迹（没有它，重试到第几次、有没有真的拨过号
// 全都看不出来，异步派发就成了黑盒）。
func (r *NotificationDeliveryRepo) MarkAttempted(ctx context.Context, id uint32, attempts uint32) error {
	if id == 0 {
		return adminV1.ErrorBadRequest("id is required")
	}

	if err := r.entClient.Client().NotificationDelivery.UpdateOneID(id).
		SetAttempts(attempts).
		Exec(ctx); err != nil {
		r.log.Errorf(ctx, "mark notification delivery [%d] attempted failed: %s", id, err.Error())
		return adminV1.ErrorInternalServerError("mark notification delivery attempt failed")
	}

	return nil
}

// SweepStaleSending 把 created_at 早于 before 却仍停在 SENDING 的台账行落 FAILED，返回本次清扫条数。
//
// 这一条为什么不算"伪造事实"：MarkResult 拒绝的是**改写已有结论**，而这里的行压根没有结论——
// 它既没发出去也没失败记录，留着 SENDING 只会让台账页出现一排"永远在发"的行。
// 落的是"超期未结算"这一确实发生过的事实，attempts 与 last_error 里最后一次真实报错都保留。
//
// 为什么用"查 ID → UpdateMany 再带 status 谓词"而不是逐行 MarkResult：
// 逐行写法把"这一行现在是什么状态"读进内存后无条件覆盖，撞上正在收尾的异步 handler
// 就会把刚写好的 SENT 改回 FAILED。把 status==SENDING 放进 WHERE，比较就交给 DB 在同一条
// 语句里做，抢跑的那一行自然不命中、也就不会被扫。
//
// 判据用 created_at 而不是 updated_at：台账这张表的 updated_at 从来没被写过
// （mixin 的列是 Optional 无默认，MarkResult/MarkAttempted 都不碰它），拿它当锚等于把所有行
// 判成"从未更新"；而生成的 (status, created_at) 索引正好服务这条查询。
// 异步派发的尝试预算本身也有界（4 次 × 30s + 退避 ≈ 4 分钟，见 NotificationService 的常量注释），
// 从"意图产生"起算的超期阈值只要明显大于它就是安全的。
func (r *NotificationDeliveryRepo) SweepStaleSending(ctx context.Context, before time.Time, reason string, limit int) (int, error) {
	if limit <= 0 {
		return 0, adminV1.ErrorBadRequest("limit must be positive")
	}

	// 比较侧统一成 UTC：created_at 是经 timestamppb 往返写进去的（渲染成 UTC），
	// 而调用方传进来的 time.Now() 带本地时区。Postgres 的 timestamptz 按瞬间比较无所谓，
	// 但 SQLite 驱动把时间渲染成带时区的文本，混用两种渲染就退化成了字典序比较 ——
	// 本仓的 sqlite 回归测试里，"一分钟前"刚落的行会被判成"十五分钟前"并被扫掉。
	before = before.UTC()

	ids, err := r.entClient.Client().NotificationDelivery.Query().
		Where(
			notificationdelivery.StatusEQ(notificationdelivery.StatusSending),
			notificationdelivery.CreatedAtLT(before),
		).
		Limit(limit).
		IDs(ctx)
	if err != nil {
		r.log.Errorf(ctx, "scan stale notification deliveries failed: %s", err.Error())
		return 0, adminV1.ErrorInternalServerError("scan stale notification deliveries failed")
	}
	if len(ids) == 0 {
		return 0, nil
	}

	count, err := r.entClient.Client().NotificationDelivery.Update().
		Where(
			notificationdelivery.IDIn(ids...),
			notificationdelivery.StatusEQ(notificationdelivery.StatusSending),
		).
		SetStatus(notificationdelivery.StatusFailed).
		SetLastError(reason).
		Save(ctx)
	if err != nil {
		r.log.Errorf(ctx, "sweep stale notification deliveries failed: %s", err.Error())
		return 0, adminV1.ErrorInternalServerError("sweep stale notification deliveries failed")
	}

	return count, nil
}
