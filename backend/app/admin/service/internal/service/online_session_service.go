package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/timestamppb"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	onlineSessionV1 "go-wind-admin/api/gen/go/online_session/service/v1"

	"go-wind-admin/pkg/middleware/auth"

	"go-wind-admin/app/admin/service/internal/data"
)

// OnlineSessionService 提供在线会话视图（基于 Redis 令牌对元数据）与强制下线能力。
// 会话记录在令牌签发时写入、随 refresh token 过期或吊销时删除，
// 本服务只做读取与转发吊销，不持有独立存储。
type OnlineSessionService struct {
	adminV1.OnlineSessionServiceHTTPServer

	log           *bLogger.Helper
	authenticator *data.Authenticator
}

func NewOnlineSessionService(
	ctx *bootstrap.Context,
	authenticator *data.Authenticator,
) *OnlineSessionService {
	return &OnlineSessionService{
		log:           ctx.NewLoggerHelper("online-session/service/admin-service"),
		authenticator: authenticator,
	}
}

// ListOnlineSession 返回当前在线会话列表，支持按用户名/IP 关键词过滤，
// 按登录时间倒序排序后在内存中分页。
func (s *OnlineSessionService) ListOnlineSession(ctx context.Context, req *onlineSessionV1.ListOnlineSessionRequest) (*onlineSessionV1.ListOnlineSessionResponse, error) {
	entries, err := s.authenticator.ListSessionEntries(ctx)
	if err != nil {
		s.log.Errorf(ctx, "list session entries failed: %v", err)
		return nil, err
	}

	keyword := strings.ToLower(strings.TrimSpace(req.GetKeyword()))

	items := make([]*onlineSessionV1.OnlineSession, 0, len(entries))
	for _, entry := range entries {
		meta := entry.Meta
		if meta == nil {
			continue
		}
		if keyword != "" &&
			!strings.Contains(strings.ToLower(meta.Username), keyword) &&
			!strings.Contains(strings.ToLower(meta.Ip), keyword) {
			continue
		}

		items = append(items, &onlineSessionV1.OnlineSession{
			Jti:        trans.Ptr(entry.Jti),
			UserId:     trans.Ptr(entry.UserId),
			Username:   trans.Ptr(meta.Username),
			TenantId:   trans.Ptr(meta.TenantId),
			ClientType: trans.Ptr(entry.ClientType),
			IpAddress:  trans.Ptr(meta.Ip),
			UserAgent:  trans.Ptr(meta.UserAgent),
			DeviceId:   trans.Ptr(meta.DeviceId),
			LoginAt:    timestamppb.New(time.Unix(meta.LoginAt, 0)),
		})
	}

	// 登录时间倒序：最新登录的会话排前面
	sort.Slice(items, func(i, j int) bool {
		return items[i].GetLoginAt().AsTime().After(items[j].GetLoginAt().AsTime())
	})

	total := uint64(len(items))

	// 内存分页：page 从 1 开始，缺省返回前 20 条
	page := max(int(req.GetPage()), 1)
	pageSize := int(req.GetPageSize())
	if pageSize == 0 {
		pageSize = 20
	}
	start := (page - 1) * pageSize
	if start >= len(items) {
		items = nil
	} else {
		end := start + pageSize
		if end > len(items) {
			end = len(items)
		}
		items = items[start:end]
	}

	return &onlineSessionV1.ListOnlineSessionResponse{
		Items: items,
		Total: total,
	}, nil
}

// ListMyOnlineSession 返回当前登录用户的全部在线会话（个人中心自助视图）。
// 用户身份取自认证上下文，无需（也不允许）指定他人；按登录时间倒序，不分页
// （单用户会话数量有限）。current 标记当前请求所属会话，供前端禁用"踢自己"。
func (s *OnlineSessionService) ListMyOnlineSession(ctx context.Context, req *onlineSessionV1.ListMyOnlineSessionRequest) (*onlineSessionV1.ListOnlineSessionResponse, error) {
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	entries, err := s.authenticator.ListSessionEntries(ctx)
	if err != nil {
		s.log.Errorf(ctx, "list my session entries failed for user [%d]: %v", operator.GetUserId(), err)
		return nil, err
	}

	items := make([]*onlineSessionV1.OnlineSession, 0, len(entries))
	for _, entry := range entries {
		if entry.UserId != operator.GetUserId() {
			continue
		}
		meta := entry.Meta
		if meta == nil {
			continue
		}
		items = append(items, &onlineSessionV1.OnlineSession{
			Jti:        trans.Ptr(entry.Jti),
			UserId:     trans.Ptr(entry.UserId),
			Username:   trans.Ptr(meta.Username),
			TenantId:   trans.Ptr(meta.TenantId),
			ClientType: trans.Ptr(entry.ClientType),
			IpAddress:  trans.Ptr(meta.Ip),
			UserAgent:  trans.Ptr(meta.UserAgent),
			DeviceId:   trans.Ptr(meta.DeviceId),
			LoginAt:    timestamppb.New(time.Unix(meta.LoginAt, 0)),
			Current:    trans.Ptr(entry.Jti == operator.GetJti()),
		})
	}

	// 登录时间倒序：最新登录的会话排前面
	sort.Slice(items, func(i, j int) bool {
		return items[i].GetLoginAt().AsTime().After(items[j].GetLoginAt().AsTime())
	})

	return &onlineSessionV1.ListOnlineSessionResponse{
		Items: items,
		Total: uint64(len(items)),
	}, nil
}

// RevokeMyOnlineSession 当前用户强制下线自己的指定会话。
// 仅允许操作属于本人的会话：jti 以当前用户身份定位 Redis 键，
// 会话不存在（已过期/已下线）时明确报错而非静默成功。
func (s *OnlineSessionService) RevokeMyOnlineSession(ctx context.Context, req *onlineSessionV1.RevokeMyOnlineSessionRequest) (*onlineSessionV1.RevokeMyOnlineSessionResponse, error) {
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	jti := req.GetJti()
	if jti == "" {
		return nil, adminV1.ErrorBadRequest("jti is required")
	}

	// 存在性校验：键由 (clientType, 当前uid, jti) 构成，天然限定只能操作本人会话；
	// 不存在时报错，避免用户误以为下线成功
	meta, err := s.authenticator.GetSessionMeta(ctx, req.GetClientType(), operator.GetUserId(), jti)
	if err != nil {
		s.log.Errorf(ctx, "get my session meta failed for user [%d] jti [%s]: %v", operator.GetUserId(), jti, err)
		return nil, err
	}
	if meta == nil {
		return nil, adminV1.ErrorNotFound("session not found or already offline")
	}

	if err = s.authenticator.RevokeTokenByJti(ctx, trans.Ptr(req.GetClientType()), operator.GetUserId(), jti); err != nil {
		s.log.Errorf(ctx, "revoke my session failed for user [%d] jti [%s]: %v", operator.GetUserId(), jti, err)
		return nil, err
	}

	s.log.Infof(ctx, "user [%d] revoked own session [%s]", operator.GetUserId(), jti)
	return &onlineSessionV1.RevokeMyOnlineSessionResponse{}, nil
}

// ForceLogoutSession 强制下线指定会话：吊销其访问令牌、刷新令牌与会话元数据。
func (s *OnlineSessionService) ForceLogoutSession(ctx context.Context, req *onlineSessionV1.ForceLogoutSessionRequest) (*onlineSessionV1.ForceLogoutSessionResponse, error) {
	userId := req.GetUserId()
	jti := req.GetJti()
	if userId == 0 || jti == "" {
		return nil, adminV1.ErrorBadRequest("user id and jti are required")
	}

	if err := s.authenticator.RevokeTokenByJti(ctx, trans.Ptr(req.GetClientType()), userId, jti); err != nil {
		s.log.Errorf(ctx, "force logout session failed for user [%d] jti [%s]: %v", userId, jti, err)
		return nil, err
	}

	s.log.Infof(ctx, "session [%s] of user [%d] force logged out", jti, userId)
	return &onlineSessionV1.ForceLogoutSessionResponse{}, nil
}
