package http

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/reliability/circuitbreaker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTracedClient_Do(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintln(w, "Hello, client")
	}))
	defer ts.Close()
	c, err := New()
	require.NoError(t, err)
	cb, err := New(WithCircuitBreaker("test", circuitbreaker.Setting{}))
	require.NoError(t, err)
	ct, err := New(WithTransport(&http.Transport{}))
	require.NoError(t, err)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, ts.URL, nil)
	require.NoError(t, err)
	reqErr, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "", nil)
	require.NoError(t, err)
	reqErr.Header.Set(encoding.AcceptEncodingHeader, "gzip")
	u, err := req.URL.Parse(ts.URL)
	require.NoError(t, err)
	opName := opName(http.MethodGet, u.Scheme, u.Host)
	opNameError := "HTTP GET"

	type args struct {
		c   Client
		req *http.Request
	}
	tests := []struct {
		name        string
		args        args
		wantErr     bool
		wantOpName  string
		wantCounter int
	}{
		{name: "response", args: args{c: c, req: req}, wantErr: false, wantOpName: opName, wantCounter: 1},
		{name: "response with circuit breaker", args: args{c: cb, req: req}, wantErr: false, wantOpName: opName, wantCounter: 1},
		{name: "response with custom transport", args: args{c: ct, req: req}, wantErr: false, wantOpName: opName, wantCounter: 1},
		{name: "error", args: args{c: cb, req: reqErr}, wantErr: true, wantOpName: opNameError, wantCounter: 0},
		{name: "error with circuit breaker", args: args{c: cb, req: reqErr}, wantErr: true, wantOpName: opNameError, wantCounter: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rsp, err := tt.args.c.Do(tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, rsp)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, rsp)
				require.NoError(t, rsp.Body.Close())
			}
		})
	}
}

func TestTracedClient_Do_Redirect(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://google.com", http.StatusSeeOther)
	}))
	defer ts.Close()
	c, err := New(WithCheckRedirect(func(_ *http.Request, _ []*http.Request) error {
		return errors.New("stop redirects")
	}))
	require.NoError(t, err)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, ts.URL, nil)
	require.NoError(t, err)

	res, err := c.Do(req)
	defer require.NoError(t, res.Body.Close())

	require.Errorf(t, err, "stop redirects")
	assert.NotNil(t, res)
	assert.Equal(t, http.StatusSeeOther, res.StatusCode)
}

func TestNew(t *testing.T) {
	type args struct {
		oo []OptionFunc
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "success", args: args{oo: []OptionFunc{
			WithTimeout(time.Second),
			WithCircuitBreaker("test", circuitbreaker.Setting{}),
			WithTransport(&http.Transport{}),
			WithCheckRedirect(func(_ *http.Request, _ []*http.Request) error { return nil }),
		}}, wantErr: false},
		{name: "failure, invalid timeout", args: args{oo: []OptionFunc{WithTimeout(0 * time.Second)}}, wantErr: true},
		{name: "failure, invalid circuit breaker", args: args{[]OptionFunc{WithCircuitBreaker("", circuitbreaker.Setting{})}}, wantErr: true},
		{name: "failure, invalid transport", args: args{[]OptionFunc{WithTransport(nil)}}, wantErr: true},
		{name: "failure, invalid check redirect", args: args{[]OptionFunc{WithCheckRedirect(nil)}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.oo...)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}

func TestDecompress(t *testing.T) {
	const msg = "hello, client!"
	ts1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, msg)
	}))
	defer ts1.Close()

	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		var b bytes.Buffer
		cw := gzip.NewWriter(&b)
		_, err := cw.Write([]byte(msg))
		if err != nil {
			return
		}
		err = cw.Close()
		if err != nil {
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		_, err = fmt.Fprint(w, b.String())
		if err != nil {
			return
		}
	}))
	defer ts2.Close()

	ts3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		var b bytes.Buffer
		cw, _ := flate.NewWriter(&b, 8)
		_, err := cw.Write([]byte(msg))
		if err != nil {
			return
		}
		err = cw.Close()
		if err != nil {
			return
		}
		w.Header().Set("Content-Encoding", "deflate")
		_, err = fmt.Fprint(w, b.String())
		if err != nil {
			return
		}
	}))
	defer ts3.Close()

	c, err := New()
	require.NoError(t, err)

	tests := []struct {
		name string
		url  string
	}{
		{"no compression", ts1.URL},
		{encodingGzip, ts2.URL},
		{encodingDeflate, ts3.URL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, tt.url, nil)
			require.NoError(t, err)
			rsp, err := c.Do(req)
			require.NoError(t, err)

			b, err := io.ReadAll(rsp.Body)
			require.NoError(t, err)
			require.Equal(t, msg, string(b))
			require.NoError(t, rsp.Body.Close())
		})
	}
}

func TestTracedClient_Do_DecompressError(t *testing.T) {
	// Server claims gzip but sends garbage — transport auto-decompression is
	// disabled so our decompress() call in Do() sees the invalid body.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		_, _ = w.Write([]byte("not valid gzip"))
	}))
	defer ts.Close()

	c, err := New(WithTransport(&http.Transport{DisableCompression: true}))
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, ts.URL, nil)
	require.NoError(t, err)

	rsp, err := c.Do(req)
	require.Error(t, err)
	if rsp != nil {
		_ = rsp.Body.Close()
	}
	assert.Nil(t, rsp)
}

func TestDecompressUnit(t *testing.T) {
	const msg = "hello, patron!"

	t.Run("gzip success", func(t *testing.T) {
		var b bytes.Buffer
		w := gzip.NewWriter(&b)
		_, err := w.Write([]byte(msg))
		require.NoError(t, err)
		require.NoError(t, w.Close())

		rc, err := decompress("gzip", io.NopCloser(&b))
		require.NoError(t, err)
		data, err := io.ReadAll(rc)
		require.NoError(t, err)
		assert.Equal(t, msg, string(data))
		require.NoError(t, rc.Close())
	})

	t.Run("gzip invalid data", func(t *testing.T) {
		rc, err := decompress("gzip", io.NopCloser(strings.NewReader("not gzip")))
		require.Error(t, err)
		assert.Nil(t, rc)
	})

	t.Run("unknown encoding passthrough", func(t *testing.T) {
		rc, err := decompress("br", io.NopCloser(strings.NewReader(msg)))
		require.NoError(t, err)
		data, err := io.ReadAll(rc)
		require.NoError(t, err)
		assert.Equal(t, msg, string(data))
	})
}
