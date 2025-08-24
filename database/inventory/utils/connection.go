package utils

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	
	"../config"
)

// DatabaseManager manages database connections and health
type DatabaseManager struct {
	config      *config.Config
	db          *sqlx.DB
	redis       *redis.Client
	healthMutex sync.RWMutex
	isHealthy   bool
	lastCheck   time.Time
}

// NewDatabaseManager creates a new database manager
func NewDatabaseManager(cfg *config.Config) (*DatabaseManager, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	dm := &DatabaseManager{
		config:    cfg,
		isHealthy: false,
	}

	// Initialize PostgreSQL connection
	if err := dm.initPostgreSQL(); err != nil {
		return nil, fmt.Errorf("failed to initialize postgresql: %w", err)
	}

	// Initialize Redis connection
	if err := dm.initRedis(); err != nil {
		log.Printf("Warning: Failed to initialize Redis: %v", err)
		// Redis is optional for caching, don't fail if it's unavailable
	}

	// Start health monitoring
	go dm.startHealthMonitoring()

	return dm, nil
}

// initPostgreSQL initializes PostgreSQL connection
func (dm *DatabaseManager) initPostgreSQL() error {
	dsn := dm.config.Database.GetDSN()
	
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to postgresql: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(dm.config.Database.MaxOpenConns)
	db.SetMaxIdleConns(dm.config.Database.MaxIdleConns)
	db.SetConnMaxLifetime(dm.config.Database.ConnMaxLifetime)
	db.SetConnMaxIdleTime(dm.config.Database.ConnMaxIdleTime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), dm.config.Database.QueryTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping postgresql: %w", err)
	}

	dm.db = db
	log.Printf("PostgreSQL connection established successfully")
	return nil
}

// initRedis initializes Redis connection
func (dm *DatabaseManager) initRedis() error {
	rdb := redis.NewClient(&redis.Options{
		Addr:               dm.config.Redis.GetAddress(),
		Password:           dm.config.Redis.Password,
		DB:                 dm.config.Redis.Database,
		MaxRetries:         dm.config.Redis.MaxRetries,
		PoolSize:           dm.config.Redis.PoolSize,
		PoolTimeout:        dm.config.Redis.PoolTimeout,
		IdleTimeout:        dm.config.Redis.IdleTimeout,
		IdleCheckFrequency: dm.config.Redis.IdleCheckFrequency,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to ping redis: %w", err)
	}

	dm.redis = rdb
	log.Printf("Redis connection established successfully")
	return nil
}

// GetDB returns the PostgreSQL database connection
func (dm *DatabaseManager) GetDB() *sqlx.DB {
	return dm.db
}

// GetRedis returns the Redis client
func (dm *DatabaseManager) GetRedis() *redis.Client {
	return dm.redis
}

// IsHealthy returns the health status of the database connections
func (dm *DatabaseManager) IsHealthy() bool {
	dm.healthMutex.RLock()
	defer dm.healthMutex.RUnlock()
	return dm.isHealthy
}

// GetLastHealthCheck returns the timestamp of the last health check
func (dm *DatabaseManager) GetLastHealthCheck() time.Time {
	dm.healthMutex.RLock()
	defer dm.healthMutex.RUnlock()
	return dm.lastCheck
}

// startHealthMonitoring starts the health monitoring goroutine
func (dm *DatabaseManager) startHealthMonitoring() {
	ticker := time.NewTicker(dm.config.Database.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			dm.performHealthCheck()
		}
	}
}

// performHealthCheck performs a health check on database connections
func (dm *DatabaseManager) performHealthCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	healthy := true
	
	// Check PostgreSQL
	if dm.db != nil {
		if err := dm.db.PingContext(ctx); err != nil {
			log.Printf("PostgreSQL health check failed: %v", err)
			healthy = false
		}
	} else {
		healthy = false
	}

	// Check Redis (optional)
	if dm.redis != nil {
		if err := dm.redis.Ping(ctx).Err(); err != nil {
			log.Printf("Redis health check failed: %v", err)
			// Redis failure doesn't make the service unhealthy since it's optional
		}
	}

	dm.healthMutex.Lock()
	dm.isHealthy = healthy
	dm.lastCheck = time.Now()
	dm.healthMutex.Unlock()

	if !healthy {
		log.Printf("Database health check failed at %v", dm.lastCheck)
	}
}

// Close closes all database connections
func (dm *DatabaseManager) Close() error {
	var errs []error

	if dm.db != nil {
		if err := dm.db.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close postgresql: %w", err))
		}
	}

	if dm.redis != nil {
		if err := dm.redis.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close redis: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing connections: %v", errs)
	}

	return nil
}

// WithTransaction executes a function within a database transaction
func (dm *DatabaseManager) WithTransaction(ctx context.Context, fn func(*sqlx.Tx) error) error {
	tx, err := dm.db.BeginTxx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction error: %v, rollback error: %w", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetStats returns database connection statistics
func (dm *DatabaseManager) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})

	if dm.db != nil {
		dbStats := dm.db.Stats()
		stats["postgresql"] = map[string]interface{}{
			"open_connections":     dbStats.OpenConnections,
			"in_use":              dbStats.InUse,
			"idle":                dbStats.Idle,
			"wait_count":          dbStats.WaitCount,
			"wait_duration":       dbStats.WaitDuration.String(),
			"max_idle_closed":     dbStats.MaxIdleClosed,
			"max_idle_time_closed": dbStats.MaxIdleTimeClosed,
			"max_lifetime_closed": dbStats.MaxLifetimeClosed,
		}
	}

	if dm.redis != nil {
		poolStats := dm.redis.PoolStats()
		stats["redis"] = map[string]interface{}{
			"hits":       poolStats.Hits,
			"misses":     poolStats.Misses,
			"timeouts":   poolStats.Timeouts,
			"total_conns": poolStats.TotalConns,
			"idle_conns":  poolStats.IdleConns,
			"stale_conns": poolStats.StaleConns,
		}
	}

	dm.healthMutex.RLock()
	stats["health"] = map[string]interface{}{
		"is_healthy":     dm.isHealthy,
		"last_check":     dm.lastCheck.Format(time.RFC3339),
	}
	dm.healthMutex.RUnlock()

	return stats
}

// Cache provides a simple caching interface using Redis
type Cache struct {
	client *redis.Client
	config *config.RedisConfig
}

// NewCache creates a new cache instance
func (dm *DatabaseManager) NewCache() *Cache {
	if dm.redis == nil {
		return nil
	}
	
	return &Cache{
		client: dm.redis,
		config: &dm.config.Redis,
	}
}

// Set stores a value in cache with default expiration
func (c *Cache) Set(ctx context.Context, key string, value interface{}) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("cache not available")
	}
	
	return c.client.Set(ctx, key, value, c.config.DefaultExpiration).Err()
}

// SetWithTTL stores a value in cache with custom expiration
func (c *Cache) SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("cache not available")
	}
	
	return c.client.Set(ctx, key, value, ttl).Err()
}

// Get retrieves a value from cache
func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	if c == nil || c.client == nil {
		return "", fmt.Errorf("cache not available")
	}
	
	return c.client.Get(ctx, key).Result()
}

// Delete removes a value from cache
func (c *Cache) Delete(ctx context.Context, key string) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("cache not available")
	}
	
	return c.client.Del(ctx, key).Err()
}

// Exists checks if a key exists in cache
func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	if c == nil || c.client == nil {
		return false, fmt.Errorf("cache not available")
	}
	
	result := c.client.Exists(ctx, key)
	if result.Err() != nil {
		return false, result.Err()
	}
	
	return result.Val() > 0, nil
}