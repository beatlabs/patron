package v2

import (
	"net/http"
	"net/http/pprof"

	patronhttp "github.com/beatlabs/patron/component/http"
	"github.com/beatlabs/patron/log"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func registerMetricsHandler(mux *mux.Router) {
	mux.HandleFunc(patronhttp.MetricsPath, promhttp.Handler().ServeHTTP).Methods(http.MethodGet)
	log.Debug("registered HTTP metrics handler")
}

func registerPprofHandlers(mux *mux.Router) {
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	log.Debug("registered HTTP pprof handlers")
}
