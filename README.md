# Confluent Kafka Terraform Automation

使用 Terraform + YAML 自動化管理 Confluent Platform 的 User、Topic 及 Role Binding。

## 架構

```
tenants/
  └── up9999.yaml      ← 定義 tenant 的 password、topics、權限
locals.tf              ← 解析 YAML → Terraform 資源
users.tf               ← SCRAM-SHA-256 user
topics.tf              ← Kafka topic
role_bindings.tf       ← MDS role binding (read/write)
```

## 快速開始

### 1. 新增 Tenant

在 `tenants/` 目錄建立 YAML 檔（檔名 = tenant 名稱）：

```yaml
# tenants/up9999.yaml
password: test123

manage_topics:
  UP9999_order_events: 12   # topic 名稱: partition 數
  UP9999_error_logs: 3

extra_read_topics:           # 額外讀取的 topic
  - public_news_topic

extra_write_topics:          # 額外寫入的 topic
  - test_topic
```

### 2. 環境設定

建立各環境的 tfvars 檔：

```hcl
# envs/DEV.tfvars
environment             = "DEV"
cluster_id              = "your-cluster-id"
mds_authority           = "mds-dev.example.com:8090"
kafka_bootstrap_servers = ["kafka-dev.example.com:9092"]
kafka_tls_enabled       = true
```

### 3. 執行

透過 Jenkins Pipeline 執行：

| 參數 | 選項 | 說明 |
|------|------|------|
| ENV | DEV / SIT / UAT / PROD | 目標環境 |
| ACTION | plan / apply / destroy | Terraform 操作 |
| AUTO_APPROVE | true / false | 是否跳過人工確認 |

### 4. 本地開發

```bash
# 初始化 (使用 local backend)
terraform init

# 檢查格式
terraform fmt -recursive

# 驗證語法
terraform validate

# 預覽變更
terraform plan -var-file="envs/DEV.tfvars"
```

## Replication Factor

| 環境 | Replication Factor |
|------|-------------------|
| DEV  | 1 |
| SIT  | 1 |
| UAT  | 3 |
| PROD | 3 |

可透過 `default_replication_factor` 變數覆寫。

## 權限模型

每個 tenant 自動取得：

| 權限 | 範圍 | 說明 |
|------|------|------|
| DeveloperWrite | `<tenant>_*` | 前綴匹配的所有 topic |
| DeveloperWrite | `extra_write_topics` | 指定的額外 topic |
| DeveloperRead | `<tenant>_*` | 前綴匹配的所有 topic |
| DeveloperRead | `extra_read_topics` | 指定的額外 topic |

## Jenkins Credentials 設定

| Credential ID | 說明 |
|--------------|------|
| `gitlab-project-id` | GitLab Project ID |
| `gitlab-tf-token` | GitLab Personal Access Token |
| `kafka-sasl-username` | Kafka 管理者帳號 |
| `kafka-sasl-password` | Kafka 管理者密碼 |
| `mds-api-key` | MDS 管理者帳號 (e.g., admin) |
| `mds-api-secret` | MDS 管理者密碼 |
