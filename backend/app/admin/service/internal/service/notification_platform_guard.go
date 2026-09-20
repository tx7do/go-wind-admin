package service

import (
	"context"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"

	"go-wind-admin/pkg/middleware/auth"
)

// requirePlatformAdmin 把通知域的配置与台账挡在平台管理员之内。
//
// 渠道、路由规则、投递台账三张表都没有 tenant_id 列（形状与理由见
// docs/notification_domain_design.md §6 决策点 1）：租户侧能读就读到全平台的 SMTP 主机名与
// 投递记录，能写就改得动全站的事件路由。实测（2026-09-20，企业版租户管理员 token）
// GET /admin/v1/notification-channels 回 200 并列出三条平台通道，POST 真实落了一行
// （created_by 是那个租户用户）。而规则/台账那两个口之所以拒绝，只是因为
// ServiceTagToBusinessModule 少登记了这两个服务 ⇒ business_module 为空 ⇒ 租户闸门把
// "未归类"按不在任何白名单内拒绝。一道护栏不该靠漏配存在，所以判定显式放这里。
func requirePlatformAdmin(ctx context.Context, log *bLogger.Helper, resource string) error {
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return err
	}

	if operator.GetIsPlatformAdmin() {
		return nil
	}

	log.Errorf(ctx, "operator [%d] (tenant %d) is not a platform administrator, %s access denied",
		operator.GetUserId(), operator.GetTenantId(), resource)

	return adminV1.ErrorForbidden("%s is platform-level configuration, only platform administrators may access it", resource)
}
