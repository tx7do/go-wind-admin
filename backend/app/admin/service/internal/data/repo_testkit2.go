// 跨包（service 层）测试装配：导出免 bootstrap.Context 的 repo 构造器，
// 字段初始化必须与生产 NewXxxRepo 逐字段一致，改生产构造器须同步此处。
//
// 说明：
//   - 本文件覆盖 repo_testkit.go 之外批次的 repo（plan / plan_module / plan_quota /
//     language / dict_type / dict_entry / position / api / permission_group / menu）。
//     各构造器与对应 *_repo.go 的生产构造器逐字段对齐（含 mapper/converter 初始化
//     与 init() 调用），唯一差异是 log 一律 bLogger.NewHelper(bLogger.NopLogger())；
//     entClient 由调用方传入（测试场景为 enttest.NewEntClientForTest 的 SQLite 内存库）。
//   - 生产构造器中经由 *bootstrap.Context 注入的仅是日志助手，无其他隐藏依赖；
//     依赖 redis/minio 等外部件的构造器不在此导出。
//   - DictEntryRepo 的关联子 repo（DictEntryI18nRepo）由本文件内私有构造器组装，
//     装配关系对齐生产 NewDictEntryRepo。
package data

import (
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/admin/service/internal/data/ent"
	entApi "go-wind-admin/app/admin/service/internal/data/ent/api"
	entMenu "go-wind-admin/app/admin/service/internal/data/ent/menu"
	entPermissionGroup "go-wind-admin/app/admin/service/internal/data/ent/permissiongroup"
	entPlan "go-wind-admin/app/admin/service/internal/data/ent/plan"
	entPlanModule "go-wind-admin/app/admin/service/internal/data/ent/planmodule"
	entPlanQuota "go-wind-admin/app/admin/service/internal/data/ent/planquota"
	entPosition "go-wind-admin/app/admin/service/internal/data/ent/position"

	dictV1 "go-wind-admin/api/gen/go/dict/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

// NewPlanRepoForTest 与生产 NewPlanRepo 逐字段一致（log 换 NopLogger），并调用 init()。
func NewPlanRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *PlanRepo {
	repo := &PlanRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[identityV1.Plan, ent.Plan](),
		versionConverter: mapper.NewEnumTypeConverter[identityV1.Plan_Version, entPlan.Version](
			identityV1.Plan_Version_name, identityV1.Plan_Version_value,
		),
		expiryPolicyConv: mapper.NewEnumTypeConverter[identityV1.Plan_ExpiryPolicy, entPlan.ExpiryPolicy](
			identityV1.Plan_ExpiryPolicy_name, identityV1.Plan_ExpiryPolicy_value,
		),
	}

	repo.init()

	return repo
}

// NewPlanModuleRepoForTest 与生产 NewPlanModuleRepo 逐字段一致（log 换 NopLogger），并调用 init()。
func NewPlanModuleRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *PlanModuleRepo {
	repo := &PlanModuleRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[identityV1.PlanModule, ent.PlanModule](),
		moduleConv: mapper.NewEnumTypeConverter[identityV1.Module, entPlanModule.Module](
			identityV1.Module_name, identityV1.Module_value,
		),
	}

	repo.init()

	return repo
}

// NewPlanQuotaRepoForTest 与生产 NewPlanQuotaRepo 逐字段一致（log 换 NopLogger），并调用 init()。
func NewPlanQuotaRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *PlanQuotaRepo {
	repo := &PlanQuotaRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[identityV1.PlanQuota, ent.PlanQuota](),
		quotaTypeConv: mapper.NewEnumTypeConverter[identityV1.PlanQuota_QuotaType, entPlanQuota.QuotaType](
			identityV1.PlanQuota_QuotaType_name, identityV1.PlanQuota_QuotaType_value,
		),
	}

	repo.init()

	return repo
}

// NewPositionRepoForTest 与生产 NewPositionRepo 逐字段一致（log 换 NopLogger），并调用 init()。
func NewPositionRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *PositionRepo {
	repo := &PositionRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[identityV1.Position, ent.Position](),
		statusConverter: mapper.NewEnumTypeConverter[identityV1.Position_Status, entPosition.Status](
			identityV1.Position_Status_name, identityV1.Position_Status_value,
		),
		typeConverter: mapper.NewEnumTypeConverter[identityV1.Position_Type, entPosition.Type](
			identityV1.Position_Type_name, identityV1.Position_Type_value,
		),
	}

	repo.init()

	return repo
}

// NewApiRepoForTest 与生产 NewApiRepo 逐字段一致（log 换 NopLogger），并调用 init()。
// 绑定子 repo（SyncMerge/级联清理依赖）由本文件内私有构造器组装，装配关系对齐生产。
func NewApiRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *ApiRepo {
	repo := &ApiRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[permissionV1.Api, ent.Api](),
		statusConverter: mapper.NewEnumTypeConverter[permissionV1.Api_Status, entApi.Status](
			permissionV1.Api_Status_name, permissionV1.Api_Status_value,
		),
		scopeConverter: mapper.NewEnumTypeConverter[permissionV1.Api_Scope, entApi.Scope](
			permissionV1.Api_Scope_name, permissionV1.Api_Scope_value,
		),
		businessModuleConverter: mapper.NewEnumTypeConverter[identityV1.Module, entApi.BusinessModule](
			identityV1.Module_name, identityV1.Module_value,
		),
		permissionApi: testkitPermissionApiRepo(entClient),
	}

	repo.init()

	return repo
}

// NewPermissionGroupRepoForTest 与生产 NewPermissionGroupRepo 逐字段一致（log 换 NopLogger），并调用 init()。
func NewPermissionGroupRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *PermissionGroupRepo {
	repo := &PermissionGroupRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[permissionV1.PermissionGroup, ent.PermissionGroup](),
		statusConverter: mapper.NewEnumTypeConverter[permissionV1.PermissionGroup_Status, entPermissionGroup.Status](
			permissionV1.PermissionGroup_Status_name, permissionV1.PermissionGroup_Status_value,
		),
	}

	repo.init()

	return repo
}

// NewMenuRepoForTest 与生产 NewMenuRepo 逐字段一致（log 换 NopLogger），并调用 init()。
func NewMenuRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *MenuRepo {
	repo := &MenuRepo{
		log:             bLogger.NewHelper(bLogger.NopLogger()),
		entClient:       entClient,
		mapper:          mapper.NewCopierMapper[permissionV1.Menu, ent.Menu](),
		statusConverter: mapper.NewEnumTypeConverter[permissionV1.Menu_Status, entMenu.Status](permissionV1.Menu_Status_name, permissionV1.Menu_Status_value),
		typeConverter:   mapper.NewEnumTypeConverter[permissionV1.Menu_Type, entMenu.Type](permissionV1.Menu_Type_name, permissionV1.Menu_Type_value),
		moduleConverter: mapper.NewEnumTypeConverter[identityV1.Module, entMenu.Module](identityV1.Module_name, identityV1.Module_value),
	}

	repo.init()

	return repo
}

// NewLanguageRepoForTest 与生产 NewLanguageRepo 逐字段一致（log 换 NopLogger），并调用 init()。
func NewLanguageRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *LanguageRepo {
	repo := &LanguageRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[dictV1.Language, ent.Language](),
	}

	repo.init()

	return repo
}

// NewDictTypeRepoForTest 与生产 NewDictTypeRepo 逐字段一致（log 换 NopLogger），并调用 init()。
func NewDictTypeRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *DictTypeRepo {
	repo := &DictTypeRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[dictV1.DictType, ent.DictType](),
	}

	repo.init()

	return repo
}

// testkitDictEntryI18nRepo 私有构造器：与生产 NewDictEntryI18nRepo 逐字段一致
// （log 换 NopLogger），仅供本文件 NewDictEntryRepoForTest 组装关联子 repo 使用。
func testkitDictEntryI18nRepo(entClient *entCrud.EntClient[*ent.Client]) *DictEntryI18nRepo {
	repo := &DictEntryI18nRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[dictV1.DictEntryI18N, ent.DictEntryI18n](),
	}

	repo.init()

	return repo
}

// NewDictEntryRepoForTest 与生产 NewDictEntryRepo 逐字段一致（log 换 NopLogger，
// 关联 i18n 子 repo 由 testkitDictEntryI18nRepo 组装），并调用 init()。
func NewDictEntryRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *DictEntryRepo {
	repo := &DictEntryRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[dictV1.DictEntry, ent.DictEntry](),
		i18n:      testkitDictEntryI18nRepo(entClient),
	}

	repo.init()

	return repo
}
