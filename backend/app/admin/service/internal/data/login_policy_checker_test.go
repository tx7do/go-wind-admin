package data

import (
	"testing"
	"time"
)

func TestMatchLoginPolicyIpBlacklist(t *testing.T) {
	policies := []EffectivePolicy{
		{TargetID: 0, Value: "10.0.0.5", Type: "BLACKLIST", Method: "IP"},
		{TargetID: 0, Value: "172.16.0.0/16", Type: "BLACKLIST", Method: "IP"},
	}
	if blocked, _ := MatchLoginPolicy(policies, 0, "10.0.0.5", "", time.Now()); !blocked {
		t.Fatalf("exact IP blacklist should block")
	}
	if blocked, _ := MatchLoginPolicy(policies, 0, "172.16.99.1", "", time.Now()); !blocked {
		t.Fatalf("CIDR blacklist should block")
	}
	if blocked, _ := MatchLoginPolicy(policies, 0, "192.168.1.1", "", time.Now()); blocked {
		t.Fatalf("unlisted IP should pass")
	}
	// 非法策略值不命中（配置错误不阻断全部登录）
	bad := []EffectivePolicy{{TargetID: 0, Value: "not-an-ip", Type: "BLACKLIST", Method: "IP"}}
	if blocked, _ := MatchLoginPolicy(bad, 0, "1.2.3.4", "", time.Now()); blocked {
		t.Fatalf("malformed policy value should not block")
	}
}

func TestMatchLoginPolicyIpWhitelist(t *testing.T) {
	policies := []EffectivePolicy{
		{TargetID: 0, Value: "10.0.0.0/8", Type: "WHITELIST", Method: "IP"},
	}
	if blocked, _ := MatchLoginPolicy(policies, 0, "10.1.2.3", "", time.Now()); blocked {
		t.Fatalf("IP in whitelist should pass")
	}
	if blocked, _ := MatchLoginPolicy(policies, 0, "192.168.1.1", "", time.Now()); !blocked {
		t.Fatalf("IP outside whitelist should block")
	}
}

func TestMatchLoginPolicyTargetScope(t *testing.T) {
	policies := []EffectivePolicy{
		{TargetID: 42, Value: "10.0.0.5", Type: "BLACKLIST", Method: "IP"},
	}
	// userId=0（密码校验前的全局段）：定向条目不生效
	if blocked, _ := MatchLoginPolicy(policies, 0, "10.0.0.5", "", time.Now()); blocked {
		t.Fatalf("user-targeted policy should not apply to global check")
	}
	// 命中目标用户：生效
	if blocked, _ := MatchLoginPolicy(policies, 42, "10.0.0.5", "", time.Now()); !blocked {
		t.Fatalf("user-targeted policy should block target user")
	}
	// 非目标用户：不生效
	if blocked, _ := MatchLoginPolicy(policies, 43, "10.0.0.5", "", time.Now()); blocked {
		t.Fatalf("user-targeted policy should not apply to other users")
	}
}

func TestMatchLoginPolicyTimeWindow(t *testing.T) {
	policies := []EffectivePolicy{
		{TargetID: 0, Value: "22:00-06:00", Type: "BLACKLIST", Method: "TIME"},
	}
	at := func(h, m int) time.Time { return time.Date(2026, 1, 1, h, m, 0, 0, time.Local) }
	if blocked, _ := MatchLoginPolicy(policies, 0, "", "", at(23, 30)); !blocked {
		t.Fatalf("23:30 should be inside overnight blacklist window")
	}
	if blocked, _ := MatchLoginPolicy(policies, 0, "", "", at(3, 0)); !blocked {
		t.Fatalf("03:00 should be inside overnight blacklist window")
	}
	if blocked, _ := MatchLoginPolicy(policies, 0, "", "", at(12, 0)); blocked {
		t.Fatalf("noon should be outside overnight blacklist window")
	}
	// 跨午夜白名单：仅工作时间允许
	white := []EffectivePolicy{
		{TargetID: 0, Value: "09:00-18:00", Type: "WHITELIST", Method: "TIME"},
	}
	if blocked, _ := MatchLoginPolicy(white, 0, "", "", at(10, 0)); blocked {
		t.Fatalf("10:00 should be inside work-hours whitelist")
	}
	if blocked, _ := MatchLoginPolicy(white, 0, "", "", at(20, 0)); !blocked {
		t.Fatalf("20:00 should be outside work-hours whitelist")
	}
}

func TestMatchLoginPolicyDevice(t *testing.T) {
	policies := []EffectivePolicy{
		{TargetID: 0, Value: "device-abc", Type: "BLACKLIST", Method: "DEVICE"},
	}
	if blocked, _ := MatchLoginPolicy(policies, 0, "", "device-abc", time.Now()); !blocked {
		t.Fatalf("blacklisted device should block")
	}
	if blocked, _ := MatchLoginPolicy(policies, 0, "", "device-xyz", time.Now()); blocked {
		t.Fatalf("other device should pass")
	}
	// 空 deviceId 不匹配任何值
	if blocked, _ := MatchLoginPolicy(policies, 0, "", "", time.Now()); blocked {
		t.Fatalf("empty device id should not match")
	}
}

func TestParseHHMM(t *testing.T) {
	cases := []struct {
		in   string
		want int
		ok   bool
	}{
		{"09:00", 540, true},
		{"23:59", 1439, true},
		{"00:00", 0, true},
		{"24:00", 0, false},
		{"9:00", 540, true}, // 一位小时宽松接受（管理员手输场景）
		{"09:0", 0, false},  // 非两位分钟
		{"ab:cd", 0, false},
		{"0900", 0, false},
	}
	for _, c := range cases {
		got, ok := parseHHMM(c.in)
		if ok != c.ok || (ok && got != c.want) {
			t.Fatalf("parseHHMM(%q) = (%d,%v), want (%d,%v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestIsDigits(t *testing.T) {
	if !isDigits("13800138000") {
		t.Fatalf("pure digits should be true")
	}
	if isDigits("abc") || isDigits("138a") || isDigits("") || isDigits("138 00") {
		t.Fatalf("non-digit strings should be false")
	}
}
