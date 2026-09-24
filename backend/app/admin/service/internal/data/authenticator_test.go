package data

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"

	conf "github.com/tx7do/kratos-bootstrap/api/gen/go/conf/v1"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"

	"go-wind-admin/pkg/jwt"
)

const testRSAPrivateKey = `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCL8EdeTLDTf8AY
UODMsMTARMHyuOEDtBB0bcLMzfXNQ8QnuGMULTlgV4VACyVtkL9QaglPZHbZB6gC
hi09uhIqWpI/kj39xktK4oncbYvmuCnoIIJdycqHYYSre/kp4fjMbeny5W4+53uF
h8fX5KhgKFyJSdpvdJV9+licTUwG91SAR+twT+wdZOh3k4HU8dorPW0ztGbpIggD
nJhvQeV46mtdfq/cyrTgrS/8HMHWnYlyk2vz5HF47f1LwoRlTaYTpgL1JPmTQq4w
AvY0ZAkNkXy+YARrRfKe+SXeg8eZyjPnt7+l9rzKKVP5gxd2aooGMTO4P2bWzjXD
/5btGTTzAgMBAAECggEAGKxFuQ+mgbPdh6wC5rQoDIpS89u6+K8v04diuD98HjPb
ivFMrssGecEUomUUtUu3H5OCjrf06HEcI03K/j4nY8ZSUNkVCwCCV/K3QeEisIw5
/050Dds9VT9RZ/bUyJiqCEk83XGsTXT85184Ug1jzohvQFmAJPSWQv73zp8mT3fb
dA0mi+aq+FAJtJ8n01zzQKs79OGAeI3eK2XMXbnWbxfs0xFGnrQPlRwbkz+737ug
txZVBxWXfG/nfXymon2f98Uy/MEJko0UDC/0dk8jYB4QMqDecJSwnTi1OQgh0OdE
4oVVD/QXOmboY97yqQQDvnAlaCiAklotGmjnlCrVuQKBgQC/MHKSk7PeoBq6A5MR
36n9EuM+tNS/X19bwClAmXpLteFgQA1nTcVmV41xzPPENnw2BfIjnqO+KoyHZKry
BgONIYH43QTC7dL+c+rM/LayqyV7S5ZFZBZRevq9tjwHm6kbdg6QlobRKqqNt3C6
K3JziXT8c+JHLuHU6rWsL7PNjQKBgQC7YEJKrnTGX2FmPypShmDRj5RHtJEm3HUM
V+wjDOszNVMWQlQGEn67RKZoTcsjyb+GMANcG644vpvczbi5UUJ3S3zi0Iry1tPK
jvb7EB4GOOqA0uUEc5v8HFIv2Gdpp3rC15GTcBh+O7qH5ZyfF/9y106THVQt1pv9
fkEIeFosfwKBgCRsEVeVJc4CiDTpm2nrRxH8OChpAKKYg60R9Ynl8yNbOd1BNox4
h2OQyFRmrAW0L4OHLHLWtPD0YCMm7V4AAUswl/cV++M6tVheMtvsRM3SxugvJSiB
AbNyDzR29AarA9NEcU/gLTzJuQYYbTQ6NKqIBC5X0UKoTsNmF0f/Kmy9AoGAKgUf
OLJA28+9/vkBW7po8fX58c6rkoRz902sVfvqrvQxatd7ElWJeCOgEdoISUFQIx6X
Ukue2XjdaTn1SBHSDwCtxAuybV0B5/YBqzHlGc4fwL4Kv+HRREtxnusv3cDCRfmj
2uWTiJOKdDlo00DFd5KTO2ijXRg4qTNsECM1Ta8CgYEAlXtI6aPyXdVUWLYpz9pG
asY+ctT+zoEtmzOAb+wbKulFKZoKPDPPrRV/y+GJuBeH6/rNFOhmxjgl03CDal1P
3YPKuKmgJex1eI+Dd7WeWWvUWxKJKZ4CfDt4LKrj7gOnsIJkJxboqNhLQnqQ2ccD
/8fOCXg4RaAm5Jpcuk8qEo0=
-----END PRIVATE KEY-----`

const testRSAPublicKey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAi/BHXkyw03/AGFDgzLDE
wETB8rjhA7QQdG3CzM31zUPEJ7hjFC05YFeFQAslbZC/UGoJT2R22QeoAoYtPboS
KlqSP5I9/cZLSuKJ3G2L5rgp6CCCXcnKh2GEq3v5KeH4zG3p8uVuPud7hYfH1+So
YChciUnab3SVffpYnE1MBvdUgEfrcE/sHWTod5OB1PHaKz1tM7Rm6SIIA5yYb0Hl
eOprXX6v3Mq04K0v/BzB1p2JcpNr8+RxeO39S8KEZU2mE6YC9ST5k0KuMAL2NGQJ
DZF8vmAEa0Xynvkl3oPHmcoz57e/pfa8yilT+YMXdmqKBjEzuD9m1s41w/+W7Rk0
8wIDAQAB
-----END PUBLIC KEY-----`

func ptrString(s string) *string { return &s }

func decodeJWTHeader(t *testing.T, headerSeg string) string {
	t.Helper()
	body, err := base64.RawURLEncoding.DecodeString(headerSeg)
	if err != nil {
		t.Fatalf("decode header segment: %v", err)
	}
	return string(body)
}

func TestNewAdminAuthenticator_RS256(t *testing.T) {
	jwtCfg := &conf.Authentication_Jwt{
		Method:     "RS256",
		PrivateKey: ptrString(testRSAPrivateKey),
		PublicKey:  ptrString(testRSAPublicKey),
	}

	auth, err := newAdminAuthenticator(jwtCfg)
	if err != nil {
		t.Fatalf("newAdminAuthenticator(RS256) failed: %v", err)
	}

	exp := time.Now().Add(15 * time.Minute)
	payload := &authenticationV1.UserTokenPayload{Username: ptrString("admin"), UserId: 1}
	token, err := auth.CreateIdentity(*jwt.NewUserTokenAuthClaims(payload, &exp))
	if err != nil {
		t.Fatalf("CreateIdentity failed: %v", err)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3-part JWT, got %d", len(parts))
	}
	if header := decodeJWTHeader(t, parts[0]); !strings.Contains(header, `"alg":"RS256"`) {
		t.Fatalf("header does not declare RS256: %s", header)
	}

	if _, err := auth.AuthenticateToken(token); err != nil {
		t.Fatalf("AuthenticateToken failed: %v", err)
	}
	t.Logf("RS256 (private_key+public_key) round-trip OK")
}

// TestNewAdminAuthenticator_RS256PrivateKeyOnly 验证仅配置私钥时，
// 公钥从私钥派生仍可正常签发与校验。
func TestNewAdminAuthenticator_RS256PrivateKeyOnly(t *testing.T) {
	jwtCfg := &conf.Authentication_Jwt{
		Method:     "RS256",
		PrivateKey: ptrString(testRSAPrivateKey),
	}

	auth, err := newAdminAuthenticator(jwtCfg)
	if err != nil {
		t.Fatalf("newAdminAuthenticator(private only) failed: %v", err)
	}

	exp := time.Now().Add(15 * time.Minute)
	payload := &authenticationV1.UserTokenPayload{Username: ptrString("admin"), UserId: 1}
	token, err := auth.CreateIdentity(*jwt.NewUserTokenAuthClaims(payload, &exp))
	if err != nil {
		t.Fatalf("CreateIdentity failed: %v", err)
	}
	if _, err := auth.AuthenticateToken(token); err != nil {
		t.Fatalf("AuthenticateToken failed: %v", err)
	}
	t.Log("RS256 (private_key only, public derived) round-trip OK")
}

// TestNewAdminAuthenticator_HS256NoRegression 验证对称算法仍可正常工作。
func TestNewAdminAuthenticator_HS256NoRegression(t *testing.T) {
	jwtCfg := &conf.Authentication_Jwt{
		Method: "HS256",
		Key:    "some_api_key",
	}

	auth, err := newAdminAuthenticator(jwtCfg)
	if err != nil {
		t.Fatalf("newAdminAuthenticator(HS256) failed: %v", err)
	}
	exp := time.Now().Add(15 * time.Minute)
	payload := &authenticationV1.UserTokenPayload{Username: ptrString("admin"), UserId: 1}
	token, err := auth.CreateIdentity(*jwt.NewUserTokenAuthClaims(payload, &exp))
	if err != nil {
		t.Fatalf("CreateIdentity failed: %v", err)
	}
	if header := decodeJWTHeader(t, strings.Split(token, ".")[0]); !strings.Contains(header, `"alg":"HS256"`) {
		t.Fatalf("header does not declare HS256: %s", header)
	}
	if _, err := auth.AuthenticateToken(token); err != nil {
		t.Fatalf("HS256 AuthenticateToken failed: %v", err)
	}
	t.Log("HS256 round-trip OK (no regression)")
}

// TestNewAdminAuthenticator_AsymmetricMissingKey 验证非对称算法缺少密钥时报错。
func TestNewAdminAuthenticator_AsymmetricMissingKey(t *testing.T) {
	jwtCfg := &conf.Authentication_Jwt{Method: "RS256"}
	if _, err := newAdminAuthenticator(jwtCfg); err == nil {
		t.Fatal("expected error for RS256 without keys, got nil")
	}
}

// TestGetExpires_UsesConfig 验证过期时间优先读取配置。
func TestGetExpires_UsesConfig(t *testing.T) {
	a := &Authenticator{
		jwtCfg: &conf.Authentication_Jwt{
			AccessTokenExpires:  durationpb.New(30 * time.Second),
			RefreshTokenExpires: durationpb.New(48 * time.Hour),
		},
	}
	if got := a.GetAccessTokenExpires(authenticationV1.ClientType_admin); got != 30*time.Second {
		t.Fatalf("access expires = %v, want 30s", got)
	}
	if got := a.GetRefreshTokenExpires(authenticationV1.ClientType_admin); got != 48*time.Hour {
		t.Fatalf("refresh expires = %v, want 48h", got)
	}
}

// TestGetExpires_FallbackDefault 验证未配置时回退到默认值。
func TestGetExpires_FallbackDefault(t *testing.T) {
	a := &Authenticator{jwtCfg: &conf.Authentication_Jwt{}}
	if got := a.GetAccessTokenExpires(authenticationV1.ClientType_admin); got != DefaultAccessTokenExpires {
		t.Fatalf("access expires = %v, want default %v", got, DefaultAccessTokenExpires)
	}
	if got := a.GetRefreshTokenExpires(authenticationV1.ClientType_admin); got != DefaultRefreshTokenExpires {
		t.Fatalf("refresh expires = %v, want default %v", got, DefaultRefreshTokenExpires)
	}
}
