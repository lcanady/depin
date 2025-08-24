# DePIN Resource Inventory Database

A comprehensive database system for managing DePIN (Decentralized Physical Infrastructure Network) resources, specifically designed for GPU compute resource inventory management.

## Architecture Overview

The database system follows a layered architecture:

- **Models Layer**: Data models for resources (GPU, Provider, Common types)
- **Repository Layer**: CRUD operations and business logic
- **Migration Layer**: Database schema versioning and management
- **Configuration Layer**: Database connection and settings management
- **Scripts Layer**: Backup, recovery, and maintenance utilities

## Features

### Core Functionality
- **Provider Management**: Registration, authentication, heartbeat monitoring
- **GPU Resource Inventory**: Comprehensive GPU metadata and capability tracking
- **Health Monitoring**: Real-time resource health checks and status tracking  
- **Verification System**: Resource capability verification and scoring
- **Usage Metrics**: Time-series resource utilization tracking
- **Search & Filtering**: Advanced search capabilities with flexible filters

### Technical Features
- **PostgreSQL Backend**: Production-grade RDBMS with JSONB support
- **Redis Caching**: Optional caching layer for performance optimization
- **Connection Pooling**: Efficient database connection management
- **Transaction Support**: ACID-compliant operations with rollback support
- **Migration System**: Version-controlled schema evolution
- **Backup & Recovery**: Automated backup with disaster recovery procedures
- **Health Monitoring**: Database and service health monitoring
- **Performance Optimization**: Indexed queries and batch operations

## Quick Start

### Prerequisites
- PostgreSQL 12+ 
- Redis 6+ (optional, for caching)
- Go 1.19+

### Database Setup

1. **Create Database**:
```bash
createdb depin_inventory
```

2. **Run Migrations**:
```bash
go run ./cmd/migrate --command=up
```

3. **Check Status**:
```bash
go run ./cmd/migrate --command=status
```

### Configuration

Create a configuration file or use environment variables:

```yaml
database:
  host: localhost
  port: 5432
  database: depin_inventory
  user: postgres
  password: your_password
  ssl_mode: disable

redis:
  host: localhost
  port: 6379
  database: 0
```

### Basic Usage

```go
package main

import (
    "context"
    "github.com/your-org/depin/database/inventory/config"
    "github.com/your-org/depin/database/inventory/repositories"
)

func main() {
    // Load configuration
    cfg := config.DefaultConfig()
    
    // Create repository manager
    rm, err := repositories.NewRepositoryManager(cfg)
    if err != nil {
        panic(err)
    }
    defer rm.Close()
    
    ctx := context.Background()
    
    // Access repositories
    providers := rm.Providers()
    gpus := rm.GPUs()
    
    // Example: List all active GPU resources
    activeGPUs, err := gpus.ListByStatus(ctx, "active")
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Found %d active GPUs\n", len(activeGPUs))
}
```

## API Reference

### Provider Repository

```go
// Create a new provider
provider := &provider.ProviderResource{
    Name: "Compute Provider Inc",
    Email: "admin@computeprovider.com",
    Status: provider.ProviderStatusActive,
    // ... other fields
}
err := providerRepo.Create(ctx, provider)

// Search providers with filters
filter := &provider.ProviderSearchFilter{
    Regions: []string{"us-east-1"},
    MinReputation: &0.8,
}
results, err := providerRepo.Search(ctx, filter, nil, pagination)

// Record heartbeat
heartbeat := &provider.ProviderHeartbeat{
    ProviderID: providerID,
    Status: "healthy",
    SystemMetrics: provider.SystemMetrics{
        CPUUtilization: 45.2,
        MemoryUtilization: 68.5,
    },
}
err := providerRepo.RecordHeartbeat(ctx, heartbeat)
```

### GPU Repository

```go
// Create GPU resource
gpu := &gpu.GPUResource{
    BaseResource: common.BaseResource{
        ProviderID: providerID,
        Name: "NVIDIA RTX 4090",
        Status: common.ResourceStatusActive,
        Region: "us-west-2",
    },
    Vendor: "NVIDIA",
    Specs: gpu.GPUSpecs{
        MemoryTotalMB: 24576,
        CUDACores: 16384,
        Architecture: "Ada Lovelace",
    },
    Capabilities: gpu.GPUCapabilities{
        SupportsCUDA: true,
        SupportsTensorOps: true,
    },
}
err := gpuRepo.Create(ctx, gpu)

// Search available GPUs
filter := &gpu.GPUSearchFilter{
    MinMemoryMB: &16384,
    SupportsCUDA: &true,
    IsAllocated: &false,
}
available, err := gpuRepo.Search(ctx, filter, nil, pagination)

// Allocate GPU
err := gpuRepo.MarkAsAllocated(ctx, gpuID, allocationID, time.Now())
```

### Transaction Support

```go
// Perform operations in a transaction
err := rm.WithTransaction(ctx, func(txRM repositories.RepositoryManager) error {
    // Create provider
    err := txRM.Providers().Create(ctx, provider)
    if err != nil {
        return err // Transaction will rollback
    }
    
    // Create associated GPUs
    for _, gpu := range gpus {
        gpu.ProviderID = provider.ID
        err := txRM.GPUs().Create(ctx, gpu)
        if err != nil {
            return err // Transaction will rollback
        }
    }
    
    return nil // Transaction will commit
})
```

## Database Schema

### Key Tables

- **providers**: Provider registration and metadata
- **gpu_resources**: GPU inventory with specifications and status
- **provider_heartbeats**: Time-series provider health data
- **health_checks**: Resource health monitoring records
- **verifications**: Resource capability verification results
- **usage_metrics**: Time-series resource utilization data
- **gpu_processes**: Active processes on GPU resources
- **gpu_benchmarks**: Performance benchmark results
- **gpu_allocations**: Resource allocation tracking

### Indexes

The system includes optimized indexes for:
- Provider lookups by email, status, region
- GPU searches by vendor, capabilities, allocation status
- Time-series queries on heartbeats and metrics
- Resource filtering by multiple criteria

## CLI Tools

### Migration Management

```bash
# Apply all pending migrations
go run ./cmd/migrate --command=up

# Rollback latest migration
go run ./cmd/migrate --command=down

# Check migration status
go run ./cmd/migrate --command=status
```

### Backup Management

```bash
# Create full backup
go run ./cmd/backup --command=create --type=full

# Create incremental backup
go run ./cmd/backup --command=create --type=incremental

# List available backups
go run ./cmd/backup --command=list

# Restore from backup
go run ./cmd/backup --command=restore --path=/path/to/backup.sql.gz --target=depin_inventory

# Verify backup integrity
go run ./cmd/backup --command=verify --path=/path/to/backup.sql.gz

# Run scheduled backups
go run ./cmd/backup --command=schedule

# Test disaster recovery
go run ./cmd/backup --command=dr-test
```

## Performance Considerations

### Indexing Strategy
- B-tree indexes on frequently queried columns
- GIN indexes on JSONB columns for flexible queries
- Partial indexes for filtered queries
- Composite indexes for multi-column searches

### Query Optimization
- Use connection pooling (configured via MaxOpenConns)
- Implement pagination for large result sets
- Use batch operations for multiple inserts/updates
- Cache frequently accessed data in Redis

### Monitoring
- Database connection statistics
- Query performance metrics
- Cache hit/miss ratios
- Slow query identification

## Testing

### Integration Tests

Run the complete integration test suite:

```bash
# Set up test database
createdb depin_inventory_test

# Run tests
go test ./test -v

# Run with coverage
go test ./test -cover -v
```

### Test Environment Variables

- `TEST_DB_HOST`: Test database host (default: localhost)
- `TEST_DB_USER`: Test database user (default: postgres)
- `TEST_DB_PASSWORD`: Test database password

## Deployment

### Docker Deployment

```dockerfile
FROM golang:1.19-alpine AS builder
COPY . /app
WORKDIR /app
RUN go build -o migrate ./cmd/migrate
RUN go build -o backup ./cmd/backup

FROM alpine:latest
RUN apk add --no-cache postgresql-client
COPY --from=builder /app/migrate /app/backup /usr/local/bin/
```

### Kubernetes Deployment

The system includes Kubernetes-ready configuration with:
- Health checks for database connectivity
- Resource limits and requests
- ConfigMap and Secret integration
- Persistent volume claims for backup storage

## Maintenance

### Regular Tasks

1. **Backup Management**:
   - Daily automated backups
   - Weekly backup verification
   - Monthly disaster recovery tests

2. **Data Cleanup**:
   - Remove old heartbeat records (>7 days)
   - Archive old usage metrics (>30 days)
   - Clean up expired verifications

3. **Performance Monitoring**:
   - Monitor connection pool usage
   - Identify slow queries
   - Optimize indexes as needed

### Troubleshooting

**Connection Issues**:
- Check database connectivity and credentials
- Verify connection pool settings
- Monitor connection usage patterns

**Performance Issues**:
- Analyze slow query logs
- Check index usage with EXPLAIN ANALYZE
- Monitor memory and CPU usage

**Data Integrity**:
- Run VACUUM and ANALYZE regularly
- Check foreign key constraints
- Verify backup integrity

## Contributing

1. Follow Go coding standards and conventions
2. Add tests for new functionality
3. Update documentation for API changes
4. Use semantic versioning for database migrations
5. Test against supported PostgreSQL versions

## License

This project is licensed under the MIT License. See LICENSE file for details.