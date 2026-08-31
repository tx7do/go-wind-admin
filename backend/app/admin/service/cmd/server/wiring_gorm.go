//go:build gorm_backend
// +build gorm_backend

package main

import (
	gormCrud "github.com/tx7do/go-crud/gorm"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	gormData "go-wind-admin/app/admin/service/internal/data/gorm"
)

// GORM 后端的装配骨架(Phase 4 占位,go build -tags gorm_backend)。
//
// 与 wiring_ent.go 平行:基础设施/认证与鉴权/服务层/传输层四个小节最终同构,
// 仅仓储层互斥(ent *data.XxxRepo ↔ gorm *gorm.XxxRepo)。
//
// 当前阻塞清单(按解除顺序,全部解除后照 wiring_ent.go 补齐 initApp 即可切换):
//
//  1. [已完成] 上游 logger 接口错位与 gorm 包滞后:kratos-bootstrap/database/gorm
//     v0.1.6 与 go-wind/log 接口不匹配;gorm 包 40 个仓储的 log 字段仍为旧版
//     kratos Helper。已在 data/gorm 内修复(自建 client + helperLogger 适配 +
//     全量 logger 迁移),gorm 包现已可编译。
//  2. [未动] 仓储缺口:data/gorm 缺 UserCredentialRepo / UserMfaFactorRepo /
//     AuditLogArchiveRepo 三个仓储(涉及密码加密器与归档任务依赖)。
//  3. [未动] 服务层类型耦合(最大缺口):全部 service 构造器形参是 ent 具体类型
//     (如 *data.RoleRepo),gorm 仓储无法传入。需先做 repo 接口抽取——把各
//     Repo 接口提到中立包,ent/gorm 双实现分别满足——这正是 orm 切换 Phase 4
//     的主体工作。完成后本文件的仓储小节直接喂给与 wiring_ent.go 相同的服务层。
//  4. [未动] 认证与鉴权链:data.NewAuthorizerProvider 形参同为 ent 类型,
//     与第 3 项一同解。
//
// 基础设施(Redis/MinIO/令牌缓存/限流/验证码等)与 ORM 无关,切换时从
// wiring_ent.go 原样复制即可。

// gormRepos GORM 后端的仓储集合(与 wiring_ent.go 仓储小节一一对应)。
type gormRepos struct {
	// 身份与账号
	userRoleRepo     *gormData.UserRoleRepo
	userOrgUnitRepo  *gormData.UserOrgUnitRepo
	userPositionRepo *gormData.UserPositionRepo
	membershipRepo   *gormData.MembershipRepo
	userRepo         *gormData.UserRepo
	// 组织架构与租户
	orgUnitRepo     *gormData.OrgUnitRepo
	positionRepo    *gormData.PositionRepo
	tenantRepo      *gormData.TenantRepo
	tenantUsageRepo *gormData.TenantUsageRepo
	// RBAC
	permissionApiRepo   *gormData.PermissionApiRepo
	permissionMenuRepo  *gormData.PermissionMenuRepo
	rolePermissionRepo  *gormData.RolePermissionRepo
	permissionRepo      *gormData.PermissionRepo
	roleRepo            *gormData.RoleRepo
	permissionGroupRepo *gormData.PermissionGroupRepo
	apiRepo             *gormData.ApiRepo
	menuRepo            *gormData.MenuRepo
	// 审计日志
	apiAuditLogRepo         *gormData.ApiAuditLogRepo
	loginAuditLogRepo       *gormData.LoginAuditLogRepo
	operationAuditLogRepo   *gormData.OperationAuditLogRepo
	permissionAuditLogRepo  *gormData.PermissionAuditLogRepo
	dataAccessAuditLogRepo  *gormData.DataAccessAuditLogRepo
	policyEvaluationLogRepo *gormData.PolicyEvaluationLogRepo
	// 字典与多语言
	dictTypeRepo      *gormData.DictTypeRepo
	dictEntryI18nRepo *gormData.DictEntryI18nRepo
	dictEntryRepo     *gormData.DictEntryRepo
	languageRepo      *gormData.LanguageRepo
	// 套餐与计费
	planRepo       *gormData.PlanRepo
	planQuotaRepo  *gormData.PlanQuotaRepo
	planModuleRepo *gormData.PlanModuleRepo
	// 任务 / 文件 / 运维观测
	taskRepo      *gormData.TaskRepo
	backupRepo    *gormData.BackupRepo
	fileRepo      *gormData.FileRepo
	dashboardRepo *gormData.DashboardRepo
	// 站内信
	internalMessageRepo          *gormData.InternalMessageRepo
	internalMessageCategoryRepo  *gormData.InternalMessageCategoryRepo
	internalMessageRecipientRepo *gormData.InternalMessageRecipientRepo
}

// newGormRepos 构造 GORM 后端全部仓储。签名全部为 (ctx, client) 平铺形态,
// 不像 ent 侧存在子仓储聚合;成员仓储独立可用,聚合仓储按需自行组合。
// TODO(Phase 4): UserCredentialRepo / UserMfaFactorRepo / AuditLogArchiveRepo
// 另:BackupRepo / TenantUsageRepo / DashboardRepo 三个构造器不接 client,为 IsConfigured()=false 的空壳实现,真实实现同样待 Phase 4。
// 三个仓储 gorm 侧尚无实现(见文件头阻塞清单第 2 项)。
func newGormRepos(ctx *bootstrap.Context) (*gormRepos, *gormCrud.Client, error) {
	client, err := gormData.NewGormClient(ctx)
	if err != nil {
		return nil, nil, err
	}

	r := &gormRepos{
		// 身份与账号
		userRoleRepo:     gormData.NewUserRoleRepo(ctx, client),
		userOrgUnitRepo:  gormData.NewUserOrgUnitRepo(ctx, client),
		userPositionRepo: gormData.NewUserPositionRepo(ctx, client),
		membershipRepo:   gormData.NewMembershipRepo(ctx, client),
		userRepo:         gormData.NewUserRepo(ctx, client),
		// 组织架构与租户
		orgUnitRepo:     gormData.NewOrgUnitRepo(ctx, client),
		positionRepo:    gormData.NewPositionRepo(ctx, client),
		tenantRepo:      gormData.NewTenantRepo(ctx, client),
		tenantUsageRepo: gormData.NewTenantUsageRepo(ctx),
		// RBAC
		permissionApiRepo:   gormData.NewPermissionApiRepo(ctx, client),
		permissionMenuRepo:  gormData.NewPermissionMenuRepo(ctx, client),
		rolePermissionRepo:  gormData.NewRolePermissionRepo(ctx, client),
		permissionRepo:      gormData.NewPermissionRepo(ctx, client),
		roleRepo:            gormData.NewRoleRepo(ctx, client),
		permissionGroupRepo: gormData.NewPermissionGroupRepo(ctx, client),
		apiRepo:             gormData.NewApiRepo(ctx, client),
		menuRepo:            gormData.NewMenuRepo(ctx, client),
		// 审计日志
		apiAuditLogRepo:         gormData.NewApiAuditLogRepo(ctx, client),
		loginAuditLogRepo:       gormData.NewLoginAuditLogRepo(ctx, client),
		operationAuditLogRepo:   gormData.NewOperationAuditLogRepo(ctx, client),
		permissionAuditLogRepo:  gormData.NewPermissionAuditLogRepo(ctx, client),
		dataAccessAuditLogRepo:  gormData.NewDataAccessAuditLogRepo(ctx, client),
		policyEvaluationLogRepo: gormData.NewPolicyEvaluationLogRepo(ctx, client),
		// 字典与多语言
		dictTypeRepo:      gormData.NewDictTypeRepo(ctx, client),
		dictEntryI18nRepo: gormData.NewDictEntryI18nRepo(ctx, client),
		dictEntryRepo:     gormData.NewDictEntryRepo(ctx, client),
		languageRepo:      gormData.NewLanguageRepo(ctx, client),
		// 套餐与计费
		planRepo:       gormData.NewPlanRepo(ctx, client),
		planQuotaRepo:  gormData.NewPlanQuotaRepo(ctx, client),
		planModuleRepo: gormData.NewPlanModuleRepo(ctx, client),
		// 任务 / 文件 / 运维观测
		taskRepo:      gormData.NewTaskRepo(ctx, client),
		backupRepo:    gormData.NewBackupRepo(ctx),
		fileRepo:      gormData.NewFileRepo(ctx, client),
		dashboardRepo: gormData.NewDashboardRepo(ctx),
		// 站内信
		internalMessageRepo:          gormData.NewInternalMessageRepo(ctx, client),
		internalMessageCategoryRepo:  gormData.NewInternalMessageCategoryRepo(ctx, client),
		internalMessageRecipientRepo: gormData.NewInternalMessageRecipientRepo(ctx, client),
	}
	return r, client, nil
}
