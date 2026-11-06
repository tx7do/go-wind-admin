package passwordcrypto

import (
	"encoding/base64"
	"strings"
	"testing"

	gocrypto "github.com/tx7do/go-utils/crypto"
)

// TestEnvelopeRoundTrip 混合信封往返：加密→解密还原明文。
func TestEnvelopeRoundTrip(t *testing.T) {
	for _, pw := range []string{"Abcd@1234", "中文密码+符号!@#", "x"} {
		env, err := EncryptPasswordEnvelope(pw)
		if err != nil {
			t.Fatalf("encrypt: %v", err)
		}
		if !strings.HasPrefix(env, EnvelopePrefix) {
			t.Fatalf("envelope prefix missing: %q", env[:20])
		}
		got, err := DecryptTransferPassword(env)
		if err != nil {
			t.Fatalf("decrypt: %v", err)
		}
		if got != pw {
			t.Fatalf("round-trip mismatch: %q != %q", got, pw)
		}
	}
}

// TestDecryptLegacyCompat 旧前端形态（base64 AES-CBC key=IV）仍被接受。
func TestDecryptLegacyCompat(t *testing.T) {
	legacy, err := gocrypto.AesEncrypt([]byte("LegacyP@ss9"), gocrypto.DefaultAESKey, nil)
	if err != nil {
		t.Fatalf("legacy encrypt: %v", err)
	}
	legacyB64 := base64.StdEncoding.EncodeToString(legacy) // 前端发送的是 base64 字符串
	plain, err := DecryptTransferPassword(legacyB64)
	if err != nil {
		t.Fatalf("legacy decrypt: %v", err)
	}
	if plain != "LegacyP@ss9" {
		t.Fatalf("legacy mismatch: %q", plain)
	}
}

// TestDecryptGarbageRejected 垃圾输入一律报错不恐慌。
func TestDecryptGarbageRejected(t *testing.T) {
	for _, bad := range []string{
		EnvelopePrefix + "!!!:???:**:",
		"not-base64!!",
		"",
	} {
		if _, err := DecryptTransferPassword(bad); err == nil {
			t.Fatalf("garbage should fail: %q", bad)
		}
	}
}
