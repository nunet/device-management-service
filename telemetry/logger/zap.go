// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

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
