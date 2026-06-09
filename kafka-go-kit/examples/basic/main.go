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
	// Configuration — adjust these to your environment
	// =============================================
	cfg := kafkakit.Config{
		Brokers: []string{"localhost:9092"},
		Topic:   "my-topic",

		// SASL authentication (leave Username empty to skip auth)
		Username:  "",
		Password:  "",
		Mechanism: kafkakit.SASLPlain, // or SASLSCRAMSHA256, SASLSCRAMSHA512

		// TLS (set to true if your broker requires TLS)
		UseTLS: false,
	}

	// =============================================
	// Example: Producer
	// =============================================
	producer, err := kafkakit.NewProducer(cfg)
	if err != nil {
		log.Fatalf("Failed to create producer: %v", err)
	}
	defer producer.Close()

	ctx := context.Background()

	// Send a simple string message
	err = producer.SendString(ctx, "user-123", `{"event":"login","user":"jay"}`)
	if err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}
	fmt.Println("✅ Message sent successfully!")

	// Send a batch of messages
	err = producer.SendBatch(ctx, []kafkakit.Message{
		{Key: []byte("key-1"), Value: []byte("value-1")},
		{Key: []byte("key-2"), Value: []byte("value-2")},
		{Key: []byte("key-3"), Value: []byte("value-3")},
	})
	if err != nil {
		log.Fatalf("Failed to send batch: %v", err)
	}
	fmt.Println("✅ Batch sent successfully!")

	// =============================================
	// Example: Consumer
	// =============================================
	consumerCfg := cfg
	consumerCfg.GroupID = "my-consumer-group"

	consumer, err := kafkakit.NewConsumer(consumerCfg)
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	// Graceful shutdown on Ctrl+C
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\n🛑 Shutting down consumer...")
		cancel()
	}()

	fmt.Println("📡 Consuming messages... (Ctrl+C to stop)")
	err = consumer.Consume(ctx, func(msg kafkakit.Message) error {
		fmt.Printf("📨 Received: topic=%s partition=%d offset=%d key=%s value=%s\n",
			msg.Topic, msg.Partition, msg.Offset, msg.KeyString(), msg.ValueString())
		return nil
	})
	if err != nil && err != context.Canceled {
		log.Fatalf("Consumer error: %v", err)
	}
}
