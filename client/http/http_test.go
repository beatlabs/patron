package http

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/reliability/circuitbreaker"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestTracedClient_Do(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "true", r.Header.Get("Mockpfx-Ids-Sampled"))
		assert.NotEmpty(t, r.Header.Get("Mockpfx-Ids-Spanid"))
		assert.NotEmpty(t, r.Header.Get("Mockpfx-Ids-Traceid"))
		_, _ = fmt.Fprintln(w, "Hello, client")
	}))
	defer ts.Close()
	mtr := mocktracer.New()
	opentracing.SetGlobalTracer(mtr)
	c, err := New()
	assert.NoError(t, err)
	cb, err := New(WithCircuitBreaker("test", circuitbreaker.Setting{}))
	assert.NoError(t, err)
	ct, err := New(WithTransport(&http.Transport{}))
	assert.NoError(t, err)
	req, err := http.NewRequest("GET", ts.URL, nil)
	assert.NoError(t, err)
	reqErr, err := http.NewRequest("GET", "", nil)
	assert.NoError(t, err)
	reqErr.Header.Set(encoding.AcceptEncodingHeader, "gzip")
	u, err := req.URL.Parse(ts.URL)
	assert.NoError(t, err)
	opName := opName("GET", u.Scheme, u.Host)
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
				assert.Error(t, err)
				assert.Nil(t, rsp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, rsp)
			}
			sp := mtr.FinishedSpans()[0]
			assert.NotNil(t, sp)
			assert.Equal(t, tt.wantOpName, sp.OperationName)
			mtr.Reset()
			// Test counters.
			assert.Equal(t, tt.wantCounter, testutil.CollectAndCount(reqDurationMetrics, "client_http_request_duration_seconds"))
			reqDurationMetrics.Reset()
		})
	}
}

func TestTracedClient_Do_Redirect(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://google.com", http.StatusSeeOther)
	}))
	defer ts.Close()
	c, err := New(WithCheckRedirect(func(req *http.Request, via []*http.Request) error {
		return errors.New("stop redirects")
	}))
	assert.NoError(t, err)
	req, err := http.NewRequest("GET", ts.URL, nil)
	assert.NoError(t, err)

	res, err := c.Do(req)

	assert.Errorf(t, err, "stop redirects")
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
			WithCheckRedirect(func(req *http.Request, via []*http.Request) error { return nil }),
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
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}

func TestDecompress(t *testing.T) {
	const msg = "hello, client!"
	ts1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, msg)
	}))
	defer ts1.Close()

	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		_, err = fmt.Fprint(w, b.String())
		if err != nil {
			return
		}
	}))
	defer ts2.Close()

	ts3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		_, err = fmt.Fprint(w, b.String())
		if err != nil {
			return
		}
	}))
	defer ts3.Close()

	c, err := New()
	assert.NoError(t, err)

	tests := []struct {
		name string
		hdr  string
		url  string
	}{
		{"no compression", "", ts1.URL},
		{"gzip", "gzip", ts2.URL},
		{"deflate", "deflate", ts3.URL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.url, nil)
			assert.NoError(t, err)
			req.Header.Add(encoding.AcceptEncodingHeader, tt.hdr)
			rsp, err := c.Do(req)
			assert.Nil(t, err)

			b, err := ioutil.ReadAll(rsp.Body)
			assert.Nil(t, err)
			body := string(b)
			assert.Equal(t, msg, body)
		})
	}
}

func TestOpName(t *testing.T) {
	type args struct {
		method string
		scheme string
		host   string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"get http host", args{"GET", "http", "host"}, "GET http://host"},
		{"post https host:port", args{"POST", "https", "host:443"}, "POST https://host:443"},
		{"empty method", args{"", "http", "host"}, " http://host"},
		{"empty scheme", args{"GET", "", "host"}, "GET ://host"},
		{"empty host", args{"GET", "http", ""}, "GET http://"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := opName(tt.args.method, tt.args.scheme, tt.args.host); got != tt.want {
				t.Errorf("opName() = %v, want %v", got, tt.want)
			}
		})
	}
}
