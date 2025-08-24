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

	"github.com/lcanady/depin/monitoring/heartbeat/internal/monitor"
	"github.com/lcanady/depin/monitoring/heartbeat/internal/service"
	pb "github.com/lcanady/depin/monitoring/heartbeat/proto"
	"github.com/lcanady/depin/database/inventory/repositories"
	"github.com/lcanady/depin/database/inventory/config"
)

const (
	serviceName = "heartbeat-service"
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
	}).Info("Starting heartbeat monitoring service")
	
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
	
	// Create heartbeat monitor
	monitorConfig := &monitor.MonitorConfig{
		DefaultHeartbeatInterval: viper.GetDuration("monitoring.heartbeat_interval"),
		HeartbeatTimeout:        viper.GetDuration("monitoring.heartbeat_timeout"),
		MaxMissedHeartbeats:     viper.GetInt("monitoring.max_missed_heartbeats"),
		HealthCheckInterval:     viper.GetDuration("monitoring.health_check_interval"),
		CleanupInterval:         viper.GetDuration("monitoring.cleanup_interval"),
		EventBufferSize:         viper.GetInt("monitoring.event_buffer_size"),
		MaxEventStreams:         viper.GetInt("monitoring.max_event_streams"),
	}
	
	heartbeatMonitor := monitor.NewHeartbeatMonitor(logger, repositoryManager, monitorConfig)
	
	// Start heartbeat monitor
	ctx := context.Background()
	if err := heartbeatMonitor.Start(ctx); err != nil {
		logger.WithError(err).Fatal("Failed to start heartbeat monitor")
	}
	defer heartbeatMonitor.Stop()
	
	// Create heartbeat service
	serviceConfig := &service.HeartbeatServiceConfig{
		MaxEventStreams:        viper.GetInt("service.max_event_streams"),
		EventBufferSize:        viper.GetInt("service.event_buffer_size"),
		MaxAvailabilityStreams: viper.GetInt("service.max_availability_streams"),
		AvailabilityBufferSize: viper.GetInt("service.availability_buffer_size"),
		DefaultStreamTimeout:   viper.GetDuration("service.default_stream_timeout"),
	}
	
	heartbeatService := service.NewHeartbeatService(
		logger,
		heartbeatMonitor,
		repositoryManager,
		serviceConfig,
	)
	
	// Create gRPC server
	grpcServer := grpc.NewServer()
	
	// Register services
	pb.RegisterHeartbeatServiceServer(grpcServer, heartbeatService)
	
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
		Handler:        createHTTPHandler(logger, repositoryManager, heartbeatMonitor),
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
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	
	// Stop HTTP server
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.WithError(err).Error("Failed to shutdown HTTP server gracefully")
	}
	
	// Stop gRPC server
	grpcServer.GracefulStop()
	
	logger.Info("Heartbeat monitoring service stopped")
}

func loadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/heartbeat-service")
	
	// Set defaults
	viper.SetDefault("server.grpc_port", 8082)
	viper.SetDefault("server.http_port", 8083)
	viper.SetDefault("server.shutdown_timeout", "30s")
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")
	
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.database", "depin_heartbeat")
	viper.SetDefault("database.username", "heartbeat_user")
	viper.SetDefault("database.ssl_mode", "prefer")
	viper.SetDefault("database.max_open_connections", 25)
	viper.SetDefault("database.max_idle_connections", 10)
	viper.SetDefault("database.connection_max_lifetime", "300s")
	
	viper.SetDefault("monitoring.heartbeat_interval", "30s")
	viper.SetDefault("monitoring.heartbeat_timeout", "60s")
	viper.SetDefault("monitoring.max_missed_heartbeats", 3)
	viper.SetDefault("monitoring.health_check_interval", "10s")
	viper.SetDefault("monitoring.cleanup_interval", "5m")
	viper.SetDefault("monitoring.event_buffer_size", 1000)
	viper.SetDefault("monitoring.max_event_streams", 100)
	
	viper.SetDefault("service.max_event_streams", 100)
	viper.SetDefault("service.event_buffer_size", 1000)
	viper.SetDefault("service.max_availability_streams", 50)
	viper.SetDefault("service.availability_buffer_size", 500)
	viper.SetDefault("service.default_stream_timeout", "1h")
	
	// Allow environment variable overrides
	viper.AutomaticEnv()
	viper.SetEnvPrefix("HEARTBEAT")
	
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; use defaults and environment variables
			return nil
		}
		return err
	}
	
	return nil
}

func createHTTPHandler(logger *logrus.Logger, repositoryManager repositories.RepositoryManager, monitor *monitor.HeartbeatMonitor) http.Handler {
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
		w.Write([]byte("# Heartbeat monitoring service metrics\n"))
		w.Write([]byte("# TODO: Implement Prometheus metrics\n"))
	})
	
	// Version endpoint
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fmt.Sprintf(`{"service":"%s","version":"%s"}`, serviceName, version)))
	})
	
	// System health endpoint (JSON)
	mux.HandleFunc("/system-health", func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		systemHealth, err := monitor.GetSystemHealth(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("Error getting system health: %v", err)))
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		// In a real implementation, you would marshal the systemHealth to JSON
		w.Write([]byte(`{"status":"healthy","message":"System health endpoint - JSON marshaling not implemented"}`))
	})
	
	return mux
}