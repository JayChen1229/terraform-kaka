# =============================================================================
# Provider Requirements
# =============================================================================

terraform {
  required_version = ">= 1.3.0"

  required_providers {
    # Kafka — SCRAM credential + Topic 管理
    kafka = {
      source  = "Mongey/kafka"
      version = "~> 0.7"
    }

    # SOPS — 加密 Tenant YAML 檔案
    sops = {
      source  = "carlpett/sops"
      version = "~> 1.0"
    }
  }
}

# =============================================================================
# Provider Configuration
# =============================================================================

# Kafka Provider — Topic + SCRAM User 管理
provider "kafka" {
  bootstrap_servers = var.kafka_bootstrap_servers
  tls_enabled       = var.kafka_tls_enabled
  skip_tls_verify   = var.kafka_skip_tls_verify

  sasl_username  = var.kafka_sasl_username
  sasl_password  = var.kafka_sasl_password
  sasl_mechanism = "scram-sha256"
}
