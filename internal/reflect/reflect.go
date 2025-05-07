// Copyright (c) 2025 Yahya Qadeer Dar. All rights reserved.
// Use of this source code is governed by an Apache 2.0 license that can be found in the LICENSE file.

// Package reflect provides enhanced reflection utilities for the ORigaMi ORM.
// It extends Go's standard reflection capabilities with caching and specialized
// functionality for database operations and model handling.
package reflect

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"unicode"
	
	"github.com/YahyaDar/ORigaMi/errors"
)

// fieldCache stores cached field information to avoid repeated reflection
// operations on the same types
var (
	fieldCache     = make(map[reflect.Type]map[string]*FieldInfo)
	fieldCacheLock sync.RWMutex
)

// TagKey is the struct tag key used for ORigaMi ORM annotations
const TagKey = "origami"

// FieldInfo stores enhanced field information for a struct field
type FieldInfo struct {
	// Name is the field name in the struct
	Name string
	
	// DBName is the field name in the database
	DBName string
	
	// Type is the Go type of the field
	Type reflect.Type
	
	// Index is the index of the field in the struct
	Index []int
	
	// IsAnonymous indicates if this is an anonymous (embedded) field
	IsAnonymous bool
	
	// IsPrimaryKey indicates if this field is a primary key
	IsPrimaryKey bool
	
	// IsAutoIncrement indicates if this field is auto-incrementing
	IsAutoIncrement bool
	
	// IsUnique indicates if this field has a unique constraint
	IsUnique bool
	
	// IsIndex indicates if this field has an index
	IsIndex bool
	
	// IsNotNull indicates if this field is not nullable
	IsNotNull bool
	
	// Size specifies the size/length for the field (e.g., varchar(255))
	Size int
	
	// Precision specifies the precision for decimal fields
	Precision int
	
	// Scale specifies the scale for decimal fields
	Scale int
	
	// Default specifies the default value for the field
	Default string
	
	// RawTag contains the raw tag string
	RawTag string
	
	// TagSettings contains parsed tag settings
	TagSettings map[string]string
	
	// Referenced holds information about referenced models for relationships
	Referenced *ReferenceInfo
	
	// IsIgnored indicates if this field should be ignored by the ORM
	IsIgnored bool
	
	// IsReadOnly indicates if this field is read-only
	IsReadOnly bool
	
	// IsWriteOnly indicates if this field is write-only
	IsWriteOnly bool
}

// ReferenceInfo stores information about referenced models
type ReferenceInfo struct {
	// Model is the referenced model name
	Model string
	
	// Field is the referenced field name
	Field string
	
	// OnDelete specifies the ON DELETE action
	OnDelete string
	
	// OnUpdate specifies the ON UPDATE action
	OnUpdate string
}

// ModelInfo stores model information extracted from a struct
type ModelInfo struct {
	// Name is the model name
	Name string
	
	// Type is the model's Go type
	Type reflect.Type
	
	// DBName is the database table name
	DBName string
	
	// Fields maps field names to field information
	Fields map[string]*FieldInfo
	
	// FieldsByDBName maps database field names to field information
	FieldsByDBName map[string]*FieldInfo
	
	// PrimaryKey contains the primary key field name(s)
	PrimaryKey []string
	
	// AutoIncrement contains the auto-incrementing field name (if any)
	AutoIncrement string
	
	// Indexes maps index names to field names
	Indexes map[string][]string
	
	// UniqueIndexes maps unique index names to field names
	UniqueIndexes map[string][]string
	
	// TagSettings contains model-level tag settings
	TagSettings map[string]string
}

// ExtractModelInfo extracts model information from a struct
func ExtractModelInfo(model interface{}) (*ModelInfo, error) {
	modelType := IndirectType(TypeOf(model))
	if modelType.Kind() != reflect.Struct {
		return nil, errors.NewModelError("model must be a struct", nil).
			WithModel(fmt.Sprintf("%T", model))
	}
	
	info := &ModelInfo{
		Name:           modelType.Name(),
		Type:           modelType,
		DBName:         ToSnakeCase(modelType.Name()),
		Fields:         make(map[string]*FieldInfo),
		FieldsByDBName: make(map[string]*FieldInfo),
		PrimaryKey:     make([]string, 0),
		Indexes:        make(map[string][]string),
		UniqueIndexes:  make(map[string][]string),
		TagSettings:    make(map[string]string),
	}
	
	// Process struct-level tags from the origami tag if present
	if structTag, ok := modelType.FieldByName("origami"); ok {
		if tag, ok := structTag.Tag.Lookup(TagKey); ok {
			info.TagSettings = ParseTagSettings(tag)
			
			// Apply table name override if specified
			if table, ok := info.TagSettings["table"]; ok && table != "" {
				info.DBName = table
			}
		}
	}
	
	// Process fields
	fields, err := ExtractFields(model, "")
	if err != nil {
		return nil, err
	}
	
	for _, field := range fields {
		info.Fields[field.Name] = field
		info.FieldsByDBName[field.DBName] = field
		
		if field.IsPrimaryKey {
			info.PrimaryKey = append(info.PrimaryKey, field.Name)
		}
		
		if field.IsAutoIncrement {
			info.AutoIncrement = field.Name
		}
		
		if field.IsIndex {
			indexName := field.TagSettings["index"]
			if indexName == "" {
				indexName = "idx_" + info.DBName + "_" + field.DBName
			}
			
			if info.Indexes[indexName] == nil {
				info.Indexes[indexName] = make([]string, 0)
			}
			info.Indexes[indexName] = append(info.Indexes[indexName], field.DBName)
		}
		
		if field.IsUnique {
			indexName := field.TagSettings["uniqueIndex"]
			if indexName == "" {
				indexName = "udx_" + info.DBName + "_" + field.DBName
			}
			
			if info.UniqueIndexes[indexName] == nil {
				info.UniqueIndexes[indexName] = make([]string, 0)
			}
			info.UniqueIndexes[indexName] = append(info.UniqueIndexes[indexName], field.DBName)
		}
	}
	
	return info, nil
}

// ExtractFields extracts field information from a struct
func ExtractFields(model interface{}, prefix string) ([]*FieldInfo, error) {
	modelType := IndirectType(TypeOf(model))
	if modelType.Kind() != reflect.Struct {
		return nil, errors.NewModelError("model must be a struct", nil).
			WithModel(fmt.Sprintf("%T", model))
	}
	
	// Check cache first
	fieldCacheLock.RLock()
	cachedFields, ok := fieldCache[modelType]
	fieldCacheLock.RUnlock()
	
	if ok {
		// Convert cache map to slice
		fields := make([]*FieldInfo, 0, len(cachedFields))
		for _, field := range cachedFields {
			// Skip embedded fields from results if prefix is set
			if prefix != "" && field.IsAnonymous {
				continue
			}
			
			// Clone to avoid modifying cached data
			fieldCopy := *field
			fields = append(fields, &fieldCopy)
		}
		return fields, nil
	}
	
	// Parse all fields
	numField := modelType.NumField()
	structFields := make([]*FieldInfo, 0, numField)
	fieldMap := make(map[string]*FieldInfo)
	
	for i := 0; i < numField; i++ {
		sf := modelType.Field(i)
		
		// Skip unexported fields
		if sf.PkgPath != "" && !sf.Anonymous {
			continue
		}
		
		fi := &FieldInfo{
			Name:       sf.Name,
			DBName:     ToSnakeCase(sf.Name),
			Type:       sf.Type,
			Index:      sf.Index,
			IsAnonymous: sf.Anonymous,
			RawTag:     string(sf.Tag),
			TagSettings: make(map[string]string),
		}
		
		// Handle anonymous (embedded) fields
		if sf.Anonymous {
			fieldType := IndirectType(sf.Type)
			if fieldType.Kind() == reflect.Struct {
				// Skip if this is an unexported embedded field from another package
				if sf.PkgPath != "" {
					continue
				}
				
				// Process embedded struct fields
				embeddedPrefix := prefix
				if embeddedPrefix == "" {
					embeddedPrefix = sf.Name
				} else {
					embeddedPrefix = embeddedPrefix + "." + sf.Name
				}
				
				embeddedFields, err := ExtractFields(reflect.New(fieldType).Elem().Interface(), embeddedPrefix)
				if err != nil {
					return nil, err
				}
				
				for _, ef := range embeddedFields {
					// Skip if field with same name already exists in the parent struct
					if _, exists := fieldMap[ef.Name]; !exists {
						ef.Index = append([]int{i}, ef.Index...)
						structFields = append(structFields, ef)
						fieldMap[ef.Name] = ef
					}
				}
				
				continue
			}
		}
		
		// Process field tags
		if tag, ok := sf.Tag.Lookup(TagKey); ok {
			fi.TagSettings = ParseTagSettings(tag)
			
			// Handle field name override
			if name, ok := fi.TagSettings["column"]; ok && name != "" {
				fi.DBName = name
			}
			
			// Handle special flags
			fi.IsPrimaryKey = HasTagOption(tag, "primary_key") || HasTagOption(tag, "primaryKey")
			fi.IsAutoIncrement = HasTagOption(tag, "auto_increment") || HasTagOption(tag, "autoIncrement")
			fi.IsUnique = HasTagOption(tag, "unique")
			fi.IsIndex = HasTagOption(tag, "index")
			fi.IsNotNull = HasTagOption(tag, "not_null") || HasTagOption(tag, "notNull")
			fi.IsIgnored = HasTagOption(tag, "-") || HasTagOption(tag, "ignore")
			fi.IsReadOnly = HasTagOption(tag, "readonly") || HasTagOption(tag, "readOnly")
			fi.IsWriteOnly = HasTagOption(tag, "writeonly") || HasTagOption(tag, "writeOnly")
			
			// Handle size specification
			if size, ok := fi.TagSettings["size"]; ok {
				fmt.Sscanf(size, "%d", &fi.Size)
			}
			
			// Handle precision and scale
			if precision, ok := fi.TagSettings["precision"]; ok {
				fmt.Sscanf(precision, "%d", &fi.Precision)
				
				if scale, ok := fi.TagSettings["scale"]; ok {
					fmt.Sscanf(scale, "%d", &fi.Scale)
				}
			}
			
			// Handle default value
			if def, ok := fi.TagSettings["default"]; ok {
				fi.Default = def
			}
			
			// Handle foreign key references
			if ref, ok := fi.TagSettings["references"]; ok {
				parts := strings.Split(ref, ".")
				if len(parts) == 2 {
					fi.Referenced = &ReferenceInfo{
						Model: parts[0],
						Field: parts[1],
					}
					
					if onDelete, ok := fi.TagSettings["onDelete"]; ok {
						fi.Referenced.OnDelete = onDelete
					}
					
					if onUpdate, ok := fi.TagSettings["onUpdate"]; ok {
						fi.Referenced.OnUpdate = onUpdate
					}
				}
			}
		}
		
		structFields = append(structFields, fi)
		fieldMap[fi.Name] = fi
	}
	
	// Update cache
	fieldCacheLock.Lock()
	fieldCache[modelType] = fieldMap
	fieldCacheLock.Unlock()
	
	return structFields, nil
}

// TypeOf returns the reflection Type of the value
// If the value is nil, it panics
func TypeOf(value interface{}) reflect.Type {
	valueType := reflect.TypeOf(value)
	if valueType == nil {
		panic(errors.NewInternalError("nil value passed to TypeOf", nil))
	}
	return valueType
}

// ValueOf returns the reflection Value of the value
// If the value is nil, it returns a zero Value and an error
func ValueOf(value interface{}) (reflect.Value, error) {
	if value == nil {
		return reflect.Value{}, errors.NewInternalError("nil value passed to ValueOf", nil)
	}
	return reflect.ValueOf(value), nil
}

// IndirectType dereferences pointer types to get the underlying type
func IndirectType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// IndirectValue dereferences pointer values to get the underlying value
func IndirectValue(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	return v
}

// ParseTagSettings parses tag string into a map of settings
func ParseTagSettings(tag string) map[string]string {
	settings := make(map[string]string)
	parts := strings.Split(tag, ";")
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		
		keyValue := strings.SplitN(part, ":", 2)
		key := strings.TrimSpace(keyValue[0])
		
		if key == "" {
			continue
		}
		
		var value string
		if len(keyValue) > 1 {
			value = strings.TrimSpace(keyValue[1])
		}
		
		settings[key] = value
	}
	
	return settings
}

// HasTagOption checks if a tag contains a specific option flag
func HasTagOption(tag, option string) bool {
	parts := strings.Split(tag, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == option {
			return true
		}
	}
	return false
}

// ToSnakeCase converts a camelCase or PascalCase string to snake_case
func ToSnakeCase(s string) string {
	if s == "" {
		return ""
	}
	
	var result strings.Builder
	result.Grow(len(s) + 5) // Allocate a bit more for underscores
	
	prevLower := false
	
	for i, r := range s {
		isLower := unicode.IsLower(r)
		
		if i > 0 {
			// If we encounter an uppercase letter after a lowercase, add underscore
			if !isLower && prevLower {
				result.WriteRune('_')
			}
			
			// If we encounter uppercase letters in sequence followed by a lowercase,
			// add an underscore before the last uppercase letter
			if isLower && i > 1 && !prevLower && unicode.IsUpper(rune(s[i-1])) && i > 2 && unicode.IsUpper(rune(s[i-2])) {
				l := result.Len()
				resultStr := result.String()
				result.Reset()
				result.WriteString(resultStr[:l-1])
				result.WriteRune('_')
				result.WriteRune(unicode.ToLower(rune(s[i-1])))
			}
		}
		
		result.WriteRune(unicode.ToLower(r))
		prevLower = isLower
	}
	
	return result.String()
}

// ToCamelCase converts a snake_case string to camelCase
func ToCamelCase(s string) string {
	if s == "" {
		return ""
	}
	
	var result strings.Builder
	result.Grow(len(s))
	
	capNext := false
	
	for i, r := range s {
		if r == '_' {
			capNext = true
			continue
		}
		
		if i == 0 {
			result.WriteRune(unicode.ToLower(r))
		} else if capNext {
			result.WriteRune(unicode.ToUpper(r))
			capNext = false
		} else {
			result.WriteRune(r)
		}
	}
	
	return result.String()
}

// ToPascalCase converts a snake_case string to PascalCase
func ToPascalCase(s string) string {
	if s == "" {
		return ""
	}
	
	var result strings.Builder
	result.Grow(len(s))
	
	capNext := true
	
	for _, r := range s {
		if r == '_' {
			capNext = true
			continue
		}
		
		if capNext {
			result.WriteRune(unicode.ToUpper(r))
			capNext = false
		} else {
			result.WriteRune(r)
		}
	}
	
	return result.String()
}

// CreateInstance creates a new instance of the given type
func CreateInstance(t reflect.Type) (interface{}, error) {
	if t == nil {
		return nil, errors.NewInternalError("nil type passed to CreateInstance", nil)
	}
	
	// Handle different kinds of types
	switch t.Kind() {
	case reflect.Ptr:
		// For pointer types, create an instance of the element type and return a pointer
		elem, err := CreateInstance(t.Elem())
		if err != nil {
			return nil, err
		}
		
		// Create a new pointer to the element
		ptr := reflect.New(reflect.TypeOf(elem))
		ptr.Elem().Set(reflect.ValueOf(elem))
		return ptr.Interface(), nil
		
	case reflect.Struct:
		// For struct types, create a new zero-initialized instance
		return reflect.New(t).Elem().Interface(), nil
		
	case reflect.Slice:
		// For slice types, create an empty slice
		return reflect.MakeSlice(t, 0, 0).Interface(), nil
		
	case reflect.Map:
		// For map types, create an empty map
		return reflect.MakeMap(t).Interface(), nil
		
	default:
		// For other types, create a zero-initialized value
		return reflect.Zero(t).Interface(), nil
	}
}

// GetFieldValues extracts field values from a struct into a map
func GetFieldValues(model interface{}, onlyFields ...string) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	
	// Get model information
	mi, err := ExtractModelInfo(model)
	if err != nil {
		return nil, err
	}
	
	// Get value of the model
	modelValue, err := ValueOf(model)
	if err != nil {
		return nil, err
	}
	modelValue = IndirectValue(modelValue)
	
	// Create a set of fields to include if onlyFields is specified
	includeFields := make(map[string]bool)
	for _, field := range onlyFields {
		includeFields[field] = true
	}
	
	// Extract values for each field
	for name, field := range mi.Fields {
		// Skip ignored fields, read-only fields, or fields not in the include list
		if field.IsIgnored || field.IsReadOnly {
			continue
		}
		
		if len(onlyFields) > 0 && !includeFields[name] {
			continue
		}
		
		// Get field value by field index
		fieldValue := modelValue.FieldByIndex(field.Index)
		
		// Add to result map, using the database field name as the key
		result[field.DBName] = fieldValue.Interface()
	}
	
	return result, nil
}

// SetFieldValues sets field values on a struct from a map
func SetFieldValues(model interface{}, values map[string]interface{}) error {
	// Get model information
	mi, err := ExtractModelInfo(model)
	if err != nil {
		return err
	}
	
	// Get value of the model
	modelValue, err := ValueOf(model)
	if err != nil {
		return err
	}
	
	// Ensure model is addressable
	if !modelValue.CanAddr() {
		return errors.NewInternalError("model must be addressable (a pointer)", nil)
	}
	
	modelValue = IndirectValue(modelValue)
	
	// Set values for each field
	for dbName, value := range values {
		// Find field by database name
		field, ok := mi.FieldsByDBName[dbName]
		if !ok {
			// Field not found, skip it
			continue
		}
		
		// Skip ignored or write-only fields
		if field.IsIgnored || field.IsWriteOnly {
			continue
		}
		
		// Get field value by field index
		fieldValue := modelValue.FieldByIndex(field.Index)
		
		// Only set if field is addressable and can be set
		if fieldValue.CanAddr() && fieldValue.CanSet() {
			// Convert value to correct type if needed
			sourceValue := reflect.ValueOf(value)
			if sourceValue.Type().AssignableTo(fieldValue.Type()) {
				fieldValue.Set(sourceValue)
			} else {
				// Try to convert between compatible types
				if sourceValue.Type().ConvertibleTo(fieldValue.Type()) {
					fieldValue.Set(sourceValue.Convert(fieldValue.Type()))
				} else {
					return errors.NewModelError("cannot set field value: incompatible types", nil).
						WithField(field.Name).
						WithValue(value)
				}
			}
		}
	}
	
	return nil
}

// ValidateStruct validates a struct against its validation tags
func ValidateStruct(model interface{}) error {
	// Get model information
	mi, err := ExtractModelInfo(model)
	if err != nil {
		return err
	}
	
	// Get value of the model
	modelValue, err := ValueOf(model)
	if err != nil {
		return err
	}
	modelValue = IndirectValue(modelValue)
	
	// Validate each field
	for _, field := range mi.Fields {
		// Skip ignored fields
		if field.IsIgnored {
			continue
		}
		
		// Get field value by field index
		fieldValue := modelValue.FieldByIndex(field.Index)
		
		// Check not null constraint
		if field.IsNotNull {
			isZero := fieldValue.IsZero()
			if isZero {
				return errors.NewValidationError("field cannot be null", nil).
					WithField(field.Name).
					WithModel(mi.Name)
			}
		}
		
		// Add more validations here as needed
		// (e.g., regex patterns, min/max values, custom validations)
	}
	
	return nil
}

// ClearCache clears the reflection cache
func ClearCache() {
	fieldCacheLock.Lock()
	defer fieldCacheLock.Unlock()
	
	fieldCache = make(map[reflect.Type]map[string]*FieldInfo)
}

var (
	matchFirstCapRe = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCapRe   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

// ToSnakeCaseRegex converts a string to snake_case using regex
// This is an alternative implementation using regex
func ToSnakeCaseRegex(str string) string {
	snake := matchFirstCapRe.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCapRe.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

// IsStructOrStructPtr checks if a value is a struct or pointer to struct
func IsStructOrStructPtr(value interface{}) bool {
	t := reflect.TypeOf(value)
	if t == nil {
		return false
	}
	
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	
	return t.Kind() == reflect.Struct
}
