package v2

import (
	"errors"
	"fmt"
	"os"

	"github.com/IBM/sarama"
	"github.com/beatlabs/patron/internal/validation"
)

// DefaultConfig creates a default Sarama configuration with idempotency enabled.
// See also:
// * https://pkg.go.dev/github.com/Shopify/sarama#RequiredAcks
// * https://pkg.go.dev/github.com/Shopify/sarama#Config
func DefaultConfig(name string, idempotent bool) (*sarama.Config, error) {
	host, err := os.Hostname()
	if err != nil {
		return nil, errors.New("failed to get hostname")
	}

	cfg := sarama.NewConfig()
	cfg.ClientID = fmt.Sprintf("%s-%s", host, name)

	if idempotent {
		cfg.Net.MaxOpenRequests = 1
		cfg.Producer.Idempotent = idempotent
	}

	cfg.Producer.RequiredAcks = sarama.WaitForAll

	return cfg, nil
}

func NewSyncProducer(brokers []string, cfg *sarama.Config) (*SyncProducer, error) {
	if validation.IsStringSliceEmpty(brokers) {
		return nil, errors.New("brokers are empty or have an empty value")
	}
	if cfg == nil {
		return nil, errors.New("no sarama configuration specified")
	}

	// required for any SyncProducer; 'Errors' is already true by default for both async/sync producers
	cfg.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer(brokers, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create sync producer: %w", err)
	}

	return &SyncProducer{producer: producer}, nil
}

func NewAsyncProducer(brokers []string, cfg *sarama.Config) (*AsyncProducer, error) {
	if validation.IsStringSliceEmpty(brokers) {
		return nil, errors.New("brokers are empty or have an empty value")
	}
	if cfg == nil {
		return nil, errors.New("no sarama configuration specified")
	}
	producer, err := sarama.NewAsyncProducer(brokers, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create async producer: %w", err)
	}
	return &AsyncProducer{producer: producer}, nil
}
