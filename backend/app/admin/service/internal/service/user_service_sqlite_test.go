// UserService 的 SQLite 内存库 + miniredis 集成测试（白盒，包内测试）。
//
// 装配范式（对齐 authentication_service_sqlite_test.go）：
//   - ent 仓储（user_credential(+config) / role / position / org_unit / tenant）
//     走 data 包 repo_testkit 构造器（enttest SQLite 内存库）；
//   - authenticator（Update / EditUserPassword 密码重置后的吊销链路）以
//     miniredis 假 client + 测试 HS256 密钥经 data.NewAuthenticatorForTest 构造；
//   - userRepo 为接口（data.UserRepo），用本文件桩回放列表/计数/按 id 用户、
//     并记录 Create / Update / Delete / AssignUserRole 调用（嵌入接口惯用法，
//     未覆写方法一旦被调用即 panic，测试即失败）。
//
// 覆盖目标：user_service.go 的 List / Get（关联实体名称富集：租户 / 角色(含单值与
// 列表两轨道) / 组织单元 / 职位，nil 项跳过）、Count / UserExists（委托）、
// Create（入参守卫 / 操作人与租户注入 / 角色存在性与类型校验(经 roleRepo 真实
// 分页过滤) / 默认密码落凭证(经 userCredentialRepo 真实 bcrypt+复杂度校验)）、
// Update（角色校验 / 跨租户重置密码拒绝 / 平台管理员与租户内重置(真实
// ResetCredential) / 重置后吊销 / UpdatedBy+mask 注入）、Delete（默认管理员、
// 自删守卫与正常删除）、EditUserPassword（跨租户拒绝与同租户/平台重置）、
// init / createDefaultUser / createDefaultUserCredentials（空库播种、凭证自愈补种、
// overrideUserID 绑定）。
//
// 跳过段（②外部链路，详见汇报）：
//   - OneToMany 分支（constants.DefaultUserTenantRelationType 为编译期常量
//     UserTenantRelationOneToOne，membershipRepo 装配分支为死代码，membershipRepo 置 nil）；
//   - fetchRelationInfo 各仓储的 DB 故障分支（不可注入）。
package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/password"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/enttest"

	conf "github.com/tx7do/kratos-bootstrap/api/gen/go/conf/v1"
	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"

	"go-wind-admin/pkg/constants"
	"go-wind-admin/pkg/middleware/auth"
)

// userServiceUserRepoStub 是 UserService 测试专用的 data.UserRepo 桩：
// 嵌入接口获得默认方法集（未覆写方法一旦被调用即 nil 接口 panic，测试即失败）。
// 只覆写服务层会用到的八个方法：Get（按 id 回放）、List（回放固定列表）、
// Count（回放固定计数）、Create（记录并回放占位用户）、Update / Delete /
// AssignUserRole（记录调用）、UserExists（回放固定响应）。
// 桩数据与实体表无关；真实落库的只有凭证表（经真实 userCredentialRepo）。
type userServiceUserRepoStub struct {
	data.UserRepo
	usersByID     map[uint32]*identityV1.User
	listUsers     []*identityV1.User
	count         int
	creates       []*identityV1.CreateUserRequest
	updates       []*identityV1.UpdateUserRequest
	deletes       []uint32
	assignedRoles []*permissionV1.UserRole
}

func (s *userServiceUserRepoStub) Get(_ context.Context, req *identityV1.GetUserRequest) (*identityV1.User, error) {
	if u, ok := s.usersByID[req.GetId()]; ok {
		return u, nil
	}
	return nil, fmt.Errorf("user %d not found (userService stub)", req.GetId())
}

func (s *userServiceUserRepoStub) List(_ context.Context, _ *paginationV1.PagingRequest) (*identityV1.ListUserResponse, error) {
	return &identityV1.ListUserResponse{Items: s.listUsers, Total: uint64(len(s.listUsers))}, nil
}

func (s *userServiceUserRepoStub) Count(_ context.Context, _ *paginationV1.PagingRequest) (int, error) {
	return s.count, nil
}

func (s *userServiceUserRepoStub) Create(_ context.Context, req *identityV1.CreateUserRequest) (*identityV1.User, error) {
	s.creates = append(s.creates, req)
	// 直接透传请求体内的租户指针（含 nil=平台），保持 Create 返回行的租户语义。
	return &identityV1.User{Id: trans.Ptr(uint32(66001)), TenantId: req.Data.TenantId}, nil
}

func (s *userServiceUserRepoStub) Update(_ context.Context, req *identityV1.UpdateUserRequest) error {
	s.updates = append(s.updates, req)
	return nil
}

func (s *userServiceUserRepoStub) Delete(_ context.Context, req *identityV1.DeleteUserRequest) error {
	s.deletes = append(s.deletes, req.GetId())
	return nil
}

func (s *userServiceUserRepoStub) UserExists(_ context.Context, _ *identityV1.UserExistsRequest) (*identityV1.UserExistsResponse, error) {
	return &identityV1.UserExistsResponse{Exist: false}, nil
}

func (s *userServiceUserRepoStub) AssignUserRole(_ context.Context, data *permissionV1.UserRole) error {
	s.assignedRoles = append(s.assignedRoles, data)
	return nil
}

// userServiceEnv 聚合 UserService 测试装配（每测试全新 enttest client + 全新 miniredis）。
type userServiceEnv struct {
	svc  *UserService
	stub *userServiceUserRepoStub
	ctx  context.Context // SystemContext（仅用于种子数据写入）
}

// newUserServiceForTest 白盒复刻 NewUserService 的字段初始化：log 换 NopLogger，
// ent 仓储用 repo_testkit 构造器，authenticator 经 miniredis + 测试密钥构造，
// userRepo 用本文件桩；membershipRepo 仅 OneToMany 分支（编译期死代码）使用，置 nil。
// 注意：不经 NewUserService 构造，svc.init()（默认数据播种）仅在专门测试中显式调用。
func newUserServiceForTest(t *testing.T) *userServiceEnv {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	passwordCrypto := password.NewBCryptCrypto()
	configRepo := data.NewConfigRepoForTest(entClient, rdb)
	userCredentialRepo := data.NewUserCredentialRepoForTest(entClient, passwordCrypto, configRepo)
	userTokenCache := data.NewUserTokenCacheForTest(rdb)
	jwtCfg := &conf.Authentication_Jwt{Method: "HS256", Key: authSvcTestJWTKey}
	authenticator, err := data.NewAuthenticatorForTest(jwtCfg, userTokenCache)
	require.NoError(t, err)

	stub := &userServiceUserRepoStub{usersByID: make(map[uint32]*identityV1.User)}

	svc := &UserService{
		log:                bLogger.NewHelper(bLogger.NopLogger()),
		userRepo:           stub,
		userCredentialRepo: userCredentialRepo,
		roleRepo:           data.NewRoleRepoForTest(entClient),
		positionRepo:       data.NewPositionRepoForTest(entClient),
		orgUnitRepo:        data.NewOrgUnitRepoForTest(entClient),
		tenantRepo:         data.NewTenantRepoForTest(entClient),
		membershipRepo:     nil,
		authenticator:      authenticator,
	}

	return &userServiceEnv{
		svc:  svc,
		stub: stub,
		ctx:  enttest.NewSystemContext(context.Background()),
	}
}

// seedUserSvcRole 落库一个角色并按 (tenant, code) 反查 ID。
func (e *userServiceEnv) seedUserSvcRole(t *testing.T, tenantID *uint32, roleType permissionV1.Role_Type, roleCode string) uint32 {
	t.Helper()
	require.NoError(t, e.svc.roleRepo.Create(e.ctx, &permissionV1.CreateRoleRequest{
		Data: &permissionV1.Role{
			TenantId: tenantID,
			Name:     trans.Ptr("UserService 角色 " + roleCode),
			Code:     trans.Ptr(roleCode),
			Status:   permissionV1.Role_ON.Enum(),
			Type:     roleType.Enum(),
		},
	}))
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

// seedUserSvcTenant 落库一个启用租户并返回其 ID。
func (e *userServiceEnv) seedUserSvcTenant(t *testing.T, name, code string) uint32 {
	t.Helper()
	created, err := e.svc.tenantRepo.Create(e.ctx, &identityV1.Tenant{
		Name:        trans.Ptr(name),
		Code:        trans.Ptr(code),
		Status:      identityV1.Tenant_ON.Enum(),
		Type:        identityV1.Tenant_TRIAL.Enum(),
		AuditStatus: identityV1.Tenant_APPROVED.Enum(),
	})
	require.NoError(t, err)
	require.NotNil(t, created.GetId())
	return created.GetId()
}

// seedUserSvcCredential 落库一条 USERNAME/PASSWORD_HASH 凭证（bcrypt 经复杂度校验）。
func (e *userServiceEnv) seedUserSvcCredential(t *testing.T, tenantID *uint32, userID uint32, identifier string) {
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
			Status:         authenticationV1.UserCredential_ENABLED.Enum(),
		},
	}))
}

// credentialCount 统计凭证表行数（无过滤）。
func (e *userServiceEnv) credentialCount(t *testing.T) int {
	t.Helper()
	count, err := e.svc.userCredentialRepo.Count(e.ctx, nil)
	require.NoError(t, err)
	return count
}

// findCred 以明文密码直查凭证（needDecrypt=false），返回匹配到的用户 ID。
func (e *userServiceEnv) findCred(t *testing.T, tenantID uint32, identifier, plainPassword string) (uint32, error) {
	t.Helper()
	return e.svc.userCredentialRepo.FindUserCredential(e.ctx, tenantID,
		authenticationV1.UserCredential_USERNAME, identifier, plainPassword, false)
}

// opCtx 构造带操作人载荷的上下文（UserId / TenantId / IsPlatformAdmin 按需指定）。
func opCtx(ctx context.Context, userID, tenantID uint32, platformAdmin bool) context.Context {
	payload := &authenticationV1.UserTokenPayload{UserId: userID, IsPlatformAdmin: trans.Ptr(platformAdmin)}
	if tenantID > 0 {
		payload.TenantId = trans.Ptr(tenantID)
		// 租户上下文默认携带租户管理员旗标：user/role 写入口已加
		// requireTenantOrPlatformAdmin 服务内闸门，测试里的租户操作人均为管理流。
		payload.IsTenantAdmin = trans.Ptr(true)
	}
	return auth.NewContext(ctx, payload)
}

// registerOperator 回放操作人的占位用户行（服务在授权分支前会按操作人 ID
// 取用户行，桩需回放该行才能推进到目标分支；行内容仅含 ID）。
func (e *userServiceEnv) registerOperator(userID uint32) {
	e.stub.usersByID[userID] = &identityV1.User{Id: trans.Ptr(userID)}
}

// TestUserServiceSqlite_ListEnrichment 验证 List 的关联实体名称富集：
// 租户名、角色（列表与单值两轨道）名称/编码、组织单元名称（列表与单值）、
// 职位名称（列表与单值）按真实关联表回填；列表中的 nil 项被跳过。
func TestUserServiceSqlite_ListEnrichment(t *testing.T) {
	e := newUserServiceForTest(t)

	tenantID := e.seedUserSvcTenant(t, "UserService 富集租户", "USERSVC_TENANT_ENRICH")
	tenantRoleID := e.seedUserSvcRole(t, trans.Ptr(tenantID), permissionV1.Role_TENANT, "USERSVC_ROLE_TENANT_ENRICH")
	systemRoleID := e.seedUserSvcRole(t, nil, permissionV1.Role_SYSTEM, "USERSVC_ROLE_SYSTEM_ENRICH")

	// 组织单元与职位（真实落库，经列表反查 ID）。
	require.NoError(t, e.svc.orgUnitRepo.Create(e.ctx, &identityV1.CreateOrgUnitRequest{
		Data: &identityV1.OrgUnit{
			TenantId: trans.Ptr(tenantID),
			Name:     trans.Ptr("UserService 富集组织单元"),
			Code:     trans.Ptr("USERSVC_ORGUNIT_ENRICH"),
			Status:   identityV1.OrgUnit_ON.Enum(),
			Type:     identityV1.OrgUnit_DEPARTMENT.Enum(),
		},
	}))
	orgListResp, err := e.svc.orgUnitRepo.List(e.ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	var orgUnitID uint32
	for _, item := range orgListResp.GetItems() {
		if item.GetCode() == "USERSVC_ORGUNIT_ENRICH" {
			orgUnitID = item.GetId()
		}
	}
	require.NotZero(t, orgUnitID, "组织单元应落库可反查")

	require.NoError(t, e.svc.positionRepo.Create(e.ctx, &identityV1.CreatePositionRequest{
		Data: &identityV1.Position{
			TenantId:  trans.Ptr(tenantID),
			OrgUnitId: trans.Ptr(orgUnitID), // 职位挂靠组织单元（表外键，缺省 0 会被外键约束拒绝）
			Name:      trans.Ptr("UserService 富集职位"),
			Code:      trans.Ptr("USERSVC_POSITION_ENRICH"),
			Status:    identityV1.Position_ON.Enum(),
			// LEADER 此前因 proto 枚举名与 ent 枚举 DB 值 LEAD 错位而无法写入
			//（转换产出非法值被列校验拒绝），对齐后此处可显式行使——
			// 全值往返断言见 position 仓储层 TypeAllValuesLand。
			Type: identityV1.Position_LEADER.Enum(),
		},
	}))
	posListResp, err := e.svc.positionRepo.List(e.ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	var positionID uint32
	for _, item := range posListResp.GetItems() {
		if item.GetCode() == "USERSVC_POSITION_ENRICH" {
			positionID = item.GetId()
		}
	}
	require.NotZero(t, positionID, "职位应落库可反查")

	// 桩回放两条用户（含一条 nil 项验证跳过）：
	// u1 走多值轨道（RoleIds / OrgUnitIds / PositionIds），u2 走单值轨道（RoleId /
	// OrgUnitId / PositionId）。
	u1 := &identityV1.User{
		Id:          trans.Ptr(uint32(88001)),
		TenantId:    trans.Ptr(tenantID),
		RoleIds:     []uint32{tenantRoleID},
		OrgUnitIds:  []uint32{orgUnitID},
		PositionIds: []uint32{positionID},
	}
	u2 := &identityV1.User{
		Id:         trans.Ptr(uint32(88002)),
		RoleId:     trans.Ptr(systemRoleID),
		OrgUnitId:  trans.Ptr(orgUnitID),
		PositionId: trans.Ptr(positionID),
	}
	e.stub.listUsers = []*identityV1.User{u1, nil, u2}

	resp, err := e.svc.List(e.ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	// 服务层不做 nil 项过滤（透传仓储列表），仅关联聚合跳过 nil 项：
	// 此处应为 3 项（两条有效 + 一条未触碰的 nil），nil 项不参与富集。
	require.Len(t, resp.GetItems(), 3, "nil 项透传不删除（仅聚合阶段跳过）")
	enriched := 0
	for _, item := range resp.GetItems() {
		if item == nil {
			continue
		}
		enriched++
		switch item.GetId() {
		case 88001:
			require.Equal(t, "UserService 富集租户", item.GetTenantName(), "租户名应回填")
			require.Equal(t, []string{"UserService 角色 USERSVC_ROLE_TENANT_ENRICH"}, item.GetRoleNames(), "多值角色轨道应回填角色名")
			require.Equal(t, []string{"USERSVC_ROLE_TENANT_ENRICH"}, item.GetRoles(), "多值角色轨道应回填角色码")
			require.Equal(t, "UserService 富集组织单元", item.GetOrgUnitNames()[0], "多值组织单元轨道应回填名称")
			require.Equal(t, "UserService 富集职位", item.GetPositionNames()[0], "多值职位轨道应回填名称")
		case 88002:
			require.Empty(t, item.GetTenantName(), "平台用户不回填租户名")
			// 单值 RoleId 轨道：与租户/组织单元/职位的单值轨道一致，
			// 单值角色引用参与角色富集，回填角色名与角色码。
			require.Equal(t, []string{"UserService 角色 USERSVC_ROLE_SYSTEM_ENRICH"}, item.GetRoleNames(), "单值角色轨道应回填角色名")
			require.Equal(t, []string{"USERSVC_ROLE_SYSTEM_ENRICH"}, item.GetRoles(), "单值角色轨道应回填角色码")
			require.Equal(t, "UserService 富集组织单元", item.GetOrgUnitName(), "单值组织单元轨道应回填名称")
			require.Equal(t, "UserService 富集职位", item.GetPositionName(), "单值职位轨道应回填名称")
		default:
			t.Fatalf("列表中出现未回放的用户 %d", item.GetId())
		}
	}
	require.Equal(t, 2, enriched, "仅两条有效用户参与富集")
}

// TestUserServiceSqlite_GetEnrichment 验证 Get 单条查询的富集回填。
func TestUserServiceSqlite_GetEnrichment(t *testing.T) {
	e := newUserServiceForTest(t)
	tenantID := e.seedUserSvcTenant(t, "UserService 单条富集租户", "USERSVC_TENANT_GET")
	e.stub.usersByID[88003] = &identityV1.User{
		Id:       trans.Ptr(uint32(88003)),
		TenantId: trans.Ptr(tenantID),
	}

	resp, err := e.svc.Get(e.ctx, &identityV1.GetUserRequest{
		QueryBy: &identityV1.GetUserRequest_Id{Id: 88003},
	})
	require.NoError(t, err)
	require.Equal(t, "UserService 单条富集租户", resp.GetTenantName(), "Get 应回填租户名")
}

// TestUserServiceSqlite_CountAndUserExists 验证 Count / UserExists 对仓储的委托。
func TestUserServiceSqlite_CountAndUserExists(t *testing.T) {
	e := newUserServiceForTest(t)
	e.stub.count = 7

	countResp, err := e.svc.Count(e.ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(7), countResp.GetCount())

	existsResp, err := e.svc.UserExists(e.ctx, &identityV1.UserExistsRequest{
		QueryBy: &identityV1.UserExistsRequest_Id{Id: 123},
	})
	require.NoError(t, err)
	require.False(t, existsResp.GetExist(), "桩固定回放不存在")
}

// TestUserServiceSqlite_CreateGuards 验证 Create 的入参守卫：
// nil 数据 400；无操作人 401；操作人用户行缺失错误透传；空角色列表与
// 不存在/类型不匹配的角色 400。
func TestUserServiceSqlite_CreateGuards(t *testing.T) {
	e := newUserServiceForTest(t)
	base := context.Background()
	tenantID := e.seedUserSvcTenant(t, "UserService 守卫租户", "USERSVC_TENANT_GUARD")
	systemRoleID := e.seedUserSvcRole(t, nil, permissionV1.Role_SYSTEM, "USERSVC_ROLE_SYSTEM_GUARD")
	tenantRoleID := e.seedUserSvcRole(t, trans.Ptr(tenantID), permissionV1.Role_TENANT, "USERSVC_ROLE_TENANT_GUARD")
	require.NotZero(t, systemRoleID)
	require.NotZero(t, tenantRoleID)

	// 操作人占位行（下方平台/租户操作人分支用）。
	e.registerOperator(9001)
	e.registerOperator(9002)

	// nil 数据。
	resp, err := e.svc.Create(base, &identityV1.CreateUserRequest{})
	require.Error(t, err)
	require.True(t, adminV1.IsBadRequest(err))
	require.Nil(t, resp)

	// 无操作人。
	resp, err = e.svc.Create(base, &identityV1.CreateUserRequest{
		Data: &identityV1.User{Username: trans.Ptr("usersvc-noop")},
	})
	require.Error(t, err)
	require.True(t, authenticationV1.IsUnauthorized(err), "无操作人应 401")
	require.Nil(t, resp)

	// 操作人用户行缺失（stub 不回放 9xxx 之外的键）：错误透传。
	missingOp := auth.NewContext(base, &authenticationV1.UserTokenPayload{UserId: 99999})
	resp, err = e.svc.Create(missingOp, &identityV1.CreateUserRequest{
		Data: &identityV1.User{Username: trans.Ptr("usersvc-missingop")},
	})
	require.Error(t, err)
	require.Nil(t, resp)

	// 平台操作人但空角色列表。
	platformOp := opCtx(e.ctx, 9001, 0, true)
	resp, err = e.svc.Create(platformOp, &identityV1.CreateUserRequest{
		Data: &identityV1.User{Username: trans.Ptr("usersvc-noroles")},
	})
	require.Error(t, err)
	require.True(t, adminV1.IsBadRequest(err), "空角色列表应 400")
	require.Nil(t, resp)

	// 角色不存在。
	resp, err = e.svc.Create(platformOp, &identityV1.CreateUserRequest{
		Data: &identityV1.User{Username: trans.Ptr("usersvc-ghostrole"), RoleIds: []uint32{987654}},
	})
	require.Error(t, err)
	require.True(t, adminV1.IsBadRequest(err), "不存在的角色应 400 some roles not found")
	require.Nil(t, resp)

	// 租户操作人请求平台（SYSTEM）角色：类型过滤不命中。
	tenantOp := opCtx(e.ctx, 9002, tenantID, false)
	resp, err = e.svc.Create(tenantOp, &identityV1.CreateUserRequest{
		Data: &identityV1.User{
			Username: trans.Ptr("usersvc-crosstyperole"),
			RoleIds:  []uint32{systemRoleID},
		},
	})
	require.Error(t, err)
	require.True(t, adminV1.IsBadRequest(err), "租户操作人请求 SYSTEM 角色应被类型过滤拒绝")
	require.Nil(t, resp)
}

// TestUserServiceSqlite_CreatePlatform 验证平台操作人的创建路径：
// 未显式给密码时注入默认密码，凭证行经真实 bcrypt 落库（自定义密码同理）；
// 弱密码被复杂度策略拒绝；CreatedBy 注入操作人。
func TestUserServiceSqlite_CreatePlatform(t *testing.T) {
	e := newUserServiceForTest(t)
	systemRoleID := e.seedUserSvcRole(t, nil, permissionV1.Role_SYSTEM, "USERSVC_ROLE_SYSTEM_CREATE")
	e.registerOperator(9001)
	platformOp := opCtx(e.ctx, 9001, 0, true)

	// 默认密码路径。
	_, err := e.svc.Create(platformOp, &identityV1.CreateUserRequest{
		Data: &identityV1.User{
			Username: trans.Ptr("usersvc-default-pw"),
			RoleIds:  []uint32{systemRoleID},
		},
	})
	require.NoError(t, err, "默认密码创建应成功")
	require.Equal(t, 1, e.credentialCount(t), "应恰好落库一条凭证")
	matched, ferr := e.findCred(t, 0, "usersvc-default-pw", constants.DefaultUserPassword)
	require.NoError(t, ferr, "默认密码应能通过凭证校验")
	require.Equal(t, uint32(66001), matched, "凭证应绑定桩返回的用户 ID")
	// 记录的创建请求：CreatedBy 被注入操作人 ID。
	require.Len(t, e.stub.creates, 1)
	require.Equal(t, uint32(9001), e.stub.creates[0].Data.GetCreatedBy(), "CreatedBy 应注入操作人")

	// 自定义密码路径。
	_, err = e.svc.Create(platformOp, &identityV1.CreateUserRequest{
		Data: &identityV1.User{
			Username: trans.Ptr("usersvc-custom-pw"),
			RoleIds:  []uint32{systemRoleID},
		},
		Password: trans.Ptr("CustomP@ss777"),
	})
	require.NoError(t, err)
	require.Equal(t, 2, e.credentialCount(t))
	matched, ferr = e.findCred(t, 0, "usersvc-custom-pw", "CustomP@ss777")
	require.NoError(t, ferr, "自定义密码应能通过凭证校验")
	require.Equal(t, uint32(66001), matched)
	_, ferr = e.findCred(t, 0, "usersvc-custom-pw", constants.DefaultUserPassword)
	require.True(t, authenticationV1.IsInvalidPassword(ferr), "旧密码不应通过校验")

	// 弱密码路径。复杂度错误自凭证仓储原样透传（与 ResetCredential /
	// ChangeCredential 的既有透传范式一致），前端可据此给出可操作提示。
	_, err = e.svc.Create(platformOp, &identityV1.CreateUserRequest{
		Data: &identityV1.User{
			Username: trans.Ptr("usersvc-weak-pw"),
			RoleIds:  []uint32{systemRoleID},
		},
		Password: trans.Ptr("short"),
	})
	require.Error(t, err)
	require.True(t, adminV1.IsBadRequest(err), "弱密码应被复杂度策略拒绝")
	require.Contains(t, err.Error(), "password too short", "复杂度拒绝的具体原因应透传（长度不足）")
	require.Equal(t, 2, e.credentialCount(t), "弱密码不应落库")
}

// TestUserServiceSqlite_CreateTenant 验证租户操作人的创建路径：
// 请求被改写为操作人租户（TenantId 覆盖 + TENANT 类型角色过滤），
// 凭证行绑定租户并可用默认密码校验。
func TestUserServiceSqlite_CreateTenant(t *testing.T) {
	e := newUserServiceForTest(t)
	tenantID := e.seedUserSvcTenant(t, "UserService 创建租户", "USERSVC_TENANT_CREATE")
	tenantRoleID := e.seedUserSvcRole(t, trans.Ptr(tenantID), permissionV1.Role_TENANT, "USERSVC_ROLE_TENANT_CREATE")
	e.registerOperator(9002)
	tenantOp := opCtx(e.ctx, 9002, tenantID, false)

	_, err := e.svc.Create(tenantOp, &identityV1.CreateUserRequest{
		Data: &identityV1.User{
			Username: trans.Ptr("usersvc-tenant-created"),
			RoleIds:  []uint32{tenantRoleID},
		},
	})
	require.NoError(t, err, "租户内创建应成功")
	require.Equal(t, 1, e.credentialCount(t), "应恰好落库一条租户凭证")
	// 记录的创建请求：TenantId 被强制覆盖为操作人租户。
	require.Len(t, e.stub.creates, 1)
	require.Equal(t, tenantID, e.stub.creates[0].Data.GetTenantId(), "请求租户应被覆盖为操作人租户")
	matched, ferr := e.findCred(t, tenantID, "usersvc-tenant-created", constants.DefaultUserPassword)
	require.NoError(t, ferr, "租户凭证应能以默认密码通过校验")
	require.Equal(t, uint32(66001), matched)
	_, ferr = e.findCred(t, 0, "usersvc-tenant-created", constants.DefaultUserPassword)
	require.Error(t, ferr, "平台(0)上下文不应命中租户凭证")
}

// TestUserServiceSqlite_UpdatePasswordReset 验证 Update 的密码重置与注入：
// 无密码路径记录 userRepo.Update（UpdatedBy 注入、update_mask 追加 updated_by、
// Data.Id 对齐目标）；平台管理员重置密码走真实 ResetCredential（旧失效新生效）；
// 弱密码被拒；租户操作人重置跨租户用户被 403。
func TestUserServiceSqlite_UpdatePasswordReset(t *testing.T) {
	e := newUserServiceForTest(t)
	systemRoleID := e.seedUserSvcRole(t, nil, permissionV1.Role_SYSTEM, "USERSVC_ROLE_SYSTEM_UPDATE")
	tenantA := e.seedUserSvcTenant(t, "UserService 更新租户甲", "USERSVC_TENANT_UPDATE_A")
	tenantB := e.seedUserSvcTenant(t, "UserService 更新租户乙", "USERSVC_TENANT_UPDATE_B")

	e.seedUserSvcCredential(t, nil, 10501, "usersvc-reset-target")
	// Username 必填：ResetCredential 以目标用户行（而非请求体）的用户名为标识符定位凭证。
	e.stub.usersByID[10501] = &identityV1.User{Id: trans.Ptr(uint32(10501)), Username: trans.Ptr("usersvc-reset-target")}
	e.seedUserSvcCredential(t, trans.Ptr(tenantB), 10502, "usersvc-cross-tenant-target")
	e.stub.usersByID[10502] = &identityV1.User{Id: trans.Ptr(uint32(10502)), TenantId: trans.Ptr(tenantB), Username: trans.Ptr("usersvc-cross-tenant-target")}

	// 操作人占位行（下方平台/租户操作人分支用）。
	e.registerOperator(9001)
	e.registerOperator(9002)

	platformOp := opCtx(e.ctx, 9001, 0, true)

	// 无密码路径：更新被记录，UpdatedBy 注入、mask 追加 updated_by、Data.Id 对齐。
	_, err := e.svc.Update(platformOp, &identityV1.UpdateUserRequest{
		Id: 10501,
		Data: &identityV1.User{
			Username: trans.Ptr("usersvc-reset-target"),
			RoleIds:  []uint32{systemRoleID},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"nickname"}},
	})
	require.NoError(t, err)
	require.Len(t, e.stub.updates, 1, "无密码路径应记录一次用户更新")
	recorded := e.stub.updates[0]
	require.Equal(t, uint32(10501), recorded.GetId())
	require.Equal(t, uint32(10501), recorded.Data.GetId(), "Data.Id 应对齐目标用户")
	require.Equal(t, uint32(9001), recorded.Data.GetUpdatedBy(), "UpdatedBy 应注入操作人")
	require.Contains(t, recorded.UpdateMask.GetPaths(), "updated_by", "mask 应追加 updated_by")

	// 平台管理员重置密码：旧失效、新生效。
	_, err = e.svc.Update(platformOp, &identityV1.UpdateUserRequest{
		Id:       10501,
		Password: trans.Ptr(encryptLoginPassword(t, "ResetP@ss888")),
		Data: &identityV1.User{
			Username: trans.Ptr("usersvc-reset-target"),
			RoleIds:  []uint32{systemRoleID},
		},
	})
	require.NoError(t, err, "平台管理员重置应成功")
	_, ferr := e.findCred(t, 0, "usersvc-reset-target", "ResetP@ss888")
	require.NoError(t, ferr, "重置后的新密码应通过校验")
	_, ferr = e.findCred(t, 0, "usersvc-reset-target", constants.DefaultUserPassword)
	require.True(t, authenticationV1.IsInvalidPassword(ferr), "重置前的旧密码应失效")

	// 弱密码重置被拒：复杂度错误经 ResetCredential 原样透传（既有透传范式）。
	_, err = e.svc.Update(platformOp, &identityV1.UpdateUserRequest{
		Id:       10501,
		Password: trans.Ptr(encryptLoginPassword(t, "abc")),
		Data: &identityV1.User{
			Username: trans.Ptr("usersvc-reset-target"),
			RoleIds:  []uint32{systemRoleID},
		},
	})
	require.Error(t, err)
	require.True(t, adminV1.IsBadRequest(err), "弱密码重置应被复杂度策略拒绝")
	require.Contains(t, err.Error(), "password too short", "复杂度拒绝的具体原因应透传（长度不足）")

	// 租户操作人重置跨租户用户：403。
	// 角色校验会先于租户检查执行：租户操作人的请求须携带本租户（TENANT）角色
	// 才能走到跨租户拒绝分支（否则先被"some roles not found"拦截）。
	tenantRoleA := e.seedUserSvcRole(t, trans.Ptr(tenantA), permissionV1.Role_TENANT, "USERSVC_ROLE_TENANT_UPDATE_A")
	tenantOp := opCtx(e.ctx, 9002, tenantA, false)
	resp, err := e.svc.Update(tenantOp, &identityV1.UpdateUserRequest{
		Id:       10502,
		Password: trans.Ptr(encryptLoginPassword(t, "HackedP@ss999")),
		Data: &identityV1.User{
			Username: trans.Ptr("usersvc-cross-tenant-target"),
			RoleIds:  []uint32{tenantRoleA},
		},
	})
	require.Error(t, err)
	require.True(t, adminV1.IsForbidden(err), "跨租户重置应 403")
	require.Nil(t, resp)
	_, ferr = e.findCred(t, tenantB, "usersvc-cross-tenant-target", constants.DefaultUserPassword)
	require.NoError(t, ferr, "跨租户目标的密码不应被改动")
}

// TestUserServiceSqlite_EditUserPassword 验证 EditUserPassword：
// 无操作人 401；目标用户行缺失错误透传；跨租户 403；
// 平台管理员与本租户管理员重置走真实 ResetCredential（新旧密码切换生效）。
func TestUserServiceSqlite_EditUserPassword(t *testing.T) {
	e := newUserServiceForTest(t)
	base := context.Background()
	tenantA := e.seedUserSvcTenant(t, "UserService 改密租户甲", "USERSVC_TENANT_PW_A")
	tenantB := e.seedUserSvcTenant(t, "UserService 改密租户乙", "USERSVC_TENANT_PW_B")

	e.seedUserSvcCredential(t, nil, 10601, "usersvc-edit-target-platform")
	// Username 必填：EditUserPassword 以目标用户行的用户名为标识符定位凭证。
	e.stub.usersByID[10601] = &identityV1.User{Id: trans.Ptr(uint32(10601)), Username: trans.Ptr("usersvc-edit-target-platform")}
	e.seedUserSvcCredential(t, trans.Ptr(tenantA), 10602, "usersvc-edit-target-tenant")
	e.stub.usersByID[10602] = &identityV1.User{Id: trans.Ptr(uint32(10602)), TenantId: trans.Ptr(tenantA), Username: trans.Ptr("usersvc-edit-target-tenant")}
	e.seedUserSvcCredential(t, trans.Ptr(tenantB), 10603, "usersvc-edit-target-foreign")
	e.stub.usersByID[10603] = &identityV1.User{Id: trans.Ptr(uint32(10603)), TenantId: trans.Ptr(tenantB), Username: trans.Ptr("usersvc-edit-target-foreign")}

	// 操作人占位行（下方平台/租户管理员分支用）。
	e.registerOperator(9001)
	e.registerOperator(9003)

	// 无操作人。
	resp, err := e.svc.EditUserPassword(base, &identityV1.EditUserPasswordRequest{UserId: 10601, NewPassword: "x"})
	require.Error(t, err)
	require.True(t, authenticationV1.IsUnauthorized(err))
	require.Nil(t, resp)

	// 目标用户行缺失。
	missingOp := auth.NewContext(base, &authenticationV1.UserTokenPayload{UserId: 9001})
	resp, err = e.svc.EditUserPassword(missingOp, &identityV1.EditUserPasswordRequest{UserId: 88888, NewPassword: "x"})
	require.Error(t, err)
	require.Nil(t, resp)

	// 跨租户：租户甲管理员改租户乙用户 → 403。
	// EditUserPassword 已加 requireTenantOrPlatformAdmin 服务内闸门，
	// 操作者必须带租户管理员旗标（改密是管理员动作）。
	tenantOpA := auth.NewContext(e.ctx, &authenticationV1.UserTokenPayload{
		UserId:        9003,
		TenantId:      trans.Ptr(tenantA),
		IsTenantAdmin: trans.Ptr(true),
	})
	resp, err = e.svc.EditUserPassword(tenantOpA, &identityV1.EditUserPasswordRequest{
		UserId:      10603,
		NewPassword: encryptLoginPassword(t, "HackedP@ss888"),
	})
	require.Error(t, err)
	require.True(t, adminV1.IsForbidden(err), "跨租户改密应 403")
	require.Nil(t, resp)
	_, ferr := e.findCred(t, tenantB, "usersvc-edit-target-foreign", constants.DefaultUserPassword)
	require.NoError(t, ferr, "跨租户目标密码不应被改动")

	// 平台管理员重置（目标为平台用户）。
	platformOp := opCtx(e.ctx, 9001, 0, true)
	_, err = e.svc.EditUserPassword(platformOp, &identityV1.EditUserPasswordRequest{
		UserId:      10601,
		NewPassword: encryptLoginPassword(t, "PlatformP@ss66"),
	})
	require.NoError(t, err)
	_, ferr = e.findCred(t, 0, "usersvc-edit-target-platform", "PlatformP@ss66")
	require.NoError(t, ferr, "重置后新密码应生效")
	_, ferr = e.findCred(t, 0, "usersvc-edit-target-platform", constants.DefaultUserPassword)
	require.True(t, authenticationV1.IsInvalidPassword(ferr), "旧密码应失效")

	// 本租户管理员改本租户用户。
	_, err = e.svc.EditUserPassword(tenantOpA, &identityV1.EditUserPasswordRequest{
		UserId:      10602,
		NewPassword: encryptLoginPassword(t, "TenantP@ss55"),
	})
	require.NoError(t, err)
	_, ferr = e.findCred(t, tenantA, "usersvc-edit-target-tenant", "TenantP@ss55")
	require.NoError(t, ferr, "本租户重置后新密码应生效")
	_, ferr = e.findCred(t, tenantA, "usersvc-edit-target-tenant", constants.DefaultUserPassword)
	require.True(t, authenticationV1.IsInvalidPassword(ferr), "旧密码应失效")
}

// TestUserServiceSqlite_DeleteGuards 验证 Delete 的守卫与正常路径：
// 无操作人 401；操作人/目标用户行缺失透传；id=1 与"平台 admin 用户名+租户0"
// 双条件默认管理员保护；自删保护；正常目标删除被记录。
func TestUserServiceSqlite_DeleteGuards(t *testing.T) {
	e := newUserServiceForTest(t)
	base := context.Background()

	// 无操作人。
	resp, err := e.svc.Delete(base, &identityV1.DeleteUserRequest{QueryBy: &identityV1.DeleteUserRequest_Id{Id: 1}})
	require.Error(t, err)
	require.True(t, authenticationV1.IsUnauthorized(err))
	require.Nil(t, resp)

	// 操作人占位行（下方操作人分支用）。
	e.registerOperator(9001)
	platformOp := opCtx(e.ctx, 9001, 0, true)

	// 操作人用户行缺失。
	missingOp := auth.NewContext(base, &authenticationV1.UserTokenPayload{UserId: 99999})
	resp, err = e.svc.Delete(missingOp, &identityV1.DeleteUserRequest{QueryBy: &identityV1.DeleteUserRequest_Id{Id: 1}})
	require.Error(t, err)
	require.Nil(t, resp)

	// 目标用户行缺失。
	resp, err = e.svc.Delete(platformOp, &identityV1.DeleteUserRequest{QueryBy: &identityV1.DeleteUserRequest_Id{Id: 77777}})
	require.Error(t, err)
	require.Nil(t, resp)

	// id=1 默认管理员保护。
	e.stub.usersByID[1] = &identityV1.User{Id: trans.Ptr(uint32(1))}
	resp, err = e.svc.Delete(platformOp, &identityV1.DeleteUserRequest{QueryBy: &identityV1.DeleteUserRequest_Id{Id: 1}})
	require.Error(t, err)
	require.True(t, adminV1.IsBadRequest(err), "id=1 应触发默认管理员保护")
	require.Nil(t, resp)

	// 平台 admin 用户名 + 租户 0 的改名后兜底保护。
	e.stub.usersByID[5] = &identityV1.User{
		Id:       trans.Ptr(uint32(5)),
		Username: trans.Ptr(constants.DefaultAdminUserName),
	}
	resp, err = e.svc.Delete(platformOp, &identityV1.DeleteUserRequest{QueryBy: &identityV1.DeleteUserRequest_Id{Id: 5}})
	require.Error(t, err)
	require.True(t, adminV1.IsBadRequest(err), "平台 admin 用户名应触发兜底保护")
	require.Nil(t, resp)

	// 自删保护（操作人 9001 已在上方注册占位行）。
	resp, err = e.svc.Delete(platformOp, &identityV1.DeleteUserRequest{QueryBy: &identityV1.DeleteUserRequest_Id{Id: 9001}})
	require.Error(t, err)
	require.True(t, adminV1.IsBadRequest(err), "自删应被拒绝")
	require.Nil(t, resp)

	// 正常目标：删除被记录。
	e.stub.usersByID[10701] = &identityV1.User{Id: trans.Ptr(uint32(10701))}
	_, err = e.svc.Delete(platformOp, &identityV1.DeleteUserRequest{QueryBy: &identityV1.DeleteUserRequest_Id{Id: 10701}})
	require.NoError(t, err)
	require.Equal(t, []uint32{10701}, e.stub.deletes, "删除调用应被记录")
}

// TestUserServiceSqlite_InitSeedsEmptyDB 验证空库初始化播种：
// DefaultUsers 经 userRepo.Create 落创建记录、DefaultUserCredentials 经真实
// userCredentialRepo 落凭证行（绑定 Create 返回的用户 ID）、DefaultUserRoles
// 经 AssignUserRole 落角色关联记录（OneToOne 分支）。
func TestUserServiceSqlite_InitSeedsEmptyDB(t *testing.T) {
	e := newUserServiceForTest(t)
	require.Equal(t, 0, e.credentialCount(t))

	e.svc.init()

	require.Len(t, e.stub.creates, len(constants.DefaultUsers), "默认用户应逐一经仓储创建")
	require.Len(t, e.stub.assignedRoles, len(constants.DefaultUserRoles), "OneToOne 分支应落默认用户角色关联记录")
	require.Equal(t, 1, e.credentialCount(t), "默认凭证应落库一行")
	matched, ferr := e.findCred(t, 0, "admin", constants.DefaultUserPassword)
	require.NoError(t, ferr, "播种凭证应可经默认口令校验")
	require.Equal(t, uint32(66001), matched, "播种凭证应绑定 Create 返回的用户 ID（overrideUserID）")
}

// TestUserServiceSqlite_InitReseedsMissingCredentials 验证凭证自愈补种：
// 用户表非空而凭证表为空（历史半初始化故障特征）时，init 重建默认凭证
// 并绑定种子内的 UserId。
func TestUserServiceSqlite_InitReseedsMissingCredentials(t *testing.T) {
	e := newUserServiceForTest(t)
	e.stub.count = 1 // 用户表非空：跳过 createDefaultUser
	e.stub.usersByID[1] = &identityV1.User{Id: trans.Ptr(uint32(1))}

	e.svc.init()

	require.Empty(t, e.stub.creates, "用户表非空不应再建默认用户")
	require.Empty(t, e.stub.assignedRoles, "不应再落默认用户角色关联记录")
	require.Equal(t, 1, e.credentialCount(t), "自愈补种应重建默认凭证")
	matched, ferr := e.findCred(t, 0, "admin", constants.DefaultUserPassword)
	require.NoError(t, ferr)
	require.Equal(t, uint32(1), matched, "补种凭证应保留种子内的 UserId（overrideUserID=0 路径）")
}

// TestUserServiceSqlite_CreateDefaultUserCredentialsOverride 验证
// createDefaultUserCredentials 的 overrideUserID 绑定语义：
// 显式传入的用户 ID 覆盖种子内的 UserId。
func TestUserServiceSqlite_CreateDefaultUserCredentialsOverride(t *testing.T) {
	e := newUserServiceForTest(t)
	require.NoError(t, e.svc.createDefaultUserCredentials(e.ctx, 2002))
	require.Equal(t, 1, e.credentialCount(t))
	matched, ferr := e.findCred(t, 0, "admin", constants.DefaultUserPassword)
	require.NoError(t, ferr)
	require.Equal(t, uint32(2002), matched, "凭证应绑定显式传入的 overrideUserID")
}
