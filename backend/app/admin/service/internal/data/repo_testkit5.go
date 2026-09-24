// 跨包（service 层）测试装配：导出免 bootstrap.Context 的 repo 构造器，
// 字段初始化必须与生产 NewXxxRepo 逐字段一致，改生产构造器须同步此处。
//
// 说明：
//   - 本文件为批 3c（AuthenticationService / UserService 测试）所需的构造器，分三类：
//     1) 纯 ent 构造（LoginPolicyRepo / UserMfaFactorRepo / PermissionRepo /
//     RoleOrgUnitRepo / RoleFieldPermissionRepo）：字段与生产构造器逐字段对齐
//     （含 mapper/converter 初始化与 init() 调用），唯一差异是 log 一律
//     bLogger.NewHelper(bLogger.NopLogger())；entClient 由调用方传入（测试场景
//     为 enttest.NewEntClientForTest 的 SQLite 内存库）。PermissionRepo /
//     RoleOrgUnitRepo / RoleFieldPermissionRepo 与 repo_testkit.go 内同名
//     testkit 私有构造器逻辑一致，此处为跨包（service 层）可见的导出副本。
//     2) redis 依赖构造（ConfigRepo / UserTokenCache / LoginRateLimiter /
//     MfaChallengeCache）：redis 客户端由调用方注入，测试场景一律传
//     miniredis 假 client（与 data 包内 authenticator_test.go /
//     user_token_cache_test.go 的注入范式一致）。ConfigRepo 生产构造器中的
//     subscribeInvalidations 常驻 goroutine 是跨实例失效广播订阅（运行期副作用
//     而非字段初始化），测试场景单实例无广播需求，此处不启动。
//     3) UserCredentialRepo：生产构造器参数（passwordCrypto / configRepo）原样保留，
//     由调用方传 bCrypt 实现与 miniredis 注入的 ConfigRepo，装配关系与生产一致。
//     4) Authenticator：对齐 NewAuthenticator 的字段装配（jwtCfg +
//     userTokenCache + newAdminAuthenticator），但省去 applyJwtKeyOverrides
//     （环境变量密钥覆盖——测试直接传显式密钥）并把启动失败 panic 改为返回
//     error，便于测试用临时密钥构造。
//   - 依赖 minio/smtp/短信等外部件的构造器不在此导出。
package data

import (
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/redis/go-redis/v9"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/password"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/loginpolicy"
	"go-wind-admin/app/admin/service/internal/data/ent/permission"
	"go-wind-admin/app/admin/service/internal/data/ent/sysconfig"
	"go-wind-admin/app/admin/service/internal/data/ent/usercredential"

	conf "github.com/tx7do/kratos-bootstrap/api/gen/go/conf/v1"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	configV1 "go-wind-admin/api/gen/go/config/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

// NewLoginPolicyRepoForTest 与生产 NewLoginPolicyRepo 逐字段一致（log 换 NopLogger），并调用 init()。
func NewLoginPolicyRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *LoginPolicyRepo {
	repo := &LoginPolicyRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[authenticationV1.LoginPolicy, ent.LoginPolicy](),
		typeConverter: mapper.NewEnumTypeConverter[authenticationV1.LoginPolicy_Type, loginpolicy.Type](
			authenticationV1.LoginPolicy_Type_name,
			authenticationV1.LoginPolicy_Type_value,
		),
		methodConverter: mapper.NewEnumTypeConverter[authenticationV1.LoginPolicy_Method, loginpolicy.Method](
			authenticationV1.LoginPolicy_Method_name,
			authenticationV1.LoginPolicy_Method_value,
		),
	}

	repo.init()

	return repo
}

// NewUserMfaFactorRepoForTest 与生产 NewUserMfaFactorRepo 逐字段一致（log 换 NopLogger）。
func NewUserMfaFactorRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *UserMfaFactorRepo {
	return &UserMfaFactorRepo{
		entClient: entClient,
		log:       bLogger.NewHelper(bLogger.NopLogger()),
	}
}

// NewPermissionRepoForTest 与生产 NewPermissionRepo 逐字段一致（log 换 NopLogger；
// 关联子 repo 复用 repo_testkit.go 的 testkit 私有构造器），并调用 init()。
func NewPermissionRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *PermissionRepo {
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

// NewRoleOrgUnitRepoForTest 与生产 NewRoleOrgUnitRepo 逐字段一致（log 换 NopLogger）。
func NewRoleOrgUnitRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *RoleOrgUnitRepo {
	return &RoleOrgUnitRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
	}
}

// NewRoleFieldPermissionRepoForTest 与生产 NewRoleFieldPermissionRepo 逐字段一致
// （log 换 NopLogger）。
func NewRoleFieldPermissionRepoForTest(entClient *entCrud.EntClient[*ent.Client]) *RoleFieldPermissionRepo {
	return &RoleFieldPermissionRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
	}
}

// NewUserCredentialRepoForTest 与生产 NewUserCredentialRepo 逐字段一致
// （log 换 NopLogger；passwordCrypto / configRepo 按生产签名保留，由调用方注入），
// 并调用 init()。
func NewUserCredentialRepoForTest(
	entClient *entCrud.EntClient[*ent.Client],
	passwordCrypto password.Crypto,
	configRepo *ConfigRepo,
) *UserCredentialRepo {
	repo := &UserCredentialRepo{
		log:                     bLogger.NewHelper(bLogger.NopLogger()),
		entClient:               entClient,
		passwordCrypto:          passwordCrypto,
		configRepo:              configRepo,
		mapper:                  mapper.NewCopierMapper[authenticationV1.UserCredential, ent.UserCredential](),
		statusConverter:         mapper.NewEnumTypeConverter[authenticationV1.UserCredential_Status, usercredential.Status](authenticationV1.UserCredential_Status_name, authenticationV1.UserCredential_Status_value),
		identityTypeConverter:   mapper.NewEnumTypeConverter[authenticationV1.UserCredential_IdentityType, usercredential.IdentityType](authenticationV1.UserCredential_IdentityType_name, authenticationV1.UserCredential_IdentityType_value),
		credentialTypeConverter: mapper.NewEnumTypeConverter[authenticationV1.UserCredential_CredentialType, usercredential.CredentialType](authenticationV1.UserCredential_CredentialType_name, authenticationV1.UserCredential_CredentialType_value),
	}

	repo.init()

	return repo
}

// NewConfigRepoForTest 与生产 NewConfigRepo 逐字段一致（log 换 NopLogger；
// rdb 由调用方注入——测试场景为 miniredis 假 client；运行期副作用
// subscribeInvalidations goroutine 不启动，见文件头说明），并调用 init()。
func NewConfigRepoForTest(entClient *entCrud.EntClient[*ent.Client], rdb *redis.Client) *ConfigRepo {
	repo := &ConfigRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		rdb:       rdb,
		mapper:    mapper.NewCopierMapper[configV1.Config, ent.SysConfig](),
		valueTypeConverter: mapper.NewEnumTypeConverter[configV1.Config_ConfigValueType, sysconfig.ValueType](
			configV1.Config_ConfigValueType_name,
			configV1.Config_ConfigValueType_value,
		),
		cache: make(map[string]sysConfigCacheEntry),
	}

	repo.init()

	return repo
}

// NewUserTokenCacheForTest 与生产 NewUserTokenCache 逐字段一致
// （log 换 NopLogger；rdb 由调用方注入——测试场景为 miniredis 假 client）。
func NewUserTokenCacheForTest(rdb *redis.Client) *UserTokenCache {
	return &UserTokenCache{
		rdb: rdb,
		log: bLogger.NewHelper(bLogger.NopLogger()),
	}
}

// NewLoginRateLimiterForTest 与生产 NewLoginRateLimiter 逐字段一致
// （log 换 NopLogger；rdb 由调用方注入——测试场景为 miniredis 假 client）。
func NewLoginRateLimiterForTest(rdb *redis.Client) *LoginRateLimiter {
	return &LoginRateLimiter{
		rdb: rdb,
		log: bLogger.NewHelper(bLogger.NopLogger()),
	}
}

// NewMfaChallengeCacheForTest 与生产 NewMfaChallengeCache 逐字段一致
// （log 换 NopLogger；rdb 由调用方注入——测试场景为 miniredis 假 client）。
func NewMfaChallengeCacheForTest(rdb *redis.Client) *MfaChallengeCache {
	return &MfaChallengeCache{
		rdb: rdb,
		log: bLogger.NewHelper(bLogger.NopLogger()),
	}
}

// NewAuthenticatorForTest 对齐生产 NewAuthenticator 的字段装配
// （jwtCfg + userTokenCache + newAdminAuthenticator，log 换 NopLogger），
// 唯一差异：省去 applyJwtKeyOverrides 环境变量密钥覆盖、构造失败改为返回 error
// 而非 panic（见文件头说明）。jwtCfg 由调用方传入测试密钥。
func NewAuthenticatorForTest(jwtCfg *conf.Authentication_Jwt, userTokenCache *UserTokenCache) (*Authenticator, error) {
	adminAuth, err := newAdminAuthenticator(jwtCfg)
	if err != nil {
		return nil, err
	}

	a := Authenticator{
		log:                bLogger.NewHelper(bLogger.NopLogger()),
		AdminAuthenticator: adminAuth,
		jwtCfg:             jwtCfg,
		userTokenCache:     userTokenCache,
	}

	return &a, nil
}
