# =============================================================================
# Locals — YAML 解析 & 資料展開
# =============================================================================

locals {
  # ---------------------------------------------------------------------------
  # 環境對應的 Replication Factor
  # ---------------------------------------------------------------------------
  env_replication_factor = {
    DEV  = 1
    SIT  = 1
    UAT  = 3
    PROD = 3
  }

  replication_factor = (
    var.default_replication_factor != null
    ? var.default_replication_factor
    : local.env_replication_factor[var.environment]
  )

  # ---------------------------------------------------------------------------
  # 讀取 & 解析所有 Tenant YAML
  # ---------------------------------------------------------------------------
  tenant_files = fileset(var.tenant_dir, "*.yaml")
}

data "sops_file" "tenants" {
  for_each    = local.tenant_files
  source_file = "${var.tenant_dir}/${each.key}"
}

locals {
  tenants = {
    for f in local.tenant_files :
    trimsuffix(f, ".yaml") => yamldecode(data.sops_file.tenants[f].raw)
  }

  # ---------------------------------------------------------------------------
  # 展開 manage_topics → 用於建立 Topic
  # 統一格式: topicName: { partitions: 12, config: { retention.ms: "86400000" } }
  # ---------------------------------------------------------------------------
  managed_topics = flatten([
    for user, cfg in local.tenants : [
      for topic, settings in try(cfg.manage_topics, {}) : {
        user       = user
        topic      = topic
        partitions = try(settings.partitions, 1)
        config     = try(settings.config, {})
      }
    ]
  ])

  # ---------------------------------------------------------------------------
  # 展開 extra_write_topics → 用於特定 Topic 寫入權限
  # ---------------------------------------------------------------------------
  specific_writes = flatten([
    for user, cfg in local.tenants : [
      for topic in try(cfg.extra_write_topics, []) : {
        user  = user
        topic = topic
      }
    ]
  ])

  # ---------------------------------------------------------------------------
  # 展開 extra_read_topics → 用於特定 Topic 讀取權限
  # ---------------------------------------------------------------------------
  specific_reads = flatten([
    for user, cfg in local.tenants : [
      for topic in try(cfg.extra_read_topics, []) : {
        user  = user
        topic = topic
      }
    ]
  ])
}
