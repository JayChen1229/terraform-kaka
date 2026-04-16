# 安裝 terraform
mv terraform /usr/bin/
chmod 755 /usr/bin/terraform
terraform --version

# 安裝 age
tar -xvf age-v1.3.1-linux-amd64.tar.gz
mv age/age age/age-keygen /usr/bin/
chmod 755 /usr/bin/age /usr/bin/age-keygen
age --version

# 安裝 sops
mv sops-v3.12.1.linux.amd64 /usr/bin/sops
chmod 755 /usr/bin/sops
sops --version

# 安裝 yq
mv yq_linux_amd64 /usr/bin/yq
chmod 755 /usr/bin/yq
yq --version 

# 生成加密金鑰
mkdir -p ~/.config/sops/age
age-keygen ~/.config/sops/age/keys.txt 
git config --global http.postBuffer 524288000
