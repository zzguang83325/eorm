package eorm

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// 结构体反射缓存的存储库名称（内部使用，不对外暴露）
const structCacheRepository = "__eorm_struct_cache__"

// structFieldInfo 存储单个字段的缓存信息
type structFieldInfo struct {
	fieldIndex int          // 字段索引
	columnName string       // 列名（从 tag 解析）
	fieldType  reflect.Type // 字段类型
	fieldKind  reflect.Kind // 字段种类
	canSet     bool         // 是否可设置（可导出）
}

// structCacheInfo 存储整个结构体的缓存信息
type structCacheInfo struct {
	fields []structFieldInfo // 字段信息列表
}

// getStructCacheInfo 获取或创建结构体的缓存信息
// 使用 localCache 缓存结构体的反射信息，避免重复解析
func getStructCacheInfo(structType reflect.Type) *structCacheInfo {
	// 使用 Type 的字符串表示作为缓存键
	// 这样可以自动处理多数据库同名表的问题（不同包的同名结构体有不同的 Type）
	cacheKey := structType.String()

	// 尝试从本地缓存获取
	if cached, ok := LocalCacheGet(structCacheRepository, cacheKey); ok {
		if info, ok := cached.(*structCacheInfo); ok {
			return info
		}
	}

	// 缓存未命中，解析结构体字段信息
	info := &structCacheInfo{
		fields: make([]structFieldInfo, 0, structType.NumField()),
	}

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)

		// 解析列名（只解析一次，后续从缓存读取）
		colName := field.Tag.Get("column")
		if colName == "" {
			colName = field.Tag.Get("db")
		}
		if colName == "" {
			colName = field.Tag.Get("json")
		}
		if colName == "-" {
			continue
		}
		if colName == "" {
			colName = strings.ToLower(field.Name)
		}

		// 处理逗号分隔的 tag（如 json:"id,omitempty"）
		if idx := strings.Index(colName, ","); idx != -1 {
			colName = colName[:idx]
		}

		if colName == "-" {
			continue
		}

		// 存储字段信息
		info.fields = append(info.fields, structFieldInfo{
			fieldIndex: i,
			columnName: colName,
			fieldType:  field.Type,
			fieldKind:  field.Type.Kind(),
			canSet:     field.IsExported(), // Go 1.17+ 使用 IsExported 判断是否可导出
		})
	}

	// 存入本地缓存（永不过期，因为结构体定义在运行时不会改变）
	LocalCacheSet(structCacheRepository, cacheKey, info, 0)

	return info
}

// ToStruct converts a single Record to a struct.
// dest must be a pointer to a struct.
func ToStruct(r *Record, dest interface{}) error {
	if r == nil {
		return fmt.Errorf("eorm: record is nil")
	}
	if dest == nil {
		return fmt.Errorf("eorm: dest is nil")
	}

	val := reflect.ValueOf(dest)
	if val.Kind() != reflect.Ptr || (val.Elem().Kind() != reflect.Slice && val.Elem().Kind() != reflect.Struct) {
		return fmt.Errorf("eorm: dest must be a pointer to a struct")
	}

	return setStructFromRecord(val.Elem(), r)
}

// FromStruct populates a Record from a struct.
func FromStruct(src interface{}, r *Record) error {
	if src == nil {
		return fmt.Errorf("eorm: src is nil")
	}
	if r == nil {
		return fmt.Errorf("eorm: record is nil")
	}

	val := reflect.ValueOf(src)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("eorm: src must be a struct or pointer to struct")
	}

	return setRecordFromStruct(r, val)
}

// ToRecord converts a struct to a new Record.
func ToRecord(src interface{}) *Record {
	r := NewRecord()
	_ = FromStruct(src, r)
	return r
}

func setStructFromRecord(structVal reflect.Value, r *Record) error {
	structType := structVal.Type()

	// 获取缓存的结构体信息（首次调用会解析并缓存，后续直接使用缓存）
	cacheInfo := getStructCacheInfo(structType)

	// 使用缓存的字段信息，避免重复反射解析
	for _, fieldInfo := range cacheInfo.fields {
		fieldVal := structVal.Field(fieldInfo.fieldIndex)

		// 使用缓存的 canSet 信息
		if !fieldInfo.canSet {
			continue
		}

		val := r.Get(fieldInfo.columnName)
		if val == nil {
			continue
		}

		if err := setFieldValue(fieldVal, val); err != nil {
			// 获取字段名用于错误信息
			fieldName := structType.Field(fieldInfo.fieldIndex).Name
			return fmt.Errorf("field %s: %v", fieldName, err)
		}
	}
	return nil
}

func setRecordFromStruct(r *Record, structVal reflect.Value) error {
	structType := structVal.Type()

	// 获取缓存的结构体信息（首次调用会解析并缓存，后续直接使用缓存）
	cacheInfo := getStructCacheInfo(structType)

	// 使用缓存的字段信息，避免重复反射解析
	for _, fieldInfo := range cacheInfo.fields {
		fieldVal := structVal.Field(fieldInfo.fieldIndex)

		if !fieldVal.CanInterface() {
			continue
		}

		// 跳过 nil 指针字段，这样它们就不会被包含在 Record 中
		if fieldVal.Kind() == reflect.Ptr && fieldVal.IsNil() {
			continue
		}

		r.Set(fieldInfo.columnName, fieldVal.Interface())
	}
	return nil
}

func setFieldValue(field reflect.Value, value interface{}) error {
	v := reflect.ValueOf(value)

	// Handle pointer target
	if field.Kind() == reflect.Ptr {
		if v.Kind() == reflect.Ptr && v.IsNil() {
			field.Set(reflect.Zero(field.Type()))
			return nil
		}
		// Create new instance of pointer type if needed
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return setFieldValue(field.Elem(), value)
	}

	// Unpack pointer value
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			field.Set(reflect.Zero(field.Type()))
			return nil
		}
		v = v.Elem()
		value = v.Interface()
	}

	// Try direct set
	if v.Type().AssignableTo(field.Type()) {
		field.Set(v)
		return nil
	}

	// Advanced conversions
	switch field.Kind() {
	case reflect.String:
		field.SetString(fmt.Sprint(value))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// 处理 []byte 类型
		if b, ok := value.([]byte); ok {
			value = string(b)
		}
		val, err := Convert.ToInt64WithError(value)
		if err != nil {
			return err
		}
		field.SetInt(val)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		// 处理 []byte 类型
		if b, ok := value.([]byte); ok {
			value = string(b)
		}
		val, err := Convert.ToUint64WithError(value)
		if err != nil {
			return err
		}
		field.SetUint(val)
	case reflect.Float32, reflect.Float64:
		// 处理 []byte 类型
		if b, ok := value.([]byte); ok {
			value = string(b)
		}
		val, err := Convert.ToFloat64WithError(value)
		if err != nil {
			return err
		}
		field.SetFloat(val)
	case reflect.Bool:
		// 处理 []byte 类型
		if b, ok := value.([]byte); ok {
			value = string(b)
		}
		val, err := Convert.ToBoolWithError(value)
		if err != nil {
			return err
		}
		field.SetBool(val)
	default:
		// Special handling for time.Time
		if field.Type() == reflect.TypeOf(time.Time{}) {
			t, err := Convert.ToTimeWithError(value)
			if err != nil {
				return err
			}
			field.Set(reflect.ValueOf(t))
			return nil
		}
		return fmt.Errorf("cannot convert %T to %s", value, field.Type())
	}

	return nil
}

// ToStructs converts a slice of Records to a slice of structs or struct pointers.
// dest must be a pointer to a slice.
func ToStructs(records []*Record, dest interface{}) error {
	if dest == nil {
		return fmt.Errorf("eorm: dest cannot be nil")
	}

	val := reflect.ValueOf(dest)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("eorm: dest must be a pointer to a slice")
	}

	sliceVal := val.Elem()
	// Clear the slice before filling it
	sliceVal.Set(reflect.MakeSlice(sliceVal.Type(), 0, len(records)))
	elemType := sliceVal.Type().Elem()

	// Handle both slice of structs and slice of struct pointers
	isPtr := elemType.Kind() == reflect.Ptr
	var baseType reflect.Type
	if isPtr {
		baseType = elemType.Elem()
	} else {
		baseType = elemType
	}

	for i := range records {
		newElem := reflect.New(baseType)
		if err := ToStruct(records[i], newElem.Interface()); err != nil {
			return err
		}

		if isPtr {
			sliceVal.Set(reflect.Append(sliceVal, newElem))
		} else {
			sliceVal.Set(reflect.Append(sliceVal, newElem.Elem()))
		}
	}

	return nil
}

// derefPointer 解引用指针，返回实际值
// 如果是 nil 指针，返回 nil
// 如果不是指针，直接返回原值
func derefPointer(a any) any {
	if a == nil {
		return nil
	}
	v := reflect.ValueOf(a)
	// 如果是指针，解引用
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	// 安全检查：确保可以调用 Interface()
	if v.CanInterface() {
		return v.Interface()
	}
	return nil
}

// convertStruct 类型转换函数命名空间
type convertStruct struct{}

// Convert 提供类型转换函数
var Convert = convertStruct{}

// ToBoolWithError 将任意类型转换为 bool，转换失败返回错误
// 支持的字符串格式：
// - 标准格式：1, t, T, true, TRUE, True, 0, f, F, false, FALSE, False
// - 扩展格式：yes, Yes, YES, no, No, NO, on, On, ON, off, Off, OFF
func (convertStruct) ToBoolWithError(a any) (bool, error) {
	// 解引用指针
	a = derefPointer(a)
	if a == nil {
		return false, fmt.Errorf("cannot convert nil to bool")
	}
	switch v := a.(type) {
	case bool:
		return v, nil
	case int, int8, int16, int32, int64:
		return reflect.ValueOf(v).Int() != 0, nil
	case uint, uint8, uint16, uint32, uint64:
		return reflect.ValueOf(v).Uint() != 0, nil
	case float32, float64:
		return reflect.ValueOf(v).Float() != 0, nil
	case string:
		// 先尝试标准的 ParseBool
		if b, err := strconv.ParseBool(v); err == nil {
			return b, nil
		}
		// 扩展支持：yes/no, on/off（大小写不敏感）
		lower := strings.ToLower(strings.TrimSpace(v))
		switch lower {
		case "yes", "on":
			return true, nil
		case "no", "off":
			return false, nil
		default:
			return false, fmt.Errorf("cannot parse %q as bool", v)
		}
	default:
		return false, fmt.Errorf("cannot convert %T to bool", a)
	}
}

// ToBool 将任意类型转换为 bool，转换失败返回默认值
func (convertStruct) ToBool(a any, defaultValue ...bool) bool {
	v, err := Convert.ToBoolWithError(a)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return false
	}
	return v
}

// ToIntWithError 将任意类型转换为 int，转换失败返回错误
func (convertStruct) ToIntWithError(a any) (int, error) {
	// 解引用指针
	a = derefPointer(a)
	if a == nil {
		return 0, fmt.Errorf("cannot convert nil to int")
	}
	switch v := a.(type) {
	case int:
		return v, nil
	case int8:
		return int(v), nil
	case int16:
		return int(v), nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case uint:
		return int(v), nil
	case uint8:
		return int(v), nil
	case uint16:
		return int(v), nil
	case uint32:
		return int(v), nil
	case uint64:
		return int(v), nil
	case float32:
		return int(v), nil
	case float64:
		return int(v), nil
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	case string:
		return strconv.Atoi(v)
	default:
		return 0, fmt.Errorf("cannot convert %T to int", a)
	}
}

// ToInt 将任意类型转换为 int，转换失败返回默认值
func (convertStruct) ToInt(a any, defaultValue ...int) int {
	v, err := Convert.ToIntWithError(a)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return v
}

// ToInt64WithError 将任意类型转换为 int64，转换失败返回错误
func (convertStruct) ToInt64WithError(a any) (int64, error) {
	// 解引用指针
	a = derefPointer(a)
	if a == nil {
		return 0, fmt.Errorf("cannot convert nil to int64")
	}
	switch v := a.(type) {
	case int:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case uint:
		return int64(v), nil
	case uint8:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		return int64(v), nil
	case float32:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to int64", a)
	}
}

// ToInt64 将任意类型转换为 int64，转换失败返回默认值
func (convertStruct) ToInt64(a any, defaultValue ...int64) int64 {
	v, err := Convert.ToInt64WithError(a)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return v
}

// ToInt8WithError 将任意类型转换为 int8，转换失败返回错误
func (convertStruct) ToInt8WithError(a any) (int8, error) {
	v, err := Convert.ToInt64WithError(a)
	if err != nil {
		return 0, err
	}
	if v < -128 || v > 127 {
		return 0, fmt.Errorf("value %d overflows int8", v)
	}
	return int8(v), nil
}

// ToInt8 将任意类型转换为 int8，转换失败返回默认值
func (convertStruct) ToInt8(a any, defaultValue ...int8) int8 {
	v, err := Convert.ToInt8WithError(a)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return v
}

// ToInt16WithError 将任意类型转换为 int16，转换失败返回错误
func (convertStruct) ToInt16WithError(a any) (int16, error) {
	v, err := Convert.ToInt64WithError(a)
	if err != nil {
		return 0, err
	}
	if v < -32768 || v > 32767 {
		return 0, fmt.Errorf("value %d overflows int16", v)
	}
	return int16(v), nil
}

// ToInt16 将任意类型转换为 int16，转换失败返回默认值
func (convertStruct) ToInt16(a any, defaultValue ...int16) int16 {
	v, err := Convert.ToInt16WithError(a)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return v
}

// ToInt32WithError 将任意类型转换为 int32，转换失败返回错误
func (convertStruct) ToInt32WithError(a any) (int32, error) {
	v, err := Convert.ToInt64WithError(a)
	if err != nil {
		return 0, err
	}
	if v < -2147483648 || v > 2147483647 {
		return 0, fmt.Errorf("value %d overflows int32", v)
	}
	return int32(v), nil
}

// ToInt32 将任意类型转换为 int32，转换失败返回默认值
func (convertStruct) ToInt32(a any, defaultValue ...int32) int32 {
	v, err := Convert.ToInt32WithError(a)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return v
}

// ToUintWithError 将任意类型转换为 uint，转换失败返回错误
func (convertStruct) ToUintWithError(a any) (uint, error) {
	v, err := Convert.ToInt64WithError(a)
	if err != nil {
		return 0, err
	}
	if v < 0 {
		return 0, fmt.Errorf("cannot convert negative value %d to uint", v)
	}
	return uint(v), nil
}

// ToUint 将任意类型转换为 uint，转换失败返回默认值
func (convertStruct) ToUint(a any, defaultValue ...uint) uint {
	v, err := Convert.ToUintWithError(a)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return v
}

// ToUint8WithError 将任意类型转换为 uint8，转换失败返回错误
func (convertStruct) ToUint8WithError(a any) (uint8, error) {
	v, err := Convert.ToInt64WithError(a)
	if err != nil {
		return 0, err
	}
	if v < 0 || v > 255 {
		return 0, fmt.Errorf("value %d overflows uint8", v)
	}
	return uint8(v), nil
}

// ToUint8 将任意类型转换为 uint8，转换失败返回默认值
func (convertStruct) ToUint8(a any, defaultValue ...uint8) uint8 {
	v, err := Convert.ToUint8WithError(a)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return v
}

// ToUint16WithError 将任意类型转换为 uint16，转换失败返回错误
func (convertStruct) ToUint16WithError(a any) (uint16, error) {
	v, err := Convert.ToInt64WithError(a)
	if err != nil {
		return 0, err
	}
	if v < 0 || v > 65535 {
		return 0, fmt.Errorf("value %d overflows uint16", v)
	}
	return uint16(v), nil
}

// ToUint16 将任意类型转换为 uint16，转换失败返回默认值
func (convertStruct) ToUint16(a any, defaultValue ...uint16) uint16 {
	v, err := Convert.ToUint16WithError(a)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return v
}

// ToUint32WithError 将任意类型转换为 uint32，转换失败返回错误
func (convertStruct) ToUint32WithError(a any) (uint32, error) {
	v, err := Convert.ToInt64WithError(a)
	if err != nil {
		return 0, err
	}
	if v < 0 || v > 4294967295 {
		return 0, fmt.Errorf("value %d overflows uint32", v)
	}
	return uint32(v), nil
}

// ToUint32 将任意类型转换为 uint32，转换失败返回默认值
func (convertStruct) ToUint32(a any, defaultValue ...uint32) uint32 {
	v, err := Convert.ToUint32WithError(a)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return v
}

// ToUint64WithError 将任意类型转换为 uint64，转换失败返回错误
func (convertStruct) ToUint64WithError(a any) (uint64, error) {
	v, err := Convert.ToInt64WithError(a)
	if err != nil {
		return 0, err
	}
	if v < 0 {
		return 0, fmt.Errorf("cannot convert negative value %d to uint64", v)
	}
	return uint64(v), nil
}

// ToUint64 将任意类型转换为 uint64，转换失败返回默认值
func (convertStruct) ToUint64(a any, defaultValue ...uint64) uint64 {
	v, err := Convert.ToUint64WithError(a)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return v
}

// ToFloat32WithError 将任意类型转换为 float32，转换失败返回错误
func (convertStruct) ToFloat32WithError(a any) (float32, error) {
	v, err := Convert.ToFloat64WithError(a)
	if err != nil {
		return 0, err
	}
	return float32(v), nil
}

// ToFloat32 将任意类型转换为 float32，转换失败返回默认值
func (convertStruct) ToFloat32(a any, defaultValue ...float32) float32 {
	v, err := Convert.ToFloat32WithError(a)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0.0
	}
	return v
}

// ToStringWithError 将任意类型转换为 string，转换失败返回错误
func (convertStruct) ToStringWithError(a any) (string, error) {
	// 解引用指针
	a = derefPointer(a)
	if a == nil {
		return "", nil
	}
	switch v := a.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v), nil
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v), nil
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32), nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(v), nil
	default:
		bs, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("cannot convert %T to string: %w", a, err)
		}
		return string(bs), nil
	}
}

// ToString 将任意类型转换为 string，转换失败返回默认值
func (convertStruct) ToString(a any, defaultValue ...string) string {
	v, err := Convert.ToStringWithError(a)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return ""
	}
	return v
}

// ToFloat64WithError 将任意类型转换为 float64，转换失败返回错误
func (convertStruct) ToFloat64WithError(a any) (float64, error) {
	// 解引用指针
	a = derefPointer(a)
	if a == nil {
		return 0, fmt.Errorf("cannot convert nil to float64")
	}
	switch v := a.(type) {
	case int:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case bool:
		if v {
			return 1.0, nil
		}
		return 0.0, nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", a)
	}
}

// ToFloat64 将任意类型转换为 float64，转换失败返回默认值
func (convertStruct) ToFloat64(a any, defaultValue ...float64) float64 {
	v, err := Convert.ToFloat64WithError(a)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0.0
	}
	return v
}

// ToDurationWithError 将任意类型转换为 time.Duration，转换失败返回错误
func (convertStruct) ToDurationWithError(a any) (time.Duration, error) {
	// 解引用指针
	a = derefPointer(a)
	if a == nil {
		return 0, fmt.Errorf("cannot convert nil to time.Duration")
	}
	switch v := a.(type) {
	case time.Duration:
		return v, nil
	case int, int8, int16, int32, int64:
		return time.Duration(reflect.ValueOf(v).Int()), nil
	case uint, uint8, uint16, uint32, uint64:
		return time.Duration(reflect.ValueOf(v).Uint()), nil
	case float32, float64:
		return time.Duration(reflect.ValueOf(v).Float()), nil
	case string:
		return time.ParseDuration(v)
	default:
		return 0, fmt.Errorf("cannot convert %T to time.Duration", a)
	}
}

// ToDuration 将任意类型转换为 time.Duration，转换失败返回默认值
func (convertStruct) ToDuration(a any, defaultValue ...time.Duration) time.Duration {
	v, err := Convert.ToDurationWithError(a)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return v
}

// ToIntPtr 将 int 值转换为 *int 指针
func (convertStruct) ToIntPtr(v int) *int {
	return &v
}

// ToInt8Ptr 将 int8 值转换为 *int8 指针
func (convertStruct) ToInt8Ptr(v int8) *int8 {
	return &v
}

// ToInt16Ptr 将 int16 值转换为 *int16 指针
func (convertStruct) ToInt16Ptr(v int16) *int16 {
	return &v
}

// ToInt32Ptr 将 int32 值转换为 *int32 指针
func (convertStruct) ToInt32Ptr(v int32) *int32 {
	return &v
}

// ToInt64Ptr 将 int64 值转换为 *int64 指针
func (convertStruct) ToInt64Ptr(v int64) *int64 {
	return &v
}

// ToUintPtr 将 uint 值转换为 *uint 指针
func (convertStruct) ToUintPtr(v uint) *uint {
	return &v
}

// ToUint8Ptr 将 uint8 值转换为 *uint8 指针
func (convertStruct) ToUint8Ptr(v uint8) *uint8 {
	return &v
}

// ToUint16Ptr 将 uint16 值转换为 *uint16 指针
func (convertStruct) ToUint16Ptr(v uint16) *uint16 {
	return &v
}

// ToUint32Ptr 将 uint32 值转换为 *uint32 指针
func (convertStruct) ToUint32Ptr(v uint32) *uint32 {
	return &v
}

// ToUint64Ptr 将 uint64 值转换为 *uint64 指针
func (convertStruct) ToUint64Ptr(v uint64) *uint64 {
	return &v
}

// ToFloat32Ptr 将 float32 值转换为 *float32 指针
func (convertStruct) ToFloat32Ptr(v float32) *float32 {
	return &v
}

// ToFloat64Ptr 将 float64 值转换为 *float64 指针
func (convertStruct) ToFloat64Ptr(v float64) *float64 {
	return &v
}

// ToBoolPtr 将 bool 值转换为 *bool 指针
func (convertStruct) ToBoolPtr(v bool) *bool {
	return &v
}

// ToStringPtr 将 string 值转换为 *string 指针
func (convertStruct) ToStringPtr(v string) *string {
	return &v
}

// IntPtrValue 将 *int 指针转换为 int 值，nil 返回默认值
func (convertStruct) IntPtrValue(ptr *int, defaultValue ...int) int {
	if ptr == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return *ptr
}

// Int8PtrValue 将 *int8 指针转换为 int8 值，nil 返回默认值
func (convertStruct) Int8PtrValue(ptr *int8, defaultValue ...int8) int8 {
	if ptr == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return *ptr
}

// Int16PtrValue 将 *int16 指针转换为 int16 值，nil 返回默认值
func (convertStruct) Int16PtrValue(ptr *int16, defaultValue ...int16) int16 {
	if ptr == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return *ptr
}

// Int32PtrValue 将 *int32 指针转换为 int32 值，nil 返回默认值
func (convertStruct) Int32PtrValue(ptr *int32, defaultValue ...int32) int32 {
	if ptr == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return *ptr
}

// Int64PtrValue 将 *int64 指针转换为 int64 值，nil 返回默认值
func (convertStruct) Int64PtrValue(ptr *int64, defaultValue ...int64) int64 {
	if ptr == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return *ptr
}

// UintPtrValue 将 *uint 指针转换为 uint 值，nil 返回默认值
func (convertStruct) UintPtrValue(ptr *uint, defaultValue ...uint) uint {
	if ptr == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return *ptr
}

// Uint8PtrValue 将 *uint8 指针转换为 uint8 值，nil 返回默认值
func (convertStruct) Uint8PtrValue(ptr *uint8, defaultValue ...uint8) uint8 {
	if ptr == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return *ptr
}

// Uint16PtrValue 将 *uint16 指针转换为 uint16 值，nil 返回默认值
func (convertStruct) Uint16PtrValue(ptr *uint16, defaultValue ...uint16) uint16 {
	if ptr == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return *ptr
}

// Uint32PtrValue 将 *uint32 指针转换为 uint32 值，nil 返回默认值
func (convertStruct) Uint32PtrValue(ptr *uint32, defaultValue ...uint32) uint32 {
	if ptr == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return *ptr
}

// Uint64PtrValue 将 *uint64 指针转换为 uint64 值，nil 返回默认值
func (convertStruct) Uint64PtrValue(ptr *uint64, defaultValue ...uint64) uint64 {
	if ptr == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	return *ptr
}

// Float32PtrValue 将 *float32 指针转换为 float32 值，nil 返回默认值
func (convertStruct) Float32PtrValue(ptr *float32, defaultValue ...float32) float32 {
	if ptr == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0.0
	}
	return *ptr
}

// Float64PtrValue 将 *float64 指针转换为 float64 值，nil 返回默认值
func (convertStruct) Float64PtrValue(ptr *float64, defaultValue ...float64) float64 {
	if ptr == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0.0
	}
	return *ptr
}

// BoolPtrValue 将 *bool 指针转换为 bool 值，nil 返回默认值
func (convertStruct) BoolPtrValue(ptr *bool, defaultValue ...bool) bool {
	if ptr == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return false
	}
	return *ptr
}

// StringPtrValue 将 *string 指针转换为 string 值，nil 返回默认值
func (convertStruct) StringPtrValue(ptr *string, defaultValue ...string) string {
	if ptr == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return ""
	}
	return *ptr
}

// ToTimeWithError 将任意类型转换为 time.Time，转换失败返回错误
func (convertStruct) ToTimeWithError(a any) (time.Time, error) {
	// 解引用指针
	a = derefPointer(a)
	if a == nil {
		return time.Time{}, fmt.Errorf("cannot convert nil to time.Time")
	}
	switch v := a.(type) {
	case time.Time:
		return v, nil
	case string:
		// 空字符串返回零值
		if v == "" {
			return time.Time{}, nil
		}

		// 根据字符串长度和特征快速判断格式
		length := len(v)

		// 标准日期时间格式：2006-01-02 15:04:05 (19位) - 最常用，优先处理
		if length == 19 && v[10] == ' ' {
			if v[4] == '-' {
				if t, err := time.Parse("2006-01-02 15:04:05", v); err == nil {
					return t, nil
				}
			} else if v[4] == '/' {
				if t, err := time.Parse("2006/01/02 15:04:05", v); err == nil {
					return t, nil
				}
			}
		}

		// 带纳秒的格式 (>19位) - 数据库常用
		if length > 19 && length < 30 && v[10] == ' ' {
			if v[4] == '-' {
				if t, err := time.Parse("2006-01-02 15:04:05.999999999", v); err == nil {
					return t, nil
				}
			} else if v[4] == '/' {
				if t, err := time.Parse("2006/01/02 15:04:05.999999999", v); err == nil {
					return t, nil
				}
			}
		}

		// 日期格式：2006-01-02 或 2006/01/02 (10位)
		if length == 10 {
			if v[4] == '-' {
				if t, err := time.Parse("2006-01-02", v); err == nil {
					return t, nil
				}
			} else if v[4] == '/' {
				if t, err := time.Parse("2006/01/02", v); err == nil {
					return t, nil
				}
			}
		}

		// ISO 8601 / RFC3339 格式（包含 T）
		if length >= 20 && v[10] == 'T' {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				return t, nil
			}
			if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
				return t, nil
			}
		}

		// 紧凑格式：20060102 (8位)
		if length == 8 && v[0] >= '0' && v[0] <= '9' {
			if t, err := time.Parse("20060102", v); err == nil {
				return t, nil
			}
		}

		// 时间格式：15:04:05 (8位) 或 15:04 (5位)
		if length == 8 && v[2] == ':' && v[5] == ':' {
			if t, err := time.Parse("15:04:05", v); err == nil {
				return t, nil
			}
		}
		if length == 5 && v[2] == ':' {
			if t, err := time.Parse("15:04", v); err == nil {
				return t, nil
			}
		}

		// 标准日期时间格式：2006-01-02 15:04:05 (19位)
		if length == 19 && v[10] == ' ' {
			if v[4] == '-' {
				if t, err := time.Parse("2006-01-02 15:04:05", v); err == nil {
					return t, nil
				}
			} else if v[4] == '/' {
				if t, err := time.Parse("2006/01/02 15:04:05", v); err == nil {
					return t, nil
				}
			}
		}

		// 带纳秒的格式 (>19位)
		if length > 19 && v[10] == ' ' {
			if v[4] == '-' {
				if t, err := time.Parse("2006-01-02 15:04:05.999999999", v); err == nil {
					return t, nil
				}
			} else if v[4] == '/' {
				if t, err := time.Parse("2006/01/02 15:04:05.999999999", v); err == nil {
					return t, nil
				}
			}
		}

		// ISO 8601 / RFC3339 格式（包含 T）
		if length >= 20 && v[10] == 'T' {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				return t, nil
			}
			if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
				return t, nil
			}
		}

		// 中文格式
		if length >= 11 {
			// 检查是否包含中文字符
			hasChineseYear := false
			for _, r := range v {
				if r == '年' {
					hasChineseYear = true
					break
				}
			}
			if hasChineseYear {
				formats := []string{
					"2006年01月02日 15:04:05",
					"2006年01月02日",
				}
				for _, f := range formats {
					if t, err := time.Parse(f, v); err == nil {
						return t, nil
					}
				}
			}
		}

		// 其他 RFC 格式（不常用，放在最后）
		formats := []string{
			time.RFC1123,
			time.RFC1123Z,
			time.RFC822,
			time.RFC822Z,
			time.RFC850,
			time.ANSIC,
			time.UnixDate,
			time.RubyDate,
		}
		for _, f := range formats {
			if t, err := time.Parse(f, v); err == nil {
				return t, nil
			}
		}

		return time.Time{}, fmt.Errorf("cannot parse string %q to time.Time", v)
	case int64:
		// Unix 时间戳（秒）
		return time.Unix(v, 0), nil
	case int:
		// Unix 时间戳（秒）
		return time.Unix(int64(v), 0), nil
	case int32:
		// Unix 时间戳（秒）
		return time.Unix(int64(v), 0), nil
	case []byte:
		// 处理 []byte 类型
		return Convert.ToTimeWithError(string(v))
	default:
		return time.Time{}, fmt.Errorf("cannot convert %T to time.Time", a)
	}
}

// ToTime 将任意类型转换为 time.Time，转换失败返回默认值
func (convertStruct) ToTime(a any, defaultValue ...time.Time) time.Time {
	v, err := Convert.ToTimeWithError(a)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return time.Time{}
	}
	return v
}
