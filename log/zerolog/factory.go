package zerolog

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"

	"github.com/beatlabs/patron/log"
	"github.com/rs/zerolog"
)

// Create creates a zerolog factory with default settings.
func Create(lvl log.Level) log.FactoryFunc {
	zerolog.LevelFieldName = "lvl"
	zerolog.MessageFieldName = "msg"
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zl := zerolog.New(os.Stderr).With().Timestamp().Logger().Hook(sourceHook{skip: 7})
	return func(f map[string]interface{}) log.Logger {
		return NewLogger(&zl, lvl, f)
	}
}

type sourceHook struct {
	skip int
}

func (sh sourceHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	k, v, ok := sourceFields(sh.skip)
	if !ok {
		return
	}
	e.Str(k, v)
}

func sourceFields(skip int) (key string, src string, ok bool) {
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
