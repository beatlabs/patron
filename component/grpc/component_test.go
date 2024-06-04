package grpc

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/examples"
	patrontrace "github.com/beatlabs/patron/observability/trace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

var (
	tracePublisher *tracesdk.TracerProvider
	traceExporter  = tracetest.NewInMemoryExporter()
)

func TestMain(m *testing.M) {
	os.Setenv("OTEL_BSP_SCHEDULE_DELAY", "100")

	tracePublisher = patrontrace.Setup("test", nil, traceExporter)

	os.Exit(m.Run())
}

func TestCreate(t *testing.T) {
	t.Parallel()
	type args struct {
		port int
	}
	tests := map[string]struct {
		args   args
		expErr string
	}{
		"success":      {args: args{port: 60000}},
		"invalid port": {args: args{port: -1}, expErr: "port is invalid: -1"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := New(tt.args.port,
				WithServerOptions(grpc.ConnectionTimeout(1*time.Second)),
				WithReflection())
			if tt.expErr != "" {
				assert.EqualError(t, err, tt.expErr)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.args.port, got.port)
				assert.NotNil(t, got.Server())
			}
		})
	}
}

func TestComponent_Run_Unary(t *testing.T) {
	t.Cleanup(func() { traceExporter.Reset() })

	cmp, err := New(60000, WithReflection())
	require.NoError(t, err)
	examples.RegisterGreeterServer(cmp.Server(), &server{})
	ctx, cnl := context.WithCancel(context.Background())
	chDone := make(chan struct{})
	go func() {
		assert.NoError(t, cmp.Run(ctx))
		chDone <- struct{}{}
	}()
	conn, err := grpc.DialContext(ctx, "localhost:60000", grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock())
	require.NoError(t, err)
	c := examples.NewGreeterClient(conn)

	type args struct {
		requestName string
	}
	tests := map[string]struct {
		args   args
		expErr string
	}{
		"success": {args: args{requestName: "TEST"}},
		"error":   {args: args{requestName: "ERROR"}, expErr: "rpc error: code = Unknown desc = ERROR"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Cleanup(func() { traceExporter.Reset() })

			reqCtx := metadata.AppendToOutgoingContext(ctx, correlation.HeaderID, "123")
			r, err := c.SayHello(reqCtx, &examples.HelloRequest{Firstname: tt.args.requestName})

			assert.NoError(t, tracePublisher.ForceFlush(ctx))

			if tt.expErr != "" {
				assert.EqualError(t, err, tt.expErr)
				assert.Nil(t, r)

				time.Sleep(time.Second)
				spans := traceExporter.GetSpans()
				assert.Len(t, spans, 1)

				expectedSpan := tracetest.SpanStub{
					Name:     "examples.Greeter/SayHello",
					SpanKind: trace.SpanKindServer,
					Status: tracesdk.Status{
						Code:        codes.Error,
						Description: "ERROR",
					},
				}

				assertSpan(t, expectedSpan, spans[0])

			} else {
				require.NoError(t, err)
				assert.Equal(t, r.GetMessage(), "Hello TEST")

				time.Sleep(time.Second)
				spans := traceExporter.GetSpans()
				assert.Len(t, spans, 1)

				expectedSpan := tracetest.SpanStub{
					Name:     "examples.Greeter/SayHello",
					SpanKind: trace.SpanKindServer,
					Status: tracesdk.Status{
						Code: codes.Unset,
					},
				}

				assertSpan(t, expectedSpan, spans[0])
			}
		})
	}
	cnl()
	require.NoError(t, conn.Close())
	<-chDone
}

func TestComponent_Run_Stream(t *testing.T) {
	t.Cleanup(func() { traceExporter.Reset() })

	cmp, err := New(60000, WithReflection())
	require.NoError(t, err)
	examples.RegisterGreeterServer(cmp.Server(), &server{})
	ctx, cnl := context.WithCancel(context.Background())
	chDone := make(chan struct{})
	go func() {
		assert.NoError(t, cmp.Run(ctx))
		chDone <- struct{}{}
	}()
	conn, err := grpc.DialContext(ctx, "localhost:60000", grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock())
	require.NoError(t, err)
	c := examples.NewGreeterClient(conn)

	assert.NoError(t, tracePublisher.ForceFlush(ctx))

	type args struct {
		requestName string
	}
	tests := map[string]struct {
		args   args
		expErr string
	}{
		"success": {args: args{requestName: "TEST"}},
		"error":   {args: args{requestName: "ERROR"}, expErr: "rpc error: code = Unknown desc = ERROR"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Cleanup(func() { traceExporter.Reset() })

			reqCtx := metadata.AppendToOutgoingContext(ctx, correlation.HeaderID, "123")
			client, err := c.SayHelloStream(reqCtx, &examples.HelloRequest{Firstname: tt.args.requestName})
			assert.NoError(t, err)
			resp, err := client.Recv()

			assert.NoError(t, tracePublisher.ForceFlush(ctx))

			if tt.expErr != "" {
				assert.EqualError(t, err, tt.expErr)
				assert.Nil(t, resp)

				time.Sleep(time.Second)
				spans := traceExporter.GetSpans()
				assert.Len(t, spans, 1)

				expectedSpan := tracetest.SpanStub{
					Name:     "examples.Greeter/SayHelloStream",
					SpanKind: trace.SpanKindServer,
					Status: tracesdk.Status{
						Code:        codes.Error,
						Description: "ERROR",
					},
				}

				assertSpan(t, expectedSpan, spans[0])
			} else {
				require.NoError(t, err)
				assert.Equal(t, resp.GetMessage(), "Hello TEST")

				time.Sleep(time.Second)
				spans := traceExporter.GetSpans()
				assert.Len(t, spans, 1)

				expectedSpan := tracetest.SpanStub{
					Name:     "examples.Greeter/SayHelloStream",
					SpanKind: trace.SpanKindServer,
					Status: tracesdk.Status{
						Code: codes.Unset,
					},
				}

				assertSpan(t, expectedSpan, spans[0])
			}

			// TODO: Assert metrics
			// assert.GreaterOrEqual(t, testutil.CollectAndCount(rpcHandledMetric, "component_grpc_handled_total"), 1)
			// rpcHandledMetric.Reset()
			// assert.GreaterOrEqual(t, testutil.CollectAndCount(rpcLatencyMetric, "component_grpc_handled_seconds"), 1)
			// rpcLatencyMetric.Reset()

			assert.NoError(t, client.CloseSend())
		})
	}
	cnl()
	require.NoError(t, conn.Close())
	<-chDone
}

type server struct {
	examples.UnimplementedGreeterServer
}

func (s *server) SayHello(_ context.Context, in *examples.HelloRequest) (*examples.HelloReply, error) {
	if in.GetFirstname() == "ERROR" {
		return nil, errors.New("ERROR")
	}
	return &examples.HelloReply{Message: "Hello " + in.GetFirstname()}, nil
}

func (s *server) SayHelloStream(req *examples.HelloRequest, srv examples.Greeter_SayHelloStreamServer) error {
	if req.GetFirstname() == "ERROR" {
		return errors.New("ERROR")
	}

	return srv.Send(&examples.HelloReply{Message: "Hello " + req.GetFirstname()})
}

func assertSpan(t *testing.T, expected tracetest.SpanStub, got tracetest.SpanStub) {
	assert.Equal(t, expected.Name, got.Name)
	assert.Equal(t, expected.SpanKind, got.SpanKind)
	assert.Equal(t, expected.Status, got.Status)
}
