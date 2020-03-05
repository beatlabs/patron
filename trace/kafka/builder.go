package kafka

import (
	"errors"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/beatlabs/patron/log"
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
// to construct a Kafka (synchronous or asynchronous) producer.
type Builder struct {
	Brokers     []string
	Cfg         *sarama.Config
	Enc         encoding.EncodeFunc
	ContentType string
	Errors      []error
}

// NewBuilder initiates the AsyncProducer builder chain.
// The builder instantiates the component using default values for
// EncodeFunc and Content-Type header.
func NewBuilder(brokers []string) *Builder {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V0_11_0_0

	errs := []error{}
	if len(brokers) == 0 {
		errs = append(errs, errors.New("brokers list is empty"))
	}

	return &Builder{
		Brokers:     brokers,
		Cfg:         cfg,
		Enc:         json.Encode,
		ContentType: json.Type,
		Errors:      errs,
	}
}

// WithTimeout sets the dial timeout for the AsyncProducer.
func (b *Builder) WithTimeout(dial time.Duration) *Builder {
	if dial <= 0 {
		b.Errors = append(b.Errors, errors.New("dial timeout has to be positive"))
		return b
	}
	b.Cfg.Net.DialTimeout = dial
	log.Info(fieldSetMsg, "dial timeout", dial)
	return b
}

// WithVersion sets the kafka versionfor the AsyncProducer.
func (b *Builder) WithVersion(version string) *Builder {
	if version == "" {
		b.Errors = append(b.Errors, errors.New("version is required"))
		return b
	}
	v, err := sarama.ParseKafkaVersion(version)
	if err != nil {
		b.Errors = append(b.Errors, errors.New("failed to parse kafka version"))
		return b
	}
	log.Info(fieldSetMsg, "version", version)
	b.Cfg.Version = v

	return b
}

// WithRequiredAcksPolicy adjusts how many replica acknowledgements
// broker must see before responding.
func (b *Builder) WithRequiredAcksPolicy(ack RequiredAcks) *Builder {
	if !isValidRequiredAcks(ack) {
		b.Errors = append(b.Errors, errors.New("invalid value for required acks policy provided"))
		return b
	}
	log.Info(fieldSetMsg, "required acks", ack)
	b.Cfg.Producer.RequiredAcks = sarama.RequiredAcks(ack)
	return b
}

// WithEncoder sets a specific encoder implementation and Content-Type string header;
// if no option is provided it defaults to json.
func (b *Builder) WithEncoder(enc encoding.EncodeFunc, contentType string) *Builder {
	if enc == nil {
		b.Errors = append(b.Errors, errors.New("encoder is nil"))
	} else {
		log.Info(fieldSetMsg, "encoder", enc)
		b.Enc = enc
	}
	if contentType == "" {
		b.Errors = append(b.Errors, errors.New("content type is empty"))
	} else {
		log.Info(fieldSetMsg, "content type", contentType)
		b.ContentType = contentType
	}

	return b
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
