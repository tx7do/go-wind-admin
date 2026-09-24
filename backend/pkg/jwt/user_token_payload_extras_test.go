package jwt

import (
	"testing"
	"time"

	jwtV5 "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"

	authn "github.com/tx7do/kratos-authn/engine"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
)

// TestNewUserTokenAuthClaims_OptionalFieldsSet 声明构建侧的可选字段
// 分支：jti、字段权限隐藏集、平台/租户管理员标记与显式过期时间必须
// 如实落位。这些字段分别驱动令牌轮换定位、读写路径的字段裁剪/剥离
// 与管理员旁路判定，漏写即静默丢权限语义。
func TestNewUserTokenAuthClaims_OptionalFieldsSet(t *testing.T) {
	payload := NewUserTokenPayload("carol", 21, 31, nil, nil, nil, nil, nil)
	payload.Jti = trans.Ptr("jti-123")
	payload.HiddenFields = []string{"user.password", "user.phone"}
	payload.IsPlatformAdmin = trans.Ptr(true)
	payload.IsTenantAdmin = trans.Ptr(true)
	exp := time.Now().Add(2 * time.Hour)

	claims := NewUserTokenAuthClaims(payload, &exp)
	require.NotNil(t, claims)

	assert.Equal(t, "jti-123", (*claims)[authn.ClaimFieldJwtID], "jti 应原样写入")
	assert.Equal(t, []string{"user.password", "user.phone"},
		(*claims)[ClaimFieldHiddenFields], "隐藏字段集应原样写入")
	assert.Equal(t, true, (*claims)[ClaimFieldIsPlatformAdmin])
	assert.Equal(t, true, (*claims)[ClaimFieldIsTenantAdmin])
	assert.EqualValues(t, exp.Unix(), (*claims)[authn.ClaimFieldExpirationTime],
		"显式过期时间应落位到 exp 声明")
}

// TestNewUserTokenAuthClaims_OmittedOptionals 反向分支：可选字段缺省
// （nil / 空切片）与过期时间为 nil 时，对应声明不得出现。
// 显式 false 的管理员标记走 != nil 分支、落位为 false 而非缺省。
func TestNewUserTokenAuthClaims_OmittedOptionals(t *testing.T) {
	payload := NewUserTokenPayload("dave", 22, 32, nil, nil, nil, nil, nil)
	payload.IsPlatformAdmin = trans.Ptr(false)

	claims := NewUserTokenAuthClaims(payload, nil)
	require.NotNil(t, claims)

	assert.Equal(t, false, (*claims)[ClaimFieldIsPlatformAdmin],
		"显式 false 应落位为 false（区别于缺省）")
	for _, absent := range []string{
		authn.ClaimFieldJwtID,
		ClaimFieldHiddenFields,
		ClaimFieldIsTenantAdmin,
		authn.ClaimFieldExpirationTime,
	} {
		assert.NotContains(t, *claims, absent, "缺省的可选字段 %q 不应出现", absent)
	}
}

// TestNewUserTokenPayloadWithClaims_OptionalFieldsRoundTrip 解析侧对称：
// jti、隐藏字段集、平台/租户管理员标记从声明还原到 payload；
// 显式 false 的管理员标记不置位（解析侧按 ok && ipa 处理）。
func TestNewUserTokenPayloadWithClaims_OptionalFieldsRoundTrip(t *testing.T) {
	t.Run("all set", func(t *testing.T) {
		claims := &authn.AuthClaims{
			authn.ClaimFieldJwtID:     "jti-456",
			ClaimFieldHiddenFields:    []string{"user.email"},
			ClaimFieldIsPlatformAdmin: true,
			ClaimFieldIsTenantAdmin:   true,
		}
		payload, err := NewUserTokenPayloadWithClaims(claims)
		require.NoError(t, err)
		require.NotNil(t, payload)
		assert.EqualValues(t, "jti-456", payload.GetJti())
		assert.Equal(t, []string{"user.email"}, payload.GetHiddenFields())
		assert.True(t, payload.GetIsPlatformAdmin())
		assert.True(t, payload.GetIsTenantAdmin())
	})
	t.Run("explicit false flags not set", func(t *testing.T) {
		claims := &authn.AuthClaims{
			ClaimFieldIsPlatformAdmin: false,
			ClaimFieldIsTenantAdmin:   false,
		}
		payload, err := NewUserTokenPayloadWithClaims(claims)
		require.NoError(t, err)
		assert.False(t, payload.GetIsPlatformAdmin(), "false 标记不应置位字段")
		assert.False(t, payload.GetIsTenantAdmin(), "false 标记不应置位字段")
	})
}

// TestNewUserTokenPayloadWithClaims_DataScopeUnitsParsing dsu 声明是
// 逗号连接的十进制串：非法片段（非数字、负号、超 uint64）必须被跳过，
// 合法片段保序还原。UNIT 类数据范围的越界片段混入不应导致整条解析失败。
func TestNewUserTokenPayloadWithClaims_DataScopeUnitsParsing(t *testing.T) {
	claims := &authn.AuthClaims{
		ClaimFieldDataScopeUnits: "12,abc,34,-5,99999999999999999999999,7",
	}
	payload, err := NewUserTokenPayloadWithClaims(claims)
	require.NoError(t, err)
	assert.Equal(t, []uint64{12, 34, 7}, payload.GetDataScopeUnitIds(),
		"仅合法十进制片段应被还原，非法片段跳过")
}

// TestNewUserTokenPayloadWithClaims_DataScopesFiltering dss 声明解析的
// 过滤规则：UNSPECIFIED 与未知名必须剔除；fail-open 会把"未定范围"
// 当成合法范围放行查询。
func TestNewUserTokenPayloadWithClaims_DataScopesFiltering(t *testing.T) {
	claims := &authn.AuthClaims{
		ClaimFieldDataScopes: []string{
			"SELF",
			"DATA_SCOPE_UNSPECIFIED",
			"NOT_A_SCOPE_NAME",
			"ALL",
		},
	}
	payload, err := NewUserTokenPayloadWithClaims(claims)
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]identityV1.DataScope{identityV1.DataScope_SELF, identityV1.DataScope_ALL},
		payload.GetDataScopes(),
		"仅已定义且非 UNSPECIFIED 的范围名应被保留")
}

// TestNewUserTokenPayloadWithClaims_MalformedClaimTypes 全字段类型错误
// 的声明：每个 getter 都走错误分支（记日志、字段留空），整体不得报错
// 也不得 panic。解析侧面对的是已验签但仍可能异构的 claims，类型不符
// 必须逐字段跳过而非整体失败。
func TestNewUserTokenPayloadWithClaims_MalformedClaimTypes(t *testing.T) {
	claims := &authn.AuthClaims{
		authn.ClaimFieldSubject:  42,
		authn.ClaimFieldJwtID:    42,
		ClaimFieldUserID:         "not-uid",
		ClaimFieldTenantID:       "not-tid",
		ClaimFieldClientID:       42,
		ClaimFieldDeviceID:       42,
		ClaimFieldRoleCodes:      []interface{}{42},
		ClaimFieldDataScope:      42,
		ClaimFieldDataScopes:     []interface{}{42},
		ClaimFieldDataScopeUnits: 42,
		ClaimFieldHiddenFields:   []interface{}{42},
		ClaimFieldOrgUnitID:      "not-ouid",
	}
	payload, err := NewUserTokenPayloadWithClaims(claims)
	require.NoError(t, err, "类型不符应逐字段跳过而非整体失败")
	require.NotNil(t, payload)

	assert.Empty(t, payload.GetUsername())
	assert.Empty(t, payload.GetJti())
	assert.EqualValues(t, 0, payload.GetUserId())
	assert.EqualValues(t, 0, payload.GetTenantId())
	assert.Empty(t, payload.GetClientId())
	assert.Empty(t, payload.GetDeviceId())
	assert.Empty(t, payload.GetRoles())
	assert.EqualValues(t, 0, payload.GetDataScope())
	assert.Empty(t, payload.GetDataScopes())
	assert.Empty(t, payload.GetDataScopeUnitIds())
	assert.Empty(t, payload.GetHiddenFields())
	assert.EqualValues(t, 0, payload.GetOrgUnitId())
	assert.False(t, payload.GetIsPlatformAdmin())
	assert.False(t, payload.GetIsTenantAdmin())
}

// TestNewUserTokenPayloadWithJwtMapClaims_ListTypeVariants MapClaims 路径
// （审计中间件以 ParseUnverified 解析客户端可伪造的令牌）的列表类型
// 分支：[]string 形式的角色码原样接收；[]interface{} 中非字符串元素剔除；
// 隐藏字段集同理；bool 形式的管理员标记按真值置位。
func TestNewUserTokenPayloadWithJwtMapClaims_ListTypeVariants(t *testing.T) {
	t.Run("string slice role codes", func(t *testing.T) {
		mapClaims := jwtV5.MapClaims{
			ClaimFieldRoleCodes: []string{"r1", "r2"},
		}
		payload, err := NewUserTokenPayloadWithJwtMapClaims(mapClaims)
		require.NoError(t, err)
		assert.Equal(t, []string{"r1", "r2"}, payload.GetRoles())
	})
	t.Run("mixed interface list filters non-strings", func(t *testing.T) {
		mapClaims := jwtV5.MapClaims{
			ClaimFieldRoleCodes:    []interface{}{"r1", 42, nil},
			ClaimFieldHiddenFields: []interface{}{"user.age", 42},
		}
		payload, err := NewUserTokenPayloadWithJwtMapClaims(mapClaims)
		require.NoError(t, err)
		assert.Equal(t, []string{"r1"}, payload.GetRoles(),
			"非字符串角色码元素必须被剔除")
		assert.Equal(t, []string{"user.age"}, payload.GetHiddenFields(),
			"非字符串隐藏字段元素必须被剔除")
	})
	t.Run("admin flags", func(t *testing.T) {
		mapClaims := jwtV5.MapClaims{
			ClaimFieldIsPlatformAdmin: true,
			ClaimFieldIsTenantAdmin:   true,
		}
		payload, err := NewUserTokenPayloadWithJwtMapClaims(mapClaims)
		require.NoError(t, err)
		assert.True(t, payload.GetIsPlatformAdmin())
		assert.True(t, payload.GetIsTenantAdmin())
	})
	t.Run("numeric fields with wrong types stay empty", func(t *testing.T) {
		mapClaims := jwtV5.MapClaims{
			authn.ClaimFieldSubject: 42,
			ClaimFieldUserID:        "not-a-number",
			ClaimFieldTenantID:      "not-a-number",
			ClaimFieldDataScope:     "NOT_A_SCOPE_NAME",
		}
		payload, err := NewUserTokenPayloadWithJwtMapClaims(mapClaims)
		require.NoError(t, err)
		assert.Empty(t, payload.GetUsername(), "非字符串形式的 sub 不得置位用户名")
		assert.EqualValues(t, 0, payload.GetUserId(), "字符串形式的 uid 不得置位")
		assert.EqualValues(t, 0, payload.GetTenantId(), "字符串形式的 tid 不得置位")
		assert.EqualValues(t, 0, payload.GetDataScope(), "未知范围名不得置位")
	})
}

// TestNewUserTokenPayloadWithJwtMapClaims_InvalidRoleCodesType 角色码
// 声明为非列表类型（标量）时必须整体拒绝而非静默忽略：角色码是 RBAC
// 判定输入，静默丢角色会把"该拒绝的请求"变成"无角色请求"。
func TestNewUserTokenPayloadWithJwtMapClaims_InvalidRoleCodesType(t *testing.T) {
	for _, bad := range []interface{}{42, "admin"} {
		t.Run("scalar role codes", func(t *testing.T) {
			mapClaims := jwtV5.MapClaims{
				ClaimFieldRoleCodes: bad,
			}
			payload, err := NewUserTokenPayloadWithJwtMapClaims(mapClaims)
			require.Error(t, err, "标量形式的角色码必须报错")
			assert.Nil(t, payload)
		})
	}
}

// TestIsTokenExpired 过期判定的边界：nil 声明视为过期；无 exp 声明视为
// 未过期；leeway（60s）内的刚过期令牌仍可用；超过 leeway 的过期令牌
// 判定过期。
// 注：exp 以 int64 写入时（当前 NewUserTokenAuthClaims 的写入形态），
// 引擎侧 parseNumericDate 仅接受 float64/json.Number，会按"无 exp 声明"
// 处理返回 false——此处钉住该现状；若写入侧改为 float64，本用例期望值
// 应翻转。
func TestIsTokenExpired(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name   string
		claims *authn.AuthClaims
		want   bool
	}{
		{"nil claims", nil, true},
		{"no exp claim", &authn.AuthClaims{}, false},
		{"expired beyond leeway", &authn.AuthClaims{
			authn.ClaimFieldExpirationTime: float64(now.Add(-10 * time.Minute).Unix()),
		}, true},
		{"expired within leeway", &authn.AuthClaims{
			authn.ClaimFieldExpirationTime: float64(now.Add(-30 * time.Second).Unix()),
		}, false},
		{"future expiration", &authn.AuthClaims{
			authn.ClaimFieldExpirationTime: float64(now.Add(10 * time.Minute).Unix()),
		}, false},
		{"int64 exp treated as absent (current quirk)", &authn.AuthClaims{
			authn.ClaimFieldExpirationTime: int64(now.Add(-10 * time.Minute).Unix()),
		}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsTokenExpired(tc.claims))
		})
	}
}

// TestIsTokenNotValidYet 未生效判定的边界：nil 声明视为未生效；无 nbf
// 声明视为已生效；leeway 内的未来 nbf 视为已生效；超过 leeway 的未来
// nbf 判定未生效。
func TestIsTokenNotValidYet(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name   string
		claims *authn.AuthClaims
		want   bool
	}{
		{"nil claims", nil, true},
		{"no nbf claim", &authn.AuthClaims{}, false},
		{"nbf beyond leeway in future", &authn.AuthClaims{
			authn.ClaimFieldNotBefore: float64(now.Add(10 * time.Minute).Unix()),
		}, true},
		{"nbf within leeway", &authn.AuthClaims{
			authn.ClaimFieldNotBefore: float64(now.Add(30 * time.Second).Unix()),
		}, false},
		{"nbf in past", &authn.AuthClaims{
			authn.ClaimFieldNotBefore: float64(now.Add(-10 * time.Minute).Unix()),
		}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsTokenNotValidYet(tc.claims))
		})
	}
}
