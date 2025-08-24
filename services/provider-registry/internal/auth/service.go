package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/lcanady/depin/services/provider-registry/pkg/types"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrTokenExpired       = errors.New("token expired")
	ErrInvalidToken       = errors.New("invalid token")
)

// Service handles authentication and authorization for providers
type Service struct {
	jwtSecret    []byte
	tokenExpiry  time.Duration
	bcryptCost   int
	providerRepo ProviderRepository
}

// ProviderRepository defines the interface for provider data access
type ProviderRepository interface {
	GetByID(id uuid.UUID) (*types.Provider, error)
	GetByEmail(email string) (*types.Provider, error)
	Create(provider *types.Provider) error
	Update(provider *types.Provider) error
	UpdateLastSeen(id uuid.UUID) error
}

// NewService creates a new authentication service
func NewService(jwtSecret string, tokenExpiry time.Duration, providerRepo ProviderRepository) *Service {
	return &Service{
		jwtSecret:    []byte(jwtSecret),
		tokenExpiry:  tokenExpiry,
		bcryptCost:   bcrypt.DefaultCost,
		providerRepo: providerRepo,
	}
}

// GenerateAPIKey generates a new API key for a provider
func (s *Service) GenerateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// HashAPIKey creates a bcrypt hash of the API key
func (s *Service) HashAPIKey(apiKey string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(apiKey), s.bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyAPIKey verifies an API key against its hash
func (s *Service) VerifyAPIKey(apiKey, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(apiKey))
	return err == nil
}

// GenerateToken generates a JWT token for authenticated providers
func (s *Service) GenerateToken(provider *types.Provider) (string, time.Time, error) {
	expiresAt := time.Now().Add(s.tokenExpiry)
	
	claims := jwt.MapClaims{
		"provider_id": provider.ID.String(),
		"email":       provider.Email,
		"role":        "provider",
		"iat":         time.Now().Unix(),
		"exp":         expiresAt.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

// ValidateToken validates a JWT token and returns the provider ID
func (s *Service) ValidateToken(tokenString string) (uuid.UUID, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return uuid.Nil, err
	}

	if !token.Valid {
		return uuid.Nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, ErrInvalidToken
	}

	// Check expiration
	if exp, ok := claims["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			return uuid.Nil, ErrTokenExpired
		}
	}

	// Extract provider ID
	if providerIDStr, ok := claims["provider_id"].(string); ok {
		providerID, err := uuid.Parse(providerIDStr)
		if err != nil {
			return uuid.Nil, ErrInvalidToken
		}
		return providerID, nil
	}

	return uuid.Nil, ErrInvalidToken
}

// Authenticate verifies provider credentials and returns a token
func (s *Service) Authenticate(req *types.AuthRequest) (*types.AuthResponse, error) {
	// Parse provider ID
	providerID, err := uuid.Parse(req.ProviderID)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Get provider from repository
	provider, err := s.providerRepo.GetByID(providerID)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Verify API key
	if !s.VerifyAPIKey(req.ApiKey, provider.ApiKeyHash) {
		return nil, ErrInvalidCredentials
	}

	// Update last seen
	if err := s.providerRepo.UpdateLastSeen(provider.ID); err != nil {
		// Log error but don't fail authentication
	}

	// Generate token
	token, expiresAt, err := s.GenerateToken(provider)
	if err != nil {
		return nil, err
	}

	return &types.AuthResponse{
		Token:     token,
		TokenType: "Bearer",
		ExpiresAt: expiresAt,
	}, nil
}

// CreateProvider creates a new provider with authentication credentials
func (s *Service) CreateProvider(req *types.RegistrationRequest) (*types.RegistrationResponse, error) {
	// Generate API key
	apiKey, err := s.GenerateAPIKey()
	if err != nil {
		return nil, err
	}

	// Hash API key
	apiKeyHash, err := s.HashAPIKey(apiKey)
	if err != nil {
		return nil, err
	}

	// Create provider
	provider := &types.Provider{
		ID:           uuid.New(),
		Name:         req.Name,
		Email:        req.Email,
		Organization: req.Organization,
		Status:       types.ProviderStatusPending,
		ApiKeyHash:   apiKeyHash,
		PublicKey:    req.PublicKey,
		Endpoints:    req.Endpoints,
		Metadata:     req.Metadata,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	// Save provider
	if err := s.providerRepo.Create(provider); err != nil {
		return nil, err
	}

	return &types.RegistrationResponse{
		ProviderID: provider.ID,
		ApiKey:     apiKey,
		Status:     string(provider.Status),
		Message:    "Provider registered successfully. Activation pending.",
	}, nil
}

// GetProviderByToken extracts and returns provider information from token
func (s *Service) GetProviderByToken(tokenString string) (*types.Provider, error) {
	providerID, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	return s.providerRepo.GetByID(providerID)
}