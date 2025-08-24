package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	
	"../../models/resources/common"
	"../../models/resources/provider"
	"../utils"
)

// providerRepository implements ProviderRepository
type providerRepository struct {
	db    *sqlx.DB
	cache *utils.Cache
}

// NewProviderRepository creates a new provider repository
func NewProviderRepository(dm *utils.DatabaseManager) ProviderRepository {
	return &providerRepository{
		db:    dm.GetDB(),
		cache: dm.NewCache(),
	}
}

// Create creates a new provider resource
func (r *providerRepository) Create(ctx context.Context, entity *provider.ProviderResource) error {
	if entity.ID == uuid.Nil {
		entity.ID = uuid.New()
	}
	entity.CreatedAt = time.Now()
	entity.UpdatedAt = time.Now()
	entity.Version = 1
	
	query := `
		INSERT INTO providers (
			id, name, email, organization, status, api_key_hash, public_key,
			endpoints, metadata, resource_summary, last_seen, last_heartbeat,
			heartbeat_interval_seconds, health_status, consecutive_failures,
			registered_at, activated_at, suspended_at, reputation, reliability_score,
			avg_response_time_ms, uptime_percentage, total_allocations,
			successful_allocations, failed_allocations, current_allocations,
			created_at, updated_at, version
		) VALUES (
			:id, :name, :email, :organization, :status, :api_key_hash, :public_key,
			:endpoints, :metadata, :resource_summary, :last_seen, :last_heartbeat,
			:heartbeat_interval_seconds, :health_status, :consecutive_failures,
			:registered_at, :activated_at, :suspended_at, :reputation, :reliability_score,
			:avg_response_time_ms, :uptime_percentage, :total_allocations,
			:successful_allocations, :failed_allocations, :current_allocations,
			:created_at, :updated_at, :version
		)
	`
	
	_, err := r.db.NamedExecContext(ctx, query, entity)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}
	
	// Invalidate cache
	r.invalidateCache(entity.ID)
	
	return nil
}

// GetByID retrieves a provider by ID
func (r *providerRepository) GetByID(ctx context.Context, id uuid.UUID) (*provider.ProviderResource, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("provider:%s", id)
	if r.cache != nil {
		if cached, err := r.cache.Get(ctx, cacheKey); err == nil {
			if cachedProvider, ok := cached.(*provider.ProviderResource); ok {
				return cachedProvider, nil
			}
		}
	}
	
	var entity provider.ProviderResource
	query := `
		SELECT 
			id, name, email, organization, status, api_key_hash, public_key,
			endpoints, metadata, resource_summary, last_seen, last_heartbeat,
			heartbeat_interval_seconds, health_status, consecutive_failures,
			registered_at, activated_at, suspended_at, reputation, reliability_score,
			avg_response_time_ms, uptime_percentage, total_allocations,
			successful_allocations, failed_allocations, current_allocations,
			created_at, updated_at, version
		FROM providers 
		WHERE id = $1
	`
	
	err := r.db.GetContext(ctx, &entity, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("provider not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}
	
	// Cache the result
	if r.cache != nil {
		r.cache.SetWithTTL(ctx, cacheKey, &entity, 15*time.Minute)
	}
	
	return &entity, nil
}

// GetByEmail retrieves a provider by email
func (r *providerRepository) GetByEmail(ctx context.Context, email string) (*provider.ProviderResource, error) {
	var entity provider.ProviderResource
	query := `
		SELECT 
			id, name, email, organization, status, api_key_hash, public_key,
			endpoints, metadata, resource_summary, last_seen, last_heartbeat,
			heartbeat_interval_seconds, health_status, consecutive_failures,
			registered_at, activated_at, suspended_at, reputation, reliability_score,
			avg_response_time_ms, uptime_percentage, total_allocations,
			successful_allocations, failed_allocations, current_allocations,
			created_at, updated_at, version
		FROM providers 
		WHERE email = $1
	`
	
	err := r.db.GetContext(ctx, &entity, query, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("provider not found with email: %s", email)
		}
		return nil, fmt.Errorf("failed to get provider by email: %w", err)
	}
	
	return &entity, nil
}

// GetByApiKeyHash retrieves a provider by API key hash
func (r *providerRepository) GetByApiKeyHash(ctx context.Context, apiKeyHash string) (*provider.ProviderResource, error) {
	var entity provider.ProviderResource
	query := `
		SELECT 
			id, name, email, organization, status, api_key_hash, public_key,
			endpoints, metadata, resource_summary, last_seen, last_heartbeat,
			heartbeat_interval_seconds, health_status, consecutive_failures,
			registered_at, activated_at, suspended_at, reputation, reliability_score,
			avg_response_time_ms, uptime_percentage, total_allocations,
			successful_allocations, failed_allocations, current_allocations,
			created_at, updated_at, version
		FROM providers 
		WHERE api_key_hash = $1
	`
	
	err := r.db.GetContext(ctx, &entity, query, apiKeyHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("provider not found with API key hash")
		}
		return nil, fmt.Errorf("failed to get provider by API key hash: %w", err)
	}
	
	return &entity, nil
}

// Update updates a provider resource
func (r *providerRepository) Update(ctx context.Context, entity *provider.ProviderResource) error {
	entity.UpdatedAt = time.Now()
	entity.Version++
	
	query := `
		UPDATE providers SET
			name = :name, email = :email, organization = :organization, status = :status,
			api_key_hash = :api_key_hash, public_key = :public_key, endpoints = :endpoints,
			metadata = :metadata, resource_summary = :resource_summary, last_seen = :last_seen,
			last_heartbeat = :last_heartbeat, heartbeat_interval_seconds = :heartbeat_interval_seconds,
			health_status = :health_status, consecutive_failures = :consecutive_failures,
			registered_at = :registered_at, activated_at = :activated_at, suspended_at = :suspended_at,
			reputation = :reputation, reliability_score = :reliability_score,
			avg_response_time_ms = :avg_response_time_ms, uptime_percentage = :uptime_percentage,
			total_allocations = :total_allocations, successful_allocations = :successful_allocations,
			failed_allocations = :failed_allocations, current_allocations = :current_allocations,
			updated_at = :updated_at, version = :version
		WHERE id = :id AND version = :version - 1
	`
	
	result, err := r.db.NamedExecContext(ctx, query, entity)
	if err != nil {
		return fmt.Errorf("failed to update provider: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("provider not found or version conflict")
	}
	
	// Invalidate cache
	r.invalidateCache(entity.ID)
	
	return nil
}

// Delete deletes a provider
func (r *providerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM providers WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete provider: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("provider not found: %s", id)
	}
	
	// Invalidate cache
	r.invalidateCache(id)
	
	return nil
}

// CreateBatch creates multiple providers in a single transaction
func (r *providerRepository) CreateBatch(ctx context.Context, entities []*provider.ProviderResource) error {
	if len(entities) == 0 {
		return nil
	}
	
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	query := `
		INSERT INTO providers (
			id, name, email, organization, status, api_key_hash, public_key,
			endpoints, metadata, resource_summary, created_at, updated_at, version
		) VALUES (
			:id, :name, :email, :organization, :status, :api_key_hash, :public_key,
			:endpoints, :metadata, :resource_summary, :created_at, :updated_at, :version
		)
	`
	
	now := time.Now()
	for _, entity := range entities {
		if entity.ID == uuid.Nil {
			entity.ID = uuid.New()
		}
		entity.CreatedAt = now
		entity.UpdatedAt = now
		entity.Version = 1
		
		_, err := tx.NamedExecContext(ctx, query, entity)
		if err != nil {
			return fmt.Errorf("failed to insert provider in batch: %w", err)
		}
	}
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}

// UpdateBatch updates multiple providers
func (r *providerRepository) UpdateBatch(ctx context.Context, entities []*provider.ProviderResource) error {
	if len(entities) == 0 {
		return nil
	}
	
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	query := `
		UPDATE providers SET
			name = :name, status = :status, health_status = :health_status,
			last_seen = :last_seen, last_heartbeat = :last_heartbeat,
			resource_summary = :resource_summary, updated_at = :updated_at,
			version = :version
		WHERE id = :id AND version = :version - 1
	`
	
	now := time.Now()
	for _, entity := range entities {
		entity.UpdatedAt = now
		entity.Version++
		
		result, err := tx.NamedExecContext(ctx, query, entity)
		if err != nil {
			return fmt.Errorf("failed to update provider in batch: %w", err)
		}
		
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get affected rows: %w", err)
		}
		
		if rowsAffected == 0 {
			return fmt.Errorf("provider not found or version conflict: %s", entity.ID)
		}
	}
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}

// DeleteBatch deletes multiple providers
func (r *providerRepository) DeleteBatch(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	
	query := `DELETE FROM providers WHERE id = ANY($1)`
	
	result, err := r.db.ExecContext(ctx, query, pq.Array(ids))
	if err != nil {
		return fmt.Errorf("failed to delete providers: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	
	if rowsAffected != int64(len(ids)) {
		return fmt.Errorf("expected to delete %d providers, but deleted %d", len(ids), rowsAffected)
	}
	
	// Invalidate cache for all deleted providers
	for _, id := range ids {
		r.invalidateCache(id)
	}
	
	return nil
}

// Count returns the total number of providers
func (r *providerRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM providers`
	
	err := r.db.GetContext(ctx, &count, query)
	if err != nil {
		return 0, fmt.Errorf("failed to count providers: %w", err)
	}
	
	return count, nil
}

// List lists providers with pagination
func (r *providerRepository) List(ctx context.Context, limit, offset int) ([]provider.ProviderResource, error) {
	var entities []provider.ProviderResource
	query := `
		SELECT 
			id, name, email, organization, status, api_key_hash, public_key,
			endpoints, metadata, resource_summary, last_seen, last_heartbeat,
			heartbeat_interval_seconds, health_status, consecutive_failures,
			registered_at, activated_at, suspended_at, reputation, reliability_score,
			avg_response_time_ms, uptime_percentage, total_allocations,
			successful_allocations, failed_allocations, current_allocations,
			created_at, updated_at, version
		FROM providers 
		ORDER BY created_at DESC 
		LIMIT $1 OFFSET $2
	`
	
	err := r.db.SelectContext(ctx, &entities, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list providers: %w", err)
	}
	
	return entities, nil
}

// Search performs a filtered search on providers
func (r *providerRepository) Search(ctx context.Context, filter *provider.ProviderSearchFilter, sort *common.SortOption, pagination *common.PaginationOptions) (*common.SearchResult[provider.ProviderResource], error) {
	whereClause, args := r.buildWhereClause(filter)
	orderClause := r.buildOrderClause(sort)
	
	// Count total results
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM providers %s", whereClause)
	var total int64
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to count search results: %w", err)
	}
	
	// Get paginated results
	query := fmt.Sprintf(`
		SELECT 
			id, name, email, organization, status, api_key_hash, public_key,
			endpoints, metadata, resource_summary, last_seen, last_heartbeat,
			heartbeat_interval_seconds, health_status, consecutive_failures,
			registered_at, activated_at, suspended_at, reputation, reliability_score,
			avg_response_time_ms, uptime_percentage, total_allocations,
			successful_allocations, failed_allocations, current_allocations,
			created_at, updated_at, version
		FROM providers %s %s LIMIT $%d OFFSET $%d
	`, whereClause, orderClause, len(args)+1, len(args)+2)
	
	args = append(args, pagination.Limit, pagination.Offset)
	
	var items []provider.ProviderResource
	err = r.db.SelectContext(ctx, &items, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search: %w", err)
	}
	
	return &common.SearchResult[provider.ProviderResource]{
		Items:   items,
		Total:   total,
		Limit:   pagination.Limit,
		Offset:  pagination.Offset,
		HasMore: pagination.Offset+len(items) < int(total),
	}, nil
}

// CountByFilter counts providers matching the filter
func (r *providerRepository) CountByFilter(ctx context.Context, filter *provider.ProviderSearchFilter) (int64, error) {
	whereClause, args := r.buildWhereClause(filter)
	query := fmt.Sprintf("SELECT COUNT(*) FROM providers %s", whereClause)
	
	var count int64
	err := r.db.GetContext(ctx, &count, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to count providers by filter: %w", err)
	}
	
	return count, nil
}

// ListByStatus lists providers by status
func (r *providerRepository) ListByStatus(ctx context.Context, status provider.ProviderStatus) ([]provider.ProviderResource, error) {
	var entities []provider.ProviderResource
	query := `
		SELECT 
			id, name, email, organization, status, api_key_hash, public_key,
			endpoints, metadata, resource_summary, last_seen, last_heartbeat,
			heartbeat_interval_seconds, health_status, consecutive_failures,
			registered_at, activated_at, suspended_at, reputation, reliability_score,
			avg_response_time_ms, uptime_percentage, total_allocations,
			successful_allocations, failed_allocations, current_allocations,
			created_at, updated_at, version
		FROM providers 
		WHERE status = $1
		ORDER BY name
	`
	
	err := r.db.SelectContext(ctx, &entities, query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to list providers by status: %w", err)
	}
	
	return entities, nil
}

// ListByHealthStatus lists providers by health status
func (r *providerRepository) ListByHealthStatus(ctx context.Context, healthStatus provider.HealthStatus) ([]provider.ProviderResource, error) {
	var entities []provider.ProviderResource
	query := `
		SELECT 
			id, name, email, organization, status, api_key_hash, public_key,
			endpoints, metadata, resource_summary, last_seen, last_heartbeat,
			heartbeat_interval_seconds, health_status, consecutive_failures,
			registered_at, activated_at, suspended_at, reputation, reliability_score,
			avg_response_time_ms, uptime_percentage, total_allocations,
			successful_allocations, failed_allocations, current_allocations,
			created_at, updated_at, version
		FROM providers 
		WHERE health_status = $1
		ORDER BY last_heartbeat DESC
	`
	
	err := r.db.SelectContext(ctx, &entities, query, healthStatus)
	if err != nil {
		return nil, fmt.Errorf("failed to list providers by health status: %w", err)
	}
	
	return entities, nil
}

// RecordHeartbeat records a provider heartbeat
func (r *providerRepository) RecordHeartbeat(ctx context.Context, heartbeat *provider.ProviderHeartbeat) error {
	if heartbeat.ID == uuid.Nil {
		heartbeat.ID = uuid.New()
	}
	heartbeat.Timestamp = time.Now()
	
	query := `
		INSERT INTO provider_heartbeats (
			id, provider_id, timestamp, status, resource_summary, system_metrics,
			response_time_ms, message, version
		) VALUES (
			:id, :provider_id, :timestamp, :status, :resource_summary, :system_metrics,
			:response_time_ms, :message, :version
		)
	`
	
	_, err := r.db.NamedExecContext(ctx, query, heartbeat)
	if err != nil {
		return fmt.Errorf("failed to record heartbeat: %w", err)
	}
	
	// Update provider's last_heartbeat and health_status
	updateQuery := `
		UPDATE providers 
		SET last_heartbeat = $2, health_status = $3, consecutive_failures = 0, updated_at = NOW()
		WHERE id = $1
	`
	
	healthStatus := provider.HealthStatusHealthy
	if heartbeat.Status != "healthy" {
		healthStatus = provider.HealthStatusDegraded
	}
	
	_, err = r.db.ExecContext(ctx, updateQuery, heartbeat.ProviderID, heartbeat.Timestamp, healthStatus)
	if err != nil {
		return fmt.Errorf("failed to update provider heartbeat: %w", err)
	}
	
	// Invalidate cache
	r.invalidateCache(heartbeat.ProviderID)
	
	return nil
}

// GetLatestHeartbeat gets the latest heartbeat for a provider
func (r *providerRepository) GetLatestHeartbeat(ctx context.Context, providerID uuid.UUID) (*provider.ProviderHeartbeat, error) {
	var heartbeat provider.ProviderHeartbeat
	query := `
		SELECT id, provider_id, timestamp, status, resource_summary, system_metrics,
		       response_time_ms, message, version
		FROM provider_heartbeats
		WHERE provider_id = $1
		ORDER BY timestamp DESC
		LIMIT 1
	`
	
	err := r.db.GetContext(ctx, &heartbeat, query, providerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no heartbeat found for provider: %s", providerID)
		}
		return nil, fmt.Errorf("failed to get latest heartbeat: %w", err)
	}
	
	return &heartbeat, nil
}

// buildWhereClause builds WHERE clause for search filters
func (r *providerRepository) buildWhereClause(filter *provider.ProviderSearchFilter) (string, []interface{}) {
	if filter == nil {
		return "", []interface{}{}
	}
	
	var conditions []string
	var args []interface{}
	argIndex := 1
	
	// Add provider-specific filters
	if len(filter.Organizations) > 0 {
		conditions = append(conditions, fmt.Sprintf("organization = ANY($%d)", argIndex))
		args = append(args, pq.Array(filter.Organizations))
		argIndex++
	}
	
	if len(filter.HealthStatuses) > 0 {
		statuses := make([]string, len(filter.HealthStatuses))
		for i, status := range filter.HealthStatuses {
			statuses[i] = string(status)
		}
		conditions = append(conditions, fmt.Sprintf("health_status = ANY($%d)", argIndex))
		args = append(args, pq.Array(statuses))
		argIndex++
	}
	
	if filter.MinReputation != nil {
		conditions = append(conditions, fmt.Sprintf("reputation >= $%d", argIndex))
		args = append(args, *filter.MinReputation)
		argIndex++
	}
	
	if filter.MinReliability != nil {
		conditions = append(conditions, fmt.Sprintf("reliability_score >= $%d", argIndex))
		args = append(args, *filter.MinReliability)
		argIndex++
	}
	
	if filter.MinUptimePercent != nil {
		conditions = append(conditions, fmt.Sprintf("uptime_percentage >= $%d", argIndex))
		args = append(args, *filter.MinUptimePercent)
		argIndex++
	}
	
	if filter.MaxResponseTimeMs != nil {
		conditions = append(conditions, fmt.Sprintf("avg_response_time_ms <= $%d", argIndex))
		args = append(args, *filter.MaxResponseTimeMs)
		argIndex++
	}
	
	if filter.HeartbeatAfter != nil {
		conditions = append(conditions, fmt.Sprintf("last_heartbeat >= $%d", argIndex))
		args = append(args, *filter.HeartbeatAfter)
		argIndex++
	}
	
	// Add base search filter conditions
	conditions, args = r.addBaseFilterConditions(conditions, args, &filter.SearchFilter, argIndex)
	
	if len(conditions) == 0 {
		return "", args
	}
	
	return "WHERE " + strings.Join(conditions, " AND "), args
}

// addBaseFilterConditions adds common filter conditions
func (r *providerRepository) addBaseFilterConditions(conditions []string, args []interface{}, filter *common.SearchFilter, argIndex int) ([]string, []interface{}) {
	if len(filter.Regions) > 0 {
		conditions = append(conditions, fmt.Sprintf("metadata->>'region' = ANY($%d)", argIndex))
		args = append(args, pq.Array(filter.Regions))
		argIndex++
	}
	
	if filter.CreatedAfter != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIndex))
		args = append(args, *filter.CreatedAfter)
		argIndex++
	}
	
	if filter.CreatedBefore != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIndex))
		args = append(args, *filter.CreatedBefore)
	}
	
	return conditions, args
}

// buildOrderClause builds ORDER BY clause
func (r *providerRepository) buildOrderClause(sort *common.SortOption) string {
	if sort == nil {
		return "ORDER BY created_at DESC"
	}
	
	field := sort.Field
	order := "ASC"
	if strings.ToLower(sort.Order) == "desc" {
		order = "DESC"
	}
	
	// Validate field to prevent SQL injection
	validFields := map[string]bool{
		"name": true, "created_at": true, "updated_at": true, "reputation": true,
		"reliability_score": true, "uptime_percentage": true, "last_heartbeat": true,
	}
	
	if !validFields[field] {
		field = "created_at"
	}
	
	return fmt.Sprintf("ORDER BY %s %s", field, order)
}

// invalidateCache invalidates cache entries for a provider
func (r *providerRepository) invalidateCache(providerID uuid.UUID) {
	if r.cache == nil {
		return
	}
	
	ctx := context.Background()
	cacheKey := fmt.Sprintf("provider:%s", providerID)
	r.cache.Delete(ctx, cacheKey)
}

// Health check methods
func (r *providerRepository) IsHealthy(ctx context.Context) bool {
	err := r.db.PingContext(ctx)
	return err == nil
}

func (r *providerRepository) GetLastHealthCheck() time.Time {
	// Implementation would track the last health check time
	return time.Now()
}

func (r *providerRepository) GetStats() map[string]interface{} {
	stats := r.db.Stats()
	return map[string]interface{}{
		"open_connections": stats.OpenConnections,
		"in_use":          stats.InUse,
		"idle":            stats.Idle,
	}
}

// Additional provider-specific methods will continue in the next part...
// (UpdateLastSeen, UpdateHealthStatus, etc.)