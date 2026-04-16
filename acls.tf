# =============================================================================
# Kafka ACLs — 傳統 ACL 管理機制
# =============================================================================

# A1. 前綴匹配寫入 (Write)
resource "kafka_acl" "prefix_write_op" {
  for_each = nonsensitive(local.tenants) # 修改這裡

  resource_name                = "${each.key}_"
  resource_type                = "Topic"
  resource_pattern_type_filter = "Prefixed"
  acl_principal                = "User:${each.key}"
  acl_host                     = "*"
  acl_operation                = "Write"
  acl_permission_type          = "Allow"
}

# A2. 前綴匹配寫入 (Describe)
resource "kafka_acl" "prefix_write_describe" {
  for_each = nonsensitive(local.tenants) # 修改這裡

  resource_name                = "${each.key}_"
  resource_type                = "Topic"
  resource_pattern_type_filter = "Prefixed"
  acl_principal                = "User:${each.key}"
  acl_host                     = "*"
  acl_operation                = "Describe"
  acl_permission_type          = "Allow"
}

# B1. 特定 Topic 寫入 (Write)
resource "kafka_acl" "specific_write_op" {
  for_each = {
    for x in nonsensitive(local.specific_writes) : "${x.user}_write_${x.topic}" => x
  }

  resource_name                = each.value.topic
  resource_type                = "Topic"
  resource_pattern_type_filter = "Literal"
  acl_principal                = "User:${each.value.user}"
  acl_host                     = "*"
  acl_operation                = "Write"
  acl_permission_type          = "Allow"
}

# B2. 特定 Topic 寫入 (Describe)
resource "kafka_acl" "specific_write_describe" {
  for_each = {
    for x in nonsensitive(local.specific_writes) : "${x.user}_write_${x.topic}" => x
  }

  resource_name                = each.value.topic
  resource_type                = "Topic"
  resource_pattern_type_filter = "Literal"
  acl_principal                = "User:${each.value.user}"
  acl_host                     = "*"
  acl_operation                = "Describe"
  acl_permission_type          = "Allow"
}

# C1. 前綴匹配讀取 (Read)
resource "kafka_acl" "prefix_read_op" {
  for_each = nonsensitive(local.tenants) # 修改這裡

  resource_name                = "${each.key}_"
  resource_type                = "Topic"
  resource_pattern_type_filter = "Prefixed"
  acl_principal                = "User:${each.key}"
  acl_host                     = "*"
  acl_operation                = "Read"
  acl_permission_type          = "Allow"
}

# C2. 前綴匹配讀取 (Describe)
resource "kafka_acl" "prefix_read_describe" {
  for_each = nonsensitive(local.tenants) # 修改這裡

  resource_name                = "${each.key}_"
  resource_type                = "Topic"
  resource_pattern_type_filter = "Prefixed"
  acl_principal                = "User:${each.key}"
  acl_host                     = "*"
  acl_operation                = "Describe"
  acl_permission_type          = "Allow"
}

# C3. Consumer Group 讀取
resource "kafka_acl" "prefix_group_read" {
  for_each = nonsensitive(local.tenants) # 修改這裡

  resource_name                = "${each.key}_"
  resource_type                = "Group"
  resource_pattern_type_filter = "Prefixed"
  acl_principal                = "User:${each.key}"
  acl_host                     = "*"
  acl_operation                = "Read"
  acl_permission_type          = "Allow"
}

# D1. 特定 Topic 讀取 (Read)
resource "kafka_acl" "specific_read_op" {
  for_each = {
    for x in nonsensitive(local.specific_reads) : "${x.user}_read_${x.topic}" => x
  }

  resource_name                = each.value.topic
  resource_type                = "Topic"
  resource_pattern_type_filter = "Literal"
  acl_principal                = "User:${each.value.user}"
  acl_host                     = "*"
  acl_operation                = "Read"
  acl_permission_type          = "Allow"
}

# D2. 特定 Topic 讀取 (Describe)
resource "kafka_acl" "specific_read_describe" {
  for_each = {
    for x in nonsensitive(local.specific_reads) : "${x.user}_read_${x.topic}" => x
  }

  resource_name                = each.value.topic
  resource_type                = "Topic"
  resource_pattern_type_filter = "Literal"
  acl_principal                = "User:${each.value.user}"
  acl_host                     = "*"
  acl_operation                = "Describe"
  acl_permission_type          = "Allow"
}
