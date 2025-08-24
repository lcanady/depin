package test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	
	"../config"
	"../repositories"
	"../../models/resources/common"
	"../../models/resources/gpu"
	"../../models/resources/provider"
)

// TestDatabaseIntegration tests the complete database integration
func TestDatabaseIntegration(t *testing.T) {
	// Skip if not in integration test environment
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	
	// Create test configuration
	cfg := getTestConfig()
	
	// Create repository manager
	rm, err := repositories.NewRepositoryManager(cfg)
	require.NoError(t, err)
	defer rm.Close()
	
	// Wait for database to be ready
	ctx := context.Background()
	require.True(t, rm.IsHealthy(ctx), "Database should be healthy")
	
	t.Run("ProviderOperations", func(t *testing.T) {
		testProviderOperations(t, rm)
	})
	
	t.Run("GPUOperations", func(t *testing.T) {
		testGPUOperations(t, rm)
	})
	
	t.Run("HealthCheckOperations", func(t *testing.T) {
		testHealthCheckOperations(t, rm)
	})
	
	t.Run("VerificationOperations", func(t *testing.T) {
		testVerificationOperations(t, rm)
	})
	
	t.Run("UsageMetricsOperations", func(t *testing.T) {
		testUsageMetricsOperations(t, rm)
	})
	
	t.Run("TransactionOperations", func(t *testing.T) {
		testTransactionOperations(t, rm)
	})
}

func testProviderOperations(t *testing.T, rm repositories.RepositoryManager) {
	ctx := context.Background()
	providerRepo := rm.Providers()
	
	// Create test provider
	testProvider := &provider.ProviderResource{
		Name:         "Test Provider",
		Email:        "test@example.com",
		Organization: "Test Org",
		Status:       provider.ProviderStatusActive,
		ApiKeyHash:   "hash123",
		PublicKey:    "publickey123",
		Endpoints: []provider.ProviderEndpoint{
			{
				Type:     "api",
				URL:      "https://test.example.com",
				Port:     443,
				Protocol: "https",
				Secure:   true,
			},
		},
		Metadata: provider.ProviderMetadata{
			Region:     "us-east-1",
			DataCenter: "dc-1",
			Tags:       map[string]string{"env": "test"},
		},
		HealthStatus: provider.HealthStatusHealthy,
	}
	
	// Test Create
	err := providerRepo.Create(ctx, testProvider)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, testProvider.ID)
	
	// Test GetByID
	retrieved, err := providerRepo.GetByID(ctx, testProvider.ID)
	require.NoError(t, err)
	assert.Equal(t, testProvider.Name, retrieved.Name)
	assert.Equal(t, testProvider.Email, retrieved.Email)
	
	// Test GetByEmail
	byEmail, err := providerRepo.GetByEmail(ctx, testProvider.Email)
	require.NoError(t, err)
	assert.Equal(t, testProvider.ID, byEmail.ID)
	
	// Test Update
	testProvider.Name = "Updated Provider"
	err = providerRepo.Update(ctx, testProvider)
	require.NoError(t, err)
	
	updated, err := providerRepo.GetByID(ctx, testProvider.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Provider", updated.Name)
	
	// Test Search
	filter := &provider.ProviderSearchFilter{
		SearchFilter: common.SearchFilter{
			Regions: []string{"us-east-1"},
		},
	}
	pagination := &common.PaginationOptions{Limit: 10, Offset: 0}
	
	results, err := providerRepo.Search(ctx, filter, nil, pagination)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results.Items), 1)
	
	// Test heartbeat
	heartbeat := &provider.ProviderHeartbeat{
		ProviderID: testProvider.ID,
		Status:     "healthy",
		SystemMetrics: provider.SystemMetrics{
			CPUUtilization:    50.0,
			MemoryUtilization: 60.0,
		},
		ResponseTimeMs: 100,
		Message:        "All systems operational",
	}
	
	err = providerRepo.RecordHeartbeat(ctx, heartbeat)
	require.NoError(t, err)
	
	// Verify heartbeat was recorded
	latestHeartbeat, err := providerRepo.GetLatestHeartbeat(ctx, testProvider.ID)
	require.NoError(t, err)
	assert.Equal(t, "healthy", latestHeartbeat.Status)
}

func testGPUOperations(t *testing.T, rm repositories.RepositoryManager) {
	ctx := context.Background()
	gpuRepo := rm.GPUs()
	providerRepo := rm.Providers()
	
	// Create test provider first
	testProvider := &provider.ProviderResource{
		Name:   "GPU Provider",
		Email:  "gpu@example.com",
		Status: provider.ProviderStatusActive,
	}
	err := providerRepo.Create(ctx, testProvider)
	require.NoError(t, err)
	
	// Create test GPU
	testGPU := &gpu.GPUResource{
		BaseResource: common.BaseResource{
			ProviderID: testProvider.ID,
			Type:       common.ResourceTypeGPU,
			Name:       "Test GPU",
			Status:     common.ResourceStatusActive,
			Region:     "us-west-2",
			Tags:       []string{"test", "nvidia"},
		},
		UUID:   "gpu-uuid-123",
		Index:  0,
		Vendor: "NVIDIA",
		Specs: gpu.GPUSpecs{
			MemoryTotalMB:     16384,
			CUDACores:        10496,
			BaseClockMHz:     1410,
			BoostClockMHz:    1770,
			Architecture:     "Ampere",
			ComputeCapability: "8.6",
		},
		Capabilities: gpu.GPUCapabilities{
			SupportsCUDA:      true,
			SupportsOpenCL:    true,
			SupportsTensorOps: true,
			PrecisionTypes:   []string{"fp32", "fp16", "int8"},
		},
		DriverInfo: gpu.GPUDriverInfo{
			Version:      "470.82.01",
			CUDAVersion:  "11.4",
			IsCompatible: true,
		},
		VerificationStatus: "verified",
		IsAllocated:       false,
	}
	
	// Test Create
	err = gpuRepo.Create(ctx, testGPU)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, testGPU.ID)
	
	// Test GetByID
	retrieved, err := gpuRepo.GetByID(ctx, testGPU.ID)
	require.NoError(t, err)
	assert.Equal(t, testGPU.Name, retrieved.Name)
	assert.Equal(t, testGPU.Vendor, retrieved.Vendor)
	
	// Test GetByUUID
	byUUID, err := gpuRepo.GetByUUID(ctx, testGPU.UUID)
	require.NoError(t, err)
	assert.Equal(t, testGPU.ID, byUUID.ID)
	
	// Test GetByProviderID
	providerGPUs, err := gpuRepo.GetByProviderID(ctx, testProvider.ID)
	require.NoError(t, err)
	assert.Len(t, providerGPUs, 1)
	
	// Test Search
	filter := &gpu.GPUSearchFilter{
		SearchFilter: common.SearchFilter{
			ProviderIDs: []uuid.UUID{testProvider.ID},
		},
		GPUVendors: []string{"NVIDIA"},
		SupportsCUDA: &[]bool{true}[0],
	}
	pagination := &common.PaginationOptions{Limit: 10, Offset: 0}
	
	results, err := gpuRepo.Search(ctx, filter, nil, pagination)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results.Items), 1)
	
	// Test allocation
	allocationID := uuid.New()
	err = gpuRepo.MarkAsAllocated(ctx, testGPU.ID, allocationID, time.Now())
	require.NoError(t, err)
	
	allocated, err := gpuRepo.GetByID(ctx, testGPU.ID)
	require.NoError(t, err)
	assert.True(t, allocated.IsAllocated)
	assert.Equal(t, allocationID, *allocated.CurrentAllocation)
	
	// Test release
	err = gpuRepo.MarkAsReleased(ctx, testGPU.ID, time.Now())
	require.NoError(t, err)
	
	released, err := gpuRepo.GetByID(ctx, testGPU.ID)
	require.NoError(t, err)
	assert.False(t, released.IsAllocated)
	assert.Nil(t, released.CurrentAllocation)
}

func testHealthCheckOperations(t *testing.T, rm repositories.RepositoryManager) {
	ctx := context.Background()
	healthRepo := rm.HealthChecks()
	
	resourceID := uuid.New()
	
	// Create test health check
	healthCheck := &common.HealthCheck{
		ResourceID:     resourceID,
		CheckType:      "connectivity",
		Status:         "healthy",
		Message:        "Connection successful",
		ResponseTime:   50,
		Metadata:       common.JSONData{"endpoint": "https://test.com"},
	}
	
	// Test Create
	err := healthRepo.CreateHealthCheck(ctx, healthCheck)
	require.NoError(t, err)
	
	// Test GetLatestHealthCheck
	latest, err := healthRepo.GetLatestHealthCheck(ctx, resourceID, "connectivity")
	require.NoError(t, err)
	assert.Equal(t, "healthy", latest.Status)
	
	// Test GetHealthCheckHistory
	history, err := healthRepo.GetHealthCheckHistory(ctx, resourceID, time.Now().Add(-time.Hour), 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(history), 1)
}

func testVerificationOperations(t *testing.T, rm repositories.RepositoryManager) {
	ctx := context.Background()
	verifyRepo := rm.Verifications()
	
	resourceID := uuid.New()
	
	// Create test verification
	verification := &common.Verification{
		ResourceID:  resourceID,
		Type:        "performance",
		Status:      "valid",
		Score:       85.5,
		Details:     common.JSONData{"benchmark": "cuda", "score": 1000},
		ExpiresAt:   time.Now().Add(30 * 24 * time.Hour), // 30 days
		VerifierID:  "verifier-1",
	}
	
	// Test Create
	err := verifyRepo.CreateVerification(ctx, verification)
	require.NoError(t, err)
	
	// Test GetLatestVerification
	latest, err := verifyRepo.GetLatestVerification(ctx, resourceID, "performance")
	require.NoError(t, err)
	assert.Equal(t, "valid", latest.Status)
	assert.Equal(t, 85.5, latest.Score)
	
	// Test GetVerificationHistory
	history, err := verifyRepo.GetVerificationHistory(ctx, resourceID, 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(history), 1)
	
	// Test GetVerificationStats
	stats, err := verifyRepo.GetVerificationStats(ctx, 24*time.Hour)
	require.NoError(t, err)
	assert.Contains(t, stats, "total_verifications")
}

func testUsageMetricsOperations(t *testing.T, rm repositories.RepositoryManager) {
	ctx := context.Background()
	metricsRepo := rm.UsageMetrics()
	
	resourceID := uuid.New()
	
	// Create test usage metrics
	metrics := []*common.Usage{
		{
			ResourceID:     resourceID,
			MetricType:     "gpu_utilization",
			Value:          75.5,
			Unit:           "percent",
			Timestamp:      time.Now().Add(-time.Hour),
		},
		{
			ResourceID:     resourceID,
			MetricType:     "gpu_utilization",
			Value:          80.2,
			Unit:           "percent",
			Timestamp:      time.Now().Add(-30*time.Minute),
		},
	}
	
	// Test RecordUsageBatch
	err := metricsRepo.RecordUsageBatch(ctx, metrics)
	require.NoError(t, err)
	
	// Test GetLatestUsage
	latest, err := metricsRepo.GetLatestUsage(ctx, resourceID, "gpu_utilization")
	require.NoError(t, err)
	assert.Equal(t, 80.2, latest.Value)
	
	// Test GetUsageHistory
	history, err := metricsRepo.GetUsageHistory(ctx, resourceID, "gpu_utilization", time.Now().Add(-2*time.Hour), 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(history), 2)
	
	// Test GetAverageUsage
	avgUsage, err := metricsRepo.GetAverageUsage(ctx, resourceID, "gpu_utilization", 2*time.Hour)
	require.NoError(t, err)
	assert.Greater(t, avgUsage, 0.0)
	
	// Test GetPeakUsage
	peakUsage, err := metricsRepo.GetPeakUsage(ctx, resourceID, "gpu_utilization", 2*time.Hour)
	require.NoError(t, err)
	assert.Equal(t, 80.2, peakUsage)
}

func testTransactionOperations(t *testing.T, rm repositories.RepositoryManager) {
	ctx := context.Background()
	
	// Test successful transaction
	err := rm.WithTransaction(ctx, func(txRM repositories.RepositoryManager) error {
		// Create provider in transaction
		testProvider := &provider.ProviderResource{
			Name:   "TX Provider",
			Email:  "tx@example.com",
			Status: provider.ProviderStatusActive,
		}
		
		err := txRM.Providers().Create(ctx, testProvider)
		if err != nil {
			return err
		}
		
		// Create GPU in same transaction
		testGPU := &gpu.GPUResource{
			BaseResource: common.BaseResource{
				ProviderID: testProvider.ID,
				Type:       common.ResourceTypeGPU,
				Name:       "TX GPU",
				Status:     common.ResourceStatusActive,
			},
			Vendor: "AMD",
		}
		
		return txRM.GPUs().Create(ctx, testGPU)
	})
	require.NoError(t, err)
	
	// Test transaction rollback
	var providerID uuid.UUID
	err = rm.WithTransaction(ctx, func(txRM repositories.RepositoryManager) error {
		// Create provider
		testProvider := &provider.ProviderResource{
			Name:   "Rollback Provider",
			Email:  "rollback@example.com",
			Status: provider.ProviderStatusActive,
		}
		
		err := txRM.Providers().Create(ctx, testProvider)
		if err != nil {
			return err
		}
		providerID = testProvider.ID
		
		// Force an error to trigger rollback
		return assert.AnError
	})
	require.Error(t, err)
	
	// Verify provider was not created due to rollback
	_, err = rm.Providers().GetByID(ctx, providerID)
	require.Error(t, err)
}

// getTestConfig returns test database configuration
func getTestConfig() *config.Config {
	cfg := config.DefaultConfig()
	
	// Override with test database settings
	cfg.Database.Database = "depin_inventory_test"
	cfg.Database.Host = getEnvOrDefault("TEST_DB_HOST", "localhost")
	cfg.Database.Port = 5432
	cfg.Database.User = getEnvOrDefault("TEST_DB_USER", "postgres")
	cfg.Database.Password = getEnvOrDefault("TEST_DB_PASSWORD", "")
	
	// Disable Redis for tests
	cfg.Redis.Host = ""
	
	return cfg
}

// getEnvOrDefault returns environment variable or default value
func getEnvOrDefault(env, defaultValue string) string {
	if value := os.Getenv(env); value != "" {
		return value
	}
	return defaultValue
}