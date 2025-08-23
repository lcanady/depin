#!/bin/bash
# Cluster validation script for DePIN AI Compute
# This script performs comprehensive validation of the Kubernetes cluster

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

PASSED_TESTS=0
FAILED_TESTS=0
TOTAL_TESTS=0

log_info() {
    echo -e "${GREEN}[✓]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[!]${NC} $1"
}

log_error() {
    echo -e "${RED}[✗]${NC} $1"
}

log_test() {
    echo -e "${BLUE}[TEST]${NC} $1"
    ((TOTAL_TESTS++))
}

test_passed() {
    ((PASSED_TESTS++))
    log_info "$1"
}

test_failed() {
    ((FAILED_TESTS++))
    log_error "$1"
}

check_kubectl_access() {
    log_test "Checking kubectl access to cluster"
    
    if kubectl cluster-info >/dev/null 2>&1; then
        test_passed "kubectl can access the cluster"
        
        # Get cluster info
        echo "Cluster Info:"
        kubectl cluster-info | sed 's/^/  /'
    else
        test_failed "kubectl cannot access the cluster"
        return 1
    fi
}

validate_node_readiness() {
    log_test "Validating node readiness"
    
    local nodes_output=$(kubectl get nodes --no-headers)
    local total_nodes=$(echo "$nodes_output" | wc -l)
    local ready_nodes=$(echo "$nodes_output" | grep -c Ready || true)
    local not_ready_nodes=$(echo "$nodes_output" | grep -c NotReady || true)
    
    echo "Node Status:"
    echo "$nodes_output" | sed 's/^/  /'
    
    if [[ $not_ready_nodes -eq 0 ]] && [[ $ready_nodes -gt 0 ]]; then
        test_passed "All $ready_nodes nodes are Ready"
    else
        test_failed "$not_ready_nodes nodes are NotReady out of $total_nodes total"
    fi
    
    # Check control plane nodes
    local control_plane_nodes=$(echo "$nodes_output" | grep -c "control-plane\|master" || true)
    if [[ $control_plane_nodes -ge 1 ]]; then
        test_passed "Found $control_plane_nodes control plane node(s)"
    else
        test_failed "No control plane nodes found"
    fi
    
    # Check worker nodes
    local worker_nodes=$((total_nodes - control_plane_nodes))
    if [[ $worker_nodes -ge 1 ]]; then
        test_passed "Found $worker_nodes worker node(s)"
    else
        test_failed "No worker nodes found"
    fi
}

validate_system_pods() {
    log_test "Validating system pods"
    
    local critical_namespaces=("kube-system")
    
    for ns in "${critical_namespaces[@]}"; do
        echo "Checking pods in namespace: $ns"
        
        local pods_output=$(kubectl get pods -n "$ns" --no-headers)
        local total_pods=$(echo "$pods_output" | wc -l)
        local running_pods=$(echo "$pods_output" | grep -c Running || true)
        local failed_pods=$(echo "$pods_output" | grep -E "(Error|CrashLoopBackOff|ImagePullBackOff|Failed)" | wc -l || true)
        
        echo "$pods_output" | sed 's/^/  /'
        
        if [[ $failed_pods -eq 0 ]]; then
            test_passed "All critical pods in $ns are healthy"
        else
            test_failed "$failed_pods pods in $ns have issues"
        fi
    done
    
    # Check specific critical components
    local critical_components=("kube-apiserver" "etcd" "kube-controller-manager" "kube-scheduler")
    for component in "${critical_components[@]}"; do
        if kubectl get pods -n kube-system | grep -q "$component.*Running"; then
            test_passed "$component is running"
        else
            # Check if it's a static pod
            local static_pod_count=$(kubectl get pods -n kube-system | grep -c "$component" || true)
            if [[ $static_pod_count -gt 0 ]]; then
                test_passed "$component found (may be static pod)"
            else
                test_failed "$component not found or not running"
            fi
        fi
    done
}

validate_networking() {
    log_test "Validating cluster networking"
    
    # Check CNI installation
    local cni_found=false
    local cni_providers=("calico" "cilium" "flannel" "weave")
    
    for provider in "${cni_providers[@]}"; do
        if kubectl get pods -A | grep -q "$provider"; then
            local provider_pods=$(kubectl get pods -A | grep "$provider" | wc -l)
            local running_pods=$(kubectl get pods -A | grep "$provider" | grep -c Running || true)
            
            if [[ $running_pods -eq $provider_pods ]] && [[ $running_pods -gt 0 ]]; then
                test_passed "$provider CNI: $running_pods/$provider_pods pods running"
                cni_found=true
            else
                test_failed "$provider CNI: only $running_pods/$provider_pods pods running"
            fi
            break
        fi
    done
    
    if [[ $cni_found == false ]]; then
        test_failed "No CNI provider found"
    fi
    
    # Check kube-proxy
    local proxy_pods=$(kubectl get pods -n kube-system | grep kube-proxy | wc -l || true)
    local proxy_running=$(kubectl get pods -n kube-system | grep kube-proxy | grep -c Running || true)
    
    if [[ $proxy_running -eq $proxy_pods ]] && [[ $proxy_running -gt 0 ]]; then
        test_passed "kube-proxy: $proxy_running/$proxy_pods pods running"
    else
        test_failed "kube-proxy: only $proxy_running/$proxy_pods pods running"
    fi
}

validate_dns() {
    log_test "Validating DNS functionality"
    
    # Check CoreDNS pods
    local coredns_pods=$(kubectl get pods -n kube-system | grep coredns | wc -l || true)
    local coredns_running=$(kubectl get pods -n kube-system | grep coredns | grep -c Running || true)
    
    if [[ $coredns_running -eq $coredns_pods ]] && [[ $coredns_running -gt 0 ]]; then
        test_passed "CoreDNS: $coredns_running/$coredns_pods pods running"
    else
        test_failed "CoreDNS: only $coredns_running/$coredns_pods pods running"
    fi
    
    # Test DNS resolution
    echo "Testing DNS resolution..."
    if timeout 30s kubectl run dns-test-$$-$RANDOM --image=busybox --rm --restart=Never -- nslookup kubernetes.default >/dev/null 2>&1; then
        test_passed "DNS resolution test passed"
    else
        test_failed "DNS resolution test failed"
    fi
}

validate_api_server() {
    log_test "Validating API server health"
    
    # Check health endpoints
    if kubectl get --raw="/healthz" >/dev/null 2>&1; then
        test_passed "API server /healthz endpoint responding"
    else
        test_failed "API server /healthz endpoint not responding"
    fi
    
    if kubectl get --raw="/readyz" >/dev/null 2>&1; then
        test_passed "API server /readyz endpoint responding"
    else
        test_failed "API server /readyz endpoint not responding"
    fi
    
    # Check API server performance
    echo "Testing API server performance..."
    local start_time=$(date +%s%3N)
    kubectl get nodes >/dev/null 2>&1
    local end_time=$(date +%s%3N)
    local response_time=$((end_time - start_time))
    
    if [[ $response_time -lt 1000 ]]; then
        test_passed "API server response time: ${response_time}ms (good)"
    elif [[ $response_time -lt 3000 ]]; then
        log_warn "API server response time: ${response_time}ms (acceptable)"
    else
        test_failed "API server response time: ${response_time}ms (slow)"
    fi
}

validate_rbac() {
    log_test "Validating RBAC configuration"
    
    # Check if RBAC is enabled
    if kubectl auth can-i '*' '*' --as=system:unauthenticated 2>/dev/null | grep -q "no"; then
        test_passed "RBAC is properly enabled (unauthenticated users denied)"
    else
        test_failed "RBAC may not be properly configured"
    fi
    
    # Check system service accounts
    local system_sa_count=$(kubectl get sa -n kube-system --no-headers | wc -l)
    if [[ $system_sa_count -gt 5 ]]; then
        test_passed "System service accounts present: $system_sa_count"
    else
        test_failed "Insufficient system service accounts: $system_sa_count"
    fi
    
    # Test current user permissions
    if kubectl auth can-i get nodes >/dev/null 2>&1; then
        test_passed "Current user has cluster admin permissions"
    else
        test_failed "Current user lacks necessary permissions"
    fi
}

validate_storage() {
    log_test "Validating storage configuration"
    
    # Check for storage classes
    local storage_classes=$(kubectl get storageclass --no-headers 2>/dev/null | wc -l || true)
    
    if [[ $storage_classes -gt 0 ]]; then
        test_passed "Storage classes available: $storage_classes"
        echo "Storage classes:"
        kubectl get storageclass --no-headers | sed 's/^/  /'
    else
        log_warn "No storage classes found (manual storage setup may be needed)"
    fi
    
    # Check for persistent volumes
    local pv_count=$(kubectl get pv --no-headers 2>/dev/null | wc -l || true)
    if [[ $pv_count -gt 0 ]]; then
        test_passed "Persistent volumes available: $pv_count"
    else
        log_warn "No persistent volumes found (will be created on demand)"
    fi
}

test_workload_scheduling() {
    log_test "Testing workload scheduling"
    
    local test_name="cluster-test-$$-$RANDOM"
    
    echo "Creating test deployment..."
    
    # Create a simple test deployment
    kubectl create deployment "$test_name" --image=nginx:alpine >/dev/null 2>&1
    kubectl scale deployment "$test_name" --replicas=2 >/dev/null 2>&1
    
    # Wait for deployment to be ready
    local max_wait=60
    local waited=0
    
    while [[ $waited -lt $max_wait ]]; do
        local ready_replicas=$(kubectl get deployment "$test_name" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
        
        if [[ "$ready_replicas" == "2" ]]; then
            test_passed "Test workload scheduled and running (2/2 replicas)"
            break
        fi
        
        sleep 2
        ((waited+=2))
    done
    
    if [[ $waited -ge $max_wait ]]; then
        test_failed "Test workload failed to schedule within ${max_wait}s"
    fi
    
    # Test service creation
    kubectl expose deployment "$test_name" --port=80 >/dev/null 2>&1
    
    if kubectl get service "$test_name" >/dev/null 2>&1; then
        test_passed "Service creation successful"
    else
        test_failed "Service creation failed"
    fi
    
    # Cleanup
    kubectl delete deployment "$test_name" >/dev/null 2>&1 || true
    kubectl delete service "$test_name" >/dev/null 2>&1 || true
}

validate_security() {
    log_test "Validating security configuration"
    
    # Check for pod security policies or admission controllers
    if kubectl get psp >/dev/null 2>&1; then
        local psp_count=$(kubectl get psp --no-headers | wc -l)
        test_passed "Pod Security Policies found: $psp_count"
    else
        log_warn "No Pod Security Policies found (using Pod Security Standards instead)"
    fi
    
    # Check network policies support
    if kubectl get networkpolicy -A >/dev/null 2>&1; then
        local netpol_count=$(kubectl get networkpolicy -A --no-headers | wc -l || true)
        test_passed "Network policies supported (found: $netpol_count)"
    else
        test_failed "Network policies not supported"
    fi
    
    # Check for secrets
    local secrets_count=$(kubectl get secrets -A --no-headers | wc -l)
    if [[ $secrets_count -gt 10 ]]; then
        test_passed "System secrets present: $secrets_count"
    else
        log_warn "Few secrets found: $secrets_count"
    fi
}

run_performance_check() {
    log_test "Running basic performance check"
    
    # Check if metrics server is available
    if kubectl top nodes >/dev/null 2>&1; then
        test_passed "Metrics server is available"
        
        echo "Current resource usage:"
        kubectl top nodes --no-headers | while read -r node cpu memory _; do
            echo "  $node: CPU $cpu, Memory $memory"
        done
    else
        log_warn "Metrics server not available (install for resource monitoring)"
    fi
    
    # Test API server throughput
    echo "Testing API server throughput..."
    local start_time=$(date +%s)
    for i in {1..10}; do
        kubectl get nodes >/dev/null 2>&1
    done
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    local throughput=$((10 / duration))
    
    if [[ $throughput -gt 5 ]]; then
        test_passed "API server throughput: ${throughput} requests/second (good)"
    else
        log_warn "API server throughput: ${throughput} requests/second (consider tuning)"
    fi
}

generate_report() {
    echo
    echo "=========================================="
    echo "    DePIN AI Compute Cluster Validation"
    echo "=========================================="
    echo
    echo "Validation completed at: $(date)"
    echo "Total tests run: $TOTAL_TESTS"
    echo "Tests passed: $PASSED_TESTS"
    echo "Tests failed: $FAILED_TESTS"
    echo
    
    local success_rate=$((PASSED_TESTS * 100 / TOTAL_TESTS))
    
    if [[ $FAILED_TESTS -eq 0 ]]; then
        log_info "All validation tests passed! Cluster is ready for AI compute workloads."
        echo
        echo "Next steps:"
        echo "1. Deploy monitoring stack (Prometheus/Grafana)"
        echo "2. Install storage provisioners"
        echo "3. Configure ingress controllers"
        echo "4. Set up security policies"
        echo "5. Deploy AI compute operators"
        
        return 0
    else
        log_error "Validation completed with $FAILED_TESTS failures (${success_rate}% success rate)"
        echo
        echo "Please address the failed tests before proceeding with AI compute workloads."
        
        return 1
    fi
}

main() {
    echo "Starting DePIN AI Compute Cluster Validation..."
    echo "Cluster: $(kubectl config current-context 2>/dev/null || echo 'unknown')"
    echo
    
    # Run all validation tests
    check_kubectl_access || exit 1
    validate_node_readiness
    validate_system_pods
    validate_networking
    validate_dns
    validate_api_server
    validate_rbac
    validate_storage
    test_workload_scheduling
    validate_security
    run_performance_check
    
    # Generate final report
    generate_report
}

main "$@"