package grpc

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"testing"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/beatlabs/patron/examples"
)

const (
	bufSize = 1024 * 1024
	target  = "bufnet"
)

var lis *bufconn.Listener

type server struct{}

func (s *server) SayHelloStream(_ *examples.HelloRequest, _ examples.Greeter_SayHelloStreamServer) error {
	return status.Error(codes.Unavailable, "streaming not supported")
}

func (s *server) SayHello(_ context.Context, req *examples.HelloRequest) (*examples.HelloReply, error) {
	if req.Firstname == "" {
		return nil, status.Error(codes.InvalidArgument, "first name cannot be empty")
	}
	return &examples.HelloReply{Message: fmt.Sprintf("Hello %s!", req.Firstname)}, nil
}

func TestMain(m *testing.M) {
	lis = bufconn.Listen(bufSize)
	s := grpc.NewServer()
	examples.RegisterGreeterServer(s, &server{})
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatal(err)
		}
	}()

	code := m.Run()

	s.GracefulStop()

	os.Exit(code)
}

func bufDialer(_ context.Context, _ string) (net.Conn, error) {
	return lis.Dial()
}

func TestDial(t *testing.T) {
	conn, err := Dial(target, grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	assert.NoError(t, err)
	assert.NotNil(t, conn)
	assert.NoError(t, conn.Close())
}

func TestDialContext(t *testing.T) {
	type args struct {
		opts []grpc.DialOption
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {
			args: args{
				opts: []grpc.DialOption{grpc.WithContextDialer(bufDialer), grpc.WithInsecure()},
			},
		},
		"failure missing grpc.WithInsecure()": {
			args:        args{},
			expectedErr: "grpc: no transport security set (use grpc.WithInsecure() explicitly or set credentials)",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gotConn, err := DialContext(context.Background(), target, tt.args.opts...)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				assert.Nil(t, gotConn)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, gotConn)
			}
		})
	}
}

func TestSayHello(t *testing.T) {
	mtr := mocktracer.New()
	opentracing.SetGlobalTracer(mtr)

	ctx := context.Background()
	conn, err := DialContext(ctx, target, grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	require.NoError(t, err)
	defer func() {
		require.NoError(t, conn.Close())
	}()

	client := examples.NewGreeterClient(conn)

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
		"internal": {
			req:         nil, /* oops */
			wantErr:     true,
			wantCode:    codes.Internal,
			wantMsg:     "grpc: error while marshaling: proto: Marshal called with nil",
			wantCounter: 1,
		},
	}

	for n, tc := range tt {
		t.Run(n, func(t *testing.T) {
			res, err := client.SayHello(ctx, tc.req)
			if tc.wantErr {
				require.Nil(t, res)
				require.Error(t, err)

				rpcStatus, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, tc.wantCode, rpcStatus.Code())
				require.Equal(t, tc.wantMsg, rpcStatus.Message())
			} else {
				require.NoError(t, err)
				require.NotNil(t, res)
				require.Equal(t, tc.wantMsg, res.GetMessage())
			}

			// Tracing
			wantSpanTags := map[string]interface{}{
				"component": "grpc-client",
				"version":   "dev",
				"span.kind": ext.SpanKindEnum("producer"),
				"error":     tc.wantErr,
			}
			assert.Equal(t, wantSpanTags, mtr.FinishedSpans()[0].Tags())
			mtr.Reset()

			// Metrics
			assert.Equal(t, tc.wantCounter, testutil.CollectAndCount(rpcDurationMetrics))
			rpcDurationMetrics.Reset()
		})
	}
}
