#!/bin/bash

# DePIN AI Compute - Grafana Deployment Script
# This script deploys Grafana with monitoring dashboards for the DePIN AI compute platform

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="monitoring"
CONTEXT="${KUBECTL_CONTEXT:-$(kubectl config current-context)}"
DRY_RUN="${DRY_RUN:-false}"

log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}"
    exit 1
}

check_prerequisites() {
    log "Checking prerequisites..."
    
    # Check if kubectl is available
    if ! command -v kubectl &> /dev/null; then
        error "kubectl is not installed or not in PATH"
    fi
    
    # Check if kustomize is available
    if ! command -v kustomize &> /dev/null; then
        error "kustomize is not installed or not in PATH"
    fi
    
    # Check cluster connectivity
    if ! kubectl cluster-info &> /dev/null; then
        error "Cannot connect to Kubernetes cluster"
    fi
    
    # Check if Prometheus is already deployed (dependency)
    if ! kubectl get deployment prometheus-server -n monitoring &> /dev/null; then
        warn "Prometheus server not found in monitoring namespace. Grafana datasources may fail until Prometheus is deployed."
    fi
    
    log "Prerequisites check completed"
}

generate_secrets() {
    log "Generating secure secrets..."
    
    # Generate admin password if not provided
    if [ -z "${GRAFANA_ADMIN_PASSWORD:-}" ]; then
        GRAFANA_ADMIN_PASSWORD=$(openssl rand -base64 32)
        log "Generated admin password: $GRAFANA_ADMIN_PASSWORD"
        log "Please save this password securely!"
    fi
    
    # Generate secret key
    GRAFANA_SECRET_KEY=$(openssl rand -base64 32 | base64 -w 0)
    
    # Update secret file with generated values
    kubectl create secret generic grafana-secret \
        --namespace="$NAMESPACE" \
        --from-literal=admin-password="$GRAFANA_ADMIN_PASSWORD" \
        --from-literal=secret-key="$GRAFANA_SECRET_KEY" \
        --dry-run=client -o yaml > /tmp/grafana-secret.yaml
        
    if [ "$DRY_RUN" = "false" ]; then
        kubectl apply -f /tmp/grafana-secret.yaml
    fi
}

deploy_grafana() {
    log "Deploying Grafana to cluster context: $CONTEXT"
    
    # Create namespace if it doesn't exist
    if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
        log "Creating namespace: $NAMESPACE"
        kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    fi
    
    # Deploy using kustomize
    local kustomize_dir="../../../k8s/monitoring/grafana"
    if [ ! -f "$kustomize_dir/kustomization.yaml" ]; then
        error "Kustomization file not found at $kustomize_dir/kustomization.yaml"
    fi
    
    if [ "$DRY_RUN" = "true" ]; then
        log "DRY RUN - Would deploy the following resources:"
        kustomize build "$kustomize_dir"
    else
        log "Applying Grafana manifests..."
        kustomize build "$kustomize_dir" | kubectl apply -f -
    fi
}

wait_for_deployment() {
    if [ "$DRY_RUN" = "true" ]; then
        return
    fi
    
    log "Waiting for Grafana deployment to be ready..."
    
    # Wait for deployment to be available
    if ! kubectl wait --for=condition=available --timeout=600s deployment/grafana -n "$NAMESPACE"; then
        error "Grafana deployment failed to become ready within 10 minutes"
    fi
    
    # Wait for all pods to be ready
    if ! kubectl wait --for=condition=ready --timeout=300s pods -l app.kubernetes.io/name=grafana -n "$NAMESPACE"; then
        error "Grafana pods failed to become ready within 5 minutes"
    fi
    
    log "Grafana deployment is ready!"
}

verify_deployment() {
    if [ "$DRY_RUN" = "true" ]; then
        return
    fi
    
    log "Verifying Grafana deployment..."
    
    # Check if service is accessible
    local service_status
    service_status=$(kubectl get service grafana -n "$NAMESPACE" -o jsonpath='{.status}')
    if [ -z "$service_status" ]; then
        error "Grafana service is not accessible"
    fi
    
    # Check if pods are running
    local ready_pods
    ready_pods=$(kubectl get pods -l app.kubernetes.io/name=grafana -n "$NAMESPACE" -o jsonpath='{.items[*].status.phase}' | grep -c "Running" || echo "0")
    if [ "$ready_pods" -eq 0 ]; then
        error "No Grafana pods are running"
    fi
    
    log "Deployment verification completed successfully"
    log "Ready pods: $ready_pods"
}

show_access_info() {
    if [ "$DRY_RUN" = "true" ]; then
        return
    fi
    
    log "Grafana Access Information:"
    echo "=============================="
    
    # Service information
    local service_ip
    service_ip=$(kubectl get service grafana -n "$NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "pending")
    
    echo "Service Type: $(kubectl get service grafana -n "$NAMESPACE" -o jsonpath='{.spec.type}')"
    echo "Service IP: $service_ip"
    echo "Port: $(kubectl get service grafana -n "$NAMESPACE" -o jsonpath='{.spec.ports[0].port}')"
    
    # Ingress information
    if kubectl get ingress grafana -n "$NAMESPACE" &> /dev/null; then
        local ingress_host
        ingress_host=$(kubectl get ingress grafana -n "$NAMESPACE" -o jsonpath='{.spec.rules[0].host}')
        echo "Ingress URL: https://$ingress_host"
    fi
    
    # Port forward command
    echo ""
    echo "To access Grafana locally, run:"
    echo "kubectl port-forward -n $NAMESPACE service/grafana 3000:3000"
    echo "Then visit: http://localhost:3000"
    echo ""
    echo "Default credentials:"
    echo "Username: admin"
    echo "Password: (check the grafana-secret in the $NAMESPACE namespace)"
    echo ""
    echo "To get the password:"
    echo "kubectl get secret grafana-secret -n $NAMESPACE -o jsonpath='{.data.admin-password}' | base64 -d && echo"
}

main() {
    log "Starting Grafana deployment for DePIN AI Compute platform"
    
    check_prerequisites
    generate_secrets
    deploy_grafana
    wait_for_deployment
    verify_deployment
    show_access_info
    
    log "Grafana deployment completed successfully!"
    log "Grafana is now ready to visualize your DePIN AI compute metrics"
}

# Handle script arguments
case "${1:-deploy}" in
    "deploy")
        main
        ;;
    "destroy")
        log "Destroying Grafana deployment..."
        if kubectl get namespace "$NAMESPACE" &> /dev/null; then
            kustomize build "../../../k8s/monitoring/grafana" | kubectl delete -f - || true
        fi
        log "Grafana deployment destroyed"
        ;;
    "status")
        kubectl get all -l app.kubernetes.io/name=grafana -n "$NAMESPACE"
        ;;
    *)
        echo "Usage: $0 {deploy|destroy|status}"
        echo "  deploy  - Deploy Grafana (default)"
        echo "  destroy - Remove Grafana deployment"
        echo "  status  - Show deployment status"
        exit 1
        ;;
esac