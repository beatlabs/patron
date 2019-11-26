package log

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetup(t *testing.T) {
	tests := map[string]struct {
		f       FactoryFunc
		wantErr bool
	}{
		"failure with nil factory": {nil, true},
		"success":                  {func(map[string]interface{}) Logger { return nil }, false},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := Setup(tt.f, nil)
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
}

func (t *testLogger) Sub(map[string]interface{}) Logger {
	return t
}

func (t *testLogger) Panic(args ...interface{}) {
	t.panicCount++
}

func (t *testLogger) Panicf(msg string, args ...interface{}) {
	t.panicCount++
}

func (t *testLogger) Fatal(args ...interface{}) {
	t.fatalCount++
}

func (t *testLogger) Fatalf(msg string, args ...interface{}) {
	t.fatalCount++
}

func (t *testLogger) Error(args ...interface{}) {
	t.errorCount++
}

func (t *testLogger) Errorf(msg string, args ...interface{}) {
	t.errorCount++
}

func (t *testLogger) Warn(args ...interface{}) {
	t.warnCount++
}

func (t *testLogger) Warnf(msg string, args ...interface{}) {
	t.warnCount++
}

func (t *testLogger) Info(args ...interface{}) {
	t.infoCount++
}

func (t *testLogger) Infof(msg string, args ...interface{}) {
	t.infoCount++
}

func (t *testLogger) Debug(args ...interface{}) {
	t.debugCount++
}

func (t *testLogger) Debugf(msg string, args ...interface{}) {
	t.debugCount++
}
