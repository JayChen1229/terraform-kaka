#!/bin/bash

# ===============================================================
# SOPS Migration Helper Script (Local Encryption)
# 用途：當你在 .sops.yaml 填入公鑰後，利用此腳本批次加密現有明碼檔案
# ===============================================================

set -e

echo "🔒 準備遷移到 SOPS..."

if grep -q "age1your_public_key_here_please_replace_me" ".sops.yaml"; then
    echo "❌ 發現 .sops.yaml 裡的 age 公鑰還是預設值！"
    echo "請先在你的 Terraform VM 上產生好金鑰，並把拿到的公鑰填入 .sops.yaml 裡。"
    exit 1
fi

# 加密現有 YAML 檔案
echo "🔐 正在加密所有 tenants 下的 YAML 檔案..."
shopt -s globstar 2>/dev/null || true
for file in tenants/**/*.yaml; do
    if [ -f "$file" ]; then
        if grep -q "sops:" "$file"; then
             echo "   ⏭️  [跳過] 已經被加密: $file"
        else
             sops --encrypt --in-place "$file"
             echo "   ✅  [完成] 已加密: $file (僅加密 password 欄位)"
        fi
    fi
done

echo "🎉 所有設定已完成！"
echo ""
echo "👉 下一步："
echo "1. 測試解密：執行 'sops -d tenants/DEV/UP9999.yaml' 看看是否成功"
echo "2. Jenkins 設定：將 $KEY_FILE 的內容（完整字串）"
echo "   加入至 Jenkins 的 Secret Text 憑證中，變數名稱設為 SOPS_AGE_KEY"
echo "   並在 Jenkinsfile Terraform 執行時 export SOPS_AGE_KEY"
