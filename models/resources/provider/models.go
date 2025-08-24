package provider

import (
	"time"

	"github.com/google/uuid"
	"../common"
)

// ProviderResource represents a provider in the resource inventory
type ProviderResource struct {
	// Basic Provider Information
	ID           uuid.UUID      `json:"id" db:"id"`
	Name         string         `json:"name" db:"name"`
	Email        string         `json:"email" db:"email"`
	Organization string         `json:"organization" db:"organization"`
	Status       ProviderStatus `json:"status" db:"status"`
	
	// Authentication & Security
	ApiKeyHash   string    `json:"-" db:"api_key_hash"`
	PublicKey    string    `json:"public_key" db:"public_key"`
	
	// Network Endpoints
	Endpoints    []ProviderEndpoint `json:"endpoints" db:"endpoints"`
	
	// Metadata and Configuration
	Metadata     ProviderMetadata `json:"metadata" db:"metadata"`
	
	// Resource Summary
	ResourceSummary ProviderResourceSummary `json:"resource_summary" db:"resource_summary"`
	
	// Heartbeat and Health
	LastSeen            *time.Time    `json:"last_seen" db:"last_seen"`
	LastHeartbeat       *time.Time    `json:"last_heartbeat" db:"last_heartbeat"`
	HeartbeatInterval   int32         `json:"heartbeat_interval_seconds" db:"heartbeat_interval_seconds"`
	HealthStatus        HealthStatus  `json:"health_status" db:"health_status"`
	ConsecutiveFailures int32         `json:"consecutive_failures" db:"consecutive_failures"`
	
	// Registration and Lifecycle
	RegisteredAt time.Time `json:"registered_at" db:"registered_at"`
	ActivatedAt  *time.Time `json:"activated_at" db:"activated_at"`
	SuspendedAt  *time.Time `json:"suspended_at" db:"suspended_at"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
	Version      int       `json:"version" db:"version"`
	
	// Performance Metrics
	Reputation      float64 `json:"reputation" db:"reputation"`
	ReliabilityScore float64 `json:"reliability_score" db:"reliability_score"`
	ResponseTimeMs   int32   `json:"avg_response_time_ms" db:"avg_response_time_ms"`
	UptimePercentage float64 `json:"uptime_percentage" db:"uptime_percentage"`
	
	// Resource Allocation Statistics
	TotalAllocations      int64 `json:"total_allocations" db:"total_allocations"`
	SuccessfulAllocations int64 `json:"successful_allocations" db:"successful_allocations"`
	FailedAllocations     int64 `json:"failed_allocations" db:"failed_allocations"`
	CurrentAllocations    int32 `json:"current_allocations" db:"current_allocations"`
}

// ProviderStatus represents the current status of a provider
type ProviderStatus string

const (
	ProviderStatusPending   ProviderStatus = "pending"
	ProviderStatusActive    ProviderStatus = "active"
	ProviderStatusInactive  ProviderStatus = "inactive"
	ProviderStatusSuspended ProviderStatus = "suspended"
	ProviderStatusBlocked   ProviderStatus = "blocked"
)

// HealthStatus represents the health status of a provider
type HealthStatus string

const (
	HealthStatusHealthy     HealthStatus = "healthy"
	HealthStatusDegraded    HealthStatus = "degraded"
	HealthStatusUnhealthy   HealthStatus = "unhealthy"
	HealthStatusUnreachable HealthStatus = "unreachable"
	HealthStatusUnknown     HealthStatus = "unknown"
)

// ProviderEndpoint represents a network endpoint for a provider
type ProviderEndpoint struct {
	Type     string `json:"type" db:"type"` // api, grpc, websocket
	URL      string `json:"url" db:"url"`
	Port     int    `json:"port" db:"port"`
	Protocol string `json:"protocol" db:"protocol"`
	Secure   bool   `json:"secure" db:"secure"`
	Priority int32  `json:"priority" db:"priority"` // Lower numbers = higher priority
	IsActive bool   `json:"is_active" db:"is_active"`
}

// ProviderMetadata contains additional provider information
type ProviderMetadata struct {
	Region              string            `json:"region" db:"region"`
	DataCenter          string            `json:"data_center" db:"data_center"`
	SupportedFormats    []string          `json:"supported_formats" db:"supported_formats"`
	Certifications      []string          `json:"certifications" db:"certifications"`
	Tags                map[string]string `json:"tags" db:"tags"`
	Version             string            `json:"version" db:"version"`
	HardwareGeneration  string            `json:"hardware_generation" db:"hardware_generation"`
	SecurityLevel       string            `json:"security_level" db:"security_level"`
	ComplianceStandards []string          `json:"compliance_standards" db:"compliance_standards"`
	SLALevel            string            `json:"sla_level" db:"sla_level"`
}

// ProviderResourceSummary contains aggregate resource information
type ProviderResourceSummary struct {
	TotalGPUs           int32 `json:"total_gpus" db:"total_gpus"`
	AvailableGPUs       int32 `json:"available_gpus" db:"available_gpus"`
	AllocatedGPUs       int32 `json:"allocated_gpus" db:"allocated_gpus"`
	OfflineGPUs         int32 `json:"offline_gpus" db:"offline_gpus"`
	
	TotalMemoryMB       int64 `json:"total_memory_mb" db:"total_memory_mb"`
	AvailableMemoryMB   int64 `json:"available_memory_mb" db:"available_memory_mb"`
	AllocatedMemoryMB   int64 `json:"allocated_memory_mb" db:"allocated_memory_mb"`
	
	TotalCUDACores      int64 `json:"total_cuda_cores" db:"total_cuda_cores"`
	AvailableCUDACores  int64 `json:"available_cuda_cores" db:"available_cuda_cores"`
	
	TotalTensorCores    int64 `json:"total_tensor_cores" db:"total_tensor_cores"`
	AvailableTensorCores int64 `json:"available_tensor_cores" db:"available_tensor_cores"`
	
	GPUVendors          []string `json:"gpu_vendors" db:"gpu_vendors"`
	GPUArchitectures    []string `json:"gpu_architectures" db:"gpu_architectures"`
	
	LastResourceUpdate  time.Time `json:"last_resource_update" db:"last_resource_update"`
}

// ProviderHeartbeat represents a heartbeat record
type ProviderHeartbeat struct {
	ID               uuid.UUID `json:"id" db:"id"`
	ProviderID       uuid.UUID `json:"provider_id" db:"provider_id"`
	Timestamp        time.Time `json:"timestamp" db:"timestamp"`
	Status           string    `json:"status" db:"status"`
	ResourceSummary  ProviderResourceSummary `json:"resource_summary" db:"resource_summary"`
	SystemMetrics    SystemMetrics `json:"system_metrics" db:"system_metrics"`
	ResponseTimeMs   int32     `json:"response_time_ms" db:"response_time_ms"`
	Message          string    `json:"message" db:"message"`
	Version          string    `json:"version" db:"version"`
}

// SystemMetrics contains system-level metrics from the provider
type SystemMetrics struct {
	CPUUtilization    float64 `json:"cpu_utilization" db:"cpu_utilization"`
	MemoryUtilization float64 `json:"memory_utilization" db:"memory_utilization"`
	DiskUtilization   float64 `json:"disk_utilization" db:"disk_utilization"`
	NetworkRxBps      int64   `json:"network_rx_bps" db:"network_rx_bps"`
	NetworkTxBps      int64   `json:"network_tx_bps" db:"network_tx_bps"`
	LoadAverage       float64 `json:"load_average" db:"load_average"`
	UptimeSeconds     int64   `json:"uptime_seconds" db:"uptime_seconds"`
}

// ProviderCapabilityAssessment contains capability verification results
type ProviderCapabilityAssessment struct {
	ID                 uuid.UUID `json:"id" db:"id"`
	ProviderID         uuid.UUID `json:"provider_id" db:"provider_id"`
	AssessmentType     string    `json:"assessment_type" db:"assessment_type"`
	OverallScore       float64   `json:"overall_score" db:"overall_score"`
	PerformanceScore   float64   `json:"performance_score" db:"performance_score"`
	ReliabilityScore   float64   `json:"reliability_score" db:"reliability_score"`
	SecurityScore      float64   `json:"security_score" db:"security_score"`
	ComplianceScore    float64   `json:"compliance_score" db:"compliance_score"`
	Details            common.JSONData `json:"details" db:"details"`
	AssessedAt         time.Time `json:"assessed_at" db:"assessed_at"`
	AssessedBy         string    `json:"assessed_by" db:"assessed_by"`
	ExpiresAt          time.Time `json:"expires_at" db:"expires_at"`
	Status             string    `json:"status" db:"status"` // valid, expired, revoked
}

// ProviderIncident represents an incident or issue with a provider
type ProviderIncident struct {
	ID          uuid.UUID `json:"id" db:"id"`
	ProviderID  uuid.UUID `json:"provider_id" db:"provider_id"`
	Type        string    `json:"type" db:"type"` // outage, performance, security, compliance
	Severity    string    `json:"severity" db:"severity"` // low, medium, high, critical
	Title       string    `json:"title" db:"title"`
	Description string    `json:"description" db:"description"`
	Status      string    `json:"status" db:"status"` // open, investigating, resolved, closed
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
	ResolvedAt  *time.Time `json:"resolved_at" db:"resolved_at"`
	Metadata    common.JSONData `json:"metadata" db:"metadata"`
	ReportedBy  string    `json:"reported_by" db:"reported_by"`
}

// ProviderSearchFilter extends common search filter for provider-specific searches
type ProviderSearchFilter struct {
	common.SearchFilter
	
	// Provider-specific filters
	Organizations      []string          `json:"organizations,omitempty"`
	HealthStatuses     []HealthStatus    `json:"health_statuses,omitempty"`
	MinReputation      *float64          `json:"min_reputation,omitempty"`
	MaxReputation      *float64          `json:"max_reputation,omitempty"`
	MinReliability     *float64          `json:"min_reliability,omitempty"`
	MinUptimePercent   *float64          `json:"min_uptime_percent,omitempty"`
	MaxResponseTimeMs  *int32            `json:"max_response_time_ms,omitempty"`
	MinTotalGPUs       *int32            `json:"min_total_gpus,omitempty"`
	MinAvailableGPUs   *int32            `json:"min_available_gpus,omitempty"`
	SLALevels          []string          `json:"sla_levels,omitempty"`
	SecurityLevels     []string          `json:"security_levels,omitempty"`
	Certifications     []string          `json:"certifications,omitempty"`
	ComplianceStandards []string         `json:"compliance_standards,omitempty"`
	SupportedFormats   []string          `json:"supported_formats,omitempty"`
	RegisteredAfter    *time.Time        `json:"registered_after,omitempty"`
	RegisteredBefore   *time.Time        `json:"registered_before,omitempty"`
	HeartbeatAfter     *time.Time        `json:"heartbeat_after,omitempty"`
	HeartbeatBefore    *time.Time        `json:"heartbeat_before,omitempty"`
}

// ProviderSummary provides a summary view for listings
type ProviderSummary struct {
	ID                 uuid.UUID           `json:"id"`
	Name               string              `json:"name"`
	Organization       string              `json:"organization"`
	Status             ProviderStatus      `json:"status"`
	HealthStatus       HealthStatus        `json:"health_status"`
	Region             string              `json:"region"`
	DataCenter         string              `json:"data_center"`
	TotalGPUs          int32               `json:"total_gpus"`
	AvailableGPUs      int32               `json:"available_gpus"`
	TotalMemoryMB      int64               `json:"total_memory_mb"`
	AvailableMemoryMB  int64               `json:"available_memory_mb"`
	Reputation         float64             `json:"reputation"`
	ReliabilityScore   float64             `json:"reliability_score"`
	UptimePercentage   float64             `json:"uptime_percentage"`
	LastHeartbeat      *time.Time          `json:"last_heartbeat"`
	RegisteredAt       time.Time           `json:"registered_at"`
	GPUVendors         []string            `json:"gpu_vendors"`
	SLALevel           string              `json:"sla_level"`
}