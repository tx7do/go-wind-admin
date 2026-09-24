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
)

// NotificationChannelService 通知渠道管理（平台级配置）。
// 一期实现 EMAIL（SMTP）渠道：CRUD 留在本服务，"发"这件事已经交还给 NotificationService
// （渠道选择策略与 SMTP 调用收在 data/channel/email_sender.go 一处）。
type NotificationChannelService struct {
	adminV1.NotificationChannelServiceHTTPServer

	log  *bLogger.Helper
	repo *data.NotificationChannelRepo

	notifier Notifier
}

func NewNotificationChannelService(
	ctx *bootstrap.Context,
	repo *data.NotificationChannelRepo,
	notifier Notifier,
) *NotificationChannelService {
	return &NotificationChannelService{
		log:      ctx.NewLoggerHelper("notification-channel/service/admin-service"),
		repo:     repo,
		notifier: notifier,
	}
}

func (s *NotificationChannelService) ListNotificationChannel(ctx context.Context, req *paginationV1.PagingRequest) (*notificationChannelV1.ListNotificationChannelResponse, error) {
	if err := requirePlatformAdmin(ctx, s.log, "notification channels"); err != nil {
		return nil, err
	}

	return s.repo.List(ctx, req)
}

func (s *NotificationChannelService) GetNotificationChannel(ctx context.Context, req *notificationChannelV1.GetNotificationChannelRequest) (*notificationChannelV1.NotificationChannel, error) {
	if req == nil || req.GetId() == 0 {
		return nil, adminV1.ErrorBadRequest("id is required")
	}

	if err := requirePlatformAdmin(ctx, s.log, "notification channels"); err != nil {
		return nil, err
	}

	return s.repo.Get(ctx, req.GetId())
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

	id, err := s.repo.Create(ctx, req, operator.UserId)
	if err != nil {
		return nil, err
	}

	return s.repo.Get(ctx, id)
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
	if req == nil || req.GetId() == 0 {
		return nil, adminV1.ErrorBadRequest("id is required")
	}
	if req.GetRecipient() == "" {
		return nil, adminV1.ErrorBadRequest("recipient is required")
	}

	if err := requirePlatformAdmin(ctx, s.log, "notification channels"); err != nil {
		return nil, err
	}

	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	title, body := mailtext.ChannelTestEmail(ctx, req.GetId(), operator.UserId)

	if _, err = s.notifier.SendDirect(ctx, &notificationV1.SendDirectNotificationRequest{
		EventType:      notificationV1.EventType_CHANNEL_TEST_EMAIL,
		ChannelId:      trans.Ptr(req.GetId()),
		Target:         req.GetRecipient(),
		Title:          title,
		Content:        body,
		OperatorUserId: trans.Ptr(operator.UserId),
	}); err != nil {
		s.log.Errorf(ctx, "send test email via channel [%d] to [%s] failed: %v", req.GetId(), req.GetRecipient(), err)
		return nil, adminV1.ErrorBadRequest("%s", "send test email failed: "+err.Error())
	}

	s.log.Infof(ctx, "test email sent via channel [%d] to [%s] by operator [%d]", req.GetId(), req.GetRecipient(), operator.UserId)
	return &emptypb.Empty{}, nil
}

func itoa(v uint32) string {
	return strconv.FormatUint(uint64(v), 10)
}
