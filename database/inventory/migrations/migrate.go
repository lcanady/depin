package migrations

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"../config"
	"../utils"
)

//go:embed *.sql
var migrationFiles embed.FS

// Migration represents a database migration
type Migration struct {
	Version   int
	Name      string
	UpSQL     string
	DownSQL   string
	AppliedAt *time.Time
}

// Migrator handles database migrations
type Migrator struct {
	db     *sqlx.DB
	config *config.DatabaseConfig
}

// NewMigrator creates a new migrator instance
func NewMigrator(dm *utils.DatabaseManager, cfg *config.DatabaseConfig) *Migrator {
	return &Migrator{
		db:     dm.GetDB(),
		config: cfg,
	}
}

// createMigrationsTable creates the migrations tracking table
func (m *Migrator) createMigrationsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			checksum VARCHAR(64) NOT NULL
		);
		
		CREATE INDEX IF NOT EXISTS idx_schema_migrations_applied_at 
		ON schema_migrations(applied_at);
	`
	
	_, err := m.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}
	
	return nil
}

// loadMigrations loads all migration files from the embedded filesystem
func (m *Migrator) loadMigrations() ([]*Migration, error) {
	var migrations []*Migration
	migrationMap := make(map[int]*Migration)
	
	err := fs.WalkDir(migrationFiles, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if d.IsDir() || !strings.HasSuffix(path, ".sql") {
			return nil
		}
		
		// Parse filename: 001_initial_schema.up.sql or 001_initial_schema.down.sql
		filename := filepath.Base(path)
		parts := strings.Split(filename, "_")
		if len(parts) < 2 {
			return fmt.Errorf("invalid migration filename format: %s", filename)
		}
		
		versionStr := parts[0]
		version, err := strconv.Atoi(versionStr)
		if err != nil {
			return fmt.Errorf("invalid version number in filename %s: %w", filename, err)
		}
		
		// Extract name and direction
		name := strings.TrimSuffix(filename, ".up.sql")
		name = strings.TrimSuffix(name, ".down.sql")
		name = strings.TrimPrefix(name, versionStr+"_")
		
		isUp := strings.HasSuffix(filename, ".up.sql")
		
		// Read migration content
		content, err := migrationFiles.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", path, err)
		}
		
		// Get or create migration
		migration, exists := migrationMap[version]
		if !exists {
			migration = &Migration{
				Version: version,
				Name:    name,
			}
			migrationMap[version] = migration
		}
		
		// Set SQL content based on direction
		if isUp {
			migration.UpSQL = string(content)
		} else {
			migration.DownSQL = string(content)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to load migrations: %w", err)
	}
	
	// Convert map to sorted slice
	for _, migration := range migrationMap {
		// Validate that both up and down migrations exist
		if migration.UpSQL == "" {
			return nil, fmt.Errorf("missing up migration for version %d", migration.Version)
		}
		if migration.DownSQL == "" {
			return nil, fmt.Errorf("missing down migration for version %d", migration.Version)
		}
		migrations = append(migrations, migration)
	}
	
	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})
	
	return migrations, nil
}

// getAppliedMigrations returns all applied migrations from the database
func (m *Migrator) getAppliedMigrations(ctx context.Context) (map[int]*Migration, error) {
	appliedMigrations := make(map[int]*Migration)
	
	query := `
		SELECT version, name, applied_at, checksum 
		FROM schema_migrations 
		ORDER BY version
	`
	
	rows, err := m.db.QueryxContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var version int
		var name string
		var appliedAt time.Time
		var checksum string
		
		if err := rows.Scan(&version, &name, &appliedAt, &checksum); err != nil {
			return nil, fmt.Errorf("failed to scan migration row: %w", err)
		}
		
		appliedMigrations[version] = &Migration{
			Version:   version,
			Name:      name,
			AppliedAt: &appliedAt,
		}
	}
	
	return appliedMigrations, nil
}

// recordMigration records a migration as applied
func (m *Migrator) recordMigration(ctx context.Context, tx *sqlx.Tx, migration *Migration) error {
	checksum := m.calculateChecksum(migration.UpSQL)
	
	query := `
		INSERT INTO schema_migrations (version, name, applied_at, checksum)
		VALUES ($1, $2, NOW(), $3)
	`
	
	_, err := tx.ExecContext(ctx, query, migration.Version, migration.Name, checksum)
	if err != nil {
		return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
	}
	
	return nil
}

// removeMigrationRecord removes a migration record
func (m *Migrator) removeMigrationRecord(ctx context.Context, tx *sqlx.Tx, version int) error {
	query := `DELETE FROM schema_migrations WHERE version = $1`
	
	_, err := tx.ExecContext(ctx, query, version)
	if err != nil {
		return fmt.Errorf("failed to remove migration record %d: %w", version, err)
	}
	
	return nil
}

// calculateChecksum calculates a simple checksum for migration content
func (m *Migrator) calculateChecksum(content string) string {
	// Simple hash for content verification
	// In production, you might want to use SHA256 or similar
	hash := 0
	for _, char := range content {
		hash = ((hash << 5) - hash) + int(char)
		hash = hash & hash // Convert to 32-bit integer
	}
	return fmt.Sprintf("%x", hash)
}

// Up applies all pending migrations
func (m *Migrator) Up(ctx context.Context) error {
	// Create migrations table if it doesn't exist
	if err := m.createMigrationsTable(ctx); err != nil {
		return err
	}
	
	// Load all migrations
	migrations, err := m.loadMigrations()
	if err != nil {
		return err
	}
	
	// Get applied migrations
	appliedMigrations, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return err
	}
	
	// Apply pending migrations
	for _, migration := range migrations {
		if _, exists := appliedMigrations[migration.Version]; exists {
			fmt.Printf("Migration %d (%s) already applied, skipping\n", 
				migration.Version, migration.Name)
			continue
		}
		
		fmt.Printf("Applying migration %d: %s\n", migration.Version, migration.Name)
		
		// Execute migration in a transaction
		err := m.executeInTransaction(ctx, func(tx *sqlx.Tx) error {
			// Apply the migration
			if _, err := tx.ExecContext(ctx, migration.UpSQL); err != nil {
				return fmt.Errorf("failed to execute migration SQL: %w", err)
			}
			
			// Record the migration
			if err := m.recordMigration(ctx, tx, migration); err != nil {
				return err
			}
			
			return nil
		})
		
		if err != nil {
			return fmt.Errorf("failed to apply migration %d: %w", migration.Version, err)
		}
		
		fmt.Printf("Migration %d applied successfully\n", migration.Version)
	}
	
	fmt.Printf("All migrations applied successfully\n")
	return nil
}

// Down rolls back the latest migration
func (m *Migrator) Down(ctx context.Context) error {
	// Get applied migrations
	appliedMigrations, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return err
	}
	
	if len(appliedMigrations) == 0 {
		fmt.Printf("No migrations to roll back\n")
		return nil
	}
	
	// Find the latest migration
	latestVersion := 0
	for version := range appliedMigrations {
		if version > latestVersion {
			latestVersion = version
		}
	}
	
	// Load all migrations to get the down SQL
	migrations, err := m.loadMigrations()
	if err != nil {
		return err
	}
	
	var targetMigration *Migration
	for _, migration := range migrations {
		if migration.Version == latestVersion {
			targetMigration = migration
			break
		}
	}
	
	if targetMigration == nil {
		return fmt.Errorf("migration %d not found in migration files", latestVersion)
	}
	
	fmt.Printf("Rolling back migration %d: %s\n", latestVersion, targetMigration.Name)
	
	// Execute rollback in a transaction
	err = m.executeInTransaction(ctx, func(tx *sqlx.Tx) error {
		// Execute the down migration
		if _, err := tx.ExecContext(ctx, targetMigration.DownSQL); err != nil {
			return fmt.Errorf("failed to execute rollback SQL: %w", err)
		}
		
		// Remove the migration record
		if err := m.removeMigrationRecord(ctx, tx, latestVersion); err != nil {
			return err
		}
		
		return nil
	})
	
	if err != nil {
		return fmt.Errorf("failed to roll back migration %d: %w", latestVersion, err)
	}
	
	fmt.Printf("Migration %d rolled back successfully\n", latestVersion)
	return nil
}

// Status shows the current migration status
func (m *Migrator) Status(ctx context.Context) error {
	// Create migrations table if it doesn't exist
	if err := m.createMigrationsTable(ctx); err != nil {
		return err
	}
	
	// Load all migrations
	migrations, err := m.loadMigrations()
	if err != nil {
		return err
	}
	
	// Get applied migrations
	appliedMigrations, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return err
	}
	
	fmt.Printf("Migration Status:\n")
	fmt.Printf("================\n\n")
	
	for _, migration := range migrations {
		status := "PENDING"
		appliedAt := ""
		
		if applied, exists := appliedMigrations[migration.Version]; exists {
			status = "APPLIED"
			if applied.AppliedAt != nil {
				appliedAt = applied.AppliedAt.Format("2006-01-02 15:04:05")
			}
		}
		
		fmt.Printf("%03d %-20s %-10s %s\n", 
			migration.Version, 
			migration.Name, 
			status, 
			appliedAt)
	}
	
	pendingCount := len(migrations) - len(appliedMigrations)
	fmt.Printf("\nSummary: %d applied, %d pending\n", 
		len(appliedMigrations), 
		pendingCount)
	
	return nil
}

// executeInTransaction executes a function within a database transaction
func (m *Migrator) executeInTransaction(ctx context.Context, fn func(*sqlx.Tx) error) error {
	tx, err := m.db.BeginTxx(ctx, nil)
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