package http

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/reliability/circuitbreaker"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/mocktracer"
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
	cb, err := New(CircuitBreaker("test", circuitbreaker.Setting{}))
	assert.NoError(t, err)
	ct, err := New(Transport(&http.Transport{}))
	assert.NoError(t, err)
	req, err := http.NewRequest("GET", ts.URL, nil)
	assert.NoError(t, err)
	reqErr, err := http.NewRequest("GET", "", nil)
	assert.NoError(t, err)
	opName := opName("GET", ts.URL)
	opNameError := "HTTP GET"

	type args struct {
		c   Client
		req *http.Request
	}
	tests := []struct {
		name       string
		args       args
		wantErr    bool
		wantOpName string
	}{
		{name: "respose", args: args{c: c, req: req}, wantErr: false, wantOpName: opName},
		{name: "response with circuit breaker", args: args{c: cb, req: req}, wantErr: false, wantOpName: opName},
		{name: "respose with custom transport", args: args{c: ct, req: req}, wantErr: false, wantOpName: opName},
		{name: "error", args: args{c: cb, req: reqErr}, wantErr: true, wantOpName: opNameError},
		{name: "error with circuit breaker", args: args{c: cb, req: reqErr}, wantErr: true, wantOpName: opNameError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rsp, err := tt.args.c.Do(context.Background(), tt.args.req)
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
		})
	}
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
		{name: "success", args: args{oo: []OptionFunc{Timeout(time.Second), CircuitBreaker("test", circuitbreaker.Setting{}), Transport(&http.Transport{})}}, wantErr: false},
		{name: "failure, invalid timeout", args: args{oo: []OptionFunc{Timeout(0 * time.Second)}}, wantErr: true},
		{name: "failure, invalid circuit breaker", args: args{[]OptionFunc{CircuitBreaker("", circuitbreaker.Setting{})}}, wantErr: true},
		{name: "failure, invalid transport", args: args{[]OptionFunc{Transport(nil)}}, wantErr: true},
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

func TestHTTPStartFinishSpan(t *testing.T) {
	mtr := mocktracer.New()
	opentracing.SetGlobalTracer(mtr)
	req, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)
	sp, req := span("/", "corID", req)
	assert.NotNil(t, sp)
	assert.NotNil(t, req)
	assert.IsType(t, &mocktracer.MockSpan{}, sp)
	jsp := sp.(*mocktracer.MockSpan)
	assert.NotNil(t, jsp)
	assert.Equal(t, "GET /", jsp.OperationName)
	finishSpan(jsp, 200)
	assert.NotNil(t, jsp)
	rawSpan := mtr.FinishedSpans()[0]
	assert.Equal(t, map[string]interface{}{
		"span.kind":        ext.SpanKindRPCServerEnum,
		"component":        "http-client",
		"error":            false,
		"http.method":      "GET",
		"http.status_code": uint16(200),
		"http.url":         "/",
		"version":          "dev",
		"correlationID":    "corID",
	}, rawSpan.Tags())
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
			rsp, err := c.Do(context.Background(), req)
			assert.Nil(t, err)

			b, err := ioutil.ReadAll(rsp.Body)
			assert.Nil(t, err)
			body := string(b)
			assert.Equal(t, msg, body)
		})
	}
}
