package oauth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"ondapile/internal/store"

	"github.com/jackc/pgx/v5"
	"golang.org/x/oauth2"
)

// TokenStore provides encrypted storage for OAuth tokens in PostgreSQL.
type TokenStore struct {
	s             *store.Store
	encryptionKey []byte
}

// NewTokenStore creates a new TokenStore.
func NewTokenStore(s *store.Store, encryptionKey []byte) *TokenStore {
	return &TokenStore{
		s:             s,
		encryptionKey: encryptionKey,
	}
}

// encryptToken encrypts a single token string using AES-256-GCM.
func (ts *TokenStore) encryptToken(token string) ([]byte, error) {
	// Wrap token in a map to reuse encryption logic pattern
	data := map[string]string{"token": token}
	plaintext, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal token: %w", err)
	}

	block, err := aes.NewCipher(ts.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// decryptToken decrypts a single token string using AES-256-GCM.
func (ts *TokenStore) decryptToken(ciphertext []byte) (string, error) {
	block, err := aes.NewCipher(ts.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	var data map[string]string
	if err := json.Unmarshal(plaintext, &data); err != nil {
		return "", fmt.Errorf("failed to unmarshal token: %w", err)
	}

	return data["token"], nil
}

// Save stores an OAuth token for the given account and provider.
// The access_token and refresh_token are encrypted; token_type and expiry are stored plaintext.
func (ts *TokenStore) Save(ctx context.Context, accountID, provider string, token *oauth2.Token) error {
	// Encrypt access token
	accessTokenEnc, err := ts.encryptToken(token.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to encrypt access token: %w", err)
	}

	// Encrypt refresh token if present
	var refreshTokenEnc []byte
	if token.RefreshToken != "" {
		refreshTokenEnc, err = ts.encryptToken(token.RefreshToken)
		if err != nil {
			return fmt.Errorf("failed to encrypt refresh token: %w", err)
		}
	}

	// Marshal scopes as JSON
	scopesJSON, err := json.Marshal(token.Extra("scope"))
	if err != nil {
		scopesJSON = []byte("[]")
	}

	// Use upsert to handle both insert and update cases
	q := `
		INSERT INTO oauth_tokens (account_id, provider, access_token_enc, refresh_token_enc, token_type, expiry, scopes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (account_id, provider)
		DO UPDATE SET
			access_token_enc = EXCLUDED.access_token_enc,
			refresh_token_enc = EXCLUDED.refresh_token_enc,
			token_type = EXCLUDED.token_type,
			expiry = EXCLUDED.expiry,
			scopes = EXCLUDED.scopes,
			updated_at = NOW()
	`

	_, err = ts.s.Pool.Exec(ctx, q,
		accountID,
		provider,
		accessTokenEnc,
		refreshTokenEnc,
		token.TokenType,
		token.Expiry,
		scopesJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to save oauth token: %w", err)
	}

	return nil
}

// Load retrieves an OAuth token for the given account and provider.
// Returns nil if no token exists.
func (ts *TokenStore) Load(ctx context.Context, accountID, provider string) (*oauth2.Token, error) {
	q := `
		SELECT access_token_enc, refresh_token_enc, token_type, expiry
		FROM oauth_tokens
		WHERE account_id = $1 AND provider = $2
	`

	var accessTokenEnc, refreshTokenEnc []byte
	var tokenType string
	var expiry *time.Time

	err := ts.s.Pool.QueryRow(ctx, q, accountID, provider).Scan(
		&accessTokenEnc,
		&refreshTokenEnc,
		&tokenType,
		&expiry,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load oauth token: %w", err)
	}

	// Decrypt access token
	accessToken, err := ts.decryptToken(accessTokenEnc)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt access token: %w", err)
	}

	// Decrypt refresh token if present
	var refreshToken string
	if len(refreshTokenEnc) > 0 {
		refreshToken, err = ts.decryptToken(refreshTokenEnc)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt refresh token: %w", err)
		}
	}

	token := &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    tokenType,
	}

	if expiry != nil {
		token.Expiry = *expiry
	}

	return token, nil
}

// Delete removes the OAuth token for the given account and provider.
func (ts *TokenStore) Delete(ctx context.Context, accountID, provider string) error {
	q := `DELETE FROM oauth_tokens WHERE account_id = $1 AND provider = $2`
	_, err := ts.s.Pool.Exec(ctx, q, accountID, provider)
	if err != nil {
		return fmt.Errorf("failed to delete oauth token: %w", err)
	}
	return nil
}
