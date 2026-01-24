package eorm

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
)

// ToJson converts any model, struct or Record to a JSON string.
// It is designed to be highly robust and safe:
// 1. Handles nil and typed nil pointers gracefully (returns "{}").
// 2. Disables HTML escaping for cleaner output in non-browser contexts.
// 3. Catches marshaling errors to prevent panics.
func ToJson(v interface{}) string {
	if isNil(v) {
		return "{}"
	}

	// Use a buffer and encoder to customize behavior
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false) // Keep symbols like <, >, & as is

	if err := enc.Encode(v); err != nil {
		return "{}"
	}

	// json.Encoder adds a newline at the end, let's trim it
	res := buf.Bytes()
	if len(res) > 0 && res[len(res)-1] == '\n' {
		res = res[:len(res)-1]
	}

	return string(res)
}

// isNil checks if an interface is truly nil, including typed nil pointers.
func isNil(i interface{}) bool {
	if i == nil {
		return true
	}
	v := reflect.ValueOf(i)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return v.IsNil()
	}
	return false
}

// isZeroValue 检查一个值是否为零值
// 零值包括：nil, 0, 0.0, false, ""
func isZeroValue(val interface{}) bool {
	if val == nil {
		return true
	}

	v := reflect.ValueOf(val)
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.String:
		return v.String() == ""
	case reflect.Bool:
		return !v.Bool()
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	default:
		return false
	}
}

// findKeywordIgnoringQuotes 在 SQL 中寻找指定的关键字，自动跳过引号内的字符串和注释，不区分大小写
// direction: 1 表示从前往后找，-1 表示从后往前找 (从后往前暂不支持注释，实际在 eorm 中主要用到正向寻找边界)
func findKeywordIgnoringQuotes(sql, keyword string, direction int) int {
	kwLen := len(keyword)
	if kwLen == 0 || len(sql) < kwLen {
		return -1
	}

	upperSQL := strings.ToUpper(sql)
	upperKW := strings.ToUpper(keyword)

	inSingleQuote := false
	inDoubleQuote := false
	inSingleLineComment := false
	inMultiLineComment := false
	escaped := false

	// 为简化逻辑并保证 100% 正确，无论是正向还是反向，我们都采用正向扫描状态机
	// 如果是反向找，我们通过一次完整正向扫描记录最后一个合法的匹配位置
	lastPos := -1

	for i := 0; i <= len(sql)-kwLen; i++ {
		char := sql[i]

		// 1. 处理注释状态
		if !inSingleQuote && !inDoubleQuote {
			if !inSingleLineComment && !inMultiLineComment {
				// 尝试进入单行注释
				if i+1 < len(sql) && sql[i:i+2] == "--" {
					inSingleLineComment = true
					i++ // 跳过 -
					continue
				}
				// 尝试进入多行注释
				if i+1 < len(sql) && sql[i:i+2] == "/*" {
					inMultiLineComment = true
					i++ // 跳过 *
					continue
				}
			} else if inSingleLineComment {
				if char == '\n' {
					inSingleLineComment = false
				}
				continue
			} else if inMultiLineComment {
				if i+1 < len(sql) && sql[i:i+2] == "*/" {
					inMultiLineComment = false
					i++
				}
				continue
			}
		}

		// 2. 处理字符串状态
		if !inSingleLineComment && !inMultiLineComment {
			if char == '\'' && !inDoubleQuote && !escaped {
				// 处理标准 SQL 转义 ''
				if i+1 < len(sql) && sql[i+1] == '\'' {
					i++ // 跳过下一个单引号
				} else {
					inSingleQuote = !inSingleQuote
				}
			} else if char == '"' && !inSingleQuote && !escaped {
				inDoubleQuote = !inDoubleQuote
			}

			if char == '\\' {
				escaped = !escaped
			} else {
				escaped = false
			}
		}

		// 3. 匹配关键字 (必须在非字符串、非注释状态下)
		if !inSingleQuote && !inDoubleQuote && !inSingleLineComment && !inMultiLineComment {
			if upperSQL[i:i+kwLen] == upperKW {
				// 边界检查：前后不能是字母数字或下划线
				isStartBoundary := i == 0 || !isAlphaNum(sql[i-1])
				isEndBoundary := i+kwLen == len(sql) || !isAlphaNum(sql[i+kwLen])
				if isStartBoundary && isEndBoundary {
					if direction > 0 {
						return i
					}
					lastPos = i
				}
			}
		}
	}

	return lastPos
}

func isAlphaNum(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}
