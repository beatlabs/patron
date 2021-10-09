package simple

import (
	"testing"

	"github.com/beatlabs/patron/component/kafka"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	type args struct {
		name    string
		brokers []string
		topic   string
		proc    kafka.BatchProcessorFunc
		options OptionFunc
	}
	tests := map[string]struct {
		args        args
		expectedErr bool
	}{
		"success": {
			args: args{
				name:    "name",
				brokers: []string{"localhost"},
				topic:   "topic",
				proc:    func(batch kafka.Batch) error { return nil },
				options: FailureStrategy(kafka.ExitStrategy),
			},
		},
		"missing name": {
			args: args{
				brokers: []string{"localhost"},
				topic:   "topic",
				proc:    func(batch kafka.Batch) error { return nil },
			},
			expectedErr: true,
		},
		"missing broker": {
			args: args{
				name:  "name",
				topic: "topic",
				proc:  func(batch kafka.Batch) error { return nil },
			},
			expectedErr: true,
		},
		"missing topic": {
			args: args{
				name:    "name",
				brokers: []string{"localhost"},
				proc:    func(batch kafka.Batch) error { return nil },
			},
			expectedErr: true,
		},
		"missing proc": {
			args: args{
				name:    "name",
				brokers: []string{"localhost"},
				topic:   "topic",
			},
			expectedErr: true,
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			comp, err := New(tt.args.name, tt.args.brokers, tt.args.topic, tt.args.proc)
			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, kafka.ExitStrategy, comp.failStrategy)
			}
		})
	}
}
