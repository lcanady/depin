# DePIN RBAC Configuration

This directory contains Role-Based Access Control (RBAC) configurations for the DePIN AI compute infrastructure, implementing fine-grained permissions and service accounts for secure operations.

## Overview

The DePIN RBAC framework implements the principle of least privilege, providing dedicated service accounts and roles for different components and workload types. This ensures that each service has only the permissions necessary for its specific function.

## Directory Structure

```
rbac/
├── README.md                    # This file
├── service-accounts/            # Service account definitions
│   └── depin-service-accounts.yaml
└── roles/                       # Role and binding definitions
    ├── ai-compute-roles.yaml
    ├── monitoring-roles.yaml
    ├── operator-roles.yaml
    └── edge-roles.yaml
```

## Service Accounts

### AI Compute Service Accounts

- **depin-ai-compute**: Standard AI compute workloads
- **depin-ai-coordinator**: AI workload orchestration and scaling
- **depin-crypto**: Cryptographic operations (high security)

### System Service Accounts

- **depin-monitoring**: Prometheus, Grafana, metrics collection
- **depin-logging**: Fluentd, log collection and forwarding
- **depin-log-storage**: Elasticsearch, OpenSearch storage
- **depin-operator**: DePIN operators and controllers
- **depin-backup**: Velero, backup and disaster recovery
- **depin-cert-manager**: TLS certificate management

### Edge Service Accounts

- **depin-api-gateway**: API gateway and routing
- **depin-auth**: Authentication and authorization services
- **depin-ingress**: Ingress controllers and load balancers
- **depin-rate-limiter**: Rate limiting and traffic control

## Permission Matrix

| Service Account | Namespace Scope | Resources | Verbs | Notes |
|----------------|----------------|-----------|-------|--------|
| depin-ai-compute | depin-ai-compute | pods, configmaps, secrets, services, endpoints, events, pvc | get, list, watch, create (events) | Standard compute workload |
| depin-ai-coordinator | depin-ai-compute | pods, deployments, services, configmaps, secrets, hpa | get, list, watch, create, update, patch, delete | Orchestration permissions |
| depin-monitoring | cluster-wide | all resources | get, list, watch | Metrics scraping |
| depin-logging | cluster-wide | pods, pods/log, events, nodes, namespaces | get, list, watch | Log collection |
| depin-operator | cluster-wide | all resources | * | Full operator permissions |
| depin-api-gateway | cross-namespace | services, endpoints, ingresses | get, list, watch | Service discovery |
| depin-auth | depin-edge | secrets, configmaps, services, events | get, list, watch, create, update, patch, delete | Auth management |

## Security Principles

### Least Privilege

Each service account is granted only the minimum permissions required for its function:

- **AI Compute**: Read-only access to configuration, limited pod operations
- **Monitoring**: Read-only cluster-wide access for metrics collection
- **Operators**: Full permissions only for authorized system operators
- **Edge Services**: Network-focused permissions with limited cluster access

### Namespace Isolation

Permissions are scoped to specific namespaces where possible:

- **depin-ai-compute**: AI workload permissions
- **depin-secure**: Restricted access for sensitive operations
- **depin-system**: System component permissions
- **depin-edge**: External-facing service permissions

### Cross-Namespace Access

Limited cross-namespace access for specific use cases:

- **Monitoring**: Cluster-wide read access for metrics
- **Logging**: Cluster-wide read access for logs
- **API Gateway**: Cross-namespace service discovery
- **Operators**: Full cluster management capabilities

## Deployment Instructions

### Prerequisites

- Kubernetes cluster with RBAC enabled
- kubectl access with cluster-admin privileges
- DePIN namespaces already created

### Installation

1. **Deploy Service Accounts**:
   ```bash
   kubectl apply -f service-accounts/depin-service-accounts.yaml
   ```

2. **Deploy RBAC Roles and Bindings**:
   ```bash
   kubectl apply -f roles/ai-compute-roles.yaml
   kubectl apply -f roles/monitoring-roles.yaml
   kubectl apply -f roles/operator-roles.yaml
   kubectl apply -f roles/edge-roles.yaml
   ```

### Validation

Verify service accounts are created:

```bash
kubectl get serviceaccounts --all-namespaces -l 'depin.ai/component'
```

Test permissions for specific service accounts:

```bash
# Test AI compute permissions
kubectl auth can-i get pods --as=system:serviceaccount:depin-ai-compute:depin-ai-compute -n depin-ai-compute

# Test monitoring permissions
kubectl auth can-i get pods --as=system:serviceaccount:depin-system:depin-monitoring --all-namespaces

# Test operator permissions
kubectl auth can-i '*' '*' --as=system:serviceaccount:depin-system:depin-operator
```

## Role Definitions

### AI Compute Roles

**depin-ai-compute-role** (Namespace: depin-ai-compute)
- Pod management: get, list, watch
- Configuration access: configmaps, secrets (read-only)
- Service discovery: services, endpoints (read-only)
- Event logging: events (create)
- Storage access: persistentvolumeclaims (read-only)

**depin-ai-coordinator-role** (Namespace: depin-ai-compute)
- Full pod lifecycle: create, update, patch, delete
- Deployment management: deployments, replicasets
- Service management: create, update, patch
- Autoscaling: horizontalpodautoscalers
- Configuration management: configmaps, secrets

### Monitoring Roles

**depin-prometheus-role** (Cluster-wide)
- Resource discovery: nodes, services, endpoints, pods
- Metrics access: nodes/metrics
- Configuration access: configmaps (read-only)
- Ingress monitoring: ingresses, networkpolicies
- Resource monitoring: deployments, daemonsets, replicasets
- Storage monitoring: storageclasses, volumeattachments, pv, pvc

**depin-grafana-role** (Namespace: depin-system)
- Dashboard management: configmaps, secrets (full access)
- Data source access: services, endpoints
- Pod information: pods (read-only)

**depin-logging-role** (Cluster-wide)
- Log collection: pods, pods/log
- Node information: nodes
- Namespace discovery: namespaces
- Event collection: events

### Operator Roles

**depin-operator-role** (Cluster-wide)
- Full resource management: all resources (all verbs)
- CRD management: customresourcedefinitions
- RBAC management: roles, rolebindings, clusterroles, clusterrolebindings
- Network management: networkpolicies, ingresses
- Storage management: storageclasses, volumeattachments
- Policy management: poddisruptionbudgets, podsecuritypolicies
- Node management: nodes (update, patch)

**depin-cert-manager-role** (Cluster-wide)
- Certificate management: certificates, certificaterequests, issuers, clusterissuers
- Secret management: secrets (full access for certificates)
- Configuration access: configmaps (read-only)
- Event logging: events (create, patch)
- Ingress management: ingresses (update, patch for TLS)
- Service management: services (for ACME challenges)

**depin-backup-role** (Cluster-wide)
- Full backup access: all resources (all verbs)
- Snapshot management: volumesnapshots, volumesnapshotclasses, volumesnapshotcontents
- Velero resources: all velero.io resources

### Edge Roles

**depin-api-gateway-role** (Namespace: depin-edge)
- Service discovery: services, endpoints
- Configuration access: configmaps, secrets (read-only)
- Pod information: pods (read-only)
- Event logging: events (create)

**depin-api-gateway-cluster-role** (Cluster-wide)
- Cross-namespace service discovery: services, endpoints
- Namespace information: namespaces
- Ingress information: ingresses

**depin-auth-role** (Namespace: depin-edge)
- Secret management: secrets (full access for JWT keys)
- Configuration access: configmaps (read-only)
- Event logging: events (create, patch)
- Service access: services, endpoints

**depin-ingress-role** (Cluster-wide)
- Ingress management: ingresses, ingressclasses (read, update status)
- Backend discovery: services, endpoints
- Certificate access: secrets (read-only for TLS)
- Configuration access: configmaps (read-only)
- Event logging: events (create, patch)
- Node information: nodes (for external IP assignment)

## Security Best Practices

### Token Management

- Service account tokens are automatically mounted
- Tokens are rotated according to Kubernetes defaults
- Unused tokens should be cleaned up regularly

### Permission Auditing

Regular auditing of permissions:

```bash
# List all cluster role bindings
kubectl get clusterrolebindings -o wide

# Check specific service account permissions
kubectl describe clusterrolebinding <binding-name>

# Test specific permissions
kubectl auth can-i <verb> <resource> --as=<service-account>
```

### Monitoring Access

Monitor RBAC violations and access patterns:

- Track failed authorization attempts
- Monitor privilege escalation attempts
- Alert on unusual access patterns
- Regular access reviews

## Troubleshooting

### Common Issues

1. **Permission Denied Errors**
   - Check if correct service account is being used
   - Verify role bindings are in place
   - Confirm resource names and namespaces

2. **Cross-Namespace Access Issues**
   - Verify cluster roles vs. namespace roles
   - Check cluster role bindings
   - Confirm namespace selectors

3. **Service Account Token Issues**
   - Check if token is properly mounted
   - Verify service account exists
   - Check for token expiration

### Debug Commands

```bash
# Check service account details
kubectl get serviceaccount <name> -n <namespace> -o yaml

# List role bindings for a service account
kubectl get rolebindings,clusterrolebindings -o wide | grep <service-account>

# Test specific permissions
kubectl auth can-i <verb> <resource> --as=system:serviceaccount:<namespace>:<service-account>

# Check what a service account can do
kubectl auth can-i --list --as=system:serviceaccount:<namespace>:<service-account>
```

## Maintenance

### Regular Tasks

1. **Monthly Permission Review**: Audit service account permissions
2. **Quarterly Access Review**: Remove unused service accounts and bindings
3. **Annual Security Review**: Comprehensive RBAC assessment

### Updates and Changes

1. Test RBAC changes in non-production environment
2. Use principle of least privilege for new permissions
3. Document all RBAC modifications
4. Monitor for permission-related errors after changes

## References

- [Kubernetes RBAC Documentation](https://kubernetes.io/docs/reference/access-authn-authz/rbac/)
- [Service Account Documentation](https://kubernetes.io/docs/reference/access-authn-authz/service-accounts-admin/)
- [RBAC Best Practices](https://kubernetes.io/docs/concepts/security/rbac-good-practices/)