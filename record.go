package eorm

import (
	"encoding/json"
	"strings"
	"sync"
	"time"
)

type noCopy struct{}

func (*noCopy) Lock() {}

// Record represents a single record in the database, similar to JFinal's ActiveRecord
// columns 保留原始大小写用于生成 SQL，lowerKeyMap 用于大小写不敏感的快速查找
type Record struct {
	columns     map[string]interface{} // 原始键名 -> 值
	lowerKeyMap map[string]string      // 小写键名 -> 原始键名（用于快速查找）
	noCopy      noCopy
	mu          sync.RWMutex
}

// NewRecord creates a new empty Record
func NewRecord() *Record {
	return &Record{
		columns:     make(map[string]interface{}),
		lowerKeyMap: make(map[string]string),
	}
}

// FromMap (函数版) 从 map 创建新 Record
// 常用于 JSON 解析后的数据：record := eorm.FromMap(jsonMap)
func FromMap(m map[string]interface{}) *Record {
	return NewRecord().FromMap(m)
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
// 自动解引用指针类型，存储实际值（nil 指针除外）
func (r *Record) Set(column string, value interface{}) *Record {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 使用 derefPointer 函数自动解引用指针类型
	// derefPointer 会处理 nil 指针和多层指针的情况
	value = derefPointer(value)

	lowerKey := strings.ToLower(column)

	// 如果已存在相同小写键名的字段，更新原有字段
	if existingKey, exists := r.lowerKeyMap[lowerKey]; exists {
		r.columns[existingKey] = value
		return r
	}

	// 新字段：保存原始大小写和映射关系
	r.columns[column] = value
	r.lowerKeyMap[lowerKey] = column
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

// Keys returns all column names
func (r *Record) Keys() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	keys := make([]string, 0, len(r.columns))
	for k := range r.columns {
		keys = append(keys, k)
	}
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
	}
}

// Clear clears all columns
func (r *Record) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.columns = make(map[string]interface{})
	r.lowerKeyMap = make(map[string]string)
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
func (r *Record) ToJson() string {
	data, err := r.MarshalJSON()
	if err != nil {
		return "{}"
	}
	return string(data)
}

// MarshalJSON implements the json.Marshaler interface
func (r *Record) MarshalJSON() ([]byte, error) {
	if r == nil {
		return []byte("{}"), nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return json.Marshal(r.columns)
}

// Clone creates a deep copy of the Record
func (r *Record) Clone() *Record {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	newRecord := NewRecord()
	for k, v := range r.columns {
		// 这里对 map 和 slice 进行简单的深拷贝处理
		// 数据库返回的通常是基本类型、time.Time 或 []byte
		newRecord.setDirect(k, cloneValue(v))
	}
	return newRecord
}

// cloneValue 辅助函数，用于克隆 interface{} 里面的值
func cloneValue(v interface{}) interface{} {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case []byte:
		newByte := make([]byte, len(val))
		copy(newByte, val)
		return newByte
	// 常见的 map 和 slice 需要递归
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

// UnmarshalJSON implements the json.Unmarshaler interface
func (r *Record) UnmarshalJSON(data []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 清空现有数据
	r.columns = make(map[string]interface{})
	r.lowerKeyMap = make(map[string]string)

	// 反序列化
	if err := json.Unmarshal(data, &r.columns); err != nil {
		return err
	}

	// 重建小写映射
	for k := range r.columns {
		r.lowerKeyMap[strings.ToLower(k)] = k
	}

	return nil
}

// FromJson parses JSON string into the Record
func (r *Record) FromJson(jsonStr string) error {
	return r.UnmarshalJSON([]byte(jsonStr))
}

// ToStruct converts the Record to a struct
func (r *Record) ToStruct(dest interface{}) error {
	return ToStruct(r, dest)
}

// FromStruct populates the Record from a struct
func (r *Record) FromStruct(src interface{}) error {
	return FromStruct(src, r)
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
