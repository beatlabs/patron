package log

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetup(t *testing.T) {
	t.Run("JSON", func(t *testing.T) {
		cfg := &Config{
			Attributes: []slog.Attr{},
			IsJSON:     true,
			Level:      "debug",
		}
		Setup(cfg)
		assert.NotNil(t, slog.Default())
	})

	t.Run("Text", func(t *testing.T) {
		cfg := &Config{
			Attributes: []slog.Attr{},
			IsJSON:     false,
			Level:      "debug",
		}
		Setup(cfg)
		assert.NotNil(t, slog.Default())
	})
}

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

func TestSetLevelAndCheckEnable(t *testing.T) {
	Setup(&Config{
		Attributes: []slog.Attr{},
		IsJSON:     true,
		Level:      "info",
	})

	assert.True(t, Enabled(slog.LevelInfo))
	assert.False(t, Enabled(slog.LevelDebug))

	SetLevel("debug")

	assert.True(t, Enabled(slog.LevelDebug))
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
