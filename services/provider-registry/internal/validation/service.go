package validation

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/lcanady/depin/services/provider-registry/pkg/types"
)

var (
	ErrInvalidPublicKey    = errors.New("invalid public key format")
	ErrInvalidEndpoint     = errors.New("invalid endpoint configuration")
	ErrDuplicateEndpoint   = errors.New("duplicate endpoint detected")
	ErrInvalidEmail        = errors.New("invalid email format")
	ErrInvalidName         = errors.New("invalid provider name")
	ErrMissingRequiredTerm = errors.New("terms and conditions must be accepted")
)

// Service handles validation of registration requests and provider data
type Service struct {
	emailRegex      *regexp.Regexp
	nameRegex       *regexp.Regexp
	allowedProtocols map[string]bool
	requiredPorts    map[string]int
}

// NewService creates a new validation service
func NewService() *Service {
	// RFC 5322 compliant email regex (simplified)
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	
	// Provider name: alphanumeric, spaces, hyphens, underscores, 3-100 chars
	nameRegex := regexp.MustCompile(`^[a-zA-Z0-9\s\-_]{3,100}$`)
	
	allowedProtocols := map[string]bool{
		"http":      true,
		"https":     true,
		"grpc":      true,
		"grpc+tls":  true,
		"websocket": true,
		"wss":       true,
	}
	
	requiredPorts := map[string]int{
		"http":      80,
		"https":     443,
		"grpc":      9000,
		"grpc+tls":  9443,
		"websocket": 8080,
		"wss":       8443,
	}
	
	return &Service{
		emailRegex:       emailRegex,
		nameRegex:        nameRegex,
		allowedProtocols: allowedProtocols,
		requiredPorts:    requiredPorts,
	}
}

// ValidateRegistrationRequest validates a complete registration request
func (s *Service) ValidateRegistrationRequest(req *types.RegistrationRequest) []types.ValidationError {
	var errors []types.ValidationError
	
	// Validate name
	if err := s.ValidateName(req.Name); err != nil {
		errors = append(errors, types.ValidationError{
			Field:   "name",
			Message: err.Error(),
			Code:    "INVALID_NAME",
		})
	}
	
	// Validate email
	if err := s.ValidateEmail(req.Email); err != nil {
		errors = append(errors, types.ValidationError{
			Field:   "email",
			Message: err.Error(),
			Code:    "INVALID_EMAIL",
		})
	}
	
	// Validate public key
	if err := s.ValidatePublicKey(req.PublicKey); err != nil {
		errors = append(errors, types.ValidationError{
			Field:   "public_key",
			Message: err.Error(),
			Code:    "INVALID_PUBLIC_KEY",
		})
	}
	
	// Validate endpoints
	if endpointErrors := s.ValidateEndpoints(req.Endpoints); len(endpointErrors) > 0 {
		errors = append(errors, endpointErrors...)
	}
	
	// Validate terms acceptance
	if !req.Terms {
		errors = append(errors, types.ValidationError{
			Field:   "terms",
			Message: "Terms and conditions must be accepted",
			Code:    "TERMS_NOT_ACCEPTED",
		})
	}
	
	return errors
}

// ValidateName validates provider name format and constraints
func (s *Service) ValidateName(name string) error {
	if len(strings.TrimSpace(name)) == 0 {
		return errors.New("provider name is required")
	}
	
	if len(name) < 3 {
		return errors.New("provider name must be at least 3 characters")
	}
	
	if len(name) > 100 {
		return errors.New("provider name must be less than 100 characters")
	}
	
	if !s.nameRegex.MatchString(name) {
		return errors.New("provider name contains invalid characters")
	}
	
	return nil
}

// ValidateEmail validates email format
func (s *Service) ValidateEmail(email string) error {
	if len(strings.TrimSpace(email)) == 0 {
		return errors.New("email is required")
	}
	
	if !s.emailRegex.MatchString(email) {
		return ErrInvalidEmail
	}
	
	return nil
}

// ValidatePublicKey validates RSA public key format and strength
func (s *Service) ValidatePublicKey(publicKeyPEM string) error {
	if len(strings.TrimSpace(publicKeyPEM)) == 0 {
		return errors.New("public key is required")
	}
	
	// Parse PEM block
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return errors.New("invalid PEM format")
	}
	
	// Validate block type
	if block.Type != "PUBLIC KEY" && block.Type != "RSA PUBLIC KEY" {
		return fmt.Errorf("invalid PEM block type: expected 'PUBLIC KEY' or 'RSA PUBLIC KEY', got '%s'", block.Type)
	}
	
	// Parse public key
	var pubKey interface{}
	var err error
	
	if block.Type == "PUBLIC KEY" {
		pubKey, err = x509.ParsePKIXPublicKey(block.Bytes)
	} else {
		pubKey, err = x509.ParsePKCS1PublicKey(block.Bytes)
	}
	
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}
	
	// Ensure it's an RSA key
	rsaKey, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return errors.New("public key must be RSA")
	}
	
	// Check key size (minimum 2048 bits for security)
	keySize := rsaKey.N.BitLen()
	if keySize < 2048 {
		return fmt.Errorf("RSA key size must be at least 2048 bits, got %d", keySize)
	}
	
	return nil
}

// ValidateEndpoints validates provider endpoints
func (s *Service) ValidateEndpoints(endpoints []types.Endpoint) []types.ValidationError {
	var errors []types.ValidationError
	
	if len(endpoints) == 0 {
		errors = append(errors, types.ValidationError{
			Field:   "endpoints",
			Message: "At least one endpoint is required",
			Code:    "MISSING_ENDPOINTS",
		})
		return errors
	}
	
	// Track unique endpoints to prevent duplicates
	seen := make(map[string]bool)
	
	for i, endpoint := range endpoints {
		fieldPrefix := fmt.Sprintf("endpoints[%d]", i)
		
		// Validate endpoint type
		if endpoint.Type == "" {
			errors = append(errors, types.ValidationError{
				Field:   fieldPrefix + ".type",
				Message: "Endpoint type is required",
				Code:    "MISSING_TYPE",
			})
		}
		
		// Validate URL
		if err := s.validateEndpointURL(endpoint); err != nil {
			errors = append(errors, types.ValidationError{
				Field:   fieldPrefix + ".url",
				Message: err.Error(),
				Code:    "INVALID_URL",
			})
		}
		
		// Validate protocol
		if err := s.validateEndpointProtocol(endpoint); err != nil {
			errors = append(errors, types.ValidationError{
				Field:   fieldPrefix + ".protocol",
				Message: err.Error(),
				Code:    "INVALID_PROTOCOL",
			})
		}
		
		// Check for duplicates
		key := fmt.Sprintf("%s:%s:%d", endpoint.Type, endpoint.URL, endpoint.Port)
		if seen[key] {
			errors = append(errors, types.ValidationError{
				Field:   fieldPrefix,
				Message: "Duplicate endpoint detected",
				Code:    "DUPLICATE_ENDPOINT",
			})
		}
		seen[key] = true
	}
	
	return errors
}

// validateEndpointURL validates endpoint URL format
func (s *Service) validateEndpointURL(endpoint types.Endpoint) error {
	if endpoint.URL == "" {
		return errors.New("endpoint URL is required")
	}
	
	parsedURL, err := url.Parse(endpoint.URL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}
	
	if parsedURL.Scheme == "" {
		return errors.New("URL scheme is required")
	}
	
	if parsedURL.Host == "" {
		return errors.New("URL host is required")
	}
	
	return nil
}

// validateEndpointProtocol validates endpoint protocol
func (s *Service) validateEndpointProtocol(endpoint types.Endpoint) error {
	if endpoint.Protocol == "" {
		return errors.New("endpoint protocol is required")
	}
	
	protocol := strings.ToLower(endpoint.Protocol)
	if !s.allowedProtocols[protocol] {
		return fmt.Errorf("unsupported protocol: %s", endpoint.Protocol)
	}
	
	// Validate port consistency
	if endpoint.Port <= 0 {
		// Use default port for protocol
		if defaultPort, exists := s.requiredPorts[protocol]; exists {
			endpoint.Port = defaultPort
		}
	}
	
	// Validate port range
	if endpoint.Port < 1 || endpoint.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", endpoint.Port)
	}
	
	// Validate secure flag consistency
	secureProtocols := map[string]bool{
		"https":     true,
		"grpc+tls":  true,
		"wss":       true,
	}
	
	if secureProtocols[protocol] && !endpoint.Secure {
		return fmt.Errorf("protocol %s requires secure flag to be true", endpoint.Protocol)
	}
	
	return nil
}

// ValidateProvider validates an existing provider record
func (s *Service) ValidateProvider(provider *types.Provider) []types.ValidationError {
	var errors []types.ValidationError
	
	// Validate basic fields
	if err := s.ValidateName(provider.Name); err != nil {
		errors = append(errors, types.ValidationError{
			Field:   "name",
			Message: err.Error(),
			Code:    "INVALID_NAME",
		})
	}
	
	if err := s.ValidateEmail(provider.Email); err != nil {
		errors = append(errors, types.ValidationError{
			Field:   "email",
			Message: err.Error(),
			Code:    "INVALID_EMAIL",
		})
	}
	
	if err := s.ValidatePublicKey(provider.PublicKey); err != nil {
		errors = append(errors, types.ValidationError{
			Field:   "public_key",
			Message: err.Error(),
			Code:    "INVALID_PUBLIC_KEY",
		})
	}
	
	// Validate endpoints
	if endpointErrors := s.ValidateEndpoints(provider.Endpoints); len(endpointErrors) > 0 {
		errors = append(errors, endpointErrors...)
	}
	
	return errors
}