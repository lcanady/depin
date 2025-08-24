package nvml

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"github.com/lcanady/depin/hardware/detection/common"
)

// NVMLDetector implements GPU detection for NVIDIA GPUs using NVML
type NVMLDetector struct {
	mu              sync.RWMutex
	initialized     bool
	logger          common.Logger
	config          *common.Config
	devices         []nvml.Device
	deviceInfoCache map[string]*common.GPUInfo
	lastDiscovery   time.Time
}

// NewNVMLDetector creates a new NVIDIA GPU detector
func NewNVMLDetector(logger common.Logger, config *common.Config) *NVMLDetector {
	return &NVMLDetector{
		logger:          logger,
		config:          config,
		deviceInfoCache: make(map[string]*common.GPUInfo),
	}
}

// Initialize initializes the NVML library
func (n *NVMLDetector) Initialize(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.initialized {
		return nil
	}

	n.logger.Info("Initializing NVML detector")
	
	ret := nvml.Init()
	if ret != nvml.SUCCESS {
		return fmt.Errorf("failed to initialize NVML: %v", nvml.ErrorString(ret))
	}

	// Verify NVML is working by getting driver version
	version, ret := nvml.SystemGetDriverVersion()
	if ret != nvml.SUCCESS {
		nvml.Shutdown()
		return fmt.Errorf("failed to get NVIDIA driver version: %v", nvml.ErrorString(ret))
	}

	n.logger.Infof("NVML initialized successfully, driver version: %s", version)
	n.initialized = true
	return nil
}

// Cleanup shuts down the NVML library
func (n *NVMLDetector) Cleanup() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.initialized {
		return nil
	}

	n.logger.Info("Shutting down NVML detector")
	ret := nvml.Shutdown()
	if ret != nvml.SUCCESS {
		return fmt.Errorf("failed to shutdown NVML: %v", nvml.ErrorString(ret))
	}

	n.initialized = false
	return nil
}

// IsAvailable checks if NVIDIA GPUs are available
func (n *NVMLDetector) IsAvailable() bool {
	ret := nvml.Init()
	if ret != nvml.SUCCESS {
		return false
	}

	count, ret := nvml.DeviceGetCount()
	nvml.Shutdown()
	
	return ret == nvml.SUCCESS && count > 0
}

// GetVendorName returns the vendor name
func (n *NVMLDetector) GetVendorName() string {
	return "nvidia"
}

// DiscoverGPUs discovers all NVIDIA GPUs
func (n *NVMLDetector) DiscoverGPUs(ctx context.Context) ([]common.GPUInfo, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.initialized {
		return nil, fmt.Errorf("NVML detector not initialized")
	}

	count, ret := nvml.DeviceGetCount()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("failed to get device count: %v", nvml.ErrorString(ret))
	}

	var gpus []common.GPUInfo
	n.devices = make([]nvml.Device, count)

	for i := 0; i < count; i++ {
		device, ret := nvml.DeviceGetHandleByIndex(i)
		if ret != nvml.SUCCESS {
			n.logger.Warnf("Failed to get device handle for index %d: %v", i, nvml.ErrorString(ret))
			continue
		}

		n.devices[i] = device
		gpuInfo, err := n.getDeviceInfo(device, int32(i))
		if err != nil {
			n.logger.Warnf("Failed to get info for device %d: %v", i, err)
			continue
		}

		gpus = append(gpus, *gpuInfo)
		n.deviceInfoCache[gpuInfo.ID] = gpuInfo
	}

	n.lastDiscovery = time.Now()
	n.logger.Infof("Discovered %d NVIDIA GPUs", len(gpus))
	return gpus, nil
}

// GetGPUInfo gets detailed information about a specific GPU
func (n *NVMLDetector) GetGPUInfo(ctx context.Context, gpuID string) (*common.GPUInfo, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if cached, exists := n.deviceInfoCache[gpuID]; exists {
		// Refresh the status information
		index, err := n.getDeviceIndexFromID(gpuID)
		if err != nil {
			return nil, err
		}

		if index >= 0 && index < len(n.devices) {
			status, err := n.getDeviceStatus(n.devices[index])
			if err == nil {
				cached.Status = *status
				cached.LastSeen = time.Now()
			}
		}
		return cached, nil
	}

	return nil, fmt.Errorf("GPU with ID %s not found", gpuID)
}

// MonitorChanges monitors GPU changes
func (n *NVMLDetector) MonitorChanges(ctx context.Context, callback common.ChangeCallback) error {
	ticker := time.NewTicker(time.Duration(n.config.MonitoringIntervalSeconds) * time.Second)
	defer ticker.Stop()

	previousGPUs := make(map[string]common.GPUInfo)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			currentGPUs, err := n.DiscoverGPUs(ctx)
			if err != nil {
				n.logger.Errorf("Error during GPU discovery: %v", err)
				continue
			}

			// Compare with previous state
			currentMap := make(map[string]common.GPUInfo)
			for _, gpu := range currentGPUs {
				currentMap[gpu.ID] = gpu
			}

			// Check for new or modified GPUs
			for id, current := range currentMap {
				if previous, exists := previousGPUs[id]; exists {
					if n.hasGPUChanged(previous, current) {
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

			// Check for removed GPUs
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
func (n *NVMLDetector) RunBenchmark(ctx context.Context, gpuID string, benchmarkType common.BenchmarkType, duration time.Duration) (*common.BenchmarkResult, error) {
	// This is a placeholder for actual benchmark implementation
	// In a real implementation, you would run CUDA or OpenCL benchmarks
	
	result := &common.BenchmarkResult{
		BenchmarkType:   n.getBenchmarkTypeName(benchmarkType),
		TestName:        "NVML Basic Test",
		Score:           0.0,
		Unit:            "GFLOPS",
		DurationSeconds: int32(duration.Seconds()),
		Metadata:        make(map[string]string),
		Timestamp:       time.Now(),
	}

	// Get basic GPU info for metadata
	gpuInfo, err := n.GetGPUInfo(ctx, gpuID)
	if err != nil {
		return nil, err
	}

	result.Metadata["gpu_name"] = gpuInfo.Name
	result.Metadata["gpu_uuid"] = gpuInfo.UUID
	result.Metadata["cuda_cores"] = strconv.Itoa(int(gpuInfo.Specs.CUDACores))
	result.Metadata["memory_mb"] = strconv.FormatInt(gpuInfo.Specs.MemoryTotalMB, 10)

	// Placeholder benchmark score based on specs
	// In real implementation, this would run actual compute tests
	result.Score = float64(gpuInfo.Specs.CUDACores) * 2.5 // Rough GFLOPS estimate

	return result, nil
}

// Helper methods

func (n *NVMLDetector) getDeviceInfo(device nvml.Device, index int32) (*common.GPUInfo, error) {
	// Get basic device information
	name, ret := device.GetName()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("failed to get device name: %v", nvml.ErrorString(ret))
	}

	uuid, ret := device.GetUUID()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("failed to get device UUID: %v", nvml.ErrorString(ret))
	}

	// Get memory information
	memory, ret := device.GetMemoryInfo()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("failed to get memory info: %v", nvml.ErrorString(ret))
	}

	// Get specifications
	specs, err := n.getDeviceSpecs(device)
	if err != nil {
		return nil, err
	}

	// Get current status
	status, err := n.getDeviceStatus(device)
	if err != nil {
		return nil, err
	}

	// Get capabilities
	capabilities, err := n.getDeviceCapabilities(device)
	if err != nil {
		return nil, err
	}

	// Get driver info
	driverInfo, err := n.getDriverInfo()
	if err != nil {
		return nil, err
	}

	gpuInfo := &common.GPUInfo{
		ID:              fmt.Sprintf("nvidia-%d", index),
		Name:            name,
		Vendor:          "nvidia",
		UUID:            uuid,
		Index:           index,
		Specs:           *specs,
		Status:          *status,
		Capabilities:    *capabilities,
		Driver:          *driverInfo,
		LastSeen:        time.Now(),
		DiscoverySource: "nvml",
	}

	return gpuInfo, nil
}

func (n *NVMLDetector) getDeviceSpecs(device nvml.Device) (*common.GPUSpecs, error) {
	// Get memory info
	memory, ret := device.GetMemoryInfo()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("failed to get memory info: %v", nvml.ErrorString(ret))
	}

	// Get compute capability
	major, minor, ret := device.GetCudaComputeCapability()
	computeCapability := ""
	if ret == nvml.SUCCESS {
		computeCapability = fmt.Sprintf("%d.%d", major, minor)
	}

	// Get power limits
	powerLimit, ret := device.GetPowerManagementLimitConstraints()
	defaultPowerLimit := int32(0)
	maxPowerLimit := int32(0)
	if ret == nvml.SUCCESS {
		defaultPowerLimit = int32(powerLimit.DefaultLimit / 1000) // Convert mW to W
		maxPowerLimit = int32(powerLimit.MaxLimit / 1000)
	}

	// Get current power limit
	currentPowerLimit, ret := device.GetPowerManagementDefaultLimit()
	if ret != nvml.SUCCESS {
		currentPowerLimit = uint32(defaultPowerLimit * 1000) // Convert back to mW for consistency
	}

	// Get clock speeds
	memClockMax, ret := device.GetMaxClockInfo(nvml.CLOCK_MEM)
	memClock := int32(0)
	if ret == nvml.SUCCESS {
		memClock = int32(memClockMax)
	}

	smClockMax, ret := device.GetMaxClockInfo(nvml.CLOCK_SM)
	smClock := int32(0)
	if ret == nvml.SUCCESS {
		smClock = int32(smClockMax)
	}

	// Get architecture (approximate from compute capability)
	architecture := n.getArchitectureFromComputeCapability(computeCapability)

	// Get SM count (approximation)
	smCount := n.estimateSMCount(device)

	specs := &common.GPUSpecs{
		MemoryTotalMB:          int64(memory.Total / (1024 * 1024)),
		MemoryBandwidthGBPS:    n.estimateMemoryBandwidth(device),
		CUDACores:              n.estimateCUDACores(computeCapability, smCount),
		StreamProcessors:       0, // N/A for NVIDIA
		ExecutionUnits:         0, // N/A for NVIDIA
		TensorCores:            n.estimateTensorCores(architecture),
		BaseClockMHz:           0, // Would need more complex detection
		BoostClockMHz:          smClock,
		MemoryClockMHz:         memClock,
		Architecture:           architecture,
		ComputeCapability:      computeCapability,
		SMCount:                smCount,
		PowerLimitWatts:        int32(currentPowerLimit / 1000),
		DefaultPowerLimitWatts: defaultPowerLimit,
		BusType:                "PCIe",
		BusWidth:               "x16", // Default assumption
		PCIeGeneration:         0,     // Would need PCI info
	}

	return specs, nil
}

func (n *NVMLDetector) getDeviceStatus(device nvml.Device) (*common.GPUStatus, error) {
	// Get utilization
	utilization, ret := device.GetUtilizationRates()
	gpuUtil := int32(0)
	memUtil := int32(0)
	if ret == nvml.SUCCESS {
		gpuUtil = int32(utilization.Gpu)
		memUtil = int32(utilization.Memory)
	}

	// Get memory info
	memory, ret := device.GetMemoryInfo()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("failed to get memory info: %v", nvml.ErrorString(ret))
	}

	// Get temperature
	temp, ret := device.GetTemperature(nvml.TEMPERATURE_GPU)
	temperature := int32(0)
	if ret == nvml.SUCCESS {
		temperature = int32(temp)
	}

	// Get power draw
	power, ret := device.GetPowerUsage()
	powerDraw := int32(0)
	if ret == nvml.SUCCESS {
		powerDraw = int32(power / 1000) // Convert mW to W
	}

	// Get clock speeds
	memClock, ret := device.GetClockInfo(nvml.CLOCK_MEM)
	currentMemClock := int32(0)
	if ret == nvml.SUCCESS {
		currentMemClock = int32(memClock)
	}

	smClock, ret := device.GetClockInfo(nvml.CLOCK_SM)
	currentSMClock := int32(0)
	if ret == nvml.SUCCESS {
		currentSMClock = int32(smClock)
	}

	// Get running processes
	processes, err := n.getRunningProcesses(device)
	if err != nil {
		n.logger.Warnf("Failed to get running processes: %v", err)
		processes = []common.ProcessInfo{}
	}

	// Determine state
	state := common.StateIdle
	if gpuUtil > 5 || len(processes) > 0 {
		state = common.StateBusy
	}

	status := &common.GPUStatus{
		State:                     state,
		GPUUtilization:            gpuUtil,
		MemoryUtilization:         memUtil,
		MemoryUsedMB:              int64(memory.Used / (1024 * 1024)),
		MemoryFreeMB:              int64(memory.Free / (1024 * 1024)),
		TemperatureGPU:            temperature,
		TemperatureMemory:         0, // NVML doesn't typically expose memory temp
		PowerDrawWatts:            powerDraw,
		CurrentGPUClockMHz:        currentSMClock,
		CurrentMemoryClockMHz:     currentMemClock,
		Processes:                 processes,
	}

	return status, nil
}

func (n *NVMLDetector) getDeviceCapabilities(device nvml.Device) (*common.GPUCapabilities, error) {
	// Check compute capability
	major, minor, ret := device.GetCudaComputeCapability()
	computeCapability := ""
	if ret == nvml.SUCCESS {
		computeCapability = fmt.Sprintf("%d.%d", major, minor)
	}

	// Check ECC support
	eccSupport, ret := device.GetTotalEccErrors(nvml.MEMORY_ERROR_TYPE_CORRECTED, nvml.ECC_COUNTER_TYPE_AGGREGATE)
	supportsECC := (ret == nvml.SUCCESS)

	// ECC enabled status
	eccMode, _, ret := device.GetEccMode()
	eccEnabled := (ret == nvml.SUCCESS && eccMode == nvml.FEATURE_ENABLED)

	// Estimate capabilities based on compute capability and architecture
	arch := n.getArchitectureFromComputeCapability(computeCapability)
	
	capabilities := &common.GPUCapabilities{
		SupportsCUDA:           true,
		SupportsOpenCL:         true,
		SupportsVulkan:         n.supportsVulkan(arch),
		SupportsDirectX:        false, // Typically not exposed on Linux servers
		SupportsTensorOps:      n.supportsTensorOps(computeCapability),
		SupportsMixedPrecision: n.supportsMixedPrecision(computeCapability),
		SupportsRayTracing:     n.supportsRayTracing(arch),
		SupportsECC:            supportsECC,
		ECCEnabled:             eccEnabled,
		SupportsUnifiedMemory:  n.supportsUnifiedMemory(computeCapability),
		SupportsMIG:            n.supportsMIG(arch),
		SupportsSRIOV:          false, // Requires special detection
		PrecisionTypes:         n.getSupportedPrecisionTypes(computeCapability),
		MaxThreadsPerBlock:     n.getMaxThreadsPerBlock(computeCapability),
		MaxBlocksPerGrid:       65535, // Standard CUDA limit
	}

	return capabilities, nil
}

func (n *NVMLDetector) getDriverInfo() (*common.DriverInfo, error) {
	version, ret := nvml.SystemGetDriverVersion()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("failed to get driver version: %v", nvml.ErrorString(ret))
	}

	cudaVersion, ret := nvml.SystemGetCudaDriverVersion()
	cudaVersionStr := ""
	if ret == nvml.SUCCESS {
		major := cudaVersion / 1000
		minor := (cudaVersion % 1000) / 10
		cudaVersionStr = fmt.Sprintf("%d.%d", major, minor)
	}

	driverInfo := &common.DriverInfo{
		Version:       version,
		CUDAVersion:   cudaVersionStr,
		ROCmVersion:   "", // N/A for NVIDIA
		LevelZeroVersion: "", // N/A for NVIDIA
		InstallDate:   time.Time{}, // Not available through NVML
		IsCompatible:  true,
		SupportedAPIs: []string{"CUDA", "OpenCL"},
	}

	return driverInfo, nil
}

func (n *NVMLDetector) getRunningProcesses(device nvml.Device) ([]common.ProcessInfo, error) {
	processes, ret := device.GetComputeRunningProcesses()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("failed to get running processes: %v", nvml.ErrorString(ret))
	}

	var processInfos []common.ProcessInfo
	for _, process := range processes {
		name := fmt.Sprintf("process-%d", process.Pid)
		// In a real implementation, you might resolve the process name
		
		processInfos = append(processInfos, common.ProcessInfo{
			PID:           int32(process.Pid),
			Name:          name,
			MemoryUsageMB: int64(process.UsedGpuMemory / (1024 * 1024)),
			Type:          "compute",
		})
	}

	return processInfos, nil
}

// Helper methods for estimation and capabilities

func (n *NVMLDetector) getArchitectureFromComputeCapability(cc string) string {
	switch {
	case strings.HasPrefix(cc, "8.9"):
		return "Ada Lovelace"
	case strings.HasPrefix(cc, "8.6"):
		return "Ampere"
	case strings.HasPrefix(cc, "8.0"):
		return "Ampere"
	case strings.HasPrefix(cc, "7.5"):
		return "Turing"
	case strings.HasPrefix(cc, "7.0"):
		return "Volta"
	case strings.HasPrefix(cc, "6."):
		return "Pascal"
	case strings.HasPrefix(cc, "5."):
		return "Maxwell"
	default:
		return "Unknown"
	}
}

func (n *NVMLDetector) estimateSMCount(device nvml.Device) int32 {
	// This is a simplified estimation - in practice you'd query device attributes
	return 80 // Default reasonable value
}

func (n *NVMLDetector) estimateMemoryBandwidth(device nvml.Device) int64 {
	// This would require more complex calculations based on memory type and bus width
	return 500 // GB/s - placeholder
}

func (n *NVMLDetector) estimateCUDACores(cc string, smCount int32) int32 {
	switch {
	case strings.HasPrefix(cc, "8."):
		return smCount * 128 // Ada/Ampere: 128 cores per SM
	case strings.HasPrefix(cc, "7.5"):
		return smCount * 64  // Turing: 64 cores per SM
	case strings.HasPrefix(cc, "7.0"):
		return smCount * 64  // Volta: 64 cores per SM
	case strings.HasPrefix(cc, "6."):
		return smCount * 128 // Pascal: 128 cores per SM
	default:
		return smCount * 128 // Default
	}
}

func (n *NVMLDetector) estimateTensorCores(arch string) int32 {
	switch arch {
	case "Ada Lovelace":
		return 4 // 4th gen
	case "Ampere":
		return 4 // 3rd gen
	case "Turing":
		return 8 // 2nd gen
	case "Volta":
		return 8 // 1st gen
	default:
		return 0
	}
}

func (n *NVMLDetector) supportsTensorOps(cc string) bool {
	// Tensor cores available from compute capability 7.0+
	if cc == "" {
		return false
	}
	major, _ := strconv.Atoi(strings.Split(cc, ".")[0])
	return major >= 7
}

func (n *NVMLDetector) supportsMixedPrecision(cc string) bool {
	// Mixed precision support from compute capability 7.0+
	return n.supportsTensorOps(cc)
}

func (n *NVMLDetector) supportsRayTracing(arch string) bool {
	return arch == "Ada Lovelace" || arch == "Ampere" || arch == "Turing"
}

func (n *NVMLDetector) supportsVulkan(arch string) bool {
	// Most modern NVIDIA GPUs support Vulkan
	return arch != "Unknown"
}

func (n *NVMLDetector) supportsUnifiedMemory(cc string) bool {
	if cc == "" {
		return false
	}
	major, _ := strconv.Atoi(strings.Split(cc, ".")[0])
	return major >= 6
}

func (n *NVMLDetector) supportsMIG(arch string) bool {
	return arch == "Ampere" || arch == "Ada Lovelace"
}

func (n *NVMLDetector) getSupportedPrecisionTypes(cc string) []string {
	types := []string{"fp32"}
	
	if cc == "" {
		return types
	}
	
	major, _ := strconv.Atoi(strings.Split(cc, ".")[0])
	
	if major >= 5 {
		types = append(types, "fp16")
	}
	if major >= 6 {
		types = append(types, "int8")
	}
	if major >= 7 {
		types = append(types, "int4")
	}
	
	return types
}

func (n *NVMLDetector) getMaxThreadsPerBlock(cc string) int32 {
	if cc == "" {
		return 1024
	}
	major, _ := strconv.Atoi(strings.Split(cc, ".")[0])
	if major >= 2 {
		return 1024
	}
	return 512
}

func (n *NVMLDetector) getBenchmarkTypeName(bt common.BenchmarkType) string {
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

func (n *NVMLDetector) hasGPUChanged(previous, current common.GPUInfo) bool {
	// Compare key status fields
	return previous.Status.GPUUtilization != current.Status.GPUUtilization ||
		previous.Status.MemoryUtilization != current.Status.MemoryUtilization ||
		previous.Status.State != current.Status.State ||
		len(previous.Status.Processes) != len(current.Status.Processes)
}

func (n *NVMLDetector) getDeviceIndexFromID(gpuID string) (int, error) {
	if !strings.HasPrefix(gpuID, "nvidia-") {
		return -1, fmt.Errorf("invalid NVIDIA GPU ID format: %s", gpuID)
	}
	
	indexStr := strings.TrimPrefix(gpuID, "nvidia-")
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return -1, fmt.Errorf("invalid GPU index in ID: %s", gpuID)
	}
	
	return index, nil
}