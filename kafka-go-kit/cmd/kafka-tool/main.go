package main

import (
	"bufio"
	"context"
	crypto_rand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	kafkakit "github.com/jay/kafka-go-kit"
)

const version = "1.0.0"

const usage = `kafka-tool v` + version + ` — 純 Go Kafka CLI 工具

使用方式:
  kafka-tool produce [選項]        發送訊息到 Kafka
  kafka-tool consume [選項]        從 Kafka 接收訊息

Produce 範例:
  # 直接發送一筆訊息
  kafka-tool produce -b localhost:9092 -t my-topic -m "hello world"

  # 帶 SASL 認證
  kafka-tool produce -b broker:9093 -t my-topic -u user -p pass -m "data"

  # 從 stdin 讀取（每行一筆訊息，Ctrl+D 結束）
  echo "hello" | kafka-tool produce -b localhost:9092 -t my-topic

  # 帶 key 發送
  kafka-tool produce -b localhost:9092 -t my-topic -k my-key -m "my-value"

Consume 範例:
  # 持續接收訊息（Ctrl+C 停止）
  kafka-tool consume -b localhost:9092 -t my-topic -g my-group

  # 帶 SASL 認證
  kafka-tool consume -b broker:9093 -t my-topic -g my-group -u user -p pass

共用選項:
  -b, --broker        string   Kafka broker 地址（多個用逗號分隔，預設 localhost:9092）
  -t, --topic         string   Kafka topic（必填）
  -u, --user          string   SASL 使用者名稱（選填）
  -p, --pass          string   SASL 密碼（選填）
  --mechanism         string   SASL 機制: PLAIN, SCRAM-SHA-256, SCRAM-SHA-512（預設 PLAIN）
  --tls                        啟用 TLS
  --ca-cert           string   CA 憑證檔案路徑（PEM 格式，用於自簽憑證）
  --client-cert       string   Client 憑證檔案路徑（PEM 格式，用於 mTLS）
  --client-key        string   Client 私鑰檔案路徑（PEM 格式，用於 mTLS）
  --tls-skip-verify            跳過 TLS 憑證驗證（僅限測試用）

Produce 專用選項:
  -m, --message  string   要發送的訊息內容（不指定則從 stdin 讀取）
  -k, --key      string   訊息 key（選填）

Consume 專用選項:
  -g, --group    string   Consumer Group ID（必填，使用 --temp 時可省略）
  --temp                  自動產生暫時 group（格式: {user}_tmp-{random}）
  --json                  輸出 JSON 格式
  --no-header             不顯示 header 資訊

  -v, --version           顯示版本
  -h, --help              顯示幫助
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "produce", "prod", "p":
		runProduce(os.Args[2:])
	case "consume", "cons", "c":
		runConsume(os.Args[2:])
	case "-v", "--version", "version":
		fmt.Printf("kafka-tool v%s\n", version)
	case "-h", "--help", "help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "❌ 未知指令: %s\n\n", os.Args[1])
		fmt.Print(usage)
		os.Exit(1)
	}
}

// ─── Argument Parser ────────────────────────────────────────────────

type commonArgs struct {
	broker        string
	topic         string
	user          string
	pass          string
	mechanism     string
	tls           bool
	caCert        string
	clientCert    string
	clientKey     string
	tlsSkipVerify bool
}

type produceArgs struct {
	commonArgs
	message string
	key     string
}

type consumeArgs struct {
	commonArgs
	group    string
	json     bool
	noHeader bool
	temp     bool
}

func parseCommonArgs(args []string, i int, c *commonArgs) int {
	if i >= len(args) {
		return i
	}
	switch args[i] {
	case "-b", "--broker":
		i++
		if i < len(args) {
			c.broker = args[i]
		}
	case "-t", "--topic":
		i++
		if i < len(args) {
			c.topic = args[i]
		}
	case "-u", "--user":
		i++
		if i < len(args) {
			c.user = args[i]
		}
	case "-p", "--pass":
		i++
		if i < len(args) {
			c.pass = args[i]
		}
	case "--mechanism":
		i++
		if i < len(args) {
			c.mechanism = args[i]
		}
	case "--tls":
		c.tls = true
	case "--ca-cert":
		i++
		if i < len(args) {
			c.caCert = args[i]
			c.tls = true // 指定憑證自動啟用 TLS
		}
	case "--client-cert":
		i++
		if i < len(args) {
			c.clientCert = args[i]
			c.tls = true
		}
	case "--client-key":
		i++
		if i < len(args) {
			c.clientKey = args[i]
			c.tls = true
		}
	case "--tls-skip-verify":
		c.tlsSkipVerify = true
		c.tls = true
	default:
		return -1 // not a common arg
	}
	return i
}

func parseProduceArgs(args []string) produceArgs {
	pa := produceArgs{
		commonArgs: commonArgs{
			broker:    "localhost:9092",
			mechanism: "PLAIN",
		},
	}
	for i := 0; i < len(args); i++ {
		if j := parseCommonArgs(args, i, &pa.commonArgs); j >= 0 {
			i = j
			continue
		}
		switch args[i] {
		case "-m", "--message":
			i++
			if i < len(args) {
				pa.message = args[i]
			}
		case "-k", "--key":
			i++
			if i < len(args) {
				pa.key = args[i]
			}
		case "-h", "--help":
			fmt.Print(usage)
			os.Exit(0)
		default:
			fmt.Fprintf(os.Stderr, "❌ 未知選項: %s\n", args[i])
			os.Exit(1)
		}
	}
	return pa
}

func parseConsumeArgs(args []string) consumeArgs {
	ca := consumeArgs{
		commonArgs: commonArgs{
			broker:    "localhost:9092",
			mechanism: "PLAIN",
		},
	}
	for i := 0; i < len(args); i++ {
		if j := parseCommonArgs(args, i, &ca.commonArgs); j >= 0 {
			i = j
			continue
		}
		switch args[i] {
		case "-g", "--group":
			i++
			if i < len(args) {
				ca.group = args[i]
			}
		case "--temp":
			ca.temp = true
		case "--json":
			ca.json = true
		case "--no-header":
			ca.noHeader = true
		case "-h", "--help":
			fmt.Print(usage)
			os.Exit(0)
		default:
			fmt.Fprintf(os.Stderr, "❌ 未知選項: %s\n", args[i])
			os.Exit(1)
		}
	}
	return ca
}

func buildConfig(c commonArgs) kafkakit.Config {
	brokers := strings.Split(c.broker, ",")
	for i := range brokers {
		brokers[i] = strings.TrimSpace(brokers[i])
	}

	var mechanism kafkakit.SASLMechanism
	switch strings.ToUpper(c.mechanism) {
	case "SCRAM-SHA-256":
		mechanism = kafkakit.SASLSCRAMSHA256
	case "SCRAM-SHA-512":
		mechanism = kafkakit.SASLSCRAMSHA512
	default:
		mechanism = kafkakit.SASLPlain
	}

	return kafkakit.Config{
		Brokers:        brokers,
		Topic:          c.topic,
		Username:       c.user,
		Password:       c.pass,
		Mechanism:      mechanism,
		UseTLS:         c.tls,
		CACertFile:     c.caCert,
		ClientCertFile: c.clientCert,
		ClientKeyFile:  c.clientKey,
		TLSSkipVerify:  c.tlsSkipVerify,
	}
}

// ─── Produce ────────────────────────────────────────────────────────

func runProduce(args []string) {
	pa := parseProduceArgs(args)

	if pa.topic == "" {
		fmt.Fprintln(os.Stderr, "❌ 必須指定 topic (-t)")
		os.Exit(1)
	}

	// Read from stdin to check if interactive
	stat, _ := os.Stdin.Stat()
	interactive := (stat.Mode() & os.ModeCharDevice) != 0

	cfg := buildConfig(pa.commonArgs)

	// Optimize producer settings for instant delivery if interactive or sending single message.
	// Also register OnProduce callback to verify partition and offset of the sent messages.
	if pa.message != "" || interactive {
		cfg.BatchSize = 1
		cfg.BatchTimeout = 10 * time.Millisecond
		cfg.OnProduce = func(msg kafkakit.Message, err error) {
			if err == nil {
				fmt.Fprintf(os.Stderr, "ℹ️  發送成功: partition=%d offset=%d\n", msg.Partition, msg.Offset)
			}
		}
	}

	producer, err := kafkakit.NewProducer(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 建立 Producer 失敗: %v\n", err)
		os.Exit(1)
	}
	defer producer.Close()

	ctx := context.Background()

	// If -m is provided, send that single message and exit
	if pa.message != "" {
		err = producer.Send(ctx, toBytes(pa.key), []byte(pa.message))
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ 發送失敗: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "✅ 訊息已發送到 topic [%s]\n", pa.topic)
		return
	}

	if interactive {
		fmt.Fprintf(os.Stderr, "📝 輸入訊息（每行即時發送，Ctrl+D 結束）：\n")
	}

	scanner := bufio.NewScanner(os.Stdin)
	// Increase buffer size to 1MB for large messages
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	count := 0

	if interactive {
		// Interactive mode: send each line immediately for instant feedback
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}
			err = producer.Send(ctx, toBytes(pa.key), []byte(line))
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ 發送失敗: %v\n", err)
				os.Exit(1)
			}
			count++
			fmt.Fprintf(os.Stderr, "✅ 已發送 (%d)\n", count)
		}
	} else {
		// Pipe mode: batch send for throughput
		const batchSize = 500
		batch := make([]kafkakit.Message, 0, batchSize)

		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}
			batch = append(batch, kafkakit.Message{
				Key:   toBytes(pa.key),
				Value: []byte(line),
			})
			if len(batch) >= batchSize {
				if err = producer.SendBatch(ctx, batch); err != nil {
					fmt.Fprintf(os.Stderr, "❌ 批次發送失敗: %v\n", err)
					os.Exit(1)
				}
				count += len(batch)
				batch = batch[:0]
			}
		}

		// Flush remaining messages
		if len(batch) > 0 {
			if err = producer.SendBatch(ctx, batch); err != nil {
				fmt.Fprintf(os.Stderr, "❌ 批次發送失敗: %v\n", err)
				os.Exit(1)
			}
			count += len(batch)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ 讀取 stdin 失敗: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "✅ 共發送 %d 筆訊息到 topic [%s]\n", count, pa.topic)
}

// ─── Consume ────────────────────────────────────────────────────────

func runConsume(args []string) {
	ca := parseConsumeArgs(args)

	if ca.topic == "" {
		fmt.Fprintln(os.Stderr, "❌ 必須指定 topic (-t)")
		os.Exit(1)
	}

	// --temp: 自動產生暫時 group name
	if ca.temp {
		if ca.user == "" {
			fmt.Fprintln(os.Stderr, "❌ 使用 --temp 必須指定使用者 (-u)")
			os.Exit(1)
		}
		ca.group = ca.user + "_tmp-" + randomHex(4)
	}

	if ca.group == "" {
		fmt.Fprintln(os.Stderr, "❌ 必須指定 consumer group (-g) 或使用 --temp")
		os.Exit(1)
	}

	cfg := buildConfig(ca.commonArgs)
	cfg.GroupID = ca.group

	consumer, err := kafkakit.NewConsumer(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 建立 Consumer 失敗: %v\n", err)
		os.Exit(1)
	}
	defer consumer.Close()

	// Graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "\n🛑 正在關閉 Consumer...")
		cancel()
	}()

	if !ca.noHeader {
		if ca.temp {
			fmt.Fprintf(os.Stderr, "📡 正在監聽 topic [%s] (temp group: %s)... Ctrl+C 停止\n", ca.topic, ca.group)
			fmt.Fprintf(os.Stderr, "ℹ️  暫時 group，Kafka 會在閒置後自動清除\n")
		} else {
			fmt.Fprintf(os.Stderr, "📡 正在監聽 topic [%s] (group: %s)... Ctrl+C 停止\n", ca.topic, ca.group)
		}
	}

	// 使用 UTC+8 時區顯示時間（訊息時間為 Kafka producer 發送時的時間戳）
	loc := time.FixedZone("UTC+8", 8*60*60)
	jsonEnc := json.NewEncoder(os.Stdout)

	err = consumer.Consume(ctx, func(msg kafkakit.Message) error {
		msgTime := msg.Time.In(loc)
		if ca.json {
			out := map[string]any{
				"topic":     msg.Topic,
				"partition": msg.Partition,
				"offset":    msg.Offset,
				"key":       msg.KeyString(),
				"timestamp": msgTime.Format("2006-01-02T15:04:05.000+08:00"),
			}
			// If value is valid JSON, embed as raw JSON; otherwise as string
			trimmed := strings.TrimSpace(msg.ValueString())
			if json.Valid([]byte(trimmed)) {
				out["value"] = json.RawMessage(trimmed)
			} else {
				out["value"] = msg.ValueString()
			}
			jsonEnc.Encode(out)
		} else if ca.noHeader {
			// Raw value only — useful for piping
			fmt.Println(msg.ValueString())
		} else {
			fmt.Printf("[%s] partition=%d offset=%d key=%s | %s\n",
				msgTime.Format("15:04:05"),
				msg.Partition, msg.Offset,
				msg.KeyString(), msg.ValueString(),
			)
		}
		return nil
	})

	if err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "❌ Consumer 錯誤: %v\n", err)
		os.Exit(1)
	}
}

// ─── Helpers ────────────────────────────────────────────────────────

func toBytes(s string) []byte {
	if s == "" {
		return nil
	}
	return []byte(s)
}

func randomHex(n int) string {
	b := make([]byte, n)
	crypto_rand.Read(b)
	return hex.EncodeToString(b)
}
