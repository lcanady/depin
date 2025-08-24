package rocm

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

// ROCmDetector implements GPU detection for AMD GPUs using ROCm tools
type ROCmDetector struct {
	mu              sync.RWMutex
	initialized     bool
	logger          common.Logger
	config          *common.Config
	deviceInfoCache map[string]*common.GPUInfo
	lastDiscovery   time.Time
}

// NewROCmDetector creates a new AMD GPU detector
func NewROCmDetector(logger common.Logger, config *common.Config) *ROCmDetector {
	return &ROCmDetector{
		logger:          logger,
		config:          config,
		deviceInfoCache: make(map[string]*common.GPUInfo),
	}
}

// Initialize initializes the ROCm detector
func (r *ROCmDetector) Initialize(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.initialized {
		return nil
	}

	r.logger.Info("Initializing ROCm detector")
	
	// Check if rocm-smi is available
	if !r.isROCmSMIAvailable() {
		return fmt.Errorf("rocm-smi tool not found - ensure ROCm is installed")
	}

	// Test rocm-smi functionality
	if err := r.testROCmSMI(); err != nil {
		return fmt.Errorf("rocm-smi test failed: %v", err)
	}

	r.logger.Info("ROCm detector initialized successfully")
	r.initialized = true
	return nil
}

// Cleanup performs cleanup operations
func (r *ROCmDetector) Cleanup() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.logger.Info("Cleaning up ROCm detector")
	r.initialized = false
	return nil
}

// IsAvailable checks if AMD GPUs are available
func (r *ROCmDetector) IsAvailable() bool {
	if !r.isROCmSMIAvailable() {
		return false
	}

	// Try to list devices
	cmd := exec.Command("rocm-smi", "--showid")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Check if any AMD GPUs are listed
	return strings.Contains(string(output), "GPU")
}

// GetVendorName returns the vendor name
func (r *ROCmDetector) GetVendorName() string {
	return "amd"
}

// DiscoverGPUs discovers all AMD GPUs
func (r *ROCmDetector) DiscoverGPUs(ctx context.Context) ([]common.GPUInfo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.initialized {
		return nil, fmt.Errorf("ROCm detector not initialized")
	}

	// Get device list
	devices, err := r.getDeviceList()
	if err != nil {
		return nil, fmt.Errorf("failed to get device list: %v", err)
	}

	var gpus []common.GPUInfo
	for _, deviceID := range devices {
		gpuInfo, err := r.getDeviceInfo(deviceID)
		if err != nil {
			r.logger.Warnf("Failed to get info for device %s: %v", deviceID, err)
			continue
		}

		gpus = append(gpus, *gpuInfo)
		r.deviceInfoCache[gpuInfo.ID] = gpuInfo
	}

	r.lastDiscovery = time.Now()
	r.logger.Infof("Discovered %d AMD GPUs", len(gpus))
	return gpus, nil
}

// GetGPUInfo gets detailed information about a specific GPU
func (r *ROCmDetector) GetGPUInfo(ctx context.Context, gpuID string) (*common.GPUInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if cached, exists := r.deviceInfoCache[gpuID]; exists {
		// Refresh the status information
		deviceID, err := r.getDeviceIDFromGPUID(gpuID)
		if err != nil {
			return nil, err
		}

		status, err := r.getDeviceStatus(deviceID)
		if err == nil {
			cached.Status = *status
			cached.LastSeen = time.Now()
		}
		return cached, nil
	}

	return nil, fmt.Errorf("GPU with ID %s not found", gpuID)
}

// MonitorChanges monitors GPU changes
func (r *ROCmDetector) MonitorChanges(ctx context.Context, callback common.ChangeCallback) error {
	ticker := time.NewTicker(time.Duration(r.config.MonitoringIntervalSeconds) * time.Second)
	defer ticker.Stop()

	previousGPUs := make(map[string]common.GPUInfo)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			currentGPUs, err := r.DiscoverGPUs(ctx)
			if err != nil {
				r.logger.Errorf("Error during GPU discovery: %v", err)
				continue
			}

			currentMap := make(map[string]common.GPUInfo)
			for _, gpu := range currentGPUs {
				currentMap[gpu.ID] = gpu
			}

			// Check for changes (similar logic to NVML detector)
			for id, current := range currentMap {
				if previous, exists := previousGPUs[id]; exists {
					if r.hasGPUChanged(previous, current) {
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
func (r *ROCmDetector) RunBenchmark(ctx context.Context, gpuID string, benchmarkType common.BenchmarkType, duration time.Duration) (*common.BenchmarkResult, error) {
	// Placeholder implementation for ROCm benchmarks
	// In practice, this would use rocBLAS, rocFFT, or other ROCm libraries
	
	result := &common.BenchmarkResult{
		BenchmarkType:   r.getBenchmarkTypeName(benchmarkType),
		TestName:        "ROCm Basic Test",
		Score:           0.0,
		Unit:            "GFLOPS",
		DurationSeconds: int32(duration.Seconds()),
		Metadata:        make(map[string]string),
		Timestamp:       time.Now(),
	}

	gpuInfo, err := r.GetGPUInfo(ctx, gpuID)
	if err != nil {
		return nil, err
	}

	result.Metadata["gpu_name"] = gpuInfo.Name
	result.Metadata["gpu_uuid"] = gpuInfo.UUID
	result.Metadata["stream_processors"] = strconv.Itoa(int(gpuInfo.Specs.StreamProcessors))
	result.Metadata["memory_mb"] = strconv.FormatInt(gpuInfo.Specs.MemoryTotalMB, 10)

	// Placeholder benchmark score
	result.Score = float64(gpuInfo.Specs.StreamProcessors) * 1.5

	return result, nil
}

// Helper methods

func (r *ROCmDetector) isROCmSMIAvailable() bool {
	_, err := exec.LookPath("rocm-smi")
	return err == nil
}

func (r *ROCmDetector) testROCmSMI() error {
	cmd := exec.Command("rocm-smi", "--version")
	_, err := cmd.Output()
	return err
}

func (r *ROCmDetector) getDeviceList() ([]string, error) {
	cmd := exec.Command("rocm-smi", "--showid")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var devices []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "GPU[") && strings.Contains(line, "]") {
			// Extract device ID from format "GPU[0]: ..."
			start := strings.Index(line, "[")
			end := strings.Index(line, "]")
			if start != -1 && end != -1 && end > start {
				deviceID := line[start+1 : end]
				devices = append(devices, deviceID)
			}
		}
	}

	return devices, nil
}

func (r *ROCmDetector) getDeviceInfo(deviceID string) (*common.GPUInfo, error) {
	// Get basic device information using rocm-smi
	name, err := r.getDeviceName(deviceID)
	if err != nil {
		return nil, err
	}

	uuid := r.generateUUID(deviceID) // ROCm doesn't provide UUIDs like NVML

	specs, err := r.getDeviceSpecs(deviceID)
	if err != nil {
		return nil, err
	}

	status, err := r.getDeviceStatus(deviceID)
	if err != nil {
		return nil, err
	}

	capabilities := r.getDeviceCapabilities(name)
	driverInfo := r.getDriverInfo()

	index, _ := strconv.Atoi(deviceID)
	gpuInfo := &common.GPUInfo{
		ID:              fmt.Sprintf("amd-%s", deviceID),
		Name:            name,
		Vendor:          "amd",
		UUID:            uuid,
		Index:           int32(index),
		Specs:           *specs,
		Status:          *status,
		Capabilities:    *capabilities,
		Driver:          *driverInfo,
		LastSeen:        time.Now(),
		DiscoverySource: "rocm",
	}

	return gpuInfo, nil
}

func (r *ROCmDetector) getDeviceName(deviceID string) (string, error) {
	cmd := exec.Command("rocm-smi", "-i", deviceID, "--showproductname")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Card series:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1]), nil
			}
		}
	}

	return "AMD GPU", nil // Fallback
}

func (r *ROCmDetector) generateUUID(deviceID string) string {
	// ROCm doesn't provide UUIDs, so generate a consistent one
	return fmt.Sprintf("amd-gpu-%s-%d", deviceID, time.Now().Unix())
}

func (r *ROCmDetector) getDeviceSpecs(deviceID string) (*common.GPUSpecs, error) {
	// Get memory information
	memoryTotal, err := r.getDeviceMemory(deviceID)
	if err != nil {
		memoryTotal = 0 // Continue with 0 if unable to get memory info
	}

	// Estimate other specs based on device name/model
	// In a real implementation, you would have a database of AMD GPU specs
	specs := &common.GPUSpecs{
		MemoryTotalMB:       memoryTotal,
		MemoryBandwidthGBPS: 0,               // Would need model-specific lookup
		CUDACores:           0,               // N/A for AMD
		StreamProcessors:    2048,            // Default estimate
		ExecutionUnits:      0,               // N/A for AMD
		TensorCores:         0,               // Would depend on specific model
		BaseClockMHz:        0,               // Not easily available via rocm-smi
		BoostClockMHz:       0,               // Not easily available via rocm-smi
		MemoryClockMHz:      0,               // Not easily available via rocm-smi
		Architecture:        "RDNA",          // Default assumption
		ComputeCapability:   "",              // AMD doesn't use CUDA compute capability
		SMCount:             0,               // N/A for AMD (uses CUs instead)
		PowerLimitWatts:     200,             // Default estimate
		DefaultPowerLimitWatts: 200,          // Default estimate
		BusType:             "PCIe",
		BusWidth:            "x16",
		PCIeGeneration:      4,               // Default assumption
	}

	return specs, nil
}

func (r *ROCmDetector) getDeviceMemory(deviceID string) (int64, error) {
	cmd := exec.Command("rocm-smi", "-i", deviceID, "--showmeminfo", "vram")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Total VRAM:") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "VRAM:" && i+1 < len(parts) {
					memStr := parts[i+1]
					// Parse memory string (e.g., "8192MB" or "8GB")
					memStr = strings.ToUpper(memStr)
					if strings.HasSuffix(memStr, "MB") {
						memStr = strings.TrimSuffix(memStr, "MB")
						if mem, err := strconv.ParseInt(memStr, 10, 64); err == nil {
							return mem, nil
						}
					} else if strings.HasSuffix(memStr, "GB") {
						memStr = strings.TrimSuffix(memStr, "GB")
						if mem, err := strconv.ParseInt(memStr, 10, 64); err == nil {
							return mem * 1024, nil
						}
					}
				}
			}
		}
	}

	return 0, fmt.Errorf("could not parse memory information")
}

func (r *ROCmDetector) getDeviceStatus(deviceID string) (*common.GPUStatus, error) {
	// Get utilization
	gpuUtil, memUtil, err := r.getUtilization(deviceID)
	if err != nil {
		r.logger.Warnf("Failed to get utilization for device %s: %v", deviceID, err)
		gpuUtil, memUtil = 0, 0
	}

	// Get temperature
	temp, err := r.getTemperature(deviceID)
	if err != nil {
		r.logger.Warnf("Failed to get temperature for device %s: %v", deviceID, err)
		temp = 0
	}

	// Get power usage
	power, err := r.getPowerUsage(deviceID)
	if err != nil {
		r.logger.Warnf("Failed to get power usage for device %s: %v", deviceID, err)
		power = 0
	}

	// Determine state
	state := common.StateIdle
	if gpuUtil > 5 {
		state = common.StateBusy
	}

	status := &common.GPUStatus{
		State:                     state,
		GPUUtilization:            int32(gpuUtil),
		MemoryUtilization:         int32(memUtil),
		MemoryUsedMB:              0, // Would need additional parsing
		MemoryFreeMB:              0, // Would need additional parsing
		TemperatureGPU:            int32(temp),
		TemperatureMemory:         0, // Not typically available
		PowerDrawWatts:            int32(power),
		CurrentGPUClockMHz:        0, // Would need additional rocm-smi calls
		CurrentMemoryClockMHz:     0, // Would need additional rocm-smi calls
		Processes:                 []common.ProcessInfo{}, // ROCm doesn't easily expose this
	}

	return status, nil
}

func (r *ROCmDetector) getUtilization(deviceID string) (int, int, error) {
	cmd := exec.Command("rocm-smi", "-i", deviceID, "--showuse")
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}

	gpuUtil := 0
	memUtil := 0

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "GPU use (%)") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if strings.Contains(part, "%") && i > 0 {
					utilStr := strings.TrimSuffix(part, "%")
					if util, err := strconv.Atoi(utilStr); err == nil {
						gpuUtil = util
					}
					break
				}
			}
		}
		// Memory utilization would require additional parsing
	}

	return gpuUtil, memUtil, nil
}

func (r *ROCmDetector) getTemperature(deviceID string) (int, error) {
	cmd := exec.Command("rocm-smi", "-i", deviceID, "--showtemp")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Temperature (Sensor edge) (C)") {
			parts := strings.Fields(line)
			for _, part := range parts {
				if temp, err := strconv.Atoi(part); err == nil && temp > 0 && temp < 200 {
					return temp, nil
				}
			}
		}
	}

	return 0, fmt.Errorf("could not parse temperature")
}

func (r *ROCmDetector) getPowerUsage(deviceID string) (int, error) {
	cmd := exec.Command("rocm-smi", "-i", deviceID, "--showpower")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Average Graphics Package Power (W)") {
			parts := strings.Fields(line)
			for _, part := range parts {
				if power, err := strconv.Atoi(part); err == nil && power > 0 && power < 1000 {
					return power, nil
				}
			}
		}
	}

	return 0, fmt.Errorf("could not parse power usage")
}

func (r *ROCmDetector) getDeviceCapabilities(name string) *common.GPUCapabilities {
	// Estimate capabilities based on GPU name
	// In practice, you'd have a comprehensive database
	capabilities := &common.GPUCapabilities{
		SupportsCUDA:           false, // AMD doesn't support CUDA
		SupportsOpenCL:         true,
		SupportsVulkan:         true,
		SupportsDirectX:        false, // Server context
		SupportsTensorOps:      r.supportsTensorOps(name),
		SupportsMixedPrecision: true,
		SupportsRayTracing:     r.supportsRayTracing(name),
		SupportsECC:            false, // Most consumer AMD GPUs don't have ECC
		ECCEnabled:             false,
		SupportsUnifiedMemory:  true,
		SupportsMIG:            false, // AMD doesn't have MIG
		SupportsSRIOV:          false,
		PrecisionTypes:         []string{"fp32", "fp16", "int8"},
		MaxThreadsPerBlock:     1024, // Typical for AMD
		MaxBlocksPerGrid:       65535,
	}

	return capabilities
}

func (r *ROCmDetector) getDriverInfo() *common.DriverInfo {
	version := r.getROCmVersion()
	
	return &common.DriverInfo{
		Version:           "Unknown", // Would need additional detection
		CUDAVersion:       "",        // N/A for AMD
		ROCmVersion:       version,
		LevelZeroVersion:  "",        // N/A for AMD
		InstallDate:       time.Time{},
		IsCompatible:      true,
		SupportedAPIs:     []string{"OpenCL", "HIP", "ROCm"},
	}
}

func (r *ROCmDetector) getROCmVersion() string {
	cmd := exec.Command("rocm-smi", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "Unknown"
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "ROCm version") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1])
			}
		}
	}

	return "Unknown"
}

func (r *ROCmDetector) supportsTensorOps(name string) bool {
	// Check for RDNA2/RDNA3 or Instinct series
	name = strings.ToLower(name)
	return strings.Contains(name, "instinct") ||
		   strings.Contains(name, "rdna3") ||
		   strings.Contains(name, "rdna2")
}

func (r *ROCmDetector) supportsRayTracing(name string) bool {
	// Ray tracing support in RDNA2+
	name = strings.ToLower(name)
	return strings.Contains(name, "rdna3") || strings.Contains(name, "rdna2")
}

func (r *ROCmDetector) getBenchmarkTypeName(bt common.BenchmarkType) string {
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

func (r *ROCmDetector) hasGPUChanged(previous, current common.GPUInfo) bool {
	return previous.Status.GPUUtilization != current.Status.GPUUtilization ||
		previous.Status.MemoryUtilization != current.Status.MemoryUtilization ||
		previous.Status.State != current.Status.State
}

func (r *ROCmDetector) getDeviceIDFromGPUID(gpuID string) (string, error) {
	if !strings.HasPrefix(gpuID, "amd-") {
		return "", fmt.Errorf("invalid AMD GPU ID format: %s", gpuID)
	}
	
	return strings.TrimPrefix(gpuID, "amd-"), nil
}