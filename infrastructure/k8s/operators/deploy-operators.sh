#!/bin/bash

# DePIN AI Compute - Essential Operators Deployment Script
# This script deploys all essential operators in the correct order

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
K8S_DIR="$(dirname "$SCRIPT_DIR")"
LOG_FILE="${SCRIPT_DIR}/deployment-$(date +%Y%m%d-%H%M%S).log"
TIMEOUT=300

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
}

# Check if command exists
check_command() {
    if ! command -v "$1" &> /dev/null; then
        log_error "Required command '$1' not found"
        exit 1
    fi
}

# Apply Kubernetes manifests
apply_manifest() {
    local manifest_file=$1
    local description=${2:-"$manifest_file"}
    
    if [ ! -f "$manifest_file" ]; then
        log_error "Manifest file not found: $manifest_file"
        return 1
    fi
    
    log_info "Applying $description: $manifest_file"
    
    if kubectl apply -f "$manifest_file"; then
        log_success "Successfully applied $description"
        return 0
    else
        log_error "Failed to apply $description"
        return 1
    fi
}

# Wait for deployment to be ready
wait_for_deployment() {
    local namespace=$1
    local deployment=$2
    local timeout=${3:-$TIMEOUT}
    
    log_info "Waiting for deployment '$deployment' in namespace '$namespace' to be ready..."
    
    if kubectl wait --namespace="$namespace" \
        --for=condition=available deployment "$deployment" \
        --timeout="${timeout}s"; then
        log_success "Deployment '$deployment' is ready"
        return 0
    else
        log_error "Deployment '$deployment' failed to become ready within ${timeout}s"
        return 1
    fi
}

# Wait for daemonset to be ready
wait_for_daemonset() {
    local namespace=$1
    local daemonset=$2
    local timeout=${3:-$TIMEOUT}
    
    log_info "Waiting for daemonset '$daemonset' in namespace '$namespace' to be ready..."
    
    if kubectl wait --namespace="$namespace" \
        --for=condition=ready pod \
        --selector="app=$daemonset" \
        --timeout="${timeout}s"; then
        log_success "DaemonSet '$daemonset' is ready"
        return 0
    else
        log_error "DaemonSet '$daemonset' failed to become ready within ${timeout}s"
        return 1
    fi
}

# Wait for statefulset to be ready
wait_for_statefulset() {
    local namespace=$1
    local statefulset=$2
    local timeout=${3:-$TIMEOUT}
    
    log_info "Waiting for statefulset '$statefulset' in namespace '$namespace' to be ready..."
    
    if kubectl wait --namespace="$namespace" \
        --for=condition=ready pod \
        --selector="app=$statefulset" \
        --timeout="${timeout}s"; then
        log_success "StatefulSet '$statefulset' is ready"
        return 0
    else
        log_error "StatefulSet '$statefulset' failed to become ready within ${timeout}s"
        return 1
    fi
}

# Deploy cert-manager CRDs
deploy_cert_manager_crds() {
    log_info "Deploying cert-manager CRDs..."
    
    if kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.12.0/cert-manager.crds.yaml; then
        log_success "cert-manager CRDs deployed successfully"
        return 0
    else
        log_error "Failed to deploy cert-manager CRDs"
        return 1
    fi
}

# Main deployment function
main() {
    log_info "Starting DePIN AI Compute Essential Operators Deployment"
    log_info "Log file: $LOG_FILE"
    
    # Check required commands
    check_command kubectl
    
    # Check cluster connectivity
    log_info "Checking Kubernetes cluster connectivity..."
    if ! kubectl cluster-info &>/dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    log_success "Kubernetes cluster is accessible"
    
    # Phase 1: Deploy Namespaces and Service Accounts
    log_info "=== Phase 1: Deploying Namespaces and Service Accounts ==="
    
    apply_manifest "${K8S_DIR}/monitoring/namespace.yaml" "Monitoring namespace"
    apply_manifest "${K8S_DIR}/monitoring/logging-namespace.yaml" "Logging namespace"
    apply_manifest "${SCRIPT_DIR}/ingress-controller-namespace.yaml" "Ingress controller namespace"
    apply_manifest "${SCRIPT_DIR}/cert-manager-namespace.yaml" "cert-manager namespace"
    
    # Deploy service accounts and RBAC
    apply_manifest "${SCRIPT_DIR}/service-accounts.yaml" "Service accounts and RBAC"
    
    sleep 10
    
    # Phase 2: Deploy cert-manager
    log_info "=== Phase 2: Deploying cert-manager ==="
    
    deploy_cert_manager_crds
    apply_manifest "${SCRIPT_DIR}/cert-manager-deployment.yaml" "cert-manager deployments"
    
    # Wait for cert-manager to be ready
    wait_for_deployment "cert-manager" "cert-manager"
    wait_for_deployment "cert-manager" "cert-manager-cainjector"
    wait_for_deployment "cert-manager" "cert-manager-webhook"
    
    sleep 30  # Allow webhook to be fully ready
    
    # Deploy cert-manager issuers and certificates
    apply_manifest "${SCRIPT_DIR}/cert-manager-issuers.yaml" "cert-manager issuers and certificates"
    
    # Phase 3: Deploy Ingress Controller
    log_info "=== Phase 3: Deploying Ingress Controller ==="
    
    apply_manifest "${SCRIPT_DIR}/nginx-ingress-controller.yaml" "NGINX ingress controller"
    
    # Wait for ingress controller to be ready
    wait_for_daemonset "ingress-nginx" "ingress-nginx"
    
    # Phase 4: Deploy Monitoring Stack
    log_info "=== Phase 4: Deploying Monitoring Stack ==="
    
    # Deploy Prometheus
    apply_manifest "${K8S_DIR}/monitoring/prometheus-config.yaml" "Prometheus configuration"
    apply_manifest "${K8S_DIR}/monitoring/prometheus-deployment.yaml" "Prometheus deployment"
    wait_for_deployment "monitoring" "prometheus"
    
    # Deploy AlertManager
    apply_manifest "${K8S_DIR}/monitoring/alertmanager-config.yaml" "AlertManager configuration"
    apply_manifest "${K8S_DIR}/monitoring/alertmanager-deployment.yaml" "AlertManager deployment"
    wait_for_deployment "monitoring" "alertmanager"
    
    # Deploy Grafana
    apply_manifest "${K8S_DIR}/monitoring/grafana-config.yaml" "Grafana configuration"
    apply_manifest "${K8S_DIR}/monitoring/grafana-deployment.yaml" "Grafana deployment"
    wait_for_deployment "monitoring" "grafana"
    
    # Phase 5: Deploy Logging Stack
    log_info "=== Phase 5: Deploying Logging Stack ==="
    
    # Deploy Elasticsearch
    apply_manifest "${K8S_DIR}/monitoring/elasticsearch-deployment.yaml" "Elasticsearch deployment"
    wait_for_statefulset "logging" "elasticsearch"
    
    sleep 60  # Allow Elasticsearch cluster to form
    
    # Deploy Fluent Bit
    apply_manifest "${K8S_DIR}/monitoring/fluent-bit-config.yaml" "Fluent Bit configuration"
    apply_manifest "${K8S_DIR}/monitoring/fluent-bit-deployment.yaml" "Fluent Bit deployment"
    wait_for_daemonset "logging" "fluent-bit"
    
    # Deploy Kibana
    apply_manifest "${K8S_DIR}/monitoring/kibana-deployment.yaml" "Kibana deployment"
    wait_for_deployment "logging" "kibana"
    
    # Phase 6: Deploy Ingress Resources
    log_info "=== Phase 6: Deploying Ingress Resources ==="
    
    sleep 30  # Allow certificates to be issued
    apply_manifest "${SCRIPT_DIR}/monitoring-ingress.yaml" "Monitoring and logging ingress"
    
    # Phase 7: Validation
    log_info "=== Phase 7: Running Health Validation ==="
    
    sleep 60  # Allow all services to stabilize
    
    if [ -x "${SCRIPT_DIR}/health-validation.sh" ]; then
        "${SCRIPT_DIR}/health-validation.sh"
        health_status=$?
        
        if [ $health_status -eq 0 ]; then
            log_success "All operators deployed and healthy!"
        elif [ $health_status -lt 5 ]; then
            log_warning "Deployment completed with minor issues (exit code: $health_status)"
        else
            log_error "Deployment completed but operators have issues (exit code: $health_status)"
        fi
    else
        log_warning "Health validation script not found or not executable"
    fi
    
    # Final summary
    log_info "=== Deployment Summary ==="
    log_success "Essential operators deployment completed!"
    log_info "Deployed components:"
    log_info "  ✓ cert-manager (TLS certificate automation)"
    log_info "  ✓ NGINX Ingress Controller (external access)"
    log_info "  ✓ Prometheus (metrics collection)"
    log_info "  ✓ AlertManager (alert routing)"
    log_info "  ✓ Grafana (metrics visualization)"
    log_info "  ✓ Elasticsearch (log storage)"
    log_info "  ✓ Fluent Bit (log collection)"
    log_info "  ✓ Kibana (log visualization)"
    
    log_info "Access URLs (after DNS configuration):"
    log_info "  - Grafana: https://grafana.depin-ai-compute.local/"
    log_info "  - Prometheus: https://prometheus.depin-ai-compute.local/"
    log_info "  - AlertManager: https://alertmanager.depin-ai-compute.local/"
    log_info "  - Kibana: https://kibana.depin-ai-compute.local/"
    
    log_info "Deployment complete. Log saved to: $LOG_FILE"
    
    return 0
}

# Script execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
    exit_code=$?
    exit $exit_code
fi