package config

import (
	"fmt"
	"time"
)

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	// Connection settings
	Host     string `yaml:"host" env:"DB_HOST" default:"localhost"`
	Port     int    `yaml:"port" env:"DB_PORT" default:"5432"`
	Database string `yaml:"database" env:"DB_NAME" default:"depin_inventory"`
	User     string `yaml:"user" env:"DB_USER" default:"postgres"`
	Password string `yaml:"password" env:"DB_PASSWORD" default:""`
	SSLMode  string `yaml:"ssl_mode" env:"DB_SSL_MODE" default:"disable"`
	
	// Connection pool settings
	MaxOpenConns    int           `yaml:"max_open_conns" env:"DB_MAX_OPEN_CONNS" default:"25"`
	MaxIdleConns    int           `yaml:"max_idle_conns" env:"DB_MAX_IDLE_CONNS" default:"25"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" env:"DB_CONN_MAX_LIFETIME" default:"5m"`
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time" env:"DB_CONN_MAX_IDLE_TIME" default:"5m"`
	
	// Query settings
	QueryTimeout    time.Duration `yaml:"query_timeout" env:"DB_QUERY_TIMEOUT" default:"30s"`
	
	// Migration settings
	MigrationsPath  string `yaml:"migrations_path" env:"DB_MIGRATIONS_PATH" default:"./database/inventory/migrations"`
	
	// Backup settings
	BackupPath      string        `yaml:"backup_path" env:"DB_BACKUP_PATH" default:"./backups"`
	BackupInterval  time.Duration `yaml:"backup_interval" env:"DB_BACKUP_INTERVAL" default:"24h"`
	BackupRetention time.Duration `yaml:"backup_retention" env:"DB_BACKUP_RETENTION" default:"168h"` // 7 days
	
	// Performance settings
	EnableQueryLog      bool `yaml:"enable_query_log" env:"DB_ENABLE_QUERY_LOG" default:"false"`
	SlowQueryThreshold  time.Duration `yaml:"slow_query_threshold" env:"DB_SLOW_QUERY_THRESHOLD" default:"1s"`
	
	// Health check settings
	HealthCheckInterval time.Duration `yaml:"health_check_interval" env:"DB_HEALTH_CHECK_INTERVAL" default:"30s"`
	
	// Monitoring settings
	EnableMetrics       bool `yaml:"enable_metrics" env:"DB_ENABLE_METRICS" default:"true"`
	MetricsPort         int  `yaml:"metrics_port" env:"DB_METRICS_PORT" default:"9090"`
}

// GetDSN returns the PostgreSQL data source name
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
	)
}

// Validate validates the database configuration
func (c *DatabaseConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("database port must be between 1 and 65535")
	}
	if c.Database == "" {
		return fmt.Errorf("database name is required")
	}
	if c.User == "" {
		return fmt.Errorf("database user is required")
	}
	if c.MaxOpenConns <= 0 {
		return fmt.Errorf("max_open_conns must be positive")
	}
	if c.MaxIdleConns <= 0 {
		return fmt.Errorf("max_idle_conns must be positive")
	}
	if c.MaxIdleConns > c.MaxOpenConns {
		return fmt.Errorf("max_idle_conns cannot be greater than max_open_conns")
	}
	if c.ConnMaxLifetime <= 0 {
		return fmt.Errorf("conn_max_lifetime must be positive")
	}
	if c.QueryTimeout <= 0 {
		return fmt.Errorf("query_timeout must be positive")
	}
	if c.BackupInterval <= 0 {
		return fmt.Errorf("backup_interval must be positive")
	}
	if c.BackupRetention <= 0 {
		return fmt.Errorf("backup_retention must be positive")
	}
	
	return nil
}

// RedisConfig holds Redis configuration for caching
type RedisConfig struct {
	Host     string        `yaml:"host" env:"REDIS_HOST" default:"localhost"`
	Port     int           `yaml:"port" env:"REDIS_PORT" default:"6379"`
	Password string        `yaml:"password" env:"REDIS_PASSWORD" default:""`
	Database int           `yaml:"database" env:"REDIS_DB" default:"0"`
	
	// Pool settings
	MaxRetries         int           `yaml:"max_retries" env:"REDIS_MAX_RETRIES" default:"3"`
	PoolSize           int           `yaml:"pool_size" env:"REDIS_POOL_SIZE" default:"10"`
	PoolTimeout        time.Duration `yaml:"pool_timeout" env:"REDIS_POOL_TIMEOUT" default:"4s"`
	IdleTimeout        time.Duration `yaml:"idle_timeout" env:"REDIS_IDLE_TIMEOUT" default:"5m"`
	IdleCheckFrequency time.Duration `yaml:"idle_check_frequency" env:"REDIS_IDLE_CHECK_FREQ" default:"1m"`
	
	// Cache settings
	DefaultExpiration  time.Duration `yaml:"default_expiration" env:"REDIS_DEFAULT_EXPIRATION" default:"1h"`
	
	// Health settings
	HealthCheckInterval time.Duration `yaml:"health_check_interval" env:"REDIS_HEALTH_CHECK_INTERVAL" default:"30s"`
}

// GetAddress returns the Redis address
func (c *RedisConfig) GetAddress() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// Validate validates the Redis configuration
func (c *RedisConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("redis host is required")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("redis port must be between 1 and 65535")
	}
	if c.Database < 0 {
		return fmt.Errorf("redis database must be non-negative")
	}
	if c.MaxRetries < 0 {
		return fmt.Errorf("max_retries must be non-negative")
	}
	if c.PoolSize <= 0 {
		return fmt.Errorf("pool_size must be positive")
	}
	if c.PoolTimeout <= 0 {
		return fmt.Errorf("pool_timeout must be positive")
	}
	if c.IdleTimeout <= 0 {
		return fmt.Errorf("idle_timeout must be positive")
	}
	
	return nil
}

// Config holds the complete inventory database configuration
type Config struct {
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	
	// Application settings
	Environment string `yaml:"environment" env:"ENVIRONMENT" default:"development"`
	LogLevel    string `yaml:"log_level" env:"LOG_LEVEL" default:"info"`
	
	// Service discovery settings
	ServiceName         string        `yaml:"service_name" env:"SERVICE_NAME" default:"inventory-db"`
	ServicePort         int           `yaml:"service_port" env:"SERVICE_PORT" default:"8080"`
	
	// Resource cleanup settings
	CleanupInterval     time.Duration `yaml:"cleanup_interval" env:"CLEANUP_INTERVAL" default:"1h"`
	StaleResourceTTL    time.Duration `yaml:"stale_resource_ttl" env:"STALE_RESOURCE_TTL" default:"24h"`
	HeartbeatTolerance  time.Duration `yaml:"heartbeat_tolerance" env:"HEARTBEAT_TOLERANCE" default:"5m"`
	
	// Performance settings
	BatchSize           int  `yaml:"batch_size" env:"BATCH_SIZE" default:"100"`
	EnableBatchInsert   bool `yaml:"enable_batch_insert" env:"ENABLE_BATCH_INSERT" default:"true"`
	
	// Security settings
	EnableEncryption    bool   `yaml:"enable_encryption" env:"ENABLE_ENCRYPTION" default:"false"`
	EncryptionKey       string `yaml:"encryption_key" env:"ENCRYPTION_KEY" default:""`
	
	// Monitoring settings
	EnableTracing       bool   `yaml:"enable_tracing" env:"ENABLE_TRACING" default:"true"`
	TracingEndpoint     string `yaml:"tracing_endpoint" env:"TRACING_ENDPOINT" default:""`
}

// Validate validates the complete configuration
func (c *Config) Validate() error {
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("database config: %w", err)
	}
	
	if err := c.Redis.Validate(); err != nil {
		return fmt.Errorf("redis config: %w", err)
	}
	
	if c.ServicePort <= 0 || c.ServicePort > 65535 {
		return fmt.Errorf("service_port must be between 1 and 65535")
	}
	
	if c.CleanupInterval <= 0 {
		return fmt.Errorf("cleanup_interval must be positive")
	}
	
	if c.StaleResourceTTL <= 0 {
		return fmt.Errorf("stale_resource_ttl must be positive")
	}
	
	if c.HeartbeatTolerance <= 0 {
		return fmt.Errorf("heartbeat_tolerance must be positive")
	}
	
	if c.BatchSize <= 0 {
		return fmt.Errorf("batch_size must be positive")
	}
	
	if c.EnableEncryption && c.EncryptionKey == "" {
		return fmt.Errorf("encryption_key is required when encryption is enabled")
	}
	
	return nil
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Database: DatabaseConfig{
			Host:            "localhost",
			Port:            5432,
			Database:        "depin_inventory",
			User:            "postgres",
			Password:        "",
			SSLMode:         "disable",
			MaxOpenConns:    25,
			MaxIdleConns:    25,
			ConnMaxLifetime: 5 * time.Minute,
			ConnMaxIdleTime: 5 * time.Minute,
			QueryTimeout:    30 * time.Second,
			MigrationsPath:  "./database/inventory/migrations",
			BackupPath:      "./backups",
			BackupInterval:  24 * time.Hour,
			BackupRetention: 168 * time.Hour, // 7 days
			EnableQueryLog:  false,
			SlowQueryThreshold: time.Second,
			HealthCheckInterval: 30 * time.Second,
			EnableMetrics:   true,
			MetricsPort:     9090,
		},
		Redis: RedisConfig{
			Host:               "localhost",
			Port:               6379,
			Password:           "",
			Database:           0,
			MaxRetries:         3,
			PoolSize:           10,
			PoolTimeout:        4 * time.Second,
			IdleTimeout:        5 * time.Minute,
			IdleCheckFrequency: time.Minute,
			DefaultExpiration:  time.Hour,
			HealthCheckInterval: 30 * time.Second,
		},
		Environment:         "development",
		LogLevel:            "info",
		ServiceName:         "inventory-db",
		ServicePort:         8080,
		CleanupInterval:     time.Hour,
		StaleResourceTTL:    24 * time.Hour,
		HeartbeatTolerance:  5 * time.Minute,
		BatchSize:           100,
		EnableBatchInsert:   true,
		EnableEncryption:    false,
		EncryptionKey:       "",
		EnableTracing:       true,
		TracingEndpoint:     "",
	}
}