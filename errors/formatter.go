// Copyright (c) 2025 Yahya Qadeer Dar. All rights reserved.
// Use of this source code is governed by an Apache 2.0 license that can be found in the LICENSE file.

// Package errors provides standardized error handling for the ORigaMi ORM.
// The formatter.go file contains utilities for consistent error formatting and presentation.
package errors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"time"
)

// ErrorFormatter defines the interface for formatting errors
type ErrorFormatter interface {
	// Format converts an error into a formatted string representation
	Format(err error) string
	
	// FormatWithContext adds contextual information to an error message
	FormatWithContext(err error, context map[string]interface{}) string
	
	// FormatJSON returns a JSON representation of the error
	FormatJSON(err error) ([]byte, error)
}

// DefaultFormatter is the standard error formatter for ORigaMi errors
type DefaultFormatter struct {
	// IncludeTimestamp determines if timestamps should be included in error messages
	IncludeTimestamp bool
	
	// IncludeStackTrace determines if stack traces should be included in error messages
	IncludeStackTrace bool
	
	// MaxStackDepth is the maximum number of stack frames to include
	MaxStackDepth int
}

// NewDefaultFormatter creates a new DefaultFormatter with recommended settings
func NewDefaultFormatter() *DefaultFormatter {
	return &DefaultFormatter{
		IncludeTimestamp: true,
		IncludeStackTrace: true,
		MaxStackDepth: 10,
	}
}

// Format converts an error into a formatted string representation
func (f *DefaultFormatter) Format(err error) string {
	if err == nil {
		return ""
	}
	
	var buffer bytes.Buffer
	
	// Add timestamp if configured
	if f.IncludeTimestamp {
		buffer.WriteString(fmt.Sprintf("[%s] ", time.Now().UTC().Format("2006-01-02 15:04:05")))
	}
	
	// Handle wrapped errors
	if wrapper, ok := err.(interface{ Unwrap() error }); ok {
		innerErr := wrapper.Unwrap()
		if innerErr != nil {
			buffer.WriteString(fmt.Sprintf("%s: %s", err.Error(), innerErr.Error()))
		} else {
			buffer.WriteString(err.Error())
		}
	} else {
		buffer.WriteString(err.Error())
	}
	
	// Add stack trace if configured
	if f.IncludeStackTrace {
		buffer.WriteString("\nStack Trace:\n")
		buffer.WriteString(f.getStackTrace(f.MaxStackDepth))
	}
	
	return buffer.String()
}

// FormatWithContext adds contextual information to an error message
func (f *DefaultFormatter) FormatWithContext(err error, context map[string]interface{}) string {
	if err == nil {
		return ""
	}
	
	var buffer bytes.Buffer
	buffer.WriteString(f.Format(err))
	buffer.WriteString("\nContext:\n")
	
	// Add context information
	if len(context) > 0 {
		for k, v := range context {
			buffer.WriteString(fmt.Sprintf("  %s: %v\n", k, v))
		}
	} else {
		buffer.WriteString("  No context provided\n")
	}
	
	return buffer.String()
}

// FormatJSON returns a JSON representation of the error
func (f *DefaultFormatter) FormatJSON(err error) ([]byte, error) {
	if err == nil {
		return []byte("null"), nil
	}
	
	errorMap := map[string]interface{}{
		"message": err.Error(),
		"time":    time.Now().UTC().Format(time.RFC3339),
	}
	
	// Add stack trace if configured
	if f.IncludeStackTrace {
		errorMap["stack_trace"] = strings.Split(f.getStackTrace(f.MaxStackDepth), "\n")
	}
	
	// Include original error type
	errorMap["type"] = fmt.Sprintf("%T", err)
	
	// Handle wrapped errors
	if wrapper, ok := err.(interface{ Unwrap() error }); ok {
		innerErr := wrapper.Unwrap()
		if innerErr != nil {
			errorMap["wrapped_error"] = innerErr.Error()
			errorMap["wrapped_type"] = fmt.Sprintf("%T", innerErr)
		}
	}
	
	return json.MarshalIndent(errorMap, "", "  ")
}

// getStackTrace returns a formatted stack trace limited to the specified depth
func (f *DefaultFormatter) getStackTrace(maxDepth int) string {
	var buffer bytes.Buffer
	
	// Skip the first 3 frames which are related to this error formatting code
	skip := 3
	
	// Collect stack frames up to the maximum depth
	for i := 0; i < maxDepth; i++ {
		pc, file, line, ok := runtime.Caller(skip + i)
		if !ok {
			break
		}
		
		// Get function name
		fn := runtime.FuncForPC(pc)
		funcName := "unknown"
		if fn != nil {
			funcName = fn.Name()
		}
		
		// Simplify file path by removing everything before the last /src/
		idx := strings.LastIndex(file, "/src/")
		if idx >= 0 {
			file = file[idx+5:]
		}
		
		buffer.WriteString(fmt.Sprintf("  %d: %s\n    %s:%d\n", i, funcName, file, line))
	}
	
	return buffer.String()
}

// PrettyFormat returns a human-readable formatted error message
func PrettyFormat(err error) string {
	return NewDefaultFormatter().Format(err)
}

// JSONFormat returns a JSON representation of the error
func JSONFormat(err error) (string, error) {
	data, err := NewDefaultFormatter().FormatJSON(err)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
