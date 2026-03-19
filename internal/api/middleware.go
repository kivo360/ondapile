package api

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func AuthMiddleware(apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-API-KEY")
		if key == "" {
			// Also accept ?key= query param (for browser img tags)
			key = c.Query("key")
		}
		if key == "" {
			Unauthorized(c)
			return
		}
		if key != apiKey {
			Unauthorized(c)
			return
		}
		c.Next()
	}
}

// CORSMiddleware adds CORS headers.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, X-API-KEY")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// ProviderFilter extracts optional query param filters.
type ProviderFilter struct {
	Provider  *string
	Status    *string
	AccountID *string
	IsGroup   *bool
}

func GetProviderFilter(c *gin.Context) ProviderFilter {
	f := ProviderFilter{}

	if v := c.Query("provider"); v != "" {
		f.Provider = &v
	}
	if v := c.Query("status"); v != "" {
		f.Status = &v
	}
	if v := c.Query("account_id"); v != "" {
		f.AccountID = &v
	}
	if v := c.Query("is_group"); v != "" {
		b := strings.ToLower(v) == "true"
		f.IsGroup = &b
	}

	return f
}

// PaginationParams extracts cursor and limit from query.
type PaginationParams struct {
	Cursor string
	Limit  int
}

func GetPagination(c *gin.Context) PaginationParams {
	p := PaginationParams{
		Cursor: c.Query("cursor"),
		Limit:  25,
	}
	if l := c.Query("limit"); l != "" {
		if n := parseIntOrZero(l); n > 0 {
			p.Limit = n
			if p.Limit > 100 {
				p.Limit = 100
			}
		}
	}
	return p
}

func parseIntOrZero(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// RateLimitMiddleware enforces per-IP rate limiting with standard headers.
func RateLimitMiddleware(requestsPerSecond float64, burst int) gin.HandlerFunc {
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var mu sync.Mutex
	clients := make(map[string]*client)

	// Cleanup stale entries every minute
	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()
			for ip, c := range clients {
				if time.Since(c.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()

		mu.Lock()
		cl, exists := clients[ip]
		if !exists {
			cl = &client{limiter: rate.NewLimiter(rate.Limit(requestsPerSecond), burst)}
			clients[ip] = cl
		}
		cl.lastSeen = time.Now()
		mu.Unlock()

		// Set rate limit headers
		resetTime := time.Now().Add(time.Second).Unix()
		remaining := cl.limiter.Tokens()
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", burst))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", int(remaining)))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime))

		if !cl.limiter.Allow() {
			c.Header("Retry-After", "1")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"object":  "error",
				"status":  429,
				"code":    "RATE_LIMITED",
				"message": "Too many requests. Please slow down.",
			})
			return
		}

		c.Next()
	}
}
