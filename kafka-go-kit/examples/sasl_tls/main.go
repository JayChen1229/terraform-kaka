package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	kafkakit "github.com/jay/kafka-go-kit"
)

func main() {
	// =============================================
	// SASL/SCRAM + TLS configuration
	// =============================================
	cfg := kafkakit.Config{
		Brokers: []string{
			"broker1.example.com:9093",
			"broker2.example.com:9093",
			"broker3.example.com:9093",
		},
		Topic: "secure-topic",

		// SASL authentication
		Username:  "kafka-user",
		Password:  "kafka-password",
		Mechanism: kafkakit.SASLSCRAMSHA512,

		// Enable TLS
		UseTLS: true,

		// Consumer group
		GroupID: "secure-consumer-group",
	}

	// =============================================
	// Producer: send with headers
	// =============================================
	producer, err := kafkakit.NewProducer(cfg)
	if err != nil {
		log.Fatalf("Failed to create producer: %v", err)
	}
	defer producer.Close()

	ctx := context.Background()

	// Send batch with custom headers
	err = producer.SendBatch(ctx, []kafkakit.Message{
		{
			Key:   []byte("order-001"),
			Value: []byte(`{"order_id":"001","amount":99.99}`),
			Headers: []kafkakit.Header{
				{Key: "source", Value: []byte("payment-service")},
				{Key: "version", Value: []byte("v2")},
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to send: %v", err)
	}
	fmt.Println("✅ Secure message sent!")

	// =============================================
	// Consumer: manual commit
	// =============================================
	consumer, err := kafkakit.NewConsumer(cfg)
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\n🛑 Shutting down...")
		cancel()
	}()

	fmt.Println("📡 Consuming with manual commit...")
	err = consumer.ConsumeManualCommit(ctx, func(msg kafkakit.Message, commit func() error) error {
		fmt.Printf("📨 Received: key=%s value=%s\n", msg.KeyString(), msg.ValueString())

		// Print headers
		for _, h := range msg.Headers {
			fmt.Printf("   📎 Header: %s = %s\n", h.Key, string(h.Value))
		}

		// Process the message... then commit
		if err := commit(); err != nil {
			return fmt.Errorf("failed to commit: %w", err)
		}
		fmt.Println("   ✅ Committed!")
		return nil
	})
	if err != nil && err != context.Canceled {
		log.Fatalf("Consumer error: %v", err)
	}
}
