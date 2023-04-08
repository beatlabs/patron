package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/beatlabs/patron"
	v2 "github.com/beatlabs/patron/component/http/v2"
	"github.com/beatlabs/patron/component/http/v2/router/httprouter"
	"golang.org/x/exp/slog"
)

func createHttpRouter() (patron.Component, error) {
	handler := func(rw http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			msg := "failed to read body"
			http.Error(rw, msg, http.StatusBadRequest)
			slog.ErrorCtx(req.Context(), msg)
			return
		}

		slog.InfoCtx(req.Context(), "HTTP request received: %s", string(body))
		rw.WriteHeader(http.StatusOK)
	}

	var routes v2.Routes
	routes.Append(v2.NewGetRoute("/", handler))
	rr, err := routes.Result()
	if err != nil {
		return nil, fmt.Errorf("failed to create routes: %w", err)
	}

	router, err := httprouter.New(httprouter.WithRoutes(rr...))
	if err != nil {
		return nil, fmt.Errorf("failed to create http router: %w", err)
	}

	return v2.New(router)
}
