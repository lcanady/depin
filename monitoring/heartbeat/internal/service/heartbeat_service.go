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
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/lcanady/depin/monitoring/heartbeat/proto"
	"github.com/lcanady/depin/monitoring/heartbeat/internal/monitor"
	"github.com/lcanady/depin/monitoring/heartbeat/pkg/types"
	"github.com/lcanady/depin/database/inventory/repositories"
	provider_models "github.com/lcanady/depin/models/resources/provider"
)

// HeartbeatService implements the heartbeat monitoring gRPC service
type HeartbeatService struct {
	pb.UnimplementedHeartbeatServiceServer
	
	logger             *logrus.Logger
	heartbeatMonitor   *monitor.HeartbeatMonitor
	repositoryManager  repositories.RepositoryManager
	
	// Event streaming
	eventStreams       map[string]chan *types.HeartbeatEvent
	availabilityStreams map[string]chan *types.AvailabilityEvent
	streamMutex        sync.RWMutex
	
	// Configuration
	config *HeartbeatServiceConfig
}

// HeartbeatServiceConfig contains configuration for the heartbeat service
type HeartbeatServiceConfig struct {
	MaxEventStreams         int
	EventBufferSize         int
	MaxAvailabilityStreams  int
	AvailabilityBufferSize  int
	DefaultStreamTimeout    time.Duration
}

// NewHeartbeatService creates a new heartbeat service instance
func NewHeartbeatService(
	logger *logrus.Logger,
	heartbeatMonitor *monitor.HeartbeatMonitor,
	repositoryManager repositories.RepositoryManager,
	config *HeartbeatServiceConfig,
) *HeartbeatService {
	
	if config == nil {
		config = &HeartbeatServiceConfig{
			MaxEventStreams:         100,
			EventBufferSize:         1000,
			MaxAvailabilityStreams:  50,
			AvailabilityBufferSize:  500,
			DefaultStreamTimeout:    time.Hour,
		}
	}
	
	return &HeartbeatService{
		logger:              logger,
		heartbeatMonitor:    heartbeatMonitor,
		repositoryManager:   repositoryManager,
		config:              config,
		eventStreams:        make(map[string]chan *types.HeartbeatEvent),
		availabilityStreams: make(map[string]chan *types.AvailabilityEvent),
	}
}

// SendHeartbeat processes a heartbeat from a provider
func (hs *HeartbeatService) SendHeartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	hs.logger.WithFields(logrus.Fields{
		"provider_id":   req.ProviderId,
		"provider_name": req.ProviderName,
		"status":        req.Status.String(),
	}).Debug("Received heartbeat")
	
	// Convert protobuf request to internal types
	heartbeat, err := hs.convertFromProtoHeartbeat(req)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid heartbeat data: %v", err)
	}
	
	// Process heartbeat through monitor
	response, err := hs.heartbeatMonitor.ProcessHeartbeat(ctx, heartbeat)
	if err != nil {
		hs.logger.WithError(err).Error("Failed to process heartbeat")
		return nil, status.Errorf(codes.Internal, "Failed to process heartbeat: %v", err)
	}
	
	// Convert response to protobuf
	pbResponse := hs.convertToProtoHeartbeatResponse(response)
	
	return pbResponse, nil
}

// StreamHeartbeats streams heartbeat events
func (hs *HeartbeatService) StreamHeartbeats(req *pb.HeartbeatStreamRequest, stream pb.HeartbeatService_StreamHeartbeatsServer) error {
	hs.logger.WithFields(logrus.Fields{
		"provider_ids":       req.ProviderIds,
		"status_filter":      req.StatusFilter,
		"include_metrics":    req.IncludeMetrics,
		"include_resources":  req.IncludeResources,
	}).Info("Starting heartbeat stream")
	
	// Create stream ID and event channel
	streamID := uuid.New().String()
	eventChan := make(chan *types.HeartbeatEvent, hs.config.EventBufferSize)
	
	hs.streamMutex.Lock()
	if len(hs.eventStreams) >= hs.config.MaxEventStreams {
		hs.streamMutex.Unlock()
		return status.Errorf(codes.ResourceExhausted, "Maximum number of event streams reached")
	}
	hs.eventStreams[streamID] = eventChan
	hs.streamMutex.Unlock()
	
	// Register with monitor
	if err := hs.heartbeatMonitor.AddEventStream(streamID, eventChan); err != nil {
		hs.removeEventStream(streamID)
		return status.Errorf(codes.Internal, "Failed to register event stream: %v", err)
	}
	
	defer func() {
		hs.removeEventStream(streamID)
		hs.heartbeatMonitor.RemoveEventStream(streamID)
	}()
	
	// Stream events
	for {
		select {
		case event := <-eventChan:
			if hs.shouldSendEvent(event, req) {
				pbEvent := hs.convertToProtoHeartbeatEvent(event)
				if err := stream.Send(pbEvent); err != nil {
					hs.logger.WithError(err).Error("Failed to send heartbeat event")
					return err
				}
			}
			
		case <-stream.Context().Done():
			hs.logger.Info("Heartbeat stream closed by client")
			return nil
			
		case <-time.After(hs.config.DefaultStreamTimeout):
			hs.logger.Info("Heartbeat stream timeout")
			return status.Errorf(codes.DeadlineExceeded, "Stream timeout")
		}
	}
}

// GetProviderHealth returns provider health status
func (hs *HeartbeatService) GetProviderHealth(ctx context.Context, req *pb.ProviderHealthRequest) (*pb.ProviderHealthResponse, error) {
	hs.logger.WithField("provider_id", req.ProviderId).Debug("Getting provider health")
	
	providerID, err := uuid.Parse(req.ProviderId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid provider ID: %v", err)
	}
	
	// Get health status from monitor
	healthStatus, err := hs.heartbeatMonitor.GetProviderHealth(ctx, providerID, req.IncludeHistory)
	if err != nil {
		return nil, err
	}
	
	// Convert to protobuf response
	response := hs.convertToProtoProviderHealth(healthStatus)
	
	return response, nil
}

// GetSystemHealth returns overall system health
func (hs *HeartbeatService) GetSystemHealth(ctx context.Context, req *pb.SystemHealthRequest) (*pb.SystemHealthResponse, error) {
	hs.logger.Debug("Getting system health")
	
	// Get system health from monitor
	systemHealth, err := hs.heartbeatMonitor.GetSystemHealth(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get system health: %v", err)
	}
	
	// Convert to protobuf response
	response := hs.convertToProtoSystemHealth(systemHealth, req)
	
	return response, nil
}

// ConfigureMonitoring configures monitoring parameters for a provider
func (hs *HeartbeatService) ConfigureMonitoring(ctx context.Context, req *pb.MonitoringConfigRequest) (*pb.MonitoringConfigResponse, error) {
	hs.logger.WithField("provider_id", req.ProviderId).Info("Configuring monitoring")
	
	providerID, err := uuid.Parse(req.ProviderId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid provider ID: %v", err)
	}
	
	// Update provider configuration in database
	// This is a simplified implementation - full version would validate and apply all config
	if req.Config != nil {
		if req.Config.HeartbeatIntervalSeconds > 0 {
			err := hs.repositoryManager.Providers().UpdateResponseTime(
				ctx, providerID, req.Config.HeartbeatIntervalSeconds)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Failed to update configuration: %v", err)
			}
		}
	}
	
	response := &pb.MonitoringConfigResponse{
		Success:       true,
		Message:       "Monitoring configuration updated successfully",
		AppliedConfig: req.Config, // Echo back the config
	}
	
	return response, nil
}

// GetResourceAvailability returns current resource availability
func (hs *HeartbeatService) GetResourceAvailability(ctx context.Context, req *pb.AvailabilityRequest) (*pb.AvailabilityResponse, error) {
	hs.logger.WithFields(logrus.Fields{
		"resource_types": req.ResourceTypes,
		"region":         req.Region,
	}).Debug("Getting resource availability")
	
	// Get available resources from database
	var resources []*types.ResourceAvailability
	
	for _, resourceType := range req.ResourceTypes {
		switch resourceType {
		case "gpu":
			gpus, err := hs.getAvailableGPUs(ctx, req)
			if err != nil {
				hs.logger.WithError(err).Error("Failed to get available GPUs")
				continue
			}
			resources = append(resources, gpus...)
			
		case "cpu", "memory", "storage":
			// Implementation for other resource types would go here
			hs.logger.Debugf("Resource type %s not fully implemented", resourceType)
		}
	}
	
	// Create summary
	summary := hs.createAvailabilitySummary(resources, req.ResourceTypes)
	
	response := &pb.AvailabilityResponse{
		Resources:   hs.convertToProtoResourceAvailability(resources),
		Summary:     summary,
		GeneratedAt: timestamppb.Now(),
	}
	
	return response, nil
}

// SubscribeToAvailability subscribes to availability change events
func (hs *HeartbeatService) SubscribeToAvailability(req *pb.AvailabilitySubscriptionRequest, stream pb.HeartbeatService_SubscribeToAvailabilityServer) error {
	hs.logger.WithFields(logrus.Fields{
		"regions":                req.Regions,
		"notify_on_allocation":   req.NotifyOnAllocation,
		"notify_on_release":      req.NotifyOnRelease,
		"notify_on_degradation":  req.NotifyOnDegradation,
	}).Info("Starting availability subscription")
	
	// Create stream ID and event channel
	streamID := uuid.New().String()
	eventChan := make(chan *types.AvailabilityEvent, hs.config.AvailabilityBufferSize)
	
	hs.streamMutex.Lock()
	if len(hs.availabilityStreams) >= hs.config.MaxAvailabilityStreams {
		hs.streamMutex.Unlock()
		return status.Errorf(codes.ResourceExhausted, "Maximum number of availability streams reached")
	}
	hs.availabilityStreams[streamID] = eventChan
	hs.streamMutex.Unlock()
	
	defer hs.removeAvailabilityStream(streamID)
	
	// Stream availability events
	for {
		select {
		case event := <-eventChan:
			if hs.shouldSendAvailabilityEvent(event, req) {
				pbEvent := hs.convertToProtoAvailabilityEvent(event)
				if err := stream.Send(pbEvent); err != nil {
					hs.logger.WithError(err).Error("Failed to send availability event")
					return err
				}
			}
			
		case <-stream.Context().Done():
			hs.logger.Info("Availability stream closed by client")
			return nil
			
		case <-time.After(hs.config.DefaultStreamTimeout):
			hs.logger.Info("Availability stream timeout")
			return status.Errorf(codes.DeadlineExceeded, "Stream timeout")
		}
	}
}

// Conversion methods

func (hs *HeartbeatService) convertFromProtoHeartbeat(req *pb.HeartbeatRequest) (*types.Heartbeat, error) {
	providerID, err := uuid.Parse(req.ProviderId)
	if err != nil {
		return nil, fmt.Errorf("invalid provider ID: %w", err)
	}
	
	heartbeat := &types.Heartbeat{
		ProviderID:   providerID,
		ProviderName: req.ProviderName,
		Status:       hs.convertFromProtoProviderStatus(req.Status),
		Version:      req.Version,
		Metadata:     req.Metadata,
		Timestamp:    req.Timestamp.AsTime(),
	}
	
	// Convert system metrics
	if req.SystemMetrics != nil {
		heartbeat.SystemMetrics = &types.SystemMetrics{
			CPUUtilization:     req.SystemMetrics.CpuUtilization,
			MemoryUtilization:  req.SystemMetrics.MemoryUtilization,
			DiskUtilization:    req.SystemMetrics.DiskUtilization,
			NetworkRxMBPS:      req.SystemMetrics.NetworkRxMbps,
			NetworkTxMBPS:      req.SystemMetrics.NetworkTxMbps,
			LoadAverage:        req.SystemMetrics.LoadAverage,
			UptimeSeconds:      req.SystemMetrics.UptimeSeconds,
			TemperatureCelsius: req.SystemMetrics.TemperatureCelsius,
			PowerConsumption:   req.SystemMetrics.PowerConsumptionWatts,
		}
	}
	
	// Convert resource statuses
	for _, resource := range req.Resources {
		resourceStatus := &types.ResourceStatus{
			ResourceID:   resource.ResourceId,
			ResourceType: resource.ResourceType,
			Name:         resource.Name,
			State:        hs.convertFromProtoResourceState(resource.State),
			Utilization:  resource.Utilization,
			Issues:       resource.Issues,
			LastUpdated:  resource.LastUpdated.AsTime(),
		}
		
		heartbeat.ResourceStatuses = append(heartbeat.ResourceStatuses, resourceStatus)
	}
	
	return heartbeat, nil
}

func (hs *HeartbeatService) convertToProtoHeartbeatResponse(response *types.HeartbeatResponse) *pb.HeartbeatResponse {
	pbResponse := &pb.HeartbeatResponse{
		Accepted:              response.Accepted,
		Message:               response.Message,
		NextHeartbeatInterval: int32(response.NextHeartbeatInterval),
		Warnings:              response.Warnings,
		ServerTimestamp:       timestamppb.New(response.ServerTimestamp),
	}
	
	// Convert required checks
	for _, check := range response.RequiredChecks {
		pbCheck := &pb.HealthCheck{
			CheckId:          check.CheckID,
			CheckType:        check.CheckType,
			Description:      check.Description,
			FrequencySeconds: int32(check.Frequency),
			Parameters:       check.Parameters,
			Required:         check.Required,
		}
		pbResponse.RequiredChecks = append(pbResponse.RequiredChecks, pbCheck)
	}
	
	return pbResponse
}

func (hs *HeartbeatService) convertToProtoHeartbeatEvent(event *types.HeartbeatEvent) *pb.HeartbeatEvent {
	return &pb.HeartbeatEvent{
		EventType:    hs.convertToProtoHeartbeatEventType(event.Type),
		ProviderId:   event.ProviderID.String(),
		ProviderName: event.ProviderName,
		OldStatus:    hs.convertToProtoProviderStatus(event.OldStatus),
		NewStatus:    hs.convertToProtoProviderStatus(event.NewStatus),
		ResourceId:   event.ResourceID,
		Message:      event.Message,
		Metadata:     hs.convertMetadataToProto(event.Metadata),
		Timestamp:    timestamppb.New(event.Timestamp),
	}
}

func (hs *HeartbeatService) convertToProtoProviderHealth(healthStatus *types.ProviderHealthStatus) *pb.ProviderHealthResponse {
	response := &pb.ProviderHealthResponse{
		ProviderId:          healthStatus.ProviderID.String(),
		ProviderName:        healthStatus.ProviderName,
		Status:              hs.convertToProtoProviderStatus(healthStatus.Status),
		HealthScore:         healthStatus.HealthScore,
		LastHeartbeat:       timestamppb.New(healthStatus.LastHeartbeat),
		NextExpectedHeartbeat: timestamppb.New(healthStatus.NextExpectedHeartbeat),
	}
	
	// Convert latest metrics
	if healthStatus.LatestMetrics != nil {
		response.LatestMetrics = &pb.SystemMetrics{
			CpuUtilization:         healthStatus.LatestMetrics.CPUUtilization,
			MemoryUtilization:      healthStatus.LatestMetrics.MemoryUtilization,
			DiskUtilization:        healthStatus.LatestMetrics.DiskUtilization,
			NetworkRxMbps:          healthStatus.LatestMetrics.NetworkRxMBPS,
			NetworkTxMbps:          healthStatus.LatestMetrics.NetworkTxMBPS,
			LoadAverage:            healthStatus.LatestMetrics.LoadAverage,
			UptimeSeconds:          healthStatus.LatestMetrics.UptimeSeconds,
			TemperatureCelsius:     healthStatus.LatestMetrics.TemperatureCelsius,
			PowerConsumptionWatts:  healthStatus.LatestMetrics.PowerConsumption,
		}
	}
	
	// Convert resources
	for _, resource := range healthStatus.Resources {
		pbResource := &pb.ResourceStatus{
			ResourceId:   resource.ResourceID,
			ResourceType: resource.ResourceType,
			Name:         resource.Name,
			State:        hs.convertToProtoResourceState(resource.State),
			Utilization:  resource.Utilization,
			Issues:       resource.Issues,
			LastUpdated:  timestamppb.New(resource.LastUpdated),
		}
		response.Resources = append(response.Resources, pbResource)
	}
	
	// Convert health summary
	if healthStatus.HealthSummary != nil {
		response.HealthSummary = &pb.HealthSummary{
			UptimePercentage:                healthStatus.HealthSummary.UptimePercentage,
			AvgResponseTimeMs:               healthStatus.HealthSummary.AvgResponseTimeMs,
			TotalResources:                  int32(healthStatus.HealthSummary.TotalResources),
			HealthyResources:                int32(healthStatus.HealthSummary.HealthyResources),
			DegradedResources:               int32(healthStatus.HealthSummary.DegradedResources),
			FailedResources:                 int32(healthStatus.HealthSummary.FailedResources),
			ConsecutiveSuccessfulHeartbeats: int32(healthStatus.HealthSummary.ConsecutiveSuccessfulHeartbeats),
			ConsecutiveFailedHeartbeats:     int32(healthStatus.HealthSummary.ConsecutiveFailedHeartbeats),
		}
		
		if !healthStatus.HealthSummary.LastIncident.IsZero() {
			response.HealthSummary.LastIncident = timestamppb.New(healthStatus.HealthSummary.LastIncident)
		}
	}
	
	return response
}

func (hs *HeartbeatService) convertToProtoSystemHealth(systemHealth *types.SystemHealthOverview, req *pb.SystemHealthRequest) *pb.SystemHealthResponse {
	response := &pb.SystemHealthResponse{
		OverallStatus:      hs.convertToProtoSystemStatus(systemHealth.OverallStatus),
		OverallHealthScore: systemHealth.OverallHealthScore,
		GeneratedAt:        timestamppb.New(systemHealth.GeneratedAt),
	}
	
	// Convert statistics
	if systemHealth.Statistics != nil {
		response.Statistics = &pb.SystemStatistics{
			TotalProviders:        int32(systemHealth.Statistics.TotalProviders),
			HealthyProviders:      int32(systemHealth.Statistics.HealthyProviders),
			DegradedProviders:     int32(systemHealth.Statistics.DegradedProviders),
			UnhealthyProviders:    int32(systemHealth.Statistics.UnhealthyProviders),
			OfflineProviders:      int32(systemHealth.Statistics.OfflineProviders),
			TotalResources:        int32(systemHealth.Statistics.TotalResources),
			AvailableResources:    int32(systemHealth.Statistics.AvailableResources),
			AllocatedResources:    int32(systemHealth.Statistics.AllocatedResources),
			ErrorResources:        int32(systemHealth.Statistics.ErrorResources),
			AvgUptimePercentage:   systemHealth.Statistics.AvgUptimePercentage,
			AvgResponseTimeMs:     systemHealth.Statistics.AvgResponseTimeMs,
		}
	}
	
	return response
}

// Helper methods for conversions
func (hs *HeartbeatService) convertFromProtoProviderStatus(status pb.ProviderStatus) provider_models.HealthStatus {
	switch status {
	case pb.ProviderStatus_HEALTHY:
		return provider_models.HealthStatusHealthy
	case pb.ProviderStatus_DEGRADED:
		return provider_models.HealthStatusDegraded
	case pb.ProviderStatus_UNHEALTHY:
		return provider_models.HealthStatusUnhealthy
	case pb.ProviderStatus_OFFLINE:
		return provider_models.HealthStatusUnreachable
	default:
		return provider_models.HealthStatusUnknown
	}
}

func (hs *HeartbeatService) convertToProtoProviderStatus(status provider_models.HealthStatus) pb.ProviderStatus {
	switch status {
	case provider_models.HealthStatusHealthy:
		return pb.ProviderStatus_HEALTHY
	case provider_models.HealthStatusDegraded:
		return pb.ProviderStatus_DEGRADED
	case provider_models.HealthStatusUnhealthy:
		return pb.ProviderStatus_UNHEALTHY
	case provider_models.HealthStatusUnreachable:
		return pb.ProviderStatus_OFFLINE
	default:
		return pb.ProviderStatus_UNKNOWN
	}
}

func (hs *HeartbeatService) convertFromProtoResourceState(state pb.ResourceState) types.ResourceState {
	switch state {
	case pb.ResourceState_AVAILABLE:
		return types.ResourceAvailable
	case pb.ResourceState_ALLOCATED:
		return types.ResourceAllocated
	case pb.ResourceState_BUSY:
		return types.ResourceBusy
	case pb.ResourceState_ERROR:
		return types.ResourceError
	case pb.ResourceState_MAINTENANCE_MODE:
		return types.ResourceMaintenance
	case pb.ResourceState_OFFLINE:
		return types.ResourceOffline
	default:
		return types.ResourceUnknown
	}
}

func (hs *HeartbeatService) convertToProtoResourceState(state types.ResourceState) pb.ResourceState {
	switch state {
	case types.ResourceAvailable:
		return pb.ResourceState_AVAILABLE
	case types.ResourceAllocated:
		return pb.ResourceState_ALLOCATED
	case types.ResourceBusy:
		return pb.ResourceState_BUSY
	case types.ResourceError:
		return pb.ResourceState_ERROR
	case types.ResourceMaintenance:
		return pb.ResourceState_MAINTENANCE_MODE
	case types.ResourceOffline:
		return pb.ResourceState_OFFLINE
	default:
		return pb.ResourceState_RESOURCE_UNKNOWN
	}
}

func (hs *HeartbeatService) convertToProtoHeartbeatEventType(eventType types.HeartbeatEventType) pb.HeartbeatEvent_EventType {
	switch eventType {
	case types.HeartbeatReceived:
		return pb.HeartbeatEvent_HEARTBEAT_RECEIVED
	case types.StatusChanged:
		return pb.HeartbeatEvent_STATUS_CHANGED
	case types.ResourceChanged:
		return pb.HeartbeatEvent_RESOURCE_CHANGED
	case types.ThresholdExceeded:
		return pb.HeartbeatEvent_THRESHOLD_EXCEEDED
	case types.ConnectionLost:
		return pb.HeartbeatEvent_CONNECTION_LOST
	case types.ConnectionRestored:
		return pb.HeartbeatEvent_CONNECTION_RESTORED
	}
	return pb.HeartbeatEvent_HEARTBEAT_RECEIVED
}

func (hs *HeartbeatService) convertToProtoSystemStatus(status types.SystemStatus) pb.SystemStatus {
	switch status {
	case types.SystemHealthy:
		return pb.SystemStatus_SYSTEM_HEALTHY
	case types.SystemDegraded:
		return pb.SystemStatus_SYSTEM_DEGRADED
	case types.SystemUnhealthy:
		return pb.SystemStatus_SYSTEM_UNHEALTHY
	case types.SystemCritical:
		return pb.SystemStatus_SYSTEM_CRITICAL
	default:
		return pb.SystemStatus_SYSTEM_UNKNOWN
	}
}

// Helper methods
func (hs *HeartbeatService) shouldSendEvent(event *types.HeartbeatEvent, req *pb.HeartbeatStreamRequest) bool {
	// Filter by provider IDs if specified
	if len(req.ProviderIds) > 0 {
		found := false
		for _, id := range req.ProviderIds {
			if id == event.ProviderID.String() {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Filter by status if specified
	if len(req.StatusFilter) > 0 {
		found := false
		for _, status := range req.StatusFilter {
			if status == hs.convertToProtoProviderStatus(event.NewStatus) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	return true
}

func (hs *HeartbeatService) shouldSendAvailabilityEvent(event *types.AvailabilityEvent, req *pb.AvailabilitySubscriptionRequest) bool {
	// Filter by regions if specified
	if len(req.Regions) > 0 {
		found := false
		for _, region := range req.Regions {
			if region == event.Region {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Filter by event type based on subscription preferences
	switch event.EventType {
	case types.AvailabilityResourceAllocated:
		return req.NotifyOnAllocation
	case types.AvailabilityResourceReleased:
		return req.NotifyOnRelease
	case types.AvailabilityResourceDegraded:
		return req.NotifyOnDegradation
	default:
		return true
	}
}

func (hs *HeartbeatService) removeEventStream(streamID string) {
	hs.streamMutex.Lock()
	defer hs.streamMutex.Unlock()
	
	if eventChan, exists := hs.eventStreams[streamID]; exists {
		close(eventChan)
		delete(hs.eventStreams, streamID)
	}
}

func (hs *HeartbeatService) removeAvailabilityStream(streamID string) {
	hs.streamMutex.Lock()
	defer hs.streamMutex.Unlock()
	
	if eventChan, exists := hs.availabilityStreams[streamID]; exists {
		close(eventChan)
		delete(hs.availabilityStreams, streamID)
	}
}

func (hs *HeartbeatService) convertMetadataToProto(metadata map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for key, value := range metadata {
		result[key] = fmt.Sprintf("%v", value)
	}
	return result
}

// Additional methods for availability and resource management
func (hs *HeartbeatService) getAvailableGPUs(ctx context.Context, req *pb.AvailabilityRequest) ([]*types.ResourceAvailability, error) {
	// Get available GPUs from database
	gpus, err := hs.repositoryManager.GPUs().ListAvailable(ctx, req.Region)
	if err != nil {
		return nil, fmt.Errorf("failed to get available GPUs: %w", err)
	}
	
	var resources []*types.ResourceAvailability
	for _, gpu := range gpus {
		// Convert to resource availability
		resource := &types.ResourceAvailability{
			ResourceID:    gpu.ID.String(),
			ResourceType:  "gpu",
			ProviderID:    gpu.ProviderID,
			Name:          gpu.Name,
			State:         types.ResourceAvailable,
			HealthScore:   85.0, // Would be calculated from actual health metrics
			LastUpdated:   time.Now(),
			Capabilities: &types.ResourceCapabilities{
				GPUVendor:       gpu.Vendor,
				GPUArchitecture: gpu.Specs.Architecture,
				MemoryMB:        gpu.Specs.MemoryTotalMB,
				CUDACores:       gpu.Specs.CUDACores,
				TensorCores:     gpu.Specs.TensorCores,
				SupportedAPIs:   []string{"CUDA", "OpenCL"}, // Would come from capabilities
			},
		}
		
		resources = append(resources, resource)
	}
	
	return resources, nil
}

func (hs *HeartbeatService) createAvailabilitySummary(resources []*types.ResourceAvailability, resourceTypes []string) *pb.AvailabilitySummary {
	summary := &pb.AvailabilitySummary{
		TotalResources:     int32(len(resources)),
		AvailableResources: int32(len(resources)), // All returned resources are available
		MatchingResources:  int32(len(resources)), // All returned resources match criteria
		ResourcesByType:    make(map[string]int32),
		ResourcesByProvider: make(map[string]int32),
		ResourcesByRegion:  make(map[string]int32),
	}
	
	// Count by type, provider, and region
	for _, resource := range resources {
		summary.ResourcesByType[resource.ResourceType]++
		summary.ResourcesByProvider[resource.ProviderID.String()]++
		
		// Region would come from provider metadata or resource location
		region := "default" // Placeholder
		summary.ResourcesByRegion[region]++
	}
	
	return summary
}

func (hs *HeartbeatService) convertToProtoResourceAvailability(resources []*types.ResourceAvailability) []*pb.ResourceAvailability {
	var pbResources []*pb.ResourceAvailability
	
	for _, resource := range resources {
		pbResource := &pb.ResourceAvailability{
			ResourceId:    resource.ResourceID,
			ResourceType:  resource.ResourceType,
			ProviderId:    resource.ProviderID.String(),
			ProviderName:  resource.ProviderName,
			Name:          resource.Name,
			State:         hs.convertToProtoResourceState(resource.State),
			HealthScore:   resource.HealthScore,
			LastUpdated:   timestamppb.New(resource.LastUpdated),
		}
		
		// Convert capabilities
		if resource.Capabilities != nil {
			pbResource.Capabilities = &pb.ResourceCapabilities{
				GpuVendor:        resource.Capabilities.GPUVendor,
				GpuArchitecture:  resource.Capabilities.GPUArchitecture,
				MemoryMb:         resource.Capabilities.MemoryMB,
				CudaCores:        resource.Capabilities.CUDACores,
				TensorCores:      resource.Capabilities.TensorCores,
				SupportedApis:    resource.Capabilities.SupportedAPIs,
				CpuCores:         resource.Capabilities.CPUCores,
				CpuThreads:       resource.Capabilities.CPUThreads,
				CpuFrequencyGhz:  resource.Capabilities.CPUFrequencyGHz,
				CpuArchitecture:  resource.Capabilities.CPUArchitecture,
				TotalMemoryMb:    resource.Capabilities.TotalMemoryMB,
				MemoryType:       resource.Capabilities.MemoryType,
				StorageCapacityGb: resource.Capabilities.StorageCapacityGB,
				StorageType:      resource.Capabilities.StorageType,
				MaxIops:          resource.Capabilities.MaxIOPS,
			}
		}
		
		pbResources = append(pbResources, pbResource)
	}
	
	return pbResources
}

func (hs *HeartbeatService) convertToProtoAvailabilityEvent(event *types.AvailabilityEvent) *pb.AvailabilityEvent {
	return &pb.AvailabilityEvent{
		EventType:     hs.convertToProtoAvailabilityEventType(event.EventType),
		ResourceId:    event.ResourceID,
		ResourceType:  event.ResourceType,
		ProviderId:    event.ProviderID.String(),
		ProviderName:  event.ProviderName,
		OldState:      hs.convertToProtoResourceState(event.OldState),
		NewState:      hs.convertToProtoResourceState(event.NewState),
		Region:        event.Region,
		Message:       event.Message,
		Timestamp:     timestamppb.New(event.Timestamp),
	}
}

func (hs *HeartbeatService) convertToProtoAvailabilityEventType(eventType types.AvailabilityEventType) pb.AvailabilityEvent_EventType {
	switch eventType {
	case types.AvailabilityResourceAvailable:
		return pb.AvailabilityEvent_RESOURCE_AVAILABLE
	case types.AvailabilityResourceAllocated:
		return pb.AvailabilityEvent_RESOURCE_ALLOCATED
	case types.AvailabilityResourceReleased:
		return pb.AvailabilityEvent_RESOURCE_RELEASED
	case types.AvailabilityResourceDegraded:
		return pb.AvailabilityEvent_RESOURCE_DEGRADED
	case types.AvailabilityResourceRestored:
		return pb.AvailabilityEvent_RESOURCE_RESTORED
	case types.AvailabilityResourceOffline:
		return pb.AvailabilityEvent_RESOURCE_OFFLINE
	}
	return pb.AvailabilityEvent_RESOURCE_AVAILABLE
}

// Define the availability event types
type AvailabilityEventType int

const (
	AvailabilityResourceAvailable AvailabilityEventType = iota
	AvailabilityResourceAllocated
	AvailabilityResourceReleased
	AvailabilityResourceDegraded
	AvailabilityResourceRestored
	AvailabilityResourceOffline
)

// AvailabilityEvent represents an availability change event
type AvailabilityEvent struct {
	EventType    AvailabilityEventType `json:"event_type"`
	ResourceID   string                `json:"resource_id"`
	ResourceType string                `json:"resource_type"`
	ProviderID   uuid.UUID             `json:"provider_id"`
	ProviderName string                `json:"provider_name"`
	OldState     types.ResourceState   `json:"old_state"`
	NewState     types.ResourceState   `json:"new_state"`
	Region       string                `json:"region"`
	Message      string                `json:"message"`
	Timestamp    time.Time             `json:"timestamp"`
}