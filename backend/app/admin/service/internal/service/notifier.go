package service

import (
	"context"

	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"
)

// Notifier 是业务 service 唯一依赖的通知出口。
//
// 声明在使用方所在的包、由 NotificationService 实现（Go 惯例：接口属于消费者）。
// 业务代码只关心"这是哪个事件、发给谁、正文是什么"，不关心渠道选择与台账。
//
// 契约：投递未成功时同时返回非 nil 的 response（含台账 ID 与 SKIPPED/FAILED 状态）
// 和一个 error；调用方按自己的语境把 error 翻成对外错误码（HTTP 层报 4xx/5xx、
// 后台任务只记日志）。response 用于把台账 ID 带到日志里。
type Notifier interface {
	// SendDirect 直发：调用方已握有投递目标（邮箱地址等），不经用户解析。
	SendDirect(ctx context.Context, req *notificationV1.SendDirectNotificationRequest) (*notificationV1.SendNotificationResponse, error)
}
