package eorm

import (
	"fmt"
	"regexp"
	"strings"
)

// SQLSecurityValidator SQL安全验证器
// 提供全面的SQL注入检测和安全验证功能
type SQLSecurityValidator struct {
	// 危险关键字模式
	dangerousPatterns []SecurityPattern
	// 预编译的危险模式正则
	compiledDangerous []*regexp.Regexp
	// 预编译的危险函数正则
	compiledFunctions []*regexp.Regexp
	// 允许的函数列表
	allowedFunctions []string
	// 最大SQL长度限制
	maxSQLLength int
}

// SecurityPattern 安全模式定义
type SecurityPattern struct {
	Pattern     string // 正则表达式模式
	Description string // 模式描述
	Severity    string // 严重程度: "high", "medium", "low"
}

// NewSQLSecurityValidator 创建新的SQL安全验证器
func NewSQLSecurityValidator() *SQLSecurityValidator {
	v := &SQLSecurityValidator{
		dangerousPatterns: getDefaultDangerousPatterns(),
		allowedFunctions:  getDefaultAllowedFunctions(),
		maxSQLLength:      10000, // 默认最大10KB
	}
	v.precompile()
	return v
}

// precompile 预编译所有正则表达式
func (v *SQLSecurityValidator) precompile() {
	// 预编译危险模式
	v.compiledDangerous = make([]*regexp.Regexp, len(v.dangerousPatterns))
	for i, p := range v.dangerousPatterns {
		v.compiledDangerous[i] = regexp.MustCompile(p.Pattern)
	}

	// 预编译危险函数
	dangerousFuncs := getDangerousFunctionPatterns()
	v.compiledFunctions = make([]*regexp.Regexp, len(dangerousFuncs))
	for i, f := range dangerousFuncs {
		v.compiledFunctions[i] = regexp.MustCompile(f)
	}
}

// ValidateSQL 验证SQL语句的安全性
func (v *SQLSecurityValidator) ValidateSQL(sql string) error {
	cleanSQL := strings.TrimSpace(sql)
	if cleanSQL == "" {
		return fmt.Errorf("%w: empty SQL statement", ErrInvalidSQL)
	}

	// 检查SQL长度
	if len(cleanSQL) > v.maxSQLLength {
		return fmt.Errorf("%w: SQL statement too long, exceeds %d character limit", ErrInvalidSQL, v.maxSQLLength)
	}
	upperSQL := strings.ToUpper(cleanSQL)

	// 基本格式检查
	if err := v.checkBasicFormat(upperSQL); err != nil {
		return err
	}

	// SQL注入检测
	if err := v.detectSQLInjection(cleanSQL, upperSQL); err != nil {
		return err
	}

	// 危险函数检测
	if err := v.detectDangerousFunctions(upperSQL); err != nil {
		return err
	}

	// 语法结构检查
	if err := v.checkSyntaxStructure(cleanSQL); err != nil {
		return err
	}

	return nil
}

// checkBasicFormat 检查基本SQL格式
func (v *SQLSecurityValidator) checkBasicFormat(upperSQL string) error {
	// 必须是SELECT语句或CTE（以WITH开头）
	if !strings.HasPrefix(upperSQL, "SELECT") && !strings.HasPrefix(upperSQL, "WITH") {
		return fmt.Errorf("%w: only SELECT statements are supported", ErrUnsupportedSQL)
	}

	// 检查是否包含必要的FROM子句
	if !strings.Contains(upperSQL, " FROM ") {
		return fmt.Errorf("%w: missing FROM clause", ErrInvalidSQL)
	}

	return nil
}

// detectSQLInjection 检测SQL注入攻击
func (v *SQLSecurityValidator) detectSQLInjection(cleanSQL, upperSQL string) error {
	// 使用预编译的危险模式进行检测
	for i, re := range v.compiledDangerous {
		if re.MatchString(upperSQL) {
			pattern := v.dangerousPatterns[i]
			return fmt.Errorf("%w: %s (severity: %s)",
				ErrSQLInjection, pattern.Description, pattern.Severity)
		}
	}

	// 检查字符串字面量中的可疑内容
	if err := v.checkStringLiterals(cleanSQL); err != nil {
		return err
	}

	// 检查注释注入
	if err := v.checkCommentInjection(cleanSQL); err != nil {
		return err
	}

	return nil
}

// detectDangerousFunctions 检测危险函数调用
func (v *SQLSecurityValidator) detectDangerousFunctions(upperSQL string) error {
	for _, re := range v.compiledFunctions {
		if re.MatchString(upperSQL) {
			return fmt.Errorf("%w: dangerous function call detected", ErrSQLInjection)
		}
	}
	return nil
}

// getDangerousFunctionPatterns 获取危险函数正则列表
func getDangerousFunctionPatterns() []string {
	return []string{
		`\bLOAD_FILE\b`,
		`\bINTO\s+OUTFILE\b`,
		`\bINTO\s+DUMPFILE\b`,
		`\bSYSTEM\b`,
		`\bEXEC\b`,
		`\bEXECUTE\b`,
		`\bSP_\w+`,
		`\bXP_\w+`,
		`\bDBMS_\w+`,
		`\bUTL_\w+`,
	}
}

// checkStringLiterals 检查字符串字面量中的可疑内容
func (v *SQLSecurityValidator) checkStringLiterals(sql string) error {
	// 提取所有字符串字面量
	literals := v.extractStringLiterals(sql)

	for _, literal := range literals {
		upperLiteral := strings.ToUpper(literal)

		// 改进：仅拦截真正的 DML 拼接尝试，而不是普通文本
		// 正常的 SELECT 可能出现在文章中，但 "SELECT; DROP" 或 "SELECT --" 是注入指纹
		injectionSignatures := []string{
			";", "--", "/*", "*/", "#",
		}

		// 如果字符串中包含 SQL 关键字且同时包含注入指纹，则判定为高风险
		suspiciousKeywords := []string{
			"SELECT", "INSERT", "UPDATE", "DELETE", "DROP", "UNION",
		}

		hasKeyword := false
		for _, kw := range suspiciousKeywords {
			if strings.Contains(upperLiteral, kw) {
				hasKeyword = true
				break
			}
		}

		if hasKeyword {
			for _, sig := range injectionSignatures {
				if strings.Contains(literal, sig) {
					return fmt.Errorf("%w: detected SQL injection signature in text: %s",
						ErrSQLInjection, sig)
				}
			}
		}
	}

	return nil
}

// checkCommentInjection 检查注释注入
func (v *SQLSecurityValidator) checkCommentInjection(sql string) error {
	// 检查SQL注释
	commentPatterns := []string{
		`--`,  // 单行注释
		`/\*`, // 块注释开始
		`\*/`, // 块注释结束
		`#`,   // MySQL风格注释
	}

	for _, pattern := range commentPatterns {
		if matched, _ := regexp.MatchString(pattern, sql); matched {
			return fmt.Errorf("%w: comment injection detected", ErrSQLInjection)
		}
	}

	return nil
}

// checkSyntaxStructure 检查语法结构
func (v *SQLSecurityValidator) checkSyntaxStructure(sql string) error {
	// 检查括号平衡
	if err := v.checkParenthesesBalance(sql); err != nil {
		return err
	}

	// 检查引号平衡
	if err := v.checkQuotesBalance(sql); err != nil {
		return err
	}

	// 检查分号（可能的语句分隔符）
	if strings.Contains(sql, ";") {
		return fmt.Errorf("%w: semicolon detected, possible multi-statement injection", ErrSQLInjection)
	}

	return nil
}

// extractStringLiterals 提取字符串字面量 (支持 ” 转义)
func (v *SQLSecurityValidator) extractStringLiterals(sql string) []string {
	var literals []string
	var current strings.Builder
	inString := false
	var stringChar byte

	lenSQL := len(sql)
	for i := 0; i < lenSQL; i++ {
		char := sql[i]

		if !inString && (char == '\'' || char == '"') {
			inString = true
			stringChar = char
			current.Reset()
		} else if inString && char == stringChar {
			// 1. 检查 SQL 标准转义 (两个连续单引号)
			if char == '\'' && i+1 < lenSQL && sql[i+1] == '\'' {
				current.WriteByte('\'')
				i++ // 跳过下一个单引号
				continue
			}
			// 2. 检查 C 风格转义 (\')
			if i > 0 && sql[i-1] == '\\' && (i < 2 || sql[i-2] != '\\') {
				// 已经在 current 中处理了
				continue
			}
			inString = false
			literals = append(literals, current.String())
		} else if inString {
			// 如果是反斜杠，需要注意是否是真正的转义
			if char == '\\' && i+1 < lenSQL && (sql[i+1] == '\'' || sql[i+1] == '"' || sql[i+1] == '\\') {
				current.WriteByte(sql[i+1])
				i++
				continue
			}
			current.WriteByte(char)
		}
	}

	return literals
}

// checkParenthesesBalance 检查括号平衡
func (v *SQLSecurityValidator) checkParenthesesBalance(sql string) error {
	count := 0
	inString := false
	var stringChar byte

	for i := 0; i < len(sql); i++ {
		char := sql[i]

		// 处理字符串字面量
		if !inString && (char == '\'' || char == '"') {
			inString = true
			stringChar = char
			continue
		}
		if inString && char == stringChar {
			if i > 0 && sql[i-1] == '\\' {
				continue
			}
			inString = false
			continue
		}
		if inString {
			continue
		}

		// 计算括号
		if char == '(' {
			count++
		} else if char == ')' {
			count--
			if count < 0 {
				return fmt.Errorf("%w: unmatched parentheses, extra closing parenthesis", ErrInvalidSQL)
			}
		}
	}

	if count != 0 {
		return fmt.Errorf("%w: unmatched parentheses, missing %d closing parentheses", ErrInvalidSQL, count)
	}

	return nil
}

// checkQuotesBalance 检查引号平衡
func (v *SQLSecurityValidator) checkQuotesBalance(sql string) error {
	singleQuoteCount := 0
	doubleQuoteCount := 0

	for i := 0; i < len(sql); i++ {
		char := sql[i]

		// 跳过转义字符
		if i > 0 && sql[i-1] == '\\' {
			continue
		}

		if char == '\'' {
			singleQuoteCount++
		} else if char == '"' {
			doubleQuoteCount++
		}
	}

	if singleQuoteCount%2 != 0 {
		return fmt.Errorf("%w: unmatched single quotes", ErrInvalidSQL)
	}

	if doubleQuoteCount%2 != 0 {
		return fmt.Errorf("%w: unmatched double quotes", ErrInvalidSQL)
	}

	return nil
}

// getDefaultDangerousPatterns 获取默认的危险模式列表
func getDefaultDangerousPatterns() []SecurityPattern {
	return []SecurityPattern{
		// High risk patterns
		{`;\s*DROP\s+`, "multiple statements: DROP statement injection", "high"},
		{`;\s*DELETE\s+`, "multiple statements: DELETE statement injection", "high"},
		{`;\s*INSERT\s+`, "multiple statements: INSERT statement injection", "high"},
		{`;\s*UPDATE\s+`, "multiple statements: UPDATE statement injection", "high"},
		{`;\s*CREATE\s+`, "multiple statements: CREATE statement injection", "high"},
		{`;\s*ALTER\s+`, "multiple statements: ALTER statement injection", "high"},
		{`;\s*TRUNCATE\s+`, "multiple statements: TRUNCATE statement injection", "high"},

		// Medium risk patterns
		{`\bUNION\s+SELECT\b`, "UNION injection", "medium"},
		{`\bOR\s+1\s*=\s*1\b`, "Classic OR injection", "medium"},
		{`\bAND\s+1\s*=\s*1\b`, "Classic AND injection", "medium"},
		{`OR\s+'1'\s*=\s*'1'`, "String OR injection", "medium"},
		{`AND\s+'1'\s*=\s*'1'`, "String AND injection", "medium"},
		{`OR\s+"1"\s*=\s*"1"`, "Double quote OR injection", "medium"},

		// Low risk patterns (suspicious but potentially legitimate)
		{`\bINTO\s+OUTFILE\b`, "File output", "low"},
		{`\bLOAD\s+DATA\b`, "Data loading", "low"},
		{`\bSLEEP\s*\(`, "Time delay function", "low"},
		{`\bBENCHMARK\s*\(`, "Performance test function", "low"},
	}
}

// getDefaultAllowedFunctions 获取默认允许的函数列表
func getDefaultAllowedFunctions() []string {
	return []string{
		// 聚合函数
		"COUNT", "SUM", "AVG", "MIN", "MAX",
		// 字符串函数
		"CONCAT", "SUBSTRING", "LENGTH", "UPPER", "LOWER", "TRIM",
		// 日期函数
		"NOW", "CURDATE", "CURTIME", "DATE", "TIME", "YEAR", "MONTH", "DAY",
		// 数学函数
		"ABS", "CEIL", "FLOOR", "ROUND", "MOD",
		// 条件函数
		"IF", "IFNULL", "NULLIF", "COALESCE", "CASE",
	}
}

// SetMaxSQLLength 设置最大SQL长度限制
func (v *SQLSecurityValidator) SetMaxSQLLength(length int) {
	if length > 0 {
		v.maxSQLLength = length
	}
}

// AddDangerousPattern 添加自定义危险模式
func (v *SQLSecurityValidator) AddDangerousPattern(pattern, description, severity string) {
	v.dangerousPatterns = append(v.dangerousPatterns, SecurityPattern{
		Pattern:     pattern,
		Description: description,
		Severity:    severity,
	})
}

// AddAllowedFunction 添加允许的函数
func (v *SQLSecurityValidator) AddAllowedFunction(function string) {
	v.allowedFunctions = append(v.allowedFunctions, strings.ToUpper(function))
}

// IsAllowedFunction 检查函数是否被允许
func (v *SQLSecurityValidator) IsAllowedFunction(function string) bool {
	upperFunc := strings.ToUpper(function)
	for _, allowed := range v.allowedFunctions {
		if allowed == upperFunc {
			return true
		}
	}
	return false
}
