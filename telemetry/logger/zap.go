package logger

import (
	"os"
	"sync"

	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"gitlab.com/nunet/device-management-service/internal/config"
)

var (
	err    error
	once   sync.Once
	logger *otelzap.Logger
)

type Logger struct {
	*zap.Logger
}

func (l *Logger) init() error {
	if _, debug := os.LookupEnv("NUNET_DEBUG"); debug || config.GetConfig().General.Debug {
		zapConfig := zap.NewDevelopmentConfig()
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		l.Logger, _ = zapConfig.Build()

	} else {
		l.Logger, err = zap.NewProduction()
	}

	return err
}

// New takes in a package to initialize the new Logger in.
func New(pkg string) *Logger {
	Log := &Logger{}
	err = Log.init()
	if err != nil {
		panic(err)
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
