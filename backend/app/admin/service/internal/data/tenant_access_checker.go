package data

import (
	"context"
	"time"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	entCrud "github.com/tx7do/go-crud/entgo"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/api"
	"go-wind-admin/app/admin/service/internal/data/ent/plan"
	"go-wind-admin/app/admin/service/internal/data/ent/planmodule"
	"go-wind-admin/app/admin/service/internal/data/ent/tenant"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	appViewer "go-wind-admin/pkg/entgo/viewer"
	"go-wind-admin/pkg/middleware/auth"
)

// TenantAccessCheckerImpl 是 TenantAccessChecker 接口的 data 层实现。
// 仅对租户用户（tenantId>0）调用（中间件已保证），平台管理员（tid=0）不会进入此检查。
type TenantAccessCheckerImpl struct {
	entClient *ent.Client
	log       *bLogger.Helper
}

func NewTenantAccessCheckerImpl(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) auth.TenantAccessChecker {
	return &TenantAccessCheckerImpl{
		entClient: entClient.Client(),
		log:       ctx.NewLoggerHelper("tenant-access-checker/admin-service"),
	}
}

// CheckTenantAccess 实现 auth.TenantAccessChecker。
// tenantId 由中间件保证 > 0。
func (c *TenantAccessCheckerImpl) CheckTenantAccess(ctx context.Context, tenantId uint32, path string, method string) error {
	sysCtx := appViewer.NewSystemViewerContext(ctx)

	// 1. 查租户状态与到期时间（WithPlan 预载套餐以读取 expiry_policy）
	t, err := c.entClient.Tenant.Query().
		Where(tenant.IDEQ(tenantId)).
		WithPlan().
		Only(sysCtx)
	if err != nil || t == nil {
		c.log.Errorf(ctx, "tenant access check: tenant %d not found: %v", tenantId, err)
		return adminV1.ErrorForbidden("access denied")
	}

	if t.Status != nil && *t.Status != tenant.StatusOn {
		return adminV1.ErrorForbidden("tenant is not active")
	}

	// 2. 到期只读判定（READONLY 策略 + 已到期）
	planId := uint32(0)
	var planExpiryPolicy plan.ExpiryPolicy
	if t.Edges.Plan != nil {
		planId = t.Edges.Plan.ID
		if t.Edges.Plan.ExpiryPolicy != nil {
			planExpiryPolicy = *t.Edges.Plan.ExpiryPolicy
		}
	}
	planExpired := t.ExpiredAt != nil && !t.ExpiredAt.IsZero() && t.ExpiredAt.Before(time.Now())

	if planExpired && planExpiryPolicy == plan.ExpiryPolicyReadonly {
		switch method {
		case "GET", "HEAD", "OPTIONS":
			// 只读放行，但仍需过白名单（见第 3 步）
		default:
			return adminV1.ErrorForbidden("tenant is read-only due to expiry")
		}
	}

	// 3. 套餐模块白名单
	bizModule := identityV1.Module_MODULE_UNSPECIFIED
	a, aerr := c.entClient.Api.Query().
		Where(api.PathEQ(path), api.MethodEQ(method)).
		Only(sysCtx)
	if aerr == nil && a != nil {
		if a.BusinessModule != nil {
			bizModule = mapApiBusinessModuleToProto(*a.BusinessModule)
		}
	} else {
		// 找不到对应 API 记录，无法判定模块。fail-closed 拒绝。
		c.log.Errorf(ctx, "tenant access check: api not found for %s %s", method, path)
		return adminV1.ErrorForbidden("access denied")
	}

	if bizModule == identityV1.Module_MODULE_UNSPECIFIED {
		// 未归类模块：不在任何白名单内，拒绝。
		return adminV1.ErrorForbidden("module not allowed")
	}

	if planId == 0 {
		// 租户未关联套餐，无白名单，拒绝所有业务模块。
		return adminV1.ErrorForbidden("no subscription plan")
	}

	allowed, lerr := c.isModuleAllowed(sysCtx, planId, bizModule)
	if lerr != nil {
		c.log.Errorf(ctx, "tenant access check: whitelist query failed: %v", lerr)
		return adminV1.ErrorForbidden("access denied")
	}
	if !allowed {
		return adminV1.ErrorForbidden("module not allowed")
	}

	return nil
}

// isModuleAllowed 查询套餐白名单是否包含指定模块。
func (c *TenantAccessCheckerImpl) isModuleAllowed(ctx context.Context, planId uint32, module identityV1.Module) (bool, error) {
	entModule := mapProtoModuleToEnt(module)
	if entModule == "" {
		return false, nil
	}
	cnt, err := c.entClient.PlanModule.Query().
		Where(
			planmodule.HasPlanWith(plan.IDEQ(planId)),
			planmodule.ModuleEQ(entModule),
		).
		Count(ctx)
	if err != nil {
		return false, err
	}
	return cnt > 0, nil
}

// mapProtoModuleToEnt 将 proto Module 值映射到 ent planmodule.Module 字符串。
// 返回空串表示 UNSPECIFIED。
func mapProtoModuleToEnt(m identityV1.Module) planmodule.Module {
	switch m {
	case identityV1.Module_DASHBOARD:
		return planmodule.ModuleDashboard
	case identityV1.Module_OPM:
		return planmodule.ModuleOpm
	case identityV1.Module_SYSTEM:
		return planmodule.ModuleSystem
	case identityV1.Module_DICT:
		return planmodule.ModuleDict
	case identityV1.Module_TENANT:
		return planmodule.ModuleTenant
	case identityV1.Module_PERMISSION:
		return planmodule.ModulePermission
	case identityV1.Module_LOG:
		return planmodule.ModuleLog
	case identityV1.Module_INTERNAL_MESSAGE:
		return planmodule.ModuleInternalMessage
	case identityV1.Module_FILE:
		return planmodule.ModuleFile
	case identityV1.Module_TASK:
		return planmodule.ModuleTask
	default:
		return ""
	}
}

// mapApiBusinessModuleToProto 将 ent api.BusinessModule 字符串枚举映射到 proto identityV1.Module。
// 返回 MODULE_UNSPECIFIED 表示未归类。
func mapApiBusinessModuleToProto(m api.BusinessModule) identityV1.Module {
	switch m {
	case api.BusinessModuleDashboard:
		return identityV1.Module_DASHBOARD
	case api.BusinessModuleOpm:
		return identityV1.Module_OPM
	case api.BusinessModuleSystem:
		return identityV1.Module_SYSTEM
	case api.BusinessModuleDict:
		return identityV1.Module_DICT
	case api.BusinessModuleTenant:
		return identityV1.Module_TENANT
	case api.BusinessModulePermission:
		return identityV1.Module_PERMISSION
	case api.BusinessModuleLog:
		return identityV1.Module_LOG
	case api.BusinessModuleInternalMessage:
		return identityV1.Module_INTERNAL_MESSAGE
	case api.BusinessModuleFile:
		return identityV1.Module_FILE
	case api.BusinessModuleTask:
		return identityV1.Module_TASK
	default:
		return identityV1.Module_MODULE_UNSPECIFIED
	}
}
