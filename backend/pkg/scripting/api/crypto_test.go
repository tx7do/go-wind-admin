package api

import (
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	lua "github.com/yuin/gopher-lua"

	"go-wind-admin/pkg/crypto"
)

func TestRegisterCrypto_Module(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	logger := log.NewHelper(log.DefaultLogger)

	// Initialize global encryptor for testing
	err := crypto.InitGlobalEncryptor("test-encryption-key-32-bytes-long!", true)
	assert.NoError(t, err)

	RegisterCrypto(L, logger)

	script := `
		local crypto = require "kratos_crypto"

		-- Verify all functions exist
		assert(crypto.encrypt ~= nil, "encrypt should exist")
		assert(crypto.decrypt ~= nil, "decrypt should exist")
		assert(crypto.is_encrypted ~= nil, "is_encrypted should exist")
		assert(crypto.encrypt_payload ~= nil, "encrypt_payload should exist")
		assert(crypto.decrypt_payload ~= nil, "decrypt_payload should exist")
		assert(crypto.has_encrypted_payload ~= nil, "has_encrypted_payload should exist")
		assert(crypto.encrypt_json ~= nil, "encrypt_json should exist")
		assert(crypto.decrypt_json ~= nil, "decrypt_json should exist")
		assert(crypto.hash_sha256 ~= nil, "hash_sha256 should exist")

		return true
	`

	err = L.DoString(script)
	assert.NoError(t, err)

	t.Log("✓ Crypto module structure test passed")
}

func TestCryptoAPI_EncryptDecrypt(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	logger := log.NewHelper(log.DefaultLogger)

	// Initialize global encryptor
	err := crypto.InitGlobalEncryptor("test-encryption-key-32-bytes-long!", true)
	assert.NoError(t, err)

	RegisterCrypto(L, logger)

	script := `
		local crypto = require "kratos_crypto"

		-- Test basic encryption/decryption
		local plaintext = "Hello, World! This is sensitive data."
		local encrypted = crypto.encrypt(plaintext)
		local decrypted = crypto.decrypt(encrypted)

		assert(encrypted ~= plaintext, "Encrypted should differ from plaintext")
		assert(crypto.is_encrypted(encrypted), "Should be marked as encrypted")
		assert(not crypto.is_encrypted(plaintext), "Plaintext should not be marked as encrypted")
		assert(decrypted == plaintext, "Decrypted should match original plaintext")

		return true
	`

	err = L.DoString(script)
	assert.NoError(t, err)

	t.Log("✓ Encrypt/Decrypt test passed")
}

func TestCryptoAPI_EncryptPayload(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	logger := log.NewHelper(log.DefaultLogger)

	// Initialize global encryptor
	err := crypto.InitGlobalEncryptor("test-encryption-key-32-bytes-long!", true)
	assert.NoError(t, err)

	RegisterCrypto(L, logger)

	script := `
		local crypto = require "kratos_crypto"

		-- Test payload encryption
		local payload = {
			username = "john_doe",
			email = "john@example.com",
			password = "super_secret",
			age = 30,
			active = true
		}

		local encrypted_payload = crypto.encrypt_payload(payload)

		assert(crypto.has_encrypted_payload(encrypted_payload), "Should have encrypted payload")
		assert(encrypted_payload._is_encrypted == true, "Should be marked as encrypted")
		assert(encrypted_payload._encrypted_config ~= nil, "Should have encrypted config")

		-- Decrypt and verify
		local decrypted_payload = crypto.decrypt_payload(encrypted_payload)

		assert(decrypted_payload.username == "john_doe", "Username should match")
		assert(decrypted_payload.email == "john@example.com", "Email should match")
		assert(decrypted_payload.password == "super_secret", "Password should match")
		assert(decrypted_payload.age == 30, "Age should match")
		assert(decrypted_payload.active == true, "Active should match")

		return true
	`

	err = L.DoString(script)
	assert.NoError(t, err)

	t.Log("✓ Encrypt/Decrypt payload test passed")
}

func TestCryptoAPI_EncryptJSON(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	logger := log.NewHelper(log.DefaultLogger)

	// Initialize global encryptor
	err := crypto.InitGlobalEncryptor("test-encryption-key-32-bytes-long!", true)
	assert.NoError(t, err)

	RegisterCrypto(L, logger)

	script := `
		local crypto = require "kratos_crypto"

		-- Test JSON encryption
		local data = {
			items = {"apple", "banana", "orange"},
			count = 3,
			metadata = {
				created_at = 1234567890,
				author = "test_user"
			}
		}

		local encrypted_json = crypto.encrypt_json(data)
		assert(crypto.is_encrypted(encrypted_json), "JSON should be encrypted")

		local decrypted_data = crypto.decrypt_json(encrypted_json)
		assert(decrypted_data.count == 3, "Count should match")
		assert(#decrypted_data.items == 3, "Items length should match")
		assert(decrypted_data.items[1] == "apple", "First item should be apple")
		assert(decrypted_data.metadata.author == "test_user", "Author should match")

		return true
	`

	err = L.DoString(script)
	assert.NoError(t, err)

	t.Log("✓ Encrypt/Decrypt JSON test passed")
}

func TestCryptoAPI_SHA256Hash(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	logger := log.NewHelper(log.DefaultLogger)

	// Note: Crypto encryptor initialization not required for hashing
	RegisterCrypto(L, logger)

	script := `
		local crypto = require "kratos_crypto"

		-- Test SHA-256 hashing
		local data = "Hello, World!"
		local hash1 = crypto.hash_sha256(data)
		local hash2 = crypto.hash_sha256(data)

		-- Same input should produce same hash
		assert(hash1 == hash2, "Hashes should be deterministic")

		-- Different input should produce different hash
		local hash3 = crypto.hash_sha256("Different data")
		assert(hash1 ~= hash3, "Different inputs should produce different hashes")

		-- Hash should be hex string of correct length (64 characters for SHA-256)
		assert(#hash1 == 64, "SHA-256 hash should be 64 hex characters")

		-- Known hash test (optional - for verification)
		local known_hash = crypto.hash_sha256("test")
		-- SHA-256 of "test" is: 9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08
		assert(known_hash == "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08", "Known hash should match")

		return true
	`

	err3 := L.DoString(script)
	assert.NoError(t, err3)

	t.Log("✓ SHA-256 hash test passed")
}

func TestCryptoAPI_EmptyStrings(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	logger := log.NewHelper(log.DefaultLogger)

	// Initialize global encryptor
	err := crypto.InitGlobalEncryptor("test-encryption-key-32-bytes-long!", true)
	assert.NoError(t, err)

	RegisterCrypto(L, logger)

	script := `
		local crypto = require "kratos_crypto"

		-- Test empty string handling
		local encrypted_empty = crypto.encrypt("")
		local decrypted_empty = crypto.decrypt(encrypted_empty)

		assert(decrypted_empty == "", "Empty string should encrypt and decrypt correctly")

		return true
	`

	err2 := L.DoString(script)
	assert.NoError(t, err2)

	t.Log("✓ Empty string test passed")
}
