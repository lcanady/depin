#!/bin/bash

# DePIN Compliance Check Script
# This script validates compliance with security standards and best practices

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Global variables
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPLIANCE_REPORT="${SCRIPT_DIR}/compliance-report-$(date +%Y%m%d-%H%M%S).json"
TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0
WARNINGS=0

# Compliance frameworks
declare -A COMPLIANCE_STANDARDS=(
    ["CIS"]="Center for Internet Security Kubernetes Benchmark"
    ["NIST"]="NIST Cybersecurity Framework"
    ["SOC2"]="SOC 2 Security Controls"
    ["PCI-DSS"]="Payment Card Industry Data Security Standard"
    ["ISO27001"]="ISO/IEC 27001 Information Security Management"
)

# Initialize report
init_report() {
    cat > "$COMPLIANCE_REPORT" << EOF
{
  "compliance_report": {
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)",
    "cluster_name": "depin-ai-compute",
    "report_version": "1.0",
    "standards": [],
    "summary": {
      "total_checks": 0,
      "passed": 0,
      "failed": 0,
      "warnings": 0,
      "compliance_score": 0
    },
    "checks": []
  }
}
EOF
}

# Logging functions
log_check() {
    local standard="$1"
    local control="$2"
    local description="$3"
    local status="$4"
    local details="$5"
    local severity="${6:-medium}"
    
    ((TOTAL_CHECKS++))
    
    case "$status" in
        "PASS")
            echo -e "${GREEN}[PASS]${NC} $standard-$control: $description"
            ((PASSED_CHECKS++))
            ;;
        "FAIL")
            echo -e "${RED}[FAIL]${NC} $standard-$control: $description"
            echo -e "  ${RED}Details:${NC} $details"
            ((FAILED_CHECKS++))
            ;;
        "WARN")
            echo -e "${YELLOW}[WARN]${NC} $standard-$control: $description"
            echo -e "  ${YELLOW}Details:${NC} $details"
            ((WARNINGS++))
            ;;
    esac
    
    # Add to JSON report
    local check_json=$(cat << EOF
{
  "standard": "$standard",
  "control": "$control",
  "description": "$description",
  "status": "$status",
  "details": "$details",
  "severity": "$severity",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)"
}
EOF
    )
    
    # Append to checks array in JSON report
    tmp=$(mktemp)
    jq --argjson check "$check_json" '.compliance_report.checks += [$check]' "$COMPLIANCE_REPORT" > "$tmp"
    mv "$tmp" "$COMPLIANCE_REPORT"
}

# CIS Kubernetes Benchmark checks
check_cis_controls() {
    echo -e "${BLUE}=== CIS Kubernetes Benchmark v1.7.0 ===${NC}"
    
    # CIS 1.1.1 - API server configuration
    if kubectl get pods -n kube-system -l component=kube-apiserver -o jsonpath='{.items[*].spec.containers[0].command}' | grep -q -- "--anonymous-auth=false"; then
        log_check "CIS" "1.1.1" "Anonymous authentication disabled" "PASS" "API server has anonymous auth disabled" "high"
    else
        log_check "CIS" "1.1.1" "Anonymous authentication disabled" "FAIL" "API server allows anonymous authentication" "high"
    fi
    
    # CIS 1.1.6 - API server insecure port
    if kubectl get pods -n kube-system -l component=kube-apiserver -o jsonpath='{.items[*].spec.containers[0].command}' | grep -q -- "--insecure-port=0"; then
        log_check "CIS" "1.1.6" "Insecure port disabled" "PASS" "API server insecure port is disabled" "high"
    else
        log_check "CIS" "1.1.6" "Insecure port disabled" "FAIL" "API server insecure port may be enabled" "high"
    fi
    
    # CIS 1.1.7 - API server secure port
    if kubectl get pods -n kube-system -l component=kube-apiserver -o jsonpath='{.items[*].spec.containers[0].command}' | grep -q -- "--secure-port"; then
        log_check "CIS" "1.1.7" "Secure port configured" "PASS" "API server secure port is configured" "medium"
    else
        log_check "CIS" "1.1.7" "Secure port configured" "WARN" "API server secure port configuration not verified" "medium"
    fi
    
    # CIS 1.1.8 - Profiling disabled
    if kubectl get pods -n kube-system -l component=kube-apiserver -o jsonpath='{.items[*].spec.containers[0].command}' | grep -q -- "--profiling=false"; then
        log_check "CIS" "1.1.8" "Profiling disabled" "PASS" "API server profiling is disabled" "medium"
    else
        log_check "CIS" "1.1.8" "Profiling disabled" "FAIL" "API server profiling may be enabled" "medium"
    fi
    
    # CIS 1.1.9 - Repair malformed requests
    if kubectl get pods -n kube-system -l component=kube-apiserver -o jsonpath='{.items[*].spec.containers[0].command}' | grep -q -- "--repair-malformed-updates=false"; then
        log_check "CIS" "1.1.9" "Repair malformed requests disabled" "PASS" "API server doesn't repair malformed requests" "low"
    else
        log_check "CIS" "1.1.9" "Repair malformed requests disabled" "WARN" "API server may repair malformed requests" "low"
    fi
    
    # CIS 1.1.11 - Audit log path
    if kubectl get pods -n kube-system -l component=kube-apiserver -o jsonpath='{.items[*].spec.containers[0].command}' | grep -q -- "--audit-log-path"; then
        log_check "CIS" "1.1.11" "Audit log path configured" "PASS" "API server audit logging is configured" "high"
    else
        log_check "CIS" "1.1.11" "Audit log path configured" "FAIL" "API server audit logging not configured" "high"
    fi
    
    # CIS 5.1.1 - Image vulnerability scanning
    local images_with_vulnerabilities=$(kubectl get pods --all-namespaces -o jsonpath='{.items[*].spec.containers[*].image}' | tr ' ' '\n' | sort -u | wc -l)
    if [ "$images_with_vulnerabilities" -gt 0 ]; then
        log_check "CIS" "5.1.1" "Container image vulnerabilities" "WARN" "Found $images_with_vulnerabilities unique container images - vulnerability scanning recommended" "medium"
    fi
    
    # CIS 5.1.3 - Minimize capabilities
    local pods_with_added_caps=$(kubectl get pods --all-namespaces -o json | jq -r '.items[] | select(.spec.containers[].securityContext.capabilities.add // [] | length > 0) | "\(.metadata.namespace)/\(.metadata.name)"' | wc -l)
    if [ "$pods_with_added_caps" -eq 0 ]; then
        log_check "CIS" "5.1.3" "Minimize additional capabilities" "PASS" "No pods with additional capabilities found" "medium"
    else
        log_check "CIS" "5.1.3" "Minimize additional capabilities" "WARN" "$pods_with_added_caps pods have additional capabilities" "medium"
    fi
    
    # CIS 5.1.4 - Don't use privileged containers
    local privileged_pods=$(kubectl get pods --all-namespaces -o json | jq -r '.items[] | select(.spec.containers[].securityContext.privileged // false) | "\(.metadata.namespace)/\(.metadata.name)"' | wc -l)
    if [ "$privileged_pods" -eq 0 ]; then
        log_check "CIS" "5.1.4" "No privileged containers" "PASS" "No privileged containers found" "high"
    else
        log_check "CIS" "5.1.4" "No privileged containers" "FAIL" "$privileged_pods privileged containers found" "high"
    fi
    
    # CIS 5.2.3 - Minimize admission of containers with allowPrivilegeEscalation
    local privilege_escalation_pods=$(kubectl get pods --all-namespaces -o json | jq -r '.items[] | select(.spec.containers[].securityContext.allowPrivilegeEscalation // true) | "\(.metadata.namespace)/\(.metadata.name)"' | wc -l)
    if [ "$privilege_escalation_pods" -eq 0 ]; then
        log_check "CIS" "5.2.3" "Minimize privilege escalation" "PASS" "No containers allow privilege escalation" "high"
    else
        log_check "CIS" "5.2.3" "Minimize privilege escalation" "FAIL" "$privilege_escalation_pods containers allow privilege escalation" "high"
    fi
    
    # CIS 5.2.4 - Minimize admission of root containers
    local root_containers=$(kubectl get pods --all-namespaces -o json | jq -r '.items[] | select(.spec.containers[].securityContext.runAsUser // 0 == 0 or .spec.securityContext.runAsUser // 0 == 0) | "\(.metadata.namespace)/\(.metadata.name)"' | wc -l)
    if [ "$root_containers" -eq 0 ]; then
        log_check "CIS" "5.2.4" "Minimize root containers" "PASS" "No containers running as root" "high"
    else
        log_check "CIS" "5.2.4" "Minimize root containers" "FAIL" "$root_containers containers running as root" "high"
    fi
    
    # CIS 5.7.2 - Apply Security Context to Pods and Containers
    local pods_without_security_context=$(kubectl get pods --all-namespaces -o json | jq -r '.items[] | select(.spec.securityContext == null and (.spec.containers[] | .securityContext == null)) | "\(.metadata.namespace)/\(.metadata.name)"' | wc -l)
    if [ "$pods_without_security_context" -eq 0 ]; then
        log_check "CIS" "5.7.2" "Security context configured" "PASS" "All pods have security context configured" "medium"
    else
        log_check "CIS" "5.7.2" "Security context configured" "FAIL" "$pods_without_security_context pods missing security context" "medium"
    fi
}

# NIST Cybersecurity Framework checks
check_nist_controls() {
    echo -e "${BLUE}=== NIST Cybersecurity Framework ===${NC}"
    
    # NIST ID.AM-2 - Asset inventory
    local total_resources=$(kubectl api-resources --verbs=list --namespaced -o name 2>/dev/null | wc -l)
    if [ "$total_resources" -gt 0 ]; then
        log_check "NIST" "ID.AM-2" "Asset inventory maintained" "PASS" "Kubernetes API provides comprehensive asset inventory" "medium"
    else
        log_check "NIST" "ID.AM-2" "Asset inventory maintained" "FAIL" "Unable to enumerate cluster resources" "high"
    fi
    
    # NIST PR.AC-1 - Access control policy
    local rbac_enabled=$(kubectl auth can-i --list --as=system:anonymous 2>/dev/null | wc -l)
    if [ "$rbac_enabled" -eq 0 ]; then
        log_check "NIST" "PR.AC-1" "Access control policy enforced" "PASS" "RBAC prevents anonymous access" "high"
    else
        log_check "NIST" "PR.AC-1" "Access control policy enforced" "FAIL" "Anonymous access may be permitted" "high"
    fi
    
    # NIST PR.AC-4 - Access permissions and authorizations
    local depin_service_accounts=$(kubectl get serviceaccounts --all-namespaces -l 'depin.ai/component' -o name | wc -l)
    if [ "$depin_service_accounts" -gt 0 ]; then
        log_check "NIST" "PR.AC-4" "Dedicated service accounts" "PASS" "$depin_service_accounts DePIN service accounts configured" "medium"
    else
        log_check "NIST" "PR.AC-4" "Dedicated service accounts" "WARN" "No DePIN-specific service accounts found" "medium"
    fi
    
    # NIST PR.DS-1 - Data at rest protection
    local encrypted_secrets=$(kubectl get secrets --all-namespaces -o json | jq -r '.items[] | select(.type != "kubernetes.io/service-account-token") | "\(.metadata.namespace)/\(.metadata.name)"' | wc -l)
    if [ "$encrypted_secrets" -gt 0 ]; then
        log_check "NIST" "PR.DS-1" "Data at rest encryption" "PASS" "$encrypted_secrets secrets stored in etcd (encrypted)" "high"
    else
        log_check "NIST" "PR.DS-1" "Data at rest encryption" "WARN" "No application secrets found" "medium"
    fi
    
    # NIST PR.DS-2 - Data in transit protection
    local tls_ingresses=$(kubectl get ingresses --all-namespaces -o json | jq -r '.items[] | select(.spec.tls != null) | "\(.metadata.namespace)/\(.metadata.name)"' | wc -l)
    local total_ingresses=$(kubectl get ingresses --all-namespaces --no-headers 2>/dev/null | wc -l)
    if [ "$total_ingresses" -gt 0 ] && [ "$tls_ingresses" -eq "$total_ingresses" ]; then
        log_check "NIST" "PR.DS-2" "Data in transit encryption" "PASS" "All ingresses use TLS encryption" "high"
    elif [ "$total_ingresses" -gt 0 ]; then
        log_check "NIST" "PR.DS-2" "Data in transit encryption" "WARN" "$((total_ingresses - tls_ingresses)) ingresses without TLS" "high"
    else
        log_check "NIST" "PR.DS-2" "Data in transit encryption" "WARN" "No ingresses found to evaluate" "medium"
    fi
    
    # NIST DE.AE-1 - Event monitoring
    local monitoring_pods=$(kubectl get pods --all-namespaces -l 'app.kubernetes.io/name=prometheus' -o name | wc -l)
    if [ "$monitoring_pods" -gt 0 ]; then
        log_check "NIST" "DE.AE-1" "Event monitoring configured" "PASS" "Prometheus monitoring deployed" "high"
    else
        log_check "NIST" "DE.AE-1" "Event monitoring configured" "FAIL" "No monitoring system detected" "high"
    fi
    
    # NIST DE.CM-1 - Continuous monitoring
    local daemonset_monitoring=$(kubectl get daemonsets --all-namespaces -l 'depin.ai/component' -o name | wc -l)
    if [ "$daemonset_monitoring" -gt 0 ]; then
        log_check "NIST" "DE.CM-1" "Continuous monitoring" "PASS" "$daemonset_monitoring monitoring daemonsets deployed" "medium"
    else
        log_check "NIST" "DE.CM-1" "Continuous monitoring" "WARN" "No DePIN monitoring daemonsets found" "medium"
    fi
    
    # NIST RS.RP-1 - Response plan execution
    local incident_response_config=$(kubectl get configmaps --all-namespaces -l 'depin.ai/component=security-monitoring' -o name | wc -l)
    if [ "$incident_response_config" -gt 0 ]; then
        log_check "NIST" "RS.RP-1" "Incident response procedures" "PASS" "Security incident response configuration found" "medium"
    else
        log_check "NIST" "RS.RP-1" "Incident response procedures" "WARN" "No incident response configuration found" "medium"
    fi
}

# SOC 2 Security Controls checks
check_soc2_controls() {
    echo -e "${BLUE}=== SOC 2 Security Controls ===${NC}"
    
    # SOC2 CC6.1 - Logical access security measures
    local network_policies=$(kubectl get networkpolicies --all-namespaces --no-headers 2>/dev/null | wc -l)
    if [ "$network_policies" -gt 0 ]; then
        log_check "SOC2" "CC6.1" "Network access controls" "PASS" "$network_policies network policies configured" "high"
    else
        log_check "SOC2" "CC6.1" "Network access controls" "FAIL" "No network policies found" "high"
    fi
    
    # SOC2 CC6.2 - Authentication and authorization
    local pod_security_policies=$(kubectl get psp --no-headers 2>/dev/null | wc -l || echo "0")
    local pod_security_standards=$(kubectl get namespaces -l 'pod-security.kubernetes.io/enforce' --no-headers 2>/dev/null | wc -l)
    
    if [ "$pod_security_policies" -gt 0 ] || [ "$pod_security_standards" -gt 0 ]; then
        log_check "SOC2" "CC6.2" "Pod security enforcement" "PASS" "Pod security controls configured" "high"
    else
        log_check "SOC2" "CC6.2" "Pod security enforcement" "FAIL" "No pod security controls found" "high"
    fi
    
    # SOC2 CC6.3 - User access management
    local cluster_admin_bindings=$(kubectl get clusterrolebindings -o json | jq -r '.items[] | select(.roleRef.name == "cluster-admin") | .metadata.name' | wc -l)
    if [ "$cluster_admin_bindings" -lt 5 ]; then
        log_check "SOC2" "CC6.3" "Privileged access management" "PASS" "Limited cluster-admin bindings ($cluster_admin_bindings)" "high"
    else
        log_check "SOC2" "CC6.3" "Privileged access management" "WARN" "Many cluster-admin bindings ($cluster_admin_bindings)" "high"
    fi
    
    # SOC2 CC7.1 - System monitoring
    local security_monitoring=$(kubectl get prometheusrules --all-namespaces -l 'depin.ai/component=security-monitoring' --no-headers 2>/dev/null | wc -l)
    if [ "$security_monitoring" -gt 0 ]; then
        log_check "SOC2" "CC7.1" "Security monitoring rules" "PASS" "Security monitoring rules configured" "high"
    else
        log_check "SOC2" "CC7.1" "Security monitoring rules" "WARN" "No security monitoring rules found" "medium"
    fi
    
    # SOC2 CC7.2 - Change management
    local admission_webhooks=$(kubectl get validatingadmissionwebhooks -o name 2>/dev/null | wc -l)
    if [ "$admission_webhooks" -gt 0 ]; then
        log_check "SOC2" "CC7.2" "Change control mechanisms" "PASS" "$admission_webhooks admission webhooks configured" "medium"
    else
        log_check "SOC2" "CC7.2" "Change control mechanisms" "WARN" "No admission webhooks found" "medium"
    fi
}

# Generate final compliance report
generate_compliance_report() {
    local compliance_score=$(( (PASSED_CHECKS * 100) / TOTAL_CHECKS ))
    
    # Update summary in JSON report
    tmp=$(mktemp)
    jq --arg total "$TOTAL_CHECKS" \
       --arg passed "$PASSED_CHECKS" \
       --arg failed "$FAILED_CHECKS" \
       --arg warnings "$WARNINGS" \
       --arg score "$compliance_score" \
       '.compliance_report.summary = {
         "total_checks": ($total | tonumber),
         "passed": ($passed | tonumber),
         "failed": ($failed | tonumber),
         "warnings": ($warnings | tonumber),
         "compliance_score": ($score | tonumber)
       }' "$COMPLIANCE_REPORT" > "$tmp"
    mv "$tmp" "$COMPLIANCE_REPORT"
    
    # Add standards information
    tmp=$(mktemp)
    jq '.compliance_report.standards = [
      {"name": "CIS", "description": "Center for Internet Security Kubernetes Benchmark"},
      {"name": "NIST", "description": "NIST Cybersecurity Framework"},
      {"name": "SOC2", "description": "SOC 2 Security Controls"}
    ]' "$COMPLIANCE_REPORT" > "$tmp"
    mv "$tmp" "$COMPLIANCE_REPORT"
    
    echo
    echo -e "${BLUE}=== COMPLIANCE REPORT SUMMARY ===${NC}"
    echo -e "Total Checks: $TOTAL_CHECKS"
    echo -e "Passed: ${GREEN}$PASSED_CHECKS${NC}"
    echo -e "Failed: ${RED}$FAILED_CHECKS${NC}"
    echo -e "Warnings: ${YELLOW}$WARNINGS${NC}"
    echo -e "Compliance Score: ${BLUE}$compliance_score%${NC}"
    echo
    echo -e "Detailed report saved to: ${COMPLIANCE_REPORT}"
    
    # Generate recommendations
    echo -e "\n${BLUE}=== RECOMMENDATIONS ===${NC}"
    
    if [ $FAILED_CHECKS -eq 0 ] && [ $WARNINGS -eq 0 ]; then
        echo -e "${GREEN}✓${NC} Excellent! All compliance checks passed."
        echo -e "${GREEN}✓${NC} Continue monitoring and maintain security posture."
    elif [ $FAILED_CHECKS -eq 0 ]; then
        echo -e "${YELLOW}!${NC} Good compliance score with some warnings."
        echo -e "${YELLOW}!${NC} Address warnings to achieve full compliance."
    elif [ $compliance_score -ge 80 ]; then
        echo -e "${YELLOW}!${NC} Acceptable compliance score but needs improvement."
        echo -e "${RED}!${NC} Priority: Address failed security controls."
        echo -e "${YELLOW}!${NC} Review and remediate warning items."
    else
        echo -e "${RED}✗${NC} Poor compliance score - immediate action required."
        echo -e "${RED}✗${NC} High Priority: Fix critical security failures."
        echo -e "${RED}✗${NC} Implement comprehensive security review process."
    fi
}

# Main execution function
main() {
    echo -e "${BLUE}DePIN Kubernetes Compliance Assessment${NC}"
    echo -e "Report: $COMPLIANCE_REPORT"
    echo
    
    # Check kubectl connectivity
    if ! kubectl cluster-info > /dev/null 2>&1; then
        echo -e "${RED}Error: Cannot connect to Kubernetes cluster${NC}"
        exit 1
    fi
    
    # Initialize report
    init_report
    
    # Run compliance checks
    check_cis_controls
    echo
    check_nist_controls
    echo
    check_soc2_controls
    echo
    
    # Generate final report
    generate_compliance_report
    
    # Exit with error code if critical failures
    if [ $FAILED_CHECKS -gt 0 ]; then
        exit 1
    fi
}

# Execute main function
main "$@"