package intel

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lcanady/depin/hardware/detection/common"
)

// IntelDetector implements GPU detection for Intel GPUs using Level Zero and system tools
type IntelDetector struct {
	mu              sync.RWMutex
	initialized     bool
	logger          common.Logger
	config          *common.Config
	deviceInfoCache map[string]*common.GPUInfo
	lastDiscovery   time.Time
}

// NewIntelDetector creates a new Intel GPU detector
func NewIntelDetector(logger common.Logger, config *common.Config) *IntelDetector {
	return &IntelDetector{
		logger:          logger,
		config:          config,
		deviceInfoCache: make(map[string]*common.GPUInfo),
	}
}

// Initialize initializes the Intel GPU detector
func (i *IntelDetector) Initialize(ctx context.Context) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.initialized {
		return nil
	}

	i.logger.Info("Initializing Intel GPU detector")
	
	// Check if intel-gpu-tools are available
	if !i.isIntelGPUTopAvailable() {
		return fmt.Errorf("intel_gpu_top tool not found - ensure Intel GPU tools are installed")
	}

	// Test basic functionality
	if err := i.testIntelGPUTools(); err != nil {
		return fmt.Errorf("Intel GPU tools test failed: %v", err)
	}

	i.logger.Info("Intel GPU detector initialized successfully")
	i.initialized = true
	return nil
}

// Cleanup performs cleanup operations
func (i *IntelDetector) Cleanup() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.logger.Info("Cleaning up Intel GPU detector")
	i.initialized = false
	return nil
}

// IsAvailable checks if Intel GPUs are available
func (i *IntelDetector) IsAvailable() bool {
	if !i.isIntelGPUTopAvailable() {
		return false
	}

	// Check for Intel GPU devices in /dev/dri
	cmd := exec.Command("ls", "/dev/dri/")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Look for render nodes (renderD128, etc.)
	return strings.Contains(string(output), "renderD")
}

// GetVendorName returns the vendor name
func (i *IntelDetector) GetVendorName() string {
	return "intel"
}

// DiscoverGPUs discovers all Intel GPUs
func (i *IntelDetector) DiscoverGPUs(ctx context.Context) ([]common.GPUInfo, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if !i.initialized {
		return nil, fmt.Errorf("Intel detector not initialized")
	}

	// Get device list using lspci and /dev/dri
	devices, err := i.getDeviceList()
	if err != nil {
		return nil, fmt.Errorf("failed to get device list: %v", err)
	}

	var gpus []common.GPUInfo
	for idx, deviceID := range devices {
		gpuInfo, err := i.getDeviceInfo(deviceID, int32(idx))
		if err != nil {
			i.logger.Warnf("Failed to get info for device %s: %v", deviceID, err)
			continue
		}

		gpus = append(gpus, *gpuInfo)
		i.deviceInfoCache[gpuInfo.ID] = gpuInfo
	}

	i.lastDiscovery = time.Now()
	i.logger.Infof("Discovered %d Intel GPUs", len(gpus))
	return gpus, nil
}

// GetGPUInfo gets detailed information about a specific GPU
func (i *IntelDetector) GetGPUInfo(ctx context.Context, gpuID string) (*common.GPUInfo, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if cached, exists := i.deviceInfoCache[gpuID]; exists {
		// Refresh the status information
		deviceID, err := i.getDeviceIDFromGPUID(gpuID)
		if err != nil {
			return nil, err
		}

		status, err := i.getDeviceStatus(deviceID)
		if err == nil {
			cached.Status = *status
			cached.LastSeen = time.Now()
		}
		return cached, nil
	}

	return nil, fmt.Errorf("GPU with ID %s not found", gpuID)
}

// MonitorChanges monitors GPU changes
func (i *IntelDetector) MonitorChanges(ctx context.Context, callback common.ChangeCallback) error {
	ticker := time.NewTicker(time.Duration(i.config.MonitoringIntervalSeconds) * time.Second)
	defer ticker.Stop()

	previousGPUs := make(map[string]common.GPUInfo)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			currentGPUs, err := i.DiscoverGPUs(ctx)
			if err != nil {
				i.logger.Errorf("Error during GPU discovery: %v", err)
				continue
			}

			currentMap := make(map[string]common.GPUInfo)
			for _, gpu := range currentGPUs {
				currentMap[gpu.ID] = gpu
			}

			// Check for changes
			for id, current := range currentMap {
				if previous, exists := previousGPUs[id]; exists {
					if i.hasGPUChanged(previous, current) {
						callback(common.GPUChange{
							Type:        common.ChangeModified,
							GPU:         current,
							Timestamp:   time.Now(),
							Description: "GPU status or configuration changed",
						})
					}
				} else {
					callback(common.GPUChange{
						Type:        common.ChangeAdded,
						GPU:         current,
						Timestamp:   time.Now(),
						Description: "New GPU detected",
					})
				}
			}

			for id, previous := range previousGPUs {
				if _, exists := currentMap[id]; !exists {
					callback(common.GPUChange{
						Type:        common.ChangeRemoved,
						GPU:         previous,
						Timestamp:   time.Now(),
						Description: "GPU removed or no longer accessible",
					})
				}
			}

			previousGPUs = currentMap
		}
	}
}

// RunBenchmark runs a benchmark on the specified GPU
func (i *IntelDetector) RunBenchmark(ctx context.Context, gpuID string, benchmarkType common.BenchmarkType, duration time.Duration) (*common.BenchmarkResult, error) {
	// Placeholder implementation for Intel GPU benchmarks
	// In practice, this would use Level Zero, OpenCL, or Intel's compute benchmarks
	
	result := &common.BenchmarkResult{
		BenchmarkType:   i.getBenchmarkTypeName(benchmarkType),
		TestName:        "Intel GPU Basic Test",
		Score:           0.0,
		Unit:            "GFLOPS",
		DurationSeconds: int32(duration.Seconds()),
		Metadata:        make(map[string]string),
		Timestamp:       time.Now(),
	}

	gpuInfo, err := i.GetGPUInfo(ctx, gpuID)
	if err != nil {
		return nil, err
	}

	result.Metadata["gpu_name"] = gpuInfo.Name
	result.Metadata["gpu_uuid"] = gpuInfo.UUID
	result.Metadata["execution_units"] = strconv.Itoa(int(gpuInfo.Specs.ExecutionUnits))
	result.Metadata["memory_mb"] = strconv.FormatInt(gpuInfo.Specs.MemoryTotalMB, 10)

	// Placeholder benchmark score
	result.Score = float64(gpuInfo.Specs.ExecutionUnits) * 8.0 // Rough estimate

	return result, nil
}

// Helper methods

func (i *IntelDetector) isIntelGPUTopAvailable() bool {
	_, err := exec.LookPath("intel_gpu_top")
	return err == nil
}

func (i *IntelDetector) testIntelGPUTools() error {
	// Try to run intel_gpu_top for a very short time to test availability
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, "intel_gpu_top", "-s", "100") // 100ms sampling
	err := cmd.Start()
	if err != nil {
		return err
	}
	
	// Let it run briefly then kill it
	go func() {
		time.Sleep(200 * time.Millisecond)
		cmd.Process.Kill()
	}()
	
	cmd.Wait() // Wait for it to finish (either naturally or killed)
	return nil
}

func (i *IntelDetector) getDeviceList() ([]string, error) {
	// Use lspci to find Intel GPU devices
	cmd := exec.Command("lspci", "-d", "8086:", "-nn")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var devices []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(strings.ToLower(line), "vga") || 
		   strings.Contains(strings.ToLower(line), "display") ||
		   strings.Contains(strings.ToLower(line), "graphics") {
			// Extract PCI ID from format "00:02.0 VGA..."
			parts := strings.Fields(line)
			if len(parts) > 0 {
				devices = append(devices, parts[0])
			}
		}
	}

	return devices, nil
}

func (i *IntelDetector) getDeviceInfo(deviceID string, index int32) (*common.GPUInfo, error) {
	// Get basic device information
	name, err := i.getDeviceName(deviceID)
	if err != nil {
		name = "Intel GPU" // Fallback
	}

	uuid := i.generateUUID(deviceID, index)

	specs, err := i.getDeviceSpecs(deviceID)
	if err != nil {
		return nil, err
	}

	status, err := i.getDeviceStatus(deviceID)
	if err != nil {
		return nil, err
	}

	capabilities := i.getDeviceCapabilities(name)
	driverInfo := i.getDriverInfo()

	gpuInfo := &common.GPUInfo{
		ID:              fmt.Sprintf("intel-%d", index),
		Name:            name,
		Vendor:          "intel",
		UUID:            uuid,
		Index:           index,
		Specs:           *specs,
		Status:          *status,
		Capabilities:    *capabilities,
		Driver:          *driverInfo,
		LastSeen:        time.Now(),
		DiscoverySource: "intel-tools",
	}

	return gpuInfo, nil
}

func (i *IntelDetector) getDeviceName(deviceID string) (string, error) {
	cmd := exec.Command("lspci", "-s", deviceID)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	line := strings.TrimSpace(string(output))
	// Extract name after the device class
	parts := strings.SplitN(line, ":", 3)
	if len(parts) >= 3 {
		return strings.TrimSpace(parts[2]), nil
	}

	return "Intel GPU", nil
}

func (i *IntelDetector) generateUUID(deviceID string, index int32) string {
	// Generate a consistent UUID for Intel GPUs
	return fmt.Sprintf("intel-gpu-%s-%d", strings.Replace(deviceID, ":", "-", -1), index)
}

func (i *IntelDetector) getDeviceSpecs(deviceID string) (*common.GPUSpecs, error) {
	// Intel GPU specs are harder to detect programmatically
	// This would typically require a database of known Intel GPU specifications
	
	specs := &common.GPUSpecs{
		MemoryTotalMB:        i.estimateMemory(deviceID),
		MemoryBandwidthGBPS:  i.estimateMemoryBandwidth(deviceID),
		CUDACores:           0,    // N/A for Intel
		StreamProcessors:    0,    // N/A for Intel
		ExecutionUnits:      i.estimateExecutionUnits(deviceID),
		TensorCores:         0,    // Intel has XMX units, not tensor cores
		BaseClockMHz:        0,    // Not easily detectable
		BoostClockMHz:       0,    // Not easily detectable
		MemoryClockMHz:      0,    // Not easily detectable
		Architecture:        i.estimateArchitecture(deviceID),
		ComputeCapability:   "",   // Intel doesn't use CUDA compute capability
		SMCount:             0,    // Intel uses EUs, not SMs
		PowerLimitWatts:     i.estimatePowerLimit(deviceID),
		DefaultPowerLimitWatts: i.estimatePowerLimit(deviceID),
		BusType:            "PCIe",
		BusWidth:           "x16",
		PCIeGeneration:     4,     // Default assumption
	}

	return specs, nil
}

func (i *IntelDetector) getDeviceStatus(deviceID string) (*common.GPUStatus, error) {
	// Try to get status using intel_gpu_top
	utilization, err := i.getUtilization(deviceID)
	if err != nil {
		i.logger.Warnf("Failed to get utilization: %v", err)
		utilization = 0
	}

	// Determine state
	state := common.StateIdle
	if utilization > 5 {
		state = common.StateBusy
	}

	status := &common.GPUStatus{
		State:                     state,
		GPUUtilization:            int32(utilization),
		MemoryUtilization:         0, // Not easily available
		MemoryUsedMB:              0, // Not easily available
		MemoryFreeMB:              0, // Not easily available
		TemperatureGPU:            0, // Not easily available via standard tools
		TemperatureMemory:         0, // Not available
		PowerDrawWatts:            0, // Not easily available
		CurrentGPUClockMHz:        0, // Not easily available
		CurrentMemoryClockMHz:     0, // Not easily available
		Processes:                 []common.ProcessInfo{}, // Not easily available
	}

	return status, nil
}

func (i *IntelDetector) getUtilization(deviceID string) (int, error) {
	// This would require running intel_gpu_top and parsing output
	// For now, return 0 as placeholder
	return 0, nil
}

func (i *IntelDetector) getDeviceCapabilities(name string) *common.GPUCapabilities {
	// Estimate capabilities based on GPU name/generation
	capabilities := &common.GPUCapabilities{
		SupportsCUDA:           false, // Intel doesn't support CUDA
		SupportsOpenCL:         true,
		SupportsVulkan:         true,
		SupportsDirectX:        false, // Server context
		SupportsTensorOps:      i.supportsTensorOps(name),
		SupportsMixedPrecision: true,
		SupportsRayTracing:     i.supportsRayTracing(name),
		SupportsECC:            false, // Consumer Intel GPUs don't have ECC
		ECCEnabled:             false,
		SupportsUnifiedMemory:  true,  // Intel GPUs use system memory
		SupportsMIG:            false, // Intel doesn't have MIG equivalent
		SupportsSRIOV:          i.supportsSRIOV(name),
		PrecisionTypes:         []string{"fp32", "fp16", "int8"},
		MaxThreadsPerBlock:     1024, // Typical for Intel
		MaxBlocksPerGrid:       65535,
	}

	return capabilities
}

func (i *IntelDetector) getDriverInfo() *common.DriverInfo {
	version := i.getDriverVersion()
	levelZeroVersion := i.getLevelZeroVersion()
	
	return &common.DriverInfo{
		Version:           version,
		CUDAVersion:       "",              // N/A for Intel
		ROCmVersion:       "",              // N/A for Intel
		LevelZeroVersion:  levelZeroVersion,
		InstallDate:       time.Time{},
		IsCompatible:      true,
		SupportedAPIs:     []string{"OpenCL", "Level Zero", "SYCL"},
	}
}

func (i *IntelDetector) getDriverVersion() string {
	// Try to get Intel graphics driver version
	cmd := exec.Command("modinfo", "i915")
	output, err := cmd.Output()
	if err != nil {
		return "Unknown"
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "version:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1])
			}
		}
	}

	return "Unknown"
}

func (i *IntelDetector) getLevelZeroVersion() string {
	// Check for Level Zero installation
	cmd := exec.Command("find", "/usr", "-name", "*level-zero*", "-type", "f")
	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		return ""
	}
	
	return "Available" // Placeholder - would need more sophisticated detection
}

// Estimation methods (would need actual database in production)

func (i *IntelDetector) estimateMemory(deviceID string) int64 {
	// Intel integrated GPUs typically share system memory
	// Discrete GPUs would have dedicated memory
	// This is a placeholder - would need device-specific detection
	return 4096 // 4GB default estimate
}

func (i *IntelDetector) estimateMemoryBandwidth(deviceID string) int64 {
	// Placeholder estimate
	return 100 // GB/s
}

func (i *IntelDetector) estimateExecutionUnits(deviceID string) int32 {
	// Would need device-specific lookup table
	return 96 // Common for many Intel GPUs
}

func (i *IntelDetector) estimateArchitecture(deviceID string) string {
	// Would need device ID to architecture mapping
	return "Xe" // Default to latest architecture
}

func (i *IntelDetector) estimatePowerLimit(deviceID string) int32 {
	// Integrated GPUs typically have lower power limits
	return 100 // Watts
}

func (i *IntelDetector) supportsTensorOps(name string) bool {
	// Xe-based GPUs support XMX (tensor-like operations)
	name = strings.ToLower(name)
	return strings.Contains(name, "xe") || strings.Contains(name, "arc")
}

func (i *IntelDetector) supportsRayTracing(name string) bool {
	// Arc GPUs support ray tracing
	name = strings.ToLower(name)
	return strings.Contains(name, "arc")
}

func (i *IntelDetector) supportsSRIOV(name string) bool {
	// Some Intel server GPUs support SR-IOV
	name = strings.ToLower(name)
	return strings.Contains(name, "server") || strings.Contains(name, "data center")
}

func (i *IntelDetector) getBenchmarkTypeName(bt common.BenchmarkType) string {
	switch bt {
	case common.BenchmarkCompute:
		return "compute"
	case common.BenchmarkMemory:
		return "memory"
	case common.BenchmarkTensor:
		return "tensor"
	default:
		return "unknown"
	}
}

func (i *IntelDetector) hasGPUChanged(previous, current common.GPUInfo) bool {
	return previous.Status.GPUUtilization != current.Status.GPUUtilization ||
		previous.Status.State != current.Status.State
}

func (i *IntelDetector) getDeviceIDFromGPUID(gpuID string) (string, error) {
	if !strings.HasPrefix(gpuID, "intel-") {
		return "", fmt.Errorf("invalid Intel GPU ID format: %s", gpuID)
	}
	
	// Extract index and map back to device ID
	// This is simplified - in practice you'd maintain a mapping
	return "00:02.0", nil // Placeholder
}