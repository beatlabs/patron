package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestCreate(t *testing.T) {
	type args struct {
		port int
	}
	tests := map[string]struct {
		args   args
		expErr string
	}{
		"success":      {args: args{port: 60000}},
		"invalid port": {args: args{port: -1}, expErr: "port is invalid: -1\n"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := New(tt.args.port).WithOptions(grpc.ConnectionTimeout(1 * time.Second)).Create()
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

type server struct {
	UnimplementedGreeterServer
}

func (s *server) SayHello(_ context.Context, in *HelloRequest) (*HelloReply, error) {
	if in.GetFirstname() == "ERROR" {
		return nil, errors.New("ERROR")
	}
	return &HelloReply{Message: "Hello " + in.GetFirstname()}, nil
}

func (s *server) SayHelloStream(req *HelloRequest, srv Greeter_SayHelloStreamServer) error {
	if req.GetFirstname() == "ERROR" {
		return errors.New("ERROR")
	}

	return srv.Send(&HelloReply{Message: "Hello " + req.GetFirstname()})
}

func TestComponent_Run_Unary(t *testing.T) {
	cmp, err := New(60000).Create()
	require.NoError(t, err)
	RegisterGreeterServer(cmp.Server(), &server{})
	ctx, cnl := context.WithCancel(context.Background())
	chDone := make(chan struct{})
	go func() {
		assert.NoError(t, cmp.Run(ctx))
		chDone <- struct{}{}
	}()
	conn, err := grpc.Dial("localhost:60000", grpc.WithInsecure(), grpc.WithBlock())
	require.NoError(t, err)
	c := NewGreeterClient(conn)

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
		t.Run(name, func(t *testing.T) {
			r, err := c.SayHello(ctx, &HelloRequest{Firstname: tt.args.requestName})
			if tt.expErr != "" {
				assert.EqualError(t, err, tt.expErr)
				assert.Nil(t, r)
			} else {
				require.NoError(t, err)
				assert.Equal(t, r.GetMessage(), "Hello TEST")
			}
		})
	}
	cnl()
	require.NoError(t, conn.Close())
	<-chDone
}

func TestComponent_Run_Stream(t *testing.T) {
	cmp, err := New(60000).Create()
	require.NoError(t, err)
	RegisterGreeterServer(cmp.Server(), &server{})
	ctx, cnl := context.WithCancel(context.Background())
	chDone := make(chan struct{})
	go func() {
		assert.NoError(t, cmp.Run(ctx))
		chDone <- struct{}{}
	}()
	conn, err := grpc.Dial("localhost:60000", grpc.WithInsecure(), grpc.WithBlock())
	require.NoError(t, err)
	c := NewGreeterClient(conn)

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
		t.Run(name, func(t *testing.T) {
			client, err := c.SayHelloStream(ctx, &HelloRequest{Firstname: tt.args.requestName})
			assert.NoError(t, err)
			resp, err := client.Recv()
			if tt.expErr != "" {
				assert.EqualError(t, err, tt.expErr)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				assert.Equal(t, resp.GetMessage(), "Hello TEST")
			}
			assert.NoError(t, client.CloseSend())
		})
	}
	cnl()
	require.NoError(t, conn.Close())
	<-chDone
}
