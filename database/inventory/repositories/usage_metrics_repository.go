package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	
	"../../models/resources/common"
	"../utils"
)

// usageMetricsRepository implements UsageMetricsRepository
type usageMetricsRepository struct {
	db *sqlx.DB
}

// NewUsageMetricsRepository creates a new usage metrics repository
func NewUsageMetricsRepository(dm *utils.DatabaseManager) UsageMetricsRepository {
	return &usageMetricsRepository{
		db: dm.GetDB(),
	}
}

// NewUsageMetricsRepositoryWithTx creates a repository with transaction
func NewUsageMetricsRepositoryWithTx(tx *sqlx.Tx) UsageMetricsRepository {
	return &usageMetricsRepository{
		db: tx,
	}
}

// Create creates a new usage metric
func (r *usageMetricsRepository) Create(ctx context.Context, entity *common.Usage) error {
	if entity.ID == uuid.Nil {
		entity.ID = uuid.New()
	}
	entity.CollectionTime = time.Now()
	
	query := `
		INSERT INTO usage_metrics (
			id, resource_id, resource_type, metric_type, value, unit,
			timestamp, collection_time
		) VALUES (
			:id, :resource_id, :resource_type, :metric_type, :value, :unit,
			:timestamp, :collection_time
		)
	`
	
	_, err := r.db.NamedExecContext(ctx, query, entity)
	if err != nil {
		return fmt.Errorf("failed to create usage metric: %w", err)
	}
	
	return nil
}

// GetByID retrieves a usage metric by ID
func (r *usageMetricsRepository) GetByID(ctx context.Context, id uuid.UUID) (*common.Usage, error) {
	var entity common.Usage
	query := `
		SELECT id, resource_id, resource_type, metric_type, value, unit,
		       timestamp, collection_time
		FROM usage_metrics 
		WHERE id = $1
	`
	
	err := r.db.GetContext(ctx, &entity, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("usage metric not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get usage metric: %w", err)
	}
	
	return &entity, nil
}

// RecordUsage records a single usage metric
func (r *usageMetricsRepository) RecordUsage(ctx context.Context, usage *common.Usage) error {
	return r.Create(ctx, usage)
}

// RecordUsageBatch records multiple usage metrics in a single transaction
func (r *usageMetricsRepository) RecordUsageBatch(ctx context.Context, usageMetrics []*common.Usage) error {
	return r.CreateBatch(ctx, usageMetrics)
}

// GetUsageHistory gets usage history for a resource
func (r *usageMetricsRepository) GetUsageHistory(ctx context.Context, resourceID uuid.UUID, metricType string, since time.Time, limit int) ([]common.Usage, error) {
	var entities []common.Usage
	query := `
		SELECT id, resource_id, resource_type, metric_type, value, unit,
		       timestamp, collection_time
		FROM usage_metrics
		WHERE resource_id = $1 AND metric_type = $2 AND timestamp >= $3
		ORDER BY timestamp DESC
		LIMIT $4
	`
	
	err := r.db.SelectContext(ctx, &entities, query, resourceID, metricType, since, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage history: %w", err)
	}
	
	return entities, nil
}

// GetLatestUsage gets the latest usage metric for a resource
func (r *usageMetricsRepository) GetLatestUsage(ctx context.Context, resourceID uuid.UUID, metricType string) (*common.Usage, error) {
	var entity common.Usage
	query := `
		SELECT id, resource_id, resource_type, metric_type, value, unit,
		       timestamp, collection_time
		FROM usage_metrics
		WHERE resource_id = $1 AND metric_type = $2
		ORDER BY timestamp DESC
		LIMIT 1
	`
	
	err := r.db.GetContext(ctx, &entity, query, resourceID, metricType)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no usage metric found for resource %s with type %s", resourceID, metricType)
		}
		return nil, fmt.Errorf("failed to get latest usage: %w", err)
	}
	
	return &entity, nil
}

// GetAverageUsage gets average usage over a time range
func (r *usageMetricsRepository) GetAverageUsage(ctx context.Context, resourceID uuid.UUID, metricType string, timeRange time.Duration) (float64, error) {
	since := time.Now().Add(-timeRange)
	
	var avgUsage float64
	query := `
		SELECT COALESCE(AVG(value), 0) as avg_value
		FROM usage_metrics
		WHERE resource_id = $1 AND metric_type = $2 AND timestamp >= $3
	`
	
	err := r.db.GetContext(ctx, &avgUsage, query, resourceID, metricType, since)
	if err != nil {
		return 0, fmt.Errorf("failed to get average usage: %w", err)
	}
	
	return avgUsage, nil
}

// GetPeakUsage gets peak usage over a time range
func (r *usageMetricsRepository) GetPeakUsage(ctx context.Context, resourceID uuid.UUID, metricType string, timeRange time.Duration) (float64, error) {
	since := time.Now().Add(-timeRange)
	
	var peakUsage float64
	query := `
		SELECT COALESCE(MAX(value), 0) as peak_value
		FROM usage_metrics
		WHERE resource_id = $1 AND metric_type = $2 AND timestamp >= $3
	`
	
	err := r.db.GetContext(ctx, &peakUsage, query, resourceID, metricType, since)
	if err != nil {
		return 0, fmt.Errorf("failed to get peak usage: %w", err)
	}
	
	return peakUsage, nil
}

// GetUsageStatistics gets comprehensive usage statistics
func (r *usageMetricsRepository) GetUsageStatistics(ctx context.Context, resourceIDs []uuid.UUID, timeRange time.Duration) (map[string]interface{}, error) {
	if len(resourceIDs) == 0 {
		return make(map[string]interface{}), nil
	}
	
	since := time.Now().Add(-timeRange)
	stats := make(map[string]interface{})
	
	// Overall statistics
	var overallStats struct {
		TotalMetrics int64   `db:"total_metrics"`
		AvgValue     float64 `db:"avg_value"`
		MaxValue     float64 `db:"max_value"`
		MinValue     float64 `db:"min_value"`
	}
	
	query := `
		SELECT 
			COUNT(*) as total_metrics,
			COALESCE(AVG(value), 0) as avg_value,
			COALESCE(MAX(value), 0) as max_value,
			COALESCE(MIN(value), 0) as min_value
		FROM usage_metrics
		WHERE resource_id = ANY($1) AND timestamp >= $2
	`
	
	err := r.db.GetContext(ctx, &overallStats, query, pq.Array(resourceIDs), since)
	if err != nil {
		return nil, fmt.Errorf("failed to get overall stats: %w", err)
	}
	
	stats["overall"] = overallStats
	
	// Statistics by metric type
	var typeStats []struct {
		MetricType   string  `db:"metric_type"`
		Count        int64   `db:"count"`
		AvgValue     float64 `db:"avg_value"`
		MaxValue     float64 `db:"max_value"`
	}
	
	query = `
		SELECT 
			metric_type,
			COUNT(*) as count,
			COALESCE(AVG(value), 0) as avg_value,
			COALESCE(MAX(value), 0) as max_value
		FROM usage_metrics
		WHERE resource_id = ANY($1) AND timestamp >= $2
		GROUP BY metric_type
		ORDER BY metric_type
	`
	
	err = r.db.SelectContext(ctx, &typeStats, query, pq.Array(resourceIDs), since)
	if err != nil {
		return nil, fmt.Errorf("failed to get type stats: %w", err)
	}
	
	stats["by_type"] = typeStats
	
	// Statistics by resource
	var resourceStats []struct {
		ResourceID uuid.UUID `db:"resource_id"`
		Count      int64     `db:"count"`
		AvgValue   float64   `db:"avg_value"`
	}
	
	query = `
		SELECT 
			resource_id,
			COUNT(*) as count,
			COALESCE(AVG(value), 0) as avg_value
		FROM usage_metrics
		WHERE resource_id = ANY($1) AND timestamp >= $2
		GROUP BY resource_id
		ORDER BY count DESC
	`
	
	err = r.db.SelectContext(ctx, &resourceStats, query, pq.Array(resourceIDs), since)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource stats: %w", err)
	}
	
	stats["by_resource"] = resourceStats
	
	return stats, nil
}

// CleanupOldUsageMetrics removes old usage metrics
func (r *usageMetricsRepository) CleanupOldUsageMetrics(ctx context.Context, olderThan time.Time) (int64, error) {
	query := `DELETE FROM usage_metrics WHERE collection_time < $1`
	
	result, err := r.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old usage metrics: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}
	
	return rowsAffected, nil
}

// Update updates a usage metric (rarely used)
func (r *usageMetricsRepository) Update(ctx context.Context, entity *common.Usage) error {
	query := `
		UPDATE usage_metrics SET
			metric_type = :metric_type, value = :value, unit = :unit,
			timestamp = :timestamp, collection_time = :collection_time
		WHERE id = :id
	`
	
	result, err := r.db.NamedExecContext(ctx, query, entity)
	if err != nil {
		return fmt.Errorf("failed to update usage metric: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("usage metric not found: %s", entity.ID)
	}
	
	return nil
}

// Delete deletes a usage metric
func (r *usageMetricsRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM usage_metrics WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete usage metric: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("usage metric not found: %s", id)
	}
	
	return nil
}

// CreateBatch creates multiple usage metrics
func (r *usageMetricsRepository) CreateBatch(ctx context.Context, entities []*common.Usage) error {
	if len(entities) == 0 {
		return nil
	}
	
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	query := `
		INSERT INTO usage_metrics (
			id, resource_id, resource_type, metric_type, value, unit,
			timestamp, collection_time
		) VALUES (
			:id, :resource_id, :resource_type, :metric_type, :value, :unit,
			:timestamp, :collection_time
		)
	`
	
	now := time.Now()
	for _, entity := range entities {
		if entity.ID == uuid.Nil {
			entity.ID = uuid.New()
		}
		entity.CollectionTime = now
		
		_, err := tx.NamedExecContext(ctx, query, entity)
		if err != nil {
			return fmt.Errorf("failed to insert usage metric in batch: %w", err)
		}
	}
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}

// UpdateBatch updates multiple usage metrics
func (r *usageMetricsRepository) UpdateBatch(ctx context.Context, entities []*common.Usage) error {
	if len(entities) == 0 {
		return nil
	}
	
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	query := `
		UPDATE usage_metrics SET
			metric_type = :metric_type, value = :value, unit = :unit,
			timestamp = :timestamp, collection_time = :collection_time
		WHERE id = :id
	`
	
	for _, entity := range entities {
		result, err := tx.NamedExecContext(ctx, query, entity)
		if err != nil {
			return fmt.Errorf("failed to update usage metric in batch: %w", err)
		}
		
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get affected rows: %w", err)
		}
		
		if rowsAffected == 0 {
			return fmt.Errorf("usage metric not found: %s", entity.ID)
		}
	}
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}

// DeleteBatch deletes multiple usage metrics
func (r *usageMetricsRepository) DeleteBatch(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	
	query := `DELETE FROM usage_metrics WHERE id = ANY($1)`
	
	result, err := r.db.ExecContext(ctx, query, pq.Array(ids))
	if err != nil {
		return fmt.Errorf("failed to delete usage metrics: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	
	if rowsAffected != int64(len(ids)) {
		return fmt.Errorf("expected to delete %d usage metrics, but deleted %d", len(ids), rowsAffected)
	}
	
	return nil
}

// Count returns the total number of usage metrics
func (r *usageMetricsRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM usage_metrics`
	
	err := r.db.GetContext(ctx, &count, query)
	if err != nil {
		return 0, fmt.Errorf("failed to count usage metrics: %w", err)
	}
	
	return count, nil
}

// Health check methods
func (r *usageMetricsRepository) IsHealthy(ctx context.Context) bool {
	err := r.db.PingContext(ctx)
	return err == nil
}

func (r *usageMetricsRepository) GetLastHealthCheck() time.Time {
	return time.Now()
}

func (r *usageMetricsRepository) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"repository": "usage_metrics",
	}
}