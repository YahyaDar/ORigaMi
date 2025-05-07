package errors

import (
	"errors"
	"fmt"
)

// Standard errors provides exported error variables for common error cases.
var (
	// ErrNotFound indicates that a requested entity could not be found.
	ErrNotFound = errors.New("entity not found")

	// ErrInvalidModel indicates that a model is invalid for ORM operations.
	ErrInvalidModel = errors.New("invalid model structure")

	// ErrInvalidOperation indicates that the requested operation is invalid in the current context.
	ErrInvalidOperation = errors.New("invalid operation")

	// ErrConnectionFailed indicates a failure to establish a database connection.
	ErrConnectionFailed = errors.New("database connection failed")

	// ErrTransactionFailed indicates a failure during a transaction operation.
	ErrTransactionFailed = errors.New("transaction operation failed")

	// ErrQueryFailed indicates a failure during query execution.
	ErrQueryFailed = errors.New("query execution failed")

	// ErrMigrationFailed indicates a failure during migration operations.
	ErrMigrationFailed = errors.New("migration failed")

	// ErrValidationFailed indicates a validation failure on a model.
	ErrValidationFailed = errors.New("validation failed")

	// ErrPluginRegistrationFailed indicates a failure to register a plugin.
	ErrPluginRegistrationFailed = errors.New("plugin registration failed")
)

// Error types specific to the ORM.
type (
	// Error is the base interface for all ORigaMi-specific errors.
	Error interface {
		error
		OrigamiError() bool
	}

	// QueryError represents an error that occurs during SQL query execution.
	QueryError struct {
		Query   string
		Message string
		Err     error
	}

	// ModelError represents an error related to a model definition or operation.
	ModelError struct {
		Model   string
		Message string
		Err     error
	}

	// ValidationError represents field validation errors for a model.
	ValidationError struct {
		Model  string
		Fields map[string]string
		Err    error
	}

	// ConnectionError represents errors that occur when connecting to a database.
	ConnectionError struct {
		Driver  string
		Message string
		Err     error
	}

	// TransactionError represents errors that occur during a transaction.
	TransactionError struct {
		Operation string
		Message   string
		Err       error
	}

	// MigrationError represents errors that occur during schema migrations.
	MigrationError struct {
		Version string
		Message string
		Err     error
	}

	// PluginError represents errors related to plugins.
	PluginError struct {
		Plugin  string
		Message string
		Err     error
	}
)

// OrigamiError identifies this as an ORigaMi error.
func (e *QueryError) OrigamiError() bool { return true }

// Error returns the error message.
func (e *QueryError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("query error: %s: %v", e.Message, e.Err)
	}
	return fmt.Sprintf("query error: %s", e.Message)
}

// Unwrap returns the underlying error.
func (e *QueryError) Unwrap() error { return e.Err }

// OrigamiError identifies this as an ORigaMi error.
func (e *ModelError) OrigamiError() bool { return true }

// Error returns the error message.
func (e *ModelError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("model error (%s): %s: %v", e.Model, e.Message, e.Err)
	}
	return fmt.Sprintf("model error (%s): %s", e.Model, e.Message)
}

// Unwrap returns the underlying error.
func (e *ModelError) Unwrap() error { return e.Err }

// OrigamiError identifies this as an ORigaMi error.
func (e *ValidationError) OrigamiError() bool { return true }

// Error returns the error message.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error (%s): %d field(s) failed validation", e.Model, len(e.Fields))
}

// Unwrap returns the underlying error.
func (e *ValidationError) Unwrap() error { return e.Err }

// FieldErrors returns the map of validation errors by field.
func (e *ValidationError) FieldErrors() map[string]string {
	return e.Fields
}

// OrigamiError identifies this as an ORigaMi error.
func (e *ConnectionError) OrigamiError() bool { return true }

// Error returns the error message.
func (e *ConnectionError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("connection error (%s): %s: %v", e.Driver, e.Message, e.Err)
	}
	return fmt.Sprintf("connection error (%s): %s", e.Driver, e.Message)
}

// Unwrap returns the underlying error.
func (e *ConnectionError) Unwrap() error { return e.Err }

// OrigamiError identifies this as an ORigaMi error.
func (e *TransactionError) OrigamiError() bool { return true }

// Error returns the error message.
func (e *TransactionError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("transaction error (%s): %s: %v", e.Operation, e.Message, e.Err)
	}
	return fmt.Sprintf("transaction error (%s): %s", e.Operation, e.Message)
}

// Unwrap returns the underlying error.
func (e *TransactionError) Unwrap() error { return e.Err }

// OrigamiError identifies this as an ORigaMi error.
func (e *MigrationError) OrigamiError() bool { return true }

// Error returns the error message.
func (e *MigrationError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("migration error (version %s): %s: %v", e.Version, e.Message, e.Err)
	}
	return fmt.Sprintf("migration error (version %s): %s", e.Version, e.Message)
}

// Unwrap returns the underlying error.
func (e *MigrationError) Unwrap() error { return e.Err }

// OrigamiError identifies this as an ORigaMi error.
func (e *PluginError) OrigamiError() bool { return true }

// Error returns the error message.
func (e *PluginError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("plugin error (%s): %s: %v", e.Plugin, e.Message, e.Err)
	}
	return fmt.Sprintf("plugin error (%s): %s", e.Plugin, e.Message)
}

// Unwrap returns the underlying error.
func (e *PluginError) Unwrap() error { return e.Err }

// NewQueryError creates a new QueryError.
func NewQueryError(query, message string, err error) *QueryError {
	return &QueryError{Query: query, Message: message, Err: err}
}

// NewModelError creates a new ModelError.
func NewModelError(model, message string, err error) *ModelError {
	return &ModelError{Model: model, Message: message, Err: err}
}

// NewValidationError creates a new ValidationError.
func NewValidationError(model string, fields map[string]string, err error) *ValidationError {
	return &ValidationError{Model: model, Fields: fields, Err: err}
}

// NewConnectionError creates a new ConnectionError.
func NewConnectionError(driver, message string, err error) *ConnectionError {
	return &ConnectionError{Driver: driver, Message: message, Err: err}
}

// NewTransactionError creates a new TransactionError.
func NewTransactionError(operation, message string, err error) *TransactionError {
	return &TransactionError{Operation: operation, Message: message, Err: err}
}

// NewMigrationError creates a new MigrationError.
func NewMigrationError(version, message string, err error) *MigrationError {
	return &MigrationError{Version: version, Message: message, Err: err}
}

// NewPluginError creates a new PluginError.
func NewPluginError(plugin, message string, err error) *PluginError {
	return &PluginError{Plugin: plugin, Message: message, Err: err}
}

// Is reports whether any error in err's tree matches target.
// It's a wrapper around the standard errors.Is function.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's tree that matches the target type.
// It's a wrapper around the standard errors.As function.
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

// Wrap wraps an error with a message.
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// Wrapf wraps an error with a formatted message.
func Wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}
