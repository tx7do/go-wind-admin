package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"github.com/redis/go-redis/v9"
	"github.com/tx7do/go-utils/jwtutil"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/proto"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
)

const (
	// MfaChallengeTTL 挑战/注册操作上下文的有效期。
	MfaChallengeTTL = 5 * time.Minute

	// mfaEnrollCooldown 注册发起冷却（防已登录用户循环 Start 塞 Redis）。
	mfaEnrollCooldown = 30 * time.Second

	// 登录挑战上下文 key 前缀：mfa:login:<operation_id>
	mfaLoginChallengeKeyFmt = "mfa:login:%s"
	// 注册上下文 key 前缀：mfa:enroll：<operation_id>
	mfaEnrollChallengeKeyFmt = "mfa:enroll:%s"
)

var ErrMfaChallengeNotFound = errors.New("mfa challenge not found or expired")

// MfaLoginChallengeContext 登录挑战上下文。
// 密码校验通过且用户绑定 TOTP 后，由 doGrantTypePassword 写入；
// VerifyMFAChallenge 取出并用于签发真 token。
type MfaLoginChallengeContext struct {
	Payload    *authenticationV1.UserTokenPayload
	ClientType authenticationV1.ClientType
}

// MfaEnrollChallengeContext 注册上下文。
// StartEnrollMethod 生成 TOTP secret 后写入；ConfirmEnrollMethod 取出 secret 校验首码。
type MfaEnrollChallengeContext struct {
	Secret   string
	TenantID uint32
	UserID   uint32
}

// MfaChallengeCache MFA 操作上下文缓存。
// 所有操作均 verify-and-delete 单次有效（取即删），与 captchaClient 一致。
type MfaChallengeCache struct {
	log *bLogger.Helper
	rdb *redis.Client
}

func NewMfaChallengeCache(ctx *bootstrap.Context, rdb *redis.Client) *MfaChallengeCache {
	return &MfaChallengeCache{
		rdb: rdb,
		log: ctx.NewLoggerHelper("mfa-challenge/cache"),
	}
}

// SetLoginChallenge 写入登录挑战上下文，返回 operation_id。
func (c *MfaChallengeCache) SetLoginChallenge(ctx context.Context, payload *authenticationV1.UserTokenPayload, clientType authenticationV1.ClientType) (string, error) {
	opId := newOperationID()
	data, err := proto.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal login challenge payload failed: %w", err)
	}
	envelope := mfaLoginChallengeEnvelope{
		Payload:    data,
		ClientType: int32(clientType),
	}
	raw, err := json.Marshal(envelope)
	if err != nil {
		return "", fmt.Errorf("marshal login challenge envelope failed: %w", err)
	}
	key := fmt.Sprintf(mfaLoginChallengeKeyFmt, opId)
	if err := c.rdb.Set(ctx, key, raw, MfaChallengeTTL).Err(); err != nil {
		c.log.Errorf(ctx, "set login challenge failed: %s", err.Error())
		return "", fmt.Errorf("set login challenge failed")
	}
	return opId, nil
}

// mfaLoginFailKeyFmt 登录挑战失败计数 key：mfa:loginfail:<operation_id>
const mfaLoginFailKeyFmt = "mfa:loginfail:%s"

// MaxLoginChallengeFailures 单个登录挑战允许的验证失败次数上限。
// 达到上限即作废挑战（用户需重新走密码登录）。TOTP 有效码空间 10^6、
// 每窗口 ±1 共 3 个有效码，3 次失败内随机命中的概率约 3×3/10^6，
// 兼顾"输错可重试"的体验与封死逐码暴破的安全。
const MaxLoginChallengeFailures = 3

// PeekLoginChallenge 只读登录挑战上下文（不删除，供失败重试）。
func (c *MfaChallengeCache) PeekLoginChallenge(ctx context.Context, opId string) (*MfaLoginChallengeContext, error) {
	key := fmt.Sprintf(mfaLoginChallengeKeyFmt, opId)
	raw, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrMfaChallengeNotFound
		}
		c.log.Errorf(ctx, "peek login challenge failed: %s", err.Error())
		return nil, fmt.Errorf("peek mfa challenge failed")
	}

	var envelope mfaLoginChallengeEnvelope
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		return nil, fmt.Errorf("unmarshal login challenge envelope failed: %w", err)
	}
	payload := &authenticationV1.UserTokenPayload{}
	if err := proto.Unmarshal(envelope.Payload, payload); err != nil {
		return nil, fmt.Errorf("unmarshal login challenge payload failed: %w", err)
	}
	return &MfaLoginChallengeContext{
		Payload:    payload,
		ClientType: authenticationV1.ClientType(envelope.ClientType),
	}, nil
}

// RecordLoginFailure 记录一次验证失败并返回挑战是否已达失败上限。
// 达上限时顺带作废挑战上下文（幂等：计数 key 与挑战 key 同 TTL 生命周期）。
func (c *MfaChallengeCache) RecordLoginFailure(ctx context.Context, opId string) (exceeded bool) {
	failKey := fmt.Sprintf(mfaLoginFailKeyFmt, opId)
	n, err := c.rdb.Incr(ctx, failKey).Result()
	if err != nil {
		// Redis 异常按已达上限处理（fail-closed，防计数失效后的无限重试）
		c.log.Errorf(ctx, "incr login challenge fail count failed: %s", err.Error())
		return true
	}
	c.rdb.Expire(ctx, failKey, MfaChallengeTTL)
	if n >= MaxLoginChallengeFailures {
		c.rdb.Del(ctx, fmt.Sprintf(mfaLoginChallengeKeyFmt, opId))
		return true
	}
	return false
}

// ConsumeLoginChallenge 消耗登录挑战（失败上限 / 因子缺失等无需裁决的场景）。
// 注意：验证通过的发 token 路径不能用它——普通 DEL 幂等，无法裁决并发双花，
// 必须走 TakeLoginChallengeAtomic。
func (c *MfaChallengeCache) ConsumeLoginChallenge(ctx context.Context, opId string) {
	c.rdb.Del(ctx, fmt.Sprintf(mfaLoginChallengeKeyFmt, opId), fmt.Sprintf(mfaLoginFailKeyFmt, opId))
}

// takeAndDelScript 原子 GET+DEL：返回键值或 nil（不存在/已被并发消耗）。
var takeAndDelScript = redis.NewScript(`
	local stored = redis.call('GET', KEYS[1])
	if not stored then
		return false
	end
	redis.call('DEL', KEYS[1])
	return stored
`)

// TakeLoginChallengeAtomic 原子消耗登录挑战，返回是否抢到（裁决权）。
// 验证通过后的发 token 路径用它做最终裁决：并发请求中仅先抢到者签发 token，
// 后到者（值已被删）拒绝——防止 Peek+DEL 组合下同 opId 双花签发两组令牌。
// 顺带清理失败计数 key。
func (c *MfaChallengeCache) TakeLoginChallengeAtomic(ctx context.Context, opId string) bool {
	key := fmt.Sprintf(mfaLoginChallengeKeyFmt, opId)
	res, err := takeAndDelScript.Run(ctx, c.rdb, []string{key}).Result()
	if err != nil {
		c.log.Errorf(ctx, "atomic take login challenge failed: %s", err.Error())
		return false
	}
	if res == nil {
		return false
	}
	c.rdb.Del(ctx, fmt.Sprintf(mfaLoginFailKeyFmt, opId))
	return true
}

// TryAcquireEnrollCooldown 注册频控：同一用户冷却期内只允许发起一次 StartEnroll。
// 防止已登录用户循环 Start 塞满 Redis（每条 5 分钟 TTL）的低成本 DoS。
func (c *MfaChallengeCache) TryAcquireEnrollCooldown(ctx context.Context, tenantID, userID uint32) bool {
	key := fmt.Sprintf("mfa:enrollcd:%d:%d", tenantID, userID)
	ok, err := c.rdb.SetNX(ctx, key, 1, mfaEnrollCooldown).Result()
	if err != nil {
		// Redis 异常放行（频控是防御性优化，不阻断正常注册）
		c.log.Errorf(ctx, "acquire enroll cooldown failed: %s", err.Error())
		return true
	}
	return ok
}

// SetEnrollChallenge 写入注册上下文，返回 operation_id。
func (c *MfaChallengeCache) SetEnrollChallenge(ctx context.Context, secret string, tenantID, userID uint32) (string, error) {
	opId := newOperationID()
	envelope := MfaEnrollChallengeContext{
		Secret:   secret,
		TenantID: tenantID,
		UserID:   userID,
	}
	raw, err := json.Marshal(envelope)
	if err != nil {
		return "", fmt.Errorf("marshal enroll challenge envelope failed: %w", err)
	}
	key := fmt.Sprintf(mfaEnrollChallengeKeyFmt, opId)
	if err := c.rdb.Set(ctx, key, raw, MfaChallengeTTL).Err(); err != nil {
		c.log.Errorf(ctx, "set enroll challenge failed: %s", err.Error())
		return "", fmt.Errorf("set enroll challenge failed")
	}
	return opId, nil
}

// PeekEnrollChallenge 只读注册上下文（不删除）。
// 注册流程允许首码输错重试：ConfirmEnrollMethod 校验失败后 operation 仍有效，
// 仅在落库成功后调用 DeleteEnrollChallenge 消耗。并发重复 Confirm 由
// (tenant,user,method) 唯一索引 + StartEnrollMethod 已绑定预检兜底。
func (c *MfaChallengeCache) PeekEnrollChallenge(ctx context.Context, opId string) (*MfaEnrollChallengeContext, error) {
	key := fmt.Sprintf(mfaEnrollChallengeKeyFmt, opId)
	raw, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrMfaChallengeNotFound
		}
		c.log.Errorf(ctx, "peek enroll challenge failed: %s", err.Error())
		return nil, fmt.Errorf("peek mfa challenge failed")
	}
	var envelope MfaEnrollChallengeContext
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		return nil, fmt.Errorf("unmarshal enroll challenge envelope failed: %w", err)
	}
	return &envelope, nil
}

// DeleteEnrollChallenge 删除注册上下文（注册成功后消耗）。
func (c *MfaChallengeCache) DeleteEnrollChallenge(ctx context.Context, opId string) {
	key := fmt.Sprintf(mfaEnrollChallengeKeyFmt, opId)
	c.rdb.Del(ctx, key)
}

// mfaLoginChallengeEnvelope 是登录挑战上下文的传输封装。
// payload 为 proto 序列化的 UserTokenPayload（含 roles/data_scope 等），clientType 为枚举值。
type mfaLoginChallengeEnvelope struct {
	Payload    []byte
	ClientType int32
}

func newOperationID() string {
	// 复用 jwtutil 的加密随机串生成（与 refresh token 同一来源），保证 operation_id
	// 不可预测且全局唯一。
	id, _ := jwtutil.NewRefreshToken()
	return id
}
