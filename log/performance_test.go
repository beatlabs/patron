package log_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/log/std"
	"github.com/beatlabs/patron/log/zerolog"
)

func Benchmark_LogFormatsComplex(b *testing.B) {
	loggers := map[string]log.Logger{
		"std":     std.New(&bytes.Buffer{}, log.DebugLevel, nil),
		"zerolog": zerolog.New(&bytes.Buffer{}, log.DebugLevel, nil),
	}

	for name, logger := range loggers {
		b.Run(fmt.Sprintf("logger_%s", name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				subLog := logger.Sub(map[string]interface{}{
					"string":  "with spaces",
					"string2": `with "quoted" part`,
					"string3": "with \n newline",
					"string4": `with\slashes\`,
					"simple":  `value`,
					"integer": 23748,
				})
				subLog.Infof(`testing %d\n\t"`, 1)
			}
		})
	}
}

func Benchmark_LogFormatsSimple(b *testing.B) {
	loggers := map[string]log.Logger{
		"std":     std.New(&bytes.Buffer{}, log.DebugLevel, nil),
		"zerolog": zerolog.New(&bytes.Buffer{}, log.DebugLevel, nil),
	}

	for name, logger := range loggers {
		b.Run(fmt.Sprintf("logger_%s", name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				logger.Infof(`testing %d"`, 1)
			}
		})
	}
}
