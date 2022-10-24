package grpc

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/examples"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

var mtr = mocktracer.New()

func TestMain(m *testing.M) {
	opentracing.SetGlobalTracer(mtr)
	code := m.Run()
	os.Exit(code)
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
	t.Cleanup(func() { mtr.Reset() })
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
			t.Cleanup(func() { mtr.Reset() })
			reqCtx := metadata.AppendToOutgoingContext(ctx, correlation.HeaderID, "123")
			r, err := c.SayHello(reqCtx, &examples.HelloRequest{Firstname: tt.args.requestName})
			if tt.expErr != "" {
				assert.EqualError(t, err, tt.expErr)
				assert.Nil(t, r)
			} else {
				require.NoError(t, err)
				assert.Equal(t, r.GetMessage(), "Hello TEST")

				assert.Len(t, mtr.FinishedSpans(), 1)

				expectedTags := map[string]interface{}{
					"component":     "gRPC-server",
					"correlationID": "123",
					"error":         false,
					"span.kind":     ext.SpanKindEnum("consumer"),
					"version":       "dev",
				}

				for _, span := range mtr.FinishedSpans() {
					assert.Equal(t, expectedTags, span.Tags())
				}

				assert.GreaterOrEqual(t, testutil.CollectAndCount(rpcHandledMetric, "component_grpc_handled_total"), 1)
				rpcHandledMetric.Reset()
				assert.GreaterOrEqual(t, testutil.CollectAndCount(rpcLatencyMetric, "component_grpc_handled_seconds"), 1)
				rpcLatencyMetric.Reset()
			}
		})
	}
	cnl()
	require.NoError(t, conn.Close())
	<-chDone
}

func TestComponent_Run_Stream(t *testing.T) {
	t.Cleanup(func() { mtr.Reset() })
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
			t.Cleanup(func() { mtr.Reset() })
			reqCtx := metadata.AppendToOutgoingContext(ctx, correlation.HeaderID, "123")
			client, err := c.SayHelloStream(reqCtx, &examples.HelloRequest{Firstname: tt.args.requestName})
			assert.NoError(t, err)
			resp, err := client.Recv()
			if tt.expErr != "" {
				assert.EqualError(t, err, tt.expErr)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				assert.Equal(t, resp.GetMessage(), "Hello TEST")
			}

			assert.Len(t, mtr.FinishedSpans(), 1)

			expectedTags := map[string]interface{}{
				"component":     "gRPC-server",
				"correlationID": "123",
				"error":         err != nil,
				"span.kind":     ext.SpanKindEnum("consumer"),
				"version":       "dev",
			}

			for _, span := range mtr.FinishedSpans() {
				assert.Equal(t, expectedTags, span.Tags())
			}

			assert.GreaterOrEqual(t, testutil.CollectAndCount(rpcHandledMetric, "component_grpc_handled_total"), 1)
			rpcHandledMetric.Reset()
			assert.GreaterOrEqual(t, testutil.CollectAndCount(rpcLatencyMetric, "component_grpc_handled_seconds"), 1)
			rpcLatencyMetric.Reset()

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
