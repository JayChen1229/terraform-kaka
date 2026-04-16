# =============================================================================
# Kafka SCRAM User 建立
# =============================================================================
# 為每個 tenant 建立 SCRAM-SHA-256 認證
# tenant key (YAML 檔名) = username
# =============================================================================

resource "kafka_user_scram_credential" "users" {
  for_each = nonsensitive(local.tenants)

  username        = each.key
  scram_mechanism = "SCRAM-SHA-256"
  password        = each.value.password
}

