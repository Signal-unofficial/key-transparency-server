//
// Copyright 2025 Signal Messenger, LLC
// SPDX-License-Identifier: AGPL-3.0-only
//

package util

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

type Logger struct {
	logger *zerolog.Logger
}

var instance *Logger

func SetLoggerInstance(l *zerolog.Logger) {
	instance = &Logger{l}
}

func Log() *Logger {
	if instance == nil {
		instance = _defaultLogger()
		instance.Warnf("default logger in use. SetLoggerInstance() should be called first")
	}
	return instance
}

func _defaultLogger() *Logger {
	zeroLogLogger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).With().Caller().Timestamp().Logger()

	return &Logger{&zeroLogLogger}
}

func (l *Logger) Infof(format string, v ...interface{}) {
	l.logger.Info().Msgf(format, v...)
}

func (l *Logger) Warnf(format string, v ...interface{}) {
	l.logger.Warn().Msgf(format, v...)
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	l.logger.Error().Msgf(format, v...)
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.logger.Fatal().Msgf(format, v...)
}
