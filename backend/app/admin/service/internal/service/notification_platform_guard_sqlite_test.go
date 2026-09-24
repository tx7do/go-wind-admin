// 通知域平台侧判定（requirePlatformAdmin）的 SQLite 集成测试。
//
// 为什么单独一个文件、为什么值得测：渠道 / 路由规则 / 投递台账三张表都没有 tenant_id 列，
// 租户侧一旦能写就改得动全站的事件路由，一旦能读就读到全平台的 SMTP 主机与投递记录。
// 2026-09-20 之前挡着租户的是"这两个服务没登记进 ServiceTagToBusinessModule ⇒ 闸门按未归类拒绝"
// 这一层偶然（渠道服务登记了 SYSTEM，所以那时 GET/POST /admin/v1/notification-channels
// 用企业版租户令牌实测是 **200**，并且真落了一行）。判定挪到显式层之后，这条测试锁的就是
// "偶然"变成"有意"这件事本身 —— 所以它要覆盖全部 14 个方法，漏一个就等于没挡。
package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"

	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"
	notificationChannelV1 "go-wind-admin/api/gen/go/notification_channel/service/v1"

	"go-wind-admin/app/admin/service/internal/data/enttest"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"

	"go-wind-admin/pkg/middleware/auth"
)

// tenantAdminCtx 一个企业版租户管理员的令牌上下文：ita=true、ipa 缺省为假。
func tenantAdminCtx() context.Context {
	return auth.NewContext(enttest.NewSystemViewerCtx(context.Background()),
		&authenticationV1.UserTokenPayload{UserId: 900, TenantId: trans.Ptr(uint32(7)), IsTenantAdmin: trans.Ptr(true)})
}

func platformAdminCtx(userId uint32) context.Context {
	return auth.NewContext(enttest.NewSystemViewerCtx(context.Background()),
		&authenticationV1.UserTokenPayload{UserId: userId, IsPlatformAdmin: trans.Ptr(true)})
}

func TestNotificationPlatformGuardSqlite_ChannelRoutesDenyTenantAdmin(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newNotificationChannelServiceForTest(t, entClient)
	tenantCtx := tenantAdminCtx()

	_, err := svc.ListNotificationChannel(tenantCtx, &paginationV1.PagingRequest{})
	require.True(t, adminV1.IsForbidden(err), "租户管理员不得列出平台渠道: %v", err)

	_, err = svc.GetNotificationChannel(tenantCtx, &notificationChannelV1.GetNotificationChannelRequest{Id: 1})
	require.True(t, adminV1.IsForbidden(err), "租户管理员不得读取平台渠道: %v", err)

	_, err = svc.CreateNotificationChannel(tenantCtx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name: trans.Ptr("tenant-write"),
			Type: notificationChannelV1.NotificationChannel_EMAIL.Enum(),
		},
	})
	require.True(t, adminV1.IsForbidden(err), "租户管理员不得新增平台渠道: %v", err)

	_, err = svc.UpdateNotificationChannel(tenantCtx, &notificationChannelV1.UpdateNotificationChannelRequest{
		Id:   1,
		Data: &notificationChannelV1.NotificationChannel{Remark: trans.Ptr("hijacked")},
	})
	require.True(t, adminV1.IsForbidden(err), "租户管理员不得改平台渠道: %v", err)

	_, err = svc.DeleteNotificationChannel(tenantCtx, &notificationChannelV1.DeleteNotificationChannelRequest{Id: 1})
	require.True(t, adminV1.IsForbidden(err), "租户管理员不得删平台渠道: %v", err)

	_, err = svc.SendTestEmail(tenantCtx, &notificationChannelV1.SendTestEmailRequest{Id: 1, Recipient: "x@y.test"})
	require.True(t, adminV1.IsForbidden(err), "租户管理员不得借平台渠道发信: %v", err)

	// 拒绝必须发生在落库之前：被挡的那次创建一行都没留下。
	cnt, cntErr := entClient.Client().NotificationChannel.Query().Count(context.Background())
	require.NoError(t, cntErr)
	require.Zero(t, cnt, "四个被拒的方法都不能写 sys_notification_channels")
}

func TestNotificationPlatformGuardSqlite_RuleRoutesDenyTenantAdmin(t *testing.T) {
	env := newRuleSvcEnv(t)
	tenantCtx := tenantAdminCtx()

	_, err := env.svc.ListNotificationRule(tenantCtx, &paginationV1.PagingRequest{})
	require.True(t, adminV1.IsForbidden(err), "租户管理员不得列出事件路由: %v", err)

	_, err = env.svc.GetNotificationRule(tenantCtx, &notificationV1.GetNotificationRuleRequest{Id: 1})
	require.True(t, adminV1.IsForbidden(err), "租户管理员不得读取事件路由: %v", err)

	_, err = env.svc.CreateNotificationRule(tenantCtx, &notificationV1.CreateNotificationRuleRequest{
		Data: &notificationV1.NotificationRule{
			EventType: notificationV1.EventType_PASSWORD_RESET_CODE.Enum(),
			Channel:   notificationV1.Channel_WEBHOOK.Enum(),
		},
	})
	require.True(t, adminV1.IsForbidden(err), "租户管理员不得新增事件路由: %v", err)

	_, err = env.svc.UpdateNotificationRule(tenantCtx, &notificationV1.UpdateNotificationRuleRequest{
		Id:   1,
		Data: &notificationV1.NotificationRule{Remark: trans.Ptr("hijacked")},
	})
	require.True(t, adminV1.IsForbidden(err), "租户管理员不得改事件路由: %v", err)

	_, err = env.svc.DeleteNotificationRule(tenantCtx, &notificationV1.DeleteNotificationRuleRequest{Id: 1})
	require.True(t, adminV1.IsForbidden(err), "租户管理员不得删事件路由: %v", err)

	_, err = env.svc.TestDispatchNotification(tenantCtx, &notificationV1.TestDispatchNotificationRequest{Id: 1})
	require.True(t, adminV1.IsForbidden(err), "租户管理员不得触发测试投递: %v", err)

	require.Empty(t, env.notifier.calls, "被拒的测试投递不得走到通知缝")
}

func TestNotificationPlatformGuardSqlite_LedgerRoutesDenyTenantAdmin(t *testing.T) {
	env := newNotificationServiceForTest(t)
	tenantCtx := tenantAdminCtx()

	_, err := env.svc.ListNotificationDelivery(tenantCtx, &paginationV1.PagingRequest{})
	require.True(t, adminV1.IsForbidden(err), "租户管理员不得列出全平台台账: %v", err)

	_, err = env.svc.GetNotificationDelivery(tenantCtx, &notificationV1.GetNotificationDeliveryRequest{Id: 1})
	require.True(t, adminV1.IsForbidden(err), "租户管理员不得读取他租户的投递记录: %v", err)
}

// 正向对照：同一个方法，平台管理员令牌照常放行 —— 挡住租户不等于挡住平台。
func TestNotificationPlatformGuardSqlite_PlatformAdminStillAllowed(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	chans := newNotificationChannelServiceForTest(t, entClient)
	rules := newRuleSvcEnv(t)
	ledger := newNotificationServiceForTest(t)
	ctx := platformAdminCtx(700)

	_, err := chans.ListNotificationChannel(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	_, err = rules.svc.ListNotificationRule(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	_, err = ledger.svc.ListNotificationDelivery(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
}

// 没有令牌时不该由这条判定编出一句 403：原始认证错误原样上抛（不吞错、不改写语义）。
func TestNotificationPlatformGuardSqlite_MissingTokenPropagatesAuthError(t *testing.T) {
	chans := newNotificationChannelServiceForTest(t, enttest.NewEntClientForTest(t))

	_, err := chans.ListNotificationChannel(context.Background(), &paginationV1.PagingRequest{})
	require.Error(t, err)
	require.False(t, adminV1.IsForbidden(err), "无令牌应是 401 认证错误，不是本判定造的 403: %v", err)
}
