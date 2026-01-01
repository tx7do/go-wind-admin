package auth

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
)

// AccessTokenChecker 定义访问令牌检查接口
type AccessTokenChecker interface {
	Exists(ctx context.Context, userID uint32, accessToken string) bool
}

type AccessTokenCheckerFunc func(ctx context.Context, userID uint32, accessToken string) bool

func (f AccessTokenCheckerFunc) Exists(ctx context.Context, userID uint32, accessToken string) bool {
	return f(ctx, userID, accessToken)
}

type options struct {
	log *log.Helper

	accessTokenChecker AccessTokenChecker // 访问令牌检查器

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

// WithAccessTokenCheckerFunc 设置访问令牌检查器函数
func WithAccessTokenCheckerFunc(fnc AccessTokenCheckerFunc) Option {
	return func(opts *options) {
		opts.accessTokenChecker = fnc
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
