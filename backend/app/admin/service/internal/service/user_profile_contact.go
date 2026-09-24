package service

import (
	"context"
	"strings"
	"time"

	"github.com/tx7do/go-utils/trans"
	"google.golang.org/protobuf/types/known/emptypb"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"
	"go-wind-admin/pkg/mailtext"
	"go-wind-admin/pkg/middleware/auth"
)

// sendContactVCode 向指定邮箱发送业务验证码（绑定联系方式场景）。
func (s *UserProfileService) sendContactVCode(ctx context.Context, contact string) error {
	code := generateVCode()
	if err := s.vcodeCache.Save("bind_contact", contact, code, 10*time.Minute); err != nil {
		return authenticationV1.ErrorInternalServerError("save verification code failed")
	}

	title, content := mailtext.ContactBindCode(ctx, code)
	resp, err := s.notifier.SendDirect(ctx, &notificationV1.SendDirectNotificationRequest{
		EventType: notificationV1.EventType_CONTACT_BIND_CODE,
		Target:    contact,
		Title:     title,
		Content:   content,
	})
	if err != nil {
		s.log.Errorf(ctx, "bind-contact: send mail to [%s] failed (delivery %d): %s",
			contact, resp.GetDeliveryId(), err.Error())
		if resp.GetStatus() == notificationV1.DeliveryStatus_SKIPPED {
			return authenticationV1.ErrorInternalServerError("email channel is not configured")
		}
		return authenticationV1.ErrorInternalServerError("send verification email failed")
	}
	return nil
}

// BindContact 绑定手机号码/邮箱（第一步）：向新联系方式发送验证码。
// 二期接入短信渠道后可扩展 MOBILE 类型；当前仅支持 EMAIL。
func (s *UserProfileService) BindContact(ctx context.Context, req *identityV1.BindContactRequest) (*emptypb.Empty, error) {
	bindEmail := req.GetEmail()
	if bindEmail == nil || strings.TrimSpace(bindEmail.GetEmail()) == "" {
		return nil, authenticationV1.ErrorBadRequest("only email binding is supported")
	}
	contact := strings.TrimSpace(bindEmail.GetEmail())

	if err := s.sendContactVCode(ctx, contact); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// VerifyContact 校验验证码并完成联系方式绑定（写入 EMAIL 登录凭证）。
func (s *UserProfileService) VerifyContact(ctx context.Context, req *identityV1.VerifyContactRequest) (*emptypb.Empty, error) {
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	verifyEmail := req.GetEmail()
	if verifyEmail == nil || strings.TrimSpace(verifyEmail.GetEmail()) == "" {
		return nil, authenticationV1.ErrorBadRequest("only email verification is supported")
	}
	contact := strings.TrimSpace(verifyEmail.GetEmail())
	code := strings.TrimSpace(verifyEmail.GetCode())
	if contact == "" || code == "" {
		return nil, authenticationV1.ErrorBadRequest("contact and code are required")
	}

	if !s.vcodeCache.Verify("bind_contact", contact, code) {
		return nil, authenticationV1.ErrorBadRequest("invalid or expired verification code")
	}

	// 写入 EMAIL 登录凭证（identifier=邮箱；credential 为占位哈希——该凭证不用于密码校验）
	if err = s.userCredentialRepo.Create(ctx, &authenticationV1.CreateUserCredentialRequest{
		Data: &authenticationV1.UserCredential{
			TenantId:       trans.Ptr(operator.GetTenantId()),
			UserId:         trans.Ptr(operator.GetUserId()),
			IdentityType:   authenticationV1.UserCredential_EMAIL.Enum(),
			Identifier:     trans.Ptr(contact),
			CredentialType: authenticationV1.UserCredential_PASSWORD_HASH.Enum(),
			Credential:     trans.Ptr(dummyPasswordHash),
			IsPrimary:      trans.Ptr(false),
			Status:         authenticationV1.UserCredential_ENABLED.Enum(),
		},
	}); err != nil {
		s.log.Errorf(ctx, "bind contact: create EMAIL credential failed for user [%d]: %s", operator.GetUserId(), err.Error())
		return nil, err
	}

	s.log.Infof(ctx, "user [%d] bound email [%s]", operator.GetUserId(), contact)
	return &emptypb.Empty{}, nil
}

// dummyPasswordHash 合法 bcrypt 占位（EMAIL 凭证不用于密码校验）
const dummyPasswordHash = "$2a$10$1sbpKmhQDpXLHnDnEQ1nLe3oOnYyP2bUJyqHcX2T0Fq1qfyoXOrPm"
