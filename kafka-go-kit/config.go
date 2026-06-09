package kafkakit

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
)

// SASLMechanism defines the SASL authentication mechanism.
type SASLMechanism string

const (
	SASLPlain       SASLMechanism = "PLAIN"
	SASLSCRAMSHA256 SASLMechanism = "SCRAM-SHA-256"
	SASLSCRAMSHA512 SASLMechanism = "SCRAM-SHA-512"
)

// Config holds the Kafka connection configuration.
type Config struct {
	// Brokers is a list of Kafka broker addresses (e.g., ["localhost:9092"]).
	Brokers []string

	// Topic is the Kafka topic to produce to or consume from.
	Topic string

	// --- Authentication (optional) ---

	// Username for SASL authentication. Leave empty to disable auth.
	Username string

	// Password for SASL authentication.
	Password string

	// Mechanism is the SASL mechanism to use (PLAIN, SCRAM-SHA-256, SCRAM-SHA-512).
	// Defaults to PLAIN if Username is set and Mechanism is empty.
	Mechanism SASLMechanism

	// --- TLS (optional) ---

	// UseTLS enables TLS encryption for broker connections.
	UseTLS bool

	// TLSConfig allows providing a custom TLS configuration.
	// If nil and UseTLS is true, a default TLS config is used.
	TLSConfig *tls.Config

	// CACertFile is the path to a PEM-encoded CA certificate file.
	// Used to verify the broker's server certificate (e.g., self-signed or internal CA).
	CACertFile string

	// ClientCertFile is the path to a PEM-encoded client certificate file (for mTLS).
	ClientCertFile string

	// ClientKeyFile is the path to a PEM-encoded client private key file (for mTLS).
	ClientKeyFile string

	// TLSSkipVerify disables server certificate verification.
	// WARNING: Only use for testing. Do NOT use in production.
	TLSSkipVerify bool

	// --- Consumer-specific settings ---

	// GroupID is the consumer group ID (required for consumer).
	GroupID string

	// --- Timeouts ---

	// DialTimeout is the timeout for establishing connections. Default: 10s.
	DialTimeout time.Duration

	// ReadTimeout is the timeout for read operations. Default: 30s.
	ReadTimeout time.Duration

	// WriteTimeout is the timeout for write operations. Default: 30s.
	WriteTimeout time.Duration

	// --- Writer Tuning ---

	// RequiredAcks specifies the acknowledgment level (0 = None, 1 = One, -1 = All).
	// Defaults to RequireAll (-1) if nil.
	RequiredAcks *int

	// BatchSize is the maximum number of messages to batch before sending. Default: 100.
	BatchSize int

	// BatchTimeout is the maximum time to wait before flushing a batch. Default: 1s.
	BatchTimeout time.Duration

	// OnProduce is an optional callback called when a message is successfully produced or failed.
	// It receives the message with partition and offset populated, or an error.
	OnProduce func(msg Message, err error)
}

// Validate checks if the configuration has required fields.
func (c *Config) Validate() error {
	if len(c.Brokers) == 0 {
		return fmt.Errorf("kafkakit: at least one broker address is required")
	}
	if c.Topic == "" {
		return fmt.Errorf("kafkakit: topic is required")
	}
	return nil
}

// saslMechanism returns the sasl.Mechanism based on the config, or nil if no auth.
func (c *Config) saslMechanism() (sasl.Mechanism, error) {
	if c.Username == "" {
		return nil, nil
	}

	mechanism := c.Mechanism
	if mechanism == "" {
		mechanism = SASLPlain
	}

	switch mechanism {
	case SASLPlain:
		return &plain.Mechanism{
			Username: c.Username,
			Password: c.Password,
		}, nil

	case SASLSCRAMSHA256:
		m, err := scram.Mechanism(scram.SHA256, c.Username, c.Password)
		if err != nil {
			return nil, fmt.Errorf("kafkakit: failed to create SCRAM-SHA-256 mechanism: %w", err)
		}
		return m, nil

	case SASLSCRAMSHA512:
		m, err := scram.Mechanism(scram.SHA512, c.Username, c.Password)
		if err != nil {
			return nil, fmt.Errorf("kafkakit: failed to create SCRAM-SHA-512 mechanism: %w", err)
		}
		return m, nil

	default:
		return nil, fmt.Errorf("kafkakit: unsupported SASL mechanism: %s", mechanism)
	}
}

// tlsConfig returns the TLS config or nil.
func (c *Config) tlsConfig() (*tls.Config, error) {
	if !c.UseTLS {
		return nil, nil
	}
	if c.TLSConfig != nil {
		return c.TLSConfig, nil
	}

	tlsCfg := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: c.TLSSkipVerify,
	}

	// Load CA certificate
	if c.CACertFile != "" {
		caCert, err := os.ReadFile(c.CACertFile)
		if err != nil {
			return nil, fmt.Errorf("kafkakit: failed to read CA cert file %q: %w", c.CACertFile, err)
		}
		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("kafkakit: failed to parse CA cert file %q", c.CACertFile)
		}
		tlsCfg.RootCAs = caPool
	}

	// Load client certificate + key (mTLS)
	if c.ClientCertFile != "" && c.ClientKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(c.ClientCertFile, c.ClientKeyFile)
		if err != nil {
			return nil, fmt.Errorf("kafkakit: failed to load client cert/key: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}

	return tlsCfg, nil
}

// dialTimeout returns the configured dial timeout or default.
func (c *Config) dialTimeout() time.Duration {
	if c.DialTimeout > 0 {
		return c.DialTimeout
	}
	return 10 * time.Second
}

// readTimeout returns the configured read timeout or default.
func (c *Config) readTimeout() time.Duration {
	if c.ReadTimeout > 0 {
		return c.ReadTimeout
	}
	return 30 * time.Second
}

// writeTimeout returns the configured write timeout or default.
func (c *Config) writeTimeout() time.Duration {
	if c.WriteTimeout > 0 {
		return c.WriteTimeout
	}
	return 30 * time.Second
}
