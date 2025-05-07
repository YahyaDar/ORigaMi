// Copyright (c) 2025 Yahya Qadeer Dar. All rights reserved.
// Use of this source code is governed by an Apache 2.0 license that can be found in the LICENSE file.

// Package log provides a comprehensive logging system for the ORigaMi ORM.
// It offers structured logging with multiple levels, formats, and outputs.
package log

import (
	"context"
	"io"
	"time"
)

// Level represents the severity level of a log message
type Level int

const (
	// TraceLevel represents extremely detailed information
	TraceLevel Level = iota
	// DebugLevel represents debug information
	DebugLevel
	// InfoLevel represents general operational information
	InfoLevel
	// WarnLevel represents non-critical issues that should be addressed
	WarnLevel
	// ErrorLevel represents errors that might still allow the application to continue
	ErrorLevel
	// FatalLevel represents severe errors that prevent further execution
	FatalLevel
	// SilentLevel disables all logging when used
	SilentLevel
)

// String returns the string representation of the log level
func (l Level) String() string {
	switch l {
	case TraceLevel:
		return "TRACE"
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	case SilentLevel:
		return "SILENT"
	default:
		return "UNKNOWN"
	}
}

// Color returns ANSI color code for the log level
func (l Level) Color() string {
	switch l {
	case TraceLevel:
		return "\033[37m" // White
	case DebugLevel:
		return "\033[36m" // Cyan
	case InfoLevel:
		return "\033[32m" // Green
	case WarnLevel:
		return "\033[33m" // Yellow
	case ErrorLevel:
		return "\033[31m" // Red
	case FatalLevel:
		return "\033[35m" // Magenta
	default:
		return "\033[0m" // Default
	}
}

// Field represents a key-value pair in a structured log entry
type Field struct {
	Key   string
	Value interface{}
}

// F creates a new log field
func F(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Fields is a collection of Field objects
type Fields []Field

// Logger is the main interface for the logging system
type Logger interface {
	// Standard logging methods with variadic fields
	Trace(msg string, fields ...Field)
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)

	// Context-aware logging methods
	TraceContext(ctx context.Context, msg string, fields ...Field)
	DebugContext(ctx context.Context, msg string, fields ...Field)
	InfoContext(ctx context.Context, msg string, fields ...Field)
	WarnContext(ctx context.Context, msg string, fields ...Field)
	ErrorContext(ctx context.Context, msg string, fields ...Field)
	FatalContext(ctx context.Context, msg string, fields ...Field)

	// Formatted logging methods
	Tracef(format string, args ...interface{})
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})

	// With methods to create derived loggers
	WithField(key string, value interface{}) Logger
	WithFields(fields ...Field) Logger
	WithContext(ctx context.Context) Logger
	WithError(err error) Logger
	WithLevel(level Level) Logger

	// Configuration methods
	SetLevel(level Level)
	GetLevel() Level
	SetFormatter(formatter Formatter)
	AddHook(hook Hook)
	AddWriter(writer io.Writer)
	SetOutput(output io.Writer)
}

// Formatter defines the interface for formatting log entries
type Formatter interface {
	// Format converts a log entry into a byte slice
	Format(entry *Entry) ([]byte, error)
}

// Hook defines the interface for log hooks
type Hook interface {
	// Levels returns the levels this hook should be fired for
	Levels() []Level
	
	// Fire executes the hook for a log entry
	Fire(entry *Entry) error
}

// Entry represents a single log entry
type Entry struct {
	// Logger reference to the logger that created the entry
	Logger Logger
	
	// Time when the entry was created
	Time time.Time
	
	// Level severity level
	Level Level
	
	// Message log message
	Message string
	
	// Fields structured data associated with the entry
	Fields Fields
	
	// Context associated with the log entry
	Context context.Context
	
	// Error associated with the log entry
	Error error
	
	// Caller information about where the log was called from
	Caller *CallerInfo
}

// CallerInfo contains information about the source code location of a log call
type CallerInfo struct {
	File     string
	Line     int
	Function string
	Package  string
}

// Option represents a configuration option for the logger
type Option func(*LoggerConfig)

// LoggerConfig holds the configuration for a logger
type LoggerConfig struct {
	// Level is the minimum severity level to log
	Level Level
	
	// Output destinations for log entries
	Outputs []io.Writer
	
	// Formatter to use for log entries
	Formatter Formatter
	
	// Hooks to fire for log entries
	Hooks []Hook
	
	// ReportCaller determines if the caller information should be included
	ReportCaller bool
	
	// CallerSkipFrames is the number of frames to skip when reporting caller
	CallerSkipFrames int
	
	// EnableColors enables ANSI color codes in the output
	EnableColors bool
	
	// EnableAsync enables asynchronous logging
	EnableAsync bool
	
	// AsyncBufferSize is the channel buffer size for async logging
	AsyncBufferSize int
	
	// ExitFunc is the function to call on Fatal-level messages
	ExitFunc func(int)
	
	// TimeFormat is the format for time stamps
	TimeFormat string
	
	// EnableSampling enables log sampling to reduce volume
	EnableSampling bool
	
	// SampleRate defines the sampling rate (e.g. 0.1 = 10%)
	SampleRate float64
}

// Clock represents a time source
type Clock interface {
	Now() time.Time
}

// SystemClock is the standard clock using system time
type SystemClock struct{}

// Now returns the current system time
func (c *SystemClock) Now() time.Time {
	return time.Now()
}
