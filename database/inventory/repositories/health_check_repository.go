package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	
	"../../models/resources/common"
	"../utils"
)

// healthCheckRepository implements HealthCheckRepository
type healthCheckRepository struct {
	db *sqlx.DB
}

// NewHealthCheckRepository creates a new health check repository
func NewHealthCheckRepository(dm *utils.DatabaseManager) HealthCheckRepository {
	return &healthCheckRepository{
		db: dm.GetDB(),
	}
}

// NewHealthCheckRepositoryWithTx creates a repository with transaction
func NewHealthCheckRepositoryWithTx(tx *sqlx.Tx) HealthCheckRepository {
	return &healthCheckRepository{
		db: tx,
	}
}

// Create creates a new health check
func (r *healthCheckRepository) Create(ctx context.Context, entity *common.HealthCheck) error {
	if entity.ID == uuid.Nil {
		entity.ID = uuid.New()
	}
	entity.CheckedAt = time.Now()
	
	query := `
		INSERT INTO health_checks (
			id, resource_id, resource_type, check_type, status, message,
			response_time_ms, metadata, checked_at
		) VALUES (
			:id, :resource_id, :resource_type, :check_type, :status, :message,
			:response_time_ms, :metadata, :checked_at
		)
	`
	
	_, err := r.db.NamedExecContext(ctx, query, entity)
	if err != nil {
		return fmt.Errorf("failed to create health check: %w", err)
	}
	
	return nil
}

// GetByID retrieves a health check by ID
func (r *healthCheckRepository) GetByID(ctx context.Context, id uuid.UUID) (*common.HealthCheck, error) {
	var entity common.HealthCheck
	query := `
		SELECT id, resource_id, resource_type, check_type, status, message,
		       response_time_ms, metadata, checked_at
		FROM health_checks 
		WHERE id = $1
	`
	
	err := r.db.GetContext(ctx, &entity, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("health check not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get health check: %w", err)
	}
	
	return &entity, nil
}

// CreateHealthCheck creates a health check record
func (r *healthCheckRepository) CreateHealthCheck(ctx context.Context, healthCheck *common.HealthCheck) error {
	return r.Create(ctx, healthCheck)
}

// GetLatestHealthCheck gets the latest health check for a resource
func (r *healthCheckRepository) GetLatestHealthCheck(ctx context.Context, resourceID uuid.UUID, checkType string) (*common.HealthCheck, error) {
	var entity common.HealthCheck
	query := `
		SELECT id, resource_id, resource_type, check_type, status, message,
		       response_time_ms, metadata, checked_at
		FROM health_checks
		WHERE resource_id = $1 AND check_type = $2
		ORDER BY checked_at DESC
		LIMIT 1
	`
	
	err := r.db.GetContext(ctx, &entity, query, resourceID, checkType)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no health check found for resource %s with type %s", resourceID, checkType)
		}
		return nil, fmt.Errorf("failed to get latest health check: %w", err)
	}
	
	return &entity, nil
}

// GetHealthCheckHistory gets health check history for a resource
func (r *healthCheckRepository) GetHealthCheckHistory(ctx context.Context, resourceID uuid.UUID, since time.Time, limit int) ([]common.HealthCheck, error) {
	var entities []common.HealthCheck
	query := `
		SELECT id, resource_id, resource_type, check_type, status, message,
		       response_time_ms, metadata, checked_at
		FROM health_checks
		WHERE resource_id = $1 AND checked_at >= $2
		ORDER BY checked_at DESC
		LIMIT $3
	`
	
	err := r.db.SelectContext(ctx, &entities, query, resourceID, since, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get health check history: %w", err)
	}
	
	return entities, nil
}

// ListFailedHealthChecks lists failed health checks
func (r *healthCheckRepository) ListFailedHealthChecks(ctx context.Context, since time.Time, limit int) ([]common.HealthCheck, error) {
	var entities []common.HealthCheck
	query := `
		SELECT id, resource_id, resource_type, check_type, status, message,
		       response_time_ms, metadata, checked_at
		FROM health_checks
		WHERE status != 'healthy' AND checked_at >= $1
		ORDER BY checked_at DESC
		LIMIT $2
	`
	
	err := r.db.SelectContext(ctx, &entities, query, since, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list failed health checks: %w", err)
	}
	
	return entities, nil
}

// CleanupOldHealthChecks removes old health check records
func (r *healthCheckRepository) CleanupOldHealthChecks(ctx context.Context, olderThan time.Time) (int64, error) {
	query := `DELETE FROM health_checks WHERE checked_at < $1`
	
	result, err := r.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old health checks: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}
	
	return rowsAffected, nil
}

// Update updates a health check (not typically used)
func (r *healthCheckRepository) Update(ctx context.Context, entity *common.HealthCheck) error {
	query := `
		UPDATE health_checks SET
			status = :status, message = :message, response_time_ms = :response_time_ms,
			metadata = :metadata, checked_at = :checked_at
		WHERE id = :id
	`
	
	result, err := r.db.NamedExecContext(ctx, query, entity)
	if err != nil {
		return fmt.Errorf("failed to update health check: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("health check not found: %s", entity.ID)
	}
	
	return nil
}

// Delete deletes a health check
func (r *healthCheckRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM health_checks WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete health check: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("health check not found: %s", id)
	}
	
	return nil
}

// CreateBatch creates multiple health checks
func (r *healthCheckRepository) CreateBatch(ctx context.Context, entities []*common.HealthCheck) error {
	if len(entities) == 0 {
		return nil
	}
	
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	query := `
		INSERT INTO health_checks (
			id, resource_id, resource_type, check_type, status, message,
			response_time_ms, metadata, checked_at
		) VALUES (
			:id, :resource_id, :resource_type, :check_type, :status, :message,
			:response_time_ms, :metadata, :checked_at
		)
	`
	
	now := time.Now()
	for _, entity := range entities {
		if entity.ID == uuid.Nil {
			entity.ID = uuid.New()
		}
		entity.CheckedAt = now
		
		_, err := tx.NamedExecContext(ctx, query, entity)
		if err != nil {
			return fmt.Errorf("failed to insert health check in batch: %w", err)
		}
	}
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}

// UpdateBatch updates multiple health checks
func (r *healthCheckRepository) UpdateBatch(ctx context.Context, entities []*common.HealthCheck) error {
	if len(entities) == 0 {
		return nil
	}
	
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	query := `
		UPDATE health_checks SET
			status = :status, message = :message, response_time_ms = :response_time_ms,
			metadata = :metadata, checked_at = :checked_at
		WHERE id = :id
	`
	
	for _, entity := range entities {
		result, err := tx.NamedExecContext(ctx, query, entity)
		if err != nil {
			return fmt.Errorf("failed to update health check in batch: %w", err)
		}
		
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get affected rows: %w", err)
		}
		
		if rowsAffected == 0 {
			return fmt.Errorf("health check not found: %s", entity.ID)
		}
	}
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}

// DeleteBatch deletes multiple health checks
func (r *healthCheckRepository) DeleteBatch(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	
	query := `DELETE FROM health_checks WHERE id = ANY($1)`
	
	result, err := r.db.ExecContext(ctx, query, pq.Array(ids))
	if err != nil {
		return fmt.Errorf("failed to delete health checks: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	
	if rowsAffected != int64(len(ids)) {
		return fmt.Errorf("expected to delete %d health checks, but deleted %d", len(ids), rowsAffected)
	}
	
	return nil
}

// Count returns the total number of health checks
func (r *healthCheckRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM health_checks`
	
	err := r.db.GetContext(ctx, &count, query)
	if err != nil {
		return 0, fmt.Errorf("failed to count health checks: %w", err)
	}
	
	return count, nil
}

// Health check methods
func (r *healthCheckRepository) IsHealthy(ctx context.Context) bool {
	err := r.db.PingContext(ctx)
	return err == nil
}

func (r *healthCheckRepository) GetLastHealthCheck() time.Time {
	return time.Now()
}

func (r *healthCheckRepository) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"repository": "health_checks",
	}
}