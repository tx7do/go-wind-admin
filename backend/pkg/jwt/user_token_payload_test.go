package jwt

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/tx7do/go-utils/trans"
	authn "github.com/tx7do/kratos-authn/engine"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
)

func TestNewUserTokenPayload(t *testing.T) {
	ou := int64(42)
	uid := int64(1)
	tid := int64(2)
	username := "test_user"
	roleCodes := []string{"admin"}
	client := "client_123"
	device := "device_1"
	ds := identityV1.DataScope(1)

	payload := NewUserTokenPayload(
		username, uid, tid, &ou, roleCodes,
		&ds, &client, &device,
	)
	assert.NotNil(t, payload)
	assert.Equal(t, uid, payload.GetUserId())
	assert.Equal(t, tid, payload.GetTenantId())
	assert.Equal(t, username, payload.GetUsername())
	assert.Equal(t, client, payload.GetClientId())
	assert.Equal(t, device, payload.GetDeviceId())
	assert.Equal(t, []string{"admin"}, payload.GetRoles())
	if payload.DataScope != nil {
		assert.Equal(t, ds, payload.GetDataScope())
	}
	if payload.OrgUnitId != nil {
		assert.Equal(t, ou, payload.GetOrgUnitId())
	}
}

func TestNewUserTokenAuthClaims(t *testing.T) {
	ou := int64(7)
	uid := int64(3)
	tid := int64(4)
	username := "alice"
	roleCodes := []string{"editor"}
	client := "cli"
	device := "dev"
	ds := identityV1.DataScope(2)

	payload := NewUserTokenPayload(
		username, uid, tid, &ou, roleCodes,
		&ds, &client, &device,
	)

	claims := NewUserTokenAuthClaims(payload, nil)
	assert.NotNil(t, claims)

	// subject
	assert.Equal(t, username, (*claims)[authn.ClaimFieldSubject])
	// numeric fields
	assert.Equal(t, uid, (*claims)[ClaimFieldUserID])
	assert.Equal(t, tid, (*claims)[ClaimFieldTenantID])
	// client/device/roles
	assert.Equal(t, client, (*claims)[ClaimFieldClientID])
	assert.Equal(t, device, (*claims)[ClaimFieldDeviceID])
	assert.Equal(t, roleCodes, (*claims)[ClaimFieldRoleCodes])
	// data scope stored as string
	assert.Equal(t, ds.String(), (*claims)[ClaimFieldDataScope])
	// org unit
	assert.Equal(t, ou, (*claims)[ClaimFieldOrgUnitID])
}

func TestNewUserTokenPayloadWithClaims(t *testing.T) {
	user := identityV1.User{
		Id:       trans.Ptr(int64(5)),
		TenantId: trans.Ptr(int64(6)),
		Username: trans.Ptr("bob"),
		Roles:    []string{"viewer"},
	}

	client := "cid"
	device := "did"
	ds := identityV1.DataScope(3)
	ou := int64(9)

	claims := &authn.AuthClaims{
		authn.ClaimFieldSubject: user.GetUsername(),
		ClaimFieldUserID:        user.GetId(),
		ClaimFieldTenantID:      user.GetTenantId(),
		ClaimFieldClientID:      client,
		ClaimFieldDeviceID:      device,
		ClaimFieldRoleCodes:     user.Roles,
		ClaimFieldDataScope:     ds.String(),
		ClaimFieldOrgUnitID:     ou,
	}

	payload, err := NewUserTokenPayloadWithClaims(claims)
	assert.NoError(t, err)
	assert.NotNil(t, payload)
	assert.Equal(t, user.GetUsername(), payload.GetUsername())
	assert.Equal(t, user.GetId(), payload.GetUserId())
	assert.Equal(t, user.GetTenantId(), payload.GetTenantId())
	assert.Equal(t, client, payload.GetClientId())
	assert.Equal(t, device, payload.GetDeviceId())
	assert.Equal(t, user.Roles, payload.GetRoles())
	if payload.DataScope != nil {
		assert.Equal(t, ds, payload.GetDataScope())
	}
	if payload.OrgUnitId != nil {
		assert.Equal(t, ou, payload.GetOrgUnitId())
	}
}

func TestNewUserTokenPayloadWithJwtMapClaims(t *testing.T) {
	user := identityV1.User{
		Id:       trans.Ptr(int64(10)),
		TenantId: trans.Ptr(int64(11)),
		Username: trans.Ptr("eve"),
		Roles:    []string{"r1", "r2"},
	}

	client := "c1"
	device := "d1"
	ds := identityV1.DataScope(4)
	ou := int64(21)

	// jwt.MapClaims uses float64 for numeric JSON numbers
	mapClaims := jwt.MapClaims{
		"sub":               user.GetUsername(),
		ClaimFieldUserID:    float64(user.GetId()),
		ClaimFieldTenantID:  float64(user.GetTenantId()),
		ClaimFieldClientID:  client,
		ClaimFieldDeviceID:  device,
		ClaimFieldDataScope: ds.String(),
		ClaimFieldOrgUnitID: float64(ou),
		ClaimFieldRoleCodes: []interface{}{"r1", "r2"},
	}

	payload, err := NewUserTokenPayloadWithJwtMapClaims(mapClaims)
	assert.NoError(t, err)
	assert.NotNil(t, payload)
	assert.Equal(t, user.GetUsername(), payload.GetUsername())
	assert.Equal(t, user.GetId(), payload.GetUserId())
	assert.Equal(t, user.GetTenantId(), payload.GetTenantId())
	assert.Equal(t, client, payload.GetClientId())
	assert.Equal(t, device, payload.GetDeviceId())
	assert.Equal(t, []string{"r1", "r2"}, payload.GetRoles())
	if payload.DataScope != nil {
		assert.Equal(t, ds, payload.GetDataScope())
	}
	if payload.OrgUnitId != nil {
		assert.Equal(t, ou, payload.GetOrgUnitId())
	}
}

// TestNewUserTokenDataScopesRoundTrip 验证多角色聚合数据范围（dss/dsu 轨道）
// 的 claims 序列化与解析往返一致性。
func TestNewUserTokenDataScopesRoundTrip(t *testing.T) {
	ou := int64(7)
	uid := int64(3)
	tid := int64(4)
	dsUnits := []int64{7, 8}

	payload := NewUserTokenPayload(
		"bob", uid, tid, &ou, []string{"editor"},
		nil, nil, nil,
	)
	payload.DataScopes = []identityV1.DataScope{
		identityV1.DataScope_SELF,
		identityV1.DataScope_UNIT_ONLY,
	}
	payload.DataScopeUnitIds = dsUnits

	claims := NewUserTokenAuthClaims(payload, nil)
	assert.NotNil(t, claims)

	parsed, err := NewUserTokenPayloadWithClaims(claims)
	assert.NoError(t, err)
	assert.NotNil(t, parsed)

	assert.ElementsMatch(t,
		[]identityV1.DataScope{
			identityV1.DataScope_SELF,
			identityV1.DataScope_UNIT_ONLY,
		},
		parsed.GetDataScopes())
	assert.ElementsMatch(t, dsUnits, parsed.GetDataScopeUnitIds())
}

// TestNewUserTokenPayloadWithClaimsLegacyDsFallback 验证旧令牌（仅单值 ds 声明，
// 无 dss）的解析回退：单值字段照常落位，repeated 字段保持为空，
// 由 viewer 构建侧 BuildDataScopes 做单元素回退。
func TestNewUserTokenPayloadWithClaimsLegacyDsFallback(t *testing.T) {
	claims := &authn.AuthClaims{}
	(*claims)[authn.ClaimFieldSubject] = "legacy"
	(*claims)[ClaimFieldUserID] = int64(3)
	(*claims)[ClaimFieldTenantID] = int64(4)
	(*claims)[ClaimFieldDataScope] = identityV1.DataScope_ALL.String()

	payload, err := NewUserTokenPayloadWithClaims(claims)
	assert.NoError(t, err)
	assert.NotNil(t, payload)
	assert.Equal(t, identityV1.DataScope_ALL, payload.GetDataScope())
	assert.Empty(t, payload.GetDataScopes())
	assert.Empty(t, payload.GetDataScopeUnitIds())
}
