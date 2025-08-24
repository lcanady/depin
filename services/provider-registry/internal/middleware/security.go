package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lcanady/depin/services/provider-registry/pkg/types"
)

// SecurityHeaders adds security headers to responses
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")
		
		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")
		
		// Enable XSS protection
		c.Header("X-XSS-Protection", "1; mode=block")
		
		// Only allow HTTPS
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		
		// Control referrer information
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// Content Security Policy (adjust as needed)
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'")
		
		c.Next()
	}
}

// CORSMiddleware handles CORS headers for API access
func CORSMiddleware(allowedOrigins []string, allowedMethods []string, allowedHeaders []string) gin.HandlerFunc {
	// Convert origins to map for faster lookup
	originsMap := make(map[string]bool)
	for _, origin := range allowedOrigins {
		originsMap[origin] = true
	}
	
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// Check if origin is allowed
		if origin != "" && (originsMap["*"] || originsMap[origin]) {
			c.Header("Access-Control-Allow-Origin", origin)
		} else if len(allowedOrigins) > 0 && allowedOrigins[0] == "*" {
			c.Header("Access-Control-Allow-Origin", "*")
		}
		
		// Set other CORS headers
		if len(allowedMethods) > 0 {
			c.Header("Access-Control-Allow-Methods", strings.Join(allowedMethods, ", "))
		}
		
		if len(allowedHeaders) > 0 {
			c.Header("Access-Control-Allow-Headers", strings.Join(allowedHeaders, ", "))
		}
		
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400") // 24 hours
		
		// Handle preflight requests
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		
		c.Next()
	}
}

// RequestID generates and adds a unique request ID to each request
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// RateLimiter implements a simple in-memory rate limiter
type RateLimiter struct {
	mu       sync.RWMutex
	clients  map[string]*clientInfo
	requests int           // requests per window
	window   time.Duration // time window
}

type clientInfo struct {
	requests  int
	window    time.Time
	blocked   time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requests int, window time.Duration) *RateLimiter {
	limiter := &RateLimiter{
		clients:  make(map[string]*clientInfo),
		requests: requests,
		window:   window,
	}
	
	// Cleanup goroutine
	go limiter.cleanup()
	
	return limiter
}

// RateLimit middleware function
func (rl *RateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		
		if !rl.allow(clientIP) {
			c.JSON(http.StatusTooManyRequests, types.ErrorResponse{
				Error:     "Rate Limit Exceeded",
				Message:   "Too many requests. Please try again later.",
				Code:      http.StatusTooManyRequests,
				Timestamp: time.Now(),
				RequestID: c.GetString("request_id"),
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// allow checks if a client is allowed to make a request
func (rl *RateLimiter) allow(clientIP string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	now := time.Now()
	client, exists := rl.clients[clientIP]
	
	if !exists {
		rl.clients[clientIP] = &clientInfo{
			requests: 1,
			window:   now.Add(rl.window),
		}
		return true
	}
	
	// Check if client is currently blocked
	if now.Before(client.blocked) {
		return false
	}
	
	// Reset window if expired
	if now.After(client.window) {
		client.requests = 1
		client.window = now.Add(rl.window)
		client.blocked = time.Time{} // Clear any existing block
		return true
	}
	
	// Check if limit exceeded
	if client.requests >= rl.requests {
		// Block client for remaining window time
		client.blocked = client.window
		return false
	}
	
	client.requests++
	return true
}

// cleanup removes old client entries
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute * 10) // Cleanup every 10 minutes
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for clientIP, client := range rl.clients {
				// Remove clients whose window and block have expired
				if now.After(client.window) && (client.blocked.IsZero() || now.After(client.blocked)) {
					delete(rl.clients, clientIP)
				}
			}
			rl.mu.Unlock()
		}
	}
}

// TimeoutMiddleware adds request timeout using context
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set a timeout on the request context
		c.Request = c.Request.WithContext(c.Request.Context())
		c.Set("timeout", timeout)
		c.Next()
	}
}

// RecoveryMiddleware handles panics gracefully
func RecoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, err interface{}) {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:     "Internal Server Error",
			Message:   "An unexpected error occurred",
			Code:      http.StatusInternalServerError,
			Timestamp: time.Now(),
			RequestID: c.GetString("request_id"),
		})
		c.Abort()
	})
}