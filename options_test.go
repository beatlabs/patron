package patron

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slog"
)

func TestLogFields(t *testing.T) {
	defaultAttrs := defaultLogAttrs("test", "1.0")
	attrs := []slog.Attr{slog.String("key", "value")}
	attrs1 := defaultLogAttrs("name1", "version1")
	type args struct {
		fields []slog.Attr
	}
	tests := map[string]struct {
		args        args
		want        logConfig
		expectedErr string
	}{
		"empty attributes": {args: args{fields: nil}, expectedErr: "attributes are empty"},
		"success":          {args: args{fields: attrs}, want: logConfig{attrs: append(defaultAttrs, attrs...)}},
		"no overwrite":     {args: args{fields: attrs1}, want: logConfig{attrs: defaultAttrs}},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			svc := &Service{
				logConfig: logConfig{
					attrs: defaultAttrs,
				},
			}

			err := WithLogFields(tt.args.fields...)(svc)

			if tt.expectedErr == "" {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, svc.logConfig)
			} else {
				assert.EqualError(t, err, tt.expectedErr)
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
		assert.Nil(t, nil, svc.sighupHandler)
	})

	t.Run("non empty value for sighup handler", func(t *testing.T) {
		t.Parallel()

		svc := &Service{}
		comp := &testSighupAlterable{}

		err := WithSIGHUP(testSighupHandle(comp))(svc)
		assert.Equal(t, nil, err)
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
