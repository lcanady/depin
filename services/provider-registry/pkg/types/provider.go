package types

import (
	"time"

	"github.com/google/uuid"
)

// ProviderStatus represents the current status of a provider
type ProviderStatus string

const (
	ProviderStatusPending   ProviderStatus = "pending"
	ProviderStatusActive    ProviderStatus = "active"
	ProviderStatusInactive  ProviderStatus = "inactive"
	ProviderStatusSuspended ProviderStatus = "suspended"
)

// Provider represents a compute resource provider in the network
type Provider struct {
	ID           uuid.UUID      `json:"id" yaml:"id"`
	Name         string         `json:"name" yaml:"name" binding:"required,min=3,max=100"`
	Email        string         `json:"email" yaml:"email" binding:"required,email"`
	Organization string         `json:"organization,omitempty" yaml:"organization,omitempty"`
	Status       ProviderStatus `json:"status" yaml:"status"`
	ApiKey       string         `json:"-" yaml:"-"` // Never expose in JSON
	ApiKeyHash   string         `json:"-" yaml:"-"` // Store hashed version
	PublicKey    string         `json:"public_key" yaml:"public_key"`
	Endpoints    []Endpoint     `json:"endpoints" yaml:"endpoints"`
	Metadata     Metadata       `json:"metadata" yaml:"metadata"`
	CreatedAt    time.Time      `json:"created_at" yaml:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at" yaml:"updated_at"`
	LastSeen     *time.Time     `json:"last_seen,omitempty" yaml:"last_seen,omitempty"`
}

// Endpoint represents a network endpoint for a provider
type Endpoint struct {
	Type     string `json:"type" yaml:"type" binding:"required"` // "api", "grpc", "websocket"
	URL      string `json:"url" yaml:"url" binding:"required,url"`
	Port     int    `json:"port,omitempty" yaml:"port,omitempty"`
	Protocol string `json:"protocol" yaml:"protocol" binding:"required"`
	Secure   bool   `json:"secure" yaml:"secure"`
}

// Metadata contains additional provider information
type Metadata struct {
	Region           string            `json:"region,omitempty" yaml:"region,omitempty"`
	DataCenter       string            `json:"data_center,omitempty" yaml:"data_center,omitempty"`
	SupportedFormats []string          `json:"supported_formats,omitempty" yaml:"supported_formats,omitempty"`
	Certifications   []string          `json:"certifications,omitempty" yaml:"certifications,omitempty"`
	Tags             map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Version          string            `json:"version,omitempty" yaml:"version,omitempty"`
}

// RegistrationRequest represents a provider registration request
type RegistrationRequest struct {
	Name         string            `json:"name" binding:"required,min=3,max=100"`
	Email        string            `json:"email" binding:"required,email"`
	Organization string            `json:"organization,omitempty"`
	PublicKey    string            `json:"public_key" binding:"required"`
	Endpoints    []Endpoint        `json:"endpoints" binding:"required,min=1"`
	Metadata     Metadata          `json:"metadata,omitempty"`
	Terms        bool              `json:"terms" binding:"required"` // Must accept terms
}

// RegistrationResponse represents the response to a successful registration
type RegistrationResponse struct {
	ProviderID uuid.UUID `json:"provider_id"`
	ApiKey     string    `json:"api_key"`
	Status     string    `json:"status"`
	Message    string    `json:"message"`
	ExpiresAt  time.Time `json:"expires_at,omitempty"`
}

// TokenClaims represents JWT token claims for provider authentication
type TokenClaims struct {
	ProviderID uuid.UUID `json:"provider_id"`
	Email      string    `json:"email"`
	Role       string    `json:"role"`
	IssuedAt   int64     `json:"iat"`
	ExpiresAt  int64     `json:"exp"`
}

// AuthRequest represents an authentication request
type AuthRequest struct {
	ProviderID string `json:"provider_id" binding:"required"`
	ApiKey     string `json:"api_key" binding:"required"`
}

// AuthResponse represents the response to a successful authentication
type AuthResponse struct {
	Token     string    `json:"token"`
	TokenType string    `json:"token_type"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error            string            `json:"error"`
	Message          string            `json:"message"`
	Code             int               `json:"code"`
	ValidationErrors []ValidationError `json:"validation_errors,omitempty"`
	Timestamp        time.Time         `json:"timestamp"`
	RequestID        string            `json:"request_id,omitempty"`
}