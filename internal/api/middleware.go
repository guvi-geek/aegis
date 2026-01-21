package api

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// JWTAuthMiddleware validates JWT tokens
func JWTAuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "Authorization header required",
				Code:  "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "Invalid authorization header format",
				Code:  "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Parse and validate token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "Invalid or expired token",
				Code:  "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		// Extract claims and set API key in context
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if apiKey, ok := claims["api_key"].(string); ok {
				c.Set("api_key", apiKey)
			} else {
				c.Set("api_key", tokenString) // Fallback to token itself
			}
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

// ErrorHandlerMiddleware handles errors and returns standard format
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
