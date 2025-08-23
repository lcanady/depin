#!/bin/bash

# DePIN AI Compute - Essential Operators Health Validation Script
# This script validates the health and functionality of all essential operators

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="${SCRIPT_DIR}/health-check-$(date +%Y%m%d-%H%M%S).log"
TIMEOUT=300
FAILED_CHECKS=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log() {
    echo -e "[$(date +'%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

log_info() {
    log "${BLUE}[INFO]${NC} $*"
}

log_success() {
    log "${GREEN}[SUCCESS]${NC} $*"
}

log_warning() {
    log "${YELLOW}[WARNING]${NC} $*"
}

log_error() {
    log "${RED}[ERROR]${NC} $*"
    ((FAILED_CHECKS++))
}

# Check if command exists
check_command() {
    if ! command -v "$1" &> /dev/null; then
        log_error "Required command '$1' not found"
        exit 1
    fi
}

# Wait for pod to be ready
wait_for_pod_ready() {
    local namespace=$1
    local selector=$2
    local timeout=${3:-$TIMEOUT}
    
    log_info "Waiting for pods with selector '$selector' in namespace '$namespace' to be ready..."
    
    if kubectl wait --namespace="$namespace" \
        --for=condition=ready pod \
        --selector="$selector" \
        --timeout="${timeout}s" &>/dev/null; then
        log_success "Pods with selector '$selector' are ready"
        return 0
    else
        log_error "Pods with selector '$selector' failed to become ready within ${timeout}s"
        return 1
    fi
}

# Check service endpoint
check_service_endpoint() {
    local namespace=$1
    local service=$2
    local port=$3
    local path=${4:-"/"}
    
    log_info "Checking service endpoint: $service.$namespace:$port$path"
    
    # Use port-forward to test the endpoint
    kubectl port-forward -n "$namespace" "svc/$service" "8080:$port" &
    local pf_pid=$!
    sleep 5
    
    if curl -f -s "http://localhost:8080$path" &>/dev/null; then
        log_success "Service endpoint $service.$namespace:$port$path is accessible"
        kill $pf_pid 2>/dev/null || true
        return 0
    else
        log_error "Service endpoint $service.$namespace:$port$path is not accessible"
        kill $pf_pid 2>/dev/null || true
        return 1
    fi
}

# Check certificate readiness
check_certificate() {
    local namespace=$1
    local cert_name=$2
    
    log_info "Checking certificate '$cert_name' in namespace '$namespace'"
    
    if kubectl get certificate -n "$namespace" "$cert_name" -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' | grep -q "True"; then
        log_success "Certificate '$cert_name' is ready"
        return 0
    else
        log_warning "Certificate '$cert_name' is not ready yet"
        return 1
    fi
}

# Main validation function
main() {
    log_info "Starting DePIN AI Compute Essential Operators Health Validation"
    log_info "Log file: $LOG_FILE"
    
    # Check required commands
    check_command kubectl
    check_command curl
    
    # Check cluster connectivity
    log_info "Checking Kubernetes cluster connectivity..."
    if ! kubectl cluster-info &>/dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    log_success "Kubernetes cluster is accessible"
    
    # 1. Check Monitoring Stack
    log_info "=== Validating Monitoring Stack ==="
    
    # Check monitoring namespace
    if kubectl get namespace monitoring &>/dev/null; then
        log_success "Monitoring namespace exists"
    else
        log_error "Monitoring namespace does not exist"
    fi
    
    # Check Prometheus
    wait_for_pod_ready "monitoring" "app=prometheus,component=server" || true
    check_service_endpoint "monitoring" "prometheus" "9090" "/-/healthy" || true
    
    # Check Grafana
    wait_for_pod_ready "monitoring" "app=grafana,component=server" || true
    check_service_endpoint "monitoring" "grafana" "3000" "/api/health" || true
    
    # Check AlertManager
    wait_for_pod_ready "monitoring" "app=alertmanager,component=server" || true
    check_service_endpoint "monitoring" "alertmanager" "9093" "/-/healthy" || true
    
    # 2. Check Logging Stack
    log_info "=== Validating Logging Stack ==="
    
    # Check logging namespace
    if kubectl get namespace logging &>/dev/null; then
        log_success "Logging namespace exists"
    else
        log_error "Logging namespace does not exist"
    fi
    
    # Check Elasticsearch
    wait_for_pod_ready "logging" "app=elasticsearch,component=server" || true
    check_service_endpoint "logging" "elasticsearch" "9200" "/_cluster/health" || true
    
    # Check Fluent Bit
    wait_for_pod_ready "logging" "app=fluent-bit,component=log-collector" || true
    check_service_endpoint "logging" "fluent-bit" "2020" "/api/v1/health" || true
    
    # Check Kibana
    wait_for_pod_ready "logging" "app=kibana,component=server" || true
    check_service_endpoint "logging" "kibana" "5601" "/api/status" || true
    
    # 3. Check Ingress Controller
    log_info "=== Validating Ingress Controller ==="
    
    # Check ingress-nginx namespace
    if kubectl get namespace ingress-nginx &>/dev/null; then
        log_success "ingress-nginx namespace exists"
    else
        log_error "ingress-nginx namespace does not exist"
    fi
    
    # Check NGINX Ingress Controller
    wait_for_pod_ready "ingress-nginx" "app=ingress-nginx,component=controller" || true
    check_service_endpoint "ingress-nginx" "nginx-ingress-controller-metrics" "10254" "/healthz" || true
    
    # 4. Check Cert-Manager
    log_info "=== Validating Cert-Manager ==="
    
    # Check cert-manager namespace
    if kubectl get namespace cert-manager &>/dev/null; then
        log_success "cert-manager namespace exists"
    else
        log_error "cert-manager namespace does not exist"
    fi
    
    # Check cert-manager components
    wait_for_pod_ready "cert-manager" "app=cert-manager,component=controller" || true
    wait_for_pod_ready "cert-manager" "app=cainjector,component=cainjector" || true
    wait_for_pod_ready "cert-manager" "app=webhook,component=webhook" || true
    
    # Check cluster issuers
    if kubectl get clusterissuer letsencrypt-staging &>/dev/null; then
        log_success "Let's Encrypt staging issuer exists"
    else
        log_error "Let's Encrypt staging issuer does not exist"
    fi
    
    if kubectl get clusterissuer letsencrypt-prod &>/dev/null; then
        log_success "Let's Encrypt production issuer exists"
    else
        log_error "Let's Encrypt production issuer does not exist"
    fi
    
    # Check certificates
    check_certificate "ingress-nginx" "depin-wildcard-cert" || true
    check_certificate "monitoring" "monitoring-tls-cert" || true
    check_certificate "logging" "logging-tls-cert" || true
    
    # 5. Integration Tests
    log_info "=== Running Integration Tests ==="
    
    # Test ingress connectivity
    log_info "Testing ingress routes..."
    if kubectl get ingress -A --no-headers | grep -q .; then
        log_success "Ingress resources are configured"
        kubectl get ingress -A --no-headers | while read -r namespace name class hosts address ports age; do
            log_info "  - $namespace/$name: $hosts"
        done
    else
        log_warning "No ingress resources found"
    fi
    
    # Test metrics collection
    log_info "Testing metrics collection..."
    kubectl port-forward -n monitoring svc/prometheus 9090:9090 &
    local pf_pid=$!
    sleep 5
    
    if curl -s "http://localhost:9090/api/v1/targets" | grep -q '"health":"up"'; then
        log_success "Prometheus is collecting metrics from targets"
    else
        log_warning "Prometheus may not be collecting metrics properly"
    fi
    kill $pf_pid 2>/dev/null || true
    
    # Test log aggregation
    log_info "Testing log aggregation..."
    if kubectl get daemonset -n logging fluent-bit -o jsonpath='{.status.numberReady}' | grep -q '[1-9]'; then
        log_success "Fluent Bit is running on nodes and collecting logs"
    else
        log_warning "Fluent Bit may not be collecting logs properly"
    fi
    
    # 6. Security Validation
    log_info "=== Validating Security Configuration ==="
    
    # Check pod security contexts
    log_info "Checking pod security contexts..."
    if kubectl get pods -A -o jsonpath='{.items[*].spec.securityContext.runAsNonRoot}' | grep -q "true"; then
        log_success "Pods are configured with non-root security context"
    else
        log_warning "Some pods may be running as root"
    fi
    
    # Check network policies
    if kubectl get networkpolicy -A --no-headers | grep -q .; then
        log_success "Network policies are configured"
    else
        log_warning "No network policies found"
    fi
    
    # Final report
    log_info "=== Health Validation Summary ==="
    if [ $FAILED_CHECKS -eq 0 ]; then
        log_success "All essential operators are healthy and operational!"
        log_success "DePIN AI Compute cluster is ready for production workloads"
    elif [ $FAILED_CHECKS -lt 5 ]; then
        log_warning "Essential operators are mostly healthy with $FAILED_CHECKS minor issues"
        log_warning "Review the warnings above and address any critical issues"
    else
        log_error "Essential operators have $FAILED_CHECKS issues that need attention"
        log_error "Please resolve the errors before proceeding with production workloads"
    fi
    
    log_info "Health validation complete. Log saved to: $LOG_FILE"
    return $FAILED_CHECKS
}

# Script execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
    exit_code=$?
    exit $exit_code
fi