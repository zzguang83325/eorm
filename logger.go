package eorm

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
)

// LogLevel defines the severity of the log
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger interface defines simple behavior for logging with structured fields
type Logger interface {
	// Log records a log entry. fields is optional (can be nil).
	Log(level LogLevel, msg string, fields map[string]interface{})
}

// slogLogger is an adapter for log/slog
type slogLogger struct {
	logger *slog.Logger
}

func (s *slogLogger) Log(level LogLevel, msg string, fields map[string]interface{}) {
	l := s.logger
	if l == nil {
		l = slog.Default()
	}

	// Convert map to slice of key-value pairs for slog with stable order
	var args []interface{}
	if len(fields) > 0 {
		args = make([]interface{}, 0, len(fields)*2)

		// Priority keys to print first in specific order
		priorityKeys := []string{"db", "duration", "sql", "args", "error"}
		processedKeys := make(map[string]bool)

		// 1. Process priority keys first
		for _, k := range priorityKeys {
			if v, ok := fields[k]; ok {
				if k == "args" {
					if slice, ok := v.([]interface{}); ok {
						v = formatValue(slice)
					}
				}
				args = append(args, k, v)
				processedKeys[k] = true
			}
		}

		// 2. Sort remaining keys
		remainingKeys := make([]string, 0, len(fields)-len(processedKeys))
		for k := range fields {
			if !processedKeys[k] {
				remainingKeys = append(remainingKeys, k)
			}
		}
		sort.Strings(remainingKeys)

		// 3. Process remaining keys
		for _, k := range remainingKeys {
			v := fields[k]
			args = append(args, k, v)
		}
	}

	switch level {
	case LevelDebug:
		l.Debug(msg, args...)
	case LevelInfo:
		l.Info(msg, args...)
	case LevelWarn:
		l.Warn(msg, args...)
	case LevelError:
		l.Error(msg, args...)
	}
}

// NewSlogLogger creates a Logger that uses log/slog
func NewSlogLogger(logger *slog.Logger) Logger {
	return &slogLogger{logger: logger}
}

// formatValue formats a log field value
func formatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return fmt.Sprintf("'%s'", val)
	case []interface{}:
		var strs []string
		for _, item := range val {
			if s, ok := item.(string); ok {
				strs = append(strs, fmt.Sprintf("'%s'", s))
			} else {
				strs = append(strs, fmt.Sprintf("%v", item))
			}
		}
		return fmt.Sprintf("[%s]", strings.Join(strs, ", "))
	default:
		return fmt.Sprintf("%v", val)
	}
}

var (
	currentLogger Logger = &slogLogger{logger: nil}
	debug         bool
	re            = regexp.MustCompile(`\s+`)
	// 全局编码检测器实例（内部使用）
	globalEncodingDetector *internalEncodingDetector
)

// internalEncodingDetector 内部编码检测器，不对外暴露
type internalEncodingDetector struct {
	encodings map[string]encoding.Encoding
}

// newInternalEncodingDetector 创建内部编码检测器
func newInternalEncodingDetector() *internalEncodingDetector {
	return &internalEncodingDetector{
		encodings: map[string]encoding.Encoding{
			"GBK":          simplifiedchinese.GBK,
			"GB18030":      simplifiedchinese.GB18030,
			"GB2312":       simplifiedchinese.HZGB2312,
			"Big5":         traditionalchinese.Big5,
			"Shift_JIS":    japanese.ShiftJIS,
			"EUC-JP":       japanese.EUCJP,
			"EUC-KR":       korean.EUCKR,
			"Windows-1252": charmap.Windows1252,
			"Windows-1251": charmap.Windows1251,
			"ISO-8859-1":   charmap.ISO8859_1,
		},
	}
}

// fixTextEncoding 修复文本编码问题（内部函数，不对外暴露）
func (d *internalEncodingDetector) fixTextEncoding(text string) string {
	// 如果已经是有效的 UTF-8，直接返回
	if utf8.ValidString(text) {
		return text
	}

	data := []byte(text)

	// 按优先级尝试常见编码
	encodingPriority := []string{"GBK", "Big5", "GB18030", "GB2312", "Shift_JIS", "EUC-JP", "EUC-KR"}

	for _, encodingName := range encodingPriority {
		if enc, exists := d.encodings[encodingName]; exists {
			decoder := enc.NewDecoder()
			decoded, err := decoder.Bytes(data)
			if err != nil {
				continue
			}

			if utf8.Valid(decoded) {
				result := string(decoded)
				// 检查转换结果是否合理
				if d.isValidResult(result, data, encodingName) {
					// 在调试模式下记录转换
					if debug {
						LogDebug("编码转换成功", map[string]interface{}{
							"原始文本": text,
							"检测编码": encodingName,
							"转换结果": result,
						})
					}
					return result
				}
			}
		}
	}

	// 如果所有编码都失败，返回原始文本
	return text
}

// isValidResult 检查转换结果是否合理
func (d *internalEncodingDetector) isValidResult(text string, originalData []byte, encoding string) bool {
	if len(text) == 0 {
		return false
	}

	chineseCount := 0
	totalChars := 0
	invalidChars := 0
	halfWidthKatakana := 0

	for _, r := range text {
		totalChars++

		if r >= 0x4E00 && r <= 0x9FFF { // 中文字符
			chineseCount++
		} else if r == 0xFFFD { // 无效字符
			invalidChars++
		} else if r >= 0xFF61 && r <= 0xFF9F { // 半角片假名
			halfWidthKatakana++
		}
	}

	if totalChars == 0 {
		return false
	}

	// 检查各种合理性指标
	invalidRatio := float64(invalidChars) / float64(totalChars)
	katakanaRatio := float64(halfWidthKatakana) / float64(totalChars)

	// 基本合理性检查
	if invalidRatio > 0.2 { // 无效字符太多
		return false
	}

	if katakanaRatio > 0.3 { // 半角片假名太多，可能是错误的 Shift-JIS 转换
		return false
	}

	// 根据编码类型进行特定检查
	switch encoding {
	case "GBK", "GB18030", "GB2312":
		// 中文编码应该包含中文字符
		return chineseCount > 0 && d.isValidGBKPattern(originalData)
	case "Big5":
		// Big5 编码应该包含中文字符
		return chineseCount > 0 && d.isValidBig5Pattern(originalData)
	case "Shift_JIS", "EUC-JP":
		// 日文编码可能不包含中文字符，但不应该有太多无效字符
		return invalidRatio < 0.1
	default:
		// 其他编码至少要有一些合理的字符
		return chineseCount > 0 || (invalidRatio < 0.1 && totalChars > 0)
	}
}

// isValidGBKPattern 检查是否符合 GBK 编码模式
func (d *internalEncodingDetector) isValidGBKPattern(data []byte) bool {
	if len(data) < 2 {
		return false
	}

	validPairs := 0
	totalPairs := 0

	for i := 0; i < len(data)-1; i++ {
		b1, b2 := data[i], data[i+1]
		// GBK 编码范围
		if (b1 >= 0x81 && b1 <= 0xFE) && (b2 >= 0x40 && b2 <= 0xFE && b2 != 0x7F) {
			totalPairs++
			// 更精确的 GBK 范围
			if (b1 >= 0x81 && b1 <= 0xA0) || (b1 >= 0xAA && b1 <= 0xFE) {
				validPairs++
			}
			i++ // 跳过下一个字节
		}
	}

	if totalPairs == 0 {
		return false
	}

	// 有效字符对比例应该较高
	return float64(validPairs)/float64(totalPairs) > 0.6
}

// isValidBig5Pattern 检查是否符合 Big5 编码模式
func (d *internalEncodingDetector) isValidBig5Pattern(data []byte) bool {
	if len(data) < 2 {
		return false
	}

	validPairs := 0
	totalPairs := 0

	for i := 0; i < len(data)-1; i++ {
		b1, b2 := data[i], data[i+1]
		// Big5 编码范围
		if (b1 >= 0xA1 && b1 <= 0xFE) && (b2 >= 0x40 && b2 <= 0xFE && b2 != 0x7F) {
			totalPairs++
			// 更精确的 Big5 范围
			if (b1 >= 0xA1 && b1 <= 0xC6) || (b1 >= 0xC9 && b1 <= 0xF9) {
				validPairs++
			}
			i++ // 跳过下一个字节
		}
	}

	if totalPairs == 0 {
		return false
	}

	// 有效字符对比例应该较高
	return float64(validPairs)/float64(totalPairs) > 0.7
}

// fixStringEncoding 修复字符串编码的全局函数（内部使用，不对外暴露）
func fixStringEncoding(text string) string {
	if globalEncodingDetector == nil {
		globalEncodingDetector = newInternalEncodingDetector()
	}
	return globalEncodingDetector.fixTextEncoding(text)
}

// SetLogger sets the global logger
func SetLogger(l Logger) {
	currentLogger = l
}

// SetDebugMode enables or disables debug mode
func SetDebugMode(enabled bool) {
	debug = enabled
	if enabled {
		// 如果全局 slog 还不支持 Debug 级别，则强制设置一个输出到标准输出的 Debug 级别 slog
		if !slog.Default().Enabled(context.Background(), slog.LevelDebug) {
			slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))
		}
	}
}

// IsDebugEnabled returns true if debug mode is enabled
func IsDebugEnabled() bool {
	return debug
}

// cleanSQL removes newlines, tabs and multiple spaces from SQL string
func cleanSQL(sql string) string {
	return strings.TrimSpace(re.ReplaceAllString(sql, " "))
}

// LogSQL logs SQL statement, parameters and execution time in debug mode
func LogSQL(dbName string, sql string, args []interface{}, duration time.Duration) {
	if debug {
		fields := map[string]interface{}{
			"db":       dbName,
			"sql":      cleanSQL(sql),
			"duration": duration.String(),
		}
		if len(args) > 0 {
			fields["args"] = args
		}
		currentLogger.Log(LevelDebug, "SQL log", fields)
	}
}

func LogSQLError(dbName string, sql string, args []interface{}, duration time.Duration, err error) {

	// 自动修复错误信息的编码问题
	errorMsg := fixStringEncoding(err.Error())

	fields := map[string]interface{}{
		"db":       dbName,
		"sql":      cleanSQL(sql),
		"duration": duration.String(),
		"error":    errorMsg,
		"caller":   getCaller(), // 添加调用位置
	}
	if len(args) > 0 {
		fields["args"] = args
	}
	currentLogger.Log(LevelError, "SQL failed log", fields)
}

func getCaller() string {
	callers := make([]uintptr, 10)
	count := runtime.Callers(2, callers) // 从第2层开始获取
	var callerStack []string
	frames := runtime.CallersFrames(callers)
	for i := 0; i < count && i < 10; i++ { // 最多显示5层调用
		frame, more := frames.Next()
		if !more {
			break
		}

		pc := frame.PC
		file := frame.File
		line := frame.Line

		// 获取函数名
		fn := runtime.FuncForPC(pc)
		funcName := "unknown"
		if fn != nil {
			funcName = fn.Name()
			// 只显示函数名，去掉包路径
			if idx := strings.LastIndex(funcName, "/"); idx >= 0 {
				funcName = funcName[idx+1:]
			}
			if idx := strings.LastIndex(funcName, "."); idx >= 0 {
				funcName = funcName[idx+1:]
			}
		}

		// 只显示文件名
		fileName := file
		if idx := strings.LastIndex(file, "/"); idx >= 0 {
			fileName = file[idx+1:]
		}
		if idx := strings.LastIndex(file, "\\"); idx >= 0 {
			fileName = file[idx+1:]
		}

		callerStack = append(callerStack, fmt.Sprintf("%s(%s:%d)", funcName, fileName, line))
	}
	callerInfo := " [" + strings.Join(callerStack, " → ") + "]"
	return callerInfo
}

// LogInfo logs info message
func LogInfo(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	currentLogger.Log(LevelInfo, msg, f)
}

// LogWarn logs warning message
func LogWarn(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	currentLogger.Log(LevelWarn, msg, f)
}

// LogError logs error message
func LogError(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	currentLogger.Log(LevelError, msg, f)
}

// LogDebug logs debug message
func LogDebug(msg string, fields ...map[string]interface{}) {
	if debug {
		var f map[string]interface{}
		if len(fields) > 0 {
			f = fields[0]
		}
		currentLogger.Log(LevelDebug, msg, f)
	}
}

// Sync flushes any buffered log entries
func Sync() {
	if s, ok := currentLogger.(interface{ Sync() error }); ok {
		_ = s.Sync()
	}
}

// InitLogger initializes the logger with a specific level to stdout
// InitLogger initializes the global slog logger with a specific level to console
func InitLogger(level string) {
	// Determine log level
	slogLevel := slog.LevelInfo
	if strings.EqualFold(level, "debug") {
		slogLevel = slog.LevelDebug
		SetDebugMode(true)
	} else if strings.EqualFold(level, "warn") {
		slogLevel = slog.LevelWarn
	} else if strings.EqualFold(level, "error") {
		slogLevel = slog.LevelError
	}

	// Set global slog default with TextHandler to stdout
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slogLevel,
	})
	slog.SetDefault(slog.New(handler))

	// Reset currentLogger to use the new global default
	SetLogger(&slogLogger{logger: nil})
}

// InitLoggerWithFile initializes the logger to both console and a file using slog
func InitLoggerWithFile(level string, filePath string) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Fprintf(os.Stderr, "eorm: Failed to open log file: %v\n", err)
		return
	}

	// Determine log level
	slogLevel := slog.LevelInfo
	if strings.EqualFold(level, "debug") {
		slogLevel = slog.LevelDebug
		SetDebugMode(true)
	} else if strings.EqualFold(level, "warn") {
		slogLevel = slog.LevelWarn
	} else if strings.EqualFold(level, "error") {
		slogLevel = slog.LevelError
	}

	// Create a multi-writer for both console and file
	multiWriter := io.MultiWriter(os.Stdout, file)

	// Set global slog default with TextHandler
	handler := slog.NewTextHandler(multiWriter, &slog.HandlerOptions{
		Level: slogLevel,
	})
	slog.SetDefault(slog.New(handler))

	// Reset currentLogger to use the new global default
	SetLogger(&slogLogger{logger: nil})
}
