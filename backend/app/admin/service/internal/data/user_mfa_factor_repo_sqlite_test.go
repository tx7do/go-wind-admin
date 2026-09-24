package data

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	entUserMfaFactor "go-wind-admin/app/admin/service/internal/data/ent/usermfafactor"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newUserMfaFactorRepoSqlite 用 enttest helper 构造 UserMfaFactorRepo，
// 逐字段复刻 NewUserMfaFactorRepo（log 换 NopLogger；该 repo 无 init()）。
func newUserMfaFactorRepoSqlite(t *testing.T) *UserMfaFactorRepo {
	t.Helper()
	return &UserMfaFactorRepo{
		entClient: enttest.NewEntClientForTest(t),
		log:       bLogger.NewHelper(bLogger.NopLogger()),
	}
}

// TestUserMfaFactorRepoSqlite_CreateAndFindEnabledTotp 验证 CreateTotpFactor
// 写入（tenant/user/method/status/secret/displayName 落库）与
// HasEnabledTotp / FindEnabledTotpForUser 的归属校验与 secret 还原。
func TestUserMfaFactorRepoSqlite_CreateAndFindEnabledTotp(t *testing.T) {
	repo := newUserMfaFactorRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	factorID, err := repo.CreateTotpFactor(ctx, 1, 10, "SECRET123", "认证器A")
	require.NoError(t, err, "CreateTotpFactor 应成功")
	require.NotZero(t, factorID, "新因子 ID 应非零")

	rows, err := repo.entClient.Client().UserMfaFactor.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "user_mfa_factors 应有 1 行")
	row := rows[0]
	require.Equal(t, factorID, row.ID, "直查 ID 应与返回 ID 一致")
	require.NotNil(t, row.TenantID)
	require.Equal(t, uint32(1), *row.TenantID, "tenant_id 应按参数落库")
	require.NotNil(t, row.UserID)
	require.Equal(t, uint32(10), *row.UserID, "user_id 应按参数落库")
	require.NotNil(t, row.Method)
	require.Equal(t, entUserMfaFactor.MethodTotp, *row.Method, "method 应为 TOTP")
	require.NotNil(t, row.Status)
	require.Equal(t, entUserMfaFactor.StatusEnabled, *row.Status, "status 应为 ENABLED")
	require.NotNil(t, row.DisplayName)
	require.Equal(t, "认证器A", *row.DisplayName, "display_name 应按参数落库")
	// 测试环境下全局加密器未初始化，EncryptIfNeeded 为直通；secret 应可经
	// DecryptIfNeeded（同样直通）还原为明文。
	require.NotNil(t, row.SecretHash, "secret_hash 应落库")
	require.Equal(t, "SECRET123", *row.SecretHash, "测试环境 secret 应直通落库（加密器未初始化）")

	// HasEnabledTotp：命中与未命中（不同用户/不同租户）
	ok, err := repo.HasEnabledTotp(ctx, 1, 10)
	require.NoError(t, err)
	require.True(t, ok, "已绑定 ENABLED TOTP 的 (tenant,user) 应返回 true")
	ok, err = repo.HasEnabledTotp(ctx, 1, 999)
	require.NoError(t, err)
	require.False(t, ok, "未绑定 TOTP 的用户应返回 false")
	ok, err = repo.HasEnabledTotp(ctx, 2, 10)
	require.NoError(t, err)
	require.False(t, ok, "不同租户下同一用户应返回 false")

	// FindEnabledTotpForUser：命中返回因子 ID 与明文 secret
	gotID, plain, err := repo.FindEnabledTotpForUser(ctx, 1, 10)
	require.NoError(t, err, "已绑定的 ENABLED TOTP 应可取出")
	require.Equal(t, factorID, gotID, "返回的因子 ID 应一致")
	require.Equal(t, "SECRET123", plain, "secret 应被还原为明文")

	// FindEnabledTotpForUser：未命中返回错误
	_, _, err = repo.FindEnabledTotpForUser(ctx, 2, 10)
	require.Error(t, err, "不同租户应查不到该用户因子")
	_, _, err = repo.FindEnabledTotpForUser(ctx, 1, 999)
	require.Error(t, err, "未绑定用户应查不到因子")
}

// TestUserMfaFactorRepoSqlite_ListByUserMapping 验证 ListByUser 只返回
// 指定 (tenant,user) 的因子，且 Method / Enabled 字段按
// toMFAMethod / toEnrolledEnabled 的映射逐对成立。
func TestUserMfaFactorRepoSqlite_ListByUserMapping(t *testing.T) {
	repo := newUserMfaFactorRepoSqlite(t)
	client := repo.entClient.Client()
	ctx := enttest.NewSystemViewerCtx(context.Background())

	// 为 (1,10) 每种 method 各建一行。(tenant_id,user_id,method) 有唯一约束，
	// TOTP 行用 DISABLED（同时覆盖 Enabled=false 映射分支），其余三种 method 用 ENABLED
	// （覆盖 Enabled=true 分支）；四种 method 映射均逐对覆盖。
	methodRows := map[string]bool{}
	for _, method := range []entUserMfaFactor.Method{
		entUserMfaFactor.MethodTotp,
		entUserMfaFactor.MethodSms,
		entUserMfaFactor.MethodEmail,
		entUserMfaFactor.MethodWebauthn,
	} {
		status := entUserMfaFactor.StatusEnabled
		if method == entUserMfaFactor.MethodTotp {
			status = entUserMfaFactor.StatusDisabled
		}
		require.NoError(t, client.UserMfaFactor.Create().
			SetTenantID(1).
			SetUserID(10).
			SetMethod(method).
			SetStatus(status).
			SetSecretHash("secret-"+string(method)).
			SetDisplayName("display-"+string(method)).
			Exec(ctx), "建 %s 行应成功", method)
		methodRows[string(method)] = true
	}
	// 其他 (tenant,user) 的行不应混入
	require.NoError(t, client.UserMfaFactor.Create().
		SetTenantID(2).
		SetUserID(10).
		SetMethod(entUserMfaFactor.MethodSms).
		SetStatus(entUserMfaFactor.StatusEnabled).
		SetSecretHash("secret-other-tenant").
		SetDisplayName("display-other-tenant").
		Exec(ctx), "建他租户行应成功")

	infos, err := repo.ListByUser(ctx, 1, 10)
	require.NoError(t, err)
	require.Len(t, infos, 4, "应只返回 (1,10) 的 4 行（含 1 行 DISABLED TOTP）")

	// method 映射：ent 四种 method → proto 四种枚举，各出现一次
	methodCount := map[authenticationV1.MFAMethod]int{}
	displayByMethod := map[authenticationV1.MFAMethod]string{}
	enabledByMethod := map[authenticationV1.MFAMethod]bool{}
	for _, info := range infos {
		methodCount[info.Method]++
		displayByMethod[info.Method] = info.DisplayName
		enabledByMethod[info.Method] = info.Enabled
	}
	require.Equal(t, map[authenticationV1.MFAMethod]int{
		authenticationV1.MFAMethod_TOTP:     1,
		authenticationV1.MFAMethod_SMS:      1,
		authenticationV1.MFAMethod_EMAIL:    1,
		authenticationV1.MFAMethod_WEBAUTHN: 1,
	}, methodCount, "四种 method 应各经映射出现一次")
	require.Equal(t, "display-TOTP", displayByMethod[authenticationV1.MFAMethod_TOTP], "DISABLED TOTP 行 display_name 应回传")
	require.Equal(t, "display-SMS", displayByMethod[authenticationV1.MFAMethod_SMS], "SMS 行 display_name 应回传")
	require.Equal(t, "display-EMAIL", displayByMethod[authenticationV1.MFAMethod_EMAIL], "EMAIL 行 display_name 应回传")
	require.Equal(t, "display-WEBAUTHN", displayByMethod[authenticationV1.MFAMethod_WEBAUTHN], "WEBAUTHN 行 display_name 应回传")
	require.False(t, enabledByMethod[authenticationV1.MFAMethod_TOTP],
		"DISABLED 行的 Enabled 应映射为 false")
	require.True(t, enabledByMethod[authenticationV1.MFAMethod_SMS],
		"ENABLED 行的 Enabled 应映射为 true")
	require.True(t, enabledByMethod[authenticationV1.MFAMethod_EMAIL],
		"ENABLED 行的 Enabled 应映射为 true")
	require.True(t, enabledByMethod[authenticationV1.MFAMethod_WEBAUTHN],
		"ENABLED 行的 Enabled 应映射为 true")

	// 其他归属：查不到行
	otherInfos, err := repo.ListByUser(ctx, 9, 10)
	require.NoError(t, err)
	require.Empty(t, otherInfos, "无归属行时应返回空列表")
}

// TestUserMfaFactorRepoSqlite_DeleteForUserAndByMethod 验证
// DeleteForUser 的归属强制校验与 DeleteAllByUserMethod 的租户+方法范围删除。
func TestUserMfaFactorRepoSqlite_DeleteForUserAndByMethod(t *testing.T) {
	repo := newUserMfaFactorRepoSqlite(t)
	client := repo.entClient.Client()
	ctx := enttest.NewSystemViewerCtx(context.Background())

	totpOwn, err := client.UserMfaFactor.Create().
		SetTenantID(1).SetUserID(10).SetMethod(entUserMfaFactor.MethodTotp).
		SetStatus(entUserMfaFactor.StatusEnabled).SetSecretHash("s1").Save(ctx)
	require.NoError(t, err)
	smsA, err := client.UserMfaFactor.Create().
		SetTenantID(1).SetUserID(10).SetMethod(entUserMfaFactor.MethodSms).
		SetStatus(entUserMfaFactor.StatusEnabled).SetSecretHash("s2").Save(ctx)
	require.NoError(t, err)
	// (tenant_id,user_id,method) 唯一：同租户他用户的 SMS 行 + 他租户的 SMS 行
	require.NoError(t, client.UserMfaFactor.Create().
		SetTenantID(1).SetUserID(11).SetMethod(entUserMfaFactor.MethodSms).
		SetStatus(entUserMfaFactor.StatusEnabled).SetSecretHash("s3").Exec(ctx))
	require.NoError(t, client.UserMfaFactor.Create().
		SetTenantID(2).SetUserID(20).SetMethod(entUserMfaFactor.MethodSms).
		SetStatus(entUserMfaFactor.StatusEnabled).SetSecretHash("s4").Exec(ctx))
	rows, err := client.UserMfaFactor.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 4)

	// DeleteForUser：正确归属删除成功
	deleted, err := repo.DeleteForUser(ctx, 1, 10, totpOwn.ID)
	require.NoError(t, err)
	require.True(t, deleted, "正确归属的删除应返回 true")
	remain, _ := client.UserMfaFactor.Query().All(ctx)
	require.Len(t, remain, 3, "删 1 行后应剩 3 行")

	// DeleteForUser：跨归属（错误租户）删除不命中
	deleted, err = repo.DeleteForUser(ctx, 2, 10, smsA.ID)
	require.NoError(t, err)
	require.False(t, deleted, "错误租户的删除应返回 false（防越权删他人因子）")
	remain, _ = client.UserMfaFactor.Query().All(ctx)
	require.Len(t, remain, 3, "越权删除不应移除任何行")

	// DeleteAllByUserMethod：只删 (1,10) 的 SMS 一行；他用户/他租户的 SMS 行保留
	n, err := repo.DeleteAllByUserMethod(ctx, 1, 10, entUserMfaFactor.MethodSms)
	require.NoError(t, err)
	require.Equal(t, 1, n, "应只删除 (1,10) 的 SMS 行")
	remain, _ = client.UserMfaFactor.Query().All(ctx)
	require.Len(t, remain, 2, "他用户与他租户的 SMS 行应保留")

	// DeleteAllByUserMethod：无命中返回 0
	n, err = repo.DeleteAllByUserMethod(ctx, 1, 10, entUserMfaFactor.MethodEmail)
	require.NoError(t, err)
	require.Zero(t, n, "无命中应返回 0")
}

// TestUserMfaFactorRepoSqlite_GetFindFirstAndUpdateLastUsed 验证
// GetFactorById / FindFirstByUser 的命中与未命中，以及 UpdateLastUsed 的落库。
func TestUserMfaFactorRepoSqlite_GetFindFirstAndUpdateLastUsed(t *testing.T) {
	repo := newUserMfaFactorRepoSqlite(t)
	client := repo.entClient.Client()
	ctx := enttest.NewSystemViewerCtx(context.Background())

	rowA, err := client.UserMfaFactor.Create().
		SetTenantID(3).SetUserID(30).SetMethod(entUserMfaFactor.MethodEmail).
		SetStatus(entUserMfaFactor.StatusEnabled).SetSecretHash("s5").Save(ctx)
	require.NoError(t, err)
	require.NotZero(t, rowA.ID)
	require.NoError(t, client.UserMfaFactor.Create().
		SetTenantID(3).SetUserID(31).SetMethod(entUserMfaFactor.MethodTotp).
		SetStatus(entUserMfaFactor.StatusEnabled).SetSecretHash("s6").Exec(ctx))

	// GetFactorById：命中
	tid, uid, found, err := repo.GetFactorById(ctx, rowA.ID)
	require.NoError(t, err)
	require.True(t, found, "存在的因子应命中")
	require.Equal(t, uint32(3), tid, "应返回归属租户")
	require.Equal(t, uint32(30), uid, "应返回归属用户")

	// GetFactorById：未命中（found=false，不报错）
	tid, uid, found, err = repo.GetFactorById(ctx, 424242)
	require.NoError(t, err)
	require.False(t, found, "不存在的因子应返回 found=false")
	require.Zero(t, tid)
	require.Zero(t, uid)

	// FindFirstByUser：按用户+方法命中首行
	tid, uid, found, err = repo.FindFirstByUser(ctx, 31, entUserMfaFactor.MethodTotp)
	require.NoError(t, err)
	require.True(t, found, "存在 (user,method) 行应命中")
	require.Equal(t, uint32(3), tid)
	require.Equal(t, uint32(31), uid)

	// FindFirstByUser：无行命中
	tid, uid, found, err = repo.FindFirstByUser(ctx, 999, entUserMfaFactor.MethodTotp)
	require.NoError(t, err)
	require.False(t, found, "无行命中应返回 found=false")
	require.Zero(t, tid)
	require.Zero(t, uid)

	// UpdateLastUsed：更新最近使用时间
	at := time.Now().Add(-time.Hour)
	require.NoError(t, repo.UpdateLastUsed(ctx, 3, 30, rowA.ID, at))
	after, err := client.UserMfaFactor.Query().Where(entUserMfaFactor.IDEQ(rowA.ID)).Only(ctx)
	require.NoError(t, err)
	require.NotNil(t, after.LastUsedAt, "last_used_at 应被更新")
	require.Equal(t, at.Unix(), after.LastUsedAt.Unix(), "last_used_at 应等于传入时间（秒级比较）")

	// UpdateLastUsed：归属不符不命中、不报错
	require.NoError(t, repo.UpdateLastUsed(ctx, 9, 30, rowA.ID, at))
	after2, err := client.UserMfaFactor.Query().Where(entUserMfaFactor.IDEQ(rowA.ID)).Only(ctx)
	require.NoError(t, err)
	require.NotNil(t, after2.LastUsedAt)
	require.Equal(t, at.Unix(), after2.LastUsedAt.Unix(), "归属不符的更新不应改变 last_used_at")
}
