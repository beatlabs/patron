// Package zerolog is a concrete implementation of the log abstractions based on zerolog.
package zerolog

import (
	"fmt"
	"io"
	"path"
	"path/filepath"
	"runtime"
	"strings"
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

var (
	defaultSourceHook sourceHook = sourceHookByPackagePath{
		packagePath: "vendor/github.com/beatlabs/",
	}
	defaultSourceHookWithFormat = defaultSourceHook
)

func init() {
	zerolog.LevelFieldName = "lvl"
	zerolog.MessageFieldName = "msg"
	zerolog.TimeFieldFormat = time.RFC3339Nano
}

// Logger based on zerolog.
type Logger struct {
	logger  *zerolog.Logger
	loggerf *zerolog.Logger
	level   log.Level
}

// New returns a new logger.
func New(out io.Writer, lvl log.Level, f map[string]interface{}) log.Logger {
	zl := zerolog.New(out).With().Timestamp().Logger().Hook(defaultSourceHook)
	zlf := zerolog.New(out).With().Timestamp().Logger().Hook(defaultSourceHookWithFormat)

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
	logger := l.logger.With().Fields(ff).Logger()
	loggerf := l.loggerf.With().Fields(ff).Logger()
	return &Logger{logger: &logger, loggerf: &loggerf, level: l.level}
}

// Panic logging.
func (l *Logger) Panic(args ...interface{}) {
	log.IncreasePanicCounter()
	l.logger.Panic().Msg(fmt.Sprint(args...))
}

// Panicf logging with message.
func (l *Logger) Panicf(msg string, args ...interface{}) {
	log.IncreasePanicCounter()
	l.loggerf.Panic().Msgf(msg, args...)
}

// Fatal logging.
func (l *Logger) Fatal(args ...interface{}) {
	log.IncreaseFatalCounter()
	l.logger.Fatal().Msg(fmt.Sprint(args...))
}

// Fatalf logging with message.
func (l *Logger) Fatalf(msg string, args ...interface{}) {
	log.IncreaseFatalCounter()
	l.loggerf.Fatal().Msgf(msg, args...)
}

// Error logging.
func (l *Logger) Error(args ...interface{}) {
	log.IncreaseErrorCounter()
	l.logger.Error().Msg(fmt.Sprint(args...))
}

// Errorf logging with message.
func (l *Logger) Errorf(msg string, args ...interface{}) {
	log.IncreaseErrorCounter()
	l.loggerf.Error().Msgf(msg, args...)
}

// Warn logging.
func (l *Logger) Warn(args ...interface{}) {
	log.IncreaseWarnCounter()
	l.logger.Warn().Msg(fmt.Sprint(args...))
}

// Warnf logging with message.
func (l *Logger) Warnf(msg string, args ...interface{}) {
	log.IncreaseWarnCounter()
	l.loggerf.Warn().Msgf(msg, args...)
}

// Info logging.
func (l *Logger) Info(args ...interface{}) {
	log.IncreaseInfoCounter()
	l.logger.Info().Msg(fmt.Sprint(args...))
}

// Infof logging with message.
func (l *Logger) Infof(msg string, args ...interface{}) {
	log.IncreaseInfoCounter()
	l.loggerf.Info().Msgf(msg, args...)
}

// Debug logging.
func (l *Logger) Debug(args ...interface{}) {
	log.IncreaseDebugCounter()
	l.logger.Debug().Msg(fmt.Sprint(args...))
}

// Debugf logging with message.
func (l *Logger) Debugf(msg string, args ...interface{}) {
	log.IncreaseDebugCounter()
	l.loggerf.Debug().Msgf(msg, args...)
}

// Level of the logger.
func (l *Logger) Level() log.Level {
	return l.level
}

type sourceHook interface {
	Run(e *zerolog.Event, _ zerolog.Level, _ string)
}

var (
	_ sourceHook = &sourceHookByPackagePath{}
	_ sourceHook = &sourceHookWithSkip{}
)

type sourceHookByPackagePath struct {
	packagePath string
}

func (sh sourceHookByPackagePath) Run(e *zerolog.Event, _ zerolog.Level, _ string) {
	k, v, ok := sh.sourceFields()
	if !ok {
		return
	}
	e.Str(k, v)
}

func (sh sourceHookByPackagePath) sourceFields() (key, src string, ok bool) {
	var file string
	var line int

	skip := 5
	for {
		_, file, line, ok = runtime.Caller(skip)
		if !ok {
			return
		}

		if !strings.Contains(file, sh.packagePath) {
			break
		}

		skip++
	}

	// do not provide file/line number information for
	// log lines originated by beatlabs packages
	if strings.Contains(file, "src/runtime/") {
		ok = false
		return
	}

	src = getSource(file, line)
	key = "src"
	ok = true
	return key, src, ok
}

type sourceHookWithSkip struct {
	skip int
}

func (sh sourceHookWithSkip) Run(e *zerolog.Event, _ zerolog.Level, _ string) {
	k, v, ok := sh.sourceFields()
	if !ok {
		return
	}
	e.Str(k, v)
}

func (sh sourceHookWithSkip) sourceFields() (key, src string, ok bool) {
	_, file, line, ok := runtime.Caller(sh.skip)
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
