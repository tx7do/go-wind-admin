package logging

import (
	"context"
	"crypto/ecdsa"

	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
)

type WriteApiLogFunc func(ctx context.Context, data *auditV1.ApiAuditLog) error
type WriteLoginLogFunc func(ctx context.Context, data *auditV1.LoginAuditLog) error

type options struct {
	writeApiLogFunc   WriteApiLogFunc   // 写入API审计日志函数
	writeLoginLogFunc WriteLoginLogFunc // 写入登录审计日志函数

	loginOperation  string // 登录操作名称
	logoutOperation string // 登出操作名称

	ecPrivateKey *ecdsa.PrivateKey // 私钥（加密存储）
	ecPublicKey  *ecdsa.PublicKey  // 公钥（可公开）
}

type Option func(*options)

func WithWriteApiLogFunc(fnc WriteApiLogFunc) Option {
	return func(opts *options) {
		opts.writeApiLogFunc = fnc
	}
}

func WithWriteLoginLogFunc(fnc WriteLoginLogFunc) Option {
	return func(opts *options) {
		opts.writeLoginLogFunc = fnc
	}
}

func WithLoginOperation(operation string) Option {
	return func(opts *options) {
		opts.loginOperation = operation
	}
}

func WithLogoutOperation(operation string) Option {
	return func(opts *options) {
		opts.logoutOperation = operation
	}
}

func WithECPrivateKey(key *ecdsa.PrivateKey) Option {
	return func(opts *options) {
		opts.ecPrivateKey = key
	}
}

func WithECPublicKey(key *ecdsa.PublicKey) Option {
	return func(opts *options) {
		opts.ecPublicKey = key
	}
}
