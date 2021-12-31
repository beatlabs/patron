package correlation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIDFromContext(t *testing.T) {
	t.Parallel()
	ctxWith := ContextWithID(context.Background(), "123")
	type args struct {
		ctx context.Context
	}
	tests := map[string]struct {
		args args
	}{
		"with existing id":    {args: args{ctx: ctxWith}},
		"without existing id": {args: args{ctx: context.Background()}},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := IDFromContext(tt.args.ctx)
			assert.NotEmpty(t, got)
		})
	}
}

func TestContextWithID(t *testing.T) {
	ctx := ContextWithID(context.Background(), "123")
	assert.Equal(t, "123", ctx.Value(idKey).(string))
}
