// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package log

import (
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/luxfi/log"
)

type Logger struct {
	log.Logger
}

// InitLogger initializes logger with alias and sets the log level and format with the original [os.StdErr] interface
// along with the context logger.
func InitLogger(alias string, level string, jsonFormat bool, writer io.Writer) (Logger, error) {
	// Parse log level
	logLevel, err := ParseLogLevel(level)
	if err != nil {
		return Logger{}, err
	}

	var handler slog.Handler
	if jsonFormat {
		handler = log.JSONHandlerWithLevel(writer, logLevel)
		// Add context to the handler
		handler = handler.WithAttrs([]slog.Attr{
			slog.String("logger", fmt.Sprintf("%s Chain", alias)),
		})
	} else {
		useColor := false
		chainStr := fmt.Sprintf("<%s Chain> ", alias)
		termHandler := log.NewTerminalHandlerWithLevel(writer, logLevel, useColor)
		// TODO: Add chain prefix to terminal handler when API is available
		handler = termHandler
		_ = chainStr
	}

	// Create logger
	logger := log.Root().With("chain", alias)
	
	c := Logger{
		Logger: logger,
	}

	log.SetDefault(c.Logger)
	return c, nil
}

// SetLogLevel sets the log level of initialized log handler.
func (l *Logger) SetLogLevel(level string) error {
	// Parse and set new log level
	logLevel, err := ParseLogLevel(level)
	if err != nil {
		return err
	}
	
	// Set the level on the logger
	l.SetLevel(logLevel)
	
	return nil
}

// ParseLogLevel parses a string log level
func ParseLogLevel(level string) (slog.Level, error) {
	switch strings.ToLower(level) {
	case "trace":
		return log.LevelTrace, nil
	case "debug":
		return log.LevelDebug, nil
	case "info":
		return log.LevelInfo, nil
	case "warn", "warning":
		return log.LevelWarn, nil
	case "error":
		return log.LevelError, nil
	case "crit", "critical":
		return log.LevelCrit, nil
	default:
		return log.LevelInfo, fmt.Errorf("unknown log level: %s", level)
	}
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