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
	"../../models/resources/gpu"
	"../utils"
)

// gpuRepository implements GPURepository
type gpuRepository struct {
	db    *sqlx.DB
	cache *utils.Cache
}

// NewGPURepository creates a new GPU repository
func NewGPURepository(dm *utils.DatabaseManager) GPURepository {
	return &gpuRepository{
		db:    dm.GetDB(),
		cache: dm.NewCache(),
	}
}

// Create creates a new GPU resource
func (r *gpuRepository) Create(ctx context.Context, entity *gpu.GPUResource) error {
	if entity.ID == uuid.Nil {
		entity.ID = uuid.New()
	}
	entity.CreatedAt = time.Now()
	entity.UpdatedAt = time.Now()
	entity.Version = 1
	
	query := `
		INSERT INTO gpu_resources (
			id, provider_id, type, name, status, region, data_center, tags,
			uuid, index, vendor, specs, current_status, capabilities, driver_info,
			discovery_source, last_discovered, verification_status, last_verified,
			is_allocated, current_allocation, allocation_start_time,
			avg_utilization, peak_utilization, uptime_percentage,
			created_at, updated_at, last_heartbeat, version
		) VALUES (
			:id, :provider_id, :type, :name, :status, :region, :data_center, :tags,
			:uuid, :index, :vendor, :specs, :current_status, :capabilities, :driver_info,
			:discovery_source, :last_discovered, :verification_status, :last_verified,
			:is_allocated, :current_allocation, :allocation_start_time,
			:avg_utilization, :peak_utilization, :uptime_percentage,
			:created_at, :updated_at, :last_heartbeat, :version
		)
	`
	
	_, err := r.db.NamedExecContext(ctx, query, entity)
	if err != nil {
		return fmt.Errorf("failed to create GPU resource: %w", err)
	}
	
	// Invalidate cache
	r.invalidateCache(entity.ID)
	
	return nil
}

// GetByID retrieves a GPU by ID
func (r *gpuRepository) GetByID(ctx context.Context, id uuid.UUID) (*gpu.GPUResource, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("gpu:%s", id)
	if r.cache != nil {
		if cached, err := r.cache.Get(ctx, cacheKey); err == nil {
			if cachedGPU, ok := cached.(*gpu.GPUResource); ok {
				return cachedGPU, nil
			}
		}
	}
	
	var entity gpu.GPUResource
	query := `
		SELECT 
			id, provider_id, type, name, status, region, data_center, tags,
			uuid, index, vendor, specs, current_status, capabilities, driver_info,
			discovery_source, last_discovered, verification_status, last_verified,
			is_allocated, current_allocation, allocation_start_time,
			avg_utilization, peak_utilization, uptime_percentage,
			created_at, updated_at, last_heartbeat, version
		FROM gpu_resources 
		WHERE id = $1
	`
	
	err := r.db.GetContext(ctx, &entity, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("GPU resource not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get GPU resource: %w", err)
	}
	
	// Cache the result
	if r.cache != nil {
		r.cache.SetWithTTL(ctx, cacheKey, &entity, 15*time.Minute)
	}
	
	return &entity, nil
}

// GetByUUID retrieves a GPU by UUID
func (r *gpuRepository) GetByUUID(ctx context.Context, uuid string) (*gpu.GPUResource, error) {
	var entity gpu.GPUResource
	query := `
		SELECT 
			id, provider_id, type, name, status, region, data_center, tags,
			uuid, index, vendor, specs, current_status, capabilities, driver_info,
			discovery_source, last_discovered, verification_status, last_verified,
			is_allocated, current_allocation, allocation_start_time,
			avg_utilization, peak_utilization, uptime_percentage,
			created_at, updated_at, last_heartbeat, version
		FROM gpu_resources 
		WHERE uuid = $1
	`
	
	err := r.db.GetContext(ctx, &entity, query, uuid)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("GPU resource not found with UUID: %s", uuid)
		}
		return nil, fmt.Errorf("failed to get GPU resource by UUID: %w", err)
	}
	
	return &entity, nil
}

// GetByProviderID retrieves all GPUs for a provider
func (r *gpuRepository) GetByProviderID(ctx context.Context, providerID uuid.UUID) ([]gpu.GPUResource, error) {
	var entities []gpu.GPUResource
	query := `
		SELECT 
			id, provider_id, type, name, status, region, data_center, tags,
			uuid, index, vendor, specs, current_status, capabilities, driver_info,
			discovery_source, last_discovered, verification_status, last_verified,
			is_allocated, current_allocation, allocation_start_time,
			avg_utilization, peak_utilization, uptime_percentage,
			created_at, updated_at, last_heartbeat, version
		FROM gpu_resources 
		WHERE provider_id = $1
		ORDER BY name, index
	`
	
	err := r.db.SelectContext(ctx, &entities, query, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get GPUs by provider ID: %w", err)
	}
	
	return entities, nil
}

// ListByVendor lists GPUs by vendor
func (r *gpuRepository) ListByVendor(ctx context.Context, vendor string) ([]gpu.GPUResource, error) {
	var entities []gpu.GPUResource
	query := `
		SELECT 
			id, provider_id, type, name, status, region, data_center, tags,
			uuid, index, vendor, specs, current_status, capabilities, driver_info,
			discovery_source, last_discovered, verification_status, last_verified,
			is_allocated, current_allocation, allocation_start_time,
			avg_utilization, peak_utilization, uptime_percentage,
			created_at, updated_at, last_heartbeat, version
		FROM gpu_resources 
		WHERE vendor = $1
		ORDER BY name, index
	`
	
	err := r.db.SelectContext(ctx, &entities, query, vendor)
	if err != nil {
		return nil, fmt.Errorf("failed to list GPUs by vendor: %w", err)
	}
	
	return entities, nil
}

// ListByStatus lists GPUs by status
func (r *gpuRepository) ListByStatus(ctx context.Context, status common.ResourceStatus) ([]gpu.GPUResource, error) {
	var entities []gpu.GPUResource
	query := `
		SELECT 
			id, provider_id, type, name, status, region, data_center, tags,
			uuid, index, vendor, specs, current_status, capabilities, driver_info,
			discovery_source, last_discovered, verification_status, last_verified,
			is_allocated, current_allocation, allocation_start_time,
			avg_utilization, peak_utilization, uptime_percentage,
			created_at, updated_at, last_heartbeat, version
		FROM gpu_resources 
		WHERE status = $1
		ORDER BY last_heartbeat DESC NULLS LAST
	`
	
	err := r.db.SelectContext(ctx, &entities, query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to list GPUs by status: %w", err)
	}
	
	return entities, nil
}

// ListAvailable lists available GPUs in a region
func (r *gpuRepository) ListAvailable(ctx context.Context, region string) ([]gpu.GPUResource, error) {
	var entities []gpu.GPUResource
	query := `
		SELECT 
			id, provider_id, type, name, status, region, data_center, tags,
			uuid, index, vendor, specs, current_status, capabilities, driver_info,
			discovery_source, last_discovered, verification_status, last_verified,
			is_allocated, current_allocation, allocation_start_time,
			avg_utilization, peak_utilization, uptime_percentage,
			created_at, updated_at, last_heartbeat, version
		FROM gpu_resources 
		WHERE region = $1 AND status = 'active' AND is_allocated = false
		ORDER BY uptime_percentage DESC, avg_utilization ASC
	`
	
	err := r.db.SelectContext(ctx, &entities, query, region)
	if err != nil {
		return nil, fmt.Errorf("failed to list available GPUs: %w", err)
	}
	
	return entities, nil
}

// Update updates a GPU resource
func (r *gpuRepository) Update(ctx context.Context, entity *gpu.GPUResource) error {
	entity.UpdatedAt = time.Now()
	entity.Version++
	
	query := `
		UPDATE gpu_resources SET
			provider_id = :provider_id, type = :type, name = :name, status = :status,
			region = :region, data_center = :data_center, tags = :tags, uuid = :uuid,
			index = :index, vendor = :vendor, specs = :specs, current_status = :current_status,
			capabilities = :capabilities, driver_info = :driver_info, discovery_source = :discovery_source,
			last_discovered = :last_discovered, verification_status = :verification_status,
			last_verified = :last_verified, is_allocated = :is_allocated,
			current_allocation = :current_allocation, allocation_start_time = :allocation_start_time,
			avg_utilization = :avg_utilization, peak_utilization = :peak_utilization,
			uptime_percentage = :uptime_percentage, updated_at = :updated_at,
			last_heartbeat = :last_heartbeat, version = :version
		WHERE id = :id AND version = :version - 1
	`
	
	result, err := r.db.NamedExecContext(ctx, query, entity)
	if err != nil {
		return fmt.Errorf("failed to update GPU resource: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("GPU resource not found or version conflict")
	}
	
	// Invalidate cache
	r.invalidateCache(entity.ID)
	
	return nil
}

// Delete deletes a GPU resource
func (r *gpuRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM gpu_resources WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete GPU resource: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("GPU resource not found: %s", id)
	}
	
	// Invalidate cache
	r.invalidateCache(id)
	
	return nil
}

// CreateBatch creates multiple GPUs in a single transaction
func (r *gpuRepository) CreateBatch(ctx context.Context, entities []*gpu.GPUResource) error {
	if len(entities) == 0 {
		return nil
	}
	
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	query := `
		INSERT INTO gpu_resources (
			id, provider_id, type, name, status, region, data_center, tags,
			uuid, index, vendor, specs, current_status, capabilities, driver_info,
			discovery_source, last_discovered, verification_status,
			is_allocated, avg_utilization, peak_utilization, uptime_percentage,
			created_at, updated_at, version
		) VALUES (
			:id, :provider_id, :type, :name, :status, :region, :data_center, :tags,
			:uuid, :index, :vendor, :specs, :current_status, :capabilities, :driver_info,
			:discovery_source, :last_discovered, :verification_status,
			:is_allocated, :avg_utilization, :peak_utilization, :uptime_percentage,
			:created_at, :updated_at, :version
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
			return fmt.Errorf("failed to insert GPU resource in batch: %w", err)
		}
	}
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}

// UpdateBatch updates multiple GPU resources
func (r *gpuRepository) UpdateBatch(ctx context.Context, entities []*gpu.GPUResource) error {
	if len(entities) == 0 {
		return nil
	}
	
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	query := `
		UPDATE gpu_resources SET
			status = :status, current_status = :current_status, specs = :specs,
			last_discovered = :last_discovered, is_allocated = :is_allocated,
			avg_utilization = :avg_utilization, peak_utilization = :peak_utilization,
			uptime_percentage = :uptime_percentage, updated_at = :updated_at,
			last_heartbeat = :last_heartbeat, version = :version
		WHERE id = :id AND version = :version - 1
	`
	
	now := time.Now()
	for _, entity := range entities {
		entity.UpdatedAt = now
		entity.Version++
		
		result, err := tx.NamedExecContext(ctx, query, entity)
		if err != nil {
			return fmt.Errorf("failed to update GPU resource in batch: %w", err)
		}
		
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get affected rows: %w", err)
		}
		
		if rowsAffected == 0 {
			return fmt.Errorf("GPU resource not found or version conflict: %s", entity.ID)
		}
	}
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}

// DeleteBatch deletes multiple GPU resources
func (r *gpuRepository) DeleteBatch(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	
	query := `DELETE FROM gpu_resources WHERE id = ANY($1)`
	
	result, err := r.db.ExecContext(ctx, query, pq.Array(ids))
	if err != nil {
		return fmt.Errorf("failed to delete GPU resources: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	
	if rowsAffected != int64(len(ids)) {
		return fmt.Errorf("expected to delete %d GPU resources, but deleted %d", len(ids), rowsAffected)
	}
	
	// Invalidate cache for all deleted GPUs
	for _, id := range ids {
		r.invalidateCache(id)
	}
	
	return nil
}

// Count returns the total number of GPU resources
func (r *gpuRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM gpu_resources`
	
	err := r.db.GetContext(ctx, &count, query)
	if err != nil {
		return 0, fmt.Errorf("failed to count GPU resources: %w", err)
	}
	
	return count, nil
}

// List lists GPU resources with pagination
func (r *gpuRepository) List(ctx context.Context, limit, offset int) ([]gpu.GPUResource, error) {
	var entities []gpu.GPUResource
	query := `
		SELECT 
			id, provider_id, type, name, status, region, data_center, tags,
			uuid, index, vendor, specs, current_status, capabilities, driver_info,
			discovery_source, last_discovered, verification_status, last_verified,
			is_allocated, current_allocation, allocation_start_time,
			avg_utilization, peak_utilization, uptime_percentage,
			created_at, updated_at, last_heartbeat, version
		FROM gpu_resources 
		ORDER BY created_at DESC 
		LIMIT $1 OFFSET $2
	`
	
	err := r.db.SelectContext(ctx, &entities, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list GPU resources: %w", err)
	}
	
	return entities, nil
}

// Search performs a filtered search on GPU resources
func (r *gpuRepository) Search(ctx context.Context, filter *gpu.GPUSearchFilter, sort *common.SortOption, pagination *common.PaginationOptions) (*common.SearchResult[gpu.GPUResource], error) {
	whereClause, args := r.buildWhereClause(filter)
	orderClause := r.buildOrderClause(sort)
	
	// Count total results
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM gpu_resources %s", whereClause)
	var total int64
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to count search results: %w", err)
	}
	
	// Get paginated results
	query := fmt.Sprintf(`
		SELECT 
			id, provider_id, type, name, status, region, data_center, tags,
			uuid, index, vendor, specs, current_status, capabilities, driver_info,
			discovery_source, last_discovered, verification_status, last_verified,
			is_allocated, current_allocation, allocation_start_time,
			avg_utilization, peak_utilization, uptime_percentage,
			created_at, updated_at, last_heartbeat, version
		FROM gpu_resources %s %s LIMIT $%d OFFSET $%d
	`, whereClause, orderClause, len(args)+1, len(args)+2)
	
	args = append(args, pagination.Limit, pagination.Offset)
	
	var items []gpu.GPUResource
	err = r.db.SelectContext(ctx, &items, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search: %w", err)
	}
	
	return &common.SearchResult[gpu.GPUResource]{
		Items:   items,
		Total:   total,
		Limit:   pagination.Limit,
		Offset:  pagination.Offset,
		HasMore: pagination.Offset+len(items) < int(total),
	}, nil
}

// CountByFilter counts GPU resources matching the filter
func (r *gpuRepository) CountByFilter(ctx context.Context, filter *gpu.GPUSearchFilter) (int64, error) {
	whereClause, args := r.buildWhereClause(filter)
	query := fmt.Sprintf("SELECT COUNT(*) FROM gpu_resources %s", whereClause)
	
	var count int64
	err := r.db.GetContext(ctx, &count, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to count GPU resources by filter: %w", err)
	}
	
	return count, nil
}

// Allocation operations
func (r *gpuRepository) MarkAsAllocated(ctx context.Context, gpuID uuid.UUID, allocationID uuid.UUID, startTime time.Time) error {
	query := `
		UPDATE gpu_resources 
		SET is_allocated = true, current_allocation = $2, allocation_start_time = $3, updated_at = NOW()
		WHERE id = $1 AND is_allocated = false
	`
	
	result, err := r.db.ExecContext(ctx, query, gpuID, allocationID, startTime)
	if err != nil {
		return fmt.Errorf("failed to mark GPU as allocated: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("GPU not found or already allocated: %s", gpuID)
	}
	
	// Invalidate cache
	r.invalidateCache(gpuID)
	
	return nil
}

// MarkAsReleased marks a GPU as released from allocation
func (r *gpuRepository) MarkAsReleased(ctx context.Context, gpuID uuid.UUID, endTime time.Time) error {
	query := `
		UPDATE gpu_resources 
		SET is_allocated = false, current_allocation = NULL, allocation_start_time = NULL, updated_at = NOW()
		WHERE id = $1 AND is_allocated = true
	`
	
	result, err := r.db.ExecContext(ctx, query, gpuID)
	if err != nil {
		return fmt.Errorf("failed to mark GPU as released: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("GPU not found or not allocated: %s", gpuID)
	}
	
	// Invalidate cache
	r.invalidateCache(gpuID)
	
	return nil
}

// GetAllocatedGPUs gets all currently allocated GPUs
func (r *gpuRepository) GetAllocatedGPUs(ctx context.Context, providerID *uuid.UUID) ([]gpu.GPUResource, error) {
	var entities []gpu.GPUResource
	var query string
	var args []interface{}
	
	if providerID != nil {
		query = `
			SELECT 
				id, provider_id, type, name, status, region, data_center, tags,
				uuid, index, vendor, specs, current_status, capabilities, driver_info,
				discovery_source, last_discovered, verification_status, last_verified,
				is_allocated, current_allocation, allocation_start_time,
				avg_utilization, peak_utilization, uptime_percentage,
				created_at, updated_at, last_heartbeat, version
			FROM gpu_resources 
			WHERE is_allocated = true AND provider_id = $1
			ORDER BY allocation_start_time DESC
		`
		args = []interface{}{*providerID}
	} else {
		query = `
			SELECT 
				id, provider_id, type, name, status, region, data_center, tags,
				uuid, index, vendor, specs, current_status, capabilities, driver_info,
				discovery_source, last_discovered, verification_status, last_verified,
				is_allocated, current_allocation, allocation_start_time,
				avg_utilization, peak_utilization, uptime_percentage,
				created_at, updated_at, last_heartbeat, version
			FROM gpu_resources 
			WHERE is_allocated = true
			ORDER BY allocation_start_time DESC
		`
		args = []interface{}{}
	}
	
	err := r.db.SelectContext(ctx, &entities, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get allocated GPUs: %w", err)
	}
	
	return entities, nil
}

// Status update operations
func (r *gpuRepository) UpdateCurrentStatus(ctx context.Context, gpuID uuid.UUID, status *gpu.GPUCurrentStatus) error {
	query := `
		UPDATE gpu_resources 
		SET current_status = $2, updated_at = NOW(), last_heartbeat = NOW()
		WHERE id = $1
	`
	
	result, err := r.db.ExecContext(ctx, query, gpuID, status)
	if err != nil {
		return fmt.Errorf("failed to update GPU current status: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("GPU resource not found: %s", gpuID)
	}
	
	// Invalidate cache
	r.invalidateCache(gpuID)
	
	return nil
}

// UpdateLastDiscovered updates the last discovered timestamp
func (r *gpuRepository) UpdateLastDiscovered(ctx context.Context, gpuID uuid.UUID, timestamp time.Time) error {
	query := `
		UPDATE gpu_resources 
		SET last_discovered = $2, updated_at = NOW()
		WHERE id = $1
	`
	
	result, err := r.db.ExecContext(ctx, query, gpuID, timestamp)
	if err != nil {
		return fmt.Errorf("failed to update last discovered: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("GPU resource not found: %s", gpuID)
	}
	
	// Invalidate cache
	r.invalidateCache(gpuID)
	
	return nil
}

// UpdateVerificationStatus updates the verification status
func (r *gpuRepository) UpdateVerificationStatus(ctx context.Context, gpuID uuid.UUID, status string, verifiedAt *time.Time) error {
	query := `
		UPDATE gpu_resources 
		SET verification_status = $2, last_verified = $3, updated_at = NOW()
		WHERE id = $1
	`
	
	result, err := r.db.ExecContext(ctx, query, gpuID, status, verifiedAt)
	if err != nil {
		return fmt.Errorf("failed to update verification status: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("GPU resource not found: %s", gpuID)
	}
	
	// Invalidate cache
	r.invalidateCache(gpuID)
	
	return nil
}

// UpdateUtilizationMetrics updates utilization metrics
func (r *gpuRepository) UpdateUtilizationMetrics(ctx context.Context, gpuID uuid.UUID, avg, peak float64) error {
	query := `
		UPDATE gpu_resources 
		SET avg_utilization = $2, peak_utilization = $3, updated_at = NOW()
		WHERE id = $1
	`
	
	result, err := r.db.ExecContext(ctx, query, gpuID, avg, peak)
	if err != nil {
		return fmt.Errorf("failed to update utilization metrics: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("GPU resource not found: %s", gpuID)
	}
	
	// Invalidate cache
	r.invalidateCache(gpuID)
	
	return nil
}

// UpdateUptimePercentage updates uptime percentage
func (r *gpuRepository) UpdateUptimePercentage(ctx context.Context, gpuID uuid.UUID, uptime float64) error {
	query := `
		UPDATE gpu_resources 
		SET uptime_percentage = $2, updated_at = NOW()
		WHERE id = $1
	`
	
	result, err := r.db.ExecContext(ctx, query, gpuID, uptime)
	if err != nil {
		return fmt.Errorf("failed to update uptime percentage: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("GPU resource not found: %s", gpuID)
	}
	
	// Invalidate cache
	r.invalidateCache(gpuID)
	
	return nil
}

// buildWhereClause builds WHERE clause for GPU search filters
func (r *gpuRepository) buildWhereClause(filter *gpu.GPUSearchFilter) (string, []interface{}) {
	if filter == nil {
		return "", []interface{}{}
	}
	
	var conditions []string
	var args []interface{}
	argIndex := 1
	
	// Provider filter
	if len(filter.ProviderIDs) > 0 {
		conditions = append(conditions, fmt.Sprintf("provider_id = ANY($%d)", argIndex))
		args = append(args, pq.Array(filter.ProviderIDs))
		argIndex++
	}
	
	// Status filter
	if len(filter.Statuses) > 0 {
		statuses := make([]string, len(filter.Statuses))
		for i, status := range filter.Statuses {
			statuses[i] = string(status)
		}
		conditions = append(conditions, fmt.Sprintf("status = ANY($%d)", argIndex))
		args = append(args, pq.Array(statuses))
		argIndex++
	}
	
	// GPU-specific filters
	if len(filter.GPUVendors) > 0 {
		conditions = append(conditions, fmt.Sprintf("vendor = ANY($%d)", argIndex))
		args = append(args, pq.Array(filter.GPUVendors))
		argIndex++
	}
	
	if filter.MinMemoryMB != nil {
		conditions = append(conditions, fmt.Sprintf("specs->>'memory_total_mb' >= $%d", argIndex))
		args = append(args, *filter.MinMemoryMB)
		argIndex++
	}
	
	if filter.MaxMemoryMB != nil {
		conditions = append(conditions, fmt.Sprintf("specs->>'memory_total_mb' <= $%d", argIndex))
		args = append(args, *filter.MaxMemoryMB)
		argIndex++
	}
	
	if filter.MinCUDACores != nil {
		conditions = append(conditions, fmt.Sprintf("specs->>'cuda_cores' >= $%d", argIndex))
		args = append(args, *filter.MinCUDACores)
		argIndex++
	}
	
	if filter.SupportsCUDA != nil {
		conditions = append(conditions, fmt.Sprintf("capabilities->>'supports_cuda' = $%d", argIndex))
		args = append(args, *filter.SupportsCUDA)
		argIndex++
	}
	
	if filter.IsAllocated != nil {
		conditions = append(conditions, fmt.Sprintf("is_allocated = $%d", argIndex))
		args = append(args, *filter.IsAllocated)
		argIndex++
	}
	
	// Region filter
	if len(filter.Regions) > 0 {
		conditions = append(conditions, fmt.Sprintf("region = ANY($%d)", argIndex))
		args = append(args, pq.Array(filter.Regions))
		argIndex++
	}
	
	if len(conditions) == 0 {
		return "", args
	}
	
	return "WHERE " + strings.Join(conditions, " AND "), args
}

// buildOrderClause builds ORDER BY clause
func (r *gpuRepository) buildOrderClause(sort *common.SortOption) string {
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
		"name": true, "created_at": true, "updated_at": true, "vendor": true,
		"region": true, "last_heartbeat": true, "avg_utilization": true,
		"uptime_percentage": true, "allocation_start_time": true,
	}
	
	if !validFields[field] {
		field = "created_at"
	}
	
	return fmt.Sprintf("ORDER BY %s %s", field, order)
}

// invalidateCache invalidates cache entries for a GPU
func (r *gpuRepository) invalidateCache(gpuID uuid.UUID) {
	if r.cache == nil {
		return
	}
	
	ctx := context.Background()
	cacheKey := fmt.Sprintf("gpu:%s", gpuID)
	r.cache.Delete(ctx, cacheKey)
}

// Health check methods
func (r *gpuRepository) IsHealthy(ctx context.Context) bool {
	err := r.db.PingContext(ctx)
	return err == nil
}

func (r *gpuRepository) GetLastHealthCheck() time.Time {
	return time.Now()
}

func (r *gpuRepository) GetStats() map[string]interface{} {
	stats := r.db.Stats()
	return map[string]interface{}{
		"open_connections": stats.OpenConnections,
		"in_use":          stats.InUse,
		"idle":            stats.Idle,
	}
}

// Additional GPU-specific methods (CreateProcess, CreateBenchmark, etc.) 
// would be implemented here...