package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func AdminAuthMiddleware(adminAPIKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != adminAPIKey {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "Unauthorized",
				Code:  "UNAUTHORIZED",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RateLimiter manages rate limiting per API key
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rps      float64
	burst    int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rps float64, burst int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rps:      rps,
		burst:    burst,
	}
}

// GetLimiter gets or creates a limiter for an API key
func (rl *RateLimiter) GetLimiter(key string) *rate.Limiter {
	rl.mu.RLock()
	limiter, exists := rl.limiters[key]
	rl.mu.RUnlock()

	if exists {
		return limiter
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Double-check after acquiring write lock
	if limiter, exists := rl.limiters[key]; exists {
		return limiter
	}

	// Create new limiter
	limiter = rate.NewLimiter(rate.Limit(rl.rps), rl.burst)
	rl.limiters[key] = limiter

	// Cleanup old limiters periodically (simple implementation)
	go func() {
		time.Sleep(1 * time.Hour)
		rl.mu.Lock()
		delete(rl.limiters, key)
		rl.mu.Unlock()
	}()

	return limiter
}

// RateLimitMiddleware creates rate limiting middleware
func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey, exists := c.Get("api_key")
		if !exists {
			apiKey = c.ClientIP() // Fallback to IP if no API key
		}

		key := apiKey.(string)
		lim := limiter.GetLimiter(key)

		if !lim.Allow() {
			c.JSON(http.StatusTooManyRequests, ErrorResponse{
				Error: "Rate limit exceeded",
				Code:  "RATE_LIMIT_EXCEEDED",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			log.Error().Err(err).Msg("Request error")

			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error: err.Error(),
				Code:  "INTERNAL_ERROR",
			})
		}
	}
}
