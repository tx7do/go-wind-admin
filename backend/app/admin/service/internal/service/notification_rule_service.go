package service

import (
	"context"
	"strings"

	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/emptypb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"

	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/pkg/constants"
	appViewer "go-wind-admin/pkg/entgo/viewer"
	"go-wind-admin/pkg/mailtext"
	"go-wind-admin/pkg/middleware/auth"
)

// NotificationRuleService 通知路由规则管理（平台级配置）。
//
// 这张表是投递时"事件 → 渠道 + 派发方式"的唯一运行时真相（见 NotificationService.resolveRoute），
// 本服务只管它的增删改查与一次测试投递，不参与投递决策：两条路径读同一张表，不各存一份副本。
type NotificationRuleService struct {
	adminV1.NotificationRuleServiceHTTPServer

	log      *bLogger.Helper
	repo     *data.NotificationRuleRepo
	channel  *data.NotificationChannelRepo
	notifier Notifier
}

func NewNotificationRuleService(
	ctx *bootstrap.Context,
	repo *data.NotificationRuleRepo,
	channelRepo *data.NotificationChannelRepo,
	notifier Notifier,
) *NotificationRuleService {
	svc := &NotificationRuleService{
		log:      ctx.NewLoggerHelper("notification-rule/service/admin-service"),
		repo:     repo,
		channel:  channelRepo,
		notifier: notifier,
	}

	svc.init()

	return svc
}

// init 空表播种默认路由（同 DefaultMenus 的 count==0 守卫）。
//
// 为什么不是"按键缺一补一"（config_service 的 SeedDefaults 形态）：规则行是可删的实体，
// 管理员删掉 INTERNAL_MESSAGE 一行是"这个事件不再通知"的决定，每次启动补回来就等于禁用无效。
// 代价必须写清楚：**新增一个事件类型时，已部署实例不会自动多出这一行**，
// 该事件的投递会立刻拿到"no enabled channel routing rule for event type X"这句报错
// （缝在这里不静默丢弃），修法是在规则页手工新增一行 —— 这与菜单新增要靠「菜单同步」同理。
func (s *NotificationRuleService) init() {
	ctx := appViewer.NewSystemViewerContext(context.Background())

	count, err := s.repo.Count(ctx)
	if err != nil {
		s.log.Errorf(ctx, "count notification rules failed, skip seeding: %s", err.Error())
		return
	}
	if count > 0 {
		return
	}

	for _, rule := range constants.DefaultNotificationRules {
		if _, err = s.repo.Create(ctx, rule, 0); err != nil {
			s.log.Errorf(ctx, "seed notification rule [%s] failed: %s", rule.GetEventType().String(), err.Error())
		}
	}
}

func (s *NotificationRuleService) ListNotificationRule(ctx context.Context, req *paginationV1.PagingRequest) (*notificationV1.ListNotificationRuleResponse, error) {
	return s.repo.List(ctx, req)
}

func (s *NotificationRuleService) GetNotificationRule(ctx context.Context, req *notificationV1.GetNotificationRuleRequest) (*notificationV1.NotificationRule, error) {
	if req == nil || req.GetId() == 0 {
		return nil, adminV1.ErrorBadRequest("id is required")
	}

	return s.repo.Get(ctx, req.GetId())
}

func (s *NotificationRuleService) CreateNotificationRule(ctx context.Context, req *notificationV1.CreateNotificationRuleRequest) (*notificationV1.NotificationRule, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	id, err := s.repo.Create(ctx, req.Data, operator.UserId)
	if err != nil {
		return nil, err
	}

	return s.repo.Get(ctx, id)
}

func (s *NotificationRuleService) UpdateNotificationRule(ctx context.Context, req *notificationV1.UpdateNotificationRuleRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err = s.repo.Update(ctx, req, operator.UserId); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *NotificationRuleService) DeleteNotificationRule(ctx context.Context, req *notificationV1.DeleteNotificationRuleRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, adminV1.ErrorBadRequest("id is required")
	}

	if err := s.repo.Delete(ctx, req.GetId()); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// TestDispatchNotification 按这条规则当场走一遍完整投递链：路由、渠道选择、脱敏、台账全部照旧。
//
// 刻意不"顺手把异步改成同步来测"：管理员要知道的是真投递会发生什么，改掉派发方式测的就不是
// 这条规则了。异步规则因此回 SENDING + 台账 ID，结论去台账页按那个 ID 看；
// 想当场拿到 SMTP 报错原文，用的是渠道配置页的「测试邮件」（CHANNEL_TEST_EMAIL 一条同步规则）。
//
// 失败时以 4xx 把原始错误回出去、不返回响应体（与 SendTestEmail 一致）：这个按钮唯一有用的
// 输出就是"为什么没发出去"，把它塞进日志换一句"操作失败"等于没做这个功能。
func (s *NotificationRuleService) TestDispatchNotification(ctx context.Context, req *notificationV1.TestDispatchNotificationRequest) (*notificationV1.TestDispatchNotificationResponse, error) {
	if req == nil || req.GetId() == 0 {
		return nil, adminV1.ErrorBadRequest("id is required")
	}

	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	rule, err := s.repo.Get(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	eventType := rule.GetEventType()
	if ch := rule.GetChannel(); ch == notificationV1.Channel_INTERNAL {
		return nil, adminV1.ErrorBadRequest(
			"rule [%d] routes to %s: internal messages need a real message body, send one from the internal-message page instead",
			rule.GetId(), ch.String())
	}

	target, channelID, err := s.testDispatchTarget(ctx, rule, req.GetTarget())
	if err != nil {
		return nil, err
	}

	title, content := mailtext.RuleTestNotification(ctx, rule.GetId(), eventType.String())
	if req.GetTitle() != "" {
		title = req.GetTitle()
	}
	if req.GetContent() != "" {
		content = req.GetContent()
	}

	resp, err := s.notifier.SendDirect(ctx, &notificationV1.SendDirectNotificationRequest{
		EventType:      eventType,
		ChannelId:      channelID,
		Target:         target,
		Title:          title,
		Content:        content,
		OperatorUserId: trans.Ptr(operator.UserId),
	})
	if err != nil {
		s.log.Errorf(ctx, "test dispatch of rule [%d] (%s → %s) to [%s] failed: %v",
			rule.GetId(), eventType.String(), rule.GetChannel().String(), target, err)

		return nil, adminV1.ErrorBadRequest("%s", err.Error())
	}

	s.log.Infof(ctx, "test dispatch of rule [%d] by operator [%d]: delivery [%d] %s",
		rule.GetId(), operator.UserId, resp.GetDeliveryId(), resp.GetStatus().String())

	return &notificationV1.TestDispatchNotificationResponse{
		DeliveryId: resp.GetDeliveryId(),
		Status:     resp.GetStatus(),
	}, nil
}

// testDispatchTarget 定出这次测试投递的目标地址，以及该钉住哪一条渠道配置。
//
// 管理员填了就用它（这是"我要发到这个地址"的显式决定，与渠道登记值无关）。
// 留空时只有 WEBHOOK 兜得出默认值：那条渠道登记的 webhook_url 就是"这个事件该发去哪"，
// 于是顺手把 channel_id 也钉成同一行 —— 否则 sender 会另选一条启用的 WEBHOOK 配置，
// 管理员在 A 行点测试、实际打到 B 行。EMAIL 没有可兜的收件地址，留空即报错。
func (s *NotificationRuleService) testDispatchTarget(ctx context.Context, rule *notificationV1.NotificationRule, requested string) (string, *uint32, error) {
	if target := strings.TrimSpace(requested); target != "" {
		return target, nil, nil
	}

	if rule.GetChannel() != notificationV1.Channel_WEBHOOK {
		return "", nil, adminV1.ErrorBadRequest("target is required for a %s channel", rule.GetChannel().String())
	}

	account, err := s.channel.GetFirstEnabledWebhookChannel(ctx)
	if err != nil {
		return "", nil, err
	}
	if account.URL == "" {
		return "", nil, adminV1.ErrorBadRequest("channel [%d] has no webhook_url configured", account.ID)
	}

	return account.URL, trans.Ptr(account.ID), nil
}
