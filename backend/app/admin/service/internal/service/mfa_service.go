package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image/png"
	"strconv"
	"time"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent/privacy"
	"go-wind-admin/app/admin/service/internal/data/ent/usermfafactor"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	ktransport "github.com/go-kratos/kratos/v2/transport"

	"go-wind-admin/pkg/middleware/auth"
	"go-wind-admin/pkg/netutil"

	"github.com/tx7do/go-crud/viewer"

	"github.com/pquerna/otp"
	otpTotp "github.com/pquerna/otp/totp"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
)

const (
	// mfaTotpIssuer TOTP otpauth URI 中的发行方标识（认证器 App 内展示归属）。
	mfaTotpIssuer = "GoWindAdmin"
	// mfaTotpSkew 允许的时间窗口偏移（±1 个 30s 周期），防客户端/服务端时钟小幅漂移。
	mfaTotpSkew = 1
)

// MfaService 实现 adminV1.MfaServiceHTTPServer。
//
// 本轮仅落地 TOTP：
//   - 管理面（GetMFAStatus/ListEnrolledMethods/StartEnrollMethod/ConfirmEnrollMethod/DisableMFA/RevokeMFADevice）
//     需登录态，operator 从 auth.FromContext(ctx) 取得，强制只能操作本人因子。
//   - 登录挑战面（VerifyMFAChallenge）免鉴权，operation_id 由登录流程签发并存入
//     MfaChallengeCache（含 UserTokenPayload + ClientType），验证通过后用 authenticator 签发真 token。
//
// 非 TOTP 方法、StartMFAChallenge、备份码相关 RPC 本轮返回 UNIMPLEMENTED。
type MfaService struct {
	log *bLogger.Helper

	mfaFactorRepo     *data.UserMfaFactorRepo
	mfaChallengeCache *data.MfaChallengeCache
	authenticator     *data.Authenticator
	rateLimiter       *data.LoginRateLimiter
}

func NewMfaService(
	ctx *bootstrap.Context,
	mfaFactorRepo *data.UserMfaFactorRepo,
	mfaChallengeCache *data.MfaChallengeCache,
	authenticator *data.Authenticator,
	rateLimiter *data.LoginRateLimiter,
) *MfaService {
	return &MfaService{
		log:               ctx.NewLoggerHelper("mfa/service/admin-service"),
		mfaFactorRepo:     mfaFactorRepo,
		mfaChallengeCache: mfaChallengeCache,
		authenticator:     authenticator,
		rateLimiter:       rateLimiter,
	}
}

// GetMFAStatus 查询当前登录用户 MFA 总览。
func (s *MfaService) GetMFAStatus(ctx context.Context, _ *authenticationV1.GetMFAStatusRequest) (*authenticationV1.GetMFAStatusResponse, error) {
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	uid := operator.GetUserId()
	tid := operator.GetTenantId()
	infos, err := s.mfaFactorRepo.ListByUser(ctx, tid, uid)
	if err != nil {
		return nil, authenticationV1.ErrorInternalServerError("query mfa status failed")
	}
	hasTotp := false
	for _, i := range infos {
		if i.Method == authenticationV1.MFAMethod_TOTP && i.Enabled {
			hasTotp = true
			break
		}
	}
	resp := &authenticationV1.GetMFAStatusResponse{
		Enabled:     hasTotp,
		Enforcement: authenticationV1.MFAEnforcement_MFA_NOT_REQUIRED,
	}
	if hasTotp {
		resp.Enforcement = authenticationV1.MFAEnforcement_MFA_REQUIRED
		resp.Enrolled = buildEnrolledProto(infos)
	}
	return resp, nil
}

// ListEnrolledMethods 列出当前登录用户的 MFA 因子（不含 secret）。
func (s *MfaService) ListEnrolledMethods(ctx context.Context, _ *authenticationV1.ListEnrolledMethodsRequest) (*authenticationV1.ListEnrolledMethodsResponse, error) {
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	infos, err := s.mfaFactorRepo.ListByUser(ctx, operator.GetTenantId(), operator.GetUserId())
	if err != nil {
		return nil, authenticationV1.ErrorInternalServerError("list mfa methods failed")
	}
	return &authenticationV1.ListEnrolledMethodsResponse{
		Items: buildEnrolledProto(infos),
	}, nil
}

// StartEnrollMethod 开始注册 MFA 方法。仅 TOTP 本轮实现。
func (s *MfaService) StartEnrollMethod(ctx context.Context, req *authenticationV1.StartEnrollMethodRequest) (*authenticationV1.StartEnrollMethodResponse, error) {
	if req.GetMethod() != authenticationV1.MFAMethod_TOTP {
		return nil, authenticationV1.ErrorBadRequest("only TOTP is supported")
	}
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	// 频控：同一用户 30s 冷却期内只允许发起一次注册（防循环 Start 塞 Redis）
	if !s.mfaChallengeCache.TryAcquireEnrollCooldown(ctx, operator.GetTenantId(), operator.GetUserId()) {
		return nil, authenticationV1.ErrorBadRequest("enroll request too frequent, retry later")
	}

	// 预检：已绑定 TOTP 时拒绝重复注册（(tenant,user,method) 唯一约束兜底，
	// 但提前返回友好错误，避免 Confirm 阶段撞唯一索引报笼统 500）。
	// 重新绑定需先 DisableMFA 解绑。
	if has, herr := s.mfaFactorRepo.HasEnabledTotp(ctx, operator.GetTenantId(), operator.GetUserId()); herr != nil {
		return nil, authenticationV1.ErrorInternalServerError("check mfa status failed")
	} else if has {
		return nil, authenticationV1.ErrorBadRequest("totp already enrolled, disable it first")
	}

	// 生成 TOTP key：issuer 标识发行方，account 用 userId 防 App 内同名冲突。
	key, err := otpTotp.Generate(otpTotp.GenerateOpts{
		Issuer:      mfaTotpIssuer,
		AccountName: fmt.Sprintf("uid:%d", operator.GetUserId()),
	})
	if err != nil {
		s.log.Errorf(ctx, "generate totp key failed: %s", err.Error())
		return nil, authenticationV1.ErrorInternalServerError("generate totp key failed")
	}

	opId, err := s.mfaChallengeCache.SetEnrollChallenge(ctx, key.Secret(), operator.GetTenantId(), operator.GetUserId())
	if err != nil {
		s.log.Errorf(ctx, "set enroll challenge failed: %s", err.Error())
		return nil, authenticationV1.ErrorInternalServerError("start enroll failed")
	}

	qrUri, err := totpQrDataUri(key)
	if err != nil {
		s.log.Errorf(ctx, "encode totp qr failed: %s", err.Error())
		return nil, authenticationV1.ErrorInternalServerError("encode totp qr failed")
	}

	return &authenticationV1.StartEnrollMethodResponse{
		Result: &authenticationV1.StartEnrollMethodResponse_Totp{
			Totp: &authenticationV1.TOTPResult{
				Secret:        key.Secret(),
				OtpAuthUrl:    key.URL(),
				QrCodeDataUri: qrUri,
			},
		},
		OperationId: opId,
		ExpiresAt:   timestamppb.New(time.Now().Add(data.MfaChallengeTTL)),
	}, nil
}

// ConfirmEnrollMethod 确认注册：校验首码通过则落库因子。
func (s *MfaService) ConfirmEnrollMethod(ctx context.Context, req *authenticationV1.ConfirmEnrollMethodRequest) (*authenticationV1.ConfirmEnrollMethodResponse, error) {
	if req.GetMethod() != authenticationV1.MFAMethod_TOTP {
		return nil, authenticationV1.ErrorBadRequest("only TOTP is supported")
	}
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	// 注册流程允许首码输错重试：peek 不消耗，落库成功才删（与登录挑战的
	// 取出即删不同——登录挑战失败必须作废防暴破，注册有登录态且首码空间同样 10^6，
	// 重试面由 operation TTL + 已绑定预检 + 唯一索引共同约束）
	enrollCtx, err := s.mfaChallengeCache.PeekEnrollChallenge(ctx, req.GetOperationId())
	if err != nil {
		return nil, authenticationV1.ErrorBadRequest("invalid or expired enroll operation")
	}
	// 防 operation_id 跨用户劫持：注册上下文绑定的人必须等于当前 operator
	if enrollCtx.TenantID != operator.GetTenantId() || enrollCtx.UserID != operator.GetUserId() {
		return nil, authenticationV1.ErrorForbidden("enroll operation user mismatch")
	}

	// 校验首码：用默认 Validate（±1 窗口，等同 ValidateCustom 的默认参数语义）
	if !otpTotp.Validate(req.GetTotpCode(), enrollCtx.Secret) {
		return &authenticationV1.ConfirmEnrollMethodResponse{Success: false}, nil
	}

	factorId, err := s.mfaFactorRepo.CreateTotpFactor(ctx, enrollCtx.TenantID, enrollCtx.UserID, enrollCtx.Secret, req.GetDisplay())
	if err != nil {
		return nil, authenticationV1.ErrorInternalServerError("create mfa factor failed")
	}
	s.mfaChallengeCache.DeleteEnrollChallenge(ctx, req.GetOperationId())
	return &authenticationV1.ConfirmEnrollMethodResponse{
		Success:      true,
		CredentialId: fmt.Sprintf("%d", factorId),
	}, nil
}

// VerifyMFAChallenge 验证登录 MFA 挑战。通过则签发真 token 并返回 LoginResponse。
// 免鉴权：operation_id 由登录流程在密码校验通过、待二次验证阶段签发。
func (s *MfaService) VerifyMFAChallenge(ctx context.Context, req *authenticationV1.VerifyMFAChallengeRequest) (*authenticationV1.LoginResponse, error) {
	// 本接口在鉴权白名单内、无 auth 中间件注入 viewer，而 ent 隐私层
	// （TenantPrivacy mixin）要求 ViewerContext 存在——与登录流程的
	// resetContextForLogin 同款处理：Noop viewer + privacy.Allow。
	// 挑战上下文绑定的 userId/tenantId 来自密码校验通过后的 tokenPayload，
	// 越权面由 operation_id 单次有效 + 归属校验兜住。
	ctx = viewer.WithContext(ctx, viewer.NewNoopContext())
	ctx = privacy.DecisionContext(ctx, privacy.Allow)

	// Peek 不消耗：允许失败重试（上限见 RecordLoginFailure），通过或超限才 Consume
	challengeCtx, err := s.mfaChallengeCache.PeekLoginChallenge(ctx, req.GetOperationId())
	if err != nil {
		return nil, authenticationV1.ErrorBadRequest("invalid or expired mfa operation")
	}

	payload := challengeCtx.Payload
	if payload == nil {
		return nil, authenticationV1.ErrorInternalServerError("mfa challenge payload missing")
	}

	// 审计贯通：本接口请求体无 username（中间件解析不到），无论成功失败
	// 都通过响应头把挑战上下文中的用户名回传给登录审计中间件（兜底填充）。
	// 非 ASCII 用户名会构成非法 HTTP 头值（kratos 会丢弃整个响应头），过滤之。
	if tr, tok := ktransport.FromServerContext(ctx); tok {
		if htr, hok := tr.(*khttp.Transport); hok && isASCII(payload.GetUsername()) {
			htr.ReplyHeader().Set("X-Audit-Username", payload.GetUsername())
		}
	}
	uid := payload.GetUserId()
	tid := payload.GetTenantId()

	// 取该用户 ENABLED 的 TOTP 因子并解密 secret
	factorId, plainSecret, err := s.mfaFactorRepo.FindEnabledTotpForUser(ctx, tid, uid)
	if err != nil {
		// 因子缺失（如验证期间被解绑/被管理员重置）：作废挑战走重新登录
		s.log.Errorf(ctx, "find totp factor for login mfa failed uid=%d: %s", uid, err.Error())
		s.mfaChallengeCache.ConsumeLoginChallenge(ctx, req.GetOperationId())
		return nil, authenticationV1.ErrorForbidden("mfa verification failed")
	}

	// 校验 TOTP 码：±1 窗口（防时钟漂移），默认周期 30s、6 位、SHA1。
	// digits/algorithm 用常量：totp.Generate 产出的 key 恒为 6 位 SHA1，这里与之对齐。
	ok, verr := otpTotp.ValidateCustom(req.GetTotpCode(), plainSecret, time.Now(), otpTotp.ValidateOpts{
		Period:    30,
		Skew:      mfaTotpSkew,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if verr != nil || !ok {
		// 错码：记失败计数，达上限作废挑战；同时计入登录限流（IP+用户名双维度）
		exceeded := s.mfaChallengeCache.RecordLoginFailure(ctx, req.GetOperationId())
		if s.rateLimiter != nil {
			if _, _, _, cerr := s.rateLimiter.CheckAndIncr(ctx, netutil.ClientIPFromContext(ctx), payload.GetUsername()); cerr != nil {
				s.log.Errorf(ctx, "mfa rate limiter incr failed: %s", cerr.Error())
			}
		}
		if exceeded {
			return nil, authenticationV1.ErrorForbidden("too many invalid mfa attempts, please login again")
		}
		return nil, authenticationV1.ErrorForbidden("invalid mfa code")
	}

	// 通过：原子消耗挑战做最终裁决（并发同 opId 仅先抢到者发 token，防双花），
	// 更新 last_used_at（best-effort）、清零限流，签发真 token
	if !s.mfaChallengeCache.TakeLoginChallengeAtomic(ctx, req.GetOperationId()) {
		return nil, authenticationV1.ErrorForbidden("mfa challenge already consumed")
	}
	_ = s.mfaFactorRepo.UpdateLastUsed(ctx, tid, uid, factorId, time.Now())

	// 兑现登录 MFA 闸门的承诺："限流清零在 VerifyMFAChallenge 通过后"。
	// 不清零会导致绑定 MFA 的用户失败计数只增不减（密码错误与 MFA 登录成功
	// 都不 Reset），比未绑定用户更容易触发限流锁定。
	if s.rateLimiter != nil {
		s.rateLimiter.Reset(ctx, netutil.ClientIPFromContext(ctx), payload.GetUsername())
	}


	accessToken, refreshToken, err := s.authenticator.CreateUserToken(ctx, challengeCtx.ClientType, payload)
	if err != nil {
		return nil, err
	}

	return &authenticationV1.LoginResponse{
		TokenType:        authenticationV1.TokenType_bearer,
		AccessToken:      accessToken,
		RefreshToken:     trans.Ptr(refreshToken),
		ExpiresIn:        int64(s.authenticator.GetAccessTokenExpires(challengeCtx.ClientType).Seconds()),
		RefreshExpiresIn: trans.Ptr(int64(s.authenticator.GetRefreshTokenExpires(challengeCtx.ClientType).Seconds())),
	}, nil
}

// DisableMFA 禁用/移除 MFA 凭证。
//   - 不传 user_id：操作当前登录用户本人（解绑自己的因子）。
//   - 传 user_id 指定他人：仅平台管理员允许，用于用户认证器丢失时的救援重置
//     （按 method 清空该用户该方法全部因子），记告警日志留痕。
func (s *MfaService) DisableMFA(ctx context.Context, req *authenticationV1.DisableMFARequest) (*emptypb.Empty, error) {
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	// 管理端救援路径：目标用户由因子行定位（凭 credential_id），或按 user_id+method 清空
	if target := req.GetUserId(); target != 0 && target != operator.GetUserId() {
		if !operator.GetIsPlatformAdmin() {
			return nil, authenticationV1.ErrorForbidden("only platform admin can reset mfa for others")
		}

		if req.GetCredentialId() != "" {
			factorId, perr := parseFactorId(req.GetCredentialId())
			if perr != nil {
				return nil, authenticationV1.ErrorBadRequest("invalid credential id")
			}
			tid, uid, found, gerr := s.mfaFactorRepo.GetFactorById(ctx, factorId)
			if gerr != nil {
				return nil, authenticationV1.ErrorInternalServerError("disable mfa failed")
			}
			if !found || uid != target {
				return nil, authenticationV1.ErrorNotFound("mfa credential not found")
			}
			ok, derr := s.mfaFactorRepo.DeleteForUser(ctx, tid, uid, factorId)
			if derr != nil || !ok {
				return nil, authenticationV1.ErrorInternalServerError("disable mfa failed")
			}
		} else {
			method, merr := methodToEntity(req.GetMethod())
			if merr != nil {
				return nil, authenticationV1.ErrorBadRequest("method required when resetting by user")
			}
			// 目标用户可能在任意租户：以平台管理员视角全量查找其因子行定位 tenant
			tid, uid, found, gerr := s.mfaFactorRepo.FindFirstByUser(ctx, target, method)
			if gerr != nil {
				return nil, authenticationV1.ErrorInternalServerError("disable mfa failed")
			}
			if !found {
				return nil, authenticationV1.ErrorNotFound("mfa credential not found")
			}
			if _, derr := s.mfaFactorRepo.DeleteAllByUserMethod(ctx, tid, uid, method); derr != nil {
				return nil, authenticationV1.ErrorInternalServerError("disable mfa failed")
			}
		}

		s.log.Warnf(ctx, "admin [%d] reset mfa for user [%d], reason=%s", operator.GetUserId(), target, req.GetReason())
		return &emptypb.Empty{}, nil
	}

	// 本人路径：按 credential_id 精确解绑；未传 credential_id 时按 method 清空
	// 本人该方法全部因子（兑现 proto 注释"指定凭证 id 或仅按方法禁用全部"的契约）
	if req.GetCredentialId() == "" {
		method, merr := methodToEntity(req.GetMethod())
		if merr != nil {
			return nil, authenticationV1.ErrorBadRequest("credential_id or method required")
		}
		n, derr := s.mfaFactorRepo.DeleteAllByUserMethod(ctx, operator.GetTenantId(), operator.GetUserId(), method)
		if derr != nil {
			return nil, authenticationV1.ErrorInternalServerError("disable mfa failed")
		}
		if n == 0 {
			return nil, authenticationV1.ErrorNotFound("mfa credential not found")
		}
		return &emptypb.Empty{}, nil
	}
	factorId, err := parseFactorId(req.GetCredentialId())
	if err != nil {
		return nil, authenticationV1.ErrorBadRequest("invalid credential id")
	}
	ok, err := s.mfaFactorRepo.DeleteForUser(ctx, operator.GetTenantId(), operator.GetUserId(), factorId)
	if err != nil {
		return nil, authenticationV1.ErrorInternalServerError("disable mfa failed")
	}
	if !ok {
		return nil, authenticationV1.ErrorNotFound("mfa credential not found")
	}
	return &emptypb.Empty{}, nil
}

// RevokeMFADevice 撤销指定 MFA 凭证（按 id），强制归属校验。
func (s *MfaService) RevokeMFADevice(ctx context.Context, req *authenticationV1.RevokeMFADeviceRequest) (*emptypb.Empty, error) {
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	factorId, err := parseFactorId(req.GetCredentialId())
	if err != nil {
		return nil, authenticationV1.ErrorBadRequest("invalid credential id")
	}
	ok, err := s.mfaFactorRepo.DeleteForUser(ctx, operator.GetTenantId(), operator.GetUserId(), factorId)
	if err != nil {
		return nil, authenticationV1.ErrorInternalServerError("revoke mfa device failed")
	}
	if !ok {
		return nil, authenticationV1.ErrorNotFound("mfa credential not found")
	}
	return &emptypb.Empty{}, nil
}

// buildEnrolledProto 将仓储返回的因子元信息列表转为 proto EnrolledMethod 列表。
// 不含 secret。
func buildEnrolledProto(infos []data.EnrolledFactorInfo) []*authenticationV1.EnrolledMethod {
	out := make([]*authenticationV1.EnrolledMethod, 0, len(infos))
	for _, i := range infos {
		em := &authenticationV1.EnrolledMethod{
			Id:         fmt.Sprintf("%d", i.ID),
			Method:     i.Method,
			Display:    i.DisplayName,
			Enabled:    i.Enabled,
			CreatedAt:  nil,
			LastUsedAt: nil,
		}
		if i.CreatedAt != nil {
			em.CreatedAt = timestamppb.New(*i.CreatedAt)
		}
		if i.LastUsedAt != nil {
			em.LastUsedAt = timestamppb.New(*i.LastUsedAt)
		}
		out = append(out, em)
	}
	return out
}

// methodToEntity 将 proto MFAMethod 映射为 ent 因子方法枚举（仅 TOTP 本轮支持）。
func methodToEntity(m authenticationV1.MFAMethod) (usermfafactor.Method, error) {
	switch m {
	case authenticationV1.MFAMethod_TOTP:
		return usermfafactor.MethodTotp, nil
	}
	return "", fmt.Errorf("unsupported mfa method: %v", m)
}

// isASCII 判断字符串是否全为 ASCII（HTTP 头值合法性）。
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 0x7f {
			return false
		}
	}
	return s != ""
}

// parseFactorId 将 proto 返回的字符串形式 credential_id 解析为 uint32 主键。
func parseFactorId(s string) (uint32, error) {
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(v), nil
}

// totpQrDataUri 将 otp.Key 的二维码图渲染为 PNG data URI，供前端 <img> 展示。
// 生成器（totp.Generate）产生的 key 是标准 otpauth URI，编码为二维码后可被
// Google Authenticator 等认证器 App 扫码导入。
func totpQrDataUri(key *otp.Key) (string, error) {
	if key == nil {
		return "", fmt.Errorf("nil totp key")
	}
	img, err := key.Image(240, 240)
	if err != nil {
		return "", fmt.Errorf("render qr failed: %w", err)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", fmt.Errorf("encode qr png failed: %w", err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
