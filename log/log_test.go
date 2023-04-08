package log

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slog"
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
