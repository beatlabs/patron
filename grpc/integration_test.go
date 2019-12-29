package grpc

import (
	"context"
	"testing"

	"github.com/beatlabs/patron/errors"

	"google.golang.org/grpc"

	"github.com/beatlabs/patron/grpc/helloworld"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type server struct {
	helloworld.UnimplementedGreeterServer
}

func (s *server) SayHello(ctx context.Context, in *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	if in.GetName() == "ERROR" {
		return nil, errors.New("ERROR")
	}
	return &helloworld.HelloReply{Message: "Hello " + in.GetName()}, nil
}

func TestComponent_Run(t *testing.T) {
	cmp, err := New(":60000").Create()
	require.NoError(t, err)
	helloworld.RegisterGreeterServer(cmp.Server(), &server{})
	ctx, cnl := context.WithCancel(context.Background())
	go func() {
		assert.NoError(t, cmp.Run(ctx))
	}()
	defer cnl()
	conn, err := grpc.Dial("localhost:60000", grpc.WithInsecure(), grpc.WithBlock())
	require.NoError(t, err)
	defer func() {
		require.NoError(t, conn.Close())
	}()
	c := helloworld.NewGreeterClient(conn)

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
			r, err := c.SayHello(ctx, &helloworld.HelloRequest{Name: tt.args.requestName})
			if tt.expErr != "" {
				assert.EqualError(t, err, tt.expErr)
				assert.Nil(t, r)
			} else {
				require.NoError(t, err)
				assert.Equal(t, r.GetMessage(), "Hello TEST")
			}
		})
	}
}
