package kafkakit

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/segmentio/kafka-go"
)

// Producer wraps kafka.Writer to provide a simple interface for sending messages.
type Producer struct {
	writer *kafka.Writer
	config *Config
}

// NewProducer creates a new Kafka producer from the given config.
func NewProducer(cfg Config) (*Producer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	mechanism, err := cfg.saslMechanism()
	if err != nil {
		return nil, err
	}

	tlsCfg, err := cfg.tlsConfig()
	if err != nil {
		return nil, err
	}

	transport := &kafka.Transport{
		DialTimeout: cfg.dialTimeout(),
		TLS:         tlsCfg,
	}
	if mechanism != nil {
		transport.SASL = mechanism
	}

	requiredAcks := kafka.RequireAll
	if cfg.RequiredAcks != nil {
		requiredAcks = kafka.RequiredAcks(*cfg.RequiredAcks)
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: cfg.writeTimeout(),
		ReadTimeout:  cfg.readTimeout(),
		Transport:    transport,
		RequiredAcks: requiredAcks,
		// Enable automatic topic creation (requires broker setting).
		AllowAutoTopicCreation: true,
	}

	if cfg.BatchSize > 0 {
		writer.BatchSize = cfg.BatchSize
	}
	if cfg.BatchTimeout > 0 {
		writer.BatchTimeout = cfg.BatchTimeout
	}
	if cfg.OnProduce != nil {
		writer.Completion = func(messages []kafka.Message, err error) {
			for _, m := range messages {
				cfg.OnProduce(fromKafkaMessage(m), err)
			}
		}
	}

	return &Producer{
		writer: writer,
		config: &cfg,
	}, nil
}

// Send sends a single message with the given key and value.
func (p *Producer) Send(ctx context.Context, key, value []byte) error {
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   key,
		Value: value,
	})
}

// SendString sends a single message with string key and value.
func (p *Producer) SendString(ctx context.Context, key, value string) error {
	return p.Send(ctx, []byte(key), []byte(value))
}

// SendValue sends a single message with only a value (no key).
func (p *Producer) SendValue(ctx context.Context, value []byte) error {
	return p.Send(ctx, nil, value)
}

// SendBatch sends multiple messages at once.
func (p *Producer) SendBatch(ctx context.Context, messages []Message) error {
	kafkaMessages := make([]kafka.Message, len(messages))
	for i, msg := range messages {
		kafkaMessages[i] = kafka.Message{
			Key:     msg.Key,
			Value:   msg.Value,
			Headers: toKafkaHeaders(msg.Headers),
		}
	}
	return p.writer.WriteMessages(ctx, kafkaMessages...)
}

// Close closes the producer and flushes any buffered messages.
func (p *Producer) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}

// Stats returns the current producer statistics.
func (p *Producer) Stats() kafka.WriterStats {
	return p.writer.Stats()
}

// EnsureTopicExists creates the topic if it does not exist.
// numPartitions and replicationFactor control the topic configuration.
func EnsureTopicExists(ctx context.Context, cfg Config, numPartitions, replicationFactor int) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	tlsCfg, err := cfg.tlsConfig()
	if err != nil {
		return err
	}

	dialer := &kafka.Dialer{
		Timeout:   cfg.dialTimeout(),
		DualStack: true,
		TLS:       tlsCfg,
	}

	mechanism, err := cfg.saslMechanism()
	if err != nil {
		return err
	}
	if mechanism != nil {
		dialer.SASLMechanism = mechanism
	}

	// Connect to any available broker.
	var conn *kafka.Conn
	for _, broker := range cfg.Brokers {
		conn, err = dialer.DialContext(ctx, "tcp", broker)
		if err == nil {
			break
		}
	}
	if conn == nil {
		return fmt.Errorf("kafkakit: failed to connect to any broker: %w", err)
	}
	defer conn.Close()

	// Find the controller to create the topic.
	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("kafkakit: failed to find controller: %w", err)
	}

	controllerConn, err := dialer.DialContext(
		ctx,
		"tcp",
		net.JoinHostPort(controller.Host, fmt.Sprintf("%d", controller.Port)),
	)
	if err != nil {
		return fmt.Errorf("kafkakit: failed to connect to controller: %w", err)
	}
	defer controllerConn.Close()

	err = controllerConn.CreateTopics(kafka.TopicConfig{
		Topic:             cfg.Topic,
		NumPartitions:     numPartitions,
		ReplicationFactor: replicationFactor,
		ConfigEntries: []kafka.ConfigEntry{
			{
				ConfigName:  "retention.ms",
				ConfigValue: fmt.Sprintf("%d", 7*24*time.Hour/time.Millisecond), // 7 days default
			},
		},
	})
	if err != nil {
		return fmt.Errorf("kafkakit: failed to create topic %q: %w", cfg.Topic, err)
	}

	return nil
}
