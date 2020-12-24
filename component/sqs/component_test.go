package sqs

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	type args struct {
		name      string
		queueName string
		queueURL  string
		sqsAPI    sqsiface.SQSAPI
		proc      ProcessorFunc
		oo        []OptionFunc
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {
			args: args{
				name:      "name",
				queueName: "queueName",
				queueURL:  "queueURL",
				sqsAPI:    &stubSQSAPI{},
				proc:      stubProcFunc,
				oo:        []OptionFunc{Retries(5)},
			},
		},
		"missing name": {
			args: args{
				name:      "",
				queueName: "queueName",
				queueURL:  "queueURL",
				sqsAPI:    &stubSQSAPI{},
				proc:      stubProcFunc,
				oo:        []OptionFunc{Retries(5)},
			},
			expectedErr: "component name is empty",
		},
		"missing queue name": {
			args: args{
				name:      "name",
				queueName: "",
				queueURL:  "queueURL",
				sqsAPI:    &stubSQSAPI{},
				proc:      stubProcFunc,
				oo:        []OptionFunc{Retries(5)},
			},
			expectedErr: "queue name is empty",
		},
		"missing queue URL": {
			args: args{
				name:      "name",
				queueName: "queueName",
				queueURL:  "",
				sqsAPI:    &stubSQSAPI{},
				proc:      stubProcFunc,
				oo:        []OptionFunc{Retries(5)},
			},
			expectedErr: "queue URL is empty",
		},
		"missing queue SQS API": {
			args: args{
				name:      "name",
				queueName: "queueName",
				queueURL:  "queueURL",
				sqsAPI:    nil,
				proc:      stubProcFunc,
				oo:        []OptionFunc{Retries(5)},
			},
			expectedErr: "SQS API is nil",
		},
		"missing process function": {
			args: args{
				name:      "name",
				queueName: "queueName",
				queueURL:  "queueURL",
				sqsAPI:    &stubSQSAPI{},
				proc:      nil,
				oo:        []OptionFunc{Retries(5)},
			},
			expectedErr: "process function is nil",
		},
		"retry option fails": {
			args: args{
				name:      "name",
				queueName: "queueName",
				queueURL:  "queueURL",
				sqsAPI:    &stubSQSAPI{},
				proc:      stubProcFunc,
				oo:        []OptionFunc{RetryWait(-1 * time.Second)},
			},
			expectedErr: "retry wait time should be a positive number",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := New(tt.args.name, tt.args.queueName, tt.args.queueURL, tt.args.sqsAPI, tt.args.proc, tt.args.oo...)

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}

func stubProcFunc(context.Context, Batch) {
}
