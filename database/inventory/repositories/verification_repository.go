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

// verificationRepository implements VerificationRepository
type verificationRepository struct {
	db *sqlx.DB
}

// NewVerificationRepository creates a new verification repository
func NewVerificationRepository(dm *utils.DatabaseManager) VerificationRepository {
	return &verificationRepository{
		db: dm.GetDB(),
	}
}

// NewVerificationRepositoryWithTx creates a repository with transaction
func NewVerificationRepositoryWithTx(tx *sqlx.Tx) VerificationRepository {
	return &verificationRepository{
		db: tx,
	}
}

// Create creates a new verification
func (r *verificationRepository) Create(ctx context.Context, entity *common.Verification) error {
	if entity.ID == uuid.Nil {
		entity.ID = uuid.New()
	}
	entity.VerifiedAt = time.Now()
	
	query := `
		INSERT INTO verifications (
			id, resource_id, resource_type, type, status, score, details,
			verified_at, expires_at, verifier_id
		) VALUES (
			:id, :resource_id, :resource_type, :type, :status, :score, :details,
			:verified_at, :expires_at, :verifier_id
		)
	`
	
	_, err := r.db.NamedExecContext(ctx, query, entity)
	if err != nil {
		return fmt.Errorf("failed to create verification: %w", err)
	}
	
	return nil
}

// GetByID retrieves a verification by ID
func (r *verificationRepository) GetByID(ctx context.Context, id uuid.UUID) (*common.Verification, error) {
	var entity common.Verification
	query := `
		SELECT id, resource_id, resource_type, type, status, score, details,
		       verified_at, expires_at, verifier_id
		FROM verifications 
		WHERE id = $1
	`
	
	err := r.db.GetContext(ctx, &entity, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("verification not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get verification: %w", err)
	}
	
	return &entity, nil
}

// CreateVerification creates a verification record
func (r *verificationRepository) CreateVerification(ctx context.Context, verification *common.Verification) error {
	return r.Create(ctx, verification)
}

// GetLatestVerification gets the latest verification for a resource
func (r *verificationRepository) GetLatestVerification(ctx context.Context, resourceID uuid.UUID, verificationType string) (*common.Verification, error) {
	var entity common.Verification
	query := `
		SELECT id, resource_id, resource_type, type, status, score, details,
		       verified_at, expires_at, verifier_id
		FROM verifications
		WHERE resource_id = $1 AND type = $2 AND expires_at > NOW()
		ORDER BY verified_at DESC
		LIMIT 1
	`
	
	err := r.db.GetContext(ctx, &entity, query, resourceID, verificationType)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no valid verification found for resource %s with type %s", resourceID, verificationType)
		}
		return nil, fmt.Errorf("failed to get latest verification: %w", err)
	}
	
	return &entity, nil
}

// GetVerificationHistory gets verification history for a resource
func (r *verificationRepository) GetVerificationHistory(ctx context.Context, resourceID uuid.UUID, limit int) ([]common.Verification, error) {
	var entities []common.Verification
	query := `
		SELECT id, resource_id, resource_type, type, status, score, details,
		       verified_at, expires_at, verifier_id
		FROM verifications
		WHERE resource_id = $1
		ORDER BY verified_at DESC
		LIMIT $2
	`
	
	err := r.db.SelectContext(ctx, &entities, query, resourceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get verification history: %w", err)
	}
	
	return entities, nil
}

// ListExpiredVerifications lists expired verifications
func (r *verificationRepository) ListExpiredVerifications(ctx context.Context, asOf time.Time) ([]common.Verification, error) {
	var entities []common.Verification
	query := `
		SELECT id, resource_id, resource_type, type, status, score, details,
		       verified_at, expires_at, verifier_id
		FROM verifications
		WHERE expires_at <= $1 AND status = 'valid'
		ORDER BY expires_at ASC
	`
	
	err := r.db.SelectContext(ctx, &entities, query, asOf)
	if err != nil {
		return nil, fmt.Errorf("failed to list expired verifications: %w", err)
	}
	
	return entities, nil
}

// MarkExpiredVerifications marks verifications as expired
func (r *verificationRepository) MarkExpiredVerifications(ctx context.Context, asOf time.Time) (int64, error) {
	query := `
		UPDATE verifications 
		SET status = 'expired' 
		WHERE expires_at <= $1 AND status = 'valid'
	`
	
	result, err := r.db.ExecContext(ctx, query, asOf)
	if err != nil {
		return 0, fmt.Errorf("failed to mark expired verifications: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}
	
	return rowsAffected, nil
}

// GetVerificationStats gets verification statistics
func (r *verificationRepository) GetVerificationStats(ctx context.Context, timeRange time.Duration) (map[string]interface{}, error) {
	since := time.Now().Add(-timeRange)
	
	stats := make(map[string]interface{})
	
	// Total verifications in time range
	var total int64
	query := `SELECT COUNT(*) FROM verifications WHERE verified_at >= $1`
	err := r.db.GetContext(ctx, &total, query, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get total verifications: %w", err)
	}
	stats["total_verifications"] = total
	
	// Verifications by status
	var statusStats []struct {
		Status string `db:"status"`
		Count  int64  `db:"count"`
	}
	query = `
		SELECT status, COUNT(*) as count
		FROM verifications 
		WHERE verified_at >= $1
		GROUP BY status
	`
	err = r.db.SelectContext(ctx, &statusStats, query, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get status stats: %w", err)
	}
	
	statusMap := make(map[string]int64)
	for _, stat := range statusStats {
		statusMap[stat.Status] = stat.Count
	}
	stats["by_status"] = statusMap
	
	// Average verification scores by type
	var scoreStats []struct {
		Type     string  `db:"type"`
		AvgScore float64 `db:"avg_score"`
	}
	query = `
		SELECT type, AVG(score) as avg_score
		FROM verifications 
		WHERE verified_at >= $1 AND score IS NOT NULL
		GROUP BY type
	`
	err = r.db.SelectContext(ctx, &scoreStats, query, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get score stats: %w", err)
	}
	
	scoreMap := make(map[string]float64)
	for _, stat := range scoreStats {
		scoreMap[stat.Type] = stat.AvgScore
	}
	stats["average_scores"] = scoreMap
	
	return stats, nil
}

// Update updates a verification
func (r *verificationRepository) Update(ctx context.Context, entity *common.Verification) error {
	query := `
		UPDATE verifications SET
			status = :status, score = :score, details = :details,
			expires_at = :expires_at, verifier_id = :verifier_id
		WHERE id = :id
	`
	
	result, err := r.db.NamedExecContext(ctx, query, entity)
	if err != nil {
		return fmt.Errorf("failed to update verification: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("verification not found: %s", entity.ID)
	}
	
	return nil
}

// Delete deletes a verification
func (r *verificationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM verifications WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete verification: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("verification not found: %s", id)
	}
	
	return nil
}

// CreateBatch creates multiple verifications
func (r *verificationRepository) CreateBatch(ctx context.Context, entities []*common.Verification) error {
	if len(entities) == 0 {
		return nil
	}
	
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	query := `
		INSERT INTO verifications (
			id, resource_id, resource_type, type, status, score, details,
			verified_at, expires_at, verifier_id
		) VALUES (
			:id, :resource_id, :resource_type, :type, :status, :score, :details,
			:verified_at, :expires_at, :verifier_id
		)
	`
	
	now := time.Now()
	for _, entity := range entities {
		if entity.ID == uuid.Nil {
			entity.ID = uuid.New()
		}
		entity.VerifiedAt = now
		
		_, err := tx.NamedExecContext(ctx, query, entity)
		if err != nil {
			return fmt.Errorf("failed to insert verification in batch: %w", err)
		}
	}
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}

// UpdateBatch updates multiple verifications
func (r *verificationRepository) UpdateBatch(ctx context.Context, entities []*common.Verification) error {
	if len(entities) == 0 {
		return nil
	}
	
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	query := `
		UPDATE verifications SET
			status = :status, score = :score, details = :details,
			expires_at = :expires_at, verifier_id = :verifier_id
		WHERE id = :id
	`
	
	for _, entity := range entities {
		result, err := tx.NamedExecContext(ctx, query, entity)
		if err != nil {
			return fmt.Errorf("failed to update verification in batch: %w", err)
		}
		
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get affected rows: %w", err)
		}
		
		if rowsAffected == 0 {
			return fmt.Errorf("verification not found: %s", entity.ID)
		}
	}
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}

// DeleteBatch deletes multiple verifications
func (r *verificationRepository) DeleteBatch(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	
	query := `DELETE FROM verifications WHERE id = ANY($1)`
	
	result, err := r.db.ExecContext(ctx, query, pq.Array(ids))
	if err != nil {
		return fmt.Errorf("failed to delete verifications: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	
	if rowsAffected != int64(len(ids)) {
		return fmt.Errorf("expected to delete %d verifications, but deleted %d", len(ids), rowsAffected)
	}
	
	return nil
}

// Count returns the total number of verifications
func (r *verificationRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM verifications`
	
	err := r.db.GetContext(ctx, &count, query)
	if err != nil {
		return 0, fmt.Errorf("failed to count verifications: %w", err)
	}
	
	return count, nil
}

// Health check methods
func (r *verificationRepository) IsHealthy(ctx context.Context) bool {
	err := r.db.PingContext(ctx)
	return err == nil
}

func (r *verificationRepository) GetLastHealthCheck() time.Time {
	return time.Now()
}

func (r *verificationRepository) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"repository": "verifications",
	}
}