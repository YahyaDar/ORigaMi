// Copyright (c) 2025 Yahya Qaedeer Dar. All rights reserved.
// Use of this source code is governed by an Apache 2.0 license that can be found in the LICENSE file.

package log

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// TextFormatter formats log entries as human-readable text
type TextFormatter struct {
	// DisableColors disables ANSI color codes in the output
	DisableColors bool
	
	// DisableTimestamp disables the timestamp in the output
	DisableTimestamp bool
	
	// TimestampFormat sets the format for the timestamp
	TimestampFormat string
	
	// DisableCaller disables the caller information in the output
	DisableCaller bool
	
	// DisableLevel disables the level in the output
	DisableLevel bool
	
	// DisableQuote disables quoting of strings
	DisableQuote bool
	
	// SortFields sorts fields by key
	SortFields bool
	
	// FieldMap is a custom map for field names
	FieldMap map[string]string
	
	// PadLevelText pads the level text to a fixed width
	PadLevelText bool
}

// NewTextFormatter creates a new TextFormatter with default settings
func NewTextFormatter() *TextFormatter {
	return &TextFormatter{
		DisableColors:    false,
		DisableTimestamp: false,
		TimestampFormat:  "2006-01-02 15:04:05.000",
		DisableCaller:    false,
		DisableLevel:     false,
		DisableQuote:     false,
		SortFields:       true,
		PadLevelText:     true,
	}
}

// Format formats a log entry as text
func (f *TextFormatter) Format(entry *Entry) ([]byte, error) {
	var b *bytes.Buffer
	if entry.Logger != nil {
		if v, ok := entry.Logger.(*DefaultLogger); ok {
			if v.config.EnableColors && !f.DisableColors {
				b = getColorBuffer(entry.Level)
			}
		}
	}
	
	if b == nil {
		b = &bytes.Buffer{}
	}
	
	// Add timestamp
	if !f.DisableTimestamp {
		f.writeTimestamp(b, entry)
	}
	
	// Add log level
	if !f.DisableLevel {
		f.writeLevel(b, entry)
	}
	
	// Add caller info
	if !f.DisableCaller && entry.Caller != nil {
		f.writeCaller(b, entry)
	}
	
	// Add message
	b.WriteString(entry.Message)
	
	// Add fields
	if len(entry.Fields) > 0 {
		b.WriteString(" ")
		f.writeFields(b, entry)
	}
	
	// Add newline
	b.WriteByte('\n')
	
	// Add color reset if needed
	if entry.Logger != nil {
		if v, ok := entry.Logger.(*DefaultLogger); ok {
			if v.config.EnableColors && !f.DisableColors {
				b.WriteString("\033[0m") // Reset color
			}
		}
	}
	
	return b.Bytes(), nil
}

// writeTimestamp writes the timestamp to the buffer
func (f *TextFormatter) writeTimestamp(b *bytes.Buffer, entry *Entry) {
	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = "2006-01-02 15:04:05.000"
	}
	
	b.WriteString("[")
	b.WriteString(entry.Time.Format(timestampFormat))
	b.WriteString("] ")
}

// writeLevel writes the log level to the buffer
func (f *TextFormatter) writeLevel(b *bytes.Buffer, entry *Entry) {
	level := entry.Level.String()
	
	if f.PadLevelText {
		// Ensure all levels are the same width
		switch entry.Level {
		case TraceLevel:
			level = "TRACE"
		case DebugLevel:
			level = "DEBUG"
		case InfoLevel:
			level = "INFO "
		case WarnLevel:
			level = "WARN "
		case ErrorLevel:
			level = "ERROR"
		case FatalLevel:
			level = "FATAL"
		}
	}
	
	b.WriteString("[")
	b.WriteString(level)
	b.WriteString("] ")
}

// writeCaller writes the caller information to the buffer
func (f *TextFormatter) writeCaller(b *bytes.Buffer, entry *Entry) {
	if entry.Caller == nil {
		return
	}
	
	b.WriteString("[")
	b.WriteString(fmt.Sprintf("%s:%d", entry.Caller.File, entry.Caller.Line))
	b.WriteString("] ")
}

// writeFields writes the fields to the buffer
func (f *TextFormatter) writeFields(b *bytes.Buffer, entry *Entry) {
	if len(entry.Fields) == 0 {
		return
	}
	
	// Sort fields if configured
	fields := entry.Fields
	if f.SortFields {
		fields = sortFields(fields)
	}
	
	b.WriteString("{")
	
	for i, field := range fields {
		if i > 0 {
			b.WriteString(", ")
		}
		
		// Write key
		f.writeKey(b, field.Key)
		b.WriteString("=")
		
		// Write value
		f.writeValue(b, field.Value)
	}
	
	b.WriteString("}")
}

// writeKey writes a field key to the buffer
func (f *TextFormatter) writeKey(b *bytes.Buffer, key string) {
	// Use field map if configured
	if f.FieldMap != nil {
		if mappedKey, ok := f.FieldMap[key]; ok {
			key = mappedKey
		}
	}
	
	b.WriteString(key)
}

// writeValue writes a field value to the buffer
func (f *TextFormatter) writeValue(b *bytes.Buffer, value interface{}) {
	switch v := value.(type) {
	case string:
		if !f.DisableQuote && needsQuoting(v) {
			fmt.Fprintf(b, "%q", v)
		} else {
			b.WriteString(v)
		}
	case error:
		if v != nil {
			fmt.Fprintf(b, "%q", v.Error())
		} else {
			b.WriteString("null")
		}
	default:
		fmt.Fprint(b, v)
	}
}

// needsQuoting returns true if the string contains spaces or special characters
func needsQuoting(s string) bool {
	return strings.ContainsAny(s, " \t\r\n\"=:{},[]")
}

// JSONFormatter formats log entries as JSON
type JSONFormatter struct {
	// DisableTimestamp disables the timestamp in the output
	DisableTimestamp bool
	
	// TimestampFormat sets the format for the timestamp
	TimestampFormat string
	
	// DisableCaller disables the caller information in the output
	DisableCaller bool
	
	// PrettyPrint enables pretty-printing of JSON
	PrettyPrint bool
	
	// FieldMap is a custom map for field names
	FieldMap map[string]string
}

// NewJSONFormatter creates a new JSONFormatter with default settings
func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{
		DisableTimestamp: false,
		TimestampFormat:  time.RFC3339Nano,
		DisableCaller:    false,
		PrettyPrint:      false,
	}
}

// Format formats a log entry as JSON
func (f *JSONFormatter) Format(entry *Entry) ([]byte, error) {
	data := make(map[string]interface{})
	
	// Add timestamp
	if !f.DisableTimestamp {
		timestampFormat := f.TimestampFormat
		if timestampFormat == "" {
			timestampFormat = time.RFC3339Nano
		}
		data["time"] = entry.Time.Format(timestampFormat)
	}
	
	// Add log level
	data["level"] = entry.Level.String()
	
	// Add message
	data["msg"] = entry.Message
	
	// Add caller info
	if !f.DisableCaller && entry.Caller != nil {
		data["caller"] = fmt.Sprintf("%s:%d", entry.Caller.File, entry.Caller.Line)
		data["function"] = entry.Caller.Function
	}
	
	// Add fields
	for _, field := range entry.Fields {
		key := field.Key
		
		// Use field map if configured
		if f.FieldMap != nil {
			if mappedKey, ok := f.FieldMap[key]; ok {
				key = mappedKey
			}
		}
		
		data[key] = field.Value
	}
	
	// Marshal to JSON
	var encoded []byte
	var err error
	
	if f.PrettyPrint {
		encoded, err = json.MarshalIndent(data, "", "  ")
	} else {
		encoded, err = json.Marshal(data)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to marshal log entry to JSON: %w", err)
	}
	
	// Add newline
	encoded = append(encoded, '\n')
	
	return encoded, nil
}

// sortFields sorts fields by key
func sortFields(fields Fields) Fields {
	sorted := make(Fields, len(fields))
	copy(sorted, fields)
	
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Key < sorted[j].Key
	})
	
	return sorted
}

// getColorBuffer returns a buffer with the color for the given level
func getColorBuffer(level Level) *bytes.Buffer {
	b := &bytes.Buffer{}
	b.WriteString(level.Color())
	return b
}

// Common configuration options

// WithLevel returns an option to set the minimum severity level to log
func WithLevel(level Level) Option {
	return func(cfg *LoggerConfig) {
		cfg.Level = level
	}
}

// WithOutput returns an option to set the output destination
func WithOutput(output io.Writer) Option {
	return func(cfg *LoggerConfig) {
		cfg.Outputs = []io.Writer{output}
	}
}

// WithFormatter returns an option to set the formatter
func WithFormatter(formatter Formatter) Option {
	return func(cfg *LoggerConfig) {
		cfg.Formatter = formatter
	}
}

// WithColors returns an option to enable or disable colors
func WithColors(enable bool) Option {
	return func(cfg *LoggerConfig) {
		cfg.EnableColors = enable
	}
}

// WithCaller returns an option to enable or disable caller information
func WithCaller(enable bool) Option {
	return func(cfg *LoggerConfig) {
		cfg.ReportCaller = enable
	}
}

// WithAsync returns an option to enable or disable asynchronous logging
func WithAsync(enable bool, bufferSize int) Option {
	return func(cfg *LoggerConfig) {
		cfg.EnableAsync = enable
		if bufferSize > 0 {
			cfg.AsyncBufferSize = bufferSize
		}
	}
}

// WithSampling returns an option to enable or disable log sampling
func WithSampling(enable bool, rate float64) Option {
	return func(cfg *LoggerConfig) {
		cfg.EnableSampling = enable
		cfg.SampleRate = rate
	}
}

// Global logger
var std = NewLogger()

// SetDefaultLogger sets the global logger
func SetDefaultLogger(logger *DefaultLogger) {
	std = logger
}

// Global logging functions

// Trace logs a message at the trace level
func Trace(msg string, fields ...Field) {
	std.Trace(msg, fields...)
}

// Debug logs a message at the debug level
func Debug(msg string, fields ...Field) {
	std.Debug(msg, fields...)
}

// Info logs a message at the info level
func Info(msg string, fields ...Field) {
	std.Info(msg, fields...)
}

// Warn logs a message at the warn level
func Warn(msg string, fields ...Field) {
	std.Warn(msg, fields...)
}

// Error logs a message at the error level
func Error(msg string, fields ...Field) {
	std.Error(msg, fields...)
}

// Fatal logs a message at the fatal level and then exits
func Fatal(msg string, fields ...Field) {
	std.Fatal(msg, fields...)
}
