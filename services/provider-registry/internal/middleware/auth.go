package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lcanady/depin/services/provider-registry/internal/auth"
	"github.com/lcanady/depin/services/provider-registry/pkg/types"
)

// AuthService interface for middleware
type AuthService interface {
	ValidateToken(token string) (uuid.UUID, error)
	GetProviderByToken(token string) (*types.Provider, error)
}

// AuthMiddleware provides JWT-based authentication for protected routes
func AuthMiddleware(authService AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, types.ErrorResponse{
				Error:     "Unauthorized",
				Message:   "Authorization header required",
				Code:      http.StatusUnauthorized,
				Timestamp: c.GetTime("start_time"),
				RequestID: c.GetString("request_id"),
			})
			c.Abort()
			return
		}

		// Check Bearer token format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, types.ErrorResponse{
				Error:     "Unauthorized",
				Message:   "Invalid authorization format. Expected: Bearer <token>",
				Code:      http.StatusUnauthorized,
				Timestamp: c.GetTime("start_time"),
				RequestID: c.GetString("request_id"),
			})
			c.Abort()
			return
		}

		token := parts[1]

		// Validate token
		providerID, err := authService.ValidateToken(token)
		if err != nil {
			status := http.StatusUnauthorized
			message := "Invalid token"

			switch err {
			case auth.ErrTokenExpired:
				message = "Token expired"
			case auth.ErrInvalidToken:
				message = "Invalid token"
			}

			c.JSON(status, types.ErrorResponse{
				Error:     "Unauthorized",
				Message:   message,
				Code:      status,
				Timestamp: c.GetTime("start_time"),
				RequestID: c.GetString("request_id"),
			})
			c.Abort()
			return
		}

		// Store provider ID in context for use by handlers
		c.Set("provider_id", providerID)
		c.Set("authenticated", true)

		c.Next()
	}
}

// OptionalAuthMiddleware provides optional authentication (doesn't fail if no token)
func OptionalAuthMiddleware(authService AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				token := parts[1]
				if providerID, err := authService.ValidateToken(token); err == nil {
					c.Set("provider_id", providerID)
					c.Set("authenticated", true)
				}
			}
		}
		c.Next()
	}
}

// RequireProviderStatus ensures the authenticated provider has specific status
func RequireProviderStatus(authService AuthService, allowedStatuses ...types.ProviderStatus) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, exists := c.Get("provider_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, types.ErrorResponse{
				Error:     "Unauthorized",
				Message:   "Authentication required",
				Code:      http.StatusUnauthorized,
				Timestamp: c.GetTime("start_time"),
				RequestID: c.GetString("request_id"),
			})
			c.Abort()
			return
		}

		// Get provider details
		authHeader := c.GetHeader("Authorization")
		parts := strings.SplitN(authHeader, " ", 2)
		token := parts[1]

		provider, err := authService.GetProviderByToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, types.ErrorResponse{
				Error:     "Unauthorized",
				Message:   "Invalid provider",
				Code:      http.StatusUnauthorized,
				Timestamp: c.GetTime("start_time"),
				RequestID: c.GetString("request_id"),
			})
			c.Abort()
			return
		}

		// Check if provider status is allowed
		allowed := false
		for _, status := range allowedStatuses {
			if provider.Status == status {
				allowed = true
				break
			}
		}

		if !allowed {
			c.JSON(http.StatusForbidden, types.ErrorResponse{
				Error:     "Forbidden",
				Message:   "Provider status does not allow this operation",
				Code:      http.StatusForbidden,
				Timestamp: c.GetTime("start_time"),
				RequestID: c.GetString("request_id"),
			})
			c.Abort()
			return
		}

		// Store provider in context
		c.Set("provider", provider)
		c.Next()
	}
}

// GetProviderFromContext extracts provider ID from Gin context
func GetProviderFromContext(c *gin.Context) (uuid.UUID, bool) {
	if providerID, exists := c.Get("provider_id"); exists {
		if id, ok := providerID.(uuid.UUID); ok {
			return id, true
		}
	}
	return uuid.Nil, false
}

// GetProviderObjectFromContext extracts provider object from Gin context
func GetProviderObjectFromContext(c *gin.Context) (*types.Provider, bool) {
	if provider, exists := c.Get("provider"); exists {
		if p, ok := provider.(*types.Provider); ok {
			return p, true
		}
	}
	return nil, false
}