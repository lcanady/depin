package repositories

import (
	"context"
	"fmt"
	"time"

	"../config"
	"../utils"
)

// repositoryManager implements RepositoryManager
type repositoryManager struct {
	dbManager       *utils.DatabaseManager
	providers       ProviderRepository
	gpus           GPURepository
	healthChecks   HealthCheckRepository
	verifications  VerificationRepository
	usageMetrics   UsageMetricsRepository
}

// NewRepositoryManager creates a new repository manager
func NewRepositoryManager(cfg *config.Config) (RepositoryManager, error) {
	// Create database manager
	dbManager, err := utils.NewDatabaseManager(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create database manager: %w", err)
	}
	
	// Initialize all repositories
	rm := &repositoryManager{
		dbManager:       dbManager,
		providers:       NewProviderRepository(dbManager),
		gpus:           NewGPURepository(dbManager),
		healthChecks:   NewHealthCheckRepository(dbManager),
		verifications:  NewVerificationRepository(dbManager),
		usageMetrics:   NewUsageMetricsRepository(dbManager),
	}
	
	return rm, nil
}

// Repository access methods
func (rm *repositoryManager) Providers() ProviderRepository {
	return rm.providers
}

func (rm *repositoryManager) GPUs() GPURepository {
	return rm.gpus
}

func (rm *repositoryManager) HealthChecks() HealthCheckRepository {
	return rm.healthChecks
}

func (rm *repositoryManager) Verifications() VerificationRepository {
	return rm.verifications
}

func (rm *repositoryManager) UsageMetrics() UsageMetricsRepository {
	return rm.usageMetrics
}

// Close closes all database connections
func (rm *repositoryManager) Close() error {
	return rm.dbManager.Close()
}

// Health monitoring
func (rm *repositoryManager) IsHealthy(ctx context.Context) bool {
	return rm.dbManager.IsHealthy()
}

func (rm *repositoryManager) GetLastHealthCheck() time.Time {
	return rm.dbManager.GetLastHealthCheck()
}

func (rm *repositoryManager) GetStats() map[string]interface{} {
	return rm.dbManager.GetStats()
}

// WithTransaction executes a function within a database transaction
func (rm *repositoryManager) WithTransaction(ctx context.Context, fn func(RepositoryManager) error) error {
	return rm.dbManager.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		// Create a transaction-aware repository manager
		txRM := &transactionRepositoryManager{
			dbManager: rm.dbManager,
			tx:        tx,
		}
		
		// Initialize transaction-aware repositories
		txRM.providers = NewProviderRepositoryWithTx(tx, rm.dbManager.NewCache())
		txRM.gpus = NewGPURepositoryWithTx(tx, rm.dbManager.NewCache())
		txRM.healthChecks = NewHealthCheckRepositoryWithTx(tx)
		txRM.verifications = NewVerificationRepositoryWithTx(tx)
		txRM.usageMetrics = NewUsageMetricsRepositoryWithTx(tx)
		
		return fn(txRM)
	})
}

// transactionRepositoryManager is a repository manager that works within a transaction
type transactionRepositoryManager struct {
	dbManager       *utils.DatabaseManager
	tx              *sqlx.Tx
	providers       ProviderRepository
	gpus           GPURepository
	healthChecks   HealthCheckRepository
	verifications  VerificationRepository
	usageMetrics   UsageMetricsRepository
}

func (trm *transactionRepositoryManager) Providers() ProviderRepository {
	return trm.providers
}

func (trm *transactionRepositoryManager) GPUs() GPURepository {
	return trm.gpus
}

func (trm *transactionRepositoryManager) HealthChecks() HealthCheckRepository {
	return trm.healthChecks
}

func (trm *transactionRepositoryManager) Verifications() VerificationRepository {
	return trm.verifications
}

func (trm *transactionRepositoryManager) UsageMetrics() UsageMetricsRepository {
	return trm.usageMetrics
}

func (trm *transactionRepositoryManager) Close() error {
	// Transaction repositories don't manage the connection
	return nil
}

func (trm *transactionRepositoryManager) IsHealthy(ctx context.Context) bool {
	return trm.dbManager.IsHealthy()
}

func (trm *transactionRepositoryManager) GetLastHealthCheck() time.Time {
	return trm.dbManager.GetLastHealthCheck()
}

func (trm *transactionRepositoryManager) GetStats() map[string]interface{} {
	return trm.dbManager.GetStats()
}

func (trm *transactionRepositoryManager) WithTransaction(ctx context.Context, fn func(RepositoryManager) error) error {
	// Nested transactions are not supported, just execute the function
	return fn(trm)
}