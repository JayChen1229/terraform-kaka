pipeline {
    agent any

    // 加入這個 options 區塊
    options {
        ansiColor('xterm') // 告訴 Jenkins 使用 xterm 終端機的顏色對應表
    }

    parameters {
        choice(name: 'ENV', choices: ['DEV', 'SIT', 'UAT', 'PROD'], description: '請選擇要執行的環境')

        // ---------- 操作類型 ----------
        choice(name: 'TYPE', choices: [
            '',
            'createTransactionTopic',
            'createAuditTopic',
            'createUser',
            'createProducerACL',
            'createConsumerACL',
            'alterTopicPartition',
            'alterTopicSetting',
            'deleteTransactionTopic',
            'deleteUser',
            'deleteProducerACL',
            'deleteConsumerGroupACL'
        ], description: '操作類型 (留空表示不在此 pipeline 修改 YAML，直接跑 Terraform)')

        // ---------- 操作相關參數 ----------
        string(name: 'ISSUE_KEY',          defaultValue: '', description: 'Jira Issue Key，例如: KAFKA-123')
        string(name: 'USER_ACCOUNT',       defaultValue: '', description: 'Kafka 使用者帳號')
        string(name: 'USER_PASSWORD',      defaultValue: '', description: 'Kafka 使用者密碼 (createUser 時使用)')
        string(name: 'TOPIC_NAME',         defaultValue: '', description: 'Topic 名稱')
        string(name: 'PARTITIONS',         defaultValue: '', description: 'Partition 數量')
        string(name: 'REPLICATION_FACTOR', defaultValue: '', description: 'Replication Factor (不填則依環境預設)')
        string(name: 'GROUP_ID',           defaultValue: '', description: 'Consumer Group ID (createConsumerACL / deleteConsumerGroupACL 使用)')
        /*
            alterTopicSetting 的 ALTER_SETTING 使用 key=value 格式，多個設定以逗號分隔，例如：
                retention.ms=604800000,max.message.bytes=1048576
        */
        string(name: 'ALTER_SETTING', defaultValue: '', description: 'Topic 進階設定 (key=value,key=value)，常用參數：\n  retention.ms       — 資料保留時間 (ms)，例如 604800000 = 7天\n  retention.bytes    — 每個 Partition 最大保留 bytes，例如 1073741824 = 1GB\n  max.message.bytes  — 單則訊息大小上限 (bytes)，例如 1048576 = 1MB\n  cleanup.policy     — 清理策略，delete (預設) 或 compact\n  min.insync.replicas — 最少同步副本數，建議 PROD 設為 2\n  segment.bytes      — Segment 檔案大小，例如 1073741824 = 1GB')
    }

    environment {
        GITLAB_PROJECT_ID  = 'test-group5015415%2Fterraform'
        GITLAB_REPO_URL    = 'https://gitlab.com/test-group5015415/terraform.git'
        TF_VAR_environment = "${params.ENV}"
        TENANT_YAML        = "tenants/${params.ENV}/${params.USER_ACCOUNT}.yaml"
        ENV_LOWER          = "${params.ENV.toLowerCase()}"
    }

    stages {
        // =====================================================================
        // Stage 1: Checkout
        // =====================================================================
        stage('Checkout') {
            steps {
                checkout scm
                // 強制清除可能從上次中斷殘留下來的髒資料（但保留 .terraform 快取加速建構）
                sh 'git restore . && git clean -fd -e .terraform/'
            }
        }

        // =====================================================================
        // Stage 2: Mutate YAML (依 TYPE 操作對應的 tenant YAML)
        // 完整流程: yq 修改 → git commit → (Terraform執行後) git push
        // =====================================================================
        stage('Mutate YAML') {
            when {
                expression { params.TYPE != '' }
            }
            steps {
                withCredentials([
                    usernamePassword(credentialsId: 'GitLab', usernameVariable: 'GIT_USERNAME', passwordVariable: 'GIT_PASSWORD')
                ]) {
                    script {
                        def type        = params.TYPE
                        def env         = params.ENV
                        def userAccount = params.USER_ACCOUNT
                        def yamlPath    = "tenants/${env}/${userAccount}.yaml"

                        echo "🔧 TYPE=${type} | ENV=${env} | USER=${userAccount}"

                        switch (type) {

                            // --------------------------------------------------
                            // 建立 Transaction / Audit Topic
                            // --------------------------------------------------
                            case 'createTransactionTopic':
                            case 'createAuditTopic':
                                validateParams(['USER_ACCOUNT', 'TOPIC_NAME', 'PARTITIONS'])
                                ensureYamlExists(yamlPath, userAccount)

                                def partitions = params.PARTITIONS.toInteger()
                                def rfFlag     = params.REPLICATION_FACTOR?.trim() ?
                                    ".config.\"replication.factor\" = \"${params.REPLICATION_FACTOR}\" |" : ''

                                sh """
                                    yq -i '.manage_topics."${params.TOPIC_NAME}".partitions = ${partitions}' "${yamlPath}"
                                    echo "  ✅ Topic ${params.TOPIC_NAME} 已加入 manage_topics (partitions=${partitions})"
                                """
                                break

                            // --------------------------------------------------
                            // 建立使用者 (新增整個 YAML)
                            // --------------------------------------------------
                            case 'createUser':
                                validateParams(['USER_ACCOUNT', 'USER_PASSWORD'])

                                if (fileExists(yamlPath)) {
                                    error("⛔ 使用者 ${userAccount} 的 YAML 已存在！YAML Path: ${yamlPath}")
                                }

                                sh """
                                    mkdir -p \$(dirname "${yamlPath}")
                                    cat > "${yamlPath}" << 'YAML_EOF'
# =======================
# Tenant: ${userAccount}
# =======================
password: ${params.USER_PASSWORD}
manage_topics: {}
extra_read_topics: []
extra_write_topics: []
YAML_EOF
                                    echo "  ✅ 已建立使用者 YAML 原始檔: ${yamlPath}"

                                    # 呼叫 SOPS 對密碼欄位進行加密
                                    sops --encrypt --in-place "${yamlPath}"
                                    echo "  🔒 已透過 SOPS 加密使用者密碼: ${yamlPath}"
                                """
                                break

                            // --------------------------------------------------
                            // 建立 Producer ACL (新增 extra_write_topics)
                            // --------------------------------------------------
                            case 'createProducerACL':
                                validateParams(['USER_ACCOUNT', 'TOPIC_NAME'])
                                ensureYamlExists(yamlPath, userAccount)

                                sh """
                                    # 若 extra_write_topics 內還沒有此 topic 才新增
                                    HAS=\$(yq '.extra_write_topics // [] | contains(["${params.TOPIC_NAME}"])' "${yamlPath}")
                                    if [ "\$HAS" = "false" ]; then
                                        yq -i '.extra_write_topics += ["${params.TOPIC_NAME}"]' "${yamlPath}"
                                        echo "  ✅ Producer ACL: ${params.TOPIC_NAME} 已加入 extra_write_topics"
                                    else
                                        echo "  ⚠️ Producer ACL: ${params.TOPIC_NAME} 已存在，跳過"
                                    fi
                                """
                                break

                            // --------------------------------------------------
                            // 建立 Consumer ACL (新增 extra_read_topics)
                            // --------------------------------------------------
                            case 'createConsumerACL':
                                validateParams(['USER_ACCOUNT', 'TOPIC_NAME', 'GROUP_ID'])
                                ensureYamlExists(yamlPath, userAccount)

                                sh """
                                    HAS=\$(yq '.extra_read_topics // [] | contains(["${params.TOPIC_NAME}"])' "${yamlPath}")
                                    if [ "\$HAS" = "false" ]; then
                                        yq -i '.extra_read_topics += ["${params.TOPIC_NAME}"]' "${yamlPath}"
                                        echo "  ✅ Consumer ACL: ${params.TOPIC_NAME} 已加入 extra_read_topics"
                                    else
                                        echo "  ⚠️ Consumer ACL: ${params.TOPIC_NAME} 已存在，跳過"
                                    fi
                                """
                                break

                            // --------------------------------------------------
                            // 調整 Topic Partition 數量
                            // --------------------------------------------------
                            case 'alterTopicPartition':
                                validateParams(['USER_ACCOUNT', 'TOPIC_NAME', 'PARTITIONS'])
                                ensureYamlExists(yamlPath, userAccount)

                                def newPartitions = params.PARTITIONS.toInteger()
                                sh """
                                    # 取得目前的 partitions
                                    CURRENT=\$(yq '.manage_topics."${params.TOPIC_NAME}".partitions' "${yamlPath}")
                                    echo "  目前 partitions: \$CURRENT → 新值: ${newPartitions}"

                                    if [ "${newPartitions}" -le "\$CURRENT" ]; then
                                        echo "  ⛔ 新的 partition 數 (${newPartitions}) 必須大於目前值 (\$CURRENT)！"
                                        exit 1
                                    fi

                                    yq -i '.manage_topics."${params.TOPIC_NAME}".partitions = ${newPartitions}' "${yamlPath}"
                                    echo "  ✅ Partition 已更新為 ${newPartitions}"
                                """
                                break

                            // --------------------------------------------------
                            // 調整 Topic 進階設定 (ALTER_SETTING)
                            // 格式: retention.ms=604800000,max.message.bytes=1048576
                            // --------------------------------------------------
                            case 'alterTopicSetting':
                                validateParams(['USER_ACCOUNT', 'TOPIC_NAME', 'ALTER_SETTING'])
                                ensureYamlExists(yamlPath, userAccount)

                                // 在 Groovy 層拆解 key=value，再逐一呼叫 yq
                                // 避免 Shell/Groovy 字串插值混用的 MissingPropertyException
                                params.ALTER_SETTING.split(',').each { kv ->
                                    def trimmed = kv.trim()
                                    def eqIdx   = trimmed.indexOf('=')
                                    if (eqIdx > 0) {
                                        def cfgKey = trimmed.substring(0, eqIdx).trim()
                                        def cfgVal = trimmed.substring(eqIdx + 1).trim()
                                        sh """yq -i '.manage_topics."${params.TOPIC_NAME}".config."${cfgKey}" = "${cfgVal}"' "${yamlPath}" """
                                        echo "    ✅ config.${cfgKey} = ${cfgVal}"
                                    }
                                }
                                echo "  ✅ Topic 設定已全部更新"
                                break

                            // --------------------------------------------------
                            // 刪除 Transaction Topic (從 manage_topics 移除)
                            // --------------------------------------------------
                            case 'deleteTransactionTopic':
                                validateParams(['USER_ACCOUNT', 'TOPIC_NAME'])
                                ensureYamlExists(yamlPath, userAccount)

                                sh """
                                    yq -i 'del(.manage_topics."${params.TOPIC_NAME}")' "${yamlPath}"
                                    echo "  ✅ Topic ${params.TOPIC_NAME} 已從 manage_topics 移除"
                                """
                                break

                            // --------------------------------------------------
                            // 刪除使用者 (刪除整個 YAML 檔)
                            // --------------------------------------------------
                            case 'deleteUser':
                                validateParams(['USER_ACCOUNT'])
                                ensureYamlExists(yamlPath, userAccount)

                                sh """
                                    rm "${yamlPath}"
                                    echo "  ✅ 使用者 YAML 已刪除: ${yamlPath}"
                                """
                                break

                            // --------------------------------------------------
                            // 刪除 Producer ACL (從 extra_write_topics 移除)
                            // --------------------------------------------------
                            case 'deleteProducerACL':
                                validateParams(['USER_ACCOUNT', 'TOPIC_NAME'])
                                ensureYamlExists(yamlPath, userAccount)

                                sh """
                                    yq -i '.extra_write_topics = (.extra_write_topics // [] | map(select(. != "${params.TOPIC_NAME}")))' "${yamlPath}"
                                    echo "  ✅ Producer ACL: ${params.TOPIC_NAME} 已從 extra_write_topics 移除"
                                """
                                break

                            // --------------------------------------------------
                            // 刪除 Consumer Group ACL (從 extra_read_topics 移除)
                            // --------------------------------------------------
                            case 'deleteConsumerGroupACL':
                                validateParams(['USER_ACCOUNT', 'TOPIC_NAME', 'GROUP_ID'])
                                ensureYamlExists(yamlPath, userAccount)

                                sh """
                                    yq -i '.extra_read_topics = (.extra_read_topics // [] | map(select(. != "${params.TOPIC_NAME}")))' "${yamlPath}"
                                    echo "  ✅ Consumer ACL: ${params.TOPIC_NAME} 已從 extra_read_topics 移除"
                                """
                                break

                            default:
                                error("❌ 不支援的 TYPE: ${type}")
                        }

                        // 暫存變更到 git (真正 push 在 Terraform Apply 後)
                        sh """
                            git config user.email 'jenkins@ci.local'
                            git config user.name  'Jenkins CI'
                            git add tenants/
                            git diff --cached --quiet || git commit -m "[${params.ISSUE_KEY}] ${type}: ${userAccount} (${env})"
                        """
                    }
                }
            }
        }

        // =====================================================================
        // Stage 3: Terraform Init
        // =====================================================================
        stage('Terraform Init') {
            steps {
                withCredentials([
                    usernamePassword(credentialsId: 'GitLab',      usernameVariable: 'TF_HTTP_USERNAME', passwordVariable: 'TF_HTTP_PASSWORD'),
                    usernamePassword(credentialsId: 'kafka-admin',  usernameVariable: 'KAFKA_ADMIN_USER', passwordVariable: 'KAFKA_ADMIN_PASS'),
                    string(credentialsId: 'sops-age-key', variable: 'SOPS_AGE_KEY')
                ]) {
                    sh """
                        echo "====================================="
                        echo "🔧 Terraform Init — ENV: ${params.ENV}"
                        echo "====================================="

                        export TF_HTTP_ADDRESS="https://gitlab.com/api/v4/projects/\${GITLAB_PROJECT_ID}/terraform/state/confluent-${params.ENV.toLowerCase()}"
                        export TF_HTTP_LOCK_ADDRESS="\${TF_HTTP_ADDRESS}/lock"
                        export TF_HTTP_UNLOCK_ADDRESS="\${TF_HTTP_ADDRESS}/lock"
                        export TF_HTTP_LOCK_METHOD="POST"
                        export TF_HTTP_UNLOCK_METHOD="DELETE"

                        export TF_VAR_kafka_sasl_username="\${KAFKA_ADMIN_USER}"
                        export TF_VAR_kafka_sasl_password="\${KAFKA_ADMIN_PASS}"

                        terraform init -input=false
                    """
                }
            }
        }

        // =====================================================================
        // Stage 4: Terraform Validate
        // =====================================================================
        stage('Terraform Validate') {
            steps {
                sh '''
                    echo "✅ Validating Terraform configuration..."
                    terraform validate
                '''
            }
        }

        // =====================================================================
        // Stage 5: Terraform Plan
        // =====================================================================
        stage('Terraform Plan') {
            steps {
                withCredentials([
                    usernamePassword(credentialsId: 'GitLab',      usernameVariable: 'TF_HTTP_USERNAME', passwordVariable: 'TF_HTTP_PASSWORD'),
                    usernamePassword(credentialsId: 'kafka-admin',  usernameVariable: 'KAFKA_ADMIN_USER', passwordVariable: 'KAFKA_ADMIN_PASS'),
                    string(credentialsId: 'sops-age-key', variable: 'SOPS_AGE_KEY')
                ]) {
                    sh """
                        echo "📋 Running Terraform Plan..."

                        export TF_HTTP_ADDRESS="https://gitlab.com/api/v4/projects/\${GITLAB_PROJECT_ID}/terraform/state/confluent-${params.ENV.toLowerCase()}"
                        export TF_HTTP_LOCK_ADDRESS="\${TF_HTTP_ADDRESS}/lock"
                        export TF_HTTP_UNLOCK_ADDRESS="\${TF_HTTP_ADDRESS}/lock"
                        export TF_HTTP_LOCK_METHOD="POST"
                        export TF_HTTP_UNLOCK_METHOD="DELETE"

                        export TF_VAR_kafka_sasl_username="\${KAFKA_ADMIN_USER}"
                        export TF_VAR_kafka_sasl_password="\${KAFKA_ADMIN_PASS}"

                        terraform plan \\
                            -var-file="envs/${params.ENV}.tfvars" \\
                            -out=tfplan \\
                            -input=false
                    """
                }
            }
        }

        // =====================================================================
        // Stage 6: Terraform Apply
        // =====================================================================
        stage('Terraform Apply') {
            steps {
                withCredentials([
                    usernamePassword(credentialsId: 'GitLab',      usernameVariable: 'TF_HTTP_USERNAME', passwordVariable: 'TF_HTTP_PASSWORD'),
                    usernamePassword(credentialsId: 'kafka-admin',  usernameVariable: 'KAFKA_ADMIN_USER', passwordVariable: 'KAFKA_ADMIN_PASS'),
                    string(credentialsId: 'sops-age-key', variable: 'SOPS_AGE_KEY')
                ]) {
                    sh """
                        echo "🚀 Applying Terraform changes on ${params.ENV}..."

                        export TF_HTTP_ADDRESS="https://gitlab.com/api/v4/projects/\${GITLAB_PROJECT_ID}/terraform/state/confluent-${params.ENV.toLowerCase()}"
                        export TF_HTTP_LOCK_ADDRESS="\${TF_HTTP_ADDRESS}/lock"
                        export TF_HTTP_UNLOCK_ADDRESS="\${TF_HTTP_ADDRESS}/lock"
                        export TF_HTTP_LOCK_METHOD="POST"
                        export TF_HTTP_UNLOCK_METHOD="DELETE"

                        export TF_VAR_kafka_sasl_username="\${KAFKA_ADMIN_USER}"
                        export TF_VAR_kafka_sasl_password="\${KAFKA_ADMIN_PASS}"

                        terraform apply -input=false tfplan
                    """
                }
            }
        }

        // =====================================================================
        // Stage 7: Git Push (Apply 成功後才推回 GitLab)
        // =====================================================================
        stage('Git Push') {
            when {
                expression { params.TYPE != '' }
            }
            steps {
                withCredentials([
                    usernamePassword(credentialsId: 'GitLab', usernameVariable: 'GIT_USERNAME', passwordVariable: 'GIT_PASSWORD')
                ]) {
                    sh """
                        echo "📤 推送 YAML 變更回 GitLab..."
                        git remote set-url origin https://\${GIT_USERNAME}:\${GIT_PASSWORD}@gitlab.com/test-group5015415/terraform.git
                        git push origin HEAD:main
                        echo "  ✅ 推送成功"
                    """
                }
            }
        }

    }

    post {
        always {
            echo "🏁 Pipeline 執行完畢 — ENV: ${params.ENV} | TYPE: ${params.TYPE}"
            // 移除 cleanWs() 以保留 .terraform 套件快取，並透過剛開始的 checkout stage 來清理髒環境
        }
        success {
            echo "✅ 成功！"
        }
        failure {
            echo "❌ 失敗！請檢查 log。"
        }
    }
}

// =============================================================================
// Helper Functions
// =============================================================================

/** 驗證必填參數（未填時直接 abort pipeline） */
def validateParams(List<String> paramNames) {
    paramNames.each { name ->
        def val = params[name]?.trim()
        if (!val) {
            error("⛔ 必填參數 [${name}] 未提供！")
        }
    }
}

/**
 * 確認使用者的 YAML 存在。
 * 如果不存在，提示錯誤訊息引導先執行 createUser。
 */
def ensureYamlExists(String yamlPath, String userAccount) {
    if (!fileExists(yamlPath)) {
        error("⛔ 找不到 ${yamlPath}！\n請先執行 TYPE=createUser 建立使用者 ${userAccount} 的設定檔。")
    }
}