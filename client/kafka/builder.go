package kafka

import (
	"errors"
	"fmt"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/encoding/json"
	patronErrors "github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/internal/validation"
	"github.com/beatlabs/patron/log"
	"github.com/opentracing/opentracing-go"
)

// RequiredAcks is used in Produce Requests to tell the broker how many replica acknowledgements
// it must see before responding.
type RequiredAcks int16

const (
	// NoResponse doesn't send any response, the TCP ACK is all you get.
	NoResponse RequiredAcks = 0
	// WaitForLocal waits for only the local commit to succeed before responding.
	WaitForLocal RequiredAcks = 1
	// WaitForAll waits for all in-sync replicas to commit before responding.
	WaitForAll RequiredAcks = -1
)

const fieldSetMsg = "Setting property '%v' for '%v'"

// AsyncBuilder gathers all required and optional properties, in order
// to construct a Kafka AsyncProducer.
type AsyncBuilder struct {
	brokers     []string
	cfg         *sarama.Config
	chErr       chan error
	tag         opentracing.Tag
	enc         encoding.EncodeFunc
	contentType string
	errors      []error
}

// NewBuilder initiates the AsyncProducer builder chain.
// The builder instantiates the component using default values for
// EncodeFunc and Content-Type header.
func NewBuilder(brokers []string) *AsyncBuilder {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V0_11_0_0

	errs := []error{}
	if validation.IsStringSliceEmpty(brokers) {
		errs = append(errs, errors.New("brokers are empty or have an empty value"))
	}

	return &AsyncBuilder{
		brokers:     brokers,
		cfg:         cfg,
		chErr:       make(chan error),
		tag:         opentracing.Tag{Key: "type", Value: "async"},
		enc:         json.Encode,
		contentType: json.Type,
		errors:      errs,
	}
}

// WithTimeout sets the dial timeout for the AsyncProducer.
func (ab *AsyncBuilder) WithTimeout(dial time.Duration) *AsyncBuilder {
	if dial <= 0 {
		ab.errors = append(ab.errors, errors.New("dial timeout has to be positive"))
		return ab
	}
	ab.cfg.Net.DialTimeout = dial
	log.Info(fieldSetMsg, "dial timeout", dial)
	return ab
}

// WithVersion sets the kafka versionfor the AsyncProducer.
func (ab *AsyncBuilder) WithVersion(version string) *AsyncBuilder {
	if version == "" {
		ab.errors = append(ab.errors, errors.New("version is required"))
		return ab
	}
	v, err := sarama.ParseKafkaVersion(version)
	if err != nil {
		ab.errors = append(ab.errors, errors.New("failed to parse kafka version"))
		return ab
	}
	log.Info(fieldSetMsg, "version", version)
	ab.cfg.Version = v

	return ab
}

// WithRequiredAcksPolicy adjusts how many replica acknowledgements
// broker must see before responding.
func (ab *AsyncBuilder) WithRequiredAcksPolicy(ack RequiredAcks) *AsyncBuilder {
	if !isValidRequiredAcks(ack) {
		ab.errors = append(ab.errors, errors.New("invalid value for required acks policy provided"))
		return ab
	}
	log.Info(fieldSetMsg, "required acks", ack)
	ab.cfg.Producer.RequiredAcks = sarama.RequiredAcks(ack)
	return ab
}

// WithEncoder sets a specific encoder implementation and Content-Type string header;
// if no option is provided it defaults to json.
func (ab *AsyncBuilder) WithEncoder(enc encoding.EncodeFunc, contentType string) *AsyncBuilder {
	if enc == nil {
		ab.errors = append(ab.errors, errors.New("encoder is nil"))
	} else {
		log.Info(fieldSetMsg, "encoder", enc)
		ab.enc = enc
	}
	if contentType == "" {
		ab.errors = append(ab.errors, errors.New("content type is empty"))
	} else {
		log.Info(fieldSetMsg, "content type", contentType)
		ab.contentType = contentType
	}

	return ab
}

// Create constructs the AsyncProducer component by applying the gathered properties.
func (ab *AsyncBuilder) Create() (*AsyncProducer, error) {

	if len(ab.errors) > 0 {
		return nil, patronErrors.Aggregate(ab.errors...)
	}

	prodClient, err := sarama.NewClient(ab.brokers, ab.cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create async producer client: %w", err)
	}
	prod, err := sarama.NewAsyncProducerFromClient(prodClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create async producer: %w", err)
	}

	ap := AsyncProducer{
		cfg:         ab.cfg,
		prodClient:  prodClient,
		prod:        prod,
		chErr:       ab.chErr,
		enc:         ab.enc,
		contentType: ab.contentType,
		tag:         ab.tag,
	}

	go ap.propagateError()
	return &ap, nil
}

func isValidRequiredAcks(ack RequiredAcks) bool {
	switch ack {
	case
		NoResponse,
		WaitForLocal,
		WaitForAll:
		return true
	}
	return false
}
