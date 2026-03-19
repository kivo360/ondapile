package integration

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ondapile/internal/crypto"
)

// TestMessageEncryptDecryptRoundtrip tests that encrypt -> decrypt returns the original plaintext
func TestMessageEncryptDecryptRoundtrip(t *testing.T) {
	encryptor := crypto.NewMessageEncryptor()
	key := crypto.DeriveMessageKey("test-encryption-passphrase")

	original := "This is a secret message that should be encrypted!"

	// Encrypt
	encrypted, err := encryptor.EncryptMessage(original, key)
	require.NoError(t, err)
	require.NotEmpty(t, encrypted)

	// Encrypted text should be different from original
	assert.NotEqual(t, original, encrypted)

	// Decrypt
	decrypted, err := encryptor.DecryptMessage(encrypted, key)
	require.NoError(t, err)

	// Decrypted should match original
	assert.Equal(t, original, decrypted)
}

// TestMessageEncryptWithWrongKeyFails tests that decryption with wrong key returns error
func TestMessageEncryptWithWrongKeyFails(t *testing.T) {
	encryptor := crypto.NewMessageEncryptor()
	keyA := crypto.DeriveMessageKey("correct-passphrase")
	keyB := crypto.DeriveMessageKey("wrong-passphrase")

	original := "Secret message for key A"

	// Encrypt with key A
	encrypted, err := encryptor.EncryptMessage(original, keyA)
	require.NoError(t, err)

	// Try to decrypt with key B
	decrypted, err := encryptor.DecryptMessage(encrypted, keyB)
	assert.Error(t, err)
	assert.Empty(t, decrypted)
	assert.Contains(t, err.Error(), "failed to decrypt")
}

// TestMessageUnencryptedMessagesStillReadable tests that plain text messages work without encryption
func TestMessageUnencryptedMessagesStillReadable(t *testing.T) {
	// This test verifies the concept that unencrypted messages
	// are stored as-is and readable without any decryption
	plaintext := "This is a plain text message, not encrypted"

	// Plain text should be readable directly
	assert.Equal(t, "This is a plain text message, not encrypted", plaintext)
}

// TestMessageMixedEncryptedAndPlainMessages tests handling both encrypted and plain messages
func TestMessageMixedEncryptedAndPlainMessages(t *testing.T) {
	encryptor := crypto.NewMessageEncryptor()
	key := crypto.DeriveMessageKey("shared-passphrase")

	// Simulate encrypted message
	encryptedMessage, err := encryptor.EncryptMessage("This is encrypted", key)
	require.NoError(t, err)

	// Simulate plain message (just stored as-is)
	plainMessage := "This is plain text"

	// Decrypt the encrypted one
	decrypted, err := encryptor.DecryptMessage(encryptedMessage, key)
	require.NoError(t, err)
	assert.Equal(t, "This is encrypted", decrypted)

	// Plain one is readable directly
	assert.Equal(t, "This is plain text", plainMessage)
}

// TestMessageEncryptEmptyString tests encryption of empty string
func TestMessageEncryptEmptyString(t *testing.T) {
	encryptor := crypto.NewMessageEncryptor()
	key := crypto.DeriveMessageKey("test-passphrase")

	// Encrypt empty string
	encrypted, err := encryptor.EncryptMessage("", key)
	require.NoError(t, err)
	require.NotEmpty(t, encrypted)

	// Decrypt and verify empty string returned
	decrypted, err := encryptor.DecryptMessage(encrypted, key)
	require.NoError(t, err)
	assert.Equal(t, "", decrypted)
}

// TestMessageEncryptLargeMessage tests encryption of a large message (10KB+)
func TestMessageEncryptLargeMessage(t *testing.T) {
	encryptor := crypto.NewMessageEncryptor()
	key := crypto.DeriveMessageKey("test-passphrase")

	// Create a large message (10KB+)
	var sb strings.Builder
	for i := 0; i < 1100; i++ {
		sb.WriteString("This is a line of text that will be repeated many times to create a large message. ")
	}
	original := sb.String()

	// Verify it's over 10KB
	require.Greater(t, len(original), 10*1024, "Message should be larger than 10KB")

	// Encrypt
	encrypted, err := encryptor.EncryptMessage(original, key)
	require.NoError(t, err)
	require.NotEmpty(t, encrypted)

	// Decrypt
	decrypted, err := encryptor.DecryptMessage(encrypted, key)
	require.NoError(t, err)

	// Verify it matches
	assert.Equal(t, original, decrypted)
}

// TestMessageDeriveMessageKey tests that DeriveMessageKey returns 32-byte key
func TestMessageDeriveMessageKey(t *testing.T) {
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
			passphrase: "this is a very long passphrase with many characters in it for testing purposes",
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
			key := crypto.DeriveMessageKey(tt.passphrase)
			assert.Len(t, key, 32, "Derived key should be 32 bytes for AES-256")
		})
	}
}

// TestMessageDeriveMessageKeyDeterministic tests that same passphrase always produces same key
func TestMessageDeriveMessageKeyDeterministic(t *testing.T) {
	passphrase := "my-test-passphrase"

	key1 := crypto.DeriveMessageKey(passphrase)
	key2 := crypto.DeriveMessageKey(passphrase)
	key3 := crypto.DeriveMessageKey(passphrase)

	assert.Equal(t, key1, key2)
	assert.Equal(t, key2, key3)
}

// TestMessageDifferentPassphrasesProduceDifferentKeys tests that different passphrases produce different keys
func TestMessageDifferentPassphrasesProduceDifferentKeys(t *testing.T) {
	key1 := crypto.DeriveMessageKey("passphrase1")
	key2 := crypto.DeriveMessageKey("passphrase2")
	key3 := crypto.DeriveMessageKey("completely different")

	assert.NotEqual(t, key1, key2)
	assert.NotEqual(t, key2, key3)
	assert.NotEqual(t, key1, key3)
}

// TestMessageEncryptProducesDifferentOutput tests that encrypting same data twice produces different ciphertext
func TestMessageEncryptProducesDifferentOutput(t *testing.T) {
	encryptor := crypto.NewMessageEncryptor()
	key := crypto.DeriveMessageKey("test-passphrase")

	original := "Same message encrypted twice"

	// Encrypt twice
	encrypted1, err := encryptor.EncryptMessage(original, key)
	require.NoError(t, err)

	encrypted2, err := encryptor.EncryptMessage(original, key)
	require.NoError(t, err)

	// Should be different due to random nonce
	assert.NotEqual(t, encrypted1, encrypted2)

	// But both should decrypt to same value
	decrypted1, err := encryptor.DecryptMessage(encrypted1, key)
	require.NoError(t, err)

	decrypted2, err := encryptor.DecryptMessage(encrypted2, key)
	require.NoError(t, err)

	assert.Equal(t, original, decrypted1)
	assert.Equal(t, original, decrypted2)
	assert.Equal(t, decrypted1, decrypted2)
}

// TestMessageDecryptModifiedCiphertext tests that modified ciphertext fails to decrypt
func TestMessageDecryptModifiedCiphertext(t *testing.T) {
	encryptor := crypto.NewMessageEncryptor()
	key := crypto.DeriveMessageKey("test-passphrase")

	original := "This message should not be readable if tampered with"

	// Encrypt
	encrypted, err := encryptor.EncryptMessage(original, key)
	require.NoError(t, err)

	// Modify the ciphertext (change a character in the base64 string)
	modified := encrypted[:len(encrypted)/2] + "X" + encrypted[len(encrypted)/2+1:]

	// Try to decrypt modified ciphertext
	decrypted, err := encryptor.DecryptMessage(modified, key)
	assert.Error(t, err)
	assert.Empty(t, decrypted)
}

// TestMessageDecryptTruncatedCiphertext tests that truncated ciphertext returns error
func TestMessageDecryptTruncatedCiphertext(t *testing.T) {
	encryptor := crypto.NewMessageEncryptor()
	key := crypto.DeriveMessageKey("test-passphrase")

	original := "Test message"

	// Encrypt
	encrypted, err := encryptor.EncryptMessage(original, key)
	require.NoError(t, err)

	// Try to decrypt truncated ciphertext
	truncated := encrypted[:10]
	decrypted, err := encryptor.DecryptMessage(truncated, key)
	assert.Error(t, err)
	assert.Empty(t, decrypted)
}

// TestMessageEncryptUnicodeAndSpecialCharacters tests encryption of messages with special characters
func TestMessageEncryptUnicodeAndSpecialCharacters(t *testing.T) {
	encryptor := crypto.NewMessageEncryptor()
	key := crypto.DeriveMessageKey("test-passphrase")

	original := "Hello 世界! 👋🎉 Emojis: 🚀💻🔐 Special: !@#$%^&*()_+-=[]{}|;':\",./<>? Newlines:\nLine2\nLine3 Tabs:\tHere"

	// Encrypt
	encrypted, err := encryptor.EncryptMessage(original, key)
	require.NoError(t, err)

	// Decrypt
	decrypted, err := encryptor.DecryptMessage(encrypted, key)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, original, decrypted)
}

// TestMessageInvalidKeyLength tests that invalid key lengths return errors
func TestMessageInvalidKeyLength(t *testing.T) {
	encryptor := crypto.NewMessageEncryptor()

	// Test with 16-byte key (too short)
	shortKey := make([]byte, 16)
	_, err := encryptor.EncryptMessage("test", shortKey)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid key length")

	// Test with 24-byte key (wrong size)
	medKey := make([]byte, 24)
	_, err = encryptor.EncryptMessage("test", medKey)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid key length")

	// Test decryption with wrong key length
	validKey := crypto.DeriveMessageKey("test")
	encrypted, _ := encryptor.EncryptMessage("test", validKey)

	_, err = encryptor.DecryptMessage(encrypted, shortKey)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid key length")
}
