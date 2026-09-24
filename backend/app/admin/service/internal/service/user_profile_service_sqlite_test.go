// UserProfileService 的 SQLite 内存库 + miniredis 集成测试（白盒，包内测试）。
//
// 覆盖目标：
//   - GetUser：角色码回填（经真实 RoleRepo.ListRoleCodesByRoleIds 查询已落库角色）、
//     stub 查询 ID 必为操作人本人、无角色声明时角色列表为空、用户查询失败与
//     缺操作人声明的报错分支。
//   - UpdateUser / DeleteAvatar / UploadAvatar(ImageUrl)：操作人 ID 注入
//     （req.Id / req.Data.Id 一律盖为操作人）、字段掩码仅含 avatar、
//     url 头像路径的原样写入与回显；以及 UploadAvatar 的全部校验分支
//     （非 base64 / 空 / 非图片 / 超限 / 无来源 / 缺操作人声明）。
//   - ChangePassword：AES+base64 密文解密链路（go-utils crypto 默认密钥）、
//     旧口令 bcrypt 校验、新口令复杂度（过短/弱口令）拒绝、改密成功后凭证哈希
//     更新与等保历史口令记录、以及 RevokeUserTokenAllClientTypes 经 miniredis
//     把两个客户端类型下的全部访问令牌清空；另有密文格式错误、旧口令错误、
//     凭证不存在等分支。
//   - 历史口令策略：改密成功后再改回近期用过的口令被拒。
//
// 跳过项：
//   - vcodeCache：生产构造器签名取 *bLogger.Context（跨包不可构造），且该字段
//     不被任何被测路径触碰，构造时置 nil。
//   - mc（MinIOClient）：需真实 MinIO 实例，头像 base64 直传的落 OSS 分支
//     （校验通过后的 UploadFile 调用）跳过；其前的全部本地校验分支均已覆盖。
package service

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	gocrypto "github.com/tx7do/go-utils/crypto"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/emptypb"

	conf "github.com/tx7do/kratos-bootstrap/api/gen/go/conf/v1"

	entCrud "github.com/tx7do/go-crud/entgo"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/usercredential"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/middleware/auth"
	"go-wind-admin/pkg/oss"
)

// userProfileUserRepoStub 是 UserProfileService 专用的 data.UserRepo 桩：
// 嵌入接口获得默认方法集（未覆写方法一旦被调用即 nil 接口 panic，测试即失败），
// 只覆写被测路径会用到的 Get / Update——Get 按配置返回占位用户或错误并记录
// 被查询的 ID，Update 机械记录透传的请求供断言操作人注入与掩码。
type userProfileUserRepoStub struct {
	data.UserRepo
	getUserResult      *identityV1.User
	getUserErr         error
	capturedGetQueryID uint32
	capturedUpdates    []*identityV1.UpdateUserRequest
}

func (s *userProfileUserRepoStub) Get(_ context.Context, req *identityV1.GetUserRequest) (*identityV1.User, error) {
	if idWrapper, ok := req.GetQueryBy().(*identityV1.GetUserRequest_Id); ok {
		s.capturedGetQueryID = idWrapper.Id
	}
	if s.getUserErr != nil {
		return nil, s.getUserErr
	}
	return s.getUserResult, nil
}

func (s *userProfileUserRepoStub) Update(_ context.Context, req *identityV1.UpdateUserRequest) error {
	s.capturedUpdates = append(s.capturedUpdates, req)
	return nil
}

// userProfileServiceTestEnv 汇集服务构造产物：svc 为被测服务（userRepo 为桩），
// entClient 供落库断言，tokenCache 即注入 authenticator 的同一实例
// （跨包私有字段无法直接取，供改密吊销链路的令牌键断言）。
type userProfileServiceTestEnv struct {
	svc        *UserProfileService
	stub       *userProfileUserRepoStub
	entClient  *entCrud.EntClient[*ent.Client]
	tokenCache *data.UserTokenCache
}

// newUserProfileServiceForTest 白盒复刻 NewUserProfileService 的字段初始化：
// log 换 NopLogger；userRepo 用本文件桩；roleRepo 走既有 testkit；
// userCredentialRepo / configRepo 走 repo_testkit5（miniredis 假 client 注入，
// 装配关系与生产 wiring_ent.go 一致）；authenticator 走 repo_testkit5 注入式
// 构造；notifier（换绑验证码邮件出口）与 mc（需 MinIO 实例）
// 置 nil，见文件头跳过说明。
func newUserProfileServiceForTest(t *testing.T) userProfileServiceTestEnv {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	tokenCache := data.NewUserTokenCacheForTest(rdb)
	authenticator, err := data.NewAuthenticatorForTest(&conf.Authentication_Jwt{
		Method: "HS256",
		Key:    "user-profile-svc-test-hs256-key",
	}, tokenCache)
	require.NoError(t, err, "构造测试 Authenticator 应成功")

	configRepo := data.NewConfigRepoForTest(entClient, rdb)
	passwordCrypto := data.NewPasswordCrypto()
	userCredentialRepo := data.NewUserCredentialRepoForTest(entClient, passwordCrypto, configRepo)

	stub := &userProfileUserRepoStub{}
	svc := &UserProfileService{
		log:                bLogger.NewHelper(bLogger.NopLogger()),
		userRepo:           stub,
		roleRepo:           data.NewRoleRepoForTest(entClient),
		userCredentialRepo: userCredentialRepo,
		authenticator:      authenticator,
		notifier:           nil,
		vcodeCache:         nil,
		mc:                 nil,
	}
	return userProfileServiceTestEnv{svc: svc, stub: stub, entClient: entClient, tokenCache: tokenCache}
}

// profileEncCredential 按前端口径构造 AES+base64 传输密文
// （与 ChangeCredential 解密口径一致：默认 AES 密钥、nil IV）。
func profileEncCredential(t *testing.T, plain string) string {
	t.Helper()
	cipherText, err := gocrypto.AesEncrypt([]byte(plain), gocrypto.DefaultAESKey, nil)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(cipherText)
}

// seedProfileUserCredential 经 ent 直插一枚 username/password_hash 凭证行，
// 哈希走生产 bcrypt 实现（NewPasswordCrypto）。
func seedProfileUserCredential(t *testing.T, entClient *entCrud.EntClient[*ent.Client], ctx context.Context, userID uint32, identifier, plainPassword string) {
	t.Helper()
	pc := data.NewPasswordCrypto()
	hash, err := pc.Encrypt(plainPassword)
	require.NoError(t, err)
	_, err = entClient.Client().UserCredential.Create().
		SetUserID(userID).
		SetIdentityType(usercredential.IdentityTypeUsername).
		SetIdentifier(identifier).
		SetCredentialType(usercredential.CredentialTypePasswordHash).
		SetCredential(hash).
		SetCreatedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err, "直插凭证行应成功")
}

// profileRowCredentialHash 直查当前凭证哈希，供改密前后比对。
func profileRowCredentialHash(t *testing.T, entClient *entCrud.EntClient[*ent.Client], ctx context.Context) (string, *string) {
	t.Helper()
	row, err := entClient.Client().UserCredential.Query().Only(ctx)
	require.NoError(t, err)
	require.NotNil(t, row.Credential, "凭证哈希应存在")
	return *row.Credential, row.ExtraInfo
}

// TestUserProfileServiceSqlite_GetUser_EnrichesRoleCodes 验证 GetUser 的
// 角色码回填：stub 返回的角色 ID 经真实 RoleRepo 查询落库角色回填角色码；
// 查询 ID 必为操作人本人。
func TestUserProfileServiceSqlite_GetUser_EnrichesRoleCodes(t *testing.T) {
	env := newUserProfileServiceForTest(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	// 经真实 RoleRepo.Create 落库一枚角色（testkit 装配含其关联子 repo）。
	require.NoError(t, env.svc.roleRepo.Create(ctx, &permissionV1.CreateRoleRequest{
		Data: &permissionV1.Role{
			Name: trans.Ptr("档案服务角色"),
			Code: trans.Ptr("PROFILE_ROLE_A"),
		},
	}))
	roles, err := env.entClient.Client().Role.Query().All(ctx)
	require.NoError(t, err)
	var roleID uint32
	for _, r := range roles {
		if r.Code != nil && *r.Code == "PROFILE_ROLE_A" {
			roleID = r.ID
		}
	}
	require.NotZero(t, roleID, "应能查到落库角色的 ID")

	env.stub.getUserResult = &identityV1.User{
		Id:      trans.Ptr(uint32(4242)),
		RoleIds: []uint32{roleID},
	}
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 4242})

	resp, err := env.svc.GetUser(opCtx, &emptypb.Empty{})
	require.NoError(t, err)
	require.Equal(t, []string{"PROFILE_ROLE_A"}, resp.GetRoles(), "角色码应经真实 RoleRepo 查询回填")
	require.EqualValues(t, 4242, env.stub.capturedGetQueryID, "查询 ID 应为操作人本人而非任意指定")
}

// TestUserProfileServiceSqlite_GetUser_NoRolesAndErrors 验证无角色声明时
// 角色列表为空；用户查询失败与缺操作人声明均报错。
func TestUserProfileServiceSqlite_GetUser_NoRolesAndErrors(t *testing.T) {
	env := newUserProfileServiceForTest(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	env.stub.getUserResult = &identityV1.User{Id: trans.Ptr(uint32(4242))}
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 4242})

	resp, err := env.svc.GetUser(opCtx, &emptypb.Empty{})
	require.NoError(t, err)
	require.Empty(t, resp.GetRoles(), "无角色声明时角色列表应为空")
	require.EqualValues(t, 4242, env.stub.capturedGetQueryID)

	env.stub.getUserErr = errors.New("stub-database-boom")
	_, err = env.svc.GetUser(opCtx, &emptypb.Empty{})
	require.Error(t, err, "用户查询失败应报错（NotFound）")

	_, err = env.svc.GetUser(ctx, &emptypb.Empty{})
	require.Error(t, err, "缺操作人声明的查询应被拒绝")
}

// TestUserProfileServiceSqlite_UpdateUser_OperatorInjection 验证 UpdateUser
// 的操作人注入：req.Id 与 req.Data.Id 一律盖为操作人 ID 后透传仓储层；
// 缺操作人声明报错且不触达仓储层。
func TestUserProfileServiceSqlite_UpdateUser_OperatorInjection(t *testing.T) {
	env := newUserProfileServiceForTest(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 4242})

	_, err := env.svc.UpdateUser(opCtx, &identityV1.UpdateUserRequest{
		Data: &identityV1.User{Nickname: trans.Ptr("服务层改昵称")},
	})
	require.NoError(t, err)
	require.Len(t, env.stub.capturedUpdates, 1, "应恰好透传一次更新请求")
	captured := env.stub.capturedUpdates[0]
	require.EqualValues(t, 4242, captured.Id, "req.Id 应被盖为操作人 ID")
	require.EqualValues(t, 4242, captured.Data.GetId(), "req.Data.Id 应被盖为操作人 ID")

	_, err = env.svc.UpdateUser(ctx, &identityV1.UpdateUserRequest{
		Data: &identityV1.User{Nickname: trans.Ptr("不应生效")},
	})
	require.Error(t, err, "缺操作人声明的更新应被拒绝")
	require.Len(t, env.stub.capturedUpdates, 1, "被拒绝的更新不应触达仓储层")
}

// TestUserProfileServiceSqlite_DeleteAvatar_ClearsAvatar 验证 DeleteAvatar
// 透传的更新：掩码仅含 avatar、头像值清空、目标 ID 为操作人；
// 缺操作人声明报错且不触达仓储层。
func TestUserProfileServiceSqlite_DeleteAvatar_ClearsAvatar(t *testing.T) {
	env := newUserProfileServiceForTest(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 4242})

	_, err := env.svc.DeleteAvatar(opCtx, &emptypb.Empty{})
	require.NoError(t, err)
	require.Len(t, env.stub.capturedUpdates, 1)
	captured := env.stub.capturedUpdates[0]
	require.Equal(t, uint32(4242), captured.Data.GetId(), "目标 ID 应为操作人")
	require.Empty(t, captured.Data.GetAvatar(), "头像值应被清空")
	require.NotNil(t, captured.UpdateMask, "更新应携带 avatar 掩码")
	require.Equal(t, []string{"avatar"}, captured.UpdateMask.GetPaths(), "掩码应仅含 avatar")

	_, err = env.svc.DeleteAvatar(ctx, &emptypb.Empty{})
	require.Error(t, err, "缺操作人声明的删除应被拒绝")
	require.Len(t, env.stub.capturedUpdates, 1, "被拒绝的删除不应触达仓储层")
}

// TestUserProfileServiceSqlite_UploadAvatar_UrlPathAndValidation 验证
// ImageUrl 路径的原样写入回显，与全部本地校验分支
// （非 base64 / 空 / 非图片 / 超限 / 无来源 / 缺操作人声明）。
func TestUserProfileServiceSqlite_UploadAvatar_UrlPathAndValidation(t *testing.T) {
	env := newUserProfileServiceForTest(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 4242})

	// ImageUrl 路径：不触碰 OSS，原样写入并回显。
	const avatarURL = "https://cdn.example.test/avatar.png"
	resp, err := env.svc.UploadAvatar(opCtx, &identityV1.UploadAvatarRequest{
		Source: &identityV1.UploadAvatarRequest_ImageUrl{ImageUrl: avatarURL},
	})
	require.NoError(t, err)
	require.Equal(t, avatarURL, resp.GetUrl(), "url 路径应原样回显")
	require.Len(t, env.stub.capturedUpdates, 1)
	captured := env.stub.capturedUpdates[0]
	require.Equal(t, uint32(4242), captured.Data.GetId(), "目标 ID 应为操作人")
	require.Equal(t, avatarURL, captured.Data.GetAvatar(), "url 路径的头像值应原样写入")
	require.Equal(t, []string{"avatar"}, captured.UpdateMask.GetPaths(), "掩码应仅含 avatar")

	// 非法 base64。
	_, err = env.svc.UploadAvatar(opCtx, &identityV1.UploadAvatarRequest{
		Source: &identityV1.UploadAvatarRequest_ImageBase64{ImageBase64: "%%%not-base64%%%"},
	})
	require.Error(t, err, "非 base64 数据应被拒绝")
	require.Contains(t, err.Error(), "invalid avatar base64 data")

	// 空 base64（解码为 0 字节）。
	_, err = env.svc.UploadAvatar(opCtx, &identityV1.UploadAvatarRequest{
		Source: &identityV1.UploadAvatarRequest_ImageBase64{ImageBase64: ""},
	})
	require.Error(t, err, "空头像数据应被拒绝")
	require.Contains(t, err.Error(), "empty avatar data")

	// 可解码但非图片内容（嗅探为 text/plain）。
	_, err = env.svc.UploadAvatar(opCtx, &identityV1.UploadAvatarRequest{
		Source: &identityV1.UploadAvatarRequest_ImageBase64{ImageBase64: base64.StdEncoding.EncodeToString([]byte("plain text, definitely not an image"))},
	})
	require.Error(t, err, "非图片 MIME 应被拒绝")
	require.Contains(t, err.Error(), "only image files are allowed for avatar")

	// 超过上传上限（50 MiB）。
	_, err = env.svc.UploadAvatar(opCtx, &identityV1.UploadAvatarRequest{
		Source: &identityV1.UploadAvatarRequest_ImageBase64{ImageBase64: base64.StdEncoding.EncodeToString(make([]byte, oss.MaxUploadSize+1))},
	})
	require.Error(t, err, "超限头像应被拒绝")
	require.Contains(t, err.Error(), "avatar exceeds max size")

	// 无来源（oneof 未置）。
	_, err = env.svc.UploadAvatar(opCtx, &identityV1.UploadAvatarRequest{})
	require.Error(t, err, "无来源的头像上传应被拒绝")
	require.Contains(t, err.Error(), "invalid avatar source")

	// 缺操作人声明。
	_, err = env.svc.UploadAvatar(ctx, &identityV1.UploadAvatarRequest{
		Source: &identityV1.UploadAvatarRequest_ImageUrl{ImageUrl: avatarURL},
	})
	require.Error(t, err, "缺操作人声明的上传应被拒绝")

	// 以上校验失败分支均不应触达仓储层。
	require.Len(t, env.stub.capturedUpdates, 1, "校验失败分支不应透传更新")
}

// TestUserProfileServiceSqlite_ChangePassword_HappyPath 验证改密全链路：
// 密文解密（AES+base64，默认密钥）→ 旧口令 bcrypt 校验 → 新口令复杂度通过 →
// 凭证哈希更新为新城、等保历史口令记录追加 → 两个客户端类型下的全部
// 访问令牌经 miniredis 清空（改密强制下线）。
func TestUserProfileServiceSqlite_ChangePassword_HappyPath(t *testing.T) {
	env := newUserProfileServiceForTest(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const uid = 4242
	seedProfileUserCredential(t, env.entClient, ctx, uid, "profile-user-a", "OldPass@1234")

	// 预置两个客户端类型下的访问令牌，验证改密后的全端吊销。
	require.NoError(t, env.tokenCache.AddAccessToken(ctx, authenticationV1.ClientType_admin, uid, "jti-prof-a", "tok-prof-a", time.Hour))
	require.NoError(t, env.tokenCache.AddAccessToken(ctx, authenticationV1.ClientType_app, uid, "jti-prof-b", "tok-prof-b", time.Hour))
	require.Len(t, env.tokenCache.GetAccessTokens(ctx, authenticationV1.ClientType_admin, uid), 1)
	require.Len(t, env.tokenCache.GetAccessTokens(ctx, authenticationV1.ClientType_app, uid), 1)

	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{
		UserId:   uid,
		Username: trans.Ptr("profile-user-a"),
	})

	_, err := env.svc.ChangePassword(opCtx, &identityV1.ChangePasswordRequest{
		OldPassword: profileEncCredential(t, "OldPass@1234"),
		NewPassword: profileEncCredential(t, "NewPass@5678"),
	})
	require.NoError(t, err, "口令合规的改密应成功")

	// 凭证哈希已换新：新城通过校验、旧城不再通过。
	pc := data.NewPasswordCrypto()
	newHash, extraInfo := profileRowCredentialHash(t, env.entClient, ctx)
	ok, verr := pc.Verify("NewPass@5678", newHash)
	require.NoError(t, verr)
	require.True(t, ok, "改密后新口令应通过 bcrypt 校验")
	ok, _ = pc.Verify("OldPass@1234", newHash)
	require.False(t, ok, "改密后旧口令不应再通过校验")

	// 等保历史口令：旧哈希被追加进 extra_info 的历史列表。
	require.NotNil(t, extraInfo, "改密成功应追加历史口令记录")
	require.Contains(t, *extraInfo, "password_history", "extra_info 应携带历史口令列表键")

	// 改密强制下线：两个客户端类型下的访问令牌全部清空。
	require.Empty(t, env.tokenCache.GetAccessTokens(ctx, authenticationV1.ClientType_admin, uid), "改密后 admin 端令牌应全部吊销")
	require.Empty(t, env.tokenCache.GetAccessTokens(ctx, authenticationV1.ClientType_app, uid), "改密后 app 端令牌应全部吊销")
}

// TestUserProfileServiceSqlite_ChangePassword_RejectsWeakNewPassword 验证
// 新口令不达标的拒绝分支：过短与弱口令（字符类别不足）均报复杂度错误，
// 且不改写已存凭证。
func TestUserProfileServiceSqlite_ChangePassword_RejectsWeakNewPassword(t *testing.T) {
	env := newUserProfileServiceForTest(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const uid = 4243
	seedProfileUserCredential(t, env.entClient, ctx, uid, "profile-user-b", "OldPass@1234")

	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{
		UserId:   uid,
		Username: trans.Ptr("profile-user-b"),
	})

	// 过短。
	_, err := env.svc.ChangePassword(opCtx, &identityV1.ChangePasswordRequest{
		OldPassword: profileEncCredential(t, "OldPass@1234"),
		NewPassword: profileEncCredential(t, "Ab@1"),
	})
	require.Error(t, err, "过短新口令应被拒绝")
	require.Contains(t, err.Error(), "password too short")

	// 弱口令（仅小写+数字两类）。
	_, err = env.svc.ChangePassword(opCtx, &identityV1.ChangePasswordRequest{
		OldPassword: profileEncCredential(t, "OldPass@1234"),
		NewPassword: profileEncCredential(t, "abcdefgh1"),
	})
	require.Error(t, err, "弱新口令应被拒绝")
	require.Contains(t, err.Error(), "does not meet complexity requirements")

	// 凭证未被改写。
	pc := data.NewPasswordCrypto()
	hash, extraInfo := profileRowCredentialHash(t, env.entClient, ctx)
	ok, verr := pc.Verify("OldPass@1234", hash)
	require.NoError(t, verr)
	require.True(t, ok, "被拒绝的改密不应改写已存凭证")
	require.Nil(t, extraInfo, "被拒绝的改密不应追加历史记录")
}

// TestUserProfileServiceSqlite_ChangePassword_BadCredentials 验证密文格式
// 非法、旧口令错误、凭证不存在与缺操作人声明的拒绝分支；凭证均不被改写。
func TestUserProfileServiceSqlite_ChangePassword_BadCredentials(t *testing.T) {
	env := newUserProfileServiceForTest(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const uid = 4244
	seedProfileUserCredential(t, env.entClient, ctx, uid, "profile-user-c", "OldPass@1234")

	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{
		UserId:   uid,
		Username: trans.Ptr("profile-user-c"),
	})

	// 旧口令密文非 base64。
	_, err := env.svc.ChangePassword(opCtx, &identityV1.ChangePasswordRequest{
		OldPassword: "%%%not-base64%%%",
		NewPassword: profileEncCredential(t, "NewPass@5678"),
	})
	require.Error(t, err, "旧口令密文格式非法应被拒绝")
	require.Contains(t, err.Error(), "invalid old credential format")

	// 新口令密文非 base64。
	_, err = env.svc.ChangePassword(opCtx, &identityV1.ChangePasswordRequest{
		OldPassword: profileEncCredential(t, "OldPass@1234"),
		NewPassword: "###not-base64###",
	})
	require.Error(t, err, "新口令密文格式非法应被拒绝")
	require.Contains(t, err.Error(), "invalid new credential format")

	// 旧口令错误。
	_, err = env.svc.ChangePassword(opCtx, &identityV1.ChangePasswordRequest{
		OldPassword: profileEncCredential(t, "WrongOldPass@9"),
		NewPassword: profileEncCredential(t, "NewPass@5678"),
	})
	require.Error(t, err, "旧口令错误应被拒绝")
	require.Contains(t, err.Error(), "invalid old password")

	// 凭证不存在（操作人用户名无对应凭证行）。
	ghostCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{
		UserId:   uid,
		Username: trans.Ptr("ghost-user"),
	})
	_, err = env.svc.ChangePassword(ghostCtx, &identityV1.ChangePasswordRequest{
		OldPassword: profileEncCredential(t, "OldPass@1234"),
		NewPassword: profileEncCredential(t, "NewPass@5678"),
	})
	require.Error(t, err, "不存在的凭证应报 NotFound")
	require.Contains(t, err.Error(), "user credential not found")

	// 缺操作人声明。
	_, err = env.svc.ChangePassword(ctx, &identityV1.ChangePasswordRequest{
		OldPassword: profileEncCredential(t, "OldPass@1234"),
		NewPassword: profileEncCredential(t, "NewPass@5678"),
	})
	require.Error(t, err, "缺操作人声明的改密应被拒绝")

	// 以上分支均不应改写凭证。
	pc := data.NewPasswordCrypto()
	hash, extraInfo := profileRowCredentialHash(t, env.entClient, ctx)
	ok, verr := pc.Verify("OldPass@1234", hash)
	require.NoError(t, verr)
	require.True(t, ok, "全部拒绝分支后已存凭证不应被改写")
	require.Nil(t, extraInfo, "拒绝分支不应追加历史记录")
}

// TestUserProfileServiceSqlite_ChangePassword_HistoryReplayRejected 验证
// 等保历史口令策略：改密成功后再改回近期用过的口令被拒，凭证保持新城。
func TestUserProfileServiceSqlite_ChangePassword_HistoryReplayRejected(t *testing.T) {
	env := newUserProfileServiceForTest(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const uid = 4245
	seedProfileUserCredential(t, env.entClient, ctx, uid, "profile-user-d", "OldPass@1234")

	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{
		UserId:   uid,
		Username: trans.Ptr("profile-user-d"),
	})

	// 第一次改密：OldPass@1234 → NewPass@5678（成功）。
	_, err := env.svc.ChangePassword(opCtx, &identityV1.ChangePasswordRequest{
		OldPassword: profileEncCredential(t, "OldPass@1234"),
		NewPassword: profileEncCredential(t, "NewPass@5678"),
	})
	require.NoError(t, err)

	// 回放近期口令：旧哈希已进历史列表，应被拒。
	_, err = env.svc.ChangePassword(opCtx, &identityV1.ChangePasswordRequest{
		OldPassword: profileEncCredential(t, "NewPass@5678"),
		NewPassword: profileEncCredential(t, "OldPass@1234"),
	})
	require.Error(t, err, "改回近期用过的口令应被拒绝")
	require.Contains(t, err.Error(), "new password must differ from recent passwords")

	// 凭证仍为新城。
	pc := data.NewPasswordCrypto()
	hash, _ := profileRowCredentialHash(t, env.entClient, ctx)
	ok, verr := pc.Verify("NewPass@5678", hash)
	require.NoError(t, verr)
	require.True(t, ok, "被拒的回放不应改写凭证")
}
