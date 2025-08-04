// Copyright (C) 2024, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package logging

import (
	"log"
	"os"
)

// Logger interface for logging
type Logger interface {
	Info(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
}

// DefaultLogger is a simple logger implementation
type DefaultLogger struct {
	level  string
	logger *log.Logger
}

// NewLogger creates a new logger
func NewLogger(level string) Logger {
	return &DefaultLogger{
		level:  level,
		logger: log.New(os.Stdout, "[LUX-SDK] ", log.LstdFlags),
	}
}

// NewNoop creates a no-op logger for testing
func NewNoop() Logger {
	return &NoopLogger{}
}

// Info logs an info message
func (l *DefaultLogger) Info(msg string, args ...interface{}) {
	if len(args) > 0 {
		l.logger.Printf("[INFO] %s %v", msg, args)
	} else {
		l.logger.Printf("[INFO] %s", msg)
	}
}

// Debug logs a debug message
func (l *DefaultLogger) Debug(msg string, args ...interface{}) {
	if l.level == "debug" {
		if len(args) > 0 {
			l.logger.Printf("[DEBUG] %s %v", msg, args)
		} else {
			l.logger.Printf("[DEBUG] %s", msg)
		}
	}
}

// Error logs an error message
func (l *DefaultLogger) Error(msg string, args ...interface{}) {
	if len(args) > 0 {
		l.logger.Printf("[ERROR] %s %v", msg, args)
	} else {
		l.logger.Printf("[ERROR] %s", msg)
	}
}

// Warn logs a warning message
func (l *DefaultLogger) Warn(msg string, args ...interface{}) {
	if len(args) > 0 {
		l.logger.Printf("[WARN] %s %v", msg, args)
	} else {
		l.logger.Printf("[WARN] %s", msg)
	}
}

// NoopLogger is a logger that does nothing
type NoopLogger struct{}

func (n *NoopLogger) Info(msg string, args ...interface{})  {}
func (n *NoopLogger) Debug(msg string, args ...interface{}) {}
func (n *NoopLogger) Error(msg string, args ...interface{}) {}
func (n *NoopLogger) Warn(msg string, args ...interface{})  {}