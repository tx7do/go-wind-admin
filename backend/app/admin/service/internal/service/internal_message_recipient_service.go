package service

import (
	"context"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	internalMessageV1 "go-wind-admin/api/gen/go/internal_message/service/v1"

	appViewer "go-wind-admin/pkg/entgo/viewer"
)

type InternalMessageRecipientService struct {
	adminV1.InternalMessageRecipientServiceHTTPServer

	log *bLogger.Helper

	internalMessageRepo          *data.InternalMessageRepo
	internalMessageRecipientRepo *data.InternalMessageRecipientRepo
}

func NewInternalMessageRecipientService(
	ctx *bootstrap.Context,
	internalMessageRepo *data.InternalMessageRepo,
	internalMessageRecipientRepo *data.InternalMessageRecipientRepo,
) *InternalMessageRecipientService {
	return &InternalMessageRecipientService{
		log:                          ctx.NewLoggerHelper("internal-message-recipient/service/admin-service"),
		internalMessageRepo:          internalMessageRepo,
		internalMessageRecipientRepo: internalMessageRecipientRepo,
	}
}

// ListUserInbox 获取用户的收件箱列表 (通知类)
func (s *InternalMessageRecipientService) ListUserInbox(ctx context.Context, req *paginationV1.PagingRequest) (*internalMessageV1.ListUserInboxResponse, error) {
	resp, err := s.internalMessageRecipientRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	// 一次 IN 查询批量回填消息本体：逐条 Get 是 N+1（每页 10 条 = 11 次查询），
	// 且收件箱是头部徽标的轮询高频路径。
	//
	// 这一步必须以 SystemViewer 读，不能用读者的 viewer：平台公告的父消息行落在租户 0，
	// 而收件行的读取按读者租户过滤—— TenantPrivacy 会给本查询再 AND 上 tenant_id=读者租户，
	// 于是公告进了收件箱却没有标题正文（推送侧从内存 DTO 取正文，反而是全的，两路径就此分叉）。
	// messageIds 不是外部输入：它们全部来自上面已按读者租户过滤过的收件行，
	// "能读到这条收件行"就是"能读到它的父消息"的授权凭据。
	messageIds := make([]uint32, 0, len(resp.Items))
	for _, d := range resp.Items {
		if d.MessageId != nil {
			messageIds = append(messageIds, d.GetMessageId())
		}
	}

	messages, err := s.internalMessageRepo.ListByIds(appViewer.NewSystemViewerContext(ctx), messageIds)
	if err != nil {
		// 回填失败只影响 title/content 展示，不阻断整页返回，与此前逐条 Get 失败仅跳过保持一致。
		s.log.Errorf(ctx, "list user inbox failed, batch get messages failed: %s", err)
		return resp, nil
	}

	for _, d := range resp.Items {
		if msg, ok := messages[d.GetMessageId()]; ok {
			d.Title = msg.Title
			d.Content = msg.Content
		}
	}

	return resp, nil
}

func (s *InternalMessageRecipientService) DeleteNotificationFromInbox(ctx context.Context, req *internalMessageV1.DeleteNotificationFromInboxRequest) (*emptypb.Empty, error) {
	err := s.internalMessageRecipientRepo.DeleteNotificationFromInbox(ctx, req)
	return &emptypb.Empty{}, err
}

// MarkNotificationAsRead 将通知标记为已读
func (s *InternalMessageRecipientService) MarkNotificationAsRead(ctx context.Context, req *internalMessageV1.MarkNotificationAsReadRequest) (*emptypb.Empty, error) {
	err := s.internalMessageRecipientRepo.MarkNotificationAsRead(ctx, req)
	return &emptypb.Empty{}, err
}

// MarkNotificationsStatus 标记特定用户的某些或所有通知的状态
func (s *InternalMessageRecipientService) MarkNotificationsStatus(ctx context.Context, req *internalMessageV1.MarkNotificationsStatusRequest) (*emptypb.Empty, error) {
	err := s.internalMessageRecipientRepo.MarkNotificationsStatus(ctx, req)
	return &emptypb.Empty{}, err
}
