package middleware

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"

	"github.com/beatlabs/patron/component/http/auth"
	"github.com/beatlabs/patron/component/http/cache"
	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/observability/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"golang.org/x/time/rate"
)

const (
	gzipHeader       = "gzip"
	deflateHeader    = "deflate"
	identityHeader   = "identity"
	anythingHeader   = "*"
	appVersionHeader = "X-App-Version"
	appNameHeader    = "X-App-Name"
)

type responseWriter struct {
	status              int
	statusHeaderWritten bool
	capturePayload      bool
	responsePayload     bytes.Buffer
	writer              http.ResponseWriter
}

func newResponseWriter(w http.ResponseWriter, capturePayload bool) *responseWriter {
	return &responseWriter{status: -1, statusHeaderWritten: false, writer: w, capturePayload: capturePayload}
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
	if !w.statusHeaderWritten {
		w.status = http.StatusOK
		w.statusHeaderWritten = true
	}

	value, err := w.writer.Write(d)
	if err != nil {
		return value, err
	}

	if w.capturePayload {
		w.responsePayload.Write(d)
	}

	return value, err
}

// WriteHeader writes the internal Header and saves the status for retrieval.
func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.writer.WriteHeader(code)
	w.statusHeaderWritten = true
}

// Func type declaration of middleware func.
type Func func(next http.Handler) http.Handler

// NewRecovery creates a Func that ensures recovery and no panic.
func NewRecovery() Func {
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
					slog.Error("recovering from a failure", log.ErrorAttr(err), slog.String("stack", string(debug.Stack())))
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// NewAppNameVersion adds an app method header and an app path header to all responses. Existing values of these headers are overwritten.
func NewAppNameVersion(name, version string) (Func, error) {
	if name == "" {
		return nil, errors.New("app name cannot be empty")
	}

	if version == "" {
		return nil, errors.New("app version cannot be empty")
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(appVersionHeader, version)
			w.Header().Set(appNameHeader, name)
			next.ServeHTTP(w, r)
		})
	}, nil
}

// NewAuth creates a Func that implements authentication using an Authenticator.
func NewAuth(auth auth.Authenticator) Func {
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

// NewLoggingTracing creates a Func that continues a tracing span and finishes it.
func NewLoggingTracing(path string, statusCodeLogger StatusCodeLoggerHandler) (Func, error) {
	if path == "" {
		return nil, errors.New("path cannot be empty")
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			corID := getOrSetCorrelationID(r.Header)
			lw := newResponseWriter(w, true)

			otelhttp.NewMiddleware(path)(next).ServeHTTP(lw, r)
			logRequestResponse(corID, lw, r)
			if log.Enabled(slog.LevelError) && statusCodeLogger.shouldLog(lw.status) {
				log.FromContext(r.Context()).Error("failed route execution", slog.String("path", path),
					slog.Int("status", lw.status), slog.String("payload", lw.responsePayload.String()))
			}
		})
	}, nil
}

// NewInjectObservability injects a correlation ID unless one is already present.
func NewInjectObservability() Func {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			corID := getOrSetCorrelationID(r.Header)
			ctx := correlation.ContextWithID(r.Context(), corID)
			ctx = log.WithContext(ctx, slog.With(slog.String(correlation.ID, corID)))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func getOrSetCorrelationID(h http.Header) string {
	cor, ok := h[correlation.HeaderID]
	if !ok {
		corID := correlation.New()
		h.Set(correlation.HeaderID, corID)
		return corID
	}
	if len(cor) == 0 {
		corID := correlation.New()
		h.Set(correlation.HeaderID, corID)
		return corID
	}
	if cor[0] == "" {
		corID := correlation.New()
		h.Set(correlation.HeaderID, corID)
		return corID
	}
	return cor[0]
}

// NewRateLimiting creates a Func that adds a rate limit to a route.
func NewRateLimiting(limiter *rate.Limiter) (Func, error) {
	if limiter == nil {
		return nil, errors.New("limiter cannot be nil")
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow() {
				slog.Debug("limiting requests...")
				http.Error(w, "Requests greater than limit", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}, nil
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
// q is > 1 -> 1.0.
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
		return "", errors.New("no valid compression encoding accepted by client")
	}

	keys := make([]float64, 0, len(weighted))
	for k := range weighted {
		keys = append(keys, k)
	}
	sort.Float64s(keys)
	return weighted[keys[len(keys)-1]], nil
}

// NewCompression initializes a compression middleware.
// As per Section 3.5 of the HTTP/1.1 RFC, GZIP and Deflate compression methods are supported.
// https://tools.ietf.org/html/rfc2616#section-14.3
func NewCompression(deflateLevel int, ignoreRoutes ...string) (Func, error) {
	if deflateLevel < -2 || deflateLevel > 9 { // https://www.rfc-editor.org/rfc/rfc1950#page-6
		return nil, fmt.Errorf("invalid compression level %d: want value in range [-2, 9]", deflateLevel)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ignore(ignoreRoutes, r.URL.String()) {
				next.ServeHTTP(w, r)
				return
			}

			hdr := r.Header.Get(encoding.AcceptEncodingHeader)
			selectedEncoding, err := parseAcceptEncoding(hdr)
			if err != nil {
				slog.Debug("encoding is not supported in compression middleware, "+
					"and client doesn't accept anything else", slog.String("header", hdr))
				http.Error(w, http.StatusText(http.StatusNotAcceptable), http.StatusNotAcceptable)
				return
			}

			dw := &dynamicCompressionResponseWriter{w, selectedEncoding, nil, 0, deflateLevel}

			defer func(c io.Closer) {
				err := c.Close()
				if err != nil {
					msgErr := "error in deferred call to Close() method on compression middleware"
					if isErrConnectionReset(err) {
						slog.Info(msgErr, slog.String("header", hdr), log.ErrorAttr(err))
					} else {
						slog.Error(msgErr, slog.String("header", hdr), log.ErrorAttr(err))
					}
				}
			}(dw)

			next.ServeHTTP(dw, r)
		})
	}, nil
}

// isErrConnectionReset detects if an error has happened due to a connection reset, broken pipe or similar.
// Implementation is copied from AWS SDK, package request. We assume that it is a complete genuine implementation.
func isErrConnectionReset(err error) bool {
	errMsg := err.Error()

	// See the explanation here: https://github.com/aws/aws-sdk-go/issues/2525#issuecomment-519263830
	// It is a little vague, but it seems they mean that this specific error happens when we stopped reading for some reason
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
			w.ResponseWriter.WriteHeader(statusCode)
			return
		}

		switch w.Encoding {
		case gzipHeader:
			w.writer = gzip.NewWriter(w.ResponseWriter)
			w.ResponseWriter.Header().Set(encoding.ContentEncodingHeader, gzipHeader)
			w.ResponseWriter.Header().Del(encoding.ContentLengthHeader)
		case deflateHeader:
			var err error
			w.writer, err = flate.NewWriter(w.ResponseWriter, w.deflateLevel)
			if err != nil {
				w.writer = w.ResponseWriter
			} else {
				w.ResponseWriter.Header().Set(encoding.ContentEncodingHeader, deflateHeader)
				w.ResponseWriter.Header().Del(encoding.ContentLengthHeader)
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

// NewCaching creates a cache layer as a middleware
// when used as part of a middleware chain any middleware later in the chain,
// will not be executed, but the headers it appends will be part of the cache.
func NewCaching(rc *cache.RouteCache) (Func, error) {
	if rc == nil {
		return nil, errors.New("route cache cannot be nil")
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}
			err := cache.Handler(w, r, rc, next)
			if err != nil {
				slog.Error("error encountered in the caching middleware", log.ErrorAttr(err))
				return
			}
		})
	}, nil
}

// Chain chains middlewares to a handler func.
func Chain(f http.Handler, mm ...Func) http.Handler {
	for i := len(mm) - 1; i >= 0; i-- {
		f = mm[i](f)
	}
	return f
}

func logRequestResponse(corID string, w *responseWriter, r *http.Request) {
	if !log.Enabled(slog.LevelDebug) {
		return
	}

	remoteAddr := r.RemoteAddr
	if i := strings.LastIndex(remoteAddr, ":"); i != -1 {
		remoteAddr = remoteAddr[:i]
	}

	attrs := []slog.Attr{
		slog.String(correlation.ID, corID),
		slog.String("method", r.Method),
		slog.String("url", r.URL.String()),
		slog.Int("status", w.Status()),
		slog.String("remote-address", remoteAddr),
		slog.String("proto", r.Proto),
	}

	log.FromContext(r.Context()).LogAttrs(r.Context(), slog.LevelDebug, "request log", attrs...)
}

// stripQueryString returns a path without the query string.
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
