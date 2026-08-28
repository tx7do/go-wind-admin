package logging

import (
	"context"
	"crypto/ecdsa"

	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
)

type WriteApiLogFunc func(ctx context.Context, data *auditV1.ApiAuditLog) error
type WriteLoginLogFunc func(ctx context.Context, data *auditV1.LoginAuditLog) error
type WriteOperationAuditLogFunc func(ctx context.Context, data *auditV1.OperationAuditLog) error
type WritePermissionAuditLogFunc func(ctx context.Context, data *auditV1.PermissionAuditLog) error
type WriteDataAccessAuditLogFunc func(ctx context.Context, data *auditV1.DataAccessAuditLog) error

type options struct {
	writeApiLogFunc                WriteApiLogFunc                // 写入API审计日志函数
	writeLoginLogFunc              WriteLoginLogFunc              // 写入登录审计日志函数
	writeOperationAuditLogFunc    WriteOperationAuditLogFunc    // 写入操作审计日志函数
	writePermissionAuditLogFunc   WritePermissionAuditLogFunc   // 写入权限审计日志函数
	writeDataAccessAuditLogFunc   WriteDataAccessAuditLogFunc   // 写入数据访问审计日志函数

	loginOperations  []string // 登录操作名称集合（登录、MFA 验证等）
	logoutOperation string   // 登出操作名称

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

func WithWriteOperationAuditLogFunc(fnc WriteOperationAuditLogFunc) Option {
	return func(opts *options) {
		opts.writeOperationAuditLogFunc = fnc
	}
}

func WithWritePermissionAuditLogFunc(fnc WritePermissionAuditLogFunc) Option {
	return func(opts *options) {
		opts.writePermissionAuditLogFunc = fnc
	}
}

func WithWriteDataAccessAuditLogFunc(fnc WriteDataAccessAuditLogFunc) Option {
	return func(opts *options) {
		opts.writeDataAccessAuditLogFunc = fnc
	}
}

func WithLoginOperation(operation ...string) Option {
	return func(opts *options) {
		opts.loginOperations = operation
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
