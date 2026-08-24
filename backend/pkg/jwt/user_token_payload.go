package jwt

import (
	"context"
	"errors"
	"fmt"
	"time"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"github.com/golang-jwt/jwt/v5"
	"github.com/tx7do/go-utils/trans"

	authn "github.com/tx7do/kratos-authn/engine"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
)

const (
	ClaimFieldUserName  = authn.ClaimFieldSubject // 用户名
	ClaimFieldUserID    = "uid"                   // 用户 ID
	ClaimFieldTenantID  = "tid"                   // 租户 ID
	ClaimFieldClientID  = "cid"                   // 客户端 ID
	ClaimFieldDeviceID  = "did"                   // 设备 ID
	ClaimFieldRoleCodes = "roc"                   // 角色码列表
	ClaimFieldDataScope = "ds"                    // 数据范围
	ClaimFieldOrgUnitID = "ouid"                  // 组织单元 ID

	ClaimFieldIsPlatformAdmin = "ipa" // 是否平台管理员
	ClaimFieldIsTenantAdmin   = "ita" // 是否租户管理员
)

const (
	// defaultTokenLeeway 令牌时间容差，防止因时间不同步导致的验证失败
	defaultTokenLeeway = 60 * time.Second

	// DefaultTokenExpiration 默认令牌过期时间：2 小时
	DefaultTokenExpiration = 2 * time.Hour

	// DefaultRefreshTokenExpiration 默认刷新令牌过期时间：7 天
	DefaultRefreshTokenExpiration = 7 * 24 * time.Hour
)

// NewUserTokenPayload 创建用户令牌
func NewUserTokenPayload(
	username string,
	userID uint32,
	tenantID uint32,
	orgUnitID *uint32,
	roleCodes []string,
	dataScope *identityV1.DataScope,
	clientID *string,
	deviceID *string,
) *authenticationV1.UserTokenPayload {
	return &authenticationV1.UserTokenPayload{
		Username:  trans.Ptr(username),
		UserId:    userID,
		TenantId:  trans.Ptr(tenantID),
		OrgUnitId: orgUnitID,
		Roles:     roleCodes,
		ClientId:  clientID,
		DeviceId:  deviceID,
		DataScope: dataScope,
	}
}

// NewUserTokenAuthClaims 创建用户令牌认证声明
func NewUserTokenAuthClaims(
	tokenPayload *authenticationV1.UserTokenPayload,
	expirationTime *time.Time,
) *authn.AuthClaims {
	authClaims := authn.AuthClaims{
		ClaimFieldUserName:       tokenPayload.GetUsername(),
		ClaimFieldUserID:         tokenPayload.GetUserId(),
		ClaimFieldTenantID:       tokenPayload.GetTenantId(),
		authn.ClaimFieldIssuedAt: time.Now().Unix(),
	}

	if expirationTime != nil {
		authClaims[authn.ClaimFieldExpirationTime] = expirationTime.Unix()
	}

	if tokenPayload.Jti != nil {
		authClaims[authn.ClaimFieldJwtID] = tokenPayload.GetJti()
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
		authClaims[ClaimFieldIsPlatformAdmin] = tokenPayload.GetIsPlatformAdmin()
	}
	if tokenPayload.IsTenantAdmin != nil {
		authClaims[ClaimFieldIsTenantAdmin] = tokenPayload.GetIsTenantAdmin()
	}

	return &authClaims
}

// NewUserTokenPayloadWithClaims 从认证声明创建用户令牌
func NewUserTokenPayloadWithClaims(claims *authn.AuthClaims) (*authenticationV1.UserTokenPayload, error) {
	payload := &authenticationV1.UserTokenPayload{}

	sub, err := claims.GetSubject()
	if err != nil {
		bLogger.GetLogger().Error(context.Background(), fmt.Sprintf("GetSubject failed: %v", err))
	}
	if sub != "" {
		payload.Username = trans.Ptr(sub)
	}

	jti, err := claims.GetJwtID()
	if err != nil {
		bLogger.GetLogger().Error(context.Background(), fmt.Sprintf("GetJwtID failed: %v", err))
	}
	if jti != "" {
		payload.Jti = trans.Ptr(jti)
	}

	userId, err := claims.GetUint32(ClaimFieldUserID)
	if err != nil {
		bLogger.GetLogger().Error(context.Background(), fmt.Sprintf("GetUint32 ClaimFieldUserID failed: %v", err))
	}
	if userId != 0 {
		payload.UserId = userId
	}

	tenantId, err := claims.GetUint32(ClaimFieldTenantID)
	if err != nil {
		bLogger.GetLogger().Error(context.Background(), fmt.Sprintf("GetUint32 ClaimFieldTenantID failed: %v", err))
	}
	if tenantId != 0 {
		payload.TenantId = trans.Ptr(tenantId)
	}

	clientId, err := claims.GetString(ClaimFieldClientID)
	if err != nil {
		bLogger.GetLogger().Error(context.Background(), fmt.Sprintf("GetString ClaimFieldClientID failed: %v", err))
	}
	if clientId != "" {
		payload.ClientId = trans.Ptr(clientId)
	}

	deviceId, err := claims.GetString(ClaimFieldDeviceID)
	if err != nil {
		bLogger.GetLogger().Error(context.Background(), fmt.Sprintf("GetString ClaimFieldDeviceID failed: %v", err))
	}
	if deviceId != "" {
		payload.DeviceId = trans.Ptr(deviceId)
	}

	roleCodes, err := claims.GetStrings(ClaimFieldRoleCodes)
	if err != nil {
		bLogger.GetLogger().Error(context.Background(), fmt.Sprintf("GetStrings ClaimFieldRoleCodes failed: %v", err))
	}
	if roleCodes != nil {
		payload.Roles = roleCodes
	}

	dataScope, err := claims.GetString(ClaimFieldDataScope)
	if err != nil {
		bLogger.GetLogger().Error(context.Background(), fmt.Sprintf("GetString ClaimFieldDataScope failed: %v", err))
	}
	if dataScope != "" {
		v, ok := identityV1.DataScope_value[dataScope]
		if ok {
			payload.DataScope = trans.Ptr(identityV1.DataScope(v))
		}
	}

	orgUnitID, err := claims.GetUint32(ClaimFieldOrgUnitID)
	if err != nil {
		bLogger.GetLogger().Error(context.Background(), fmt.Sprintf("GetUint32 ClaimFieldOrgUnitID failed: %v", err))
	}
	if orgUnitID != 0 {
		payload.OrgUnitId = trans.Ptr(orgUnitID)
	}

	if ipa, ok := (*claims)[ClaimFieldIsPlatformAdmin].(bool); ok && ipa {
		payload.IsPlatformAdmin = trans.Ptr(true)
	}
	if ita, ok := (*claims)[ClaimFieldIsTenantAdmin].(bool); ok && ita {
		payload.IsTenantAdmin = trans.Ptr(true)
	}

	return payload, nil
}

// NewUserTokenPayloadWithJwtMapClaims 从 JWT MapClaims 创建用户令牌。
//
// 安全说明：该函数会被审计日志中间件在请求路径上用 ParseUnverified（不验签）
// 解析客户端可伪造的 Authorization 头，因此对每个 claim 必须用两值类型断言，
// 避免 claim 类型不符时 panic 崩溃请求 goroutine（未认证远程 DoS）。
func NewUserTokenPayloadWithJwtMapClaims(claims jwt.MapClaims) (*authenticationV1.UserTokenPayload, error) {
	payload := &authenticationV1.UserTokenPayload{}

	sub, err := claims.GetSubject()
	if err != nil {
		bLogger.GetLogger().Error(context.Background(), fmt.Sprintf("GetSubject failed: %v", err))
	}
	if sub != "" {
		payload.Username = trans.Ptr(sub)
	}

	// JSON 数字在 Go 里解码为 float64；这里两值断言，类型不符则跳过而非 panic。
	if userId, ok := claims[ClaimFieldUserID].(float64); ok {
		payload.UserId = uint32(userId)
	}

	if tenantId, ok := claims[ClaimFieldTenantID].(float64); ok {
		payload.TenantId = trans.Ptr(uint32(tenantId))
	}

	if clientId, ok := claims[ClaimFieldClientID].(string); ok {
		payload.ClientId = trans.Ptr(clientId)
	}

	if deviceId, ok := claims[ClaimFieldDeviceID].(string); ok {
		payload.DeviceId = trans.Ptr(deviceId)
	}

	if dataScope, ok := claims[ClaimFieldDataScope].(string); ok {
		if v, vok := identityV1.DataScope_value[dataScope]; vok {
			payload.DataScope = trans.Ptr(identityV1.DataScope(v))
		}
	}

	if orgUnitID, ok := claims[ClaimFieldOrgUnitID].(float64); ok {
		payload.OrgUnitId = trans.Ptr(uint32(orgUnitID))
	}

	if ipa, ok := claims[ClaimFieldIsPlatformAdmin].(bool); ok && ipa {
		payload.IsPlatformAdmin = trans.Ptr(true)
	}
	if ita, ok := claims[ClaimFieldIsTenantAdmin].(bool); ok && ita {
		payload.IsTenantAdmin = trans.Ptr(true)
	}

	roleCodes := claims[ClaimFieldRoleCodes]
	if roleCodes != nil {
		switch itf := roleCodes.(type) {
		case []interface{}:
			for _, rc := range itf {
				// 数组元素同样两值断言，类型不符则跳过该元素。
				if rcStr, rcOk := rc.(string); rcOk {
					payload.Roles = append(payload.Roles, rcStr)
				}
			}

		case []string:
			payload.Roles = itf

		default:
			return nil, errors.New("invalid roleCodes type")
		}
	}

	return payload, nil
}

// IsTokenExpired 检查令牌是否过期
func IsTokenExpired(claims *authn.AuthClaims) bool {
	if claims == nil {
		return true
	}

	exp, _ := claims.GetExpirationTime()
	if exp == nil {
		// 没有 exp 声明时不认为是过期（按原逻辑）
		return false
	}

	now := time.Now().UTC()
	return now.After(exp.Time.UTC().Add(defaultTokenLeeway))
}

// IsTokenNotValidYet 检查令牌是否未生效
func IsTokenNotValidYet(claims *authn.AuthClaims) bool {
	if claims == nil {
		return true
	}

	nbf, _ := claims.GetNotBefore()
	if nbf == nil {
		// 没有 nbf 声明时不认为是未生效
		return false
	}

	now := time.Now().UTC()
	return now.Add(defaultTokenLeeway).Before(nbf.UTC())
}
