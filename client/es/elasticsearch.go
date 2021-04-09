// Package es provides a client with included tracing capabilities.
package es

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/trace"
	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/elastic/go-elasticsearch/v8/estransport"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	defaultHostEnv = "PATRON_ES_DEFAULT_HOST"
	defaultPortEnv = "PATRON_ES_DEFAULT_PORT"

	defaultHost = "http://localhost"
	defaultPort = "9200"

	respondentTag = "respondent"

	opName  = "Elasticsearch Call"
	cmpName = "go-elasticsearch"
)

var reqDurationMetrics *prometheus.HistogramVec

func init() {
	reqDurationMetrics = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "client",
			Subsystem: "elasticsearch",
			Name:      "request_duration_seconds",
			Help:      "Elasticsearch requests completed by the client.",
		},
		[]string{"method", "url", "status_code"},
	)
	prometheus.MustRegister(reqDurationMetrics)
}

type tracingInfo struct {
	user  string
	hosts []string
}

func (t *tracingInfo) startSpan(req *http.Request) (opentracing.Span, error) {
	if req == nil {
		return nil, fmt.Errorf("request is empty")
	}

	uri := req.URL.RequestURI()
	method := req.Method

	var bodyFmt string
	if req.Body != nil {
		if rawBody, err := ioutil.ReadAll(req.Body); err == nil {
			req.Body = ioutil.NopCloser(bytes.NewReader(rawBody))
			bodyFmt = string(rawBody)
		}
	}

	sp, _ := opentracing.StartSpanFromContext(req.Context(), opName)
	ext.Component.Set(sp, cmpName)
	ext.DBType.Set(sp, "elasticsearch")
	ext.DBUser.Set(sp, t.user)

	ext.HTTPUrl.Set(sp, uri)
	ext.HTTPMethod.Set(sp, method)
	ext.DBStatement.Set(sp, bodyFmt)

	hostsFmt := "[" + strings.Join(t.hosts, ", ") + "]"
	sp.SetTag(trace.HostsTag, hostsFmt)
	sp.SetTag(trace.VersionTag, trace.Version)

	return sp, nil
}

func endSpan(sp opentracing.Span, rsp *http.Response) {
	// In cases where more than one host is given, the selected one is only known at this time
	sp.SetTag(respondentTag, rsp.Request.URL.Host)

	ext.HTTPStatusCode.Set(sp, uint16(rsp.StatusCode))
	ext.Error.Set(sp, rsp.StatusCode >= http.StatusInternalServerError)
	sp.Finish()
}

type transportClient struct {
	client *estransport.Client
	tracingInfo
}

// Perform wraps elasticsearch Perform with tracing functionality
func (c *transportClient) Perform(req *http.Request) (*http.Response, error) {
	sp, err := c.startSpan(req)
	if err != nil {
		log.FromContext(req.Context()).Errorf("failed to start span: %v", err)
	}
	start := time.Now()
	rsp, err := c.client.Perform(req)
	if err != nil || rsp == nil {
		trace.SpanError(sp)
		return rsp, err
	}

	observeResponse(req, sp, rsp, start)
	return rsp, nil
}

func observeResponse(req *http.Request, sp opentracing.Span, rsp *http.Response, start time.Time) {
	endSpan(sp, rsp)
	reqDurationMetrics.WithLabelValues(req.Method, req.URL.Host, strconv.Itoa(rsp.StatusCode)).
		Observe(time.Since(start).Seconds())
}

// Config is a wrapper for elasticsearch.Config
type Config elasticsearch.Config

// Client is a wrapper for elasticsearch.Client
type Client struct {
	elasticsearch.Client
}

// NewDefaultClient returns an empty ES client with sane defaults
func NewDefaultClient() (*Client, error) {
	return NewClient(Config{})
}

// NewClient is a modified version of elasticsearch.NewClient
// that injects a tracing-ready transport.
func NewClient(cfg Config) (*Client, error) {
	urls, err := addrsToURLs(cfg.Addresses)
	if err != nil {
		return nil, fmt.Errorf("cannot create client: %s", err)
	}

	if len(urls) == 0 {
		// Fallback to default values
		addr := getAddrFromEnv()
		u, err := url.Parse(addr)
		if err != nil {
			return nil, err
		}
		urls = append(urls, u)
		cfg.Addresses = append(cfg.Addresses, addr)
	}

	esTransportClient := estransport.New(estransport.Config{
		URLs:     urls,
		Username: cfg.Username,
		Password: cfg.Password,
		APIKey:   cfg.APIKey,

		Transport: cfg.Transport,
		Logger:    cfg.Logger,
	})
	tracingInfo := tracingInfo{
		user:  cfg.Username,
		hosts: cfg.Addresses,
	}
	tp := &transportClient{
		client:      esTransportClient,
		tracingInfo: tracingInfo,
	}

	return &Client{
		elasticsearch.Client{
			Transport: tp,
			API:       esapi.New(tp),
		},
	}, nil
}

func addrsToURLs(addrs []string) ([]*url.URL, error) {
	urls := make([]*url.URL, 0, len(addrs))
	for _, addr := range addrs {
		u, err := url.Parse(strings.TrimRight(addr, "/"))
		if err != nil {
			return nil, fmt.Errorf("cannot parse url: %v", err)
		}

		urls = append(urls, u)
	}
	return urls, nil
}

func getAddrFromEnv() string {
	host, found := os.LookupEnv(defaultHostEnv)
	if !found {
		host = defaultHost
	}
	port, found := os.LookupEnv(defaultPortEnv)
	if !found {
		port = defaultPort
	}

	return host + ":" + port
}
