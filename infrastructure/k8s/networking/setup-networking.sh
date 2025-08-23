#!/bin/bash
# Network setup script for DePIN AI Compute cluster
# This script configures CNI, DNS, and network policies

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CALICO_VERSION="${CALICO_VERSION:-v3.26.1}"
CNI_PROVIDER="${CNI_PROVIDER:-calico}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if kubectl is available and cluster is reachable
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster. Please check your kubeconfig."
        exit 1
    fi
    
    # Check if cluster nodes are ready
    local ready_nodes=$(kubectl get nodes --no-headers | grep -c Ready || true)
    local total_nodes=$(kubectl get nodes --no-headers | wc -l)
    
    if [[ $ready_nodes -eq 0 ]]; then
        log_error "No ready nodes found in the cluster"
        exit 1
    fi
    
    log_info "Found $ready_nodes ready nodes out of $total_nodes total nodes"
    
    # Check if CNI is already installed
    if kubectl get pods -n kube-system | grep -E "(calico|flannel|weave|cilium)" &> /dev/null; then
        log_warn "CNI appears to already be installed"
        read -p "Continue anyway? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
}

install_calico() {
    log_info "Installing Calico CNI..."
    
    # Install Tigera Calico operator
    log_info "Installing Tigera operator..."
    kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/${CALICO_VERSION}/manifests/tigera-operator.yaml || true
    
    # Wait for operator to be ready
    log_info "Waiting for Tigera operator to be ready..."
    kubectl wait --for=condition=Available=True --timeout=300s deployment/tigera-operator -n tigera-operator
    
    # Apply custom Calico installation
    log_info "Applying Calico installation configuration..."
    kubectl apply -f "${SCRIPT_DIR}/calico-install.yaml"
    
    # Wait for Calico to be ready
    log_info "Waiting for Calico pods to be ready..."
    kubectl wait --for=condition=Ready --timeout=300s pod -l k8s-app=calico-node -n calico-system
    kubectl wait --for=condition=Ready --timeout=300s pod -l k8s-app=calico-kube-controllers -n calico-system
    
    # Apply custom Calico configuration
    log_info "Applying Calico custom configuration..."
    kubectl apply -f "${SCRIPT_DIR}/calico-config.yaml"
    
    log_info "Calico installation completed"
}

install_cilium() {
    log_info "Installing Cilium CNI..."
    
    # Install Cilium using Helm
    if ! command -v helm &> /dev/null; then
        log_error "Helm is required for Cilium installation but not found"
        exit 1
    fi
    
    # Add Cilium Helm repository
    helm repo add cilium https://helm.cilium.io/
    helm repo update
    
    # Install Cilium
    helm install cilium cilium/cilium \
        --version 1.14.2 \
        --namespace kube-system \
        --set cluster.name=depin-ai-compute \
        --set cluster.id=1 \
        --set ipam.mode=kubernetes \
        --set kubeProxyReplacement=strict \
        --set operator.replicas=2 \
        --set rollOutCiliumPods=true \
        --set tunnel=vxlan \
        --set ipv4NativeRoutingCIDR=10.244.0.0/16 \
        --set prometheus.enabled=true \
        --set operator.prometheus.enabled=true \
        --set hubble.enabled=true \
        --set hubble.metrics.enabled="{dns,drop,tcp,flow,icmp,http}" \
        --set hubble.relay.enabled=true \
        --set hubble.ui.enabled=true
    
    # Wait for Cilium to be ready
    log_info "Waiting for Cilium pods to be ready..."
    kubectl wait --for=condition=Ready --timeout=300s pod -l k8s-app=cilium -n kube-system
    
    log_info "Cilium installation completed"
}

configure_dns() {
    log_info "Configuring DNS services..."
    
    # Apply enhanced CoreDNS configuration
    kubectl apply -f "${SCRIPT_DIR}/coredns-config.yaml"
    
    # Restart CoreDNS pods to apply new configuration
    log_info "Restarting CoreDNS pods..."
    kubectl delete pod -l k8s-app=kube-dns -n kube-system
    
    # Wait for CoreDNS pods to be ready
    log_info "Waiting for CoreDNS pods to be ready..."
    sleep 10
    kubectl wait --for=condition=Ready --timeout=120s pod -l k8s-app=kube-dns -n kube-system
    
    log_info "DNS configuration completed"
}

setup_network_policies() {
    log_info "Setting up network policies..."
    
    # Create monitoring namespace if it doesn't exist
    kubectl create namespace monitoring --dry-run=client -o yaml | kubectl apply -f -
    kubectl label namespace monitoring name=monitoring --overwrite
    
    # Apply network policies from Calico config
    # (Network policies are included in calico-config.yaml)
    
    log_info "Network policies setup completed"
}

validate_networking() {
    log_info "Validating network configuration..."
    
    # Check CNI pods
    log_info "Checking CNI pod status..."
    if [[ "$CNI_PROVIDER" == "calico" ]]; then
        kubectl get pods -n calico-system
        
        # Check Calico node status
        kubectl exec -n calico-system ds/calico-node -- calicoctl node status || true
    elif [[ "$CNI_PROVIDER" == "cilium" ]]; then
        kubectl get pods -n kube-system -l k8s-app=cilium
        
        # Check Cilium status
        kubectl exec -n kube-system ds/cilium -- cilium status --brief || true
    fi
    
    # Check CoreDNS
    log_info "Checking CoreDNS status..."
    kubectl get pods -n kube-system -l k8s-app=kube-dns
    
    # Test DNS resolution
    log_info "Testing DNS resolution..."
    kubectl run dns-test --image=busybox --rm -it --restart=Never -- nslookup kubernetes.default.svc.cluster.local || true
    
    # Check network policies
    log_info "Checking network policies..."
    kubectl get networkpolicy --all-namespaces
    
    # Test pod-to-pod connectivity
    log_info "Testing pod-to-pod connectivity..."
    kubectl run network-test-1 --image=nginx --rm -it --restart=Never --labels="app=network-test" -- echo "Pod 1 created" || true
    kubectl run network-test-2 --image=busybox --rm -it --restart=Never -- wget -qO- network-test-1 || true
    
    log_info "Network validation completed"
}

cleanup() {
    log_info "Cleaning up test resources..."
    kubectl delete pod dns-test --ignore-not-found=true
    kubectl delete pod network-test-1 --ignore-not-found=true
    kubectl delete pod network-test-2 --ignore-not-found=true
}

show_usage() {
    cat <<EOF
Usage: $0 [OPTIONS]

Options:
    --cni-provider PROVIDER    CNI provider to install (calico|cilium) [default: calico]
    --calico-version VERSION   Calico version to install [default: v3.26.1]
    --skip-validation         Skip network validation tests
    --dns-only                Only configure DNS services
    --help                    Show this help message

Examples:
    # Install Calico CNI with default settings
    $0

    # Install Cilium CNI
    $0 --cni-provider cilium

    # Only configure DNS (assume CNI already installed)
    $0 --dns-only

    # Skip validation tests
    $0 --skip-validation
EOF
}

main() {
    local skip_validation=false
    local dns_only=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --cni-provider)
                CNI_PROVIDER="$2"
                shift 2
                ;;
            --calico-version)
                CALICO_VERSION="$2"
                shift 2
                ;;
            --skip-validation)
                skip_validation=true
                shift
                ;;
            --dns-only)
                dns_only=true
                shift
                ;;
            --help)
                show_usage
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done
    
    # Validate CNI provider
    if [[ "$CNI_PROVIDER" != "calico" && "$CNI_PROVIDER" != "cilium" ]]; then
        log_error "Unsupported CNI provider: $CNI_PROVIDER"
        exit 1
    fi
    
    log_info "Starting network setup for DePIN AI Compute cluster..."
    log_info "CNI Provider: $CNI_PROVIDER"
    
    check_prerequisites
    
    if [[ "$dns_only" != "true" ]]; then
        # Install CNI
        case "$CNI_PROVIDER" in
            "calico")
                install_calico
                ;;
            "cilium")
                install_cilium
                ;;
        esac
        
        setup_network_policies
    fi
    
    configure_dns
    
    if [[ "$skip_validation" != "true" ]]; then
        validate_networking
        cleanup
    fi
    
    log_info "Network setup completed successfully!"
    log_info ""
    log_info "Next steps:"
    log_info "1. Verify all nodes are ready: kubectl get nodes"
    log_info "2. Check CNI pods: kubectl get pods -n kube-system"
    log_info "3. Test DNS: kubectl run test --image=busybox --rm -it -- nslookup kubernetes"
}

# Trap to cleanup on script exit
trap cleanup EXIT

main "$@"