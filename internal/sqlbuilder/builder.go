// Copyright (c) 2025 Yahya Qadeer Dar. All rights reserved.
// Use of this source code is governed by an Apache 2.0 license that can be found in the LICENSE file.

// Package sqlbuilder provides utilities for constructing SQL queries in a
// database-agnostic way. It handles parameter binding, SQL injection protection,
// and dialect-specific syntax.
package sqlbuilder

import (
	"fmt"
	"strings"
	"sync"
	"time"
	
	"github.com/YahyaDar/ORigaMi/errors"
)

// Dialect represents SQL dialect-specific behavior
type Dialect interface {
	// Placeholder returns the placeholder for a parameter at the given position
	Placeholder(pos int) string
	
	// Quote quotes an identifier (table, column)
	Quote(identifier string) string
	
	// EscapeLike escapes special characters in LIKE patterns
	EscapeLike(value string) string
	
	// FormatBool formats a boolean value for this dialect
	FormatBool(value bool) string
	
	// FormatTime formats a time value for this dialect
	FormatTime(value time.Time) string
	
	// LimitOffset returns LIMIT/OFFSET SQL for the dialect
	LimitOffset(limit, offset int64) string
	
	// DriverName returns the name of the SQL driver for this dialect
	DriverName() string
	
	// InsertReturning generates SQL to return inserted IDs
	InsertReturning(query string, pkColumn string) string
	
	// SupportUpsert returns whether the dialect supports upsert operations
	SupportUpsert() bool
}

// Builder constructs SQL queries
type Builder struct {
	// dialect is the SQL dialect to use
	dialect Dialect
	
	// buffer accumulates the SQL query
	buffer strings.Builder
	
	// args stores the query arguments
	args []interface{}
	
	// argPosition tracks the current argument position
	argPosition int
	
	// sections tracks sections of the SQL query
	sections map[string]int
	
	// Lock for thread safety
	mu sync.Mutex
}

// NewBuilder creates a new SQL builder with the given dialect
func NewBuilder(dialect Dialect) *Builder {
	return &Builder{
		dialect:  dialect,
		args:     make([]interface{}, 0),
		sections: make(map[string]int),
	}
}

// Reset clears the builder for reuse
func (b *Builder) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.buffer.Reset()
	b.args = b.args[:0]
	b.argPosition = 0
	b.sections = make(map[string]int)
}

// SQL returns the built SQL query
func (b *Builder) SQL() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	return b.buffer.String()
}

// Args returns the query arguments
func (b *Builder) Args() []interface{} {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	// Return a copy to prevent modification
	argsCopy := make([]interface{}, len(b.args))
	copy(argsCopy, b.args)
	return argsCopy
}

// Append adds a string to the query
func (b *Builder) Append(s string) *Builder {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.buffer.WriteString(s)
	return b
}

// AppendQuoted adds a quoted identifier to the query
func (b *Builder) AppendQuoted(identifier string) *Builder {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.buffer.WriteString(b.dialect.Quote(identifier))
	return b
}

// AppendPlaceholder adds a parameter placeholder to the query
func (b *Builder) AppendPlaceholder() *Builder {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.argPosition++
	b.buffer.WriteString(b.dialect.Placeholder(b.argPosition))
	return b
}

// Arg adds an argument to the query
func (b *Builder) Arg(arg interface{}) *Builder {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.args = append(b.args, arg)
	return b
}

// AppendWithArgs adds a SQL fragment with arguments
func (b *Builder) AppendWithArgs(sql string, args ...interface{}) *Builder {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.buffer.WriteString(sql)
	b.args = append(b.args, args...)
	return b
}

// AppendWithPlaceholders adds a string to the query with placeholders for arguments
func (b *Builder) AppendWithPlaceholders(sql string, args ...interface{}) *Builder {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	// Replace '?' with dialect-specific placeholders
	parts := strings.Split(sql, "?")
	for i, part := range parts {
		b.buffer.WriteString(part)
		if i < len(parts)-1 {
			b.argPosition++
			b.buffer.WriteString(b.dialect.Placeholder(b.argPosition))
		}
	}
	
	b.args = append(b.args, args...)
	return b
}

// MarkSection marks the current position in the query with a name
func (b *Builder) MarkSection(name string) *Builder {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.sections[name] = b.buffer.Len()
	return b
}

// ReplaceSection replaces a previously marked section with new content
func (b *Builder) ReplaceSection(name, content string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	pos, ok := b.sections[name]
	if !ok {
		return errors.NewInternalError("section not marked", nil).
			WithContext("section", name)
	}
	
	// Get the current SQL
	sql := b.buffer.String()
	
	// Reset the buffer
	b.buffer.Reset()
	
	// Write the part before the section
	b.buffer.WriteString(sql[:pos])
	
	// Write the new section content
	b.buffer.WriteString(content)
	
	// Write the part after the section
	b.buffer.WriteString(sql[pos:])
	
	return nil
}

// Where adds a WHERE clause to the query
func (b *Builder) Where(condition string, args ...interface{}) *Builder {
	b.mu.Lock()
	
	// Check if WHERE has already been added
	sql := b.buffer.String()
	hasWhere := strings.Contains(strings.ToUpper(sql), " WHERE ")
	
	b.mu.Unlock()
	
	if hasWhere {
		return b.Append(" AND ").AppendWithPlaceholders(condition, args...)
	}
	
	return b.Append(" WHERE ").AppendWithPlaceholders(condition, args...)
}

// Select starts a SELECT query
func (b *Builder) Select(columns ...string) *Builder {
	b.Reset()
	b.Append("SELECT ")
	
	if len(columns) == 0 {
		b.Append("*")
	} else {
		for i, col := range columns {
			if i > 0 {
				b.Append(", ")
			}
			
			// If it contains a space, assume it's a raw expression or alias
			if strings.Contains(col, " ") || strings.Contains(col, "(") {
				b.Append(col)
			} else {
				b.AppendQuoted(col)
			}
		}
	}
	
	return b
}

// From adds a FROM clause to the query
func (b *Builder) From(table string) *Builder {
	return b.Append(" FROM ").AppendQuoted(table)
}

// Join adds a JOIN clause to the query
func (b *Builder) Join(joinType, table, condition string) *Builder {
	return b.Append(" ").
		Append(joinType).
		Append(" JOIN ").
		AppendQuoted(table).
		Append(" ON ").
		Append(condition)
}

// OrderBy adds an ORDER BY clause to the query
func (b *Builder) OrderBy(columns ...string) *Builder {
	if len(columns) == 0 {
		return b
	}
	
	b.Append(" ORDER BY ")
	
	for i, col := range columns {
		if i > 0 {
			b.Append(", ")
		}
		
		// Check for descending order indicator
		if strings.HasSuffix(col, " DESC") || strings.HasSuffix(col, " desc") {
			parts := strings.Fields(col)
			b.AppendQuoted(parts[0]).Append(" DESC")
		} else if strings.HasSuffix(col, " ASC") || strings.HasSuffix(col, " asc") {
			parts := strings.Fields(col)
			b.AppendQuoted(parts[0]).Append(" ASC")
		} else {
			b.AppendQuoted(col)
		}
	}
	
	return b
}

// GroupBy adds a GROUP BY clause to the query
func (b *Builder) GroupBy(columns ...string) *Builder {
	if len(columns) == 0 {
		return b
	}
	
	b.Append(" GROUP BY ")
	
	for i, col := range columns {
		if i > 0 {
			b.Append(", ")
		}
		
		// If it contains a function or special syntax, don't quote it
		if strings.Contains(col, "(") {
			b.Append(col)
		} else {
			b.AppendQuoted(col)
		}
	}
	
	return b
}

// Having adds a HAVING clause to the query
func (b *Builder) Having(condition string, args ...interface{}) *Builder {
	return b.Append(" HAVING ").AppendWithPlaceholders(condition, args...)
}

// Limit adds a LIMIT clause to the query
func (b *Builder) Limit(limit int64) *Builder {
	return b.Append(b.dialect.LimitOffset(limit, -1))
}

// Offset adds an OFFSET clause to the query
func (b *Builder) Offset(offset int64) *Builder {
	return b.Append(b.dialect.LimitOffset(-1, offset))
}

// LimitOffset adds both LIMIT and OFFSET clauses to the query
func (b *Builder) LimitOffset(limit, offset int64) *Builder {
	return b.Append(b.dialect.LimitOffset(limit, offset))
}

// Insert starts an INSERT query
func (b *Builder) Insert(table string) *Builder {
	b.Reset()
	return b.Append("INSERT INTO ").AppendQuoted(table)
}

// Columns adds column names to an INSERT query
func (b *Builder) Columns(columns ...string) *Builder {
	b.Append(" (")
	
	for i, col := range columns {
		if i > 0 {
			b.Append(", ")
		}
		b.AppendQuoted(col)
	}
	
	return b.Append(")")
}

// Values adds values to an INSERT query
func (b *Builder) Values(valuesList ...interface{}) *Builder {
	b.Append(" VALUES (")
	
	for i, value := range valuesList {
		if i > 0 {
			b.Append(", ")
		}
		
		// Add placeholder and argument
		b.AppendPlaceholder().Arg(value)
	}
	
	return b.Append(")")
}

// MultipleValues adds multiple rows of values to an INSERT query
func (b *Builder) MultipleValues(rows [][]interface{}) *Builder {
	b.Append(" VALUES ")
	
	for i, row := range rows {
		if i > 0 {
			b.Append(", ")
		}
		
		b.Append("(")
		for j, value := range row {
			if j > 0 {
				b.Append(", ")
			}
			
			// Add placeholder and argument
			b.AppendPlaceholder().Arg(value)
		}
		b.Append(")")
	}
	
	return b
}

// Update starts an UPDATE query
func (b *Builder) Update(table string) *Builder {
	b.Reset()
	return b.Append("UPDATE ").AppendQuoted(table)
}

// Set adds a SET clause to an UPDATE query
func (b *Builder) Set(column string, value interface{}) *Builder {
	// Check if SET has already been added
	sql := b.SQL()
	if !strings.Contains(strings.ToUpper(sql), " SET ") {
		b.Append(" SET ")
	} else {
		b.Append(", ")
	}
	
	return b.AppendQuoted(column).Append(" = ").AppendPlaceholder().Arg(value)
}

// SetMap adds multiple SET clauses from a map to an UPDATE query
func (b *Builder) SetMap(values map[string]interface{}) *Builder {
	// Check if SET has already been added
	sql := b.SQL()
	if !strings.Contains(strings.ToUpper(sql), " SET ") {
		b.Append(" SET ")
	}
	
	first := !strings.Contains(sql, "=")
	
	for column, value := range values {
		if !first {
			b.Append(", ")
		}
		
		b.AppendQuoted(column).Append(" = ").AppendPlaceholder().Arg(value)
		first = false
	}
	
	return b
}

// Delete starts a DELETE query
func (b *Builder) Delete() *Builder {
	b.Reset()
	return b.Append("DELETE")
}

// From adds a FROM clause to a DELETE query
func (b *Builder) DeleteFrom(table string) *Builder {
	b.Reset()
	return b.Append("DELETE FROM ").AppendQuoted(table)
}

// Returning adds a RETURNING clause to an INSERT, UPDATE, or DELETE query
func (b *Builder) Returning(column string) *Builder {
	sql := b.SQL()
	
	// Only add if the dialect supports it
	if strings.HasPrefix(sql, "INSERT") {
		return b.Append(b.dialect.InsertReturning(sql, column))
	}
	
	// For other statements, just append RETURNING if supported
	if b.dialect.SupportUpsert() {
		return b.Append(" RETURNING ").AppendQuoted(column)
	}
	
	return b
}

// Union adds a UNION clause between two queries
func (b *Builder) Union(otherSQL string, otherArgs ...interface{}) *Builder {
	return b.Append(" UNION ").AppendWithArgs(otherSQL, otherArgs...)
}

// UnionAll adds a UNION ALL clause between two queries
func (b *Builder) UnionAll(otherSQL string, otherArgs ...interface{}) *Builder {
	return b.Append(" UNION ALL ").AppendWithArgs(otherSQL, otherArgs...)
}

// Count creates a COUNT query
func (b *Builder) Count(column string) *Builder {
	b.Reset()
	
	if column == "" || column == "*" {
		return b.Append("SELECT COUNT(*)")
	}
	
	return b.Append("SELECT COUNT(").AppendQuoted(column).Append(")")
}

// CreateTable starts a CREATE TABLE query
func (b *Builder) CreateTable(table string, ifNotExists bool) *Builder {
	b.Reset()
	b.Append("CREATE TABLE ")
	
	if ifNotExists {
		b.Append("IF NOT EXISTS ")
	}
	
	return b.AppendQuoted(table)
}

// AddColumn adds a column definition to a CREATE TABLE query
func (b *Builder) AddColumn(column string, dataType string, constraints ...string) *Builder {
	// Check if any column has already been added
	sql := b.SQL()
	if strings.Contains(sql, "(") {
		b.Append(", ")
	} else {
		b.Append(" (")
	}
	
	b.AppendQuoted(column).Append(" ").Append(dataType)
	
	for _, constraint := range constraints {
		b.Append(" ").Append(constraint)
	}
	
	return b
}

// PrimaryKey adds a PRIMARY KEY constraint to a CREATE TABLE query
func (b *Builder) PrimaryKey(columns ...string) *Builder {
	b.Append(", PRIMARY KEY (")
	
	for i, col := range columns {
		if i > 0 {
			b.Append(", ")
		}
		b.AppendQuoted(col)
	}
	
	return b.Append(")")
}

// UniqueKey adds a UNIQUE constraint to a CREATE TABLE query
func (b *Builder) UniqueKey(name string, columns ...string) *Builder {
	b.Append(", CONSTRAINT ").AppendQuoted(name).Append(" UNIQUE (")
	
	for i, col := range columns {
		if i > 0 {
			b.Append(", ")
		}
		b.AppendQuoted(col)
	}
	
	return b.Append(")")
}

// ForeignKey adds a FOREIGN KEY constraint to a CREATE TABLE query
func (b *Builder) ForeignKey(name, column, refTable, refColumn string, onDelete, onUpdate string) *Builder {
	b.Append(", CONSTRAINT ").AppendQuoted(name).
		Append(" FOREIGN KEY (").AppendQuoted(column).Append(")").
		Append(" REFERENCES ").AppendQuoted(refTable).Append("(").AppendQuoted(refColumn).Append(")")
	
	if onDelete != "" {
		b.Append(" ON DELETE ").Append(onDelete)
	}
	
	if onUpdate != "" {
		b.Append(" ON UPDATE ").Append(onUpdate)
	}
	
	return b
}

// CloseParenthesis closes the parenthesis in a CREATE TABLE query
func (b *Builder) CloseParenthesis() *Builder {
	return b.Append(")")
}

// AlterTable starts an ALTER TABLE query
func (b *Builder) AlterTable(table string) *Builder {
	b.Reset()
	return b.Append("ALTER TABLE ").AppendQuoted(table)
}

// AddColumnToTable adds a column to an existing table
func (b *Builder) AddColumnToTable(column, dataType string, constraints ...string) *Builder {
	b.Append(" ADD COLUMN ").AppendQuoted(column).Append(" ").Append(dataType)
	
	for _, constraint := range constraints {
		b.Append(" ").Append(constraint)
	}
	
	return b
}

// RenameTable renames a table
func (b *Builder) RenameTable(newName string) *Builder {
	return b.Append(" RENAME TO ").AppendQuoted(newName)
}

// DropColumn drops a column from a table
func (b *Builder) DropColumn(column string) *Builder {
	return b.Append(" DROP COLUMN ").AppendQuoted(column)
}

// CreateIndex starts a CREATE INDEX query
func (b *Builder) CreateIndex(name string, table string, unique bool) *Builder {
	b.Reset()
	b.Append("CREATE ")
	
	if unique {
		b.Append("UNIQUE ")
	}
	
	return b.Append("INDEX ").AppendQuoted(name).Append(" ON ").AppendQuoted(table)
}

// IndexColumns adds column list to a CREATE INDEX query
func (b *Builder) IndexColumns(columns ...string) *Builder {
	b.Append(" (")
	
	for i, col := range columns {
		if i > 0 {
			b.Append(", ")
		}
		
		// Parse column name and direction
		parts := strings.Fields(col)
		b.AppendQuoted(parts[0])
		
		if len(parts) > 1 {
			b.Append(" ").Append(parts[1])
		}
	}
	
	return b.Append(")")
}

// DropTable starts a DROP TABLE query
func (b *Builder) DropTable(table string, ifExists bool) *Builder {
	b.Reset()
	b.Append("DROP TABLE ")
	
	if ifExists {
		b.Append("IF EXISTS ")
	}
	
	return b.AppendQuoted(table)
}

// DropIndex starts a DROP INDEX query
func (b *Builder) DropIndex(name string, ifExists bool) *Builder {
	b.Reset()
	b.Append("DROP INDEX ")
	
	if ifExists {
		b.Append("IF EXISTS ")
	}
	
	return b.AppendQuoted(name)
}

// Transaction related queries

// BeginTransaction returns SQL to begin a transaction
func (b *Builder) BeginTransaction() *Builder {
	b.Reset()
	return b.Append("BEGIN")
}

// CommitTransaction returns SQL to commit a transaction
func (b *Builder) CommitTransaction() *Builder {
	b.Reset()
	return b.Append("COMMIT")
}

// RollbackTransaction returns SQL to rollback a transaction
func (b *Builder) RollbackTransaction() *Builder {
	b.Reset()
	return b.Append("ROLLBACK")
}

// Raw adds raw SQL to the query
func (b *Builder) Raw(sql string) *Builder {
	return b.Append(sql)
}

// RawWithArgs adds raw SQL with arguments to the query
func (b *Builder) RawWithArgs(sql string, args ...interface{}) *Builder {
	return b.AppendWithArgs(sql, args...)
}

// WithArgs adds arguments to the query without modifying the SQL
func (b *Builder) WithArgs(args ...interface{}) *Builder {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.args = append(b.args, args...)
	return b
}

// ToSQL returns the SQL query and arguments
func (b *Builder) ToSQL() (string, []interface{}) {
	return b.SQL(), b.Args()
}

// String returns the SQL query as a string (implements fmt.Stringer)
func (b *Builder) String() string {
	return b.SQL()
}

// EscapeLike escapes special characters in LIKE patterns
func (b *Builder) EscapeLike(value string) string {
	return b.dialect.EscapeLike(value)
}

// QuotedTableColumn returns a properly quoted table.column string
func (b *Builder) QuotedTableColumn(table, column string) string {
	return fmt.Sprintf("%s.%s", b.dialect.Quote(table), b.dialect.Quote(column))
}

// Subquery adds a subquery
func (b *Builder) Subquery(subquery *Builder, alias string) *Builder {
	b.Append("(").Append(subquery.SQL()).Append(")")
	
	if alias != "" {
		b.Append(" AS ").AppendQuoted(alias)
	}
	
	// Add subquery args
	b.mu.Lock()
	b.args = append(b.args, subquery.args...)
	b.mu.Unlock()
	
	return b
}

// Exists adds an EXISTS clause with a subquery
func (b *Builder) Exists(subquery *Builder) *Builder {
	b.Append("EXISTS (").Append(subquery.SQL()).Append(")")
	
	// Add subquery args
	b.mu.Lock()
	b.args = append(b.args, subquery.args...)
	b.mu.Unlock()
	
	return b
}

// NotExists adds a NOT EXISTS clause with a subquery
func (b *Builder) NotExists(subquery *Builder) *Builder {
	b.Append("NOT EXISTS (").Append(subquery.SQL()).Append(")")
	
	// Add subquery args
	b.mu.Lock()
	b.args = append(b.args, subquery.args...)
	b.mu.Unlock()
	
	return b
}

// PostgresDialect implements the Dialect interface for PostgreSQL
type PostgresDialect struct{}

// Placeholder returns the placeholder for a parameter at the given position for PostgreSQL
func (d *PostgresDialect) Placeholder(pos int) string {
	return fmt.Sprintf("$%d", pos)
}

// Quote quotes an identifier for PostgreSQL
func (d *PostgresDialect) Quote(identifier string) string {
	// Handle table.column format
	if strings.Contains(identifier, ".") {
		parts := strings.Split(identifier, ".")
		var quoted []string
		for _, part := range parts {
			quoted = append(quoted, fmt.Sprintf(`"%s"`, part))
		}
		return strings.Join(quoted, ".")
	}
	
	return fmt.Sprintf(`"%s"`, identifier)
}

// EscapeLike escapes special characters in LIKE patterns for PostgreSQL
func (d *PostgresDialect) EscapeLike(value string) string {
	// PostgreSQL uses backslash as the default escape character
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	value = strings.ReplaceAll(value, `_`, `\_`)
	return value
}

// FormatBool formats a boolean value for PostgreSQL
func (d *PostgresDialect) FormatBool(value bool) string {
	if value {
		return "TRUE"
	}
	return "FALSE"
}

// FormatTime formats a time value for PostgreSQL
func (d *PostgresDialect) FormatTime(value time.Time) string {
	return "'" + value.Format("2006-01-02 15:04:05.999999") + "'"
}

// LimitOffset returns LIMIT/OFFSET SQL for PostgreSQL
func (d *PostgresDialect) LimitOffset(limit, offset int64) string {
	var sql string
	if limit >= 0 {
		sql = fmt.Sprintf(" LIMIT %d", limit)
	}
	if offset >= 0 {
		sql += fmt.Sprintf(" OFFSET %d", offset)
	}
	return sql
}

// DriverName returns the name of the SQL driver for PostgreSQL
func (d *PostgresDialect) DriverName() string {
	return "postgres"
}

// InsertReturning generates SQL to return inserted IDs for PostgreSQL
func (d *PostgresDialect) InsertReturning(query string, pkColumn string) string {
	return fmt.Sprintf(" RETURNING %s", d.Quote(pkColumn))
}

// SupportUpsert returns whether PostgreSQL supports upsert operations
func (d *PostgresDialect) SupportUpsert() bool {
	return true
}

// MySQLDialect implements the Dialect interface for MySQL
type MySQLDialect struct{}

// Placeholder returns the placeholder for a parameter at the given position for MySQL
func (d *MySQLDialect) Placeholder(pos int) string {
	return "?"
}

// Quote quotes an identifier for MySQL
func (d *MySQLDialect) Quote(identifier string) string {
	// Handle table.column format
	if strings.Contains(identifier, ".") {
		parts := strings.Split(identifier, ".")
		var quoted []string
		for _, part := range parts {
			quoted = append(quoted, fmt.Sprintf("`%s`", part))
		}
		return strings.Join(quoted, ".")
	}
	
	return fmt.Sprintf("`%s`", identifier)
}

// EscapeLike escapes special characters in LIKE patterns for MySQL
func (d *MySQLDialect) EscapeLike(value string) string {
	// MySQL allows custom escape character
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	value = strings.ReplaceAll(value, `_`, `\_`)
	return value
}

// FormatBool formats a boolean value for MySQL
func (d *MySQLDialect) FormatBool(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

// FormatTime formats a time value for MySQL
func (d *MySQLDialect) FormatTime(value time.Time) string {
	return "'" + value.Format("2006-01-02 15:04:05.999999") + "'"
}

// LimitOffset returns LIMIT/OFFSET SQL for MySQL
func (d *MySQLDialect) LimitOffset(limit, offset int64) string {
	var sql string
	if limit >= 0 {
		if offset >= 0 {
			sql = fmt.Sprintf(" LIMIT %d, %d", offset, limit)
		} else {
			sql = fmt.Sprintf(" LIMIT %d", limit)
		}
	} else if offset >= 0 {
		// MySQL requires a large LIMIT when using just OFFSET
		sql = fmt.Sprintf(" LIMIT %d, 18446744073709551615", offset)
	}
	return sql
}

// DriverName returns the name of the SQL driver for MySQL
func (d *MySQLDialect) DriverName() string {
	return "mysql"
}

// InsertReturning generates SQL to return inserted IDs for MySQL
// MySQL doesn't support RETURNING, but we can use LAST_INSERT_ID()
func (d *MySQLDialect) InsertReturning(query string, pkColumn string) string {
	return ""
}

// SupportUpsert returns whether MySQL supports upsert operations
func (d *MySQLDialect) SupportUpsert() bool {
	return true
}

// SQLiteDialect implements the Dialect interface for SQLite
type SQLiteDialect struct{}

// Placeholder returns the placeholder for a parameter at the given position for SQLite
func (d *SQLiteDialect) Placeholder(pos int) string {
	return "?"
}

// Quote quotes an identifier for SQLite
func (d *SQLiteDialect) Quote(identifier string) string {
	// Handle table.column format
	if strings.Contains(identifier, ".") {
		parts := strings.Split(identifier, ".")
		var quoted []string
		for _, part := range parts {
			quoted = append(quoted, fmt.Sprintf(`"%s"`, part))
		}
		return strings.Join(quoted, ".")
	}
	
	return fmt.Sprintf(`"%s"`, identifier)
}

// EscapeLike escapes special characters in LIKE patterns for SQLite
func (d *SQLiteDialect) EscapeLike(value string) string {
	// SQLite allows custom escape character
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	value = strings.ReplaceAll(value, `_`, `\_`)
	return value
}

// FormatBool formats a boolean value for SQLite
func (d *SQLiteDialect) FormatBool(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

// FormatTime formats a time value for SQLite
func (d *SQLiteDialect) FormatTime(value time.Time) string {
	return "'" + value.Format("2006-01-02 15:04:05.999999") + "'"
}

// LimitOffset returns LIMIT/OFFSET SQL for SQLite
func (d *SQLiteDialect) LimitOffset(limit, offset int64) string {
	var sql string
	if limit >= 0 {
		sql = fmt.Sprintf(" LIMIT %d", limit)
	}
	if offset >= 0 {
		sql += fmt.Sprintf(" OFFSET %d", offset)
	}
	return sql
}

// DriverName returns the name of the SQL driver for SQLite
func (d *SQLiteDialect) DriverName() string {
	return "sqlite3"
}

// InsertReturning generates SQL to return inserted IDs for SQLite
func (d *SQLiteDialect) InsertReturning(query string, pkColumn string) string {
	return ""
}

// SupportUpsert returns whether SQLite supports upsert operations
func (d *SQLiteDialect) SupportUpsert() bool {
	return true
}

// NewPostgresBuilder creates a new SQL builder for PostgreSQL
func NewPostgresBuilder() *Builder {
	return NewBuilder(&PostgresDialect{})
}

// NewMySQLBuilder creates a new SQL builder for MySQL
func NewMySQLBuilder() *Builder {
	return NewBuilder(&MySQLDialect{})
}

// NewSQLiteBuilder creates a new SQL builder for SQLite
func NewSQLiteBuilder() *Builder {
	return NewBuilder(&SQLiteDialect{})
}

// GetBuilderForDialect returns a builder for the given dialect name
func GetBuilderForDialect(dialect string) (*Builder, error) {
	switch strings.ToLower(dialect) {
	case "postgres", "postgresql":
		return NewPostgresBuilder(), nil
	case "mysql":
		return NewMySQLBuilder(), nil
	case "sqlite", "sqlite3":
		return NewSQLiteBuilder(), nil
	default:
		return nil, fmt.Errorf("unsupported dialect: %s", dialect)
	}
}
