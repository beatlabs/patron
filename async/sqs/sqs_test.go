package sqs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	type args struct {
		region string
		id     string
		secret string
		token  string
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {
			args: args{region: "region", id: "id", secret: "secret", token: "token"},
		},
		"missing region": {
			args:        args{region: "", id: "id", secret: "secret", token: "token"},
			expectedErr: "AWS region not provided",
		},
		"missing id": {
			args:        args{region: "region", id: "", secret: "secret", token: "token"},
			expectedErr: "AWS id not provided",
		},
		"missing secret": {
			args:        args{region: "region", id: "id", secret: "", token: "token"},
			expectedErr: "AWS secret not provided",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := NewConfig(tt.args.region, tt.args.id, tt.args.secret, tt.args.token)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.args.region, got.region)
				assert.Equal(t, tt.args.id, got.id)
				assert.Equal(t, tt.args.secret, got.secret)
				assert.Equal(t, tt.args.token, got.token)
			}
		})
	}
}
