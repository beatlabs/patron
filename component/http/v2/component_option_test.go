package v2

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTLS(t *testing.T) {
	t.Parallel()
	type args struct {
		cert string
		key  string
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success":      {args: args{cert: "cert", key: "key"}},
		"missing cert": {args: args{cert: "", key: "key"}, expectedErr: "cert file or key file was empty"},
		"missing key":  {args: args{cert: "cert", key: ""}, expectedErr: "cert file or key file was empty"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cmp := &Component{}
			err := WithTLS(tt.args.cert, tt.args.key)(cmp)

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.args.cert, cmp.certFile)
				assert.Equal(t, tt.args.key, cmp.keyFile)
			}
		})
	}
}

func TestReadTimeout(t *testing.T) {
	t.Parallel()
	type args struct {
		rt time.Duration
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success":      {args: args{rt: time.Second}},
		"missing cert": {args: args{rt: -1 * time.Second}, expectedErr: "negative or zero read timeout provided"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cmp := &Component{}
			err := WithReadTimeout(tt.args.rt)(cmp)

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.args.rt, cmp.readTimeout)
			}
		})
	}
}

func TestWriteTimeout(t *testing.T) {
	t.Parallel()
	type args struct {
		wt time.Duration
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success":      {args: args{wt: time.Second}},
		"missing cert": {args: args{wt: -1 * time.Second}, expectedErr: "negative or zero write timeout provided"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cmp := &Component{}
			err := WithWriteTimeout(tt.args.wt)(cmp)

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.args.wt, cmp.writeTimeout)
			}
		})
	}
}

func TestHandlerTimeout(t *testing.T) {
	t.Parallel()
	type args struct {
		wt time.Duration
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success":      {args: args{wt: time.Second}},
		"missing cert": {args: args{wt: -1 * time.Second}, expectedErr: "negative or zero handler timeout provided"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cmp := &Component{}
			err := WithHandlerTimeout(tt.args.wt)(cmp)

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.args.wt, cmp.handlerTimeout)
			}
		})
	}
}

func TestShutdownGracePeriod(t *testing.T) {
	t.Parallel()
	type args struct {
		gp time.Duration
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success":      {args: args{gp: time.Second}},
		"missing cert": {args: args{gp: -1 * time.Second}, expectedErr: "negative or zero shutdown grace period timeout provided"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cmp := &Component{}
			err := WithShutdownGracePeriod(tt.args.gp)(cmp)

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.args.gp, cmp.shutdownGracePeriod)
			}
		})
	}
}

func TestPort(t *testing.T) {
	t.Parallel()
	type args struct {
		port int
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success":      {args: args{port: 50000}},
		"missing cert": {args: args{port: 120000}, expectedErr: "invalid HTTP Port provided"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cmp := &Component{}
			err := WithPort(tt.args.port)(cmp)

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.args.port, cmp.port)
			}
		})
	}
}
