# =============================================================================
# Kafka Topic 建立
# =============================================================================
# 根據每個 tenant YAML 中的 manage_topics 建立 Topic
# - topic name = YAML key (e.g., UP9999_order_events)
# - partitions = YAML value (e.g., 12)
# - replication_factor = 依環境自動計算 (DEV/SIT=1, UAT/PROD=3)
# =============================================================================

resource "kafka_topic" "managed" {
  for_each = {
    for t in nonsensitive(local.managed_topics) : t.topic => t
  }

  name               = each.value.topic
  partitions         = each.value.partitions
  replication_factor = local.replication_factor

  config = each.value.config
}

