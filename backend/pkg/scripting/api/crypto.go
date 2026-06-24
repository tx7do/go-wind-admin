package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/go-kratos/kratos/v2/log"
	lua "github.com/yuin/gopher-lua"

	"go-wind-admin/pkg/crypto"
	"go-wind-admin/pkg/scripting/internal/convert"
)

// RegisterCrypto registers the Crypto API for Lua as a requireable module
func RegisterCrypto(L *lua.LState, logger *log.Helper) {
	// Register in package.preload so it can be required
	L.PreloadModule("kratos_crypto", LoaderCrypto(logger))
}

// buildCryptoModule 构建并压入 crypto 模块到栈顶（供 RegisterCrypto 与 LoaderCrypto 复用）。
func buildCryptoModule(L *lua.LState, logger *log.Helper) {
	// Create crypto module
	cryptoModule := L.NewTable()

	// crypto.encrypt(plaintext)
	// Encrypts a string using AES-256-GCM
	// Returns: encrypted_text (with "enc:" prefix)
	cryptoModule.RawSetString("encrypt", L.NewFunction(func(L *lua.LState) int {
		plaintext := L.CheckString(1)

		encrypted, err := crypto.EncryptIfNeeded(plaintext)
		if err != nil {
			L.RaiseError("encryption failed: %v", err)
			return 0
		}

		L.Push(lua.LString(encrypted))
		return 1
	}))

	// crypto.decrypt(ciphertext)
	// Decrypts a string encrypted with crypto.encrypt
	// Returns: plaintext
	cryptoModule.RawSetString("decrypt", L.NewFunction(func(L *lua.LState) int {
		ciphertext := L.CheckString(1)

		decrypted, err := crypto.DecryptIfNeeded(ciphertext)
		if err != nil {
			L.RaiseError("decryption failed: %v", err)
			return 0
		}

		L.Push(lua.LString(decrypted))
		return 1
	}))

	// crypto.is_encrypted(data)
	// Checks if a string is encrypted (has "enc:" prefix)
	// Returns: boolean
	cryptoModule.RawSetString("is_encrypted", L.NewFunction(func(L *lua.LState) int {
		data := L.CheckString(1)
		isEncrypted := crypto.IsEncrypted(data)
		L.Push(lua.LBool(isEncrypted))
		return 1
	}))

	// crypto.encrypt_payload(table)
	// Encrypts a Lua table as JSON
	// Returns: table with encrypted data and metadata
	cryptoModule.RawSetString("encrypt_payload", L.NewFunction(func(L *lua.LState) int {
		payloadTable := L.CheckTable(1)

		// ConvertCode Lua table to Go map
		payloadMap := convert.ToGoValue(payloadTable).(map[string]interface{})

		// Encrypt the payload
		encrypted, err := crypto.EncryptPayload(payloadMap)
		if err != nil {
			L.RaiseError("payload encryption failed: %v", err)
			return 0
		}

		// ConvertCode back to Lua table
		result := convert.ToLuaValue(L, encrypted)
		L.Push(result)
		return 1
	}))

	// crypto.decrypt_payload(encrypted_table)
	// Decrypts a payload encrypted with crypto.encrypt_payload
	// Returns: table with decrypted data
	cryptoModule.RawSetString("decrypt_payload", L.NewFunction(func(L *lua.LState) int {
		encryptedTable := L.CheckTable(1)

		// ConvertCode Lua table to Go map
		encryptedMap := convert.ToGoValue(encryptedTable).(map[string]interface{})

		// Decrypt the payload
		decrypted, err := crypto.DecryptPayload(encryptedMap)
		if err != nil {
			L.RaiseError("payload decryption failed: %v", err)
			return 0
		}

		// ConvertCode back to Lua table
		result := convert.ToLuaValue(L, decrypted)
		L.Push(result)
		return 1
	}))

	// crypto.has_encrypted_payload(table)
	// Checks if a table contains encrypted payload
	// Returns: boolean
	cryptoModule.RawSetString("has_encrypted_payload", L.NewFunction(func(L *lua.LState) int {
		payloadTable := L.CheckTable(1)

		// ConvertCode Lua table to Go map
		payloadMap := convert.ToGoValue(payloadTable).(map[string]interface{})

		// Check if encrypted
		hasEncrypted := crypto.HasEncryptedPayload(payloadMap)
		L.Push(lua.LBool(hasEncrypted))
		return 1
	}))

	// crypto.encrypt_json(lua_table)
	// Encrypts a Lua table by converting to JSON first
	// Returns: encrypted string
	cryptoModule.RawSetString("encrypt_json", L.NewFunction(func(L *lua.LState) int {
		dataTable := L.CheckTable(1)

		// ConvertCode to Go value
		goValue := convert.ToGoValue(dataTable)

		// Marshal to JSON
		jsonData, err := json.Marshal(goValue)
		if err != nil {
			L.RaiseError("JSON marshal failed: %v", err)
			return 0
		}

		// Encrypt the JSON string
		encrypted, err := crypto.EncryptIfNeeded(string(jsonData))
		if err != nil {
			L.RaiseError("encryption failed: %v", err)
			return 0
		}

		L.Push(lua.LString(encrypted))
		return 1
	}))

	// crypto.decrypt_json(encrypted_string)
	// Decrypts an encrypted string and parses as JSON
	// Returns: lua table
	cryptoModule.RawSetString("decrypt_json", L.NewFunction(func(L *lua.LState) int {
		encrypted := L.CheckString(1)

		// Decrypt the string
		decrypted, err := crypto.DecryptIfNeeded(encrypted)
		if err != nil {
			L.RaiseError("decryption failed: %v", err)
			return 0
		}

		// Parse JSON
		var result interface{}
		if err := json.Unmarshal([]byte(decrypted), &result); err != nil {
			L.RaiseError("JSON unmarshal failed: %v", err)
			return 0
		}

		// ConvertCode to Lua value
		luaValue := convert.ToLuaValue(L, result)
		L.Push(luaValue)
		return 1
	}))

	// crypto.hash_sha256(data)
	// Returns SHA-256 hash of the input string (hex encoded)
	cryptoModule.RawSetString("hash_sha256", L.NewFunction(func(L *lua.LState) int {
		data := L.CheckString(1)

		// Calculate SHA-256 hash
		hash := sha256.Sum256([]byte(data))

		// ConvertCode to hex string
		hexHash := hex.EncodeToString(hash[:])

		L.Push(lua.LString(hexHash))
		return 1
	}))

	L.Push(cryptoModule)
}

// LoaderCrypto 返回 crypto 模块（kratos_crypto）的 loader，供 go-scripts 引擎 RegisterModule 使用。
func LoaderCrypto(logger *log.Helper) lua.LGFunction {
	return func(L *lua.LState) int {
		buildCryptoModule(L, logger)
		return 1
	}
}
