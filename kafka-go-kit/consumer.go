package kafkakit

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// MessageHandler is a callback function that processes a consumed message.
// Return nil to commit the offset, or an error to stop consuming.
type MessageHandler func(msg Message) error

// Consumer wraps kafka.Reader to provide a simple interface for receiving messages.
type Consumer struct {
	reader *kafka.Reader
	config *Config
}

// NewConsumer creates a new Kafka consumer from the given config.
// GroupID must be set in the config for consumer group functionality.
func NewConsumer(cfg Config) (*Consumer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if cfg.GroupID == "" {
		return nil, fmt.Errorf("kafkakit: GroupID is required for consumer")
	}

	mechanism, err := cfg.saslMechanism()
	if err != nil {
		return nil, err
	}

	tlsCfg, err := cfg.tlsConfig()
	if err != nil {
		return nil, err
	}

	dialer := &kafka.Dialer{
		Timeout:   cfg.dialTimeout(),
		DualStack: true,
		TLS:       tlsCfg,
	}
	if mechanism != nil {
		dialer.SASLMechanism = mechanism
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.Brokers,
		Topic:    cfg.Topic,
		GroupID:  cfg.GroupID,
		Dialer:   dialer,
		MinBytes: 1,       // 1 byte
		MaxBytes: 10 << 20, // 10 MB
	})

	return &Consumer{
		reader: reader,
		config: &cfg,
	}, nil
}

// ReadMessage reads and returns the next message from Kafka.
// The message offset is automatically committed when using consumer groups.
// This call blocks until a message is available or the context is cancelled.
func (c *Consumer) ReadMessage(ctx context.Context) (Message, error) {
	m, err := c.reader.ReadMessage(ctx)
	if err != nil {
		return Message{}, fmt.Errorf("kafkakit: failed to read message: %w", err)
	}
	return fromKafkaMessage(m), nil
}

// FetchMessage fetches the next message WITHOUT committing the offset.
// Use CommitMessage() after processing to manually commit.
func (c *Consumer) FetchMessage(ctx context.Context) (Message, error) {
	m, err := c.reader.FetchMessage(ctx)
	if err != nil {
		return Message{}, fmt.Errorf("kafkakit: failed to fetch message: %w", err)
	}
	return fromKafkaMessage(m), nil
}

// CommitMessage commits the offset for a previously fetched message.
func (c *Consumer) CommitMessage(ctx context.Context, msg Message) error {
	return c.reader.CommitMessages(ctx, kafka.Message{
		Topic:     msg.Topic,
		Partition: msg.Partition,
		Offset:    msg.Offset,
	})
}

// Consume starts consuming messages and calls the handler for each message.
// This blocks until the context is cancelled or the handler returns an error.
// Messages are automatically committed after the handler returns nil.
func (c *Consumer) Consume(ctx context.Context, handler MessageHandler) error {
	backoff := time.Second
	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("kafkakit: error reading message (retry in %v): %v", backoff, err)
			select {
			case <-time.After(backoff):
				if backoff < 30*time.Second {
					backoff *= 2
				}
			case <-ctx.Done():
				return ctx.Err()
			}
			continue
		}

		backoff = time.Second // reset on success

		if err := handler(fromKafkaMessage(m)); err != nil {
			return fmt.Errorf("kafkakit: handler error: %w", err)
		}
	}
}

// ConsumeManualCommit starts consuming with manual offset commit control.
// The handler receives the message; call CommitMessage after successful processing.
func (c *Consumer) ConsumeManualCommit(ctx context.Context, handler func(msg Message, commit func() error) error) error {
	backoff := time.Second
	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("kafkakit: error fetching message (retry in %v): %v", backoff, err)
			select {
			case <-time.After(backoff):
				if backoff < 30*time.Second {
					backoff *= 2
				}
			case <-ctx.Done():
				return ctx.Err()
			}
			continue
		}

		backoff = time.Second // reset on success

		msg := fromKafkaMessage(m)
		commitFn := func() error {
			return c.reader.CommitMessages(ctx, m)
		}

		if err := handler(msg, commitFn); err != nil {
			return fmt.Errorf("kafkakit: handler error: %w", err)
		}
	}
}

// Close closes the consumer.
func (c *Consumer) Close() error {
	if c.reader != nil {
		return c.reader.Close()
	}
	return nil
}

// Stats returns the current consumer statistics.
func (c *Consumer) Stats() kafka.ReaderStats {
	return c.reader.Stats()
}
