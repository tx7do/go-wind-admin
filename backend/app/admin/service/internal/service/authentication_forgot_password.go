package service

import (
	appViewer "go-wind-admin/pkg/entgo/viewer"
	"strings"
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/emptypb"
	"github.com/tx7do/go-utils/trans"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"
	"go-wind-admin/pkg/mailtext"
)

// generateVCode 生成 6 位数字验证码。
// 使用纳秒时间取模：验证码仅用于一次性校验，安全强度由
// 「单次有效 + 10 分钟 TTL + 服务端比对」保证，不依赖密码学随机。
func generateVCode() string {
	return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
}

// ForgotPassword 忘记密码：向 identifier（必须为已绑定的邮箱凭证）发送
// 重置验证码。用户不存在时同样返回成功，防止通过接口枚举有效邮箱。
func (s *AuthenticationService) ForgotPassword(ctx context.Context, req *authenticationV1.ForgotPasswordRequest) (*emptypb.Empty, error) {
	ctx = appViewer.NewSystemViewerContext(ctx)
	identifier := strings.TrimSpace(req.GetIdentifier())
	if identifier == "" {
		return nil, authenticationV1.ErrorBadRequest("identifier is required")
	}

	cred, credErr := s.userCredentialRepo.GetByIdentifier(ctx, &authenticationV1.GetUserCredentialByIdentifierRequest{
		IdentityType: authenticationV1.UserCredential_EMAIL,
		Identifier:   identifier,
	})
	userId := cred.GetUserId()
	if credErr != nil || userId == 0 {
		s.log.Infof(ctx, "forgot-password: no EMAIL credential for [%s] (silent ok)", identifier)
		return &emptypb.Empty{}, nil
	}

	code := generateVCode()
	if err := s.vcodeCache.Save("reset_password", identifier, code, 10*time.Minute); err != nil {
		return nil, authenticationV1.ErrorInternalServerError("save verification code failed")
	}

	// 文案按请求的 Accept-Language 选语言：这一步在免鉴权白名单上，上下文里没有
	// token 级的 locale 可用，请求头是唯一入口（见 pkg/mailtext 包注释）。
	title, content := mailtext.PasswordResetCode(ctx, code)
	resp, err := s.notifier.SendDirect(ctx, &notificationV1.SendDirectNotificationRequest{
		EventType:       notificationV1.EventType_PASSWORD_RESET_CODE,
		Target:          identifier,
		RecipientUserId: trans.Ptr(userId),
		Title:           title,
		Content:         content,
	})
	if err != nil {
		s.log.Errorf(ctx, "forgot-password: send mail to [%s] failed (delivery %d): %s",
			identifier, resp.GetDeliveryId(), err.Error())
		// SKIPPED＝一条都没发出去：没配启用的 EMAIL 渠道。这与"SMTP 报错"是两回事，
		// 前者要管理员去渠道页配置，分开报才不用翻日志才知道差哪一步。
		if resp.GetStatus() == notificationV1.DeliveryStatus_SKIPPED {
			return nil, authenticationV1.ErrorInternalServerError("email channel is not configured")
		}
		return nil, authenticationV1.ErrorInternalServerError("send verification email failed")
	}

	s.log.Infof(ctx, "forgot-password: reset code sent to [%s] for user [%d]", identifier, userId)
	return &emptypb.Empty{}, nil
}

// ResetPasswordByCode 凭邮箱验证码重置密码（免鉴权）。
// 校验通过后重置密码（密码策略/加密在 repo 层处理）并吊销该用户全部会话。
func (s *AuthenticationService) ResetPasswordByCode(ctx context.Context, req *authenticationV1.ResetPasswordByCodeRequest) (*emptypb.Empty, error) {
	ctx = appViewer.NewSystemViewerContext(ctx)
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
		IdentityType: authenticationV1.UserCredential_EMAIL,
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
