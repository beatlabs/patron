package http

import (
	"compress/flate"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/time/rate"

	"github.com/beatlabs/patron/component/http/auth"
	"github.com/beatlabs/patron/component/http/cache"
	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/trace"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	tracinglog "github.com/opentracing/opentracing-go/log"
)

const (
	serverComponent = "http-server"
	fieldNameError  = "error"

	// compression algorithms
	gzipHeader     = "gzip"
	deflateHeader  = "deflate"
	identityHeader = "identity"
	anythingHeader = "*"
)

type responseWriter struct {
	status              int
	statusHeaderWritten bool
	payload             []byte
	writer              http.ResponseWriter
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{status: -1, statusHeaderWritten: false, writer: w}
}

// Status returns the http response status.
func (w *responseWriter) Status() int {
	return w.status
}

// Header returns the Header.
func (w *responseWriter) Header() http.Header {
	return w.writer.Header()
}

// Write to the internal responseWriter and sets the status if not set already.
func (w *responseWriter) Write(d []byte) (int, error) {
	value, err := w.writer.Write(d)
	if err != nil {
		return value, err
	}

	w.payload = d

	if !w.statusHeaderWritten {
		w.status = http.StatusOK
		w.statusHeaderWritten = true
	}

	return value, err
}

// WriteHeader writes the internal Header and saves the status for retrieval.
func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.writer.WriteHeader(code)
	w.statusHeaderWritten = true
}

// MiddlewareFunc type declaration of middleware func.
type MiddlewareFunc func(next http.Handler) http.Handler

// NewRecoveryMiddleware creates a MiddlewareFunc that ensures recovery and no panic.
func NewRecoveryMiddleware() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if r := recover(); r != nil {
					var err error
					switch x := r.(type) {
					case string:
						err = errors.New(x)
					case error:
						err = x
					default:
						err = errors.New("unknown panic")
					}
					_ = err
					log.Errorf("recovering from an error: %v: %s", err, string(debug.Stack()))
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// NewAuthMiddleware creates a MiddlewareFunc that implements authentication using an Authenticator.
func NewAuthMiddleware(auth auth.Authenticator) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authenticated, err := auth.Authenticate(r)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			if !authenticated {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// NewLoggingTracingMiddleware creates a MiddlewareFunc that continues a tracing span and finishes it.
// It also logs the HTTP request on debug logging level
func NewLoggingTracingMiddleware(path string, statusCodeLogger statusCodeLoggerHandler) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			corID := getOrSetCorrelationID(r.Header)
			sp, r := span(path, corID, r)
			lw := newResponseWriter(w)
			next.ServeHTTP(lw, r)
			finishSpan(sp, lw.Status(), lw.payload)
			logRequestResponse(corID, lw, r)
			statusCodeErrorLogging(r.Context(), statusCodeLogger, lw.Status(), lw.payload, path)
		})
	}
}

// NewRateLimitingMiddleware creates a MiddlewareFunc that adds a rate limit to a route.
func NewRateLimitingMiddleware(limiter *rate.Limiter) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow() {
				log.Debug("Limiting requests...")
				http.Error(w, "Requests greater than limit", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ignore checks if the given url ignored from compression or not.
func ignore(ignoreRoutes []string, url string) bool {
	for _, iURL := range ignoreRoutes {
		if strings.HasPrefix(url, iURL) {
			return true
		}
	}

	return false
}

func parseAcceptEncoding(header string) (string, error) {
	if header == "" {
		return identityHeader, nil
	}

	if header == anythingHeader {
		return anythingHeader, nil
	}

	weighted := make(map[float64]string)

	algorithms := strings.Split(header, ",")
	for _, a := range algorithms {
		algAndWeight := strings.Split(a, ";")
		algorithm := strings.TrimSpace(algAndWeight[0])

		if notSupportedCompression(algorithm) {
			continue
		}

		if len(algAndWeight) != 2 {
			addWithWeight(weighted, 1.0, algorithm)
			continue
		}

		weight := parseWeight(algAndWeight[1])
		addWithWeight(weighted, weight, algorithm)
	}

	return selectByWeight(weighted)
}

func addWithWeight(mapWeighted map[float64]string, weight float64, algorithm string) {
	if _, ok := mapWeighted[weight]; !ok {
		mapWeighted[weight] = algorithm
	}
}

func notSupportedCompression(algorithm string) bool {
	return gzipHeader != algorithm && deflateHeader != algorithm && anythingHeader != algorithm && identityHeader != algorithm
}

// When not present, the default value is 1 according to https://developer.mozilla.org/en-US/docs/Glossary/Quality_values
// q not present or can’t be parsed -> 1.0
// q is < 0 -> 0.0
// q is > 1 -> 1.0
func parseWeight(qStr string) float64 {
	qAndWeight := strings.Split(qStr, "=")
	if len(qAndWeight) != 2 {
		return 1.0
	}

	parsedWeight, err := strconv.ParseFloat(qAndWeight[1], 32)
	if err != nil {
		return 1.0
	}

	return math.Min(1.0, math.Max(0.0, parsedWeight))
}

func selectByWeight(weighted map[float64]string) (string, error) {
	if len(weighted) == 0 {
		return "", fmt.Errorf("no valid compression encoding accepted by client")
	}

	keys := make([]float64, 0, len(weighted))
	for k := range weighted {
		keys = append(keys, k)
	}
	sort.Float64s(keys)
	return weighted[keys[len(keys)-1]], nil
}

// NewCompressionMiddleware initializes a compression middleware.
// As per Section 3.5 of the HTTP/1.1 RFC, GZIP and Deflate compression methods are supported.
// https://tools.ietf.org/html/rfc2616#section-14.3
func NewCompressionMiddleware(deflateLevel int, ignoreRoutes ...string) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ignore(ignoreRoutes, r.URL.String()) {
				next.ServeHTTP(w, r)
				return
			}

			hdr := r.Header.Get(encoding.AcceptEncodingHeader)
			selectedEncoding, err := parseAcceptEncoding(hdr)
			if err != nil {
				log.Debugf("encoding %q is not supported in compression middleware, "+
					"and client doesn't accept anything else", hdr)
				http.Error(w, http.StatusText(http.StatusNotAcceptable), http.StatusNotAcceptable)
				return
			}

			dw := &dynamicCompressionResponseWriter{w, selectedEncoding, nil, 0, deflateLevel}

			defer func(c io.Closer) {
				err := c.Close()
				if err != nil {
					msgErr := fmt.Sprintf("error in deferred call to Close() method on %v compression middleware : %v", hdr, err.Error())
					if isErrConnectionReset(err) {
						log.Info(msgErr)
					} else {
						log.Error(msgErr)
					}
				}
			}(dw)

			next.ServeHTTP(dw, r)
		})
	}
}

// isErrConnectionReset detects if an error has happened due to a connection reset, broken pipe or similar.
// Implementation is copied from AWS SDK, package request. We assume that it is a complete genuine implementation.
func isErrConnectionReset(err error) bool {
	errMsg := err.Error()

	// See the explanation here: https://github.com/aws/aws-sdk-go/issues/2525#issuecomment-519263830
	// It is a little bit vague, but it seems they mean that this specific error happens when we stopped reading for some reason
	// even though there was something to read. This might have happened due to a wrong length header for example.
	// So it might've been our error, not an error of remote server when it closes connection unexpectedly.
	if strings.Contains(errMsg, "read: connection reset") {
		return false
	}

	if strings.Contains(errMsg, "connection reset") ||
		strings.Contains(errMsg, "broken pipe") {
		return true
	}

	return false
}

// bodyAllowedForStatus reports whether a given response status code
// permits a body. See RFC 7230, section 3.3.
// https://github.com/golang/go/blob/6551763a60ce25d171feaa69089a7f1ca60f43b6/src/net/http/transfer.go#L452-L464
func bodyAllowedForStatus(status int) bool {
	switch {
	case status >= 100 && status <= 199:
		return false
	case status == 204:
		return false
	case status == 304:
		return false
	}
	return true
}

// dynamicCompressionResponseWriter uses gzip/deflate compression on a response body only once the status code is known
// so that http.ErrBodyNotAllowed can be avoided in the case of 204/304 response status.
type dynamicCompressionResponseWriter struct {
	http.ResponseWriter
	Encoding     string
	writer       io.Writer
	statusCode   int
	deflateLevel int
}

func (w *dynamicCompressionResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode

	if w.writer == nil {
		if !bodyAllowedForStatus(w.statusCode) {
			// no body allowed so can't compress
			// don't try to write compression header (1f 8b) to body to avoid http.ErrBodyNotAllowed
			w.writer = w.ResponseWriter
			return
		}

		switch w.Encoding {
		case gzipHeader:
			w.writer = gzip.NewWriter(w.ResponseWriter)
			w.ResponseWriter.Header().Set(encoding.ContentEncodingHeader, gzipHeader)
		case deflateHeader:
			var err error
			w.writer, err = flate.NewWriter(w.ResponseWriter, w.deflateLevel)
			if err != nil {
				w.writer = w.ResponseWriter
			} else {
				w.ResponseWriter.Header().Set(encoding.ContentEncodingHeader, deflateHeader)
			}
		case identityHeader, "":
			w.ResponseWriter.Header().Set(encoding.ContentEncodingHeader, identityHeader)
			fallthrough
		// `*`, `identity` and others must fall through here to be served without compression
		default:
			w.writer = w.ResponseWriter
		}
	}

	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *dynamicCompressionResponseWriter) Write(data []byte) (int, error) {
	if w.statusCode == 0 {
		w.WriteHeader(http.StatusOK)
	}

	return w.writer.Write(data)
}

func (w *dynamicCompressionResponseWriter) Close() error {
	if rc, ok := w.writer.(io.Closer); ok {
		return rc.Close()
	}
	return nil
}

// NewCachingMiddleware creates a cache layer as a middleware
// when used as part of a middleware chain any middleware later in the chain,
// will not be executed, but the headers it appends will be part of the cache
func NewCachingMiddleware(rc *cache.RouteCache) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}
			err := cache.Handler(w, r, rc, next)
			if err != nil {
				log.Errorf("error encountered in the caching middleware: %v", err)
				return
			}
		})
	}
}

// MiddlewareChain chains middlewares to a handler func.
func MiddlewareChain(f http.Handler, mm ...MiddlewareFunc) http.Handler {
	for i := len(mm) - 1; i >= 0; i-- {
		f = mm[i](f)
	}
	return f
}

func logRequestResponse(corID string, w *responseWriter, r *http.Request) {
	if !log.Enabled(log.DebugLevel) {
		return
	}

	remoteAddr := r.RemoteAddr
	if i := strings.LastIndex(remoteAddr, ":"); i != -1 {
		remoteAddr = remoteAddr[:i]
	}

	info := map[string]interface{}{
		"request": map[string]interface{}{
			"remote-address": remoteAddr,
			"method":         r.Method,
			"url":            r.URL,
			"proto":          r.Proto,
			"status":         w.Status(),
			"referer":        r.Referer(),
			"user-agent":     r.UserAgent(),
			correlation.ID:   corID,
		},
	}
	log.Sub(info).Debug()
}

func statusCodeErrorLogging(ctx context.Context, statusCodeLogger statusCodeLoggerHandler, statusCode int, payload []byte, path string) {
	if !log.Enabled(log.ErrorLevel) {
		return
	}

	if statusCodeLogger.shouldLog(statusCode) {
		log.FromContext(ctx).Errorf("%s %d error: %v", path, statusCode, string(payload))
	}
}

func getOrSetCorrelationID(h http.Header) string {
	cor, ok := h[correlation.HeaderID]
	if !ok {
		corID := uuid.New().String()
		h.Set(correlation.HeaderID, corID)
		return corID
	}
	if len(cor) == 0 {
		corID := uuid.New().String()
		h.Set(correlation.HeaderID, corID)
		return corID
	}
	if cor[0] == "" {
		corID := uuid.New().String()
		h.Set(correlation.HeaderID, corID)
		return corID
	}
	return cor[0]
}

func span(path, corID string, r *http.Request) (opentracing.Span, *http.Request) {
	ctx, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))
	if err != nil && err != opentracing.ErrSpanContextNotFound {
		log.Errorf("failed to extract HTTP span: %v", err)
	}

	strippedPath, err := stripQueryString(path)
	if err != nil {
		log.Warnf("unable to strip query string %q: %v", path, err)
		strippedPath = path
	}

	sp := opentracing.StartSpan(opName(r.Method, strippedPath), ext.RPCServerOption(ctx))
	ext.HTTPMethod.Set(sp, r.Method)
	ext.HTTPUrl.Set(sp, r.URL.String())
	ext.Component.Set(sp, serverComponent)
	sp.SetTag(trace.VersionTag, trace.Version)
	sp.SetTag(correlation.ID, corID)
	return sp, r.WithContext(opentracing.ContextWithSpan(r.Context(), sp))
}

// stripQueryString returns a path without the query string
func stripQueryString(path string) (string, error) {
	u, err := url.Parse(path)
	if err != nil {
		return "", err
	}

	if len(u.RawQuery) == 0 {
		return path, nil
	}

	return path[:len(path)-len(u.RawQuery)-1], nil
}

func finishSpan(sp opentracing.Span, code int, payload []byte) {
	ext.HTTPStatusCode.Set(sp, uint16(code))
	isError := code >= http.StatusInternalServerError
	if isError && len(payload) != 0 {
		sp.LogFields(tracinglog.String(fieldNameError, string(payload)))
	}
	ext.Error.Set(sp, isError)
	sp.Finish()
}

func opName(method, path string) string {
	return method + " " + path
}
