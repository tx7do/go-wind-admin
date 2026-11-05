package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/tx7do/go-crud/viewer"
	"github.com/tx7do/go-utils/trans"
	"google.golang.org/protobuf/types/known/emptypb"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"

	"go-wind-admin/pkg/mailtext"

	"github.com/tx7do/go-utils/notify/sms"
)

// generateVCode 生成 6 位数字验证码（crypto/rand）。
// 此前用纳秒时间取模——攻击者按发包时刻即可把候选集缩到极小，等于半公开。
// 空间强度由「密码学随机 + 单次有效 + 失败锁定（vcodeMaxVerifyAttempts）+
// 发码频控」共同保证。
func generateVCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// resetIdentityTypeOf 按.identifier 的形态选凭证类型：像手机号（无 @ 且能归一）→ PHONE 走短信，
// 其余（邮箱/账号名）→ EMAIL 维持既有行为。防枚举口径两边一致：查无此人一律静默 200。
func resetIdentityTypeOf(identifier string) authenticationV1.UserCredential_IdentityType {
	if !strings.Contains(identifier, "@") {
		if _, _, err := sms.NormalizePhoneNumber(identifier); err == nil {
			return authenticationV1.UserCredential_PHONE
		}
	}
	return authenticationV1.UserCredential_EMAIL
}

// ForgotPassword 忘记密码：向 identifier（已绑定的邮箱凭证或手机号凭证）发送
// 重置验证码——邮箱走 SMTP，手机号走 SMS（显式点名，不经路由表）。
// 用户不存在时同样返回成功，防止通过接口枚举有效账号。
// 发码频控（同标识 60s 冷却）：超频的请求一律静默 200——既压住把发码端点当
// 短信/邮件轰炸器的滥用，也不给爆破者任何「该标识被限流」的可观测差异。
func (s *AuthenticationService) ForgotPassword(ctx context.Context, req *authenticationV1.ForgotPasswordRequest) (*emptypb.Empty, error) {
	ctx = viewer.WithSystemContext(ctx)
	identifier := strings.TrimSpace(req.GetIdentifier())
	if identifier == "" {
		return nil, authenticationV1.ErrorBadRequest("identifier is required")
	}

	if s.vcodeCache != nil && !s.vcodeCache.Throttle("reset_password", identifier, time.Minute) {
		s.log.Infof(ctx, "forgot-password: send throttled for [%s] (silent ok)", identifier)
		return &emptypb.Empty{}, nil
	}

	identityType := resetIdentityTypeOf(identifier)
	cred, credErr := s.userCredentialRepo.GetByIdentifier(ctx, &authenticationV1.GetUserCredentialByIdentifierRequest{
		IdentityType: identityType,
		Identifier:   identifier,
	})
	userId := cred.GetUserId()
	if credErr != nil || userId == 0 {
		s.log.Infof(ctx, "forgot-password: no %s credential for [%s] (silent ok)", identityType.String(), identifier)
		return &emptypb.Empty{}, nil
	}

	code, gErr := generateVCode()
	if gErr != nil {
		// 熵源故障极罕见：宁拒绝发码也不降级为可预测的时间戳码
		s.log.Errorf(ctx, "forgot-password: generate vcode failed: %s", gErr.Error())
		return nil, authenticationV1.ErrorInternalServerError("generate verification code failed")
	}
	if err := s.vcodeCache.Save("reset_password", identifier, code, 10*time.Minute); err != nil {
		return nil, authenticationV1.ErrorInternalServerError("save verification code failed")
	}

	// 文案语言：优先收件用户的偏好 locale（个人中心设置），未设置回落请求
	// Accept-Language（mailtext.Resolve 的兜底链）。
	if user, uErr := s.userRepo.Get(ctx, &identityV1.GetUserRequest{
		QueryBy: &identityV1.GetUserRequest_Id{Id: userId},
	}); uErr == nil && user.GetLocale() != "" {
		if loc, ok := mailtext.LocaleOfTag(user.GetLocale()); ok {
			ctx = mailtext.WithLocale(ctx, loc)
		}
	}

	sendReq := &notificationV1.SendDirectNotificationRequest{
		EventType:       notificationV1.EventType_PASSWORD_RESET_CODE,
		Target:          identifier,
		RecipientUserId: trans.Ptr(userId),
	}
	if identityType == authenticationV1.UserCredential_PHONE {
		// 短信正文由服务商审核模板渲染，code 经 params 交给渠道；台账文案只做记录。
		sendReq.Channel = trans.Ptr(notificationV1.Channel_SMS)
		sendReq.Params = map[string]string{"code": code}
		sendReq.Title, sendReq.Content = mailtext.SmsPasswordResetCode(ctx, code)
	} else {
		sendReq.Title, sendReq.Content = renderPwdResetCode(ctx, s.mailer, code)
	}

	resp, err := s.notifier.SendDirect(ctx, sendReq)
	if err != nil {
		s.log.Errorf(ctx, "forgot-password: send code to [%s] failed (delivery %d): %s",
			identifier, resp.GetDeliveryId(), err.Error())
		// SKIPPED＝一条都没发出去：没配启用的对应渠道。这与"渠道报错"是两回事，
		// 前者要管理员去渠道页配置，分开报才不用翻日志才知道差哪一步。
		if resp.GetStatus() == notificationV1.DeliveryStatus_SKIPPED {
			if identityType == authenticationV1.UserCredential_PHONE {
				return nil, authenticationV1.ErrorInternalServerError("sms channel is not configured")
			}
			return nil, authenticationV1.ErrorInternalServerError("email channel is not configured")
		}
		return nil, authenticationV1.ErrorInternalServerError("send verification message failed")
	}

	s.log.Infof(ctx, "forgot-password: reset code sent to [%s] for user [%d]", identifier, userId)
	return &emptypb.Empty{}, nil
}

// ResetPasswordByCode 凭邮箱验证码重置密码（免鉴权）。
// 校验通过后重置密码（密码策略/加密在 repo 层处理）并吊销该用户全部会话。
func (s *AuthenticationService) ResetPasswordByCode(ctx context.Context, req *authenticationV1.ResetPasswordByCodeRequest) (*emptypb.Empty, error) {
	ctx = viewer.WithSystemContext(ctx)
	identifier := strings.TrimSpace(req.GetIdentifier())
	code := strings.TrimSpace(req.GetCode())
	newPassword := req.GetNewPassword()
	if identifier == "" || code == "" || newPassword == "" {
		return nil, authenticationV1.ErrorBadRequest("identifier, code and new password are required")
	}

	if !s.vcodeCache.Verify("reset_password", identifier, code) {
		return nil, authenticationV1.ErrorBadRequest("invalid or expired verification code")
	}

	cred, credErr := s.userCredentialRepo.GetByIdentifier(ctx, &authenticationV1.GetUserCredentialByIdentifierRequest{
		IdentityType: resetIdentityTypeOf(identifier),
		Identifier:   identifier,
	})
	if credErr != nil || cred.GetUserId() == 0 {
		return nil, authenticationV1.ErrorBadRequest("invalid or expired verification code")
	}
	userId := cred.GetUserId()

	user, err := s.userRepo.Get(ctx, &identityV1.GetUserRequest{
		QueryBy: &identityV1.GetUserRequest_Id{Id: userId},
	})
	if err != nil {
		return nil, err
	}

	if err = s.userCredentialRepo.ResetCredential(ctx, &authenticationV1.ResetCredentialRequest{
		IdentityType:  authenticationV1.UserCredential_USERNAME,
		Identifier:    user.GetUsername(),
		NewCredential: newPassword,
		NeedDecrypt:   true,
	}); err != nil {
		s.log.Errorf(ctx, "reset password by code failed for user [%d]: %s", userId, err.Error())
		return nil, err
	}

	if err = s.authenticator.RevokeUserTokenAllClientTypes(ctx, userId); err != nil {
		s.log.Errorf(ctx, "revoke tokens after reset-by-code failed for user [%d]: %s", userId, err.Error())
	}

	s.log.Infof(ctx, "password reset by code done for user [%d]", userId)
	return &emptypb.Empty{}, nil
}
