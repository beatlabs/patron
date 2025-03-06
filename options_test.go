package patron

import (
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/beatlabs/patron/observability"
	"github.com/beatlabs/patron/observability/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogFields(t *testing.T) {
	attrs := []slog.Attr{slog.String("key", "value")}
	attrs1 := []slog.Attr{slog.String("name1", "version1")}

	expectedSuccess := observability.Config{LogConfig: log.Config{
		Attributes: attrs,
	}}
	expectedNoOverwrite := observability.Config{LogConfig: log.Config{
		Attributes: []slog.Attr{slog.String("name1", "version2")},
	}}

	type args struct {
		fields []slog.Attr
	}
	tests := map[string]struct {
		args        args
		want        observability.Config
		expectedErr string
	}{
		"empty attributes": {args: args{fields: nil}, expectedErr: "attributes are empty"},
		"success":          {args: args{fields: attrs}, want: expectedSuccess},
		"no overwrite":     {args: args{fields: attrs1}, want: expectedNoOverwrite},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			svc := &Service{
				observabilityCfg: observability.Config{},
			}

			err := WithLogFields(tt.args.fields...)(svc)

			if tt.expectedErr == "" {
				require.NoError(t, err)
				assert.Equal(t, tt.want, svc.observabilityCfg)
			} else {
				require.EqualError(t, err, tt.expectedErr)
			}
		})
	}
}

func TestSIGHUP(t *testing.T) {
	t.Parallel()

	t.Run("empty value for sighup handler", func(t *testing.T) {
		t.Parallel()
		svc := &Service{}

		err := WithSIGHUP(nil)(svc)
		assert.Equal(t, errors.New("provided WithSIGHUP handler was nil"), err)
		assert.Nil(t, svc.sighupHandler)
	})

	t.Run("non empty value for sighup handler", func(t *testing.T) {
		t.Parallel()

		svc := &Service{}
		comp := &testSighupAlterable{}

		err := WithSIGHUP(testSighupHandle(comp))(svc)
		require.NoError(t, err)
		assert.NotNil(t, svc.sighupHandler)
		svc.sighupHandler()
		assert.Equal(t, 1, comp.value)
	})
}

type testSighupAlterable struct {
	value int
}

func testSighupHandle(value *testSighupAlterable) func() {
	return func() {
		value.value = 1
	}
}

func TestShutdownTimeout(t *testing.T) {
	tests := map[string]struct {
		timeout     time.Duration
		expectedErr string
	}{
		"negative timeout": {
			timeout:     -5 * time.Second,
			expectedErr: "shutdown timeout must be positive",
		},
		"zero timeout": {
			timeout:     0,
			expectedErr: "shutdown timeout must be positive",
		},
		"positive timeout": {
			timeout:     30 * time.Second,
			expectedErr: "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			svc := &Service{}

			err := WithShutdownTimeout(tt.timeout)(svc)

			if tt.expectedErr == "" {
				require.NoError(t, err)
				assert.Equal(t, tt.timeout, svc.shutdownTimeout)
			} else {
				require.EqualError(t, err, tt.expectedErr)
			}
		})
	}
}
