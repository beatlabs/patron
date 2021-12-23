package v2

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponseWriter(t *testing.T) {
	rc := httptest.NewRecorder()
	rw := newResponseWriter(rc, true)

	_, err := rw.Write([]byte("test"))
	assert.NoError(t, err)
	rw.WriteHeader(202)

	assert.Equal(t, 202, rw.status, "status expected 202 but got %d", rw.status)
	assert.Len(t, rw.Header(), 1, "Header count expected to be 1")
	assert.True(t, rw.statusHeaderWritten, "expected to be true")
	assert.Equal(t, "test", rc.Body.String(), "body expected to be test but was %s", rc.Body.String())
}

func TestStripQueryString(t *testing.T) {
	type args struct {
		path string
	}
	tests := map[string]struct {
		args         args
		expectedPath string
		expectedErr  error
	}{
		"query string 1": {
			args:         args{path: "foo?bar=value1&baz=value2"},
			expectedPath: "foo",
		},
		"query string 2": {
			args:         args{path: "/foo?bar=value1&baz=value2"},
			expectedPath: "/foo",
		},
		"query string 3": {
			args:         args{path: "http://foo/bar?baz=value1"},
			expectedPath: "http://foo/bar",
		},
		"no query string": {
			args:         args{path: "http://foo/bar"},
			expectedPath: "http://foo/bar",
		},
		"empty": {
			args:         args{path: ""},
			expectedPath: "",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			s, err := stripQueryString(tt.args.path)
			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedPath, s)
			}
		})
	}
}
