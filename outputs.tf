# =============================================================================
# Outputs
# =============================================================================

output "environment" {
  description = "目前部署的環境"
  value       = var.environment
}

output "replication_factor" {
  description = "目前使用的 Replication Factor"
  value       = local.replication_factor
}

output "tenant_list" {
  description = "所有 Tenant 名稱"
  value       = keys(local.tenants)
}

output "created_users" {
  description = "已建立的 SCRAM User"
  value = {
    for k, v in kafka_user_scram_credential.users : k => {
      username        = v.username
      scram_mechanism = v.scram_mechanism
    }
  }
}

output "created_topics" {
  description = "已建立的 Topic 及其 Partition 數"
  value = {
    for k, v in kafka_topic.managed : k => {
      name               = v.name
      partitions         = v.partitions
      replication_factor = v.replication_factor
    }
  }
}
