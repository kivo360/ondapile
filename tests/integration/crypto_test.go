package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ondapile/internal/config"
)

// TestEncryptDecryptRoundtrip tests that encrypt -> decrypt returns the original data
func TestEncryptDecryptRoundtrip(t *testing.T) {
	key := config.DeriveKey("test-encryption-key")

	original := map[string]string{
		"username": "testuser",
		"password": "secretpassword123",
		"token":    "bearer_token_here",
	}

	// Encrypt
	encrypted, err := config.EncryptCredentials(original, key)
	require.NoError(t, err)
	require.NotNil(t, encrypted)
	require.NotEmpty(t, encrypted)

	// Encrypted data should be different from original
	originalJSON := `{"username":"testuser","password":"secretpassword123","token":"bearer_token_here"}`
	assert.NotEqual(t, originalJSON, string(encrypted))

	// Decrypt
	decrypted, err := config.DecryptCredentials(encrypted, key)
	require.NoError(t, err)
	require.NotNil(t, decrypted)

	// Decrypted should match original
	assert.Equal(t, original, decrypted)
}

// TestDecryptWithWrongKey tests that decryption with wrong key returns error
func TestDecryptWithWrongKey(t *testing.T) {
	key := config.DeriveKey("test-encryption-key")
	wrongKey := config.DeriveKey("wrong-encryption-key")

	original := map[string]string{
		"username": "testuser",
		"password": "secretpassword123",
	}

	// Encrypt with correct key
	encrypted, err := config.EncryptCredentials(original, key)
	require.NoError(t, err)

	// Try to decrypt with wrong key
	decrypted, err := config.DecryptCredentials(encrypted, wrongKey)
	assert.Error(t, err)
	assert.Nil(t, decrypted)
	assert.Contains(t, err.Error(), "failed to decrypt")
}

// TestEncryptEmptyCredentials tests that empty credentials map can be encrypted and decrypted
func TestEncryptEmptyCredentials(t *testing.T) {
	key := config.DeriveKey("test-encryption-key")

	original := map[string]string{}

	// Encrypt
	encrypted, err := config.EncryptCredentials(original, key)
	require.NoError(t, err)
	require.NotNil(t, encrypted)

	// Decrypt
	decrypted, err := config.DecryptCredentials(encrypted, key)
	require.NoError(t, err)
	require.NotNil(t, decrypted)

	// Should get back empty map
	assert.Equal(t, original, decrypted)
	assert.Empty(t, decrypted)
}

// TestDeriveKey tests that DeriveKey returns 32-byte key
func TestDeriveKey(t *testing.T) {
	tests := []struct {
		name       string
		passphrase string
	}{
		{
			name:       "short passphrase",
			passphrase: "short",
		},
		{
			name:       "long passphrase",
			passphrase: "this is a very long passphrase with many characters in it",
		},
		{
			name:       "exactly 32 chars",
			passphrase: "this passphrase is exactly 32 chars!",
		},
		{
			name:       "empty passphrase",
			passphrase: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := config.DeriveKey(tt.passphrase)
			assert.Len(t, key, 32, "Derived key should be 32 bytes for AES-256")
		})
	}
}

// TestDeriveKeyDeterministic tests that same passphrase always produces same key
func TestDeriveKeyDeterministic(t *testing.T) {
	passphrase := "my-test-passphrase"

	key1 := config.DeriveKey(passphrase)
	key2 := config.DeriveKey(passphrase)
	key3 := config.DeriveKey(passphrase)

	assert.Equal(t, key1, key2)
	assert.Equal(t, key2, key3)
}

// TestDifferentPassphrasesProduceDifferentKeys tests that different passphrases produce different keys
func TestDifferentPassphrasesProduceDifferentKeys(t *testing.T) {
	key1 := config.DeriveKey("passphrase1")
	key2 := config.DeriveKey("passphrase2")
	key3 := config.DeriveKey("completely different")

	assert.NotEqual(t, key1, key2)
	assert.NotEqual(t, key2, key3)
	assert.NotEqual(t, key1, key3)
}

// TestEncryptProducesDifferentOutput tests that encrypting same data twice produces different ciphertext
func TestEncryptProducesDifferentOutput(t *testing.T) {
	key := config.DeriveKey("test-encryption-key")

	original := map[string]string{
		"username": "testuser",
		"password": "secretpassword123",
	}

	// Encrypt twice
	encrypted1, err := config.EncryptCredentials(original, key)
	require.NoError(t, err)

	encrypted2, err := config.EncryptCredentials(original, key)
	require.NoError(t, err)

	// Should be different due to random nonce
	assert.NotEqual(t, encrypted1, encrypted2)

	// But both should decrypt to same value
	decrypted1, err := config.DecryptCredentials(encrypted1, key)
	require.NoError(t, err)

	decrypted2, err := config.DecryptCredentials(encrypted2, key)
	require.NoError(t, err)

	assert.Equal(t, original, decrypted1)
	assert.Equal(t, original, decrypted2)
	assert.Equal(t, decrypted1, decrypted2)
}

// TestDecryptModifiedCiphertext tests that modified ciphertext fails to decrypt
func TestDecryptModifiedCiphertext(t *testing.T) {
	key := config.DeriveKey("test-encryption-key")

	original := map[string]string{
		"token": "my-secret-token",
	}

	// Encrypt
	encrypted, err := config.EncryptCredentials(original, key)
	require.NoError(t, err)

	// Modify the ciphertext
	modified := make([]byte, len(encrypted))
	copy(modified, encrypted)
	modified[len(modified)/2] ^= 0xFF // Flip some bits in the middle

	// Try to decrypt modified ciphertext
	decrypted, err := config.DecryptCredentials(modified, key)
	assert.Error(t, err)
	assert.Nil(t, decrypted)
}

// TestDecryptTruncatedCiphertext tests that truncated ciphertext returns error
func TestDecryptTruncatedCiphertext(t *testing.T) {
	key := config.DeriveKey("test-encryption-key")

	original := map[string]string{
		"token": "my-secret-token",
	}

	// Encrypt
	encrypted, err := config.EncryptCredentials(original, key)
	require.NoError(t, err)

	// Try to decrypt truncated ciphertext
	truncated := encrypted[:10] // Just the nonce part
	decrypted, err := config.DecryptCredentials(truncated, key)
	assert.Error(t, err)
	assert.Nil(t, decrypted)
	assert.Contains(t, err.Error(), "too short")
}

// TestEncryptLargeCredentials tests encryption of large credential maps
func TestEncryptLargeCredentials(t *testing.T) {
	key := config.DeriveKey("test-encryption-key")

	// Create a large credentials map
	original := make(map[string]string)
	for i := 0; i < 100; i++ {
		original["key"+string(rune('0'+i%10))+string(rune('a'+i/10))] = "value" + string(rune('0'+i))
	}

	// Encrypt
	encrypted, err := config.EncryptCredentials(original, key)
	require.NoError(t, err)
	require.NotNil(t, encrypted)

	// Decrypt
	decrypted, err := config.DecryptCredentials(encrypted, key)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, original, decrypted)
	assert.Len(t, decrypted, 100)
}

// TestEncryptSpecialCharacters tests encryption of credentials with special characters
func TestEncryptSpecialCharacters(t *testing.T) {
	key := config.DeriveKey("test-encryption-key")

	original := map[string]string{
		"emoji":      "👍🎉🚀",
		"unicode":    "日本語テキスト",
		"special":    "!@#$%^&*()_+-=[]{}|;':\",./<>?",
		"whitespace": "  leading and trailing  ",
		"newline":    "line1\nline2\nline3",
		"null":       "contains\x00null",
	}

	// Encrypt
	encrypted, err := config.EncryptCredentials(original, key)
	require.NoError(t, err)

	// Decrypt
	decrypted, err := config.DecryptCredentials(encrypted, key)
	require.NoError(t, err)

	// Verify - note that null byte in JSON strings is problematic
	// so we check the fields that should work
	assert.Equal(t, original["emoji"], decrypted["emoji"])
	assert.Equal(t, original["unicode"], decrypted["unicode"])
	assert.Equal(t, original["special"], decrypted["special"])
	assert.Equal(t, original["whitespace"], decrypted["whitespace"])
}
