package jwt

import (
	"errors"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/golang-jwt/jwt/v5"
	"github.com/tx7do/go-utils/trans"

	authn "github.com/tx7do/kratos-authn/engine"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	userV1 "go-wind-admin/api/gen/go/user/service/v1"
)

const (
	ClaimFieldUserName        = authn.ClaimFieldSubject // 用户名
	ClaimFieldUserID          = "uid"                   // 用户 ID
	ClaimFieldTenantID        = "tid"                   // 租户 ID
	ClaimFieldClientID        = "cid"                   // 客户端 ID
	ClaimFieldDeviceID        = "did"                   // 设备 ID
	ClaimFieldRoleCodes       = "roc"                   // 角色码列表
	ClaimFieldIsPlatformAdmin = "pad"                   // 是否为平台管理员
	ClaimFieldIsTenantAdmin   = "tad"                   // 是否为租户管理员
	ClaimFieldDataScope       = "ds"                    // 数据范围
	ClaimFieldOrgUnitID       = "ouid"                  // 组织单元 ID
)

// NewUserTokenPayload 创建用户令牌
func NewUserTokenPayload(
	username string,
	userID uint32,
	tenantID uint32,
	orgUnitID *uint32,
	roleCodes []string,
	dataScope *userV1.Role_DataScope,
	clientID *string,
	deviceID *string,
	isPlatformAdmin *bool,
	isTenantAdmin *bool,
) *authenticationV1.UserTokenPayload {
	if isTenantAdmin != nil && *isTenantAdmin == true {
		if tenantID == 0 {
			*isTenantAdmin = false
		}
	}

	return &authenticationV1.UserTokenPayload{
		Username:        trans.Ptr(username),
		UserId:          userID,
		TenantId:        trans.Ptr(tenantID),
		OrgUnitId:       orgUnitID,
		Roles:           roleCodes,
		ClientId:        clientID,
		DeviceId:        deviceID,
		IsPlatformAdmin: isPlatformAdmin,
		IsTenantAdmin:   isTenantAdmin,
		DataScope:       dataScope,
	}
}

func NewUserTokenAuthClaims(
	tokenPayload *authenticationV1.UserTokenPayload,
) *authn.AuthClaims {
	authClaims := authn.AuthClaims{
		ClaimFieldUserName: tokenPayload.GetUsername(),
		ClaimFieldUserID:   tokenPayload.GetUserId(),
		ClaimFieldTenantID: tokenPayload.GetTenantId(),
	}

	if len(tokenPayload.Roles) > 0 {
		authClaims[ClaimFieldRoleCodes] = tokenPayload.Roles
	}
	if tokenPayload.DeviceId != nil {
		authClaims[ClaimFieldDeviceID] = tokenPayload.GetDeviceId()
	}
	if tokenPayload.ClientId != nil {
		authClaims[ClaimFieldClientID] = tokenPayload.GetClientId()
	}

	if tokenPayload.DataScope != nil {
		authClaims[ClaimFieldDataScope] = tokenPayload.GetDataScope().String()
	}
	if tokenPayload.OrgUnitId != nil {
		authClaims[ClaimFieldOrgUnitID] = tokenPayload.GetOrgUnitId()
	}

	if tokenPayload.IsPlatformAdmin != nil {
		intValue := 0
		if tokenPayload.GetIsPlatformAdmin() {
			intValue = 1
		}
		authClaims[ClaimFieldIsPlatformAdmin] = intValue
	}

	if tokenPayload.IsTenantAdmin != nil {
		intValue := 0
		if tokenPayload.GetIsTenantAdmin() {
			intValue = 1
		}
		authClaims[ClaimFieldIsTenantAdmin] = intValue
	}

	return &authClaims
}

func NewUserTokenPayloadWithClaims(claims *authn.AuthClaims) (*authenticationV1.UserTokenPayload, error) {
	payload := &authenticationV1.UserTokenPayload{}

	sub, err := claims.GetSubject()
	if err != nil {
		log.Errorf("GetSubject failed: %v", err)
	}
	if sub != "" {
		payload.Username = trans.Ptr(sub)
	}

	userId, err := claims.GetUint32(ClaimFieldUserID)
	if err != nil {
		log.Errorf("GetUint32 ClaimFieldUserID failed: %v", err)
	}
	if userId != 0 {
		payload.UserId = userId
	}

	tenantId, err := claims.GetUint32(ClaimFieldTenantID)
	if err != nil {
		log.Errorf("GetUint32 ClaimFieldTenantID failed: %v", err)
	}
	if tenantId != 0 {
		payload.TenantId = trans.Ptr(tenantId)
	}

	clientId, err := claims.GetString(ClaimFieldClientID)
	if err != nil {
		log.Errorf("GetString ClaimFieldClientID failed: %v", err)
	}
	if clientId != "" {
		payload.ClientId = trans.Ptr(clientId)
	}

	deviceId, err := claims.GetString(ClaimFieldDeviceID)
	if err != nil {
		log.Errorf("GetString ClaimFieldDeviceID failed: %v", err)
	}
	if deviceId != "" {
		payload.DeviceId = trans.Ptr(deviceId)
	}

	roleCodes, err := claims.GetStrings(ClaimFieldRoleCodes)
	if err != nil {
		log.Errorf("GetStrings ClaimFieldRoleCodes failed: %v", err)
	}
	if roleCodes != nil {
		payload.Roles = roleCodes
	}

	dataScope, err := claims.GetString(ClaimFieldDataScope)
	if err != nil {
		log.Errorf("GetString ClaimFieldDataScope failed: %v", err)
	}
	if dataScope != "" {
		v, ok := userV1.Role_DataScope_value[dataScope]
		if ok {
			payload.DataScope = trans.Ptr(userV1.Role_DataScope(v))
		}
	}

	orgUnitID, err := claims.GetUint32(ClaimFieldOrgUnitID)
	if err != nil {
		log.Errorf("GetUint32 ClaimFieldOrgUnitID failed: %v", err)
	}
	if orgUnitID != 0 {
		payload.OrgUnitId = trans.Ptr(orgUnitID)
	}

	isPlatformAdmin, err := claims.GetInt(ClaimFieldIsPlatformAdmin)
	if err != nil {
		log.Errorf("GetBool ClaimFieldIsPlatformAdmin failed: %v", err)
	}
	b := isPlatformAdmin != 0
	payload.IsPlatformAdmin = trans.Ptr(b)

	isTenantAdmin, err := claims.GetInt(ClaimFieldIsTenantAdmin)
	if err != nil {
		log.Errorf("GetBool ClaimFieldIsTenantAdmin failed: %v", err)
	}
	b = isTenantAdmin != 0
	payload.IsTenantAdmin = trans.Ptr(b)

	return payload, nil
}

func NewUserTokenPayloadWithJwtMapClaims(claims jwt.MapClaims) (*authenticationV1.UserTokenPayload, error) {
	payload := &authenticationV1.UserTokenPayload{}

	sub, err := claims.GetSubject()
	if err != nil {
		log.Errorf("GetSubject failed: %v", err)
	}
	if sub != "" {
		payload.Username = trans.Ptr(sub)
	}

	userId, _ := claims[ClaimFieldUserID]
	if userId != nil {
		payload.UserId = uint32(userId.(float64))
	}

	tenantId, _ := claims[ClaimFieldTenantID]
	if tenantId != nil {
		payload.TenantId = trans.Ptr(uint32(tenantId.(float64)))
	}

	clientId, _ := claims[ClaimFieldClientID]
	if clientId != nil {
		payload.ClientId = trans.Ptr(clientId.(string))
	}

	deviceId, _ := claims[ClaimFieldDeviceID]
	if deviceId != nil {
		payload.DeviceId = trans.Ptr(deviceId.(string))
	}

	dataScope, _ := claims[ClaimFieldDataScope]
	if dataScope != nil {
		v, ok := userV1.Role_DataScope_value[dataScope.(string)]
		if ok {
			payload.DataScope = trans.Ptr(userV1.Role_DataScope(v))
		}
	}

	orgUnitID, _ := claims[ClaimFieldOrgUnitID]
	if orgUnitID != nil {
		payload.OrgUnitId = trans.Ptr(uint32(orgUnitID.(float64)))
	}

	roleCodes, _ := claims[ClaimFieldRoleCodes]
	if roleCodes != nil {
		switch itf := roleCodes.(type) {
		case []interface{}:
			for _, rc := range itf {
				payload.Roles = append(payload.Roles, rc.(string))
			}

		case []string:
			payload.Roles = itf

		default:
			return nil, errors.New("invalid roleCodes type")
		}
	}

	isPlatformAdmin, _ := claims[ClaimFieldIsPlatformAdmin]
	if isPlatformAdmin != nil {
		intValue := isPlatformAdmin.(float64)
		b := false
		if intValue != 0 {
			b = true
		}
		payload.IsPlatformAdmin = trans.Ptr(b)
	}

	isTenantAdmin, _ := claims[ClaimFieldIsTenantAdmin]
	if isTenantAdmin != nil {
		intValue := isTenantAdmin.(float64)
		b := false
		if intValue != 0 {
			b = true
		}
		payload.IsTenantAdmin = trans.Ptr(b)
	}

	return payload, nil
}
