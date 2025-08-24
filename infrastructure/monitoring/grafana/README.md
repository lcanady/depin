# Grafana Monitoring Setup for DePIN AI Compute

This directory contains the complete Grafana setup for monitoring the DePIN AI compute platform, providing comprehensive visualization and alerting capabilities.

## Overview

The Grafana deployment includes:
- Production-ready HA configuration with 2 replicas
- Persistent storage for configurations and dashboards
- Pre-configured Prometheus data source integration
- Built-in dashboards for system, Kubernetes, and DePIN-specific metrics
- Secure authentication and authorization
- Automated dashboard provisioning
- SSL/TLS termination via ingress

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Ingress       │────│   Grafana        │────│   Prometheus    │
│   (SSL/TLS)     │    │   (HA Setup)     │    │   Data Source   │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │
                       ┌────────▼────────┐
                       │  Persistent     │
                       │  Storage        │
                       │  (Dashboards &  │
                       │   Config)       │
                       └─────────────────┘
```

## Quick Start

1. **Deploy Grafana:**
   ```bash
   cd infrastructure/monitoring/grafana
   ./deploy.sh
   ```

2. **Access Grafana:**
   ```bash
   # Port forward for local access
   kubectl port-forward -n monitoring service/grafana 3000:3000
   
   # Get admin password
   kubectl get secret grafana-secret -n monitoring -o jsonpath='{.data.admin-password}' | base64 -d
   ```

3. **Visit:** http://localhost:3000
   - Username: `admin`
   - Password: (from secret above)

## Deployment Options

### Production Deployment
```bash
# Standard deployment
./deploy.sh

# With custom admin password
GRAFANA_ADMIN_PASSWORD="your-secure-password" ./deploy.sh
```

### Development/Testing
```bash
# Dry run to see what would be deployed
DRY_RUN=true ./deploy.sh

# Deploy to specific context
KUBECTL_CONTEXT="dev-cluster" ./deploy.sh
```

### Management Commands
```bash
# Check deployment status
./deploy.sh status

# Remove deployment
./deploy.sh destroy
```

## Configuration

### Data Sources

The deployment automatically configures:
- **Prometheus**: Primary metrics data source
- **Alertmanager**: Alert management integration

Data sources are provisioned via ConfigMap and will auto-discover Prometheus endpoints.

### Dashboard Organization

Dashboards are organized into folders:

- **System Monitoring**: Node-level metrics (CPU, memory, disk, network)
- **Kubernetes**: Cluster metrics (pods, nodes, resources)
- **DePIN AI Compute**: Platform-specific metrics (compute nodes, workloads, economics)
- **Applications**: Application-specific dashboards

### Built-in Dashboards

1. **System Overview** (`system-overview`)
   - System uptime and health
   - CPU utilization across nodes
   - Memory usage patterns
   - Disk I/O performance
   - Network traffic analysis

2. **Kubernetes Cluster** (`kubernetes-cluster`)
   - Node and pod status
   - Resource utilization by namespace
   - Top consumers by CPU/memory
   - Cluster capacity planning

3. **DePIN AI Compute Network** (`depin-ai-compute`)
   - Active compute nodes
   - GPU utilization
   - AI workload distribution
   - Network bandwidth for model distribution
   - Economic metrics (token rewards, compute hours)

## Authentication & Security

### Built-in Authentication
- Admin user with secure password generation
- Session management with configurable timeouts
- RBAC integration with Kubernetes service accounts

### Security Features
- Secure cookie configuration
- XSS protection enabled
- CSRF protection
- Secure secret management via Kubernetes secrets

### Network Security
- Internal cluster communication via service discovery
- SSL/TLS termination at ingress level
- Network policies for traffic restriction

## Persistence & Backup

### Storage Configuration
- Primary PVC: 10Gi for dashboards and configuration
- Backup PVC: 5Gi for automated backups
- StorageClass: `standard` (configurable)

### Data Retention
- Dashboard configurations persist across restarts
- Plugin data and user preferences maintained
- Automated backup capabilities built-in

## Monitoring & Alerting

### Health Checks
- Liveness probe on `/api/health` endpoint
- Readiness probe with configurable timeouts
- Prometheus scraping enabled for Grafana metrics

### Resource Management
- CPU: 250m request, 500m limit
- Memory: 512Mi request, 1Gi limit
- Horizontal scaling ready (2 replicas by default)

## Customization

### Adding Custom Dashboards

1. Create dashboard JSON in `dashboards/grafana/`
2. Add to ConfigMap in `k8s/monitoring/grafana/dashboards.yaml`
3. Redeploy: `kubectl apply -k k8s/monitoring/grafana/`

### Modifying Configuration

Edit configuration files and apply changes:
```bash
# Update Grafana config
kubectl apply -f k8s/monitoring/grafana/configmap.yaml

# Update data sources
kubectl apply -f k8s/monitoring/grafana/datasources.yaml

# Restart pods to pick up changes
kubectl rollout restart deployment/grafana -n monitoring
```

### Adding Data Sources

1. Edit `k8s/monitoring/grafana/datasources.yaml`
2. Add new data source configuration
3. Apply: `kubectl apply -f k8s/monitoring/grafana/datasources.yaml`

## Troubleshooting

### Common Issues

1. **Pods not starting:**
   ```bash
   kubectl logs -l app.kubernetes.io/name=grafana -n monitoring
   kubectl describe pod -l app.kubernetes.io/name=grafana -n monitoring
   ```

2. **Data source connection issues:**
   ```bash
   # Check Prometheus connectivity
   kubectl exec -it deployment/grafana -n monitoring -- curl http://prometheus-server.monitoring.svc.cluster.local:9090/api/v1/query?query=up
   ```

3. **Storage issues:**
   ```bash
   kubectl get pvc -n monitoring
   kubectl describe pvc grafana-pvc -n monitoring
   ```

4. **Ingress not accessible:**
   ```bash
   kubectl get ingress grafana -n monitoring
   kubectl describe ingress grafana -n monitoring
   ```

### Debug Mode

Enable debug logging:
```bash
kubectl patch deployment grafana -n monitoring -p '{"spec":{"template":{"spec":{"containers":[{"name":"grafana","env":[{"name":"GF_LOG_LEVEL","value":"debug"}]}]}}}}'
```

## Integration with DePIN Platform

### Metrics Collection

The setup expects these DePIN-specific metrics:
- `depin_active_workloads`: Current AI workloads by type
- `depin_network_bytes_total`: Network transfer metrics
- `depin_token_rewards_total`: Economic rewards tracking
- `depin_compute_time_hours`: Compute time utilization
- `nvidia_gpu_utilization`: GPU usage metrics

### Custom Metrics

To add custom metrics for your DePIN implementation:

1. Instrument your application with Prometheus client libraries
2. Expose metrics on `/metrics` endpoint
3. Configure Prometheus to scrape your service
4. Create custom dashboards using the metrics

### Alert Integration

Configure alerts for DePIN-specific conditions:
```yaml
# Example alert rule
groups:
  - name: depin-compute
    rules:
      - alert: LowComputeCapacity
        expr: sum(up{job="depin-compute-node"}) < 3
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "DePIN compute capacity is low"
```

## Performance Tuning

### Resource Optimization

For large deployments, consider:
- Increasing replica count for HA
- Adjusting memory limits based on dashboard count
- Using faster storage classes for better performance

### Query Optimization

- Use recording rules for complex queries
- Implement proper time range restrictions
- Utilize dashboard variables for filtering

## Support

For issues specific to the DePIN AI compute platform:
1. Check logs: `kubectl logs -l app.kubernetes.io/name=grafana -n monitoring`
2. Verify metrics: Access Prometheus UI to confirm metric availability
3. Test connectivity: Use port-forward to isolate network issues

For Grafana-specific issues, refer to the [official documentation](https://grafana.com/docs/).