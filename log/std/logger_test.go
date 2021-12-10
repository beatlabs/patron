package std

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/beatlabs/patron/log"
	"github.com/stretchr/testify/assert"
)

func TestNewLogger(t *testing.T) {
	var b bytes.Buffer
	logger := New(&b, log.InfoLevel, map[string]interface{}{"name": "john doe", "age": 18})
	assert.NotNil(t, logger.debug)
	assert.NotNil(t, logger.info)
	assert.NotNil(t, logger.warn)
	assert.NotNil(t, logger.error)
	assert.NotNil(t, logger.fatal)
	assert.NotNil(t, logger.panic)
	assert.Equal(t, log.InfoLevel, logger.Level())
	assert.Equal(t, logger.fields, map[string]interface{}{"name": "john doe", "age": 18})
	assert.Contains(t, logger.fieldsLine, "age=18")
	assert.Contains(t, logger.fieldsLine, "name=john doe")
}

func TestNewWithFlagsLogger(t *testing.T) {
	var b bytes.Buffer
	logger := NewWithFlags(&b, log.InfoLevel, map[string]interface{}{"name": "john doe", "age": 18}, 0)
	assert.NotNil(t, logger.debug)
	assert.NotNil(t, logger.info)
	assert.NotNil(t, logger.warn)
	assert.NotNil(t, logger.error)
	assert.NotNil(t, logger.fatal)
	assert.NotNil(t, logger.panic)
	assert.Equal(t, log.InfoLevel, logger.Level())
	assert.Equal(t, logger.fields, map[string]interface{}{"name": "john doe", "age": 18})
	assert.Contains(t, logger.fieldsLine, "age=18")
	assert.Contains(t, logger.fieldsLine, "name=john doe")
	year := time.Now().Format("2006")
	assert.NotContains(t, logger.fieldsLine, year)
}

func TestNewSub(t *testing.T) {
	var b bytes.Buffer
	logger := New(&b, log.InfoLevel, map[string]interface{}{"name": "john doe"})
	assert.NotNil(t, logger)
	subLogger := logger.Sub(map[string]interface{}{"age": 18}).(*Logger)
	assert.NotNil(t, subLogger.debug)
	assert.NotNil(t, subLogger.info)
	assert.NotNil(t, subLogger.warn)
	assert.NotNil(t, subLogger.error)
	assert.NotNil(t, subLogger.fatal)
	assert.NotNil(t, subLogger.panic)
	assert.Equal(t, log.InfoLevel, subLogger.Level())
	assert.Equal(t, subLogger.fields, map[string]interface{}{"name": "john doe", "age": 18})
	assert.Contains(t, subLogger.fieldsLine, "age=18")
	assert.Contains(t, subLogger.fieldsLine, "name=john doe")
}

func TestLogger(t *testing.T) {
	// BEWARE: Since we are testing the log output change in line number of statements affect the test outcome
	var b bytes.Buffer
	logger := New(&b, log.DebugLevel, map[string]interface{}{"name": "john doe", "age": 18})

	type args struct {
		lvl  log.Level
		msg  string
		args []interface{}
	}
	tests := map[string]struct {
		args args
	}{
		"debug":  {args: args{lvl: log.DebugLevel, args: []interface{}{"hello world"}}},
		"debugf": {args: args{lvl: log.DebugLevel, msg: "Hi, %s", args: []interface{}{"John"}}},
		"info":   {args: args{lvl: log.InfoLevel, args: []interface{}{"hello world"}}},
		"infof":  {args: args{lvl: log.InfoLevel, msg: "Hi, %s", args: []interface{}{"John"}}},
		"warn":   {args: args{lvl: log.WarnLevel, args: []interface{}{"hello world"}}},
		"warnf":  {args: args{lvl: log.WarnLevel, msg: "Hi, %s", args: []interface{}{"John"}}},
		"error":  {args: args{lvl: log.ErrorLevel, args: []interface{}{"hello world"}}},
		"errorf": {args: args{lvl: log.ErrorLevel, msg: "Hi, %s", args: []interface{}{"John"}}},
		"panic":  {args: args{lvl: log.PanicLevel, args: []interface{}{"hello world"}}},
		"panicf": {args: args{lvl: log.PanicLevel, msg: "Hi, %s", args: []interface{}{"John"}}},
	}
	for name, tt := range tests {
		tst := tt
		t.Run(name, func(t *testing.T) {
			defer b.Reset()

			switch tst.args.lvl {
			case log.DebugLevel:
				if tst.args.msg == "" {
					logger.Debug(tst.args.args...)
				} else {
					logger.Debugf(tst.args.msg, tst.args.args...)
				}
			case log.InfoLevel:
				if tst.args.msg == "" {
					logger.Info(tst.args.args...)
				} else {
					logger.Infof(tst.args.msg, tst.args.args...)
				}
			case log.WarnLevel:
				if tst.args.msg == "" {
					logger.Warn(tst.args.args...)
				} else {
					logger.Warnf(tst.args.msg, tst.args.args...)
				}
			case log.ErrorLevel:
				if tst.args.msg == "" {
					logger.Error(tst.args.args...)
				} else {
					logger.Errorf(tst.args.msg, tst.args.args...)
				}
			case log.PanicLevel:
				if tst.args.msg == "" {
					assert.Panics(t, func() {
						logger.Panic(tst.args.args...)
					})
				} else {
					assert.Panics(t, func() {
						logger.Panicf(tst.args.msg, tst.args.args...)
					})
				}
			}

			if tst.args.msg == "" {
				assert.Contains(t, b.String(), fmt.Sprintf("lvl=%s age=18 name=john doe hello world", levelMap[tst.args.lvl]))
			} else {
				assert.Contains(t, b.String(), fmt.Sprintf("lvl=%s age=18 name=john doe Hi, John", levelMap[tst.args.lvl]))
			}
		})
	}
}

func TestLogger_shouldLog(t *testing.T) {
	t.Parallel()
	type args struct {
		lvl log.Level
	}
	tests := map[string]struct {
		setupLevel log.Level
		args       args
		want       bool
	}{
		"setup debug,passing debug":    {setupLevel: log.DebugLevel, args: args{lvl: log.DebugLevel}, want: true},
		"setup debug,passing info":     {setupLevel: log.DebugLevel, args: args{lvl: log.InfoLevel}, want: true},
		"setup debug,passing warn":     {setupLevel: log.DebugLevel, args: args{lvl: log.WarnLevel}, want: true},
		"setup debug,passing error":    {setupLevel: log.DebugLevel, args: args{lvl: log.ErrorLevel}, want: true},
		"setup debug,passing panic":    {setupLevel: log.DebugLevel, args: args{lvl: log.PanicLevel}, want: true},
		"setup debug,passing fatal":    {setupLevel: log.DebugLevel, args: args{lvl: log.FatalLevel}, want: true},
		"setup info,passing debug":     {setupLevel: log.InfoLevel, args: args{lvl: log.DebugLevel}, want: false},
		"setup info,passing info":      {setupLevel: log.InfoLevel, args: args{lvl: log.InfoLevel}, want: true},
		"setup info,passing warn":      {setupLevel: log.InfoLevel, args: args{lvl: log.WarnLevel}, want: true},
		"setup info,passing error":     {setupLevel: log.InfoLevel, args: args{lvl: log.ErrorLevel}, want: true},
		"setup info,passing panic":     {setupLevel: log.InfoLevel, args: args{lvl: log.PanicLevel}, want: true},
		"setup info,passing fatal":     {setupLevel: log.InfoLevel, args: args{lvl: log.FatalLevel}, want: true},
		"setup warn,passing debug":     {setupLevel: log.WarnLevel, args: args{lvl: log.DebugLevel}, want: false},
		"setup warn,passing info":      {setupLevel: log.WarnLevel, args: args{lvl: log.InfoLevel}, want: false},
		"setup warn,passing warn":      {setupLevel: log.WarnLevel, args: args{lvl: log.WarnLevel}, want: true},
		"setup warn,passing error":     {setupLevel: log.WarnLevel, args: args{lvl: log.ErrorLevel}, want: true},
		"setup warn,passing panic":     {setupLevel: log.WarnLevel, args: args{lvl: log.PanicLevel}, want: true},
		"setup warn,passing fatal":     {setupLevel: log.WarnLevel, args: args{lvl: log.FatalLevel}, want: true},
		"setup error,passing debug":    {setupLevel: log.ErrorLevel, args: args{lvl: log.DebugLevel}, want: false},
		"setup error,passing info":     {setupLevel: log.ErrorLevel, args: args{lvl: log.InfoLevel}, want: false},
		"setup error,passing warn":     {setupLevel: log.ErrorLevel, args: args{lvl: log.WarnLevel}, want: false},
		"setup error,passing error":    {setupLevel: log.ErrorLevel, args: args{lvl: log.ErrorLevel}, want: true},
		"setup error,passing panic":    {setupLevel: log.ErrorLevel, args: args{lvl: log.PanicLevel}, want: true},
		"setup error,passing fatal":    {setupLevel: log.ErrorLevel, args: args{lvl: log.FatalLevel}, want: true},
		"setup fatal,passing debug":    {setupLevel: log.FatalLevel, args: args{lvl: log.DebugLevel}, want: false},
		"setup fatal,passing info":     {setupLevel: log.FatalLevel, args: args{lvl: log.InfoLevel}, want: false},
		"setup fatal,passing warn":     {setupLevel: log.FatalLevel, args: args{lvl: log.WarnLevel}, want: false},
		"setup fatal,passing error":    {setupLevel: log.FatalLevel, args: args{lvl: log.ErrorLevel}, want: false},
		"setup fatal,passing fatal":    {setupLevel: log.FatalLevel, args: args{lvl: log.FatalLevel}, want: true},
		"setup fatal,passing panic":    {setupLevel: log.FatalLevel, args: args{lvl: log.PanicLevel}, want: true},
		"setup panic,passing debug":    {setupLevel: log.PanicLevel, args: args{lvl: log.DebugLevel}, want: false},
		"setup panic,passing info":     {setupLevel: log.PanicLevel, args: args{lvl: log.InfoLevel}, want: false},
		"setup panic,passing warn":     {setupLevel: log.PanicLevel, args: args{lvl: log.WarnLevel}, want: false},
		"setup panic,passing error":    {setupLevel: log.PanicLevel, args: args{lvl: log.ErrorLevel}, want: false},
		"setup panic,passing fatal":    {setupLevel: log.PanicLevel, args: args{lvl: log.FatalLevel}, want: false},
		"setup panic,passing panic":    {setupLevel: log.PanicLevel, args: args{lvl: log.PanicLevel}, want: true},
		"setup no level,passing debug": {setupLevel: log.NoLevel, args: args{lvl: log.DebugLevel}, want: false},
		"setup no level,passing info":  {setupLevel: log.NoLevel, args: args{lvl: log.InfoLevel}, want: false},
		"setup no level,passing warn":  {setupLevel: log.NoLevel, args: args{lvl: log.WarnLevel}, want: false},
		"setup no level,passing error": {setupLevel: log.NoLevel, args: args{lvl: log.ErrorLevel}, want: false},
		"setup no level,passing panic": {setupLevel: log.NoLevel, args: args{lvl: log.PanicLevel}, want: false},
		"setup no level,passing fatal": {setupLevel: log.NoLevel, args: args{lvl: log.FatalLevel}, want: false},
	}
	for name, tt := range tests {
		tst := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			l := &Logger{level: tst.setupLevel}
			assert.Equal(t, tst.want, l.shouldLog(tst.args.lvl))
		})
	}
}

func TestLogger_shouldNotLog(t *testing.T) {
	var b bytes.Buffer
	logger := New(&b, log.NoLevel, map[string]interface{}{"name": "john doe", "age": 18})

	logger.Debug("123")
	logger.Debugf("123 %s", "123")
	logger.Info("123")
	logger.Infof("123 %s", "123")
	logger.Warn("123")
	logger.Warnf("123 %s", "123")
	logger.Warn("123")
	logger.Warnf("123 %s", "123")
	logger.Error("123")
	logger.Errorf("123 %s", "123")
	logger.Fatal("123")
	logger.Fatalf("123 %s", "123")
	logger.Panic("123")
	logger.Panicf("123 %s", "123")

	assert.Empty(t, b.String())
}

var buf bytes.Buffer

func BenchmarkLogger(b *testing.B) {
	var tmpBuf bytes.Buffer
	logger := New(&tmpBuf, log.DebugLevel, map[string]interface{}{"name": "john doe", "age": 18})
	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		logger.Debugf("Hello %s!", "John")
	}
	buf = tmpBuf
}
