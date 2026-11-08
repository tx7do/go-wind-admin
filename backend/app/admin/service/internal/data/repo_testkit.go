// 跨包（service 层）测试装配：导出免 bootstrap.Context 的 repo 构造器，
// 字段初始化必须与生产 NewXxxRepo 逐字段一致，改生产构造器须同步此处。
//
// 说明：
//   - 各构造器与对应 *_repo.go 的生产构造器逐字段对齐（含 mapper/converter 初始化与
//     init() 调用），唯一差异是 log 一律 bLogger.NewHelper(bLogger.NopLogger())；
//     entClient 由调用方传入（测试场景为 enttest.NewEntClientForTest 的 SQLite 内存库）。
//   - 生产构造器中经由 *bootstrap.Context 注入的仅是日志助手，无其他隐藏依赖；
//     依赖 redis/minio 等外部件的构造器不在此导出。
//   - RoleRepo / OrgUnitRepo 的关联子 repo 由本文件内 testkitXxx 私有构造器组装，
//     组装关系对齐 cmd/server/wiring_ent.go 的仓储层装配小节。
//   - NewTenantUsageRepoForTest 的 authenticator 参数按生产签名保留：
//     TenantUsageRepo 仅在 CleanupTenantData/EnforceExpiryPolicies 中解引用它且有
//     nil 守卫，GetUsage 不触碰；测试不覆盖吊销令牌链路时可传 nil。
package data

import (
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/internalmessage"
	"go-wind-admin/app/admin/service/internal/data/ent/internalmessagerecipient"
	"go-wind-admin/app/admin/service/internal/data/ent/orgunit"
	"go-wind-admin/app/admin/service/internal/data/ent/permission"
	"go-wind-admin/app/admin/service/internal/data/ent/role"
	"go-wind-admin/app/admin/service/internal/data/ent/rolemetadata"
	"go-wind-admin/app/admin/service/internal/data/ent/rolepermission"
	"go-wind-admin/app/admin/service/internal/data/ent/tenant"
	"go-wind-admin/app/admin/service/internal/data/ent/userorgunit"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	internalMessageV1 "go-wind-admin/api/gen/go/internal_message/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

// NewTenantRepoForTest 逐字段复刻 tenant_repo.go 的 NewTenantRepo（log 换 NopLogger）。
func NewTenantRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *TenantRepo {
	repo := &TenantRepo{
		log:                  bLogger.NewHelper(bLogger.NopLogger()),
		entClient:            entClient,
		mapper:               mapper.NewCopierMapper[identityV1.Tenant, ent.Tenant](),
		statusConverter:      mapper.NewEnumTypeConverter[identityV1.Tenant_Status, tenant.Status](identityV1.Tenant_Status_name, identityV1.Tenant_Status_value),
		typeConverter:        mapper.NewEnumTypeConverter[identityV1.Tenant_Type, tenant.Type](identityV1.Tenant_Type_name, identityV1.Tenant_Type_value),
		auditStatusConverter: mapper.NewEnumTypeConverter[identityV1.Tenant_AuditStatus, tenant.AuditStatus](identityV1.Tenant_AuditStatus_name, identityV1.Tenant_AuditStatus_value),
	}

	repo.init()

	return repo
}

// NewTenantUsageRepoForTest 逐字段复刻 tenant_usage_repo.go 的 NewTenantUsageRepo
// （log 换 NopLogger；authenticator 语义见文件头说明）。
func NewTenantUsageRepoForTest(entClient *entCrud.EntClient[*ent.Client], authenticator *Authenticator) *TenantUsageRepo {
	return &TenantUsageRepo{
		entClient:     entClient,
		authenticator: authenticator,
		log:           bLogger.NewHelper(bLogger.NopLogger()),
	}
}

// NewRoleRepoForTest 逐字段复刻 role_repo.go 的 NewRoleRepo（log 换 NopLogger；
// 关联子 repo 由 testkit 私有构造器组装，对齐 wiring_ent.go 的装配关系）。
func NewRoleRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *RoleRepo {
	repo := &RoleRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[permissionV1.Role, ent.Role](),
		statusConverter: mapper.NewEnumTypeConverter[permissionV1.Role_Status, role.Status](
			permissionV1.Role_Status_name,
			permissionV1.Role_Status_value,
		),
		typeConverter: mapper.NewEnumTypeConverter[permissionV1.Role_Type, role.Type](
			permissionV1.Role_Type_name,
			permissionV1.Role_Type_value,
		),
		dataScopeConverter: mapper.NewEnumTypeConverter[identityV1.DataScope, role.DataScope](
			identityV1.DataScope_name,
			identityV1.DataScope_value,
		),
		permissionRepo:          testkitPermissionRepo(entClient),
		rolePermissionRepo:      testkitRolePermissionRepo(entClient),
		roleOrgUnitRepo:         testkitRoleOrgUnitRepo(entClient),
		roleMetadataRepo:        testkitRoleMetadataRepo(entClient),
		roleFieldPermissionRepo: testkitRoleFieldPermissionRepo(entClient),
	}

	repo.init()

	return repo
}

// NewOrgUnitRepoForTest 逐字段复刻 org_unit_repo.go 的 NewOrgUnitRepo
// （log 换 NopLogger；userOrgUnitRepo / roleOrgUnitRepo 由 testkit 私有构造器组装）。
func NewOrgUnitRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *OrgUnitRepo {
	repo := &OrgUnitRepo{
		log:             bLogger.NewHelper(bLogger.NopLogger()),
		entClient:       entClient,
		userOrgUnitRepo: testkitUserOrgUnitRepo(entClient),
		roleOrgUnitRepo: testkitRoleOrgUnitRepo(entClient),
		mapper:          mapper.NewCopierMapper[identityV1.OrgUnit, ent.OrgUnit](),
		typeConverter:   mapper.NewEnumTypeConverter[identityV1.OrgUnit_Type, orgunit.Type](identityV1.OrgUnit_Type_name, identityV1.OrgUnit_Type_value),
		statusConverter: mapper.NewEnumTypeConverter[identityV1.OrgUnit_Status, orgunit.Status](identityV1.OrgUnit_Status_name, identityV1.OrgUnit_Status_value),
	}

	repo.init()

	return repo
}

// NewNotificationPreferenceRepoForTest 逐字段复刻 notification_preference_repo.go 的
// NewNotificationPreferenceRepo（log 换 NopLogger）。
func NewNotificationPreferenceRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *NotificationPreferenceRepo {
	return &NotificationPreferenceRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
	}
}

// NewInternalMessageRepoForTest 逐字段复刻 internal_message_repo.go 的
// NewInternalMessageRepo（log 换 NopLogger）。
func NewInternalMessageRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *InternalMessageRepo {
	repo := &InternalMessageRepo{
		log:             bLogger.NewHelper(bLogger.NopLogger()),
		entClient:       entClient,
		mapper:          mapper.NewCopierMapper[internalMessageV1.InternalMessage, ent.InternalMessage](),
		statusConverter: mapper.NewEnumTypeConverter[internalMessageV1.InternalMessage_Status, internalmessage.Status](internalMessageV1.InternalMessage_Status_name, internalMessageV1.InternalMessage_Status_value),
		typeConverter:   mapper.NewEnumTypeConverter[internalMessageV1.InternalMessage_Type, internalmessage.Type](internalMessageV1.InternalMessage_Type_name, internalMessageV1.InternalMessage_Type_value),
	}

	repo.init()

	return repo
}

// NewInternalMessageCategoryRepoForTest 逐字段复刻 internal_message_category_repo.go 的
// NewInternalMessageCategoryRepo（log 换 NopLogger）。
func NewInternalMessageCategoryRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *InternalMessageCategoryRepo {
	repo := &InternalMessageCategoryRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[internalMessageV1.InternalMessageCategory, ent.InternalMessageCategory](),
	}

	repo.init()

	return repo
}

// NewInternalMessageRecipientRepoForTest 逐字段复刻 internal_message_recipient_repo.go 的
// NewInternalMessageRecipientRepo（log 换 NopLogger）。
func NewInternalMessageRecipientRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *InternalMessageRecipientRepo {
	repo := &InternalMessageRecipientRepo{
		log:             bLogger.NewHelper(bLogger.NopLogger()),
		entClient:       entClient,
		mapper:          mapper.NewCopierMapper[internalMessageV1.InternalMessageRecipient, ent.InternalMessageRecipient](),
		statusConverter: mapper.NewEnumTypeConverter[internalMessageV1.InternalMessageRecipient_Status, internalmessagerecipient.Status](internalMessageV1.InternalMessageRecipient_Status_name, internalMessageV1.InternalMessageRecipient_Status_value),
	}

	repo.init()

	return repo
}

// ---------------------------------------------------------------------------
// testkit 私有子 repo 构造器（不导出）：逐字段复刻各自生产构造器，
// 仅供本文件的 ForTest 导出构造器组装使用。
// ---------------------------------------------------------------------------

// testkitRolePermissionRepo 复刻 role_permission_repo.go 的 NewRolePermissionRepo。
func testkitRolePermissionRepo(entClient *entCrud.EntClient[*ent.Client]) *RolePermissionRepo {
	repo := &RolePermissionRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[permissionV1.RolePermission, ent.RolePermission](),
		statusConverter: mapper.NewEnumTypeConverter[permissionV1.RolePermission_Status, rolepermission.Status](
			permissionV1.RolePermission_Status_name,
			permissionV1.RolePermission_Status_value,
		),
		effectConverter: mapper.NewEnumTypeConverter[permissionV1.RolePermission_EffectiveStatus, rolepermission.Effect](
			permissionV1.RolePermission_EffectiveStatus_name,
			permissionV1.RolePermission_EffectiveStatus_value,
		),
	}

	repo.init()

	return repo
}

// testkitRoleOrgUnitRepo 复刻 role_org_unit_repo.go 的 NewRoleOrgUnitRepo。
func testkitRoleOrgUnitRepo(entClient *entCrud.EntClient[*ent.Client]) *RoleOrgUnitRepo {
	repo := &RoleOrgUnitRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
	}

	return repo
}

// testkitPermissionApiRepo 复刻 permission_api_repo.go 的 NewPermissionApiRepo。
func testkitPermissionApiRepo(entClient *entCrud.EntClient[*ent.Client]) *PermissionApiRepo {
	return &PermissionApiRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
	}
}

// testkitPermissionMenuRepo 复刻 permission_menu_repo.go 的 NewPermissionMenuRepo。
func testkitPermissionMenuRepo(entClient *entCrud.EntClient[*ent.Client]) *PermissionMenuRepo {
	return &PermissionMenuRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
	}
}

// testkitPermissionRepo 复刻 permission_repo.go 的 NewPermissionRepo。
func testkitPermissionRepo(entClient *entCrud.EntClient[*ent.Client]) *PermissionRepo {
	repo := &PermissionRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[permissionV1.Permission, ent.Permission](),
		statusConverter: mapper.NewEnumTypeConverter[permissionV1.Permission_Status, permission.Status](
			permissionV1.Permission_Status_name,
			permissionV1.Permission_Status_value,
		),
		permissionApiRepo:  testkitPermissionApiRepo(entClient),
		permissionMenuRepo: testkitPermissionMenuRepo(entClient),
	}

	repo.init()

	return repo
}

// testkitRoleMetadataRepo 复刻 role_metadata_repo.go 的 NewRoleMetadataRepo。
func testkitRoleMetadataRepo(entClient *entCrud.EntClient[*ent.Client]) *RoleMetadataRepo {
	repo := &RoleMetadataRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[permissionV1.RoleMetadata, ent.RoleMetadata](),
		syncPolicyConverter: mapper.NewEnumTypeConverter[permissionV1.RoleMetadata_SyncPolicy, rolemetadata.SyncPolicy](
			permissionV1.RoleMetadata_SyncPolicy_name,
			permissionV1.RoleMetadata_SyncPolicy_value,
		),
		scopeConverter: mapper.NewEnumTypeConverter[permissionV1.RoleMetadata_Scope, rolemetadata.Scope](
			permissionV1.RoleMetadata_Scope_name,
			permissionV1.RoleMetadata_Scope_value,
		),
	}

	repo.init()

	return repo
}

// testkitRoleFieldPermissionRepo 复刻 role_field_permission_repo.go 的 NewRoleFieldPermissionRepo。
func testkitRoleFieldPermissionRepo(entClient *entCrud.EntClient[*ent.Client]) *RoleFieldPermissionRepo {
	repo := &RoleFieldPermissionRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
	}

	return repo
}

// testkitUserOrgUnitRepo 复刻 user_org_unit_repo.go 的 NewUserOrgUnitRepo。
func testkitUserOrgUnitRepo(entClient *entCrud.EntClient[*ent.Client]) *UserOrgUnitRepo {
	return &UserOrgUnitRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		statusConverter: mapper.NewEnumTypeConverter[identityV1.UserOrgUnit_Status, userorgunit.Status](
			identityV1.UserOrgUnit_Status_name,
			identityV1.UserOrgUnit_Status_value,
		),
	}
}
