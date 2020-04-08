package kafka

import (
	"errors"
	"fmt"
	"time"

	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/encoding/json"
	patronErrors "github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/internal/validation"
	"github.com/beatlabs/patron/log"

	"github.com/Shopify/sarama"
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

// Builder gathers all required and optional properties, in order
// to construct a Kafka AsyncProducer/SyncProducer.
type Builder struct {
	brokers     []string
	cfg         *sarama.Config
	enc         encoding.EncodeFunc
	contentType string
	errors      []error
}

// NewBuilder initiates the AsyncProducer/SyncProducer builder chain.
// The builder instantiates the component using default values for
// EncodeFunc and Content-Type header.
func NewBuilder(brokers []string) *Builder {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V0_11_0_0

	errs := []error{}
	if validation.IsStringSliceEmpty(brokers) {
		errs = append(errs, errors.New("brokers are empty or have an empty value"))
	}

	return &Builder{
		brokers:     brokers,
		cfg:         cfg,
		enc:         json.Encode,
		contentType: json.Type,
		errors:      errs,
	}
}

// WithTimeout sets the dial timeout for the sync or async producer.
func (ab *Builder) WithTimeout(dial time.Duration) *Builder {
	if dial <= 0 {
		ab.errors = append(ab.errors, errors.New("dial timeout has to be positive"))
		return ab
	}
	ab.cfg.Net.DialTimeout = dial
	log.Info(fieldSetMsg, "dial timeout", dial)
	return ab
}

// WithVersion sets the kafka versionfor the AsyncProducer/SyncProducer.
func (ab *Builder) WithVersion(version string) *Builder {
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
func (ab *Builder) WithRequiredAcksPolicy(ack RequiredAcks) *Builder {
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
func (ab *Builder) WithEncoder(enc encoding.EncodeFunc, contentType string) *Builder {
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

// CreateAsync constructs the AsyncProducer component by applying the gathered properties.
func (ab *Builder) CreateAsync() (*AsyncProducer, <-chan error, error) {

	if len(ab.errors) > 0 {
		return nil, nil, patronErrors.Aggregate(ab.errors...)
	}

	ap := AsyncProducer{
		baseProducer: baseProducer{
			messageStatus: messageStatus,
			deliveryType:  "async",
			cfg:           ab.cfg,
			enc:           ab.enc,
			contentType:   ab.contentType,
			tag:           opentracing.Tag{Key: "type", Value: "async"},
		},
	}

	var err error
	ap.prodClient, err = sarama.NewClient(ab.brokers, ab.cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create producer client: %w", err)
	}

	ap.asyncProd, err = sarama.NewAsyncProducerFromClient(ap.prodClient)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create async producer: %w", err)
	}
	ap.chErr = make(chan error)

	go ap.propagateError()

	return &ap, ap.chErr, nil
}

// CreateSync constructs the SyncProducer component by applying the gathered properties.
func (ab *Builder) CreateSync() (*SyncProducer, error) {
	if len(ab.errors) > 0 {
		return nil, patronErrors.Aggregate(ab.errors...)
	}

	// required for any SyncProducer; 'Errors' is already true by default for both async/sync producers
	ab.cfg.Producer.Return.Successes = true

	p := SyncProducer{
		baseProducer: baseProducer{
			messageStatus: messageStatus,
			deliveryType:  "sync",
			cfg:           ab.cfg,
			enc:           ab.enc,
			contentType:   ab.contentType,
			tag:           opentracing.Tag{Key: "type", Value: "sync"},
		},
	}

	var err error
	p.prodClient, err = sarama.NewClient(ab.brokers, ab.cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer client: %w", err)
	}

	p.syncProd, err = sarama.NewSyncProducerFromClient(p.prodClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create sync producer: %w", err)
	}

	return &p, nil
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
