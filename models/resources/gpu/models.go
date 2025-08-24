package gpu

import (
	"time"

	"github.com/google/uuid"
	"../common"
)

// GPUResource represents a GPU resource in the inventory database
type GPUResource struct {
	common.BaseResource
	
	// GPU Identification
	UUID                string `json:"uuid" db:"uuid"`
	Index               int32  `json:"index" db:"index"`
	Vendor              string `json:"vendor" db:"vendor"`
	
	// Hardware Specifications
	Specs               GPUSpecs `json:"specs" db:"specs"`
	
	// Current Status
	CurrentStatus       GPUCurrentStatus `json:"current_status" db:"current_status"`
	
	// Capabilities
	Capabilities        GPUCapabilities `json:"capabilities" db:"capabilities"`
	
	// Driver Information
	DriverInfo          GPUDriverInfo `json:"driver_info" db:"driver_info"`
	
	// Discovery Information
	DiscoverySource     string    `json:"discovery_source" db:"discovery_source"`
	LastDiscovered      time.Time `json:"last_discovered" db:"last_discovered"`
	
	// Verification Status
	VerificationStatus  string    `json:"verification_status" db:"verification_status"`
	LastVerified        *time.Time `json:"last_verified" db:"last_verified"`
	
	// Allocation Information
	IsAllocated         bool      `json:"is_allocated" db:"is_allocated"`
	CurrentAllocation   *uuid.UUID `json:"current_allocation" db:"current_allocation"`
	AllocationStartTime *time.Time `json:"allocation_start_time" db:"allocation_start_time"`
	
	// Performance History
	AverageUtilization  float64   `json:"avg_utilization" db:"avg_utilization"`
	PeakUtilization     float64   `json:"peak_utilization" db:"peak_utilization"`
	UptimePercentage    float64   `json:"uptime_percentage" db:"uptime_percentage"`
}

// GPUSpecs contains detailed hardware specifications
type GPUSpecs struct {
	// Memory
	MemoryTotalMB       int64  `json:"memory_total_mb" db:"memory_total_mb"`
	MemoryBandwidthGBPS int64  `json:"memory_bandwidth_gbps" db:"memory_bandwidth_gbps"`
	
	// Compute Units (vendor-specific)
	CUDACores           int32  `json:"cuda_cores" db:"cuda_cores"`
	StreamProcessors    int32  `json:"stream_processors" db:"stream_processors"`
	ExecutionUnits      int32  `json:"execution_units" db:"execution_units"`
	TensorCores         int32  `json:"tensor_cores" db:"tensor_cores"`
	
	// Clock Speeds (MHz)
	BaseClockMHz        int32  `json:"base_clock_mhz" db:"base_clock_mhz"`
	BoostClockMHz       int32  `json:"boost_clock_mhz" db:"boost_clock_mhz"`
	MemoryClockMHz      int32  `json:"memory_clock_mhz" db:"memory_clock_mhz"`
	
	// Architecture
	Architecture        string `json:"architecture" db:"architecture"`
	ComputeCapability   string `json:"compute_capability" db:"compute_capability"`
	SMCount             int32  `json:"sm_count" db:"sm_count"`
	
	// Power
	PowerLimitWatts        int32 `json:"power_limit_watts" db:"power_limit_watts"`
	DefaultPowerLimitWatts int32 `json:"default_power_limit_watts" db:"default_power_limit_watts"`
	
	// Connectivity
	BusType             string `json:"bus_type" db:"bus_type"`
	BusWidth            string `json:"bus_width" db:"bus_width"`
	PCIeGeneration      int32  `json:"pcie_generation" db:"pcie_generation"`
}

// GPUCurrentStatus contains real-time status information
type GPUCurrentStatus struct {
	State               string  `json:"state" db:"state"` // idle, busy, offline, error
	
	// Utilization (0-100%)
	GPUUtilization      int32   `json:"gpu_utilization" db:"gpu_utilization"`
	MemoryUtilization   int32   `json:"memory_utilization" db:"memory_utilization"`
	
	// Memory Usage
	MemoryUsedMB        int64   `json:"memory_used_mb" db:"memory_used_mb"`
	MemoryFreeMB        int64   `json:"memory_free_mb" db:"memory_free_mb"`
	
	// Temperature (Celsius)
	TemperatureGPU      int32   `json:"temperature_gpu" db:"temperature_gpu"`
	TemperatureMemory   int32   `json:"temperature_memory" db:"temperature_memory"`
	
	// Power Usage
	PowerDrawWatts      int32   `json:"power_draw_watts" db:"power_draw_watts"`
	
	// Current Clock Speeds
	CurrentGPUClockMHz    int32 `json:"current_gpu_clock_mhz" db:"current_gpu_clock_mhz"`
	CurrentMemoryClockMHz int32 `json:"current_memory_clock_mhz" db:"current_memory_clock_mhz"`
	
	// Status Timestamp
	StatusUpdatedAt     time.Time `json:"status_updated_at" db:"status_updated_at"`
}

// GPUCapabilities describes what the GPU can do
type GPUCapabilities struct {
	// Compute APIs
	SupportsCUDA         bool     `json:"supports_cuda" db:"supports_cuda"`
	SupportsOpenCL       bool     `json:"supports_opencl" db:"supports_opencl"`
	SupportsVulkan       bool     `json:"supports_vulkan" db:"supports_vulkan"`
	SupportsDirectX      bool     `json:"supports_directx" db:"supports_directx"`
	
	// AI/ML Features
	SupportsTensorOps    bool     `json:"supports_tensor_ops" db:"supports_tensor_ops"`
	SupportsMixedPrecision bool   `json:"supports_mixed_precision" db:"supports_mixed_precision"`
	SupportsRayTracing   bool     `json:"supports_ray_tracing" db:"supports_ray_tracing"`
	
	// Memory Features
	SupportsECC          bool     `json:"supports_ecc" db:"supports_ecc"`
	ECCEnabled           bool     `json:"ecc_enabled" db:"ecc_enabled"`
	SupportsUnifiedMemory bool    `json:"supports_unified_memory" db:"supports_unified_memory"`
	
	// Virtualization
	SupportsMIG          bool     `json:"supports_mig" db:"supports_mig"`
	SupportsSRIOV        bool     `json:"supports_sriov" db:"supports_sriov"`
	
	// Supported precision types
	PrecisionTypes       []string `json:"precision_types" db:"precision_types"`
	
	// Maximum dimensions
	MaxThreadsPerBlock   int32    `json:"max_threads_per_block" db:"max_threads_per_block"`
	MaxBlocksPerGrid     int32    `json:"max_blocks_per_grid" db:"max_blocks_per_grid"`
}

// GPUDriverInfo contains driver information
type GPUDriverInfo struct {
	Version             string    `json:"version" db:"version"`
	CUDAVersion         string    `json:"cuda_version" db:"cuda_version"`
	ROCmVersion         string    `json:"rocm_version" db:"rocm_version"`
	LevelZeroVersion    string    `json:"level_zero_version" db:"level_zero_version"`
	InstallDate         time.Time `json:"install_date" db:"install_date"`
	IsCompatible        bool      `json:"is_compatible" db:"is_compatible"`
	SupportedAPIs       []string  `json:"supported_apis" db:"supported_apis"`
}

// GPUProcess represents a process using the GPU
type GPUProcess struct {
	ID            uuid.UUID `json:"id" db:"id"`
	GPUID         uuid.UUID `json:"gpu_id" db:"gpu_id"`
	PID           int32     `json:"pid" db:"pid"`
	Name          string    `json:"name" db:"name"`
	MemoryUsageMB int64     `json:"memory_usage_mb" db:"memory_usage_mb"`
	ProcessType   string    `json:"process_type" db:"process_type"` // compute, graphics
	StartTime     time.Time `json:"start_time" db:"start_time"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// GPUBenchmark represents a benchmark result
type GPUBenchmark struct {
	ID                uuid.UUID `json:"id" db:"id"`
	GPUID             uuid.UUID `json:"gpu_id" db:"gpu_id"`
	BenchmarkType     string    `json:"benchmark_type" db:"benchmark_type"`
	TestName          string    `json:"test_name" db:"test_name"`
	Score             float64   `json:"score" db:"score"`
	Unit              string    `json:"unit" db:"unit"`
	DurationSeconds   int32     `json:"duration_seconds" db:"duration_seconds"`
	Metadata          common.JSONData `json:"metadata" db:"metadata"`
	BenchmarkedAt     time.Time `json:"benchmarked_at" db:"benchmarked_at"`
	BenchmarkerVersion string   `json:"benchmarker_version" db:"benchmarker_version"`
}

// GPUAllocation represents current GPU allocation
type GPUAllocation struct {
	ID              uuid.UUID `json:"id" db:"id"`
	GPUID           uuid.UUID `json:"gpu_id" db:"gpu_id"`
	ProviderID      uuid.UUID `json:"provider_id" db:"provider_id"`
	ConsumerID      uuid.UUID `json:"consumer_id" db:"consumer_id"`
	AllocationID    string    `json:"allocation_id" db:"allocation_id"`
	Status          string    `json:"status" db:"status"` // allocated, running, completed, failed
	AllocatedAt     time.Time `json:"allocated_at" db:"allocated_at"`
	StartedAt       *time.Time `json:"started_at" db:"started_at"`
	CompletedAt     *time.Time `json:"completed_at" db:"completed_at"`
	ExpectedEndTime *time.Time `json:"expected_end_time" db:"expected_end_time"`
	ActualEndTime   *time.Time `json:"actual_end_time" db:"actual_end_time"`
	Configuration   common.JSONData `json:"configuration" db:"configuration"`
}

// GPUSearchFilter extends common search filter for GPU-specific searches
type GPUSearchFilter struct {
	common.SearchFilter
	
	// GPU-specific filters
	MinCUDACores       *int32    `json:"min_cuda_cores,omitempty"`
	MaxCUDACores       *int32    `json:"max_cuda_cores,omitempty"`
	MinTensorCores     *int32    `json:"min_tensor_cores,omitempty"`
	GPUVendors         []string  `json:"gpu_vendors,omitempty"`
	Architectures      []string  `json:"architectures,omitempty"`
	ComputeCapabilities []string `json:"compute_capabilities,omitempty"`
	SupportsCUDA       *bool     `json:"supports_cuda,omitempty"`
	SupportsOpenCL     *bool     `json:"supports_opencl,omitempty"`
	SupportsTensorOps  *bool     `json:"supports_tensor_ops,omitempty"`
	SupportsMIG        *bool     `json:"supports_mig,omitempty"`
	ECCEnabled         *bool     `json:"ecc_enabled,omitempty"`
	MinPowerWatts      *int32    `json:"min_power_watts,omitempty"`
	MaxPowerWatts      *int32    `json:"max_power_watts,omitempty"`
	IsAllocated        *bool     `json:"is_allocated,omitempty"`
	VerificationStatus []string  `json:"verification_status,omitempty"`
	MinUptime          *float64  `json:"min_uptime,omitempty"`
	MinAvgUtilization  *float64  `json:"min_avg_utilization,omitempty"`
	MaxAvgUtilization  *float64  `json:"max_avg_utilization,omitempty"`
}

// GPUResourceSummary provides a summary view for listings
type GPUResourceSummary struct {
	ID                 uuid.UUID `json:"id"`
	ProviderID         uuid.UUID `json:"provider_id"`
	Name               string    `json:"name"`
	Vendor             string    `json:"vendor"`
	Status             common.ResourceStatus `json:"status"`
	MemoryTotalMB      int64     `json:"memory_total_mb"`
	CUDACores          int32     `json:"cuda_cores"`
	Architecture       string    `json:"architecture"`
	ComputeCapability  string    `json:"compute_capability"`
	IsAllocated        bool      `json:"is_allocated"`
	GPUUtilization     int32     `json:"gpu_utilization"`
	MemoryUtilization  int32     `json:"memory_utilization"`
	LastHeartbeat      *time.Time `json:"last_heartbeat"`
	Region             string    `json:"region"`
	DataCenter         string    `json:"data_center"`
	VerificationStatus string    `json:"verification_status"`
	UptimePercentage   float64   `json:"uptime_percentage"`
}