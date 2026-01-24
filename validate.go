package eorm

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidateTableName validates if table name is valid (public interface)
// Can be called externally to validate table names in advance
func ValidateTableName(table string) error {
	return validateIdentifier(table)
}

// validateSafeSQL 检查 SQL 片段中是否包含潜在的注入风险字符（如分号、注释符等）
// 适用于 Select, OrderBy, Join Condition 等直接拼接的部分
func validateSafeSQL(sqlPart string) error {
	if sqlPart == "" {
		return nil
	}

	// 严禁分号（防止多语句执行）
	if strings.Contains(sqlPart, ";") {
		return fmt.Errorf("unsafe SQL detected: semicolon not allowed in SQL fragment")
	}

	// 严禁注释符（防止语法截断）
	if strings.Contains(sqlPart, "--") || strings.Contains(sqlPart, "/*") {
		return fmt.Errorf("unsafe SQL detected: comments not allowed in SQL fragment")
	}

	return nil
}

// Pre-compiled regular expressions for better performance
var (
	// identifierPattern matches valid SQL identifiers
	// Supported formats: table_name, schema.table_name
	// Rules: starts with letter or underscore, followed by letters/digits/underscores
	identifierPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*(\.[a-zA-Z_][a-zA-Z0-9_]*)?$`)
)

const (
	// Maximum identifier length (most databases limit between 64-128)
	maxIdentifierLength = 128
)

// ErrInvalidTableName represents an invalid table name error
type ErrInvalidTableName struct {
	Name   string
	Reason string
}

func (e *ErrInvalidTableName) Error() string {
	return fmt.Sprintf("invalid table name '%s': %s", e.Name, e.Reason)
}

// validateIdentifier validates SQL identifiers (table names/column names etc.)
// Rules:
//   - Length between 1-128 characters
//   - Starts with letter or underscore
//   - Contains only letters, digits, underscores
//   - Optional support for schema.table format
//
// Returns error if identifier is invalid
func validateIdentifier(name string) error {
	if name == "" {
		return &ErrInvalidTableName{Name: name, Reason: "name cannot be empty"}
	}

	if len(name) > maxIdentifierLength {
		return &ErrInvalidTableName{Name: name, Reason: fmt.Sprintf("name exceeds maximum length of %d characters", maxIdentifierLength)}
	}

	if !identifierPattern.MatchString(name) {
		return &ErrInvalidTableName{Name: name, Reason: "name contains invalid characters or format (only letters, numbers, underscores allowed; must start with letter or underscore; optional schema.table format)"}
	}

	return nil
}
