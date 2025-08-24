package common

import (
	"time"

	"github.com/google/uuid"
)

// ResourceStatus represents the status of any resource in the system
type ResourceStatus string

const (
	ResourceStatusUnknown     ResourceStatus = "unknown"
	ResourceStatusActive      ResourceStatus = "active"
	ResourceStatusInactive    ResourceStatus = "inactive"
	ResourceStatusMaintenance ResourceStatus = "maintenance"
	ResourceStatusOffline     ResourceStatus = "offline"
	ResourceStatusError       ResourceStatus = "error"
)

// ResourceType represents the type of resource
type ResourceType string

const (
	ResourceTypeGPU     ResourceType = "gpu"
	ResourceTypeCPU     ResourceType = "cpu"
	ResourceTypeStorage ResourceType = "storage"
	ResourceTypeNetwork ResourceType = "network"
)

// BaseResource contains common fields for all resources
type BaseResource struct {
	ID            uuid.UUID      `json:"id" db:"id"`
	ProviderID    uuid.UUID      `json:"provider_id" db:"provider_id"`
	Type          ResourceType   `json:"type" db:"type"`
	Name          string         `json:"name" db:"name"`
	Status        ResourceStatus `json:"status" db:"status"`
	Region        string         `json:"region" db:"region"`
	DataCenter    string         `json:"data_center" db:"data_center"`
	Tags          []string       `json:"tags" db:"tags"`
	CreatedAt     time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at" db:"updated_at"`
	LastHeartbeat *time.Time     `json:"last_heartbeat" db:"last_heartbeat"`
	Version       int            `json:"version" db:"version"`
}

// HealthCheck represents a health check result
type HealthCheck struct {
	ID          uuid.UUID `json:"id" db:"id"`
	ResourceID  uuid.UUID `json:"resource_id" db:"resource_id"`
	CheckType   string    `json:"check_type" db:"check_type"`
	Status      string    `json:"status" db:"status"`
	Message     string    `json:"message" db:"message"`
	ResponseTime int32    `json:"response_time_ms" db:"response_time_ms"`
	Metadata    JSONData  `json:"metadata" db:"metadata"`
	CheckedAt   time.Time `json:"checked_at" db:"checked_at"`
}

// JSONData is a helper type for storing JSON data in database
type JSONData map[string]interface{}

// Verification represents a resource verification result
type Verification struct {
	ID           uuid.UUID `json:"id" db:"id"`
	ResourceID   uuid.UUID `json:"resource_id" db:"resource_id"`
	Type         string    `json:"type" db:"type"`
	Status       string    `json:"status" db:"status"`
	Score        float64   `json:"score" db:"score"`
	Details      JSONData  `json:"details" db:"details"`
	VerifiedAt   time.Time `json:"verified_at" db:"verified_at"`
	ExpiresAt    time.Time `json:"expires_at" db:"expires_at"`
	VerifierID   string    `json:"verifier_id" db:"verifier_id"`
}

// Usage represents resource usage metrics
type Usage struct {
	ID             uuid.UUID `json:"id" db:"id"`
	ResourceID     uuid.UUID `json:"resource_id" db:"resource_id"`
	MetricType     string    `json:"metric_type" db:"metric_type"`
	Value          float64   `json:"value" db:"value"`
	Unit           string    `json:"unit" db:"unit"`
	Timestamp      time.Time `json:"timestamp" db:"timestamp"`
	CollectionTime time.Time `json:"collection_time" db:"collection_time"`
}

// SearchFilter represents search criteria for resources
type SearchFilter struct {
	ProviderIDs    []uuid.UUID    `json:"provider_ids,omitempty"`
	ResourceTypes  []ResourceType `json:"resource_types,omitempty"`
	Statuses       []ResourceStatus `json:"statuses,omitempty"`
	Regions        []string       `json:"regions,omitempty"`
	DataCenters    []string       `json:"data_centers,omitempty"`
	Tags           []string       `json:"tags,omitempty"`
	MinMemoryMB    *int64         `json:"min_memory_mb,omitempty"`
	MaxMemoryMB    *int64         `json:"max_memory_mb,omitempty"`
	GPUVendors     []string       `json:"gpu_vendors,omitempty"`
	Architectures  []string       `json:"architectures,omitempty"`
	MinCUDACores   *int32         `json:"min_cuda_cores,omitempty"`
	SupportsCUDA   *bool          `json:"supports_cuda,omitempty"`
	SupportsOpenCL *bool          `json:"supports_opencl,omitempty"`
	CreatedAfter   *time.Time     `json:"created_after,omitempty"`
	CreatedBefore  *time.Time     `json:"created_before,omitempty"`
	LastSeenAfter  *time.Time     `json:"last_seen_after,omitempty"`
	LastSeenBefore *time.Time     `json:"last_seen_before,omitempty"`
}

// SortOption represents sorting options for search results
type SortOption struct {
	Field string `json:"field"` // name, created_at, updated_at, memory, cores, etc.
	Order string `json:"order"` // asc, desc
}

// PaginationOptions represents pagination options
type PaginationOptions struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// SearchResult represents a search result with pagination
type SearchResult[T any] struct {
	Items      []T   `json:"items"`
	Total      int64 `json:"total"`
	Limit      int   `json:"limit"`
	Offset     int   `json:"offset"`
	HasMore    bool  `json:"has_more"`
}