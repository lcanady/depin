package common

import (
	"context"
	"time"
)

// GPUDetector defines the interface for GPU detection implementations
type GPUDetector interface {
	// Initialize the detector
	Initialize(ctx context.Context) error
	
	// Cleanup resources
	Cleanup() error
	
	// Check if the detector is available on this system
	IsAvailable() bool
	
	// Get the vendor name this detector handles
	GetVendorName() string
	
	// Discover all GPUs managed by this detector
	DiscoverGPUs(ctx context.Context) ([]GPUInfo, error)
	
	// Get detailed information about a specific GPU
	GetGPUInfo(ctx context.Context, gpuID string) (*GPUInfo, error)
	
	// Monitor GPU changes
	MonitorChanges(ctx context.Context, callback ChangeCallback) error
	
	// Run benchmarks on a GPU
	RunBenchmark(ctx context.Context, gpuID string, benchmarkType BenchmarkType, duration time.Duration) (*BenchmarkResult, error)
}

// ChangeCallback is called when GPU changes are detected
type ChangeCallback func(change GPUChange)

// GPUChange represents a change in GPU state
type GPUChange struct {
	Type        ChangeType
	GPU         GPUInfo
	Timestamp   time.Time
	Description string
}

// ChangeType represents the type of GPU change
type ChangeType int

const (
	ChangeUnknown ChangeType = iota
	ChangeAdded
	ChangeRemoved
	ChangeModified
	ChangePerformanceUpdate
)

// BenchmarkType represents different types of benchmarks
type BenchmarkType int

const (
	BenchmarkCompute BenchmarkType = iota
	BenchmarkMemory
	BenchmarkTensor
)

// GPUInfo contains comprehensive information about a GPU
type GPUInfo struct {
	// Identification
	ID             string
	Name           string
	Vendor         string
	UUID           string
	Index          int32
	
	// Hardware Specifications
	Specs          GPUSpecs
	
	// Current Status
	Status         GPUStatus
	
	// Capabilities
	Capabilities   GPUCapabilities
	
	// Driver Information
	Driver         DriverInfo
	
	// Discovery Metadata
	LastSeen       time.Time
	DiscoverySource string
}

// GPUSpecs contains hardware specifications
type GPUSpecs struct {
	// Memory
	MemoryTotalMB     int64
	MemoryBandwidthGBPS int64
	
	// Compute Units (vendor-specific naming)
	CUDACores         int32  // NVIDIA
	StreamProcessors  int32  // AMD
	ExecutionUnits    int32  // Intel
	TensorCores       int32  // If available
	
	// Clock Speeds (MHz)
	BaseClockMHz      int32
	BoostClockMHz     int32
	MemoryClockMHz    int32
	
	// Architecture
	Architecture      string
	ComputeCapability string
	SMCount           int32  // Streaming Multiprocessors
	
	// Power
	PowerLimitWatts        int32
	DefaultPowerLimitWatts int32
	
	// Connectivity
	BusType           string
	BusWidth          string
	PCIeGeneration    int32
}

// GPUStatus contains current status information
type GPUStatus struct {
	State             GPUState
	
	// Utilization (0-100%)
	GPUUtilization    int32
	MemoryUtilization int32
	
	// Memory Usage
	MemoryUsedMB      int64
	MemoryFreeMB      int64
	
	// Temperature (Celsius)
	TemperatureGPU    int32
	TemperatureMemory int32
	
	// Power Usage
	PowerDrawWatts    int32
	
	// Current Clock Speeds
	CurrentGPUClockMHz    int32
	CurrentMemoryClockMHz int32
	
	// Active Processes
	Processes         []ProcessInfo
}

// GPUState represents the current state of a GPU
type GPUState int

const (
	StateUnknown GPUState = iota
	StateIdle
	StateBusy
	StateOffline
	StateError
)

// String returns the string representation of GPUState
func (s GPUState) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateBusy:
		return "busy"
	case StateOffline:
		return "offline"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// ProcessInfo contains information about processes using the GPU
type ProcessInfo struct {
	PID           int32
	Name          string
	MemoryUsageMB int64
	Type          string // compute, graphics
}

// GPUCapabilities describes what the GPU can do
type GPUCapabilities struct {
	// Compute APIs
	SupportsCUDA     bool
	SupportsOpenCL   bool
	SupportsVulkan   bool
	SupportsDirectX  bool
	
	// AI/ML Features
	SupportsTensorOps    bool
	SupportsMixedPrecision bool
	SupportsRayTracing   bool
	
	// Memory Features
	SupportsECC       bool
	ECCEnabled        bool
	SupportsUnifiedMemory bool
	
	// Virtualization
	SupportsMIG       bool // Multi-Instance GPU
	SupportsSRIOV     bool // SR-IOV
	
	// Supported precision types
	PrecisionTypes    []string // fp32, fp16, int8, etc.
	
	// Maximum dimensions
	MaxThreadsPerBlock int32
	MaxBlocksPerGrid   int32
}

// DriverInfo contains driver information
type DriverInfo struct {
	Version           string
	CUDAVersion       string    // NVIDIA
	ROCmVersion       string    // AMD
	LevelZeroVersion  string    // Intel
	InstallDate       time.Time
	IsCompatible      bool
	SupportedAPIs     []string
}

// BenchmarkResult contains benchmark results
type BenchmarkResult struct {
	BenchmarkType  string
	TestName       string
	Score          float64
	Unit           string
	DurationSeconds int32
	Metadata       map[string]string
	Timestamp      time.Time
}

// SystemInfo provides system-wide GPU information
type SystemInfo struct {
	TotalGPUs           int32
	AvailableGPUs       int32
	TotalMemoryMB       int64
	AvailableMemoryMB   int64
	SupportedVendors    []string
	GPUCountByVendor    map[string]int32
}

// DetectorRegistry manages multiple GPU detectors
type DetectorRegistry interface {
	// Register a detector
	RegisterDetector(detector GPUDetector) error
	
	// Get all available detectors
	GetAvailableDetectors() []GPUDetector
	
	// Get detector by vendor
	GetDetectorByVendor(vendor string) GPUDetector
	
	// Initialize all detectors
	InitializeAll(ctx context.Context) error
	
	// Cleanup all detectors
	CleanupAll() error
}

// Logger interface for detection operations
type Logger interface {
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// Configuration for GPU detection
type Config struct {
	// Polling intervals
	MonitoringIntervalSeconds int
	BenchmarkTimeoutSeconds   int
	DiscoveryTimeoutSeconds   int
	
	// Feature flags
	EnableNVIDIA    bool
	EnableAMD       bool
	EnableIntel     bool
	EnableBenchmarks bool
	
	// Logging
	LogLevel        string
	
	// Vendor-specific configs
	NVMLConfig      map[string]interface{}
	ROCmConfig      map[string]interface{}
	IntelConfig     map[string]interface{}
}