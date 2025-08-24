package internal

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	
	"github.com/lcanady/depin/hardware/detection/common"
	"github.com/lcanady/depin/hardware/detection/nvml"
	"github.com/lcanady/depin/hardware/detection/rocm"
	"github.com/lcanady/depin/hardware/detection/intel"
)

// GPUDiscoveryService implements the main GPU discovery service
type GPUDiscoveryService struct {
	mu       sync.RWMutex
	registry common.DetectorRegistry
	config   *ServiceConfig
	logger   *logrus.Logger
	
	// State management
	running        bool
	lastDiscovery  time.Time
	gpuCache       map[string]common.GPUInfo
	changeCallback common.ChangeCallback
	
	// Context for background operations
	ctx    context.Context
	cancel context.CancelFunc
}

// ServiceConfig contains configuration for the GPU discovery service
type ServiceConfig struct {
	// Detector configuration
	DetectorConfig common.Config
	
	// Service-specific settings
	EnableAutoDiscovery     bool
	AutoDiscoveryInterval   time.Duration
	CacheTimeout           time.Duration
	
	// gRPC server settings
	GRPCPort        int
	GRPCMaxMsgSize  int
	
	// Monitoring settings
	MetricsEnabled  bool
	MetricsPort     int
	
	// Logging settings
	LogLevel        string
	LogFormat       string
}

// NewGPUDiscoveryService creates a new GPU discovery service
func NewGPUDiscoveryService(config *ServiceConfig) (*GPUDiscoveryService, error) {
	if config == nil {
		return nil, fmt.Errorf("service configuration is required")
	}

	// Setup logger
	logger := logrus.New()
	level, err := logrus.ParseLevel(config.LogLevel)
	if err != nil {
		logger.SetLevel(logrus.InfoLevel)
	} else {
		logger.SetLevel(level)
	}
	
	if config.LogFormat == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{})
	}

	// Create detector registry
	registry := common.NewRegistry(logger)

	// Create service context
	ctx, cancel := context.WithCancel(context.Background())

	service := &GPUDiscoveryService{
		registry:   registry,
		config:     config,
		logger:     logger,
		gpuCache:   make(map[string]common.GPUInfo),
		ctx:        ctx,
		cancel:     cancel,
	}

	// Initialize the service
	if err := service.initialize(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize service: %v", err)
	}

	return service, nil
}

// initialize sets up the GPU detectors and performs initial discovery
func (s *GPUDiscoveryService) initialize() error {
	s.logger.Info("Initializing GPU Discovery Service")

	// Register GPU detectors based on configuration
	if err := s.registerDetectors(); err != nil {
		return fmt.Errorf("failed to register detectors: %v", err)
	}

	// Initialize all available detectors
	if err := s.registry.InitializeAll(s.ctx); err != nil {
		return fmt.Errorf("failed to initialize detectors: %v", err)
	}

	// Perform initial GPU discovery
	if err := s.performInitialDiscovery(); err != nil {
		s.logger.Warnf("Initial GPU discovery failed: %v", err)
		// Don't fail initialization if no GPUs are found - they might be added later
	}

	// Start background monitoring if enabled
	if s.config.EnableAutoDiscovery {
		go s.runAutoDiscovery()
	}

	s.running = true
	s.logger.Info("GPU Discovery Service initialized successfully")
	return nil
}

// registerDetectors registers all enabled GPU detectors
func (s *GPUDiscoveryService) registerDetectors() error {
	// Register NVIDIA detector if enabled
	if s.config.DetectorConfig.EnableNVIDIA {
		nvmlDetector := nvml.NewNVMLDetector(s.logger, &s.config.DetectorConfig)
		if err := s.registry.RegisterDetector(nvmlDetector); err != nil {
			s.logger.Errorf("Failed to register NVML detector: %v", err)
		} else {
			s.logger.Debug("Registered NVIDIA/NVML detector")
		}
	}

	// Register AMD detector if enabled
	if s.config.DetectorConfig.EnableAMD {
		rocmDetector := rocm.NewROCmDetector(s.logger, &s.config.DetectorConfig)
		if err := s.registry.RegisterDetector(rocmDetector); err != nil {
			s.logger.Errorf("Failed to register ROCm detector: %v", err)
		} else {
			s.logger.Debug("Registered AMD/ROCm detector")
		}
	}

	// Register Intel detector if enabled
	if s.config.DetectorConfig.EnableIntel {
		intelDetector := intel.NewIntelDetector(s.logger, &s.config.DetectorConfig)
		if err := s.registry.RegisterDetector(intelDetector); err != nil {
			s.logger.Errorf("Failed to register Intel detector: %v", err)
		} else {
			s.logger.Debug("Registered Intel detector")
		}
	}

	availableDetectors := s.registry.GetAvailableDetectors()
	if len(availableDetectors) == 0 {
		return fmt.Errorf("no GPU detectors are available on this system")
	}

	s.logger.Infof("Registered %d available GPU detectors", len(availableDetectors))
	return nil
}

// performInitialDiscovery discovers all GPUs and populates the cache
func (s *GPUDiscoveryService) performInitialDiscovery() error {
	s.logger.Info("Performing initial GPU discovery")

	gpus, err := s.registry.DiscoverAllGPUs(s.ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Update cache
	s.gpuCache = make(map[string]common.GPUInfo)
	for _, gpu := range gpus {
		s.gpuCache[gpu.ID] = gpu
	}

	s.lastDiscovery = time.Now()
	s.logger.Infof("Initial discovery completed: found %d GPUs", len(gpus))

	return nil
}

// runAutoDiscovery runs periodic GPU discovery in the background
func (s *GPUDiscoveryService) runAutoDiscovery() {
	ticker := time.NewTicker(s.config.AutoDiscoveryInterval)
	defer ticker.Stop()

	s.logger.Info("Starting auto-discovery background process")

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Info("Stopping auto-discovery due to context cancellation")
			return
		case <-ticker.C:
			s.logger.Debug("Running periodic GPU discovery")
			if err := s.refreshGPUCache(); err != nil {
				s.logger.Errorf("Periodic discovery failed: %v", err)
			}
		}
	}
}

// refreshGPUCache refreshes the GPU cache with latest information
func (s *GPUDiscoveryService) refreshGPUCache() error {
	gpus, err := s.registry.DiscoverAllGPUs(s.ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Compare with existing cache to detect changes
	oldCache := s.gpuCache
	newCache := make(map[string]common.GPUInfo)
	
	for _, gpu := range gpus {
		newCache[gpu.ID] = gpu
		
		// Check for changes
		if oldGPU, exists := oldCache[gpu.ID]; exists {
			if s.hasGPUChanged(oldGPU, gpu) && s.changeCallback != nil {
				go s.changeCallback(common.GPUChange{
					Type:        common.ChangeModified,
					GPU:         gpu,
					Timestamp:   time.Now(),
					Description: "GPU status updated during auto-discovery",
				})
			}
		} else if s.changeCallback != nil {
			// New GPU detected
			go s.changeCallback(common.GPUChange{
				Type:        common.ChangeAdded,
				GPU:         gpu,
				Timestamp:   time.Now(),
				Description: "New GPU detected during auto-discovery",
			})
		}
	}

	// Check for removed GPUs
	if s.changeCallback != nil {
		for id, oldGPU := range oldCache {
			if _, exists := newCache[id]; !exists {
				go s.changeCallback(common.GPUChange{
					Type:        common.ChangeRemoved,
					GPU:         oldGPU,
					Timestamp:   time.Now(),
					Description: "GPU removed or no longer accessible",
				})
			}
		}
	}

	s.gpuCache = newCache
	s.lastDiscovery = time.Now()
	
	s.logger.Debugf("GPU cache refreshed: %d GPUs", len(newCache))
	return nil
}

// Public API methods

// DiscoverGPUs discovers all available GPUs
func (s *GPUDiscoveryService) DiscoverGPUs(forceRefresh bool, vendorFilter []string) ([]common.GPUInfo, error) {
	if !s.running {
		return nil, fmt.Errorf("service is not running")
	}

	// Check if we need to refresh the cache
	s.mu.RLock()
	cacheAge := time.Since(s.lastDiscovery)
	shouldRefresh := forceRefresh || cacheAge > s.config.CacheTimeout
	s.mu.RUnlock()

	if shouldRefresh {
		if err := s.refreshGPUCache(); err != nil {
			s.logger.Errorf("Failed to refresh GPU cache: %v", err)
			// Continue with cached data if available
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Apply vendor filter if specified
	var filteredGPUs []common.GPUInfo
	for _, gpu := range s.gpuCache {
		if len(vendorFilter) == 0 {
			filteredGPUs = append(filteredGPUs, gpu)
		} else {
			for _, vendor := range vendorFilter {
				if gpu.Vendor == vendor {
					filteredGPUs = append(filteredGPUs, gpu)
					break
				}
			}
		}
	}

	return filteredGPUs, nil
}

// GetGPUInfo gets information about a specific GPU
func (s *GPUDiscoveryService) GetGPUInfo(gpuID string, includeBenchmarks bool) (*common.GPUInfo, []common.BenchmarkResult, error) {
	if !s.running {
		return nil, nil, fmt.Errorf("service is not running")
	}

	// Try to get from cache first
	s.mu.RLock()
	cachedGPU, exists := s.gpuCache[gpuID]
	s.mu.RUnlock()

	if !exists {
		// Try to get directly from registry
		gpu, err := s.registry.GetGPUInfo(s.ctx, gpuID)
		if err != nil {
			return nil, nil, fmt.Errorf("GPU not found: %v", err)
		}
		
		// Update cache
		s.mu.Lock()
		s.gpuCache[gpuID] = *gpu
		s.mu.Unlock()
		
		cachedGPU = *gpu
	}

	var benchmarks []common.BenchmarkResult
	if includeBenchmarks {
		// Run basic benchmarks
		benchmarkTypes := []common.BenchmarkType{
			common.BenchmarkCompute,
			common.BenchmarkMemory,
		}

		for _, benchType := range benchmarkTypes {
			result, err := s.registry.RunBenchmark(s.ctx, gpuID, benchType, 10*time.Second)
			if err != nil {
				s.logger.Warnf("Benchmark failed for GPU %s: %v", gpuID, err)
				continue
			}
			benchmarks = append(benchmarks, *result)
		}
	}

	return &cachedGPU, benchmarks, nil
}

// GetSystemSummary returns a summary of all GPUs in the system
func (s *GPUDiscoveryService) GetSystemSummary(includeOffline bool) (*common.SystemInfo, error) {
	if !s.running {
		return nil, fmt.Errorf("service is not running")
	}

	systemInfo, err := s.registry.GetSystemInfo(s.ctx)
	if err != nil {
		return nil, err
	}

	// Apply offline filter if needed
	if !includeOffline {
		s.mu.RLock()
		onlineCount := int32(0)
		for _, gpu := range s.gpuCache {
			if gpu.Status.State != common.StateOffline {
				onlineCount++
			}
		}
		s.mu.RUnlock()
		
		systemInfo.TotalGPUs = onlineCount
	}

	return systemInfo, nil
}

// StartMonitoring starts monitoring GPU changes
func (s *GPUDiscoveryService) StartMonitoring(callback common.ChangeCallback) error {
	if !s.running {
		return fmt.Errorf("service is not running")
	}

	s.mu.Lock()
	s.changeCallback = callback
	s.mu.Unlock()

	// Start monitoring in the background
	go func() {
		if err := s.registry.MonitorAllGPUs(s.ctx, callback); err != nil {
			s.logger.Errorf("GPU monitoring failed: %v", err)
		}
	}()

	s.logger.Info("GPU monitoring started")
	return nil
}

// RunBenchmark runs a benchmark on a specific GPU
func (s *GPUDiscoveryService) RunBenchmark(gpuID string, benchmarkTypes []string, duration time.Duration) ([]common.BenchmarkResult, error) {
	if !s.running {
		return nil, fmt.Errorf("service is not running")
	}

	var results []common.BenchmarkResult

	for _, benchTypeName := range benchmarkTypes {
		var benchType common.BenchmarkType
		switch benchTypeName {
		case "compute":
			benchType = common.BenchmarkCompute
		case "memory":
			benchType = common.BenchmarkMemory
		case "tensor":
			benchType = common.BenchmarkTensor
		default:
			s.logger.Warnf("Unknown benchmark type: %s", benchTypeName)
			continue
		}

		result, err := s.registry.RunBenchmark(s.ctx, gpuID, benchType, duration)
		if err != nil {
			s.logger.Errorf("Benchmark %s failed for GPU %s: %v", benchTypeName, gpuID, err)
			continue
		}

		results = append(results, *result)
	}

	return results, nil
}

// GetServiceStatus returns the current status of the service
func (s *GPUDiscoveryService) GetServiceStatus() *ServiceStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	detectorStatus := s.registry.GetDetectorStatus()

	return &ServiceStatus{
		Running:         s.running,
		LastDiscovery:   s.lastDiscovery,
		CachedGPUCount:  len(s.gpuCache),
		DetectorStatus:  detectorStatus,
		AutoDiscovery:   s.config.EnableAutoDiscovery,
		MonitoringActive: s.changeCallback != nil,
	}
}

// Shutdown gracefully shuts down the service
func (s *GPUDiscoveryService) Shutdown() error {
	s.logger.Info("Shutting down GPU Discovery Service")

	s.mu.Lock()
	s.running = false
	s.mu.Unlock()

	// Cancel context to stop background operations
	s.cancel()

	// Cleanup detectors
	if err := s.registry.CleanupAll(); err != nil {
		s.logger.Errorf("Error during detector cleanup: %v", err)
		return err
	}

	s.logger.Info("GPU Discovery Service shutdown completed")
	return nil
}

// Helper methods

func (s *GPUDiscoveryService) hasGPUChanged(old, new common.GPUInfo) bool {
	// Compare key status fields that indicate meaningful changes
	return old.Status.State != new.Status.State ||
		   old.Status.GPUUtilization != new.Status.GPUUtilization ||
		   old.Status.MemoryUtilization != new.Status.MemoryUtilization ||
		   old.Status.TemperatureGPU != new.Status.TemperatureGPU ||
		   len(old.Status.Processes) != len(new.Status.Processes)
}

// ServiceStatus represents the current status of the service
type ServiceStatus struct {
	Running         bool
	LastDiscovery   time.Time
	CachedGPUCount  int
	DetectorStatus  map[string]common.DetectorStatus
	AutoDiscovery   bool
	MonitoringActive bool
}

// DefaultServiceConfig returns a default service configuration
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		DetectorConfig: common.Config{
			MonitoringIntervalSeconds: 30,
			BenchmarkTimeoutSeconds:   60,
			DiscoveryTimeoutSeconds:   30,
			EnableNVIDIA:             true,
			EnableAMD:                true,
			EnableIntel:              true,
			EnableBenchmarks:         true,
			LogLevel:                 "info",
		},
		EnableAutoDiscovery:   true,
		AutoDiscoveryInterval: 60 * time.Second,
		CacheTimeout:         5 * time.Minute,
		GRPCPort:             50051,
		GRPCMaxMsgSize:       4 * 1024 * 1024, // 4MB
		MetricsEnabled:       true,
		MetricsPort:          9090,
		LogLevel:             "info",
		LogFormat:            "text",
	}
}