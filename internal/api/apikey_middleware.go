package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ondapile/internal/store"
)

// ApiKeyMiddleware validates API keys against the database.
// It extracts the key from Authorization header (Bearer token),
// falls back to X-API-KEY header, then ?key= query param.
// The key is SHA-256 hashed and looked up in the apikey table.
// Valid keys must be enabled and not expired.
// Sets "organization_id" in gin context for downstream handlers.
func ApiKeyMiddleware(apiKeys *store.ApiKeyStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract key from various sources (in order of preference)
		var rawKey string

		// 1. Try Authorization: Bearer <key>
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			const prefix = "Bearer "
			if strings.HasPrefix(authHeader, prefix) {
				rawKey = strings.TrimPrefix(authHeader, prefix)
			}
		}

		// 2. Fall back to X-API-KEY header
		if rawKey == "" {
			rawKey = c.GetHeader("X-API-KEY")
		}

		// 3. Fall back to ?key= query param
		if rawKey == "" {
			rawKey = c.Query("key")
		}

		// No key provided
		if rawKey == "" {
			Unauthorized(c)
			return
		}

		// SHA-256 hash the raw key
		hash := sha256.Sum256([]byte(rawKey))
		keyHash := hex.EncodeToString(hash[:])

		// Look up in database
		apiKey, err := apiKeys.LookupByKeyHash(c.Request.Context(), keyHash)
		if err != nil {
			Unauthorized(c)
			return
		}

		// Key not found
		if apiKey == nil {
			Unauthorized(c)
			return
		}

		// Check if enabled (null = enabled)
		if apiKey.Enabled != nil && !*apiKey.Enabled {
			Unauthorized(c)
			return
		}

		// Check if expired
		if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
			Unauthorized(c)
			return
		}

		// Set organization_id in context for downstream handlers
		c.Set("organization_id", apiKey.ReferenceID)

		// Fire-and-forget: update lastRequest timestamp
		go func() {
			// Use a new context with timeout for the async update
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = apiKeys.UpdateLastUsed(ctx, apiKey.ID)
		}()

		c.Next()
	}
}
