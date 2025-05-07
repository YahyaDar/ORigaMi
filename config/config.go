// Copyright (c) 2025 YahyaQadeer Dar. All rights reserved.
// Use of this source code is governed by an Apache 2.0 license that can be found in the LICENSE file.

// Package config provides a comprehensive configuration system for the ORigaMi ORM.
// It supports multiple configuration sources, validation, and dynamic updates.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/YahyaDar/ORigaMi/errors"
)

// Provider defines the interface for configuration providers
type Provider interface {
	// Get returns a configuration value
	Get(key string) (interface{}, bool)
	
	// Set sets a configuration value
	Set(key string, value interface{})
	
	// Has checks if a configuration key exists
	Has(key string) bool
	
	// Keys returns all configuration keys
	Keys() []string
	
	// Sub returns a sub-configuration
	Sub(key string) Provider
	
	// AllSettings returns all settings as a map
	AllSettings() map[string]interface{}
	
	// AllSettingsFlattened returns all settings as a flattened map
	AllSettingsFlattened() map[string]interface{}
	
	// LoadFrom loads configuration from a source
	LoadFrom(source Source) error
}

// Source defines the interface for configuration sources
type Source interface {
	// Load loads configuration into the provider
	Load(provider Provider) error
	
	// Name returns the source name
	Name() string
}

// Validator defines the interface for configuration validation
type Validator interface {
	// Validate validates the configuration
	Validate() error
}

// Config is the main configuration container
type Config struct {
	// mu protects access to the configuration
	mu sync.RWMutex
	
	// values stores the configuration values
	values map[string]interface{}
	
	// providers stores the ordered configuration providers
	providers []Provider
	
	// validators stores the configuration validators
	validators []Validator
	
	// envPrefix is the prefix for environment variables
	envPrefix string
	
	// defaultValues stores the default configuration values
	defaultValues map[string]interface{}
}

// Option is a function that configures a Config
type Option func(*Config)

// New creates a new configuration with the given options
func New(options ...Option) *Config {
	config := &Config{
		values:        make(map[string]interface{}),
		providers:     make([]Provider, 0),
		validators:    make([]Validator, 0),
		defaultValues: make(map[string]interface{}),
	}
	
	// Apply options
	for _, option := range options {
		option(config)
	}
	
	// Add default memory provider if none exists
	if len(config.providers) == 0 {
		config.providers = append(config.providers, NewMemoryProvider())
	}
	
	// Apply default values
	for k, v := range config.defaultValues {
		config.Set(k, v)
	}
	
	return config
}

// Get retrieves a configuration value
func (c *Config) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	// Search in reverse order to prioritize later providers
	for i := len(c.providers) - 1; i >= 0; i-- {
		if value, ok := c.providers[i].Get(key); ok {
			return value, true
		}
	}
	
	// Check local values
	if value, ok := c.values[key]; ok {
		return value, true
	}
	
	return nil, false
}

// GetString retrieves a string configuration value
func (c *Config) GetString(key string) (string, error) {
	value, ok := c.Get(key)
	if !ok {
		return "", errors.NewConfigError("key not found", nil).WithKey(key)
	}
	
	if str, ok := value.(string); ok {
		return str, nil
	}
	
	return fmt.Sprintf("%v", value), nil
}

// GetInt retrieves an integer configuration value
func (c *Config) GetInt(key string) (int, error) {
	value, ok := c.Get(key)
	if !ok {
		return 0, errors.NewConfigError("key not found", nil).WithKey(key)
	}
	
	switch v := value.(type) {
	case int:
		return v, nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case float32:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		var i int
		if _, err := fmt.Sscanf(v, "%d", &i); err != nil {
			return 0, errors.NewConfigError("invalid integer value", err).WithKey(key)
		}
		return i, nil
	}
	
	return 0, errors.NewConfigError("invalid integer value", nil).WithKey(key).WithValue(value)
}

// GetBool retrieves a boolean configuration value
func (c *Config) GetBool(key string) (bool, error) {
	value, ok := c.Get(key)
	if !ok {
		return false, errors.NewConfigError("key not found", nil).WithKey(key)
	}
	
	switch v := value.(type) {
	case bool:
		return v, nil
	case int:
		return v != 0, nil
	case string:
		switch strings.ToLower(v) {
		case "true", "yes", "1", "on", "t", "y":
			return true, nil
		case "false", "no", "0", "off", "f", "n":
			return false, nil
		}
	}
	
	return false, errors.NewConfigError("invalid boolean value", nil).WithKey(key).WithValue(value)
}

// GetDuration retrieves a duration configuration value
func (c *Config) GetDuration(key string) (time.Duration, error) {
	value, ok := c.Get(key)
	if !ok {
		return 0, errors.NewConfigError("key not found", nil).WithKey(key)
	}
	
	switch v := value.(type) {
	case time.Duration:
		return v, nil
	case int:
		return time.Duration(v) * time.Second, nil
	case int64:
		return time.Duration(v) * time.Second, nil
	case float64:
		return time.Duration(v * float64(time.Second)), nil
	case string:
		d, err := time.ParseDuration(v)
		if err != nil {
			return 0, errors.NewConfigError("invalid duration value", err).WithKey(key)
		}
		return d, nil
	}
	
	return 0, errors.NewConfigError("invalid duration value", nil).WithKey(key).WithValue(value)
}

// GetFloat retrieves a float configuration value
func (c *Config) GetFloat(key string) (float64, error) {
	value, ok := c.Get(key)
	if !ok {
		return 0, errors.NewConfigError("key not found", nil).WithKey(key)
	}
	
	switch v := value.(type) {
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		var f float64
		if _, err := fmt.Sscanf(v, "%f", &f); err != nil {
			return 0, errors.NewConfigError("invalid float value", err).WithKey(key)
		}
		return f, nil
	}
	
	return 0, errors.NewConfigError("invalid float value", nil).WithKey(key).WithValue(value)
}

// GetStringSlice retrieves a string slice configuration value
func (c *Config) GetStringSlice(key string) ([]string, error) {
	value, ok := c.Get(key)
	if !ok {
		return nil, errors.NewConfigError("key not found", nil).WithKey(key)
	}
	
	switch v := value.(type) {
	case []string:
		return v, nil
	case []interface{}:
		result := make([]string, len(v))
		for i, val := range v {
			result[i] = fmt.Sprintf("%v", val)
		}
		return result, nil
	case string:
		if v == "" {
			return []string{}, nil
		}
		return strings.Split(v, ","), nil
	}
	
	return nil, errors.NewConfigError("invalid string slice value", nil).WithKey(key).WithValue(value)
}

// GetStruct retrieves a struct configuration value
func (c *Config) GetStruct(key string, result interface{}) error {
	value, ok := c.Get(key)
	if !ok {
		return errors.NewConfigError("key not found", nil).WithKey(key)
	}
	
	resultValue := reflect.ValueOf(result)
	if resultValue.Kind() != reflect.Ptr || resultValue.IsNil() {
		return errors.NewConfigError("result must be a non-nil pointer", nil)
	}
	
	// Direct assignment for matching types
	if reflect.TypeOf(value) == resultValue.Elem().Type() {
		resultValue.Elem().Set(reflect.ValueOf(value))
		return nil
	}
	
	// Convert to map then JSON marshal/unmarshal
	switch v := value.(type) {
	case map[string]interface{}:
		data, err := json.Marshal(v)
		if err != nil {
			return errors.NewConfigError("failed to marshal struct data", err).WithKey(key)
		}
		if err = json.Unmarshal(data, result); err != nil {
			return errors.NewConfigError("failed to unmarshal struct data", err).WithKey(key)
		}
		return nil
	}
	
	return errors.NewConfigError("invalid struct value", nil).WithKey(key).WithValue(value)
}

// Set sets a configuration value
func (c *Config) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if len(c.providers) > 0 {
		c.providers[len(c.providers)-1].Set(key, value)
	} else {
		c.values[key] = value
	}
}

// Has checks if a configuration key exists
func (c *Config) Has(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	// Check in providers
	for i := len(c.providers) - 1; i >= 0; i-- {
		if c.providers[i].Has(key) {
			return true
		}
	}
	
	// Check local values
	_, ok := c.values[key]
	return ok
}

// AllSettings returns all settings as a map
func (c *Config) AllSettings() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	result := make(map[string]interface{})
	
	// Start with local values
	for k, v := range c.values {
		result[k] = v
	}
	
	// Add provider values in order, overwriting as we go
	for _, provider := range c.providers {
		for k, v := range provider.AllSettings() {
			result[k] = v
		}
	}
	
	return result
}

// Sub returns a sub-configuration
func (c *Config) Sub(key string) Provider {
	if !c.Has(key) {
		return nil
	}
	
	value, _ := c.Get(key)
	if subMap, ok := value.(map[string]interface{}); ok {
		provider := NewMemoryProvider()
		for k, v := range subMap {
			provider.Set(k, v)
		}
		return provider
	}
	
	return nil
}

// LoadFrom loads configuration from a source
func (c *Config) LoadFrom(source Source) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Create a memory provider for the new source
	provider := NewMemoryProvider()
	
	// Load into the provider
	if err := source.Load(provider); err != nil {
		return errors.NewConfigError("failed to load configuration", err).
			WithValue(source.Name())
	}
	
	// Add the provider
	c.providers = append(c.providers, provider)
	
	// Validate configuration
	return c.validate()
}

// AddValidator adds a validator to the configuration
func (c *Config) AddValidator(validator Validator) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.validators = append(c.validators, validator)
}

// Validate validates the configuration
func (c *Config) Validate() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	return c.validate()
}

// validate performs validation (internal, no locking)
func (c *Config) validate() error {
	for _, validator := range c.validators {
		if err := validator.Validate(); err != nil {
			return err
		}
	}
	
	return nil
}

// MemoryProvider is a memory-based configuration provider
type MemoryProvider struct {
	values map[string]interface{}
	mu     sync.RWMutex
}

// NewMemoryProvider creates a new memory-based configuration provider
func NewMemoryProvider() *MemoryProvider {
	return &MemoryProvider{
		values: make(map[string]interface{}),
	}
}

// Get retrieves a configuration value
func (p *MemoryProvider) Get(key string) (interface{}, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	// Handle nested keys
	parts := strings.Split(key, ".")
	curr := p.values
	
	for i, part := range parts {
		if i == len(parts)-1 {
			val, ok := curr[part]
			return val, ok
		}
		
		next, ok := curr[part]
		if !ok {
			return nil, false
		}
		
		nextMap, ok := next.(map[string]interface{})
		if !ok {
			return nil, false
		}
		
		curr = nextMap
	}
	
	return nil, false
}

// Set sets a configuration value
func (p *MemoryProvider) Set(key string, value interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Handle nested keys
	parts := strings.Split(key, ".")
	curr := p.values
	
	for i, part := range parts {
		if i == len(parts)-1 {
			curr[part] = value
			return
		}
		
		next, ok := curr[part]
		if !ok {
			next = make(map[string]interface{})
			curr[part] = next
		}
		
		nextMap, ok := next.(map[string]interface{})
		if !ok {
			// Replace with a map
			nextMap = make(map[string]interface{})
			curr[part] = nextMap
		}
		
		curr = nextMap
	}
}

// Has checks if a configuration key exists
func (p *MemoryProvider) Has(key string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	// Handle nested keys
	parts := strings.Split(key, ".")
	curr := p.values
	
	for i, part := range parts {
		if i == len(parts)-1 {
			_, ok := curr[part]
			return ok
		}
		
		next, ok := curr[part]
		if !ok {
			return false
		}
		
		nextMap, ok := next.(map[string]interface{})
		if !ok {
			return false
		}
		
		curr = nextMap
	}
	
	return false
}

// Keys returns all configuration keys
func (p *MemoryProvider) Keys() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	var keys []string
	p.collectKeys("", p.values, &keys)
	return keys
}

// collectKeys recursively collects keys from a nested map
func (p *MemoryProvider) collectKeys(prefix string, m map[string]interface{}, keys *[]string) {
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		
		*keys = append(*keys, key)
		
		if subMap, ok := v.(map[string]interface{}); ok {
			p.collectKeys(key, subMap, keys)
		}
	}
}

// Sub returns a sub-configuration
func (p *MemoryProvider) Sub(key string) Provider {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	// Handle nested keys
	parts := strings.Split(key, ".")
	curr := p.values
	
	for _, part := range parts {
		next, ok := curr[part]
		if !ok {
			return nil
		}
		
		nextMap, ok := next.(map[string]interface{})
		if !ok {
			return nil
		}
		
		curr = nextMap
	}
	
	provider := NewMemoryProvider()
	for k, v := range curr {
		provider.Set(k, v)
	}
	
	return provider
}

// AllSettings returns all settings as a map
func (p *MemoryProvider) AllSettings() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	// Deep copy to avoid race conditions
	return deepCopyMap(p.values)
}

// AllSettingsFlattened returns all settings as a flattened map
func (p *MemoryProvider) AllSettingsFlattened() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	result := make(map[string]interface{})
	p.flatten("", p.values, result)
	return result
}

// flatten recursively flattens a nested map
func (p *MemoryProvider) flatten(prefix string, nested map[string]interface{}, flat map[string]interface{}) {
	for k, v := range nested {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		
		if subMap, ok := v.(map[string]interface{}); ok {
			p.flatten(key, subMap, flat)
		} else {
			flat[key] = v
		}
	}
}

// LoadFrom loads configuration from a source
func (p *MemoryProvider) LoadFrom(source Source) error {
	return source.Load(p)
}

// deepCopyMap creates a deep copy of a map
func deepCopyMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	for k, v := range m {
		switch value := v.(type) {
		case map[string]interface{}:
			result[k] = deepCopyMap(value)
		case []interface{}:
			result[k] = deepCopySlice(value)
		default:
			result[k] = v
		}
	}
	
	return result
}

// deepCopySlice creates a deep copy of a slice
func deepCopySlice(s []interface{}) []interface{} {
	result := make([]interface{}, len(s))
	
	for i, v := range s {
		switch value := v.(type) {
		case map[string]interface{}:
			result[i] = deepCopyMap(value)
		case []interface{}:
			result[i] = deepCopySlice(value)
		default:
			result[i] = v
		}
	}
	
	return result
}

// FileSource is a file-based configuration source
type FileSource struct {
	path     string
	optional bool
}

// NewFileSource creates a new file-based configuration source
func NewFileSource(path string, optional bool) *FileSource {
	return &FileSource{
		path:     path,
		optional: optional,
	}
}

// Load loads configuration from a file
func (s *FileSource) Load(provider Provider) error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) && s.optional {
			return nil
		}
		return errors.NewConfigError("failed to read config file", err).
			WithValue(s.path)
	}
	
	var result map[string]interface{}
	
	// Determine file type from extension
	ext := strings.ToLower(filepath.Ext(s.path))
	
	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &result); err != nil {
			return errors.NewConfigError("failed to parse JSON config", err).
				WithValue(s.path)
		}
	case ".yaml", ".yml":
		// Implemented in options.go using yaml.Unmarshal
		return errors.NewConfigError("YAML support requires yaml.v3 package", nil).
			WithValue(s.path)
	default:
		return errors.NewConfigError("unsupported config file format", nil).
			WithValue(s.path)
	}
	
	// Set values in provider
	for k, v := range result {
		provider.Set(k, v)
	}
	
	return nil
}

// Name returns the source name
func (s *FileSource) Name() string {
	return fmt.Sprintf("file(%s)", s.path)
}

// EnvSource is an environment-based configuration source
type EnvSource struct {
	prefix    string
	lowercase bool
}

// NewEnvSource creates a new environment-based configuration source
func NewEnvSource(prefix string, lowercase bool) *EnvSource {
	return &EnvSource{
		prefix:    prefix,
		lowercase: lowercase,
	}
}

// Load loads configuration from environment variables
func (s *EnvSource) Load(provider Provider) error {
	env := os.Environ()
	
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			continue
		}
		
		key := parts[0]
		value := parts[1]
		
		// Check if key has the prefix
		if s.prefix != "" && !strings.HasPrefix(key, s.prefix) {
			continue
		}
		
		// Remove prefix
		if s.prefix != "" {
			key = key[len(s.prefix):]
			// Remove leading underscore if present
			if strings.HasPrefix(key, "_") {
				key = key[1:]
			}
		}
		
		// Convert to lowercase if needed
		if s.lowercase {
			key = strings.ToLower(key)
		}
		
		// Replace underscores with dots
		key = strings.ReplaceAll(key, "_", ".")
		
		// Set value in provider
		provider.Set(key, value)
	}
	
	return nil
}

// Name returns the source name
func (s *EnvSource) Name() string {
	return fmt.Sprintf("env(%s)", s.prefix)
}

// Default global configuration
var defaultConfig = New()

// LoadFile loads configuration from a file
func LoadFile(path string, optional bool) error {
	return defaultConfig.LoadFrom(NewFileSource(path, optional))
}

// LoadEnv loads configuration from environment variables
func LoadEnv(prefix string) error {
	return defaultConfig.LoadFrom(NewEnvSource(prefix, true))
}

// Get retrieves a configuration value from the default configuration
func Get(key string) (interface{}, bool) {
	return defaultConfig.Get(key)
}

// Set sets a configuration value in the default configuration
func Set(key string, value interface{}) {
	defaultConfig.Set(key, value)
}

// GetString retrieves a string configuration value from the default configuration
func GetString(key string) (string, error) {
	return defaultConfig.GetString(key)
}

// GetInt retrieves an integer configuration value from the default configuration
func GetInt(key string) (int, error) {
	return defaultConfig.GetInt(key)
}

// GetBool retrieves a boolean configuration value from the default configuration
func GetBool(key string) (bool, error) {
	return defaultConfig.GetBool(key)
}

// GetDuration retrieves a duration configuration value from the default configuration
func GetDuration(key string) (time.Duration, error) {
	return defaultConfig.GetDuration(key)
}

// Validate validates the default configuration
func Validate() error {
	return defaultConfig.Validate()
}
