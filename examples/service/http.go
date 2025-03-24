package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/beatlabs/patron"
	patronhttp "github.com/beatlabs/patron/component/http"
	"github.com/beatlabs/patron/component/http/router"
	"github.com/beatlabs/patron/observability/log"
)

func createHTTPRouter() (patron.Component, error) {
	handler := func(rw http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			msg := "failed to read body"
			http.Error(rw, msg, http.StatusBadRequest)
			log.FromContext(req.Context()).Error(msg)
			return
		}

		log.FromContext(req.Context()).Info("HTTP request received", "body", string(body))
		rw.WriteHeader(http.StatusOK)
	}

	var routes patronhttp.Routes
	routes.Append(patronhttp.NewRoute("GET /", handler))
	rr, err := routes.Result()
	if err != nil {
		return nil, fmt.Errorf("failed to create routes: %w", err)
	}

	rt, err := router.New(router.WithRoutes(rr...))
	if err != nil {
		return nil, fmt.Errorf("failed to create http router: %w", err)
	}

	return patronhttp.New(rt)
}
