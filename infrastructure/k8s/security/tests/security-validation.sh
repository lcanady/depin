#!/bin/bash

# DePIN Security Validation Script
# This script validates the security configuration of the DePIN AI compute infrastructure

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Global variables
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="${SCRIPT_DIR}/security-validation-$(date +%Y%m%d-%H%M%S).log"
FAILURES=0
CHECKS=0

# Logging functions
log() {
    echo -e "$1" | tee -a "$LOG_FILE"
}

log_info() {
    log "${BLUE}[INFO]${NC} $1"
}

log_success() {
    log "${GREEN}[PASS]${NC} $1"
}

log_warning() {
    log "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    log "${RED}[FAIL]${NC} $1"
    ((FAILURES++))
}

# Check functions
check_kubectl() {
    log_info "Checking kubectl connectivity..."
    if kubectl cluster-info > /dev/null 2>&1; then
        log_success "kubectl connectivity verified"
    else
        log_error "kubectl cannot connect to cluster"
        exit 1
    fi
    ((CHECKS++))
}

check_namespaces() {
    log_info "Validating DePIN namespaces..."
    
    local required_namespaces=("depin-ai-compute" "depin-secure" "depin-system" "depin-edge" "depin-default")
    
    for ns in "${required_namespaces[@]}"; do
        if kubectl get namespace "$ns" > /dev/null 2>&1; then
            log_success "Namespace $ns exists"
        else
            log_error "Missing required namespace: $ns"
        fi
        ((CHECKS++))
    done
}

check_pod_security_standards() {
    log_info "Validating Pod Security Standards..."
    
    # Check namespace labels for Pod Security Standards
    local namespaces=("depin-ai-compute" "depin-secure" "depin-system" "depin-edge")
    
    for ns in "${namespaces[@]}"; do
        local labels=$(kubectl get namespace "$ns" -o jsonpath='{.metadata.labels}' 2>/dev/null || echo "{}")
        
        if echo "$labels" | grep -q "pod-security.kubernetes.io/enforce"; then
            log_success "Pod Security Standards enforced in namespace $ns"
        else
            log_error "Pod Security Standards not configured for namespace $ns"
        fi
        ((CHECKS++))
    done
}

check_network_policies() {
    log_info "Validating Network Policies..."
    
    local namespaces=("depin-ai-compute" "depin-secure" "depin-edge")
    
    for ns in "${namespaces[@]}"; do
        local policy_count=$(kubectl get networkpolicies -n "$ns" --no-headers 2>/dev/null | wc -l)
        
        if [ "$policy_count" -gt 0 ]; then
            log_success "Network policies found in namespace $ns ($policy_count policies)"
            
            # Check for default-deny policies
            if kubectl get networkpolicy -n "$ns" -o name 2>/dev/null | grep -q "default-deny"; then
                log_success "Default-deny policy found in namespace $ns"
            else
                log_warning "No default-deny policy found in namespace $ns"
            fi
        else
            log_error "No network policies found in namespace $ns"
        fi
        ((CHECKS++))
    done
}

check_rbac_configuration() {
    log_info "Validating RBAC configuration..."
    
    # Check for DePIN service accounts
    local service_accounts=("depin-ai-compute" "depin-monitoring" "depin-logging" "depin-operator")
    
    for sa in "${service_accounts[@]}"; do
        local found=false
        
        # Check in relevant namespaces
        for ns in depin-ai-compute depin-system depin-edge; do
            if kubectl get serviceaccount "$sa" -n "$ns" > /dev/null 2>&1; then
                log_success "Service account $sa found in namespace $ns"
                found=true
                break
            fi
        done
        
        if [ "$found" = false ]; then
            log_error "Service account $sa not found in any namespace"
        fi
        ((CHECKS++))
    done
    
    # Check for overprivileged roles
    local cluster_admin_bindings=$(kubectl get clusterrolebindings -o json | jq -r '.items[] | select(.roleRef.name == "cluster-admin") | .metadata.name')
    
    log_info "Checking for cluster-admin bindings..."
    if [ -n "$cluster_admin_bindings" ]; then
        log_warning "Found cluster-admin bindings: $cluster_admin_bindings"
    else
        log_success "No unexpected cluster-admin bindings found"
    fi
    ((CHECKS++))
}

check_admission_controllers() {
    log_info "Validating admission controllers..."
    
    # Check if admission controllers are configured
    local admission_config=$(kubectl get validatingadmissionwebhooks -o name 2>/dev/null | grep -c depin || echo "0")
    
    if [ "$admission_config" -gt 0 ]; then
        log_success "DePIN admission controllers configured ($admission_config webhooks)"
    else
        log_warning "No DePIN admission controllers found"
    fi
    ((CHECKS++))
    
    # Check OPA Gatekeeper if deployed
    if kubectl get namespace gatekeeper-system > /dev/null 2>&1; then
        local constraint_count=$(kubectl get constraints --all-namespaces --no-headers 2>/dev/null | wc -l)
        if [ "$constraint_count" -gt 0 ]; then
            log_success "OPA Gatekeeper constraints found ($constraint_count constraints)"
        else
            log_warning "OPA Gatekeeper deployed but no constraints found"
        fi
    else
        log_info "OPA Gatekeeper not deployed (optional)"
    fi
    ((CHECKS++))
}

check_audit_logging() {
    log_info "Validating audit logging configuration..."
    
    # Check for audit policy in cluster configuration
    local audit_forwarder=$(kubectl get daemonset -n depin-system depin-audit-forwarder > /dev/null 2>&1 && echo "found" || echo "missing")
    
    if [ "$audit_forwarder" = "found" ]; then
        log_success "Audit log forwarder deployed"
        
        # Check if pods are running
        local running_pods=$(kubectl get pods -n depin-system -l depin.ai/component=audit-forwarder --no-headers | grep -c "Running" || echo "0")
        if [ "$running_pods" -gt 0 ]; then
            log_success "Audit forwarder pods running ($running_pods pods)"
        else
            log_error "Audit forwarder pods not running"
        fi
    else
        log_error "Audit log forwarder not deployed"
    fi
    ((CHECKS++))
}

check_tls_certificates() {
    log_info "Validating TLS certificates..."
    
    # Check for cert-manager
    if kubectl get namespace cert-manager > /dev/null 2>&1; then
        log_success "cert-manager namespace found"
        
        # Check for certificates
        local cert_count=$(kubectl get certificates --all-namespaces --no-headers 2>/dev/null | wc -l)
        if [ "$cert_count" -gt 0 ]; then
            log_success "TLS certificates found ($cert_count certificates)"
        else
            log_warning "cert-manager deployed but no certificates found"
        fi
    else
        log_warning "cert-manager not deployed"
    fi
    ((CHECKS++))
}

test_security_policies() {
    log_info "Testing security policies with sample workloads..."
    
    # Test 1: Try to create privileged pod (should fail)
    local privileged_test=$(cat << EOF
apiVersion: v1
kind: Pod
metadata:
  name: security-test-privileged
  namespace: depin-ai-compute
  labels:
    depin.ai/test: security-validation
spec:
  containers:
  - name: test
    image: busybox:latest
    securityContext:
      privileged: true
    command: ["sleep", "60"]
EOF
)
    
    if echo "$privileged_test" | kubectl apply --dry-run=server -f - > /dev/null 2>&1; then
        log_error "Privileged pod creation not blocked by security policies"
    else
        log_success "Privileged pod creation correctly blocked"
    fi
    ((CHECKS++))
    
    # Test 2: Try to create pod without security labels (should fail in restricted namespaces)
    local unlabeled_test=$(cat << EOF
apiVersion: v1
kind: Pod
metadata:
  name: security-test-unlabeled
  namespace: depin-secure
spec:
  containers:
  - name: test
    image: busybox:latest
    command: ["sleep", "60"]
EOF
)
    
    if echo "$unlabeled_test" | kubectl apply --dry-run=server -f - > /dev/null 2>&1; then
        log_warning "Unlabeled pod creation allowed (may be expected based on policy)"
    else
        log_success "Unlabeled pod creation correctly blocked"
    fi
    ((CHECKS++))
}

check_security_monitoring() {
    log_info "Validating security monitoring..."
    
    # Check for security monitoring rules
    if kubectl get prometheusrules -n depin-system depin-security-alerts > /dev/null 2>&1; then
        log_success "Security alerting rules configured"
    else
        log_warning "Security alerting rules not found"
    fi
    ((CHECKS++))
    
    # Check for monitoring components
    local monitoring_components=("prometheus" "grafana" "alertmanager")
    
    for component in "${monitoring_components[@]}"; do
        local pod_count=$(kubectl get pods --all-namespaces -l "app.kubernetes.io/name=$component" --no-headers 2>/dev/null | wc -l)
        
        if [ "$pod_count" -gt 0 ]; then
            log_success "Monitoring component $component running ($pod_count pods)"
        else
            log_warning "Monitoring component $component not found"
        fi
        ((CHECKS++))
    done
}

generate_security_report() {
    log_info "Generating security validation report..."
    
    cat << EOF >> "$LOG_FILE"

========================================
SECURITY VALIDATION REPORT
========================================
Timestamp: $(date)
Total Checks: $CHECKS
Failed Checks: $FAILURES
Success Rate: $(( (CHECKS - FAILURES) * 100 / CHECKS ))%

RECOMMENDATIONS:
EOF
    
    if [ $FAILURES -eq 0 ]; then
        log_success "All security checks passed! ✅"
        echo "- Security configuration is compliant with DePIN standards" >> "$LOG_FILE"
        echo "- Continue monitoring security metrics and alerts" >> "$LOG_FILE"
    else
        log_error "$FAILURES security checks failed"
        echo "- Review failed checks and remediate issues" >> "$LOG_FILE"
        echo "- Re-run validation after fixes are applied" >> "$LOG_FILE"
        echo "- Consider implementing additional security controls" >> "$LOG_FILE"
    fi
    
    echo "" >> "$LOG_FILE"
    echo "Detailed log saved to: $LOG_FILE"
}

# Main execution
main() {
    log_info "Starting DePIN Security Validation"
    log_info "Log file: $LOG_FILE"
    
    check_kubectl
    check_namespaces
    check_pod_security_standards
    check_network_policies
    check_rbac_configuration
    check_admission_controllers
    check_audit_logging
    check_tls_certificates
    test_security_policies
    check_security_monitoring
    
    generate_security_report
    
    # Exit with error code if any checks failed
    exit $FAILURES
}

# Run main function
main "$@"