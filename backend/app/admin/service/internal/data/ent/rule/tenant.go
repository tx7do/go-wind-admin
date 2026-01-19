package rule

import (
	"context"

	"github.com/tx7do/go-crud/entgo/rule"

	"go-wind-admin/app/admin/service/internal/data/ent/privacy"
)

func TenantQueryPolicy() privacy.QueryPolicy {
	return privacy.QueryPolicy{
		privacy.FilterFunc(func(ctx context.Context, f privacy.Filter) error {
			return rule.TenantFilterRule(ctx, f)
		}), // 物理租户隔离
		//privacy.FilterFunc(func(ctx context.Context, f privacy.Filter) error {
		//	return rule.PermissionRule(ctx, f)
		//}), // 业务权限隔离
	}
}
