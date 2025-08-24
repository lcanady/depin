package tracker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/go-redis/redis/v8"

	"github.com/lcanady/depin/monitoring/heartbeat/pkg/types"
	"github.com/lcanady/depin/database/inventory/repositories"
	common "github.com/lcanady/depin/models/resources/common"
)

// AvailabilityTracker tracks resource availability and synchronizes inventory
type AvailabilityTracker struct {
	logger             *logrus.Logger
	repositoryManager  repositories.RepositoryManager
	redisClient        *redis.Client
	config             *TrackerConfig
	
	// Resource cache
	resourceCache      sync.Map // map[resourceID]*CachedResource
	providerResources  sync.Map // map[providerID][]string (resource IDs)
	
	// Event streaming
	eventStreams       map[string]chan *types.AvailabilityEvent
	eventStreamsMutex  sync.RWMutex
	
	// Background processes
	syncTicker         *time.Ticker
	cleanupTicker      *time.Ticker
	stopChan           chan struct{}
	wg                 sync.WaitGroup
}

// TrackerConfig contains configuration for availability tracking
type TrackerConfig struct {
	SyncInterval          time.Duration
	CacheTTL              time.Duration
	MaxCachedResources    int
	CleanupInterval       time.Duration
	EventBufferSize       int
	MaxEventStreams       int
	StalenessThreshold    time.Duration
	RedisKeyPrefix        string
}

// CachedResource represents a cached resource with metadata
type CachedResource struct {
	Resource         *types.ResourceAvailability `json:"resource"`
	LastUpdated      time.Time                   `json:"last_updated"`
	LastSeen         time.Time                   `json:"last_seen"`
	ChangeGeneration int64                       `json:"change_generation"`
	mutex            sync.RWMutex
}

// InventorySnapshot represents a point-in-time inventory snapshot
type InventorySnapshot struct {
	Timestamp          time.Time                            `json:"timestamp"`
	TotalResources     int                                  `json:"total_resources"`
	AvailableResources int                                  `json:"available_resources"`
	AllocatedResources int                                  `json:"allocated_resources"`
	ErrorResources     int                                  `json:"error_resources"`
	ResourcesByType    map[string]int                       `json:"resources_by_type"`
	ResourcesByProvider map[string]int                      `json:"resources_by_provider"`
	ResourcesByRegion  map[string]int                       `json:"resources_by_region"`
	Resources          []*types.ResourceAvailability        `json:"resources"`
}

// NewAvailabilityTracker creates a new availability tracker
func NewAvailabilityTracker(
	logger *logrus.Logger,
	repositoryManager repositories.RepositoryManager,
	redisClient *redis.Client,
	config *TrackerConfig,
) *AvailabilityTracker {
	
	if config == nil {
		config = &TrackerConfig{
			SyncInterval:       30 * time.Second,
			CacheTTL:           5 * time.Minute,
			MaxCachedResources: 10000,
			CleanupInterval:    10 * time.Minute,
			EventBufferSize:    1000,
			MaxEventStreams:    50,
			StalenessThreshold: 10 * time.Minute,
			RedisKeyPrefix:     "depin:availability:",
		}
	}
	
	return &AvailabilityTracker{
		logger:            logger,
		repositoryManager: repositoryManager,
		redisClient:       redisClient,
		config:            config,
		eventStreams:      make(map[string]chan *types.AvailabilityEvent),
		stopChan:          make(chan struct{}),
	}
}

// Start begins the availability tracking service
func (at *AvailabilityTracker) Start(ctx context.Context) error {
	at.logger.Info("Starting availability tracker")
	
	// Load initial resource state
	if err := at.loadInitialState(ctx); err != nil {
		return fmt.Errorf("failed to load initial state: %w", err)
	}
	
	// Start background processes
	at.syncTicker = time.NewTicker(at.config.SyncInterval)
	at.cleanupTicker = time.NewTicker(at.config.CleanupInterval)
	
	at.wg.Add(2)
	go at.syncWorker()
	go at.cleanupWorker()
	
	at.logger.Info("Availability tracker started successfully")
	return nil
}

// Stop stops the availability tracking service
func (at *AvailabilityTracker) Stop() error {
	at.logger.Info("Stopping availability tracker")
	
	close(at.stopChan)
	
	if at.syncTicker != nil {
		at.syncTicker.Stop()
	}
	if at.cleanupTicker != nil {
		at.cleanupTicker.Stop()
	}
	
	at.wg.Wait()
	
	// Close all event streams
	at.eventStreamsMutex.Lock()
	for streamID, eventChan := range at.eventStreams {
		close(eventChan)
		delete(at.eventStreams, streamID)
	}
	at.eventStreamsMutex.Unlock()
	
	at.logger.Info("Availability tracker stopped")
	return nil
}

// UpdateResourceAvailability updates the availability of a resource
func (at *AvailabilityTracker) UpdateResourceAvailability(ctx context.Context, update *types.ResourceAvailabilityUpdate) error {
	resourceID := update.ResourceID
	
	at.logger.WithFields(logrus.Fields{
		"resource_id":   resourceID,
		"resource_type": update.ResourceType,
		"old_state":     update.OldState,
		"new_state":     update.NewState,
	}).Debug("Updating resource availability")
	
	// Get cached resource or create new one
	cachedResource := at.getCachedResource(resourceID)
	
	cachedResource.mutex.Lock()
	defer cachedResource.mutex.Unlock()
	
	oldState := cachedResource.Resource.State
	oldAvailability := *cachedResource.Resource
	
	// Update resource state and metadata
	cachedResource.Resource.State = update.NewState
	cachedResource.Resource.HealthScore = update.HealthScore
	cachedResource.Resource.LastUpdated = time.Now()
	cachedResource.LastSeen = time.Now()
	cachedResource.ChangeGeneration++
	
	// Update current metrics if provided
	if update.CurrentMetrics != nil {
		cachedResource.Resource.CurrentMetrics = update.CurrentMetrics
	}
	
	// Store in cache
	at.resourceCache.Store(resourceID, cachedResource)
	
	// Update Redis cache
	if err := at.updateRedisCache(ctx, resourceID, cachedResource); err != nil {
		at.logger.WithError(err).Error("Failed to update Redis cache")
	}
	
	// Update database
	if err := at.updateDatabaseRecord(ctx, cachedResource.Resource); err != nil {
		at.logger.WithError(err).Error("Failed to update database record")
	}
	
	// Generate and broadcast event if state changed
	if oldState != update.NewState {
		event := &types.AvailabilityEvent{
			EventType:    at.determineEventType(oldState, update.NewState),
			ResourceID:   resourceID,
			ResourceType: update.ResourceType,
			ProviderID:   update.ProviderID,
			ProviderName: update.ProviderName,
			OldState:     oldState,
			NewState:     update.NewState,
			Region:       update.Region,
			Message:      fmt.Sprintf("Resource state changed from %s to %s", oldState, update.NewState),
			Timestamp:    time.Now(),
		}
		
		at.broadcastAvailabilityEvent(event)
	}
	
	return nil
}

// GetResourceAvailability gets current resource availability
func (at *AvailabilityTracker) GetResourceAvailability(ctx context.Context, filter *types.AvailabilityFilter) ([]*types.ResourceAvailability, error) {
	var resources []*types.ResourceAvailability
	
	// Check cache first
	at.resourceCache.Range(func(key, value interface{}) bool {
		cachedResource := value.(*CachedResource)
		cachedResource.mutex.RLock()
		resource := cachedResource.Resource
		cachedResource.mutex.RUnlock()
		
		if at.matchesFilter(resource, filter) {
			// Create a copy to avoid concurrent access issues
			resourceCopy := *resource
			resources = append(resources, &resourceCopy)
		}
		
		return true
	})
	
	// If cache is empty or stale, refresh from database
	if len(resources) == 0 || at.isCacheStale() {
		dbResources, err := at.refreshFromDatabase(ctx, filter)
		if err != nil {
			return nil, fmt.Errorf("failed to refresh from database: %w", err)
		}
		resources = dbResources
	}
	
	// Sort by last updated (most recent first)
	at.sortResourcesByLastUpdated(resources)
	
	return resources, nil
}

// GetInventorySnapshot creates a point-in-time inventory snapshot
func (at *AvailabilityTracker) GetInventorySnapshot(ctx context.Context) (*InventorySnapshot, error) {
	snapshot := &InventorySnapshot{
		Timestamp:           time.Now(),
		ResourcesByType:     make(map[string]int),
		ResourcesByProvider: make(map[string]int),
		ResourcesByRegion:   make(map[string]int),
	}
	
	// Get all resources
	resources, err := at.GetResourceAvailability(ctx, &types.AvailabilityFilter{})
	if err != nil {
		return nil, fmt.Errorf("failed to get resource availability: %w", err)
	}
	
	// Calculate statistics
	snapshot.TotalResources = len(resources)
	snapshot.Resources = resources
	
	for _, resource := range resources {
		// Count by state
		switch resource.State {
		case types.ResourceAvailable:
			snapshot.AvailableResources++
		case types.ResourceAllocated:
			snapshot.AllocatedResources++
		case types.ResourceError:
			snapshot.ErrorResources++
		}
		
		// Count by type
		snapshot.ResourcesByType[resource.ResourceType]++
		
		// Count by provider
		snapshot.ResourcesByProvider[resource.ProviderID.String()]++
		
		// Count by region (would need to be extracted from resource metadata)
		region := "default" // Placeholder - would extract from resource or provider
		snapshot.ResourcesByRegion[region]++
	}
	
	return snapshot, nil
}

// AddAvailabilityEventStream adds a new event stream for availability events
func (at *AvailabilityTracker) AddAvailabilityEventStream(streamID string, eventChan chan *types.AvailabilityEvent) error {
	at.eventStreamsMutex.Lock()
	defer at.eventStreamsMutex.Unlock()
	
	if len(at.eventStreams) >= at.config.MaxEventStreams {
		return fmt.Errorf("maximum event streams reached")
	}
	
	at.eventStreams[streamID] = eventChan
	return nil
}

// RemoveAvailabilityEventStream removes an event stream
func (at *AvailabilityTracker) RemoveAvailabilityEventStream(streamID string) {
	at.eventStreamsMutex.Lock()
	defer at.eventStreamsMutex.Unlock()
	
	if eventChan, exists := at.eventStreams[streamID]; exists {
		close(eventChan)
		delete(at.eventStreams, streamID)
	}
}

// Private methods

func (at *AvailabilityTracker) loadInitialState(ctx context.Context) error {
	// Load GPU resources
	gpus, err := at.repositoryManager.GPUs().List(ctx, 1000, 0)
	if err != nil {
		return fmt.Errorf("failed to load GPUs: %w", err)
	}
	
	for _, gpu := range gpus {
		resource := &types.ResourceAvailability{
			ResourceID:    gpu.ID.String(),
			ResourceType:  "gpu",
			ProviderID:    gpu.ProviderID,
			Name:          gpu.Name,
			State:         at.mapGPUStateToResourceState(gpu.Status),
			HealthScore:   gpu.UptimePercentage,
			LastUpdated:   gpu.LastDiscovered,
			Capabilities: &types.ResourceCapabilities{
				GPUVendor:       gpu.Vendor,
				GPUArchitecture: gpu.Specs.Architecture,
				MemoryMB:        gpu.Specs.MemoryTotalMB,
				CUDACores:       gpu.Specs.CUDACores,
				TensorCores:     gpu.Specs.TensorCores,
			},
		}
		
		cachedResource := &CachedResource{
			Resource:         resource,
			LastUpdated:      time.Now(),
			LastSeen:         time.Now(),
			ChangeGeneration: 1,
		}
		
		at.resourceCache.Store(gpu.ID.String(), cachedResource)
	}
	
	at.logger.Infof("Loaded %d initial resources", len(gpus))
	return nil
}

func (at *AvailabilityTracker) getCachedResource(resourceID string) *CachedResource {
	if cached, ok := at.resourceCache.Load(resourceID); ok {
		return cached.(*CachedResource)
	}
	
	// Create new cached resource
	cachedResource := &CachedResource{
		Resource: &types.ResourceAvailability{
			ResourceID:  resourceID,
			State:       types.ResourceUnknown,
			LastUpdated: time.Now(),
		},
		LastUpdated:      time.Now(),
		LastSeen:         time.Now(),
		ChangeGeneration: 1,
	}
	
	return cachedResource
}

func (at *AvailabilityTracker) updateRedisCache(ctx context.Context, resourceID string, cachedResource *CachedResource) error {
	key := fmt.Sprintf("%s%s", at.config.RedisKeyPrefix, resourceID)
	
	// Serialize cached resource (simplified - would use proper JSON marshaling)
	value := fmt.Sprintf("%s:%s:%d:%s",
		cachedResource.Resource.State,
		cachedResource.Resource.ResourceType,
		cachedResource.ChangeGeneration,
		cachedResource.LastUpdated.Format(time.RFC3339))
	
	return at.redisClient.SetEX(ctx, key, value, at.config.CacheTTL).Err()
}

func (at *AvailabilityTracker) updateDatabaseRecord(ctx context.Context, resource *types.ResourceAvailability) error {
	// Update based on resource type
	switch resource.ResourceType {
	case "gpu":
		gpuID, err := uuid.Parse(resource.ResourceID)
		if err != nil {
			return fmt.Errorf("invalid GPU ID: %w", err)
		}
		
		// Update GPU status and metrics
		return at.repositoryManager.GPUs().UpdateLastDiscovered(ctx, gpuID, resource.LastUpdated)
		
	default:
		at.logger.Debugf("Database update not implemented for resource type: %s", resource.ResourceType)
		return nil
	}
}

func (at *AvailabilityTracker) determineEventType(oldState, newState types.ResourceState) types.AvailabilityEventType {
	switch {
	case oldState != types.ResourceAvailable && newState == types.ResourceAvailable:
		return types.AvailabilityResourceAvailable
	case oldState == types.ResourceAvailable && newState == types.ResourceAllocated:
		return types.AvailabilityResourceAllocated
	case oldState == types.ResourceAllocated && newState == types.ResourceAvailable:
		return types.AvailabilityResourceReleased
	case newState == types.ResourceError:
		return types.AvailabilityResourceDegraded
	case oldState == types.ResourceError && newState != types.ResourceError:
		return types.AvailabilityResourceRestored
	case newState == types.ResourceOffline:
		return types.AvailabilityResourceOffline
	default:
		return types.AvailabilityResourceAvailable
	}
}

func (at *AvailabilityTracker) broadcastAvailabilityEvent(event *types.AvailabilityEvent) {
	at.eventStreamsMutex.RLock()
	defer at.eventStreamsMutex.RUnlock()
	
	for streamID, eventChan := range at.eventStreams {
		select {
		case eventChan <- event:
		default:
			at.logger.WithField("stream_id", streamID).Warn("Availability event stream buffer full, dropping event")
		}
	}
}

func (at *AvailabilityTracker) matchesFilter(resource *types.ResourceAvailability, filter *types.AvailabilityFilter) bool {
	if filter == nil {
		return true
	}
	
	// Filter by resource type
	if len(filter.ResourceTypes) > 0 {
		found := false
		for _, resourceType := range filter.ResourceTypes {
			if resourceType == resource.ResourceType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Filter by provider
	if filter.ProviderID != uuid.Nil && filter.ProviderID != resource.ProviderID {
		return false
	}
	
	// Filter by state
	if len(filter.States) > 0 {
		found := false
		for _, state := range filter.States {
			if state == resource.State {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Filter by health score
	if filter.MinHealthScore > 0 && resource.HealthScore < filter.MinHealthScore {
		return false
	}
	
	// Filter by region
	if filter.Region != "" {
		// Would check resource or provider region - placeholder logic
		return true
	}
	
	return true
}

func (at *AvailabilityTracker) isCacheStale() bool {
	// Simple staleness check - in practice would be more sophisticated
	return false
}

func (at *AvailabilityTracker) refreshFromDatabase(ctx context.Context, filter *types.AvailabilityFilter) ([]*types.ResourceAvailability, error) {
	// Refresh from database based on filter
	var resources []*types.ResourceAvailability
	
	// This would implement database refresh logic
	// For now, return empty slice
	return resources, nil
}

func (at *AvailabilityTracker) sortResourcesByLastUpdated(resources []*types.ResourceAvailability) {
	// Simple sort implementation - would use proper sorting
	// For now, resources are already in order
}

func (at *AvailabilityTracker) mapGPUStateToResourceState(gpuStatus common.ResourceStatus) types.ResourceState {
	switch gpuStatus {
	case common.ResourceStatusAvailable:
		return types.ResourceAvailable
	case common.ResourceStatusAllocated:
		return types.ResourceAllocated
	case common.ResourceStatusError:
		return types.ResourceError
	case common.ResourceStatusOffline:
		return types.ResourceOffline
	default:
		return types.ResourceUnknown
	}
}

// Background workers

func (at *AvailabilityTracker) syncWorker() {
	defer at.wg.Done()
	
	at.logger.Info("Starting availability sync worker")
	
	for {
		select {
		case <-at.syncTicker.C:
			at.performSync()
		case <-at.stopChan:
			at.logger.Info("Availability sync worker stopping")
			return
		}
	}
}

func (at *AvailabilityTracker) cleanupWorker() {
	defer at.wg.Done()
	
	at.logger.Info("Starting availability cleanup worker")
	
	for {
		select {
		case <-at.cleanupTicker.C:
			at.performCleanup()
		case <-at.stopChan:
			at.logger.Info("Availability cleanup worker stopping")
			return
		}
	}
}

func (at *AvailabilityTracker) performSync() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Sync cache with database
	at.resourceCache.Range(func(key, value interface{}) bool {
		resourceID := key.(string)
		cachedResource := value.(*CachedResource)
		
		// Check if resource is stale
		cachedResource.mutex.RLock()
		timeSinceLastSeen := time.Since(cachedResource.LastSeen)
		cachedResource.mutex.RUnlock()
		
		if timeSinceLastSeen > at.config.StalenessThreshold {
			// Resource is stale, remove from cache
			at.resourceCache.Delete(resourceID)
			
			// Generate offline event
			event := &types.AvailabilityEvent{
				EventType:    types.AvailabilityResourceOffline,
				ResourceID:   resourceID,
				ResourceType: cachedResource.Resource.ResourceType,
				ProviderID:   cachedResource.Resource.ProviderID,
				ProviderName: cachedResource.Resource.ProviderName,
				OldState:     cachedResource.Resource.State,
				NewState:     types.ResourceOffline,
				Message:      "Resource marked as stale and removed from cache",
				Timestamp:    time.Now(),
			}
			
			at.broadcastAvailabilityEvent(event)
			
			at.logger.WithFields(logrus.Fields{
				"resource_id":   resourceID,
				"stale_duration": timeSinceLastSeen,
			}).Debug("Removed stale resource from cache")
		}
		
		return true
	})
	
	at.logger.Debug("Availability sync completed")
}

func (at *AvailabilityTracker) performCleanup() {
	// Clean up old Redis keys
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	pattern := fmt.Sprintf("%s*", at.config.RedisKeyPrefix)
	keys, err := at.redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		at.logger.WithError(err).Error("Failed to get Redis keys for cleanup")
		return
	}
	
	cleaned := 0
	for _, key := range keys {
		// Check if key is expired
		ttl, err := at.redisClient.TTL(ctx, key).Result()
		if err != nil {
			continue
		}
		
		if ttl <= 0 {
			if err := at.redisClient.Del(ctx, key).Err(); err != nil {
				at.logger.WithError(err).WithField("key", key).Error("Failed to delete expired Redis key")
			} else {
				cleaned++
			}
		}
	}
	
	if cleaned > 0 {
		at.logger.WithField("cleaned_keys", cleaned).Debug("Redis cleanup completed")
	}
}

// Additional type definitions for availability tracking

// ResourceAvailabilityUpdate represents an update to resource availability
type ResourceAvailabilityUpdate struct {
	ResourceID     string              `json:"resource_id"`
	ResourceType   string              `json:"resource_type"`
	ProviderID     uuid.UUID           `json:"provider_id"`
	ProviderName   string              `json:"provider_name"`
	OldState       types.ResourceState `json:"old_state"`
	NewState       types.ResourceState `json:"new_state"`
	HealthScore    float64             `json:"health_score"`
	CurrentMetrics interface{}         `json:"current_metrics,omitempty"`
	Region         string              `json:"region"`
}

// AvailabilityFilter defines filters for resource availability queries
type AvailabilityFilter struct {
	ResourceTypes    []string            `json:"resource_types,omitempty"`
	ProviderID       uuid.UUID           `json:"provider_id,omitempty"`
	States           []types.ResourceState `json:"states,omitempty"`
	MinHealthScore   float64             `json:"min_health_score,omitempty"`
	Region           string              `json:"region,omitempty"`
	IncludeAllocated bool                `json:"include_allocated"`
	Limit            int                 `json:"limit,omitempty"`
	Offset           int                 `json:"offset,omitempty"`
}