// Package log provides logging abstractions.
package log

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// Level type definition.
type Level string

const (
	// DebugLevel definition.
	DebugLevel Level = "debug"
	// InfoLevel definition.
	InfoLevel Level = "info"
	// WarnLevel definition.
	WarnLevel Level = "warn"
	// ErrorLevel definition.
	ErrorLevel Level = "error"
	// FatalLevel definition.
	FatalLevel Level = "fatal"
	// PanicLevel definition.
	PanicLevel Level = "panic"
	// NoLevel definition.
	NoLevel Level = ""
)

var (
	levelOrder = map[Level]int{
		DebugLevel: 0,
		InfoLevel:  1,
		WarnLevel:  2,
		ErrorLevel: 3,
		FatalLevel: 4,
		PanicLevel: 5,
		NoLevel:    6,
	}
	logCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "observability",
			Subsystem: "log",
			Name:      "counter",
			Help:      "Counts logger calls per level",
		},
		[]string{"level"},
	)
)

// Logger interface definition.
type Logger interface {
	Sub(map[string]interface{}) Logger
	Fatal(...interface{})
	Fatalf(string, ...interface{})
	Panic(...interface{})
	Panicf(string, ...interface{})
	Error(...interface{})
	Errorf(string, ...interface{})
	Warn(...interface{})
	Warnf(string, ...interface{})
	Info(...interface{})
	Infof(string, ...interface{})
	Debug(...interface{})
	Debugf(string, ...interface{})
	Level() Level
}

type ctxKey struct{}

var (
	logger Logger = &fmtLogger{}
	once   sync.Once
)

func init() {
	prometheus.MustRegister(logCounter)
}

// LevelCount returns the total level prometheus counter.
func LevelCount(level string) prometheus.Counter {
	return logCounter.WithLabelValues(level)
}

// ResetLogCounter resets the prometheus counter.
func ResetLogCounter() {
	logCounter.Reset()
}

// IncreaseFatalCounter by one.
func IncreaseFatalCounter() {
	logCounter.WithLabelValues(string(FatalLevel)).Inc()
}

// IncreasePanicCounter by one.
func IncreasePanicCounter() {
	logCounter.WithLabelValues(string(PanicLevel)).Inc()
}

// IncreaseErrorCounter by one.
func IncreaseErrorCounter() {
	logCounter.WithLabelValues(string(ErrorLevel)).Inc()
}

// IncreaseWarnCounter by one.
func IncreaseWarnCounter() {
	logCounter.WithLabelValues(string(WarnLevel)).Inc()
}

// IncreaseInfoCounter by one.
func IncreaseInfoCounter() {
	logCounter.WithLabelValues(string(InfoLevel)).Inc()
}

// IncreaseDebugCounter by one.
func IncreaseDebugCounter() {
	logCounter.WithLabelValues(string(DebugLevel)).Inc()
}

// LevelOrder returns the numerical order of the level.
func LevelOrder(lvl Level) int {
	return levelOrder[lvl]
}

// Setup logging by providing a logger. The logger can only be set once.
func Setup(l Logger) error {
	if l == nil {
		return errors.New("logger is nil")
	}
	once.Do(func() {
		logger = l
	})

	return nil
}

// FromContext returns the logger, if it exists in the context, or nil.
func FromContext(ctx context.Context) Logger {
	if l, ok := ctx.Value(ctxKey{}).(Logger); ok {
		if l == nil {
			return logger
		}
		return l
	}
	return logger
}

// WithContext associates a logger to a context.
func WithContext(ctx context.Context, l Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// Sub returns a sub logger with new fields attached.
func Sub(ff map[string]interface{}) Logger {
	return logger.Sub(ff)
}

// Panic logging.
func Panic(args ...interface{}) {
	logger.Panic(args...)
}

// Panicf logging with message.
func Panicf(msg string, args ...interface{}) {
	logger.Panicf(msg, args...)
}

// Fatal logging.
func Fatal(args ...interface{}) {
	logger.Fatal(args...)
}

// Fatalf logging with message.
func Fatalf(msg string, args ...interface{}) {
	logger.Fatalf(msg, args...)
}

// Error logging.
func Error(args ...interface{}) {
	logger.Error(args...)
}

// Errorf logging with message.
func Errorf(msg string, args ...interface{}) {
	logger.Errorf(msg, args...)
}

// Warn logging.
func Warn(args ...interface{}) {
	logger.Warn(args...)
}

// Warnf logging with message.
func Warnf(msg string, args ...interface{}) {
	logger.Warnf(msg, args...)
}

// Info logging.
func Info(args ...interface{}) {
	logger.Info(args...)
}

// Infof logging with message.
func Infof(msg string, args ...interface{}) {
	logger.Infof(msg, args...)
}

// Debug logging.
func Debug(args ...interface{}) {
	logger.Debug(args...)
}

// Debugf logging with message.
func Debugf(msg string, args ...interface{}) {
	logger.Debugf(msg, args...)
}

// Enabled returns true for the appropriate level otherwise false.
func Enabled(l Level) bool {
	return levelOrder[logger.Level()] <= levelOrder[l]
}

type fmtLogger struct{}

// Sub returns a sub logger with new fields attached.
func (fl *fmtLogger) Sub(map[string]interface{}) Logger {
	return fl
}

// Panic logging.
func (fl *fmtLogger) Panic(args ...interface{}) {
	IncreasePanicCounter()
	fmt.Println(args...)
	panic(args)
}

// Panicf logging with message.
func (fl *fmtLogger) Panicf(msg string, args ...interface{}) {
	IncreasePanicCounter()
	fmt.Printf(appendNewLine(msg), args...)
	panic(args)
}

// Fatal logging.
func (fl *fmtLogger) Fatal(args ...interface{}) {
	IncreaseFatalCounter()
	fmt.Println(args...)
	os.Exit(1)
}

// Fatalf logging with message.
func (fl *fmtLogger) Fatalf(msg string, args ...interface{}) {
	IncreaseFatalCounter()
	fmt.Printf(appendNewLine(msg), args...)
	os.Exit(1)
}

// Error logging.
func (fl *fmtLogger) Error(args ...interface{}) {
	IncreaseErrorCounter()
	fmt.Println(args...)
}

// Errorf logging with message.
func (fl *fmtLogger) Errorf(msg string, args ...interface{}) {
	IncreaseErrorCounter()
	fmt.Printf(appendNewLine(msg), args...)
}

// Warn logging.
func (fl *fmtLogger) Warn(args ...interface{}) {
	IncreaseWarnCounter()
	fmt.Println(args...)
}

// Warnf logging with message.
func (fl *fmtLogger) Warnf(msg string, args ...interface{}) {
	IncreaseWarnCounter()
	fmt.Printf(appendNewLine(msg), args...)
}

// Info logging.
func (fl *fmtLogger) Info(args ...interface{}) {
	IncreaseInfoCounter()
	fmt.Println(args...)
}

// Infof logging with message.
func (fl *fmtLogger) Infof(msg string, args ...interface{}) {
	IncreaseInfoCounter()
	fmt.Printf(appendNewLine(msg), args...)
}

// Debug logging.
func (fl *fmtLogger) Debug(args ...interface{}) {
	IncreaseDebugCounter()
	fmt.Println(args...)
}

// Debugf logging with message.
func (fl *fmtLogger) Debugf(msg string, args ...interface{}) {
	IncreaseDebugCounter()
	fmt.Printf(appendNewLine(msg), args...)
}

// Level returns always the debug level.
func (fl *fmtLogger) Level() Level {
	return DebugLevel
}

func appendNewLine(msg string) string {
	if len(msg) == 0 || msg[len(msg)-1] != '\n' {
		return msg + "\n"
	}
	return msg
}
