package crypto

import (
	"fmt"
	"testing"
)

func TestEncryptDecryptPayload(t *testing.T) {
	// Initialize encryptor
	InitGlobalEncryptor("test-key-for-payload-encryption", true)

	tests := []struct {
		name    string
		payload map[string]interface{}
	}{
		{
			name: "email configuration",
			payload: map[string]interface{}{
				"host":     "imap.gmail.com",
				"port":     993,
				"username": "user@example.com",
				"password": "super-secret-password",
				"tls":      true,
			},
		},
		{
			name: "api credentials",
			payload: map[string]interface{}{
				"api_url":    "https://api.example.com",
				"api_key":    "sk-1234567890abcdef",
				"api_secret": "very-secret-key",
			},
		},
		{
			name: "mixed types",
			payload: map[string]interface{}{
				"string":  "value",
				"number":  42,
				"float":   3.14,
				"boolean": true,
				"nested": map[string]interface{}{
					"key": "value",
				},
			},
		},
		{
			name: "with task metadata",
			payload: map[string]interface{}{
				"task_id":   123,
				"task_type": "email_processor",
				"username":  "admin",
				"password":  "secret",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt payload
			encrypted, err := EncryptPayload(tt.payload)
			if err != nil {
				t.Fatalf("EncryptPayload() error = %v", err)
			}

			// Verify encrypted structure
			if !HasEncryptedPayload(encrypted) {
				t.Error("Encrypted payload should have encrypted marker")
			}

			if _, ok := encrypted[EncryptedConfigKey]; !ok {
				t.Error("Encrypted payload should have encrypted config key")
			}

			// Verify task metadata is preserved
			if taskID, ok := tt.payload["task_id"]; ok {
				if encrypted["task_id"] != taskID {
					t.Errorf("task_id should be preserved, got %v, want %v", encrypted["task_id"], taskID)
				}
			}

			// Decrypt payload
			decrypted, err := DecryptPayload(encrypted)
			if err != nil {
				t.Fatalf("DecryptPayload() error = %v", err)
			}

			// Verify all fields match
			for key, expectedValue := range tt.payload {
				actualValue, ok := decrypted[key]
				if !ok {
					t.Errorf("Decrypted payload missing key %q", key)
					continue
				}

				// Compare values (handle type conversions for numbers)
				if !compareValues(expectedValue, actualValue) {
					t.Errorf("Decrypted value for %q = %v (%T), want %v (%T)",
						key, actualValue, actualValue, expectedValue, expectedValue)
				}
			}
		})
	}
}

func TestDecryptPayload_Unencrypted(t *testing.T) {
	// Test backward compatibility with unencrypted payloads
	payload := map[string]interface{}{
		"host":     "imap.gmail.com",
		"username": "user",
		"password": "pass",
	}

	// Should return payload as-is
	decrypted, err := DecryptPayload(payload)
	if err != nil {
		t.Fatalf("DecryptPayload() should handle unencrypted payload, error = %v", err)
	}

	if len(decrypted) != len(payload) {
		t.Errorf("Unencrypted payload should be returned as-is")
	}

	for key, value := range payload {
		if decrypted[key] != value {
			t.Errorf("Value for %q = %v, want %v", key, decrypted[key], value)
		}
	}
}

func TestHasEncryptedPayload(t *testing.T) {
	tests := []struct {
		name    string
		payload map[string]interface{}
		want    bool
	}{
		{
			name: "encrypted payload",
			payload: map[string]interface{}{
				IsEncryptedKey:     true,
				EncryptedConfigKey: "enc:data",
			},
			want: true,
		},
		{
			name: "unencrypted payload",
			payload: map[string]interface{}{
				"host":     "imap.gmail.com",
				"username": "user",
			},
			want: false,
		},
		{
			name:    "empty payload",
			payload: map[string]interface{}{},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasEncryptedPayload(tt.payload); got != tt.want {
				t.Errorf("HasEncryptedPayload() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEncryptPayload_PreservesMetadata(t *testing.T) {
	InitGlobalEncryptor("test-key", true)

	payload := map[string]interface{}{
		"task_id":   uint32(123),
		"task_type": "email_processor",
		"username":  "admin",
		"password":  "secret123",
		"other":     "data",
	}

	encrypted, err := EncryptPayload(payload)
	if err != nil {
		t.Fatalf("EncryptPayload() error = %v", err)
	}

	// task_id should be preserved for routing
	if encrypted["task_id"] != uint32(123) {
		t.Errorf("task_id not preserved, got %v", encrypted["task_id"])
	}

	// task_type should be preserved for routing
	if encrypted["task_type"] != "email_processor" {
		t.Errorf("task_type not preserved, got %v", encrypted["task_type"])
	}

	// Sensitive fields should NOT be in plain form
	if _, exists := encrypted["username"]; exists {
		t.Error("username should be encrypted, not in plain form")
	}
	if _, exists := encrypted["password"]; exists {
		t.Error("password should be encrypted, not in plain form")
	}
}

// compareValues compares two values, handling JSON number conversion
func compareValues(expected, actual interface{}) bool {
	// Handle nil
	if expected == nil && actual == nil {
		return true
	}
	if expected == nil || actual == nil {
		return false
	}

	// Handle numeric conversions (JSON unmarshaling may change int to float64)
	switch e := expected.(type) {
	case int:
		if f, ok := actual.(float64); ok {
			return float64(e) == f
		}
	case float64:
		if i, ok := actual.(int); ok {
			return e == float64(i)
		}
	case uint32:
		if f, ok := actual.(float64); ok {
			return float64(e) == f
		}
	case map[string]interface{}:
		// Handle nested maps
		a, ok := actual.(map[string]interface{})
		if !ok {
			return false
		}
		if len(e) != len(a) {
			return false
		}
		for key, val := range e {
			if !compareValues(val, a[key]) {
				return false
			}
		}
		return true
	}

	// Direct comparison for other types (string, bool, etc.)
	return fmt.Sprintf("%v", expected) == fmt.Sprintf("%v", actual)
}

func BenchmarkEncryptPayload(b *testing.B) {
	InitGlobalEncryptor("benchmark-key", true)

	payload := map[string]interface{}{
		"host":     "imap.gmail.com",
		"port":     993,
		"username": "user@example.com",
		"password": "super-secret-password",
		"tls":      true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = EncryptPayload(payload)
	}
}

func BenchmarkDecryptPayload(b *testing.B) {
	InitGlobalEncryptor("benchmark-key", true)

	payload := map[string]interface{}{
		"host":     "imap.gmail.com",
		"port":     993,
		"username": "user@example.com",
		"password": "super-secret-password",
	}

	encrypted, _ := EncryptPayload(payload)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecryptPayload(encrypted)
	}
}
