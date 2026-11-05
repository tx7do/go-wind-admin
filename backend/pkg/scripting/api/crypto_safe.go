package api

import (
	"crypto/sha256"
	"encoding/hex"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	lua "github.com/yuin/gopher-lua"

	"github.com/tx7do/go-utils/crypto"
)

// 脚本 crypto 模块的受限版本（替代库版 host.LoaderCrypto / hostmodule.ModuleCrypto）。
//
// 库版把全局主密钥 GOWIND_CRYPTO_KEY 的 encrypt/decrypt 原语直接暴露给脚本——
// 而同一把钥匙加密着 AI provider API key、SMTP/webhook 密码、MFA TOTP secret、
// 通知渠道 settings（见 ai_provider_service / notification_channel_repo /
// user_mfa_factor_repo / channel/settings_crypto）。脚本一旦能拿到任何密文
// （cache 模块、未来的 payload 通道）即可用 crypto.decrypt 解出平台全部机密。
//
// 故这里只保留无密钥依赖的原语：散列与"是否密文"检测；加密/解密一律不暴露。
// 脚本侧如需对称加密，应由平台提供专用受限密钥后再议。

// LoaderCryptoSafe 返回受限 crypto 模块（kratos_crypto）的 Lua loader。
func LoaderCryptoSafe(_ *bLogger.Helper) lua.LGFunction {
	return func(L *lua.LState) int {
		cryptoModule := L.NewTable()

		// crypto.hash_sha256(data) → hex
		cryptoModule.RawSetString("hash_sha256", L.NewFunction(func(L *lua.LState) int {
			data := L.CheckString(1)
			hash := sha256.Sum256([]byte(data))
			L.Push(lua.LString(hex.EncodeToString(hash[:])))
			return 1
		}))

		// crypto.is_encrypted(data) → bool（"enc:" 前缀检测，无密钥依赖）
		cryptoModule.RawSetString("is_encrypted", L.NewFunction(func(L *lua.LState) int {
			data := L.CheckString(1)
			L.Push(lua.LBool(crypto.IsEncrypted(data)))
			return 1
		}))

		L.Push(cryptoModule)
		return 1
	}
}

// ModuleCryptoSafe 构建受限 crypto 模块（JS 等基于 map[string]any 桥接的语言使用）。
func ModuleCryptoSafe() map[string]any {
	return map[string]any{
		"hash_sha256": func(data string) string {
			hash := sha256.Sum256([]byte(data))
			return hex.EncodeToString(hash[:])
		},
		"is_encrypted": func(data string) bool {
			return crypto.IsEncrypted(data)
		},
	}
}
