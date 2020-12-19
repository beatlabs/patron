// Package sqs provides a native consumer for AWS SQS.
package sqs

import (
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
)

// ProcessorFunc definition of a async processor.
type ProcessorFunc func(*sqs.ReceiveMessageOutput) error

// ConsumerFactory interface for creating SQS consumer.
type ConsumerFactory interface {
	Create() (sqsiface.SQSAPI, error)
}