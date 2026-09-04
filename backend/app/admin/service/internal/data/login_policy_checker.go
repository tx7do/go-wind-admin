package data

import (
	"fmt"
	"net"
	"strings"
	"time"
)

// 登录策略匹配器：纯函数，供登录闸门调用，语义按维度独立判定——
//   - 黑名单：任一条目命中 → 拒绝
//   - 白名单：该维度存在白名单约束（全局或定向当前用户）且当前值未命中任何白名单 → 拒绝
//
// 生效范围：条目 TargetID 为 0 表示全局（约束所有用户），否则仅约束该用户；
// userId 传 0 表示只检查全局条目（密码校验前的第一段），取到 userId 后再查第二段。
//
// 维度支持：IP（精确 IP 或 CIDR）、TIME（HH:MM-HH:MM 时间窗，支持跨午夜）、
// DEVICE（device_id 精确匹配）。MAC 与 REGION 第一版不判定：HTTP 请求上下文
// 拿不到 MAC 地址，REGION 依赖 IP 地理库（后续可复用登录审计的 GeoLocation 能力接入）。
func MatchLoginPolicy(
	policies []EffectivePolicy,
	userId uint32,
	clientIP string,
	deviceId string,
	now time.Time,
) (blocked bool, reason string) {
	for _, method := range []string{"IP", "TIME", "DEVICE"} {
		// 该维度对当前用户生效的条目（全局 + 定向）
		var blacks, whites []EffectivePolicy
		for _, p := range policies {
			if p.Method != method {
				continue
			}
			if p.TargetID != 0 && p.TargetID != userId {
				continue
			}
			if p.Type == "WHITELIST" {
				whites = append(whites, p)
			} else {
				blacks = append(blacks, p)
			}
		}

		var matched func(v string) bool
		switch method {
		case "IP":
			matched = func(v string) bool { return matchIPValue(clientIP, v) }
		case "TIME":
			matched = func(v string) bool { return matchTimeWindow(now, v) }
		case "DEVICE":
			matched = func(v string) bool { return deviceId != "" && deviceId == v }
		}

		for _, p := range blacks {
			if matched(p.Value) {
				return true, p.describe(method)
			}
		}
		if len(whites) > 0 {
			hit := false
			for _, p := range whites {
				if matched(p.Value) {
					hit = true
					break
				}
			}
			if !hit {
				return true, "not in " + strings.ToLower(method) + " whitelist"
			}
		}
	}
	return false, ""
}

// matchIPValue 判定 clientIP 是否命中策略值：策略值支持精确 IP 或 CIDR 网段。
// 解析失败的策略值视为不命中（配置错误不应阻断全部登录），由管理面保证值合法。
func matchIPValue(clientIP, value string) bool {
	value = strings.TrimSpace(value)
	if clientIP == "" || value == "" {
		return false
	}
	if strings.Contains(value, "/") {
		_, ipNet, err := net.ParseCIDR(value)
		if err != nil {
			return false
		}
		ip := net.ParseIP(clientIP)
		return ip != nil && ipNet.Contains(ip)
	}
	return clientIP == value
}

// matchTimeWindow 判定当前时间是否落在 "HH:MM-HH:MM" 时间窗内（支持跨午夜，
// 如 "22:00-06:00"）。格式非法视为不命中。
func matchTimeWindow(now time.Time, value string) bool {
	parts := strings.Split(strings.TrimSpace(value), "-")
	if len(parts) != 2 {
		return false
	}
	start, ok1 := parseHHMM(parts[0])
	end, ok2 := parseHHMM(parts[1])
	if !ok1 || !ok2 || start == end {
		return false
	}
	cur := now.Hour()*60 + now.Minute()
	if start < end {
		return cur >= start && cur < end
	}
	// 跨午夜：如 22:00-06:00 → [start,24:00) ∪ [00:00,end)
	return cur >= start || cur < end
}

func parseHHMM(s string) (int, bool) {
	s = strings.TrimSpace(s)
	hh, mm, found := strings.Cut(s, ":")
	if !found || len(hh) == 0 || len(mm) != 2 {
		return 0, false
	}
	h, m := 0, 0
	for _, c := range hh {
		if c < '0' || c > '9' {
			return 0, false
		}
		h = h*10 + int(c-'0')
	}
	for _, c := range mm {
		if c < '0' || c > '9' {
			return 0, false
		}
		m = m*10 + int(c-'0')
	}
	if h > 23 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}

func (p EffectivePolicy) describe(method string) string {
	if p.Reason != "" {
		return p.Reason
	}
	return "hit " + strings.ToLower(method) + " blacklist: " + p.Value
}

// ValidateLoginPolicyValue 校验策略值格式（管理端 Create/Update 调用）。
// 值格式配错会让匹配器静默不命中——尤其白名单：存在但永不命中 = 全员被锁。
func ValidateLoginPolicyValue(method, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("policy value is empty")
	}
	switch method {
	case "IP":
		if strings.Contains(value, "/") {
			if _, _, err := net.ParseCIDR(value); err != nil {
				return fmt.Errorf("invalid CIDR: %s", value)
			}
			return nil
		}
		if net.ParseIP(value) == nil {
			return fmt.Errorf("invalid IP: %s", value)
		}
		return nil
	case "TIME":
		parts := strings.Split(value, "-")
		if len(parts) != 2 {
			return fmt.Errorf("time window must be HH:MM-HH:MM")
		}
		s1, ok1 := parseHHMM(parts[0])
		s2, ok2 := parseHHMM(parts[1])
		if !ok1 || !ok2 {
			return fmt.Errorf("invalid time format: %s", value)
		}
		if s1 == s2 {
			return fmt.Errorf("time window start equals end: %s", value)
		}
		return nil
	case "DEVICE", "MAC":
		if len(value) > 128 {
			return fmt.Errorf("value too long")
		}
		return nil
	case "REGION":
		if len(value) > 32 {
			return fmt.Errorf("region code too long")
		}
		return nil
	}
	return fmt.Errorf("unknown policy method: %s", method)
}
