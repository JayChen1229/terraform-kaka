#!/bin/bash

# ==============================================================================
# 🛠️ 參數設定區 (請務必填寫你自己的真實資訊)
# ==============================================================================
# 1. 目標環境 (dev, sit, uat, prod)
ENV="dev"

# 2. 自動化行為控制 (true/false)
AUTO_APPLY="true"    # 是否自動執行 Terraform Apply？(設為 false 則只會跑 Plan)

# 3. GitLab 驗證資訊 (Personal Access Token)
GITLAB_USER=""
GITLAB_TOKEN=""

# 4. Kafka Admin 驗證資訊
KAFKA_ADMIN_USER=""
KAFKA_ADMIN_PASS=""

# 5. SOPS 解密金鑰
SOPS_KEY=""

# ==============================================================================
# 系統變數 (通常不需要改)
# ==============================================================================
REPO_URL="https://${GITLAB_USER}:${GITLAB_TOKEN}@gitlab/group/project.git"
DIR_NAME="terraform"
ENV_LOWER=$(echo "$ENV" | tr '[:upper:]' '[:lower:]')
ENV_UPPER=$(echo "$ENV" | tr '[:lower:]' '[:upper:]')
GITLAB_PROJECT_ID="up0109%2Fterraform"

echo "================================================="
echo "🚀 啟動 Terraform 本地全自動佈署流程 - 環境: $ENV_UPPER"
echo "================================================="

# ------------------------------------------------------------------------------
# Step 1: Git 同步 (保留本地修改)
# ------------------------------------------------------------------------------
echo -e "\n📦 [Step 1] 正在與 GitLab 同步最新程式碼..."
if [ ! -d "$DIR_NAME" ]; then
    echo "   找不到本地專案，正在 Clone..."
    git clone "$REPO_URL" "$DIR_NAME"
    cd "$DIR_NAME" || exit
else
    echo "   專案已存在，正在同步雲端最新程式碼 (保留本地已修改的 YAML)..."
    cd "$DIR_NAME" || exit
    git remote set-url origin "$REPO_URL"
    # 使用 autostash 可以在拉取遠端程式碼時，暫存並保護你本地尚未 commit 的修改
    git pull origin main --rebase --autostash
fi
echo "   ✅ Git 同步完成！"

# ------------------------------------------------------------------------------
# Step 2: 載入本地變更
# ------------------------------------------------------------------------------
echo -e "\n📂 [Step 2] 載入本地變更..."
echo "   將直接套用你目前目錄下 (tenants/) 已經修改好的 YAML 設定。"

# ------------------------------------------------------------------------------
# Step 3: 設定 Terraform 與 GitLab HTTP Backend 環境變數
# ------------------------------------------------------------------------------
echo -e "\n⚙️  [Step 3] 注入環境變數 (Export)..."
export TF_HTTP_ADDRESS="https://gilab/api/v4/projects/${GITLAB_PROJECT_ID}/terraform/state/confluent-${ENV_LOWER}"
export TF_HTTP_LOCK_ADDRESS="${TF_HTTP_ADDRESS}/lock"
export TF_HTTP_UNLOCK_ADDRESS="${TF_HTTP_ADDRESS}/lock"
export TF_HTTP_LOCK_METHOD="POST"
export TF_HTTP_UNLOCK_METHOD="DELETE"
export TF_HTTP_USERNAME="${GITLAB_USER}"
export TF_HTTP_PASSWORD="${GITLAB_TOKEN}"

export TF_VAR_kafka_sasl_username="${KAFKA_ADMIN_USER}"
export TF_VAR_kafka_sasl_password="${KAFKA_ADMIN_PASS}"
export SOPS_AGE_KEY="${SOPS_KEY}"
echo "   ✅ 環境變數設定完畢！"

# ------------------------------------------------------------------------------
# Step 4: 執行 Terraform (Init, Validate, Plan)
# ------------------------------------------------------------------------------
echo -e "\n🏗️  [Step 4] 執行 Terraform 流程..."

echo "   > 修復本地 Provider 套件執行權限..."
chmod -R +x terraform-provider-plugins/ .terraform/ 2>/dev/null || true

echo "   > terraform init"
terraform init -plugin-dir="$(pwd)/terraform-provider-plugins"

echo -e "\n   > terraform validate"
terraform validate

echo -e "\n   > terraform plan"
terraform plan -var-file="envs/${ENV_UPPER}.tfvars" -out=tfplan

# ------------------------------------------------------------------------------
# Step 5: 自動 Apply
# ------------------------------------------------------------------------------
echo -e "\n================================================="
if [ "$AUTO_APPLY" = "true" ]; then
    echo "🚀 AUTO_APPLY 設為 true，開始執行 Terraform Apply..."
    if terraform apply -input=false tfplan; then
        echo "   🎉 佈署完成！"
        echo "   ⏭️ 溫馨提醒：請記得將修改好的 YAML 檔案手動 Commit 並推上 GitLab 喔！"
    else
        echo "   ❌ Terraform Apply 失敗！請檢查上方的錯誤訊息。"
        exit 1
    fi
else
    echo "🛑 AUTO_APPLY 設為 false。僅產出 Plan 結果，不套用變更。"
fi
echo "================================================="
