//go:build integration

package grpc

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/beatlabs/patron/examples"
	"github.com/beatlabs/patron/internal/test"
	"github.com/beatlabs/patron/observability/trace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type server struct {
	examples.UnimplementedGreeterServer
}

func (s *server) SayHelloStream(_ *examples.HelloRequest, _ examples.Greeter_SayHelloStreamServer) error {
	return status.Error(codes.Unavailable, "streaming not supported")
}

func (s *server) SayHello(_ context.Context, req *examples.HelloRequest) (*examples.HelloReply, error) {
	if req.GetFirstname() == "" {
		return nil, status.Error(codes.InvalidArgument, "first name cannot be empty")
	}
	return &examples.HelloReply{Message: fmt.Sprintf("Hello %s!", req.GetFirstname())}, nil
}

func TestSayHello(t *testing.T) {
	ctx := context.Background()

	client, closer, err := testServerAndClient(t)
	require.NoError(t, err)
	defer closer()

	// Tracing setup
	exp := tracetest.NewInMemoryExporter()
	tracePublisher := trace.Setup("test", nil, exp)

	// Metrics monitoring set up
	shutdownProvider, assertCollectMetrics := test.SetupMetrics(ctx, t)
	defer shutdownProvider()

	tt := map[string]struct {
		req         *examples.HelloRequest
		wantErr     bool
		wantCode    codes.Code
		wantMsg     string
		wantCounter int
	}{
		"ok": {
			req:         &examples.HelloRequest{Firstname: "John"},
			wantErr:     false,
			wantCode:    codes.OK,
			wantMsg:     "Hello John!",
			wantCounter: 1,
		},
		"invalid": {
			req:         &examples.HelloRequest{},
			wantErr:     true,
			wantCode:    codes.InvalidArgument,
			wantMsg:     "first name cannot be empty",
			wantCounter: 1,
		},
	}

	for n, tt := range tt {
		t.Run(n, func(t *testing.T) {
			t.Cleanup(func() { exp.Reset() })

			res, err := client.SayHello(ctx, tt.req)
			if tt.wantErr {
				require.Nil(t, res)
				require.Error(t, err)

				rpcStatus, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, tt.wantCode, rpcStatus.Code())
				require.Equal(t, tt.wantMsg, rpcStatus.Message())
			} else {
				require.NoError(t, err)
				require.NotNil(t, res)
				require.Equal(t, tt.wantMsg, res.GetMessage())
			}

			require.NoError(t, tracePublisher.ForceFlush(context.Background()))

			spans := exp.GetSpans().Snapshots()

			assert.Len(t, spans, 1)
			assert.Equal(t, "examples.Greeter/SayHello", spans[0].Name())
			assert.Equal(t, attribute.String("rpc.service", "examples.Greeter"), spans[0].Attributes()[0])
			assert.Equal(t, attribute.String("rpc.method", "SayHello"), spans[0].Attributes()[1])
			assert.Equal(t, attribute.String("rpc.system", "grpc"), spans[0].Attributes()[2])
			assert.Equal(t, attribute.String("net.sock.peer.addr", "127.0.0.1"), spans[0].Attributes()[3])
			// assert.Equal(t, attribute.Int64("net.sock.peer.port", 0), spans[0].Attributes()[4])
			assert.Equal(t, attribute.Int64("rpc.grpc.status_code", int64(tt.wantCode)), spans[0].Attributes()[5])

			// Metrics
			_ = assertCollectMetrics(4)
		})
	}
}

func testServerAndClient(t *testing.T) (examples.GreeterClient, func(), error) {
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, nil, err
	}

	baseServer := grpc.NewServer()
	examples.RegisterGreeterServer(baseServer, &server{})
	go func() {
		if err := baseServer.Serve(lis); err != nil {
			t.Logf("error serving server: %v", err)
		}
	}()

	closer := func() {
		baseServer.Stop()
		err := lis.Close()
		if err != nil {
			t.Logf("error closing listener: %v", err)
		}
	}

	conn, err := NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		defer closer()
		return nil, nil, err
	}

	client := examples.NewGreeterClient(conn)
	return client, closer, nil
}
