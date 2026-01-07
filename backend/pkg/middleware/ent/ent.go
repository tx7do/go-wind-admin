package ent

import (
	"context"

	"github.com/go-kratos/kratos/v2/middleware"

	"go-wind-admin/pkg/entgo/viewer"
	"go-wind-admin/pkg/metadata"
)

// Server .
func Server() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			data := metadata.FromOperatorMetadata(ctx)
			if data == nil {
				return handler(ctx, req)
			}

			userViewer := viewer.NewUserViewer(
				data.GetUserID(),
				data.GetTenantID(),
				data.GetOrgUnitID(),
				data.GetDataScope(),
			)
			ctx = viewer.NewContext(ctx, userViewer)

			return handler(ctx, req)
		}
	}
}
