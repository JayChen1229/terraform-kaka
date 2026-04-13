import os
import yaml
import json

base_dir = "tf_export/tenants"
output_file = "kafka_permissions_dashboard.html"

if not os.path.exists(base_dir):
    print(f"❌ 找不到目錄 {base_dir}")
    exit(1)

envs = [d for d in os.listdir(base_dir) if os.path.isdir(os.path.join(base_dir, d))]
env_order = {"DEV": 1, "SIT": 2, "UAT": 3, "PROD": 4}
envs.sort(key=lambda x: env_order.get(x.upper(), 99))

if not envs:
    print(f"❌ 在 {base_dir} 底下找不到任何環境資料夾！")
    exit(1)

dashboard_data = {}

print(f"🔍 正在掃描多環境目錄: {envs}，並萃取資料...")

for env in envs:
    env_dir = os.path.join(base_dir, env)
    nodes_dict = {}
    edges_list = []
    user_set = set()
    topic_partitions = {}

    def add_user_node(user_id):
        if user_id not in nodes_dict:
            clean_name = user_id.replace("U_", "")
            user_set.add(clean_name)
            nodes_dict[user_id] = {
                "id": user_id, "label": clean_name, "group": "user",
                "title": f"<div style='padding:5px;'><b>👤 系統:</b> {clean_name}</div>"
            }

    def add_topic_node(topic_id):
        if topic_id not in nodes_dict:
            nodes_dict[topic_id] = {
                "id": topic_id, "label": topic_id.replace("T_", ""), "group": "topic",
                "title": f"<div style='padding:5px;'><b>🗄️ 主題:</b> {topic_id.replace('T_', '')}</div>"
            }

    for filename in os.listdir(env_dir):
        if filename.endswith(".yaml"):
            filepath = os.path.join(env_dir, filename)
            with open(filepath, 'r', encoding='utf-8') as f:
                data = yaml.safe_load(f) or {}
                manage_topics = data.get("manage_topics") or {}
                for topic, tcfg in manage_topics.items():
                    tid = f"T_{topic}"
                    if isinstance(tcfg, dict):
                        topic_partitions[tid] = tcfg.get("partitions", "未知")
                    else:
                        topic_partitions[tid] = "未知"

    for filename in os.listdir(env_dir):
        if filename.endswith(".yaml"):
            user_name = filename.replace(".yaml", "")
            uid = f"U_{user_name}"
            add_user_node(uid)
            
            filepath = os.path.join(env_dir, filename)
            with open(filepath, 'r', encoding='utf-8') as f:
                data = yaml.safe_load(f) or {}
                
                for topic in (data.get("manage_topics") or {}).keys():
                    tid = f"T_{topic}"
                    add_topic_node(tid)
                    edges_list.append({"from": uid, "to": tid, "label": "管理", "relation": "manage"})
                    
                for topic in (data.get("extra_read_topics") or []):
                    tid = f"T_{topic}"
                    add_topic_node(tid)
                    edges_list.append({"from": uid, "to": tid, "label": "讀取", "relation": "read"})
                    
                for topic in (data.get("extra_write_topics") or []):
                    tid = f"T_{topic}"
                    add_topic_node(tid)
                    edges_list.append({"from": uid, "to": tid, "label": "寫入", "relation": "write"})

    dashboard_data[env] = {
        "nodes": list(nodes_dict.values()),
        "edges": edges_list,
        "users": sorted(list(user_set)),
        "partitions": topic_partitions
    }

print("✨ 正在渲染極速版儀表板 (預設隱藏全部節點)...")

dashboard_data_json = json.dumps(dashboard_data, ensure_ascii=False)

env_options_html = ""
for env in envs:
    env_options_html += f'<option value="{env}">🌍 部署環境: {env}</option>\n'

with open(output_file, 'w', encoding='utf-8') as f:
    html_content = f"""<!DOCTYPE html>
<html lang="zh-TW">
<head>
    <meta charset="UTF-8">
    <title>Kafka 權限與規格查詢系統</title>
    <script type="text/javascript" src="https://unpkg.com/vis-network/standalone/umd/vis-network.min.js"></script>
    <link href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css" rel="stylesheet">
    <link href="https://cdn.jsdelivr.net/npm/tom-select@2.2.2/dist/css/tom-select.css" rel="stylesheet">
    <script src="https://cdn.jsdelivr.net/npm/tom-select@2.2.2/dist/js/tom-select.complete.min.js"></script>
    
    <style>
        body {{ font-family: 'Segoe UI', Tahoma, sans-serif; margin: 0; padding: 0; background-color: #0f172a; color: #f8fafc; overflow: hidden; }}
        #mynetwork {{ position: absolute; top: 0; left: 0; width: 100vw; height: 100vh; outline: none; z-index: 1; }}
        
        .panel {{ position: absolute; top: 20px; left: 20px; background: rgba(30, 41, 59, 0.9); backdrop-filter: blur(12px); padding: 20px; border-radius: 12px; border: 1px solid #334155; z-index: 10; width: 320px; box-shadow: 0 10px 25px rgba(0,0,0,0.5); }}
        .panel h2 {{ margin-top: 0; font-size: 1.2rem; color: #e2e8f0; border-bottom: 1px solid #475569; padding-bottom: 10px; margin-bottom: 15px; }}
        
        .details-panel {{ position: absolute; top: 20px; right: 20px; background: rgba(30, 41, 59, 0.95); backdrop-filter: blur(12px); padding: 20px; border-radius: 12px; border: 1px solid #334155; z-index: 10; width: 450px; max-height: 85vh; overflow-y: auto; box-shadow: 0 10px 25px rgba(0,0,0,0.5); display: none; transition: 0.3s; }}
        .details-panel h2 {{ margin-top: 0; font-size: 1.1rem; color: #38bdf8; border-bottom: 1px solid #475569; padding-bottom: 10px; margin-bottom: 15px; display: flex; justify-content: space-between; }}
        
        table {{ width: 100%; border-collapse: collapse; font-size: 0.9rem; }}
        th {{ text-align: left; padding: 10px; background-color: #1e293b; color: #94a3b8; border-bottom: 2px solid #334155; position: sticky; top: 0; }}
        td {{ padding: 10px; border-bottom: 1px solid #334155; word-break: break-all; }}
        tr:hover {{ background-color: #1e293b; }}
        .text-center {{ text-align: center; }}
        
        .badge {{ padding: 3px 8px; border-radius: 4px; font-size: 0.8rem; font-weight: bold; color: #fff; }}
        .b-manage {{ background-color: #10b981; }}
        .b-read {{ background-color: #38bdf8; }}
        .b-write {{ background-color: #f43f5e; }}

        .search-box {{ width: 100%; padding: 10px; border-radius: 8px; border: 1px solid #475569; background-color: #1e293b; color: #fff; font-size: 1rem; margin-bottom: 10px; outline: none; cursor: pointer; }}
        .env-box {{ border-color: #10b981; color: #10b981; font-weight: bold; }}
        
        .filter-item {{ display: flex; align-items: center; margin: 10px 0; cursor: pointer; font-size: 0.95rem; }}
        .filter-item input {{ margin-right: 12px; transform: scale(1.2); cursor: pointer; }}
        .color-dot {{ width: 12px; height: 12px; border-radius: 50%; margin-right: 10px; display: inline-block; }}
        .c-manage {{ background-color: #10b981; }}
        .c-read {{ background-color: #38bdf8; }}
        .c-write {{ background-color: #f43f5e; }}
        
        #loading {{ position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%); font-size: 1.5rem; color: #38bdf8; z-index: 5; font-weight: bold; }}
        #env-toast {{ position: absolute; bottom: 20px; left: 50%; transform: translateX(-50%); background: #10b981; color: white; padding: 10px 20px; border-radius: 8px; font-weight: bold; opacity: 0; transition: opacity 0.5s; z-index: 20; pointer-events: none; }}
        
        /* 歡迎 / 空白狀態浮水印 */
        #welcome-message {{ position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%); color: #64748b; font-size: 1.5rem; text-align: center; pointer-events: none; z-index: 2; transition: 0.3s; }}
        
        ::-webkit-scrollbar {{ width: 8px; }}
        ::-webkit-scrollbar-track {{ background: rgba(0,0,0,0.1); border-radius: 4px; }}
        ::-webkit-scrollbar-thumb {{ background: #475569; border-radius: 4px; }}
        ::-webkit-scrollbar-thumb:hover {{ background: #64748b; }}

        /* Tom Select CSS */
        .ts-wrapper {{ margin-bottom: 15px; }}
        .ts-control {{ background-color: #1e293b !important; border: 1px solid #475569 !important; border-radius: 8px !important; padding: 10px !important; color: #f8fafc !important; box-shadow: none !important; transition: all 0.3s ease; }}
        .ts-wrapper.focus .ts-control {{ border-color: #38bdf8 !important; box-shadow: 0 0 0 1px #38bdf8 !important; background-color: #0f172a !important; }}
        .ts-control > input {{ color: #f8fafc !important; font-size: 1rem !important; }}
        .ts-control .item {{ color: #f8fafc !important; }}
        .ts-dropdown {{ background-color: #1e293b !important; border: 1px solid #475569 !important; border-radius: 8px !important; box-shadow: 0 10px 25px rgba(0,0,0,0.8) !important; margin-top: 5px !important; }}
        .ts-dropdown .option {{ padding: 10px 12px !important; color: #f8fafc !important; transition: background-color 0.2s; }}
        .ts-dropdown .option:hover, .ts-dropdown .active {{ background-color: #334155 !important; color: #38bdf8 !important; }}
        .ts-wrapper.single .ts-control::after {{ border-color: #94a3b8 transparent transparent transparent !important; }}
        .ts-wrapper.single.dropdown-active .ts-control::after {{ border-color: transparent transparent #38bdf8 transparent !important; }}
    </style>
</head>
<body>
    <div id="loading"><i class="fa-solid fa-spinner fa-spin"></i> 系統載入中...</div>
    <div id="env-toast">已切換至環境</div>

    <div id="welcome-message">
        <i class="fa-solid fa-diagram-project" style="font-size: 3.5rem; margin-bottom: 15px; color: #38bdf8; opacity: 0.8;"></i><br>
        <b>請從左上角搜尋系統</b><br>
        <span style="font-size: 0.9rem; margin-top: 10px; display: inline-block; color: #475569;">
            預設隱藏圖表以確保網頁流暢度
        </span>
    </div>

    <div class="panel">
        <h2><i class="fa-solid fa-magnifying-glass"></i> Kafka 權限查詢</h2>
        <select id="env-select" class="search-box env-box">{env_options_html}</select>
        
        <select id="user-select" placeholder="🔍 請搜尋或選擇系統..."></select>
        
        <div style="height: 1px; background: #475569; margin: 15px 0;"></div>
        <label class="filter-item"><input type="checkbox" id="chk-manage" checked><span class="color-dot c-manage"></span> 管理 (Manage)</label>
        <label class="filter-item"><input type="checkbox" id="chk-read" checked><span class="color-dot c-read"></span> 讀取 (Read)</label>
        <label class="filter-item"><input type="checkbox" id="chk-write" checked><span class="color-dot c-write"></span> 寫入 (Write)</label>
    </div>

    <div class="details-panel" id="details-panel">
        <h2>
            <span><i class="fa-solid fa-list"></i> <span id="details-title">系統 Topic 列表</span></span>
            <span id="topic-count" style="font-size: 0.9rem; color: #94a3b8; font-weight: normal;"></span>
        </h2>
        <table>
            <thead>
                <tr>
                    <th>Topic 名稱</th>
                    <th class="text-center" style="width: 70px;">權限</th>
                    <th class="text-center" style="width: 80px;">Partitions</th>
                </tr>
            </thead>
            <tbody id="details-tbody"></tbody>
        </table>
    </div>

    <div id="mynetwork"></div>

    <script type="text/javascript">
        var fullData = {dashboard_data_json};
        var currentEnv = null;
        var allNodes = new vis.DataSet();
        var allEdges = new vis.DataSet();
        var connectedNodeIds = new Set();
        
        var tomSelectInstance = null;
        
        function updateDetailsTable(selectedUser) {{
            var panel = document.getElementById('details-panel');
            var tbody = document.getElementById('details-tbody');
            var countSpan = document.getElementById('topic-count');
            tbody.innerHTML = '';

            // 修改：如果是空值或 ALL 都隱藏右側面板
            if (!selectedUser || selectedUser === 'ALL' || !currentEnv) {{
                panel.style.display = 'none';
                return;
            }}

            panel.style.display = 'block';
            var cleanName = selectedUser.replace('U_', '');
            document.getElementById('details-title').innerText = cleanName + " 的 Topics";

            var userEdges = fullData[currentEnv].edges.filter(e => e.from === selectedUser);
            countSpan.innerText = "共 " + userEdges.length + " 筆";

            if (userEdges.length === 0) {{
                 tbody.innerHTML = '<tr><td colspan="3" class="text-center" style="color: #64748b; padding: 20px;">無相關 Topic 紀錄</td></tr>';
                 return;
            }}

            userEdges.sort((a, b) => a.to.localeCompare(b.to));

            userEdges.forEach(e => {{
                 var topicId = e.to;
                 var topicName = topicId.replace('T_', '');
                 var relation = e.relation;
                 var partitions = fullData[currentEnv].partitions[topicId] || '未知';
                 
                 var badgeHtml = '';
                 if (relation === 'manage') badgeHtml = '<span class="badge b-manage">管理</span>';
                 else if (relation === 'read') badgeHtml = '<span class="badge b-read">讀取</span>';
                 else if (relation === 'write') badgeHtml = '<span class="badge b-write">寫入</span>';
                 
                 var partHtml = partitions === '未知' ? `<span style="color:#64748b">${{partitions}}</span>` : `<b style="color:#f8fafc">${{partitions}}</b>`;

                 var tr = document.createElement('tr');
                 tr.innerHTML = `
                    <td style="color:#e2e8f0">${{topicName}}</td>
                    <td class="text-center">${{badgeHtml}}</td>
                    <td class="text-center">${{partHtml}}</td>
                 `;
                 tbody.appendChild(tr);
            }});
        }}

        function updateFilter() {{
            var selectedUser = tomSelectInstance ? tomSelectInstance.getValue() : '';
            var welcomeMsg = document.getElementById('welcome-message');
            
            // 控制中央浮水印的顯示與隱藏
            if (!selectedUser) {{
                welcomeMsg.style.display = 'block';
            }} else {{
                welcomeMsg.style.display = 'none';
            }}

            connectedNodeIds.clear();
            if (selectedUser && selectedUser !== 'ALL') connectedNodeIds.add(selectedUser);
            
            edgesView.refresh();
            nodesView.refresh();
            updateDetailsTable(selectedUser);
        }}

        function loadEnvironmentData(envName) {{
            currentEnv = envName;
            var envData = fullData[envName];
            if (!envData) return;

            if (tomSelectInstance) {{
                tomSelectInstance.destroy();
            }}

            var userSelect = document.getElementById('user-select');
            // ✨ 修改：加入一個完全空白的預設選項
            userSelect.innerHTML = '<option value="">-- 請先搜尋系統 --</option><option value="ALL">🌐 顯示所有系統 (效能較耗費)</option>';
            envData.users.forEach(function(u) {{
                var opt = document.createElement('option');
                opt.value = "U_" + u;
                opt.innerHTML = "👤 " + u;
                userSelect.appendChild(opt);
            }});

            tomSelectInstance = new TomSelect('#user-select', {{
                create: false,
                maxOptions: null,
                sortField: {{ field: "text", direction: "asc" }}
            }});
            
            tomSelectInstance.on('change', updateFilter);

            allNodes.clear(); allNodes.add(envData.nodes);
            allEdges.clear(); allEdges.add(envData.edges);

            allEdges.forEach(function(edge) {{
                if(edge.relation === 'manage') allEdges.update({{id: edge.id, color: {{ color: '#10b981', highlight: '#34d399' }}, width: 2 }});
                else if(edge.relation === 'read') allEdges.update({{id: edge.id, color: {{ color: '#38bdf8', highlight: '#7dd3fc' }}, width: 1, dashes: [5, 5] }});
                else if(edge.relation === 'write') allEdges.update({{id: edge.id, color: {{ color: '#f43f5e', highlight: '#fb7185' }}, width: 1.5, dashes: [2, 2] }});
            }});

            // ✨ 修改：預設設為空值，不顯示任何圖表
            tomSelectInstance.setValue('');
            
            var toast = document.getElementById('env-toast');
            toast.innerText = "✅ 已切換至 " + envName + " 環境";
            toast.style.opacity = 1;
            setTimeout(() => toast.style.opacity = 0, 2000);
        }}

        var edgesView = new vis.DataView(allEdges, {{
            filter: function (edge) {{
                var selectedUser = tomSelectInstance ? tomSelectInstance.getValue() : '';
                // ✨ 修改：如果沒有選擇任何東西，直接回傳 false 隱藏所有線條
                if (!selectedUser) return false;

                var showManage = document.getElementById('chk-manage').checked;
                var showRead = document.getElementById('chk-read').checked;
                var showWrite = document.getElementById('chk-write').checked;

                if (edge.relation === 'manage' && !showManage) return false;
                if (edge.relation === 'read' && !showRead) return false;
                if (edge.relation === 'write' && !showWrite) return false;
                
                if (selectedUser !== 'ALL' && edge.from !== selectedUser) return false;

                connectedNodeIds.add(edge.from);
                connectedNodeIds.add(edge.to);
                return true;
            }}
        }});

        var nodesView = new vis.DataView(allNodes, {{
            filter: function (node) {{
                var selectedUser = tomSelectInstance ? tomSelectInstance.getValue() : '';
                // ✨ 修改：如果沒有選擇任何東西，直接回傳 false 隱藏所有節點
                if (!selectedUser) return false;
                
                if (selectedUser === 'ALL') return true;
                return connectedNodeIds.has(node.id);
            }}
        }});

        var container = document.getElementById('mynetwork');
        var data = {{ nodes: nodesView, edges: edgesView }};
        var options = {{
            nodes: {{ shape: 'icon', font: {{ color: '#e2e8f0', size: 14, strokeWidth: 3, strokeColor: '#0f172a' }} }},
            groups: {{
                user: {{ icon: {{ face: '"Font Awesome 6 Free"', weight: "900", code: '\uf007', size: 55, color: '#38bdf8' }} }},
                topic: {{ icon: {{ face: '"Font Awesome 6 Free"', weight: "900", code: '\uf1c0', size: 40, color: '#94a3b8' }} }}
            }},
            edges: {{ font: {{ size: 12, color: '#94a3b8', strokeWidth: 0, align: 'top' }}, smooth: {{ type: 'dynamic' }}, arrows: {{ to: {{ enabled: true, scaleFactor: 0.5 }} }} }},
            physics: {{ forceAtlas2Based: {{ gravitationalConstant: -100, centralGravity: 0.01, springConstant: 0.08, springLength: 150 }}, solver: 'forceAtlas2Based', stabilization: {{ iterations: 150 }} }}
        }};

        document.fonts.ready.then(function () {{
            document.getElementById('loading').style.display = 'none';
            var network = new vis.Network(container, data, options);

            document.getElementById('env-select').addEventListener('change', function(e) {{
                loadEnvironmentData(e.target.value);
            }});
            document.getElementById('chk-manage').addEventListener('change', updateFilter);
            document.getElementById('chk-read').addEventListener('change', updateFilter);
            document.getElementById('chk-write').addEventListener('change', updateFilter);

            var initialEnv = document.getElementById('env-select').value;
            if (initialEnv) loadEnvironmentData(initialEnv);
        }});
    </script>
</body>
</html>
"""
    f.write(html_content)

print(f"✅ 轉換成功！已產生極速版 {output_file}。現在預設為隱藏狀態，開啟速度飛快！")