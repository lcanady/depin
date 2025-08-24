package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/lcanady/depin/services/verification/internal/service"
	pb "github.com/lcanady/depin/services/verification/proto"
	gpu_discovery "github.com/lcanady/depin/services/gpu-discovery/proto"
	"github.com/lcanady/depin/database/inventory/repositories"
	"github.com/lcanady/depin/database/inventory/config"
)

const (
	serviceName = "verification-service"
	version     = "1.0.0"
)

func main() {
	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)
	
	logger.WithFields(logrus.Fields{
		"service": serviceName,
		"version": version,
	}).Info("Starting verification service")
	
	// Load configuration
	if err := loadConfig(); err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}
	
	// Connect to database
	dbConfig := &config.DatabaseConfig{
		Host:                viper.GetString("database.host"),
		Port:                viper.GetInt("database.port"),
		Database:           viper.GetString("database.database"),
		Username:           viper.GetString("database.username"),
		Password:           viper.GetString("database.password"),
		SSLMode:            viper.GetString("database.ssl_mode"),
		MaxOpenConnections: viper.GetInt("database.max_open_connections"),
		MaxIdleConnections: viper.GetInt("database.max_idle_connections"),
		ConnectionMaxLifetime: viper.GetDuration("database.connection_max_lifetime"),
	}
	
	repositoryManager, err := repositories.NewRepositoryManager(context.Background(), dbConfig, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to connect to database")
	}
	defer repositoryManager.Close()
	
	// Connect to GPU discovery service
	gpuDiscoveryAddr := viper.GetString("gpu_discovery.grpc_address")
	gpuConn, err := grpc.Dial(gpuDiscoveryAddr, grpc.WithInsecure())
	if err != nil {
		logger.WithError(err).Fatal("Failed to connect to GPU discovery service")
	}
	defer gpuConn.Close()
	
	gpuClient := gpu_discovery.NewGPUDiscoveryServiceClient(gpuConn)
	
	// Create verification service
	verificationService := service.NewVerificationService(
		logger,
		gpuClient,
		repositoryManager,
		&service.VerificationServiceConfig{
			DefaultTimeout:    viper.GetDuration("verification.default_timeout"),
			MaxConcurrentJobs: viper.GetInt("verification.max_concurrent_jobs"),
			ResultRetention:   viper.GetDuration("verification.result_retention"),
		},
	)
	
	// Create gRPC server
	grpcServer := grpc.NewServer()
	
	// Register services
	pb.RegisterVerificationServiceServer(grpcServer, verificationService)
	
	// Register health check service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus(serviceName, grpc_health_v1.HealthCheckResponse_SERVING)
	
	// Register reflection service for debugging
	reflection.Register(grpcServer)
	
	// Create gRPC listener
	grpcPort := viper.GetInt("server.grpc_port")
	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		logger.WithError(err).Fatal("Failed to create gRPC listener")
	}
	
	// Start gRPC server
	go func() {
		logger.WithField("port", grpcPort).Info("Starting gRPC server")
		if err := grpcServer.Serve(grpcListener); err != nil {
			logger.WithError(err).Fatal("gRPC server failed")
		}
	}()
	
	// Create HTTP server for health checks and metrics
	httpPort := viper.GetInt("server.http_port")
	httpServer := &http.Server{
		Addr:           fmt.Sprintf(":%d", httpPort),
		Handler:        createHTTPHandler(logger, repositoryManager),
		ReadTimeout:    viper.GetDuration("server.read_timeout"),
		WriteTimeout:   viper.GetDuration("server.write_timeout"),
		IdleTimeout:    30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	
	// Start HTTP server
	go func() {
		logger.WithField("port", httpPort).Info("Starting HTTP server")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("HTTP server failed")
		}
	}()
	
	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	<-sigChan
	logger.Info("Shutdown signal received")
	
	// Graceful shutdown
	shutdownTimeout := viper.GetDuration("server.shutdown_timeout")
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	
	// Stop HTTP server
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.WithError(err).Error("Failed to shutdown HTTP server gracefully")
	}
	
	// Stop gRPC server
	grpcServer.GracefulStop()
	
	logger.Info("Verification service stopped")
}

func loadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/verification-service")
	
	// Set defaults
	viper.SetDefault("server.grpc_port", 8080)
	viper.SetDefault("server.http_port", 8081)
	viper.SetDefault("server.shutdown_timeout", "30s")
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")
	
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.database", "depin_verification")
	viper.SetDefault("database.username", "verification_user")
	viper.SetDefault("database.ssl_mode", "prefer")
	viper.SetDefault("database.max_open_connections", 25)
	viper.SetDefault("database.max_idle_connections", 10)
	viper.SetDefault("database.connection_max_lifetime", "300s")
	
	viper.SetDefault("gpu_discovery.grpc_address", "localhost:9090")
	viper.SetDefault("verification.default_timeout", "5m")
	viper.SetDefault("verification.max_concurrent_jobs", 5)
	viper.SetDefault("verification.result_retention", "24h")
	
	// Allow environment variable overrides
	viper.AutomaticEnv()
	viper.SetEnvPrefix("VERIFICATION")
	
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; use defaults and environment variables
			return nil
		}
		return err
	}
	
	return nil
}

func createHTTPHandler(logger *logrus.Logger, repositoryManager repositories.RepositoryManager) http.Handler {
	mux := http.NewServeMux()
	
	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if !repositoryManager.IsHealthy(context.Background()) {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Database unhealthy"))
			return
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	// Ready check endpoint
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Ready"))
	})
	
	// Metrics endpoint (placeholder)
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("# Verification service metrics\n"))
		w.Write([]byte("# TODO: Implement Prometheus metrics\n"))
	})
	
	// Version endpoint
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fmt.Sprintf(`{"service":"%s","version":"%s"}`, serviceName, version)))
	})
	
	return mux
}