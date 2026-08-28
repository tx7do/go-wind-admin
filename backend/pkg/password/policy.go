// Package password 提供等保要求的口令策略：复杂度、有效期、历史口令。
// 阈值均可用环境变量覆盖（部署期调优，避免改 proto 的连锁改动）：
//   - PASSWORD_MIN_LEN        最小长度，默认 8
//   - PASSWORD_MAX_AGE_DAYS   有效期天数，默认 90；<=0 表示不启用有效期
//   - PASSWORD_HISTORY_COUNT  历史口令保留条数，默认 3；<=0 表示不启用历史检查
package password

import (
	"errors"
	"os"
	"strconv"
	"unicode"
)

// ErrWeakPassword 复杂度不达标（调用方转成 4xx 响应）。
var ErrWeakPassword = errors.New("password does not meet complexity requirements: min length 8 and at least 3 of (lowercase, uppercase, digit, symbol)")

// MinLen 返回口令最小长度。
func MinLen() int {
	if v := os.Getenv("PASSWORD_MIN_LEN"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 8
}

// MaxAgeDays 返回口令有效期（天）；<=0 表示不启用。
func MaxAgeDays() int {
	if v := os.Getenv("PASSWORD_MAX_AGE_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return 90
}

// HistoryCount 返回历史口令保留条数；<=0 表示不启用历史检查。
func HistoryCount() int {
	if v := os.Getenv("PASSWORD_HISTORY_COUNT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return 3
}

// ValidateComplexity 校验明文口令复杂度：长度达标且至少包含
// 小写/大写/数字/符号 四类中的三类。
func ValidateComplexity(plain string) error {
	minLen := MinLen()
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
