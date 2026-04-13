# Kafka Infrastructure as Code (Terraform)

本專案使用 **Terraform** 作為 Infrastructure as Code (IaC) 的核心，管理跨環境的 Kafka 叢集 (Topics, Users, ACLs)。為了達到高度安全的 GitOps 架構，專案的敏感設定檔（例如使用者的密碼）已整合 **Mozilla SOPS** 搭配 **age** 進行 AES256 加密。所有的變更都透過 **Jenkins CI/CD Pipeline** 進行自動化部署。

---

## 🛠️ 前置作業與依賴工具安裝

在本地開發或測試之前，請確保你的電腦上已安裝以下工具：

### 1. Terraform 安裝
請根據你的作業系統下載並安裝 Terraform 核心。
* **macOS**: `brew install terraform`
* **Linux (Ubuntu/Debian)**: 
  ```bash
  sudo apt-get update && sudo apt-get install -y gnupg software-properties-common
  wget -O- https://apt.releases.hashicorp.com/gpg | gpg --dearmor | sudo tee /usr/share/keyrings/hashicorp-archive-keyring.gpg
  echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list
  sudo apt update && sudo apt-get install terraform
  ```
* **Windows**: 使用 [Chocolatey](https://chocolatey.org/) 執行 `choco install terraform`

### 2. age 安裝 (加密引擎)
用於產生非對稱加解密的公私鑰對。
* **macOS**: `brew install age`
* **Linux**: `sudo apt install age`
* **Windows**: `choco install age.portable`

### 3. Mozilla SOPS 安裝
SOPS 是一套用來加密 YAML/JSON 檔案的好用工具，支援精準欄位加密。
* **macOS**: `brew install sops`
* **Linux**: 
  ```bash
  wget https://github.com/getsops/sops/releases/download/v3.8.1/sops-v3.8.1.linux.amd64
  sudo mv sops-v3.8.1.linux.amd64 /usr/local/bin/sops
  sudo chmod +x /usr/local/bin/sops
  ```
  
---

## 🔐 憑證與金鑰 (Credential) 設定

### 負責加密的 Age Key 產生機制
本專案採用了 `mac_only_encrypted: true`，代表開發者只要拿到「**公鑰**」就可以修改 `topic` 等非機敏欄位；只有 **Jenkins** 或 Admin 在執行 `terraform apply` 時才需要「**私鑰**」。

1. **產生金鑰 (Admin 負責)**：
   ```bash
   age-keygen -o sops_age.key
   ```
2. **更新專案設定**：
   找出產生的公鑰（`age1...`），並填入本專案根目錄的 `.sops.yaml` 檔案中。
3. **Jenkins 憑證設定**：
   請登入你的 Jenkins，進入 `Credentials` 頁面新增以下三個 Secret：
   * **`gitlab`** (Username with password)：你的 GitLab 帳號與 Token，用於 Jenkins 下載程式碼與回推。
   * **`kafka-admin`** (Username with password)：具有 Kafka 最高權限的管理員帳號與密碼，供 Terraform 操作 Kafka 特權。
   * **`sops-age-key`** (Secret text)：請把剛才產生的 `sops_age.key` **檔案的完整內容**貼上。這樣 Terraform 即可順利解密 YAML。

---

## ☁️ Terraform State (雲端狀態) 設定

我們不建議在本機跑 Terraform Apply，因為本專案的 State 鎖定並存放在 **GitLab Terraform 託管服務 (HTTP Backend)** 內。

如果在某些除錯情況下需要**本地執行 Terraform**，你需要設定環境變數指向 GitLab：

```bash
# 1. 告訴 SOPS 你的本機私鑰位置 (如果你要看密碼的話)
export SOPS_AGE_KEY_FILE=/你的路徑/sops_age.key

# 2. 設定 GitLab HTTP State 的存取路由
export PROJECT_ID="你的 GitLab 專案 ID"
export ENV="DEV" # DEV, UAT 等
export TF_HTTP_ADDRESS="https://gitlab.com/api/v4/projects/${PROJECT_ID}/terraform/state/confluent-${ENV}"
export TF_HTTP_LOCK_ADDRESS="${TF_HTTP_ADDRESS}/lock"
export TF_HTTP_UNLOCK_ADDRESS="${TF_HTTP_ADDRESS}/lock"
export TF_HTTP_LOCK_METHOD="POST"
export TF_HTTP_UNLOCK_METHOD="DELETE"

# 3. 提供 GitLab 的 Token
export TF_HTTP_USERNAME="你的名稱"
export TF_HTTP_PASSWORD="你的 Personal Access Token"

# 4. 提供 Kafka Admin
export TF_VAR_kafka_sasl_username="Admin帳號"
export TF_VAR_kafka_sasl_password="Admin密碼"

terraform init
terraform plan -var-file="envs/${ENV}.tfvars"
```

---

## 🚀 日常開發與 CI/CD 流程

團隊成員並不需要經常在本機跑 `sops` 或 `terraform`。
1. **觸發變更**：直接進到 Jenkins 介面，填寫選項參數（如 `createUser`, `createProducerACL`, `ENV` 等）。
2. **自動修改與加密**：Jenkins 的 `Mutate YAML` stage 會自動幫你改寫 `tenants/` 對應的 YAML 檔案。如果是新建使用者，更會自動透過 SOPS 將 `password` 欄位進行加密。
3. **部署**：接著 Pipeline 自動執行 `terraform plan/apply`。
4. **版本控管**：部署成功後，Jenkins 會自動幫你把改變的 YAML 檔 `git commit` & `git push` 合併回主分支中，確保單一真相來源 (GitOps)。

---

## 🗺️ Kafka 授權視覺化儀表板

我們提供了一支自動化的 Python 腳本 (`scripts/generate_beautiful_dashboard.py`)，它能夠讀取所有的 `tenants/` 檔案並渲染出一份具有超快體驗、美觀且可互動的網頁分析圖表。

* 執行方式：`python scripts/generate_beautiful_dashboard.py`
* 該腳本支援多環境過濾、隱藏式效能加速，所產生的 **`kafka_permissions_dashboard.html`** 可以被放進任何 Web Server（如後續搭配 GitLab CI 的 Pages）上作為內部監控大盤。
