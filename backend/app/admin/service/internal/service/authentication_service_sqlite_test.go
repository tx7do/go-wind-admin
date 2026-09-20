// AuthenticationService 的 SQLite 内存库 + miniredis 集成测试（白盒，包内测试）。
//
// 装配范式：
//   - ent 仓储（tenant / role / permission / login_policy / user_mfa_factor /
//     user_credential(+config)）走 data 包 repo_testkit 构造器（enttest SQLite 内存库）；
//   - redis 依赖（user_token_cache / login_rate_limiter / mfa_challenge_cache / captcha）
//     一律 miniredis 假 client 注入（对齐 data 包 authenticator_test.go /
//     user_token_cache_test.go 的注入范式），authenticator 走
//     data.NewAuthenticatorForTest（测试 HS256 临时密钥）；
//   - userRepo 为接口（data.UserRepo），用本文件桩按 id 回放占位用户 / 按用户回放
//     角色 ID（嵌入接口惯用法，未覆写方法一旦被调用即 nil 接口 panic，测试即失败）；
//   - HTTP 头/cookie 上下文经本文件 khttp.Transporter 接口桩注入
//     （netutil.HeaderFromContext / CookieFromContext 走接口断言可命中）。
//
// 覆盖目标：authentication_service.go 的 Login / doGrantTypePassword（限流预检、
// 验证码闸门、租户解析、登录策略双段闸门、凭证校验与防枚举归一、租户纵深校验、
// 授权链（角色→权限→后台访问→角色码/管理员标志）、数据范围与字段权限聚合（协作文件）、
// MFA 闸门、令牌签发、会话元数据、最后登录记录、限流清零）、Logout、RefreshToken
// （cookie 注入 + 轮换 + 旧令牌对失效 + 会话元数据继承）、ValidateToken、WhoAmI、
// GenerateCaptcha / VerifyCaptcha，以及纯函数 normalizeLoginVerifyError /
// containsPermission / fillAdminFlags 与 cookie 助手的无传输守卫段。
//
// 跳过段（②外部链路，详见汇报）：
//   - resolveCookieSecure 与 setRefreshCookies / clearRefreshCookies 的写头段：
//     需具体 *khttp.Transport（kratos 内部类型、字段全私有无导出构造器），
//     接口桩无法命中其具体类型断言，只测守卫段；
//   - 各仓储/redis 的故障分支（DB/Redis 故障不可注入）、CaptchaEnabled 关闭分支
//     （编译期常量 true）。
package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/captcha"
	utilCrypto "github.com/tx7do/go-utils/crypto"
	"github.com/tx7do/go-utils/password"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/emptypb"

	ktransport "github.com/go-kratos/kratos/v2/transport"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/enttest"

	conf "github.com/tx7do/kratos-bootstrap/api/gen/go/conf/v1"
	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"

	"go-wind-admin/pkg/constants"
	"go-wind-admin/pkg/middleware/auth"
)

// authSvcTestKeyPrefix 本文件验证码 Redis 键前缀（对齐生产 data.NewCaptcha 的
// WithKeyPrefix 选项形态，仅取测试专属前缀便于 miniredis 直读答案）。
const authSvcTestKeyPrefix = "authsvc-test:captcha"

// authSvcTestJWTKey 测试 HS256 签名密钥（仅测试内使用）。
const authSvcTestJWTKey = "authsvc-test-hs256-secret-key-0123456789abcdef"

const (
	authSvcTestUsername  = "authsvc-admin"
	authSvcTestUserID    = uint32(7001)
	authSvcTenantCode    = "AUTHSVC_TENANT_CHAIN"
	authSvcTenantCodeOff = "AUTHSVC_TENANT_OFF"
	authSvcTenantCodeB   = "AUTHSVC_TENANT_MISMATCH"
)

// authSvcUserRepoStub 是 AuthenticationService 测试专用的 data.UserRepo 桩：
// 嵌入接口获得默认方法集（未覆写方法一旦被调用即 nil 接口 panic，测试即失败），
// 只覆写登录链路会用到的四个方法：Get（按 id 回放占位用户）、
// ListRoleIDsByUserID（按用户回放角色 ID 集）、FindUsernameByIdentifier（原样透传，
// findIdentifierErr 非空时回放错误以覆盖错误分支）、Update（记录
// recordUserLastLogin 的更新调用）。桩数据与实体表无关。
type authSvcUserRepoStub struct {
	data.UserRepo
	usersByID         map[uint32]*identityV1.User
	roleIDsByUser     map[uint32][]uint32
	updates           []*identityV1.UpdateUserRequest
	findIdentifierErr error
}

func (s *authSvcUserRepoStub) Get(_ context.Context, req *identityV1.GetUserRequest) (*identityV1.User, error) {
	if u, ok := s.usersByID[req.GetId()]; ok {
		return u, nil
	}
	return nil, fmt.Errorf("user %d not found (authSvc stub)", req.GetId())
}

func (s *authSvcUserRepoStub) ListRoleIDsByUserID(_ context.Context, userID uint32) ([]uint32, error) {
	return s.roleIDsByUser[userID], nil
}

func (s *authSvcUserRepoStub) FindUsernameByIdentifier(_ context.Context, _ uint32, identifier string) (string, uint32, error) {
	if s.findIdentifierErr != nil {
		return "", 0, s.findIdentifierErr
	}
	return identifier, 0, nil
}

func (s *authSvcUserRepoStub) Update(_ context.Context, req *identityV1.UpdateUserRequest) error {
	s.updates = append(s.updates, req)
	return nil
}

// fakeHeaderTransporter 以接口桩实现 khttp.Transporter，为
// netutil.HeaderFromContext / CookieFromContext 提供头/cookie 读取能力
// （两者对 Transporter 做接口断言，桩可命中）。
// 注意：setRefreshCookies / clearRefreshCookies 需要具体 *khttp.Transport，
// 桩不命中其具体类型断言——正好覆盖"无传输上下文直接返回"的守卫段。
type fakeHeaderTransporter struct{ req *http.Request }

func (f *fakeHeaderTransporter) Kind() ktransport.Kind { return ktransport.KindHTTP }
func (f *fakeHeaderTransporter) Endpoint() string      { return "" }
func (f *fakeHeaderTransporter) Operation() string     { return "" }
func (f *fakeHeaderTransporter) Request() *http.Request {
	return f.req
}
func (f *fakeHeaderTransporter) PathTemplate() string             { return "" }
func (f *fakeHeaderTransporter) RequestHeader() ktransport.Header { return nil }
func (f *fakeHeaderTransporter) ReplyHeader() ktransport.Header   { return nil }

// headerCtx 注入带任意头的 HTTP 传输上下文。
func headerCtx(ctx context.Context, header http.Header) context.Context {
	req := &http.Request{Header: header}
	return ktransport.NewServerContext(ctx, &fakeHeaderTransporter{req: req})
}

// authSvcEnv 聚合 AuthenticationService 测试装配
// （每测试全新 enttest client + 全新 miniredis，互不串扰）。
type authSvcEnv struct {
	svc           *AuthenticationService
	stub          *authSvcUserRepoStub
	mr            *miniredis.Miniredis
	captchaClient *captcha.Captcha
	ctx           context.Context // SystemViewer（仅用于种子数据写入）
}

// newAuthenticationServiceForTest 白盒复刻 NewAuthenticationService 的字段初始化：
// log 换 NopLogger，ent 仓储用 repo_testkit 构造器，redis 依赖经 miniredis 假 client
// 注入，userRepo 用本文件桩；membershipRepo / vcodeCache / notifier
// 仅被本批未覆盖的方法（OneToMany 分支为编译期死代码、忘记密码/换绑链路）使用，置 nil。
func newAuthenticationServiceForTest(t *testing.T) *authSvcEnv {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	// captcha：对齐生产 data.NewCaptcha 的选项形态（DriverString + 10 分钟过期），
	// 键前缀换测试专属值便于 miniredis 直读答案。
	captchaClient := captcha.NewCaptcha(rdb,
		captcha.WithDriverType(captcha.DriverString),
		captcha.WithExpire(10*time.Minute),
		captcha.WithKeyPrefix(authSvcTestKeyPrefix),
		captcha.WithStringCount(6),
		captcha.WithStringSource("ABCDEFGHJKLMNPQRSTUVWXYZ23456789"),
	)

	// 凭证仓储：bCrypt（与生产 data.NewPasswordCrypto 等价）+ miniredis 注入的 ConfigRepo。
	passwordCrypto := password.NewBCryptCrypto()
	configRepo := data.NewConfigRepoForTest(entClient, rdb)
	userCredentialRepo := data.NewUserCredentialRepoForTest(entClient, passwordCrypto, configRepo)

	// authenticator：测试 HS256 临时密钥 + miniredis 令牌缓存。
	userTokenCache := data.NewUserTokenCacheForTest(rdb)
	jwtCfg := &conf.Authentication_Jwt{Method: "HS256", Key: authSvcTestJWTKey}
	authenticator, err := data.NewAuthenticatorForTest(jwtCfg, userTokenCache)
	require.NoError(t, err)

	stub := &authSvcUserRepoStub{
		usersByID:     make(map[uint32]*identityV1.User),
		roleIDsByUser: make(map[uint32][]uint32),
	}

	svc := &AuthenticationService{
		log:                     bLogger.NewHelper(bLogger.NopLogger()),
		userRepo:                stub,
		userCredentialRepo:      userCredentialRepo,
		roleRepo:                data.NewRoleRepoForTest(entClient),
		tenantRepo:              data.NewTenantRepoForTest(entClient),
		membershipRepo:          nil,
		orgUnitRepo:             data.NewOrgUnitRepoForTest(entClient),
		roleOrgUnitRepo:         data.NewRoleOrgUnitRepoForTest(entClient),
		roleFieldPermissionRepo: data.NewRoleFieldPermissionRepoForTest(entClient),
		permissionRepo:          data.NewPermissionRepoForTest(entClient),
		authenticator:           authenticator,
		clientType:              authenticationV1.ClientType_admin,
		captchaClient:           captchaClient,
		rateLimiter:             data.NewLoginRateLimiterForTest(rdb),
		loginPolicyRepo:         data.NewLoginPolicyRepoForTest(entClient),
		mfaFactorRepo:           data.NewUserMfaFactorRepoForTest(entClient),
		mfaChallengeCache:       data.NewMfaChallengeCacheForTest(rdb),
		vcodeCache:              nil,
		notifier:                nil,
	}

	return &authSvcEnv{
		svc:           svc,
		stub:          stub,
		mr:            mr,
		captchaClient: captchaClient,
		ctx:           enttest.NewSystemViewerCtx(context.Background()),
	}
}

// seedTenant 落库一个租户并返回其 ID（状态可指定）。
func (e *authSvcEnv) seedTenant(t *testing.T, name, code string, status identityV1.Tenant_Status) uint32 {
	t.Helper()
	created, err := e.svc.tenantRepo.Create(e.ctx, &identityV1.Tenant{
		Name:        trans.Ptr(name),
		Code:        trans.Ptr(code),
		Status:      status.Enum(),
		Type:        identityV1.Tenant_TRIAL.Enum(),
		AuditStatus: identityV1.Tenant_APPROVED.Enum(),
	})
	require.NoError(t, err)
	require.NotNil(t, created.GetId())
	return created.GetId()
}

// seedBackendAccessPermission 落库 sys:access_backend 权限并返回其 ID。
func (e *authSvcEnv) seedBackendAccessPermission(t *testing.T) uint32 {
	t.Helper()
	require.NoError(t, e.svc.permissionRepo.Create(e.ctx, &permissionV1.CreatePermissionRequest{
		Data: &permissionV1.Permission{
			Name:   trans.Ptr("AuthSvc 后台访问权限"),
			Code:   trans.Ptr(constants.SystemAccessBackendPermissionCode),
			Status: permissionV1.Permission_ON.Enum(),
		},
	}))
	permIDs, err := e.svc.permissionRepo.GetPermissionIDsByCodes(e.ctx, []string{constants.SystemAccessBackendPermissionCode})
	require.NoError(t, err)
	require.Len(t, permIDs, 1, "后台访问权限应恰好落库一行")
	return permIDs[0]
}

// seedRole 落库一个角色（可选绑定权限/数据范围/字段权限）并按 (tenant, code) 反查 ID。
func (e *authSvcEnv) seedRole(
	t *testing.T,
	tenantID *uint32,
	roleType permissionV1.Role_Type,
	roleCode string,
	permIDs []uint32,
	dataScope *identityV1.DataScope,
	fieldPerms []*permissionV1.RoleFieldPermission,
) uint32 {
	t.Helper()
	data := &permissionV1.Role{
		TenantId:    tenantID,
		Name:        trans.Ptr("AuthSvc 角色 " + roleCode),
		Code:        trans.Ptr(roleCode),
		Status:      permissionV1.Role_ON.Enum(),
		Type:        roleType.Enum(),
		Permissions: permIDs,
	}
	if dataScope != nil {
		data.DataScope = dataScope
	}
	if len(fieldPerms) > 0 {
		data.FieldPermissions = fieldPerms
	}
	require.NoError(t, e.svc.roleRepo.Create(e.ctx, &permissionV1.CreateRoleRequest{Data: data}))

	var wantTenant uint32
	if tenantID != nil {
		wantTenant = *tenantID
	}
	listResp, err := e.svc.roleRepo.List(e.ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	for _, item := range listResp.GetItems() {
		if item.GetCode() == roleCode && item.GetTenantId() == wantTenant {
			return item.GetId()
		}
	}
	t.Fatalf("角色 (%d, %s) 未在列表中反查到", wantTenant, roleCode)
	return 0
}

// seedCredential 落库一条 USERNAME/PASSWORD_HASH 凭证（bcrypt 哈希经
// prepareCredential 复杂度校验后入库）。
func (e *authSvcEnv) seedCredential(t *testing.T, tenantID *uint32, userID uint32, identifier string, status authenticationV1.UserCredential_Status) {
	t.Helper()
	require.NoError(t, e.svc.userCredentialRepo.Create(e.ctx, &authenticationV1.CreateUserCredentialRequest{
		Data: &authenticationV1.UserCredential{
			UserId:         trans.Ptr(userID),
			TenantId:       tenantID,
			IdentityType:   authenticationV1.UserCredential_USERNAME.Enum(),
			Identifier:     trans.Ptr(identifier),
			CredentialType: authenticationV1.UserCredential_PASSWORD_HASH.Enum(),
			Credential:     trans.Ptr(constants.DefaultUserPassword),
			IsPrimary:      trans.Ptr(true),
			Status:         status.Enum(),
		},
	}))
}

// seedStubUser 回放一个占位用户并挂上角色 ID 集（stub 数据，不入库）。
func (e *authSvcEnv) seedStubUser(userID uint32, tenantID *uint32, username string, status identityV1.User_Status, roleIDs []uint32) {
	e.stub.usersByID[userID] = &identityV1.User{
		Id:       trans.Ptr(userID),
		TenantId: tenantID,
		Username: trans.Ptr(username),
		Status:   status.Enum(),
	}
	e.stub.roleIDsByUser[userID] = roleIDs
}

// freshCaptchaCtx 生成一对单次有效验证码并以 X-Captcha-Id / X-Captcha-Value 头注入
// 上下文（登录强制验证码闸门的通过路径用）。verifyLoginCaptcha 走 verify-and-delete，
// 每次登录消费一对新验证码。
func (e *authSvcEnv) freshCaptchaCtx(t *testing.T, ctx context.Context) context.Context {
	t.Helper()
	id, _, answer, err := e.captchaClient.Generate()
	require.NoError(t, err)
	require.NoError(t, e.captchaClient.Save(ctx, id, answer))
	header := http.Header{}
	header.Set("X-Captcha-Id", id)
	header.Set("X-Captcha-Value", answer)
	return headerCtx(ctx, header)
}

// encryptLoginPassword 按 Login 前端契约加密密码：base64(AES(明文))。
// doGrantTypePassword 以 needDecrypt=true 调 FindUserCredential，配套此编码。
func encryptLoginPassword(t *testing.T, plain string) string {
	t.Helper()
	enc, err := utilCrypto.AesEncrypt([]byte(plain), utilCrypto.DefaultAESKey, nil)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(enc)
}

// loginReq 构造密码登录请求。
func loginReq(username, encryptedPassword string) *authenticationV1.LoginRequest {
	return &authenticationV1.LoginRequest{
		GrantType:  authenticationV1.GrantType_password,
		Identifier: &authenticationV1.LoginRequest_Username{Username: username},
		Password:   trans.Ptr(encryptedPassword),
		ClientType: authenticationV1.ClientType_admin.Enum(),
	}
}

// loginHappyReq 直接以明文密码构造带验证码上下文的完整登录请求。
func (e *authSvcEnv) loginHappyReq(t *testing.T, ctx context.Context, username, plainPassword string) (context.Context, *authenticationV1.LoginRequest) {
	t.Helper()
	return e.freshCaptchaCtx(t, ctx), loginReq(username, encryptLoginPassword(t, plainPassword))
}

// TestAuthSvcSqlite_LoginGrantDispatch 验证 Login 的授权类型派发：
// refresh_token（已迁移独立端点）与 client_credentials 均返回 INVALID_GRANT_TYPE。
// 注：GrantType 的 0 值即 password（见 authentication.pb.go 枚举），非独立分支。
func TestAuthSvcSqlite_LoginGrantDispatch(t *testing.T) {
	e := newAuthenticationServiceForTest(t)
	ctx := context.Background()

	cases := []struct {
		name      string
		grantType authenticationV1.GrantType
	}{
		{"refresh_token", authenticationV1.GrantType_refresh_token},
		{"client_credentials", authenticationV1.GrantType_client_credentials},
	}
	for _, c := range cases {
		resp, err := e.svc.Login(ctx, &authenticationV1.LoginRequest{GrantType: c.grantType})
		require.Error(t, err, "%s 应报错", c.name)
		require.True(t, authenticationV1.IsInvalidGrantType(err), "%s 应为 INVALID_GRANT_TYPE", c.name)
		require.Nil(t, resp)
	}
}

// TestAuthSvcSqlite_LoginCaptchaGate 验证强制验证码闸门：
// 无传输上下文（取不到头）、空验证码头、错误验证码值均 400 拒绝；
// 正确验证码放行后进入后续凭证校验（此处以未播种凭证的 INVALID_PASSWORD 证明放行）。
func TestAuthSvcSqlite_LoginCaptchaGate(t *testing.T) {
	e := newAuthenticationServiceForTest(t)
	base := context.Background()
	req := loginReq("authsvc-nobody", encryptLoginPassword(t, constants.DefaultUserPassword))

	// 无传输上下文：HeaderFromContext 取不到头 → 闸门拒绝。
	resp, err := e.svc.Login(base, req)
	require.Error(t, err)
	require.True(t, authenticationV1.IsBadRequest(err))
	require.Nil(t, resp)

	// 有传输上下文但无验证码头。
	resp, err = e.svc.Login(headerCtx(base, http.Header{}), req)
	require.Error(t, err)
	require.True(t, authenticationV1.IsBadRequest(err))
	require.Nil(t, resp)

	// 有传输上下文、验证码头为空白值。
	blank := http.Header{}
	blank.Set("X-Captcha-Id", "  ")
	blank.Set("X-Captcha-Value", "  ")
	resp, err = e.svc.Login(headerCtx(base, blank), req)
	require.Error(t, err)
	require.True(t, authenticationV1.IsBadRequest(err))
	require.Nil(t, resp)

	// 错误验证码值：通过闸门头存在性检查后 Verify 比对失败。
	id, _, _, gerr := e.captchaClient.Generate()
	require.NoError(t, gerr)
	require.NoError(t, e.captchaClient.Save(base, id, "RIGHTANSWER"))
	wrong := http.Header{}
	wrong.Set("X-Captcha-Id", id)
	wrong.Set("X-Captcha-Value", "WRONGANSWER")
	resp, err = e.svc.Login(headerCtx(base, wrong), req)
	require.Error(t, err)
	require.True(t, authenticationV1.IsBadRequest(err))
	require.Nil(t, resp)

	// 正确验证码：闸门放行，流程推进到凭证校验（凭证未播种 → 归一化的 INVALID_PASSWORD）。
	okCtx, okReq := e.loginHappyReq(t, base, "authsvc-nobody", constants.DefaultUserPassword)
	resp, err = e.svc.Login(okCtx, okReq)
	require.Error(t, err)
	require.True(t, authenticationV1.IsInvalidPassword(err), "凭证未命中应归一化为 INVALID_PASSWORD")
	require.Nil(t, resp)
}

// TestAuthSvcSqlite_LoginTenantResolution 验证租户解析：
// 未知租户编号 / 非启用租户 / 凭证租户与用户租户不一致（纵深防御）均 400 invalid tenant。
func TestAuthSvcSqlite_LoginTenantResolution(t *testing.T) {
	e := newAuthenticationServiceForTest(t)
	base := context.Background()

	offTenantID := e.seedTenant(t, "AuthSvc 停用租户", authSvcTenantCodeOff, identityV1.Tenant_OFF)
	onTenantID := e.seedTenant(t, "AuthSvc 启用租户甲", authSvcTenantCode, identityV1.Tenant_ON)
	otherTenantID := e.seedTenant(t, "AuthSvc 启用租户乙", authSvcTenantCodeB, identityV1.Tenant_ON)
	require.NotZero(t, offTenantID)
	require.NotZero(t, onTenantID)
	require.NotZero(t, otherTenantID)

	// 未知租户编号。
	ctx1, req1 := e.loginHappyReq(t, base, "authsvc-anyone", constants.DefaultUserPassword)
	req1.TenantCode = trans.Ptr("AUTHSVC_TENANT_NO_SUCH")
	resp, err := e.svc.Login(ctx1, req1)
	require.Error(t, err)
	require.True(t, authenticationV1.IsBadRequest(err), "未知租户编号应 400")
	require.Nil(t, resp)

	// 非启用租户。
	ctx2, req2 := e.loginHappyReq(t, base, "authsvc-anyone", constants.DefaultUserPassword)
	req2.TenantCode = trans.Ptr(authSvcTenantCodeOff)
	resp, err = e.svc.Login(ctx2, req2)
	require.Error(t, err)
	require.True(t, authenticationV1.IsBadRequest(err), "停用租户应 400 且文案与未知租户一致")
	require.Nil(t, resp)

	// 凭证租户（=解析出的租户）与用户行租户不一致。
	e.seedCredential(t, trans.Ptr(onTenantID), 7601, "authsvc-mismatch-user", authenticationV1.UserCredential_ENABLED)
	e.seedStubUser(7601, trans.Ptr(otherTenantID), "authsvc-mismatch-user", identityV1.User_NORMAL, nil)
	ctx3, req3 := e.loginHappyReq(t, base, "authsvc-mismatch-user", constants.DefaultUserPassword)
	req3.TenantCode = trans.Ptr(authSvcTenantCode)
	resp, err = e.svc.Login(ctx3, req3)
	require.Error(t, err)
	require.True(t, authenticationV1.IsBadRequest(err), "凭证/用户租户不一致应 400 invalid tenant")
	require.Nil(t, resp)
}

// TestAuthSvcSqlite_LoginPolicyGate 验证登录策略闸门两段：
// 全局 DEVICE 黑名单（target_id 为空，密码校验前拦截）与用户定向 TIME 黑名单
// （target_id=userId，凭证校验通过后拦截）均 403。
func TestAuthSvcSqlite_LoginPolicyGate(t *testing.T) {
	e := newAuthenticationServiceForTest(t)
	base := context.Background()

	// 全局 DEVICE 黑名单。
	require.NoError(t, e.svc.loginPolicyRepo.Create(e.ctx, &authenticationV1.CreateLoginPolicyRequest{
		Data: &authenticationV1.LoginPolicy{
			Type:   authenticationV1.LoginPolicy_BLACKLIST.Enum(),
			Method: authenticationV1.LoginPolicy_DEVICE.Enum(),
			Value:  trans.Ptr("authsvc-bad-device"),
			Reason: trans.Ptr("authsvc test device blacklist"),
		},
	}))
	ctx1, req1 := e.loginHappyReq(t, base, authSvcTestUsername, constants.DefaultUserPassword)
	req1.DeviceId = trans.Ptr("authsvc-bad-device")
	resp, err := e.svc.Login(ctx1, req1)
	require.Error(t, err)
	require.True(t, authenticationV1.IsForbidden(err), "全局 DEVICE 黑名单应 403")
	require.Nil(t, resp)

	// 不命中设备号的同一策略：放行到后续凭证校验（未播种凭证 → INVALID_PASSWORD）。
	ctx2, req2 := e.loginHappyReq(t, base, authSvcTestUsername, constants.DefaultUserPassword)
	req2.DeviceId = trans.Ptr("authsvc-good-device")
	resp, err = e.svc.Login(ctx2, req2)
	require.Error(t, err)
	require.True(t, authenticationV1.IsInvalidPassword(err), "未命中策略但凭证未播种，应归一化 INVALID_PASSWORD")
	require.Nil(t, resp)

	// 用户定向 TIME 黑名单（任意时刻窗口全覆盖）。
	permID := e.seedBackendAccessPermission(t)
	roleID := e.seedRole(t, nil, permissionV1.Role_SYSTEM, constants.PlatformAdminRoleCode, []uint32{permID}, nil, nil)
	e.seedCredential(t, nil, authSvcTestUserID, authSvcTestUsername, authenticationV1.UserCredential_ENABLED)
	e.seedStubUser(authSvcTestUserID, nil, authSvcTestUsername, identityV1.User_NORMAL, []uint32{roleID})
	require.NoError(t, e.svc.loginPolicyRepo.Create(e.ctx, &authenticationV1.CreateLoginPolicyRequest{
		Data: &authenticationV1.LoginPolicy{
			TargetId: trans.Ptr(authSvcTestUserID),
			Type:     authenticationV1.LoginPolicy_BLACKLIST.Enum(),
			Method:   authenticationV1.LoginPolicy_TIME.Enum(),
			Value:    trans.Ptr("00:00-23:59"),
			Reason:   trans.Ptr("authsvc test time blacklist"),
		},
	}))
	ctx3, req3 := e.loginHappyReq(t, base, authSvcTestUsername, constants.DefaultUserPassword)
	resp, err = e.svc.Login(ctx3, req3)
	require.Error(t, err)
	require.True(t, authenticationV1.IsForbidden(err), "用户定向 TIME 黑名单应 403")
	require.Nil(t, resp)
}

// TestAuthSvcSqlite_LoginCredentialFailures 验证凭证校验的防枚举归一：
// 凭证不存在 / 密码错误 / email 标识符（GetUsername 为空）/ 凭证停用 / 用户行缺失，
// 一律对外归一为 INVALID_PASSWORD（真实原因仅留服务端日志）。
func TestAuthSvcSqlite_LoginCredentialFailures(t *testing.T) {
	e := newAuthenticationServiceForTest(t)
	base := context.Background()

	// 凭证不存在。
	ctx1, req1 := e.loginHappyReq(t, base, "authsvc-ghost", constants.DefaultUserPassword)
	resp, err := e.svc.Login(ctx1, req1)
	require.Error(t, err)
	require.True(t, authenticationV1.IsInvalidPassword(err))
	require.Nil(t, resp)

	// 凭证存在但密码错误。
	permID := e.seedBackendAccessPermission(t)
	roleID := e.seedRole(t, nil, permissionV1.Role_SYSTEM, constants.PlatformAdminRoleCode, []uint32{permID}, nil, nil)
	e.seedCredential(t, nil, authSvcTestUserID, authSvcTestUsername, authenticationV1.UserCredential_ENABLED)
	e.seedStubUser(authSvcTestUserID, nil, authSvcTestUsername, identityV1.User_NORMAL, []uint32{roleID})
	ctx2, req2 := e.loginHappyReq(t, base, authSvcTestUsername, "WrongP@ss9999")
	resp, err = e.svc.Login(ctx2, req2)
	require.Error(t, err)
	require.True(t, authenticationV1.IsInvalidPassword(err), "错误密码应归一化为 INVALID_PASSWORD")
	require.Nil(t, resp)

	// email 标识符：doGrantTypePassword 只读 Username 维度，email oneof 下
	// GetUsername() 为空串 → 凭证查询必然落空。
	ctx3, req3 := e.loginHappyReq(t, base, "", constants.DefaultUserPassword)
	req3.Identifier = &authenticationV1.LoginRequest_Email{Email: "someone@example.com"}
	resp, err = e.svc.Login(ctx3, req3)
	require.Error(t, err)
	require.True(t, authenticationV1.IsInvalidPassword(err))
	require.Nil(t, resp)

	// 凭证停用（ENABLED 之外的状态按不存在处理）。
	e.seedCredential(t, nil, 7602, "authsvc-disabled-cred", authenticationV1.UserCredential_DISABLED)
	e.seedStubUser(7602, nil, "authsvc-disabled-cred", identityV1.User_NORMAL, []uint32{roleID})
	ctx4, req4 := e.loginHappyReq(t, base, "authsvc-disabled-cred", constants.DefaultUserPassword)
	resp, err = e.svc.Login(ctx4, req4)
	require.Error(t, err)
	require.True(t, authenticationV1.IsInvalidPassword(err), "停用凭证应与不存在同文案")
	require.Nil(t, resp)

	// 凭证校验通过但用户行缺失（stub 不回放该用户）→ 错误原样透传。
	e.seedCredential(t, nil, 7603, "authsvc-no-user-row", authenticationV1.UserCredential_ENABLED)
	ctx5, req5 := e.loginHappyReq(t, base, "authsvc-no-user-row", constants.DefaultUserPassword)
	resp, err = e.svc.Login(ctx5, req5)
	require.Error(t, err, "用户行缺失应报错而非放行")
	require.Nil(t, resp)
}

// TestAuthSvcSqlite_LoginFullPlatformChain 验证平台上下文的完整登录链路：
// 错误密码一次（限流计数=1）→ 正确密码登录成功（token 签发、限流清零、
// 最后登录记录经 userRepo.Update 落档、会话元数据入库）→ 签发的 access token 经
// ValidateToken 解出含角色码/平台管理员标志/[ALL] 数据范围的载荷。
func TestAuthSvcSqlite_LoginFullPlatformChain(t *testing.T) {
	e := newAuthenticationServiceForTest(t)
	base := context.Background()

	permID := e.seedBackendAccessPermission(t)
	roleID := e.seedRole(t, nil, permissionV1.Role_SYSTEM, constants.PlatformAdminRoleCode, []uint32{permID}, nil, nil)
	e.seedCredential(t, nil, authSvcTestUserID, authSvcTestUsername, authenticationV1.UserCredential_ENABLED)
	e.seedStubUser(authSvcTestUserID, nil, authSvcTestUsername, identityV1.User_NORMAL, []uint32{roleID})

	// 先来一次错误密码：失败计数自增到 1。
	// 注：本测试的假传输上下文 RemoteAddr 为空，ClientIPFromContext 取得空 IP，
	// LoginRateLimiter.failKeys 会跳过空 IP 键，仅用户名维度计数（与生产行为一致）。
	badCtx, badReq := e.loginHappyReq(t, base, authSvcTestUsername, "WrongP@ss9999")
	_, err := e.svc.Login(badCtx, badReq)
	require.Error(t, err)
	usrCnt, usrErr := e.mr.Get("gowind:login:fail:user:" + authSvcTestUsername)
	require.Equal(t, "1", usrCnt, "用户名维度失败计数应为 1")
	require.NoError(t, usrErr)

	// 正确密码：登录成功。
	okCtx, okReq := e.loginHappyReq(t, base, authSvcTestUsername, constants.DefaultUserPassword)
	resp, err := e.svc.Login(okCtx, okReq)
	require.NoError(t, err, "完整授权链应登录成功")
	require.NotEmpty(t, resp.GetAccessToken(), "应签发 access token")
	require.Positive(t, resp.GetExpiresIn(), "过期时间应为正数秒")
	require.Equal(t, authenticationV1.TokenType_bearer, resp.GetTokenType())
	require.Empty(t, resp.GetMfaOperationId(), "未绑定 MFA 因子不应返回 MFA 操作标识")

	// 登录成功后限流计数清零。
	_, usrErr = e.mr.Get("gowind:login:fail:user:" + authSvcTestUsername)
	require.Error(t, usrErr, "用户名维度计数应被清零（键不存在）")

	// 最后登录记录：userRepo.Update 被调用且 fieldmask 限定 last_login_* 两字段。
	require.Len(t, e.stub.updates, 1, "recordUserLastLogin 应恰好发起一次用户更新")
	upd := e.stub.updates[0]
	require.Equal(t, authSvcTestUserID, upd.GetId())
	require.NotNil(t, upd.UpdateMask)
	require.ElementsMatch(t, []string{"last_login_at", "last_login_ip"}, upd.UpdateMask.GetPaths(),
		"最后登录记录的更新应被 fieldmask 限定在 last_login_* 两字段")

	// 签发的 access token 可被 ValidateToken 验证，且载荷含授权链注入的字段。
	valResp, err := e.svc.ValidateToken(base, &authenticationV1.ValidateTokenRequest{
		Token:         resp.GetAccessToken(),
		ClientType:    authenticationV1.ClientType_admin,
		TokenCategory: authenticationV1.TokenCategory_ACCESS,
	})
	require.NoError(t, err)
	require.True(t, valResp.GetIsValid())
	payload := valResp.GetPayload()
	require.Equal(t, authSvcTestUserID, payload.GetUserId())
	require.Equal(t, []string{constants.PlatformAdminRoleCode}, payload.GetRoles(),
		"载荷应携带授权链回填的角色码")
	require.True(t, payload.GetIsPlatformAdmin(), "平台管理员角色码应置位平台管理员标志")
	require.Equal(t, []identityV1.DataScope{identityV1.DataScope_ALL}, payload.GetDataScopes(),
		"平台上下文数据范围应聚合为 [ALL]")
	require.Equal(t, identityV1.DataScope_ALL, payload.GetDataScope(), "单值镜像字段应为 ALL")
}

// TestAuthSvcSqlite_LoginRateLimiterLockout 验证登录限流的锁定窗口：
// 连续 5 次密码错误后，第 6 次在预检阶段即被 400 拒绝（先于验证码与凭证校验），
// 双维度计数均达到阈值。
func TestAuthSvcSqlite_LoginRateLimiterLockout(t *testing.T) {
	e := newAuthenticationServiceForTest(t)
	base := context.Background()

	permID := e.seedBackendAccessPermission(t)
	roleID := e.seedRole(t, nil, permissionV1.Role_SYSTEM, constants.PlatformAdminRoleCode, []uint32{permID}, nil, nil)
	e.seedCredential(t, nil, authSvcTestUserID, authSvcTestUsername, authenticationV1.UserCredential_ENABLED)
	e.seedStubUser(authSvcTestUserID, nil, authSvcTestUsername, identityV1.User_NORMAL, []uint32{roleID})

	for i := 0; i < 5; i++ {
		ctx, req := e.loginHappyReq(t, base, authSvcTestUsername, "WrongP@ss9999")
		resp, err := e.svc.Login(ctx, req)
		require.Error(t, err, "第 %d 次错误密码应报错", i+1)
		require.True(t, authenticationV1.IsInvalidPassword(err), "第 %d 次错误密码应归一化 INVALID_PASSWORD", i+1)
		require.Nil(t, resp)
	}
	usrCnt, usrErr := e.mr.Get("gowind:login:fail:user:" + authSvcTestUsername)
	require.Equal(t, "5", usrCnt, "五次失败后用户名维度计数应为阈值 5")
	require.NoError(t, usrErr)

	// 第 6 次：即使密码与验证码都正确，也在限流预检阶段被拒。
	ctx, req := e.loginHappyReq(t, base, authSvcTestUsername, constants.DefaultUserPassword)
	resp, err := e.svc.Login(ctx, req)
	require.Error(t, err)
	require.True(t, authenticationV1.IsBadRequest(err), "锁定窗口内应 400")
	require.Contains(t, err.Error(), "too many login failures")
	require.Nil(t, resp)
}

// TestAuthSvcSqlite_LoginMfaGate 验证 MFA 闸门：
// 绑定 ENABLED TOTP 因子的用户密码校验通过后不签发 token，
// 改为返回 MfaOperationId（access token 为空）。
func TestAuthSvcSqlite_LoginMfaGate(t *testing.T) {
	e := newAuthenticationServiceForTest(t)
	base := context.Background()

	permID := e.seedBackendAccessPermission(t)
	roleID := e.seedRole(t, nil, permissionV1.Role_SYSTEM, constants.PlatformAdminRoleCode, []uint32{permID}, nil, nil)
	e.seedCredential(t, nil, 7301, "authsvc-mfa-user", authenticationV1.UserCredential_ENABLED)
	e.seedStubUser(7301, nil, "authsvc-mfa-user", identityV1.User_NORMAL, []uint32{roleID})

	// 未绑定因子：正常签发。
	ctx1, req1 := e.loginHappyReq(t, base, "authsvc-mfa-user", constants.DefaultUserPassword)
	resp, err := e.svc.Login(ctx1, req1)
	require.NoError(t, err)
	require.NotEmpty(t, resp.GetAccessToken(), "未绑定 MFA 因子应正常签发")
	require.Empty(t, resp.GetMfaOperationId())

	// 绑定 ENABLED TOTP 因子：进入 MFA 挑战分支。
	_, err = e.svc.mfaFactorRepo.CreateTotpFactor(e.ctx, 0, 7301, "JBSWY3DPEHPK3PXP", "authsvc-totp")
	require.NoError(t, err)
	ctx2, req2 := e.loginHappyReq(t, base, "authsvc-mfa-user", constants.DefaultUserPassword)
	resp, err = e.svc.Login(ctx2, req2)
	require.NoError(t, err, "MFA 闸门分支应返回挑战而非错误")
	require.Empty(t, resp.GetAccessToken(), "MFA 分支不应签发 access token")
	require.NotEmpty(t, resp.GetMfaOperationId(), "MFA 分支应返回操作标识")
}

// TestAuthSvcSqlite_TenantLoginChain 验证租户上下文登录链路与令牌聚合：
// 租户角色（带 sys:access_backend 权限 + SELF 数据范围 + User 资源字段权限）经
// tenant_code 解析登录成功，载荷含租户 ID、租户管理员标志、[SELF] 数据范围
// （含单值镜像）与按"资源.字段"聚合的隐藏字段集。
func TestAuthSvcSqlite_TenantLoginChain(t *testing.T) {
	e := newAuthenticationServiceForTest(t)
	base := context.Background()

	tenantID := e.seedTenant(t, "AuthSvc 租户链租户", authSvcTenantCode, identityV1.Tenant_ON)
	permID := e.seedBackendAccessPermission(t)
	tenantRoleID := e.seedRole(
		t,
		trans.Ptr(tenantID),
		permissionV1.Role_TENANT,
		constants.TenantAdminRoleCode,
		[]uint32{permID},
		identityV1.DataScope_SELF.Enum(),
		[]*permissionV1.RoleFieldPermission{
			{Resource: "User", HiddenFields: []string{"email", "mobile"}},
		},
	)
	e.seedCredential(t, trans.Ptr(tenantID), 7401, "authsvc-tenant-user", authenticationV1.UserCredential_ENABLED)
	e.seedStubUser(7401, trans.Ptr(tenantID), "authsvc-tenant-user", identityV1.User_NORMAL, []uint32{tenantRoleID})

	ctx, req := e.loginHappyReq(t, base, "authsvc-tenant-user", constants.DefaultUserPassword)
	req.TenantCode = trans.Ptr(authSvcTenantCode)
	resp, err := e.svc.Login(ctx, req)
	require.NoError(t, err, "租户链路应登录成功")
	require.NotEmpty(t, resp.GetAccessToken())

	valResp, err := e.svc.ValidateToken(base, &authenticationV1.ValidateTokenRequest{
		Token:         resp.GetAccessToken(),
		ClientType:    authenticationV1.ClientType_admin,
		TokenCategory: authenticationV1.TokenCategory_ACCESS,
	})
	require.NoError(t, err)
	require.True(t, valResp.GetIsValid())
	payload := valResp.GetPayload()
	require.Equal(t, tenantID, payload.GetTenantId(), "载荷应携带租户 ID")
	require.Equal(t, []string{constants.TenantAdminRoleCode}, payload.GetRoles())
	require.True(t, payload.GetIsTenantAdmin(), "租户管理员角色码应置位租户管理员标志")
	require.False(t, payload.GetIsPlatformAdmin(), "租户管理员不应置位平台管理员标志")
	require.Equal(t, []identityV1.DataScope{identityV1.DataScope_SELF}, payload.GetDataScopes(),
		"租户角色 SELF 数据范围应聚合为 [SELF]")
	require.Equal(t, identityV1.DataScope_SELF, payload.GetDataScope(), "单值镜像字段应为 SELF")
	require.ElementsMatch(t, []string{"User.email", "User.mobile"}, payload.GetHiddenFields(),
		"字段权限应按'资源.字段'聚合进令牌")
}

// TestAuthSvcSqlite_AuthorizationDeniedBranches 验证授权链的拒绝分支：
// 无角色、角色无后台访问权限、用户非 NORMAL 状态，均 403。
func TestAuthSvcSqlite_AuthorizationDeniedBranches(t *testing.T) {
	e := newAuthenticationServiceForTest(t)
	base := context.Background()

	permID := e.seedBackendAccessPermission(t)
	roleID := e.seedRole(t, nil, permissionV1.Role_SYSTEM, constants.PlatformAdminRoleCode, []uint32{permID}, nil, nil)
	noPermRoleID := e.seedRole(t, nil, permissionV1.Role_SYSTEM, "AUTHSVC_ROLE_NO_PERM", nil, nil, nil)
	require.NotZero(t, noPermRoleID)
	e.seedCredential(t, nil, authSvcTestUserID, authSvcTestUsername, authenticationV1.UserCredential_ENABLED)
	e.seedCredential(t, nil, 7605, "authsvc-norole-user", authenticationV1.UserCredential_ENABLED)
	e.seedCredential(t, nil, 7606, "authsvc-noperm-user", authenticationV1.UserCredential_ENABLED)
	e.seedCredential(t, nil, 7607, "authsvc-frozen-user", authenticationV1.UserCredential_ENABLED)

	// 用户无任何角色。
	e.seedStubUser(7605, nil, "authsvc-norole-user", identityV1.User_NORMAL, nil)
	ctx1, req1 := e.loginHappyReq(t, base, "authsvc-norole-user", constants.DefaultUserPassword)
	resp, err := e.svc.Login(ctx1, req1)
	require.Error(t, err)
	require.True(t, authenticationV1.IsForbidden(err), "无角色应 403")
	require.Nil(t, resp)

	// 角色未挂后台访问权限。
	e.seedStubUser(7606, nil, "authsvc-noperm-user", identityV1.User_NORMAL, []uint32{noPermRoleID})
	ctx2, req2 := e.loginHappyReq(t, base, "authsvc-noperm-user", constants.DefaultUserPassword)
	resp, err = e.svc.Login(ctx2, req2)
	require.Error(t, err)
	require.True(t, authenticationV1.IsForbidden(err), "无 sys:access_backend 权限应 403")
	require.Nil(t, resp)

	// 用户状态非 NORMAL。
	e.seedStubUser(7607, nil, "authsvc-frozen-user", identityV1.User_LOCKED, []uint32{roleID})
	ctx3, req3 := e.loginHappyReq(t, base, "authsvc-frozen-user", constants.DefaultUserPassword)
	resp, err = e.svc.Login(ctx3, req3)
	require.Error(t, err)
	require.True(t, authenticationV1.IsForbidden(err), "非 NORMAL 用户应 403 user is disabled")
	require.Nil(t, resp)
}

// TestAuthSvcSqlite_DisabledTenantViaRefresh 验证刷新令牌链路的租户状态检查：
// 用户行租户指向停用租户时，授权阶段（OneToOne 分支内的租户状态检查）拒绝 403。
func TestAuthSvcSqlite_DisabledTenantViaRefresh(t *testing.T) {
	e := newAuthenticationServiceForTest(t)
	base := context.Background()

	offTenantID := e.seedTenant(t, "AuthSvc 刷新停用租户", authSvcTenantCodeOff, identityV1.Tenant_OFF)
	e.seedStubUser(7501, trans.Ptr(offTenantID), "authsvc-refresh-user", identityV1.User_NORMAL, nil)

	payload := &authenticationV1.UserTokenPayload{UserId: 7501, TenantId: trans.Ptr(offTenantID), Username: trans.Ptr("authsvc-refresh-user")}
	_, refreshToken, err := e.svc.authenticator.CreateUserToken(base, authenticationV1.ClientType_admin, payload)
	require.NoError(t, err)

	req := &authenticationV1.LoginRequest{GrantType: authenticationV1.GrantType_refresh_token}
	header := http.Header{}
	header.Set("Cookie", "refresh_token="+refreshToken)
	resp, err := e.svc.RefreshToken(headerCtx(base, header), req)
	require.Error(t, err)
	require.True(t, authenticationV1.IsForbidden(err), "停用租户的刷新应 403")
	require.Nil(t, resp)
}

// TestAuthSvcSqlite_LogoutAndRevocation 验证登出与令牌吊销：
// 登录签发的令牌对在 Logout 后立即失效（ValidateToken 返回 401 已吊销）；
// 无操作人上下文的 Logout 报 401。
func TestAuthSvcSqlite_LogoutAndRevocation(t *testing.T) {
	e := newAuthenticationServiceForTest(t)
	base := context.Background()

	permID := e.seedBackendAccessPermission(t)
	roleID := e.seedRole(t, nil, permissionV1.Role_SYSTEM, constants.PlatformAdminRoleCode, []uint32{permID}, nil, nil)
	e.seedCredential(t, nil, authSvcTestUserID, authSvcTestUsername, authenticationV1.UserCredential_ENABLED)
	e.seedStubUser(authSvcTestUserID, nil, authSvcTestUsername, identityV1.User_NORMAL, []uint32{roleID})

	ctx, req := e.loginHappyReq(t, base, authSvcTestUsername, constants.DefaultUserPassword)
	resp, err := e.svc.Login(ctx, req)
	require.NoError(t, err)
	accessToken := resp.GetAccessToken()
	require.NotEmpty(t, accessToken)

	valReq := &authenticationV1.ValidateTokenRequest{
		Token:         accessToken,
		ClientType:    authenticationV1.ClientType_admin,
		TokenCategory: authenticationV1.TokenCategory_ACCESS,
	}
	valResp, err := e.svc.ValidateToken(base, valReq)
	require.NoError(t, err)
	require.True(t, valResp.GetIsValid(), "登出前令牌应有效")

	// 无操作人上下文：401。
	_, err = e.svc.Logout(base, &emptypb.Empty{})
	require.Error(t, err)
	require.True(t, authenticationV1.IsUnauthorized(err), "无操作人的 Logout 应 401")

	// 有操作人上下文：吊销该用户 admin 客户端类型的全部令牌。
	opCtx := auth.NewContext(base, &authenticationV1.UserTokenPayload{UserId: authSvcTestUserID})
	_, err = e.svc.Logout(opCtx, &emptypb.Empty{})
	require.NoError(t, err, "Logout 应成功吊销并返回空响应")

	valResp, err = e.svc.ValidateToken(base, valReq)
	require.Error(t, err, "登出后令牌应失效")
	require.True(t, authenticationV1.IsUnauthorized(err), "已吊销令牌应 401")
	require.False(t, valResp.GetIsValid())
}

// TestAuthSvcSqlite_RefreshTokenFlow 验证刷新令牌轮换全链路：
// 缺 cookie / 错误 grant / 垃圾 token 均 401；合法 refresh token（HttpOnly cookie 注入）
// 轮换出新令牌对（旧会话元数据登录时间被继承），旧令牌对原子吊销不可复用。
func TestAuthSvcSqlite_RefreshTokenFlow(t *testing.T) {
	e := newAuthenticationServiceForTest(t)
	base := context.Background()

	// 无 cookie：401。
	resp, err := e.svc.RefreshToken(base, &authenticationV1.LoginRequest{GrantType: authenticationV1.GrantType_refresh_token})
	require.Error(t, err)
	require.True(t, authenticationV1.IsIncorrectRefreshToken(err), "缺 cookie 应 401")
	require.Nil(t, resp)

	// 非 refresh_token 授权类型：400。
	headerOnly := http.Header{}
	headerOnly.Set("Cookie", "refresh_token=whatever")
	resp, err = e.svc.RefreshToken(headerCtx(base, headerOnly), &authenticationV1.LoginRequest{GrantType: authenticationV1.GrantType_password})
	require.Error(t, err)
	require.True(t, authenticationV1.IsInvalidGrantType(err), "错误授权类型应 400")
	require.Nil(t, resp)

	// 垃圾 refresh token：验签失败 401。
	garbage := http.Header{}
	garbage.Set("Cookie", "refresh_token=not-a-jwt")
	resp, err = e.svc.RefreshToken(headerCtx(base, garbage), &authenticationV1.LoginRequest{GrantType: authenticationV1.GrantType_refresh_token})
	require.Error(t, err)
	require.True(t, authenticationV1.IsIncorrectRefreshToken(err), "垃圾 refresh token 应 401")
	require.Nil(t, resp)

	// 合法轮换：先经完整登录链签发旧令牌对，并记录旧会话元数据。
	permID := e.seedBackendAccessPermission(t)
	roleID := e.seedRole(t, nil, permissionV1.Role_SYSTEM, constants.PlatformAdminRoleCode, []uint32{permID}, nil, nil)
	e.seedCredential(t, nil, authSvcTestUserID, authSvcTestUsername, authenticationV1.UserCredential_ENABLED)
	e.seedStubUser(authSvcTestUserID, nil, authSvcTestUsername, identityV1.User_NORMAL, []uint32{roleID})

	payload := &authenticationV1.UserTokenPayload{UserId: authSvcTestUserID, Username: trans.Ptr(authSvcTestUsername)}
	oldAccess, oldRefresh, err := e.svc.authenticator.CreateUserToken(base, authenticationV1.ClientType_admin, payload)
	require.NoError(t, err)
	require.NotEmpty(t, oldAccess)
	require.NotEmpty(t, oldRefresh)
	// 会话元数据走 recordSessionMeta（登录链同款入口）为旧 jti 落档，
	// 使轮换路径命中"旧会话元数据继承登录时间"分支。
	recordSessionMeta(base, e.svc.log, e.svc.authenticator, authenticationV1.ClientType_admin, payload)

	rotHeader := http.Header{}
	rotHeader.Set("Cookie", "refresh_token="+oldRefresh)
	rotResp, err := e.svc.RefreshToken(headerCtx(base, rotHeader), &authenticationV1.LoginRequest{GrantType: authenticationV1.GrantType_refresh_token})
	require.NoError(t, err, "合法 refresh token 应轮换成功")
	require.NotEmpty(t, rotResp.GetAccessToken(), "轮换应签发新 access token")
	require.Positive(t, rotResp.GetExpiresIn())

	// 新 access token 可验证且载荷含授权链字段。
	valResp, err := e.svc.ValidateToken(base, &authenticationV1.ValidateTokenRequest{
		Token:         rotResp.GetAccessToken(),
		ClientType:    authenticationV1.ClientType_admin,
		TokenCategory: authenticationV1.TokenCategory_ACCESS,
	})
	require.NoError(t, err)
	require.True(t, valResp.GetIsValid())
	require.Equal(t, []string{constants.PlatformAdminRoleCode}, valResp.GetPayload().GetRoles())

	// 旧令牌对已原子吊销：旧 access token 与旧 refresh token 均不可再用。
	oldValResp, err := e.svc.ValidateToken(base, &authenticationV1.ValidateTokenRequest{
		Token:         oldAccess,
		ClientType:    authenticationV1.ClientType_admin,
		TokenCategory: authenticationV1.TokenCategory_ACCESS,
	})
	require.Error(t, err, "旧 access token 应随轮换吊销")
	require.True(t, authenticationV1.IsUnauthorized(err))
	require.False(t, oldValResp.GetIsValid())

	reuseResp, err := e.svc.RefreshToken(headerCtx(base, rotHeader), &authenticationV1.LoginRequest{GrantType: authenticationV1.GrantType_refresh_token})
	require.Error(t, err, "旧 refresh token 不可复用")
	require.True(t, authenticationV1.IsIncorrectRefreshToken(err))
	require.Nil(t, reuseResp)
}

// TestAuthSvcSqlite_ValidateTokenBranches 验证 ValidateToken 的入参守卫与
// REFRESH 类别校验：nil 请求 / 空 token / 非法类别 400，合法 refresh token 校验通过。
func TestAuthSvcSqlite_ValidateTokenBranches(t *testing.T) {
	e := newAuthenticationServiceForTest(t)
	base := context.Background()

	resp, err := e.svc.ValidateToken(base, nil)
	require.Error(t, err)
	require.True(t, authenticationV1.IsBadRequest(err), "nil 请求应 400")
	require.Nil(t, resp)

	resp, err = e.svc.ValidateToken(base, &authenticationV1.ValidateTokenRequest{
		ClientType:    authenticationV1.ClientType_admin,
		TokenCategory: authenticationV1.TokenCategory_ACCESS,
	})
	require.Error(t, err)
	require.True(t, authenticationV1.IsBadRequest(err), "空 token 应 400")
	require.Nil(t, resp)

	resp, err = e.svc.ValidateToken(base, &authenticationV1.ValidateTokenRequest{
		Token:         "some-token",
		ClientType:    authenticationV1.ClientType_admin,
		TokenCategory: authenticationV1.TokenCategory(0),
	})
	require.Error(t, err)
	require.True(t, authenticationV1.IsBadRequest(err), "非法令牌类别应 400")
	require.Nil(t, resp)

	// 合法 refresh token 的 REFRESH 类别校验（不轮换）。
	payload := &authenticationV1.UserTokenPayload{UserId: authSvcTestUserID}
	_, refreshToken, err := e.svc.authenticator.CreateUserToken(base, authenticationV1.ClientType_admin, payload)
	require.NoError(t, err)
	resp, err = e.svc.ValidateToken(base, &authenticationV1.ValidateTokenRequest{
		Token:         refreshToken,
		ClientType:    authenticationV1.ClientType_admin,
		TokenCategory: authenticationV1.TokenCategory_REFRESH,
	})
	require.NoError(t, err)
	require.True(t, resp.GetIsValid(), "缓存中的 refresh token 校验应为真")
}

// TestAuthSvcSqlite_WhoAmI 验证 WhoAmI 的操作人回显与无操作人拒绝。
func TestAuthSvcSqlite_WhoAmI(t *testing.T) {
	e := newAuthenticationServiceForTest(t)
	base := context.Background()

	resp, err := e.svc.WhoAmI(base, &emptypb.Empty{})
	require.Error(t, err)
	require.True(t, authenticationV1.IsUnauthorized(err), "无操作人上下文应 401")
	require.Nil(t, resp)

	opCtx := auth.NewContext(base, &authenticationV1.UserTokenPayload{UserId: 4242, Username: trans.Ptr("authsvc-operator")})
	resp, err = e.svc.WhoAmI(opCtx, &emptypb.Empty{})
	require.NoError(t, err)
	require.Equal(t, uint32(4242), resp.GetUserId())
	require.Equal(t, "authsvc-operator", resp.GetUsername())
}

// TestAuthSvcSqlite_GenerateAndVerifyCaptcha 验证验证码服务闭环：
// GenerateCaptcha 落库（miniredis 中可按约定键直读答案），VerifyCaptcha
// 正确答案通过、错误答案与已消费（verify-and-delete）的答案拒绝。
func TestAuthSvcSqlite_GenerateAndVerifyCaptcha(t *testing.T) {
	e := newAuthenticationServiceForTest(t)
	base := context.Background()

	genResp, err := e.svc.GenerateCaptcha(base, &emptypb.Empty{})
	require.NoError(t, err)
	require.NotEmpty(t, genResp.GetCaptchaId(), "应返回验证码 ID")
	require.NotEmpty(t, genResp.GetImageBase64(), "应返回验证码图片")

	// 生产语义：答案按 <prefix>:<id> 存入 Redis（AGENTS.md 所述可直接读取）。
	answer, getErr := e.mr.Get(authSvcTestKeyPrefix + ":" + genResp.GetCaptchaId())
	require.NoError(t, getErr, "验证码答案应已落 Redis")

	okResp, err := e.svc.VerifyCaptcha(base, &authenticationV1.VerifyCaptchaRequest{
		CaptchaId: genResp.GetCaptchaId(),
		UserInput: answer,
	})
	require.NoError(t, err)
	require.True(t, okResp.GetValid(), "正确答案应通过校验")

	// verify-and-delete：消费后同答案再次校验为假。
	again, err := e.svc.VerifyCaptcha(base, &authenticationV1.VerifyCaptchaRequest{
		CaptchaId: genResp.GetCaptchaId(),
		UserInput: answer,
	})
	require.NoError(t, err)
	require.False(t, again.GetValid(), "已消费的验证码应失效")

	// 错误答案。
	genResp2, err := e.svc.GenerateCaptcha(base, &emptypb.Empty{})
	require.NoError(t, err)
	wrong, err := e.svc.VerifyCaptcha(base, &authenticationV1.VerifyCaptchaRequest{
		CaptchaId: genResp2.GetCaptchaId(),
		UserInput: "WRONG01",
	})
	require.NoError(t, err)
	require.False(t, wrong.GetValid(), "错误答案应不通过")
}

// TestAuthSvcSqlite_NormalizeLoginVerifyError 验证登录凭证错误的防枚举归一：
// USER_NOT_FOUND / USER_FREEZE / INVALID_PASSWORD 统一归一为 INVALID_PASSWORD，
// 其他错误原样透传。
func TestAuthSvcSqlite_NormalizeLoginVerifyError(t *testing.T) {
	specials := []error{
		authenticationV1.ErrorUserNotFound("user not found"),
		authenticationV1.ErrorUserFreeze("user frozen"),
		authenticationV1.ErrorInvalidPassword("bad password"),
	}
	for _, in := range specials {
		out := normalizeLoginVerifyError(in)
		require.Error(t, out)
		require.True(t, authenticationV1.IsInvalidPassword(out), "应归一化为 INVALID_PASSWORD")
	}

	other := fmt.Errorf("some other error")
	require.Same(t, other, normalizeLoginVerifyError(other), "其他错误应原样透传")
}

// TestAuthSvcSqlite_ContainsPermission 验证权限码列表包含性判断的纯逻辑。
func TestAuthSvcSqlite_ContainsPermission(t *testing.T) {
	require.True(t, containsPermission([]string{"a", "sys:access_backend", "b"}, "sys:access_backend"))
	require.False(t, containsPermission([]string{"a", "b"}, "sys:access_backend"))
	require.False(t, containsPermission(nil, "sys:access_backend"))
	require.False(t, containsPermission([]string{"a"}, ""))
}

// TestAuthSvcSqlite_FillAdminFlags 验证按角色码填充平台/租户管理员标志。
func TestAuthSvcSqlite_FillAdminFlags(t *testing.T) {
	payload := &authenticationV1.UserTokenPayload{}
	fillAdminFlags(payload, []string{constants.PlatformAdminRoleCode, constants.TenantAdminRoleCode})
	require.True(t, payload.GetIsPlatformAdmin(), "平台管理员角色码应置位平台管理员标志")
	require.True(t, payload.GetIsTenantAdmin(), "租户管理员角色码应置位租户管理员标志")

	plain := &authenticationV1.UserTokenPayload{}
	fillAdminFlags(plain, []string{"some_role", "another_role"})
	require.False(t, plain.GetIsPlatformAdmin())
	require.False(t, plain.GetIsTenantAdmin())
}

// TestAuthSvcSqlite_CookieHelperGuards 验证 cookie 助手与记录函数的无传输守卫段：
// 无传输上下文时 setRefreshCookies / clearRefreshCookies 静默返回；
// recordUserLastLogin 对 userID=0 / nil userRepo 直接跳过；
// recordSessionMetaAt 对 nil authenticator / nil payload 直接跳过。
func TestAuthSvcSqlite_CookieHelperGuards(t *testing.T) {
	e := newAuthenticationServiceForTest(t)
	base := context.Background()
	logHelper := bLogger.NewHelper(bLogger.NopLogger())

	require.NotPanics(t, func() {
		setRefreshCookies(base, "rt-value", 60)
		clearRefreshCookies(base)
	}, "无传输上下文时 cookie 助手应静默返回")

	// userID=0：不发起更新。
	require.NotPanics(t, func() {
		recordUserLastLogin(base, logHelper, e.stub, 0, "1.2.3.4")
	})
	require.Empty(t, e.stub.updates, "userID=0 不应发起用户更新")

	// nil userRepo：直接跳过。
	require.NotPanics(t, func() {
		recordUserLastLogin(base, logHelper, nil, 5, "1.2.3.4")
	})

	// nil authenticator / nil payload：直接跳过。
	require.NotPanics(t, func() {
		recordSessionMetaAt(base, logHelper, nil, authenticationV1.ClientType_admin, nil, 0)
		recordSessionMetaAt(base, logHelper, nil, authenticationV1.ClientType_admin, &authenticationV1.UserTokenPayload{}, 0)
	})
}
