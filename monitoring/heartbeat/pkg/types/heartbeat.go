package types

import (
	"time"

	"github.com/google/uuid"
	provider_models "github.com/lcanady/depin/models/resources/provider"
)

// Heartbeat represents a heartbeat message from a provider
type Heartbeat struct {
	ProviderID       uuid.UUID                         `json:"provider_id"`
	ProviderName     string                            `json:"provider_name"`
	Status           provider_models.HealthStatus      `json:"status"`
	SystemMetrics    *SystemMetrics                    `json:"system_metrics"`
	ResourceStatuses []*ResourceStatus                 `json:"resource_statuses"`
	Version          string                            `json:"version"`
	Metadata         map[string]string                 `json:"metadata"`
	Timestamp        time.Time                         `json:"timestamp"`
}

// HeartbeatResponse represents the response to a heartbeat
type HeartbeatResponse struct {
	Accepted               bool           `json:"accepted"`
	Message                string         `json:"message"`
	NextHeartbeatInterval  int            `json:"next_heartbeat_interval"`
	RequiredChecks         []*HealthCheck `json:"required_checks"`
	Warnings               []string       `json:"warnings"`
	ServerTimestamp        time.Time      `json:"server_timestamp"`
}

// SystemMetrics contains system-level metrics from a provider
type SystemMetrics struct {
	CPUUtilization     float64 `json:"cpu_utilization"`
	MemoryUtilization  float64 `json:"memory_utilization"`
	DiskUtilization    float64 `json:"disk_utilization"`
	NetworkRxMBPS      float64 `json:"network_rx_mbps"`
	NetworkTxMBPS      float64 `json:"network_tx_mbps"`
	LoadAverage        float64 `json:"load_average"`
	UptimeSeconds      int64   `json:"uptime_seconds"`
	TemperatureCelsius float64 `json:"temperature_celsius"`
	PowerConsumption   float64 `json:"power_consumption_watts"`
}

// ResourceStatus represents the status of a resource
type ResourceStatus struct {
	ResourceID   string        `json:"resource_id"`
	ResourceType string        `json:"resource_type"`
	Name         string        `json:"name"`
	State        ResourceState `json:"state"`
	Utilization  float64       `json:"utilization"`
	Metrics      interface{}   `json:"metrics"` // Resource-specific metrics
	Issues       []string      `json:"issues"`
	LastUpdated  time.Time     `json:"last_updated"`
}

// ResourceState defines the possible states of a resource
type ResourceState int

const (
	ResourceUnknown ResourceState = iota
	ResourceAvailable
	ResourceAllocated
	ResourceBusy
	ResourceError
	ResourceMaintenance
	ResourceOffline
)

// HealthCheck represents a health check requirement
type HealthCheck struct {
	CheckID     string            `json:"check_id"`
	CheckType   string            `json:"check_type"`
	Description string            `json:"description"`
	Frequency   int               `json:"frequency_seconds"`
	Parameters  map[string]string `json:"parameters"`
	Required    bool              `json:"required"`
}

// HeartbeatEvent represents an event in the heartbeat monitoring system
type HeartbeatEvent struct {
	Type         HeartbeatEventType             `json:"type"`
	ProviderID   uuid.UUID                      `json:"provider_id"`
	ProviderName string                         `json:"provider_name"`
	OldStatus    provider_models.HealthStatus   `json:"old_status,omitempty"`
	NewStatus    provider_models.HealthStatus   `json:"new_status"`
	ResourceID   string                         `json:"resource_id,omitempty"`
	Message      string                         `json:"message"`
	Metadata     map[string]interface{}         `json:"metadata"`
	Timestamp    time.Time                      `json:"timestamp"`
}

// HeartbeatEventType defines the types of heartbeat events
type HeartbeatEventType int

const (
	HeartbeatReceived HeartbeatEventType = iota
	StatusChanged
	ResourceChanged
	ThresholdExceeded
	ConnectionLost
	ConnectionRestored
)

// ProviderHealthStatus represents the complete health status of a provider
type ProviderHealthStatus struct {
	ProviderID            uuid.UUID                    `json:"provider_id"`
	ProviderName          string                       `json:"provider_name"`
	Status                provider_models.HealthStatus `json:"status"`
	HealthScore           float64                      `json:"health_score"`
	LatestMetrics         *SystemMetrics               `json:"latest_metrics"`
	Resources             []*ResourceStatus            `json:"resources"`
	HealthSummary         *HealthSummary               `json:"health_summary"`
	RecentIncidents       []*HealthIncident            `json:"recent_incidents"`
	LastHeartbeat         time.Time                    `json:"last_heartbeat"`
	NextExpectedHeartbeat time.Time                    `json:"next_expected_heartbeat"`
}

// HealthSummary contains aggregated health statistics
type HealthSummary struct {
	UptimePercentage                float64   `json:"uptime_percentage"`
	AvgResponseTimeMs               float64   `json:"avg_response_time_ms"`
	TotalResources                  int       `json:"total_resources"`
	HealthyResources                int       `json:"healthy_resources"`
	DegradedResources               int       `json:"degraded_resources"`
	FailedResources                 int       `json:"failed_resources"`
	ConsecutiveSuccessfulHeartbeats int       `json:"consecutive_successful_heartbeats"`
	ConsecutiveFailedHeartbeats     int       `json:"consecutive_failed_heartbeats"`
	LastIncident                    time.Time `json:"last_incident"`
}

// HealthIncident represents a health incident
type HealthIncident struct {
	IncidentID   string            `json:"incident_id"`
	ProviderID   uuid.UUID         `json:"provider_id"`
	ResourceID   string            `json:"resource_id,omitempty"`
	Type         IncidentType      `json:"type"`
	Severity     IncidentSeverity  `json:"severity"`
	Title        string            `json:"title"`
	Description  string            `json:"description"`
	Status       IncidentStatus    `json:"status"`
	OccurredAt   time.Time         `json:"occurred_at"`
	ResolvedAt   time.Time         `json:"resolved_at,omitempty"`
	LastUpdated  time.Time         `json:"last_updated"`
	Metadata     map[string]string `json:"metadata"`
}

// IncidentType defines types of health incidents
type IncidentType int

const (
	IncidentUnknown IncidentType = iota
	HeartbeatMissed
	ResourceFailure
	PerformanceDegradation
	ThresholdViolation
	ConnectivityIssue
	SecurityAlert
)

// IncidentSeverity defines incident severity levels
type IncidentSeverity int

const (
	SeverityUnknown IncidentSeverity = iota
	SeverityLow
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

// IncidentStatus defines incident status
type IncidentStatus int

const (
	IncidentStatusUnknown IncidentStatus = iota
	IncidentOpen
	IncidentInvestigating
	IncidentResolved
	IncidentClosed
)

// SystemHealthOverview represents overall system health
type SystemHealthOverview struct {
	OverallStatus      SystemStatus        `json:"overall_status"`
	OverallHealthScore float64             `json:"overall_health_score"`
	Statistics         *SystemStatistics   `json:"statistics"`
	ActiveAlerts       []*SystemAlert      `json:"active_alerts"`
	GeneratedAt        time.Time           `json:"generated_at"`
}

// SystemStatus defines overall system health status
type SystemStatus int

const (
	SystemUnknown SystemStatus = iota
	SystemHealthy
	SystemDegraded
	SystemUnhealthy
	SystemCritical
)

// SystemStatistics contains aggregated system statistics
type SystemStatistics struct {
	TotalProviders        int     `json:"total_providers"`
	HealthyProviders      int     `json:"healthy_providers"`
	DegradedProviders     int     `json:"degraded_providers"`
	UnhealthyProviders    int     `json:"unhealthy_providers"`
	OfflineProviders      int     `json:"offline_providers"`
	TotalResources        int     `json:"total_resources"`
	AvailableResources    int     `json:"available_resources"`
	AllocatedResources    int     `json:"allocated_resources"`
	ErrorResources        int     `json:"error_resources"`
	AvgUptimePercentage   float64 `json:"avg_uptime_percentage"`
	AvgResponseTimeMs     float64 `json:"avg_response_time_ms"`
}

// SystemAlert represents a system-level alert
type SystemAlert struct {
	AlertID            string             `json:"alert_id"`
	Title              string             `json:"title"`
	Description        string             `json:"description"`
	Severity           AlertSeverity      `json:"severity"`
	Status             AlertStatus        `json:"status"`
	AffectedProviders  []string           `json:"affected_providers"`
	AffectedResources  []string           `json:"affected_resources"`
	TriggeredAt        time.Time          `json:"triggered_at"`
	Metadata           map[string]string  `json:"metadata"`
}

// AlertSeverity defines alert severity levels
type AlertSeverity int

const (
	AlertSeverityUnknown AlertSeverity = iota
	AlertInfo
	AlertWarning
	AlertError
	AlertCritical
)

// AlertStatus defines alert status
type AlertStatus int

const (
	AlertStatusUnknown AlertStatus = iota
	AlertActive
	AlertAcknowledged
	AlertSuppressed
	AlertResolved
)

// ResourceAvailability represents resource availability information
type ResourceAvailability struct {
	ResourceID         string               `json:"resource_id"`
	ResourceType       string               `json:"resource_type"`
	ProviderID         uuid.UUID            `json:"provider_id"`
	ProviderName       string               `json:"provider_name"`
	Name               string               `json:"name"`
	State              ResourceState        `json:"state"`
	Capabilities       *ResourceCapabilities `json:"capabilities"`
	CurrentMetrics     interface{}          `json:"current_metrics"`
	HealthScore        float64              `json:"health_score"`
	LastUpdated        time.Time            `json:"last_updated"`
}

// ResourceCapabilities describes resource capabilities
type ResourceCapabilities struct {
	// GPU capabilities
	GPUVendor        string   `json:"gpu_vendor,omitempty"`
	GPUArchitecture  string   `json:"gpu_architecture,omitempty"`
	MemoryMB         int64    `json:"memory_mb,omitempty"`
	CUDACores        int32    `json:"cuda_cores,omitempty"`
	TensorCores      int32    `json:"tensor_cores,omitempty"`
	SupportedAPIs    []string `json:"supported_apis,omitempty"`
	
	// CPU capabilities
	CPUCores         int32    `json:"cpu_cores,omitempty"`
	CPUThreads       int32    `json:"cpu_threads,omitempty"`
	CPUFrequencyGHz  float64  `json:"cpu_frequency_ghz,omitempty"`
	CPUArchitecture  string   `json:"cpu_architecture,omitempty"`
	
	// Memory capabilities
	TotalMemoryMB    int64    `json:"total_memory_mb,omitempty"`
	MemoryType       string   `json:"memory_type,omitempty"`
	
	// Storage capabilities
	StorageCapacityGB int64   `json:"storage_capacity_gb,omitempty"`
	StorageType       string  `json:"storage_type,omitempty"`
	MaxIOPS           float64 `json:"max_iops,omitempty"`
}

// AvailabilityEvent represents an availability change event  
type AvailabilityEvent struct {
	EventType    AvailabilityEventType `json:"event_type"`
	ResourceID   string                `json:"resource_id"`
	ResourceType string                `json:"resource_type"`
	ProviderID   uuid.UUID             `json:"provider_id"`
	ProviderName string                `json:"provider_name"`
	OldState     ResourceState         `json:"old_state"`
	NewState     ResourceState         `json:"new_state"`
	Capabilities *ResourceCapabilities `json:"capabilities,omitempty"`
	Region       string                `json:"region"`
	Message      string                `json:"message"`
	Timestamp    time.Time             `json:"timestamp"`
}

// AvailabilityEventType defines the types of availability events
type AvailabilityEventType int

const (
	AvailabilityResourceAvailable AvailabilityEventType = iota
	AvailabilityResourceAllocated
	AvailabilityResourceReleased
	AvailabilityResourceDegraded
	AvailabilityResourceRestored
	AvailabilityResourceOffline
)