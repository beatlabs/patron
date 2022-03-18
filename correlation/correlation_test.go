package correlation

import (
	"context"
	"net/http"
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
	val, ok := ctx.Value(idKey).(string)
	assert.True(t, ok)
	assert.Equal(t, "123", val)
}

func TestGetOrSetHeaderID(t *testing.T) {
	t.Parallel()
	withID := http.Header{HeaderID: []string{"123"}}
	withoutID := http.Header{HeaderID: []string{}}
	withEmptyID := http.Header{HeaderID: []string{""}}
	missingHeader := http.Header{}
	type args struct {
		hdr http.Header
	}
	tests := map[string]struct {
		args args
	}{
		"with id":        {args: args{hdr: withID}},
		"without id":     {args: args{hdr: withoutID}},
		"with empty id":  {args: args{hdr: withEmptyID}},
		"missing Header": {args: args{hdr: missingHeader}},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.NotEmpty(t, GetOrSetHeaderID(tt.args.hdr))
			assert.NotEmpty(t, tt.args.hdr[HeaderID][0])
		})
	}
}
