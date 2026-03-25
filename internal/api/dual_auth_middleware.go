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

// DualAuthMiddleware tries Better Auth DB-backed key first,
// then falls back to static API key for backward compatibility.
func DualAuthMiddleware(apiKeys *store.ApiKeyStore, staticKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawKey := extractKey(c)
		if rawKey == "" {
			Unauthorized(c)
			return
		}

		// 1. Try DB-backed lookup (Better Auth keys)
		hash := sha256.Sum256([]byte(rawKey))
		keyHash := hex.EncodeToString(hash[:])
		apiKey, lookupErr := apiKeys.LookupByKeyHash(c.Request.Context(), keyHash)

		// If DB lookup failed (not just "key not found"), fail closed
		if lookupErr != nil && apiKey == nil {
			c.AbortWithStatusJSON(503, gin.H{"error": "Service temporarily unavailable"})
			return
		}
		if apiKey != nil {
			if apiKey.Enabled != nil && !*apiKey.Enabled {
				Unauthorized(c)
				return
			}
			if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
				Unauthorized(c)
				return
			}
			c.Set("organization_id", apiKey.ReferenceID)
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = apiKeys.UpdateLastUsed(ctx, apiKey.ID)
			}()
			c.Next()
			return
		}

		// 2. Fall back to static key
		if staticKey != "" && rawKey == staticKey {
			c.Next()
			return
		}

		Unauthorized(c)
	}
}

func extractKey(c *gin.Context) string {
	if auth := c.GetHeader("Authorization"); auth != "" {
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimPrefix(auth, "Bearer ")
		}
	}
	if key := c.GetHeader("X-API-KEY"); key != "" {
		return key
	}
	return c.Query("key")
}
