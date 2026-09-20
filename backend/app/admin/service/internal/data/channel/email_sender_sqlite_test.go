// EmailSender 的 SQLite 内存库测试：服务层测试用替身跳过的"渠道选择"这一段收在这里
// （见 notification_service_sqlite_test.go 的跳过项）。
//
// 覆盖两条只有接真库才暴露得出的语义：
//   - 自选渠道时错误文本与账号解析都带**真实**渠道 ID——台账的 channel_id 全靠它，
//     早先这里恒回 0，事件路由发出的邮件在账上答不出走的哪个 SMTP 账号；
//   - "没有可用渠道"归 ErrChannelNotConfigured（SKIPPED），拨号前守卫之外的 SMTP 报错
//     保持普通 error（FAILED），两者不能混。
//
// 全程不真发信：候选渠道一律不配 host，mailer.SendMail 在拨号前返回，测试不碰网络。
package channel

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"

	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/enttest"

	notificationChannelV1 "go-wind-admin/api/gen/go/notification_channel/service/v1"
)

type emailSenderEnv struct {
	sender *EmailSender
	repo   *data.NotificationChannelRepo
	ctx    context.Context
}

func newEmailSenderEnv(t *testing.T) *emailSenderEnv {
	t.Helper()

	entClient := enttest.NewEntClientForTest(t)
	repo := data.NewNotificationChannelRepoForTest(entClient)

	return &emailSenderEnv{
		sender: NewEmailSender(repo),
		repo:   repo,
		ctx:    enttest.NewSystemViewerCtx(context.Background()),
	}
}

// create 落一条渠道配置并返回其主键。
func (e *emailSenderEnv) create(t *testing.T, row *notificationChannelV1.NotificationChannel) uint32 {
	t.Helper()

	id, err := e.repo.Create(e.ctx, &notificationChannelV1.CreateNotificationChannelRequest{Data: row}, 1)
	require.NoError(t, err)
	return id
}

func req() *SendRequest {
	return &SendRequest{Target: "bob@example.com", Title: "重置验证码", Content: "您的验证码是 123456"}
}

func itoa(v uint32) string { return strconv.FormatUint(uint64(v), 10) }

// TestEmailSenderSqlite_AutoPickNamesRealChannel 自选渠道：失败文本里必须是实际命中的
// 那条配置 ID；账号拿到了、只是 host 没填 → 普通投递失败（FAILED），不是配置缺失。
func TestEmailSenderSqlite_AutoPickNamesRealChannel(t *testing.T) {
	e := newEmailSenderEnv(t)

	id := e.create(t, &notificationChannelV1.NotificationChannel{
		Name:     trans.Ptr("auto-pick-email"),
		Type:     notificationChannelV1.NotificationChannel_EMAIL.Enum(),
		SmtpPort: trans.Ptr(uint32(587)),
		Enabled:  trans.Ptr(true),
		// 故意不配 SmtpHost：SendMail 在拨号前返回，错误可控且不碰网络。
	})

	receipt, err := e.sender.Send(e.ctx, req())
	require.Error(t, err)
	require.Nil(t, receipt)
	require.Contains(t, err.Error(), "send mail via channel ["+itoa(id)+"] failed",
		"错误文本要指回真正用的那条渠道配置")
	require.Contains(t, err.Error(), "smtp host/port is not configured")
	require.False(t, errors.Is(err, ErrChannelNotConfigured),
		"已拿到账号只是 host 没填：属投递失败，不该被归成 SKIPPED")
}

// TestEmailSenderSqlite_PickAccountCarriesId 两条解析分支（自选 / 显式 ID）都要把渠道
// 主键带回来——台账补 channel_id 与错误文本共用这一个来源。
func TestEmailSenderSqlite_PickAccountCarriesId(t *testing.T) {
	e := newEmailSenderEnv(t)

	// 干扰行：ID 更小的 WEBHOOK 启用配置，不该被当作邮件账号
	idNoise := e.create(t, &notificationChannelV1.NotificationChannel{
		Name:    trans.Ptr("noise-webhook"),
		Type:    notificationChannelV1.NotificationChannel_WEBHOOK.Enum(),
		Enabled: trans.Ptr(true),
	})
	idEmail := e.create(t, &notificationChannelV1.NotificationChannel{
		Name:    trans.Ptr("picked-email"),
		Type:    notificationChannelV1.NotificationChannel_EMAIL.Enum(),
		Enabled: trans.Ptr(true),
	})
	require.Greater(t, idEmail, idNoise, "断言依赖 EMAIL 那条 ID 较大")

	account, err := e.sender.pickAccount(e.ctx, 0)
	require.NoError(t, err)
	require.Equal(t, idEmail, account.ID, "自选分支要带回真实命中的渠道 ID")

	account, err = e.sender.pickAccount(e.ctx, idEmail)
	require.NoError(t, err)
	require.Equal(t, idEmail, account.ID, "显式分支的 ID 与入参一致")
}

// TestEmailSenderSqlite_NoUsableChannelIsSkipped 只有停用渠道：一条都没配 → SKIPPED 语义。
func TestEmailSenderSqlite_NoUsableChannelIsSkipped(t *testing.T) {
	e := newEmailSenderEnv(t)

	e.create(t, &notificationChannelV1.NotificationChannel{
		Name:    trans.Ptr("disabled-email"),
		Type:    notificationChannelV1.NotificationChannel_EMAIL.Enum(),
		Enabled: trans.Ptr(false),
	})

	_, err := e.sender.Send(e.ctx, req())
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrChannelNotConfigured), "没有可用配置：SKIPPED 而非 FAILED")
}

// TestCheckUsableRejectsUnknownTlsMode 枚举外的加密方式判为"配置不可用"。
// SSL_TLS 是演示数据里真实存在的值：放行到拨号会以 SMTP 报错收场，台账把它记成 FAILED，
// 管理员于是去查 SMTP 服务，而问题其实只是那一格写歪了。
func TestCheckUsableRejectsUnknownTlsMode(t *testing.T) {
	cases := []struct {
		mode       string
		wantUsable bool
	}{
		{"SSL_TLS", false},
		{"TLS", false},
		{"", true},
		{"START_TLS", true},
		{"SSL", true},
		{"NONE", true},
	}

	for _, c := range cases {
		err := checkUsable(&data.SmtpAccount{ID: 7, TlsMode: c.mode})
		if !c.wantUsable {
			require.Error(t, err, "%q 不该放行", c.mode)
			require.True(t, errors.Is(err, ErrChannelNotConfigured), "%q 应归为配置不可用", c.mode)
			require.Contains(t, err.Error(), "channel [7] smtp_tls")
			continue
		}
		require.NoError(t, err, "%q 应放行", c.mode)
	}
}
