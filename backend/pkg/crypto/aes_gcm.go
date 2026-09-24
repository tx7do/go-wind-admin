package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	// EncryptedPrefix is added to encrypted data to distinguish from plaintext
	EncryptedPrefix = "enc:"
)

var (
	ErrInvalidKey        = errors.New("encryption key must be at least 32 bytes")
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
	ErrDecryptionFailed  = errors.New("decryption failed")
)

// Encryptor provides AES-256-GCM encryption/decryption functionality
type Encryptor struct {
	key []byte
}

// NewEncryptor creates a new encryptor with the given key
// The key will be hashed to ensure it's exactly 32 bytes for AES-256
func NewEncryptor(key string) (*Encryptor, error) {
	if key == "" {
		return nil, ErrInvalidKey
	}

	// Use SHA-256 to derive a 32-byte key from any length input
	hash := sha256.Sum256([]byte(key))

	return &Encryptor{
		key: hash[:],
	}, nil
}

// Encrypt encrypts the plaintext using AES-256-GCM
// Returns base64-encoded ciphertext with "enc:" prefix
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	// Create cipher block
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the data
	// Format: nonce + ciphertext (GCM automatically includes authentication tag)
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Encode to base64 and add prefix
	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	return EncryptedPrefix + encoded, nil
}

// Decrypt decrypts the ciphertext using AES-256-GCM
// Expects base64-encoded ciphertext with "enc:" prefix
func (e *Encryptor) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	// Check if data is encrypted (has prefix)
	if !strings.HasPrefix(ciphertext, EncryptedPrefix) {
		// Not encrypted, return as-is for backward compatibility
		return ciphertext, nil
	}

	// Remove prefix
	encoded := strings.TrimPrefix(ciphertext, EncryptedPrefix)

	// Decode from base64
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("%w: invalid base64 encoding", ErrInvalidCiphertext)
	}

	// Create cipher block
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Check minimum size (nonce + at least some data)
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("%w: ciphertext too short", ErrInvalidCiphertext)
	}

	// Extract nonce and ciphertext
	nonce, cipherData := data[:nonceSize], data[nonceSize:]

	// Decrypt the data
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	return string(plaintext), nil
}

// IsEncrypted checks if the given string is encrypted (has the prefix)
func IsEncrypted(data string) bool {
	return strings.HasPrefix(data, EncryptedPrefix)
}

// MustEncrypt encrypts data and panics on error (useful for testing)
func (e *Encryptor) MustEncrypt(plaintext string) string {
	encrypted, err := e.Encrypt(plaintext)
	if err != nil {
		panic(err)
	}
	return encrypted
}

// MustDecrypt decrypts data and panics on error (useful for testing)
func (e *Encryptor) MustDecrypt(ciphertext string) string {
	decrypted, err := e.Decrypt(ciphertext)
	if err != nil {
		panic(err)
	}
	return decrypted
}
