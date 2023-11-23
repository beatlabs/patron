package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/beatlabs/patron"
	patronhttp "github.com/beatlabs/patron/component/http"
	"github.com/beatlabs/patron/component/http/router/httprouter"
	"github.com/beatlabs/patron/log"
)

func createHttpRouter() (patron.Component, error) {
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
	routes.Append(patronhttp.NewGetRoute("/", handler))
	rr, err := routes.Result()
	if err != nil {
		return nil, fmt.Errorf("failed to create routes: %w", err)
	}

	router, err := httprouter.New(httprouter.WithRoutes(rr...))
	if err != nil {
		return nil, fmt.Errorf("failed to create http router: %w", err)
	}

	return patronhttp.New(router)
}
