# DePIN AI Compute - Kubernetes Infrastructure

This directory contains the complete Kubernetes cluster infrastructure for the DePIN AI Compute network, providing a highly available, secure, and scalable foundation for AI workloads.

## 📁 Directory Structure

```
infrastructure/k8s/
├── cluster/                    # Core cluster configuration
│   ├── main.tf                # Terraform configuration
│   ├── kubeadm-config.yaml    # Kubeadm cluster configuration
│   ├── audit-policy.yaml      # Kubernetes audit policy
│   ├── bootstrap-cluster.sh   # Cluster bootstrap script
│   ├── health-check.yaml      # Health monitoring manifests
│   └── validate-cluster.sh    # Cluster validation script
├── networking/                 # Network configuration
│   ├── calico-config.yaml     # Calico CNI configuration
│   ├── calico-install.yaml    # Calico installation manifest
│   ├── coredns-config.yaml    # Enhanced CoreDNS configuration
│   └── setup-networking.sh    # Network setup script
└── README.md                   # This file
```

## 🎯 Architecture Overview

The infrastructure implements a production-ready Kubernetes cluster with:

- **High Availability Control Plane**: 3+ control plane nodes with external load balancer
- **Secure Networking**: Calico/Cilium CNI with network policies and traffic encryption
- **Enhanced DNS**: CoreDNS with custom zones and service discovery
- **Comprehensive Security**: RBAC, Pod Security Standards, audit logging
- **Health Monitoring**: Automated health checks and alerting
- **Infrastructure as Code**: Terraform for reproducible deployments

## 🚀 Quick Start

### Prerequisites

- Linux nodes (Ubuntu 20.04+ or CentOS 8+ recommended)
- At least 3 control plane nodes and 3 worker nodes
- 2 CPUs, 4GB RAM minimum per node
- Container runtime installed (Docker/containerd)
- Load balancer for control plane API access

### 1. Bootstrap First Control Plane Node

```bash
# Copy configuration files to first control plane node
scp -r infrastructure/k8s/cluster/ user@control-plane-1:/tmp/

# SSH to first control plane node
ssh user@control-plane-1

# Become root and bootstrap cluster
sudo -i
cd /tmp/cluster/
./bootstrap-cluster.sh init-first
```

### 2. Join Additional Control Plane Nodes

```bash
# Copy join command and certificate key from first node
JOIN_COMMAND="$(cat /tmp/kubeadm-join-worker.sh | sed 's/$/ --control-plane/')"
CERTIFICATE_KEY="$(cat /tmp/certificate-key.txt)"

# On each additional control plane node
sudo -i
cd /tmp/cluster/
JOIN_COMMAND="$JOIN_COMMAND" CERTIFICATE_KEY="$CERTIFICATE_KEY" ./bootstrap-cluster.sh join-control
```

### 3. Join Worker Nodes

```bash
# Copy join command from control plane node
JOIN_COMMAND="$(cat /tmp/kubeadm-join-worker.sh)"

# On each worker node
sudo -i
cd /tmp/cluster/
JOIN_COMMAND="$JOIN_COMMAND" ./bootstrap-cluster.sh join-worker
```

### 4. Configure Networking

```bash
# From a control plane node with kubectl configured
cd /tmp/networking/
./setup-networking.sh

# Or install specific CNI provider
./setup-networking.sh --cni-provider cilium
```

### 5. Validate Cluster

```bash
# Run comprehensive cluster validation
cd /tmp/cluster/
./validate-cluster.sh
```

## ⚙️ Configuration

### Cluster Configuration

The cluster is configured with:

- **Pod CIDR**: `10.244.0.0/16`
- **Service CIDR**: `10.96.0.0/16`
- **DNS Domain**: `cluster.local`
- **API Server Port**: `6443`

### Security Features

- **Audit Logging**: Comprehensive audit policy for security monitoring
- **RBAC**: Role-based access control for fine-grained permissions
- **Network Policies**: Default-deny policies with selective allow rules
- **Pod Security Standards**: Enforced security contexts and capabilities
- **TLS Encryption**: All communication encrypted with TLS 1.2+

### Networking

- **CNI Provider**: Calico (default) or Cilium
- **Network Policy**: Enabled with default-deny rules
- **DNS**: Enhanced CoreDNS with custom zones and caching
- **Service Mesh Ready**: Compatible with Istio/Linkerd

## 📊 Monitoring and Health Checks

### Automated Health Monitoring

The cluster includes comprehensive health monitoring:

```bash
# Deploy health monitoring
kubectl apply -f cluster/health-check.yaml

# View health check results
kubectl logs -n cluster-health -l app=cluster-health-check --tail=100
```

### Manual Validation

```bash
# Run validation script
./cluster/validate-cluster.sh

# Check cluster status
kubectl get nodes -o wide
kubectl get pods --all-namespaces
kubectl cluster-info
```

## 🔧 Maintenance

### Upgrading Kubernetes

1. **Drain nodes** (one at a time):
   ```bash
   kubectl drain node-name --ignore-daemonsets --force
   ```

2. **Upgrade kubeadm** on the node:
   ```bash
   apt-get update && apt-get install -y kubeadm=1.29.x-00
   ```

3. **Upgrade node**:
   ```bash
   kubeadm upgrade node
   ```

4. **Upgrade kubelet and kubectl**:
   ```bash
   apt-get install -y kubelet=1.29.x-00 kubectl=1.29.x-00
   systemctl restart kubelet
   ```

5. **Uncordon node**:
   ```bash
   kubectl uncordon node-name
   ```

### Backup and Disaster Recovery

```bash
# Backup etcd
kubectl exec -n kube-system etcd-master-node -- etcdctl \
  --endpoints=https://127.0.0.1:2379 \
  --cacert=/etc/kubernetes/pki/etcd/ca.crt \
  --cert=/etc/kubernetes/pki/etcd/server.crt \
  --key=/etc/kubernetes/pki/etcd/server.key \
  snapshot save /backup/etcd-snapshot-$(date +%Y%m%d_%H%M%S).db

# Backup cluster configuration
kubectl get all --all-namespaces -o yaml > cluster-backup-$(date +%Y%m%d_%H%M%S).yaml
```

## 🔒 Security Hardening

### Additional Security Measures

1. **Enable admission controllers**:
   ```yaml
   # Add to kube-apiserver configuration
   --enable-admission-plugins=NodeRestriction,PodSecurityPolicy,ResourceQuota,LimitRanger
   ```

2. **Configure network policies**:
   ```bash
   kubectl apply -f networking/calico-config.yaml
   ```

3. **Enable audit logging**:
   ```bash
   # Audit logs are automatically configured in kubeadm-config.yaml
   tail -f /var/log/audit.log
   ```

## 🐛 Troubleshooting

### Common Issues

1. **Nodes not joining cluster**:
   ```bash
   # Check node status
   systemctl status kubelet
   journalctl -u kubelet -f
   
   # Reset node if needed
   kubeadm reset
   ```

2. **CNI issues**:
   ```bash
   # Check CNI pods
   kubectl get pods -n calico-system
   
   # Restart CNI
   kubectl delete pods -n calico-system -l k8s-app=calico-node
   ```

3. **DNS resolution problems**:
   ```bash
   # Test DNS
   kubectl run dns-test --image=busybox --rm -it -- nslookup kubernetes
   
   # Check CoreDNS logs
   kubectl logs -n kube-system -l k8s-app=kube-dns
   ```

4. **API server issues**:
   ```bash
   # Check API server health
   kubectl get --raw="/healthz"
   
   # Check static pod logs
   tail -f /var/log/pods/kube-system_kube-apiserver-*/*.log
   ```

### Log Locations

- **kubelet**: `/var/log/syslog` or `journalctl -u kubelet`
- **containers**: `/var/log/containers/`
- **pods**: `/var/log/pods/`
- **audit**: `/var/log/audit.log`

## 📈 Performance Optimization

### Node Performance

1. **CPU isolation** for AI workloads:
   ```bash
   # Add to kubelet configuration
   --cpu-manager-policy=static
   --topology-manager-policy=single-numa-node
   ```

2. **Memory management**:
   ```bash
   # Configure memory limits
   --enforce-node-allocatable=pods,system-reserved,kube-reserved
   --kube-reserved=cpu=100m,memory=100Mi
   --system-reserved=cpu=100m,memory=100Mi
   ```

### Network Performance

1. **Enable network acceleration**:
   ```bash
   # For Calico
   kubectl patch felixconfiguration default --type merge -p '{"spec":{"bpfEnabled":true}}'
   ```

2. **Optimize CNI settings**:
   ```bash
   # Increase network buffer sizes
   echo 'net.core.rmem_max = 134217728' >> /etc/sysctl.conf
   echo 'net.core.wmem_max = 134217728' >> /etc/sysctl.conf
   sysctl -p
   ```

## 🎯 Next Steps

After successful cluster deployment:

1. **Deploy monitoring stack** (Issue #7):
   - Prometheus for metrics
   - Grafana for visualization
   - AlertManager for alerting

2. **Configure storage** (Issue #8):
   - Deploy storage operators
   - Configure persistent volumes
   - Set up backup systems

3. **Implement security policies** (Issue #9):
   - Pod Security Standards
   - Network segmentation
   - Secret management

4. **Install AI compute operators** (Issue #10):
   - GPU operators
   - ML workflow engines
   - Resource schedulers

## 📞 Support

For issues and support:

1. Check the troubleshooting section above
2. Review cluster logs and health checks
3. Validate configuration against this documentation
4. Check GitHub issues for known problems

## 📄 References

- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Kubeadm Installation](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/)
- [Calico Documentation](https://docs.projectcalico.org/)
- [Cilium Documentation](https://docs.cilium.io/)
- [CoreDNS Documentation](https://coredns.io/)