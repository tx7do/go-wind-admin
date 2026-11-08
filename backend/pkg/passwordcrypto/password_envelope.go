// Package passwordcrypto 承载登录/改密/建用户口令传输的加解密。
//
// P1-12 B 路线（混合信封）：前端以随机 AES-256-GCM 会话密钥加密口令，
// 再用 RSA-2048 OAEP(SHA-256) 包裹会话密钥，信封格式：
//
//	"rsa-v1:<base64(wrappedKey)>:<base64(nonce)>:<base64(cipher)>"
//
// 信封编解码已下沉 go-utils/crypto（EnvelopeCodec 注入制，格式由裸原语
// 测试锁定）；本包只保留应用侧的部署契约：私钥装载（env 覆盖/开发默认
// 密钥对）、旧形态（base64(AES-128-CBC, key=IV DefaultAESKey)）过渡兼容、
// 统一解密入口 DecryptTransferPassword 的分发。
// 公钥静态内置前端（公开值不降低安全性，无分发/替换通道），私钥仅服务端
// 持有：env GOWIND_PASSWORD_RSA_PRIVATE_KEY（PEM，支持 \n 转义单行）覆盖，
// 未配置时使用内置开发密钥对（与旧 DefaultAESKey 同一公开语义，生产必须
// 覆盖）。旧形态仍被接受——前端分批切换的过渡兼容；全部前端切换完成后
// 可移除。
package passwordcrypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"os"
	"strings"
	"sync"

	gocrypto "github.com/tx7do/go-utils/crypto"
)

const (
	// EnvelopePrefix 信封格式版本前缀（库侧格式契约的本地别名）。
	EnvelopePrefix = gocrypto.EnvelopePrefix
	// EnvPrivateKey 覆盖开发默认私钥的环境变量名。
	EnvPrivateKey = "GOWIND_PASSWORD_RSA_PRIVATE_KEY"
)

// 开发默认 RSA-2048 密钥对（公开入仓，生产必须以 env 覆盖私钥）。
const (
	devPrivatePEM = `-----BEGIN PRIVATE KEY-----
MIIEuwIBADANBgkqhkiG9w0BAQEFAASCBKUwggShAgEAAoIBAQCiQ/+NZXAkXN3F
SVvq+eg07sxK1nKzLjTrsaEZ6Lbl8mLIGQkJmXAfouVy0FvIaWrC9yDmSD4t+xDH
127h51wyq29IGJ6omOmOsgE2qDcMZ21Z0O3hI+1DWWSeLl3HAHVMfjBotvxOdWOk
na/lwGTKZR2IJ/cZGrtTk8hatg79vvH4fSf5rsnIpD8TO8ptkLjXtiA9/ETJwLNY
gTmSPgRFYw3cVfIhxowS3XA84X4GgLedqfDswNbQrPiJVchjkUvzIipz1C0Gd9DW
DHB15QkjsAeR5fOEexJSfWOIwMHl6fjasGoMp+D7x8jEpyjHyz+wrKhg45mzzz7D
InLLSQgLAgMBAAECgf90rAImKNv5SV8HsqVShfCC+lV8VWxIIJXIB0vhHmCtGojs
V10V5U/wStTUpteg5wbKcebaiPK+09lz16oZey8GrYaxq5Ro7q5uRyGYPX/pois1
DfKPJjOqAopon+4W8ZjTeZ8whz1Q44TUM7IPJTTBLc6yvFLdFZJSvLj3W8vZXTkH
mtXiAx6MEphT1uRMubo+tvOH7FJheH+c9L8NHVxbt5k5rQ7oGlxurEOw35+T5Cka
ALDuNnz+A+Zm7354tsJJnwsSxOoEDVXhb8gzXEE+NDw3Tzdj9TgL6RgBwCIgRTsO
dxnV8sf3yG/waydx296A7V8LXScEh3ZZrcxKr9UCgYEA44wgOBwPfN+jAxQWaU2v
P1ae5E0ZLv8PlS/QFp9ZCwcF81Kxn16jgQIs7BZW0JiYomxu8cMxJnKxmw7TcbIb
EzJBsxKI+ltgtGwfbWkDRStOmya15mLmffG+/OLgclUMJklH3Tl3PgVtA0YpK2pR
M3RUq+U8ynaSi9FbCS44lf8CgYEAto4uCYM8Wrw7vypZ0vlF3qx7yqZCKPBuh3FG
C2UhZUOeoQZtydsNaX0fNKtEZ3ZUh3GTGhO9mQjxFZJhgIufOgrz39BrGoeSzD2K
JAoAbJsOzXxpKqe+2ITsGd+cXaUqnPdbarrh5Va5kx4iRK7RhwNOuRIGGei+dxxM
hV7MhfUCgYBVIYzAUUc+GdE8Cpzj4suI79hcY7fjTEvBLfU1MYK3zHXedfa8QET9
tBmobicmvaCFS/op+zP9RXXLJD3ZUKAui2PnIi0bNWq1taoAXgR6MVykUVRGkIPZ
ZMNmi18bh1QXUt1PF/vTuWXyjqJCevFMRd6TvR0ERl5HbxkXEzLacwKBgGsNtbGe
ZCZRfDf+fe5af761J2MPXHzlgeurYYgNJihcHwoO2xMkghj6ZvJb/SEvIyLKV5lT
OjwkIjykOslHF3jSM3Fsp2JHXq6m5/t6YgVOaL/X1VROSnxudoY1u1yvHu4toE5W
iWIUGX9MjIv6eKTUK7Yb+7f15Lu0vKr5jLaBAoGBAL1tmIMq94ggsqcJpI4/LcSs
F85pT2iSzOmoLYzg2x4QNH7tNuqn8yjsX/4voNqXkrT1g7t/Tpm64Kps+13s77gG
T8jcwfRVNn0lhIFj7g2mLFagec8G7Q5mlzumDTzXRj+Sw3SPOVip6a9p7wn24TG9
imwTPhL0crfiqbxJJAu6
-----END PRIVATE KEY-----`
	devPublicPEM = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAokP/jWVwJFzdxUlb6vno
NO7MStZysy4067GhGei25fJiyBkJCZlwH6LlctBbyGlqwvcg5kg+LfsQx9du4edc
MqtvSBieqJjpjrIBNqg3DGdtWdDt4SPtQ1lkni5dxwB1TH4waLb8TnVjpJ2v5cBk
ymUdiCf3GRq7U5PIWrYO/b7x+H0n+a7JyKQ/EzvKbZC417YgPfxEycCzWIE5kj4E
RWMN3FXyIcaMEt1wPOF+BoC3nanw7MDW0Kz4iVXIY5FL8yIqc9QtBnfQ1gxwdeUJ
I7AHkeXzhHsSUn1jiMDB5en42rBqDKfg+8fIxKcox8s/sKyoYOOZs88+wyJyy0kI
CwIDAQAB
-----END PUBLIC KEY-----`
)

// DevPublicPEM 供部署脚本/运维导出公钥（与前端 VITE_PASSWORD_RSA_PUB 同值）。
func DevPublicPEM() string { return devPublicPEM }

var (
	mu       sync.RWMutex
	privateK *rsa.PrivateKey
)

// init 解析 env 覆盖或回落开发默认对。
func init() {
	loadPrivateKey(os.Getenv(EnvPrivateKey))
}

func loadPrivateKey(pemStr string) {
	pemStr = strings.TrimSpace(pemStr)
	if pemStr == "" {
		pemStr = devPrivatePEM
	} else {
		// env 单行形态支持 \n 转义换行
		pemStr = strings.ReplaceAll(pemStr, "\\n", "\n")
	}
	key, err := gocrypto.ParsePrivateKeyPEM(pemStr)
	if err != nil {
		return
	}
	mu.Lock()
	privateK = key
	mu.Unlock()
}

// ReloadFromEnv 重新装载私钥（env 热变更/测试用）。
func ReloadFromEnv() { loadPrivateKey(os.Getenv(EnvPrivateKey)) }

// UsingDevPrivateKey 是否仍在使用内置开发私钥（部署侧据此打启动告警：
// 生产必须以 env 覆盖，否则任何持有本仓源码者可解密口令信封）。
func UsingDevPrivateKey() bool {
	key := privateKey()
	if key == nil {
		return false
	}
	dev, err := gocrypto.ParsePrivateKeyPEM(devPrivatePEM)
	if err != nil {
		return false
	}
	return key.Equal(dev)
}

func privateKey() *rsa.PrivateKey {
	mu.RLock()
	defer mu.RUnlock()
	return privateK
}

// EncryptPasswordEnvelope 混合信封加密（服务端参考实现/单测/运维工具用；
// 前端在 WebCrypto 里独立实现同格式：RSA-OAEP(SHA-256) 包裹 AES-256-GCM 密钥）。
// 编解码本体在 go-utils/crypto，此处只做私钥注入。
func EncryptPasswordEnvelope(plain string) (string, error) {
	key := privateKey()
	if key == nil {
		return "", errors.New("passwordcrypto: private key not loaded")
	}
	return gocrypto.NewEnvelopeCodecFromKey(key).Encrypt(plain)
}

// DecryptTransferPassword 统一口令传输解密入口：
//   - "rsa-v1:" 前缀 → 混合信封解密（go-utils/crypto EnvelopeCodec）；
//   - 其余 → 旧 base64+AES-128-CBC(DefaultAESKey, key=IV) 过渡兼容。
//
// 返回明文口令；格式/解密失败返回错误（调用方以 400 语义透传）。
func DecryptTransferPassword(v string) (string, error) {
	if strings.HasPrefix(v, EnvelopePrefix) {
		key := privateKey()
		if key == nil {
			return "", errors.New("passwordcrypto: private key not loaded")
		}
		return gocrypto.NewEnvelopeCodecFromKey(key).Decrypt(v)
	}
	return decryptLegacy(v)
}

// decryptLegacy 旧前端形态：base64(AES-128-CBC, key=IV=DefaultAESKey)。
func decryptLegacy(v string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		return "", errors.New("passwordcrypto: invalid legacy base64")
	}
	block, err := aes.NewCipher(gocrypto.DefaultAESKey)
	if err != nil {
		return "", err
	}
	if len(raw) == 0 || len(raw)%block.BlockSize() != 0 {
		return "", errors.New("passwordcrypto: legacy length mismatch")
	}
	plain := make([]byte, len(raw))
	cipher.NewCBCDecrypter(block, gocrypto.DefaultAESKey).CryptBlocks(plain, raw)
	return string(unpadPKCS7(plain, block.BlockSize())), nil
}

// unpadPKCS7 剥除 PKCS7 填充（非法填充按 0 长处理，交上层以 400 报格式错）。
func unpadPKCS7(data []byte, blockSize int) []byte {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return data
	}
	pad := int(data[len(data)-1])
	if pad <= 0 || pad > blockSize || pad > len(data) {
		return data
	}
	for _, b := range data[len(data)-pad:] {
		if int(b) != pad {
			return data
		}
	}
	return data[:len(data)-pad]
}
