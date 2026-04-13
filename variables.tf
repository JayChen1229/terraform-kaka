# =============================================================================
# Environment
# =============================================================================

variable "environment" {
  description = "部署環境 (DEV / SIT / UAT / PROD)"
  type        = string
  validation {
    condition     = contains(["DEV", "SIT", "UAT", "PROD"], var.environment)
    error_message = "environment 必須是 DEV, SIT, UAT, PROD 其中之一。"
  }
}

# =============================================================================
# Confluent / Kafka Cluster
# =============================================================================

variable "cluster_id" {
  description = "Kafka Cluster ID"
  type        = string
}


# =============================================================================
# Kafka Provider Connection
# =============================================================================

variable "kafka_bootstrap_servers" {
  description = "Kafka bootstrap servers (逗號分隔)"
  type        = list(string)
}

variable "kafka_tls_enabled" {
  description = "是否啟用 TLS 連線"
  type        = bool
  default     = true
}

variable "kafka_sasl_username" {
  description = "Kafka SASL 管理者帳號"
  type        = string
  sensitive   = true
}

variable "kafka_sasl_password" {
  description = "Kafka SASL 管理者密碼"
  type        = string
  sensitive   = true
}

variable "kafka_skip_tls_verify" {
  description = "是否跳過 TLS 憑證驗證 (自簽憑證環境設為 true)"
  type        = bool
  default     = false
}


# =============================================================================
# Tenant Configuration
# =============================================================================

variable "tenant_dir" {
  description = "Tenant YAML 設定檔目錄路徑"
  type        = string
  default     = "./tenants"
}

# =============================================================================
# Topic Defaults
# =============================================================================

variable "default_replication_factor" {
  description = "Topic 預設 replication factor (若未指定則依環境自動計算)"
  type        = number
  default     = null
}
