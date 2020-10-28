// Package gateway provides a gRPC-gateway component with included observability
// middleware.
package gateway

import (
	"context"
	"fmt"
	nethttp "net/http"

	"github.com/beatlabs/patron/component/http"
	"github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/log"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
)

// Component of a gRPC service.
type Component struct {
	port       int
	mux        *runtime.ServeMux
	middleware []http.MiddlewareFunc
}

// ServeMux returns the grpc-gateway request multiplexer.
func (c *Component) ServeMux() *runtime.ServeMux {
	return c.mux
}

// Run the gRPC-gateway service.
func (c *Component) Run(ctx context.Context) error {

	srv := nethttp.Server{
		Addr:    fmt.Sprintf(":%d", c.port),
		Handler: http.MiddlewareChain(c.mux, c.middleware...),
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		<-ctx.Done()
		srv.Shutdown(ctx)
	}()

	log.Infof("gRPC gateway component listening on %d", c.port)

	err := srv.ListenAndServe()
	if err != nil && err != nethttp.ErrServerClosed {
		return fmt.Errorf("failed to listen: %w", err)
	}

	return nil
}

// Builder pattern for our gRPC service.
type Builder struct {
	port       int
	middleware []http.MiddlewareFunc
	errors     []error
}

// New builder.
func New(port int) *Builder {
	b := &Builder{}
	if port <= 0 || port > 65535 {
		b.errors = append(b.errors, fmt.Errorf("port is invalid: %d", port))
		return b
	}
	b.port = port
	return b
}

func (b *Builder) WithMiddleware(mm ...http.MiddlewareFunc) *Builder {
	b.middleware = append(b.middleware, mm...)
	return b
}

// Create the gRPC component.
func (b *Builder) Create() (*Component, error) {
	if len(b.errors) != 0 {
		return nil, errors.Aggregate(b.errors...)
	}

	return &Component{
		port:       b.port,
		mux:        runtime.NewServeMux(),
		middleware: b.middleware,
	}, nil
}
