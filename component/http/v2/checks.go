package v2

import (
	"net/http"

	patronhttp "github.com/beatlabs/patron/component/http"
	"github.com/beatlabs/patron/log"
	"github.com/gorilla/mux"
)

var defaultAliveCheck = func() patronhttp.AliveStatus { return patronhttp.Alive }

func registerAliveCheckHandler(mux *mux.Router, acf patronhttp.AliveCheckFunc) {
	mux.HandleFunc(patronhttp.AlivePath, func(rw http.ResponseWriter, r *http.Request) {
		switch acf() {
		case patronhttp.Alive:
			rw.WriteHeader(http.StatusOK)
		case patronhttp.Unresponsive:
			rw.WriteHeader(http.StatusServiceUnavailable)
		default:
			rw.WriteHeader(http.StatusOK)
		}
	}).Methods(http.MethodGet)
	log.Debug("registered HTTP alive check handler")
}

var defaultReadyCheck = func() patronhttp.ReadyStatus { return patronhttp.Ready }

func registerReadyCheckHandler(mux *mux.Router, rcf patronhttp.ReadyCheckFunc) {
	mux.HandleFunc(patronhttp.ReadyPath, func(rw http.ResponseWriter, r *http.Request) {
		switch rcf() {
		case patronhttp.Ready:
			rw.WriteHeader(http.StatusOK)
		case patronhttp.NotReady:
			rw.WriteHeader(http.StatusServiceUnavailable)
		default:
			rw.WriteHeader(http.StatusOK)
		}
	}).Methods(http.MethodGet)
	log.Debug("registered HTTP ready check handler")
}
