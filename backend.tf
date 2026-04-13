# =============================================================================
# GitLab HTTP Backend
# =============================================================================
# Jenkins 需要設定以下環境變數：
#   TF_HTTP_ADDRESS       = https://gitlab.com/api/v4/projects/<PROJECT_ID>/terraform/state/<STATE_NAME>
#   TF_HTTP_LOCK_ADDRESS  = https://gitlab.com/api/v4/projects/<PROJECT_ID>/terraform/state/<STATE_NAME>/lock
#   TF_HTTP_UNLOCK_ADDRESS= https://gitlab.com/api/v4/projects/<PROJECT_ID>/terraform/state/<STATE_NAME>/lock
#   TF_HTTP_USERNAME      = gitlab-ci-token (或你的 GitLab username)
#   TF_HTTP_PASSWORD      = <GitLab Personal Access Token or CI Job Token>
#
# 範例 (Jenkins pipeline 中):
#   withCredentials([string(credentialsId: 'gitlab-tf-token', variable: 'TF_HTTP_PASSWORD')]) {
#     sh 'terraform init'
#   }
# =============================================================================

terraform {
  backend "http" {}
}
