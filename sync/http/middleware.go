package http

import (
	"net/http"

	"github.com/thebeatapp/patron/errors"
	"github.com/thebeatapp/patron/log"
	"github.com/thebeatapp/patron/sync/http/auth"
	"github.com/thebeatapp/patron/trace"
)

type responseWriter struct {
	status              int
	statusHeaderWritten bool
	writer              http.ResponseWriter
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{status: -1, statusHeaderWritten: false, writer: w}
}

// Status returns the http response status.
func (w *responseWriter) Status() int {
	return w.status
}

// Header returns the header.
func (w *responseWriter) Header() http.Header {
	return w.writer.Header()
}

// Write to the internal ResponseWriter and sets the status if not set already.
func (w *responseWriter) Write(d []byte) (int, error) {

	value, err := w.writer.Write(d)
	if err != nil {
		return value, err
	}

	if !w.statusHeaderWritten {
		w.status = http.StatusOK
		w.statusHeaderWritten = true
	}

	return value, err
}

// WriteHeader writes the internal header and saves the status for retrieval.
func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.writer.WriteHeader(code)
	w.statusHeaderWritten = true
}

// MiddlewareChain chains middlewares to a handler func.
func MiddlewareChain(f http.HandlerFunc, mm ...MiddlewareFunc) http.HandlerFunc {
	for i := len(mm) - 1; i >= 0; i-- {
		f = mm[i](f)
	}
	return f
}

// MiddlewareDefaults chains all default middlewares to handler function and returns the handler func.
func MiddlewareDefaults(trace bool, auth auth.Authenticator, path string, next http.HandlerFunc) http.HandlerFunc {
	next = recoveryMiddleware(next)
	if auth != nil {
		next = authMiddleware(auth, next)
	}
	if trace {
		next = tracingMiddleware(path, next)
	}
	return next
}

// MiddlewareFunc type declaration of middleware func.
type MiddlewareFunc func(next http.HandlerFunc) http.HandlerFunc

func tracingMiddleware(path string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sp, r := trace.HTTPSpan(path, r)
		lw := newResponseWriter(w)
		next(lw, r)
		trace.FinishHTTPSpan(sp, lw.Status())
	}
}

func recoveryMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		next(w, r)
	}
}

func authMiddleware(auth auth.Authenticator, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authenticated, err := auth.Authenticate(r)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if !authenticated {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}
