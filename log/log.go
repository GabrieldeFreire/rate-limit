package log

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

var lock = &sync.Mutex{}

const (
	LevelTrace = slog.Level(-8)
	LevelFatal = slog.Level(12)
)

type Leveler interface {
	Level() slog.Level
}

type Logger struct {
	*slog.Logger
}

func (l *Logger) Fatal(msg string, args ...interface{}) {
	ctx := context.Background()
	l.Log(ctx, LevelFatal, msg, args...)
}

var logger *Logger

func GetLogger() *Logger {
	if logger == nil {
		lock.Lock()
		defer lock.Unlock()
		if logger == nil {
			logger = &Logger{Logger: getLogger()}
		}
	}

	return logger
}

var LevelNames = map[slog.Leveler]string{
	LevelFatal: "FATAL",
}

func getLogger() *slog.Logger {
	replace := func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.SourceKey {
			source := a.Value.Any().(*slog.Source)
			source.File = filepath.Base(source.File)
			source.Function = filepath.Base(source.Function)
		}
		if a.Key == slog.LevelKey {
			level := a.Value.Any().(slog.Level)
			levelLabel, exists := LevelNames[level]
			if !exists {
				levelLabel = level.String()
			}

			a.Value = slog.StringValue(levelLabel)
		}
		return a
	}

	opts := &slog.HandlerOptions{
		AddSource:   true,
		Level:       getLevelFromEnvVar(),
		ReplaceAttr: replace,
	}

	var handler slog.Handler = slog.NewJSONHandler(os.Stdout, opts)

	logger := slog.New(handler)
	return logger
}

func getLevelFromEnvVar() slog.Level {
	logLevelString := os.Getenv("LOG_LEVEL")

	var logLevel slog.Level
	err := logLevel.UnmarshalText([]byte(logLevelString))
	if err != nil {
		return slog.LevelDebug
	}
	return logLevel
}
