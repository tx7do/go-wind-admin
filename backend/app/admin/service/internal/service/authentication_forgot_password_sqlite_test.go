// 找回密码链路（ForgotPassword / ResetPasswordByCode）的 SQLite 内存库测试。
//
// 这是免鉴权入口 + 凭证 + 通知三方的交汇处，测试盯四件事：
//   - 防枚举：identifier 查无此人一律静默 200（响应形态与"发了"完全一致），
//     不给攻击者账号存在性探测口子；
//   - 渠道分派：邮箱走 EMAIL 维持既有行为、手机号（能归一成 E.164）点名 SMS
//     并把 code 放进 params（正文由服务商审核模板渲染）；
//   - SKIPPED 语义：一条都没发出去（对应渠道没配）要与"渠道报错"分开报错，
//     管理员不看日志也知道去渠道页配置；
//   - ResetPasswordByCode：码消费型（单次有效）、三要素缺失拒绝、错误码拒绝。
package service

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/crypto"
	"github.com/tx7do/go-utils/trans"
	conf "github.com/tx7do/kratos-bootstrap/api/gen/go/conf/v1"
	bBootstrap "github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	"go-wind-admin/app/admin/service/internal/data"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"
	"go-wind-admin/pkg/constants"

	"github.com/tx7do/go-utils/notify/sms"
)

// fakeNotifier 是 Notifier 接口的测试桩：回放请求供断言，响应/错误可配置。
type fakeNotifier struct {
	reqs []*notificationV1.SendDirectNotificationRequest
	resp *notificationV1.SendNotificationResponse
	err  error
}

func (f *fakeNotifier) SendDirect(_ context.Context, req *notificationV1.SendDirectNotificationRequest) (*notificationV1.SendNotificationResponse, error) {
	f.reqs = append(f.reqs, req)
	return f.resp, f.err
}

// wireVCodeAndNotifier 给 authSvcEnv 补上 vcodeCache 与 notifier（默认 env 留 nil）。
func wireVCodeAndNotifier(t *testing.T, e *authSvcEnv, n Notifier) {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: e.mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	bctx := bBootstrap.NewContextWithParam(
		context.Background(),
		&conf.AppInfo{},
		&conf.Bootstrap{Data: &conf.Data{Redis: &conf.Data_Redis{Addr: e.mr.Addr()}}},
		bLogger.NopLogger(),
	)
	e.svc.vcodeCache = data.NewVCodeCache(bctx, rdb)
	e.svc.notifier = n
}

// savedVCode 直读 miniredis 拿到服务端生成的验证码（绕过"消费即删"的读取问题）。
func savedVCode(t *testing.T, e *authSvcEnv, identifier string) string {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: e.mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	val, err := rdb.Get(context.Background(), "gowind:vcode:reset_password:"+identifier).Result()
	require.NoError(t, err, "验证码应已写入 Redis")
	return val
}

// seedEmailCredential 落一条 EMAIL/PASSWORD_HASH 凭证。
func (e *authSvcEnv) seedEmailCredential(t *testing.T, userID int64, identifier string) {
	t.Helper()
	require.NoError(t, e.svc.userCredentialRepo.Create(e.ctx, &authenticationV1.CreateUserCredentialRequest{
		Data: &authenticationV1.UserCredential{
			UserId:         trans.Ptr(userID),
			IdentityType:   authenticationV1.UserCredential_EMAIL.Enum(),
			Identifier:     trans.Ptr(identifier),
			CredentialType: authenticationV1.UserCredential_PASSWORD_HASH.Enum(),
			Credential:     trans.Ptr(constants.DefaultUserPassword),
			IsPrimary:      trans.Ptr(true),
			Status:         authenticationV1.UserCredential_ENABLED.Enum(),
		},
	}))
}

// TestForgotPassword_SilentForUnknownIdentifier 防枚举核心语义：
// 查无此人的 identifier 也返回 200、不发任何通知——响应与"发了"无法区分。
func TestForgotPassword_SilentForUnknownIdentifier(t *testing.T) {
	e := newAuthenticationServiceForTest(t)
	notifier := &fakeNotifier{}
	wireVCodeAndNotifier(t, e, notifier)

	// 邮箱形态与手机号形态都要静默
	for _, identifier := range []string{"ghost@example.com", "13800138000"} {
		_, err := e.svc.ForgotPassword(e.ctx, &authenticationV1.ForgotPasswordRequest{
			Identifier: trans.Ptr(identifier),
		})
		require.NoError(t, err, "identifier[%s]查无此人应静默成功（防枚举）", identifier)
	}
	require.Empty(t, notifier.reqs, "静默路径不应发出任何通知")

	// 空 identifier：显式 400
	_, err := e.svc.ForgotPassword(e.ctx, &authenticationV1.ForgotPasswordRequest{})
	require.Error(t, err, "空 identifier 应报 BadRequest")
}

// TestForgotPassword_EmailBranch 验证邮箱分支：正常发码、事件/目标/收件人正确、
// 标题正文非空、验证码已按 identifier 入 Redis。
func TestForgotPassword_EmailBranch(t *testing.T) {
	e := newAuthenticationServiceForTest(t)
	notifier := &fakeNotifier{
		resp: &notificationV1.SendNotificationResponse{DeliveryId: 1, Status: notificationV1.DeliveryStatus_SENT},
	}
	wireVCodeAndNotifier(t, e, notifier)

	const identifier = "alice@example.com"
	e.seedEmailCredential(t, 5501, identifier)

	_, err := e.svc.ForgotPassword(e.ctx, &authenticationV1.ForgotPasswordRequest{
		Identifier: trans.Ptr(identifier),
	})
	require.NoError(t, err)

	require.Len(t, notifier.reqs, 1)
	req := notifier.reqs[0]
	require.Equal(t, notificationV1.EventType_PASSWORD_RESET_CODE, req.GetEventType())
	require.Equal(t, identifier, req.GetTarget())
	require.Equal(t, int64(5501), req.GetRecipientUserId())
	require.NotEmpty(t, req.GetTitle(), "邮件标题应有内容")
	require.NotEmpty(t, req.GetContent(), "邮件正文应有内容")
	require.NotContains(t, req.GetContent(), "code", "正文应为渲染后的文案而非占位符")

	// 验证码已保存且为 6 位数字
	code := savedVCode(t, e, identifier)
	require.Len(t, code, 6, "验证码应为 6 位")
	require.Regexp(t, `^\d{6}$`, code)
}

// TestForgotPassword_PhoneBranch 点名 SMS：手机号凭证走短信渠道，code 经 params
// 交给渠道（服务商审核模板渲染），台账文案只做记录。
func TestForgotPassword_PhoneBranch(t *testing.T) {
	e := newAuthenticationServiceForTest(t)
	notifier := &fakeNotifier{
		resp: &notificationV1.SendNotificationResponse{DeliveryId: 2, Status: notificationV1.DeliveryStatus_SENT},
	}
	wireVCodeAndNotifier(t, e, notifier)

	// 用能通过 NormalizePhoneNumber 的国内手机号
	const phone = "13800138000"
	_, _, err := sms.NormalizePhoneNumber(phone)
	require.NoError(t, err, "测试前置：该手机号应能归一")

	require.NoError(t, e.svc.userCredentialRepo.Create(e.ctx, &authenticationV1.CreateUserCredentialRequest{
		Data: &authenticationV1.UserCredential{
			UserId:         trans.Ptr(int64(5502)),
			IdentityType:   authenticationV1.UserCredential_PHONE.Enum(),
			Identifier:     trans.Ptr(phone),
			CredentialType: authenticationV1.UserCredential_PASSWORD_HASH.Enum(),
			Credential:     trans.Ptr(constants.DefaultUserPassword),
			IsPrimary:      trans.Ptr(true),
			Status:         authenticationV1.UserCredential_ENABLED.Enum(),
		},
	}))

	_, err = e.svc.ForgotPassword(e.ctx, &authenticationV1.ForgotPasswordRequest{
		Identifier: trans.Ptr(phone),
	})
	require.NoError(t, err)

	require.Len(t, notifier.reqs, 1)
	req := notifier.reqs[0]
	require.Equal(t, notificationV1.Channel_SMS, req.GetChannel(), "手机号应点名 SMS 渠道")
	require.NotEmpty(t, req.GetParams()["code"], "code 应经 params 交给渠道")
	require.Equal(t, int64(5502), req.GetRecipientUserId())
}

// TestForgotPassword_SkippedChannel SKIPPED＝渠道一条都没发出去（没配启用），
// 与"渠道报错"分开报错：邮箱/SMS 各自给出"去渠道页配置"的指引文案。
func TestForgotPassword_SkippedChannel(t *testing.T) {
	e := newAuthenticationServiceForTest(t)
	notifier := &fakeNotifier{
		resp: &notificationV1.SendNotificationResponse{DeliveryId: 3, Status: notificationV1.DeliveryStatus_SKIPPED},
		err:  context.DeadlineExceeded, // SKIPPED 伴随 error 返回（契约见 notifier.go 注释）
	}
	wireVCodeAndNotifier(t, e, notifier)

	e.seedEmailCredential(t, 5503, "skipped@example.com")
	_, err := e.svc.ForgotPassword(e.ctx, &authenticationV1.ForgotPasswordRequest{
		Identifier: trans.Ptr("skipped@example.com"),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "email channel is not configured",
		"SKIPPED 应报\"渠道未配置\"而非泛化的发送失败")
}

// TestResetPasswordByCode 验证重置链路：三要素缺失/错误码拒绝、正确码走通
// （口令策略/加密在 repo 层）、码消费型（重放同一码第二次失败）。
func TestResetPasswordByCode(t *testing.T) {
	e := newAuthenticationServiceForTest(t)
	notifier := &fakeNotifier{
		resp: &notificationV1.SendNotificationResponse{DeliveryId: 4, Status: notificationV1.DeliveryStatus_SENT},
	}
	wireVCodeAndNotifier(t, e, notifier)

	const identifier = "reset@example.com"
	const userID = int64(5504)
	e.seedEmailCredential(t, userID, identifier)
	// ResetCredential 按 USERNAME 凭证重置口令：需先落 USERNAME 行
	e.seedCredential(t, nil, userID, "reset_user", authenticationV1.UserCredential_ENABLED)
	// userRepo.Get 走 stub：回放占位用户供按 ID 查 locale/username
	e.seedStubUser(userID, nil, "reset_user", identityV1.User_NORMAL, nil)

	const newPassword = "N3wPass!2026"
	// NeedDecrypt 链路的载荷形态：AES(默认密钥) 加密后 base64（与前端加密链一致）
	encBytes, encErr := crypto.AesEncrypt([]byte(newPassword), crypto.DefaultAESKey, nil)
	require.NoError(t, encErr, "测试前置：AES 加密应成功")
	newPasswordWire := base64.StdEncoding.EncodeToString(encBytes)

	// 三要素缺失各拒一条
	_, err := e.svc.ResetPasswordByCode(e.ctx, &authenticationV1.ResetPasswordByCodeRequest{})
	require.Error(t, err, "三要素缺失应报 BadRequest")

	// 错误码拒绝
	_, err = e.svc.ResetPasswordByCode(e.ctx, &authenticationV1.ResetPasswordByCodeRequest{
		Identifier:  trans.Ptr(identifier),
		Code:        trans.Ptr("000000"),
		NewPassword: trans.Ptr(newPasswordWire),
	})
	require.Error(t, err, "错误码应被拒绝")

	// 正确码：先走一遍 ForgotPassword 生成真码
	_, err = e.svc.ForgotPassword(e.ctx, &authenticationV1.ForgotPasswordRequest{
		Identifier: trans.Ptr(identifier),
	})
	require.NoError(t, err)
	code := savedVCode(t, e, identifier)

	_, err = e.svc.ResetPasswordByCode(e.ctx, &authenticationV1.ResetPasswordByCodeRequest{
		Identifier:  trans.Ptr(identifier),
		Code:        trans.Ptr(code),
		NewPassword: trans.Ptr(newPasswordWire),
	})
	require.NoError(t, err, "正确码应重置成功")

	// 码消费型：重放同一码第二次必须失败
	_, err = e.svc.ResetPasswordByCode(e.ctx, &authenticationV1.ResetPasswordByCodeRequest{
		Identifier:  trans.Ptr(identifier),
		Code:        trans.Ptr(code),
		NewPassword: trans.Ptr(newPasswordWire),
	})
	require.Error(t, err, "同一码第二次应失败（单次有效）")

	// 新口令应能通过密码校验（USERNAME 凭证经 bcrypt 比对）
	resp, err := e.svc.userCredentialRepo.VerifyCredential(e.ctx, &authenticationV1.VerifyCredentialRequest{
		IdentityType: authenticationV1.UserCredential_USERNAME,
		Identifier:   "reset_user",
		Credential:   newPasswordWire,
		NeedDecrypt:  true,
	})
	require.NoError(t, err)
	require.True(t, resp.GetSuccess(), "重置后的新口令应能通过校验")
}
