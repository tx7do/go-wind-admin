package logging

import (
	"context"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
)

type WriteApiLogFunc func(ctx context.Context, data *adminV1.ApiAuditLog) error
type WriteLoginLogFunc func(ctx context.Context, data *adminV1.LoginAuditLog) error

type options struct {
	writeApiLogFunc   WriteApiLogFunc
	writeLoginLogFunc WriteLoginLogFunc
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
