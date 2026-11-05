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

	"github.com/tx7do/go-utils/notify/sms"
)

// 联系方式种类：绑定验证码按它分渠道（邮箱走 SMTP，手机号走 SMS）。
const (
	contactKindEmail = "email"
	contactKindPhone = "phone"
)

// sendContactVCode 向指定联系方式发送业务验证码（绑定场景），按种类分渠道。
//
// 手机号一路显式点名 Channel=SMS：存量库没有 PHONE_BIND_CODE 的路由规则行
// （播种只发生 在空表时），依赖路由等于让手机绑定在老实例上必失败。
// 发码频控（同联系方式 60s 冷却）：该端点已鉴权，超频直接告知调用方而非静默。
func (s *UserProfileService) sendContactVCode(ctx context.Context, kind, contact string) error {
	if s.vcodeCache != nil && !s.vcodeCache.Throttle("bind_contact", contact, time.Minute) {
		return authenticationV1.ErrorBadRequest("verification code sent too frequently, please retry later")
	}

	code, err := generateVCode()
	if err != nil {
		return authenticationV1.ErrorInternalServerError("generate verification code failed")
	}
	if err = s.vcodeCache.Save("bind_contact", contact, code, 10*time.Minute); err != nil {
		return authenticationV1.ErrorInternalServerError("save verification code failed")
	}

	// 文案语言：绑定场景操作者即收件人，读其偏好 locale（未设置回落请求头判定）。
	if operator, opErr := auth.FromContext(ctx); opErr == nil && operator.GetTenantId() > 0 {
		if user, uErr := s.userRepo.Get(ctx, &identityV1.GetUserRequest{
			QueryBy: &identityV1.GetUserRequest_Id{Id: operator.GetUserId()},
		}); uErr == nil && user.GetLocale() != "" {
			if loc, ok := mailtext.LocaleOfTag(user.GetLocale()); ok {
				ctx = mailtext.WithLocale(ctx, loc)
			}
		}
	}

	sendReq := &notificationV1.SendDirectNotificationRequest{
		Target:  contact,
		Content: "",
	}
	switch kind {
	case contactKindPhone:
		sendReq.EventType = notificationV1.EventType_PHONE_BIND_CODE
		sendReq.Channel = trans.Ptr(notificationV1.Channel_SMS)
		// 短信正文由服务商审核模板渲染，code 经 params 交给渠道；台账文案只做记录。
		sendReq.Params = map[string]string{"code": code}
		sendReq.Title, sendReq.Content = mailtext.PhoneBindCode(ctx, code)
	default:
		sendReq.EventType = notificationV1.EventType_CONTACT_BIND_CODE
		sendReq.Title, sendReq.Content = renderContactBindCode(ctx, s.mailer, code)
	}

	resp, err := s.notifier.SendDirect(ctx, sendReq)
	if err != nil {
		s.log.Errorf(ctx, "bind-contact: send %s code to [%s] failed (delivery %d): %s",
			kind, contact, resp.GetDeliveryId(), err.Error())
		if resp.GetStatus() == notificationV1.DeliveryStatus_SKIPPED {
			if kind == contactKindPhone {
				return authenticationV1.ErrorInternalServerError("sms channel is not configured")
			}
			return authenticationV1.ErrorInternalServerError("email channel is not configured")
		}
		return authenticationV1.ErrorInternalServerError("send verification message failed")
	}
	return nil
}

// BindContact 绑定手机号码/邮箱（第一步）：向新联系方式发送验证码。
// 短信渠道接入后 MOBILE 分支已启用：验证码经 SMS 渠道下发（需已配置启用的短信渠道）。
func (s *UserProfileService) BindContact(ctx context.Context, req *identityV1.BindContactRequest) (*emptypb.Empty, error) {
	if phone := req.GetPhone(); phone != nil {
		contact := strings.TrimSpace(phone.GetPhone())
		if contact == "" {
			return nil, authenticationV1.ErrorBadRequest("phone is required")
		}
		// 服务入口先做一次与投递侧同判据的格式校验：错误在第一步就回给用户，
		// 而不是等 SendDirect 把它记成一条 SKIPPED 台账。
		if _, _, err := sms.NormalizePhoneNumber(contact); err != nil {
			return nil, authenticationV1.ErrorBadRequest("invalid phone number")
		}
		if err := s.sendContactVCode(ctx, contactKindPhone, contact); err != nil {
			return nil, err
		}
		return &emptypb.Empty{}, nil
	}

	if bindEmail := req.GetEmail(); bindEmail != nil {
		contact := strings.TrimSpace(bindEmail.GetEmail())
		if contact == "" {
			return nil, authenticationV1.ErrorBadRequest("email is required")
		}
		if err := s.sendContactVCode(ctx, contactKindEmail, contact); err != nil {
			return nil, err
		}
		return &emptypb.Empty{}, nil
	}

	return nil, authenticationV1.ErrorBadRequest("contact is required")
}

// VerifyContact 校验验证码并完成联系方式绑定（写入对应类型的登录凭证）。
// 凭证 identifier=联系方式；credential 为占位哈希——该凭证不用于密码校验。
func (s *UserProfileService) VerifyContact(ctx context.Context, req *identityV1.VerifyContactRequest) (*emptypb.Empty, error) {
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	var kind, contact, code string
	if verifyPhone := req.GetPhone(); verifyPhone != nil {
		kind, contact, code = contactKindPhone, strings.TrimSpace(verifyPhone.GetPhone()), strings.TrimSpace(verifyPhone.GetCode())
	} else if verifyEmail := req.GetEmail(); verifyEmail != nil {
		kind, contact, code = contactKindEmail, strings.TrimSpace(verifyEmail.GetEmail()), strings.TrimSpace(verifyEmail.GetCode())
	}
	if contact == "" || code == "" {
		return nil, authenticationV1.ErrorBadRequest("contact and code are required")
	}

	if !s.vcodeCache.Verify("bind_contact", contact, code) {
		return nil, authenticationV1.ErrorBadRequest("invalid or expired verification code")
	}

	identityType := authenticationV1.UserCredential_EMAIL
	if kind == contactKindPhone {
		identityType = authenticationV1.UserCredential_PHONE
	}

	if err = s.userCredentialRepo.Create(ctx, &authenticationV1.CreateUserCredentialRequest{
		Data: &authenticationV1.UserCredential{
			TenantId:       trans.Ptr(operator.GetTenantId()),
			UserId:         trans.Ptr(operator.GetUserId()),
			IdentityType:   identityType.Enum(),
			Identifier:     trans.Ptr(contact),
			CredentialType: authenticationV1.UserCredential_PASSWORD_HASH.Enum(),
			Credential:     trans.Ptr(dummyPasswordHash),
			IsPrimary:      trans.Ptr(false),
			Status:         authenticationV1.UserCredential_ENABLED.Enum(),
		},
	}); err != nil {
		s.log.Errorf(ctx, "bind contact: create %s credential failed for user [%d]: %s", identityType.String(), operator.GetUserId(), err.Error())
		return nil, err
	}

	s.log.Infof(ctx, "user [%d] bound %s [%s]", operator.GetUserId(), kind, contact)
	return &emptypb.Empty{}, nil
}

// dummyPasswordHash 合法 bcrypt 占位（EMAIL/PHONE 凭证不用于密码校验）
const dummyPasswordHash = "$2a$10$1sbpKmhQDpXLHnDnEQ1nLe3oOnYyP2bUJyqHcX2T0Fq1qfyoXOrPm"
