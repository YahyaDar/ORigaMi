// Copyright (c) 2025 Yahya Qadeer Dar. All rights reserved.
// Use of this source code is governed by an Apache 2.0 license that can be found in the LICENSE file.

package config

import (
	"fmt"
	"io"
	"os"
	"time"
)

// Common configuration option functions

// WithProvider adds a provider to the configuration
func WithProvider(provider Provider) Option {
	return func(cfg *Config) {
		cfg.providers = append(cfg.providers, provider)
	}
}

// WithFile adds a file-based configuration source
func WithFile(path string, optional bool) Option {
	return func(cfg *Config) {
		// We need to immediately load the file rather than just append
		// the source because options are applied in order
		_ = cfg.LoadFrom(NewFileSource(path, optional))
	}
}

// WithEnv adds an environment-based configuration source with the given prefix
func WithEnv(prefix string) Option {
	return func(cfg *Config) {
		_ = cfg.LoadFrom(NewEnvSource(prefix, true))
	}
}

// WithDefault sets a default configuration value
func WithDefault(key string, value interface{}) Option {
	return func(cfg *Config) {
		cfg.defaultValues[key] = value
	}
}

// WithDefaults sets multiple default configuration values
func WithDefaults(values map[string]interface{}) Option {
	return func(cfg *Config) {
		for k, v := range values {
			cfg.defaultValues[k] = v
		}
	}
}

// WithValidator adds a validator to the configuration
func WithValidator(validator Validator) Option {
	return func(cfg *Config) {
		cfg.AddValidator(validator)
	}
}

// WithRequiredKeys adds a validator that ensures certain keys are present
func WithRequiredKeys(keys ...string) Option {
	return WithValidator(NewRequiredKeysValidator(keys...))
}

// RequiredKeysValidator validates that certain keys are present
type RequiredKeysValidator struct {
	keys []string
}

// NewRequiredKeysValidator creates a new RequiredKeysValidator
func NewRequiredKeysValidator(keys ...string) *RequiredKeysValidator {
	return &RequiredKeysValidator{
		keys: keys,
	}
}

// Validate checks if all required keys are present
func (v *RequiredKeysValidator) Validate() error {
	for _, key := range v.keys {
		if !defaultConfig.Has(key) {
			return fmt.Errorf("required configuration key missing: %s", key)
		}
	}
	return nil
}

// DatabaseConfig represents database connection configuration
type DatabaseConfig struct {
	// Driver is the database driver name (postgres, mysql, sqlite)
	Driver string `json:"driver"`
	
	// DSN is the database connection string
	DSN string `json:"dsn"`
	
	// MaxOpenConns is the maximum number of open connections
	MaxOpenConns int `json:"max_open_conns"`
	
	// MaxIdleConns is the maximum number of idle connections
	MaxIdleConns int `json:"max_idle_conns"`
	
	// ConnMaxLifetime is the maximum amount of time a connection may be reused
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
	
	// ConnMaxIdleTime is the maximum amount of time a connection may be idle
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time"`
}

// Validate validates the database configuration
func (c *DatabaseConfig) Validate() error {
	if c.Driver == "" {
		return fmt.Errorf("database driver cannot be empty")
	}
	
	if c.DSN == "" {
		return fmt.Errorf("database DSN cannot be empty")
	}
	
	return nil
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	// Level is the minimum severity level to log
	Level string `json:"level"`
	
	// Format is the log format (text, json)
	Format string `json:"format"`
	
	// Output is the log output destination (stdout, stderr, file)
	Output string `json:"output"`
	
	// FilePath is the path to the log file when Output is "file"
	FilePath string `json:"file_path"`
	
	// Colors enables or disables ANSI colors
	Colors bool `json:"colors"`
	
	// ReportCaller enables or disables caller information
	ReportCaller bool `json:"report_caller"`
	
	// TimeFormat is the format for timestamps
	TimeFormat string `json:"time_format"`
}

// Validate validates the logging configuration
func (c *LoggingConfig) Validate() error {
	if c.Output == "file" && c.FilePath == "" {
		return fmt.Errorf("log file path cannot be empty when output is 'file'")
	}
	
	return nil
}

// BuildLoggerOptions builds logger options from logging configuration
func (c *LoggingConfig) BuildLoggerOptions() []Option {
	options := make([]Option, 0)
	
	// Set log level
	if c.Level != "" {
		var level int
		switch c.Level {
		case "trace":
			level = 0 // TraceLevel from log package
		case "debug":
			level = 1 // DebugLevel from log package
		case "info":
			level = 2 // InfoLevel from log package
		case "warn":
			level = 3 // WarnLevel from log package
		case "error":
			level = 4 // ErrorLevel from log package
		case "fatal":
			level = 5 // FatalLevel from log package
		}
		options = append(options, WithDefault("log.level", level))
	}
	
	// Set log format
	if c.Format != "" {
		options = append(options, WithDefault("log.format", c.Format))
	}
	
	// Set log output
	if c.Output != "" {
		options = append(options, WithDefault("log.output", c.Output))
	}
	
	// Set log file path
	if c.FilePath != "" {
		options = append(options, WithDefault("log.file_path", c.FilePath))
	}
	
	// Set colors
	options = append(options, WithDefault("log.colors", c.Colors))
	
	// Set report caller
	options = append(options, WithDefault("log.report_caller", c.ReportCaller))
	
	// Set time format
	if c.TimeFormat != "" {
		options = append(options, WithDefault("log.time_format", c.TimeFormat))
	}
	
	return options
}

// GetOutput gets the log output writer based on the configuration
func (c *LoggingConfig) GetOutput() (io.Writer, error) {
	switch c.Output {
	case "stdout":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	case "file":
		if c.FilePath == "" {
			return nil, fmt.Errorf("log file path cannot be empty")
		}
		return os.OpenFile(c.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	default:
		return os.Stdout, nil // Default to stdout
	}
}

// ORMConfig represents ORM configuration
type ORMConfig struct {
	// Database is the database configuration
	Database DatabaseConfig `json:"database"`
	
	// Logging is the logging configuration
	Logging LoggingConfig `json:"logging"`
	
	// ModelPaths are paths to search for models
	ModelPaths []string `json:"model_paths"`
	
	// MigrationPath is the path to store migrations
	MigrationPath string `json:"migration_path"`
	
	// TablePrefix is the prefix for table names
	TablePrefix string `json:"table_prefix"`
	
	// SingularTable determines if table names are singular
	SingularTable bool `json:"singular_table"`
	
	// Debug enables debug logging for all queries
	Debug bool `json:"debug"`
	
	// EnableCache enables query result caching
	EnableCache bool `json:"enable_cache"`
	
	// CacheTTL is the cache time-to-live duration
	CacheTTL time.Duration `json:"cache_ttl"`
}

// Validate validates the ORM configuration
func (c *ORMConfig) Validate() error {
	if err := c.Database.Validate(); err != nil {
		return err
	}
	
	if err := c.Logging.Validate(); err != nil {
		return err
	}
	
	return nil
}

// BuildOptions builds configuration options from ORM configuration
func (c *ORMConfig) BuildOptions() []Option {
	options := make([]Option, 0)
	
	// Add database options
	options = append(options, WithDefault("database.driver", c.Database.Driver))
	options = append(options, WithDefault("database.dsn", c.Database.DSN))
	options = append(options, WithDefault("database.max_open_conns", c.Database.MaxOpenConns))
	options = append(options, WithDefault("database.max_idle_conns", c.Database.MaxIdleConns))
	options = append(options, WithDefault("database.conn_max_lifetime", c.Database.ConnMaxLifetime))
	options = append(options, WithDefault("database.conn_max_idle_time", c.Database.ConnMaxIdleTime))
	
	// Add logging options
	options = append(options, c.Logging.BuildLoggerOptions()...)
	
	// Add model options
	options = append(options, WithDefault("model.paths", c.ModelPaths))
	options = append(options, WithDefault("model.table_prefix", c.TablePrefix))
	options = append(options, WithDefault("model.singular_table", c.SingularTable))
	
	// Add migration options
	options = append(options, WithDefault("migration.path", c.MigrationPath))
	
	// Add debug option
	options = append(options, WithDefault("debug", c.Debug))
	
	// Add cache options
	options = append(options, WithDefault("cache.enabled", c.EnableCache))
	options = append(options, WithDefault("cache.ttl", c.CacheTTL))
	
	return options
}

// Load ORM configuration from file
func LoadORMConfig(path string) (*ORMConfig, error) {
	config := New(
		WithFile(path, false),
	)
	
	ormConfig := &ORMConfig{
		Database: DatabaseConfig{
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: time.Minute * 5,
			ConnMaxIdleTime: time.Minute * 2,
		},
		Logging: LoggingConfig{
			Level:        "info",
			Format:       "text",
			Output:       "stdout",
			Colors:       true,
			ReportCaller: true,
			TimeFormat:   "2006-01-02 15:04:05.000",
		},
		TablePrefix:    "",
		SingularTable:  false,
		Debug:          false,
		EnableCache:    true,
		CacheTTL:       time.Minute * 5,
	}
	
	if err := config.GetStruct("", ormConfig); err != nil {
		return nil, err
	}
	
	if err := ormConfig.Validate(); err != nil {
		return nil, err
	}
	
	return ormConfig, nil
}

// Builder provides a fluent interface for configuration
type Builder struct {
	config *Config
}

// NewBuilder creates a new configuration builder
func NewBuilder() *Builder {
	return &Builder{
		config: New(),
	}
}

// WithDefaults sets default values
func (b *Builder) WithDefaults(defaults map[string]interface{}) *Builder {
	for k, v := range defaults {
		b.config.defaultValues[k] = v
	}
	return b
}

// WithFile adds a file source
func (b *Builder) WithFile(path string, optional bool) *Builder {
	_ = b.config.LoadFrom(NewFileSource(path, optional))
	return b
}

// WithEnv adds an environment source
func (b *Builder) WithEnv(prefix string) *Builder {
	_ = b.config.LoadFrom(NewEnvSource(prefix, true))
	return b
}

// WithValidator adds a validator
func (b *Builder) WithValidator(validator Validator) *Builder {
	b.config.AddValidator(validator)
	return b
}

// WithRequiredKeys adds a required keys validator
func (b *Builder) WithRequiredKeys(keys ...string) *Builder {
	b.config.AddValidator(NewRequiredKeysValidator(keys...))
	return b
}

// Build builds the configuration
func (b *Builder) Build() (*Config, error) {
	if err := b.config.validate(); err != nil {
		return nil, err
	}
	return b.config, nil
}
