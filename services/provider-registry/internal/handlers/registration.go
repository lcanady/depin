package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lcanady/depin/services/provider-registry/internal/auth"
	"github.com/lcanady/depin/services/provider-registry/internal/middleware"
	"github.com/lcanady/depin/services/provider-registry/internal/validation"
	"github.com/lcanady/depin/services/provider-registry/pkg/types"
	"github.com/sirupsen/logrus"
)

// RegistrationHandler handles provider registration endpoints
type RegistrationHandler struct {
	authService       *auth.Service
	validationService *validation.Service
	logger            *logrus.Logger
}

// NewRegistrationHandler creates a new registration handler
func NewRegistrationHandler(authService *auth.Service, validationService *validation.Service, logger *logrus.Logger) *RegistrationHandler {
	return &RegistrationHandler{
		authService:       authService,
		validationService: validationService,
		logger:            logger,
	}
}

// Register handles provider registration requests
// @Summary Register a new provider
// @Description Register a new compute resource provider
// @Tags registration
// @Accept json
// @Produce json
// @Param registration body types.RegistrationRequest true "Registration request"
// @Success 201 {object} types.RegistrationResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 409 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /api/v1/registration/register [post]
func (h *RegistrationHandler) Register(c *gin.Context) {
	var req types.RegistrationRequest
	
	// Bind JSON request
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Failed to bind registration request")
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:     "Bad Request",
			Message:   "Invalid request format",
			Code:      http.StatusBadRequest,
			Timestamp: time.Now(),
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// Validate registration request
	validationErrors := h.validationService.ValidateRegistrationRequest(&req)
	if len(validationErrors) > 0 {
		h.logger.WithField("email", req.Email).Error("Registration validation failed")
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:            "Validation Failed",
			Message:          "Registration request contains invalid data",
			Code:             http.StatusBadRequest,
			ValidationErrors: validationErrors,
			Timestamp:        time.Now(),
			RequestID:        c.GetString("request_id"),
		})
		return
	}

	// Create provider
	response, err := h.authService.CreateProvider(&req)
	if err != nil {
		h.logger.WithError(err).WithField("email", req.Email).Error("Failed to create provider")
		
		// Handle specific errors
		status := http.StatusInternalServerError
		message := "Failed to create provider account"
		
		// TODO: Add specific error handling for duplicate email, etc.
		
		c.JSON(status, types.ErrorResponse{
			Error:     "Registration Failed",
			Message:   message,
			Code:      status,
			Timestamp: time.Now(),
			RequestID: c.GetString("request_id"),
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"provider_id": response.ProviderID,
		"email":       req.Email,
		"name":        req.Name,
	}).Info("Provider registered successfully")

	c.JSON(http.StatusCreated, response)
}

// Authenticate handles provider authentication requests
// @Summary Authenticate provider
// @Description Authenticate provider and receive access token
// @Tags authentication
// @Accept json
// @Produce json
// @Param auth body types.AuthRequest true "Authentication request"
// @Success 200 {object} types.AuthResponse
// @Failure 401 {object} types.ErrorResponse
// @Failure 400 {object} types.ErrorResponse
// @Router /api/v1/registration/auth [post]
func (h *RegistrationHandler) Authenticate(c *gin.Context) {
	var req types.AuthRequest
	
	// Bind JSON request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:     "Bad Request",
			Message:   "Invalid request format",
			Code:      http.StatusBadRequest,
			Timestamp: time.Now(),
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// Authenticate provider
	response, err := h.authService.Authenticate(&req)
	if err != nil {
		h.logger.WithError(err).WithField("provider_id", req.ProviderID).Error("Authentication failed")
		
		status := http.StatusUnauthorized
		message := "Authentication failed"
		
		if err == auth.ErrInvalidCredentials {
			message = "Invalid provider ID or API key"
		}
		
		c.JSON(status, types.ErrorResponse{
			Error:     "Authentication Failed",
			Message:   message,
			Code:      status,
			Timestamp: time.Now(),
			RequestID: c.GetString("request_id"),
		})
		return
	}

	h.logger.WithField("provider_id", req.ProviderID).Info("Provider authenticated successfully")
	c.JSON(http.StatusOK, response)
}

// GetProfile returns the authenticated provider's profile
// @Summary Get provider profile
// @Description Get the current provider's profile information
// @Tags profile
// @Produce json
// @Security BearerAuth
// @Success 200 {object} types.Provider
// @Failure 401 {object} types.ErrorResponse
// @Failure 404 {object} types.ErrorResponse
// @Router /api/v1/registration/profile [get]
func (h *RegistrationHandler) GetProfile(c *gin.Context) {
	provider, exists := middleware.GetProviderObjectFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse{
			Error:     "Unauthorized",
			Message:   "Provider information not available",
			Code:      http.StatusUnauthorized,
			Timestamp: time.Now(),
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// Don't expose sensitive fields
	safeProvider := *provider
	safeProvider.ApiKey = ""
	safeProvider.ApiKeyHash = ""

	c.JSON(http.StatusOK, safeProvider)
}

// UpdateProfile updates the authenticated provider's profile
// @Summary Update provider profile
// @Description Update the current provider's profile information
// @Tags profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param profile body types.Provider true "Updated profile"
// @Success 200 {object} types.Provider
// @Failure 400 {object} types.ErrorResponse
// @Failure 401 {object} types.ErrorResponse
// @Failure 404 {object} types.ErrorResponse
// @Router /api/v1/registration/profile [put]
func (h *RegistrationHandler) UpdateProfile(c *gin.Context) {
	currentProvider, exists := middleware.GetProviderObjectFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse{
			Error:     "Unauthorized",
			Message:   "Provider information not available",
			Code:      http.StatusUnauthorized,
			Timestamp: time.Now(),
			RequestID: c.GetString("request_id"),
		})
		return
	}

	var updateReq types.Provider
	if err := c.ShouldBindJSON(&updateReq); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:     "Bad Request",
			Message:   "Invalid request format",
			Code:      http.StatusBadRequest,
			Timestamp: time.Now(),
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// Preserve immutable fields
	updateReq.ID = currentProvider.ID
	updateReq.ApiKeyHash = currentProvider.ApiKeyHash
	updateReq.CreatedAt = currentProvider.CreatedAt
	updateReq.UpdatedAt = time.Now().UTC()
	
	// Don't allow status changes through this endpoint
	updateReq.Status = currentProvider.Status

	// Validate updated provider
	validationErrors := h.validationService.ValidateProvider(&updateReq)
	if len(validationErrors) > 0 {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:            "Validation Failed",
			Message:          "Profile update contains invalid data",
			Code:             http.StatusBadRequest,
			ValidationErrors: validationErrors,
			Timestamp:        time.Now(),
			RequestID:        c.GetString("request_id"),
		})
		return
	}

	// TODO: Update provider in repository
	// For now, return the updated provider (this would be implemented when repository is available)
	
	h.logger.WithFields(logrus.Fields{
		"provider_id": updateReq.ID,
		"email":       updateReq.Email,
	}).Info("Provider profile updated")

	// Don't expose sensitive fields
	safeProvider := updateReq
	safeProvider.ApiKey = ""
	safeProvider.ApiKeyHash = ""

	c.JSON(http.StatusOK, safeProvider)
}

// RefreshToken generates a new access token for authenticated provider
// @Summary Refresh access token
// @Description Generate a new access token using current authentication
// @Tags authentication
// @Produce json
// @Security BearerAuth
// @Success 200 {object} types.AuthResponse
// @Failure 401 {object} types.ErrorResponse
// @Router /api/v1/registration/refresh [post]
func (h *RegistrationHandler) RefreshToken(c *gin.Context) {
	provider, exists := middleware.GetProviderObjectFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse{
			Error:     "Unauthorized",
			Message:   "Provider information not available",
			Code:      http.StatusUnauthorized,
			Timestamp: time.Now(),
			RequestID: c.GetString("request_id"),
		})
		return
	}

	// Generate new token
	token, expiresAt, err := h.authService.GenerateToken(provider)
	if err != nil {
		h.logger.WithError(err).WithField("provider_id", provider.ID).Error("Failed to refresh token")
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:     "Token Refresh Failed",
			Message:   "Failed to generate new access token",
			Code:      http.StatusInternalServerError,
			Timestamp: time.Now(),
			RequestID: c.GetString("request_id"),
		})
		return
	}

	response := &types.AuthResponse{
		Token:     token,
		TokenType: "Bearer",
		ExpiresAt: expiresAt,
	}

	h.logger.WithField("provider_id", provider.ID).Info("Token refreshed successfully")
	c.JSON(http.StatusOK, response)
}

// HealthCheck provides a health check endpoint
// @Summary Health check
// @Description Check if the registration service is healthy
// @Tags health
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/registration/health [get]
func (h *RegistrationHandler) HealthCheck(c *gin.Context) {
	response := map[string]interface{}{
		"status":    "healthy",
		"service":   "provider-registry",
		"timestamp": time.Now(),
		"version":   "1.0.0",
	}
	c.JSON(http.StatusOK, response)
}