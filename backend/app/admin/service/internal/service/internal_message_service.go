package service

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-crud/viewer"
	"github.com/tx7do/go-utils/aggregator"
	"github.com/tx7do/go-utils/id"
	"github.com/tx7do/go-utils/timeutil"
	"github.com/tx7do/go-utils/trans"
	"github.com/hibiken/asynq"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"github.com/tx7do/kratos-transport/transport/sse"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	internalMessageV1 "go-wind-admin/api/gen/go/internal_message/service/v1"
	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"

	"go-wind-admin/pkg/middleware/auth"
	appViewer "go-wind-admin/pkg/entgo/viewer"
	"go-wind-admin/pkg/sseevent"
	"go-wind-admin/pkg/task"
)

// TaskEnqueuer 投递一次性 asynq 任务的接口。
// 由 TaskService 实现（其 TaskScheduler 在 asynq 配置时非 nil），
// 未配置 asynq 时 InternalMessageService.taskEnqueuer 为 nil，广播回退到 goroutine。
type TaskEnqueuer interface {
	NewTask(typeName string, msg any, opts ...asynq.Option) error
}

const (
	// defaultBroadcastTimeout 全员广播 fan-out 的总超时。
	// 脱离请求的 HTTP ctx 后由该超时兜底，避免广播 goroutine 因个别慢操作无限期挂起。
	defaultBroadcastTimeout = 5 * time.Minute

	// broadcastUserPageSize 全员广播时按页拉取用户的页大小。
	// 不用 NoPaging 全量拉取，避免用户量大时把全表用户一次性读进内存。
	broadcastUserPageSize uint32 = 1000
)

type InternalMessagePublisher interface {
	Publish(ctx context.Context, streamId sse.StreamID, event *sse.Event)
	// TryPublish 非阻塞推送：流不存在或缓冲已满时立即返回 false，不阻塞调用方。
	// 用于消息广播，避免某个慢客户端的 SSE 流缓冲塞满时卡住整个广播 fan-out。
	TryPublish(ctx context.Context, streamId sse.StreamID, event *sse.Event) bool
}

// noopInternalMessagePublisher 是 InternalMessagePublisher 的空操作实现。
// 当 SSE 服务未配置时（NewSseServer 返回 nil，不会调用 RegisterInternalMessagePublisher），
// 用它作为默认值，避免 sendNotification 解引用 nil 接口导致 panic。
// 此时站内信仍会落库，只是不通过 SSE 实时推送。
type noopInternalMessagePublisher struct{}

func (noopInternalMessagePublisher) Publish(_ context.Context, _ sse.StreamID, _ *sse.Event) {}

func (noopInternalMessagePublisher) TryPublish(_ context.Context, _ sse.StreamID, _ *sse.Event) bool {
	return false
}

type InternalMessageService struct {
	adminV1.InternalMessageServiceHTTPServer

	log *bLogger.Helper

	internalMessageRepo          *data.InternalMessageRepo
	internalMessageCategoryRepo  *data.InternalMessageCategoryRepo
	internalMessageRecipientRepo *data.InternalMessageRecipientRepo
	userRepo                     data.UserRepo

	internalMessagePublisher InternalMessagePublisher
	taskEnqueuer             TaskEnqueuer
	notifier                 Notifier
	authenticator            *data.Authenticator
	clientType               authenticationV1.ClientType
}

// unwiredNotifier 是 Notifier 的未装配占位实现。
//
// 与 noopInternalMessagePublisher 相反，这里不能静默返回成功：站内信一旦改走缝，
// 缝没接上就等于消息凭空消失且无痕迹。返回 error 让既有的 per-recipient 日志与
// failCount 把它暴露出来——装配漏注册时表现为发送日志里一行明确原因。
type unwiredNotifier struct{}

func (unwiredNotifier) SendDirect(_ context.Context, _ *notificationV1.SendDirectNotificationRequest) (*notificationV1.SendNotificationResponse, error) {
	return nil, errors.New("notifier is not wired: internal message cannot be delivered through the notification domain")
}

func NewInternalMessageService(
	ctx *bootstrap.Context,
	internalMessageRepo *data.InternalMessageRepo,
	internalMessageCategoryRepo *data.InternalMessageCategoryRepo,
	internalMessageRecipientRepo *data.InternalMessageRecipientRepo,
	userRepo data.UserRepo,
	authenticator *data.Authenticator,
	clientType authenticationV1.ClientType,
) *InternalMessageService {
	return &InternalMessageService{
		log:                          ctx.NewLoggerHelper("internal-message/service/admin-service"),
		internalMessageRepo:          internalMessageRepo,
		internalMessageCategoryRepo:  internalMessageCategoryRepo,
		internalMessageRecipientRepo: internalMessageRecipientRepo,
		userRepo:                     userRepo,
		authenticator:                authenticator,
		clientType:                   clientType,
		// 默认空操作发布者：SSE 未配置时不 panic；配置后由 RegisterInternalMessagePublisher 覆盖
		internalMessagePublisher: noopInternalMessagePublisher{},
		// taskEnqueuer 默认 nil：asynq 未配置时广播回退到 goroutine；
		// 配置后由 RegisterTaskEnqueuer 覆盖
		taskEnqueuer: nil,
		// notifier 默认"未装配"实现（一律报错）：装配后由 RegisterNotifier 换成 NotificationService
		notifier: unwiredNotifier{},
	}
}

func (s *InternalMessageService) RegisterInternalMessagePublisher(internalMessagePublisher InternalMessagePublisher) {
	s.internalMessagePublisher = internalMessagePublisher
}

// RegisterNotifier 注入通知域的投递出口。
// 与 RegisterInternalMessagePublisher 同模式：构造期给占位实现，装配期换成真身，
// 因为 NotificationService 与 InternalMessageService 互为依赖（前者经 Registry 调后者的
// 渠道实现，后者经 Notifier 走前者的缝），谁都不能作为后者的构造参数。
func (s *InternalMessageService) RegisterNotifier(notifier Notifier) {
	s.notifier = notifier
}

// RegisterTaskEnqueuer 注入 asynq 任务入队能力。
// 在 NewAsynqServer 中由 asynq_server.go 调用（与 RegisterInternalMessagePublisher 同模式）。
func (s *InternalMessageService) RegisterTaskEnqueuer(enqueuer TaskEnqueuer) {
	s.taskEnqueuer = enqueuer
}

func (s *InternalMessageService) HandleAuthorize(r *http.Request, token string) error {
	//s.log.Debugf(context.Background(), "authorizing token: %s", token)
	//s.log.Debugf(context.Background(), "authorizing token HEADER: %s", req.Header.Get("Authorization"))

	resp, err := s.authenticator.Authenticate(context.Background(), &authenticationV1.ValidateTokenRequest{
		ClientType:    s.clientType,
		Token:         token,
		TokenCategory: authenticationV1.TokenCategory_ACCESS,
	})
	if err != nil {
		s.log.Errorf(context.Background(), "token authentication failed: %s", err)
		return err
	}

	if resp.GetIsBlocked() {
		s.log.Warnf(context.Background(), "token is blocked: %s", token)
		return authenticationV1.ErrorForbidden("token is blocked")
	}
	if !resp.GetIsValid() {
		s.log.Warnf(context.Background(), "token is invalid: %s", token)
		return authenticationV1.ErrorUnauthorized("invalid token")
	}

	// 越权校验：streamID 已从 access token 改为 userId，必须保证订阅者只能订阅自己的流，
	// 否则用户 A 可通过 ?stream=<userB_id> 收到 B 的站内信通知。
	// （此前 streamID 即 token 字符串、不可伪造，无需此校验。）
	tokenUserId := resp.GetPayload().GetUserId()
	streamParam := r.URL.Query().Get("stream")
	if streamParam == "" {
		return authenticationV1.ErrorForbidden("stream user mismatch")
	}
	streamUserId, err := strconv.ParseUint(streamParam, 10, 32)
	if err != nil {
		return authenticationV1.ErrorForbidden("stream user mismatch")
	}
	if uint32(streamUserId) != tokenUserId {
		s.log.Warnf(context.Background(), "stream user mismatch: token uid=%d, stream uid=%d", tokenUserId, streamUserId)
		return authenticationV1.ErrorForbidden("stream user mismatch")
	}

	s.log.Debugf(context.Background(), "token authenticated successfully, userId: [%d]", tokenUserId)

	return nil
}

func (s *InternalMessageService) HandleSubscribe(streamID sse.StreamID, _ *sse.Subscriber) {
	s.log.Infof(context.Background(), "subscriber [%s] connected", streamID)
}

func (s *InternalMessageService) extractRelationIDs(
	messages []*internalMessageV1.InternalMessage,
	categorySet aggregator.ResourceMap[uint32, *internalMessageV1.InternalMessageCategory],
) {
	for _, p := range messages {
		if p.GetCategoryId() > 0 {
			categorySet[p.GetCategoryId()] = nil
		}
	}
}

func (s *InternalMessageService) fetchRelationInfo(
	ctx context.Context,
	categorySet aggregator.ResourceMap[uint32, *internalMessageV1.InternalMessageCategory],
) error {
	if len(categorySet) > 0 {
		categoryIds := make([]uint32, 0, len(categorySet))
		for i := range categorySet {
			categoryIds = append(categoryIds, i)
		}

		categories, err := s.internalMessageCategoryRepo.ListCategoriesByIds(ctx, categoryIds)
		if err != nil {
			s.log.Errorf(context.Background(), "query internal message category err: %v", err)
			return err
		}

		for _, g := range categories {
			categorySet[g.GetId()] = g
		}
	}

	return nil
}

func (s *InternalMessageService) bindRelations(
	messages []*internalMessageV1.InternalMessage,
	categorySet aggregator.ResourceMap[uint32, *internalMessageV1.InternalMessageCategory],
) {
	aggregator.Populate(
		messages,
		categorySet,
		func(ou *internalMessageV1.InternalMessage) uint32 { return ou.GetCategoryId() },
		func(ou *internalMessageV1.InternalMessage, c *internalMessageV1.InternalMessageCategory) {
			ou.CategoryName = c.Name
		},
	)
}

func (s *InternalMessageService) enrichRelations(ctx context.Context, messages []*internalMessageV1.InternalMessage) error {
	var categorySet = make(aggregator.ResourceMap[uint32, *internalMessageV1.InternalMessageCategory])
	s.extractRelationIDs(messages, categorySet)
	if err := s.fetchRelationInfo(ctx, categorySet); err != nil {
		return err
	}
	s.bindRelations(messages, categorySet)
	return nil
}

func (s *InternalMessageService) ListMessage(ctx context.Context, req *paginationV1.PagingRequest) (*internalMessageV1.ListInternalMessageResponse, error) {
	resp, err := s.internalMessageRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	_ = s.enrichRelations(ctx, resp.Items)

	return resp, nil
}

func (s *InternalMessageService) GetMessage(ctx context.Context, req *internalMessageV1.GetInternalMessageRequest) (*internalMessageV1.InternalMessage, error) {
	resp, err := s.internalMessageRepo.Get(ctx, req)
	if err != nil {
		return nil, err
	}

	fakeItems := []*internalMessageV1.InternalMessage{resp}
	_ = s.enrichRelations(ctx, fakeItems)

	return resp, nil
}

func (s *InternalMessageService) CreateMessage(ctx context.Context, req *internalMessageV1.CreateInternalMessageRequest) (*emptypb.Empty, error) {
	if req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	if _, err = s.internalMessageRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *InternalMessageService) UpdateMessage(ctx context.Context, req *internalMessageV1.UpdateInternalMessageRequest) (*emptypb.Empty, error) {
	if req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.Id = trans.Ptr(req.GetId())

	req.Data.UpdatedBy = trans.Ptr(operator.UserId)
	if req.UpdateMask != nil {
		req.UpdateMask.Paths = append(req.UpdateMask.Paths, "updated_by")
	}

	if err = s.internalMessageRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *InternalMessageService) DeleteMessage(ctx context.Context, req *internalMessageV1.DeleteInternalMessageRequest) (*emptypb.Empty, error) {
	// 消息本体与收件记录同事务级联删除，避免留下孤儿收件行（收件箱出现无标题幽灵记录）。
	if err := s.internalMessageRecipientRepo.DeleteMessageWithRecipients(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// RevokeMessage 撤销某条消息
func (s *InternalMessageService) RevokeMessage(ctx context.Context, req *internalMessageV1.RevokeMessageRequest) (*emptypb.Empty, error) {
	// 消息本体删除与收件人撤销收在同一个事务里执行，避免半成功留下幽灵收件记录。
	if err := s.internalMessageRecipientRepo.RevokeMessageWithMessage(ctx, req); err != nil {
		s.log.Errorf(ctx, "revoke internal message failed: [%d][%d] %s", req.GetMessageId(), req.GetUserId(), err)
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// SendMessage 发送消息
func (s *InternalMessageService) SendMessage(ctx context.Context, req *internalMessageV1.SendMessageRequest) (*internalMessageV1.SendMessageResponse, error) {
	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	var msg *internalMessageV1.InternalMessage
	if msg, err = s.internalMessageRepo.Create(ctx, &internalMessageV1.CreateInternalMessageRequest{
		Data: &internalMessageV1.InternalMessage{
			Title:      req.Title,
			Content:    trans.Ptr(req.GetContent()),
			Status:     trans.Ptr(internalMessageV1.InternalMessage_PUBLISHED),
			Type:       trans.Ptr(req.GetType()),
			CategoryId: req.CategoryId,
			CreatedBy:  trans.Ptr(operator.GetUserId()),
			CreatedAt:  timeutil.TimeToTimestamppb(&now),
		},
	}); err != nil {
		s.log.Errorf(ctx, "create internal message failed: %s", err)
		return nil, err
	}

	if req.GetTargetAll() {
		// 全员广播：消息本体已落库，fan-out 投递改为 asynq 任务。
		// 进程重启后未完成的投递由 asynq 自动重试（而非旧 goroutine 路径的静默丢失），
		// 重试幂等性由 (message_id, recipient_user_id) 唯一约束 + CreateBulk 的
		// ON CONFLICT DO NOTHING 保证。asynq 未配置时回退到后台 goroutine。
		msgId := msg.GetId()
		senderId := msg.GetCreatedBy()
		title := msg.GetTitle()
		content := msg.GetContent()
		vc, _ := viewer.FromContext(ctx)

		if s.taskEnqueuer != nil {
			if err := s.taskEnqueuer.NewTask(task.BroadcastMessageTaskType, &task.BroadcastMessageTaskData{
				MessageId: msgId,
				TenantId:  operator.GetTenantId(),
			}); err != nil {
				s.log.Errorf(ctx, "enqueue broadcast task for message [%d] failed, falling back to goroutine: %s", msgId, err)
				s.fanoutBroadcastGoroutine(msgId, senderId, title, content, vc)
			}
		} else {
			s.fanoutBroadcastGoroutine(msgId, senderId, title, content, vc)
		}
	} else {
		// 定向发送：人数少，仍同步执行，但改走通知域的缝——
		// 路由表定渠道（INTERNAL_MESSAGE → INTERNAL），缝落台账 + 调 INTERNAL 渠道落收件行。
		// 部分失败照旧只记日志、不影响 HTTP 结论（收件箱与台账各自留有痕迹）。
		if req.RecipientUserId != nil {
			if err := s.deliverViaNotifier(ctx, msg.GetId(), req.GetRecipientUserId(), operator.GetUserId(), msg.GetTitle(), msg.GetContent()); err != nil {
				s.log.Errorf(ctx, "send message [%d] to user [%d] failed: %s", msg.GetId(), req.GetRecipientUserId(), err)
			}
		} else {
			var failCount int
			for _, uid := range req.TargetUserIds {
				if err := s.deliverViaNotifier(ctx, msg.GetId(), uid, operator.GetUserId(), msg.GetTitle(), msg.GetContent()); err != nil {
					failCount++
				}
			}
			if failCount > 0 {
				s.log.Warnf(ctx, "send message [%d]: %d/%d target users failed", msg.GetId(), failCount, len(req.TargetUserIds))
			}
		}
	}

	return &internalMessageV1.SendMessageResponse{
		MessageId: msg.GetId(),
	}, nil
}

// newMessageRecipient 构造一条收件记录。
// 状态直接写 RECEIVED 并落 received_at：SENT→RECEIVED 的流转本应由客户端确认送达
// （MarkNotificationsStatus），但该接口没有 HTTP 路由也无人调用，若写 SENT，
// 按 status=RECEIVED 过滤的收件箱/未读列表将永远查不到新消息。
//
// tenantId 必须由调用方显式传入（广播路径取收件用户自己的租户，定向路径取操作人租户）：
// 广播在 asynq handler 里以 SystemViewer 运行，go-crud TenantPrivacy 只在该上下文放行，
// 留空即落 DefaultTenantID=0，收件行会变成任何租户都读不到的孤儿行。
func newMessageRecipient(messageId, recipientUserId, senderUserId, tenantId uint32, now *time.Time, title, content string) *internalMessageV1.InternalMessageRecipient {
	return &internalMessageV1.InternalMessageRecipient{
		TenantId:        trans.Ptr(tenantId),
		MessageId:       trans.Ptr(messageId),
		RecipientUserId: trans.Ptr(recipientUserId),
		Status:          trans.Ptr(internalMessageV1.InternalMessageRecipient_RECEIVED),
		ReceivedAt:      timeutil.TimeToTimestamppb(now),
		CreatedBy:       trans.Ptr(senderUserId),
		CreatedAt:       timeutil.TimeToTimestamppb(now),
		Title:           trans.Ptr(title),
		Content:         trans.Ptr(content),
	}
}

// publishNotification 尽力而为地实时推送一条收件记录。
// 站内信以落库为准，推送失败（用户离线、缓冲已满）不影响投递，客户端重连后可从收件箱补取。
//
// data 必须用 protojson 而不是 encoding/json 序列化：后者按结构体 tag 出 snake_case 键
// （message_id / created_at），而三端通知面板读的是与 REST 收件箱同形的 camelCase，
// vue-element 的 handleSseNotification 因此在 `if (!data.messageId) return` 处直接退出——
// 桌面通知与未读计数全不触发且不报错。protojson 另外把 status 输出成枚举名而非 varint。
// （protojson 会在字段间插入随机空白，消费方只能 JSON.parse，不许比对字节。）
func (s *InternalMessageService) publishNotification(ctx context.Context, recipient *internalMessageV1.InternalMessageRecipient) {
	recipientJson, err := protojson.Marshal(recipient)
	if err != nil {
		s.log.Errorf(ctx, "marshal recipient failed, skip sse push: %s", err)
		return
	}

	// streamID 用 userId：同一用户的所有在线设备订阅同一条流，
	// 库的 stream fan-out 会把该事件投递给该流的全部 subscriber，因此只需单次 publish。
	// TryPublish 非阻塞推送：流不存在（用户无在线 SSE 连接）或缓冲已满时立即跳过，
	// 避免慢客户端阻塞发送方。
	streamId := strconv.FormatUint(uint64(recipient.GetRecipientUserId()), 10)
	if ok := s.internalMessagePublisher.TryPublish(ctx, sse.StreamID(streamId), &sse.Event{
		ID:    []byte(id.NewGUIDv4(false)),
		Data:  recipientJson,
		Event: []byte(sseevent.Notification),
	}); !ok {
		s.log.Debugf(ctx, "sse try publish skipped (stream not exist or buffer full): user=%d stream=%s", recipient.GetRecipientUserId(), streamId)
	}
}

// deliverViaNotifier 把一个定向收件人交给通知域投递：路由表决定渠道，缝负责台账与
// INTERNAL 渠道的实际落库（见 internal_message_sender.go）。
//
// 站内信自此没有"绕过缝"的第二条单收件人路径——这正是 P1 结束时它还是半吊子的地方：
// 消息发没发过、走的哪个渠道、成没成，此前只在收件箱表里有半个答案。
//
// 不显式传 Channel：让 eventChannels 成为唯一声明处，路由表被绕开时测试会红。
func (s *InternalMessageService) deliverViaNotifier(ctx context.Context, messageId, recipientUserId, operatorUserId uint32, title, content string) error {
	// Target 是收件用户 ID 的十进制字符串：缝要求调用方自己给出投递目标，
	// INTERNAL 渠道的"地址"就是这个人（台账里与脱敏邮箱并列时保持可读）。
	target := strconv.FormatUint(uint64(recipientUserId), 10)

	resp, err := s.notifier.SendDirect(ctx, &notificationV1.SendDirectNotificationRequest{
		EventType:       notificationV1.EventType_INTERNAL_MESSAGE,
		Target:          target,
		Title:           title,
		Content:         content,
		RecipientUserId: trans.Ptr(recipientUserId),
		OperatorUserId:  trans.Ptr(operatorUserId),
		RelatedId:       trans.Ptr(messageId),
	})
	if err != nil {
		s.log.Errorf(ctx, "deliver message [%d] to user [%d] via notification domain failed (delivery=%d): %s",
			messageId, recipientUserId, resp.GetDeliveryId(), err)
	}

	return err
}

// sendNotification 单个收件人：落库 + 实时推送。这是 INTERNAL 渠道的投递内核，
// 由 InternalMessageSender 经缝调用（全员广播的批量形态见 executeBroadcast）。
//
// 收件行的租户归属取 ctx viewer 而不是取参数：定向路径下 viewer 与操作人同租户（两者都来自
// 同一枚令牌的 tenant_id），广播路径由 AsyncBroadcastMessage 按任务 payload 重建 viewer，
// 于是两个入口共用一条规则，不存在"调用方把租户传错 → 收件行成任何租户都读不到的孤儿行"。
// viewer 缺失时按 0 落库并留下 error 日志（与 executeBroadcast 同口径）。
func (s *InternalMessageService) sendNotification(ctx context.Context, messageId, recipientUserId, senderUserId uint32, now *time.Time, title, content string) error {
	recipientTenantId := uint32(0)
	if vc, ok := viewer.FromContext(ctx); ok {
		recipientTenantId = uint32(vc.TenantID())
	} else {
		s.log.Errorf(ctx, "send message [%d] to user [%d]: no viewer in context, recipient will be written as tenant 0", messageId, recipientUserId)
	}

	recipient := newMessageRecipient(messageId, recipientUserId, senderUserId, recipientTenantId, now, title, content)

	var err error
	var entity *internalMessageV1.InternalMessageRecipient
	if entity, err = s.internalMessageRecipientRepo.Create(ctx, recipient); err != nil {
		s.log.Errorf(ctx, "send message failed, send to user failed, %s", err)
		return err
	}
	recipient.Id = entity.Id

	s.publishNotification(ctx, recipient)

	return nil
}

// executeBroadcast 执行全员广播 fan-out：按页拉取用户 + 分批幂等写入收件记录 + 逐条 SSE 推送。
// 由 AsyncBroadcastMessage（asynq handler）和 fanoutBroadcastGoroutine（回退路径）共用。
// 注意 viewer：ent 的 TenantPrivacy 在 viewer 缺失时会返回 error，调用方必须传入带 viewer 的 ctx。
//
// 收件行的 tenant_id 取自 ctx viewer 而非留空：本方法在 asynq 路径上以 SystemViewer 运行，
// 平台上下文不会自动补租户（go-crud 只在非平台上下文强制覆盖），留空即落 0，
// 租户用户的收件箱按自己的租户过滤后一行也读不到——静默丢投递、无报错无日志。
// 取 viewer 而不是取参数，是为了让"受众范围"与"落库租户"由同一个事实决定，不可能跑偏。
func (s *InternalMessageService) executeBroadcast(ctx context.Context, messageId, senderUserId uint32, title, content string) {
	now := time.Now()

	broadcastTenantId := uint32(0)
	if vc, ok := viewer.FromContext(ctx); ok {
		broadcastTenantId = uint32(vc.TenantID())
	} else {
		s.log.Errorf(ctx, "broadcast message [%d]: no viewer in context, recipients will be written as tenant 0", messageId)
	}

	var total int
	for page := uint32(1); ; page++ {
		users, err := s.userRepo.List(ctx, &paginationV1.PagingRequest{
			Page:     trans.Ptr(page),
			PageSize: trans.Ptr(broadcastUserPageSize),
		})
		if err != nil {
			s.log.Errorf(ctx, "broadcast message [%d]: list users (page %d) failed: %s", messageId, page, err)
			break
		}
		if len(users.GetItems()) == 0 {
			break
		}

		recipients := make([]*internalMessageV1.InternalMessageRecipient, 0, len(users.GetItems()))
		for _, user := range users.GetItems() {
			recipients = append(recipients, newMessageRecipient(messageId, user.GetId(), senderUserId, broadcastTenantId, &now, title, content))
		}

		// CreateBulk 用 ON CONFLICT DO NOTHING 幂等写入：asynq 重试时已落库的行会被忽略而非报错。
		// upsert 模式不返回实体，所以推送前先按页回读主键——SSE 载荷缺 id 会让 vue-element 的
		// 通知面板在第一行守卫处静默丢弃整场广播（详见仓储方法注释）。
		if err := s.internalMessageRecipientRepo.CreateBulk(ctx, recipients); err != nil {
			s.log.Errorf(ctx, "broadcast message [%d]: bulk insert recipients (page %d) partial fail: %s", messageId, page, err)
		}
		total += len(recipients)

		userIds := make([]uint32, 0, len(recipients))
		for _, recipient := range recipients {
			userIds = append(userIds, recipient.GetRecipientUserId())
		}
		recipientIds, err := s.internalMessageRecipientRepo.IdsByMessageAndRecipients(ctx, messageId, userIds)
		if err != nil {
			// 回读失败不中断：宁可推一条缺 id 的通知，也不丢掉整个广播。
			s.log.Errorf(ctx, "broadcast message [%d]: read back recipient ids (page %d) failed: %s", messageId, page, err)
		}
		for _, recipient := range recipients {
			recipient.Id = trans.Ptr(recipientIds[recipient.GetRecipientUserId()])
			s.publishNotification(ctx, recipient)
		}

		if len(users.GetItems()) < int(broadcastUserPageSize) {
			break
		}
	}

	s.log.Infof(ctx, "broadcast message [%d] to %d recipients done", messageId, total)
}

// fanoutBroadcastGoroutine 是 asynq 未配置时的回退路径：
// 在脱离请求的后台 ctx 上异步执行 fan-out（客户端断连不会中断投递）。
// viewer 从请求 ctx 取出后贴到后台 ctx，保持租户可见性。
func (s *InternalMessageService) fanoutBroadcastGoroutine(messageId, senderUserId uint32, title, content string, vc viewer.Context) {
	go func() {
		broadcastCtx, cancel := context.WithTimeout(context.Background(), defaultBroadcastTimeout)
		defer cancel()
		if vc != nil {
			broadcastCtx = viewer.WithContext(broadcastCtx, vc)
		}
		s.executeBroadcast(broadcastCtx, messageId, senderUserId, title, content)
	}()
}

// AsyncBroadcastMessage 是 asynq 广播任务的 handler。
// 从 payload 取 messageId，从 DB 取回消息本体后执行 fan-out。
// asynq 的重试机制保证进程重启后未完成的投递自动恢复；
// 幂等性由 (message_id, recipient_user_id) 唯一约束 + CreateBulk 的 ON CONFLICT DO NOTHING 保证。
// 注意：asynq handler 的 ctx 不携带请求期的 viewer，需按 payload 的发送方租户重建，
// 否则 ent 租户隐私层会拒绝查询。
func (s *InternalMessageService) AsyncBroadcastMessage(taskType string, taskData *task.BroadcastMessageTaskData) error {
	s.log.Infof(context.Background(), "AsyncBroadcastMessage [%s] messageId=%d tenantId=%d", taskType, taskData.MessageId, taskData.TenantId)

	// asynq handler 的 ctx 不携带请求期的 viewer，ent 的 TenantPrivacy 会拒绝无 viewer 的查询。
	// 平台管理员（tid==0）用 SystemViewer 保持"全平台受众"；租户管理员必须用本租户的 viewer，
	// 因为平台上下文不给 userRepo.List 注入租户谓词，全员广播会变成跨租户全平台投递。
	// 升级前已入队的旧任务无 tenant_id 字段，按 tid==0 处理，退化为升级前的行为。
	var ctx context.Context
	if taskData.TenantId == 0 {
		ctx = appViewer.NewSystemViewerContext(context.Background())
	} else {
		ctx = viewer.WithContext(context.Background(), appViewer.NewUserViewer(0, uint64(taskData.TenantId), 0, "", nil))
	}

	msg, err := s.internalMessageRepo.Get(ctx, &internalMessageV1.GetInternalMessageRequest{
		QueryBy: &internalMessageV1.GetInternalMessageRequest_Id{
			Id: taskData.MessageId,
		},
	})
	if err != nil {
		s.log.Errorf(ctx, "AsyncBroadcastMessage: get message [%d] failed: %s", taskData.MessageId, err)
		return err
	}

	s.executeBroadcast(ctx, taskData.MessageId, msg.GetCreatedBy(), msg.GetTitle(), msg.GetContent())

	return nil
}
