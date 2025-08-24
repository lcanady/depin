package assessment

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/lcanady/depin/services/verification/pkg/types"
)

// CapabilityAssessor evaluates GPU and provider capabilities
type CapabilityAssessor struct {
	config *AssessmentConfig
}

// AssessmentConfig contains configuration for capability assessment
type AssessmentConfig struct {
	Weights    ScoreWeights
	Thresholds ScoreThresholds
	Tiers      TierThresholds
	Grades     GradeThresholds
}

// ScoreWeights defines the relative importance of different capability areas
type ScoreWeights struct {
	Compute       float64
	Memory        float64
	Tensor        float64
	Stability     float64
	Compatibility float64
	// Provider-specific weights
	Infrastructure float64
	Security       float64
	Reliability    float64
	Performance    float64
}

// ScoreThresholds defines minimum acceptable scores
type ScoreThresholds struct {
	ComputeBaseline       float64
	MemoryBaseline        float64
	TensorBaseline        float64
	StabilityBaseline     float64
	CompatibilityBaseline float64
}

// TierThresholds defines tier classification thresholds
type TierThresholds struct {
	Enterprise   float64
	Professional float64
	Standard     float64
	Basic        float64
}

// GradeThresholds defines letter grade thresholds
type GradeThresholds struct {
	A float64
	B float64
	C float64
	D float64
}

// NewCapabilityAssessor creates a new capability assessor
func NewCapabilityAssessor(config *AssessmentConfig) *CapabilityAssessor {
	return &CapabilityAssessor{
		config: config,
	}
}

// AssessGPUCapabilities evaluates GPU capabilities from benchmark results
func (ca *CapabilityAssessor) AssessGPUCapabilities(ctx context.Context, results []*types.BenchmarkResult, gpuID uuid.UUID) (*types.CapabilityAssessment, error) {
	assessment := &types.CapabilityAssessment{
		AssessedAt: time.Now(),
		ValidUntil: time.Now().Add(24 * time.Hour), // Default 24h validity
	}
	
	// Group results by test type
	resultsByType := ca.groupResultsByType(results)
	
	// Assess each capability area
	computeScore, err := ca.assessComputeCapability(resultsByType["compute"])
	if err != nil {
		return nil, fmt.Errorf("failed to assess compute capability: %w", err)
	}
	assessment.Compute = computeScore
	
	memoryScore, err := ca.assessMemoryCapability(resultsByType["memory"])
	if err != nil {
		return nil, fmt.Errorf("failed to assess memory capability: %w", err)
	}
	assessment.Memory = memoryScore
	
	tensorScore, err := ca.assessTensorCapability(resultsByType["tensor"])
	if err != nil {
		return nil, fmt.Errorf("failed to assess tensor capability: %w", err)
	}
	assessment.Tensor = tensorScore
	
	stabilityScore, err := ca.assessStabilityCapability(resultsByType["stability"])
	if err != nil {
		return nil, fmt.Errorf("failed to assess stability capability: %w", err)
	}
	assessment.Stability = stabilityScore
	
	compatibilityScore, err := ca.assessCompatibilityCapability(resultsByType["compatibility"])
	if err != nil {
		return nil, fmt.Errorf("failed to assess compatibility capability: %w", err)
	}
	assessment.Compatibility = compatibilityScore
	
	// Calculate overall score
	assessment.OverallScore = ca.calculateOverallScore(assessment)
	
	// Determine tier and grade
	assessment.Tier = ca.determineTier(assessment.OverallScore)
	
	// Determine certifications based on scores
	assessment.Certifications = ca.determineCertifications(assessment)
	
	return assessment, nil
}

// AssessProviderCapabilities evaluates provider-level capabilities
func (ca *CapabilityAssessor) AssessProviderCapabilities(ctx context.Context, gpuAssessments []*types.CapabilityAssessment, providerMetrics map[string]interface{}) (*types.CapabilityAssessment, error) {
	assessment := &types.CapabilityAssessment{
		AssessedAt: time.Now(),
		ValidUntil: time.Now().Add(7 * 24 * time.Hour), // 7 days validity for provider assessment
	}
	
	// Assess infrastructure capability
	infraScore, err := ca.assessInfrastructureCapability(providerMetrics)
	if err != nil {
		return nil, fmt.Errorf("failed to assess infrastructure capability: %w", err)
	}
	assessment.Infrastructure = infraScore
	
	// Assess security capability
	securityScore, err := ca.assessSecurityCapability(providerMetrics)
	if err != nil {
		return nil, fmt.Errorf("failed to assess security capability: %w", err)
	}
	assessment.Security = securityScore
	
	// Assess reliability capability
	reliabilityScore, err := ca.assessReliabilityCapability(providerMetrics)
	if err != nil {
		return nil, fmt.Errorf("failed to assess reliability capability: %w", err)
	}
	assessment.Reliability = reliabilityScore
	
	// Assess aggregate performance from GPU assessments
	performanceScore, err := ca.assessAggregatePerformance(gpuAssessments)
	if err != nil {
		return nil, fmt.Errorf("failed to assess aggregate performance: %w", err)
	}
	assessment.Performance = performanceScore
	
	// Calculate overall provider score
	assessment.OverallScore = ca.calculateProviderOverallScore(assessment)
	
	// Determine tier and certifications
	assessment.Tier = ca.determineTier(assessment.OverallScore)
	assessment.Certifications = ca.determineProviderCertifications(assessment, providerMetrics)
	
	return assessment, nil
}

// assessComputeCapability evaluates compute performance
func (ca *CapabilityAssessor) assessComputeCapability(results []*types.BenchmarkResult) (*types.ComputeCapability, error) {
	if len(results) == 0 {
		return nil, fmt.Errorf("no compute benchmark results provided")
	}
	
	capability := &types.ComputeCapability{}
	
	// Extract performance metrics
	var fp32Performance, fp16Performance, int8Performance, parallelEfficiency, throughput float64
	
	for _, result := range results {
		switch result.TestName {
		case "FP32 Compute Performance":
			fp32Performance = result.Score
			throughput = fp32Performance
		case "FP16 Compute Performance":
			fp16Performance = result.Score
		case "INT8 Compute Performance":
			int8Performance = result.Score
		case "Parallel Efficiency":
			parallelEfficiency = result.Score
		}
	}
	
	// Normalize scores (0-100 scale)
	capability.FP32Performance = ca.normalizeScore(fp32Performance, 1000.0, 10000.0) // 1-10 TFLOPS range
	capability.FP16Performance = ca.normalizeScore(fp16Performance, 2000.0, 20000.0) // 2-20 TFLOPS range
	capability.INT8Performance = ca.normalizeScore(int8Performance, 10.0, 100.0)    // 10-100 TOPS range
	capability.ParallelEfficiency = ca.normalizeScore(parallelEfficiency, 70.0, 100.0) // 70-100% range
	capability.ThroughputGFLOPS = throughput
	
	// Calculate overall compute score
	weights := []float64{0.3, 0.25, 0.2, 0.25} // FP32, FP16, INT8, Parallel
	scores := []float64{capability.FP32Performance, capability.FP16Performance, capability.INT8Performance, capability.ParallelEfficiency}
	
	capability.Score = ca.weightedAverage(scores, weights)
	capability.MeetsBaseline = capability.Score >= ca.config.Thresholds.ComputeBaseline
	
	return capability, nil
}

// assessMemoryCapability evaluates memory performance
func (ca *CapabilityAssessor) assessMemoryCapability(results []*types.BenchmarkResult) (*types.MemoryCapability, error) {
	if len(results) == 0 {
		return nil, fmt.Errorf("no memory benchmark results provided")
	}
	
	capability := &types.MemoryCapability{}
	
	var bandwidth, latency, capacity, eccReliability, throughput float64
	latency = 1000.0 // Default high latency if not measured
	
	for _, result := range results {
		switch result.TestName {
		case "Memory Bandwidth":
			bandwidth = result.Score
			throughput = bandwidth
		case "Memory Latency":
			latency = result.Score
		case "Memory Allocation Performance":
			capacity = result.Score
		case "Memory Coherence (ECC)":
			eccReliability = result.Score
		}
	}
	
	// Normalize scores
	capability.BandwidthScore = ca.normalizeScore(bandwidth, 100.0, 2000.0)  // 100 GB/s - 2 TB/s
	capability.LatencyScore = ca.normalizeScore(2000.0-latency, 0.0, 1500.0) // Invert latency (lower is better)
	capability.CapacityScore = ca.normalizeScore(capacity, 1000.0, 5000.0)   // Allocation rate
	capability.ECCReliability = ca.normalizeScore(eccReliability, 99.0, 100.0) // Reliability percentage
	capability.ThroughputGBps = throughput
	capability.LatencyNs = latency
	
	// Calculate overall memory score
	weights := []float64{0.4, 0.3, 0.2, 0.1} // Bandwidth, Latency, Capacity, ECC
	scores := []float64{capability.BandwidthScore, capability.LatencyScore, capability.CapacityScore, capability.ECCReliability}
	
	capability.Score = ca.weightedAverage(scores, weights)
	capability.MeetsBaseline = capability.Score >= ca.config.Thresholds.MemoryBaseline
	
	return capability, nil
}

// assessTensorCapability evaluates tensor/AI performance
func (ca *CapabilityAssessor) assessTensorCapability(results []*types.BenchmarkResult) (*types.TensorCapability, error) {
	capability := &types.TensorCapability{
		TensorCoreSupport: false, // Default assumption
		MeetsBaseline:     false,
	}
	
	// If no tensor results, return default (some GPUs don't have tensor cores)
	if len(results) == 0 {
		capability.Score = 50.0 // Neutral score for non-tensor capable GPUs
		return capability, nil
	}
	
	var tensorPerformance, mixedPrecisionSpeedup, aiEfficiency, throughput float64
	tensorCoreDetected := false
	
	for _, result := range results {
		switch result.TestName {
		case "Tensor Core Performance":
			tensorPerformance = result.Score
			tensorCoreDetected = true
			throughput = tensorPerformance
		case "Mixed Precision Speedup":
			mixedPrecisionSpeedup = result.Score
		case "AI Workload Efficiency":
			aiEfficiency = result.Score
		}
	}
	
	capability.TensorCoreSupport = tensorCoreDetected
	
	// Normalize scores
	capability.TensorPerformance = ca.normalizeScore(tensorPerformance, 50.0, 500.0) // 50-500 TOPS
	capability.MixedPrecisionSpeedup = ca.normalizeScore(mixedPrecisionSpeedup, 1.5, 4.0) // 1.5-4x speedup
	capability.AIWorkloadEfficiency = ca.normalizeScore(aiEfficiency, 80.0, 98.0) // 80-98% efficiency
	capability.ThroughputTOPS = throughput
	
	// Calculate overall tensor score
	if tensorCoreDetected {
		weights := []float64{0.5, 0.3, 0.2} // Performance, Speedup, Efficiency
		scores := []float64{capability.TensorPerformance, capability.MixedPrecisionSpeedup, capability.AIWorkloadEfficiency}
		capability.Score = ca.weightedAverage(scores, weights)
	} else {
		// Score based on general compute capability for tensor operations
		capability.Score = capability.AIWorkloadEfficiency * 0.7 // Reduced score without dedicated tensor cores
	}
	
	capability.MeetsBaseline = capability.Score >= ca.config.Thresholds.TensorBaseline
	
	return capability, nil
}

// assessStabilityCapability evaluates system stability
func (ca *CapabilityAssessor) assessStabilityCapability(results []*types.BenchmarkResult) (*types.StabilityCapability, error) {
	if len(results) == 0 {
		return nil, fmt.Errorf("no stability benchmark results provided")
	}
	
	capability := &types.StabilityCapability{}
	
	var uptime, errorRate, thermalStability, powerStability float64
	var stableHours int
	
	for _, result := range results {
		switch result.TestName {
		case "Uptime Monitoring":
			uptime = result.Score
		case "Error Rate Analysis":
			errorRate = result.Score
		case "Thermal Stability":
			thermalStability = result.Score
		case "Power Stability":
			powerStability = result.Score
		case "Sustained Performance":
			// Extract stable hours from metadata
			if hours, ok := result.Metadata["stable_hours"].(int); ok {
				stableHours = hours
			}
		}
	}
	
	// Normalize scores
	capability.UptimePercentage = uptime
	capability.ErrorRate = errorRate
	capability.ThermalStability = ca.normalizeScore(thermalStability, 80.0, 100.0)
	capability.PowerStability = ca.normalizeScore(powerStability, 90.0, 100.0)
	capability.ConsecutiveStableHours = stableHours
	
	// Calculate overall stability score
	uptimeScore := ca.normalizeScore(uptime, 95.0, 99.9)
	errorScore := ca.normalizeScore(100.0-errorRate, 99.0, 100.0) // Invert error rate
	
	weights := []float64{0.3, 0.25, 0.25, 0.2} // Uptime, Error, Thermal, Power
	scores := []float64{uptimeScore, errorScore, capability.ThermalStability, capability.PowerStability}
	
	capability.Score = ca.weightedAverage(scores, weights)
	capability.MeetsBaseline = capability.Score >= ca.config.Thresholds.StabilityBaseline
	
	return capability, nil
}

// assessCompatibilityCapability evaluates system compatibility
func (ca *CapabilityAssessor) assessCompatibilityCapability(results []*types.BenchmarkResult) (*types.CompatibilityCapability, error) {
	capability := &types.CompatibilityCapability{
		SupportedFrameworks: []string{}, // Will be populated from results
	}
	
	// Default compatibility scores for basic GPUs
	if len(results) == 0 {
		capability.Score = 50 // Neutral compatibility score
		capability.SupportedAPIs = 2       // Assume basic API support
		capability.DriverCompatibility = 80
		capability.FrameworkSupport = 3
		capability.MeetsBaseline = capability.Score >= ca.config.Thresholds.CompatibilityBaseline
		return capability, nil
	}
	
	var apiCount, driverCompat, frameworkCount int
	var frameworks []string
	
	for _, result := range results {
		switch result.TestName {
		case "API Compatibility Test":
			if count, ok := result.Metadata["supported_apis"].(int); ok {
				apiCount = count
			}
		case "Driver Compatibility Test":
			driverCompat = int(result.Score)
		case "Framework Support Test":
			if count, ok := result.Metadata["framework_count"].(int); ok {
				frameworkCount = count
			}
			if fws, ok := result.Metadata["frameworks"].([]string); ok {
				frameworks = fws
			}
		}
	}
	
	capability.SupportedAPIs = apiCount
	capability.DriverCompatibility = driverCompat
	capability.FrameworkSupport = frameworkCount
	capability.SupportedFrameworks = frameworks
	
	// Calculate compatibility score
	apiScore := ca.normalizeScore(float64(apiCount), 2.0, 8.0)     // 2-8 supported APIs
	driverScore := float64(driverCompat)                            // Already percentage
	frameworkScore := ca.normalizeScore(float64(frameworkCount), 3.0, 15.0) // 3-15 frameworks
	
	weights := []float64{0.3, 0.4, 0.3} // API, Driver, Framework
	scores := []float64{apiScore, driverScore, frameworkScore}
	
	capability.Score = int(ca.weightedAverage(scores, weights))
	capability.MeetsBaseline = capability.Score >= int(ca.config.Thresholds.CompatibilityBaseline)
	
	return capability, nil
}

// Provider-specific assessment methods
func (ca *CapabilityAssessor) assessInfrastructureCapability(metrics map[string]interface{}) (*types.InfrastructureCapability, error) {
	capability := &types.InfrastructureCapability{}
	
	// Extract infrastructure metrics
	networkPerf := ca.getMetricValue(metrics, "network_performance", 85.0)
	storagePerf := ca.getMetricValue(metrics, "storage_performance", 80.0)
	coolingEff := ca.getMetricValue(metrics, "cooling_efficiency", 90.0)
	powerRel := ca.getMetricValue(metrics, "power_reliability", 95.0)
	
	capability.NetworkPerformance = networkPerf
	capability.StoragePerformance = storagePerf
	capability.CoolingEfficiency = coolingEff
	capability.PowerReliability = powerRel
	
	// Calculate overall infrastructure score
	weights := []float64{0.25, 0.25, 0.25, 0.25}
	scores := []float64{networkPerf, storagePerf, coolingEff, powerRel}
	
	capability.Score = ca.weightedAverage(scores, weights)
	capability.MeetsBaseline = capability.Score >= 80.0 // 80% baseline for infrastructure
	
	return capability, nil
}

func (ca *CapabilityAssessor) assessSecurityCapability(metrics map[string]interface{}) (*types.SecurityCapability, error) {
	capability := &types.SecurityCapability{}
	
	encryptionGrade := ca.getMetricValue(metrics, "encryption_grade", 85.0)
	accessControlGrade := ca.getMetricValue(metrics, "access_control_grade", 90.0)
	auditCompliance := ca.getMetricValue(metrics, "audit_compliance", 80.0)
	vulnerabilityScore := ca.getMetricValue(metrics, "vulnerability_score", 95.0)
	
	capability.EncryptionGrade = encryptionGrade
	capability.AccessControlGrade = accessControlGrade
	capability.AuditCompliance = auditCompliance
	capability.VulnerabilityScore = vulnerabilityScore
	
	weights := []float64{0.3, 0.3, 0.2, 0.2}
	scores := []float64{encryptionGrade, accessControlGrade, auditCompliance, vulnerabilityScore}
	
	capability.Score = ca.weightedAverage(scores, weights)
	capability.MeetsBaseline = capability.Score >= 85.0 // Higher baseline for security
	
	return capability, nil
}

func (ca *CapabilityAssessor) assessReliabilityCapability(metrics map[string]interface{}) (*types.ReliabilityCapability, error) {
	capability := &types.ReliabilityCapability{}
	
	uptime := ca.getMetricValue(metrics, "uptime_percentage", 99.0)
	mtbf := ca.getMetricValue(metrics, "mtbf_hours", 8760.0) // 1 year default
	recoveryTime := ca.getMetricValue(metrics, "recovery_time_minutes", 5.0)
	dataIntegrity := ca.getMetricValue(metrics, "data_integrity", 99.9)
	
	capability.UptimePercentage = uptime
	capability.MTBFHours = mtbf
	capability.RecoveryTime = recoveryTime
	capability.DataIntegrity = dataIntegrity
	
	// Normalize recovery time score (lower is better)
	recoveryScore := ca.normalizeScore(60.0-recoveryTime, 0.0, 55.0) // 5 minutes max
	mtbfScore := ca.normalizeScore(mtbf, 4380.0, 17520.0) // 6 months to 2 years
	
	weights := []float64{0.4, 0.2, 0.2, 0.2}
	scores := []float64{uptime, mtbfScore, recoveryScore, dataIntegrity}
	
	capability.Score = ca.weightedAverage(scores, weights)
	capability.MeetsBaseline = capability.Score >= 95.0 // High baseline for reliability
	
	return capability, nil
}

func (ca *CapabilityAssessor) assessAggregatePerformance(gpuAssessments []*types.CapabilityAssessment) (*types.PerformanceCapability, error) {
	capability := &types.PerformanceCapability{}
	
	if len(gpuAssessments) == 0 {
		return nil, fmt.Errorf("no GPU assessments provided for aggregate performance")
	}
	
	// Aggregate GPU performance metrics
	var totalComputeScore, totalMemoryScore, totalTensorScore float64
	var validGPUs int
	
	for _, assessment := range gpuAssessments {
		if assessment.Compute != nil {
			totalComputeScore += assessment.Compute.Score
			validGPUs++
		}
		if assessment.Memory != nil {
			totalMemoryScore += assessment.Memory.Score
		}
		if assessment.Tensor != nil {
			totalTensorScore += assessment.Tensor.Score
		}
	}
	
	if validGPUs == 0 {
		return nil, fmt.Errorf("no valid GPU assessments found")
	}
	
	// Calculate aggregate scores
	avgComputeScore := totalComputeScore / float64(validGPUs)
	avgMemoryScore := totalMemoryScore / float64(validGPUs)
	avgTensorScore := totalTensorScore / float64(validGPUs)
	
	capability.AggregateGPUPerformance = avgComputeScore
	capability.NetworkThroughput = 85.0 // Would be measured from actual network tests
	capability.StorageIOPS = 10000.0    // Would be measured from storage tests
	capability.ResponseTime = 50.0      // Average response time in ms
	
	// Calculate overall performance score
	weights := []float64{0.5, 0.2, 0.2, 0.1} // GPU, Network, Storage, Response
	scores := []float64{
		(avgComputeScore + avgMemoryScore + avgTensorScore) / 3.0,
		capability.NetworkThroughput,
		ca.normalizeScore(capability.StorageIOPS, 5000.0, 50000.0),
		ca.normalizeScore(200.0-capability.ResponseTime, 0.0, 150.0), // Invert response time
	}
	
	capability.Score = ca.weightedAverage(scores, weights)
	capability.MeetsBaseline = capability.Score >= 80.0
	
	return capability, nil
}

// Helper methods
func (ca *CapabilityAssessor) groupResultsByType(results []*types.BenchmarkResult) map[string][]*types.BenchmarkResult {
	grouped := make(map[string][]*types.BenchmarkResult)
	for _, result := range results {
		testType := result.TestType
		grouped[testType] = append(grouped[testType], result)
	}
	return grouped
}

func (ca *CapabilityAssessor) normalizeScore(value, min, max float64) float64 {
	if max <= min {
		return 50.0 // Default neutral score
	}
	
	normalized := ((value - min) / (max - min)) * 100.0
	
	// Clamp to 0-100 range
	if normalized < 0 {
		return 0
	}
	if normalized > 100 {
		return 100
	}
	
	return normalized
}

func (ca *CapabilityAssessor) weightedAverage(scores, weights []float64) float64 {
	if len(scores) != len(weights) {
		return 0.0
	}
	
	var weightedSum, totalWeight float64
	for i, score := range scores {
		weightedSum += score * weights[i]
		totalWeight += weights[i]
	}
	
	if totalWeight == 0 {
		return 0.0
	}
	
	return weightedSum / totalWeight
}

func (ca *CapabilityAssessor) calculateOverallScore(assessment *types.CapabilityAssessment) float64 {
	scores := []float64{}
	weights := []float64{}
	
	if assessment.Compute != nil {
		scores = append(scores, assessment.Compute.Score)
		weights = append(weights, ca.config.Weights.Compute)
	}
	if assessment.Memory != nil {
		scores = append(scores, assessment.Memory.Score)
		weights = append(weights, ca.config.Weights.Memory)
	}
	if assessment.Tensor != nil {
		scores = append(scores, assessment.Tensor.Score)
		weights = append(weights, ca.config.Weights.Tensor)
	}
	if assessment.Stability != nil {
		scores = append(scores, assessment.Stability.Score)
		weights = append(weights, ca.config.Weights.Stability)
	}
	if assessment.Compatibility != nil {
		scores = append(scores, float64(assessment.Compatibility.Score))
		weights = append(weights, ca.config.Weights.Compatibility)
	}
	
	return ca.weightedAverage(scores, weights)
}

func (ca *CapabilityAssessor) calculateProviderOverallScore(assessment *types.CapabilityAssessment) float64 {
	scores := []float64{}
	weights := []float64{}
	
	if assessment.Infrastructure != nil {
		scores = append(scores, assessment.Infrastructure.Score)
		weights = append(weights, ca.config.Weights.Infrastructure)
	}
	if assessment.Security != nil {
		scores = append(scores, assessment.Security.Score)
		weights = append(weights, ca.config.Weights.Security)
	}
	if assessment.Reliability != nil {
		scores = append(scores, assessment.Reliability.Score)
		weights = append(weights, ca.config.Weights.Reliability)
	}
	if assessment.Performance != nil {
		scores = append(scores, assessment.Performance.Score)
		weights = append(weights, ca.config.Weights.Performance)
	}
	
	return ca.weightedAverage(scores, weights)
}

func (ca *CapabilityAssessor) determineTier(score float64) string {
	if score >= ca.config.Tiers.Enterprise {
		return "enterprise"
	} else if score >= ca.config.Tiers.Professional {
		return "professional"
	} else if score >= ca.config.Tiers.Standard {
		return "standard"
	}
	return "basic"
}

func (ca *CapabilityAssessor) determineCertifications(assessment *types.CapabilityAssessment) []string {
	var certifications []string
	
	// Award certifications based on capability scores
	if assessment.Compute != nil && assessment.Compute.Score >= 90 {
		certifications = append(certifications, "High Performance Computing")
	}
	
	if assessment.Tensor != nil && assessment.Tensor.Score >= 85 && assessment.Tensor.TensorCoreSupport {
		certifications = append(certifications, "AI/ML Optimized")
	}
	
	if assessment.Memory != nil && assessment.Memory.ECCReliability >= 99.9 {
		certifications = append(certifications, "ECC Memory Certified")
	}
	
	if assessment.Stability != nil && assessment.Stability.Score >= 95 {
		certifications = append(certifications, "Enterprise Stability")
	}
	
	return certifications
}

func (ca *CapabilityAssessor) determineProviderCertifications(assessment *types.CapabilityAssessment, metrics map[string]interface{}) []string {
	var certifications []string
	
	if assessment.Security != nil && assessment.Security.Score >= 90 {
		certifications = append(certifications, "Security Certified")
	}
	
	if assessment.Reliability != nil && assessment.Reliability.Score >= 95 {
		certifications = append(certifications, "High Availability")
	}
	
	if assessment.Infrastructure != nil && assessment.Infrastructure.Score >= 85 {
		certifications = append(certifications, "Infrastructure Excellence")
	}
	
	// Check for compliance certifications from metrics
	if complianceStandards, ok := metrics["compliance_standards"].([]string); ok {
		certifications = append(certifications, complianceStandards...)
	}
	
	return certifications
}

func (ca *CapabilityAssessor) getMetricValue(metrics map[string]interface{}, key string, defaultValue float64) float64 {
	if value, ok := metrics[key]; ok {
		switch v := value.(type) {
		case float64:
			return v
		case int:
			return float64(v)
		case int64:
			return float64(v)
		}
	}
	return defaultValue
}