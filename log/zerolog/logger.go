// Package zerolog is a concrete implementation of the log abstractions.
package zerolog

import (
	"fmt"

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

// Logger abstraction based on zerolog.
type Logger struct {
	logger *zerolog.Logger
	level  log.Level
}

// NewLogger creates a new logger.
func NewLogger(l *zerolog.Logger, lvl log.Level, f map[string]interface{}) log.Logger {
	if len(f) == 0 {
		f = make(map[string]interface{})
	}
	zl := l.Level(levelMap[lvl]).With().Fields(f).Logger()
	return &Logger{logger: &zl, level: lvl}
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
	l.logger.Panic().Msgf(msg, args...)
}

// Fatal logging.
func (l *Logger) Fatal(args ...interface{}) {
	l.logger.Fatal().Msg(fmt.Sprint(args...))
}

// Fatalf logging.
func (l *Logger) Fatalf(msg string, args ...interface{}) {
	l.logger.Fatal().Msgf(msg, args...)
}

// Error logging.
func (l *Logger) Error(args ...interface{}) {
	l.logger.Error().Msg(fmt.Sprint(args...))
}

// Errorf logging.
func (l *Logger) Errorf(msg string, args ...interface{}) {
	l.logger.Error().Msgf(msg, args...)
}

// Warn logging.
func (l *Logger) Warn(args ...interface{}) {
	l.logger.Warn().Msg(fmt.Sprint(args...))
}

// Warnf logging.
func (l *Logger) Warnf(msg string, args ...interface{}) {
	l.logger.Warn().Msgf(msg, args...)
}

// Info logging.
func (l *Logger) Info(args ...interface{}) {
	l.logger.Info().Msg(fmt.Sprint(args...))
}

// Infof logging.
func (l *Logger) Infof(msg string, args ...interface{}) {
	l.logger.Info().Msgf(msg, args...)
}

// Debug logging.
func (l *Logger) Debug(args ...interface{}) {
	l.logger.Debug().Msg(fmt.Sprint(args...))
}

// Debugf logging.
func (l *Logger) Debugf(msg string, args ...interface{}) {
	l.logger.Debug().Msgf(msg, args...)
}

// Level return the logging level.
func (l *Logger) Level() log.Level {
	return l.level
}
