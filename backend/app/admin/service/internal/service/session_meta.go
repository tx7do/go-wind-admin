package service

import (
	"context"
	"time"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	"github.com/tx7do/go-utils/netutil"

	"go-wind-admin/app/admin/service/internal/data"
)

// recordSessionMeta 令牌对签发成功后记录会话元数据（在线会话列表展示用）。
// 登录时间取当前时间。最佳努力：记录失败仅记日志，不阻断登录主流程。
func recordSessionMeta(
	ctx context.Context,
	log *bLogger.Helper,
	authenticator *data.Authenticator,
	clientType authenticationV1.ClientType,
	tokenPayload *authenticationV1.UserTokenPayload,
) {
	recordSessionMetaAt(ctx, log, authenticator, clientType, tokenPayload, 0)
}

// recordSessionMetaAt 同 recordSessionMeta，但允许显式指定登录时间（unix 秒）。
// 刷新轮换时传入旧会话的登录时间，使在线列表展示的"登录时间"不随刷新重置。
func recordSessionMetaAt(
	ctx context.Context,
	log *bLogger.Helper,
	authenticator *data.Authenticator,
	clientType authenticationV1.ClientType,
	tokenPayload *authenticationV1.UserTokenPayload,
	loginAt int64,
) {
	if authenticator == nil || tokenPayload == nil || tokenPayload.GetJti() == "" {
		return
	}
	if loginAt <= 0 {
		loginAt = time.Now().Unix()
	}

	// User-Agent 经 HTTP header 透传；非 HTTP 传输（如任务/SSE 内部调用）取不到时留空
	ua := ""
	if header := netutil.HeaderFromContext(ctx); header != nil {
		ua = header.Get("User-Agent")
	}

	meta := &data.SessionMeta{
		Username:  tokenPayload.GetUsername(),
		TenantId:  tokenPayload.GetTenantId(),
		Ip:        netutil.ClientIPFromContext(ctx),
		UserAgent: ua,
		DeviceId:  tokenPayload.GetDeviceId(),
		LoginAt:   loginAt,
	}

	err := authenticator.SaveSessionMeta(
		ctx,
		clientType,
		tokenPayload.GetUserId(),
		tokenPayload.GetJti(),
		meta,
		authenticator.GetRefreshTokenExpires(clientType),
	)
	if err != nil {
		log.Errorf(ctx, "save session meta failed for user [%d]: %v", tokenPayload.GetUserId(), err)
	}
}
