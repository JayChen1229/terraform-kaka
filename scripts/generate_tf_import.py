import sys
import os

if len(sys.argv) < 3:
    print("Usage: python3 generate_tf_import.py <all_topics.txt> <all_acls.txt>")
    print("Example: python3 generate_tf_import.py all_topics.txt all_acls.txt")
    sys.exit(1)

topics_file = sys.argv[1]
acls_file = sys.argv[2]

tenants = {}

def get_tenant(user):
    if user not in tenants:
        tenants[user] = {
            "manage_topics": {},
            "extra_read_topics": set(),
            "extra_write_topics": set()
        }
    return tenants[user]

# ==========================================
# 1️⃣ 解析 Topics (與之前相同)
# ==========================================
print("1️⃣ 解析 Topics...")
with open(topics_file, 'r', encoding='utf-8') as f:
    for line in f:
        if line.startswith('\t'): continue
        if not line.startswith('Topic:'): continue

        parts = line.split('\t')
        topic_name = ""
        partitions = 1
        configs = {}

        for p in parts:
            p = p.strip()
            if p.startswith("Topic:"):
                topic_name = p.replace("Topic:", "").strip()
            elif p.startswith("PartitionCount:"):
                partitions = int(p.replace("PartitionCount:", "").strip())
            elif p.startswith("Configs:"):
                cfg_str = p.replace("Configs:", "").strip()
                if cfg_str:
                    for kv in cfg_str.split(','):
                        if '=' in kv:
                            k, v = kv.split('=', 1)
                            if k not in ["min.insync.replicas"]:
                                configs[k] = v

        if not topic_name or topic_name.startswith('_') or topic_name in ['confluent-audit-log-events']:
            continue

        prefix = topic_name.split('_')[0]
        t = get_tenant(prefix)
        t["manage_topics"][topic_name] = {
            "partitions": partitions,
            "config": configs
        }


# ==========================================
# 2️⃣ 解析 ACLs (新增的邏輯)
# ==========================================
print("2️⃣ 解析 ACLs...")
current_topic = None

with open(acls_file, 'r', encoding='utf-8') as f:
    for line in f:
        line = line.strip()
        if not line:
            continue
            
        # 抓取目前正在解析的 Topic 名稱
        # 格式範例: Current ACLs for resource `Topic:LITERAL:CH0025_RECENTPAYEEACCOUNT_00_1`:
        if line.startswith("Current ACLs for resource `Topic:LITERAL:"):
            # 擷取兩個反引號 (`) 之間，且去掉 "Topic:LITERAL:" 的字串
            current_topic = line.split("`Topic:LITERAL:")[1].split("`:")[0]
            continue
            
        # 抓取 User 權限
        # 格式範例: User:CH0018 has Allow permission for operations: Write from hosts: *
        if current_topic and line.startswith("User:"):
            # 取出 User 名稱
            user_part = line.split(" has Allow permission for operations: ")[0]
            current_user = user_part.replace("User:", "").strip()
            
            # 取出 Operation (Read, Write, Create, Describe)
            op_part = line.split(" has Allow permission for operations: ")[1]
            operation = op_part.split(" from hosts:")[0].strip()
            
            # 判斷是否為跨權限 (User 名稱不等於 Topic 的 Prefix)
            topic_owner = current_topic.split('_')[0]
            
            if current_user != topic_owner:
                t = get_tenant(current_user)
                if operation == "Read":
                    t["extra_read_topics"].add(current_topic)
                elif operation == "Write":
                    t["extra_write_topics"].add(current_topic)


# ==========================================
# 3️⃣ 產生 YAML 設定檔
# ==========================================
print("3️⃣ 產生 YAML 設定檔...")
output_dir = "tf_export/tenants/DEV"
os.makedirs(output_dir, exist_ok=True)

for user, data in tenants.items():
    yaml_path = f"{output_dir}/{user}.yaml"
    with open(yaml_path, "w", encoding='utf-8') as yf:
        yf.write(f"password: \"PLEASE_CHANGE_ME\"\n")

        # 寫入 manage_topics
        if data["manage_topics"]:
            yf.write(f"manage_topics:\n")
            # 為了讓 YAML 比較整齊，按 topic_name 排序
            for tname in sorted(data["manage_topics"].keys()):
                tcfg = data["manage_topics"][tname]
                yf.write(f"  {tname}:\n")
                yf.write(f"    partitions: {tcfg['partitions']}\n")
                if tcfg["config"]:
                    yf.write(f"    config:\n")
                    for k, v in tcfg["config"].items():
                        yf.write(f"      {k}: \"{v}\"\n")
        else:
            yf.write("manage_topics: {}\n")

        # 寫入 extra_read_topics
        if data["extra_read_topics"]:
            yf.write(f"extra_read_topics:\n")
            # 使用 sorted 讓輸出的陣列排序固定
            for tname in sorted(list(data["extra_read_topics"])):
                yf.write(f"  - {tname}\n")
        else:
            yf.write("extra_read_topics: []\n")

        # 寫入 extra_write_topics
        if data["extra_write_topics"]:
            yf.write(f"extra_write_topics:\n")
            for tname in sorted(list(data["extra_write_topics"])):
                yf.write(f"  - {tname}\n")
        else:
            yf.write("extra_write_topics: []\n")

print(f"✅ 轉換成功！請到 {output_dir} 資料夾查看 YAML 結果。")
