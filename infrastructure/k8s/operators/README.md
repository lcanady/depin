# DePIN AI Compute - Essential Operators

This directory contains the essential operators and infrastructure services required for the DePIN AI compute Kubernetes cluster. These operators provide monitoring, logging, ingress, and certificate management capabilities.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    DePIN AI Compute Cluster                     │
├─────────────────────────────────────────────────────────────────┤
│  External Access (HTTPS/TLS)                                   │
│  ┌─────────────────┐    ┌─────────────────┐                   │
│  │ NGINX Ingress   │    │  cert-manager   │                   │
│  │ Controller      │◄───┤  (TLS Certs)    │                   │
│  └─────────────────┘    └─────────────────┘                   │
├─────────────────────────────────────────────────────────────────┤
│  Monitoring Stack                                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐    │
│  │ Prometheus  │  │ Grafana     │  │ AlertManager        │    │
│  │ (Metrics)   │  │ (Visualize) │  │ (Alert Routing)     │    │
│  └─────────────┘  └─────────────┘  └─────────────────────┘    │
├─────────────────────────────────────────────────────────────────┤
│  Logging Stack                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐    │
│  │Elasticsearch│  │ Kibana      │  │ Fluent Bit          │    │
│  │ (Storage)   │  │ (Visualize) │  │ (Log Collection)    │    │
│  └─────────────┘  └─────────────┘  └─────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

## Components

### 1. Monitoring Stack

#### Prometheus
- **Purpose**: Metrics collection and storage
- **Configuration**: `../monitoring/prometheus-config.yaml`
- **Deployment**: `../monitoring/prometheus-deployment.yaml`
- **Retention**: 30 days, 10GB storage limit
- **Features**:
  - DePIN-specific metrics collection
  - Kubernetes cluster monitoring
  - AI compute workload metrics
  - Custom alerting rules

#### Grafana
- **Purpose**: Metrics visualization and dashboards
- **Configuration**: `../monitoring/grafana-config.yaml`
- **Deployment**: `../monitoring/grafana-deployment.yaml`
- **Features**:
  - Pre-configured DePIN dashboards
  - Kubernetes cluster overview
  - Real-time monitoring
  - Alert visualization

#### AlertManager
- **Purpose**: Alert routing and notification management
- **Configuration**: `../monitoring/alertmanager-config.yaml`
- **Deployment**: `../monitoring/alertmanager-deployment.yaml`
- **Features**:
  - DePIN-specific alert routing
  - Webhook integrations
  - Alert grouping and silencing
  - High availability setup

### 2. Logging Stack (ELK/EFK)

#### Elasticsearch
- **Purpose**: Log storage and indexing
- **Deployment**: `../monitoring/elasticsearch-deployment.yaml`
- **Configuration**: 3-node cluster with auto-discovery
- **Features**:
  - Distributed log storage
  - Full-text search capabilities
  - Index lifecycle management
  - Cluster high availability

#### Fluent Bit
- **Purpose**: Log collection and forwarding
- **Configuration**: `../monitoring/fluent-bit-config.yaml`
- **Deployment**: `../monitoring/fluent-bit-deployment.yaml`
- **Features**:
  - DaemonSet deployment (runs on all nodes)
  - Kubernetes log parsing
  - DePIN-specific log parsing
  - Structured log forwarding

#### Kibana
- **Purpose**: Log visualization and analysis
- **Deployment**: `../monitoring/kibana-deployment.yaml`
- **Features**:
  - Interactive log exploration
  - Custom dashboards
  - Log pattern analysis
  - Security integration

### 3. Ingress Controller

#### NGINX Ingress Controller
- **Purpose**: External access and load balancing
- **Deployment**: `nginx-ingress-controller.yaml`
- **Features**:
  - SSL/TLS termination
  - Load balancing
  - Rate limiting
  - Security headers
  - Metrics collection

### 4. Certificate Management

#### cert-manager
- **Purpose**: Automated TLS certificate management
- **Deployment**: `cert-manager-deployment.yaml`
- **Issuers**: `cert-manager-issuers.yaml`
- **Features**:
  - Let's Encrypt integration
  - Automatic certificate renewal
  - Wildcard certificate support
  - Webhook validation

## Deployment

### Prerequisites

Ensure the following infrastructure is ready:
- ✅ Kubernetes cluster (Stream A)
- ✅ Storage classes and persistent volumes (Stream B)
- ✅ Security policies and RBAC (Stream C)

### Quick Deployment

```bash
# Deploy all essential operators
./deploy-operators.sh
```

### Manual Deployment

```bash
# 1. Deploy namespaces and service accounts
kubectl apply -f namespace.yaml
kubectl apply -f logging-namespace.yaml
kubectl apply -f ingress-controller-namespace.yaml
kubectl apply -f cert-manager-namespace.yaml
kubectl apply -f service-accounts.yaml

# 2. Deploy cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.12.0/cert-manager.crds.yaml
kubectl apply -f cert-manager-deployment.yaml
kubectl wait --for=condition=available deployment -n cert-manager --all --timeout=300s
kubectl apply -f cert-manager-issuers.yaml

# 3. Deploy ingress controller
kubectl apply -f nginx-ingress-controller.yaml
kubectl wait --for=condition=ready pod -n ingress-nginx -l app=ingress-nginx --timeout=300s

# 4. Deploy monitoring stack
kubectl apply -f ../monitoring/prometheus-config.yaml
kubectl apply -f ../monitoring/prometheus-deployment.yaml
kubectl apply -f ../monitoring/alertmanager-config.yaml
kubectl apply -f ../monitoring/alertmanager-deployment.yaml
kubectl apply -f ../monitoring/grafana-config.yaml
kubectl apply -f ../monitoring/grafana-deployment.yaml

# 5. Deploy logging stack
kubectl apply -f ../monitoring/elasticsearch-deployment.yaml
kubectl wait --for=condition=ready pod -n logging -l app=elasticsearch --timeout=600s
kubectl apply -f ../monitoring/fluent-bit-config.yaml
kubectl apply -f ../monitoring/fluent-bit-deployment.yaml
kubectl apply -f ../monitoring/kibana-deployment.yaml

# 6. Deploy ingress resources
kubectl apply -f monitoring-ingress.yaml
```

## Health Validation

Run the health validation script to ensure all operators are working correctly:

```bash
./health-validation.sh
```

The script performs comprehensive checks:
- Pod readiness and health
- Service endpoint accessibility
- Certificate status
- Integration functionality
- Security configuration

## Access URLs

After deployment and DNS configuration:

- **Grafana**: https://grafana.depin-ai-compute.local/
- **Prometheus**: https://prometheus.depin-ai-compute.local/
- **AlertManager**: https://alertmanager.depin-ai-compute.local/
- **Kibana**: https://kibana.depin-ai-compute.local/

### Authentication

- **Grafana**: admin / DePIN-AI-Grafana-2024!
- **Prometheus**: admin / prometheus-depin-2024 (HTTP Basic Auth)
- **AlertManager**: admin / alertmanager-depin-2024 (HTTP Basic Auth)
- **Kibana**: No authentication (internal cluster access only)

## Configuration

### Storage Requirements

- **Prometheus**: 50Gi (fast-ssd storage class)
- **Grafana**: 10Gi (standard-storage)
- **AlertManager**: 10Gi (standard-storage)
- **Elasticsearch**: 100Gi per node (fast-ssd storage class)
- **Kibana**: Uses ephemeral storage

### Resource Requirements

| Component | CPU Request | Memory Request | CPU Limit | Memory Limit |
|-----------|-------------|----------------|-----------|--------------|
| Prometheus | 100m | 512Mi | 1000m | 2Gi |
| Grafana | 100m | 128Mi | 500m | 512Mi |
| AlertManager | 100m | 128Mi | 500m | 512Mi |
| Elasticsearch | 100m | 1Gi | 1000m | 2Gi |
| Fluent Bit | 100m | 64Mi | 500m | 512Mi |
| Kibana | 100m | 512Mi | 1000m | 1Gi |
| NGINX Ingress | 100m | 128Mi | 1000m | 1Gi |
| cert-manager | 10m | 32Mi | 500m | 512Mi |

## Security Features

- **Non-root containers**: All containers run as non-root users
- **Security contexts**: Restricted security contexts with dropped capabilities
- **Network policies**: Traffic segmentation between namespaces
- **TLS encryption**: End-to-end encryption for all external access
- **RBAC**: Least-privilege service accounts
- **Pod Security Standards**: Enforced restricted policies

## Monitoring and Alerting

### DePIN-Specific Alerts

- **DePINNodeDown**: AI compute node unavailable
- **HighAIComputeLoad**: High CPU usage in AI workloads
- **EdgeNodeLatency**: High latency on edge nodes

### Kubernetes Alerts

- **KubernetesPodCrashLooping**: Pods restarting frequently
- **KubernetesPodNotReady**: Pods in non-ready state

### Custom Metrics

- AI compute workload metrics
- Edge network performance
- DePIN network statistics
- Resource utilization

## Troubleshooting

### Common Issues

1. **Certificates not ready**
   ```bash
   kubectl describe certificate -A
   kubectl describe certificaterequest -A
   ```

2. **Ingress not accessible**
   ```bash
   kubectl get ingress -A
   kubectl describe ingress <ingress-name> -n <namespace>
   ```

3. **Monitoring data not appearing**
   ```bash
   kubectl logs -n monitoring deployment/prometheus
   kubectl port-forward -n monitoring svc/prometheus 9090:9090
   ```

4. **Logs not appearing in Kibana**
   ```bash
   kubectl logs -n logging daemonset/fluent-bit
   kubectl port-forward -n logging svc/elasticsearch 9200:9200
   ```

### Log Locations

- Deployment logs: `deploy-operators-<timestamp>.log`
- Health check logs: `health-check-<timestamp>.log`
- Component logs: `kubectl logs -n <namespace> <pod-name>`

## Integration with DePIN Network

The essential operators are configured to integrate seamlessly with the DePIN AI compute network:

- **Service Discovery**: Automatic discovery of DePIN services
- **Metrics Collection**: Custom metrics for AI compute workloads
- **Log Aggregation**: Structured logging for DePIN components
- **Alert Routing**: DePIN-specific alert channels
- **Security Integration**: Aligned with DePIN security policies

## Maintenance

### Regular Tasks

1. **Certificate Renewal**: Automated via cert-manager
2. **Log Rotation**: Managed by Elasticsearch index lifecycle
3. **Metrics Retention**: 30-day retention for Prometheus
4. **Backup Procedures**: Configured for persistent volumes

### Updates

1. Monitor upstream releases for security updates
2. Test updates in staging environment
3. Use rolling updates for zero-downtime deployments
4. Validate functionality after updates

## Support

For issues related to the essential operators:

1. Check the health validation output
2. Review component logs
3. Verify certificate status
4. Check ingress configuration
5. Validate storage and networking

The operators are designed to be self-healing and highly available, providing robust infrastructure for the DePIN AI compute network.