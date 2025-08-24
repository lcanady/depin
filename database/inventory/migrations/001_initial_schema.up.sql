-- Migration: 001_initial_schema
-- Description: Create initial schema for DePIN resource inventory
-- Version: 1.0.0

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enable JSONB operations extension
CREATE EXTENSION IF NOT EXISTS "btree_gin";

-- Custom types for resource management
CREATE TYPE resource_status AS ENUM (
    'unknown',
    'active', 
    'inactive',
    'maintenance',
    'offline',
    'error'
);

CREATE TYPE resource_type AS ENUM (
    'gpu',
    'cpu', 
    'storage',
    'network'
);

CREATE TYPE provider_status AS ENUM (
    'pending',
    'active',
    'inactive', 
    'suspended',
    'blocked'
);

CREATE TYPE health_status AS ENUM (
    'healthy',
    'degraded',
    'unhealthy',
    'unreachable',
    'unknown'
);

CREATE TYPE gpu_state AS ENUM (
    'unknown',
    'idle',
    'busy',
    'offline',
    'error'
);

-- Providers table
CREATE TABLE providers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    organization VARCHAR(200),
    status provider_status NOT NULL DEFAULT 'pending',
    
    -- Authentication
    api_key_hash VARCHAR(255) NOT NULL,
    public_key TEXT NOT NULL,
    
    -- Network endpoints (JSONB for flexibility)
    endpoints JSONB NOT NULL DEFAULT '[]',
    
    -- Metadata and configuration
    metadata JSONB NOT NULL DEFAULT '{}',
    
    -- Resource summary (cached aggregate data)
    resource_summary JSONB NOT NULL DEFAULT '{}',
    
    -- Heartbeat and health
    last_seen TIMESTAMPTZ,
    last_heartbeat TIMESTAMPTZ,
    heartbeat_interval_seconds INTEGER DEFAULT 300,
    health_status health_status DEFAULT 'unknown',
    consecutive_failures INTEGER DEFAULT 0,
    
    -- Registration and lifecycle
    registered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    activated_at TIMESTAMPTZ,
    suspended_at TIMESTAMPTZ,
    
    -- Performance metrics
    reputation DECIMAL(3,2) DEFAULT 0.0,
    reliability_score DECIMAL(3,2) DEFAULT 0.0,
    avg_response_time_ms INTEGER DEFAULT 0,
    uptime_percentage DECIMAL(5,2) DEFAULT 0.0,
    
    -- Resource allocation statistics
    total_allocations BIGINT DEFAULT 0,
    successful_allocations BIGINT DEFAULT 0,
    failed_allocations BIGINT DEFAULT 0,
    current_allocations INTEGER DEFAULT 0,
    
    -- Audit fields
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    version INTEGER NOT NULL DEFAULT 1
);

-- GPU resources table
CREATE TABLE gpu_resources (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider_id UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    
    -- Base resource fields
    type resource_type NOT NULL DEFAULT 'gpu',
    name VARCHAR(200) NOT NULL,
    status resource_status NOT NULL DEFAULT 'unknown',
    region VARCHAR(100),
    data_center VARCHAR(100),
    tags TEXT[] DEFAULT '{}',
    
    -- GPU identification
    uuid VARCHAR(255),
    index INTEGER,
    vendor VARCHAR(50) NOT NULL,
    
    -- Hardware specifications (JSONB for complex nested data)
    specs JSONB NOT NULL DEFAULT '{}',
    
    -- Current status (JSONB for real-time metrics)
    current_status JSONB NOT NULL DEFAULT '{}',
    
    -- Capabilities (JSONB for feature flags and limits)
    capabilities JSONB NOT NULL DEFAULT '{}',
    
    -- Driver information (JSONB for version and compatibility data)
    driver_info JSONB NOT NULL DEFAULT '{}',
    
    -- Discovery information
    discovery_source VARCHAR(100),
    last_discovered TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Verification status
    verification_status VARCHAR(50) DEFAULT 'unverified',
    last_verified TIMESTAMPTZ,
    
    -- Allocation information
    is_allocated BOOLEAN DEFAULT FALSE,
    current_allocation UUID,
    allocation_start_time TIMESTAMPTZ,
    
    -- Performance history (aggregated metrics)
    avg_utilization DECIMAL(5,2) DEFAULT 0.0,
    peak_utilization DECIMAL(5,2) DEFAULT 0.0,
    uptime_percentage DECIMAL(5,2) DEFAULT 0.0,
    
    -- Audit fields
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_heartbeat TIMESTAMPTZ,
    version INTEGER NOT NULL DEFAULT 1
);

-- GPU processes table
CREATE TABLE gpu_processes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    gpu_id UUID NOT NULL REFERENCES gpu_resources(id) ON DELETE CASCADE,
    pid INTEGER NOT NULL,
    name VARCHAR(255) NOT NULL,
    memory_usage_mb BIGINT NOT NULL,
    process_type VARCHAR(50) NOT NULL, -- compute, graphics
    start_time TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- GPU benchmarks table
CREATE TABLE gpu_benchmarks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    gpu_id UUID NOT NULL REFERENCES gpu_resources(id) ON DELETE CASCADE,
    benchmark_type VARCHAR(50) NOT NULL,
    test_name VARCHAR(200) NOT NULL,
    score DECIMAL(15,3) NOT NULL,
    unit VARCHAR(20) NOT NULL,
    duration_seconds INTEGER NOT NULL,
    metadata JSONB DEFAULT '{}',
    benchmarked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    benchmarker_version VARCHAR(50)
);

-- GPU allocations table
CREATE TABLE gpu_allocations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    gpu_id UUID NOT NULL REFERENCES gpu_resources(id) ON DELETE CASCADE,
    provider_id UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    consumer_id UUID NOT NULL,
    allocation_id VARCHAR(100) NOT NULL UNIQUE,
    status VARCHAR(50) NOT NULL DEFAULT 'allocated', -- allocated, running, completed, failed
    
    -- Timing
    allocated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    expected_end_time TIMESTAMPTZ,
    actual_end_time TIMESTAMPTZ,
    
    -- Configuration
    configuration JSONB DEFAULT '{}'
);

-- Provider heartbeats table (time-series data)
CREATE TABLE provider_heartbeats (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider_id UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status VARCHAR(50) NOT NULL,
    resource_summary JSONB NOT NULL DEFAULT '{}',
    system_metrics JSONB NOT NULL DEFAULT '{}',
    response_time_ms INTEGER,
    message TEXT,
    version VARCHAR(50)
);

-- Health checks table
CREATE TABLE health_checks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resource_id UUID NOT NULL,
    resource_type resource_type NOT NULL,
    check_type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL,
    message TEXT,
    response_time_ms INTEGER,
    metadata JSONB DEFAULT '{}',
    checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Verifications table
CREATE TABLE verifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resource_id UUID NOT NULL,
    resource_type resource_type NOT NULL,
    type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL,
    score DECIMAL(5,2),
    details JSONB DEFAULT '{}',
    verified_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    verifier_id VARCHAR(100)
);

-- Usage metrics table (time-series data)
CREATE TABLE usage_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resource_id UUID NOT NULL,
    resource_type resource_type NOT NULL,
    metric_type VARCHAR(50) NOT NULL,
    value DECIMAL(15,3) NOT NULL,
    unit VARCHAR(20) NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    collection_time TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Provider capability assessments table
CREATE TABLE provider_capability_assessments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider_id UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    assessment_type VARCHAR(50) NOT NULL,
    overall_score DECIMAL(5,2) NOT NULL,
    performance_score DECIMAL(5,2),
    reliability_score DECIMAL(5,2),
    security_score DECIMAL(5,2),
    compliance_score DECIMAL(5,2),
    details JSONB DEFAULT '{}',
    assessed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    assessed_by VARCHAR(100) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'valid'
);

-- Provider incidents table
CREATE TABLE provider_incidents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider_id UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL, -- outage, performance, security, compliance
    severity VARCHAR(20) NOT NULL, -- low, medium, high, critical
    title VARCHAR(200) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'open', -- open, investigating, resolved, closed
    metadata JSONB DEFAULT '{}',
    reported_by VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);

-- Create indexes for performance

-- Provider indexes
CREATE INDEX idx_providers_status ON providers(status);
CREATE INDEX idx_providers_health_status ON providers(health_status);
CREATE INDEX idx_providers_last_heartbeat ON providers(last_heartbeat);
CREATE INDEX idx_providers_email ON providers(email);
CREATE INDEX idx_providers_reputation ON providers(reputation);
CREATE INDEX idx_providers_metadata_gin ON providers USING GIN(metadata);

-- GPU resources indexes
CREATE INDEX idx_gpu_resources_provider_id ON gpu_resources(provider_id);
CREATE INDEX idx_gpu_resources_status ON gpu_resources(status);
CREATE INDEX idx_gpu_resources_vendor ON gpu_resources(vendor);
CREATE INDEX idx_gpu_resources_region ON gpu_resources(region);
CREATE INDEX idx_gpu_resources_is_allocated ON gpu_resources(is_allocated);
CREATE INDEX idx_gpu_resources_verification_status ON gpu_resources(verification_status);
CREATE INDEX idx_gpu_resources_last_heartbeat ON gpu_resources(last_heartbeat);
CREATE INDEX idx_gpu_resources_tags ON gpu_resources USING GIN(tags);
CREATE INDEX idx_gpu_resources_specs_gin ON gpu_resources USING GIN(specs);
CREATE INDEX idx_gpu_resources_capabilities_gin ON gpu_resources USING GIN(capabilities);

-- Composite indexes for common queries
CREATE INDEX idx_gpu_resources_provider_status ON gpu_resources(provider_id, status);
CREATE INDEX idx_gpu_resources_vendor_status ON gpu_resources(vendor, status);
CREATE INDEX idx_gpu_resources_region_status ON gpu_resources(region, status);

-- Time-series data indexes
CREATE INDEX idx_provider_heartbeats_provider_timestamp ON provider_heartbeats(provider_id, timestamp DESC);
CREATE INDEX idx_health_checks_resource_timestamp ON health_checks(resource_id, checked_at DESC);
CREATE INDEX idx_usage_metrics_resource_timestamp ON usage_metrics(resource_id, timestamp DESC);
CREATE INDEX idx_usage_metrics_type_timestamp ON usage_metrics(metric_type, timestamp DESC);

-- GPU-specific indexes
CREATE INDEX idx_gpu_processes_gpu_id ON gpu_processes(gpu_id);
CREATE INDEX idx_gpu_benchmarks_gpu_type ON gpu_benchmarks(gpu_id, benchmark_type);
CREATE INDEX idx_gpu_allocations_gpu_id ON gpu_allocations(gpu_id);
CREATE INDEX idx_gpu_allocations_status ON gpu_allocations(status);
CREATE INDEX idx_gpu_allocations_consumer_id ON gpu_allocations(consumer_id);

-- Verification indexes
CREATE INDEX idx_verifications_resource_id ON verifications(resource_id);
CREATE INDEX idx_verifications_expires_at ON verifications(expires_at);
CREATE INDEX idx_verifications_status ON verifications(status);

-- Assessment indexes
CREATE INDEX idx_capability_assessments_provider_id ON provider_capability_assessments(provider_id);
CREATE INDEX idx_capability_assessments_expires_at ON provider_capability_assessments(expires_at);

-- Incident indexes
CREATE INDEX idx_provider_incidents_provider_id ON provider_incidents(provider_id);
CREATE INDEX idx_provider_incidents_status ON provider_incidents(status);
CREATE INDEX idx_provider_incidents_severity ON provider_incidents(severity);

-- Functions for automatic timestamp updates
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for automatic timestamp updates
CREATE TRIGGER update_providers_updated_at BEFORE UPDATE ON providers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_gpu_resources_updated_at BEFORE UPDATE ON gpu_resources
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_gpu_processes_updated_at BEFORE UPDATE ON gpu_processes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_provider_incidents_updated_at BEFORE UPDATE ON provider_incidents
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to clean up old time-series data
CREATE OR REPLACE FUNCTION cleanup_old_timeseries_data(retention_days INTEGER DEFAULT 30)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER := 0;
    temp_count INTEGER;
BEGIN
    -- Clean up old heartbeats
    DELETE FROM provider_heartbeats WHERE timestamp < NOW() - INTERVAL '1 day' * retention_days;
    GET DIAGNOSTICS temp_count = ROW_COUNT;
    deleted_count := deleted_count + temp_count;
    
    -- Clean up old health checks
    DELETE FROM health_checks WHERE checked_at < NOW() - INTERVAL '1 day' * retention_days;
    GET DIAGNOSTICS temp_count = ROW_COUNT;
    deleted_count := deleted_count + temp_count;
    
    -- Clean up old usage metrics
    DELETE FROM usage_metrics WHERE collection_time < NOW() - INTERVAL '1 day' * retention_days;
    GET DIAGNOSTICS temp_count = ROW_COUNT;
    deleted_count := deleted_count + temp_count;
    
    -- Clean up expired verifications
    DELETE FROM verifications WHERE expires_at < NOW();
    GET DIAGNOSTICS temp_count = ROW_COUNT;
    deleted_count := deleted_count + temp_count;
    
    -- Clean up expired capability assessments
    DELETE FROM provider_capability_assessments WHERE expires_at < NOW() AND status != 'valid';
    GET DIAGNOSTICS temp_count = ROW_COUNT;
    deleted_count := deleted_count + temp_count;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;