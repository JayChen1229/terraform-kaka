# kafka-tool

純 Go 語言 Kafka CLI 工具，零 CGO 依賴，**不用寫任何程式碼**，直接用命令列收送 Kafka 訊息。

## 特性

- ✅ **純 Go 實現** — 不依賴 librdkafka，無 CGO，跨平台編譯零障礙
- ✅ **零程式碼** — 直接用執行檔收送訊息，不需要安裝任何 runtime
- ✅ **SASL 認證** — 支援 PLAIN、SCRAM-SHA-256、SCRAM-SHA-512
- ✅ **TLS / mTLS** — 支援 CA 憑證、Client 憑證（mTLS）、跳過驗證
- ✅ **stdin 支援** — 可用管道 (pipe) 批次發送
- ✅ **JSON 輸出** — `--json` 方便後續處理

---

## 部署到其他機器（不需要裝 Go）

kafka-tool 編譯出來是**單一靜態連結執行檔**，目標機器上**不需要安裝 Go、不需要任何 library**，只要把檔案傳過去就能用。

### 預編譯版本

| 檔案 | 平台 |
|------|------|
| `kafka-tool` | macOS Apple Silicon (arm64) |
| `kafka-tool-linux-amd64` | Linux x86_64 |
| `kafka-tool-windows-amd64.exe` | Windows x86_64 |

### 部署到 Linux VM

```bash
# 1. 從你的 Mac 把執行檔傳到 VM
scp kafka-tool-linux-amd64 user@VM-IP:/usr/local/bin/kafka-tool

# 2. 在 VM 上加執行權限
ssh user@VM-IP "chmod +x /usr/local/bin/kafka-tool"

# 3. 直接使用
ssh user@VM-IP "kafka-tool consume -b kafka-broker:9092 -t my-topic -g my-group"
```

### 部署到 Windows

```powershell
# 把 kafka-tool-windows-amd64.exe 複製到 Windows 機器上，直接執行：
kafka-tool-windows-amd64.exe consume -b kafka-broker:9092 -t my-topic -g my-group
```

### 自行編譯

如果你有 Go 環境，也可以從原始碼編譯：

```bash
# 編譯當前平台
go build -o kafka-tool ./cmd/kafka-tool/

# 跨平台編譯
GOOS=linux   GOARCH=amd64 go build -o kafka-tool-linux-amd64       ./cmd/kafka-tool/
GOOS=linux   GOARCH=arm64 go build -o kafka-tool-linux-arm64       ./cmd/kafka-tool/
GOOS=windows GOARCH=amd64 go build -o kafka-tool-windows-amd64.exe ./cmd/kafka-tool/
GOOS=darwin  GOARCH=amd64 go build -o kafka-tool-darwin-amd64      ./cmd/kafka-tool/
GOOS=darwin  GOARCH=arm64 go build -o kafka-tool-darwin-arm64      ./cmd/kafka-tool/
```

---

## 使用方式

### 發送訊息 (Produce)

```bash
# 發送一筆訊息
kafka-tool produce -b localhost:9092 -t my-topic -m "hello world"

# 帶 key 發送
kafka-tool produce -b localhost:9092 -t my-topic -k user-123 -m '{"event":"login"}'

# 帶 SASL 認證 + TLS
kafka-tool produce -b broker:9093 -t my-topic -u myuser -p mypass --tls -m "secure data"

# 從 stdin 讀取（每行一筆訊息）
echo "hello" | kafka-tool produce -b localhost:9092 -t my-topic

# 從檔案批次發送
cat messages.txt | kafka-tool produce -b localhost:9092 -t my-topic

# 互動模式（每行一筆，Ctrl+D 結束）
kafka-tool produce -b localhost:9092 -t my-topic
```

### 接收訊息 (Consume)

```bash
# 持續接收訊息（Ctrl+C 停止）
kafka-tool consume -b localhost:9092 -t my-topic -g my-group

# 帶 SASL 認證 + TLS
kafka-tool consume -b broker:9093 -t my-topic -g my-group -u myuser -p mypass --tls

# JSON 格式輸出（方便 jq 處理）
kafka-tool consume -b localhost:9092 -t my-topic -g my-group --json

# 只輸出 value（方便管道處理）
kafka-tool consume -b localhost:9092 -t my-topic -g my-group --no-header

# 暫時 group（自動產生 {user}_tmp-{random}，不需要 -g，閒置後自動清除）
kafka-tool consume -b broker:9093 -t my-topic --temp -u myuser -p mypass --tls

# 搭配 jq 過濾
kafka-tool consume -b localhost:9092 -t my-topic -g my-group --json | jq '.value'
```

### 多個 Broker

```bash
kafka-tool produce -b "broker1:9092,broker2:9092,broker3:9092" -t my-topic -m "hello"
```

### 快捷指令

子命令支援縮寫：

| 完整 | 縮寫 |
|------|------|
| `produce` | `prod` 或 `p` |
| `consume` | `cons` 或 `c` |

```bash
kafka-tool p -b localhost:9092 -t my-topic -m "quick send"
kafka-tool c -b localhost:9092 -t my-topic -g my-group
```

---

## TLS 憑證設定

### 場景 1：公有 CA（如 AWS MSK、Confluent Cloud）

Broker 使用公有 CA 簽發的憑證，只需加 `--tls`：

```bash
kafka-tool consume -b broker:9093 -t my-topic -g grp -u user -p pass --tls
```

### 場景 2：自簽憑證 / 內部 CA

Broker 使用自簽或公司內部 CA 簽發的憑證，需指定 CA 檔案：

```bash
kafka-tool consume -b broker:9093 -t my-topic -g grp \
  -u user -p pass \
  --ca-cert /path/to/ca.pem
```

### 場景 3：mTLS 雙向認證

Broker 要求 client 也提供憑證（mutual TLS）：

```bash
kafka-tool consume -b broker:9093 -t my-topic -g grp \
  --ca-cert /path/to/ca.pem \
  --client-cert /path/to/client.crt \
  --client-key /path/to/client.key
```

### 場景 4：測試環境跳過憑證驗證

> ⚠️ 僅限測試，正式環境請勿使用

```bash
kafka-tool consume -b broker:9093 -t my-topic -g grp --tls-skip-verify
```

> 💡 指定 `--ca-cert`、`--client-cert`、`--client-key`、`--tls-skip-verify` 任一個都會**自動啟用 TLS**，不需要額外加 `--tls`。

---

## 參數說明

### 共用參數

| 參數 | 縮寫 | 說明 | 預設值 |
|------|------|------|--------|
| `--broker` | `-b` | Kafka broker 地址（多個用逗號分隔）| `localhost:9092` |
| `--topic` | `-t` | Kafka topic（必填）| — |
| `--user` | `-u` | SASL 使用者名稱 | — |
| `--pass` | `-p` | SASL 密碼 | — |
| `--mechanism` | — | SASL 機制: `PLAIN`, `SCRAM-SHA-256`, `SCRAM-SHA-512` | `PLAIN` |
| `--tls` | — | 啟用 TLS 加密 | `false` |
| `--ca-cert` | — | CA 憑證檔案路徑（PEM 格式）| — |
| `--client-cert` | — | Client 憑證檔案路徑（PEM，mTLS 用）| — |
| `--client-key` | — | Client 私鑰檔案路徑（PEM，mTLS 用）| — |
| `--tls-skip-verify` | — | 跳過 TLS 憑證驗證（僅限測試）| `false` |

### Produce 專用

| 參數 | 縮寫 | 說明 |
|------|------|------|
| `--message` | `-m` | 要發送的訊息（不指定則從 stdin 讀取）|
| `--key` | `-k` | 訊息 key |

### Consume 專用

| 參數 | 縮寫 | 說明 |
|------|------|------|
| `--group` | `-g` | Consumer Group ID（必填，使用 `--temp` 時可省略）|
| `--temp` | — | 自動產生暫時 group（格式: `{user}_tmp-{random}`，閒置後自動清除）|
| `--json` | — | 輸出 JSON 格式 |
| `--no-header` | — | 只輸出 value，不顯示 metadata |

---

## 輸出格式

### 預設格式
```
[15:04:05] partition=0 offset=42 key=my-key | message content here
```

### JSON 格式 (`--json`)
```json
{"topic":"my-topic","partition":0,"offset":42,"key":"my-key","value":"message content","timestamp":"2025-01-01T15:04:05.000+08:00"}
```

### Raw 格式 (`--no-header`)
```
message content here
```

---

## 完整範例：從零開始在新 Linux VM 上收資料

```bash
# === 在你的 Mac 上 ===

# 1. 編譯 Linux 版本（如果還沒編譯）
GOOS=linux GOARCH=amd64 go build -o kafka-tool-linux-amd64 ./cmd/kafka-tool/

# 2. 把執行檔和憑證一起傳到 VM
scp kafka-tool-linux-amd64 ca.pem user@10.0.1.100:/opt/kafka-tool/

# === 在 Linux VM 上 ===

# 3. 加執行權限
chmod +x /opt/kafka-tool/kafka-tool-linux-amd64

# 4. 起 consumer 開始收資料
/opt/kafka-tool/kafka-tool-linux-amd64 consume \
  -b kafka-broker:9093 \
  -t my-topic \
  -g my-consumer-group \
  -u kafka-user \
  -p kafka-password \
  --ca-cert /opt/kafka-tool/ca.pem

# 輸出：
# 📡 正在監聽 topic [my-topic] (group: my-consumer-group)... Ctrl+C 停止
# [15:04:05] partition=0 offset=0 key=user-123 | {"event":"login"}
# [15:04:06] partition=1 offset=5 key=user-456 | {"event":"logout"}
```

## License

MIT
