// NotificationChannelService 的 SQLite 内存库集成测试（白盒，包内测试）。
//
// 覆盖目标：
//   - CreateNotificationChannel / GetNotificationChannel / ListNotificationChannel /
//     UpdateNotificationChannel / DeleteNotificationChannel 的落库与字段映射：
//     枚举经转换器落库、操作人盖入 created_by、HasPassword 标识按落库密码有无回填。
//   - SendTestEmail 的全部前置校验分支（id/recipient 缺失、非 EMAIL 渠道、
//     渠道停用、SMTP 主机未配置导致的发送失败），以及这些分支落到投递台账的
//     状态归类（SKIPPED / FAILED）。
//
// 跳过项：真实 SMTP 投递（mailer.SendMail 需要外部 SMTP 服务；本测试用
// 未配置主机/端口的渠道覆盖到 SendMail 的快速失败分支为止）。
package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	notificationChannelV1 "go-wind-admin/api/gen/go/notification_channel/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/channel"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/notificationchannel"
	"go-wind-admin/app/admin/service/internal/data/ent/notificationdelivery"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/middleware/auth"
)

// newNotificationChannelServiceForTest 白盒复刻 NewNotificationChannelService 的
// 字段初始化：log 换 NopLogger，repo 走 data.NewNotificationChannelRepoForTest。
//
// notifier 给的是真货（NotificationService + 真 EmailSender，同库共台账 repo）：
// SendTestEmail 已改走 Notifier，渠道类型/启用状态的校验也随之内移到 EmailSender，
// 用 nil 或替身等于把这条路径唯一有意义的断言——SMTP/配置报错原文能否传到配置页——绕开测。
func newNotificationChannelServiceForTest(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) *NotificationChannelService {
	t.Helper()

	registry := channel.NewRegistry()
	channelRepo := data.NewNotificationChannelRepoForTest(entClient)
	registry.Register(channel.NewEmailSender(channelRepo))

	return &NotificationChannelService{
		log:  bLogger.NewHelper(bLogger.NopLogger()),
		repo: channelRepo,
		notifier: &NotificationService{
			log:          bLogger.NewHelper(bLogger.NopLogger()),
			deliveryRepo: data.NewNotificationDeliveryRepoForTest(entClient),
			channels:     registry,
		},
	}
}

// TestNotificationChannelServiceSqlite_CreateAndGet 验证创建渠道的落库与字段映射：
// 枚举（类型/TLS）经转换器落库、操作人 ID 盖入 created_by、密码加密落库；
// Get 回读的 DTO 字段与创建一致且带 HasPassword=true 标识。
func TestNotificationChannelServiceSqlite_CreateAndGet(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newNotificationChannelServiceForTest(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 71})

	created, err := svc.CreateNotificationChannel(opCtx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name:         trans.Ptr("smtp 渠道甲"),
			Type:         notificationChannelV1.NotificationChannel_EMAIL.Enum(),
			SmtpHost:     trans.Ptr("smtp.example.test"),
			SmtpPort:     trans.Ptr(uint32(465)),
			SmtpUsername: trans.Ptr("postman"),
			SmtpFrom:     trans.Ptr("no-reply@example.test"),
			SmtpTls:      notificationChannelV1.NotificationChannel_SSL.Enum(),
			Enabled:      trans.Ptr(true),
			Remark:       trans.Ptr("服务层创建的渠道"),
		},
		Password: trans.Ptr("Sup3rSecret!"),
	})
	require.NoError(t, err, "创建 EMAIL 渠道应成功")
	require.NotNil(t, created.GetId(), "创建响应应回带渠道 ID")
	require.Equal(t, "smtp 渠道甲", created.GetName())
	require.Equal(t, "smtp.example.test", created.GetSmtpHost())
	require.EqualValues(t, 465, created.GetSmtpPort())
	require.Equal(t, "postman", created.GetSmtpUsername())
	require.Equal(t, "no-reply@example.test", created.GetSmtpFrom())
	require.NotNil(t, created.GetHasPassword(), "落库了密码的渠道应带 HasPassword 标识")
	require.True(t, created.GetHasPassword())

	// 直查 ent：密码列已写入（明文或密文取决于全局加密器是否初始化，二者均合法）、
	// 操作人盖入 created_by。
	row, err := entClient.Client().NotificationChannel.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "smtp 渠道甲", row.Name)
	require.Equal(t, "smtp.example.test", *row.SMTPHost)
	require.EqualValues(t, 465, *row.SMTPPort)
	require.NotNil(t, row.SMTPPassword, "密码应经 EncryptIfNeeded 后落库")
	require.NotNil(t, row.CreatedBy, "created_by 应被盖入操作人 ID")
	require.EqualValues(t, 71, *row.CreatedBy)

	got, err := svc.GetNotificationChannel(ctx, &notificationChannelV1.GetNotificationChannelRequest{
		Id: created.GetId(),
	})
	require.NoError(t, err)
	require.Equal(t, "smtp 渠道甲", got.GetName())
	require.Equal(t, "smtp.example.test", got.GetSmtpHost())
	require.True(t, got.GetHasPassword(), "带密码渠道的 Get 视图应回 HasPassword=true")

	// 加密往返：GetDecryptedSmtpAccount 取回的密码应与写入明文一致
	//（全局加密器未初始化时为直通，已初始化时为对称加解密，两者往返等值）。
	account, err := svc.repo.GetDecryptedSmtpAccount(ctx, created.GetId())
	require.NoError(t, err)
	require.Equal(t, "Sup3rSecret!", account.Password, "密码应经 EncryptIfNeeded/DecryptIfNeeded 对称往返")
	require.True(t, account.Enabled, "启用中的渠道取回的账号应为启用态")
	require.Equal(t, "smtp.example.test", account.Host)
	require.EqualValues(t, 465, account.Port)
}

// TestNotificationChannelServiceSqlite_CreateAndGetValidation 覆盖创建与查询的
// 入参守卫分支：空名/空请求体拒绝；id 缺失、按不存在 id 查询报错。
func TestNotificationChannelServiceSqlite_CreateAndGetValidation(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newNotificationChannelServiceForTest(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 71})

	_, err := svc.CreateNotificationChannel(opCtx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name: trans.Ptr(""),
			Type: notificationChannelV1.NotificationChannel_EMAIL.Enum(),
		},
	})
	require.Error(t, err, "空渠道名应被拒绝")
	require.Contains(t, err.Error(), "channel name is required")

	_, err = svc.CreateNotificationChannel(opCtx, nil)
	require.Error(t, err, "nil 请求体应被拒绝")

	_, err = svc.GetNotificationChannel(ctx, &notificationChannelV1.GetNotificationChannelRequest{Id: 0})
	require.Error(t, err, "id=0 应被拒绝")
	require.Contains(t, err.Error(), "id is required")

	_, err = svc.GetNotificationChannel(ctx, &notificationChannelV1.GetNotificationChannelRequest{Id: 424242})
	require.Error(t, err, "按不存在的 id 查询应报错")

	_, err = svc.GetNotificationChannel(ctx, nil)
	require.Error(t, err, "nil 请求体应被拒绝")
}

// TestNotificationChannelServiceSqlite_ListHasPasswordFlag 验证 List 的
// HasPassword 标识回填：带密码与不带密码的渠道分别回填 true/false。
func TestNotificationChannelServiceSqlite_ListHasPasswordFlag(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newNotificationChannelServiceForTest(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 71})

	_, err := svc.CreateNotificationChannel(opCtx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name:    trans.Ptr("带密码渠道"),
			Type:    notificationChannelV1.NotificationChannel_EMAIL.Enum(),
			Enabled: trans.Ptr(true),
		},
		Password: trans.Ptr("SomePassword1!"),
	})
	require.NoError(t, err)

	_, err = svc.CreateNotificationChannel(opCtx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name:    trans.Ptr("无密码渠道"),
			Type:    notificationChannelV1.NotificationChannel_EMAIL.Enum(),
			Enabled: trans.Ptr(true),
		},
	})
	require.NoError(t, err)

	listResp, err := svc.ListNotificationChannel(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), listResp.GetTotal())
	require.Len(t, listResp.GetItems(), 2)
	for _, item := range listResp.GetItems() {
		switch item.GetName() {
		case "带密码渠道":
			require.True(t, item.GetHasPassword(), "带密码渠道应回填 HasPassword=true")
		case "无密码渠道":
			require.False(t, item.GetHasPassword(), "无密码渠道应回填 HasPassword=false")
		default:
			t.Fatalf("列表中出现未创建的渠道 %q", item.GetName())
		}
	}
}

// TestNotificationChannelServiceSqlite_UpdateRename 验证 Update 的改名落库；
// 密码留空表示不修改已存密码。
func TestNotificationChannelServiceSqlite_UpdateRename(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newNotificationChannelServiceForTest(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 72})

	created, err := svc.CreateNotificationChannel(opCtx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name:    trans.Ptr("改名前渠道"),
			Type:    notificationChannelV1.NotificationChannel_EMAIL.Enum(),
			Enabled: trans.Ptr(true),
		},
		Password: trans.Ptr("FirstPass1!"),
	})
	require.NoError(t, err)

	_, err = svc.UpdateNotificationChannel(opCtx, &notificationChannelV1.UpdateNotificationChannelRequest{
		Id:         created.GetId(),
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
		Data:       &notificationChannelV1.NotificationChannel{Name: trans.Ptr("改名后渠道")},
	})
	require.NoError(t, err, "改名应成功")

	row, err := entClient.Client().NotificationChannel.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "改名后渠道", row.Name, "掩码内字段 name 应被更新")
	require.NotNil(t, row.SMTPPassword, "密码留空的更新不应清除已存密码")

	got, err := svc.GetNotificationChannel(ctx, &notificationChannelV1.GetNotificationChannelRequest{
		Id: created.GetId(),
	})
	require.NoError(t, err)
	require.Equal(t, "改名后渠道", got.GetName())
	require.True(t, got.GetHasPassword(), "未动密码的渠道仍应回 HasPassword=true")

	_, err = svc.UpdateNotificationChannel(opCtx, &notificationChannelV1.UpdateNotificationChannelRequest{
		Id:   0,
		Data: &notificationChannelV1.NotificationChannel{Name: trans.Ptr("x")},
	})
	require.Error(t, err, "id=0 的更新应被拒绝")
	require.Contains(t, err.Error(), "id is required")

	_, err = svc.UpdateNotificationChannel(opCtx, &notificationChannelV1.UpdateNotificationChannelRequest{
		Id: created.GetId(),
	})
	require.Error(t, err, "nil Data 的更新应被拒绝")
}

// TestNotificationChannelServiceSqlite_Delete 验证删除后行数归零、
// 再查询报不存在；id 缺失与 nil 请求体被拒绝。
func TestNotificationChannelServiceSqlite_Delete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newNotificationChannelServiceForTest(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 72})

	created, err := svc.CreateNotificationChannel(opCtx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name:    trans.Ptr("待删除渠道"),
			Type:    notificationChannelV1.NotificationChannel_EMAIL.Enum(),
			Enabled: trans.Ptr(true),
		},
	})
	require.NoError(t, err)

	_, err = svc.DeleteNotificationChannel(ctx, &notificationChannelV1.DeleteNotificationChannelRequest{
		Id: created.GetId(),
	})
	require.NoError(t, err, "删除已存在渠道应成功")

	cnt, err := entClient.Client().NotificationChannel.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, cnt, "删除后渠道表计数应归零")

	_, err = svc.GetNotificationChannel(ctx, &notificationChannelV1.GetNotificationChannelRequest{
		Id: created.GetId(),
	})
	require.Error(t, err, "删除后的渠道应查询不到")

	_, err = svc.DeleteNotificationChannel(ctx, &notificationChannelV1.DeleteNotificationChannelRequest{Id: 0})
	require.Error(t, err, "id=0 的删除应被拒绝")
	require.Contains(t, err.Error(), "id is required")

	_, err = svc.DeleteNotificationChannel(ctx, nil)
	require.Error(t, err, "nil 请求体的删除应被拒绝")
}

// TestNotificationChannelServiceSqlite_SendTestEmailBranches 覆盖 SendTestEmail
// 的全部前置校验分支：id/recipient 缺失、停用渠道、非 EMAIL 渠道类型守卫、
// SMTP 未配置（SendMail 在拨号前快速失败，不触网）。
func TestNotificationChannelServiceSqlite_SendTestEmailBranches(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newNotificationChannelServiceForTest(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 73})

	// 非 EMAIL 渠道。
	webhook, err := svc.CreateNotificationChannel(opCtx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name:    trans.Ptr("webhook 渠道"),
			Type:    notificationChannelV1.NotificationChannel_WEBHOOK.Enum(),
			Enabled: trans.Ptr(true),
		},
	})
	require.NoError(t, err)

	// 停用的 EMAIL 渠道。
	disabled, err := svc.CreateNotificationChannel(opCtx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name:    trans.Ptr("停用渠道"),
			Type:    notificationChannelV1.NotificationChannel_EMAIL.Enum(),
			Enabled: trans.Ptr(false),
		},
	})
	require.NoError(t, err)

	// 启用但未配置 SMTP 主机/端口的 EMAIL 渠道（SendMail 在拨号前快速失败）。
	noHost, err := svc.CreateNotificationChannel(opCtx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name:    trans.Ptr("未配置主机渠道"),
			Type:    notificationChannelV1.NotificationChannel_EMAIL.Enum(),
			Enabled: trans.Ptr(true),
		},
	})
	require.NoError(t, err)

	// 落库侧：枚举经创建转换器按声明落库（WEBHOOK 渠道实体为 TypeWebhook）。
	rows, err := entClient.Client().NotificationChannel.Query().All(ctx)
	require.NoError(t, err)
	storedTypes := map[string]notificationchannel.Type{}
	for _, row := range rows {
		storedTypes[row.Name] = row.Type
	}
	require.Equal(t, notificationchannel.TypeWebhook, storedTypes["webhook 渠道"], "WEBHOOK 渠道应按声明落库")
	require.Equal(t, notificationchannel.TypeEmail, storedTypes["停用渠道"], "EMAIL 渠道应按声明落库")

	// 读路径：Type 回填修复后 WEBHOOK 渠道的 Get 视图如实呈 WEBHOOK。
	webhookView, err := svc.GetNotificationChannel(ctx, &notificationChannelV1.GetNotificationChannelRequest{
		Id: webhook.GetId(),
	})
	require.NoError(t, err)
	require.Equal(t, notificationChannelV1.NotificationChannel_WEBHOOK, webhookView.GetType(),
		"读路径应回填渠道类型（此前 WEBHOOK 视图呈缺省 EMAIL）")

	_, err = svc.SendTestEmail(ctx, nil)
	require.Error(t, err, "nil 请求体应被拒绝")

	_, err = svc.SendTestEmail(ctx, &notificationChannelV1.SendTestEmailRequest{Id: 0, Recipient: "a@b.test"})
	require.Error(t, err, "id=0 应被拒绝")
	require.Contains(t, err.Error(), "id is required")

	_, err = svc.SendTestEmail(ctx, &notificationChannelV1.SendTestEmailRequest{Id: webhook.GetId(), Recipient: ""})
	require.Error(t, err, "空收件人应被拒绝")
	require.Contains(t, err.Error(), "recipient is required")

	// 三条真正走到发送的分支：每次各留一条台账（入参守卫的三次不会）。
	_, err = svc.SendTestEmail(opCtx, &notificationChannelV1.SendTestEmailRequest{Id: disabled.GetId(), Recipient: "a@b.test"})
	require.Error(t, err, "停用渠道应被拒绝")
	require.Contains(t, err.Error(), "is disabled",
		"渠道校验已内移到 EmailSender，报错原文必须仍然透到配置页——管理员看得懂才知道去启用")

	// 启用但没配 SMTP 主机/端口：SendMail 拨号前快速失败，不触网。
	_, err = svc.SendTestEmail(opCtx, &notificationChannelV1.SendTestEmailRequest{Id: noHost.GetId(), Recipient: "a@b.test"})
	require.Error(t, err, "未配置 SMTP 主机/端口的发送应快速失败（不触网）")
	require.Contains(t, err.Error(), "smtp host/port is not configured")

	_, err = svc.SendTestEmail(opCtx, &notificationChannelV1.SendTestEmailRequest{Id: webhook.GetId(), Recipient: "a@b.test"})
	require.Error(t, err, "WEBHOOK 渠道应被类型守卫拒绝（Type 回填修复后仅 EMAIL 守卫生效）")
	require.Contains(t, err.Error(), "is not an EMAIL channel")

	// 台账归类：渠道本身用不了（停用/类型不对）＝一条都没尝试发 → SKIPPED；
	// 拿得到账号、SMTP 报的错 → FAILED。两者混成一个状态，配置页就再也分不出
	// "该去启用渠道"和"该查 SMTP 服务"。
	deliveries, err := entClient.Client().NotificationDelivery.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, deliveries, 3, "入参守卫的三次调用不该留台账")

	statusByError := map[string]notificationdelivery.Status{}
	for _, row := range deliveries {
		require.NotNil(t, row.Status)
		require.NotNil(t, row.LastError, "失败原因不写进台账，这行就没有存在意义")
		require.NotNil(t, row.ChannelID, "显式指定渠道时台账该记下用的是哪一条")
		require.Equal(t, "a***@b.test", *row.Target)
		require.Nil(t, row.SentAt, "三条都没发出去，不该有完成时间")
		statusByError[*row.LastError] = *row.Status
	}

	require.Equal(t, notificationdelivery.StatusSkipped,
		statusByError["no enabled notification channel configured: channel ["+itoa(disabled.GetId())+"] is disabled"])
	require.Equal(t, notificationdelivery.StatusSkipped,
		statusByError["no enabled notification channel configured: channel ["+itoa(webhook.GetId())+"] is not an EMAIL channel"])
	require.Equal(t, notificationdelivery.StatusFailed,
		statusByError["send mail via channel ["+itoa(noHost.GetId())+"] failed: smtp host/port is not configured"])
}
