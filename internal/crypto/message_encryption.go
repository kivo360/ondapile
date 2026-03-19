package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
)

// MessageEncryptor handles encryption and decryption of message content.
// It holds no state and provides methods for message-level encryption.
type MessageEncryptor struct{}

// NewMessageEncryptor creates a new MessageEncryptor instance.
func NewMessageEncryptor() *MessageEncryptor {
	return &MessageEncryptor{}
}

// DeriveMessageKey creates a 32-byte AES-256 key from a passphrase using SHA-256.
// This allows per-account encryption where each account can have its own passphrase.
func DeriveMessageKey(passphrase string) []byte {
	h := sha256.Sum256([]byte(passphrase))
	return h[:]
}

// EncryptMessage encrypts plaintext message text using AES-256-GCM.
// The output is base64-encoded and includes the nonce prepended to the ciphertext.
func (e *MessageEncryptor) EncryptMessage(plaintext string, key []byte) (string, error) {
	if len(key) != 32 {
		return "", fmt.Errorf("invalid key length: expected 32 bytes, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and prepend nonce: nonce || ciphertext
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Return base64-encoded string for storage in TEXT columns
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptMessage decrypts a base64-encoded ciphertext string using AES-256-GCM.
// The ciphertext must have the nonce prepended (as produced by EncryptMessage).
func (e *MessageEncryptor) DecryptMessage(ciphertext string, key []byte) (string, error) {
	if len(key) != 32 {
		return "", fmt.Errorf("invalid key length: expected 32 bytes, got %d", len(key))
	}

	// Decode base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	// Split nonce and ciphertext
	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}
