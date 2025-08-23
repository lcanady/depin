#!/bin/bash
# Bootstrap script for DePIN AI Compute Kubernetes cluster
# This script initializes a highly available Kubernetes cluster

set -euo pipefail

# Configuration
CLUSTER_NAME="${CLUSTER_NAME:-depin-ai-compute}"
K8S_VERSION="${K8S_VERSION:-1.28.0}"
CONTROL_PLANE_COUNT="${CONTROL_PLANE_COUNT:-3}"
WORKER_NODE_COUNT="${WORKER_NODE_COUNT:-3}"

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

check_requirements() {
    log_info "Checking system requirements..."
    
    # Check if running as root
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root"
        exit 1
    fi
    
    # Check required tools
    local required_tools=("kubeadm" "kubelet" "kubectl" "docker" "systemctl")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            log_error "$tool is required but not installed"
            exit 1
        fi
    done
    
    log_info "All requirements satisfied"
}

configure_system() {
    log_info "Configuring system settings..."
    
    # Disable swap
    swapoff -a
    sed -i '/ swap / s/^\(.*\)$/#\1/g' /etc/fstab
    
    # Configure iptables to see bridged traffic
    cat <<EOF > /etc/modules-load.d/k8s.conf
br_netfilter
overlay
EOF
    
    modprobe br_netfilter
    modprobe overlay
    
    cat <<EOF > /etc/sysctl.d/k8s.conf
net.bridge.bridge-nf-call-iptables = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward = 1
EOF
    
    sysctl --system
    
    # Configure containerd
    mkdir -p /etc/containerd
    containerd config default > /etc/containerd/config.toml
    sed -i 's/SystemdCgroup = false/SystemdCgroup = true/' /etc/containerd/config.toml
    
    systemctl restart containerd
    systemctl enable containerd
    
    # Start and enable kubelet
    systemctl enable --now kubelet
    
    log_info "System configuration completed"
}

init_first_control_plane() {
    log_info "Initializing first control plane node..."
    
    # Copy audit policy
    mkdir -p /etc/kubernetes
    cp "$(dirname "$0")/audit-policy.yaml" /etc/kubernetes/audit-policy.yaml
    
    # Initialize cluster
    kubeadm init --config="$(dirname "$0")/kubeadm-config.yaml" --upload-certs
    
    # Save join commands
    kubeadm token create --print-join-command > /tmp/kubeadm-join-worker.sh
    kubeadm init phase upload-certs --upload-certs 2>/dev/null | tail -1 > /tmp/certificate-key.txt
    
    # Configure kubectl for root
    mkdir -p /root/.kube
    cp -i /etc/kubernetes/admin.conf /root/.kube/config
    chown root:root /root/.kube/config
    
    log_info "First control plane node initialized"
    log_info "Worker join command saved to /tmp/kubeadm-join-worker.sh"
    log_info "Certificate key saved to /tmp/certificate-key.txt"
}

join_control_plane() {
    log_info "Joining additional control plane node..."
    
    if [[ -z "${JOIN_COMMAND:-}" ]]; then
        log_error "JOIN_COMMAND environment variable is required"
        exit 1
    fi
    
    if [[ -z "${CERTIFICATE_KEY:-}" ]]; then
        log_error "CERTIFICATE_KEY environment variable is required"
        exit 1
    fi
    
    # Copy audit policy
    mkdir -p /etc/kubernetes
    cp "$(dirname "$0")/audit-policy.yaml" /etc/kubernetes/audit-policy.yaml
    
    # Join cluster as control plane
    eval "$JOIN_COMMAND --control-plane --certificate-key $CERTIFICATE_KEY"
    
    # Configure kubectl
    mkdir -p /root/.kube
    cp -i /etc/kubernetes/admin.conf /root/.kube/config
    chown root:root /root/.kube/config
    
    log_info "Control plane node joined successfully"
}

join_worker() {
    log_info "Joining worker node..."
    
    if [[ -z "${JOIN_COMMAND:-}" ]]; then
        log_error "JOIN_COMMAND environment variable is required"
        exit 1
    fi
    
    # Join cluster as worker
    eval "$JOIN_COMMAND"
    
    log_info "Worker node joined successfully"
}

validate_cluster() {
    log_info "Validating cluster health..."
    
    # Wait for nodes to be ready
    local max_attempts=30
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        if kubectl get nodes --no-headers | grep -v NotReady >/dev/null 2>&1; then
            local ready_nodes=$(kubectl get nodes --no-headers | grep -c Ready || true)
            local total_nodes=$(kubectl get nodes --no-headers | wc -l)
            
            if [[ $ready_nodes -eq $total_nodes ]] && [[ $ready_nodes -gt 0 ]]; then
                log_info "All $ready_nodes nodes are ready"
                break
            fi
        fi
        
        log_warn "Waiting for nodes to be ready... (attempt $attempt/$max_attempts)"
        sleep 10
        ((attempt++))
    done
    
    if [[ $attempt -gt $max_attempts ]]; then
        log_error "Cluster validation failed - nodes are not ready"
        return 1
    fi
    
    # Check system pods
    kubectl get pods -n kube-system
    
    # Check cluster info
    kubectl cluster-info
    
    log_info "Cluster validation completed successfully"
}

show_usage() {
    cat <<EOF
Usage: $0 [COMMAND]

Commands:
    init-first      Initialize the first control plane node
    join-control    Join additional control plane node (requires JOIN_COMMAND and CERTIFICATE_KEY env vars)
    join-worker     Join worker node (requires JOIN_COMMAND env var)
    validate        Validate cluster health

Environment variables:
    CLUSTER_NAME            Name of the cluster (default: depin-ai-compute)
    K8S_VERSION            Kubernetes version (default: 1.28.0)
    CONTROL_PLANE_COUNT    Number of control plane nodes (default: 3)
    WORKER_NODE_COUNT      Number of worker nodes (default: 3)
    JOIN_COMMAND           Join command from kubeadm (for join operations)
    CERTIFICATE_KEY        Certificate key for control plane join

Examples:
    # Initialize first control plane
    $0 init-first

    # Join additional control plane
    JOIN_COMMAND="kubeadm join ..." CERTIFICATE_KEY="abc123..." $0 join-control

    # Join worker node
    JOIN_COMMAND="kubeadm join ..." $0 join-worker

    # Validate cluster
    $0 validate
EOF
}

main() {
    case "${1:-}" in
        "init-first")
            check_requirements
            configure_system
            init_first_control_plane
            ;;
        "join-control")
            check_requirements
            configure_system
            join_control_plane
            ;;
        "join-worker")
            check_requirements
            configure_system
            join_worker
            ;;
        "validate")
            validate_cluster
            ;;
        *)
            show_usage
            exit 1
            ;;
    esac
}

main "$@"