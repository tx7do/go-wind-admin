// Package password 提供等保要求的口令策略：复杂度、有效期、历史口令。
// 阈值不再读环境变量，统一从 sys_config 平台参数（参数管理页可改）读取：
//
//		r.configRepo.GetConfigInt(ctx, passwordPolicy.ConfigKeyMinLen, passwordPolicy.DefaultMinLen)
//
//	  - sys.password.minLen       最小长度，默认 DefaultMinLen
//	  - sys.password.maxAgeDays   有效期天数，默认 DefaultMaxAgeDays；<=0 表示不启用有效期
//	  - sys.password.historyCount 历史口令保留条数，默认 DefaultHistoryCount；<=0 表示不启用历史检查
//
// 内置键随服务启动播种进 sys_configs（is_built_in，可改不可删），
// 修改经参数管理页落库后即时生效（ConfigRepo 缓存写路径同步失效）。
package password

import (
	"errors"
	"strconv"
	"unicode"
)

// sys_config 平台参数键（与 DefaultConfigs 播种清单保持一致）。
const (
	ConfigKeyMinLen       = "sys.password.minLen"
	ConfigKeyMaxAgeDays   = "sys.password.maxAgeDays"
	ConfigKeyHistoryCount = "sys.password.historyCount"
)

// 阈值默认值：键不存在/值不可解析时 GetConfigInt 的回退值。
// 与播种进 sys_configs 的初始值一致，仅作为缺行兜底。
const (
	DefaultMinLen       = 8
	DefaultMaxAgeDays   = 90
	DefaultHistoryCount = 3
)

// ErrWeakPassword 复杂度不达标（调用方转成 4xx 响应）。
var ErrWeakPassword = errors.New("password does not meet complexity requirements: min length 8 and at least 3 of (lowercase, uppercase, digit, symbol)")

// ValidateComplexity 校验明文口令复杂度：长度达到 minLen 且至少包含
// 小写/大写/数字/符号 四类中的三类。minLen 由调用方从 sys_config 读取。
func ValidateComplexity(plain string, minLen int) error {
	if len(plain) < minLen {
		return errors.New("password too short: minimum length is " + strconv.Itoa(minLen))
	}
	var hasLower, hasUpper, hasDigit, hasSymbol bool
	for _, r := range plain {
		switch {
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSymbol = true
		}
	}
	classes := 0
	for _, b := range []bool{hasLower, hasUpper, hasDigit, hasSymbol} {
		if b {
			classes++
		}
	}
	if classes < 3 {
		return ErrWeakPassword
	}
	return nil
}
