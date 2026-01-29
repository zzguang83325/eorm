package eorm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
	"unsafe"
)

type noCopy struct{}

func (*noCopy) Lock() {}

// recursionTracker 用于跟踪深拷贝过程中的对象引用，防止循环引用导致无限递归
type recursionTracker struct {
	visited map[uintptr]interface{}
}

func newRecursionTracker() *recursionTracker {
	return &recursionTracker{
		visited: make(map[uintptr]interface{}),
	}
}

func (rt *recursionTracker) add(ptr uintptr, cloned interface{}) {
	rt.visited[ptr] = cloned
}

func (rt *recursionTracker) get(ptr uintptr) (interface{}, bool) {
	cloned, ok := rt.visited[ptr]
	return cloned, ok
}

// Record represents a single record in the database, similar to JFinal's ActiveRecord
// columns 保留原始大小写用于生成 SQL，lowerKeyMap 用于大小写不敏感的快速查找
// keys 保存字段插入顺序，用于 JSON 输出时保持顺序
type Record struct {
	mu          sync.RWMutex           // 将互斥锁放在最前面，确保正确的内存对齐
	columns     map[string]interface{} // 原始键名 -> 值
	lowerKeyMap map[string]string      // 小写键名 -> 原始键名（用于快速查找）
	keys        []string               // 保存字段插入顺序
	noCopy      noCopy                 // 辅助工具，防止 Record 被按值拷贝
}

// NewRecord creates a new empty Record
func NewRecord() *Record {
	return &Record{
		columns:     make(map[string]interface{}, 8),
		lowerKeyMap: make(map[string]string, 8),
		keys:        make([]string, 0, 8),
	}
}

var recordPool = sync.Pool{
	New: func() interface{} {
		return &Record{
			columns:     make(map[string]interface{}, 16),
			lowerKeyMap: make(map[string]string, 16),
			keys:        make([]string, 0, 16),
		}
	},
}

// NewRecordFromPool 从对象池获取 Record
// 优势：减少 GC 压力和内存分配开销
// 规范：
// 1. 获取后必须确保在不再使用时调用 Release() 归还，否则无法达到复用效果（虽然不会内存泄漏，但会失去优化意义）。
// 2. 严禁在调用 Release() 后继续使用该对象，否则会导致严重的数据竞争和不可预知的行为。
// 3. 支持在循环中连续调用以获取多个独立实例，只需确保每个实例最终都调用了 Release()。
func NewRecordFromPool() *Record {
	r := recordPool.Get().(*Record)
	r.Clear()
	return r
}

// Release 将 Record 归还到对象池
// 警告：归还后请立即将原引用置为 nil，防止误用。
func (r *Record) Release() {
	if r != nil {
		r.Clear()
		recordPool.Put(r)
	}
}

// FromMap (函数版) 从 map 创建新 Record
// 常用于 JSON 解析后的数据：record := eorm.FromMap(jsonMap)
func FromMap(m map[string]interface{}) *Record {
	return NewRecord().FromMap(m)
}

// FromRecord (函数版) 从另一个 Record 创建新 Record
// 使用深拷贝确保嵌套对象也被完整复制
func FromRecord(src *Record) *Record {
	if src == nil {
		return NewRecord()
	}
	return NewRecord().FromRecord(src)
}

// FromMap (方法版) 将 map 中的数据填充到当前 Record
// 支持链式调用：record.FromMap(map1).Set("extra", value)
func (r *Record) FromMap(m map[string]interface{}) *Record {
	for key, value := range m {
		r.Set(key, value)
	}
	return r
}

// Set sets a column value in the Record with case-insensitive support for existing columns
// 保留原始大小写用于 SQL 生成，同时维护小写映射用于快速查找
// 自动解引用基础数据类型的指针（如 *int, *string, *time.Time 等）
// 对于 Record 指针和复杂集合指针（*map, *slice），会自动克隆以防止意外的共享引用
func (r *Record) Set(column string, value interface{}) *Record {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 自动处理指针类型，防止意外的共享引用
	if value != nil {
		// 优先处理最常用的基础类型
		if isBasicType(value) {
			// 已经是基础类型，直接使用
		} else if isBasicPointer(value) {
			// 基础类型指针：自动解引用为值存储
			value = derefPointer(value)
		} else {
			switch v := value.(type) {
			case *Record:
				// Record 指针：克隆一份副本存储
				value = v.Clone()
			case *[]*Record:
				// 常见的 Record 切片指针：手动遍历克隆，避免反射
				if v != nil {
					newSlice := make([]*Record, len(*v))
					for i, item := range *v {
						if item != nil {
							newSlice[i] = item.Clone()
						}
					}
					value = newSlice
				} else {
					value = nil
				}
			case *map[string]interface{}:
				// 常见的 Map 指针：手动遍历克隆内容
				if v != nil {
					newMap := make(map[string]interface{}, len(*v))
					for k, item := range *v {
						newMap[k] = cloneValue(item)
					}
					value = newMap
				} else {
					value = nil
				}
			case *[]interface{}:
				// 通用切片指针
				if v != nil {
					newSlice := make([]interface{}, len(*v))
					for i, item := range *v {
						newSlice[i] = cloneValue(item)
					}
					value = newSlice
				} else {
					value = nil
				}
			case *[]Record:
				// Record 结构体切片指针
				if v != nil {
					newSlice := make([]Record, len(*v))
					for i, item := range *v {
						// Record 结构体在赋值给 interface{} 时会自动处理
						// 这里调用其 Clone 方法确保深度隔离
						cloned := item.Clone()
						if cloned != nil {
							newSlice[i] = *cloned
						}
					}
					value = newSlice
				} else {
					value = nil
				}
			default:
				// 对于其他不常用的复合指针类型，仍使用反射作为兜底
				rv := reflect.ValueOf(value)
				if rv.Kind() == reflect.Ptr && !rv.IsNil() {
					elem := rv.Elem()
					if elem.Kind() == reflect.Slice || elem.Kind() == reflect.Map {
						value = cloneValue(elem.Interface())
					}
				}
			}
		}
	}

	// 维护小写映射用于快速查找
	lowerKey := strings.ToLower(column)

	// 如果已存在相同小写键名的字段，更新原有字段
	if existingKey, exists := r.lowerKeyMap[lowerKey]; exists {
		r.columns[existingKey] = value
		return r
	}

	// 新字段：保存原始大小写和映射关系
	r.columns[column] = value
	r.lowerKeyMap[lowerKey] = column
	// 添加到 keys 列表以保持插入顺序
	r.keys = append(r.keys, column)
	return r
}

// SetIf 只有当 condition 为 true 时才设置字段
func (r *Record) SetIf(condition bool, column string, value interface{}) *Record {
	if condition {
		return r.Set(column, value)
	}
	return r
}

// SetIfNotNil 只有当 value 不为 nil 时才设置字段
func (r *Record) SetIfNotNil(column string, value interface{}) *Record {
	if !isNil(value) {
		return r.Set(column, value)
	}
	return r
}

// SetIfNotEmpty 只有当 value 不为空字符串时才设置字段
func (r *Record) SetIfNotEmpty(column, value string) *Record {
	if value != "" {
		return r.Set(column, value)
	}
	return r
}

// SetIfNil 只有当字段当前不存在或值为 nil 时才设置字段
func (r *Record) SetIfNil(column string, value interface{}) *Record {
	if r.Get(column) == nil {
		return r.Set(column, value)
	}
	return r
}

// SetIfEmpty 只有当字段当前不存在或值为空字符串时才设置字段
func (r *Record) SetIfEmpty(column string, value string) *Record {
	if r.GetString(column) == "" {
		return r.Set(column, value)
	}
	return r
}

// setDirect 直接设置列值，不加锁，不检查指针
// 仅供内部使用（如 scanRecords），用于从数据库扫描数据时的性能优化
// 前提条件：
// 1. Record 是新创建的局部变量，不会被并发访问
// 2. value 不是指针类型（数据库驱动返回的值都不是指针）
// 3. 列名不会重复（数据库列名唯一）
func (r *Record) setDirect(column string, value interface{}) {
	lowerKey := strings.ToLower(column)
	r.columns[column] = value
	r.lowerKeyMap[lowerKey] = column
	// 添加到 keys 列表以保持插入顺序
	r.keys = append(r.keys, column)
}

// getValue gets a column value from the Record with case-insensitive support
// 通过小写映射快速查找，O(1) 复杂度
func (r *Record) getValue(column string) interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	lowerKey := strings.ToLower(column)
	if actualKey, exists := r.lowerKeyMap[lowerKey]; exists {
		return r.columns[actualKey]
	}
	return nil
}

// Get gets a column value from the Record
func (r *Record) Get(column string) interface{} {
	return r.getValue(column)
}

// GetValues 批量获取多个字段的值
func (r *Record) GetValues(columns ...string) []interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]interface{}, len(columns))
	for i, col := range columns {
		lowerKey := strings.ToLower(col)
		if actualKey, exists := r.lowerKeyMap[lowerKey]; exists {
			result[i] = r.columns[actualKey]
		} else {
			result[i] = nil
		}
	}
	return result
}

// Transform 批量转换数据
func (r *Record) Transform(fn func(key string, value interface{}) interface{}) *Record {
	r.mu.Lock()
	defer r.mu.Unlock()

	for k, v := range r.columns {
		r.columns[k] = fn(k, v)
	}
	return r
}

// TransformValues 只转换值
func (r *Record) TransformValues(fn func(value interface{}) interface{}) *Record {
	r.mu.Lock()
	defer r.mu.Unlock()

	for k, v := range r.columns {
		r.columns[k] = fn(v)
	}
	return r
}

func (r *Record) MustGet(column string) (interface{}, error) {
	if !r.Has(column) {
		return "", fmt.Errorf("column %s not found", column)
	}
	return r.getValue(column), nil
}
func (r *Record) MustGetString(column string) (string, error) {
	if !r.Has(column) {
		return "", fmt.Errorf("column %s not found", column)
	}
	return r.GetString(column), nil
}

// GetInt gets a column value as int
// 使用 Convert 工具类进行类型转换，支持所有数值类型、字符串、bool 等
func (r *Record) GetInt(column string) int {
	return Convert.ToInt(r.getValue(column))
}

// GetInt64 gets a column value as int64
// 使用 Convert 工具类进行类型转换，支持所有数值类型、字符串、bool 等
func (r *Record) GetInt64(column string) int64 {
	return Convert.ToInt64(r.getValue(column))
}

// GetInt32 gets a column value as int32
// 使用 Convert 工具类进行类型转换，支持所有数值类型、字符串、bool 等
func (r *Record) GetInt32(column string) int32 {
	return Convert.ToInt32(r.getValue(column))
}

// GetInt16 gets a column value as int16
// 使用 Convert 工具类进行类型转换，支持所有数值类型、字符串、bool 等
func (r *Record) GetInt16(column string) int16 {
	return Convert.ToInt16(r.getValue(column))
}

// GetUint gets a column value as uint
// 使用 Convert 工具类进行类型转换，支持所有数值类型、字符串、bool 等
func (r *Record) GetUint(column string) uint {
	return Convert.ToUint(r.getValue(column))
}

// GetFloat gets a column value as float64
// 使用 Convert 工具类进行类型转换，支持所有数值类型、字符串、bool 等
func (r *Record) GetFloat(column string) float64 {
	return Convert.ToFloat64(r.getValue(column))
}

// GetFloat32 gets a column value as float32
// 使用 Convert 工具类进行类型转换，支持所有数值类型、字符串、bool 等
func (r *Record) GetFloat32(column string) float32 {
	return Convert.ToFloat32(r.getValue(column))
}

// GetBytes gets a column value as []byte
// 支持 []byte、string 等类型
func (r *Record) GetBytes(column string) []byte {
	val := r.getValue(column)
	if val == nil {
		return nil
	}
	switch v := val.(type) {
	case []byte:
		return v
	case string:
		return []byte(v)
	}
	// 其他类型转换为字符串再转字节
	return []byte(Convert.ToString(val))
}

// GetTime gets a column value as time.Time
// 使用 Convert 工具类进行类型转换，支持 time.Time、字符串等
func (r *Record) GetTime(column string) time.Time {
	return Convert.ToTime(r.getValue(column))
}

// GetString gets a column value as string
// 使用 Convert 工具类进行类型转换，支持所有类型
func (r *Record) GetString(column string) string {
	return Convert.ToString(r.getValue(column))
}

// GetBool gets a column value as bool
// 使用 Convert 工具类进行类型转换，支持 bool、数值类型、字符串等
// 字符串支持：true/false, t/f, 1/0, yes/no, on/off (大小写不敏感)
func (r *Record) GetBool(column string) bool {
	return Convert.ToBool(r.getValue(column))
}

// Has checks if a column exists in the Record
func (r *Record) Has(column string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	lowerKey := strings.ToLower(column)
	_, exists := r.lowerKeyMap[lowerKey]
	return exists
}

// ValidateRequired 验证必填字段
func (r *Record) ValidateRequired(columns ...string) error {
	for _, col := range columns {
		if !r.Has(col) {
			return fmt.Errorf("field '%s' is required", col)
		}
	}
	return nil
}

// Keys returns all column names in insertion order
func (r *Record) Keys() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	keys := make([]string, len(r.keys))
	copy(keys, r.keys)
	return keys
}

// Remove removes a column from the Record with case-insensitive support
func (r *Record) Remove(column string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	lowerKey := strings.ToLower(column)
	if actualKey, exists := r.lowerKeyMap[lowerKey]; exists {
		delete(r.columns, actualKey)
		delete(r.lowerKeyMap, lowerKey)
		// 从 keys 列表中移除
		for i, k := range r.keys {
			if k == actualKey {
				r.keys = append(r.keys[:i], r.keys[i+1:]...)
				break
			}
		}
	}
}

// Clear clears all columns
func (r *Record) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 重用已有的 map 和 slice 以减少分配，而不是重新 make
	for k := range r.columns {
		delete(r.columns, k)
	}
	for k := range r.lowerKeyMap {
		delete(r.lowerKeyMap, k)
	}
	r.keys = r.keys[:0]
}

// ToMap converts the Record to a map (returns a copy)
func (r *Record) ToMap() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	newMap := make(map[string]interface{}, len(r.columns))
	for k, v := range r.columns {
		newMap[k] = v
	}
	return newMap
}

// ToJson converts the Record to JSON string
// ToJson converts the Record to JSON string with insertion order preserved
func (r *Record) ToJson() string {
	data, err := r.MarshalJSON()
	if err != nil {
		return "{}"
	}
	return string(data)
}

// String 实现 Stringer 接口，返回 JSON 格式的字符串
// 这样可以直接使用 fmt.Printf("%v", record) 输出 JSON 格式
func (r *Record) String() string {
	return r.ToJson()
}

var jsonBufferPool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

// MarshalJSON 实现 json.Marshaler 接口，使 json.Marshal 也能保持顺序
func (r *Record) MarshalJSON() ([]byte, error) {
	if r == nil {
		return []byte("{}"), nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	buf := jsonBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer jsonBufferPool.Put(buf)

	if err := r.marshalToBuffer(buf, make(map[uintptr]bool), 0); err != nil {
		return nil, err
	}

	// 更高效的拷贝：直接返回底层数组的切片副本
	// 这仍然是安全的，因为我们创建了一个新的切片，而原 buffer 会被归还和重置
	data := buf.Bytes()
	result := make([]byte, len(data))
	copy(result, data)
	return result, nil
}

// ToIndentedJson 格式化 JSON
func (r *Record) ToIndentedJson() string {
	data, err := r.MarshalJSON()
	if err != nil {
		return "{}"
	}

	var out bytes.Buffer
	if err := json.Indent(&out, data, "", "  "); err != nil {
		return string(data)
	}

	return out.String()
}

// marshalToBuffer 优化版本，支持缓冲区传递，性能更好
func (r *Record) marshalToBuffer(buf *bytes.Buffer, visited map[uintptr]bool, depth int) error {
	const maxDepth = 100
	if depth > maxDepth {
		buf.WriteString(`{"__error":"max recursion depth exceeded"}`)
		return nil
	}

	if r == nil {
		buf.WriteString("null")
		return nil
	}

	// 循环引用检测
	currentPtr := uintptr(unsafe.Pointer(r))
	if visited[currentPtr] {
		buf.WriteString(`{"__error":"circular reference"}`)
		return nil
	}
	visited[currentPtr] = true
	defer delete(visited, currentPtr)

	if len(r.columns) == 0 {
		buf.WriteString("{}")
		return nil
	}

	buf.WriteByte('{')

	for i, k := range r.keys {
		if v, ok := r.columns[k]; ok {
			if i > 0 {
				buf.WriteByte(',')
			}

			// 写入键：使用自定义的高性能写入，避免 json.Marshal 的分配
			buf.WriteByte('"')
			writeJSONString(buf, k)
			buf.WriteString("\":")

			// 写入值
			switch val := v.(type) {
			case *Record:
				if val != nil {
					if err := val.marshalToBuffer(buf, visited, depth+1); err != nil {
						return err
					}
				} else {
					buf.WriteString("null")
				}
			case Record:
				if err := (&val).marshalToBuffer(buf, visited, depth+1); err != nil {
					return err
				}
			case string:
				buf.WriteByte('"')
				writeJSONString(buf, val)
				buf.WriteByte('"')
			case bool:
				if val {
					buf.WriteString("true")
				} else {
					buf.WriteString("false")
				}
			case nil:
				buf.WriteString("null")
			case []interface{}:
				buf.WriteByte('[')
				for i, item := range val {
					if i > 0 {
						buf.WriteByte(',')
					}
					switch itemVal := item.(type) {
					case *Record:
						if itemVal != nil {
							if err := itemVal.marshalToBuffer(buf, visited, depth+1); err != nil {
								return err
							}
						} else {
							buf.WriteString("null")
						}
					case Record:
						if err := (&itemVal).marshalToBuffer(buf, visited, depth+1); err != nil {
							return err
						}
					default:
						itemJSON, err := json.Marshal(item)
						if err != nil {
							return err
						}
						buf.Write(itemJSON)
					}
				}
				buf.WriteByte(']')
			default:
				valJSON, err := json.Marshal(v)
				if err != nil {
					return err
				}
				buf.Write(valJSON)
			}
		}
	}

	buf.WriteByte('}')
	return nil
}

// writeJSONString 优化字符串转义
func writeJSONString(buf *bytes.Buffer, s string) {
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"':
			buf.WriteString(`\"`)
		case '\\':
			buf.WriteString(`\\`)
		case '\b':
			buf.WriteString(`\b`)
		case '\f':
			buf.WriteString(`\f`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		default:
			if c < 0x20 {
				// 控制字符
				buf.WriteString(fmt.Sprintf("\\u%04x", c))
			} else {
				buf.WriteByte(c)
			}
		}
	}
}

func (r *Record) toMapRecursive(visited map[uintptr]bool, depth int) map[string]interface{} {
	const maxDepth = 100
	if depth > maxDepth {
		return map[string]interface{}{"__error": "max recursion depth exceeded"}
	}

	if r == nil {
		return nil
	}

	// 循环引用检测
	currentPtr := uintptr(unsafe.Pointer(r))
	if visited[currentPtr] {
		return map[string]interface{}{"__error": "circular reference"}
	}
	visited[currentPtr] = true
	defer delete(visited, currentPtr)

	r.mu.RLock()
	defer r.mu.RUnlock()

	// 空记录快速返回
	if len(r.columns) == 0 {
		return map[string]interface{}{}
	}

	result := make(map[string]interface{}, len(r.columns))
	// 按照 keys 列表的顺序遍历，保持插入顺序
	for _, k := range r.keys {
		if v, ok := r.columns[k]; ok {
			// 处理嵌套 Record
			switch val := v.(type) {
			case *Record:
				if val != nil {
					result[k] = val.toMapRecursive(visited, depth+1)
				} else {
					result[k] = nil
				}
			case Record:
				result[k] = (&val).toMapRecursive(visited, depth+1)
			default:
				result[k] = v
			}
		}
	}

	return result
}

// Clone creates a copy of the Record
// 创建 Record 的浅拷贝
func (r *Record) Clone() *Record {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	newRecord := NewRecord()

	// 直接复制字段
	for k, v := range r.columns {
		newRecord.setDirect(k, v)
	}
	// 复制 keys 顺序
	newRecord.keys = make([]string, len(r.keys))
	copy(newRecord.keys, r.keys)

	return newRecord
}

// isBasicType 检查是否为基础类型
func isBasicType(v interface{}) bool {
	switch v.(type) {
	case string, int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64, bool,
		time.Time, time.Duration,
		complex64, complex128,
		uintptr:
		return true
	}
	return false
}

// isBasicPointer 检查是否为基础类型指针
func isBasicPointer(v interface{}) bool {
	switch v.(type) {
	case *string, *int, *int8, *int16, *int32, *int64,
		*uint, *uint8, *uint16, *uint32, *uint64,
		*float32, *float64, *bool,
		*time.Time, *time.Duration,
		*complex64, *complex128,
		*uintptr:
		return true
	}
	return false
}

// isBasicSlice 检查是否为基础类型切片
func isBasicSlice(v interface{}) bool {
	switch v.(type) {
	case []string, []int, []int8, []int16, []int32, []int64,
		[]uint, []uint8, []uint16, []uint32, []uint64,
		[]float32, []float64, []bool:
		return true
	}
	return false
}

// cloneBasicSlice 克隆基础类型切片
func cloneBasicSlice(v interface{}) interface{} {
	switch val := v.(type) {
	case []string:
		newSlice := make([]string, len(val))
		copy(newSlice, val)
		return newSlice
	case []int:
		newSlice := make([]int, len(val))
		copy(newSlice, val)
		return newSlice
	case []int8:
		newSlice := make([]int8, len(val))
		copy(newSlice, val)
		return newSlice
	case []int16:
		newSlice := make([]int16, len(val))
		copy(newSlice, val)
		return newSlice
	case []int32:
		newSlice := make([]int32, len(val))
		copy(newSlice, val)
		return newSlice
	case []int64:
		newSlice := make([]int64, len(val))
		copy(newSlice, val)
		return newSlice
	case []uint:
		newSlice := make([]uint, len(val))
		copy(newSlice, val)
		return newSlice
	case []uint16:
		newSlice := make([]uint16, len(val))
		copy(newSlice, val)
		return newSlice
	case []uint32:
		newSlice := make([]uint32, len(val))
		copy(newSlice, val)
		return newSlice
	case []uint64:
		newSlice := make([]uint64, len(val))
		copy(newSlice, val)
		return newSlice
	case []float32:
		newSlice := make([]float32, len(val))
		copy(newSlice, val)
		return newSlice
	case []float64:
		newSlice := make([]float64, len(val))
		copy(newSlice, val)
		return newSlice
	case []bool:
		newSlice := make([]bool, len(val))
		copy(newSlice, val)
		return newSlice
	case []byte:
		newByte := make([]byte, len(val))
		copy(newByte, val)
		return newByte
	}
	return nil
}

// cloneValue 辅助函数，用于克隆 interface{} 里面的值
// clonePointer 简化指针类型的处理，为基础类型指针创建新实例并拷贝值
func clonePointer(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	// 通过类型断言优先处理高频的 time 类型，避免反射
	switch v := value.(type) {
	case *time.Time:
		if v == nil {
			return nil
		}
		newTime := *v
		return &newTime
	case *time.Duration:
		if v == nil {
			return nil
		}
		newDur := *v
		return &newDur
	}

	val := reflect.ValueOf(value)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return value
	}

	elem := val.Elem()
	kind := elem.Kind()

	// 处理其他基础类型 (数值、布尔、字符串、uintptr、复数)
	if (kind >= reflect.Bool && kind <= reflect.Complex128) || kind == reflect.String || kind == reflect.Uintptr {
		newPtr := reflect.New(elem.Type())
		newPtr.Elem().Set(elem)
		return newPtr.Interface()
	}

	return value
}

func cloneValue(v interface{}) interface{} {
	if v == nil {
		return nil
	}

	// 1. 基础值类型直接返回
	if isBasicType(v) {
		return v
	}

	// 2. 基础类型指针处理
	if isBasicPointer(v) {
		return clonePointer(v)
	}

	// 3. 基础类型切片处理
	if isBasicSlice(v) {
		return cloneBasicSlice(v)
	}

	switch val := v.(type) {
	case *Record:
		// 递归克隆嵌套的 Record
		return val.Clone()

	// 常见的 map 和 slice 需要递归
	case []*Record:
		newSlice := make([]*Record, len(val))
		for i, v2 := range val {
			newSlice[i] = v2.Clone()
		}
		return newSlice
	case []Record:
		newSlice := make([]Record, len(val))
		for i, v2 := range val {
			newSlice[i] = *v2.Clone()
		}
		return newSlice
	case map[string]interface{}:
		newMap := make(map[string]interface{})
		for k2, v2 := range val {
			newMap[k2] = cloneValue(v2)
		}
		return newMap
	case []interface{}:
		newSlice := make([]interface{}, len(val))
		for i, v2 := range val {
			newSlice[i] = cloneValue(v2)
		}
		return newSlice
	default:
		// time.Time, string, int 等基本类型是值传递的，无需特殊处理
		return v
	}
}

// deepCloneValue 深拷贝 interface{} 里面的值，使用 recursionTracker 防止循环引用
func deepCloneValue(v interface{}, tracker *recursionTracker) interface{} {
	if v == nil {
		return nil
	}

	// 1. 基础类型（直接返回，不需要克隆也不需要追踪）
	switch v.(type) {
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, uintptr, float32, float64, string, complex64, complex128, time.Time, time.Duration:
		return v
	}

	// 2. 基础类型指针和基础类型切片
	if isBasicPointer(v) {
		return clonePointer(v)
	}
	if isBasicSlice(v) {
		return cloneBasicSlice(v)
	}

	// 3. 循环引用检测
	val := reflect.ValueOf(v)
	var ptr uintptr
	kind := val.Kind()
	switch kind {
	case reflect.Ptr, reflect.Map, reflect.Slice:
		if val.IsNil() {
			return v
		}
		ptr = val.Pointer()
	default:
		ptr = 0
	}

	if ptr != 0 {
		if cloned, ok := tracker.get(ptr); ok {
			return cloned
		}
	}

	// 4. 特化类型处理
	switch concrete := v.(type) {
	case *Record:
		return concrete.deepCloneWithTracker(tracker)

	case Record:
		// 对于 Record 值，其身份由 columns 映射决定
		columnsPtr := reflect.ValueOf(concrete.columns).Pointer()
		if cloned, ok := tracker.get(columnsPtr); ok {
			// 如果 columns 已经克隆过，返回克隆后的 Record 值
			return *(cloned.(*Record))
		}
		cloned := (&concrete).deepCloneWithTracker(tracker)
		return *cloned

	case []*Record:
		newSlice := make([]*Record, len(concrete))
		if ptr != 0 {
			tracker.add(ptr, newSlice)
		}
		for i, item := range concrete {
			if item != nil {
				newSlice[i] = item.deepCloneWithTracker(tracker)
			}
		}
		return newSlice

	case []Record:
		newSlice := make([]Record, len(concrete))
		if ptr != 0 {
			tracker.add(ptr, newSlice)
		}
		for i := range concrete {
			cloned := (&concrete[i]).deepCloneWithTracker(tracker)
			newSlice[i] = *cloned
		}
		return newSlice

	case []interface{}:
		newSlice := make([]interface{}, len(concrete))
		if ptr != 0 {
			tracker.add(ptr, newSlice)
		}
		for i, item := range concrete {
			newSlice[i] = deepCloneValue(item, tracker)
		}
		return newSlice

	case map[string]*Record:
		newMap := make(map[string]*Record)
		if ptr != 0 {
			tracker.add(ptr, newMap)
		}
		for k, item := range concrete {
			if item != nil {
				newMap[k] = item.deepCloneWithTracker(tracker)
			}
		}
		return newMap

	case map[string]interface{}:
		newMap := make(map[string]interface{})
		if ptr != 0 {
			tracker.add(ptr, newMap)
		}
		for k, item := range concrete {
			newMap[k] = deepCloneValue(item, tracker)
		}
		return newMap
	}

	// 5. 其他类型走反射逻辑
	return deepCloneReflect(v, tracker)
}

// deepCloneReflect 使用反射深拷贝任意类型的值
func deepCloneReflect(v interface{}, tracker *recursionTracker) interface{} {
	if v == nil {
		return nil
	}
	val := reflect.ValueOf(v)
	return deepCloneReflectValue(val, tracker).Interface()
}

// deepCloneReflectValue 使用反射深拷贝 reflect.Value
func deepCloneReflectValue(val reflect.Value, tracker *recursionTracker) reflect.Value {
	if !val.IsValid() {
		return reflect.Value{}
	}

	// 对于可以获取指针的类型，检查循环引用
	kind := val.Kind()
	if kind == reflect.Ptr || kind == reflect.Map || kind == reflect.Slice {
		if !val.IsNil() {
			ptr := val.Pointer()
			if cloned, ok := tracker.get(ptr); ok {
				return reflect.ValueOf(cloned)
			}
		}
	}

	switch kind {
	case reflect.Ptr:
		if val.IsNil() {
			return reflect.Zero(val.Type())
		}
		// 指针类型特殊处理：先创建指针指向的对象
		clonedPtr := reflect.New(val.Type().Elem())
		// 记录指针映射
		tracker.add(val.Pointer(), clonedPtr.Interface())

		// 获取指针指向的元素
		elem := val.Elem()
		// 使用 deepCloneValue 处理元素，确保 Record 特化逻辑生效
		clonedElem := deepCloneValue(elem.Interface(), tracker)
		if clonedElem != nil {
			clonedPtr.Elem().Set(reflect.ValueOf(clonedElem))
		}
		return clonedPtr

	case reflect.Interface:
		if val.IsNil() {
			return reflect.Zero(val.Type())
		}
		// 获取接口的具体值
		elem := val.Elem()
		// 递归克隆具体值
		clonedElem := deepCloneReflectValue(elem, tracker)

		// 确保克隆后的值可以赋值给接口
		if !clonedElem.IsValid() {
			return reflect.Zero(val.Type())
		}

		// 创建新的接口值
		newInterface := reflect.New(val.Type()).Elem()
		if clonedElem.Type().AssignableTo(val.Type()) {
			newInterface.Set(clonedElem)
		} else {
			// 类型不匹配，尝试转换为 interface{}
			newInterface.Set(reflect.ValueOf(clonedElem.Interface()))
		}
		return newInterface

	case reflect.Struct:
		clonedStruct := reflect.New(val.Type()).Elem()
		for i := 0; i < val.NumField(); i++ {
			field := val.Field(i)
			if val.Type().Field(i).PkgPath == "" { // exported field
				// 使用 deepCloneValue 处理结构体字段
				clonedField := deepCloneValue(field.Interface(), tracker)
				if clonedField != nil {
					clonedStruct.Field(i).Set(reflect.ValueOf(clonedField))
				}
			}
		}
		return clonedStruct

	case reflect.Map:
		if val.IsNil() {
			return reflect.Zero(val.Type())
		}
		newMap := reflect.MakeMap(val.Type())
		tracker.add(val.Pointer(), newMap.Interface())
		for _, k := range val.MapKeys() {
			// 使用 deepCloneValue 处理 Map 的 Key 和 Value
			clonedKey := deepCloneValue(k.Interface(), tracker)
			clonedVal := deepCloneValue(val.MapIndex(k).Interface(), tracker)

			var kv, vv reflect.Value
			if clonedKey != nil {
				kv = reflect.ValueOf(clonedKey)
			} else {
				kv = reflect.Zero(val.Type().Key())
			}

			if clonedVal != nil {
				vv = reflect.ValueOf(clonedVal)
			} else {
				vv = reflect.Zero(val.Type().Elem())
			}
			newMap.SetMapIndex(kv, vv)
		}
		return newMap

	case reflect.Slice:
		if val.IsNil() {
			return reflect.Zero(val.Type())
		}
		newSlice := reflect.MakeSlice(val.Type(), val.Len(), val.Cap())
		tracker.add(val.Pointer(), newSlice.Interface())
		for i := 0; i < val.Len(); i++ {
			// 使用 deepCloneValue 处理 Slice 元素
			clonedElem := deepCloneValue(val.Index(i).Interface(), tracker)
			if clonedElem != nil {
				newSlice.Index(i).Set(reflect.ValueOf(clonedElem))
			}
		}
		return newSlice

	case reflect.Array:
		len := val.Len()
		newArray := reflect.New(val.Type()).Elem()
		for i := 0; i < len; i++ {
			newArray.Index(i).Set(deepCloneReflectValue(val.Index(i), tracker))
		}
		return newArray

	default:
		// 基本类型直接返回
		return val
	}
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (r *Record) UnmarshalJSON(data []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 清空现有数据
	r.columns = make(map[string]interface{})
	r.lowerKeyMap = make(map[string]string)
	r.keys = make([]string, 0)

	// 先反序列化到临时 map
	var tempMap map[string]interface{}
	if err := json.Unmarshal(data, &tempMap); err != nil {
		return err
	}

	// 转换值并重建映射和顺序
	for k, v := range tempMap {
		r.columns[k] = r.convertJsonValue(v)
		r.lowerKeyMap[strings.ToLower(k)] = k
		r.keys = append(r.keys, k)
	}

	return nil
}

// FromJson parses JSON string into the Record
func (r *Record) FromJson(jsonStr string) *Record {
	r.mu.Lock()
	defer r.mu.Unlock()

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return r
	}

	// 清空现有数据
	r.columns = make(map[string]interface{})
	r.lowerKeyMap = make(map[string]string)
	r.keys = make([]string, 0)

	for k, v := range data {
		r.columns[k] = r.convertJsonValue(v)
		r.lowerKeyMap[strings.ToLower(k)] = k
		r.keys = append(r.keys, k)
	}
	return r
}

// convertJsonValue 转换 JSON 值为适当类型
func (r *Record) convertJsonValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	// 处理 map
	if m, ok := value.(map[string]interface{}); ok {
		return r.convertJsonMap(m)
	}

	// 处理数组
	if slice, ok := value.([]interface{}); ok {
		return r.convertJsonArray(slice)
	}

	return value
}

// convertJsonMap 将 map 转换为 Record
func (r *Record) convertJsonMap(m map[string]interface{}) *Record {
	record := NewRecord()
	for k, v := range m {
		record.Set(k, r.convertJsonValue(v))
	}
	return record
}

// convertJsonArray 转换 JSON 数组
func (r *Record) convertJsonArray(slice []interface{}) interface{} {
	if len(slice) == 0 {
		return slice
	}

	// 检查是否可以转换为 Record 数组
	first := slice[0]
	if first == nil {
		return slice
	}

	// 如果是 map 数组，转换为 []*Record
	if _, ok := first.(map[string]interface{}); ok {
		records := make([]*Record, len(slice))
		for i, item := range slice {
			if m, ok := item.(map[string]interface{}); ok {
				records[i] = r.convertJsonMap(m)
			}
		}
		return records
	}

	// 其他类型数组，处理每个元素
	result := make([]interface{}, len(slice))
	for i, item := range slice {
		result[i] = r.convertJsonValue(item)
	}
	return result
}

// ToStruct converts the Record to a struct
func (r *Record) ToStruct(dest interface{}) error {
	return ToStruct(r, dest)
}

// FromStruct populates the Record from a struct
func (r *Record) FromStruct(src interface{}) *Record {
	_ = FromStruct(src, r)
	return r
}

// FromRecord populates the Record from another Record
// 支持链式调用：record.FromRecord(record1).Set("extra", value)
func (r *Record) FromRecord(src *Record) *Record {
	if src == nil {
		return r
	}

	// 获取源 Record 的只读锁
	src.mu.RLock()
	defer src.mu.RUnlock()

	// 获取目标 Record 的写锁
	r.mu.Lock()
	defer r.mu.Unlock()

	// 清空现有数据
	r.columns = make(map[string]interface{})
	r.lowerKeyMap = make(map[string]string)
	r.keys = make([]string, 0)

	// 直接复制值
	for key, value := range src.columns {
		r.columns[key] = value
		r.lowerKeyMap[strings.ToLower(key)] = key
	}
	// 复制 keys 顺序
	r.keys = make([]string, len(src.keys))
	copy(r.keys, src.keys)

	return r
}

// DeepClone creates a deep copy of the Record
// 创建 Record 的深拷贝，包含所有嵌套的 Record、切片和 map
func (r *Record) DeepClone() *Record {
	if r == nil {
		return nil
	}
	tracker := newRecursionTracker()
	return r.deepCloneWithTracker(tracker)
}

// deepCloneWithTracker 是内部使用的深拷贝实现，支持循环引用检测
func (r *Record) deepCloneWithTracker(tracker *recursionTracker) *Record {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	// 1. 检查是否已经处理过此指针
	ptr := uintptr(unsafe.Pointer(r))
	if cloned, ok := tracker.get(ptr); ok {
		return cloned.(*Record)
	}

	// 2. 检查是否已经处理过此 Record 的数据内容 (通过 columns 映射识别)
	var columnsPtr uintptr
	if r.columns != nil {
		columnsPtr = reflect.ValueOf(r.columns).Pointer()
		if cloned, ok := tracker.get(columnsPtr); ok {
			return cloned.(*Record)
		}
	}

	// 3. 创建新 Record 并记录映射
	newRecord := NewRecord()
	tracker.add(ptr, newRecord)
	if columnsPtr != 0 {
		tracker.add(columnsPtr, newRecord)
	}

	// 4. 拷贝基本属性
	newRecord.keys = make([]string, len(r.keys))
	copy(newRecord.keys, r.keys)

	newRecord.lowerKeyMap = make(map[string]string, len(r.lowerKeyMap))
	for k, v := range r.lowerKeyMap {
		newRecord.lowerKeyMap[k] = v
	}

	// 5. 深拷贝数据
	newRecord.columns = make(map[string]interface{}, len(r.columns))
	for k, v := range r.columns {
		newRecord.columns[k] = deepCloneValue(v, tracker)
	}

	return newRecord
}

// FromRecordDeep populates Record from another Record with deep copy
// 从另一个 Record 深拷贝填充当前 Record，支持链式调用
func (r *Record) FromRecordDeep(src *Record) *Record {
	if src == nil {
		return r
	}

	// 获取源 Record 的只读锁
	src.mu.RLock()
	defer src.mu.RUnlock()

	// 获取目标 Record 的写锁
	r.mu.Lock()
	defer r.mu.Unlock()

	// 清空现有数据
	r.columns = make(map[string]interface{})
	r.lowerKeyMap = make(map[string]string)
	r.keys = make([]string, 0)

	// 使用深拷贝复制值
	tracker := newRecursionTracker()
	// 记录 src 到 r 的映射，防止 src 内部引用自己时能正确指向 r
	tracker.add(uintptr(unsafe.Pointer(src)), r)

	for key, value := range src.columns {
		r.columns[key] = deepCloneValue(value, tracker)
		r.lowerKeyMap[strings.ToLower(key)] = key
	}
	// 复制 keys 顺序
	r.keys = make([]string, len(src.keys))
	copy(r.keys, src.keys)

	return r
}

// Str returns the column name in string format
func (r *Record) Str(column string) string {
	return r.GetString(column)
}

// Int returns the column value as int
func (r *Record) Int(column string) int {
	return r.GetInt(column)
}

// Int64 returns the column value as int64
func (r *Record) Int64(column string) int64 {
	return r.GetInt64(column)
}

// Int32 returns the column value as int32
func (r *Record) Int32(column string) int32 {
	return r.GetInt32(column)
}

// Int16 returns the column value as int16
func (r *Record) Int16(column string) int16 {
	return r.GetInt16(column)
}

// Uint returns the column value as uint
func (r *Record) Uint(column string) uint {
	return r.GetUint(column)
}

// Float returns the column value as float64
func (r *Record) Float(column string) float64 {
	return r.GetFloat(column)
}

// Float32 returns the column value as float32
func (r *Record) Float32(column string) float32 {
	return r.GetFloat32(column)
}

// Bytes returns the column value as []byte
func (r *Record) Bytes(column string) []byte {
	return r.GetBytes(column)
}

// Bool returns the column value as bool
func (r *Record) Bool(column string) bool {
	return r.GetBool(column)
}

// Time returns the column value as time.Time
func (r *Record) Time(column string) time.Time {
	return r.GetTime(column)
}

// GetRecords returns a slice of Records from a column
// 主要用途是FromJson的数据结构比较复杂,里面嵌套了其他的Record数组,
// 所以需要通过GetRecords来获取里面的Record数组
func (r *Record) GetRecords(column string) ([]*Record, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	val := r.getValue(column)
	if val == nil {
		return nil, fmt.Errorf("column '%s' not found", column)
	}

	records := convertToRecordSliceSafe(val)
	if records == nil {
		return nil, fmt.Errorf("column '%s' cannot be converted to []*Record", column)
	}
	return records, nil
}

// GetSlice 获取切片值，返回 []interface{} 和 error
func (r *Record) GetSlice(column string) ([]interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	val := r.getValue(column)
	if val == nil {
		return nil, fmt.Errorf("column '%s' not found", column)
	}

	slice := toInterfaceSlice(val)
	if slice == nil {
		return nil, fmt.Errorf("column '%s' cannot be converted to slice", column)
	}

	return slice, nil
}

// GetStringSlice 获取字符串切片
func (r *Record) GetStringSlice(column string) ([]string, error) {
	slice, err := r.GetSlice(column)
	if err != nil {
		return nil, err
	}

	result := make([]string, len(slice))
	for i, v := range slice {
		result[i] = Convert.ToString(v)
	}
	return result, nil
}

// GetIntSlice 获取整数切片
func (r *Record) GetIntSlice(column string) ([]int, error) {
	slice, err := r.GetSlice(column)
	if err != nil {
		return nil, err
	}

	result := make([]int, len(slice))
	for i, v := range slice {
		result[i] = Convert.ToInt(v)
	}
	return result, nil
}

// GetSliceByPath 通过路径获取切片
func (r *Record) GetSliceByPath(path string) ([]interface{}, error) {
	if path == "" {
		return nil, fmt.Errorf("path cannot be empty")
	}

	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid path: %s", path)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	current := r
	for i, part := range parts {
		val := current.getValue(part)
		if val == nil {
			return nil, fmt.Errorf("path '%s' not found at part '%s'", path, part)
		}

		if i == len(parts)-1 {
			slice := toInterfaceSlice(val)
			if slice == nil {
				return nil, fmt.Errorf("path '%s' cannot be converted to slice", path)
			}
			return slice, nil
		}

		nextRecord := convertToRecord(val)
		if nextRecord != nil {
			current = nextRecord
		} else {
			return nil, fmt.Errorf("path '%s' cannot be converted to Record at part '%s'", path, part)
		}
	}

	return nil, fmt.Errorf("path '%s' not found", path)
}

// GetRecord returns a single Record from a column
// 主要用途是FromJson的数据结构比较复杂,里面嵌套了其他的Record,
// 所以需要通过GetRecord来获取里面的Record
// 支持的类型：
// 1. *Record - 直接返回
// 2. Record - 返回指针
// 3. map[string]interface{} - 转换为 Record
func (r *Record) GetRecord(column string) (*Record, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	val := r.getValue(column)
	if val == nil {
		return nil, fmt.Errorf("column '%s' not found", column)
	}

	record := convertToRecord(val)
	if record == nil {
		return nil, fmt.Errorf("column '%s' cannot be converted to Record", column)
	}
	return record, nil
}

// GetRecordByPath 通过点分路径获取嵌套 Record
// 例如："level1.level2" 会先获取 level1，再从 level1 中获取 level2
// from json示例：
//
//	{
//	    "level1": {
//	        "level2": {
//	            "name": "张三",
//	            "age": 30
//	        }
//	    }
//	}
func (r *Record) GetRecordByPath(path string) (*Record, error) {
	if path == "" {
		return nil, fmt.Errorf("path cannot be empty")
	}

	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid path: %s", path)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	current := r
	for i, part := range parts {
		val := current.getValue(part)
		if val == nil {
			return nil, fmt.Errorf("path '%s' not found at part '%s'", path, part)
		}

		if i == len(parts)-1 {
			record := convertToRecord(val)
			if record == nil {
				return nil, fmt.Errorf("path '%s' cannot be converted to Record", path)
			}
			return record, nil
		}

		nextRecord := convertToRecord(val)
		if nextRecord != nil {
			current = nextRecord
		} else {
			return nil, fmt.Errorf("path '%s' cannot be converted to Record at part '%s'", path, part)
		}
	}

	return nil, fmt.Errorf("path '%s' not found", path)
}

// GetStringByPath 通过点分路径获取嵌套的字符串值
// 支持多层嵌套结构，如 "user.profile.name"
// 返回值和错误，如果路径不存在或无法转换为字符串，返回错误
func (r *Record) GetStringByPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return "", fmt.Errorf("invalid path: %s", path)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	current := r
	for i, part := range parts {
		val := current.getValue(part)
		if val == nil {
			return "", fmt.Errorf("path '%s' not found at part '%s'", path, part)
		}

		if i == len(parts)-1 {
			if record := convertToRecord(val); record != nil {
				return record.ToJson(), nil
			}
			str := Convert.ToString(val)
			return str, nil
		}

		nextRecord := convertToRecord(val)
		if nextRecord != nil {
			current = nextRecord
		} else {
			return "", fmt.Errorf("path '%s' cannot be converted to Record at part '%s'", path, part)
		}
	}

	return "", fmt.Errorf("path '%s' not found", path)
}

func convertToRecord(value interface{}) *Record {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case *Record:
		return v
	case Record:
		return &v
	case map[string]interface{}:
		return convertMapToRecord(v)
	case string:
		return convertJsonStringToRecord(v)
	case []byte:
		return convertJsonBytesToRecord(v)
	default:
		return nil
	}
}
func convertJsonStringToRecord(jsonStr string) *Record {
	if jsonStr == "" {
		return nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil
	}

	record := NewRecord()
	for k, v := range data {
		record.Set(k, v)
	}
	return record
}
func convertJsonBytesToRecord(data []byte) *Record {
	if len(data) == 0 {
		return nil
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}

	record := NewRecord()
	for k, v := range m {
		record.Set(k, v)
	}
	return record
}

// convertToRecordSliceSafe 安全的转换函数
func convertToRecordSliceSafe(value interface{}) []*Record {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case []*Record:
		// 返回副本以避免外部修改影响内部数据
		records := make([]*Record, len(v))
		copy(records, v)
		return records

	case []interface{}:
		return convertInterfaceSliceToRecords(v)
	case []map[string]interface{}:
		return convertMapSliceToRecords(v)
	default:
		rv := reflect.ValueOf(value)
		if rv.Kind() == reflect.Slice {
			return convertSliceViaReflection(rv)
		}
		return nil
	}
}
func convertSliceViaReflection(rv reflect.Value) []*Record {
	length := rv.Len()
	records := make([]*Record, 0, length)

	for i := 0; i < length; i++ {
		elem := rv.Index(i)
		if elem.CanInterface() {
			if record := convertToRecord(elem.Interface()); record != nil {
				records = append(records, record)
			}
		}
	}
	return records
}

// convertInterfaceSliceToRecords 转换 []interface{}
func convertInterfaceSliceToRecords(slice []interface{}) []*Record {
	if slice == nil {
		return nil
	}

	records := make([]*Record, 0, len(slice))
	for _, item := range slice {
		if item == nil {
			continue
		}

		switch v := item.(type) {
		case *Record:
			records = append(records, v)
		case map[string]interface{}:
			records = append(records, convertMapToRecord(v))
		default:
			// 其他类型忽略
		}
	}
	return records
}

// convertMapSliceToRecords 转换 []map[string]interface{}
func convertMapSliceToRecords(maps []map[string]interface{}) []*Record {
	records := make([]*Record, len(maps))
	for i, m := range maps {
		records[i] = convertMapToRecord(m)
	}
	return records
}

// convertMapToRecord 转换 map 为 Record
func convertMapToRecord(m map[string]interface{}) *Record {
	if m == nil {
		return nil
	}

	record := NewRecord()
	for k, v := range m {
		record.Set(k, v)
	}
	return record
}

// Delete is an alias for Remove
func (r *Record) Delete(column string) {
	r.Remove(column)
}

// Columns is an alias for Keys
func (r *Record) Columns() []string {
	return r.Keys()
}

// toInterfaceSlice 核心转换函数
func toInterfaceSlice(value interface{}) []interface{} {
	if value == nil {
		return nil
	}

	// 如果是 []interface{} 直接返回
	if slice, ok := value.([]interface{}); ok {
		return slice
	}

	// 通过反射处理
	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		// 如果不是切片/数组，尝试分割字符串或包装为单元素切片
		return wrapAsSlice(value)
	}

	// 处理切片/数组
	length := rv.Len()
	result := make([]interface{}, length)
	for i := 0; i < length; i++ {
		elem := rv.Index(i)
		if elem.CanInterface() {
			result[i] = elem.Interface()
		}
	}
	return result
}

// wrapAsSlice 将值包装为切片
func wrapAsSlice(value interface{}) []interface{} {
	if value == nil {
		return nil
	}

	// 如果是字符串，尝试分割
	if str, ok := value.(string); ok {
		return splitString(str)
	}

	// 如果是 []byte，转换为字符串再处理
	if bytes, ok := value.([]byte); ok {
		return splitString(string(bytes))
	}

	// 其他类型包装为单元素切片
	return []interface{}{value}
}

// splitString 分割字符串
func splitString(str string) []interface{} {
	if str == "" {
		return nil
	}

	// 检查常见分隔符
	delimiters := []string{",", ";", "|", " "}
	for _, delim := range delimiters {
		if strings.Contains(str, delim) {
			parts := strings.Split(str, delim)
			result := make([]interface{}, len(parts))
			for i, part := range parts {
				result[i] = strings.TrimSpace(part)
			}
			return result
		}
	}

	// 没有分隔符，作为单元素切片
	return []interface{}{str}
}

// IsEmpty checks if the Record is empty
func (r *Record) IsEmpty() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.columns) == 0
}

// Size returns the number of columns in the Record
func (r *Record) Size() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.columns)
}
