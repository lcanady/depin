# Terraform configuration for DePIN AI Compute Kubernetes Cluster
# This configuration deploys a highly available Kubernetes cluster for AI compute workloads

terraform {
  required_version = ">= 1.0"
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.23"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.11"
    }
    local = {
      source  = "hashicorp/local"
      version = "~> 2.4"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "~> 4.0"
    }
  }
}

# Variables
variable "cluster_name" {
  description = "Name of the Kubernetes cluster"
  type        = string
  default     = "depin-ai-compute"
}

variable "cluster_version" {
  description = "Kubernetes version"
  type        = string
  default     = "1.28"
}

variable "node_count" {
  description = "Number of worker nodes"
  type        = number
  default     = 3
}

variable "control_plane_count" {
  description = "Number of control plane nodes"
  type        = number
  default     = 3
}

variable "instance_type" {
  description = "Instance type for nodes"
  type        = string
  default     = "m5.xlarge"
}

# Locals
locals {
  common_tags = {
    Environment = "production"
    Project     = "depin-ai-compute"
    ManagedBy   = "terraform"
  }
}

# Generate SSH key pair for cluster nodes
resource "tls_private_key" "cluster_key" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

# Save SSH private key locally
resource "local_file" "cluster_private_key" {
  content  = tls_private_key.cluster_key.private_key_pem
  filename = "${path.module}/cluster-key.pem"
  file_permission = "0600"
}

# Save SSH public key locally
resource "local_file" "cluster_public_key" {
  content  = tls_private_key.cluster_key.public_key_openssh
  filename = "${path.module}/cluster-key.pub"
  file_permission = "0644"
}

# Output cluster information
output "cluster_name" {
  description = "Name of the Kubernetes cluster"
  value       = var.cluster_name
}

output "cluster_version" {
  description = "Kubernetes cluster version"
  value       = var.cluster_version
}

output "ssh_private_key_path" {
  description = "Path to SSH private key for cluster access"
  value       = local_file.cluster_private_key.filename
  sensitive   = true
}

output "ssh_public_key_path" {
  description = "Path to SSH public key for cluster nodes"
  value       = local_file.cluster_public_key.filename
}