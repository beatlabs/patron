package mqtt

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
)

func TestMaxMessages(t *testing.T) {
	t.Parallel()
	type args struct {
		maxMessages *int64
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {
			args: args{maxMessages: aws.Int64(5)},
		},
		"zero message size": {
			args:        args{maxMessages: aws.Int64(0)},
			expectedErr: "max messages should be between 1 and 10",
		},
		"over max message size": {
			args:        args{maxMessages: aws.Int64(11)},
			expectedErr: "max messages should be between 1 and 10",
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := &Component{}
			err := MaxMessages(*tt.args.maxMessages)(c)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.cfg.maxMessages, tt.args.maxMessages)
			}
		})
	}
}
