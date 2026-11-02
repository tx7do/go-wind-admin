package service

import (
	"context"
	"strconv"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/emptypb"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"
	notificationChannelV1 "go-wind-admin/api/gen/go/notification_channel/service/v1"

	"go-wind-admin/pkg/mailtext"
	"go-wind-admin/pkg/middleware/auth"

	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/channel"
)

// NotificationChannelService 通知渠道管理（平台级配置）。
// 一期实现 EMAIL（SMTP）渠道：CRUD 留在本服务，"发"这件事已经交还给 NotificationService
// （渠道选择策略与 SMTP 调用收在 data/channel/email_sender.go 一处）。
type NotificationChannelService struct {
	adminV1.NotificationChannelServiceHTTPServer

	log  *bLogger.Helper
	repo *data.NotificationChannelRepo

	notifier Notifier
	mailer   *TransactionMailer
}

func NewNotificationChannelService(
	ctx *bootstrap.Context,
	repo *data.NotificationChannelRepo,
	notifier Notifier,
	mailer *TransactionMailer,
) *NotificationChannelService {
	return &NotificationChannelService{
		log:      ctx.NewLoggerHelper("notification-channel/service/admin-service"),
		repo:     repo,
		notifier: notifier,
		mailer:   mailer,
	}
}

func (s *NotificationChannelService) ListNotificationChannel(ctx context.Context, req *paginationV1.PagingRequest) (*notificationChannelV1.ListNotificationChannelResponse, error) {
	if err := requirePlatformAdmin(ctx, s.log, "notification channels"); err != nil {
		return nil, err
	}

	resp, err := s.repo.List(ctx, req)
	if err != nil {
		return nil, err
	}
	for _, it := range resp.GetItems() {
		it.Settings = sanitizeSettingsForRead(it.GetType().String(), it.GetSettings())
	}
	return resp, nil
}

func (s *NotificationChannelService) GetNotificationChannel(ctx context.Context, req *notificationChannelV1.GetNotificationChannelRequest) (*notificationChannelV1.NotificationChannel, error) {
	if req == nil || req.GetId() == 0 {
		return nil, adminV1.ErrorBadRequest("id is required")
	}

	if err := requirePlatformAdmin(ctx, s.log, "notification channels"); err != nil {
		return nil, err
	}

	dto, err := s.repo.Get(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	dto.Settings = sanitizeSettingsForRead(dto.GetType().String(), dto.GetSettings())
	return dto, nil
}

func (s *NotificationChannelService) CreateNotificationChannel(ctx context.Context, req *notificationChannelV1.CreateNotificationChannelRequest) (*notificationChannelV1.NotificationChannel, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	if err := requirePlatformAdmin(ctx, s.log, "notification channels"); err != nil {
		return nil, err
	}

	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	// 扩展渠道：settings 明文 JSON 按 schema 校验 + 敏感字段加密（存储态），写入 data 供 repo 落列
	if err = s.normalizeSettingsForWrite(ctx, req.Data); err != nil {
		return nil, err
	}

	id, err := s.repo.Create(ctx, req, operator.UserId)
	if err != nil {
		return nil, err
	}

	dto, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	// 与 Get/List 同一条脱敏口径：创建响应也不回显 sensitive 字段
	dto.Settings = sanitizeSettingsForRead(dto.GetType().String(), dto.GetSettings())
	return dto, nil
}

func (s *NotificationChannelService) UpdateNotificationChannel(ctx context.Context, req *notificationChannelV1.UpdateNotificationChannelRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	if err := requirePlatformAdmin(ctx, s.log, "notification channels"); err != nil {
		return nil, err
	}

	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	req.Data.UpdatedBy = trans.Ptr(operator.UserId)

	// 扩展渠道：settings 合并（sensitive 留空 = 保留已存密文）后写回 data
	if err = s.mergeSettingsForUpdate(ctx, req); err != nil {
		return nil, err
	}

	if err = s.repo.Update(ctx, req, operator.UserId); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *NotificationChannelService) DeleteNotificationChannel(ctx context.Context, req *notificationChannelV1.DeleteNotificationChannelRequest) (*emptypb.Empty, error) {
	if err := requirePlatformAdmin(ctx, s.log, "notification channels"); err != nil {
		return nil, err
	}

	if err := s.repo.Delete(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// SendTestEmail 向指定收件人发送测试邮件，验证渠道配置是否可用。
//
// 类型/启用状态/凭据一律不在这层判断：SendDirect 显式带 channel_id 时，
// EmailSender 会做同样的校验并把原因包进 error（渠道没配/没启用/类型不对），
// 因此这里能直接把原始错误回给配置页——正是这个功能唯一有用的输出。
func (s *NotificationChannelService) SendTestEmail(ctx context.Context, req *notificationChannelV1.SendTestEmailRequest) (*emptypb.Empty, error) {
	return s.sendTest(ctx, req.GetId(), req.GetRecipient())
}

// SendTest 按渠道类型分派一条测试消息（EMAIL 走测试邮件，SMS 发一条测试验证码）。
// 老入口 SendTestEmail 委托到这里；WEBHOOK 不支持在渠道页测试——它的端到端验证
// 在路由规则页，那里测的是"事件 → 渠道 → 台账"整条链，而渠道页只能证明配置本身。
func (s *NotificationChannelService) SendTest(ctx context.Context, req *notificationChannelV1.SendTestRequest) (*emptypb.Empty, error) {
	return s.sendTest(ctx, req.GetId(), req.GetRecipient())
}

// sendTest 两代测试入口的共同实现：查类型 → 分派 → SendDirect 显式点名渠道。
func (s *NotificationChannelService) sendTest(ctx context.Context, id uint32, recipient string) (*emptypb.Empty, error) {
	if id == 0 {
		return nil, adminV1.ErrorBadRequest("id is required")
	}
	if recipient == "" {
		return nil, adminV1.ErrorBadRequest("recipient is required")
	}

	if err := requirePlatformAdmin(ctx, s.log, "notification channels"); err != nil {
		return nil, err
	}

	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	dto, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	sendReq := &notificationV1.SendDirectNotificationRequest{
		ChannelId:      trans.Ptr(id),
		Target:         recipient,
		OperatorUserId: trans.Ptr(operator.UserId),
	}

	switch dto.GetType() {
	case notificationChannelV1.NotificationChannel_EMAIL:
		sendReq.EventType = notificationV1.EventType_CHANNEL_TEST_EMAIL
		sendReq.Title, sendReq.Content = renderChannelTestEmail(ctx, s.mailer, id, operator.UserId)

	case notificationChannelV1.NotificationChannel_SMS:
		// 短信没有"发一封空信"的形态：测试就是发一条真验证码。
		// 渠道未配路由规则（存量库没有 CHANNEL_TEST_SMS 行）也必须能发——
		// 显式点名 Channel=SMS，测试不依赖路由表。
		sendReq.EventType = notificationV1.EventType_CHANNEL_TEST_SMS
		sendReq.Channel = trans.Ptr(notificationV1.Channel_SMS)
		code := generateVCode()
		sendReq.Title, sendReq.Content = mailtext.ChannelTestSms(ctx, id, operator.UserId)
		sendReq.Params = map[string]string{"code": code}

	case notificationChannelV1.NotificationChannel_WEBHOOK:
		return nil, adminV1.ErrorBadRequest("webhook channels are tested from the notification-rules page")

	default:
		// EXT_* 扩展渠道：按 schema 判定是否要求 target，正文走通用测试文案
		schema := channel.SettingsSchemaByType(dto.GetType().String())
		if schema == nil {
			return nil, adminV1.ErrorBadRequest("unsupported channel type %s", dto.GetType().String())
		}
		if schema.NeedsTarget && recipient == "" {
			return nil, adminV1.ErrorBadRequest("recipient is required (%s)", schema.TargetLabel)
		}
		sendReq.EventType = notificationV1.EventType_CHANNEL_TEST_EXT
		sendReq.Channel = trans.Ptr(notificationV1.Channel_EXT)
		sendReq.Title, sendReq.Content = mailtext.ChannelTestExt(ctx, operator.UserId)

	}

	if _, err = s.notifier.SendDirect(ctx, sendReq); err != nil {
		s.log.Errorf(ctx, "send test via channel [%d] to [%s] failed: %v", id, recipient, err)
		return nil, adminV1.ErrorBadRequest("%s", "send test message failed: "+err.Error())
	}

	s.log.Infof(ctx, "test message sent via channel [%d] to [%s] by operator [%d]", id, recipient, operator.UserId)
	return &emptypb.Empty{}, nil
}

func itoa(v uint32) string {
	return strconv.FormatUint(uint64(v), 10)
}

// normalizeSettingsForWrite 创建时把 settings 明文 JSON 规范化为存储态
// （required 校验 + sensitive 字段加密）；非扩展渠道要求 settings 为空。
func (s *NotificationChannelService) normalizeSettingsForWrite(ctx context.Context, data *notificationChannelV1.NotificationChannel) error {
	typeName := data.GetType().String()
	if channel.SettingsSchemaByType(typeName) == nil {
		if data.GetSettings() != "" {
			return adminV1.ErrorBadRequest("settings only applies to EXT_* channel types")
		}
		return nil
	}
	storeJSON, err := channel.NormalizeSettingsForStore(typeName, data.GetSettings())
	if err != nil {
		return adminV1.ErrorBadRequest("%s", err.Error())
	}
	data.Settings = trans.Ptr(storeJSON)
	return nil
}

// mergeSettingsForUpdate 更新时的 settings 合并（sensitive 留空 = 保留已存密文）。
func (s *NotificationChannelService) mergeSettingsForUpdate(ctx context.Context, req *notificationChannelV1.UpdateNotificationChannelRequest) error {
	data := req.Data
	typeName := data.GetType().String()
	if channel.SettingsSchemaByType(typeName) == nil {
		if data.GetSettings() != "" {
			return adminV1.ErrorBadRequest("settings only applies to EXT_* channel types")
		}
		return nil
	}
	_, oldStoreJSON, _, err := s.repo.GetSettingsRaw(ctx, req.GetId())
	if err != nil {
		return err
	}
	storeJSON, err := channel.MergeSettingsForStore(typeName, oldStoreJSON, data.GetSettings())
	if err != nil {
		return adminV1.ErrorBadRequest("%s", err.Error())
	}
	data.Settings = trans.Ptr(storeJSON)
	return nil
}

// sanitizeSettingsForRead 读回显脱敏（sensitive 字段值恒空串）。
func sanitizeSettingsForRead(typeName, storeJSON string) *string {
	if storeJSON == "" {
		return nil
	}
	return trans.Ptr(channel.SanitizeSettingsForDTO(typeName, storeJSON))
}

// ListChannelSchemas 全量扩展渠道 schema（前端动态表单的数据源）。
func (s *NotificationChannelService) ListChannelSchemas(ctx context.Context, _ *emptypb.Empty) (*notificationChannelV1.ListChannelSchemasResponse, error) {
	if err := requirePlatformAdmin(ctx, s.log, "notification channel schemas"); err != nil {
		return nil, err
	}
	return &notificationChannelV1.ListChannelSchemasResponse{Items: channel.ListSettingsSchemas()}, nil
}
