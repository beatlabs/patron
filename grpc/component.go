package grpc

import (
	"context"
	"fmt"
	"net"

	"github.com/beatlabs/patron/log"

	"google.golang.org/grpc/metadata"

	"github.com/beatlabs/patron/correlation"
	"github.com/google/uuid"

	"github.com/beatlabs/patron/trace"

	"github.com/beatlabs/patron/errors"
	"google.golang.org/grpc"
)

const (
	componentName = "gRPC-server"
)

// Component of a gRPC service.
type Component struct {
	port string
	srv  *grpc.Server
}

// Run the gRPC service.
func (c *Component) Run(ctx context.Context) error {
	lis, err := net.Listen("tcp", c.port)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	go func() {
		for range ctx.Done() {
			c.srv.GracefulStop()
			break
		}
	}()

	return c.srv.Serve(lis)
}

type definition struct {
	description *grpc.ServiceDesc
	service     interface{}
}

// Builder pattern for our gRPC service.
type Builder struct {
	port          string
	serverOptions []grpc.ServerOption
	definitions   []definition
	errors        []error
}

// New builder.
func New(port string, description *grpc.ServiceDesc, service interface{}) *Builder {
	b := &Builder{}
	if port == "" {
		b.errors = append(b.errors, errors.New("port is empty"))
	}
	b.port = port
	b.appendDefinition(description, service)
	return b
}

// WithOptions allows gRPC server options to be set.
func (b *Builder) WithOptions(oo ...grpc.ServerOption) *Builder {
	if len(b.errors) != 0 {
		return b
	}
	b.serverOptions = append(b.serverOptions, oo...)
	return b
}

// WithDefinition allows
func (b *Builder) WithDefinition(description *grpc.ServiceDesc, service interface{}) *Builder {
	if len(b.errors) != 0 {
		return b
	}
	b.appendDefinition(description, service)
	return b
}

// Create the gRPC component.
func (b *Builder) Create() (*Component, error) {
	if len(b.errors) != 0 {
		return nil, errors.Aggregate(b.errors...)
	}

	b.serverOptions = append(b.serverOptions, grpc.UnaryInterceptor(tracingInterceptor))

	srv := grpc.NewServer(b.serverOptions...)

	for _, def := range b.definitions {
		srv.RegisterService(def.description, def.service)
	}

	return &Component{
		port: b.port,
		srv:  srv,
	}, nil
}

func (b *Builder) appendDefinition(description *grpc.ServiceDesc, service interface{}) {
	if description == nil {
		b.errors = append(b.errors, errors.New("service description is nil"))
		return
	}
	if service == nil {
		b.errors = append(b.errors, errors.New("service implementation is nil"))
		return
	}
	b.definitions = append(b.definitions, definition{description: description, service: service})
}

func tracingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.New(make(map[string]string, 0))
	}
	corID := getCorrelationID(md)
	sp, newCtx := trace.ConsumerSpan(ctx, trace.ComponentOpName(componentName, info.FullMethod), componentName,
		corID, mapHeader(md))
	logger := log.Sub(map[string]interface{}{"correlationID": corID})
	newCtx = log.WithContext(newCtx, logger)

	resp, err = handler(newCtx, req)
	if err != nil {
		trace.SpanError(sp)
	} else {
		trace.SpanSuccess(sp)
	}
	logRequestResponse(corID, info, err)
	return resp, err
}

func getCorrelationID(md metadata.MD) string {
	values := md.Get(correlation.HeaderID)
	if len(values) == 0 {
		return uuid.New().String()
	}
	return values[0]
}

func mapHeader(md metadata.MD) map[string]string {
	mp := make(map[string]string, md.Len())
	for key, values := range md {
		mp[key] = values[0]
	}
	return mp
}

func logRequestResponse(corID string, info *grpc.UnaryServerInfo, err error) {
	if !log.Enabled(log.DebugLevel) {
		return
	}

	fields := map[string]interface{}{
		"server-type":  "grpc",
		"method":       info.FullMethod,
		correlation.ID: corID,
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	log.Sub(fields).Debug()
}
