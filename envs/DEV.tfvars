# =============================================================================
# DEV 環境設定
# =============================================================================

environment             = "DEV"
cluster_id              = "ORYzyCCUSZa5yS-hTKO4bg"

kafka_bootstrap_servers = ["efk:9092"]
kafka_tls_enabled       = true
kafka_skip_tls_verify   = true
tenant_dir              = "./tenants/DEV"
