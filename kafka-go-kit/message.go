package kafkakit

import (
	"time"

	"github.com/segmentio/kafka-go"
)

// Header represents a Kafka message header (key-value pair).
type Header struct {
	Key   string
	Value []byte
}

// Message represents a Kafka message with metadata.
type Message struct {
	// Topic is the Kafka topic the message belongs to.
	Topic string

	// Partition is the partition the message was read from / written to.
	Partition int

	// Offset is the message offset in the partition.
	Offset int64

	// Key is the message key (can be nil).
	Key []byte

	// Value is the message payload.
	Value []byte

	// Headers are optional key-value metadata attached to the message.
	Headers []Header

	// Time is the timestamp of the message.
	Time time.Time
}

// ValueString returns the message value as a string.
func (m Message) ValueString() string {
	return string(m.Value)
}

// KeyString returns the message key as a string.
func (m Message) KeyString() string {
	return string(m.Key)
}

// --- internal converters ---

func fromKafkaMessage(m kafka.Message) Message {
	headers := make([]Header, len(m.Headers))
	for i, h := range m.Headers {
		headers[i] = Header{Key: h.Key, Value: h.Value}
	}
	return Message{
		Topic:     m.Topic,
		Partition: m.Partition,
		Offset:    m.Offset,
		Key:       m.Key,
		Value:     m.Value,
		Headers:   headers,
		Time:      m.Time,
	}
}

func toKafkaHeaders(headers []Header) []kafka.Header {
	if len(headers) == 0 {
		return nil
	}
	kHeaders := make([]kafka.Header, len(headers))
	for i, h := range headers {
		kHeaders[i] = kafka.Header{Key: h.Key, Value: h.Value}
	}
	return kHeaders
}
