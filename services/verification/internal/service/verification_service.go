package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/lcanady/depin/services/verification/internal/assessment"
	"github.com/lcanady/depin/services/verification/internal/benchmarks"
	"github.com/lcanady/depin/services/verification/pkg/types"
	pb "github.com/lcanady/depin/services/verification/proto"
	gpu_discovery "github.com/lcanady/depin/services/gpu-discovery/proto"
	"github.com/lcanady/depin/database/inventory/repositories"
)

// VerificationService implements the verification gRPC service
type VerificationService struct {
	pb.UnimplementedVerificationServiceServer
	
	logger              *logrus.Logger
	gpuClient          gpu_discovery.GPUDiscoveryServiceClient
	repositoryManager  repositories.RepositoryManager
	capabilityAssessor *assessment.CapabilityAssessor
	
	// Benchmark components
	computeBenchmark *benchmarks.ComputeBenchmark
	memoryBenchmark  *benchmarks.MemoryBenchmark
	
	// Active verification tracking
	activeVerifications sync.Map
	verificationResults sync.Map
	
	// Event streaming
	eventStreams     map[string]chan *types.VerificationEvent
	eventStreamsMutex sync.RWMutex
}

// VerificationServiceConfig contains configuration for the verification service
type VerificationServiceConfig struct {
	DefaultTimeout    time.Duration
	MaxConcurrentJobs int
	ResultRetention   time.Duration
}

// NewVerificationService creates a new verification service instance
func NewVerificationService(
	logger *logrus.Logger,
	gpuClient gpu_discovery.GPUDiscoveryServiceClient,
	repositoryManager repositories.RepositoryManager,
	config *VerificationServiceConfig,
) *VerificationService {
	
	// Create benchmark configuration
	benchmarkConfig := &benchmarks.BenchmarkConfig{
		Duration:             60 * time.Second,
		WarmupIterations:     3,
		StressTestDuration:   2 * time.Minute,
		ParallelTests:        2,
		TimeoutMultiplier:    1.5,
		FP32MinGFLOPS:        1000.0,
		FP16MinGFLOPS:        2000.0,
		INT8MinTOPS:          10.0,
	}
	
	// Create assessment configuration
	assessmentConfig := &assessment.AssessmentConfig{
		Weights: assessment.ScoreWeights{
			Compute:        0.3,
			Memory:         0.2,
			Tensor:         0.2,
			Stability:      0.15,
			Compatibility:  0.15,
			Infrastructure: 0.25,
			Security:       0.25,
			Reliability:    0.25,
			Performance:    0.25,
		},
		Thresholds: assessment.ScoreThresholds{
			ComputeBaseline:       70.0,
			MemoryBaseline:        70.0,
			TensorBaseline:        60.0,
			StabilityBaseline:     80.0,
			CompatibilityBaseline: 60.0,
		},
		Tiers: assessment.TierThresholds{
			Enterprise:   85.0,
			Professional: 75.0,
			Standard:     65.0,
			Basic:        50.0,
		},
		Grades: assessment.GradeThresholds{
			A: 90.0,
			B: 80.0,
			C: 70.0,
			D: 60.0,
		},
	}
	
	return &VerificationService{
		logger:              logger,
		gpuClient:          gpuClient,
		repositoryManager:  repositoryManager,
		capabilityAssessor: assessment.NewCapabilityAssessor(assessmentConfig),
		computeBenchmark:   benchmarks.NewComputeBenchmark(gpuClient, benchmarkConfig),
		memoryBenchmark:    benchmarks.NewMemoryBenchmark(gpuClient, benchmarkConfig),
		eventStreams:       make(map[string]chan *types.VerificationEvent),
	}
}

// VerifyGPUCapability implements GPU capability verification
func (vs *VerificationService) VerifyGPUCapability(ctx context.Context, req *pb.GPUVerificationRequest) (*pb.GPUVerificationResponse, error) {
	vs.logger.WithFields(logrus.Fields{
		"gpu_id":      req.GpuId,
		"provider_id": req.ProviderId,
		"level":       req.Level.String(),
	}).Info("Starting GPU capability verification")
	
	// Generate verification ID
	verificationID := uuid.New()
	
	// Create verification request
	verificationReq := &types.VerificationRequest{
		ID:           verificationID,
		ResourceID:   uuid.MustParse(req.GpuId),
		ResourceType: "gpu",
		Level:        types.VerificationLevel(req.Level),
		TestsToRun:   req.TestsToRun,
		RequestedAt:  time.Now(),
		Priority:     5, // Default priority
	}
	
	// Track active verification
	vs.activeVerifications.Store(verificationID.String(), verificationReq)
	defer vs.activeVerifications.Delete(verificationID.String())
	
	// Send verification started event
	vs.broadcastEvent(&types.VerificationEvent{
		Type:           types.EventVerificationStarted,
		VerificationID: verificationID,
		ResourceID:     uuid.MustParse(req.GpuId),
		Message:        "GPU capability verification started",
		Timestamp:      time.Now(),
	})
	
	// Run benchmarks based on requested tests
	var allResults []*types.BenchmarkResult
	
	for _, testType := range req.TestsToRun {
		switch testType {
		case "compute":
			results, err := vs.computeBenchmark.RunComputeBenchmarks(ctx, req.GpuId, verificationID)
			if err != nil {
				vs.logger.WithError(err).Error("Compute benchmarks failed")
				return vs.createFailedResponse(verificationID, req.GpuId, err)
			}
			allResults = append(allResults, results...)
			
		case "memory":
			results, err := vs.memoryBenchmark.RunMemoryBenchmarks(ctx, req.GpuId, verificationID)
			if err != nil {
				vs.logger.WithError(err).Error("Memory benchmarks failed")
				return vs.createFailedResponse(verificationID, req.GpuId, err)
			}
			allResults = append(allResults, results...)
		}
	}
	
	// Assess capabilities
	assessment, err := vs.capabilityAssessor.AssessGPUCapabilities(ctx, allResults, uuid.MustParse(req.GpuId))
	if err != nil {
		vs.logger.WithError(err).Error("Capability assessment failed")
		return vs.createFailedResponse(verificationID, req.GpuId, err)
	}
	
	// Store verification result
	result := &types.VerificationResult{
		ID:               verificationID,
		RequestID:        verificationID,
		ResourceID:       uuid.MustParse(req.GpuId),
		ResourceType:     "gpu",
		Status:           types.StatusCompleted,
		Level:            types.VerificationLevel(req.Level),
		OverallScore:     assessment.OverallScore,
		Grade:            vs.calculateGrade(assessment.OverallScore),
		Assessment:       assessment,
		BenchmarkResults: allResults,
		StartedAt:        time.Now(),
		CompletedAt:      &[]time.Time{time.Now()}[0],
		ExpiresAt:        time.Now().Add(24 * time.Hour),
	}
	
	vs.verificationResults.Store(verificationID.String(), result)
	
	// Store result in database
	if err := vs.storeVerificationResult(ctx, result); err != nil {
		vs.logger.WithError(err).Error("Failed to store verification result")
	}
	
	// Send verification completed event
	vs.broadcastEvent(&types.VerificationEvent{
		Type:           types.EventVerificationCompleted,
		VerificationID: verificationID,
		ResourceID:     uuid.MustParse(req.GpuId),
		Message:        fmt.Sprintf("GPU verification completed with score %.1f", assessment.OverallScore),
		Timestamp:      time.Now(),
	})
	
	// Convert to protobuf response
	return vs.convertToGPUVerificationResponse(result), nil
}

// VerifyProviderCapability implements provider capability verification
func (vs *VerificationService) VerifyProviderCapability(ctx context.Context, req *pb.ProviderVerificationRequest) (*pb.ProviderVerificationResponse, error) {
	vs.logger.WithFields(logrus.Fields{
		"provider_id": req.ProviderId,
		"level":       req.Level.String(),
	}).Info("Starting provider capability verification")
	
	verificationID := uuid.New()
	
	// Get provider information from database
	provider, err := vs.repositoryManager.Providers().GetByID(ctx, uuid.MustParse(req.ProviderId))
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Provider not found: %v", err)
	}
	
	// Get provider GPUs if requested
	var gpuVerifications []*pb.GPUVerificationResponse
	if req.IncludeGpuVerification {
		gpus, err := vs.repositoryManager.GPUs().GetByProviderID(ctx, uuid.MustParse(req.ProviderId))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to get provider GPUs: %v", err)
		}
		
		for _, gpu := range gpus {
			gpuReq := &pb.GPUVerificationRequest{
				GpuId:      gpu.ID.String(),
				ProviderId: req.ProviderId,
				TestsToRun: []string{"compute", "memory"},
				Level:      req.Level,
			}
			
			gpuResp, err := vs.VerifyGPUCapability(ctx, gpuReq)
			if err != nil {
				vs.logger.WithError(err).Errorf("GPU verification failed for GPU %s", gpu.ID)
				continue
			}
			
			gpuVerifications = append(gpuVerifications, gpuResp)
		}
	}
	
	// Create provider metrics map from database data
	providerMetrics := map[string]interface{}{
		"uptime_percentage":    provider.UptimePercentage,
		"reliability_score":    provider.ReliabilityScore,
		"reputation":           provider.Reputation,
		"response_time_ms":     float64(provider.ResponseTimeMs),
		"network_performance":  85.0, // Would be measured from actual tests
		"storage_performance":  80.0, // Would be measured from actual tests
		"cooling_efficiency":   90.0, // Would be measured from sensors
		"power_reliability":    95.0, // Would be measured from UPS/power systems
		"encryption_grade":     90.0, // Would be assessed from security audit
		"access_control_grade": 85.0, // Would be assessed from access controls
		"audit_compliance":     88.0, // Would be assessed from compliance audit
		"vulnerability_score":  92.0, // Would be assessed from security scan
		"mtbf_hours":           float64(8760), // 1 year MTBF
		"recovery_time_minutes": 5.0,
		"data_integrity":       99.9,
		"compliance_standards": provider.Metadata.ComplianceStandards,
	}
	
	// Create mock GPU assessments for provider assessment
	var gpuAssessments []*types.CapabilityAssessment
	for _, gpuVerif := range gpuVerifications {
		// Convert protobuf assessment back to internal type
		assessment := vs.convertProtoToCapabilityAssessment(gpuVerif.Assessment)
		gpuAssessments = append(gpuAssessments, assessment)
	}
	
	// Assess provider capabilities
	providerAssessment, err := vs.capabilityAssessor.AssessProviderCapabilities(ctx, gpuAssessments, providerMetrics)
	if err != nil {
		vs.logger.WithError(err).Error("Provider capability assessment failed")
		return nil, status.Errorf(codes.Internal, "Provider assessment failed: %v", err)
	}
	
	// Create response
	response := &pb.ProviderVerificationResponse{
		VerificationId:    verificationID.String(),
		ProviderId:        req.ProviderId,
		Status:            pb.VerificationStatus_COMPLETED,
		Assessment:        vs.convertToProviderCapabilityAssessment(providerAssessment),
		GpuVerifications:  gpuVerifications,
		ComplianceResults: vs.createComplianceResults(req.ComplianceStandards, providerMetrics),
		VerifiedAt:        vs.timestampNow(),
		ExpiresAt:         vs.timestampFromTime(time.Now().Add(7 * 24 * time.Hour)),
	}
	
	return response, nil
}

// RunBenchmarkSuite runs a comprehensive benchmark suite
func (vs *VerificationService) RunBenchmarkSuite(ctx context.Context, req *pb.BenchmarkSuiteRequest) (*pb.BenchmarkSuiteResponse, error) {
	vs.logger.WithFields(logrus.Fields{
		"resource_id":   req.ResourceId,
		"resource_type": req.ResourceType,
		"benchmark_types": req.BenchmarkTypes,
	}).Info("Starting benchmark suite")
	
	suiteID := uuid.New()
	startTime := time.Now()
	
	var allResults []*types.BenchmarkResult
	
	// Run benchmarks based on resource type
	if req.ResourceType == "gpu" {
		for _, benchmarkType := range req.BenchmarkTypes {
			switch benchmarkType {
			case "compute":
				results, err := vs.computeBenchmark.RunComputeBenchmarks(ctx, req.ResourceId, suiteID)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "Compute benchmarks failed: %v", err)
				}
				allResults = append(allResults, results...)
				
			case "memory":
				results, err := vs.memoryBenchmark.RunMemoryBenchmarks(ctx, req.ResourceId, suiteID)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "Memory benchmarks failed: %v", err)
				}
				allResults = append(allResults, results...)
			}
		}
	}
	
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	
	// Create benchmark summary
	summary := vs.createBenchmarkSummary(allResults)
	
	response := &pb.BenchmarkSuiteResponse{
		SuiteId:     suiteID.String(),
		ResourceId:  req.ResourceId,
		Results:     vs.convertBenchmarkResults(allResults),
		Summary:     summary,
		StartedAt:   vs.timestampFromTime(startTime),
		CompletedAt: vs.timestampFromTime(endTime),
		DurationSeconds: int32(duration.Seconds()),
	}
	
	return response, nil
}

// GetVerificationStatus gets the status of a verification
func (vs *VerificationService) GetVerificationStatus(ctx context.Context, req *pb.VerificationStatusRequest) (*pb.VerificationStatusResponse, error) {
	var verificationID string
	
	switch req.Identifier.(type) {
	case *pb.VerificationStatusRequest_VerificationId:
		verificationID = req.GetVerificationId()
	case *pb.VerificationStatusRequest_ResourceId:
		// Find latest verification for resource
		// This would normally query the database
		return nil, status.Errorf(codes.Unimplemented, "Resource ID lookup not implemented")
	default:
		return nil, status.Errorf(codes.InvalidArgument, "Identifier must be provided")
	}
	
	// Check active verifications
	if activeVerif, ok := vs.activeVerifications.Load(verificationID); ok {
		req := activeVerif.(*types.VerificationRequest)
		return &pb.VerificationStatusResponse{
			VerificationId:       verificationID,
			ResourceId:          req.ResourceID.String(),
			Status:              pb.VerificationStatus_RUNNING,
			ProgressPercentage:  50, // Would be calculated based on completed tests
			CurrentTest:         "Running compute benchmarks",
			StartedAt:          vs.timestampFromTime(req.RequestedAt),
		}, nil
	}
	
	// Check completed verifications
	if result, ok := vs.verificationResults.Load(verificationID); ok {
		res := result.(*types.VerificationResult)
		return &pb.VerificationStatusResponse{
			VerificationId:     verificationID,
			ResourceId:        res.ResourceID.String(),
			Status:            pb.VerificationStatus(res.Status),
			ProgressPercentage: 100,
			StartedAt:         vs.timestampFromTime(res.StartedAt),
		}, nil
	}
	
	return nil, status.Errorf(codes.NotFound, "Verification not found")
}

// StreamVerificationResults streams verification results
func (vs *VerificationService) StreamVerificationResults(req *pb.VerificationStreamRequest, stream pb.VerificationService_StreamVerificationResultsServer) error {
	vs.logger.Info("Starting verification results stream")
	
	// Create event channel for this stream
	streamID := uuid.New().String()
	eventChan := make(chan *types.VerificationEvent, 100)
	
	vs.eventStreamsMutex.Lock()
	vs.eventStreams[streamID] = eventChan
	vs.eventStreamsMutex.Unlock()
	
	defer func() {
		vs.eventStreamsMutex.Lock()
		delete(vs.eventStreams, streamID)
		vs.eventStreamsMutex.Unlock()
		close(eventChan)
	}()
	
	// Stream events
	for {
		select {
		case event := <-eventChan:
			pbEvent := vs.convertToVerificationEvent(event)
			if err := stream.Send(pbEvent); err != nil {
				vs.logger.WithError(err).Error("Failed to send verification event")
				return err
			}
			
		case <-stream.Context().Done():
			vs.logger.Info("Verification results stream closed by client")
			return nil
		}
	}
}

// ValidateAllocation validates if a resource can handle an allocation
func (vs *VerificationService) ValidateAllocation(ctx context.Context, req *pb.AllocationValidationRequest) (*pb.AllocationValidationResponse, error) {
	vs.logger.WithFields(logrus.Fields{
		"provider_id": req.ProviderId,
		"gpu_id":      req.GpuId,
	}).Info("Validating allocation compatibility")
	
	// Get GPU information
	gpu, err := vs.repositoryManager.GPUs().GetByID(ctx, uuid.MustParse(req.GpuId))
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "GPU not found: %v", err)
	}
	
	// Get latest verification if available
	latestVerification, err := vs.repositoryManager.Verifications().GetLatestVerification(
		ctx, uuid.MustParse(req.GpuId), "capability")
	if err != nil {
		vs.logger.WithError(err).Warn("No recent verification found for GPU")
	}
	
	// Check basic compatibility
	compatible := true
	var issues []string
	var warnings []string
	compatibilityScore := 100.0
	
	// Check memory requirements
	if req.Requirements.MemoryMb > gpu.Specs.MemoryTotalMB {
		compatible = false
		issues = append(issues, fmt.Sprintf("Insufficient memory: required %d MB, available %d MB",
			req.Requirements.MemoryMb, gpu.Specs.MemoryTotalMB))
		compatibilityScore -= 30.0
	}
	
	// Check CUDA cores
	if req.Requirements.CudaCores > gpu.Specs.CUDACores {
		warnings = append(warnings, fmt.Sprintf("GPU has fewer CUDA cores than requested: %d vs %d",
			gpu.Specs.CUDACores, req.Requirements.CudaCores))
		compatibilityScore -= 10.0
	}
	
	// Check architecture compatibility
	if req.Requirements.Architecture != "" && req.Requirements.Architecture != gpu.Specs.Architecture {
		compatible = false
		issues = append(issues, fmt.Sprintf("Architecture mismatch: required %s, GPU has %s",
			req.Requirements.Architecture, gpu.Specs.Architecture))
		compatibilityScore -= 25.0
	}
	
	// Create capability match
	capabilityMatch := &pb.CapabilityMatch{
		MemorySufficient:       req.Requirements.MemoryMb <= gpu.Specs.MemoryTotalMB,
		ComputeSufficient:      req.Requirements.CudaCores <= gpu.Specs.CUDACores,
		ArchitectureCompatible: req.Requirements.Architecture == "" || req.Requirements.Architecture == gpu.Specs.Architecture,
		OverallMatchScore:      math.Max(0, compatibilityScore/100.0),
	}
	
	response := &pb.AllocationValidationResponse{
		IsCompatible:       compatible,
		CompatibilityScore: math.Max(0, compatibilityScore/100.0),
		CompatibilityIssues: issues,
		Warnings:           warnings,
		CapabilityMatch:    capabilityMatch,
		ValidatedAt:        vs.timestampNow(),
	}
	
	return response, nil
}

// Helper methods
func (vs *VerificationService) createFailedResponse(verificationID uuid.UUID, gpuID string, err error) (*pb.GPUVerificationResponse, error) {
	// Send failure event
	vs.broadcastEvent(&types.VerificationEvent{
		Type:           types.EventVerificationFailed,
		VerificationID: verificationID,
		ResourceID:     uuid.MustParse(gpuID),
		Message:        fmt.Sprintf("Verification failed: %v", err),
		Timestamp:      time.Now(),
	})
	
	return nil, status.Errorf(codes.Internal, "Verification failed: %v", err)
}

func (vs *VerificationService) broadcastEvent(event *types.VerificationEvent) {
	vs.eventStreamsMutex.RLock()
	defer vs.eventStreamsMutex.RUnlock()
	
	for _, eventChan := range vs.eventStreams {
		select {
		case eventChan <- event:
		default:
			// Channel is full, skip this stream
		}
	}
}

func (vs *VerificationService) calculateGrade(score float64) string {
	if score >= 90.0 {
		return "A"
	} else if score >= 80.0 {
		return "B"
	} else if score >= 70.0 {
		return "C"
	} else if score >= 60.0 {
		return "D"
	}
	return "F"
}

func (vs *VerificationService) storeVerificationResult(ctx context.Context, result *types.VerificationResult) error {
	// Store verification in database
	verification := &common.Verification{
		ID:              result.ID,
		ResourceID:      result.ResourceID,
		ResourceType:    result.ResourceType,
		VerificationType: "capability",
		Status:          string(result.Status),
		Score:           &result.OverallScore,
		Grade:           &result.Grade,
		Details:         result, // Store full result as JSONB
		VerifiedAt:      result.StartedAt,
		ExpiresAt:       result.ExpiresAt,
		VerifierID:      "verification-service",
	}
	
	return vs.repositoryManager.Verifications().CreateVerification(ctx, verification)
}

// Convert internal types to protobuf types (implementation would be extensive)
func (vs *VerificationService) convertToGPUVerificationResponse(result *types.VerificationResult) *pb.GPUVerificationResponse {
	// This is a simplified conversion - full implementation would convert all fields
	return &pb.GPUVerificationResponse{
		VerificationId: result.ID.String(),
		GpuId:         result.ResourceID.String(),
		Status:        pb.VerificationStatus(result.Status),
		VerifiedAt:    vs.timestampFromTime(result.StartedAt),
		ExpiresAt:     vs.timestampFromTime(result.ExpiresAt),
	}
}

// Utility methods for timestamp conversion
func (vs *VerificationService) timestampNow() *timestamppb.Timestamp {
	return timestamppb.Now()
}

func (vs *VerificationService) timestampFromTime(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t)
}

// Additional helper methods would be implemented here for full conversion between
// internal types and protobuf messages...