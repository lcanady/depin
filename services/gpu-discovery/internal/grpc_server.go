package internal

import (
	"context"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	
	pb "github.com/lcanady/depin/services/gpu-discovery/proto"
	"github.com/lcanady/depin/hardware/detection/common"
)

// GPUDiscoveryServer implements the gRPC server for GPU discovery
type GPUDiscoveryServer struct {
	pb.UnimplementedGPUDiscoveryServiceServer
	service *GPUDiscoveryService
	server  *grpc.Server
}

// NewGPUDiscoveryServer creates a new gRPC server
func NewGPUDiscoveryServer(service *GPUDiscoveryService) *GPUDiscoveryServer {
	return &GPUDiscoveryServer{
		service: service,
	}
}

// Start starts the gRPC server
func (s *GPUDiscoveryServer) Start(port int) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %v", port, err)
	}

	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(s.service.config.GRPCMaxMsgSize),
		grpc.MaxSendMsgSize(s.service.config.GRPCMaxMsgSize),
	}

	s.server = grpc.NewServer(opts...)
	pb.RegisterGPUDiscoveryServiceServer(s.server, s)

	s.service.logger.Infof("Starting gRPC server on port %d", port)
	
	go func() {
		if err := s.server.Serve(listener); err != nil {
			s.service.logger.Errorf("gRPC server failed: %v", err)
		}
	}()

	return nil
}

// Stop stops the gRPC server
func (s *GPUDiscoveryServer) Stop() {
	if s.server != nil {
		s.service.logger.Info("Stopping gRPC server")
		s.server.GracefulStop()
	}
}

// gRPC service implementations

// DiscoverGPUs discovers all GPUs on the current system
func (s *GPUDiscoveryServer) DiscoverGPUs(ctx context.Context, req *pb.DiscoverRequest) (*pb.DiscoverResponse, error) {
	s.service.logger.Debugf("DiscoverGPUs request: force_refresh=%v, vendor_filter=%v", 
		req.ForceRefresh, req.VendorFilter)

	gpus, err := s.service.DiscoverGPUs(req.ForceRefresh, req.VendorFilter)
	if err != nil {
		s.service.logger.Errorf("GPU discovery failed: %v", err)
		return nil, status.Errorf(codes.Internal, "discovery failed: %v", err)
	}

	// Convert to protobuf format
	pbGPUs := make([]*pb.GPUInfo, len(gpus))
	for i, gpu := range gpus {
		pbGPU, err := s.convertGPUInfoToProto(gpu)
		if err != nil {
			s.service.logger.Errorf("Failed to convert GPU info: %v", err)
			continue
		}
		pbGPUs[i] = pbGPU
	}

	response := &pb.DiscoverResponse{
		Gpus:          pbGPUs,
		DiscoveryTime: timestamppb.Now(),
		SystemId:      s.generateSystemID(),
	}

	s.service.logger.Infof("Discovered %d GPUs", len(pbGPUs))
	return response, nil
}

// GetGPUInfo gets detailed information about a specific GPU
func (s *GPUDiscoveryServer) GetGPUInfo(ctx context.Context, req *pb.GetGPUInfoRequest) (*pb.GetGPUInfoResponse, error) {
	s.service.logger.Debugf("GetGPUInfo request: gpu_id=%s, include_benchmarks=%v", 
		req.GpuId, req.IncludeBenchmarks)

	gpu, benchmarks, err := s.service.GetGPUInfo(req.GpuId, req.IncludeBenchmarks)
	if err != nil {
		s.service.logger.Errorf("Failed to get GPU info: %v", err)
		return nil, status.Errorf(codes.NotFound, "GPU not found: %v", err)
	}

	pbGPU, err := s.convertGPUInfoToProto(*gpu)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert GPU info: %v", err)
	}

	var pbBenchmarks []*pb.BenchmarkResult
	for _, benchmark := range benchmarks {
		pbBenchmark := s.convertBenchmarkResultToProto(benchmark)
		pbBenchmarks = append(pbBenchmarks, pbBenchmark)
	}

	response := &pb.GetGPUInfoResponse{
		Gpu:        pbGPU,
		Benchmarks: pbBenchmarks,
	}

	return response, nil
}

// MonitorGPUs monitors GPU changes (streaming)
func (s *GPUDiscoveryServer) MonitorGPUs(req *pb.MonitorRequest, stream pb.GPUDiscoveryService_MonitorGPUsServer) error {
	s.service.logger.Infof("Starting GPU monitoring stream: polling_interval=%ds", req.PollingIntervalSeconds)

	// Create a channel to receive GPU changes
	changeChan := make(chan common.GPUChange, 100)

	// Start monitoring
	err := s.service.StartMonitoring(func(change common.GPUChange) {
		select {
		case changeChan <- change:
		default:
			s.service.logger.Warn("Change channel is full, dropping GPU change event")
		}
	})
	if err != nil {
		return status.Errorf(codes.Internal, "failed to start monitoring: %v", err)
	}

	// Stream changes to client
	for {
		select {
		case <-stream.Context().Done():
			s.service.logger.Debug("GPU monitoring stream cancelled by client")
			return stream.Context().Err()
			
		case change := <-changeChan:
			pbGPU, err := s.convertGPUInfoToProto(change.GPU)
			if err != nil {
				s.service.logger.Errorf("Failed to convert GPU info for change event: %v", err)
				continue
			}

			event := &pb.GPUChangeEvent{
				ChangeType:  s.convertChangeTypeToProto(change.Type),
				Gpu:         pbGPU,
				Timestamp:   timestamppb.New(change.Timestamp),
				Description: change.Description,
			}

			if err := stream.Send(event); err != nil {
				s.service.logger.Errorf("Failed to send change event: %v", err)
				return status.Errorf(codes.Internal, "stream send failed: %v", err)
			}

		case <-time.After(30 * time.Second):
			// Send periodic heartbeat if no changes
			if req.IncludePerformanceMetrics {
				// Could send performance updates here
			}
		}
	}
}

// BenchmarkGPU performs capability benchmark on GPU
func (s *GPUDiscoveryServer) BenchmarkGPU(ctx context.Context, req *pb.BenchmarkRequest) (*pb.BenchmarkResponse, error) {
	s.service.logger.Infof("Benchmark request: gpu_id=%s, types=%v, duration=%ds", 
		req.GpuId, req.BenchmarkTypes, req.DurationSeconds)

	duration := time.Duration(req.DurationSeconds) * time.Second
	if duration <= 0 {
		duration = 30 * time.Second // Default
	}

	results, err := s.service.RunBenchmark(req.GpuId, req.BenchmarkTypes, duration)
	if err != nil {
		s.service.logger.Errorf("Benchmark failed: %v", err)
		return nil, status.Errorf(codes.Internal, "benchmark failed: %v", err)
	}

	var pbResults []*pb.BenchmarkResult
	for _, result := range results {
		pbResult := s.convertBenchmarkResultToProto(result)
		pbResults = append(pbResults, pbResult)
	}

	response := &pb.BenchmarkResponse{
		GpuId:         req.GpuId,
		Results:       pbResults,
		BenchmarkTime: timestamppb.Now(),
	}

	return response, nil
}

// GetSystemSummary gets system GPU summary
func (s *GPUDiscoveryServer) GetSystemSummary(ctx context.Context, req *pb.SystemSummaryRequest) (*pb.SystemSummaryResponse, error) {
	s.service.logger.Debug("GetSystemSummary request")

	systemInfo, err := s.service.GetSystemSummary(req.IncludeOfflineGpus)
	if err != nil {
		s.service.logger.Errorf("Failed to get system summary: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to get system summary: %v", err)
	}

	response := &pb.SystemSummaryResponse{
		TotalGpus:           systemInfo.TotalGPUs,
		AvailableGpus:       systemInfo.AvailableGPUs,
		TotalMemoryMb:       systemInfo.TotalMemoryMB,
		AvailableMemoryMb:   systemInfo.AvailableMemoryMB,
		SupportedVendors:    systemInfo.SupportedVendors,
		GpuCountByVendor:    systemInfo.GPUCountByVendor,
	}

	return response, nil
}

// Conversion helper methods

func (s *GPUDiscoveryServer) convertGPUInfoToProto(gpu common.GPUInfo) (*pb.GPUInfo, error) {
	return &pb.GPUInfo{
		GpuId:           gpu.ID,
		Name:            gpu.Name,
		Vendor:          gpu.Vendor,
		Uuid:            gpu.UUID,
		Index:           gpu.Index,
		Specs:           s.convertSpecsToProto(gpu.Specs),
		Status:          s.convertStatusToProto(gpu.Status),
		Capabilities:    s.convertCapabilitiesToProto(gpu.Capabilities),
		Driver:          s.convertDriverInfoToProto(gpu.Driver),
		LastSeen:        timestamppb.New(gpu.LastSeen),
		DiscoverySource: gpu.DiscoverySource,
	}, nil
}

func (s *GPUDiscoveryServer) convertSpecsToProto(specs common.GPUSpecs) *pb.GPUSpecs {
	return &pb.GPUSpecs{
		MemoryTotalMb:          specs.MemoryTotalMB,
		MemoryBandwidthGbps:    specs.MemoryBandwidthGBPS,
		CudaCores:              specs.CUDACores,
		StreamProcessors:       specs.StreamProcessors,
		ExecutionUnits:         specs.ExecutionUnits,
		TensorCores:            specs.TensorCores,
		BaseClockMhz:           specs.BaseClockMHz,
		BoostClockMhz:          specs.BoostClockMHz,
		MemoryClockMhz:         specs.MemoryClockMHz,
		Architecture:           specs.Architecture,
		ComputeCapability:      specs.ComputeCapability,
		SmCount:                specs.SMCount,
		PowerLimitWatts:        specs.PowerLimitWatts,
		DefaultPowerLimitWatts: specs.DefaultPowerLimitWatts,
		BusType:                specs.BusType,
		BusWidth:               specs.BusWidth,
		PcieGeneration:         specs.PCIeGeneration,
	}
}

func (s *GPUDiscoveryServer) convertStatusToProto(status common.GPUStatus) *pb.GPUStatus {
	var processes []*pb.ProcessInfo
	for _, proc := range status.Processes {
		processes = append(processes, &pb.ProcessInfo{
			Pid:           proc.PID,
			Name:          proc.Name,
			MemoryUsageMb: proc.MemoryUsageMB,
			Type:          proc.Type,
		})
	}

	return &pb.GPUStatus{
		State:                     s.convertStateToProto(status.State),
		GpuUtilization:            status.GPUUtilization,
		MemoryUtilization:         status.MemoryUtilization,
		MemoryUsedMb:              status.MemoryUsedMB,
		MemoryFreeMb:              status.MemoryFreeMB,
		TemperatureGpu:            status.TemperatureGPU,
		TemperatureMemory:         status.TemperatureMemory,
		PowerDrawWatts:            status.PowerDrawWatts,
		CurrentGpuClockMhz:        status.CurrentGPUClockMHz,
		CurrentMemoryClockMhz:     status.CurrentMemoryClockMHz,
		Processes:                 processes,
	}
}

func (s *GPUDiscoveryServer) convertStateToProto(state common.GPUState) pb.GPUStatus_State {
	switch state {
	case common.StateIdle:
		return pb.GPUStatus_IDLE
	case common.StateBusy:
		return pb.GPUStatus_BUSY
	case common.StateOffline:
		return pb.GPUStatus_OFFLINE
	case common.StateError:
		return pb.GPUStatus_ERROR
	default:
		return pb.GPUStatus_UNKNOWN
	}
}

func (s *GPUDiscoveryServer) convertCapabilitiesToProto(caps common.GPUCapabilities) *pb.GPUCapabilities {
	return &pb.GPUCapabilities{
		SupportsCuda:           caps.SupportsCUDA,
		SupportsOpencl:         caps.SupportsOpenCL,
		SupportsVulkan:         caps.SupportsVulkan,
		SupportsDirectx:        caps.SupportsDirectX,
		SupportsTensorOps:      caps.SupportsTensorOps,
		SupportsMixedPrecision: caps.SupportsMixedPrecision,
		SupportsRayTracing:     caps.SupportsRayTracing,
		SupportsEcc:            caps.SupportsECC,
		EccEnabled:             caps.ECCEnabled,
		SupportsUnifiedMemory:  caps.SupportsUnifiedMemory,
		SupportsMig:            caps.SupportsMIG,
		SupportsSriov:          caps.SupportsSRIOV,
		PrecisionTypes:         caps.PrecisionTypes,
		MaxThreadsPerBlock:     caps.MaxThreadsPerBlock,
		MaxBlocksPerGrid:       caps.MaxBlocksPerGrid,
	}
}

func (s *GPUDiscoveryServer) convertDriverInfoToProto(driver common.DriverInfo) *pb.DriverInfo {
	return &pb.DriverInfo{
		Version:          driver.Version,
		CudaVersion:      driver.CUDAVersion,
		RocmVersion:      driver.ROCmVersion,
		LevelZeroVersion: driver.LevelZeroVersion,
		InstallDate:      timestamppb.New(driver.InstallDate),
		IsCompatible:     driver.IsCompatible,
		SupportedApis:    driver.SupportedAPIs,
	}
}

func (s *GPUDiscoveryServer) convertBenchmarkResultToProto(result common.BenchmarkResult) *pb.BenchmarkResult {
	return &pb.BenchmarkResult{
		BenchmarkType:   result.BenchmarkType,
		TestName:        result.TestName,
		Score:           result.Score,
		Unit:            result.Unit,
		DurationSeconds: result.DurationSeconds,
		Metadata:        result.Metadata,
		Timestamp:       timestamppb.New(result.Timestamp),
	}
}

func (s *GPUDiscoveryServer) convertChangeTypeToProto(changeType common.ChangeType) pb.GPUChangeEvent_ChangeType {
	switch changeType {
	case common.ChangeAdded:
		return pb.GPUChangeEvent_ADDED
	case common.ChangeRemoved:
		return pb.GPUChangeEvent_REMOVED
	case common.ChangeModified:
		return pb.GPUChangeEvent_MODIFIED
	case common.ChangePerformanceUpdate:
		return pb.GPUChangeEvent_PERFORMANCE_UPDATE
	default:
		return pb.GPUChangeEvent_UNKNOWN
	}
}

func (s *GPUDiscoveryServer) generateSystemID() string {
	// Generate a system identifier - could be based on hostname, MAC address, etc.
	return fmt.Sprintf("gpu-discovery-%d", time.Now().Unix())
}