// Copyright (c) 2025 Yahya Qadeer Dar. All rights reserved.
// Use of this source code is governed by an Apache 2.0 license that can be found in the LICENSE file.

package log

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// DefaultLogger is the standard implementation of Logger
type DefaultLogger struct {
	// Configuration for this logger
	config LoggerConfig
	
	// Mutex for thread safety
	mu sync.Mutex
	
	// Default fields to include in all log entries
	defaultFields Fields
	
	// Default context to include in all log entries
	defaultContext context.Context
	
	// Channel for async logging
	entryChan chan *Entry
	
	// WaitGroup to wait for all log entries to be processed
	wg sync.WaitGroup
	
	// Clock for timestamp generation
	clock Clock
}

// NewLogger creates a new logger with the given options
func NewLogger(options ...Option) *DefaultLogger {
	// Default configuration
	cfg := LoggerConfig{
		Level:           InfoLevel,
		Outputs:         []io.Writer{os.Stdout},
		Formatter:       NewTextFormatter(),
		Hooks:           make([]Hook, 0),
		ReportCaller:    true,
		CallerSkipFrames: 2,
		EnableColors:    true,
		EnableAsync:     false,
		AsyncBufferSize: 1000,
		ExitFunc:        os.Exit,
		TimeFormat:      "2006-01-02 15:04:05.000",
		EnableSampling:  false,
		SampleRate:      1.0,
	}
	
	// Apply options
	for _, option := range options {
		option(&cfg)
	}
	
	logger := &DefaultLogger{
		config:        cfg,
		defaultFields: Fields{},
		defaultContext: context.Background(),
		clock:         &SystemClock{},
	}
	
	// Initialize async logging if enabled
	if cfg.EnableAsync {
		logger.entryChan = make(chan *Entry, cfg.AsyncBufferSize)
		go logger.processEntries()
	}
	
	return logger
}

// processEntries handles async log entries
func (l *DefaultLogger) processEntries() {
	for entry := range l.entryChan {
		l.processEntry(entry)
		l.wg.Done()
	}
}

// processEntry formats and writes a log entry
func (l *DefaultLogger) processEntry(entry *Entry) {
	data, err := l.config.Formatter.Format(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting log entry: %v\n", err)
		return
	}
	
	// Write to all outputs
	for _, output := range l.config.Outputs {
		if _, err := output.Write(data); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing log entry: %v\n", err)
		}
	}
	
	// Fire hooks
	for _, hook := range l.config.Hooks {
		for _, level := range hook.Levels() {
			if level == entry.Level {
				if err := hook.Fire(entry); err != nil {
					fmt.Fprintf(os.Stderr, "Error firing hook: %v\n", err)
				}
				break
			}
		}
	}
	
	// Handle fatal logs
	if entry.Level == FatalLevel && l.config.ExitFunc != nil {
		l.config.ExitFunc(1)
	}
}

// log creates a log entry and processes it
func (l *DefaultLogger) log(level Level, ctx context.Context, msg string, fields ...Field) {
	// Skip logging if level is not enabled
	if level < l.config.Level {
		return
	}
	
	// Apply sampling if enabled
	if l.config.EnableSampling && l.config.SampleRate < 1.0 {
		if l.config.SampleRate <= 0 {
			return
		}
		if level > ErrorLevel { // Don't sample error and fatal logs
			if randFloat() > l.config.SampleRate {
				return
			}
		}
	}
	
	// Merge default fields and context
	mergedFields := make(Fields, len(l.defaultFields)+len(fields))
	copy(mergedFields, l.defaultFields)
	copy(mergedFields[len(l.defaultFields):], fields)
	
	mergedCtx := l.defaultContext
	if ctx != nil {
		mergedCtx = ctx
	}
	
	// Create the entry
	entry := &Entry{
		Logger:  l,
		Time:    l.clock.Now(),
		Level:   level,
		Message: msg,
		Fields:  mergedFields,
		Context: mergedCtx,
	}
	
	// Add caller information if enabled
	if l.config.ReportCaller {
		entry.Caller = l.getCaller()
	}
	
	// Async or sync processing
	if l.config.EnableAsync {
		l.wg.Add(1)
		select {
		case l.entryChan <- entry:
			// Entry added to channel
		default:
			// Channel is full
			l.wg.Done()
			fmt.Fprintf(os.Stderr, "Logger channel full, dropping log entry: %s\n", msg)
		}
	} else {
		l.processEntry(entry)
	}
}

// getCaller returns information about the calling function
func (l *DefaultLogger) getCaller() *CallerInfo {
	// Skip frames to get to the actual caller
	skip := l.config.CallerSkipFrames
	
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return &CallerInfo{
			File:     "unknown",
			Line:     0,
			Function: "unknown",
			Package:  "unknown",
		}
	}
	
	// Get function name
	fn := runtime.FuncForPC(pc)
	funcName := "unknown"
	pkgName := "unknown"
	
	if fn != nil {
		funcName = fn.Name()
		// Split package and function name
		if idx := strings.LastIndex(funcName, "."); idx >= 0 {
			pkgName = funcName[:idx]
			funcName = funcName[idx+1:]
		}
	}
	
	// Simplify file path
	if idx := strings.LastIndex(file, "/"); idx >= 0 {
		file = file[idx+1:]
	}
	
	return &CallerInfo{
		File:     file,
		Line:     line,
		Function: funcName,
		Package:  pkgName,
	}
}

// Flush ensures all log entries are written
func (l *DefaultLogger) Flush() {
	if l.config.EnableAsync {
		l.wg.Wait()
	}
}

// Close shuts down the logger
func (l *DefaultLogger) Close() error {
	if l.config.EnableAsync {
		close(l.entryChan)
		l.wg.Wait()
	}
	return nil
}

// Standard logging methods implementation

// Trace logs a message at the trace level
func (l *DefaultLogger) Trace(msg string, fields ...Field) {
	l.log(TraceLevel, nil, msg, fields...)
}

// Debug logs a message at the debug level
func (l *DefaultLogger) Debug(msg string, fields ...Field) {
	l.log(DebugLevel, nil, msg, fields...)
}

// Info logs a message at the info level
func (l *DefaultLogger) Info(msg string, fields ...Field) {
	l.log(InfoLevel, nil, msg, fields...)
}

// Warn logs a message at the warn level
func (l *DefaultLogger) Warn(msg string, fields ...Field) {
	l.log(WarnLevel, nil, msg, fields...)
}

// Error logs a message at the error level
func (l *DefaultLogger) Error(msg string, fields ...Field) {
	l.log(ErrorLevel, nil, msg, fields...)
}

// Fatal logs a message at the fatal level and then exits
func (l *DefaultLogger) Fatal(msg string, fields ...Field) {
	l.log(FatalLevel, nil, msg, fields...)
}

// Context-aware logging methods

// TraceContext logs a message with context at the trace level
func (l *DefaultLogger) TraceContext(ctx context.Context, msg string, fields ...Field) {
	l.log(TraceLevel, ctx, msg, fields...)
}

// DebugContext logs a message with context at the debug level
func (l *DefaultLogger) DebugContext(ctx context.Context, msg string, fields ...Field) {
	l.log(DebugLevel, ctx, msg, fields...)
}

// InfoContext logs a message with context at the info level
func (l *DefaultLogger) InfoContext(ctx context.Context, msg string, fields ...Field) {
	l.log(InfoLevel, ctx, msg, fields...)
}

// WarnContext logs a message with context at the warn level
func (l *DefaultLogger) WarnContext(ctx context.Context, msg string, fields ...Field) {
	l.log(WarnLevel, ctx, msg, fields...)
}

// ErrorContext logs a message with context at the error level
func (l *DefaultLogger) ErrorContext(ctx context.Context, msg string, fields ...Field) {
	l.log(ErrorLevel, ctx, msg, fields...)
}

// FatalContext logs a message with context at the fatal level and then exits
func (l *DefaultLogger) FatalContext(ctx context.Context, msg string, fields ...Field) {
	l.log(FatalLevel, ctx, msg, fields...)
}

// Formatted logging methods

// Tracef logs a formatted message at the trace level
func (l *DefaultLogger) Tracef(format string, args ...interface{}) {
	l.log(TraceLevel, nil, fmt.Sprintf(format, args...))
}

// Debugf logs a formatted message at the debug level
func (l *DefaultLogger) Debugf(format string, args ...interface{}) {
	l.log(DebugLevel, nil, fmt.Sprintf(format, args...))
}

// Infof logs a formatted message at the info level
func (l *DefaultLogger) Infof(format string, args ...interface{}) {
	l.log(InfoLevel, nil, fmt.Sprintf(format, args...))
}

// Warnf logs a formatted message at the warn level
func (l *DefaultLogger) Warnf(format string, args ...interface{}) {
	l.log(WarnLevel, nil, fmt.Sprintf(format, args...))
}

// Errorf logs a formatted message at the error level
func (l *DefaultLogger) Errorf(format string, args ...interface{}) {
	l.log(ErrorLevel, nil, fmt.Sprintf(format, args...))
}

// Fatalf logs a formatted message at the fatal level and then exits
func (l *DefaultLogger) Fatalf(format string, args ...interface{}) {
	l.log(FatalLevel, nil, fmt.Sprintf(format, args...))
}

// With methods to create derived loggers

// WithField returns a logger with the given field added
func (l *DefaultLogger) WithField(key string, value interface{}) Logger {
	return l.WithFields(Field{Key: key, Value: value})
}

// WithFields returns a logger with the given fields added
func (l *DefaultLogger) WithFields(fields ...Field) Logger {
	clone := l.clone()
	clone.defaultFields = append(clone.defaultFields, fields...)
	return clone
}

// WithContext returns a logger with the given context
func (l *DefaultLogger) WithContext(ctx context.Context) Logger {
	clone := l.clone()
	clone.defaultContext = ctx
	return clone
}

// WithError returns a logger with the given error added as a field
func (l *DefaultLogger) WithError(err error) Logger {
	if err == nil {
		return l
	}
	return l.WithField("error", err.Error())
}

// WithLevel returns a logger that only logs at the given level or above
func (l *DefaultLogger) WithLevel(level Level) Logger {
	clone := l.clone()
	clone.config.Level = level
	return clone
}

// Configuration methods

// SetLevel sets the minimum severity level to log
func (l *DefaultLogger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.config.Level = level
}

// GetLevel returns the current minimum severity level
func (l *DefaultLogger) GetLevel() Level {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.config.Level
}

// SetFormatter sets the formatter to use for log entries
func (l *DefaultLogger) SetFormatter(formatter Formatter) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.config.Formatter = formatter
}

// AddHook adds a hook to the logger
func (l *DefaultLogger) AddHook(hook Hook) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.config.Hooks = append(l.config.Hooks, hook)
}

// AddWriter adds an output destination for log entries
func (l *DefaultLogger) AddWriter(writer io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.config.Outputs = append(l.config.Outputs, writer)
}

// SetOutput sets the output destination for log entries (replacing existing outputs)
func (l *DefaultLogger) SetOutput(output io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.config.Outputs = []io.Writer{output}
}

// clone creates a copy of the logger
func (l *DefaultLogger) clone() *DefaultLogger {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	clone := &DefaultLogger{
		config:        l.config,
		defaultFields: make(Fields, len(l.defaultFields)),
		defaultContext: l.defaultContext,
		clock:         l.clock,
	}
	
	// Deep copy default fields
	copy(clone.defaultFields, l.defaultFields)
	
	// Share async channel if enabled
	if l.config.EnableAsync {
		clone.entryChan = l.entryChan
		// No need to copy waitgroup as it's process-wide
	}
	
	return clone
}

// randFloat returns a random float between 0 and 1
func randFloat() float64 {
	return float64(time.Now().UnixNano()%1000) / 1000
}
