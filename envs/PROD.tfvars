# =============================================================================
# PROD 環境設定
# =============================================================================

environment             = "PROD"
cluster_id              = "your-prod-cluster-id"

kafka_bootstrap_servers = ["kafka-prod-1.example.com:9092", "kafka-prod-2.example.com:9092", "kafka-prod-3.example.com:9092"]
kafka_tls_enabled       = true
tenant_dir              = "./tenants/PROD"
