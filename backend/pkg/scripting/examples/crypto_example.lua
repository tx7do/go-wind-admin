-- Example: Using the Crypto API for encryption/decryption
-- This script demonstrates various crypto operations

local log = require "kratos_logger"
local crypto = require "kratos_crypto"
local hook = require "kratos_hook"

log.info("========================================")
log.info("  Crypto API Example")
log.info("========================================")

-- Example 1: Basic String Encryption/Decryption
local function example_basic_encryption()
    log.info("\n--- Example 1: Basic Encryption ---")

    local plaintext = "This is sensitive information!"
    log.info("Original: " .. plaintext)

    -- Encrypt the string
    local encrypted = crypto.encrypt(plaintext)
    log.info("Encrypted: " .. encrypted)

    -- Check if encrypted
    if crypto.is_encrypted(encrypted) then
        log.info("✓ Data is encrypted (has enc: prefix)")
    end

    -- Decrypt back
    local decrypted = crypto.decrypt(encrypted)
    log.info("Decrypted: " .. decrypted)

    assert(plaintext == decrypted, "Decryption should match original")
    log.info("✓ Basic encryption test passed")
end

-- Example 2: Encrypting User Credentials
local function example_credentials()
    log.info("\n--- Example 2: Encrypting Credentials ---")

    -- Encrypt sensitive credentials
    local password = "my_super_secret_password"
    local api_key = "sk-1234567890abcdef"

    local encrypted_password = crypto.encrypt(password)
    local encrypted_api_key = crypto.encrypt(api_key)

    log.info("✓ Password encrypted: " .. encrypted_password:sub(1, 20) .. "...")
    log.info("✓ API key encrypted: " .. encrypted_api_key:sub(1, 20) .. "...")

    -- Later, decrypt when needed
    local decrypted_password = crypto.decrypt(encrypted_password)
    log.info("✓ Password decrypted successfully")

    return encrypted_password, encrypted_api_key
end

-- Example 3: Payload Encryption (Complex Data)
local function example_payload_encryption()
    log.info("\n--- Example 3: Payload Encryption ---")

    -- Complex payload with multiple fields
    local user_data = {
        username = "john_doe",
        email = "john@example.com",
        password = "secret123",
        phone = "+1234567890",
        ssn = "123-45-6789",
        age = 30,
        is_active = true
    }

    log.info("Original payload:")
    log.info("  Username: " .. user_data.username)
    log.info("  Email: " .. user_data.email)

    -- Encrypt entire payload
    local encrypted_payload = crypto.encrypt_payload(user_data)

    log.info("✓ Payload encrypted")
    log.info("  Is encrypted: " .. tostring(encrypted_payload._is_encrypted))

    -- Decrypt payload
    local decrypted_payload = crypto.decrypt_payload(encrypted_payload)

    log.info("✓ Payload decrypted")
    log.info("  Username: " .. decrypted_payload.username)
    log.info("  Email: " .. decrypted_payload.email)

    assert(decrypted_payload.username == user_data.username, "Username should match")
    assert(decrypted_payload.password == user_data.password, "Password should match")

    log.info("✓ Payload encryption test passed")
end

-- Example 4: JSON Encryption
local function example_json_encryption()
    log.info("\n--- Example 4: JSON Encryption ---")

    local config = {
        database = {
            host = "localhost",
            port = 5432,
            username = "admin",
            password = "db_password"
        },
        api_keys = {
            stripe = "sk_test_123",
            sendgrid = "SG.123456"
        },
        settings = {
            debug = true,
            max_connections = 100
        }
    }

    -- Encrypt as JSON string
    local encrypted_json = crypto.encrypt_json(config)
    log.info("✓ Config encrypted as JSON")
    log.info("  Length: " .. #encrypted_json .. " bytes")

    -- Decrypt back to table
    local decrypted_config = crypto.decrypt_json(encrypted_json)
    log.info("✓ Config decrypted from JSON")
    log.info("  DB Host: " .. decrypted_config.database.host)
    log.info("  DB Password: [decrypted successfully]")

    assert(decrypted_config.database.password == "db_password", "Password should match")
    log.info("✓ JSON encryption test passed")
end

-- Example 5: SHA-256 Hashing
local function example_hashing()
    log.info("\n--- Example 5: SHA-256 Hashing ---")

    local data1 = "user@example.com"
    local data2 = "another@example.com"

    local hash1 = crypto.hash_sha256(data1)
    local hash2 = crypto.hash_sha256(data2)
    local hash1_again = crypto.hash_sha256(data1)

    log.info("Hash of '" .. data1 .. "':")
    log.info("  " .. hash1)

    log.info("Hash of '" .. data2 .. "':")
    log.info("  " .. hash2)

    -- Verify properties
    assert(hash1 == hash1_again, "Hashes should be deterministic")
    assert(hash1 ~= hash2, "Different inputs should produce different hashes")
    assert(#hash1 == 64, "SHA-256 produces 64 hex characters")

    log.info("✓ Hash properties verified")
end

-- Example 6: Hook Integration - Encrypt Sensitive Data
hook.register("user.created", "Encrypt user sensitive data", function(ctx)
    log.info("\n--- Hook: Encrypting User Data ---")

    local user_id = ctx.get("user_id")
    local email = ctx.get("email")
    local password = ctx.get("password")
    local ssn = ctx.get("ssn")

    log.info("Encrypting sensitive data for user: " .. user_id)

    -- Encrypt sensitive fields
    local encrypted_password = crypto.encrypt(password)
    local encrypted_ssn = crypto.encrypt(ssn)

    -- Store encrypted versions
    ctx.set("encrypted_password", encrypted_password)
    ctx.set("encrypted_ssn", encrypted_ssn)

    -- Create audit log hash
    local audit_hash = crypto.hash_sha256(email .. user_id)
    ctx.set("audit_hash", audit_hash)

    log.info("✓ Sensitive data encrypted")
    log.info("  Password: [encrypted]")
    log.info("  SSN: [encrypted]")
    log.info("  Audit hash: " .. audit_hash:sub(1, 16) .. "...")

    return true
end)

-- Example 7: Hook Integration - Store Encrypted Configuration
hook.register("config.save", "Save encrypted configuration", function(ctx)
    log.info("\n--- Hook: Saving Encrypted Config ---")

    local config = ctx.get("config")

    -- Encrypt entire configuration
    local encrypted_config = crypto.encrypt_json(config)

    -- Store encrypted version
    ctx.set("encrypted_config", encrypted_config)

    log.info("✓ Configuration encrypted and ready to save")
    log.info("  Size: " .. #encrypted_config .. " bytes")

    return true
end)

-- Example 8: Hook Integration - Load and Decrypt Configuration
hook.register("config.load", "Load and decrypt configuration", function(ctx)
    log.info("\n--- Hook: Loading Encrypted Config ---")

    local encrypted_config = ctx.get("encrypted_config")

    -- Decrypt configuration
    local config = crypto.decrypt_json(encrypted_config)

    -- Make available to application
    ctx.set("config", config)

    log.info("✓ Configuration decrypted and loaded")

    return true
end)

-- Example 9: Safe Payload Handling
local function example_safe_payload()
    log.info("\n--- Example 9: Safe Payload Handling ---")

    -- Create payload with sensitive data
    local task_payload = {
        task_id = "task-123",
        task_type = "email_send",
        -- Sensitive data
        email_credentials = {
            username = "smtp@example.com",
            password = "smtp_password",
            api_key = "key_123456"
        },
        recipient = "user@example.com"
    }

    -- Encrypt payload for storage (e.g., Redis, database)
    local encrypted = crypto.encrypt_payload(task_payload)

    -- Check if it's encrypted
    if crypto.has_encrypted_payload(encrypted) then
        log.info("✓ Payload is safely encrypted")
    end

    -- Note: task_id and task_type are preserved for routing
    log.info("  Task ID (preserved): " .. tostring(encrypted.task_id))
    log.info("  Task Type (preserved): " .. tostring(encrypted.task_type))
    log.info("  Sensitive data: [encrypted]")

    -- Later, when processing the task
    local decrypted = crypto.decrypt_payload(encrypted)
    log.info("✓ Payload decrypted for processing")
    log.info("  Email credentials available for sending")

    return encrypted
end

-- Run examples (uncomment to test)
-- example_basic_encryption()
-- example_credentials()
-- example_payload_encryption()
-- example_json_encryption()
-- example_hashing()
-- example_safe_payload()

log.info("\n========================================")
log.info("  Crypto hooks registered successfully")
log.info("========================================")
