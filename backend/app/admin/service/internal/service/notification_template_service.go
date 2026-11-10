package service

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"

	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/pkg/middleware/auth"
)

// NotificationTemplateService 通知模板管理（平台管理员）：
// CRUD + 试渲染。渲染内核 renderTemplate 同时被 SendDirect 的 template_code
// 路径复用，两处共享同一套占位符语义。
type NotificationTemplateService struct {
	adminV1.NotificationTemplateServiceHTTPServer

	templateRepo *data.NotificationTemplateRepo
	log          *bLogger.Helper
}

func NewNotificationTemplateService(
	ctx *bootstrap.Context,
	templateRepo *data.NotificationTemplateRepo,
) *NotificationTemplateService {
	return &NotificationTemplateService{
		templateRepo: templateRepo,
		log:          ctx.NewLoggerHelper("notification-template/service/admin-service"),
	}
}

func (s *NotificationTemplateService) ListNotificationTemplate(ctx context.Context, req *paginationV1.PagingRequest) (*notificationV1.ListNotificationTemplateResponse, error) {
	if err := requirePlatformAdmin(ctx, s.log, "notification-template/list"); err != nil {
		return nil, err
	}
	return s.templateRepo.List(ctx, req)
}

func (s *NotificationTemplateService) GetNotificationTemplate(ctx context.Context, req *notificationV1.GetNotificationTemplateRequest) (*notificationV1.NotificationTemplate, error) {
	if err := requirePlatformAdmin(ctx, s.log, "notification-template/get"); err != nil {
		return nil, err
	}
	if req.GetId() != 0 {
		return s.templateRepo.Get(ctx, req.GetId())
	}
	if code := req.GetCode(); code != "" {
		tpl, err := s.templateRepo.GetByCode(ctx, code)
		if err != nil {
			return nil, err
		}
		if tpl == nil {
			return nil, adminV1.ErrorNotFound("notification template not found")
		}
		return tpl, nil
	}
	return nil, adminV1.ErrorBadRequest("id or code is required")
}

func (s *NotificationTemplateService) CreateNotificationTemplate(ctx context.Context, req *notificationV1.CreateNotificationTemplateRequest) (*notificationV1.NotificationTemplate, error) {
	if err := requirePlatformAdmin(ctx, s.log, "notification-template/create"); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 不做"空变量集试渲染"：模板带占位符是它的本义，值在发送时才给，
	// 建立时渲染必然失败。占位符拼错留给渲染时报（带占位符名），预览按钮即可提前发现。
	id, err := s.templateRepo.Create(ctx, req.Data, currentUserId(ctx))
	if err != nil {
		return nil, err
	}

	return s.templateRepo.Get(ctx, id)
}

func (s *NotificationTemplateService) UpdateNotificationTemplate(ctx context.Context, req *notificationV1.UpdateNotificationTemplateRequest) (*emptypb.Empty, error) {
	if err := requirePlatformAdmin(ctx, s.log, "notification-template/update"); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	if err := s.templateRepo.Update(ctx, req, currentUserId(ctx)); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *NotificationTemplateService) DeleteNotificationTemplate(ctx context.Context, req *notificationV1.DeleteNotificationTemplateRequest) (*emptypb.Empty, error) {
	if err := requirePlatformAdmin(ctx, s.log, "notification-template/delete"); err != nil {
		return nil, err
	}
	if err := s.templateRepo.Delete(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// RenderNotificationTemplate 试渲染：变量集给了什么就渲染什么，
// 模板引用了变量集之外的占位符 → 400 带占位符名。
func (s *NotificationTemplateService) RenderNotificationTemplate(ctx context.Context, req *notificationV1.RenderNotificationTemplateRequest) (*notificationV1.RenderNotificationTemplateResponse, error) {
	if err := requirePlatformAdmin(ctx, s.log, "notification-template/render"); err != nil {
		return nil, err
	}

	tpl, err := s.templateRepo.Get(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	title, err := renderTemplate(tpl.GetTitleTemplate(), req.GetVariables())
	if err != nil {
		return nil, adminV1.ErrorBadRequest("render title failed: %s", err.Error())
	}
	content, err := renderTemplate(tpl.GetContentTemplate(), req.GetVariables())
	if err != nil {
		return nil, adminV1.ErrorBadRequest("render content failed: %s", err.Error())
	}

	return &notificationV1.RenderNotificationTemplateResponse{
		Title:   title,
		Content: content,
	}, nil
}

// currentUserId 取操作人 ID；系统上下文（无操作人）返回 0，repo 侧据此留空 created_by。
func currentUserId(ctx context.Context) int64 {
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return 0
	}
	return operator.GetUserId()
}

// renderTemplate 渲染 {{var}} 占位模板：单趟扫描，占位符名去变量集查值。
//
// 语义与 WEBHOOK 载荷模板（data/channel/webhook_style.go）一致但更宽——这里的变量
// 是调用方给的任意集合：
//   - 未识别占位符 → error（带占位符名）：静默渲染空串会让"变量名拼错"漏到对端
//     才现形，排查方向整个是反的；
//   - `{{` 未闭合 → 按正文原样保留（模板正文里出现 JSON 片段是合法需求，不值得为它失败）；
//   - 值不做任何转义：模板产物是纯文本（邮件正文/站内信正文），不是 JSON。
func renderTemplate(template string, vars map[string]string) (string, error) {
	if template == "" {
		return "", nil
	}
	if len(vars) == 0 {
		// 空变量集仍要检查"是否引用了占位符"（建/改模板时的语法试渲染即此路径）
		if start := strings.Index(template, "{{"); start >= 0 {
			end := strings.Index(template[start:], "}}")
			if end >= 0 {
				name := strings.TrimSpace(template[start+2 : start+end])
				if name != "" {
					return "", fmt.Errorf("placeholder {{%s}} has no value", name)
				}
			}
		}
		return template, nil
	}

	var b strings.Builder
	b.Grow(len(template) + 64)
	for {
		start := strings.Index(template, "{{")
		if start < 0 {
			b.WriteString(template)
			break
		}
		end := strings.Index(template[start:], "}}")
		if end < 0 {
			// 未闭合的 {{ 按原文保留
			b.WriteString(template)
			break
		}
		name := strings.TrimSpace(template[start+2 : start+end])
		b.WriteString(template[:start])
		if name == "" {
			// {{ }} 空占位按原文保留
			b.WriteString(template[start : start+end+2])
		} else {
			value, ok := vars[name]
			if !ok {
				return "", fmt.Errorf("placeholder {{%s}} has no value", name)
			}
			b.WriteString(value)
		}
		template = template[start+end+2:]
	}
	return b.String(), nil
}
