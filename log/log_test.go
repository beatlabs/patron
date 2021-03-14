package log

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLevelOrder(t *testing.T) {
	type args struct {
		lvl Level
	}
	tests := map[string]struct {
		args args
		want int
	}{
		"debug":    {args: args{lvl: DebugLevel}, want: 0},
		"info":     {args: args{lvl: InfoLevel}, want: 1},
		"warn":     {args: args{lvl: WarnLevel}, want: 2},
		"error":    {args: args{lvl: ErrorLevel}, want: 3},
		"fatal":    {args: args{lvl: FatalLevel}, want: 4},
		"panic":    {args: args{lvl: PanicLevel}, want: 5},
		"no level": {args: args{lvl: NoLevel}, want: 6},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, LevelOrder(tt.args.lvl))
		})
	}
}

func TestSetup(t *testing.T) {
	tests := map[string]struct {
		logger  Logger
		wantErr bool
	}{
		"failure with nil loggers": {logger: nil, wantErr: true},
		"success":                  {logger: &nilLogger{}, wantErr: false},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := Setup(tt.logger)
			if tt.wantErr {
				assert.Error(t, err, "expected error")
			} else {
				assert.NoError(t, err, "error not expected")
			}
		})
	}
}

func TestFromContext(t *testing.T) {
	logger = &nilLogger{}
	lg := &nilLogger{}
	ctxWith := WithContext(context.Background(), logger)
	ctxWithNil := WithContext(context.Background(), nil)
	type args struct {
		ctx context.Context
	}
	tests := map[string]struct {
		args args
		want Logger
	}{
		"with context logger":     {args: args{ctx: ctxWith}, want: logger},
		"without context logger":  {args: args{ctx: context.Background()}, want: lg},
		"with context nil logger": {args: args{ctx: ctxWithNil}, want: logger},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := FromContext(tt.args.ctx)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLog_Sub(t *testing.T) {
	l := testLogger{}
	logger = &l
	sl := Sub(map[string]interface{}{})
	assert.NotNil(t, sl)
}

func TestLog_Panic(t *testing.T) {
	l := testLogger{}
	logger = &l
	Panic("panic")
	assert.Equal(t, 1, l.panicCount)
}

func TestLog_Panicf(t *testing.T) {
	l := testLogger{}
	logger = &l
	Panicf("panic %s", "1")
	assert.Equal(t, 1, l.panicCount)
}

func TestLog_Fatal(t *testing.T) {
	l := testLogger{}
	logger = &l
	Fatal("fatal")
	assert.Equal(t, 1, l.fatalCount)
}

func TestLog_Fatalf(t *testing.T) {
	l := testLogger{}
	logger = &l
	Fatalf("fatal %s", "1")
	assert.Equal(t, 1, l.fatalCount)
}

func TestLog_Error(t *testing.T) {
	l := testLogger{}
	logger = &l
	Error("error")
	assert.Equal(t, 1, l.errorCount)
}

func TestLog_Errorf(t *testing.T) {
	l := testLogger{}
	logger = &l
	Errorf("error %s", "1")
	assert.Equal(t, 1, l.errorCount)
}

func TestLog_Warn(t *testing.T) {
	l := testLogger{}
	logger = &l
	Warn("warn")
	assert.Equal(t, 1, l.warnCount)
}

func TestLog_Warnf(t *testing.T) {
	l := testLogger{}
	logger = &l
	Warnf("warn %s", "1")
	assert.Equal(t, 1, l.warnCount)
}

func TestLog_Info(t *testing.T) {
	l := testLogger{}
	logger = &l
	Info("info")
	assert.Equal(t, 1, l.infoCount)
}

func TestLog_Infof(t *testing.T) {
	l := testLogger{}
	logger = &l
	Infof("info %s", "1")
	assert.Equal(t, 1, l.infoCount)
}

func TestLog_Debug(t *testing.T) {
	l := testLogger{}
	logger = &l
	Debug("debug")
	assert.Equal(t, 1, l.debugCount)
}

func TestLog_Debugf(t *testing.T) {
	l := testLogger{}
	logger = &l
	Debugf("debug %s", "1")
	assert.Equal(t, 1, l.debugCount)
}

func TestLog_Level(t *testing.T) {
	testCases := []struct {
		level   Level
		against Level
		enabled bool
	}{
		{DebugLevel, DebugLevel, true},
		{DebugLevel, InfoLevel, true},
		{InfoLevel, DebugLevel, false},
		{InfoLevel, InfoLevel, true},
		{InfoLevel, WarnLevel, true},
		{WarnLevel, InfoLevel, false},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s against %s", tc.level, tc.against), func(t *testing.T) {
			logger = &testLogger{level: tc.level}
			assert.Equal(t, tc.enabled, Enabled(tc.against))
		})
	}
}

func Test_nilLogger(t *testing.T) {
	l := &nilLogger{}
	l.Panic("test")
	l.Panicf("test", "123")
	l.Fatal("test", "123")
	l.Fatalf("test", "123")
	l.Error("test")
	l.Errorf("test", "123")
	l.Warn("test")
	l.Warnf("test", "123")
	l.Info("test")
	l.Infof("test", "123")
	l.Debug("test")
	l.Debugf("test", "123")

	assert.Equal(t, l, l.Sub(map[string]interface{}{}))
	assert.Equal(t, DebugLevel, l.Level())
}

var bCtx context.Context

func Benchmark_WithContext(b *testing.B) {
	l := Sub(map[string]interface{}{"subkey1": "subval1"})
	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		bCtx = WithContext(context.Background(), l)
	}
}

var l Logger

func Benchmark_FromContext(b *testing.B) {
	l = Sub(map[string]interface{}{"subkey1": "subval1"})
	ctx := WithContext(context.Background(), l)
	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		l = FromContext(ctx)
	}
}

type testLogger struct {
	debugCount int
	infoCount  int
	warnCount  int
	errorCount int
	fatalCount int
	panicCount int
	level      Level
}

func (t *testLogger) Sub(map[string]interface{}) Logger {
	return t
}

func (t *testLogger) Panic(_ ...interface{}) {
	t.panicCount++
}

func (t *testLogger) Panicf(_ string, _ ...interface{}) {
	t.panicCount++
}

func (t *testLogger) Fatal(_ ...interface{}) {
	t.fatalCount++
}

func (t *testLogger) Fatalf(_ string, _ ...interface{}) {
	t.fatalCount++
}

func (t *testLogger) Error(_ ...interface{}) {
	t.errorCount++
}

func (t *testLogger) Errorf(_ string, _ ...interface{}) {
	t.errorCount++
}

func (t *testLogger) Warn(_ ...interface{}) {
	t.warnCount++
}

func (t *testLogger) Warnf(_ string, _ ...interface{}) {
	t.warnCount++
}

func (t *testLogger) Info(_ ...interface{}) {
	t.infoCount++
}

func (t *testLogger) Infof(_ string, _ ...interface{}) {
	t.infoCount++
}

func (t *testLogger) Debug(_ ...interface{}) {
	t.debugCount++
}

func (t *testLogger) Debugf(_ string, _ ...interface{}) {
	t.debugCount++
}

func (t *testLogger) Level() Level {
	return t.level
}
