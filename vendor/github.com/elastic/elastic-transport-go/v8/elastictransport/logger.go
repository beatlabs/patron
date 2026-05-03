// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package elastictransport

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// Log key constants used by [LoggingInterceptor]. Custom [slog.Handler]
// implementations can switch on these keys to format round-trip data.
const (
	LogKeyMethod       = "method"
	LogKeyURL          = "url"
	LogKeyStatus       = "status"
	LogKeyDuration     = "duration"
	LogKeyRequestBody  = "request.body"
	LogKeyResponseBody = "response.body"
	LogKeyError        = "error"
)

type loggerContextKey struct{}

// ContextWithLogger returns a copy of ctx with the given [LeveledLogger]
// attached. Use this to override the transport's logger on a per-request basis,
// or to make the logger available to [InterceptorFunc] implementations via
// [LoggerFromContext].
func ContextWithLogger(ctx context.Context, l LeveledLogger) context.Context {
	return context.WithValue(ctx, loggerContextKey{}, l)
}

// LoggerFromContext returns the [LeveledLogger] attached to ctx by
// [ContextWithLogger], or nil if none is set.
//
// This is primarily useful inside [InterceptorFunc] implementations:
//
//	interceptor := func(next elastictransport.RoundTripFunc) elastictransport.RoundTripFunc {
//	    return func(req *http.Request) (*http.Response, error) {
//	        if logger := elastictransport.LoggerFromContext(req.Context()); logger != nil {
//	            logger.Debug(req.Context(), "interceptor called", "method", req.Method, "url", req.URL.String())
//	        }
//	        return next(req)
//	    }
//	}
func LoggerFromContext(ctx context.Context) LeveledLogger {
	if l, ok := ctx.Value(loggerContextKey{}).(LeveledLogger); ok {
		return l
	}
	return nil
}

// Logger defines an interface for logging request and response.
//
// Deprecated: Use [LeveledLogger] and [WithLeveledLogger] instead.
// When a [LeveledLogger] is configured, round-trip logging is handled
// automatically.
type Logger interface {
	// LogRoundTrip should not modify the request or response, except for consuming and closing the body.
	// Implementations have to check for nil values in request and response.
	LogRoundTrip(*http.Request, *http.Response, error, time.Time, time.Duration) error
	// RequestBodyEnabled makes the client pass a copy of request body to the logger.
	RequestBodyEnabled() bool
	// ResponseBodyEnabled makes the client pass a copy of response body to the logger.
	ResponseBodyEnabled() bool
}

// LeveledLogger defines a structured, leveled logger for transport-internal
// events such as connection management and node discovery.
//
// Each method accepts a human-readable message and an optional sequence of
// alternating key-value pairs (the same convention used by [log/slog]).
//
// Implementations must be safe for concurrent use.
//
// Use [WithLeveledLogger] to configure a leveled logger on the transport client.
// See [SlogLogger] for a ready-made adapter that wraps [*slog.Logger].
type LeveledLogger interface {
	Debug(ctx context.Context, msg string, keysAndValues ...any)
	Info(ctx context.Context, msg string, keysAndValues ...any)
	Warn(ctx context.Context, msg string, keysAndValues ...any)
	Error(ctx context.Context, msg string, keysAndValues ...any)
}

// SlogLogger is a [LeveledLogger] adapter that delegates to a [*slog.Logger].
//
// The optional ContextAttrs function extracts additional key-value pairs from
// the request context and prepends them to every log entry. This is useful for
// injecting trace IDs, request IDs, or other context-carried metadata without
// writing a custom [slog.Handler].
//
// Example with OpenTelemetry trace context:
//
//	logger := &elastictransport.SlogLogger{
//	    Logger: slog.Default(),
//	    ContextAttrs: func(ctx context.Context) []any {
//	        sc := trace.SpanFromContext(ctx).SpanContext()
//	        if !sc.IsValid() {
//	            return nil
//	        }
//	        return []any{"trace_id", sc.TraceID().String(), "span_id", sc.SpanID().String()}
//	    },
//	}
type SlogLogger struct {
	Logger       *slog.Logger
	ContextAttrs func(ctx context.Context) []any
}

func (l *SlogLogger) Debug(ctx context.Context, msg string, keysAndValues ...any) {
	l.Logger.DebugContext(ctx, msg, l.enrich(ctx, keysAndValues)...)
}

func (l *SlogLogger) Info(ctx context.Context, msg string, keysAndValues ...any) {
	l.Logger.InfoContext(ctx, msg, l.enrich(ctx, keysAndValues)...)
}

func (l *SlogLogger) Warn(ctx context.Context, msg string, keysAndValues ...any) {
	l.Logger.WarnContext(ctx, msg, l.enrich(ctx, keysAndValues)...)
}

func (l *SlogLogger) Error(ctx context.Context, msg string, keysAndValues ...any) {
	l.Logger.ErrorContext(ctx, msg, l.enrich(ctx, keysAndValues)...)
}

func (l *SlogLogger) enrich(ctx context.Context, kv []any) []any {
	if l.ContextAttrs == nil {
		return kv
	}
	extra := l.ContextAttrs(ctx)
	if len(extra) == 0 {
		return kv
	}
	return append(extra, kv...)
}

// OTelContextAttrs extracts the OpenTelemetry trace_id and span_id from ctx
// and returns them as structured key-value pairs suitable for use as a
// [SlogLogger.ContextAttrs] function. Returns nil when no valid span context
// is present.
//
//	logger := &elastictransport.SlogLogger{
//	    Logger:       slog.Default(),
//	    ContextAttrs: elastictransport.OTelContextAttrs,
//	}
func OTelContextAttrs(ctx context.Context) []any {
	sc := trace.SpanFromContext(ctx).SpanContext()
	if !sc.IsValid() {
		return nil
	}
	return []any{
		"trace_id", sc.TraceID().String(),
		"span_id", sc.SpanID().String(),
	}
}

// LoggingInterceptor returns an [InterceptorFunc] that logs each round-trip
// using the [LeveledLogger] from the request context (set automatically by
// the transport when [WithLeveledLogger] is configured, or manually via
// [ContextWithLogger]).
//
// Successful requests are logged at Info level; errors at Error level.
// Set enableRequestBody and/or enableResponseBody to include bodies in the
// log output.
//
// Example:
//
//	tp, _ := elastictransport.NewClient(
//	    elastictransport.WithURLs(u),
//	    elastictransport.WithLeveledLogger(logger),
//	    elastictransport.WithInterceptors(
//	        elastictransport.LoggingInterceptor(false, false),
//	    ),
//	)
func LoggingInterceptor(enableRequestBody, enableResponseBody bool) InterceptorFunc {
	return func(next RoundTripFunc) RoundTripFunc {
		return func(req *http.Request) (*http.Response, error) {
			logger := LoggerFromContext(req.Context())
			if logger == nil {
				return next(req)
			}

			start := time.Now().UTC()
			res, err := next(req)
			dur := time.Since(start)

			kvCap := 8 // method, url, duration, status
			if enableRequestBody {
				kvCap += 2
			}
			if enableResponseBody {
				kvCap += 2
			}
			kv := make([]any, 0, kvCap)
			kv = append(kv,
				LogKeyMethod, req.Method,
				LogKeyURL, req.URL.String(),
				LogKeyDuration, dur,
			)
			if err != nil {
				logger.Error(req.Context(), "request failed", append(kv, LogKeyError, err)...)
				return res, err
			}
			kv = append(kv, LogKeyStatus, resStatusCode(res))
			if enableRequestBody && req.Body != nil && req.Body != http.NoBody {
				if req.GetBody != nil {
					body, _ := req.GetBody()
					var buf bytes.Buffer
					_, _ = io.Copy(&buf, io.LimitReader(body, maxLogBodyBytes))
					_ = body.Close()
					kv = append(kv, LogKeyRequestBody, buf.String())
				}
			}
			if enableResponseBody && res != nil && res.Body != nil && res.Body != http.NoBody {
				b1, b2, dupErr := duplicateBody(res.Body)
				if dupErr == nil {
					var buf bytes.Buffer
					_, _ = io.Copy(&buf, io.LimitReader(b1, maxLogBodyBytes))
					kv = append(kv, LogKeyResponseBody, buf.String())
					res.Body = b2
				}
			}
			logger.Info(req.Context(), "request", kv...)
			return res, err
		}
	}
}

// DebuggingLogger defines the interface for a debugging logger.
//
// Deprecated: Use [LeveledLogger] and [WithLeveledLogger] instead.
type DebuggingLogger interface {
	Log(a ...interface{}) error
	Logf(format string, a ...interface{}) error
}

// TextLogger prints the log message in plain text.
//
// Deprecated: Use [LeveledLogger] and [WithLeveledLogger] instead.
type TextLogger struct {
	Output             io.Writer
	EnableRequestBody  bool
	EnableResponseBody bool
}

// ColorLogger prints the log message in a terminal-optimized plain text.
//
// Deprecated: Use [LeveledLogger] and [WithLeveledLogger] instead.
type ColorLogger struct {
	Output             io.Writer
	EnableRequestBody  bool
	EnableResponseBody bool
}

// CurlLogger prints the log message as a runnable curl command.
//
// Deprecated: Use [LeveledLogger] and [WithLeveledLogger] instead.
type CurlLogger struct {
	Output             io.Writer
	EnableRequestBody  bool
	EnableResponseBody bool
}

// JSONLogger prints the log message as JSON.
//
// Deprecated: Use [LeveledLogger] and [WithLeveledLogger] instead.
type JSONLogger struct {
	Output             io.Writer
	EnableRequestBody  bool
	EnableResponseBody bool
}

// debuggingLogger prints debug messages as plain text.
type debuggingLogger struct {
	Output io.Writer
}

// LogRoundTrip prints the information about request and response.
func (l *TextLogger) LogRoundTrip(req *http.Request, res *http.Response, err error, start time.Time, dur time.Duration) error {
	_, _ = fmt.Fprintf(l.Output, "%s %s %s [status:%d request:%s]\n",
		start.Format(time.RFC3339),
		req.Method,
		req.URL.String(),
		resStatusCode(res),
		dur.Truncate(time.Millisecond),
	)
	if l.RequestBodyEnabled() && req != nil && req.Body != nil && req.Body != http.NoBody {
		var buf bytes.Buffer
		if req.GetBody != nil {
			b, _ := req.GetBody()
			_, _ = buf.ReadFrom(b)
		} else {
			_, _ = buf.ReadFrom(req.Body)
		}
		logBodyAsText(l.Output, &buf, ">")
	}
	if l.ResponseBodyEnabled() && res != nil && res.Body != nil && res.Body != http.NoBody {
		defer func() { _ = res.Body.Close() }()
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(res.Body)
		logBodyAsText(l.Output, &buf, "<")
	}
	if err != nil {
		_, _ = fmt.Fprintf(l.Output, "! ERROR: %v\n", err)
	}
	return nil
}

// RequestBodyEnabled returns true when the request body should be logged.
func (l *TextLogger) RequestBodyEnabled() bool { return l.EnableRequestBody }

// ResponseBodyEnabled returns true when the response body should be logged.
func (l *TextLogger) ResponseBodyEnabled() bool { return l.EnableResponseBody }

// LogRoundTrip prints the information about request and response.
func (l *ColorLogger) LogRoundTrip(req *http.Request, res *http.Response, err error, start time.Time, dur time.Duration) error {
	query, _ := url.QueryUnescape(req.URL.RawQuery)
	if query != "" {
		query = "?" + query
	}

	var (
		status string
		color  string
	)

	status = res.Status
	switch {
	case res.StatusCode > 0 && res.StatusCode < 300:
		color = "\x1b[32m"
	case res.StatusCode > 299 && res.StatusCode < 500:
		color = "\x1b[33m"
	case res.StatusCode > 499:
		color = "\x1b[31m"
	default:
		status = "ERROR"
		color = "\x1b[31;4m"
	}

	_, _ = fmt.Fprintf(l.Output, "%6s \x1b[1;4m%s://%s%s\x1b[0m%s %s%s\x1b[0m \x1b[2m%s\x1b[0m\n",
		req.Method,
		req.URL.Scheme,
		req.URL.Host,
		req.URL.Path,
		query,
		color,
		status,
		dur.Truncate(time.Millisecond),
	)

	if l.RequestBodyEnabled() && req != nil && req.Body != nil && req.Body != http.NoBody {
		var buf bytes.Buffer
		if req.GetBody != nil {
			b, _ := req.GetBody()
			_, _ = buf.ReadFrom(b)
		} else {
			_, _ = buf.ReadFrom(req.Body)
		}
		_, _ = fmt.Fprint(l.Output, "\x1b[2m")
		logBodyAsText(l.Output, &buf, "       »")
		_, _ = fmt.Fprint(l.Output, "\x1b[0m")
	}

	if l.ResponseBodyEnabled() && res != nil && res.Body != nil && res.Body != http.NoBody {
		defer func() { _ = res.Body.Close() }()
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(res.Body)
		_, _ = fmt.Fprint(l.Output, "\x1b[2m")
		logBodyAsText(l.Output, &buf, "       «")
		_, _ = fmt.Fprint(l.Output, "\x1b[0m")
	}

	if err != nil {
		_, _ = fmt.Fprintf(l.Output, "\x1b[31;1m» ERROR \x1b[31m%v\x1b[0m\n", err)
	}

	if l.RequestBodyEnabled() || l.ResponseBodyEnabled() {
		_, _ = fmt.Fprintf(l.Output, "\x1b[2m%s\x1b[0m\n", strings.Repeat("─", 80))
	}
	return nil
}

// RequestBodyEnabled returns true when the request body should be logged.
func (l *ColorLogger) RequestBodyEnabled() bool { return l.EnableRequestBody }

// ResponseBodyEnabled returns true when the response body should be logged.
func (l *ColorLogger) ResponseBodyEnabled() bool { return l.EnableResponseBody }

// LogRoundTrip prints the information about request and response.
func (l *CurlLogger) LogRoundTrip(req *http.Request, res *http.Response, err error, start time.Time, dur time.Duration) error {
	var b bytes.Buffer

	var query string
	qvalues := url.Values{}
	for k, v := range req.URL.Query() {
		if k == "pretty" {
			continue
		}
		for _, qv := range v {
			qvalues.Add(k, qv)
		}
	}
	if len(qvalues) > 0 {
		query = qvalues.Encode()
	}

	b.WriteString(`curl`)
	if req.Method == "HEAD" {
		b.WriteString(" --head")
	} else {
		_, _ = fmt.Fprintf(&b, " -X %s", req.Method)
	}

	if len(req.Header) > 0 {
		for k, vv := range req.Header {
			if k == "Authorization" || k == "User-Agent" {
				continue
			}
			v := strings.Join(vv, ",")
			fmt.Fprintf(&b, " -H '%s: %s'", k, v)
		}
	}

	b.WriteString(" '")
	b.WriteString(req.URL.Scheme) //nolint:staticcheck
	b.WriteString("://")
	b.WriteString(req.URL.Host) //nolint:staticcheck
	b.WriteString(req.URL.Path) //nolint:staticcheck
	b.WriteString("?pretty")
	if query != "" {
		_, _ = fmt.Fprintf(&b, "&%s", query)
	}
	b.WriteString("'")

	if req.Body != nil && req.Body != http.NoBody {
		var buf bytes.Buffer
		if req.GetBody != nil {
			b, _ := req.GetBody()
			_, _ = buf.ReadFrom(b)
		} else {
			_, _ = buf.ReadFrom(req.Body)
		}

		b.Grow(buf.Len())
		b.WriteString(" -d \\\n'")
		_ = json.Indent(&b, buf.Bytes(), "", " ")
		b.WriteString("'")
	}

	b.WriteRune('\n')

	var status = res.Status

	_, _ = fmt.Fprintf(&b, "# => %s [%s] %s\n", start.UTC().Format(time.RFC3339), status, dur.Truncate(time.Millisecond))
	if l.ResponseBodyEnabled() && res.Body != nil && res.Body != http.NoBody {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(res.Body)

		b.Grow(buf.Len())
		b.WriteString("# ")
		_ = json.Indent(&b, buf.Bytes(), "# ", " ")
	}

	b.WriteString("\n")
	if l.ResponseBodyEnabled() && res.Body != nil && res.Body != http.NoBody {
		b.WriteString("\n")
	}

	_, _ = b.WriteTo(l.Output)

	return nil
}

// RequestBodyEnabled returns true when the request body should be logged.
func (l *CurlLogger) RequestBodyEnabled() bool { return l.EnableRequestBody }

// ResponseBodyEnabled returns true when the response body should be logged.
func (l *CurlLogger) ResponseBodyEnabled() bool { return l.EnableResponseBody }

// LogRoundTrip prints the information about request and response.
func (l *JSONLogger) LogRoundTrip(req *http.Request, res *http.Response, err error, start time.Time, dur time.Duration) error {
	// https://github.com/elastic/ecs/blob/master/schemas/http.yml
	//
	// TODO(karmi): Research performance optimization of using sync.Pool

	bsize := 200
	var b = bytes.NewBuffer(make([]byte, 0, bsize))
	var v = make([]byte, 0, bsize)

	appendTime := func(t time.Time) {
		v = v[:0]
		v = t.AppendFormat(v, time.RFC3339)
		b.Write(v)
	}

	appendQuote := func(s string) {
		v = v[:0]
		v = strconv.AppendQuote(v, s)
		b.Write(v)
	}

	appendInt := func(i int64) {
		v = v[:0]
		v = strconv.AppendInt(v, i, 10)
		b.Write(v)
	}

	port := req.URL.Port()

	b.WriteRune('{')
	// -- Timestamp
	b.WriteString(`"@timestamp":"`)
	appendTime(start.UTC())
	b.WriteRune('"')
	// -- Event
	b.WriteString(`,"event":{`)
	b.WriteString(`"duration":`)
	appendInt(dur.Nanoseconds())
	b.WriteRune('}')
	// -- URL
	b.WriteString(`,"url":{`)
	b.WriteString(`"scheme":`)
	appendQuote(req.URL.Scheme)
	b.WriteString(`,"domain":`)
	appendQuote(req.URL.Hostname())
	if port != "" {
		b.WriteString(`,"port":`)
		b.WriteString(port)
	}
	b.WriteString(`,"path":`)
	appendQuote(req.URL.Path)
	b.WriteString(`,"query":`)
	appendQuote(req.URL.RawQuery)
	b.WriteRune('}') // Close "url"
	// -- HTTP
	b.WriteString(`,"http":`)
	// ---- Request
	b.WriteString(`{"request":{`)
	b.WriteString(`"method":`)
	appendQuote(req.Method)
	if l.RequestBodyEnabled() && req != nil && req.Body != nil && req.Body != http.NoBody {
		var buf bytes.Buffer
		if req.GetBody != nil {
			b, _ := req.GetBody()
			_, _ = buf.ReadFrom(b)
		} else {
			_, _ = buf.ReadFrom(req.Body)
		}

		b.Grow(buf.Len() + 8)
		b.WriteString(`,"body":`)
		appendQuote(buf.String())
	}
	b.WriteRune('}') // Close "http.request"
	// ---- Response
	b.WriteString(`,"response":{`)
	b.WriteString(`"status_code":`)
	appendInt(int64(resStatusCode(res)))
	if l.ResponseBodyEnabled() && res != nil && res.Body != nil && res.Body != http.NoBody {
		defer func() { _ = res.Body.Close() }()
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(res.Body)

		b.Grow(buf.Len() + 8)
		b.WriteString(`,"body":`)
		appendQuote(buf.String())
	}
	b.WriteRune('}') // Close "http.response"
	b.WriteRune('}') // Close "http"
	// -- Error
	if err != nil {
		b.WriteString(`,"error":{"message":`)
		appendQuote(err.Error())
		b.WriteRune('}') // Close "error"
	}
	b.WriteRune('}')
	b.WriteRune('\n')
	_, _ = b.WriteTo(l.Output)

	return nil
}

// RequestBodyEnabled returns true when the request body should be logged.
func (l *JSONLogger) RequestBodyEnabled() bool { return l.EnableRequestBody }

// ResponseBodyEnabled returns true when the response body should be logged.
func (l *JSONLogger) ResponseBodyEnabled() bool { return l.EnableResponseBody }

// Log prints the arguments to output in default format.
func (l *debuggingLogger) Log(a ...interface{}) error {
	_, err := fmt.Fprint(l.Output, a...)
	return err
}

// Logf prints formats the arguments and prints them to output.
func (l *debuggingLogger) Logf(format string, a ...interface{}) error {
	_, err := fmt.Fprintf(l.Output, format, a...)
	return err
}

func logBodyAsText(dst io.Writer, body io.Reader, prefix string) {
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		s := scanner.Text()
		if s != "" {
			_, _ = fmt.Fprintf(dst, "%s %s\n", prefix, s)
		}
	}
}

const maxLogBodyBytes = 10 * 1024 // 10 KB cap for logged bodies

func duplicateBody(body io.ReadCloser) (io.ReadCloser, io.ReadCloser, error) {
	var (
		b1 bytes.Buffer
		b2 bytes.Buffer
		tr = io.TeeReader(body, &b2)
	)
	_, err := b1.ReadFrom(tr)
	if err != nil {
		return io.NopCloser(io.MultiReader(&b1, errorReader{err: err})), io.NopCloser(io.MultiReader(&b2, errorReader{err: err})), err
	}
	defer func() { _ = body.Close() }()

	return io.NopCloser(&b1), io.NopCloser(&b2), nil
}

func resStatusCode(res *http.Response) int {
	if res == nil {
		return -1
	}
	return res.StatusCode
}

type errorReader struct{ err error }

func (r errorReader) Read(p []byte) (int, error) { return 0, r.err }
