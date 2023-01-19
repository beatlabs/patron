package patron

import (
	"errors"
	"os"
	"testing"

	"github.com/beatlabs/patron/log/std"
	"github.com/stretchr/testify/assert"
)

func TestLogFields(t *testing.T) {
	defaultFields := defaultLogFields("test", "1.0")
	fields := map[string]interface{}{"key": "value"}
	fields1 := defaultLogFields("name1", "version1")
	type args struct {
		fields map[string]interface{}
	}
	tests := map[string]struct {
		args args
		want config
	}{
		"success":      {args: args{fields: fields}, want: config{fields: mergeFields(defaultFields, fields)}},
		"no overwrite": {args: args{fields: fields1}, want: config{fields: defaultFields}},
	}
	for name, tt := range tests {
		temp := tt
		t.Run(name, func(t *testing.T) {
			svc := &Service{
				config: config{
					fields: defaultFields,
				},
			}

			err := WithLogFields(temp.args.fields)(svc)
			assert.NoError(t, err)

			assert.Equal(t, temp.want, svc.config)
		})
	}
}

func mergeFields(ff1, ff2 map[string]interface{}) map[string]interface{} {
	ff := map[string]interface{}{}
	for k, v := range ff1 {
		ff[k] = v
	}
	for k, v := range ff2 {
		ff[k] = v
	}
	return ff
}

func TestLogger(t *testing.T) {
	logger := std.New(os.Stderr, getLogLevel(), nil)
	svc := &Service{}

	err := WithLogger(logger)(svc)
	assert.NoError(t, err)
	assert.Equal(t, logger, svc.config.logger)
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
