package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"../../models/resources/common"
	"../../models/resources/gpu"
	"../../models/resources/provider"
)

// HealthCheck interface for database health monitoring
type HealthCheck interface {
	IsHealthy(ctx context.Context) bool
	GetLastHealthCheck() time.Time
	GetStats() map[string]interface{}
}

// BaseRepository defines common operations for all resources
type BaseRepository[T any] interface {
	// CRUD operations
	Create(ctx context.Context, entity *T) error
	GetByID(ctx context.Context, id uuid.UUID) (*T, error)
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id uuid.UUID) error
	
	// Bulk operations
	CreateBatch(ctx context.Context, entities []*T) error
	UpdateBatch(ctx context.Context, entities []*T) error
	DeleteBatch(ctx context.Context, ids []uuid.UUID) error
	
	// Counting
	Count(ctx context.Context) (int64, error)
	
	// Health monitoring
	HealthCheck
}

// SearchableRepository adds search capabilities
type SearchableRepository[T any, F any] interface {
	BaseRepository[T]
	
	// Search operations
	Search(ctx context.Context, filter *F, sort *common.SortOption, pagination *common.PaginationOptions) (*common.SearchResult[T], error)
	List(ctx context.Context, limit, offset int) ([]T, error)
	
	// Filtering
	CountByFilter(ctx context.Context, filter *F) (int64, error)
}

// ProviderRepository defines operations for provider resources
type ProviderRepository interface {
	SearchableRepository[provider.ProviderResource, provider.ProviderSearchFilter]
	
	// Provider-specific operations
	GetByEmail(ctx context.Context, email string) (*provider.ProviderResource, error)
	GetByApiKeyHash(ctx context.Context, apiKeyHash string) (*provider.ProviderResource, error)
	ListByStatus(ctx context.Context, status provider.ProviderStatus) ([]provider.ProviderResource, error)
	ListByHealthStatus(ctx context.Context, healthStatus provider.HealthStatus) ([]provider.ProviderResource, error)
	
	// Heartbeat operations
	RecordHeartbeat(ctx context.Context, heartbeat *provider.ProviderHeartbeat) error
	GetLatestHeartbeat(ctx context.Context, providerID uuid.UUID) (*provider.ProviderHeartbeat, error)
	GetHeartbeatHistory(ctx context.Context, providerID uuid.UUID, since time.Time, limit int) ([]provider.ProviderHeartbeat, error)
	CleanupOldHeartbeats(ctx context.Context, olderThan time.Time) (int64, error)
	
	// Status updates
	UpdateLastSeen(ctx context.Context, providerID uuid.UUID, timestamp time.Time) error
	UpdateHealthStatus(ctx context.Context, providerID uuid.UUID, status provider.HealthStatus) error
	UpdateResourceSummary(ctx context.Context, providerID uuid.UUID, summary *provider.ProviderResourceSummary) error
	IncrementConsecutiveFailures(ctx context.Context, providerID uuid.UUID) error
	ResetConsecutiveFailures(ctx context.Context, providerID uuid.UUID) error
	
	// Performance metrics
	UpdateReputationScore(ctx context.Context, providerID uuid.UUID, reputation float64) error
	UpdateReliabilityScore(ctx context.Context, providerID uuid.UUID, reliability float64) error
	UpdateResponseTime(ctx context.Context, providerID uuid.UUID, responseTimeMs int32) error
	UpdateUptimePercentage(ctx context.Context, providerID uuid.UUID, uptime float64) error
	
	// Allocation statistics
	IncrementAllocationCounters(ctx context.Context, providerID uuid.UUID, successful bool) error
	UpdateCurrentAllocations(ctx context.Context, providerID uuid.UUID, count int32) error
	
	// Capability assessments
	CreateCapabilityAssessment(ctx context.Context, assessment *provider.ProviderCapabilityAssessment) error
	GetLatestCapabilityAssessment(ctx context.Context, providerID uuid.UUID, assessmentType string) (*provider.ProviderCapabilityAssessment, error)
	ListCapabilityAssessments(ctx context.Context, providerID uuid.UUID, limit int) ([]provider.ProviderCapabilityAssessment, error)
	
	// Incidents
	CreateIncident(ctx context.Context, incident *provider.ProviderIncident) error
	UpdateIncident(ctx context.Context, incident *provider.ProviderIncident) error
	GetIncident(ctx context.Context, incidentID uuid.UUID) (*provider.ProviderIncident, error)
	ListIncidents(ctx context.Context, providerID uuid.UUID, status string, limit int) ([]provider.ProviderIncident, error)
	
	// Summary operations
	GetSummary(ctx context.Context, providerID uuid.UUID) (*provider.ProviderSummary, error)
	ListSummaries(ctx context.Context, filter *provider.ProviderSearchFilter, limit, offset int) ([]provider.ProviderSummary, error)
}

// GPURepository defines operations for GPU resources
type GPURepository interface {
	SearchableRepository[gpu.GPUResource, gpu.GPUSearchFilter]
	
	// GPU-specific operations
	GetByProviderID(ctx context.Context, providerID uuid.UUID) ([]gpu.GPUResource, error)
	GetByUUID(ctx context.Context, uuid string) (*gpu.GPUResource, error)
	ListByVendor(ctx context.Context, vendor string) ([]gpu.GPUResource, error)
	ListByStatus(ctx context.Context, status common.ResourceStatus) ([]gpu.GPUResource, error)
	ListAvailable(ctx context.Context, region string) ([]gpu.GPUResource, error)
	
	// Allocation operations
	MarkAsAllocated(ctx context.Context, gpuID uuid.UUID, allocationID uuid.UUID, startTime time.Time) error
	MarkAsReleased(ctx context.Context, gpuID uuid.UUID, endTime time.Time) error
	GetAllocatedGPUs(ctx context.Context, providerID *uuid.UUID) ([]gpu.GPUResource, error)
	
	// Status updates
	UpdateCurrentStatus(ctx context.Context, gpuID uuid.UUID, status *gpu.GPUCurrentStatus) error
	UpdateLastDiscovered(ctx context.Context, gpuID uuid.UUID, timestamp time.Time) error
	UpdateVerificationStatus(ctx context.Context, gpuID uuid.UUID, status string, verifiedAt *time.Time) error
	
	// Performance metrics
	UpdateUtilizationMetrics(ctx context.Context, gpuID uuid.UUID, avg, peak float64) error
	UpdateUptimePercentage(ctx context.Context, gpuID uuid.UUID, uptime float64) error
	
	// Process management
	CreateProcess(ctx context.Context, process *gpu.GPUProcess) error
	UpdateProcess(ctx context.Context, process *gpu.GPUProcess) error
	DeleteProcess(ctx context.Context, processID uuid.UUID) error
	GetProcessesByGPU(ctx context.Context, gpuID uuid.UUID) ([]gpu.GPUProcess, error)
	CleanupStaleProcesses(ctx context.Context, olderThan time.Time) (int64, error)
	
	// Benchmark operations
	CreateBenchmark(ctx context.Context, benchmark *gpu.GPUBenchmark) error
	GetBenchmarksByGPU(ctx context.Context, gpuID uuid.UUID, benchmarkType string, limit int) ([]gpu.GPUBenchmark, error)
	GetLatestBenchmark(ctx context.Context, gpuID uuid.UUID, benchmarkType string) (*gpu.GPUBenchmark, error)
	DeleteOldBenchmarks(ctx context.Context, olderThan time.Time) (int64, error)
	
	// Allocation management
	CreateAllocation(ctx context.Context, allocation *gpu.GPUAllocation) error
	UpdateAllocation(ctx context.Context, allocation *gpu.GPUAllocation) error
	GetAllocation(ctx context.Context, allocationID string) (*gpu.GPUAllocation, error)
	GetAllocationsByGPU(ctx context.Context, gpuID uuid.UUID, limit int) ([]gpu.GPUAllocation, error)
	GetActiveAllocations(ctx context.Context, providerID *uuid.UUID, consumerID *uuid.UUID) ([]gpu.GPUAllocation, error)
	
	// Hardware specification queries
	GetByMemoryRange(ctx context.Context, minMB, maxMB int64) ([]gpu.GPUResource, error)
	GetByCUDACore sRange(ctx context.Context, minCores, maxCores int32) ([]gpu.GPUResource, error)
	GetByArchitecture(ctx context.Context, architecture string) ([]gpu.GPUResource, error)
	GetByCapability(ctx context.Context, capability string, required bool) ([]gpu.GPUResource, error)
	
	// Summary operations
	GetSummary(ctx context.Context, gpuID uuid.UUID) (*gpu.GPUResourceSummary, error)
	ListSummaries(ctx context.Context, filter *gpu.GPUSearchFilter, limit, offset int) ([]gpu.GPUResourceSummary, error)
	
	// Aggregation operations
	GetResourceStatsByProvider(ctx context.Context, providerID uuid.UUID) (map[string]interface{}, error)
	GetResourceStatsByRegion(ctx context.Context, region string) (map[string]interface{}, error)
	GetUtilizationStatistics(ctx context.Context, timeRange time.Duration) (map[string]interface{}, error)
}

// HealthCheckRepository defines operations for health checks
type HealthCheckRepository interface {
	BaseRepository[common.HealthCheck]
	
	// Health check operations
	CreateHealthCheck(ctx context.Context, healthCheck *common.HealthCheck) error
	GetLatestHealthCheck(ctx context.Context, resourceID uuid.UUID, checkType string) (*common.HealthCheck, error)
	GetHealthCheckHistory(ctx context.Context, resourceID uuid.UUID, since time.Time, limit int) ([]common.HealthCheck, error)
	ListFailedHealthChecks(ctx context.Context, since time.Time, limit int) ([]common.HealthCheck, error)
	CleanupOldHealthChecks(ctx context.Context, olderThan time.Time) (int64, error)
}

// VerificationRepository defines operations for resource verifications
type VerificationRepository interface {
	BaseRepository[common.Verification]
	
	// Verification operations
	CreateVerification(ctx context.Context, verification *common.Verification) error
	GetLatestVerification(ctx context.Context, resourceID uuid.UUID, verificationType string) (*common.Verification, error)
	GetVerificationHistory(ctx context.Context, resourceID uuid.UUID, limit int) ([]common.Verification, error)
	ListExpiredVerifications(ctx context.Context, asOf time.Time) ([]common.Verification, error)
	MarkExpiredVerifications(ctx context.Context, asOf time.Time) (int64, error)
	
	// Verification statistics
	GetVerificationStats(ctx context.Context, timeRange time.Duration) (map[string]interface{}, error)
}

// UsageMetricsRepository defines operations for usage metrics
type UsageMetricsRepository interface {
	BaseRepository[common.Usage]
	
	// Usage metrics operations
	RecordUsage(ctx context.Context, usage *common.Usage) error
	RecordUsageBatch(ctx context.Context, usageMetrics []*common.Usage) error
	GetUsageHistory(ctx context.Context, resourceID uuid.UUID, metricType string, since time.Time, limit int) ([]common.Usage, error)
	GetLatestUsage(ctx context.Context, resourceID uuid.UUID, metricType string) (*common.Usage, error)
	
	// Aggregation operations
	GetAverageUsage(ctx context.Context, resourceID uuid.UUID, metricType string, timeRange time.Duration) (float64, error)
	GetPeakUsage(ctx context.Context, resourceID uuid.UUID, metricType string, timeRange time.Duration) (float64, error)
	GetUsageStatistics(ctx context.Context, resourceIDs []uuid.UUID, timeRange time.Duration) (map[string]interface{}, error)
	
	// Cleanup operations
	CleanupOldUsageMetrics(ctx context.Context, olderThan time.Time) (int64, error)
}

// RepositoryManager provides access to all repositories
type RepositoryManager interface {
	// Repository access
	Providers() ProviderRepository
	GPUs() GPURepository
	HealthChecks() HealthCheckRepository
	Verifications() VerificationRepository
	UsageMetrics() UsageMetricsRepository
	
	// Lifecycle management
	Close() error
	
	// Health monitoring
	HealthCheck
	
	// Transaction support
	WithTransaction(ctx context.Context, fn func(RepositoryManager) error) error
}

// CacheRepository defines caching operations
type CacheRepository interface {
	// Basic cache operations
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string) (interface{}, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	
	// Pattern operations
	DeletePattern(ctx context.Context, pattern string) (int64, error)
	Keys(ctx context.Context, pattern string) ([]string, error)
	
	// Expiry operations
	SetTTL(ctx context.Context, key string, ttl time.Duration) error
	GetTTL(ctx context.Context, key string) (time.Duration, error)
	
	// Cache statistics
	GetStats(ctx context.Context) (map[string]interface{}, error)
	
	// Health monitoring
	IsHealthy(ctx context.Context) bool
}