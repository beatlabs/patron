// Package std is the implementation of the logger interface with the standard log package.
package std

import (
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"

	patronLog "github.com/beatlabs/patron/log"
)

var levelMap = map[patronLog.Level]string{
	patronLog.DebugLevel: "DBG",
	patronLog.InfoLevel:  "INF",
	patronLog.WarnLevel:  "WRN",
	patronLog.ErrorLevel: "ERR",
	patronLog.FatalLevel: "FTL",
	patronLog.PanicLevel: "PNC",
}

// Logger implementation of the std log.
type Logger struct {
	level      patronLog.Level
	fields     map[string]interface{}
	fieldsLine string
	out        io.Writer
	debug      *log.Logger
	info       *log.Logger
	warn       *log.Logger
	error      *log.Logger
	panic      *log.Logger
	fatal      *log.Logger
}

// New constructor.
func New(out io.Writer, lvl patronLog.Level, fields map[string]interface{}) *Logger {
	fieldsLine := createFieldsLine(fields)

	return &Logger{
		out:        out,
		debug:      createLogger(out, patronLog.DebugLevel, fieldsLine),
		info:       createLogger(out, patronLog.InfoLevel, fieldsLine),
		warn:       createLogger(out, patronLog.WarnLevel, fieldsLine),
		error:      createLogger(out, patronLog.ErrorLevel, fieldsLine),
		panic:      createLogger(out, patronLog.PanicLevel, fieldsLine),
		fatal:      createLogger(out, patronLog.FatalLevel, fieldsLine),
		level:      lvl,
		fields:     fields,
		fieldsLine: fieldsLine,
	}
}

func createFieldsLine(fields map[string]interface{}) string {
	if len(fields) == 0 {
		return ""
	}

	// always return the fields in the same order
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	sb := strings.Builder{}

	for _, key := range keys {
		sb.WriteString(key)
		sb.WriteRune('=')
		sb.WriteString(fmt.Sprintf("%v", fields[key]))
		sb.WriteRune(' ')
	}

	return sb.String()
}

func createLogger(out io.Writer, lvl patronLog.Level, fieldLine string) *log.Logger {
	logger := log.New(out, levelMap[lvl]+" "+fieldLine, log.LstdFlags|log.Lmicroseconds|log.LUTC|log.Lmsgprefix|log.Lshortfile)
	return logger
}

// Sub returns a sub logger with additional fields.
func (l *Logger) Sub(fields map[string]interface{}) patronLog.Logger {
	for key, value := range l.fields {
		fields[key] = value
	}

	return New(l.out, l.level, fields)
}

// Fatal logging.
func (l *Logger) Fatal(args ...interface{}) {
	if !l.shouldLog(patronLog.FatalLevel) {
		return
	}

	output(l.fatal, args...)
	os.Exit(1)
}

// Fatalf logging.
func (l *Logger) Fatalf(msg string, args ...interface{}) {
	if !l.shouldLog(patronLog.FatalLevel) {
		return
	}

	outputf(l.fatal, msg, args...)
	os.Exit(1)
}

// Panic logging.
func (l *Logger) Panic(args ...interface{}) {
	if !l.shouldLog(patronLog.PanicLevel) {
		return
	}

	panic(output(l.panic, args...))
}

// Panicf logging.
func (l *Logger) Panicf(msg string, args ...interface{}) {
	if !l.shouldLog(patronLog.PanicLevel) {
		return
	}

	panic(outputf(l.panic, msg, args...))
}

// Error logging.
func (l *Logger) Error(args ...interface{}) {
	if !l.shouldLog(patronLog.ErrorLevel) {
		return
	}

	output(l.error, args...)
}

// Errorf logging.
func (l *Logger) Errorf(msg string, args ...interface{}) {
	if !l.shouldLog(patronLog.ErrorLevel) {
		return
	}

	outputf(l.error, msg, args...)
}

// Warn logging.
func (l *Logger) Warn(args ...interface{}) {
	if !l.shouldLog(patronLog.WarnLevel) {
		return
	}

	output(l.warn, args...)
}

// Warnf logging.
func (l *Logger) Warnf(msg string, args ...interface{}) {
	if !l.shouldLog(patronLog.WarnLevel) {
		return
	}

	outputf(l.warn, msg, args...)
}

// Info logging.
func (l *Logger) Info(args ...interface{}) {
	if !l.shouldLog(patronLog.InfoLevel) {
		return
	}

	output(l.info, args...)
}

// Infof logging.
func (l *Logger) Infof(msg string, args ...interface{}) {
	if !l.shouldLog(patronLog.InfoLevel) {
		return
	}

	outputf(l.info, msg, args...)
}

// Debug logging.
func (l *Logger) Debug(args ...interface{}) {
	if !l.shouldLog(patronLog.DebugLevel) {
		return
	}

	output(l.debug, args...)
}

// Debugf logging.
func (l *Logger) Debugf(msg string, args ...interface{}) {
	if !l.shouldLog(patronLog.DebugLevel) {
		return
	}

	outputf(l.debug, msg, args...)
}

// Level of the logging.
func (l *Logger) Level() patronLog.Level {
	return l.level
}

func (l *Logger) shouldLog(lvl patronLog.Level) bool {
	return patronLog.LevelOrder(l.level) <= patronLog.LevelOrder(lvl)
}

func output(logger *log.Logger, args ...interface{}) string {
	msg := fmt.Sprint(args...)
	_ = logger.Output(3, msg)
	return msg
}

func outputf(logger *log.Logger, msg string, args ...interface{}) string {
	fmtMsg := fmt.Sprintf(msg, args...)
	_ = logger.Output(3, fmtMsg)
	return fmtMsg
}
