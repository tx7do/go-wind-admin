package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/pquerna/otp"
	otpTotp "github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent/usermfafactor"
)

// TestTotpGenerateAndValidate 验证 StartEnrollMethod/VerifyMFAChallenge 共用的
// TOTP 生成-校验闭环：同参数生成的 key，其当前有效码必须能被 ValidateCustom
// （±1 窗口、6 位、SHA1）验证通过；错误码必须失败。
func TestTotpGenerateAndValidate(t *testing.T) {
	key, err := otpTotp.Generate(otpTotp.GenerateOpts{
		Issuer:      mfaTotpIssuer,
		AccountName: "uid:1",
	})
	if err != nil {
		t.Fatalf("generate totp key failed: %v", err)
	}

	code, err := otpTotp.GenerateCode(key.Secret(), time.Now())
	if err != nil {
		t.Fatalf("generate code failed: %v", err)
	}

	ok, verr := otpTotp.ValidateCustom(code, key.Secret(), time.Now(), otpTotp.ValidateOpts{
		Period:    30,
		Skew:      mfaTotpSkew,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if verr != nil || !ok {
		t.Fatalf("valid code rejected: ok=%v err=%v", ok, verr)
	}

	ok, verr = otpTotp.ValidateCustom("000000", key.Secret(), time.Now(), otpTotp.ValidateOpts{
		Period:    30,
		Skew:      mfaTotpSkew,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	// 000000 恰好是当期有效码的概率是 1/10^6，测试撞上视为通过
	if verr == nil && ok && code != "000000" {
		t.Fatalf("invalid code accepted")
	}
}

// TestTotpSkewWindow 验证 ±1 时间窗口：上一周期的码（30s 前）应被接受（时钟漂移容忍），
// 两个周期前（60s 前）必须被拒绝。
func TestTotpSkewWindow(t *testing.T) {
	key, err := otpTotp.Generate(otpTotp.GenerateOpts{
		Issuer:      mfaTotpIssuer,
		AccountName: "uid:1",
	})
	if err != nil {
		t.Fatalf("generate totp key failed: %v", err)
	}

	opts := otpTotp.ValidateOpts{
		Period:    30,
		Skew:      mfaTotpSkew,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	}

	prevCode, err := otpTotp.GenerateCode(key.Secret(), time.Now().Add(-30*time.Second))
	if err != nil {
		t.Fatalf("generate prev code failed: %v", err)
	}
	if ok, _ := otpTotp.ValidateCustom(prevCode, key.Secret(), time.Now(), opts); !ok {
		t.Fatalf("prev-period code should be accepted within skew=1")
	}

	oldCode, err := otpTotp.GenerateCode(key.Secret(), time.Now().Add(-60*time.Second))
	if err != nil {
		t.Fatalf("generate old code failed: %v", err)
	}
	if ok, _ := otpTotp.ValidateCustom(oldCode, key.Secret(), time.Now(), opts); ok {
		t.Fatalf("two-period-old code should be rejected")
	}
}

// TestParseFactorId 验证 credential_id 字符串到 uint32 主键的解析。
func TestParseFactorId(t *testing.T) {
	cases := []struct {
		in      string
		want    uint32
		wantErr bool
	}{
		{"1", 1, false},
		{"4294967295", 4294967295, false},
		{"", 0, true},
		{"abc", 0, true},
		{"-1", 0, true},
		{"4294967296", 0, true}, // 超 uint32
	}
	for _, c := range cases {
		got, err := parseFactorId(c.in)
		if c.wantErr {
			if err == nil {
				t.Fatalf("parseFactorId(%q) expected error, got %d", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Fatalf("parseFactorId(%q) unexpected error: %v", c.in, err)
		}
		if got != c.want {
			t.Fatalf("parseFactorId(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

// TestTotpQrDataUri 验证 QR data URI 生成的形状：data:image/png;base64 前缀 + 非空负载。
func TestTotpQrDataUri(t *testing.T) {
	key, err := otpTotp.Generate(otpTotp.GenerateOpts{
		Issuer:      mfaTotpIssuer,
		AccountName: "uid:1",
	})
	if err != nil {
		t.Fatalf("generate totp key failed: %v", err)
	}

	uri, err := totpQrDataUri(key)
	if err != nil {
		t.Fatalf("totpQrDataUri failed: %v", err)
	}
	const prefix = "data:image/png;base64,"
	if len(uri) <= len(prefix) || uri[:len(prefix)] != prefix {
		t.Fatalf("qr data uri missing png prefix: %s...", fmt.Sprintf("%.40s", uri))
	}

	if _, err := totpQrDataUri(nil); err == nil {
		t.Fatalf("totpQrDataUri(nil) should fail")
	}
}

// TestMfaMethodToEntity 验证 methodToEntity 的映射：TOTP 映射成功，
// 其余（含未指定）一律报错。
func TestMfaMethodToEntity(t *testing.T) {
	m, err := methodToEntity(authenticationV1.MFAMethod_TOTP)
	require.NoError(t, err)
	require.Equal(t, usermfafactor.MethodTotp, m, "TOTP 应映射为实体 TOTP 方法")

	for _, bad := range []authenticationV1.MFAMethod{
		authenticationV1.MFAMethod_MFA_METHOD_UNSPECIFIED,
		authenticationV1.MFAMethod(99),
	} {
		_, err := methodToEntity(bad)
		require.Error(t, err, "非 TOTP 方法 %v 应报错", bad)
	}
}

// TestMfaIsASCII 验证 isASCII：全 ASCII 非空为真；空串、含非 ASCII 字节为假。
func TestMfaIsASCII(t *testing.T) {
	require.True(t, isASCII("plain-ascii-123"))
	require.True(t, isASCII("a\tb"))
	require.False(t, isASCII(""))
	require.False(t, isASCII("caf\xc3\xa9"))
	require.False(t, isASCII("\x80abc"))
}

// TestMfaBuildEnrolledProto 验证 buildEnrolledProto 的字段搬运：
// 元信息逐字段映射为 EnrolledMethod 列表（含时间戳转换），不含 secret。
func TestMfaBuildEnrolledProto(t *testing.T) {
	created := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	infos := []data.EnrolledFactorInfo{
		{ID: 11, Method: authenticationV1.MFAMethod_TOTP, DisplayName: "Authenticator A", Enabled: true, CreatedAt: &created, LastUsedAt: nil},
		{ID: 12, Method: authenticationV1.MFAMethod_MFA_METHOD_UNSPECIFIED, DisplayName: "", Enabled: false, CreatedAt: nil, LastUsedAt: nil},
	}
	out := buildEnrolledProto(infos)
	require.Len(t, out, 2, "元信息应逐条搬运")
	require.Equal(t, "11", out[0].GetId())
	require.Equal(t, authenticationV1.MFAMethod_TOTP, out[0].GetMethod())
	require.Equal(t, "Authenticator A", out[0].GetDisplay())
	require.True(t, out[0].GetEnabled())
	require.Equal(t, timestamppb.New(created), out[0].GetCreatedAt(), "创建时间应转换为时间戳")
	require.Nil(t, out[0].GetLastUsedAt(), "未使用时间应保持 nil")
	require.Equal(t, "12", out[1].GetId())
	require.False(t, out[1].GetEnabled())
	require.Nil(t, out[1].GetCreatedAt())
}
