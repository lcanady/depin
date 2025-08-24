-- Migration: 001_initial_schema (DOWN)
-- Description: Drop initial schema for DePIN resource inventory

-- Drop functions
DROP FUNCTION IF EXISTS cleanup_old_timeseries_data(INTEGER);
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables (in reverse dependency order)
DROP TABLE IF EXISTS provider_incidents;
DROP TABLE IF EXISTS provider_capability_assessments;
DROP TABLE IF EXISTS usage_metrics;
DROP TABLE IF EXISTS verifications;
DROP TABLE IF EXISTS health_checks;
DROP TABLE IF EXISTS provider_heartbeats;
DROP TABLE IF EXISTS gpu_allocations;
DROP TABLE IF EXISTS gpu_benchmarks;
DROP TABLE IF EXISTS gpu_processes;
DROP TABLE IF EXISTS gpu_resources;
DROP TABLE IF EXISTS providers;

-- Drop custom types
DROP TYPE IF EXISTS gpu_state;
DROP TYPE IF EXISTS health_status;
DROP TYPE IF EXISTS provider_status;
DROP TYPE IF EXISTS resource_type;
DROP TYPE IF EXISTS resource_status;