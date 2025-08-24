package types

import (
	"time"

	"github.com/google/uuid"
)

// VerificationRequest represents a request to verify capabilities
type VerificationRequest struct {
	ID           uuid.UUID       `json:"id"`
	ResourceID   uuid.UUID       `json:"resource_id"`
	ResourceType string          `json:"resource_type"` // gpu, provider, system
	Level        VerificationLevel `json:"level"`
	TestsToRun   []string        `json:"tests_to_run"`
	Config       *VerificationConfig `json:"config,omitempty"`
	RequestedBy  string          `json:"requested_by"`
	RequestedAt  time.Time       `json:"requested_at"`
	Priority     int             `json:"priority"` // 1-10, higher is more urgent
}

// VerificationConfig contains configuration for verification
type VerificationConfig struct {
	Duration            int               `json:"duration_seconds"`
	Iterations          int               `json:"iterations"`
	StressTest          bool              `json:"stress_test"`
	RequireFreshData    bool              `json:"require_fresh_data"`
	ComplianceStandards []string          `json:"compliance_standards"`
	Parameters          map[string]string `json:"parameters"`
}

// VerificationLevel defines the depth of verification
type VerificationLevel int

const (
	BasicLevel VerificationLevel = iota
	StandardLevel
	ComprehensiveLevel
	CertificationLevel
)

// VerificationStatus represents the current status of verification
type VerificationStatus int

const (
	StatusPending VerificationStatus = iota
	StatusRunning
	StatusCompleted
	StatusFailed
	StatusExpired
	StatusCancelled
)

// VerificationResult contains the results of verification
type VerificationResult struct {
	ID             uuid.UUID                  `json:"id"`
	RequestID      uuid.UUID                  `json:"request_id"`
	ResourceID     uuid.UUID                  `json:"resource_id"`
	ResourceType   string                     `json:"resource_type"`
	Status         VerificationStatus         `json:"status"`
	Level          VerificationLevel          `json:"level"`
	OverallScore   float64                    `json:"overall_score"`
	Grade          string                     `json:"grade"` // A, B, C, D, F
	Assessment     *CapabilityAssessment      `json:"assessment"`
	BenchmarkResults []*BenchmarkResult       `json:"benchmark_results"`
	ComplianceResults []*ComplianceResult     `json:"compliance_results"`
	Issues         []*VerificationIssue       `json:"issues"`
	Recommendations []string                  `json:"recommendations"`
	StartedAt      time.Time                  `json:"started_at"`
	CompletedAt    *time.Time                 `json:"completed_at"`
	ExpiresAt      time.Time                  `json:"expires_at"`
	Duration       time.Duration              `json:"duration"`
	Metadata       map[string]interface{}     `json:"metadata"`
}

// CapabilityAssessment contains detailed capability analysis
type CapabilityAssessment struct {
	OverallScore    float64                  `json:"overall_score"`
	Compute         *ComputeCapability       `json:"compute,omitempty"`
	Memory          *MemoryCapability        `json:"memory,omitempty"`
	Tensor          *TensorCapability        `json:"tensor,omitempty"`
	Stability       *StabilityCapability     `json:"stability,omitempty"`
	Compatibility   *CompatibilityCapability `json:"compatibility,omitempty"`
	Infrastructure  *InfrastructureCapability `json:"infrastructure,omitempty"`
	Security        *SecurityCapability      `json:"security,omitempty"`
	Reliability     *ReliabilityCapability   `json:"reliability,omitempty"`
	Performance     *PerformanceCapability   `json:"performance,omitempty"`
	Tier            string                   `json:"tier"`
	Certifications  []string                 `json:"certifications"`
	AssessedAt      time.Time                `json:"assessed_at"`
	ValidUntil      time.Time                `json:"valid_until"`
}

// Individual capability types
type ComputeCapability struct {
	Score               float64 `json:"score"`
	FP32Performance     float64 `json:"fp32_performance"`
	FP16Performance     float64 `json:"fp16_performance"`
	INT8Performance     float64 `json:"int8_performance"`
	ParallelEfficiency  float64 `json:"parallel_efficiency"`
	ThroughputGFLOPS    float64 `json:"throughput_gflops"`
	MeetsBaseline       bool    `json:"meets_baseline"`
}

type MemoryCapability struct {
	Score           float64 `json:"score"`
	BandwidthScore  float64 `json:"bandwidth_score"`
	LatencyScore    float64 `json:"latency_score"`
	CapacityScore   float64 `json:"capacity_score"`
	ECCReliability  float64 `json:"ecc_reliability"`
	ThroughputGBps  float64 `json:"throughput_gbps"`
	LatencyNs       float64 `json:"latency_ns"`
	MeetsBaseline   bool    `json:"meets_baseline"`
}

type TensorCapability struct {
	Score                   float64 `json:"score"`
	TensorPerformance       float64 `json:"tensor_performance"`
	MixedPrecisionSpeedup   float64 `json:"mixed_precision_speedup"`
	AIWorkloadEfficiency    float64 `json:"ai_workload_efficiency"`
	TensorCoreSupport       bool    `json:"tensor_core_support"`
	ThroughputTOPS          float64 `json:"throughput_tops"`
	MeetsBaseline           bool    `json:"meets_baseline"`
}

type StabilityCapability struct {
	Score                   float64 `json:"score"`
	UptimePercentage        float64 `json:"uptime_percentage"`
	ErrorRate               float64 `json:"error_rate"`
	ThermalStability        float64 `json:"thermal_stability"`
	PowerStability          float64 `json:"power_stability"`
	ConsecutiveStableHours  int     `json:"consecutive_stable_hours"`
	MeetsBaseline           bool    `json:"meets_baseline"`
}

type CompatibilityCapability struct {
	Score               int      `json:"score"`
	SupportedAPIs       int      `json:"supported_apis"`
	DriverCompatibility int      `json:"driver_compatibility"`
	FrameworkSupport    int      `json:"framework_support"`
	SupportedFrameworks []string `json:"supported_frameworks"`
	MeetsBaseline       bool     `json:"meets_baseline"`
}

// Provider-specific capabilities
type InfrastructureCapability struct {
	Score              float64 `json:"score"`
	NetworkPerformance float64 `json:"network_performance"`
	StoragePerformance float64 `json:"storage_performance"`
	CoolingEfficiency  float64 `json:"cooling_efficiency"`
	PowerReliability   float64 `json:"power_reliability"`
	MeetsBaseline      bool    `json:"meets_baseline"`
}

type SecurityCapability struct {
	Score              float64 `json:"score"`
	EncryptionGrade    float64 `json:"encryption_grade"`
	AccessControlGrade float64 `json:"access_control_grade"`
	AuditCompliance    float64 `json:"audit_compliance"`
	VulnerabilityScore float64 `json:"vulnerability_score"`
	MeetsBaseline      bool    `json:"meets_baseline"`
}

type ReliabilityCapability struct {
	Score            float64 `json:"score"`
	UptimePercentage float64 `json:"uptime_percentage"`
	MTBFHours        float64 `json:"mtbf_hours"`
	RecoveryTime     float64 `json:"recovery_time"`
	DataIntegrity    float64 `json:"data_integrity"`
	MeetsBaseline    bool    `json:"meets_baseline"`
}

type PerformanceCapability struct {
	Score                     float64 `json:"score"`
	AggregateGPUPerformance   float64 `json:"aggregate_gpu_performance"`
	NetworkThroughput         float64 `json:"network_throughput"`
	StorageIOPS               float64 `json:"storage_iops"`
	ResponseTime              float64 `json:"response_time"`
	MeetsBaseline             bool    `json:"meets_baseline"`
}

// BenchmarkResult contains results from a single benchmark
type BenchmarkResult struct {
	ID               uuid.UUID              `json:"id"`
	VerificationID   uuid.UUID              `json:"verification_id"`
	ResourceID       uuid.UUID              `json:"resource_id"`
	TestName         string                 `json:"test_name"`
	TestType         string                 `json:"test_type"`
	Score            float64                `json:"score"`
	Unit             string                 `json:"unit"`
	BaselineScore    float64                `json:"baseline_score"`
	PerformanceRatio float64                `json:"performance_ratio"`
	Passed           bool                   `json:"passed"`
	Metrics          []*Metric              `json:"metrics"`
	StartedAt        time.Time              `json:"started_at"`
	CompletedAt      time.Time              `json:"completed_at"`
	Duration         time.Duration          `json:"duration"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// Metric represents a performance metric
type Metric struct {
	Name         string  `json:"name"`
	Value        float64 `json:"value"`
	Unit         string  `json:"unit"`
	MinThreshold float64 `json:"min_threshold"`
	MaxThreshold float64 `json:"max_threshold"`
	WithinLimits bool    `json:"within_limits"`
}

// ComplianceResult contains compliance assessment results
type ComplianceResult struct {
	Standard            string    `json:"standard"`
	Compliant           bool      `json:"compliant"`
	CompliancePercentage float64  `json:"compliance_percentage"`
	Issues              []string  `json:"issues"`
	Recommendations     []string  `json:"recommendations"`
	AssessedAt          time.Time `json:"assessed_at"`
	ValidUntil          time.Time `json:"valid_until"`
}

// VerificationIssue represents an issue found during verification
type VerificationIssue struct {
	Severity        IssueSeverity          `json:"severity"`
	Category        string                 `json:"category"`
	Title           string                 `json:"title"`
	Description     string                 `json:"description"`
	Recommendations []string               `json:"recommendations"`
	Blocking        bool                   `json:"blocking"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// IssueSeverity defines the severity of a verification issue
type IssueSeverity int

const (
	SeverityInfo IssueSeverity = iota
	SeverityWarning
	SeverityError
	SeverityCritical
)

// AllocationCompatibility contains allocation validation results
type AllocationCompatibility struct {
	IsCompatible      bool                  `json:"is_compatible"`
	CompatibilityScore float64              `json:"compatibility_score"`
	Issues            []string              `json:"issues"`
	Warnings          []string              `json:"warnings"`
	CapabilityMatch   *CapabilityMatch      `json:"capability_match"`
	ValidatedAt       time.Time             `json:"validated_at"`
}

// CapabilityMatch contains detailed matching results
type CapabilityMatch struct {
	MemorySufficient      bool    `json:"memory_sufficient"`
	ComputeSufficient     bool    `json:"compute_sufficient"`
	ArchitectureCompatible bool   `json:"architecture_compatible"`
	APIsSupported         bool    `json:"apis_supported"`
	FrameworksSupported   bool    `json:"frameworks_supported"`
	PerformanceAdequate   bool    `json:"performance_adequate"`
	ReliabilityAdequate   bool    `json:"reliability_adequate"`
	OverallMatchScore     float64 `json:"overall_match_score"`
}

// VerificationEvent represents a verification event for streaming
type VerificationEvent struct {
	Type           EventType              `json:"type"`
	VerificationID uuid.UUID              `json:"verification_id"`
	ResourceID     uuid.UUID              `json:"resource_id"`
	Message        string                 `json:"message"`
	Timestamp      time.Time              `json:"timestamp"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// EventType defines verification event types
type EventType int

const (
	EventUnknown EventType = iota
	EventVerificationStarted
	EventVerificationCompleted
	EventVerificationFailed
	EventBenchmarkCompleted
	EventIssueDetected
	EventStatusUpdate
)