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

// The Level type definition.
type Level string

const (
	// DebugLevel level.
	DebugLevel Level = "debug"
	// InfoLevel level.
	InfoLevel Level = "info"
	// WarnLevel level.
	WarnLevel Level = "warn"
	// ErrorLevel level.
	ErrorLevel Level = "error"
	// FatalLevel level.
	FatalLevel Level = "fatal"
	// PanicLevel level.
	PanicLevel Level = "panic"
	// NoLevel level.
	NoLevel Level = ""
)

var levelOrder = map[Level]int{
	DebugLevel: 0,
	InfoLevel:  1,
	WarnLevel:  2,
	ErrorLevel: 3,
	FatalLevel: 4,
	PanicLevel: 5,
	NoLevel:    6,
}

// LevelOrder returns the numerical order of the level.
func LevelOrder(lvl Level) int {
	return levelOrder[lvl]
}

// Logger interface definition of a logger.
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
	logger     Logger = &fmtLogger{}
	once       sync.Once
	logCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "observability",
			Subsystem: "log",
			Name:      "counter",
			Help:      "Counts the log level",
		},
		[]string{"level"},
	)
)

func init() {
	prometheus.MustRegister(logCounter)
}

// Setup logging by providing a logger factory.
func Setup(l Logger) error {
	if l == nil {
		return errors.New("logger is nil")
	}
	once.Do(func() {
		logger = l
	})

	return nil
}

// FromContext returns the logger in the context or a nil logger.
func FromContext(ctx context.Context) Logger {
	if l, ok := ctx.Value(ctxKey{}).(Logger); ok {
		if l == nil {
			return logger
		}
		return l
	}
	return logger
}

// WithContext associates a logger with a context for later reuse.
func WithContext(ctx context.Context, l Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// Sub returns a sub logger with new fields attached.
func Sub(ff map[string]interface{}) Logger {
	return logger.Sub(ff)
}

// Panic logging.
func Panic(args ...interface{}) {
	logCounter.WithLabelValues(string(PanicLevel)).Inc()
	logger.Panic(args...)
}

// Panicf logging.
func Panicf(msg string, args ...interface{}) {
	logCounter.WithLabelValues(string(PanicLevel)).Inc()
	logger.Panicf(msg, args...)
}

// Fatal logging.
func Fatal(args ...interface{}) {
	logCounter.WithLabelValues(string(FatalLevel)).Inc()
	logger.Fatal(args...)
}

// Fatalf logging.
func Fatalf(msg string, args ...interface{}) {
	logCounter.WithLabelValues(string(FatalLevel)).Inc()
	logger.Fatalf(msg, args...)
}

// Error logging.
func Error(args ...interface{}) {
	logCounter.WithLabelValues(string(ErrorLevel)).Inc()
	logger.Error(args...)
}

// Errorf logging.
func Errorf(msg string, args ...interface{}) {
	logCounter.WithLabelValues(string(ErrorLevel)).Inc()
	logger.Errorf(msg, args...)
}

// Warn logging.
func Warn(args ...interface{}) {
	logCounter.WithLabelValues(string(WarnLevel)).Inc()
	logger.Warn(args...)
}

// Warnf logging.
func Warnf(msg string, args ...interface{}) {
	logCounter.WithLabelValues(string(WarnLevel)).Inc()
	logger.Warnf(msg, args...)
}

// Info logging.
func Info(args ...interface{}) {
	logCounter.WithLabelValues(string(InfoLevel)).Inc()
	logger.Info(args...)
}

// Infof logging.
func Infof(msg string, args ...interface{}) {
	logCounter.WithLabelValues(string(InfoLevel)).Inc()
	logger.Infof(msg, args...)
}

// Debug logging.
func Debug(args ...interface{}) {
	logCounter.WithLabelValues(string(DebugLevel)).Inc()
	logger.Debug(args...)
}

// Debugf logging.
func Debugf(msg string, args ...interface{}) {
	logCounter.WithLabelValues(string(DebugLevel)).Inc()
	logger.Debugf(msg, args...)
}

// Enabled shows if the logger logs for the given level.
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
	fmt.Print(args...)
	panic(args)
}

// Panicf logging.
func (fl *fmtLogger) Panicf(msg string, args ...interface{}) {
	fmt.Printf(msg, args...)
	panic(args)
}

// Fatal logging.
func (fl *fmtLogger) Fatal(args ...interface{}) {
	fmt.Print(args...)
	os.Exit(1)
}

// Fatalf logging.
func (fl *fmtLogger) Fatalf(msg string, args ...interface{}) {
	fmt.Printf(msg, args...)
	os.Exit(1)
}

// Error logging.
func (fl *fmtLogger) Error(args ...interface{}) {
	fmt.Print(args...)
}

// Errorf logging.
func (fl *fmtLogger) Errorf(msg string, args ...interface{}) {
	fmt.Printf(msg, args...)
}

// Warn logging.
func (fl *fmtLogger) Warn(args ...interface{}) {
	fmt.Print(args...)
}

// Warnf logging.
func (fl *fmtLogger) Warnf(msg string, args ...interface{}) {
	fmt.Printf(msg, args...)
}

// Info logging.
func (fl *fmtLogger) Info(args ...interface{}) {
	fmt.Print(args...)
}

// Infof logging.
func (fl *fmtLogger) Infof(msg string, args ...interface{}) {
	fmt.Printf(msg, args...)
}

// Debug logging.
func (fl *fmtLogger) Debug(args ...interface{}) {
	fmt.Print(args...)
}

// Debugf logging.
func (fl *fmtLogger) Debugf(msg string, args ...interface{}) {
	fmt.Printf(msg, args...)
}

// Level returns the debug level of the nil logger.
func (fl *fmtLogger) Level() Level {
	return DebugLevel
}
