package jwt

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/tx7do/go-utils/trans"
	authn "github.com/tx7do/kratos-authn/engine"

	userV1 "go-wind-admin/api/gen/go/user/service/v1"
)

func TestNewUserTokenPayload(t *testing.T) {
	ou := uint32(42)
	uid := uint32(1)
	tid := uint32(2)
	username := "test_user"
	roleCodes := []string{"admin"}
	isP := true
	isT := true
	client := "client_123"
	device := "device_1"
	ds := userV1.Role_DataScope(1)

	payload := NewUserTokenPayload(
		username, uid, tid, &ou, roleCodes,
		&ds, &client, &device, &isP, &isT,
	)
	assert.NotNil(t, payload)
	assert.Equal(t, uid, payload.GetUserId())
	assert.Equal(t, tid, payload.GetTenantId())
	assert.Equal(t, username, payload.GetUsername())
	assert.Equal(t, client, payload.GetClientId())
	assert.Equal(t, device, payload.GetDeviceId())
	assert.Equal(t, []string{"admin"}, payload.GetRoles())
	assert.NotNil(t, payload.GetIsPlatformAdmin())
	assert.Equal(t, true, payload.GetIsPlatformAdmin())
	if payload.DataScope != nil {
		assert.Equal(t, ds, payload.GetDataScope())
	}
	if payload.OrgUnitId != nil {
		assert.Equal(t, ou, payload.GetOrgUnitId())
	}
}

func TestNewUserTokenAuthClaims(t *testing.T) {
	ou := uint32(7)
	uid := uint32(3)
	tid := uint32(4)
	username := "alice"
	roleCodes := []string{"editor"}
	isP := false
	isT := false
	client := "cli"
	device := "dev"
	ds := userV1.Role_DataScope(2)

	payload := NewUserTokenPayload(
		username, uid, tid, &ou, roleCodes,
		&ds, &client, &device, &isP, &isT,
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
	// platform admin stored as bool
	assert.Equal(t, false, (*claims)[ClaimFieldIsPlatformAdmin])
	// org unit
	assert.Equal(t, ou, (*claims)[ClaimFieldOrgUnitID])
}

func TestNewUserTokenPayloadWithClaims(t *testing.T) {
	user := userV1.User{
		Id:       trans.Ptr(uint32(5)),
		TenantId: trans.Ptr(uint32(6)),
		Username: trans.Ptr("bob"),
		Roles:    []string{"viewer"},
	}

	client := "cid"
	device := "did"
	ds := userV1.Role_DataScope(3)
	ou := uint32(9)

	claims := &authn.AuthClaims{
		authn.ClaimFieldSubject: user.GetUsername(),
		ClaimFieldUserID:        user.GetId(),
		ClaimFieldTenantID:      user.GetTenantId(),
		ClaimFieldClientID:      client,
		ClaimFieldDeviceID:      device,
		ClaimFieldRoleCodes:     user.Roles,
		ClaimFieldDataScope:     ds.String(),
		// NewUserTokenPayloadWithClaims 使用 GetInt 来读取，0/非0 做布尔判断
		ClaimFieldIsPlatformAdmin: 1,
		ClaimFieldOrgUnitID:       ou,
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
	// IsPlatformAdmin is set from int->bool
	if payload.IsPlatformAdmin != nil {
		assert.Equal(t, true, payload.GetIsPlatformAdmin())
	}
	if payload.OrgUnitId != nil {
		assert.Equal(t, ou, payload.GetOrgUnitId())
	}
}

func TestNewUserTokenPayloadWithJwtMapClaims(t *testing.T) {
	user := userV1.User{
		Id:       trans.Ptr(uint32(10)),
		TenantId: trans.Ptr(uint32(11)),
		Username: trans.Ptr("eve"),
		Roles:    []string{"r1", "r2"},
	}

	client := "c1"
	device := "d1"
	ds := userV1.Role_DataScope(4)
	ou := uint32(21)

	// jwt.MapClaims uses float64 for numeric JSON numbers
	mapClaims := jwt.MapClaims{
		"sub":                     user.GetUsername(),
		ClaimFieldUserID:          float64(user.GetId()),
		ClaimFieldTenantID:        float64(user.GetTenantId()),
		ClaimFieldClientID:        client,
		ClaimFieldDeviceID:        device,
		ClaimFieldDataScope:       ds.String(),
		ClaimFieldIsPlatformAdmin: true,
		ClaimFieldOrgUnitID:       float64(ou),
		ClaimFieldRoleCodes:       []interface{}{"r1", "r2"},
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
	if payload.IsPlatformAdmin != nil {
		assert.Equal(t, true, payload.GetIsPlatformAdmin())
	}
	if payload.OrgUnitId != nil {
		assert.Equal(t, ou, payload.GetOrgUnitId())
	}
}
