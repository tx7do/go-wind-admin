package jwt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authn "github.com/tx7do/kratos-authn/engine"
)

// TestNewRefreshTokenAuthClaims_Validation 创建刷新令牌 claims 前必须
// 校验定位要素：userId 与 jti 缺一不可（二者是定位 Redis 刷新令牌记录、
// 完成轮换与吊销的最小信息）。缺失时必须报错，而不是产出一条无法定位
// 会话记录的 claims。
func TestNewRefreshTokenAuthClaims_Validation(t *testing.T) {
	cases := []struct {
		name   string
		userId int64
		jti    string
		errMsg string
	}{
		{"missing userId", 0, "jti-1", "refresh token claims: userId is empty"},
		{"missing jti", 7, "", "refresh token claims: jti is empty"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			claims, err := NewRefreshTokenAuthClaims(tc.userId, tc.jti, nil)
			require.Error(t, err, "缺少定位要素时必须报错")
			assert.Nil(t, claims)
			assert.EqualError(t, err, tc.errMsg)
		})
	}
}

// TestNewRefreshTokenAuthClaims_MinimalShape 合法入参产出的 claims 只携带
// uid/jti/iat（可选 exp），不得夹带任何业务权限字段——刷新令牌能脱离
// access token 独立鉴权，若混入 roc/ds 等字段会形成权限旁路面。
func TestNewRefreshTokenAuthClaims_MinimalShape(t *testing.T) {
	claims, err := NewRefreshTokenAuthClaims(7, "jti-abc", nil)
	require.NoError(t, err)
	require.NotNil(t, claims)

	assert.EqualValues(t, 7, (*claims)[ClaimFieldUserID], "uid 应为 7")
	assert.Equal(t, "jti-abc", (*claims)[authn.ClaimFieldJwtID], "jti 应原样落位")
	assert.Contains(t, *claims, authn.ClaimFieldIssuedAt, "iat 必须存在")
	assert.NotContains(t, *claims, authn.ClaimFieldExpirationTime,
		"未传过期时间时不得自带 exp")
	// 刷新令牌不得携带业务字段
	for _, forbidden := range []string{
		authn.ClaimFieldSubject,
		ClaimFieldTenantID,
		ClaimFieldRoleCodes,
		ClaimFieldDataScope,
		ClaimFieldOrgUnitID,
		ClaimFieldClientID,
		ClaimFieldDeviceID,
		ClaimFieldDataScopes,
		ClaimFieldDataScopeUnits,
		ClaimFieldHiddenFields,
		ClaimFieldIsPlatformAdmin,
		ClaimFieldIsTenantAdmin,
	} {
		assert.NotContains(t, *claims, forbidden,
			"刷新令牌 claims 不应携带业务字段 %q", forbidden)
	}
}

// TestNewRefreshTokenAuthClaims_WithExpiration 显式传入过期时间时，
// exp 声明必须按该时间落位（刷新令牌的绝对有效期由创建时的 exp 决定）。
func TestNewRefreshTokenAuthClaims_WithExpiration(t *testing.T) {
	exp := time.Now().Add(7 * 24 * time.Hour)
	claims, err := NewRefreshTokenAuthClaims(9, "jti-def", &exp)
	require.NoError(t, err)
	require.NotNil(t, claims)
	assert.EqualValues(t, exp.Unix(), (*claims)[authn.ClaimFieldExpirationTime])
}

// TestParseRefreshTokenClaims_RoundTrip 生成 → 解析往返一致性：
// uid 与 jti 必须无损还原（带 exp 与不带 exp 两条路径）。
func TestParseRefreshTokenClaims_RoundTrip(t *testing.T) {
	t.Run("without expiration", func(t *testing.T) {
		claims, err := NewRefreshTokenAuthClaims(11, "jti-roundtrip", nil)
		require.NoError(t, err)
		userId, jti, err := ParseRefreshTokenClaims(claims)
		require.NoError(t, err)
		assert.EqualValues(t, 11, userId)
		assert.Equal(t, "jti-roundtrip", jti)
	})
	t.Run("with expiration", func(t *testing.T) {
		exp := time.Now().Add(time.Hour)
		claims, err := NewRefreshTokenAuthClaims(12, "jti-roundtrip-2", &exp)
		require.NoError(t, err)
		userId, jti, err := ParseRefreshTokenClaims(claims)
		require.NoError(t, err)
		assert.EqualValues(t, 12, userId)
		assert.Equal(t, "jti-roundtrip-2", jti)
	})
}

// TestParseRefreshTokenClaims_RejectsIncomplete 解析侧对称校验：
// nil claims、缺 uid、uid 为 0、uid/jti 类型错误、缺 jti、空 jti
// 一律拒绝。刷新令牌的定位要素不完整时放行，会导致后续 Redis 定位
// 静默失败或错位到其他用户的记录。
func TestParseRefreshTokenClaims_RejectsIncomplete(t *testing.T) {
	cases := []struct {
		name   string
		claims *authn.AuthClaims
	}{
		{"nil claims", nil},
		{"empty claims", &authn.AuthClaims{}},
		{"uid wrong type", &authn.AuthClaims{ClaimFieldUserID: "not-a-number"}},
		{"jti missing", &authn.AuthClaims{ClaimFieldUserID: int64(3)}},
		{"jti wrong type", &authn.AuthClaims{ClaimFieldUserID: int64(3), authn.ClaimFieldJwtID: 42}},
		{"jti empty", &authn.AuthClaims{ClaimFieldUserID: int64(3), authn.ClaimFieldJwtID: ""}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			userId, jti, err := ParseRefreshTokenClaims(tc.claims)
			require.Error(t, err, "定位要素不完整时必须报错")
			assert.EqualValues(t, 0, userId)
			assert.Empty(t, jti)
		})
	}
}
