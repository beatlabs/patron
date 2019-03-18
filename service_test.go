package patron

import (
	"strconv"
	"math/rand"
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thebeatapp/patron/errors"
	"github.com/thebeatapp/patron/sync/http"
)

func TestNewServer(t *testing.T) {
	route := http.NewRoute("/", "GET", nil, true, nil)
	type args struct {
		name string
		opt  OptionFunc
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"success", args{name: "test", opt: Routes([]http.Route{route})}, false},
		{"failed missing name", args{name: "", opt: Routes([]http.Route{route})}, true},
		{"failed missing routes", args{name: "test", opt: Routes([]http.Route{})}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.name, "", tt.args.opt)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}

func TestServer_Run_Shutdown(t *testing.T) {
	tests := []struct {
		name    string
		cp      Component
		wantErr bool
	}{
		{name: "success", cp: &testComponent{}, wantErr: false},
		{name: "failed to run", cp: &testComponent{errorRunning: true}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("PATRON_HTTP_DEFAULT_PORT", getRandomPort())
			s, err := New("test", "", Components(tt.cp, tt.cp, tt.cp))
			assert.NoError(t, err)
			err = s.Run()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func getRandomPort() string {
	rnd:= 50000+rand.Int63n(10000);
	return strconv.FormatInt(rnd, 10) 
}

type testComponent struct {
	errorRunning bool
}

func (ts testComponent) Run(ctx context.Context) error {
	if ts.errorRunning {
		return errors.New("failed to run component")
	}
	return nil
}

func (ts testComponent) Info() map[string]interface{} {
	return map[string]interface{}{"type": "mock"}
}
