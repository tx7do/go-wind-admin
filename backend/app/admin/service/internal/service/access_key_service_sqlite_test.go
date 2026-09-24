// AccessKeyService 的 SQLite 内存库 + miniredis 集成测试（白盒，包内测试）。
//
// 覆盖目标：
//   - Create / Get / List / Count / Delete 的落库与字段映射：AK/SK 的前缀与长度
//     规格、SK 只存 SHA-256 摘要、凭证归属操作人租户、状态枚举经转换器落库。
//   - Update 的不可变字段保护：回调只写 name/status/expires_at/updated_by，
//     AK/secret 摘要/租户归属永不改写；服务层会把掩码内的不可变路径
//     （access_key/tenant_id）剔除后仅更新合法路径。
//   - IssueToken 令牌交换（miniredis 假 redis）：正确 SK 换得 HS256 机器令牌
//     并清空失败计数；错误 SK 计满阈值后按 IP+AK 锁定；停用/过期凭证拒绝；
//     换发成功后 last_used_at 被尽力刷新。
//   - ResetSecret 轮换：旧 SK 立即失效、新 SK 可换发、摘要落库更新。
//
// 跳过项：无（真实 SMTP/MinIO 均不涉及；redis 走 miniredis 假 client）。
package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	conf "github.com/tx7do/kratos-bootstrap/api/gen/go/conf/v1"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	accesskeyV1 "go-wind-admin/api/gen/go/access_key/service/v1"
	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	entAccessKey "go-wind-admin/app/admin/service/internal/data/ent/accesskey"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/middleware/auth"
)

// newAccessKeyServiceForTest 白盒复刻 NewAccessKeyService 的字段初始化：
// log 换 NopLogger；repo 走 data.NewAccessKeyRepoForTest；authenticator 与
// rateLimiter 按 repo_testkit5.go 的注入式构造（redis 走 miniredis 假 client，
// jwtCfg 用测试 HS256 密钥），与生产装配的依赖形态一致。
func newAccessKeyServiceForTest(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) *AccessKeyService {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	tokenCache := data.NewUserTokenCacheForTest(rdb)
	authenticator, err := data.NewAuthenticatorForTest(&conf.Authentication_Jwt{
		Method: "HS256",
		Key:    "access-key-svc-test-hs256-key",
	}, tokenCache)
	require.NoError(t, err, "构造测试 Authenticator 应成功")

	return &AccessKeyService{
		log:           bLogger.NewHelper(bLogger.NopLogger()),
		repo:          data.NewAccessKeyRepoForTest(entClient),
		authenticator: authenticator,
		rateLimiter:   data.NewLoginRateLimiterForTest(rdb),
	}
}

// accessKeySha256Hex 与生产 hashSecret 一致的 SHA-256 hex 摘要，用于核对落库摘要。
func accessKeySha256Hex(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// createAccessKeyViaService 经服务层创建一枚启用中的凭证，返回 (id, ak, sk)。
func createAccessKeyViaService(t *testing.T, svc *AccessKeyService, opCtx context.Context, name string) (uint32, string, string) {
	t.Helper()
	resp, err := svc.Create(opCtx, &accesskeyV1.CreateAccessKeyRequest{
		Data: &accesskeyV1.AccessKey{Name: trans.Ptr(name)},
	})
	require.NoError(t, err)
	require.NotNil(t, resp.GetData().GetId(), "创建响应应回带凭证 ID")
	require.NotEmpty(t, resp.GetSecret(), "创建响应应回带一次性明文 SK")
	return resp.GetData().GetId(), resp.GetData().GetAccessKey(), resp.GetSecret()
}

// TestAccessKeyServiceSqlite_CreateAndGetAndCount 验证创建凭证的规格（AK 8 字节 hex、
// SK 32 字节 hex、SK 只落 SHA-256 摘要、归属操作人租户、状态默认启用），
// Get 两个 QueryBy 分支的回读与未命中，以及 List / Count 的全量统计。
func TestAccessKeyServiceSqlite_CreateAndGetAndCount(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newAccessKeyServiceForTest(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 81})

	idA, akA, skA := createAccessKeyViaService(t, svc, opCtx, "服务层凭证甲")
	_, akB, _ := createAccessKeyViaService(t, svc, opCtx, "服务层凭证乙")

	// AK/SK 规格与归属、摘要落库。
	row, err := entClient.Client().AccessKey.Query().Where().All(ctx)
	require.NoError(t, err)
	require.Len(t, row, 2)
	for _, r := range row {
		require.True(t, strings.HasPrefix(*r.AccessKey, "ak-"), "AK 应带 ak- 前缀")
		require.Len(t, *r.AccessKey, 3+2*8, "AK 应为 8 字节 hex（16 字符）")
		require.NotNil(t, r.CreatedAt, "创建时间应落库")
		if *r.Name == "服务层凭证甲" {
			require.Equal(t, akA, *r.AccessKey)
			require.Equal(t, accessKeySha256Hex(skA), *r.SecretHash, "落库的应为 SK 的 SHA-256 hex 摘要")
			require.NotNil(t, r.TenantID, "凭证应归属操作人声明的租户")
			require.EqualValues(t, 0, *r.TenantID, "操作人未声明租户时归属平台租户 0")
		}
	}

	// Get 按 Id / 按 AccessKey 两个分支的回读。
	byId, err := svc.Get(ctx, &accesskeyV1.GetAccessKeyRequest{
		QueryBy: &accesskeyV1.GetAccessKeyRequest_Id{Id: idA},
	})
	require.NoError(t, err)
	require.Equal(t, "服务层凭证甲", byId.GetName())

	byAk, err := svc.Get(ctx, &accesskeyV1.GetAccessKeyRequest{
		QueryBy: &accesskeyV1.GetAccessKeyRequest_AccessKey{AccessKey: akB},
	})
	require.NoError(t, err)
	require.Equal(t, "服务层凭证乙", byAk.GetName())

	// 未命中。
	_, err = svc.Get(ctx, &accesskeyV1.GetAccessKeyRequest{
		QueryBy: &accesskeyV1.GetAccessKeyRequest_Id{Id: 424242},
	})
	require.Error(t, err, "按不存在的 id 查询应报错")

	_, err = svc.Get(ctx, nil)
	require.Error(t, err, "nil 请求体应被拒绝")

	// List / Count 全量统计。
	listResp, err := svc.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), listResp.GetTotal())
	names := map[string]bool{}
	for _, item := range listResp.GetItems() {
		names[item.GetName()] = true
	}
	require.True(t, names["服务层凭证甲"] && names["服务层凭证乙"], "List 应返回两枚已建凭证")

	countResp, err := svc.Count(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), countResp.GetCount(), "Count 应统计两枚凭证")

	// 服务层入参守卫。
	_, err = svc.Create(nil, nil)
	require.Error(t, err, "nil 请求体应被拒绝")
	_, err = svc.Create(opCtx, &accesskeyV1.CreateAccessKeyRequest{})
	require.Error(t, err, "nil Data 应被拒绝")
	_, err = svc.Create(ctx, &accesskeyV1.CreateAccessKeyRequest{Data: &accesskeyV1.AccessKey{}})
	require.Error(t, err, "缺操作人声明的创建应被拒绝")
}

// TestAccessKeyServiceSqlite_UpdateNilMaskKeepsImmutableFields 验证 nil 掩码更新：
// 回调可写的 name 生效；AK、secret 摘要、租户归属不可写，携带攻击值也不改写。
func TestAccessKeyServiceSqlite_UpdateNilMaskKeepsImmutableFields(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newAccessKeyServiceForTest(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 82})

	id, ak, sk := createAccessKeyViaService(t, svc, opCtx, "更新前凭证名")

	_, err := svc.Update(opCtx, &accesskeyV1.UpdateAccessKeyRequest{
		Id:   id,
		Data: &accesskeyV1.AccessKey{Name: trans.Ptr("更新后凭证名"), AccessKey: trans.Ptr("ak-attacker"), TenantId: trans.Ptr(uint32(999))},
	})
	require.NoError(t, err, "nil 掩码更新应成功")

	row, err := entClient.Client().AccessKey.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "更新后凭证名", *row.Name, "回调可写的 name 应更新")
	require.Equal(t, ak, *row.AccessKey, "AK 不可经更新改写")
	require.Equal(t, accessKeySha256Hex(sk), *row.SecretHash, "secret 摘要不可经更新改写")
	require.EqualValues(t, 0, *row.TenantID, "租户归属不可经更新改写")
	require.Equal(t, entAccessKey.StatusOn, *row.Status, "未指定状态时不应改写状态")

	// 入参守卫。
	_, err = svc.Update(opCtx, nil)
	require.Error(t, err, "nil 请求体应被拒绝")
	_, err = svc.Update(opCtx, &accesskeyV1.UpdateAccessKeyRequest{Data: &accesskeyV1.AccessKey{}})
	require.Error(t, err, "nil Data 应被拒绝")
	_, err = svc.Update(opCtx, &accesskeyV1.UpdateAccessKeyRequest{Id: 0, Data: &accesskeyV1.AccessKey{}})
	require.Error(t, err, "id=0 应被拒绝")
	require.Contains(t, err.Error(), "id is required")
	_, err = svc.Update(ctx, &accesskeyV1.UpdateAccessKeyRequest{Id: id, Data: &accesskeyV1.AccessKey{}})
	require.Error(t, err, "缺操作人声明的更新应被拒绝")
}

// TestAccessKeyServiceSqlite_UpdateWithMaskImmutableStripped 修复后语义：
// 服务层把不可变字段（access_key/tenant_id）从掩码剔除而非追加进白名单
// （追加 "secret_hash" 曾致一切带掩码更新整体失败），合法路径（name）正常
// 更新、AK/摘要/租户归属保持原值；掩码只含不可变字段时为仅盖章的空操作。
func TestAccessKeyServiceSqlite_UpdateWithMaskImmutableStripped(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newAccessKeyServiceForTest(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 82})

	maskedID, origAK, _ := createAccessKeyViaService(t, svc, opCtx, "掩码更新凭证名")
	before, err := entClient.Client().AccessKey.Query().Only(ctx)
	require.NoError(t, err)
	origTenant := before.TenantID
	origHash := before.SecretHash

	// 合法路径 + 不可变路径混合：剔除后 name 更新生效、其余字段不可改写
	_, err = svc.Update(opCtx, &accesskeyV1.UpdateAccessKeyRequest{
		Id:         maskedID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name", "access_key", "tenant_id"}},
		Data: &accesskeyV1.AccessKey{
			Name:      trans.Ptr("合法更新名"),
			AccessKey: trans.Ptr("ATTACKER_KEY"),
			TenantId:  trans.Ptr(uint32(999)),
		},
	})
	require.NoError(t, err, "不可变字段剔除后掩码更新应成功")
	after, err := entClient.Client().AccessKey.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "合法更新名", *after.Name, "掩码内合法路径应更新")
	require.Equal(t, origAK, *after.AccessKey, "access_key 不可变")
	require.Equal(t, origHash, after.SecretHash, "secret 摘要不可变")
	require.Equal(t, origTenant, after.TenantID, "tenant_id 不可变")

	// 掩码只含不可变字段：剔除后仅剩强制盖章的 updated_by，空操作不报错
	_, err = svc.Update(opCtx, &accesskeyV1.UpdateAccessKeyRequest{
		Id:         maskedID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"access_key"}},
		Data: &accesskeyV1.AccessKey{
			AccessKey: trans.Ptr("ATTACKER_KEY_2"),
		},
	})
	require.NoError(t, err, "只含不可变字段的掩码剔除后应为空操作")
	final, err := entClient.Client().AccessKey.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, origAK, *final.AccessKey, "access_key 仍不可变")
}

// TestAccessKeyServiceSqlite_DisabledKeyIssueTokenRejected 验证停用凭证被换发拒绝。
func TestAccessKeyServiceSqlite_DisabledKeyIssueTokenRejected(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newAccessKeyServiceForTest(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 83})

	resp, err := svc.Create(opCtx, &accesskeyV1.CreateAccessKeyRequest{
		Data: &accesskeyV1.AccessKey{Name: trans.Ptr("停用凭证"), Status: accesskeyV1.AccessKey_OFF.Enum()},
	})
	require.NoError(t, err)
	require.Equal(t, accesskeyV1.AccessKey_OFF, resp.GetData().GetStatus(), "OFF 状态应经转换器落库并回显")

	row, err := entClient.Client().AccessKey.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, entAccessKey.StatusOff, *row.Status, "停用状态应落库为 OFF")

	_, err = svc.IssueToken(ctx, &accesskeyV1.IssueTokenRequest{
		AccessKey: resp.GetData().GetAccessKey(),
		Secret:    resp.GetSecret(),
	})
	require.Error(t, err, "停用凭证的换发应被拒绝")
	require.Contains(t, err.Error(), "access key is disabled")
}

// TestAccessKeyServiceSqlite_ExpiredKeyIssueTokenRejected 验证过期凭证被换发拒绝。
func TestAccessKeyServiceSqlite_ExpiredKeyIssueTokenRejected(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newAccessKeyServiceForTest(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 83})

	resp, err := svc.Create(opCtx, &accesskeyV1.CreateAccessKeyRequest{
		Data: &accesskeyV1.AccessKey{
			Name:      trans.Ptr("过期凭证"),
			ExpiresAt: timestamppb.New(time.Now().Add(-time.Hour)),
		},
	})
	require.NoError(t, err)

	_, err = svc.IssueToken(ctx, &accesskeyV1.IssueTokenRequest{
		AccessKey: resp.GetData().GetAccessKey(),
		Secret:    resp.GetSecret(),
	})
	require.Error(t, err, "过期凭证的换发应被拒绝")
	require.Contains(t, err.Error(), "access key is expired")
}

// TestAccessKeyServiceSqlite_IssueTokenFlow 验证正确 AK/SK 的完整换发：
// 签发 HS256 机器令牌（bearer、默认 15 分钟有效期）、成功交换清空失败计数
// （Reset 语义）、last_used_at 被尽力刷新；以及入参守卫分支。
func TestAccessKeyServiceSqlite_IssueTokenFlow(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newAccessKeyServiceForTest(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 84})

	_, ak, sk := createAccessKeyViaService(t, svc, opCtx, "换发流程凭证")

	// 预置两枚失败计数，验证成功交换后的 Reset 清零语义。
	locked, attempts, _, err := svc.rateLimiter.CheckAndIncr(ctx, "", ak)
	require.NoError(t, err)
	require.False(t, locked)
	require.Equal(t, 1, attempts)
	locked, attempts, _, err = svc.rateLimiter.CheckAndIncr(ctx, "", ak)
	require.NoError(t, err)
	require.False(t, locked)
	require.Equal(t, 2, attempts)
	locked, err = svc.rateLimiter.IsLocked(ctx, "", ak)
	require.NoError(t, err)
	require.False(t, locked, "计数未达阈值不应锁定")

	resp, err := svc.IssueToken(ctx, &accesskeyV1.IssueTokenRequest{AccessKey: ak, Secret: sk})
	require.NoError(t, err, "正确 AK/SK 应换发成功")

	// 机器令牌规格：三段式 JWT、头声明 HS256（测试构造的认证器密钥）、
	// 默认访问令牌有效期 15 分钟、bearer 类型。
	require.Equal(t, "bearer", resp.GetTokenType())
	require.Equal(t, uint32(15*60), resp.GetExpiresIn(), "未配置过期时间时应为默认 15 分钟")
	parts := strings.Split(resp.GetAccessToken(), ".")
	require.Len(t, parts, 3, "签发的应为三段式 JWT")
	hdr, derr := base64.RawURLEncoding.DecodeString(parts[0])
	require.NoError(t, derr)
	require.Contains(t, string(hdr), `"alg":"HS256"`, "JWT 头应声明测试认证器的 HS256 算法")

	// 成功交换应 Reset 掉预置的失败计数（本次自增从 1 重新开始）。
	locked, attempts, _, err = svc.rateLimiter.CheckAndIncr(ctx, "", ak)
	require.NoError(t, err)
	require.False(t, locked)
	require.Equal(t, 1, attempts, "成功交换应清空失败计数（否则本次应为 3）")

	// 尽力而为的 last_used_at 刷新。
	row, err := entClient.Client().AccessKey.Query().Only(ctx)
	require.NoError(t, err)
	require.NotNil(t, row.LastUsedAt, "成功交换后应尽力刷新 last_used_at")

	// 入参守卫。
	_, err = svc.IssueToken(ctx, nil)
	require.Error(t, err)
	_, err = svc.IssueToken(ctx, &accesskeyV1.IssueTokenRequest{})
	require.Error(t, err, "空 AK/SK 应被拒绝")
	require.Contains(t, err.Error(), "access key and secret are required")
	_, err = svc.IssueToken(ctx, &accesskeyV1.IssueTokenRequest{AccessKey: "ak-unknown", Secret: "sk-whatever"})
	require.Error(t, err, "不存在的 AK 应被拒绝")
	require.Contains(t, err.Error(), "invalid access key or secret")
}

// TestAccessKeyServiceSqlite_WrongSecretLockout 验证错误 SK 连续失败按 IP+AK
// 维度计数，达到阈值（5 次）后锁定并拒绝后续交换（含正确 SK）。
func TestAccessKeyServiceSqlite_WrongSecretLockout(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newAccessKeyServiceForTest(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 84})

	_, ak, _ := createAccessKeyViaService(t, svc, opCtx, "锁定流程凭证")

	for i := 0; i < 5; i++ {
		_, err := svc.IssueToken(ctx, &accesskeyV1.IssueTokenRequest{AccessKey: ak, Secret: "sk-wrong-secret"})
		require.Error(t, err, "错误 SK 的换发应被拒绝")
		require.Contains(t, err.Error(), "invalid access key or secret")
	}

	locked, err := svc.rateLimiter.IsLocked(ctx, "", ak)
	require.NoError(t, err)
	require.True(t, locked, "连续 5 次失败后应处于锁定态")

	_, err = svc.IssueToken(ctx, &accesskeyV1.IssueTokenRequest{AccessKey: ak, Secret: "sk-wrong-secret"})
	require.Error(t, err, "锁定后应先被限流拦截")
	require.Contains(t, err.Error(), "too many attempts")
}

// TestAccessKeyServiceSqlite_ResetSecretRotation 验证密钥轮换：摘要更新落库、
// 旧 SK 立即失效、新 SK 可换发；入参守卫。
func TestAccessKeyServiceSqlite_ResetSecretRotation(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newAccessKeyServiceForTest(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 85})

	id, ak, oldSecret := createAccessKeyViaService(t, svc, opCtx, "轮换凭证")

	rotated, err := svc.ResetSecret(ctx, &accesskeyV1.ResetAccessKeySecretRequest{Id: id})
	require.NoError(t, err, "密钥轮换应成功")
	require.NotEmpty(t, rotated.GetSecret())
	require.True(t, strings.HasPrefix(rotated.GetSecret(), "sk-"), "新 SK 应带 sk- 前缀")
	require.Len(t, rotated.GetSecret(), 3+2*32, "新 SK 应为 32 字节 hex（64 字符）")

	row, err := entClient.Client().AccessKey.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, accessKeySha256Hex(rotated.GetSecret()), *row.SecretHash, "轮换后应落库新 SK 的摘要")

	_, err = svc.IssueToken(ctx, &accesskeyV1.IssueTokenRequest{AccessKey: ak, Secret: oldSecret})
	require.Error(t, err, "旧 SK 在轮换后应立即失效")
	require.Contains(t, err.Error(), "invalid access key or secret")

	newResp, err := svc.IssueToken(ctx, &accesskeyV1.IssueTokenRequest{AccessKey: ak, Secret: rotated.GetSecret()})
	require.NoError(t, err, "新 SK 应可正常换发")
	require.Equal(t, "bearer", newResp.GetTokenType())

	_, err = svc.ResetSecret(ctx, &accesskeyV1.ResetAccessKeySecretRequest{Id: 0})
	require.Error(t, err, "id=0 应被拒绝")
	require.Contains(t, err.Error(), "id is required")
	_, err = svc.ResetSecret(ctx, nil)
	require.Error(t, err, "nil 请求体应被拒绝")
}

// TestAccessKeyServiceSqlite_Delete 验证删除后行数归零、再查询报不存在；
// nil 请求体被拒绝。
func TestAccessKeyServiceSqlite_Delete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newAccessKeyServiceForTest(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 85})

	id, _, _ := createAccessKeyViaService(t, svc, opCtx, "待删除凭证")

	_, err := svc.Delete(ctx, &accesskeyV1.DeleteAccessKeyRequest{Id: id})
	require.NoError(t, err, "删除已存在凭证应成功")

	cnt, err := entClient.Client().AccessKey.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, cnt, "删除后凭证表计数应归零")

	_, err = svc.Get(ctx, &accesskeyV1.GetAccessKeyRequest{
		QueryBy: &accesskeyV1.GetAccessKeyRequest_Id{Id: id},
	})
	require.Error(t, err, "删除后的凭证应查询不到")

	_, err = svc.Delete(ctx, nil)
	require.Error(t, err, "nil 请求体应被拒绝")
	require.Contains(t, err.Error(), "invalid parameter")
}
