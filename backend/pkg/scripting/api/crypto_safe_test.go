// 本文件针对 crypto_safe.go（受限 kratos_crypto 模块）做单元测试。
//
// 该模块是安全收敛的产物：库版把全局主密钥的 encrypt/decrypt 原语暴露给脚本，
// 而同一把钥匙加密着平台全部机密（AI provider key、SMTP/webhook 密码、MFA secret、
// 通知渠道 settings）——脚本拿到任何密文即可解出平台全部机密。受限版只保留
// 无密钥依赖的 hash_sha256 与 is_encrypted 两个原语。
//
// 测试钉三件事：
//  1. Lua 与 JS 两条桥接路线的函数行为一致（同一组已知向量）；
//  2. 模块表里【不存在】encrypt/decrypt——收敛回归一旦被还原，这里立刻红；
//  3. hash 与 is_encrypted 的边界输入（空串、前缀串）。
package api

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	lua "github.com/yuin/gopher-lua"
)

// sha256Hex 测试内独立实现已知向量（避免与被测代码共用同一行逻辑而失去互证意义）。
func sha256Hex(data string) string {
	sum := sha256.Sum256([]byte(data))
	return hex.EncodeToString(sum[:])
}

// newCryptoLuaState 构造注册了受限 crypto 模块的 LState（kratos_crypto 经 require 触发 loader）。
func newCryptoLuaState(t *testing.T) *lua.LState {
	t.Helper()
	L := lua.NewState()
	t.Cleanup(L.Close)
	L.PreloadModule("kratos_crypto", LoaderCryptoSafe(bLogger.NewHelper(bLogger.NopLogger())))
	return L
}

// TestCryptoSafe_LuaHashSha256 Lua 路线：hash_sha256 已知向量 + 边界输入 + 参数校验。
func TestCryptoSafe_LuaHashSha256(t *testing.T) {
	L := newCryptoLuaState(t)

	err := L.DoString(`
		local crypto = require "kratos_crypto"

		assert(crypto.hash_sha256("abc") == "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
			"sha256(abc) should match the known vector")
		assert(crypto.hash_sha256("") == "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			"sha256 of empty string should match the known vector")
		assert(crypto.hash_sha256("中文输入") == "` + sha256Hex("中文输入") + `",
			"sha256 should handle utf-8 input")
	`)
	require.NoError(t, err)
}

// TestCryptoSafe_LuaIsEncrypted Lua 路线：is_encrypted 三态（空串/明文/"enc:" 密文）。
func TestCryptoSafe_LuaIsEncrypted(t *testing.T) {
	L := newCryptoLuaState(t)

	err := L.DoString(`
		local crypto = require "kratos_crypto"

		assert(crypto.is_encrypted("") == false, "empty string is not encrypted")
		assert(crypto.is_encrypted("plain-text") == false, "plain text is not encrypted")
		assert(crypto.is_encrypted("enc:ZXhhbXBsZQ==") == true, "enc: prefixed value is encrypted")
	`)
	require.NoError(t, err)
}

// TestCryptoSafe_LuaNoEncryptDecrypt 模块表不得暴露 encrypt/decrypt 原语——
// 受限版存在的意义就在于此，还原即回归。
func TestCryptoSafe_LuaNoEncryptDecrypt(t *testing.T) {
	L := newCryptoLuaState(t)

	err := L.DoString(`
		local crypto = require "kratos_crypto"
		assert(crypto.encrypt == nil, "encrypt must not be exposed to scripts")
		assert(crypto.decrypt == nil, "decrypt must not be exposed to scripts")
	`)
	require.NoError(t, err)
}

// TestCryptoSafe_LuaMissingArg 参数缺失时 CheckString 抛 Lua 错误。
func TestCryptoSafe_LuaMissingArg(t *testing.T) {
	L := newCryptoLuaState(t)

	err := L.DoString(`
		local crypto = require "kratos_crypto"
		local ok, perr = pcall(crypto.hash_sha256)
		assert(ok == false, "hash_sha256 without args should raise")
		local ok2, perr2 = pcall(crypto.is_encrypted)
		assert(ok2 == false, "is_encrypted without args should raise")
	`)
	require.NoError(t, err)
}

// TestCryptoSafe_JSModule JS 路线：与 Lua 同一组已知向量，函数行为一致。
func TestCryptoSafe_JSModule(t *testing.T) {
	mod := ModuleCryptoSafe()

	hashFn, ok := mod["hash_sha256"].(func(string) string)
	require.True(t, ok, "hash_sha256 should be func(string) string")
	require.Equal(t, "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad", hashFn("abc"))
	require.Equal(t, sha256Hex("中文输入"), hashFn("中文输入"))

	isEncFn, ok := mod["is_encrypted"].(func(string) bool)
	require.True(t, ok, "is_encrypted should be func(string) bool")
	require.False(t, isEncFn(""))
	require.False(t, isEncFn("plain-text"))
	require.True(t, isEncFn("enc:ZXhhbXBsZQ=="))

	// JS 路线同样不得暴露 encrypt/decrypt。
	require.NotContains(t, mod, "encrypt")
	require.NotContains(t, mod, "decrypt")
}

// --- 基准 ---

// BenchmarkCryptoSafe_JSHashSha256 纯 Go hash+hex 路径，按输入尺寸分档（64B / 1KB / 64KB）。
func BenchmarkCryptoSafe_JSHashSha256(b *testing.B) {
	mod := ModuleCryptoSafe()
	hashFn := mod["hash_sha256"].(func(string) string)
	for _, size := range []int{64, 1 << 10, 64 << 10} {
		data := make([]byte, size)
		for i := range data {
			data[i] = byte(i % 251)
		}
		s := string(data)
		b.Run(sizeLabel(size), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = hashFn(s)
			}
		})
	}
}

// BenchmarkCryptoSafe_LuaHashSha256 经 Lua 桥接的完整路径（栈调用 + CheckString + Push）。
// 栈操作严格配对：循环外取一次函数值，循环内 Push(fn)+Push(arg) → PCall → Pop(result)。
func BenchmarkCryptoSafe_LuaHashSha256(b *testing.B) {
	L := lua.NewState()
	defer L.Close()
	L.PreloadModule("kratos_crypto", LoaderCryptoSafe(bLogger.NewHelper(bLogger.NopLogger())))
	require.NoError(b, L.DoString(`
		local crypto = require "kratos_crypto"
		function bench_hash(data) return crypto.hash_sha256(data) end
	`))
	fn := L.GetGlobal("bench_hash") // 不压栈，仅取值（循环内 Push 配对）
	data := make([]byte, 1<<10)
	for i := range data {
		data[i] = byte(i % 251)
	}
	arg := lua.LString(data)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		L.Push(fn)
		L.Push(arg)
		if err := L.PCall(1, 1, nil); err != nil {
			b.Fatal(err)
		}
		L.Pop(1) // 弹出结果
	}
}

func sizeLabel(size int) string {
	switch size {
	case 64:
		return "64B"
	case 1 << 10:
		return "1KB"
	default:
		return "64KB"
	}
}
