package oauth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// stateEntry holds the mapping from state to account/provider with creation time.
type stateEntry struct {
	accountID string
	provider  string
	createdAt time.Time
}

// Handler manages OAuth callback handling and CSRF state validation.
type Handler struct {
	tokenStore *TokenStore
	states     sync.Map // map[string]*stateEntry
}

// NewHandler creates a new OAuth handler.
func NewHandler(tokenStore *TokenStore) *Handler {
	h := &Handler{
		tokenStore: tokenStore,
	}

	// Start background cleanup goroutine for expired states
	go h.cleanupLoop()

	return h
}

// cleanupLoop periodically removes expired states.
func (h *Handler) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		h.cleanupExpiredStates()
	}
}

// cleanupExpiredStates removes states older than 10 minutes.
func (h *Handler) cleanupExpiredStates() {
	expireBefore := time.Now().Add(-10 * time.Minute)

	h.states.Range(func(key, value interface{}) bool {
		entry := value.(*stateEntry)
		if entry.createdAt.Before(expireBefore) {
			h.states.Delete(key)
		}
		return true
	})
}

// GenerateState creates a cryptographically random state string and stores the
// accountID/provider mapping. The state expires after 10 minutes.
func (h *Handler) GenerateState(accountID, provider string) string {
	// Generate 32 random bytes (64 hex characters)
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based state if crypto rand fails
		return fmt.Sprintf("%s_%s_%d", accountID, provider, time.Now().UnixNano())
	}

	state := hex.EncodeToString(b)

	// Store the mapping
	h.states.Store(state, &stateEntry{
		accountID: accountID,
		provider:  provider,
		createdAt: time.Now(),
	})

	return state
}

// ValidateState validates the state parameter and returns the associated
// accountID and provider. The state is deleted after validation (one-time use).
// Returns an error if the state is invalid or expired.
func (h *Handler) ValidateState(state string) (accountID, provider string, err error) {
	if state == "" {
		return "", "", fmt.Errorf("state parameter is required")
	}

	// Load and delete the state (one-time use)
	value, loaded := h.states.LoadAndDelete(state)
	if !loaded {
		return "", "", fmt.Errorf("invalid or expired state")
	}

	entry := value.(*stateEntry)

	// Check if state is expired (10 minutes)
	if time.Since(entry.createdAt) > 10*time.Minute {
		return "", "", fmt.Errorf("state has expired")
	}

	return entry.accountID, entry.provider, nil
}
