// Package zerolog is a concrete implementation of the log abstractions.
package zerolog

import (
	"fmt"
	"io"
	"path"
	"path/filepath"
	"runtime"
	"time"

	"github.com/beatlabs/patron/log"
	"github.com/rs/zerolog"
)

var levelMap = map[log.Level]zerolog.Level{
	log.NoLevel:    zerolog.NoLevel,
	log.DebugLevel: zerolog.DebugLevel,
	log.InfoLevel:  zerolog.InfoLevel,
	log.WarnLevel:  zerolog.WarnLevel,
	log.ErrorLevel: zerolog.ErrorLevel,
	log.FatalLevel: zerolog.FatalLevel,
	log.PanicLevel: zerolog.PanicLevel,
}

func init() {
	zerolog.LevelFieldName = "lvl"
	zerolog.MessageFieldName = "msg"
	zerolog.TimeFieldFormat = time.RFC3339Nano
}

// Logger abstraction based on zerolog.
type Logger struct {
	logger  *zerolog.Logger
	loggerf *zerolog.Logger
	level   log.Level
}

// New creates a new logger.
func New(out io.Writer, lvl log.Level, f map[string]interface{}) log.Logger {
	zl := zerolog.New(out).With().Timestamp().Logger().Hook(sourceHook{skip: 4})
	zlf := zerolog.New(out).With().Timestamp().Logger().Hook(sourceHook{skip: 5})

	if len(f) == 0 {
		f = make(map[string]interface{})
	}
	logger := zl.Level(levelMap[lvl]).With().Fields(f).Logger()
	loggerf := zlf.Level(levelMap[lvl]).With().Fields(f).Logger()
	return &Logger{logger: &logger, loggerf: &loggerf, level: lvl}
}

// Sub returns a sub logger with new fields attached.
func (l *Logger) Sub(ff map[string]interface{}) log.Logger {
	if ff == nil {
		return l
	}
	sl := l.logger.With().Fields(ff).Logger()
	return &Logger{logger: &sl, level: l.level}
}

// Panic logging.
func (l *Logger) Panic(args ...interface{}) {
	l.logger.Panic().Msg(fmt.Sprint(args...))
}

// Panicf logging.
func (l *Logger) Panicf(msg string, args ...interface{}) {
	l.loggerf.Panic().Msgf(msg, args...)
}

// Fatal logging.
func (l *Logger) Fatal(args ...interface{}) {
	l.logger.Fatal().Msg(fmt.Sprint(args...))
}

// Fatalf logging.
func (l *Logger) Fatalf(msg string, args ...interface{}) {
	l.loggerf.Fatal().Msgf(msg, args...)
}

// Error logging.
func (l *Logger) Error(args ...interface{}) {
	l.logger.Error().Msg(fmt.Sprint(args...))
}

// Errorf logging.
func (l *Logger) Errorf(msg string, args ...interface{}) {
	l.loggerf.Error().Msgf(msg, args...)
}

// Warn logging.
func (l *Logger) Warn(args ...interface{}) {
	l.logger.Warn().Msg(fmt.Sprint(args...))
}

// Warnf logging.
func (l *Logger) Warnf(msg string, args ...interface{}) {
	l.loggerf.Warn().Msgf(msg, args...)
}

// Info logging.
func (l *Logger) Info(args ...interface{}) {
	l.logger.Info().Msg(fmt.Sprint(args...))
}

// Infof logging.
func (l *Logger) Infof(msg string, args ...interface{}) {
	l.loggerf.Info().Msgf(msg, args...)
}

// Debug logging.
func (l *Logger) Debug(args ...interface{}) {
	l.logger.Debug().Msg(fmt.Sprint(args...))
}

// Debugf logging.
func (l *Logger) Debugf(msg string, args ...interface{}) {
	l.loggerf.Debug().Msgf(msg, args...)
}

// Level return the logging level.
func (l *Logger) Level() log.Level {
	return l.level
}

type sourceHook struct {
	skip int
}

func (sh sourceHook) Run(e *zerolog.Event, _ zerolog.Level, _ string) {
	k, v, ok := sourceFields(sh.skip)
	if !ok {
		return
	}
	e.Str(k, v)
}

func sourceFields(skip int) (key, src string, ok bool) {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return
	}
	src = getSource(file, line)
	key = "src"
	ok = true
	return key, src, ok
}

func getSource(file string, line int) (src string) {
	if file == "" {
		return
	}
	d, f := filepath.Split(file)
	d = path.Base(d)
	if d == "." || d == "" {
		src = fmt.Sprintf("%s:%d", f, line)
	} else {
		src = fmt.Sprintf("%s/%s:%d", d, f, line)
	}
	return src
}
