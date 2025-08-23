# DePIN AI Compute Security Framework

This directory contains the comprehensive security framework for the DePIN AI compute infrastructure, implementing defense-in-depth security controls and compliance standards.

## Overview

The DePIN security framework provides multi-layered security controls designed to protect AI compute workloads and infrastructure components. It implements industry best practices and compliance requirements including CIS Kubernetes Benchmark, NIST Cybersecurity Framework, and SOC 2 controls.

## Architecture

### Security Zones

The infrastructure is divided into security zones with different trust levels:

- **depin-secure**: Highest security for cryptographic workloads (restricted PSS)
- **depin-ai-compute**: Balanced security for AI compute workloads (baseline PSS)  
- **depin-system**: Privileged access for system components (privileged PSS)
- **depin-edge**: External-facing services with controlled exposure (baseline PSS)
- **depin-default**: Standard namespace with restrictive controls (baseline PSS)

### Security Components

1. **Pod Security Standards** - Namespace-level security enforcement
2. **Network Policies** - Traffic segmentation and micro-segmentation
3. **RBAC** - Fine-grained access controls and service accounts
4. **Admission Controllers** - Policy enforcement and validation
5. **Audit Logging** - Comprehensive security event monitoring
6. **Security Testing** - Validation and penetration testing tools

## Directory Structure

```
security/
├── README.md                           # This file
├── pod-security/                       # Pod Security Standards
│   ├── namespace-security-standards.yaml
│   ├── pod-security-policy.yaml
│   └── security-context-constraints.yaml
├── network-policies/                   # Network segmentation
│   ├── default-deny.yaml
│   ├── ai-compute-policies.yaml
│   ├── system-policies.yaml
│   └── edge-policies.yaml
├── admission-controllers/              # Policy enforcement
│   ├── depin-admission-controller.yaml
│   └── open-policy-agent.yaml
├── audit-logging/                      # Security monitoring
│   ├── audit-policy.yaml
│   ├── audit-log-forwarder.yaml
│   └── security-event-rules.yaml
└── tests/                             # Security validation
    ├── security-validation.sh
    ├── penetration-testing.yaml
    └── compliance-check.sh
```

## Deployment Guide

### Prerequisites

1. Kubernetes cluster v1.25+ with proper RBAC enabled
2. kubectl access with cluster-admin privileges
3. cert-manager for TLS certificate management
4. Monitoring stack (Prometheus/Grafana) for security alerts

### Installation Steps

1. **Deploy Namespaces and Pod Security Standards**:
   ```bash
   kubectl apply -f pod-security/namespace-security-standards.yaml
   kubectl apply -f pod-security/pod-security-policy.yaml
   kubectl apply -f pod-security/security-context-constraints.yaml
   ```

2. **Configure Network Policies**:
   ```bash
   kubectl apply -f network-policies/default-deny.yaml
   kubectl apply -f network-policies/ai-compute-policies.yaml
   kubectl apply -f network-policies/system-policies.yaml
   kubectl apply -f network-policies/edge-policies.yaml
   ```

3. **Set up RBAC** (from ../rbac/ directory):
   ```bash
   kubectl apply -f ../rbac/service-accounts/depin-service-accounts.yaml
   kubectl apply -f ../rbac/roles/ai-compute-roles.yaml
   kubectl apply -f ../rbac/roles/monitoring-roles.yaml
   kubectl apply -f ../rbac/roles/operator-roles.yaml
   kubectl apply -f ../rbac/roles/edge-roles.yaml
   ```

4. **Deploy Admission Controllers**:
   ```bash
   kubectl apply -f admission-controllers/depin-admission-controller.yaml
   kubectl apply -f admission-controllers/open-policy-agent.yaml
   ```

5. **Configure Audit Logging**:
   ```bash
   kubectl apply -f audit-logging/audit-policy.yaml
   kubectl apply -f audit-logging/audit-log-forwarder.yaml
   kubectl apply -f audit-logging/security-event-rules.yaml
   ```

6. **Deploy Security Testing Tools**:
   ```bash
   kubectl apply -f tests/penetration-testing.yaml
   ```

### Validation

Run the security validation script to verify the deployment:

```bash
./tests/security-validation.sh
```

Run compliance checks:

```bash
./tests/compliance-check.sh
```

## Security Policies

### Pod Security Standards

- **Restricted**: For cryptographic workloads in `depin-secure`
  - No privileged containers
  - Run as non-root (UID > 1000)
  - Read-only root filesystem
  - All capabilities dropped
  - No host access

- **Baseline**: For AI compute workloads in `depin-ai-compute` and `depin-edge`
  - Limited capabilities (NET_BIND_SERVICE allowed)
  - Run as non-root
  - Limited host path access
  - No privilege escalation

- **Privileged**: For system components in `depin-system`
  - Full access for operators and monitoring
  - Host access for system management
  - All capabilities allowed

### Network Policies

Default-deny approach with selective allow rules:

- **Default Deny**: All namespaces start with deny-all policies
- **AI Compute**: Allows internal communication and storage access
- **System**: Allows monitoring and management traffic
- **Edge**: Allows external ingress and internal service communication

### RBAC Design

Role-based access control with principle of least privilege:

- **Service Accounts**: Dedicated accounts per component type
- **Roles**: Fine-grained permissions for specific functions
- **ClusterRoles**: Cross-namespace access for monitoring/operators
- **Bindings**: Explicit authorization mappings

## Security Monitoring

### Audit Events

Comprehensive logging of security-sensitive operations:

- Authentication and authorization events
- RBAC modifications
- Network policy changes
- Pod security context violations
- Privileged operations
- Certificate management

### Alerting Rules

Automated detection and alerting for:

- Privilege escalation attempts
- Unauthorized secret access
- Network policy violations
- Excessive authentication failures
- Pod security violations
- Suspicious container access

### Incident Response

Automated response procedures for:

- Privilege escalation: Namespace isolation and token revocation
- Unauthorized access: IP blocking and session invalidation
- Policy violations: Enhanced monitoring and audit capture

## Security Testing

### Validation Scripts

1. **security-validation.sh**: Comprehensive security configuration validation
2. **compliance-check.sh**: Multi-framework compliance assessment
3. **penetration-testing.yaml**: Automated security testing tools

### Testing Coverage

- Pod Security Standards enforcement
- Network policy effectiveness
- RBAC configuration validation
- Admission controller functionality
- Audit logging completeness
- TLS certificate management

### Compliance Frameworks

- **CIS Kubernetes Benchmark v1.7.0**: Industry security standards
- **NIST Cybersecurity Framework**: Risk-based security approach
- **SOC 2**: Security controls for service organizations

## Maintenance Procedures

### Regular Tasks

1. **Weekly Security Scans**: Run automated security assessment
2. **Monthly Compliance Checks**: Validate against compliance frameworks
3. **Quarterly Policy Review**: Update security policies and procedures
4. **Annual Penetration Testing**: External security assessment

### Monitoring and Alerting

1. Monitor security metrics through Grafana dashboards
2. Set up alerting for critical security events
3. Review audit logs regularly for anomalies
4. Track compliance metrics and trends

### Update Procedures

1. Test security changes in non-production environment
2. Follow change management processes
3. Update documentation and runbooks
4. Validate security controls after changes

## Troubleshooting

### Common Issues

1. **Pods failing to start**: Check Pod Security Standards and RBAC
2. **Network connectivity issues**: Verify network policy rules
3. **Authentication failures**: Review service account configurations
4. **Audit logging gaps**: Check audit forwarder status

### Debug Commands

```bash
# Check Pod Security Standards
kubectl get namespace -l pod-security.kubernetes.io/enforce

# Verify network policies
kubectl get networkpolicies --all-namespaces

# Check RBAC permissions
kubectl auth can-i <verb> <resource> --as=<user>

# View admission webhook status
kubectl get validatingadmissionwebhooks
kubectl get mutatingadmissionwebhooks

# Monitor security events
kubectl logs -n depin-system -l depin.ai/component=audit-forwarder
```

## Security Contacts

For security incidents or questions:

- **Security Team**: security@depin.ai
- **Incident Response**: incident-response@depin.ai
- **Compliance**: compliance@depin.ai

## Additional Resources

- [Kubernetes Security Best Practices](https://kubernetes.io/docs/concepts/security/)
- [Pod Security Standards](https://kubernetes.io/docs/concepts/security/pod-security-standards/)
- [Network Policies](https://kubernetes.io/docs/concepts/services-networking/network-policies/)
- [CIS Kubernetes Benchmark](https://www.cisecurity.org/benchmark/kubernetes)
- [NIST Cybersecurity Framework](https://www.nist.gov/cyberframework)