package logging

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/tx7do/go-utils/timeutil"
	"github.com/tx7do/go-utils/trans"
	"google.golang.org/protobuf/proto"

	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
)

type LoginAuditLogMiddleware struct {
	op *options
}

func NewLoginAuditLogMiddleware(op *options) *LoginAuditLogMiddleware {
	return &LoginAuditLogMiddleware{
		op: op,
	}
}

func (l *LoginAuditLogMiddleware) Name() string {
	return "LoginAuditLogMiddleware"
}

func (l *LoginAuditLogMiddleware) Handle(ctx context.Context, htr *http.Transport, middleErr error) {
	if l.op == nil {
		return
	}
	if htr == nil {
		return
	}

	if htr.Operation() != l.op.loginOperation && htr.Operation() != l.op.logoutOperation {
		return
	}

	// 获取错误码和是否成功
	_, reason, success := getStatusCode(middleErr)

	loginAuditLog := &auditV1.LoginAuditLog{}

	switch htr.Operation() {
	case l.op.loginOperation:
		loginAuditLog.ActionType = trans.Ptr(auditV1.LoginAuditLog_LOGIN)
	case l.op.logoutOperation:
		loginAuditLog.ActionType = trans.Ptr(auditV1.LoginAuditLog_LOGOUT)
	}

	clientIp := getClientRealIP(htr.Request())

	loginAuditLog.IpAddress = trans.Ptr(clientIp)
	loginAuditLog.CreatedAt = timeutil.TimeToTimestamppb(trans.Ptr(time.Now()))

	loginAuditLog.GeoLocation = fillGeoLocation(clientIp)

	if username, err := extractUsernameFromRequest(htr.Request()); err == nil {
		loginAuditLog.Username = trans.Ptr(username)
	}

	ut := extractAuthToken(htr)
	if ut != nil {
		loginAuditLog.UserId = trans.Ptr(ut.UserId)
		loginAuditLog.TenantId = ut.TenantId
		if loginAuditLog.Username == nil {
			loginAuditLog.Username = ut.Username
		}
	}

	// 用户设备信息
	loginAuditLog.DeviceInfo = fillDeviceInfo(htr, ut)

	// 获取客户端ID
	loginAuditLog.RequestId = trans.Ptr(getRequestId(htr.Request()))

	loginAuditLog.FailureReason = trans.Ptr(reason)

	if success {
		loginAuditLog.Status = trans.Ptr(auditV1.LoginAuditLog_SUCCESS)
	} else {
		loginAuditLog.Status = trans.Ptr(auditV1.LoginAuditLog_FAILED)
	}

	// 计算风险分数和风险等级
	riskScore := l.computeRiskScore(loginAuditLog)
	loginAuditLog.RiskScore = trans.Ptr(riskScore)
	loginAuditLog.RiskLevel = trans.Ptr(l.levelFromScore(riskScore))

	// 计算风险因素
	loginAuditLog.RiskFactors = l.computeRiskFactors(loginAuditLog)

	// 计算哈希和签名
	loginAuditLog.LogHash = trans.Ptr(l.hashLog(loginAuditLog))
	loginAuditLog.Signature = l.signature(loginAuditLog)

	// 写入日志
	if l.op.writeLoginLogFunc != nil {
		_ = l.op.writeLoginLogFunc(ctx, loginAuditLog)
	}
}

// hashLog 计算日志的 SHA256 哈希（十六进制小写字符串）
// 规则：排除 log_hash 和 signature 字段，Protobuf 确定性序列化后哈希
func (l *LoginAuditLogMiddleware) hashLog(loginAuditLog *auditV1.LoginAuditLog) string {
	if loginAuditLog == nil {
		return ""
	}

	loginAuditLog.LogHash = nil
	loginAuditLog.Signature = nil

	rawBytes, err := proto.Marshal(loginAuditLog)
	if err != nil {
		fmt.Printf("marshal log failed: %v\n", err)
		return ""
	}

	hash := sha256.Sum256(rawBytes)
	return hex.EncodeToString(hash[:])
}

// signature 生成日志的 ECDSA 数字签名
// 签名内容：tenant_id + user_id + created_at（原始时间戳） + log_hash
// 返回：ECDSA 签名字节数组（r+s 拼接，DER 格式）
func (l *LoginAuditLogMiddleware) signature(loginAuditLog *auditV1.LoginAuditLog) []byte {
	if loginAuditLog == nil || l.op.ecPrivateKey == nil {
		return nil
	}

	tenantID := loginAuditLog.GetTenantId()
	userID := loginAuditLog.GetUserId()
	username := loginAuditLog.GetUsername()
	logHash := loginAuditLog.GetLogHash()
	createdAt := loginAuditLog.GetCreatedAt()

	type signContent struct {
		TenantID uint32 `json:"tenant_id"`
		UserID   uint32 `json:"user_id"`
		Username string `json:"username"`
		Sec      int64  `json:"sec"`   // createdAt 秒数
		Nanos    int32  `json:"nanos"` // createdAt 纳秒数
		LogHash  string `json:"log_hash"`
	}
	sc := signContent{
		TenantID: tenantID,
		UserID:   userID,
		Username: username,
		LogHash:  logHash,
	}
	if createdAt != nil {
		sc.Sec = createdAt.Seconds
		sc.Nanos = createdAt.Nanos
	}

	scBytes, err := json.Marshal(sc)
	if err != nil {
		fmt.Printf("marshal sign content failed: %v\n", err)
		return nil
	}

	scHash := sha256.Sum256(scBytes)

	r, s, err := ecdsa.Sign(rand.Reader, l.op.ecPrivateKey, scHash[:])
	if err != nil {
		fmt.Printf("ECDSA sign failed: %v\n", err)
		return nil
	}

	signBytes, err := encodeDER(r, s)
	if err != nil {
		fmt.Printf("encode DER failed: %v\n", err)
		return nil
	}

	return signBytes
}

// computeRiskScore 计算登录审计日志的风险分数（0-100）
func (l *LoginAuditLogMiddleware) computeRiskScore(loginAuditLog *auditV1.LoginAuditLog) uint32 {
	if loginAuditLog == nil {
		return 0
	}

	score := 0

	// 失败登录权重较高
	if loginAuditLog.GetStatus() == auditV1.LoginAuditLog_FAILED {
		score += 50
	}

	// userId 不存在时增加风险（登录失败常见）
	if loginAuditLog.GetUserId() == 0 {
		if loginAuditLog.GetUsername() != "" {
			score += 10
		} else {
			score += 20
		}
	}

	// clientId 缺失增风险
	if di := loginAuditLog.GetDeviceInfo(); di == nil || di.GetClientId() == "" {
		score += 10
	}

	// IP 为空增风险；内网 IP 适度降低风险
	ip := loginAuditLog.GetIpAddress()
	if ip == "" {
		score += 5
	} else {
		if isPrivateIP(ip) {
			score -= 10
		}
	}

	// 截断到 0-100
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return uint32(score)
}

// levelFromScore 根据 risk_score 映射到 risk_level，阈值可按需调整。
func (l *LoginAuditLogMiddleware) levelFromScore(score uint32) auditV1.LoginAuditLog_RiskLevel {
	// 常见阈值：0-30 -> LOW；31-70 -> MEDIUM；71-100 -> HIGH
	switch {
	case score <= 30:
		return auditV1.LoginAuditLog_LOW
	case score <= 70:
		return auditV1.LoginAuditLog_MEDIUM
	default:
		return auditV1.LoginAuditLog_HIGH
	}
}

// risk factor 常量定义
const (
	RiskFactorFailedLogin      = "FAILED_LOGIN"
	RiskFactorUnknownUser      = "UNKNOWN_USER"
	RiskFactorAnonymousLogin   = "ANONYMOUS_LOGIN"
	RiskFactorUnknownDevice    = "UNKNOWN_DEVICE"
	RiskFactorMfaFailed        = "MFA_FAILED"
	RiskFactorMfaUnverified    = "MFA_UNVERIFIED"
	RiskFactorIpMissing        = "IP_MISSING"
	RiskFactorInternalIP       = "INTERNAL_IP"
	RiskFactorExternalIP       = "EXTERNAL_IP"
	RiskFactorPasswordFailure  = "PASSWORD_FAILURE"
	RiskFactorMfaFailureReason = "MFA_FAILURE_REASON"
	RiskFactorNoSession        = "NO_SESSION"
	RiskFactorNoRequestID      = "NO_REQUEST_ID"
	RiskFactorHighRiskScore    = "HIGH_RISK_SCORE"
	RiskFactorMediumRiskScore  = "MEDIUM_RISK_SCORE"
	RiskFactorLowRiskScore     = "LOW_RISK_SCORE"
)

// computeRiskFactors 基于 LoginAuditLog 的若干字段，使用无状态启发式规则返回风险因素列表（去重、排序）。
func (l *LoginAuditLogMiddleware) computeRiskFactors(la *auditV1.LoginAuditLog) []string {
	if la == nil {
		return nil
	}

	set := map[string]struct{}{}
	add := func(s string) { set[s] = struct{}{} }

	// 登录结果
	if la.GetStatus() == auditV1.LoginAuditLog_FAILED {
		add(RiskFactorFailedLogin)
	}

	// user id / username
	if la.GetUserId() == 0 {
		if la.GetUsername() != "" {
			add(RiskFactorUnknownUser)
		} else {
			add(RiskFactorAnonymousLogin)
		}
	}

	// 设备 / client id
	if di := la.GetDeviceInfo(); di == nil || di.GetClientId() == "" {
		add(RiskFactorUnknownDevice)
	}

	// MFA 状态
	mfa := strings.ToUpper(strings.TrimSpace(la.GetMfaStatus()))
	if mfa != "" {
		if strings.Contains(mfa, "FAILED") {
			add(RiskFactorMfaFailed)
		}
		if strings.Contains(mfa, "UNVERIFIED") || strings.Contains(mfa, "UNVERIFY") {
			add(RiskFactorMfaUnverified)
		}
	}

	// IP / 内外网判断
	ip := strings.TrimSpace(la.GetIpAddress())
	if ip == "" {
		add(RiskFactorIpMissing)
	} else {
		if isPrivateIP(ip) {
			add(RiskFactorInternalIP)
		} else {
			add(RiskFactorExternalIP)
		}
	}

	// 失败原因关键词
	if fr := strings.ToLower(strings.TrimSpace(la.GetFailureReason())); fr != "" {
		if strings.Contains(fr, "password") ||
			strings.Contains(fr, "pwd") ||
			strings.Contains(fr, "incorrect") {
			add(RiskFactorPasswordFailure)
		}
		if strings.Contains(fr, "mfa") {
			add(RiskFactorMfaFailureReason)
		}
	}

	// session / request id
	if la.GetSessionId() == "" {
		add(RiskFactorNoSession)
	}
	if la.GetRequestId() == "" {
		add(RiskFactorNoRequestID)
	}

	// 基于 risk_score 的衍生因子
	switch s := la.GetRiskScore(); {
	case s >= 71:
		add(RiskFactorHighRiskScore)
	case s >= 31:
		add(RiskFactorMediumRiskScore)
	default:
		if s > 0 {
			add(RiskFactorLowRiskScore)
		}
	}

	// 收集并返回稳定顺序的切片
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
