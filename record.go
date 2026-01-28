package eorm

import (
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

// String 实现 Stringer 接口，返回 JSON 格式的字符串
// 这样可以直接使用 fmt.Printf("%v", record) 输出 JSON 格式
func (r *Record) String() string {
	return r.ToJson()
}

func (r *Record) MarshalJSON() ([]byte, error) {
	if r == nil {
		return []byte("{}"), nil
	}

	// 使用递归转换
	data, err := json.Marshal(r.toMapRecursive(make(map[uintptr]bool), 0))
	if err != nil {
		return []byte("{}"), err
	}
	return data, nil
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
	for k, v := range r.columns {
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

	return newRecord
}

// cloneValue 辅助函数，用于克隆 interface{} 里面的值
func cloneValue(v interface{}) interface{} {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case *Record:
		// 递归克隆嵌套的 Record
		return val.Clone()
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
func (r *Record) FromJson(jsonStr string) *Record {
	r.mu.Lock()
	defer r.mu.Unlock()

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return r
	}

	for k, v := range data {
		r.columns[k] = r.convertJsonValue(v)
		r.lowerKeyMap[strings.ToLower(k)] = k
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

	// 直接复制值
	for key, value := range src.columns {
		r.columns[key] = value
		r.lowerKeyMap[strings.ToLower(key)] = key
	}
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
