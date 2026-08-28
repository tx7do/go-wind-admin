package password

import (
	"errors"
	"testing"
)

func TestValidateComplexity(t *testing.T) {
	cases := []struct {
		pw   string
		want error
	}{
		{"Abc12345", nil},                  // 大写+小写+数字
		{"abcd1234", nil},                  // 小写+数字... 只两类，应拒绝
		{"Abc12345", nil},                  // 重复占位
		{"Aa1!aaaa", nil},                  // 三类含符号
		{"12345678", nil},                  // 仅数字一类，拒绝
		{"Ab1", errors.New("too short")},   // 过短
		{"abcdefgh", nil},                  // 仅小写，拒绝
		{"Abcdefg1", nil},                  // 大小写+数字
		{"Aaaaaaaaa1", nil},                // 大小写+数字
		{"aaaaaaaa1!", nil},                // 小写+数字+符号
	}
	for _, c := range cases {
		// 修正期望：按四类计数重新判断
		got := ValidateComplexity(c.pw)
		classes := countClasses(c.pw)
		wantOK := len(c.pw) >= MinLen() && classes >= 3
		if wantOK && got != nil {
			t.Errorf("ValidateComplexity(%q) = %v, want nil (classes=%d)", c.pw, got, classes)
		}
		if !wantOK && got == nil {
			t.Errorf("ValidateComplexity(%q) = nil, want error (classes=%d)", c.pw, classes)
		}
	}
}

func countClasses(s string) int {
	var lower, upper, digit, symbol bool
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			lower = true
		case r >= 'A' && r <= 'Z':
			upper = true
		case r >= '0' && r <= '9':
			digit = true
		default:
			symbol = true
		}
	}
	n := 0
	for _, b := range []bool{lower, upper, digit, symbol} {
		if b {
			n++
		}
	}
	return n
}
