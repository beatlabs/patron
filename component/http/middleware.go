package http

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/beatlabs/patron/component/http/auth"
	"github.com/beatlabs/patron/component/http/cache"
	"github.com/beatlabs/patron/correlation"
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
					log.Errorf("recovering from an error %v", err)
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
func NewLoggingTracingMiddleware(path string) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			corID := getOrSetCorrelationID(r.Header)
			sp, r := span(path, corID, r)
			lw := newResponseWriter(w)
			next.ServeHTTP(lw, r)
			finishSpan(sp, lw.Status(), lw.payload)
			logRequestResponse(corID, lw, r)
		})
	}
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
