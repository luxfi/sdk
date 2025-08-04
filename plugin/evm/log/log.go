// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package log

import (
	"context"
	"fmt"
	"io"
	stdlog "log/slog"
	"runtime"
	"strings"

	ethlog "github.com/luxfi/geth/log"
	"github.com/luxfi/evm/log"
	"golang.org/x/exp/slog"
)

type Logger struct {
	ethlog.Logger

	logLevel *slog.LevelVar
}

// InitLogger initializes logger with alias and sets the log level and format with the original [os.StdErr] interface
// along with the context logger.
func InitLogger(alias string, level string, jsonFormat bool, writer io.Writer) (Logger, error) {
	logLevel := &slog.LevelVar{}

	var handler slog.Handler
	if jsonFormat {
		chainStr := fmt.Sprintf("%s Chain", alias)
		handler = log.JSONHandlerWithLevel(writer, logLevel)
		handler = &addContext{Handler: handler, logger: chainStr}
	} else {
		useColor := false
		chainStr := fmt.Sprintf("<%s Chain> ", alias)
		termHandler := log.NewTerminalHandlerWithLevel(writer, logLevel, useColor)
		termHandler.Prefix = func(r slog.Record) string {
			file, line := getSource(r)
			if file != "" {
				return fmt.Sprintf("%s%s:%d ", chainStr, file, line)
			}
			return chainStr
		}
		handler = termHandler
	}

	// Create handler with wrapper to convert exp/slog to std slog
	wrapper := &slogWrapper{handler: handler}
	c := Logger{
		Logger:   ethlog.NewLogger(wrapper),
		logLevel: logLevel,
	}

	if err := c.SetLogLevel(level); err != nil {
		return Logger{}, err
	}
	ethlog.SetDefault(c.Logger)
	return c, nil
}

// SetLogLevel sets the log level of initialized log handler.
func (l *Logger) SetLogLevel(level string) error {
	// Set log level
	logLevel, err := log.LvlFromString(level)
	if err != nil {
		return err
	}
	l.logLevel.Set(logLevel)
	return nil
}

// locationTrims are trimmed for display to avoid unwieldy log lines.
var locationTrims = []string{
	"evm/",
}

func trimPrefixes(s string) string {
	for _, prefix := range locationTrims {
		idx := strings.LastIndex(s, prefix)
		if idx >= 0 {
			s = s[idx+len(prefix):]
		}
	}
	return s
}

func getSource(r slog.Record) (string, int) {
	frames := runtime.CallersFrames([]uintptr{r.PC})
	frame, _ := frames.Next()
	return trimPrefixes(frame.File), frame.Line
}

type addContext struct {
	slog.Handler

	logger string
}

func (a *addContext) Handle(ctx context.Context, r slog.Record) error {
	r.Add(slog.String("logger", a.logger))
	file, line := getSource(r)
	if file != "" {
		r.Add(slog.String("caller", fmt.Sprintf("%s:%d", file, line)))
	}
	return a.Handler.Handle(ctx, r)
}

// slogWrapper wraps exp/slog.Handler to implement std slog.Handler
type slogWrapper struct {
	handler slog.Handler
}

func (s *slogWrapper) Enabled(ctx context.Context, level stdlog.Level) bool {
	// Convert std slog level to exp slog level
	expLevel := slog.Level(level)
	return s.handler.Enabled(ctx, expLevel)
}

func (s *slogWrapper) Handle(ctx context.Context, r stdlog.Record) error {
	// Convert std slog record to exp slog record
	expRecord := slog.Record{
		Time: r.Time,
		Level: slog.Level(r.Level),
		Message: r.Message,
		PC: r.PC,
	}
	
	// Copy attributes
	r.Attrs(func(a stdlog.Attr) bool {
		expRecord.Add(convertAttr(a))
		return true
	})
	
	return s.handler.Handle(ctx, expRecord)
}

func (s *slogWrapper) WithAttrs(attrs []stdlog.Attr) stdlog.Handler {
	expAttrs := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		expAttrs[i] = convertAttr(a)
	}
	return &slogWrapper{handler: s.handler.WithAttrs(expAttrs)}
}

func (s *slogWrapper) WithGroup(name string) stdlog.Handler {
	return &slogWrapper{handler: s.handler.WithGroup(name)}
}

func convertAttr(a stdlog.Attr) slog.Attr {
	return slog.Attr{
		Key: a.Key,
		Value: convertValue(a.Value),
	}
}

func convertValue(v stdlog.Value) slog.Value {
	switch v.Kind() {
	case stdlog.KindBool:
		return slog.BoolValue(v.Bool())
	case stdlog.KindInt64:
		return slog.Int64Value(v.Int64())
	case stdlog.KindUint64:
		return slog.Uint64Value(v.Uint64())
	case stdlog.KindFloat64:
		return slog.Float64Value(v.Float64())
	case stdlog.KindString:
		return slog.StringValue(v.String())
	case stdlog.KindTime:
		return slog.TimeValue(v.Time())
	case stdlog.KindDuration:
		return slog.DurationValue(v.Duration())
	case stdlog.KindAny:
		return slog.AnyValue(v.Any())
	default:
		return slog.AnyValue(v.Any())
	}
}
