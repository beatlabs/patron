package patron

import (
	"errors"
	"github.com/beatlabs/patron/log/std"
	"github.com/stretchr/testify/assert"
	"net/http"
	"os"
	"testing"
)

func TestComponents(t *testing.T) {
	comp := &testComponent{}

	type args struct {
		cc []Component
	}
	tests := []struct {
		name           string
		args           args
		wantComponents []Component
		wantError      error
	}{
		{
			name: "no components provided",
			args: args{
				cc: nil,
			},
			wantComponents: nil,
			wantError:      errors.New("provided components slice was empty"),
		},
		{
			name: "components provided",
			args: args{
				cc: []Component{comp},
			},
			wantComponents: []Component{comp},
			wantError:      nil,
		},
	}
	for _, tt := range tests {
		temp := tt
		t.Run(temp.name, func(t *testing.T) {
			svc := &Service{}
			err := Components(temp.args.cc...)(svc)
			assert.Equal(t, temp.wantError, err)
			assert.Equal(t, temp.wantComponents, svc.cps)
		})
	}
}

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

			err := LogFields(temp.args.fields)(svc)
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

	err := Logger(logger)(svc)
	assert.NoError(t, err)
	assert.Equal(t, logger, svc.config.logger)
}

func TestRouter(t *testing.T) {
	type args struct {
		handler http.Handler
	}

	tests := []struct {
		name        string
		args        args
		wantHandler http.Handler
		wantError   error
	}{
		{
			name: "empty value for handler",
			args: args{
				handler: nil,
			},
			wantHandler: nil,
			wantError:   errors.New("provided router is nil"),
		},
		{
			name: "non empty value for handler",
			args: args{
				handler: noopHTTPHandler{},
			},
			wantHandler: noopHTTPHandler{},
			wantError:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			temp := tt
			svc := &Service{}

			err := Router(temp.args.handler)(svc)
			assert.Equal(t, temp.wantError, err)
			assert.Equal(t, temp.wantHandler, svc.httpRouter)
		})
	}
}

type noopHTTPHandler struct{}

func (noopHTTPHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	return
}

func TestSIGHUP(t *testing.T) {
	t.Parallel()

	type args struct {
		handler func()
	}

	t.Run("empty value for sighup handler", func(t *testing.T) {
		t.Parallel()
		svc := &Service{}

		err := SIGHUP(nil)(svc)
		assert.Equal(t, errors.New("provided SIGHUP handler was nil"), err)
		assert.Nil(t, nil, svc.sighupHandler)
	})

	t.Run("non empty value for sighup handler", func(t *testing.T) {
		t.Parallel()

		svc := &Service{}
		comp := &testSighupAlterable{}

		err := SIGHUP(testSighupHandle(comp))(svc)
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
