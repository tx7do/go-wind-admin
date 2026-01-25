package auth

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
)

// AccessTokenChecker 定义访问令牌检查接口
type AccessTokenChecker interface {
	// IsValidAccessToken 检查访问令牌是否有效
	IsValidAccessToken(ctx context.Context, accessToken string, skipRedis bool) (valid bool, payload *authenticationV1.UserTokenPayload)

	// IsBlockedAccessToken 检查访问令牌是否被阻止
	IsBlockedAccessToken(ctx context.Context, accessToken string) (blocked bool)
}

type AccessTokenCheckerFunc func(ctx context.Context, accessToken string, skipRedis bool) (bool, *authenticationV1.UserTokenPayload)

func (f AccessTokenCheckerFunc) IsValidAccessToken(ctx context.Context, accessToken string, skipRedis bool) (bool, *authenticationV1.UserTokenPayload) {
	return f(ctx, accessToken, skipRedis)
}

type AccessTokenBlockerFunc func(ctx context.Context, accessToken string) bool

func (f AccessTokenBlockerFunc) IsBlockedAccessToken(ctx context.Context, accessToken string) bool {
	return f(ctx, accessToken)
}

// composedChecker 将单独的 valid/block 函数组合成一个 AccessTokenChecker
type composedChecker struct {
	valid   AccessTokenCheckerFunc
	blocker AccessTokenBlockerFunc
}

// NewAccessTokenCheckerFromFuncs 构造组合检查器
func NewAccessTokenCheckerFromFuncs(valid AccessTokenCheckerFunc, blocker AccessTokenBlockerFunc) AccessTokenChecker {
	return &composedChecker{
		valid:   valid,
		blocker: blocker,
	}
}

func (c *composedChecker) IsValidAccessToken(ctx context.Context, accessToken string, skipRedis bool) (bool, *authenticationV1.UserTokenPayload) {
	if c.valid == nil {
		// 默认认为有效（或根据需要返回 false）
		return true, nil
	}
	return c.valid(ctx, accessToken, skipRedis)
}

func (c *composedChecker) IsBlockedAccessToken(ctx context.Context, accessToken string) bool {
	if c.blocker == nil {
		// 默认不被阻止
		return false
	}
	return c.blocker(ctx, accessToken)
}

type options struct {
	log *log.Helper

	accessTokenChecker                AccessTokenChecker // 访问令牌检查器
	enableCheckRefreshTokenExpiration bool               // 是否启用刷新令牌过期检查
	enableCheckScopes                 bool               // 是否启用作用域检查

	enableAuthz bool // 是否启用鉴权

	injectOperatorId bool
	injectTenantId   bool
	injectEnt        bool
	injectMetadata   bool
}

type Option func(*options)

// WithAccessTokenChecker 设置访问令牌检查器
func WithAccessTokenChecker(checker AccessTokenChecker) Option {
	return func(opts *options) {
		opts.accessTokenChecker = checker
	}
}

func WithAccessTokenCheckerFromFuncs(valid AccessTokenCheckerFunc, blocker AccessTokenBlockerFunc) Option {
	return func(opts *options) {
		opts.accessTokenChecker = NewAccessTokenCheckerFromFuncs(valid, blocker)
	}
}

func WithEnableCheckRefreshTokenExpiration(enable bool) Option {
	return func(opts *options) {
		opts.enableCheckRefreshTokenExpiration = enable
	}
}

// WithEnableCheckScopes 设置是否启用作用域检查
func WithEnableCheckScopes(enable bool) Option {
	return func(opts *options) {
		opts.enableCheckScopes = enable
	}
}

// WithInjectOperatorId 设置是否注入操作员ID
func WithInjectOperatorId(enable bool) Option {
	return func(opts *options) {
		opts.injectOperatorId = enable
	}
}

// WithInjectTenantId 设置是否注入租户ID
func WithInjectTenantId(enable bool) Option {
	return func(opts *options) {
		opts.injectTenantId = enable
	}
}

// WithInjectEnt 设置是否注入Ent客户端
func WithInjectEnt(enable bool) Option {
	return func(opts *options) {
		opts.injectEnt = enable
	}
}

// WithInjectMetadata 设置是否注入元数据
func WithInjectMetadata(enable bool) Option {
	return func(opts *options) {
		opts.injectMetadata = enable
	}
}

// WithEnableAuthority 设置是否启用鉴权
func WithEnableAuthority(enable bool) Option {
	return func(opts *options) {
		opts.enableAuthz = enable
	}
}

// WithLogger 设置日志记录器
func WithLogger(logger log.Logger) Option {
	return func(o *options) {
		o.log = log.NewHelper(log.With(logger, "module", "auth.middleware"))
	}
}
