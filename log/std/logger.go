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
	return NewWithFlags(out, lvl, fields, log.LstdFlags|log.Lmicroseconds|log.LUTC|log.Lmsgprefix)
}

// NewWithFlags constructor.
func NewWithFlags(out io.Writer, lvl patronLog.Level, fields map[string]interface{}, flags int) *Logger {
	fieldsLine := createFieldsLine(fields)

	return &Logger{
		debug:      createLogger(out, patronLog.DebugLevel, fieldsLine, flags),
		info:       createLogger(out, patronLog.InfoLevel, fieldsLine, flags),
		warn:       createLogger(out, patronLog.WarnLevel, fieldsLine, flags),
		error:      createLogger(out, patronLog.ErrorLevel, fieldsLine, flags),
		panic:      createLogger(out, patronLog.PanicLevel, fieldsLine, flags),
		fatal:      createLogger(out, patronLog.FatalLevel, fieldsLine, flags),
		level:      lvl,
		fields:     fields,
		fieldsLine: fieldsLine,
		out:        out,
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
		writeValue(&sb, fmt.Sprintf("%v", fields[key]))
		sb.WriteRune(' ')
	}

	return sb.String()
}

func createLogger(out io.Writer, lvl patronLog.Level, fieldLine string, flags int) *log.Logger {
	logger := log.New(out, "lvl="+levelMap[lvl]+" "+fieldLine, flags)
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
	patronLog.IncreaseFatalCounter()
	if !l.shouldLog(patronLog.FatalLevel) {
		return
	}

	output(l.fatal, args...)
	os.Exit(1)
}

// Fatalf logging.
func (l *Logger) Fatalf(msg string, args ...interface{}) {
	patronLog.IncreaseFatalCounter()
	if !l.shouldLog(patronLog.FatalLevel) {
		return
	}

	outputf(l.fatal, msg, args...)
	os.Exit(1)
}

// Panic logging.
func (l *Logger) Panic(args ...interface{}) {
	patronLog.IncreasePanicCounter()
	if !l.shouldLog(patronLog.PanicLevel) {
		return
	}

	panic(output(l.panic, args...))
}

// Panicf logging.
func (l *Logger) Panicf(msg string, args ...interface{}) {
	patronLog.IncreasePanicCounter()
	if !l.shouldLog(patronLog.PanicLevel) {
		return
	}

	panic(outputf(l.panic, msg, args...))
}

// Error logging.
func (l *Logger) Error(args ...interface{}) {
	patronLog.IncreaseErrorCounter()
	if !l.shouldLog(patronLog.ErrorLevel) {
		return
	}

	output(l.error, args...)
}

// Errorf logging.
func (l *Logger) Errorf(msg string, args ...interface{}) {
	patronLog.IncreaseErrorCounter()
	if !l.shouldLog(patronLog.ErrorLevel) {
		return
	}

	outputf(l.error, msg, args...)
}

// Warn logging.
func (l *Logger) Warn(args ...interface{}) {
	patronLog.IncreaseWarnCounter()
	if !l.shouldLog(patronLog.WarnLevel) {
		return
	}

	output(l.warn, args...)
}

// Warnf logging.
func (l *Logger) Warnf(msg string, args ...interface{}) {
	patronLog.IncreaseWarnCounter()
	if !l.shouldLog(patronLog.WarnLevel) {
		return
	}

	outputf(l.warn, msg, args...)
}

// Info logging.
func (l *Logger) Info(args ...interface{}) {
	patronLog.IncreaseInfoCounter()
	if !l.shouldLog(patronLog.InfoLevel) {
		return
	}

	output(l.info, args...)
}

// Infof logging.
func (l *Logger) Infof(msg string, args ...interface{}) {
	patronLog.IncreaseInfoCounter()
	if !l.shouldLog(patronLog.InfoLevel) {
		return
	}

	outputf(l.info, msg, args...)
}

// Debug logging.
func (l *Logger) Debug(args ...interface{}) {
	patronLog.IncreaseDebugCounter()
	if !l.shouldLog(patronLog.DebugLevel) {
		return
	}

	output(l.debug, args...)
}

// Debugf logging.
func (l *Logger) Debugf(msg string, args ...interface{}) {
	patronLog.IncreaseDebugCounter()
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
	sb := strings.Builder{}
	writeValue(&sb, fmt.Sprint(args...))
	_ = logger.Output(4, fmt.Sprintf("message=%s", sb.String()))
	return sb.String()
}

func outputf(logger *log.Logger, msg string, args ...interface{}) string {
	sb := strings.Builder{}
	writeValue(&sb, fmt.Sprintf(msg, args...))
	_ = logger.Output(4, fmt.Sprintf("message=%s", sb.String()))
	return sb.String()
}

func writeValue(buf *strings.Builder, s string) {
	needsQuotes := strings.IndexFunc(s, func(r rune) bool {
		return r <= ' ' || r == '=' || r == '"'
	}) != -1

	if needsQuotes {
		buf.WriteByte('"')
	}

	start := 0
	for i, r := range s {
		if r >= 0x20 && r != '\\' && r != '"' {
			continue
		}

		if start < i {
			buf.WriteString(s[start:i])
		}

		switch r {
		case '"':
			buf.WriteString(`\"`)
		case '\\':
			buf.WriteString(`\\`)
		case '\n':
			buf.WriteString("\\n")
		case '\r':
			buf.WriteString("\\r")
		case '\t':
			buf.WriteString("\\t")
		}

		start = i + 1
	}

	if start < len(s) {
		buf.WriteString(s[start:])
	}

	if needsQuotes {
		buf.WriteByte('"')
	}
}
