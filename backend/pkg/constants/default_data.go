package constants

import (
	"time"

	"github.com/tx7do/go-utils/timeutil"
	"github.com/tx7do/go-utils/trans"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	dictV1 "go-wind-admin/api/gen/go/dict/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
	resourceV1 "go-wind-admin/api/gen/go/resource/service/v1"
)

const (
	// DefaultAdminUserName 系统初始化默认管理员用户名
	DefaultAdminUserName = "admin"
	// DefaultAdminPassword 系统初始化默认管理员用户密码
	DefaultAdminPassword = "admin"

	// DefaultUserPassword 系统初始化默认普通用户密码
	DefaultUserPassword = "12345678"

	// PlatformTenantID 平台管理员租户ID
	PlatformTenantID = uint32(0)
)

// DefaultPermissionGroups 系统初始化默认权限组数据
var DefaultPermissionGroups = []*permissionV1.PermissionGroup{
	{
		//Id:        trans.Ptr(uint32(1)),
		Name:      trans.Ptr("系统管理"),
		Path:      trans.Ptr("/"),
		Module:    trans.Ptr(SystemPermissionModule),
		SortOrder: trans.Ptr(uint32(1)),
		Status:    trans.Ptr(permissionV1.PermissionGroup_ON),
	},
	{
		//Id:        trans.Ptr(uint32(2)),
		ParentId:  trans.Ptr(uint32(1)),
		Name:      trans.Ptr("系统权限"),
		Path:      trans.Ptr("/1/2/"),
		Module:    trans.Ptr(SystemPermissionModule),
		SortOrder: trans.Ptr(uint32(1)),
		Status:    trans.Ptr(permissionV1.PermissionGroup_ON),
	},
	{
		//Id:        trans.Ptr(uint32(3)),
		ParentId:  trans.Ptr(uint32(1)),
		Name:      trans.Ptr("租户管理"),
		Path:      trans.Ptr("/1/3/"),
		Module:    trans.Ptr(SystemPermissionModule),
		SortOrder: trans.Ptr(uint32(2)),
		Status:    trans.Ptr(permissionV1.PermissionGroup_ON),
	},
	{
		//Id:        trans.Ptr(uint32(4)),
		ParentId:  trans.Ptr(uint32(1)),
		Name:      trans.Ptr("审计管理"),
		Path:      trans.Ptr("/1/4/"),
		Module:    trans.Ptr(SystemPermissionModule),
		SortOrder: trans.Ptr(uint32(3)),
		Status:    trans.Ptr(permissionV1.PermissionGroup_ON),
	},
	{
		//Id:        trans.Ptr(uint32(5)),
		ParentId:  trans.Ptr(uint32(1)),
		Name:      trans.Ptr("安全策略"),
		Path:      trans.Ptr("/1/5/"),
		Module:    trans.Ptr(SystemPermissionModule),
		SortOrder: trans.Ptr(uint32(4)),
		Status:    trans.Ptr(permissionV1.PermissionGroup_ON),
	},
}

// DefaultPermissions 系统初始化默认权限数据
var DefaultPermissions = []*permissionV1.Permission{
	{
		//Id:          trans.Ptr(uint32(1)),
		GroupId:     trans.Ptr(uint32(2)),
		Name:        trans.Ptr("访问后台"),
		Description: trans.Ptr("允许用户访问系统后台管理界面"),
		Code:        trans.Ptr(SystemAccessBackendPermissionCode),
		Status:      trans.Ptr(permissionV1.Permission_ON),
	},
	{
		//Id:          trans.Ptr(uint32(2)),
		GroupId:     trans.Ptr(uint32(2)),
		Name:        trans.Ptr("平台管理员权限"),
		Description: trans.Ptr("拥有系统所有功能的操作权限，可管理租户、用户、角色及所有资源"),
		Code:        trans.Ptr(SystemPlatformAdminPermissionCode),
		Status:      trans.Ptr(permissionV1.Permission_ON),
		MenuIds: []uint32{
			1, 2, 3, 4, 5, 6,
			10, 11,
			20, 21, 22, 23, 24,
			30, 31, 32,
			40, 41, 42,
			50, 51, 52, 53, 54, 55,
			60, 61, 62, 63, 64, 65, 66,
		},
		ApiIds: []uint32{
			1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19,
			20, 21, 22, 23, 24, 25, 26, 27, 28, 29,
			30, 31, 32, 33, 34, 35, 36, 37, 38, 39,
			40, 41, 42, 43, 44, 45, 46, 47, 48, 49,
			50, 51, 52, 53, 54, 55, 56, 57, 58, 59,
			60, 61, 62, 63, 64, 65, 66, 67, 68, 69,
			70, 71, 72, 73, 74, 75, 76, 77, 78, 79,
			80, 81, 82, 83, 84, 85, 86, 87, 88, 89,
			90, 91, 92, 93, 94, 95, 96, 97, 98, 99,
			100, 101, 102, 103, 104, 105, 106, 107, 108, 109,
			110, 111, 112, 113, 114, 115, 116, 117, 118, 119,
			120, 121, 122, 123, 124, 125, 126, 127, 128, 129,
			130, 131, 132, 133, 134, 135, 136,
		},
	},
	{
		//Id:          trans.Ptr(uint32(3)),
		GroupId:     trans.Ptr(uint32(3)),
		Name:        trans.Ptr("租户管理员权限"),
		Description: trans.Ptr("拥有租户内所有功能的操作权限，可管理用户、角色及租户内所有资源"),
		Code:        trans.Ptr(SystemTenantManagerPermissionCode),
		Status:      trans.Ptr(permissionV1.Permission_ON),
		MenuIds: []uint32{
			1, 2,
			20, 21, 22, 23, 24,
			30, 32,
			40, 41,
			50, 51,
			60, 61, 62, 63, 64,
		},
		ApiIds: []uint32{
			1, 2, 3, 4, 5, 6, 7, 8, 9,
			10, 11, 12, 13, 14, 15, 16, 17, 18, 19,
			20, 21, 22, 23, 24, 25, 26, 27, 28, 29,
			30, 31, 32, 33, 34, 35, 36, 37, 38, 39,
			40, 41, 42, 43, 44, 45, 46, 47, 48, 49,
			50, 51, 52, 53, 54, 55, 56, 57, 58, 59,
			60, 61, 62, 63, 64, 65, 66, 67, 68, 69,
			70, 71, 72, 73, 74, 75, 76, 77, 78, 79,
			80, 81, 82, 83, 84, 85, 86, 87, 88, 89,
			90, 91, 92, 93, 94, 95, 96, 97, 98, 99,
			100, 101, 102, 103, 104, 105, 106, 107, 108, 109,
			110, 111, 112, 113, 114, 115, 116, 117, 118, 119,
			120, 121, 122, 123, 124, 125, 126, 127, 128,
		},
	},

	{
		//Id:          trans.Ptr(uint32(4)),
		GroupId:     trans.Ptr(uint32(3)),
		Name:        trans.Ptr("管理租户"),
		Description: trans.Ptr("允许创建/修改/删除租户"),
		Code:        trans.Ptr(SystemManageTenantsPermissionCode),
		Status:      trans.Ptr(permissionV1.Permission_ON),
	},
	{
		//Id:          trans.Ptr(uint32(5)),
		GroupId:     trans.Ptr(uint32(4)),
		Name:        trans.Ptr("查看审计日志"),
		Description: trans.Ptr("允许查看系统操作日志"),
		Code:        trans.Ptr(SystemAuditLogsPermissionCode),
		Status:      trans.Ptr(permissionV1.Permission_ON),
	},
}

// DefaultRoles 系统初始化默认角色数据
var DefaultRoles = []*permissionV1.Role{
	{
		//Id:          trans.Ptr(uint32(1)),
		Name:        trans.Ptr(DefaultPlatformAdminRoleName),
		Code:        trans.Ptr(PlatformAdminRoleCode),
		Status:      trans.Ptr(permissionV1.Role_ON),
		Description: trans.Ptr("拥有系统所有功能的操作权限，可管理租户、用户、角色及所有资源"),
		IsProtected: trans.Ptr(true),
		Type:        trans.Ptr(permissionV1.Role_SYSTEM),
		SortOrder:   trans.Ptr(uint32(1)),
		Permissions: []uint32{1, 2, 4},
	},
	{
		//Id:          trans.Ptr(uint32(2)),
		Name:        trans.Ptr(DefaultTenantManagerRoleName + "模板"),
		Code:        trans.Ptr(TenantAdminTemplateRoleCode),
		Status:      trans.Ptr(permissionV1.Role_ON),
		Description: trans.Ptr("租户管理员角色，拥有租户内所有功能的操作权限，可管理用户、角色及租户内所有资源"),
		IsProtected: trans.Ptr(true),
		Type:        trans.Ptr(permissionV1.Role_TEMPLATE),
		SortOrder:   trans.Ptr(uint32(2)),
		Permissions: []uint32{1, 3},
	},
}

// DefaultRoleMetadata 系统初始化默认角色元数据
var DefaultRoleMetadata = []*permissionV1.RoleMetadata{
	{
		//Id:              trans.Ptr(uint32(1)),
		RoleId:          trans.Ptr(uint32(1)),
		IsTemplate:      trans.Ptr(false),
		TemplateVersion: trans.Ptr(int32(1)),
		Scope:           permissionV1.RoleMetadata_PLATFORM.Enum(),
		SyncPolicy:      permissionV1.RoleMetadata_AUTO.Enum(),
	},
	{
		//Id:              trans.Ptr(uint32(2)),
		RoleId:          trans.Ptr(uint32(2)),
		IsTemplate:      trans.Ptr(true),
		TemplateFor:     trans.Ptr(TenantAdminRoleCode),
		TemplateVersion: trans.Ptr(int32(1)),
		Scope:           permissionV1.RoleMetadata_TENANT.Enum(),
		SyncPolicy:      permissionV1.RoleMetadata_AUTO.Enum(),
	},
}

// DefaultUsers 系统初始化默认用户数据
var DefaultUsers = []*identityV1.User{
	{
		//Id:       trans.Ptr(uint32(1)),
		TenantId: trans.Ptr(uint32(0)),
		Username: trans.Ptr(DefaultAdminUserName),
		Realname: trans.Ptr("喵个咪"),
		Nickname: trans.Ptr("鹳狸猿"),
		Region:   trans.Ptr("中国"),
		Email:    trans.Ptr("admin@gmail.com"),
	},
}

// DefaultUserCredentials 系统初始化默认用户凭证数据
var DefaultUserCredentials = []*authenticationV1.UserCredential{
	{
		UserId:         trans.Ptr(uint32(1)),
		TenantId:       trans.Ptr(uint32(0)),
		IdentityType:   authenticationV1.UserCredential_USERNAME.Enum(),
		Identifier:     trans.Ptr(DefaultAdminUserName),
		CredentialType: authenticationV1.UserCredential_PASSWORD_HASH.Enum(),
		Credential:     trans.Ptr(DefaultAdminPassword),
		IsPrimary:      trans.Ptr(true),
		Status:         authenticationV1.UserCredential_ENABLED.Enum(),
	},
}

// DefaultUserRoles 系统初始化默认用户角色关系数据
var DefaultUserRoles = []*permissionV1.UserRole{
	{
		UserId:    trans.Ptr(uint32(1)),
		TenantId:  trans.Ptr(uint32(0)),
		RoleId:    trans.Ptr(uint32(1)),
		IsPrimary: trans.Ptr(true),
		Status:    permissionV1.UserRole_ACTIVE.Enum(),
	},
}

// DefaultMemberships 系统初始化默认用户成员关系数据
var DefaultMemberships = []*identityV1.Membership{
	{
		UserId:    trans.Ptr(uint32(1)),
		TenantId:  trans.Ptr(uint32(0)),
		Status:    identityV1.Membership_ACTIVE.Enum(),
		IsPrimary: trans.Ptr(true),
		RoleIds:   []uint32{1},
	},
}

// DefaultLanguages 系统初始化默认语言数据
var DefaultLanguages = []*dictV1.Language{
	{LanguageCode: trans.Ptr("zh-CN"), LanguageName: trans.Ptr("中文（简体）"), NativeName: trans.Ptr("简体中文"), IsDefault: trans.Ptr(true), IsEnabled: trans.Ptr(true)},
	{LanguageCode: trans.Ptr("zh-TW"), LanguageName: trans.Ptr("中文（繁体）"), NativeName: trans.Ptr("繁體中文"), IsDefault: trans.Ptr(false), IsEnabled: trans.Ptr(true)},
	{LanguageCode: trans.Ptr("en-US"), LanguageName: trans.Ptr("英语"), NativeName: trans.Ptr("English"), IsDefault: trans.Ptr(false), IsEnabled: trans.Ptr(true)},
	{LanguageCode: trans.Ptr("ja-JP"), LanguageName: trans.Ptr("日语"), NativeName: trans.Ptr("日本語"), IsDefault: trans.Ptr(false), IsEnabled: trans.Ptr(true)},
	{LanguageCode: trans.Ptr("ko-KR"), LanguageName: trans.Ptr("韩语"), NativeName: trans.Ptr("한국어"), IsDefault: trans.Ptr(false), IsEnabled: trans.Ptr(true)},
	{LanguageCode: trans.Ptr("es-ES"), LanguageName: trans.Ptr("西班牙语"), NativeName: trans.Ptr("Español"), IsDefault: trans.Ptr(false), IsEnabled: trans.Ptr(true)},
	{LanguageCode: trans.Ptr("fr-FR"), LanguageName: trans.Ptr("法语"), NativeName: trans.Ptr("Français"), IsDefault: trans.Ptr(false), IsEnabled: trans.Ptr(true)},
}

// DefaultMenus 系统初始化默认菜单数据
var DefaultMenus = []*resourceV1.Menu{
	{
		Id:        trans.Ptr(uint32(1)),
		ParentId:  nil,
		Type:      resourceV1.Menu_CATALOG.Enum(),
		Name:      trans.Ptr("Dashboard"),
		Path:      trans.Ptr("/dashboard"),
		Component: trans.Ptr("BasicLayout"),
		Status:    resourceV1.Menu_ON.Enum(),
		CreatedAt: timeutil.TimeToTimestamppb(trans.Ptr(time.Now())),
		Meta: &resourceV1.MenuMeta{
			Order:     trans.Ptr(int32(-1)),
			Title:     trans.Ptr("page.dashboard.title"),
			Icon:      trans.Ptr("lucide:layout-dashboard"),
			Authority: []string{"sys:platform_admin", "sys:tenant_manager"},
		},
	},
	{
		Id:        trans.Ptr(uint32(2)),
		ParentId:  trans.Ptr(uint32(1)),
		Type:      resourceV1.Menu_MENU.Enum(),
		Name:      trans.Ptr("Analytics"),
		Path:      trans.Ptr("/analytics"),
		Component: trans.Ptr("#/views/dashboard/analytics/index.vue"),
		CreatedAt: timeutil.TimeToTimestamppb(trans.Ptr(time.Now())),
		Meta: &resourceV1.MenuMeta{
			Order:     trans.Ptr(int32(-1)),
			Title:     trans.Ptr("page.dashboard.analytics"),
			Icon:      trans.Ptr("lucide:area-chart"),
			Authority: []string{"sys:platform_admin", "sys:tenant_manager"},
			AffixTab:  trans.Ptr(true),
		},
	},

	{
		Id:        trans.Ptr(uint32(3)),
		ParentId:  nil,
		Type:      resourceV1.Menu_CATALOG.Enum(),
		Name:      trans.Ptr("Profile"),
		Path:      trans.Ptr("/profile"),
		Component: trans.Ptr("BasicLayout"),
		CreatedAt: timeutil.TimeToTimestamppb(trans.Ptr(time.Now())),
		Meta: &resourceV1.MenuMeta{
			Title:      trans.Ptr("menu.profile.settings"),
			HideInMenu: trans.Ptr(true),
		},
	},
	{
		Id:        trans.Ptr(uint32(4)),
		ParentId:  trans.Ptr(uint32(3)),
		Type:      resourceV1.Menu_MENU.Enum(),
		Name:      trans.Ptr("ProfilePage"),
		Path:      trans.Ptr("/profile"),
		Component: trans.Ptr("#/views/profile/index.vue"),
		CreatedAt: timeutil.TimeToTimestamppb(trans.Ptr(time.Now())),
		Meta: &resourceV1.MenuMeta{
			Title:      trans.Ptr("menu.profile.settings"),
			Icon:       trans.Ptr("lucide:user-pen"),
			HideInMenu: trans.Ptr(true),
		},
	},

	{
		Id:        trans.Ptr(uint32(5)),
		ParentId:  nil,
		Type:      resourceV1.Menu_CATALOG.Enum(),
		Name:      trans.Ptr("Inbox"),
		Path:      trans.Ptr("/inbox"),
		Component: trans.Ptr("BasicLayout"),
		CreatedAt: timeutil.TimeToTimestamppb(trans.Ptr(time.Now())),
		Meta: &resourceV1.MenuMeta{
			Title:      trans.Ptr("menu.profile.internalMessage"),
			HideInMenu: trans.Ptr(true),
		},
	},
	{
		Id:        trans.Ptr(uint32(6)),
		ParentId:  trans.Ptr(uint32(5)),
		Type:      resourceV1.Menu_MENU.Enum(),
		Name:      trans.Ptr("InboxPage"),
		Path:      trans.Ptr("/inbox"),
		Component: trans.Ptr("#/views/message/index.vue"),
		CreatedAt: timeutil.TimeToTimestamppb(trans.Ptr(time.Now())),
		Meta: &resourceV1.MenuMeta{
			Title:      trans.Ptr("menu.profile.internalMessage"),
			Icon:       trans.Ptr("lucide:message-circle-more"),
			HideInMenu: trans.Ptr(true),
		},
	},

	{
		Id:        trans.Ptr(uint32(10)),
		ParentId:  nil,
		Type:      resourceV1.Menu_CATALOG.Enum(),
		Name:      trans.Ptr("TenantManagement"),
		Path:      trans.Ptr("/tenant"),
		Redirect:  trans.Ptr("/tenant/members"),
		Component: trans.Ptr("BasicLayout"),
		CreatedAt: timeutil.TimeToTimestamppb(trans.Ptr(time.Now())),
		Meta: &resourceV1.MenuMeta{
			Order:     trans.Ptr(int32(2000)),
			Title:     trans.Ptr("menu.tenant.moduleName"),
			Icon:      trans.Ptr("lucide:building-2"),
			Authority: []string{"sys:platform_admin"},
		},
	},
	{
		Id:        trans.Ptr(uint32(11)),
		ParentId:  trans.Ptr(uint32(10)),
		Type:      resourceV1.Menu_MENU.Enum(),
		Name:      trans.Ptr("TenantMemberManagement"),
		Path:      trans.Ptr("members"),
		Component: trans.Ptr("#/views/app/tenant/tenant/index.vue"),
		CreatedAt: timeutil.TimeToTimestamppb(trans.Ptr(time.Now())),
		Meta: &resourceV1.MenuMeta{
			Order:     trans.Ptr(int32(1)),
			Title:     trans.Ptr("menu.tenant.member"),
			Icon:      trans.Ptr("lucide:users"),
			Authority: []string{"sys:platform_admin"},
			AffixTab:  trans.Ptr(true),
		},
	},

	{
		Id:        trans.Ptr(uint32(20)),
		ParentId:  nil,
		Type:      resourceV1.Menu_CATALOG.Enum(),
		Name:      trans.Ptr("OrganizationalPersonnelManagement"),
		Path:      trans.Ptr("/opm"),
		Redirect:  trans.Ptr("/opm/users"),
		Component: trans.Ptr("BasicLayout"),
		CreatedAt: timeutil.TimeToTimestamppb(trans.Ptr(time.Now())),
		Meta: &resourceV1.MenuMeta{
			Order:     trans.Ptr(int32(2001)),
			Title:     trans.Ptr("menu.opm.moduleName"),
			Icon:      trans.Ptr("lucide:users"),
			KeepAlive: trans.Ptr(true),
			Authority: []string{"sys:platform_admin", "sys:tenant_manager"},
		},
	},
	{
		Id:        trans.Ptr(uint32(21)),
		ParentId:  trans.Ptr(uint32(20)),
		Type:      resourceV1.Menu_MENU.Enum(),
		Name:      trans.Ptr("OrgUnitManagement"),
		Path:      trans.Ptr("org-units"),
		Component: trans.Ptr("#/views/app/opm/org_unit/index.vue"),
		CreatedAt: timeutil.TimeToTimestamppb(trans.Ptr(time.Now())),
		Meta: &resourceV1.MenuMeta{
			Order:     trans.Ptr(int32(1)),
			Title:     trans.Ptr("menu.opm.orgUnit"),
			Icon:      trans.Ptr("lucide:layers"),
			Authority: []string{"sys:platform_admin", "sys:tenant_manager"},
		},
	},
	{
		Id:        trans.Ptr(uint32(22)),
		ParentId:  trans.Ptr(uint32(20)),
		Type:      resourceV1.Menu_MENU.Enum(),
		Name:      trans.Ptr("PositionManagement"),
		Path:      trans.Ptr("positions"),
		Component: trans.Ptr("#/views/app/opm/position/index.vue"),
		CreatedAt: timeutil.TimeToTimestamppb(trans.Ptr(time.Now())),
		Meta: &resourceV1.MenuMeta{
			Order:     trans.Ptr(int32(2)),
			Title:     trans.Ptr("menu.opm.position"),
			Icon:      trans.Ptr("lucide:briefcase"),
			Authority: []string{"sys:platform_admin", "sys:tenant_manager"},
		},
	},
	{
		Id:        trans.Ptr(uint32(23)),
		ParentId:  trans.Ptr(uint32(20)),
		Type:      resourceV1.Menu_MENU.Enum(),
		Name:      trans.Ptr("UserManagement"),
		Path:      trans.Ptr("users"),
		Component: trans.Ptr("#/views/app/opm/position/index.vue"),
		CreatedAt: timeutil.TimeToTimestamppb(trans.Ptr(time.Now())),
		Meta: &resourceV1.MenuMeta{
			Order:     trans.Ptr(int32(3)),
			Title:     trans.Ptr("menu.opm.user"),
			Icon:      trans.Ptr("lucide:user"),
			Authority: []string{"sys:platform_admin", "sys:tenant_manager"},
		},
	},
	{
		Id:        trans.Ptr(uint32(24)),
		ParentId:  trans.Ptr(uint32(20)),
		Type:      resourceV1.Menu_MENU.Enum(),
		Name:      trans.Ptr("UserDetail"),
		Path:      trans.Ptr("users/detail/:id"),
		Component: trans.Ptr("#/views/app/opm/user/detail/index.vue"),
		CreatedAt: timeutil.TimeToTimestamppb(trans.Ptr(time.Now())),
		Meta: &resourceV1.MenuMeta{
			Title:      trans.Ptr("menu.opm.userDetail"),
			Authority:  []string{"sys:platform_admin", "sys:tenant_manager"},
			HideInMenu: trans.Ptr(true),
		},
	},

	{
		Id:        trans.Ptr(uint32(30)),
		ParentId:  nil,
		Type:      resourceV1.Menu_CATALOG.Enum(),
		Name:      trans.Ptr("PermissionManagement"),
		Path:      trans.Ptr("/permission"),
		Redirect:  trans.Ptr("/permission/codes"),
		Component: trans.Ptr("BasicLayout"),
		CreatedAt: timeutil.TimeToTimestamppb(trans.Ptr(time.Now())),
		Meta: &resourceV1.MenuMeta{
			Order:     trans.Ptr(int32(2002)),
			Title:     trans.Ptr("menu.permission.moduleName"),
			Icon:      trans.Ptr("lucide:shield-check"),
			KeepAlive: trans.Ptr(true),
			Authority: []string{"sys:platform_admin", "sys:tenant_manager"},
		},
	},
	{
		Id:        trans.Ptr(uint32(31)),
		ParentId:  trans.Ptr(uint32(30)),
		Type:      resourceV1.Menu_MENU.Enum(),
		Name:      trans.Ptr("PermissionPointManagement"),
		Path:      trans.Ptr("codes"),
		Component: trans.Ptr("#/views/app/permission/permission/index.vue"),
		CreatedAt: timeutil.TimeToTimestamppb(trans.Ptr(time.Now())),
		Meta: &resourceV1.MenuMeta{
			Title:     trans.Ptr("menu.permission.permission"),
			Icon:      trans.Ptr("lucide:shield-ellipsis"),
			Order:     trans.Ptr(int32(1)),
			Authority: []string{"sys:platform_admin"},
		},
	},
	{
		Id:        trans.Ptr(uint32(32)),
		ParentId:  trans.Ptr(uint32(30)),
		Type:      resourceV1.Menu_MENU.Enum(),
		Name:      trans.Ptr("RoleManagement"),
		Path:      trans.Ptr("roles"),
		Component: trans.Ptr("#/views/app/permission/role/index.vue"),
		CreatedAt: timeutil.TimeToTimestamppb(trans.Ptr(time.Now())),
		Meta: &resourceV1.MenuMeta{
			Title:     trans.Ptr("menu.permission.role"),
			Icon:      trans.Ptr("lucide:shield-user"),
			Order:     trans.Ptr(int32(2)),
			Authority: []string{"sys:platform_admin", "sys:tenant_manager"},
		},
	},

	{
		Id:        trans.Ptr(uint32(40)),
		ParentId:  nil,
		Type:      resourceV1.Menu_CATALOG.Enum(),
		Name:      trans.Ptr("InternalMessageManagement"),
		Path:      trans.Ptr("/internal-message"),
		Redirect:  trans.Ptr("/internal-message/messages"),
		Component: trans.Ptr("BasicLayout"),
		CreatedAt: timeutil.TimeToTimestamppb(trans.Ptr(time.Now())),
		Meta: &resourceV1.MenuMeta{
			Order:     trans.Ptr(int32(2003)),
			Title:     trans.Ptr("menu.internalMessage.moduleName"),
			Icon:      trans.Ptr("lucide:mail"),
			KeepAlive: trans.Ptr(true),
			Authority: []string{"sys:platform_admin", "sys:tenant_manager"},
		},
	},
	{
		Id:        trans.Ptr(uint32(41)),
		ParentId:  trans.Ptr(uint32(40)),
		Type:      resourceV1.Menu_MENU.Enum(),
		Name:      trans.Ptr("InternalMessageList"),
		Path:      trans.Ptr("messages"),
		Component: trans.Ptr("#/views/app/internal_message/message/index.vue"),
		CreatedAt: timeutil.TimeToTimestamppb(trans.Ptr(time.Now())),
		Meta: &resourceV1.MenuMeta{
			Title:     trans.Ptr("menu.internalMessage.internalMessage"),
			Icon:      trans.Ptr("lucide:message-circle-more"),
			Order:     trans.Ptr(int32(1)),
			Authority: []string{"sys:platform_admin", "sys:tenant_manager"},
		},
	},
	{
		Id:        trans.Ptr(uint32(42)),
		ParentId:  trans.Ptr(uint32(40)),
		Type:      resourceV1.Menu_MENU.Enum(),
		Name:      trans.Ptr("InternalMessageCategoryManagement"),
		Path:      trans.Ptr("categories"),
		Component: trans.Ptr("#/views/app/internal_message/category/index.vue"),
		CreatedAt: timeutil.TimeToTimestamppb(trans.Ptr(time.Now())),
		Meta: &resourceV1.MenuMeta{
			Title:     trans.Ptr("menu.internalMessage.internalMessageCategory"),
			Icon:      trans.Ptr("lucide:calendar-check"),
			Order:     trans.Ptr(int32(2)),
			Authority: []string{"sys:platform_admin"},
		},
	},

	{
		Id:        trans.Ptr(uint32(50)),
		ParentId:  nil,
		Type:      resourceV1.Menu_CATALOG.Enum(),
		Name:      trans.Ptr("LogAuditManagement"),
		Path:      trans.Ptr("/log"),
		Redirect:  trans.Ptr("/log/login-audit-logs"),
		Component: trans.Ptr("BasicLayout"),
		CreatedAt: timeutil.TimeToTimestamppb(trans.Ptr(time.Now())),
		Meta: &resourceV1.MenuMeta{
			Order:     trans.Ptr(int32(2004)),
			Title:     trans.Ptr("menu.log.moduleName"),
			Icon:      trans.Ptr("lucide:logs"),
			KeepAlive: trans.Ptr(true),
			Authority: []string{"sys:platform_admin"},
		},
	},
	{
		Id:        trans.Ptr(uint32(51)),
		ParentId:  trans.Ptr(uint32(50)),
		Type:      resourceV1.Menu_MENU.Enum(),
		Name:      trans.Ptr("LoginAuditLog"),
		Path:      trans.Ptr("login-audit-logs"),
		Component: trans.Ptr("#/views/app/log/login_audit_log/index.vue"),
		CreatedAt: timeutil.TimeToTimestamppb(trans.Ptr(time.Now())),
		Meta: &resourceV1.MenuMeta{
			Title:     trans.Ptr("menu.log.loginAuditLog"),
			Icon:      trans.Ptr("lucide:user-lock"),
			Order:     trans.Ptr(int32(1)),
			Authority: []string{"sys:platform_admin"},
		},
	},
	{
		Id:        trans.Ptr(uint32(52)),
		ParentId:  trans.Ptr(uint32(50)),
		Type:      resourceV1.Menu_MENU.Enum(),
		Name:      trans.Ptr("ApiAuditLog"),
		Path:      trans.Ptr("api-audit-logs"),
		Component: trans.Ptr("#/views/app/log/api_audit_log/index.vue"),
		CreatedAt: timeutil.TimeToTimestamppb(trans.Ptr(time.Now())),
		Meta: &resourceV1.MenuMeta{
			Title:     trans.Ptr("menu.log.apiAuditLog"),
			Icon:      trans.Ptr("lucide:file-clock"),
			Order:     trans.Ptr(int32(2)),
			Authority: []string{"sys:platform_admin"},
		},
	},
	{
		Id:        trans.Ptr(uint32(53)),
		ParentId:  trans.Ptr(uint32(50)),
		Type:      resourceV1.Menu_MENU.Enum(),
		Name:      trans.Ptr("OperationAuditLog"),
		Path:      trans.Ptr("operation-audit-logs"),
		Component: trans.Ptr("#/views/app/log/operation_audit_log/index.vue"),
		CreatedAt: timeutil.TimeToTimestamppb(trans.Ptr(time.Now())),
		Meta: &resourceV1.MenuMeta{
			Title:     trans.Ptr("menu.log.operationAuditLog"),
			Icon:      trans.Ptr("lucide:shield-ellipsis"),
			Order:     trans.Ptr(int32(3)),
			Authority: []string{"sys:platform_admin"},
		},
	},
	{
		Id:        trans.Ptr(uint32(54)),
		ParentId:  trans.Ptr(uint32(50)),
		Type:      resourceV1.Menu_MENU.Enum(),
		Name:      trans.Ptr("DataAccessAuditLog"),
		Path:      trans.Ptr("data-access-audit-logs"),
		Component: trans.Ptr("#/views/app/log/data_access_audit_log/index.vue"),
		CreatedAt: timeutil.TimeToTimestamppb(trans.Ptr(time.Now())),
		Meta: &resourceV1.MenuMeta{
			Title:     trans.Ptr("menu.log.dataAccessAuditLog"),
			Icon:      trans.Ptr("lucide:shield-check"),
			Order:     trans.Ptr(int32(4)),
			Authority: []string{"sys:platform_admin"},
		},
	},
	{
		Id:        trans.Ptr(uint32(55)),
		ParentId:  trans.Ptr(uint32(50)),
		Type:      resourceV1.Menu_MENU.Enum(),
		Name:      trans.Ptr("PermissionAuditLog"),
		Path:      trans.Ptr("permission-audit-logs"),
		Component: trans.Ptr("#/views/app/log/permission_audit_log/index.vue"),
		CreatedAt: timeutil.TimeToTimestamppb(trans.Ptr(time.Now())),
		Meta: &resourceV1.MenuMeta{
			Title:     trans.Ptr("menu.log.permissionAuditLog"),
			Icon:      trans.Ptr("lucide:shield-alert"),
			Order:     trans.Ptr(int32(5)),
			Authority: []string{"sys:platform_admin"},
		},
	},

	{
		Id:        trans.Ptr(uint32(60)),
		ParentId:  nil,
		Type:      resourceV1.Menu_CATALOG.Enum(),
		Name:      trans.Ptr("System"),
		Path:      trans.Ptr("/system"),
		Redirect:  trans.Ptr("/system/menus"),
		Component: trans.Ptr("BasicLayout"),
		CreatedAt: timeutil.TimeToTimestamppb(trans.Ptr(time.Now())),
		Meta: &resourceV1.MenuMeta{
			Order:     trans.Ptr(int32(2005)),
			Title:     trans.Ptr("menu.system.moduleName"),
			Icon:      trans.Ptr("lucide:settings"),
			KeepAlive: trans.Ptr(true),
			Authority: []string{"sys:platform_admin", "sys:tenant_manager"},
		},
	},
	{
		Id:        trans.Ptr(uint32(61)),
		ParentId:  trans.Ptr(uint32(60)),
		Type:      resourceV1.Menu_MENU.Enum(),
		Name:      trans.Ptr("MenuManagement"),
		Path:      trans.Ptr("menus"),
		Component: trans.Ptr("#/views/app/system/menu/index.vue"),
		CreatedAt: timeutil.TimeToTimestamppb(trans.Ptr(time.Now())),
		Meta: &resourceV1.MenuMeta{
			Title:     trans.Ptr("menu.system.menu"),
			Icon:      trans.Ptr("lucide:square-menu"),
			Order:     trans.Ptr(int32(1)),
			Authority: []string{"sys:platform_admin"},
		},
	},
	{
		Id:        trans.Ptr(uint32(62)),
		ParentId:  trans.Ptr(uint32(60)),
		Type:      resourceV1.Menu_MENU.Enum(),
		Name:      trans.Ptr("APIManagement"),
		Path:      trans.Ptr("apis"),
		Component: trans.Ptr("#/views/app/system/api/index.vue"),
		CreatedAt: timeutil.TimeToTimestamppb(trans.Ptr(time.Now())),
		Meta: &resourceV1.MenuMeta{
			Title:     trans.Ptr("menu.system.api"),
			Icon:      trans.Ptr("lucide:route"),
			Order:     trans.Ptr(int32(2)),
			Authority: []string{"sys:platform_admin"},
		},
	},
	{
		Id:        trans.Ptr(uint32(63)),
		ParentId:  trans.Ptr(uint32(60)),
		Type:      resourceV1.Menu_MENU.Enum(),
		Name:      trans.Ptr("DictManagement"),
		Path:      trans.Ptr("dict"),
		Component: trans.Ptr("#/views/app/system/dict/index.vue"),
		CreatedAt: timeutil.TimeToTimestamppb(trans.Ptr(time.Now())),
		Meta: &resourceV1.MenuMeta{
			Title:     trans.Ptr("menu.system.dict"),
			Icon:      trans.Ptr("lucide:library-big"),
			Order:     trans.Ptr(int32(3)),
			Authority: []string{"sys:platform_admin"},
		},
	},
	{
		Id:        trans.Ptr(uint32(64)),
		ParentId:  trans.Ptr(uint32(60)),
		Type:      resourceV1.Menu_MENU.Enum(),
		Name:      trans.Ptr("FileManagement"),
		Path:      trans.Ptr("files"),
		Component: trans.Ptr("#/views/app/system/file/index.vue"),
		CreatedAt: timeutil.TimeToTimestamppb(trans.Ptr(time.Now())),
		Meta: &resourceV1.MenuMeta{
			Title:     trans.Ptr("menu.system.file"),
			Icon:      trans.Ptr("lucide:file-search"),
			Order:     trans.Ptr(int32(4)),
			Authority: []string{"sys:platform_admin", "sys:tenant_manager"},
		},
	},
	{
		Id:        trans.Ptr(uint32(65)),
		ParentId:  trans.Ptr(uint32(60)),
		Type:      resourceV1.Menu_MENU.Enum(),
		Name:      trans.Ptr("TaskManagement"),
		Path:      trans.Ptr("tasks"),
		Component: trans.Ptr("#/views/app/system/task/index.vue"),
		CreatedAt: timeutil.TimeToTimestamppb(trans.Ptr(time.Now())),
		Meta: &resourceV1.MenuMeta{
			Title:     trans.Ptr("menu.system.task"),
			Icon:      trans.Ptr("lucide:list-todo"),
			Order:     trans.Ptr(int32(5)),
			Authority: []string{"sys:platform_admin", "sys:tenant_manager"},
		},
	},
	{
		Id:        trans.Ptr(uint32(66)),
		ParentId:  trans.Ptr(uint32(60)),
		Type:      resourceV1.Menu_MENU.Enum(),
		Name:      trans.Ptr("LoginPolicyManagement"),
		Path:      trans.Ptr("login-policies"),
		Component: trans.Ptr("#/views/app/system/login_policy/index.vue"),
		CreatedAt: timeutil.TimeToTimestamppb(trans.Ptr(time.Now())),
		Meta: &resourceV1.MenuMeta{
			Title:     trans.Ptr("menu.system.loginPolicy"),
			Icon:      trans.Ptr("lucide:shield-x"),
			Order:     trans.Ptr(int32(6)),
			Authority: []string{"sys:platform_admin"},
		},
	},
}
