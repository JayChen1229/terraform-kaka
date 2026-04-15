#!/usr/bin/env bash
# =============================================================================
# apply_prod.sh — 營運機 PROD 環境 Terraform 執行腳本
# =============================================================================
# 使用方式:
#   1. 先在 Jenkins 執行 pipeline (ENV=PROD) 完成 YAML commit & push
#   2. 到營運機執行此腳本:
#
#        export KAFKA_ADMIN_USER="your-admin-user"
#        export KAFKA_ADMIN_PASS="your-admin-password"
#        export SOPS_AGE_KEY="AGE-SECRET-KEY-..."
#        export GITLAB_TOKEN="your-gitlab-token"
#        export GITLAB_USERNAME="your-username"
#
#        bash scripts/apply_prod.sh
#
#   或者如果尚未 clone:
#
#        git clone https://gitlab.com/test-group5015415/terraform.git
#        cd terraform
#        bash scripts/apply_prod.sh
# =============================================================================

set -euo pipefail

# ── 顏色定義 ──────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ── 函式 ──────────────────────────────────────────────────────────────────────
info()    { echo -e "${BLUE}ℹ️  $*${NC}"; }
success() { echo -e "${GREEN}✅ $*${NC}"; }
warn()    { echo -e "${YELLOW}⚠️  $*${NC}"; }
error()   { echo -e "${RED}❌ $*${NC}" >&2; exit 1; }

# ── 環境變數檢查 ──────────────────────────────────────────────────────────────
info "檢查必要環境變數..."

REQUIRED_VARS=(KAFKA_ADMIN_USER KAFKA_ADMIN_PASS SOPS_AGE_KEY GITLAB_TOKEN GITLAB_USERNAME)
MISSING=()
for var in "${REQUIRED_VARS[@]}"; do
    if [[ -z "${!var:-}" ]]; then
        MISSING+=("$var")
    fi
done

if [[ ${#MISSING[@]} -gt 0 ]]; then
    error "缺少以下環境變數:\n$(printf '  - %s\n' "${MISSING[@]}")\n\n請先 export 後再執行此腳本。"
fi
success "環境變數檢查通過"

# ── 固定參數 ──────────────────────────────────────────────────────────────────
ENV="PROD"
GITLAB_PROJECT_ID="test-group5015415%2Fterraform"

# ── 確保在 repo 根目錄 ────────────────────────────────────────────────────────
if [[ ! -f "versions.tf" ]]; then
    error "請在 terraform repo 根目錄下執行此腳本！\n  cd /path/to/terraform && bash scripts/apply_prod.sh"
fi

# ── 拉最新程式碼 ──────────────────────────────────────────────────────────────
info "拉取最新程式碼 (git pull)..."
git pull origin main
success "程式碼已更新"

# ── 設定 Terraform Backend 環境變數 ──────────────────────────────────────────
export TF_HTTP_ADDRESS="https://gitlab.com/api/v4/projects/${GITLAB_PROJECT_ID}/terraform/state/confluent-prod"
export TF_HTTP_LOCK_ADDRESS="${TF_HTTP_ADDRESS}/lock"
export TF_HTTP_UNLOCK_ADDRESS="${TF_HTTP_ADDRESS}/lock"
export TF_HTTP_LOCK_METHOD="POST"
export TF_HTTP_UNLOCK_METHOD="DELETE"
export TF_HTTP_USERNAME="${GITLAB_USERNAME}"
export TF_HTTP_PASSWORD="${GITLAB_TOKEN}"

# ── 設定 Terraform Kafka Provider 變數 ───────────────────────────────────────
export TF_VAR_environment="${ENV}"
export TF_VAR_kafka_sasl_username="${KAFKA_ADMIN_USER}"
export TF_VAR_kafka_sasl_password="${KAFKA_ADMIN_PASS}"

echo ""
echo -e "${BLUE}=====================================${NC}"
echo -e "${BLUE}  🏭 PROD Terraform 部署${NC}"
echo -e "${BLUE}=====================================${NC}"
echo ""

# ── Terraform Init ────────────────────────────────────────────────────────────
info "Terraform Init..."
terraform init -input=false -plugin-dir="$(pwd)/terraform-provider-plugins"
success "Init 完成"

# ── Terraform Validate ────────────────────────────────────────────────────────
info "Terraform Validate..."
terraform validate
success "Validate 通過"

# ── Terraform Plan ────────────────────────────────────────────────────────────
info "Terraform Plan..."
terraform plan \
    -var-file="envs/${ENV}.tfvars" \
    -out=tfplan \
    -input=false

echo ""
echo -e "${YELLOW}=====================================${NC}"
echo -e "${YELLOW}  ⚠️  請確認以上 Plan 內容${NC}"
echo -e "${YELLOW}=====================================${NC}"
echo ""

# ── 確認後再 Apply ────────────────────────────────────────────────────────────
read -rp "$(echo -e "${YELLOW}確定要 Apply 到 PROD 嗎？(yes/no): ${NC}")" CONFIRM

if [[ "${CONFIRM}" != "yes" ]]; then
    warn "已取消 Apply。"
    # 清理 plan 檔
    rm -f tfplan
    exit 0
fi

# ── Terraform Apply ───────────────────────────────────────────────────────────
info "Terraform Apply..."
terraform apply -input=false tfplan
success "Apply 完成！PROD 環境已更新。"

# ── 清理 ──────────────────────────────────────────────────────────────────────
rm -f tfplan
