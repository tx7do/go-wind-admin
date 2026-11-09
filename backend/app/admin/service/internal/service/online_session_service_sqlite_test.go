// OnlineSessionService 的 miniredis 集成测试（白盒，包内测试）。
//
// 会话视图基于 Redis 令牌对元数据（us:{ct}:{uid}:{jti} 键，随令牌签发写入、
// 随吊销/过期删除）；本服务只做读取与转发吊销。测试经 miniredis 假 client
// 培养真实的键值读写往返，覆盖：
//   - ListOnlineSession：全量视图的登录时间倒序、关键字（用户名/IP）过滤、
//     内存分页（page/pageSize、越界清空、缺省前 20 条）。
//   - ListMyOnlineSession：按当前用户过滤、current 标记当前会话、倒序。
//   - RevokeMyOnlineSession：仅本人会话可定位、吊销后会话元数据与访问令牌
//     一并消失、重复吊销明确报错、jti 缺失拒绝。
//   - ForceLogoutSession：强制下线指定会话（元数据+令牌清理）、入参守卫。
//
// 跳过项：无（全部依赖经 miniredis 注入，无真实外部件）。
package service

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	conf "github.com/tx7do/kratos-bootstrap/api/gen/go/conf/v1"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	onlineSessionV1 "go-wind-admin/api/gen/go/online_session/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/middleware/auth"
)

// newOnlineSessionServiceForTest 白盒复刻 NewOnlineSessionService 的字段初始化：
// log 换 NopLogger；authenticator 按 repo_testkit5.go 注入式构造（redis 走
// miniredis 假 client、jwtCfg 用测试 HS256 密钥），与生产装配依赖形态一致。
// 返回的 tokenCache 即注入 authenticator 的同一实例（跨包私有字段无法直接取，
// 供测试培养/断言访问令牌键的读写往返）。
func newOnlineSessionServiceForTest(t *testing.T) (*OnlineSessionService, *data.UserTokenCache) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	tokenCache := data.NewUserTokenCacheForTest(rdb)
	authenticator, err := data.NewAuthenticatorForTest(&conf.Authentication_Jwt{
		Method: "HS256",
		Key:    "online-session-svc-test-hs256-key",
	}, tokenCache)
	require.NoError(t, err, "构造测试 Authenticator 应成功")

	return &OnlineSessionService{
		log:           bLogger.NewHelper(bLogger.NopLogger()),
		authenticator: authenticator,
	}, tokenCache
}

// seedOnlineSession 经真实 SaveSessionMeta 路径培养一枚会话元数据，
// 并同步经真实 AddAccessToken 路径培养对应的访问令牌键（at:{ct}:{uid}:{jti}），
// 用于吊销清理断言。
func seedOnlineSession(
	t *testing.T,
	svc *OnlineSessionService,
	tokenCache *data.UserTokenCache,
	ctx context.Context,
	ct authenticationV1.ClientType,
	uid uint64,
	jti, username, ip string,
	loginAt int64,
) {
	t.Helper()
	require.NoError(t, svc.authenticator.SaveSessionMeta(ctx, ct, uid, jti, &data.SessionMeta{
		Username:  username,
		TenantId:  uid,
		Ip:        ip,
		UserAgent: "test-ua/" + jti,
		DeviceId:  "dev-" + jti,
		LoginAt:   loginAt,
	}, time.Hour), "培养会话元数据应成功")
	require.NoError(t, tokenCache.AddAccessToken(ctx, ct, uid, jti, "tok-"+jti, time.Hour), "培养访问令牌键应成功")
}

// onlineAdminOpCtx 平台管理员操作者上下文：在线会话台账/强踢已加
// requireTenantOrPlatformAdmin 服务内闸门，相关用例以此调用。
func onlineAdminOpCtx(ctx context.Context) context.Context {
	return auth.NewContext(ctx, &authenticationV1.UserTokenPayload{
		UserId:          1,
		IsPlatformAdmin: trans.Ptr(true),
	})
}

// TestOnlineSessionServiceSqlite_ListSortsAndPaginates 验证全量视图按登录时间
// 倒序、字段映射（用户名/IP/UA/设备/租户/客户端类型）、内存分页与越界清空。
func TestOnlineSessionServiceSqlite_ListSortsAndPaginates(t *testing.T) {
	svc, tokenCache := newOnlineSessionServiceForTest(t)
	ctx := enttest.NewSystemContext(context.Background())

	seedOnlineSession(t, svc, tokenCache, ctx, authenticationV1.ClientType_admin, 101, "jti-101a", "user-101", "10.1.1.1", 1000)
	seedOnlineSession(t, svc, tokenCache, ctx, authenticationV1.ClientType_admin, 102, "jti-102", "user-102", "10.2.2.2", 2000)
	seedOnlineSession(t, svc, tokenCache, ctx, authenticationV1.ClientType_app, 101, "jti-101b", "user-101", "10.1.1.3", 3000)

	// 全量：按登录时间倒序（3000 → 2000 → 1000），字段按元数据映射。
	all, err := svc.ListOnlineSession(onlineAdminOpCtx(ctx), &onlineSessionV1.ListOnlineSessionRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(3), all.GetTotal())
	require.Len(t, all.GetItems(), 3, "缺省分页应返回前 20 条，此处共 3 条")
	require.Equal(t, "jti-101b", all.GetItems()[0].GetJti(), "最新登录（3000）应排首")
	require.Equal(t, "jti-102", all.GetItems()[1].GetJti(), "次新登录（2000）应排第二")
	require.Equal(t, "jti-101a", all.GetItems()[2].GetJti(), "最早登录（1000）应排末")
	require.EqualValues(t, 3000, all.GetItems()[0].GetLoginAt().GetSeconds(), "登录时间应按元数据映射")
	require.Equal(t, "user-101", all.GetItems()[0].GetUsername(), "用户名应按元数据映射")
	require.Equal(t, "10.1.1.3", all.GetItems()[0].GetIpAddress(), "IP 应按元数据映射")
	require.Equal(t, "test-ua/jti-101b", all.GetItems()[0].GetUserAgent(), "UA 应按元数据映射")
	require.Equal(t, "dev-jti-101b", all.GetItems()[0].GetDeviceId(), "设备 ID 应按元数据映射")
	require.EqualValues(t, 101, all.GetItems()[0].GetTenantId(), "租户 ID 应按元数据映射")
	require.EqualValues(t, authenticationV1.ClientType_app, all.GetItems()[0].GetClientType(), "客户端类型应按键名还原")

	// 分页：page=1/pageSize=2 取前两条（倒序后最新的两条）。
	page1, err := svc.ListOnlineSession(onlineAdminOpCtx(ctx), &onlineSessionV1.ListOnlineSessionRequest{
		Page:     trans.Ptr(uint32(1)),
		PageSize: trans.Ptr(uint32(2)),
	})
	require.NoError(t, err)
	require.Equal(t, uint64(3), page1.GetTotal(), "总数应保持全量")
	require.Len(t, page1.GetItems(), 2)
	require.Equal(t, "jti-101b", page1.GetItems()[0].GetJti())
	require.Equal(t, "jti-102", page1.GetItems()[1].GetJti())

	// page=2/pageSize=2 取末一条。
	page2, err := svc.ListOnlineSession(onlineAdminOpCtx(ctx), &onlineSessionV1.ListOnlineSessionRequest{
		Page:     trans.Ptr(uint32(2)),
		PageSize: trans.Ptr(uint32(2)),
	})
	require.NoError(t, err)
	require.Len(t, page2.GetItems(), 1)
	require.Equal(t, "jti-101a", page2.GetItems()[0].GetJti())

	// 越界页：条目清空但总数保持。
	page3, err := svc.ListOnlineSession(onlineAdminOpCtx(ctx), &onlineSessionV1.ListOnlineSessionRequest{
		Page:     trans.Ptr(uint32(3)),
		PageSize: trans.Ptr(uint32(2)),
	})
	require.NoError(t, err)
	require.Equal(t, uint64(3), page3.GetTotal())
	require.Empty(t, page3.GetItems(), "越界页条目应清空")
}

// TestOnlineSessionServiceSqlite_ListKeywordFilter 验证关键字过滤：
// 命中用户名或 IP 之一即保留；纯空白关键字视为无过滤；未命中清空。
func TestOnlineSessionServiceSqlite_ListKeywordFilter(t *testing.T) {
	svc, tokenCache := newOnlineSessionServiceForTest(t)
	ctx := enttest.NewSystemContext(context.Background())

	seedOnlineSession(t, svc, tokenCache, ctx, authenticationV1.ClientType_admin, 201, "jti-201a", "user-201", "10.3.3.1", 1000)
	seedOnlineSession(t, svc, tokenCache, ctx, authenticationV1.ClientType_admin, 202, "jti-202", "user-202", "10.4.4.4", 2000)

	byUser, err := svc.ListOnlineSession(onlineAdminOpCtx(ctx), &onlineSessionV1.ListOnlineSessionRequest{
		Keyword: trans.Ptr("user-201"),
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), byUser.GetTotal(), "按用户名关键字应命中对应用户的会话")
	require.Len(t, byUser.GetItems(), 1)
	require.Equal(t, "jti-201a", byUser.GetItems()[0].GetJti())

	byIp, err := svc.ListOnlineSession(onlineAdminOpCtx(ctx), &onlineSessionV1.ListOnlineSessionRequest{
		Keyword: trans.Ptr("10.4.4.4"),
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), byIp.GetTotal(), "按 IP 关键字应命中对应会话")
	require.Len(t, byIp.GetItems(), 1)
	require.Equal(t, "jti-202", byIp.GetItems()[0].GetJti())

	byBlank, err := svc.ListOnlineSession(onlineAdminOpCtx(ctx), &onlineSessionV1.ListOnlineSessionRequest{
		Keyword: trans.Ptr("   "),
	})
	require.NoError(t, err)
	require.Equal(t, uint64(2), byBlank.GetTotal(), "纯空白关键字等效无过滤")

	byMiss, err := svc.ListOnlineSession(onlineAdminOpCtx(ctx), &onlineSessionV1.ListOnlineSessionRequest{
		Keyword: trans.Ptr("nobody-here"),
	})
	require.NoError(t, err)
	require.Equal(t, uint64(0), byMiss.GetTotal(), "未命中的关键字应清空结果")
	require.Empty(t, byMiss.GetItems())
}

// TestOnlineSessionServiceSqlite_ListMySessionsCurrentFlag 验证本人视图：
// 只返回本人会话、current 仅标记当前请求会话、按登录时间倒序。
func TestOnlineSessionServiceSqlite_ListMySessionsCurrentFlag(t *testing.T) {
	svc, tokenCache := newOnlineSessionServiceForTest(t)
	ctx := enttest.NewSystemContext(context.Background())

	seedOnlineSession(t, svc, tokenCache, ctx, authenticationV1.ClientType_admin, 301, "jti-301a", "user-301", "10.5.5.1", 1000)
	seedOnlineSession(t, svc, tokenCache, ctx, authenticationV1.ClientType_admin, 301, "jti-301b", "user-301", "10.5.5.2", 3000)
	seedOnlineSession(t, svc, tokenCache, ctx, authenticationV1.ClientType_admin, 302, "jti-302", "user-302", "10.6.6.6", 2000)

	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{
		UserId: 301,
		Jti:    trans.Ptr("jti-301b"),
	})

	my, err := svc.ListMyOnlineSession(opCtx, &onlineSessionV1.ListMyOnlineSessionRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), my.GetTotal(), "本人视图只应返回本人会话")
	require.Len(t, my.GetItems(), 2)
	require.Equal(t, "jti-301b", my.GetItems()[0].GetJti(), "本人会话按登录时间倒序，最新（3000）应排首")
	require.Equal(t, "jti-301a", my.GetItems()[1].GetJti())
	require.True(t, my.GetItems()[0].GetCurrent(), "与操作人 jti 一致的会话应标记 current")
	require.False(t, my.GetItems()[1].GetCurrent(), "非当前会话不应标记 current")

	// 操作人未声明 jti：全部不标记 current。
	noJtiCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 301})
	myNoJti, err := svc.ListMyOnlineSession(noJtiCtx, &onlineSessionV1.ListMyOnlineSessionRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), myNoJti.GetTotal())
	for _, item := range myNoJti.GetItems() {
		require.False(t, item.GetCurrent(), "操作人未声明 jti 时不应有任何 current 标记")
	}

	// 缺操作人声明：直接报错。
	_, err = svc.ListMyOnlineSession(ctx, &onlineSessionV1.ListMyOnlineSessionRequest{})
	require.Error(t, err, "缺操作人声明的查询应被拒绝")
}

// TestOnlineSessionServiceSqlite_RevokeMySession 验证本人会话自助下线：
// 吊销后访问令牌与会话元数据一并清理、重复吊销报会话不存在、
// 他人会话不可达、jti 缺失拒绝。
func TestOnlineSessionServiceSqlite_RevokeMySession(t *testing.T) {
	svc, tokenCache := newOnlineSessionServiceForTest(t)
	ctx := enttest.NewSystemContext(context.Background())

	const uid = 401
	seedOnlineSession(t, svc, tokenCache, ctx, authenticationV1.ClientType_admin, uid, "jti-401a", "user-401", "10.7.7.1", 1000)
	require.Len(t, tokenCache.GetAccessTokens(ctx, authenticationV1.ClientType_admin, uid), 1, "培养的访问令牌应可读取")

	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: uid})

	resp, err := svc.RevokeMyOnlineSession(opCtx, &onlineSessionV1.RevokeMyOnlineSessionRequest{
		ClientType: authenticationV1.ClientType_admin.Enum(),
		Jti:        trans.Ptr("jti-401a"),
	})
	require.NoError(t, err, "下线本人会话应成功")
	require.NotNil(t, resp)

	// 吊销后：会话元数据与访问令牌键一并消失。
	meta, err := svc.authenticator.GetSessionMeta(ctx, authenticationV1.ClientType_admin, uid, "jti-401a")
	require.NoError(t, err)
	require.Nil(t, meta, "吊销后会话元数据应消失")
	require.Empty(t, tokenCache.GetAccessTokens(ctx, authenticationV1.ClientType_admin, uid), "吊销后访问令牌应清理")

	// 重复吊销：明确报会话不存在。
	_, err = svc.RevokeMyOnlineSession(opCtx, &onlineSessionV1.RevokeMyOnlineSessionRequest{
		ClientType: authenticationV1.ClientType_admin.Enum(),
		Jti:        trans.Ptr("jti-401a"),
	})
	require.Error(t, err, "重复下线应报会话不存在")
	require.Contains(t, err.Error(), "session not found or already offline")

	// 他人会话经 (ct, 本人uid, jti) 键定位，天然不可达。
	seedOnlineSession(t, svc, tokenCache, ctx, authenticationV1.ClientType_admin, 402, "jti-402", "user-402", "10.8.8.8", 1000)
	_, err = svc.RevokeMyOnlineSession(opCtx, &onlineSessionV1.RevokeMyOnlineSessionRequest{
		ClientType: authenticationV1.ClientType_admin.Enum(),
		Jti:        trans.Ptr("jti-402"),
	})
	require.Error(t, err, "下线他人会话应报会话不存在")
	require.Contains(t, err.Error(), "session not found or already offline")
	otherMeta, err := svc.authenticator.GetSessionMeta(ctx, authenticationV1.ClientType_admin, 402, "jti-402")
	require.NoError(t, err)
	require.NotNil(t, otherMeta, "他人会话不应被波及")

	// jti 缺失。
	_, err = svc.RevokeMyOnlineSession(opCtx, &onlineSessionV1.RevokeMyOnlineSessionRequest{
		ClientType: authenticationV1.ClientType_admin.Enum(),
		Jti:        trans.Ptr(""),
	})
	require.Error(t, err, "空 jti 应被拒绝")
	require.Contains(t, err.Error(), "jti is required")

	_, err = svc.RevokeMyOnlineSession(ctx, &onlineSessionV1.RevokeMyOnlineSessionRequest{
		ClientType: authenticationV1.ClientType_admin.Enum(),
		Jti:        trans.Ptr("jti-401a"),
	})
	require.Error(t, err, "缺操作人声明的下线应被拒绝")
}

// TestOnlineSessionServiceSqlite_ForceLogoutSession 验证强制下线指定会话：
// 会话元数据与访问令牌清理、入参守卫。
func TestOnlineSessionServiceSqlite_ForceLogoutSession(t *testing.T) {
	svc, tokenCache := newOnlineSessionServiceForTest(t)
	ctx := enttest.NewSystemContext(context.Background())

	const uid = 501
	seedOnlineSession(t, svc, tokenCache, ctx, authenticationV1.ClientType_admin, uid, "jti-501", "user-501", "10.9.9.9", 1000)

	// 平台闸门：非管理员的强踢拒绝
	plainCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: uid})
	_, err := svc.ForceLogoutSession(plainCtx, &onlineSessionV1.ForceLogoutSessionRequest{
		ClientType: authenticationV1.ClientType_admin.Enum(),
		UserId:     trans.Ptr(uint64(uid)),
		Jti:        trans.Ptr("jti-501"),
	})
	require.Error(t, err, "非租户/平台管理员的强踢应被拒绝")

	// 平台管理员：强踢成功
	adminCtx := onlineAdminOpCtx(ctx)
	resp, err := svc.ForceLogoutSession(adminCtx, &onlineSessionV1.ForceLogoutSessionRequest{
		ClientType: authenticationV1.ClientType_admin.Enum(),
		UserId:     trans.Ptr(uint64(uid)),
		Jti:        trans.Ptr("jti-501"),
	})
	require.NoError(t, err, "强制下线已存在会话应成功")
	require.NotNil(t, resp)

	meta, err := svc.authenticator.GetSessionMeta(ctx, authenticationV1.ClientType_admin, uid, "jti-501")
	require.NoError(t, err)
	require.Nil(t, meta, "强制下线后会话元数据应消失")
	require.Empty(t, tokenCache.GetAccessTokens(ctx, authenticationV1.ClientType_admin, uid), "强制下线后访问令牌应清理")

	// 入参守卫：userId/jti 缺失。
	_, err = svc.ForceLogoutSession(ctx, &onlineSessionV1.ForceLogoutSessionRequest{
		ClientType: authenticationV1.ClientType_admin.Enum(),
		UserId:     trans.Ptr(uint64(0)),
		Jti:        trans.Ptr("jti-501"),
	})
	require.Error(t, err, "userId=0 应被拒绝")
	require.Contains(t, err.Error(), "user id and jti are required")

	_, err = svc.ForceLogoutSession(ctx, &onlineSessionV1.ForceLogoutSessionRequest{
		ClientType: authenticationV1.ClientType_admin.Enum(),
		UserId:     trans.Ptr(uint64(uid)),
		Jti:        trans.Ptr(""),
	})
	require.Error(t, err, "空 jti 应被拒绝")
	require.Contains(t, err.Error(), "user id and jti are required")
}
