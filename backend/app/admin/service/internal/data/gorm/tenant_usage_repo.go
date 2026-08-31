//go:build gorm_backend
// +build gorm_backend

// Package gorm 中的仓储是 ent 仓储的平行 gorm 镜像，作为"ent 为主力、gorm 为备选"脚手架的完整代码。
// 这些仓储仅由 cmd/server/wiring_gorm.go(gorm_backend 构建,ORM 切换 Phase 4 占位)装配,服务层尚未接入。
package gorm

import (
	"context"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
)

// TenantUsageRepo gorm 脚手架桩：ent 侧跨 4 表聚合（Tenant→Plan→Quotas edge 链 + User/File/ApiAuditLog
// 计数）与 29 表事务清理 + token 吊销副作用，无单一 gorm model，go-crud/gorm 无对应原语。
// 见 data/tenant_usage_repo.go。
type TenantUsageRepo struct {
	log *bLogger.Helper
}

func NewTenantUsageRepo(ctx *bootstrap.Context) *TenantUsageRepo {
	return &TenantUsageRepo{
		log: ctx.NewLoggerHelper("tenant-usage/gorm-repo/admin-service"),
	}
}

func (r *TenantUsageRepo) GetUsage(ctx context.Context, tenantId uint32) (*identityV1.TenantUsage, error) {
	return nil, identityV1.ErrorInternalServerError("gorm scaffold: GetUsage not implemented — ent 4-table aggregation + Plan→Quotas edge chain has no go-crud/gorm primitive; see data/tenant_usage_repo.go")
}

func (r *TenantUsageRepo) CleanupTenantData(ctx context.Context, tenantId uint32) error {
	return identityV1.ErrorInternalServerError("gorm scaffold: CleanupTenantData not implemented — ent 29-table transactional delete + token revocation side-effect has no go-crud/gorm primitive; see data/tenant_usage_repo.go")
}

func (r *TenantUsageRepo) EnforceExpiryPolicies(ctx context.Context) (int, error) {
	return 0, identityV1.ErrorInternalServerError("gorm scaffold: EnforceExpiryPolicies not implemented — ent cross-table expiry scan + token revocation side-effect has no go-crud/gorm primitive; see data/tenant_usage_repo.go")
}
