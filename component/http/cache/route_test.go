package cache

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponseReadWriter_Header(t *testing.T) {
	rw := newResponseReadWriter()
	rw.Header().Set("key", "value")
	assert.Equal(t, "value", rw.Header().Get("key"))
}

func TestResponseReadWriter_StatusCode(t *testing.T) {
	rw := newResponseReadWriter()
	rw.WriteHeader(http.StatusContinue)
	assert.Equal(t, http.StatusContinue, rw.statusCode)
}

func TestResponseReadWriter_ReadWrite(t *testing.T) {
	rw := newResponseReadWriter()
	str := "body"
	i, err := rw.Write([]byte(str))
	require.NoError(t, err)

	r := make([]byte, i)
	j, err := rw.Read(r)
	require.NoError(t, err)

	assert.Equal(t, i, j)
	assert.Equal(t, str, string(r))
}

func TestResponseReadWriter_ReadWriteAll(t *testing.T) {
	rw := newResponseReadWriter()
	str := "body"
	i, err := rw.Write([]byte(str))
	require.NoError(t, err)

	b, err := rw.ReadAll()
	require.NoError(t, err)

	assert.Len(t, b, i)
	assert.Equal(t, str, string(b))
}

func TestResponseReadWriter_ReadAllEmpty(t *testing.T) {
	rw := newResponseReadWriter()

	b, err := rw.ReadAll()
	require.NoError(t, err)

	assert.Empty(t, b)
	assert.Empty(t, string(b))
}

func TestHandler(t *testing.T) {
	tc := newTestingCache()
	tc.instant = func() int64 { return 1 }
	rc, errs := NewRouteCache(tc, Age{Min: time.Second, Max: 10 * time.Second})
	require.Empty(t, errs)

	req := httptest.NewRequest(http.MethodGet, "/cached?key=value", nil)
	rec := httptest.NewRecorder()

	err := Handler(rec, req, rc, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Test", "value")
		_, writeErr := w.Write([]byte("payload"))
		assert.NoError(t, writeErr)
	}))

	require.NoError(t, err)
	assert.Equal(t, "payload", rec.Body.String())
	assert.Equal(t, "value", rec.Header().Get("X-Test"))
}

func TestHTTPExecutor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/cached", nil)
	exec := httpExecutor(httptest.NewRecorder(), req, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Test", "value")
		_, err := w.Write([]byte("payload"))
		assert.NoError(t, err)
	})

	got := exec(10, "key")

	require.NoError(t, got.Err)
	assert.Equal(t, []byte("payload"), got.Response.Bytes)
	assert.Equal(t, "value", got.Response.Header.Get("X-Test"))
	assert.Equal(t, int64(10), got.LastValid)
	assert.NotEmpty(t, got.Etag)
}
