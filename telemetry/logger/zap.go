package logger

import (
	"sync"

	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.uber.org/zap"
)

var (
	once   sync.Once
	logger *otelzap.Logger
)

type Logger struct {
	*zap.Logger
}

// New creates a new Logger with the specified package name.
// It assumes that the logger has already been initialized elsewhere.
func New(pkg string) *Logger {
	Log := &Logger{
		Logger: zap.L(), // Use the globally initialized zap logger
	}

	Log.Logger = Log.Logger.With(
		zap.String("package", pkg),
	)

	return Log
}

func OtelZapLogger(pkg string) *otelzap.Logger {
	once.Do(func() {
		l := New(pkg)
		logger = otelzap.New(l.Logger)
	})
	return logger
}
