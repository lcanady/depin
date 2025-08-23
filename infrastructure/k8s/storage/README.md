# Kubernetes Storage Configuration for DePIN AI Compute

This directory contains the complete storage infrastructure configuration for the DePIN AI compute platform, including storage provisioners, storage classes, persistent volume templates, backup systems, and testing procedures.

## Overview

The storage system is designed to provide different performance tiers and reliability levels for various workload requirements:

- **High-Performance Storage**: Fast SSDs for AI compute workloads requiring low latency
- **Standard Storage**: General-purpose storage for most applications  
- **Backup Storage**: Long-term retention with emphasis on durability
- **Ephemeral Storage**: High-speed temporary storage for caching and temporary data

## Architecture

```
infrastructure/k8s/storage/
├── provisioners/          # Storage provisioner deployments
│   ├── local-path-provisioner.yaml      # Local path provisioner for development
│   ├── csi-driver-host-path.yaml        # CSI hostpath driver
│   └── csi-hostpath-controller.yaml     # CSI controller components
├── classes/              # Storage class definitions
│   ├── fast-ssd.yaml              # High-performance SSD storage
│   ├── standard-storage.yaml      # General-purpose storage
│   ├── backup-storage.yaml        # Backup and archival storage
│   └── ephemeral-storage.yaml     # Temporary/memory storage
└── backup/              # Backup and disaster recovery
    ├── velero-backup-system.yaml     # Velero backup system
    ├── backup-schedules.yaml         # Automated backup schedules
    └── disaster-recovery-procedures.yaml # DR runbooks and procedures
```

## Storage Provisioners

### Local Path Provisioner
- **Use Case**: Development and local testing
- **Features**: Simple local storage provisioning
- **Path**: `/opt/local-path-provisioner`
- **Namespace**: `local-path-storage`

### CSI Host Path Driver
- **Use Case**: Production environments with CSI support
- **Features**: Volume snapshots, expansion, advanced features
- **Namespace**: `kube-system`

## Storage Classes

### Fast SSD (`fast-ssd`)
- **Performance Tier**: High-performance
- **Use Case**: AI compute workloads, databases, high-IOPS applications
- **Access Modes**: ReadWriteOnce (RWO)
- **Volume Expansion**: Supported
- **Reclaim Policy**: Delete

### Standard Storage (`standard-storage`)
- **Performance Tier**: Standard
- **Use Case**: General applications, web servers, development
- **Access Modes**: ReadWriteOnce (RWO), ReadWriteMany (RWX)
- **Volume Expansion**: Supported
- **Reclaim Policy**: Delete
- **Default**: Yes

### Backup Storage (`backup-storage`)
- **Performance Tier**: Low (optimized for durability)
- **Use Case**: Backups, archival, long-term storage
- **Access Modes**: ReadWriteOnce (RWO), ReadOnlyMany (ROX)
- **Volume Expansion**: Supported
- **Reclaim Policy**: Retain

### Memory Storage (`memory-storage`)
- **Performance Tier**: Extreme (in-memory)
- **Use Case**: Caching, temporary data, high-speed processing
- **Access Modes**: ReadWriteOnce (RWO)
- **Volume Expansion**: Not supported
- **Reclaim Policy**: Delete

## Deployment Instructions

### 1. Deploy Storage Provisioners

```bash
# Deploy local path provisioner (for development)
kubectl apply -f provisioners/local-path-provisioner.yaml

# Deploy CSI hostpath driver (for production)
kubectl apply -f provisioners/csi-driver-host-path.yaml
kubectl apply -f provisioners/csi-hostpath-controller.yaml
```

### 2. Create Storage Classes

```bash
# Apply all storage classes
kubectl apply -f classes/

# Verify storage classes
kubectl get storageclass
```

### 3. Set Up Backup System (Optional)

```bash
# Deploy Velero backup system
kubectl apply -f backup/velero-backup-system.yaml

# Configure backup schedules
kubectl apply -f backup/backup-schedules.yaml

# Set up disaster recovery procedures
kubectl apply -f backup/disaster-recovery-procedures.yaml
```

## Usage Examples

### AI Compute Workload with Multiple Storage Types

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: ai-model-storage
spec:
  storageClassName: fast-ssd
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 100Gi
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: ai-dataset-storage
spec:
  storageClassName: standard-storage
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 500Gi
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ai-compute-node
spec:
  template:
    spec:
      containers:
      - name: ai-compute
        image: tensorflow/tensorflow:latest-gpu
        volumeMounts:
        - name: model-storage
          mountPath: /models
        - name: dataset-storage
          mountPath: /datasets
      volumes:
      - name: model-storage
        persistentVolumeClaim:
          claimName: ai-model-storage
      - name: dataset-storage
        persistentVolumeClaim:
          claimName: ai-dataset-storage
```

### Distributed Storage for Multi-Node Applications

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: shared-data
spec:
  storageClassName: standard-storage
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 1Ti
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: distributed-app
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: app-container
        image: app:latest
        volumeMounts:
        - name: shared-data
          mountPath: /shared
        - name: local-data
          mountPath: /local
      volumes:
      - name: shared-data
        persistentVolumeClaim:
          claimName: shared-data
  volumeClaimTemplates:
  - metadata:
      name: local-data
    spec:
      storageClassName: fast-ssd
      accessModes:
        - ReadWriteOnce
      resources:
        requests:
          storage: 50Gi
```

## Storage Performance Guidelines

### Fast SSD Storage
- **IOPS**: 10,000+ (4K random)
- **Throughput**: 500+ MB/s sequential
- **Latency**: <1ms
- **Use for**: Database, AI model storage, high-performance compute

### Standard Storage  
- **IOPS**: 3,000-10,000 (4K random)
- **Throughput**: 100-500 MB/s sequential
- **Latency**: 1-10ms
- **Use for**: Application data, logs, general workloads

### Backup Storage
- **IOPS**: 100-1,000 (4K random)
- **Throughput**: 50-100 MB/s sequential
- **Latency**: 10-100ms
- **Use for**: Backups, archives, cold storage

## Backup and Disaster Recovery

### Automated Backup Schedules

- **Daily Backup**: 2 AM UTC, 30-day retention
- **Weekly Full Backup**: Sunday 1 AM UTC, 90-day retention  
- **Critical Data Backup**: Every 6 hours, 7-day retention
- **Configuration Backup**: Hourly, 1-day retention

### Disaster Recovery Procedures

1. **Full Cluster Recovery**: Complete cluster restoration from backup
2. **Individual Volume Recovery**: Restore specific volumes
3. **Data Integrity Verification**: Validate restored data integrity
4. **Automated DR Testing**: Monthly disaster recovery tests

### Manual Backup Operations

```bash
# Create on-demand backup
velero backup create manual-backup-$(date +%Y%m%d) --wait

# Restore from backup
velero restore create restore-$(date +%Y%m%d) \
  --from-backup manual-backup-20231201 --wait

# List available backups
velero get backups

# Check restore status
velero get restores
```

## Testing and Validation

The storage system includes comprehensive testing procedures:

### Test Categories
- **Volume Attachment Tests**: Verify PVC binding and pod mounting
- **Performance Benchmarks**: Measure storage performance characteristics
- **Multi-Pod Access Tests**: Validate ReadWriteMany scenarios
- **Volume Expansion Tests**: Test dynamic volume expansion
- **Backup/Restore Tests**: Verify backup and recovery procedures

### Running Tests

```bash
# Run comprehensive storage tests
cd ../volumes/tests/
./run-tests.sh

# Run individual test category
kubectl apply -f volume-attachment-tests.yaml

# Check test results
kubectl logs -n storage-tests -l purpose=testing
```

### Health Monitoring

```bash
# Check storage class status
kubectl get storageclass

# Monitor PV/PVC status
kubectl get pv,pvc --all-namespaces

# View storage events
kubectl get events --field-selector involvedObject.kind=PersistentVolume
```

## Troubleshooting

### Common Issues

#### PVC Stuck in Pending
```bash
# Check storage class exists
kubectl get storageclass

# Verify provisioner is running
kubectl get pods -n local-path-storage
kubectl get pods -n kube-system | grep csi

# Check PVC details
kubectl describe pvc <pvc-name>
```

#### Pod Can't Mount Volume
```bash
# Check node storage capacity
kubectl describe node <node-name>

# Verify PVC is bound
kubectl get pvc <pvc-name>

# Check pod events
kubectl describe pod <pod-name>
```

#### Storage Performance Issues
```bash
# Run performance benchmark
kubectl apply -f ../volumes/tests/volume-attachment-tests.yaml

# Check node resources
kubectl top nodes

# Monitor storage I/O
kubectl logs -n storage-tests performance-benchmark
```

## Security Considerations

### RBAC Permissions
- Storage provisioners run with cluster-level permissions
- PVC creation restricted by namespace RBAC
- Volume snapshots require appropriate permissions

### Data Encryption
- Consider enabling encryption at rest for sensitive data
- Use encrypted storage classes for production workloads
- Implement network encryption for distributed storage

### Access Controls
- Use PodSecurityPolicies to restrict volume mount paths
- Implement resource quotas to prevent storage abuse
- Monitor storage usage and access patterns

## Monitoring and Alerting

### Metrics to Monitor
- PV/PVC creation and binding rates
- Storage utilization per node and namespace
- I/O performance metrics
- Backup success/failure rates

### Recommended Alerts
- PVC stuck in pending state > 5 minutes
- Node storage utilization > 85%
- Backup failures
- Unusual I/O patterns or performance degradation

## Maintenance Procedures

### Regular Tasks
- Monitor storage usage and capacity planning
- Review and clean up unused PVs/PVCs  
- Test backup and restore procedures monthly
- Update provisioner images and configurations

### Capacity Management
```bash
# Check storage usage by namespace
kubectl top persistentvolumeclaims --all-namespaces

# List large volumes
kubectl get pv --sort-by='.spec.capacity.storage'

# Clean up unused volumes
kubectl get pv | grep Released
```

## Support and Contacts

For issues with storage configuration:
1. Check this documentation and troubleshooting guides
2. Review Kubernetes events and logs
3. Consult the upstream documentation for storage drivers
4. Contact the platform team for assistance

---

**Note**: This storage configuration is designed for the DePIN AI compute platform. Adapt storage classes and provisioners based on your specific infrastructure and requirements.