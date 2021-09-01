package zerolog

import (
	"bytes"
	"fmt"
	"regexp"
	"testing"

	"github.com/beatlabs/patron/log"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

const (
	logMsg  = "testing"
	logMsgf = "testing 1"
)

var f = map[string]interface{}{"key": "value"}

func init() {
	// use a fixed-skip source hook
	// it correctly identifies these tests'
	// source lines, matching behavior with non-test use-cases
	defaultSourceHook = &sourceHookWithSkip{
		skip: 4,
	}
	defaultSourceHookWithFormat = &sourceHookWithSkip{
		skip: 5,
	}
}

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name           string
		f              map[string]interface{}
		lvl            log.Level
		expectedFields int
	}{
		{"without fields", nil, log.DebugLevel, 0},
		{"with fields", f, log.DebugLevel, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sl := New(&zerolog.Logger{}, tt.lvl, tt.f)
			assert.NotNil(t, sl)
			assert.NotNil(t, sl.(*Logger).loggerf)
			assert.NotNil(t, sl.(*Logger).logger)
		})
	}
}

func TestLogger_Sub(t *testing.T) {
	var b bytes.Buffer
	l := New(&b, log.DebugLevel, f)
	sl := l.Sub(map[string]interface{}{"subkey1": "subval1"})
	assert.NotNil(t, sl)
	assert.NotNil(t, sl.(*Logger).loggerf)
	assert.NotNil(t, sl.(*Logger).logger)

	sl.Debug(logMsg)
	assertLog(t, b, log.DebugLevel, logMsg)
	assert.Contains(t, b.String(), `"subkey1":"subval1"`, b.String())
}

func TestLogger_Sub_NoFields(t *testing.T) {
	var b bytes.Buffer
	l := New(&b, log.DebugLevel, f)
	sl := l.Sub(nil)
	assert.NotNil(t, sl)
	sl.Debug(logMsg)
	assertLog(t, b, log.DebugLevel, logMsg)
}

func TestLogger_Panic(t *testing.T) {
	var b bytes.Buffer
	l := New(&b, log.DebugLevel, f)
	assert.Panics(t, func() { l.Panic(logMsg) })
	assertLog(t, b, log.PanicLevel, logMsg)
}

func TestLogger_Panicf(t *testing.T) {
	var b bytes.Buffer
	l := New(&b, log.PanicLevel, f)
	assert.Panics(t, func() { l.Panicf("testing %d", 1) })
	assertLog(t, b, log.PanicLevel, logMsgf)
}

func TestLogger_Error(t *testing.T) {
	var b bytes.Buffer
	l := New(&b, log.ErrorLevel, f)
	l.Error(logMsg)
	assertLog(t, b, log.ErrorLevel, logMsg)
}

func TestLogger_Errorf(t *testing.T) {
	var b bytes.Buffer
	l := New(&b, log.ErrorLevel, f)
	l.Errorf("testing %d", 1)
	assertLog(t, b, log.ErrorLevel, logMsgf)
}

func TestLogger_Warn(t *testing.T) {
	var b bytes.Buffer
	l := New(&b, log.WarnLevel, f)
	l.Warn(logMsg)
	assertLog(t, b, log.WarnLevel, logMsg)
}

func TestLogger_Warnf(t *testing.T) {
	var b bytes.Buffer
	l := New(&b, log.WarnLevel, f)
	l.Warnf("testing %d", 1)
	assertLog(t, b, log.WarnLevel, logMsgf)
}

func TestLogger_Info(t *testing.T) {
	var b bytes.Buffer
	l := New(&b, log.InfoLevel, f)
	l.Info(logMsg)
	assertLog(t, b, log.InfoLevel, logMsg)
}

func TestLogger_Infof(t *testing.T) {
	var b bytes.Buffer
	l := New(&b, log.InfoLevel, f)
	l.Infof("testing %d", 1)
	assertLog(t, b, log.InfoLevel, logMsgf)
}

func TestLogger_Debug(t *testing.T) {
	var b bytes.Buffer
	l := New(&b, log.DebugLevel, f)
	l.Debug(logMsg)
	assertLog(t, b, log.DebugLevel, logMsg)
}

func TestLogger_Debugf(t *testing.T) {
	var b bytes.Buffer
	l := New(&b, log.DebugLevel, f)
	l.Debugf("testing %d", 1)
	assertLog(t, b, log.DebugLevel, logMsgf)
}

func assertLog(t *testing.T, b bytes.Buffer, lvl log.Level, msg string) {
	assert.Contains(t, b.String(), fmt.Sprintf(`"lvl":"%s"`, lvl), b.String())
	assert.Contains(t, b.String(), `"key":"value"`, b.String())
	assert.Contains(t, b.String(), fmt.Sprintf(`"msg":"%s"`, msg), b.String())
	assert.Regexp(t, regexp.MustCompile(`"time":".*"`), b.String())
	assert.Regexp(t, regexp.MustCompile(`"src":"zerolog/logger_test.go:.*"`), b.String())
}

func TestLog_Level(t *testing.T) {
	var b bytes.Buffer
	testCases := []log.Level{
		log.DebugLevel,
		log.InfoLevel,
		log.WarnLevel,
	}
	for _, tc := range testCases {
		t.Run(string(tc), func(t *testing.T) {
			assert.Equal(t, tc, New(&b, tc, f).Level())
		})
	}
}

var t int

func Benchmark_LoggingEnabled(b *testing.B) {
	var bf bytes.Buffer
	l := New(&bf, log.DebugLevel, f)
	l.Debugf("testing %d", 1)
	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		l.Debugf("testing %d", 1)
		t = n
	}
}

func Benchmark_LoggingDisabled(b *testing.B) {
	var bf bytes.Buffer
	l := New(&bf, log.NoLevel, f)
	l.Debugf("testing %d", 1)
	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		l.Debugf("testing %d", 1)
		t = n
	}
}

var bl log.Logger

func Benchmark_Sub(b *testing.B) {
	var bf bytes.Buffer
	l := New(&bf, log.NoLevel, f)
	ff := map[string]interface{}{"subkey1": "subval1"}
	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		bl = l.Sub(ff)
		t = n
	}
}
