// Package logging is the central logging package of the CLI. It holds our custom log formatters for zap.
package logging

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewProductionLogger returns a logger that prints Debug, Info, and Warn messages to stdout and the rest to stderr.
func NewProductionLogger() *zap.SugaredLogger {
	encoder := zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
		// These strings are meaningless - they just need to be non-empty for the console encoder.
		MessageKey: "M",
		LevelKey:   "L",
		EncodeLevel: func(lvl zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
			// Anything other than "info" logs will have a capitalized level prefix.
			if lvl != zapcore.InfoLevel {
				zapcore.CapitalColorLevelEncoder(lvl, enc)
			}
		},
	})

	infoLevels := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level == zapcore.InfoLevel
	})

	errorLevels := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return !infoLevels(level) && level != zapcore.DebugLevel
	})

	return zap.New(zapcore.NewTee(
		zapcore.NewCore(encoder, zapcore.Lock(os.Stdout), infoLevels),
		zapcore.NewCore(encoder, zapcore.Lock(os.Stderr), errorLevels),
	)).Sugar()
}

// NewDebugLogger is similar to our production logger, however it also includes debug output & stacktraces
func NewDebugLogger() *zap.SugaredLogger {
	encoder := zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
		// These strings are meaningless - they just need to be non-empty for the console encoder.
		LevelKey:      "L",
		MessageKey:    "M",
		NameKey:       "N",
		StacktraceKey: "S",
		TimeKey:       "T",
		EncodeLevel:   zapcore.CapitalColorLevelEncoder,
		EncodeTime:    zapcore.ISO8601TimeEncoder,
	})

	infoLevels := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level == zapcore.InfoLevel
	})

	errorLevels := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return !infoLevels(level)
	})

	return zap.New(zapcore.NewTee(
		zapcore.NewCore(encoder, zapcore.Lock(os.Stdout), infoLevels),
		zapcore.NewCore(encoder, zapcore.Lock(os.Stderr), errorLevels),
	)).WithOptions(
		zap.Development(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	).Sugar()
}
