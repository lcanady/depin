package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lcanady/depin/services/provider-registry/internal/auth"
	"github.com/lcanady/depin/services/provider-registry/internal/handlers"
	"github.com/lcanady/depin/services/provider-registry/internal/middleware"
	"github.com/lcanady/depin/services/provider-registry/internal/validation"
	"github.com/lcanady/depin/services/provider-registry/pkg/types"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	// Parse command line flags
	var configFile = flag.String("config", "config.yaml", "Configuration file path")
	flag.Parse()

	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	// Load configuration
	config, err := loadConfig(*configFile)
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}

	// Set log level
	if level, err := logrus.ParseLevel(config.LogLevel); err == nil {
		logger.SetLevel(level)
	}

	logger.WithField("config_file", *configFile).Info("Starting Provider Registry Service")

	// Initialize services
	validationService := validation.NewService()
	
	// TODO: Initialize provider repository (will be implemented when database integration is ready)
	// For now, we'll use a mock repository
	providerRepo := newMockProviderRepository()
	
	authService := auth.NewService(
		config.JWT.Secret,
		time.Duration(config.JWT.ExpiryHours)*time.Hour,
		providerRepo,
	)

	// Initialize handlers
	registrationHandler := handlers.NewRegistrationHandler(authService, validationService, logger)

	// Setup HTTP server
	server := setupServer(config, authService, registrationHandler, logger)

	// Start server
	addr := fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port)
	logger.WithField("address", addr).Info("Starting HTTP server")
	
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.WithError(err).Fatal("Failed to start server")
	}
}

// Config represents the application configuration
type Config struct {
	Server struct {
		Host         string   `yaml:"host"`
		Port         int      `yaml:"port"`
		ReadTimeout  int      `yaml:"read_timeout"`
		WriteTimeout int      `yaml:"write_timeout"`
		CORS         struct {
			AllowedOrigins []string `yaml:"allowed_origins"`
			AllowedMethods []string `yaml:"allowed_methods"`
			AllowedHeaders []string `yaml:"allowed_headers"`
		} `yaml:"cors"`
	} `yaml:"server"`
	JWT struct {
		Secret      string `yaml:"secret"`
		ExpiryHours int    `yaml:"expiry_hours"`
	} `yaml:"jwt"`
	RateLimit struct {
		RequestsPerMinute int `yaml:"requests_per_minute"`
		Enabled           bool `yaml:"enabled"`
	} `yaml:"rate_limit"`
	LogLevel string `yaml:"log_level"`
}

// loadConfig loads configuration from file or environment variables
func loadConfig(configFile string) (*Config, error) {
	viper.SetConfigFile(configFile)
	viper.SetConfigType("yaml")
	
	// Set defaults
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.read_timeout", 30)
	viper.SetDefault("server.write_timeout", 30)
	viper.SetDefault("server.cors.allowed_origins", []string{"*"})
	viper.SetDefault("server.cors.allowed_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	viper.SetDefault("server.cors.allowed_headers", []string{"*"})
	viper.SetDefault("jwt.secret", "your-secret-key-change-this-in-production")
	viper.SetDefault("jwt.expiry_hours", 24)
	viper.SetDefault("rate_limit.requests_per_minute", 60)
	viper.SetDefault("rate_limit.enabled", true)
	viper.SetDefault("log_level", "info")
	
	// Enable environment variable override
	viper.SetEnvPrefix("PROVIDER_REGISTRY")
	viper.AutomaticEnv()
	
	// Read configuration file (optional)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}
	
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}
	
	// Validate JWT secret in production
	if config.JWT.Secret == "your-secret-key-change-this-in-production" {
		if os.Getenv("GIN_MODE") == "release" {
			return nil, fmt.Errorf("JWT secret must be changed in production")
		}
	}
	
	return &config, nil
}

// setupServer configures and returns the HTTP server
func setupServer(config *Config, authService *auth.Service, registrationHandler *handlers.RegistrationHandler, logger *logrus.Logger) *http.Server {
	// Set Gin mode
	if config.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Global middleware
	router.Use(middleware.RequestID())
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.CORSMiddleware(
		config.Server.CORS.AllowedOrigins,
		config.Server.CORS.AllowedMethods,
		config.Server.CORS.AllowedHeaders,
	))
	router.Use(middleware.RecoveryMiddleware())

	// Rate limiting
	if config.RateLimit.Enabled {
		rateLimiter := middleware.NewRateLimiter(
			config.RateLimit.RequestsPerMinute,
			time.Minute,
		)
		router.Use(rateLimiter.RateLimit())
	}

	// Request timeout
	router.Use(middleware.TimeoutMiddleware(30 * time.Second))

	// Health check (no auth required)
	router.GET("/health", registrationHandler.HealthCheck)
	router.GET("/api/v1/registration/health", registrationHandler.HealthCheck)

	// API v1 routes
	v1 := router.Group("/api/v1/registration")
	{
		// Public endpoints
		v1.POST("/register", registrationHandler.Register)
		v1.POST("/auth", registrationHandler.Authenticate)

		// Protected endpoints
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(authService))
		protected.Use(middleware.RequireProviderStatus(authService, 
			types.ProviderStatusActive, types.ProviderStatusPending, types.ProviderStatusInactive)) // Allow most statuses for profile access
		{
			protected.GET("/profile", registrationHandler.GetProfile)
			protected.PUT("/profile", registrationHandler.UpdateProfile)
			protected.POST("/refresh", registrationHandler.RefreshToken)
		}
	}

	return &http.Server{
		Addr:         fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(config.Server.WriteTimeout) * time.Second,
	}
}