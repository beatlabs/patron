package log

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContext(t *testing.T) {
	l := slog.Default()

	t.Run("with logger", func(t *testing.T) {
		ctx := WithContext(context.Background(), l)
		assert.Equal(t, l, FromContext(ctx))
	})

	t.Run("with nil logger", func(t *testing.T) {
		ctx := WithContext(context.Background(), nil)
		assert.Equal(t, l, FromContext(ctx))
	})
}

func TestEnabled(t *testing.T) {
	type args struct {
		l slog.Level
	}
	tests := map[string]struct {
		args args
		want bool
	}{
		"Disabled": {args{slog.LevelDebug}, false},
		"Enabled":  {args{slog.LevelInfo}, true},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, Enabled(tt.args.l))
		})
	}
}

func TestErrorAttr(t *testing.T) {
	err := errors.New("error")
	errAttr := slog.Any("error", err)
	assert.Equal(t, errAttr, ErrorAttr(err))
}

var bCtx context.Context

func Benchmark_WithContext(b *testing.B) {
	l := slog.Default()
	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		bCtx = WithContext(context.Background(), l)
	}
}

var l *slog.Logger

func Benchmark_FromContext(b *testing.B) {
	l = slog.Default()
	ctx := WithContext(context.Background(), l)
	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		l = FromContext(ctx)
	}
}
