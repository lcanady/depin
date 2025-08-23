# Persistent Volumes and Volume Management

This directory contains persistent volume templates, usage examples, and testing procedures for the DePIN AI compute platform storage system.

## Directory Structure

```
volumes/
├── templates/           # Reusable PV/PVC templates
│   ├── compute-workload-pv.yaml    # High-performance volumes for AI compute
│   ├── data-storage-pv.yaml        # Standard volumes for data storage
│   └── backup-pv.yaml              # Backup volumes with retention
├── examples/           # Real-world usage examples  
│   ├── ai-workload-with-storage.yaml    # AI workload with multiple storage types
│   └── distributed-storage.yaml          # Multi-node distributed storage
└── tests/             # Volume testing and validation
    ├── volume-attachment-tests.yaml      # Comprehensive attachment tests
    ├── automated-test-suite.yaml         # Automated testing framework
    └── run-tests.sh                      # Test execution script
```

## Volume Templates

### Compute Workload Volumes
**File**: `templates/compute-workload-pv.yaml`
- **Purpose**: High-performance storage for AI compute workloads
- **Storage Class**: `fast-ssd`
- **Access Mode**: ReadWriteOnce (RWO)
- **Capacity**: 100Gi (template), customizable
- **Node Affinity**: Worker nodes optimized for compute

### Data Storage Volumes  
**File**: `templates/data-storage-pv.yaml`
- **Purpose**: General data storage with shared access
- **Storage Class**: `standard-storage`
- **Access Modes**: ReadWriteMany (RWX), ReadWriteOnce (RWO)
- **Capacity**: 500Gi (template), customizable
- **Node Affinity**: Storage-optimized nodes

### Backup Volumes
**File**: `templates/backup-pv.yaml`
- **Purpose**: Long-term backup and disaster recovery
- **Storage Class**: `backup-storage`
- **Access Modes**: ReadWriteOnce (RWO), ReadOnlyMany (ROX)
- **Capacity**: 1Ti (template), customizable
- **Reclaim Policy**: Retain

## Usage Examples

### AI Workload with Multi-Tier Storage

The `examples/ai-workload-with-storage.yaml` demonstrates a complete AI compute deployment using multiple storage tiers:

```yaml
# High-performance storage for ML models
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: ai-model-storage
spec:
  storageClassName: fast-ssd
  accessModes: [ReadWriteOnce]
  resources:
    requests:
      storage: 100Gi

# Shared storage for datasets
apiVersion: v1
kind: PersistentVolumeClaim  
metadata:
  name: ai-dataset-storage
spec:
  storageClassName: standard-storage
  accessModes: [ReadWriteMany]
  resources:
    requests:
      storage: 500Gi

# In-memory cache for temporary data
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: ai-temp-cache
spec:
  storageClassName: memory-storage
  accessModes: [ReadWriteOnce]
  resources:
    requests:
      storage: 8Gi
```

### Distributed Storage Pattern

The `examples/distributed-storage.yaml` shows how to set up shared storage across multiple compute nodes:

```yaml
# Shared data volume accessible by all nodes
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: shared-data-pvc
spec:
  storageClassName: standard-storage
  accessModes: [ReadWriteMany]
  resources:
    requests:
      storage: 1Ti

# StatefulSet with per-pod local storage
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: distributed-compute-nodes
spec:
  replicas: 3
  volumeClaimTemplates:
  - metadata:
      name: node-local-storage
    spec:
      storageClassName: fast-ssd
      accessModes: [ReadWriteOnce]
      resources:
        requests:
          storage: 50Gi
```

## Testing and Validation

### Test Suite Overview

The testing framework provides comprehensive validation of:
- Volume provisioning across all storage classes
- Volume attachment and mounting 
- Performance characteristics
- Multi-pod access patterns
- Volume expansion capabilities
- Data persistence and integrity

### Running Tests

#### Quick Test
```bash
cd tests/
./run-tests.sh
```

#### Individual Test Components
```bash
# Test volume attachment
kubectl apply -f volume-attachment-tests.yaml

# Run automated test suite  
kubectl apply -f automated-test-suite.yaml

# Check test results
kubectl logs -n storage-tests -l purpose=testing
```

### Test Categories

#### Volume Attachment Tests
- **Fast SSD**: High-performance volume provisioning and mounting
- **Standard Storage**: General-purpose volume operations  
- **Shared Storage**: ReadWriteMany access validation
- **Memory Storage**: In-memory filesystem testing

#### Performance Benchmarks
- **Sequential I/O**: Large file transfer performance
- **Random I/O**: Database-style access patterns
- **Mixed Workloads**: Real-world application simulation
- **Latency Testing**: Response time measurements

#### Functional Tests
- **Volume Expansion**: Dynamic capacity increases
- **Data Persistence**: Survival across pod restarts
- **Multi-Pod Access**: Concurrent access validation
- **Cleanup Verification**: Proper resource deallocation

### Test Results Interpretation

#### Success Criteria
- **PVC Binding**: < 30 seconds for fast-ssd, < 60 seconds for others
- **Pod Mount**: < 60 seconds across all storage classes
- **I/O Performance**: Meets minimum thresholds per storage tier
- **Data Integrity**: 100% data consistency across all operations

#### Common Performance Benchmarks
```
Fast SSD Storage:
- Sequential Write: >500 MB/s
- Sequential Read: >800 MB/s  
- Random IOPS (4K): >10,000
- Latency: <1ms

Standard Storage:
- Sequential Write: >100 MB/s
- Sequential Read: >200 MB/s
- Random IOPS (4K): >3,000
- Latency: <10ms
```

## Volume Management Best Practices

### Storage Class Selection

**Use Fast SSD (`fast-ssd`) for:**
- AI model storage and training data
- Database storage (PostgreSQL, MongoDB)
- High-throughput applications
- Real-time processing workloads

**Use Standard Storage (`standard-storage`) for:**
- Application data and logs
- Shared datasets and content
- Development and testing environments
- General-purpose applications

**Use Backup Storage (`backup-storage`) for:**
- Long-term data archival
- Disaster recovery backups
- Compliance data retention
- Infrequently accessed data

**Use Memory Storage (`memory-storage`) for:**
- Temporary cache and scratch space
- High-speed processing buffers
- Session storage and temporary files
- Performance-critical temporary data

### Sizing Guidelines

#### Compute Workloads
- **Model Storage**: 50-200Gi per model
- **Training Data**: 100Gi-10Ti depending on dataset
- **Temporary Cache**: 8-32Gi per compute node
- **Logs and Metrics**: 10-50Gi per application

#### Data Storage
- **Shared Datasets**: 100Gi-50Ti based on requirements
- **User Data**: 10-100Gi per user/tenant
- **System Backups**: 2-5x primary data size
- **Archive Storage**: Unlimited, cost-optimized

### Access Mode Selection

**ReadWriteOnce (RWO)**
- Single pod access required
- Highest performance and consistency
- Most storage implementations support
- Use for databases, logs, caches

**ReadWriteMany (RWX)**  
- Multiple pods need concurrent access
- Shared data processing workflows
- Content distribution and sharing
- May have performance implications

**ReadOnlyMany (ROX)**
- Static content distribution
- Configuration and reference data
- Archived data access
- Security-sensitive read-only data

### Performance Optimization

#### Volume Placement
- Use node affinity for compute-intensive workloads
- Collocate related volumes on same nodes when possible
- Distribute I/O load across multiple storage nodes
- Consider network topology for distributed storage

#### Capacity Planning
- Monitor usage patterns and growth trends
- Plan for peak usage scenarios
- Implement capacity alerts and automation
- Regular review and optimization cycles

## Troubleshooting Guide

### PVC Issues

#### PVC Stuck in Pending
```bash
# Check available storage classes
kubectl get storageclass

# Verify provisioner status
kubectl get pods -n local-path-storage
kubectl get pods -n kube-system | grep csi

# Check PVC events and details
kubectl describe pvc <pvc-name>
kubectl get events --field-selector involvedObject.name=<pvc-name>
```

#### PVC Mount Failures
```bash
# Check node capacity and resources
kubectl describe node <node-name>

# Verify volume attachment
kubectl get volumeattachment

# Check pod status and events
kubectl describe pod <pod-name>
kubectl logs <pod-name>
```

### Performance Issues

#### Slow I/O Performance
```bash
# Run performance benchmark
kubectl apply -f tests/volume-attachment-tests.yaml

# Check node resource utilization  
kubectl top nodes
kubectl describe node <node-name>

# Monitor storage I/O metrics
kubectl logs -n storage-tests <benchmark-pod>
```

#### Volume Expansion Failures
```bash
# Verify storage class supports expansion
kubectl get storageclass <class-name> -o yaml

# Check expansion events
kubectl describe pvc <pvc-name>

# Verify file system expansion
kubectl exec <pod-name> -- df -h /mount/path
```

### Data Issues

#### Data Loss or Corruption
```bash
# Check volume and PVC status
kubectl get pv,pvc

# Verify backup availability
velero get backups

# Check for storage system alerts
kubectl get events --all-namespaces | grep -i error
```

#### Backup and Restore Issues
```bash
# Verify Velero installation
kubectl get pods -n velero

# Check backup storage location
velero get backup-locations

# Test restore functionality
velero restore create test-restore --from-backup <backup-name>
```

## Monitoring and Maintenance

### Key Metrics to Track
- **Volume Utilization**: Storage usage per PVC and namespace
- **I/O Performance**: Throughput, IOPS, and latency metrics
- **Provisioning Time**: PVC binding and pod mounting duration
- **Error Rates**: Failed provisioning, mounting, and I/O operations

### Regular Maintenance Tasks
- **Weekly**: Review storage utilization and capacity planning
- **Monthly**: Run comprehensive test suite and performance benchmarks  
- **Quarterly**: Update provisioner versions and configurations
- **Annually**: Review storage architecture and technology updates

### Automation Opportunities
- **Capacity Alerts**: Automated notifications for high utilization
- **Performance Monitoring**: Continuous I/O performance tracking
- **Test Execution**: Scheduled validation of storage functionality
- **Backup Verification**: Automated backup testing and validation

---

This volume management system provides a robust foundation for storage operations in the DePIN AI compute platform. Regular testing, monitoring, and maintenance ensure optimal performance and reliability for all workloads.