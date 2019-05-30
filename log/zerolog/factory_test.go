package zerolog

import (
	"testing"
	"time"

	"github.com/beatlabs/patron/log"
	"github.com/stretchr/testify/assert"
)

func TestDefaultFactory(t *testing.T) {
	f := Create(log.InfoLevel)
	assert.NotNil(t, f)
}

func Test_getSource(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		wantSrc string
	}{
		{name: "empty", file: "", wantSrc: ""},
		{name: "no parent folder", file: "main.go", wantSrc: "main.go:10"},
		{name: "with parent folder", file: "/home/patron/main.go", wantSrc: "patron/main.go:10"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.wantSrc, getSource(tt.file, 10))
	}
}

func Test_sourceFields(t *testing.T) {
	key, src, ok := sourceFields(1)
	assert.True(t, ok)
	assert.Equal(t, "src", key)
	assert.Equal(t, "zerolog/factory_test.go:32", src)
}

var l log.Logger

func Benchmark_Create(b *testing.B) {
	f := Create(log.InfoLevel)
	fld := map[string]interface{}{
		"key1": "val1",
		"key2": 123,
		"key3": 123.456,
		"key4": time.Now(),
	}
	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		l = f(fld)
	}
}
