package benchmarks

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/lcanady/depin/services/verification/pkg/types"
	gpu_discovery "github.com/lcanady/depin/services/gpu-discovery/proto"
)

// ComputeBenchmark handles GPU compute performance testing
type ComputeBenchmark struct {
	gpuClient gpu_discovery.GPUDiscoveryServiceClient
	config    *BenchmarkConfig
}

// BenchmarkConfig contains configuration for benchmarks
type BenchmarkConfig struct {
	Duration             time.Duration
	WarmupIterations     int
	StressTestDuration   time.Duration
	ParallelTests        int
	TimeoutMultiplier    float64
	FP32MinGFLOPS        float64
	FP16MinGFLOPS        float64
	INT8MinTOPS          float64
}

// NewComputeBenchmark creates a new compute benchmark instance
func NewComputeBenchmark(gpuClient gpu_discovery.GPUDiscoveryServiceClient, config *BenchmarkConfig) *ComputeBenchmark {
	return &ComputeBenchmark{
		gpuClient: gpuClient,
		config:    config,
	}
}

// RunComputeBenchmarks executes all compute-related benchmarks
func (cb *ComputeBenchmark) RunComputeBenchmarks(ctx context.Context, gpuID string, verificationID uuid.UUID) ([]*types.BenchmarkResult, error) {
	var results []*types.BenchmarkResult
	
	// Get GPU information first
	gpuInfo, err := cb.getGPUInfo(ctx, gpuID)
	if err != nil {
		return nil, fmt.Errorf("failed to get GPU info: %w", err)
	}
	
	// Run FP32 compute benchmark
	fp32Result, err := cb.runFP32Benchmark(ctx, gpuID, verificationID, gpuInfo)
	if err != nil {
		return nil, fmt.Errorf("FP32 benchmark failed: %w", err)
	}
	results = append(results, fp32Result)
	
	// Run FP16 compute benchmark if supported
	if cb.supportsFP16(gpuInfo) {
		fp16Result, err := cb.runFP16Benchmark(ctx, gpuID, verificationID, gpuInfo)
		if err != nil {
			return nil, fmt.Errorf("FP16 benchmark failed: %w", err)
		}
		results = append(results, fp16Result)
	}
	
	// Run INT8 compute benchmark if supported
	if cb.supportsINT8(gpuInfo) {
		int8Result, err := cb.runINT8Benchmark(ctx, gpuID, verificationID, gpuInfo)
		if err != nil {
			return nil, fmt.Errorf("INT8 benchmark failed: %w", err)
		}
		results = append(results, int8Result)
	}
	
	// Run parallel efficiency test
	parallelResult, err := cb.runParallelEfficiencyTest(ctx, gpuID, verificationID, gpuInfo)
	if err != nil {
		return nil, fmt.Errorf("parallel efficiency test failed: %w", err)
	}
	results = append(results, parallelResult)
	
	// Run sustained performance test
	sustainedResult, err := cb.runSustainedPerformanceTest(ctx, gpuID, verificationID, gpuInfo)
	if err != nil {
		return nil, fmt.Errorf("sustained performance test failed: %w", err)
	}
	results = append(results, sustainedResult)
	
	return results, nil
}

// runFP32Benchmark executes FP32 compute performance test
func (cb *ComputeBenchmark) runFP32Benchmark(ctx context.Context, gpuID string, verificationID uuid.UUID, gpuInfo *gpu_discovery.GPUInfo) (*types.BenchmarkResult, error) {
	startTime := time.Now()
	
	// Create benchmark result
	result := &types.BenchmarkResult{
		ID:             uuid.New(),
		VerificationID: verificationID,
		ResourceID:     uuid.MustParse(gpuID),
		TestName:       "FP32 Compute Performance",
		TestType:       "compute",
		Unit:           "GFLOPS",
		StartedAt:      startTime,
		Metadata:       make(map[string]interface{}),
	}
	
	// Simulate matrix multiplication benchmark
	// In real implementation, this would invoke actual GPU compute kernels
	gflops, metrics, err := cb.simulateFP32MatrixMultiplication(ctx, gpuInfo)
	if err != nil {
		return nil, fmt.Errorf("FP32 matrix multiplication failed: %w", err)
	}
	
	endTime := time.Now()
	result.CompletedAt = endTime
	result.Duration = endTime.Sub(startTime)
	result.Score = gflops
	result.BaselineScore = cb.config.FP32MinGFLOPS
	result.PerformanceRatio = gflops / cb.config.FP32MinGFLOPS
	result.Passed = gflops >= cb.config.FP32MinGFLOPS
	result.Metrics = metrics
	
	// Add metadata
	result.Metadata["matrix_size"] = "4096x4096"
	result.Metadata["iterations"] = cb.config.WarmupIterations + 10
	result.Metadata["cuda_cores"] = gpuInfo.Specs.CudaCores
	result.Metadata["base_clock"] = gpuInfo.Specs.BaseClockMhz
	
	return result, nil
}

// runFP16Benchmark executes FP16 compute performance test
func (cb *ComputeBenchmark) runFP16Benchmark(ctx context.Context, gpuID string, verificationID uuid.UUID, gpuInfo *gpu_discovery.GPUInfo) (*types.BenchmarkResult, error) {
	startTime := time.Now()
	
	result := &types.BenchmarkResult{
		ID:             uuid.New(),
		VerificationID: verificationID,
		ResourceID:     uuid.MustParse(gpuID),
		TestName:       "FP16 Compute Performance",
		TestType:       "compute",
		Unit:           "GFLOPS",
		StartedAt:      startTime,
		Metadata:       make(map[string]interface{}),
	}
	
	// Simulate FP16 matrix multiplication
	gflops, metrics, err := cb.simulateFP16MatrixMultiplication(ctx, gpuInfo)
	if err != nil {
		return nil, fmt.Errorf("FP16 matrix multiplication failed: %w", err)
	}
	
	endTime := time.Now()
	result.CompletedAt = endTime
	result.Duration = endTime.Sub(startTime)
	result.Score = gflops
	result.BaselineScore = cb.config.FP16MinGFLOPS
	result.PerformanceRatio = gflops / cb.config.FP16MinGFLOPS
	result.Passed = gflops >= cb.config.FP16MinGFLOPS
	result.Metrics = metrics
	
	result.Metadata["matrix_size"] = "8192x8192"
	result.Metadata["precision"] = "half"
	result.Metadata["tensor_cores_used"] = cb.supportsTensorCores(gpuInfo)
	
	return result, nil
}

// runINT8Benchmark executes INT8 compute performance test
func (cb *ComputeBenchmark) runINT8Benchmark(ctx context.Context, gpuID string, verificationID uuid.UUID, gpuInfo *gpu_discovery.GPUInfo) (*types.BenchmarkResult, error) {
	startTime := time.Now()
	
	result := &types.BenchmarkResult{
		ID:             uuid.New(),
		VerificationID: verificationID,
		ResourceID:     uuid.MustParse(gpuID),
		TestName:       "INT8 Compute Performance",
		TestType:       "compute",
		Unit:           "TOPS",
		StartedAt:      startTime,
		Metadata:       make(map[string]interface{}),
	}
	
	// Simulate INT8 inference workload
	tops, metrics, err := cb.simulateINT8Inference(ctx, gpuInfo)
	if err != nil {
		return nil, fmt.Errorf("INT8 inference failed: %w", err)
	}
	
	endTime := time.Now()
	result.CompletedAt = endTime
	result.Duration = endTime.Sub(startTime)
	result.Score = tops
	result.BaselineScore = cb.config.INT8MinTOPS
	result.PerformanceRatio = tops / cb.config.INT8MinTOPS
	result.Passed = tops >= cb.config.INT8MinTOPS
	result.Metrics = metrics
	
	result.Metadata["workload_type"] = "inference"
	result.Metadata["model_size"] = "resnet50"
	result.Metadata["batch_size"] = 64
	
	return result, nil
}

// runParallelEfficiencyTest tests how well GPU utilizes parallel compute units
func (cb *ComputeBenchmark) runParallelEfficiencyTest(ctx context.Context, gpuID string, verificationID uuid.UUID, gpuInfo *gpu_discovery.GPUInfo) (*types.BenchmarkResult, error) {
	startTime := time.Now()
	
	result := &types.BenchmarkResult{
		ID:             uuid.New(),
		VerificationID: verificationID,
		ResourceID:     uuid.MustParse(gpuID),
		TestName:       "Parallel Efficiency",
		TestType:       "compute",
		Unit:           "efficiency_percentage",
		StartedAt:      startTime,
		Metadata:       make(map[string]interface{}),
	}
	
	// Test with increasing thread counts
	efficiency, metrics, err := cb.measureParallelEfficiency(ctx, gpuInfo)
	if err != nil {
		return nil, fmt.Errorf("parallel efficiency test failed: %w", err)
	}
	
	endTime := time.Now()
	result.CompletedAt = endTime
	result.Duration = endTime.Sub(startTime)
	result.Score = efficiency
	result.BaselineScore = 85.0 // 85% efficiency threshold
	result.PerformanceRatio = efficiency / 85.0
	result.Passed = efficiency >= 85.0
	result.Metrics = metrics
	
	result.Metadata["test_type"] = "thread_scaling"
	result.Metadata["max_threads"] = gpuInfo.Capabilities.MaxThreadsPerBlock
	result.Metadata["streaming_multiprocessors"] = gpuInfo.Specs.SmCount
	
	return result, nil
}

// runSustainedPerformanceTest tests performance under sustained load
func (cb *ComputeBenchmark) runSustainedPerformanceTest(ctx context.Context, gpuID string, verificationID uuid.UUID, gpuInfo *gpu_discovery.GPUInfo) (*types.BenchmarkResult, error) {
	startTime := time.Now()
	
	result := &types.BenchmarkResult{
		ID:             uuid.New(),
		VerificationID: verificationID,
		ResourceID:     uuid.MustParse(gpuID),
		TestName:       "Sustained Performance",
		TestType:       "compute",
		Unit:           "stability_percentage",
		StartedAt:      startTime,
		Metadata:       make(map[string]interface{}),
	}
	
	// Run sustained load for specified duration
	stability, metrics, err := cb.measureSustainedPerformance(ctx, gpuInfo, cb.config.StressTestDuration)
	if err != nil {
		return nil, fmt.Errorf("sustained performance test failed: %w", err)
	}
	
	endTime := time.Now()
	result.CompletedAt = endTime
	result.Duration = endTime.Sub(startTime)
	result.Score = stability
	result.BaselineScore = 95.0 // 95% stability threshold
	result.PerformanceRatio = stability / 95.0
	result.Passed = stability >= 95.0
	result.Metrics = metrics
	
	result.Metadata["test_duration"] = cb.config.StressTestDuration.String()
	result.Metadata["thermal_throttling"] = stability < 95.0
	result.Metadata["power_throttling"] = false // Would be detected in actual implementation
	
	return result, nil
}

// Helper methods for GPU capability detection
func (cb *ComputeBenchmark) supportsFP16(gpuInfo *gpu_discovery.GPUInfo) bool {
	// Check if GPU supports FP16 operations
	for _, precision := range gpuInfo.Capabilities.PrecisionTypes {
		if precision == "fp16" || precision == "half" {
			return true
		}
	}
	return false
}

func (cb *ComputeBenchmark) supportsINT8(gpuInfo *gpu_discovery.GPUInfo) bool {
	// Check if GPU supports INT8 operations
	for _, precision := range gpuInfo.Capabilities.PrecisionTypes {
		if precision == "int8" {
			return true
		}
	}
	return false
}

func (cb *ComputeBenchmark) supportsTensorCores(gpuInfo *gpu_discovery.GPUInfo) bool {
	return gpuInfo.Capabilities.SupportsTensorOps && gpuInfo.Specs.TensorCores > 0
}

func (cb *ComputeBenchmark) getGPUInfo(ctx context.Context, gpuID string) (*gpu_discovery.GPUInfo, error) {
	req := &gpu_discovery.GetGPUInfoRequest{
		GpuId: gpuID,
		IncludeBenchmarks: false,
	}
	
	resp, err := cb.gpuClient.GetGPUInfo(ctx, req)
	if err != nil {
		return nil, err
	}
	
	return resp.Gpu, nil
}

// Simulation methods (in real implementation these would use actual GPU kernels)
func (cb *ComputeBenchmark) simulateFP32MatrixMultiplication(ctx context.Context, gpuInfo *gpu_discovery.GPUInfo) (float64, []*types.Metric, error) {
	// Simulate based on GPU specifications
	cudaCores := float64(gpuInfo.Specs.CudaCores)
	baseClock := float64(gpuInfo.Specs.BaseClockMhz)
	boostClock := float64(gpuInfo.Specs.BoostClockMhz)
	
	// Theoretical GFLOPS calculation
	effectiveClock := (baseClock + boostClock) / 2.0 * 1000000 // Convert MHz to Hz
	theoreticalGFLOPS := (cudaCores * effectiveClock * 2) / 1e9 // 2 ops per core per cycle
	
	// Apply efficiency factor (typically 80-95% of theoretical)
	efficiency := 0.85 + (0.1 * (boostClock - baseClock) / boostClock)
	actualGFLOPS := theoreticalGFLOPS * efficiency
	
	// Add some variance based on "measurement"
	variance := 0.95 + (0.1 * math.Sin(float64(time.Now().UnixNano())))
	measuredGFLOPS := actualGFLOPS * variance
	
	metrics := []*types.Metric{
		{
			Name:         "theoretical_gflops",
			Value:        theoreticalGFLOPS,
			Unit:         "GFLOPS",
			MinThreshold: theoreticalGFLOPS * 0.8,
			MaxThreshold: theoreticalGFLOPS,
			WithinLimits: measuredGFLOPS >= theoreticalGFLOPS * 0.8,
		},
		{
			Name:         "efficiency",
			Value:        efficiency * 100,
			Unit:         "percentage",
			MinThreshold: 80.0,
			MaxThreshold: 100.0,
			WithinLimits: efficiency >= 0.8,
		},
		{
			Name:         "peak_utilization",
			Value:        95.0,
			Unit:         "percentage",
			MinThreshold: 85.0,
			MaxThreshold: 100.0,
			WithinLimits: true,
		},
	}
	
	return measuredGFLOPS, metrics, nil
}

func (cb *ComputeBenchmark) simulateFP16MatrixMultiplication(ctx context.Context, gpuInfo *gpu_discovery.GPUInfo) (float64, []*types.Metric, error) {
	// FP16 typically provides ~2x performance improvement over FP32
	fp32GFLOPS, _, err := cb.simulateFP32MatrixMultiplication(ctx, gpuInfo)
	if err != nil {
		return 0, nil, err
	}
	
	speedupFactor := 1.8 // Slightly less than 2x due to overhead
	if cb.supportsTensorCores(gpuInfo) {
		speedupFactor = 2.5 // Tensor cores provide additional acceleration
	}
	
	fp16GFLOPS := fp32GFLOPS * speedupFactor
	
	metrics := []*types.Metric{
		{
			Name:         "fp32_baseline_gflops",
			Value:        fp32GFLOPS,
			Unit:         "GFLOPS",
			MinThreshold: 0,
			MaxThreshold: math.Inf(1),
			WithinLimits: true,
		},
		{
			Name:         "speedup_factor",
			Value:        speedupFactor,
			Unit:         "ratio",
			MinThreshold: 1.5,
			MaxThreshold: 3.0,
			WithinLimits: speedupFactor >= 1.5,
		},
		{
			Name:         "tensor_cores_utilized",
			Value:        func() float64 { if cb.supportsTensorCores(gpuInfo) { return 100 } else { return 0 } }(),
			Unit:         "percentage",
			MinThreshold: 0,
			MaxThreshold: 100,
			WithinLimits: true,
		},
	}
	
	return fp16GFLOPS, metrics, nil
}

func (cb *ComputeBenchmark) simulateINT8Inference(ctx context.Context, gpuInfo *gpu_discovery.GPUInfo) (float64, []*types.Metric, error) {
	// INT8 provides significant speedup for inference workloads
	fp32GFLOPS, _, err := cb.simulateFP32MatrixMultiplication(ctx, gpuInfo)
	if err != nil {
		return 0, nil, err
	}
	
	// Convert GFLOPS to TOPS (operations are simpler for INT8)
	int8SpeedupFactor := 4.0 // INT8 is typically 4x faster than FP32
	if cb.supportsTensorCores(gpuInfo) {
		int8SpeedupFactor = 6.0 // Tensor cores optimize INT8 operations
	}
	
	tops := (fp32GFLOPS * int8SpeedupFactor) / 1000.0 // Convert to TOPS
	
	metrics := []*types.Metric{
		{
			Name:         "inference_throughput",
			Value:        tops * 1000, // Convert back to GOPS for display
			Unit:         "GOPS",
			MinThreshold: cb.config.INT8MinTOPS * 1000,
			MaxThreshold: math.Inf(1),
			WithinLimits: tops >= cb.config.INT8MinTOPS,
		},
		{
			Name:         "int8_speedup_factor",
			Value:        int8SpeedupFactor,
			Unit:         "ratio",
			MinThreshold: 3.0,
			MaxThreshold: 8.0,
			WithinLimits: int8SpeedupFactor >= 3.0,
		},
		{
			Name:         "quantization_accuracy",
			Value:        98.5, // Simulated accuracy loss from quantization
			Unit:         "percentage",
			MinThreshold: 95.0,
			MaxThreshold: 100.0,
			WithinLimits: true,
		},
	}
	
	return tops, metrics, nil
}

func (cb *ComputeBenchmark) measureParallelEfficiency(ctx context.Context, gpuInfo *gpu_discovery.GPUInfo) (float64, []*types.Metric, error) {
	// Simulate parallel efficiency based on GPU architecture
	smCount := float64(gpuInfo.Specs.SmCount)
	maxThreads := float64(gpuInfo.Capabilities.MaxThreadsPerBlock)
	
	// Modern GPUs typically achieve 85-95% parallel efficiency
	baseEfficiency := 0.85
	architectureBonus := 0.0
	
	// Newer architectures have better efficiency
	if contains(gpuInfo.Specs.Architecture, []string{"Ada Lovelace", "RDNA3", "Xe-HPG"}) {
		architectureBonus = 0.08
	} else if contains(gpuInfo.Specs.Architecture, []string{"Ampere", "RDNA2", "Xe-HP"}) {
		architectureBonus = 0.05
	}
	
	efficiency := (baseEfficiency + architectureBonus) * 100
	
	metrics := []*types.Metric{
		{
			Name:         "streaming_multiprocessors",
			Value:        smCount,
			Unit:         "count",
			MinThreshold: 10,
			MaxThreshold: math.Inf(1),
			WithinLimits: smCount >= 10,
		},
		{
			Name:         "max_concurrent_threads",
			Value:        smCount * maxThreads,
			Unit:         "threads",
			MinThreshold: 10000,
			MaxThreshold: math.Inf(1),
			WithinLimits: smCount * maxThreads >= 10000,
		},
		{
			Name:         "architecture_bonus",
			Value:        architectureBonus * 100,
			Unit:         "percentage",
			MinThreshold: 0,
			MaxThreshold: 15,
			WithinLimits: true,
		},
	}
	
	return efficiency, metrics, nil
}

func (cb *ComputeBenchmark) measureSustainedPerformance(ctx context.Context, gpuInfo *gpu_discovery.GPUInfo, duration time.Duration) (float64, []*types.Metric, error) {
	// Simulate sustained performance based on thermal and power characteristics
	powerLimit := float64(gpuInfo.Specs.PowerLimitWatts)
	defaultPowerLimit := float64(gpuInfo.Specs.DefaultPowerLimitWatts)
	
	// Higher power headroom typically means better sustained performance
	powerHeadroom := (powerLimit - defaultPowerLimit) / defaultPowerLimit
	
	baseSustained := 95.0 // Start with 95% sustained performance
	if powerHeadroom < 0.1 {
		baseSustained = 88.0 // Limited power headroom reduces sustained performance
	} else if powerHeadroom > 0.3 {
		baseSustained = 97.0 // Good power headroom improves sustained performance
	}
	
	// Longer duration tests might show thermal throttling
	durationMinutes := duration.Minutes()
	if durationMinutes > 5 {
		thermalPenalty := math.Min((durationMinutes - 5) * 0.5, 5.0)
		baseSustained -= thermalPenalty
	}
	
	metrics := []*types.Metric{
		{
			Name:         "power_headroom",
			Value:        powerHeadroom * 100,
			Unit:         "percentage",
			MinThreshold: 10.0,
			MaxThreshold: 50.0,
			WithinLimits: powerHeadroom >= 0.1,
		},
		{
			Name:         "thermal_throttling",
			Value:        func() float64 { if baseSustained < 95.0 { return (95.0 - baseSustained) } else { return 0 } }(),
			Unit:         "percentage",
			MinThreshold: 0,
			MaxThreshold: 10,
			WithinLimits: baseSustained >= 90.0,
		},
		{
			Name:         "test_duration",
			Value:        durationMinutes,
			Unit:         "minutes",
			MinThreshold: 1.0,
			MaxThreshold: 10.0,
			WithinLimits: true,
		},
	}
	
	return baseSustained, metrics, nil
}

// Utility function to check if a string exists in a slice
func contains(item string, slice []string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}