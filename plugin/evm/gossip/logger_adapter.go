// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package gossip

import (
	"context"
	"log/slog"

	"github.com/luxfi/log"
)

// loggerAdapter adapts luxfi/log.Logger
type loggerAdapter struct {
	logger log.Logger
}

// NewLoggerAdapter creates a new logger adapter
func NewLoggerAdapter(logger log.Logger) log.Logger {
	return &loggerAdapter{logger: logger}
}

// Write implements io.Writer
func (l *loggerAdapter) Write(p []byte) (n int, err error) {
	l.logger.Info(string(p))
	return len(p), nil
}

// Stop implements log.Logger
func (l *loggerAdapter) Stop() {
	l.logger.Stop()
}

// StopOnPanic implements log.Logger
func (l *loggerAdapter) StopOnPanic() {
	l.logger.StopOnPanic()
}

// RecoverAndPanic implements log.Logger
func (l *loggerAdapter) RecoverAndPanic(f func()) {
	l.logger.RecoverAndPanic(f)
}

// RecoverAndExit implements log.Logger
func (l *loggerAdapter) RecoverAndExit(f func(), exit func()) {
	l.logger.RecoverAndExit(f, exit)
}

// Delegate all other methods to the underlying logger
func (l *loggerAdapter) With(ctx ...interface{}) log.Logger {
	return &loggerAdapter{logger: l.logger.With(ctx...)}
}

func (l *loggerAdapter) New(ctx ...interface{}) log.Logger {
	return &loggerAdapter{logger: l.logger.New(ctx...)}
}

func (l *loggerAdapter) Log(level slog.Level, msg string, ctx ...interface{}) {
	l.logger.Log(level, msg, ctx...)
}

func (l *loggerAdapter) Trace(msg string, ctx ...interface{}) {
	l.logger.Trace(msg, ctx...)
}

func (l *loggerAdapter) Debug(msg string, ctx ...interface{}) {
	l.logger.Debug(msg, ctx...)
}

func (l *loggerAdapter) Info(msg string, ctx ...interface{}) {
	l.logger.Info(msg, ctx...)
}

func (l *loggerAdapter) Warn(msg string, ctx ...interface{}) {
	l.logger.Warn(msg, ctx...)
}

func (l *loggerAdapter) Error(msg string, ctx ...interface{}) {
	l.logger.Error(msg, ctx...)
}

func (l *loggerAdapter) Crit(msg string, ctx ...interface{}) {
	l.logger.Crit(msg, ctx...)
}

func (l *loggerAdapter) Fatal(msg string, fields ...log.Field) {
	l.logger.Fatal(msg, fields...)
}

func (l *loggerAdapter) Verbo(msg string, fields ...log.Field) {
	l.logger.Verbo(msg, fields...)
}

func (l *loggerAdapter) WithFields(fields ...log.Field) log.Logger {
	return &loggerAdapter{logger: l.logger.WithFields(fields...)}
}

func (l *loggerAdapter) WithOptions(opts ...log.Option) log.Logger {
	return &loggerAdapter{logger: l.logger.WithOptions(opts...)}
}

func (l *loggerAdapter) SetLevel(level slog.Level) {
	l.logger.SetLevel(level)
}

func (l *loggerAdapter) GetLevel() slog.Level {
	return l.logger.GetLevel()
}

func (l *loggerAdapter) EnabledLevel(lvl slog.Level) bool {
	return l.logger.EnabledLevel(lvl)
}

func (l *loggerAdapter) WriteLog(level slog.Level, msg string, attrs ...any) {
	l.logger.WriteLog(level, msg, attrs...)
}

func (l *loggerAdapter) Enabled(ctx context.Context, level slog.Level) bool {
	return l.logger.Enabled(ctx, level)
}

func (l *loggerAdapter) Handler() slog.Handler {
	return l.logger.Handler()
}
