package async

import (
	"context"
	"fmt"

	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/beatlabs/patron/encoding/protobuf"
)

// FailStrategy type definition.
type FailStrategy int

const (
	// NackExitStrategy does not acknowledge the message and exits the application on error.
	NackExitStrategy FailStrategy = iota
	// NackStrategy does not acknowledge the message, leaving it for reprocessing, and continues.
	NackStrategy
	// AckStrategy acknowledges message and continues.
	AckStrategy
)

// ProcessorFunc definition of a async processor.
type ProcessorFunc func(Message) error

// Message interface for defining messages that are handled by the async component.
type Message interface {
	Context() context.Context
	Decode(v interface{}) error
	Ack() error
	Nack() error
	Source() string
}

// ConsumerFactory interface for creating consumers.
type ConsumerFactory interface {
	Create() (Consumer, error)
}

// Consumer interface which every specific consumer has to implement.
type Consumer interface {
	Consume(context.Context) (<-chan Message, <-chan error, error)
	Close() error
}

// DetermineDecoder determines the decoder based on the content type.
func DetermineDecoder(contentType string) (encoding.DecodeRawFunc, error) {
	switch contentType {
	case json.Type, json.TypeCharset:
		return json.DecodeRaw, nil
	case protobuf.Type, protobuf.TypeGoogle:
		return protobuf.DecodeRaw, nil
	}
	return nil, fmt.Errorf("content header %s is unsupported", contentType)
}
