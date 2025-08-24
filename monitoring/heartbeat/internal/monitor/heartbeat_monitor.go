package monitor

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/lcanady/depin/monitoring/heartbeat/proto"
	"github.com/lcanady/depin/monitoring/heartbeat/pkg/types"
	"github.com/lcanady/depin/database/inventory/repositories"
	provider_models "github.com/lcanady/depin/models/resources/provider"
	common "github.com/lcanady/depin/models/resources/common"
)

// HeartbeatMonitor manages provider heartbeats and health monitoring
type HeartbeatMonitor struct {
	logger             *logrus.Logger
	repositoryManager  repositories.RepositoryManager
	config             *MonitorConfig
	
	// Active providers tracking
	activeProviders    sync.Map // map[string]*ProviderHealth
	healthChecks       sync.Map // map[string]*types.HealthCheck
	
	// Event streaming
	eventStreams       map[string]chan *types.HeartbeatEvent
	eventStreamsMutex  sync.RWMutex
	
	// Background processes
	healthCheckTicker  *time.Ticker
	cleanupTicker      *time.Ticker
	stopChan           chan struct{}
	wg                 sync.WaitGroup
}

// MonitorConfig contains configuration for heartbeat monitoring
type MonitorConfig struct {
	DefaultHeartbeatInterval  time.Duration
	HeartbeatTimeout         time.Duration
	MaxMissedHeartbeats      int
	HealthCheckInterval      time.Duration
	CleanupInterval          time.Duration
	EventBufferSize          int
	MaxEventStreams          int
}

// ProviderHealth tracks the health status of a provider
type ProviderHealth struct {
	ProviderID               uuid.UUID                      `json:"provider_id"`
	ProviderName             string                         `json:"provider_name"`
	Status                   provider_models.HealthStatus  `json:"status"`
	LastHeartbeat            time.Time                      `json:"last_heartbeat"`
	NextExpectedHeartbeat    time.Time                      `json:"next_expected_heartbeat"`
	HeartbeatInterval        time.Duration                  `json:"heartbeat_interval"`
	ConsecutiveSuccessful    int                            `json:"consecutive_successful"`
	ConsecutiveFailed        int                            `json:"consecutive_failed"`
	HealthScore              float64                        `json:"health_score"`
	SystemMetrics            *types.SystemMetrics           `json:"system_metrics"`
	ResourceStatuses         map[string]*types.ResourceStatus `json:"resource_statuses"`
	ActiveIncidents          []*types.HealthIncident        `json:"active_incidents"`
	ResponseTimes            []time.Duration                `json:"response_times"` // Last 10 response times
	mutex                    sync.RWMutex
}

// NewHeartbeatMonitor creates a new heartbeat monitor instance
func NewHeartbeatMonitor(
	logger *logrus.Logger,
	repositoryManager repositories.RepositoryManager,
	config *MonitorConfig,
) *HeartbeatMonitor {
	
	if config == nil {
		config = &MonitorConfig{
			DefaultHeartbeatInterval: 30 * time.Second,
			HeartbeatTimeout:        60 * time.Second,
			MaxMissedHeartbeats:     3,
			HealthCheckInterval:     10 * time.Second,
			CleanupInterval:         5 * time.Minute,
			EventBufferSize:         1000,
			MaxEventStreams:         100,
		}
	}
	
	monitor := &HeartbeatMonitor{
		logger:            logger,
		repositoryManager: repositoryManager,
		config:            config,
		eventStreams:      make(map[string]chan *types.HeartbeatEvent),
		stopChan:          make(chan struct{}),
	}
	
	return monitor
}

// Start begins the heartbeat monitoring service
func (hm *HeartbeatMonitor) Start(ctx context.Context) error {
	hm.logger.Info("Starting heartbeat monitor")
	
	// Load existing providers from database
	if err := hm.loadExistingProviders(ctx); err != nil {
		hm.logger.WithError(err).Error("Failed to load existing providers")
		return fmt.Errorf("failed to load existing providers: %w", err)
	}
	
	// Start background processes
	hm.healthCheckTicker = time.NewTicker(hm.config.HealthCheckInterval)
	hm.cleanupTicker = time.NewTicker(hm.config.CleanupInterval)
	
	hm.wg.Add(2)
	go hm.healthCheckWorker()
	go hm.cleanupWorker()
	
	hm.logger.Info("Heartbeat monitor started successfully")
	return nil
}

// Stop stops the heartbeat monitoring service
func (hm *HeartbeatMonitor) Stop() error {
	hm.logger.Info("Stopping heartbeat monitor")
	
	close(hm.stopChan)
	
	if hm.healthCheckTicker != nil {
		hm.healthCheckTicker.Stop()
	}
	if hm.cleanupTicker != nil {
		hm.cleanupTicker.Stop()
	}
	
	hm.wg.Wait()
	
	// Close all event streams
	hm.eventStreamsMutex.Lock()
	for streamID, eventChan := range hm.eventStreams {
		close(eventChan)
		delete(hm.eventStreams, streamID)
	}
	hm.eventStreamsMutex.Unlock()
	
	hm.logger.Info("Heartbeat monitor stopped")
	return nil
}

// ProcessHeartbeat processes an incoming heartbeat from a provider
func (hm *HeartbeatMonitor) ProcessHeartbeat(ctx context.Context, heartbeat *types.Heartbeat) (*types.HeartbeatResponse, error) {
	startTime := time.Now()
	providerID := heartbeat.ProviderID
	
	hm.logger.WithFields(logrus.Fields{
		"provider_id":   providerID,
		"provider_name": heartbeat.ProviderName,
		"status":        heartbeat.Status,
	}).Debug("Processing heartbeat")
	
	// Get or create provider health record
	providerHealth, exists := hm.getOrCreateProviderHealth(providerID, heartbeat.ProviderName)
	
	// Calculate response time
	responseTime := time.Since(startTime)
	
	// Update provider health
	oldStatus := providerHealth.Status
	hm.updateProviderHealth(providerHealth, heartbeat, responseTime)
	
	// Store heartbeat in database
	if err := hm.storeHeartbeat(ctx, heartbeat, responseTime); err != nil {
		hm.logger.WithError(err).Error("Failed to store heartbeat")
	}
	
	// Update provider record in database
	if err := hm.updateProviderInDatabase(ctx, providerHealth); err != nil {
		hm.logger.WithError(err).Error("Failed to update provider in database")
	}
	
	// Check for status changes and incidents
	hm.checkForStatusChanges(providerHealth, oldStatus)
	hm.detectIncidents(providerHealth, heartbeat)
	
	// Broadcast heartbeat event
	event := &types.HeartbeatEvent{
		Type:          types.HeartbeatReceived,
		ProviderID:    providerID,
		ProviderName:  heartbeat.ProviderName,
		NewStatus:     providerHealth.Status,
		Message:       "Heartbeat received",
		Timestamp:     time.Now(),
		Metadata:      make(map[string]interface{}),
	}
	hm.broadcastEvent(event)
	
	// Create response
	response := &types.HeartbeatResponse{
		Accepted:               true,
		Message:                "Heartbeat accepted",
		NextHeartbeatInterval:  int(providerHealth.HeartbeatInterval.Seconds()),
		ServerTimestamp:        time.Now(),
		RequiredChecks:         hm.getRequiredChecks(providerHealth),
		Warnings:               hm.getWarnings(providerHealth),
	}
	
	if !exists {
		response.Message = "Provider registered successfully"
	}
	
	return response, nil
}

// GetProviderHealth returns the health status of a provider
func (hm *HeartbeatMonitor) GetProviderHealth(ctx context.Context, providerID uuid.UUID, includeHistory bool) (*types.ProviderHealthStatus, error) {
	// Get provider health from memory
	providerHealth, exists := hm.getProviderHealth(providerID)
	if !exists {
		return nil, status.Errorf(codes.NotFound, "Provider not found or not active")
	}
	
	providerHealth.mutex.RLock()
	defer providerHealth.mutex.RUnlock()
	
	// Create health summary
	healthSummary := &types.HealthSummary{
		UptimePercentage:                hm.calculateUptimePercentage(providerID),
		AvgResponseTimeMs:               hm.calculateAverageResponseTime(providerHealth),
		TotalResources:                  len(providerHealth.ResourceStatuses),
		HealthyResources:                hm.countResourcesByState(providerHealth, types.ResourceAvailable),
		DegradedResources:               hm.countResourcesByState(providerHealth, types.ResourceBusy),
		FailedResources:                 hm.countResourcesByState(providerHealth, types.ResourceError),
		ConsecutiveSuccessfulHeartbeats: providerHealth.ConsecutiveSuccessful,
		ConsecutiveFailedHeartbeats:     providerHealth.ConsecutiveFailed,
	}
	
	// Get recent incidents
	var recentIncidents []*types.HealthIncident
	if includeHistory {
		incidents, err := hm.repositoryManager.Providers().ListIncidents(
			ctx, providerID, "", 10) // Get last 10 incidents
		if err != nil {
			hm.logger.WithError(err).Error("Failed to get recent incidents")
		} else {
			recentIncidents = hm.convertToHealthIncidents(incidents)
		}
	}
	
	healthStatus := &types.ProviderHealthStatus{
		ProviderID:           providerID,
		ProviderName:         providerHealth.ProviderName,
		Status:               providerHealth.Status,
		HealthScore:          providerHealth.HealthScore,
		LatestMetrics:        providerHealth.SystemMetrics,
		Resources:            hm.convertResourceStatuses(providerHealth.ResourceStatuses),
		HealthSummary:        healthSummary,
		RecentIncidents:      recentIncidents,
		LastHeartbeat:        providerHealth.LastHeartbeat,
		NextExpectedHeartbeat: providerHealth.NextExpectedHeartbeat,
	}
	
	return healthStatus, nil
}

// GetSystemHealth returns overall system health
func (hm *HeartbeatMonitor) GetSystemHealth(ctx context.Context) (*types.SystemHealthOverview, error) {
	var totalProviders, healthyProviders, degradedProviders, unhealthyProviders, offlineProviders int
	var totalResources, availableResources, allocatedResources, errorResources int
	var totalResponseTime float64
	var uptimeSum float64
	
	// Aggregate statistics from all providers
	hm.activeProviders.Range(func(key, value interface{}) bool {
		providerHealth := value.(*ProviderHealth)
		providerHealth.mutex.RLock()
		defer providerHealth.mutex.RUnlock()
		
		totalProviders++
		
		switch providerHealth.Status {
		case provider_models.HealthStatusHealthy:
			healthyProviders++
		case provider_models.HealthStatusDegraded:
			degradedProviders++
		case provider_models.HealthStatusUnhealthy:
			unhealthyProviders++
		case provider_models.HealthStatusUnreachable:
			offlineProviders++
		}
		
		// Count resources
		totalResources += len(providerHealth.ResourceStatuses)
		for _, resource := range providerHealth.ResourceStatuses {
			switch resource.State {
			case types.ResourceAvailable:
				availableResources++
			case types.ResourceAllocated:
				allocatedResources++
			case types.ResourceError:
				errorResources++
			}
		}
		
		// Calculate response time
		totalResponseTime += hm.calculateAverageResponseTime(providerHealth)
		uptimeSum += hm.calculateUptimePercentage(providerHealth.ProviderID)
		
		return true
	})
	
	avgResponseTime := 0.0
	avgUptime := 0.0
	if totalProviders > 0 {
		avgResponseTime = totalResponseTime / float64(totalProviders)
		avgUptime = uptimeSum / float64(totalProviders)
	}
	
	// Determine overall system status
	systemStatus := types.SystemHealthy
	if float64(unhealthyProviders)/float64(totalProviders) > 0.1 {
		systemStatus = types.SystemUnhealthy
	} else if float64(degradedProviders)/float64(totalProviders) > 0.2 {
		systemStatus = types.SystemDegraded
	}
	
	// Calculate overall health score
	healthScore := hm.calculateOverallHealthScore(
		healthyProviders, degradedProviders, unhealthyProviders, totalProviders)
	
	// Get active system alerts
	activeAlerts := hm.getActiveSystemAlerts()
	
	overview := &types.SystemHealthOverview{
		OverallStatus:      systemStatus,
		OverallHealthScore: healthScore,
		Statistics: &types.SystemStatistics{
			TotalProviders:        totalProviders,
			HealthyProviders:      healthyProviders,
			DegradedProviders:     degradedProviders,
			UnhealthyProviders:    unhealthyProviders,
			OfflineProviders:      offlineProviders,
			TotalResources:        totalResources,
			AvailableResources:    availableResources,
			AllocatedResources:    allocatedResources,
			ErrorResources:        errorResources,
			AvgUptimePercentage:   avgUptime,
			AvgResponseTimeMs:     avgResponseTime,
		},
		ActiveAlerts:  activeAlerts,
		GeneratedAt:   time.Now(),
	}
	
	return overview, nil
}

// AddEventStream adds a new event stream for heartbeat events
func (hm *HeartbeatMonitor) AddEventStream(streamID string, eventChan chan *types.HeartbeatEvent) error {
	hm.eventStreamsMutex.Lock()
	defer hm.eventStreamsMutex.Unlock()
	
	if len(hm.eventStreams) >= hm.config.MaxEventStreams {
		return fmt.Errorf("maximum event streams reached")
	}
	
	hm.eventStreams[streamID] = eventChan
	return nil
}

// RemoveEventStream removes an event stream
func (hm *HeartbeatMonitor) RemoveEventStream(streamID string) {
	hm.eventStreamsMutex.Lock()
	defer hm.eventStreamsMutex.Unlock()
	
	if eventChan, exists := hm.eventStreams[streamID]; exists {
		close(eventChan)
		delete(hm.eventStreams, streamID)
	}
}

// Private methods

func (hm *HeartbeatMonitor) loadExistingProviders(ctx context.Context) error {
	// Get all active providers from database
	activeProviders, err := hm.repositoryManager.Providers().ListByStatus(
		ctx, provider_models.ProviderStatusActive)
	if err != nil {
		return fmt.Errorf("failed to get active providers: %w", err)
	}
	
	for _, provider := range activeProviders {
		providerHealth := &ProviderHealth{
			ProviderID:            provider.ID,
			ProviderName:          provider.Name,
			Status:                provider.HealthStatus,
			HeartbeatInterval:     time.Duration(provider.HeartbeatInterval) * time.Second,
			HealthScore:           85.0, // Default health score
			ResourceStatuses:      make(map[string]*types.ResourceStatus),
			ActiveIncidents:       []*types.HealthIncident{},
			ResponseTimes:         []time.Duration{},
		}
		
		if provider.LastHeartbeat != nil {
			providerHealth.LastHeartbeat = *provider.LastHeartbeat
			providerHealth.NextExpectedHeartbeat = provider.LastHeartbeat.Add(providerHealth.HeartbeatInterval)
		}
		
		hm.activeProviders.Store(provider.ID.String(), providerHealth)
	}
	
	hm.logger.Infof("Loaded %d existing providers", len(activeProviders))
	return nil
}

func (hm *HeartbeatMonitor) getOrCreateProviderHealth(providerID uuid.UUID, providerName string) (*ProviderHealth, bool) {
	if existing, ok := hm.activeProviders.Load(providerID.String()); ok {
		return existing.(*ProviderHealth), true
	}
	
	// Create new provider health record
	providerHealth := &ProviderHealth{
		ProviderID:            providerID,
		ProviderName:          providerName,
		Status:                provider_models.HealthStatusHealthy,
		HeartbeatInterval:     hm.config.DefaultHeartbeatInterval,
		HealthScore:           85.0,
		ResourceStatuses:      make(map[string]*types.ResourceStatus),
		ActiveIncidents:       []*types.HealthIncident{},
		ResponseTimes:         []time.Duration{},
	}
	
	hm.activeProviders.Store(providerID.String(), providerHealth)
	return providerHealth, false
}

func (hm *HeartbeatMonitor) getProviderHealth(providerID uuid.UUID) (*ProviderHealth, bool) {
	if existing, ok := hm.activeProviders.Load(providerID.String()); ok {
		return existing.(*ProviderHealth), true
	}
	return nil, false
}

func (hm *HeartbeatMonitor) updateProviderHealth(
	providerHealth *ProviderHealth,
	heartbeat *types.Heartbeat,
	responseTime time.Duration,
) {
	providerHealth.mutex.Lock()
	defer providerHealth.mutex.Unlock()
	
	now := time.Now()
	
	// Update basic information
	providerHealth.LastHeartbeat = now
	providerHealth.NextExpectedHeartbeat = now.Add(providerHealth.HeartbeatInterval)
	providerHealth.SystemMetrics = heartbeat.SystemMetrics
	
	// Update resource statuses
	for _, resourceStatus := range heartbeat.ResourceStatuses {
		providerHealth.ResourceStatuses[resourceStatus.ResourceID] = resourceStatus
	}
	
	// Update response times (keep last 10)
	providerHealth.ResponseTimes = append(providerHealth.ResponseTimes, responseTime)
	if len(providerHealth.ResponseTimes) > 10 {
		providerHealth.ResponseTimes = providerHealth.ResponseTimes[1:]
	}
	
	// Update consecutive counters
	if heartbeat.Status == provider_models.HealthStatusHealthy ||
	   heartbeat.Status == provider_models.HealthStatusDegraded {
		providerHealth.ConsecutiveSuccessful++
		providerHealth.ConsecutiveFailed = 0
	} else {
		providerHealth.ConsecutiveFailed++
		providerHealth.ConsecutiveSuccessful = 0
	}
	
	// Update health status based on heartbeat status and history
	providerHealth.Status = hm.determineHealthStatus(heartbeat.Status, providerHealth)
	
	// Recalculate health score
	providerHealth.HealthScore = hm.calculateHealthScore(providerHealth)
}

func (hm *HeartbeatMonitor) determineHealthStatus(
	reportedStatus provider_models.HealthStatus,
	providerHealth *ProviderHealth,
) provider_models.HealthStatus {
	// If provider reports unhealthy, trust that
	if reportedStatus == provider_models.HealthStatusUnhealthy {
		return provider_models.HealthStatusUnhealthy
	}
	
	// Check for missed heartbeats
	timeSinceLastHeartbeat := time.Since(providerHealth.LastHeartbeat)
	if timeSinceLastHeartbeat > providerHealth.HeartbeatInterval*2 {
		return provider_models.HealthStatusUnreachable
	}
	
	// Check consecutive failed heartbeats
	if providerHealth.ConsecutiveFailed >= hm.config.MaxMissedHeartbeats {
		return provider_models.HealthStatusUnhealthy
	}
	
	// Check resource health
	errorResources := hm.countResourcesByState(providerHealth, types.ResourceError)
	totalResources := len(providerHealth.ResourceStatuses)
	if totalResources > 0 && float64(errorResources)/float64(totalResources) > 0.5 {
		return provider_models.HealthStatusUnhealthy
	}
	
	return reportedStatus
}

func (hm *HeartbeatMonitor) calculateHealthScore(providerHealth *ProviderHealth) float64 {
	score := 100.0
	
	// Penalize for consecutive failures
	if providerHealth.ConsecutiveFailed > 0 {
		score -= float64(providerHealth.ConsecutiveFailed) * 10.0
	}
	
	// Factor in resource health
	if len(providerHealth.ResourceStatuses) > 0 {
		errorResources := hm.countResourcesByState(providerHealth, types.ResourceError)
		errorRatio := float64(errorResources) / float64(len(providerHealth.ResourceStatuses))
		score -= errorRatio * 30.0
	}
	
	// Factor in response time
	avgResponseTime := hm.calculateAverageResponseTime(providerHealth)
	if avgResponseTime > 1000 { // Over 1 second
		score -= 10.0
	} else if avgResponseTime > 500 { // Over 500ms
		score -= 5.0
	}
	
	// Factor in system metrics
	if providerHealth.SystemMetrics != nil {
		if providerHealth.SystemMetrics.CPUUtilization > 90 {
			score -= 10.0
		}
		if providerHealth.SystemMetrics.MemoryUtilization > 90 {
			score -= 10.0
		}
		if providerHealth.SystemMetrics.DiskUtilization > 95 {
			score -= 15.0
		}
	}
	
	// Ensure score is between 0 and 100
	if score < 0 {
		score = 0
	}
	
	return score
}

func (hm *HeartbeatMonitor) countResourcesByState(providerHealth *ProviderHealth, state types.ResourceState) int {
	count := 0
	for _, resource := range providerHealth.ResourceStatuses {
		if resource.State == state {
			count++
		}
	}
	return count
}

func (hm *HeartbeatMonitor) calculateAverageResponseTime(providerHealth *ProviderHealth) float64 {
	if len(providerHealth.ResponseTimes) == 0 {
		return 0
	}
	
	var total time.Duration
	for _, responseTime := range providerHealth.ResponseTimes {
		total += responseTime
	}
	
	avg := total / time.Duration(len(providerHealth.ResponseTimes))
	return float64(avg.Nanoseconds()) / 1e6 // Convert to milliseconds
}

func (hm *HeartbeatMonitor) calculateUptimePercentage(providerID uuid.UUID) float64 {
	// This would normally query the database for historical data
	// For now, return a calculated value based on current health
	if providerHealth, exists := hm.getProviderHealth(providerID); exists {
		providerHealth.mutex.RLock()
		defer providerHealth.mutex.RUnlock()
		
		// Simple uptime calculation based on consecutive successful heartbeats
		total := providerHealth.ConsecutiveSuccessful + providerHealth.ConsecutiveFailed
		if total == 0 {
			return 100.0
		}
		
		return float64(providerHealth.ConsecutiveSuccessful) / float64(total) * 100.0
	}
	
	return 0.0
}

func (hm *HeartbeatMonitor) calculateOverallHealthScore(healthy, degraded, unhealthy, total int) float64 {
	if total == 0 {
		return 100.0
	}
	
	healthyWeight := float64(healthy) * 1.0
	degradedWeight := float64(degraded) * 0.7
	unhealthyWeight := float64(unhealthy) * 0.0
	
	return (healthyWeight + degradedWeight + unhealthyWeight) / float64(total) * 100.0
}

func (hm *HeartbeatMonitor) storeHeartbeat(ctx context.Context, heartbeat *types.Heartbeat, responseTime time.Duration) error {
	// Convert to database model
	dbHeartbeat := &provider_models.ProviderHeartbeat{
		ID:         uuid.New(),
		ProviderID: heartbeat.ProviderID,
		Timestamp:  heartbeat.Timestamp,
		Status:     string(heartbeat.Status),
		ResponseTimeMs: int32(responseTime.Milliseconds()),
		Message:    "Heartbeat processed successfully",
		Version:    heartbeat.Version,
	}
	
	return hm.repositoryManager.Providers().RecordHeartbeat(ctx, dbHeartbeat)
}

func (hm *HeartbeatMonitor) updateProviderInDatabase(ctx context.Context, providerHealth *ProviderHealth) error {
	providerHealth.mutex.RLock()
	defer providerHealth.mutex.RUnlock()
	
	// Update last seen timestamp
	if err := hm.repositoryManager.Providers().UpdateLastSeen(
		ctx, providerHealth.ProviderID, providerHealth.LastHeartbeat); err != nil {
		return fmt.Errorf("failed to update last seen: %w", err)
	}
	
	// Update health status
	if err := hm.repositoryManager.Providers().UpdateHealthStatus(
		ctx, providerHealth.ProviderID, providerHealth.Status); err != nil {
		return fmt.Errorf("failed to update health status: %w", err)
	}
	
	return nil
}

func (hm *HeartbeatMonitor) checkForStatusChanges(providerHealth *ProviderHealth, oldStatus provider_models.HealthStatus) {
	if providerHealth.Status != oldStatus {
		event := &types.HeartbeatEvent{
			Type:         types.StatusChanged,
			ProviderID:   providerHealth.ProviderID,
			ProviderName: providerHealth.ProviderName,
			OldStatus:    oldStatus,
			NewStatus:    providerHealth.Status,
			Message:      fmt.Sprintf("Provider status changed from %s to %s", oldStatus, providerHealth.Status),
			Timestamp:    time.Now(),
			Metadata:     map[string]interface{}{},
		}
		hm.broadcastEvent(event)
	}
}

func (hm *HeartbeatMonitor) detectIncidents(providerHealth *ProviderHealth, heartbeat *types.Heartbeat) {
	// Detect various incident types
	
	// Check for resource failures
	for _, resource := range heartbeat.ResourceStatuses {
		if resource.State == types.ResourceError && len(resource.Issues) > 0 {
			incident := &types.HealthIncident{
				IncidentID:   uuid.New().String(),
				ProviderID:   heartbeat.ProviderID,
				ResourceID:   resource.ResourceID,
				Type:         types.ResourceFailure,
				Severity:     hm.determineSeverity(resource.Issues),
				Title:        fmt.Sprintf("Resource %s failure", resource.Name),
				Description:  fmt.Sprintf("Resource issues: %v", resource.Issues),
				Status:       types.IncidentOpen,
				OccurredAt:   time.Now(),
			}
			
			providerHealth.ActiveIncidents = append(providerHealth.ActiveIncidents, incident)
			
			event := &types.HeartbeatEvent{
				Type:         types.ResourceChanged,
				ProviderID:   heartbeat.ProviderID,
				ProviderName: heartbeat.ProviderName,
				ResourceID:   resource.ResourceID,
				Message:      fmt.Sprintf("Resource failure detected: %s", resource.Name),
				Timestamp:    time.Now(),
				Metadata:     map[string]interface{}{"incident_id": incident.IncidentID},
			}
			hm.broadcastEvent(event)
		}
	}
	
	// Check for performance degradation
	if heartbeat.SystemMetrics != nil {
		if heartbeat.SystemMetrics.CPUUtilization > 95 ||
		   heartbeat.SystemMetrics.MemoryUtilization > 95 ||
		   heartbeat.SystemMetrics.DiskUtilization > 98 {
			
			event := &types.HeartbeatEvent{
				Type:         types.ThresholdExceeded,
				ProviderID:   heartbeat.ProviderID,
				ProviderName: heartbeat.ProviderName,
				Message:      "System resource utilization threshold exceeded",
				Timestamp:    time.Now(),
				Metadata:     map[string]interface{}{
					"cpu_utilization":    heartbeat.SystemMetrics.CPUUtilization,
					"memory_utilization": heartbeat.SystemMetrics.MemoryUtilization,
					"disk_utilization":   heartbeat.SystemMetrics.DiskUtilization,
				},
			}
			hm.broadcastEvent(event)
		}
	}
}

func (hm *HeartbeatMonitor) determineSeverity(issues []string) types.IncidentSeverity {
	// Simple severity determination based on keywords
	for _, issue := range issues {
		if contains(issue, []string{"critical", "fatal", "emergency"}) {
			return types.SeverityCritical
		}
		if contains(issue, []string{"error", "failed", "broken"}) {
			return types.SeverityHigh
		}
		if contains(issue, []string{"warning", "degraded", "slow"}) {
			return types.SeverityMedium
		}
	}
	return types.SeverityLow
}

func (hm *HeartbeatMonitor) broadcastEvent(event *types.HeartbeatEvent) {
	hm.eventStreamsMutex.RLock()
	defer hm.eventStreamsMutex.RUnlock()
	
	for streamID, eventChan := range hm.eventStreams {
		select {
		case eventChan <- event:
		default:
			hm.logger.WithField("stream_id", streamID).Warn("Event stream buffer full, dropping event")
		}
	}
}

func (hm *HeartbeatMonitor) getRequiredChecks(providerHealth *ProviderHealth) []*types.HealthCheck {
	var checks []*types.HealthCheck
	
	// Add standard health checks
	checks = append(checks, &types.HealthCheck{
		CheckID:     "resource_status",
		CheckType:   "resource",
		Description: "Verify all resources are healthy",
		Frequency:   30,
		Parameters:  map[string]string{"include_metrics": "true"},
		Required:    true,
	})
	
	// Add performance check if degraded
	if providerHealth.Status == provider_models.HealthStatusDegraded {
		checks = append(checks, &types.HealthCheck{
			CheckID:     "performance_check",
			CheckType:   "performance",
			Description: "Performance degradation detected, run detailed check",
			Frequency:   15,
			Parameters:  map[string]string{"detailed": "true"},
			Required:    true,
		})
	}
	
	return checks
}

func (hm *HeartbeatMonitor) getWarnings(providerHealth *ProviderHealth) []string {
	var warnings []string
	
	providerHealth.mutex.RLock()
	defer providerHealth.mutex.RUnlock()
	
	// Check response time
	if avgResponseTime := hm.calculateAverageResponseTime(providerHealth); avgResponseTime > 500 {
		warnings = append(warnings, fmt.Sprintf("High average response time: %.1fms", avgResponseTime))
	}
	
	// Check resource health
	if len(providerHealth.ResourceStatuses) > 0 {
		errorResources := hm.countResourcesByState(providerHealth, types.ResourceError)
		if errorResources > 0 {
			warnings = append(warnings, fmt.Sprintf("%d resources in error state", errorResources))
		}
	}
	
	// Check system metrics
	if providerHealth.SystemMetrics != nil {
		if providerHealth.SystemMetrics.CPUUtilization > 80 {
			warnings = append(warnings, fmt.Sprintf("High CPU utilization: %.1f%%", providerHealth.SystemMetrics.CPUUtilization))
		}
		if providerHealth.SystemMetrics.MemoryUtilization > 85 {
			warnings = append(warnings, fmt.Sprintf("High memory utilization: %.1f%%", providerHealth.SystemMetrics.MemoryUtilization))
		}
	}
	
	return warnings
}

// Background workers
func (hm *HeartbeatMonitor) healthCheckWorker() {
	defer hm.wg.Done()
	
	hm.logger.Info("Starting health check worker")
	
	for {
		select {
		case <-hm.healthCheckTicker.C:
			hm.performHealthChecks()
		case <-hm.stopChan:
			hm.logger.Info("Health check worker stopping")
			return
		}
	}
}

func (hm *HeartbeatMonitor) cleanupWorker() {
	defer hm.wg.Done()
	
	hm.logger.Info("Starting cleanup worker")
	
	for {
		select {
		case <-hm.cleanupTicker.C:
			hm.performCleanup()
		case <-hm.stopChan:
			hm.logger.Info("Cleanup worker stopping")
			return
		}
	}
}

func (hm *HeartbeatMonitor) performHealthChecks() {
	now := time.Now()
	
	hm.activeProviders.Range(func(key, value interface{}) bool {
		providerHealth := value.(*ProviderHealth)
		
		// Check for missed heartbeats
		if now.After(providerHealth.NextExpectedHeartbeat) {
			hm.handleMissedHeartbeat(providerHealth)
		}
		
		return true
	})
}

func (hm *HeartbeatMonitor) handleMissedHeartbeat(providerHealth *ProviderHealth) {
	providerHealth.mutex.Lock()
	defer providerHealth.mutex.Unlock()
	
	providerHealth.ConsecutiveFailed++
	
	// Update status if too many missed heartbeats
	oldStatus := providerHealth.Status
	if providerHealth.ConsecutiveFailed >= hm.config.MaxMissedHeartbeats {
		providerHealth.Status = provider_models.HealthStatusUnreachable
	}
	
	// Broadcast event
	event := &types.HeartbeatEvent{
		Type:         types.ConnectionLost,
		ProviderID:   providerHealth.ProviderID,
		ProviderName: providerHealth.ProviderName,
		OldStatus:    oldStatus,
		NewStatus:    providerHealth.Status,
		Message:      fmt.Sprintf("Missed heartbeat (consecutive: %d)", providerHealth.ConsecutiveFailed),
		Timestamp:    time.Now(),
		Metadata:     map[string]interface{}{},
	}
	hm.broadcastEvent(event)
	
	hm.logger.WithFields(logrus.Fields{
		"provider_id":         providerHealth.ProviderID,
		"consecutive_failed":  providerHealth.ConsecutiveFailed,
		"status":             providerHealth.Status,
	}).Warn("Missed heartbeat detected")
}

func (hm *HeartbeatMonitor) performCleanup() {
	// Clean up old heartbeat records
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	oldTime := time.Now().Add(-24 * time.Hour) // Keep last 24 hours
	
	hm.activeProviders.Range(func(key, value interface{}) bool {
		providerID := value.(*ProviderHealth).ProviderID
		
		count, err := hm.repositoryManager.Providers().CleanupOldHeartbeats(ctx, oldTime)
		if err != nil {
			hm.logger.WithError(err).WithField("provider_id", providerID).Error("Failed to cleanup old heartbeats")
		} else if count > 0 {
			hm.logger.WithFields(logrus.Fields{
				"provider_id": providerID,
				"count":       count,
			}).Debug("Cleaned up old heartbeats")
		}
		
		return true
	})
}

// Helper functions
func (hm *HeartbeatMonitor) convertToHealthIncidents(incidents []provider_models.ProviderIncident) []*types.HealthIncident {
	var result []*types.HealthIncident
	
	for _, incident := range incidents {
		healthIncident := &types.HealthIncident{
			IncidentID:   incident.ID.String(),
			ProviderID:   incident.ProviderID,
			Type:         hm.convertIncidentType(incident.Type),
			Severity:     hm.convertIncidentSeverity(incident.Severity),
			Title:        incident.Title,
			Description:  incident.Description,
			Status:       hm.convertIncidentStatus(incident.Status),
			OccurredAt:   incident.CreatedAt,
			LastUpdated:  incident.UpdatedAt,
		}
		
		if incident.ResolvedAt != nil {
			healthIncident.ResolvedAt = *incident.ResolvedAt
		}
		
		result = append(result, healthIncident)
	}
	
	return result
}

func (hm *HeartbeatMonitor) convertResourceStatuses(resourceStatuses map[string]*types.ResourceStatus) []*types.ResourceStatus {
	var result []*types.ResourceStatus
	
	for _, status := range resourceStatuses {
		result = append(result, status)
	}
	
	return result
}

func (hm *HeartbeatMonitor) convertIncidentType(incidentType string) types.IncidentType {
	switch incidentType {
	case "outage":
		return types.HeartbeatMissed
	case "performance":
		return types.PerformanceDegradation
	case "security":
		return types.SecurityAlert
	default:
		return types.IncidentUnknown
	}
}

func (hm *HeartbeatMonitor) convertIncidentSeverity(severity string) types.IncidentSeverity {
	switch severity {
	case "critical":
		return types.SeverityCritical
	case "high":
		return types.SeverityHigh
	case "medium":
		return types.SeverityMedium
	case "low":
		return types.SeverityLow
	default:
		return types.SeverityUnknown
	}
}

func (hm *HeartbeatMonitor) convertIncidentStatus(status string) types.IncidentStatus {
	switch status {
	case "open":
		return types.IncidentOpen
	case "investigating":
		return types.IncidentInvestigating
	case "resolved":
		return types.IncidentResolved
	case "closed":
		return types.IncidentClosed
	default:
		return types.IncidentStatusUnknown
	}
}

func (hm *HeartbeatMonitor) getActiveSystemAlerts() []*types.SystemAlert {
	// This would normally query active alerts from database
	// For now, return empty slice
	return []*types.SystemAlert{}
}

// Utility function
func contains(str string, substrs []string) bool {
	for _, substr := range substrs {
		if strings.Contains(strings.ToLower(str), strings.ToLower(substr)) {
			return true
		}
	}
	return false
}