package common

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Registry implements DetectorRegistry interface
type Registry struct {
	mu        sync.RWMutex
	detectors map[string]GPUDetector
	logger    Logger
}

// NewRegistry creates a new detector registry
func NewRegistry(logger Logger) *Registry {
	return &Registry{
		detectors: make(map[string]GPUDetector),
		logger:    logger,
	}
}

// RegisterDetector registers a new GPU detector
func (r *Registry) RegisterDetector(detector GPUDetector) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	vendor := detector.GetVendorName()
	if _, exists := r.detectors[vendor]; exists {
		return fmt.Errorf("detector for vendor %s already registered", vendor)
	}

	r.detectors[vendor] = detector
	r.logger.Infof("Registered GPU detector for vendor: %s", vendor)
	return nil
}

// GetAvailableDetectors returns all registered detectors that are available on the system
func (r *Registry) GetAvailableDetectors() []GPUDetector {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var available []GPUDetector
	for vendor, detector := range r.detectors {
		if detector.IsAvailable() {
			available = append(available, detector)
			r.logger.Debugf("Detector for vendor %s is available", vendor)
		} else {
			r.logger.Debugf("Detector for vendor %s is not available", vendor)
		}
	}

	return available
}

// GetDetectorByVendor returns the detector for a specific vendor
func (r *Registry) GetDetectorByVendor(vendor string) GPUDetector {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.detectors[vendor]
}

// InitializeAll initializes all available detectors
func (r *Registry) InitializeAll(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var initErrors []error
	for vendor, detector := range r.detectors {
		if !detector.IsAvailable() {
			r.logger.Debugf("Skipping initialization of %s detector - not available", vendor)
			continue
		}

		r.logger.Infof("Initializing %s detector", vendor)
		if err := detector.Initialize(ctx); err != nil {
			r.logger.Errorf("Failed to initialize %s detector: %v", vendor, err)
			initErrors = append(initErrors, fmt.Errorf("%s: %v", vendor, err))
		} else {
			r.logger.Infof("Successfully initialized %s detector", vendor)
		}
	}

	if len(initErrors) == len(r.detectors) {
		return fmt.Errorf("failed to initialize any detectors: %v", initErrors)
	}

	if len(initErrors) > 0 {
		r.logger.Warnf("Some detectors failed to initialize: %v", initErrors)
	}

	return nil
}

// CleanupAll cleans up all initialized detectors
func (r *Registry) CleanupAll() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var cleanupErrors []error
	for vendor, detector := range r.detectors {
		r.logger.Debugf("Cleaning up %s detector", vendor)
		if err := detector.Cleanup(); err != nil {
			r.logger.Errorf("Failed to cleanup %s detector: %v", vendor, err)
			cleanupErrors = append(cleanupErrors, fmt.Errorf("%s: %v", vendor, err))
		}
	}

	if len(cleanupErrors) > 0 {
		return fmt.Errorf("cleanup errors: %v", cleanupErrors)
	}

	return nil
}

// GetSystemInfo aggregates system information from all available detectors
func (r *Registry) GetSystemInfo(ctx context.Context) (*SystemInfo, error) {
	detectors := r.GetAvailableDetectors()
	
	systemInfo := &SystemInfo{
		SupportedVendors:  make([]string, 0),
		GPUCountByVendor:  make(map[string]int32),
	}

	for _, detector := range detectors {
		vendor := detector.GetVendorName()
		systemInfo.SupportedVendors = append(systemInfo.SupportedVendors, vendor)

		gpus, err := detector.DiscoverGPUs(ctx)
		if err != nil {
			r.logger.Warnf("Failed to discover GPUs for vendor %s: %v", vendor, err)
			continue
		}

		gpuCount := int32(len(gpus))
		systemInfo.GPUCountByVendor[vendor] = gpuCount
		systemInfo.TotalGPUs += gpuCount

		// Count available (idle) GPUs and aggregate memory
		for _, gpu := range gpus {
			if gpu.Status.State == StateIdle {
				systemInfo.AvailableGPUs++
			}
			systemInfo.TotalMemoryMB += gpu.Specs.MemoryTotalMB
			systemInfo.AvailableMemoryMB += gpu.Status.MemoryFreeMB
		}
	}

	return systemInfo, nil
}

// DiscoverAllGPUs discovers GPUs from all available detectors
func (r *Registry) DiscoverAllGPUs(ctx context.Context) ([]GPUInfo, error) {
	detectors := r.GetAvailableDetectors()
	
	var allGPUs []GPUInfo
	var discoveryErrors []error

	for _, detector := range detectors {
		vendor := detector.GetVendorName()
		r.logger.Debugf("Discovering GPUs for vendor: %s", vendor)

		gpus, err := detector.DiscoverGPUs(ctx)
		if err != nil {
			r.logger.Errorf("Failed to discover GPUs for vendor %s: %v", vendor, err)
			discoveryErrors = append(discoveryErrors, fmt.Errorf("%s: %v", vendor, err))
			continue
		}

		r.logger.Infof("Discovered %d GPUs for vendor %s", len(gpus), vendor)
		allGPUs = append(allGPUs, gpus...)
	}

	if len(allGPUs) == 0 && len(discoveryErrors) > 0 {
		return nil, fmt.Errorf("failed to discover any GPUs: %v", discoveryErrors)
	}

	return allGPUs, nil
}

// GetGPUInfo gets information about a specific GPU by searching all detectors
func (r *Registry) GetGPUInfo(ctx context.Context, gpuID string) (*GPUInfo, error) {
	detectors := r.GetAvailableDetectors()

	for _, detector := range detectors {
		if info, err := detector.GetGPUInfo(ctx, gpuID); err == nil {
			return info, nil
		}
	}

	return nil, fmt.Errorf("GPU with ID %s not found in any detector", gpuID)
}

// RunBenchmark runs a benchmark on a specific GPU
func (r *Registry) RunBenchmark(ctx context.Context, gpuID string, benchmarkType BenchmarkType, duration ...interface{}) (*BenchmarkResult, error) {
	detectors := r.GetAvailableDetectors()
	
	// Determine duration
	benchDuration := 30 * time.Second // default
	if len(duration) > 0 {
		if d, ok := duration[0].(time.Duration); ok {
			benchDuration = d
		}
	}

	for _, detector := range detectors {
		if result, err := detector.RunBenchmark(ctx, gpuID, benchmarkType, benchDuration); err == nil {
			return result, nil
		}
	}

	return nil, fmt.Errorf("GPU with ID %s not found for benchmarking", gpuID)
}

// MonitorAllGPUs monitors changes across all detectors
func (r *Registry) MonitorAllGPUs(ctx context.Context, callback ChangeCallback) error {
	detectors := r.GetAvailableDetectors()
	
	if len(detectors) == 0 {
		return fmt.Errorf("no available detectors for monitoring")
	}

	// Start monitoring for each detector in separate goroutines
	errorChan := make(chan error, len(detectors))
	
	for _, detector := range detectors {
		go func(d GPUDetector) {
			vendor := d.GetVendorName()
			r.logger.Infof("Starting monitoring for vendor: %s", vendor)
			
			// Wrap callback to add vendor info
			wrappedCallback := func(change GPUChange) {
				r.logger.Debugf("GPU change detected for vendor %s: %s", vendor, change.Description)
				callback(change)
			}
			
			if err := d.MonitorChanges(ctx, wrappedCallback); err != nil {
				r.logger.Errorf("Monitoring failed for vendor %s: %v", vendor, err)
				errorChan <- fmt.Errorf("%s: %v", vendor, err)
			}
		}(detector)
	}

	// Wait for context cancellation or first error
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errorChan:
		return err
	}
}

// GetRegisteredVendors returns all registered vendor names
func (r *Registry) GetRegisteredVendors() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	vendors := make([]string, 0, len(r.detectors))
	for vendor := range r.detectors {
		vendors = append(vendors, vendor)
	}

	return vendors
}

// GetDetectorStatus returns status information for all detectors
func (r *Registry) GetDetectorStatus() map[string]DetectorStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	status := make(map[string]DetectorStatus)
	for vendor, detector := range r.detectors {
		status[vendor] = DetectorStatus{
			Vendor:      vendor,
			Available:   detector.IsAvailable(),
			Initialized: true, // We don't track this in the interface currently
		}
	}

	return status
}

// DetectorStatus represents the status of a detector
type DetectorStatus struct {
	Vendor      string
	Available   bool
	Initialized bool
}

// ValidateConfiguration validates the detector configuration
func (r *Registry) ValidateConfiguration(config *Config) error {
	if config == nil {
		return fmt.Errorf("configuration cannot be nil")
	}

	if config.MonitoringIntervalSeconds <= 0 {
		return fmt.Errorf("monitoring interval must be positive")
	}

	if config.BenchmarkTimeoutSeconds <= 0 {
		return fmt.Errorf("benchmark timeout must be positive")
	}

	if config.DiscoveryTimeoutSeconds <= 0 {
		return fmt.Errorf("discovery timeout must be positive")
	}

	// Validate that at least one vendor is enabled
	if !config.EnableNVIDIA && !config.EnableAMD && !config.EnableIntel {
		return fmt.Errorf("at least one GPU vendor must be enabled")
	}

	return nil
}

// Close gracefully shuts down the registry
func (r *Registry) Close() error {
	r.logger.Info("Shutting down GPU detector registry")
	return r.CleanupAll()
}