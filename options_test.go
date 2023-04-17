package patron

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slog"
)

func toAttr(in []interface{}) []slog.Attr {
	out := make([]slog.Attr, 0, len(in))

	for _, attr := range in {
		out = append(out, attr.(slog.Attr))
	}

	return out
}

func toInterface(in []slog.Attr) []interface{} {
	out := make([]interface{}, 0, len(in))

	for _, attr := range in {
		out = append(out, attr)
	}

	return out
}

func TestLogFields(t *testing.T) {
	defaultFields := defaultLogFields("test", "1.0")
	fields := []slog.Attr{slog.String("key", "value")}
	fields1 := toAttr(defaultLogFields("name1", "version1"))
	type args struct {
		fields []slog.Attr
	}
	tests := map[string]struct {
		args        args
		want        config
		expectedErr string
	}{
		"empty fields": {args: args{fields: nil}, expectedErr: "fields are empty"},
		"success":      {args: args{fields: fields}, want: config{fields: append(defaultFields, toInterface(fields)...)}},
		"no overwrite": {args: args{fields: fields1}, want: config{fields: defaultFields}},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			svc := &Service{
				config: config{
					fields: defaultFields,
				},
			}

			err := WithLogFields(tt.args.fields...)(svc)

			if tt.expectedErr == "" {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, svc.config)
			} else {
				assert.EqualError(t, err, tt.expectedErr)
			}
		})
	}
}

func TestLogger(t *testing.T) {
	svc := &Service{}

	err := WithLogger(slog.Default())(svc)
	assert.NoError(t, err)
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
