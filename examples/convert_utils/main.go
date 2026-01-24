package main

import (
	"fmt"
	"time"

	"github.com/zzguang83325/eorm"
)

// Convert 工具函数示例
// 演示如何使用 eorm.Convert 命名空间下的类型转换函数
func main() {
	fmt.Println("========================================")
	fmt.Println("      eorm Convert 工具函数示例")
	fmt.Println("========================================")

	// 1. 基础类型转换
	fmt.Println("\n[1] 基础类型转换")
	demonstrateBasicConversions()

	// 2. 带错误处理的类型转换
	fmt.Println("\n[2] 带错误处理的类型转换")
	demonstrateErrorHandlingConversions()

	// 3. 时间类型转换
	fmt.Println("\n[3] 时间类型转换")
	demonstrateTimeConversions()

	// 4. 指针转换
	fmt.Println("\n[4] 指针转换")
	demonstratePointerConversions()

	// 5. 数据转换函数
	fmt.Println("\n[5] 数据转换函数")
	demonstrateDataConversions()

	// 6. 实际应用场景
	fmt.Println("\n[6] 实际应用场景")
	demonstrateRealWorldUsage()

	fmt.Println("\n========================================")
	fmt.Println("      Convert 工具函数示例演示完成")
	fmt.Println("========================================")
}

// demonstrateBasicConversions 基础类型转换
func demonstrateBasicConversions() {
	fmt.Println("场景：基础数据类型之间的转换")

	// 整数转换
	fmt.Println("  整数转换:")
	intVal := eorm.Convert.ToInt("123")
	fmt.Printf("    String '123' -> Int: %d\n", intVal)

	int8Val := eorm.Convert.ToInt8("100")
	fmt.Printf("    String '100' -> Int8: %d\n", int8Val)

	int16Val := eorm.Convert.ToInt16("30000")
	fmt.Printf("    String '30000' -> Int16: %d\n", int16Val)

	int32Val := eorm.Convert.ToInt32("2000000000")
	fmt.Printf("    String '2000000000' -> Int32: %d\n", int32Val)

	int64Val := eorm.Convert.ToInt64("9000000000000000000")
	fmt.Printf("    String '9000000000000000000' -> Int64: %d\n", int64Val)

	// 无符号整数转换
	fmt.Println("  无符号整数转换:")
	uintVal := eorm.Convert.ToUint("123")
	fmt.Printf("    String '123' -> Uint: %d\n", uintVal)

	uint8Val := eorm.Convert.ToUint8("200")
	fmt.Printf("    String '200' -> Uint8: %d\n", uint8Val)

	uint16Val := eorm.Convert.ToUint16("60000")
	fmt.Printf("    String '60000' -> Uint16: %d\n", uint16Val)

	uint32Val := eorm.Convert.ToUint32("4000000000")
	fmt.Printf("    String '4000000000' -> Uint32: %d\n", uint32Val)

	uint64Val := eorm.Convert.ToUint64("18000000000000000000")
	fmt.Printf("    String '18000000000000000000' -> Uint64: %d\n", uint64Val)

	// 浮点数转换
	fmt.Println("  浮点数转换:")
	float32Val := eorm.Convert.ToFloat32("3.14")
	fmt.Printf("    String '3.14' -> Float32: %.2f\n", float32Val)

	float64Val := eorm.Convert.ToFloat64("2.718281828")
	fmt.Printf("    String '2.718281828' -> Float64: %.6f\n", float64Val)

	// 布尔值转换
	fmt.Println("  布尔值转换:")
	boolVal1 := eorm.Convert.ToBool("true")
	fmt.Printf("    String 'true' -> Bool: %t\n", boolVal1)

	boolVal2 := eorm.Convert.ToBool("1")
	fmt.Printf("    String '1' -> Bool: %t\n", boolVal2)

	boolVal3 := eorm.Convert.ToBool(0)
	fmt.Printf("    Int 0 -> Bool: %t\n", boolVal3)

	boolVal4 := eorm.Convert.ToBool(1)
	fmt.Printf("    Int 1 -> Bool: %t\n", boolVal4)

	// 字符串转换
	fmt.Println("  字符串转换:")
	strVal1 := eorm.Convert.ToString(123)
	fmt.Printf("    Int 123 -> String: '%s'\n", strVal1)

	strVal2 := eorm.Convert.ToString(3.14)
	fmt.Printf("    Float 3.14 -> String: '%s'\n", strVal2)

	strVal3 := eorm.Convert.ToString(true)
	fmt.Printf("    Bool true -> String: '%s'\n", strVal3)
}

// demonstrateErrorHandlingConversions 带错误处理的类型转换
func demonstrateErrorHandlingConversions() {
	fmt.Println("场景：需要错误处理的类型转换")

	// 带错误处理的整数转换
	fmt.Println("  带错误处理的整数转换:")
	intVal, err := eorm.Convert.ToIntWithError("456")
	if err != nil {
		fmt.Printf("    转换失败: %v\n", err)
	} else {
		fmt.Printf("    String '456' -> Int: %d\n", intVal)
	}

	intVal, err = eorm.Convert.ToIntWithError("invalid")
	if err != nil {
		fmt.Printf("    String 'invalid' 转换失败: %v\n", err)
	} else {
		fmt.Printf("    String 'invalid' -> Int: %d\n", intVal)
	}

	// 带默认值的转换
	fmt.Println("  带默认值的转换:")
	intVal = eorm.Convert.ToInt("789", 0)
	fmt.Printf("    String '789' -> Int: %d\n", intVal)

	intVal = eorm.Convert.ToInt("invalid", 999)
	fmt.Printf("    String 'invalid' -> Int (默认值): %d\n", intVal)

	intVal = eorm.Convert.ToInt("invalid") // 没有默认值，返回零值
	fmt.Printf("    String 'invalid' -> Int (零值): %d\n", intVal)

	// 溢出检查
	fmt.Println("  溢出检查:")
	int8Val, err := eorm.Convert.ToInt8WithError("300") // 超出 int8 范围
	if err != nil {
		fmt.Printf("    String '300' -> Int8 失败: %v\n", err)
	} else {
		fmt.Printf("    String '300' -> Int8: %d\n", int8Val)
	}

	uintVal, err := eorm.Convert.ToUintWithError("-100") // 负数转无符号
	if err != nil {
		fmt.Printf("    String '-100' -> Uint 失败: %v\n", err)
	} else {
		fmt.Printf("    String '-100' -> Uint: %d\n", uintVal)
	}
}

// demonstrateTimeConversions 时间类型转换
func demonstrateTimeConversions() {
	fmt.Println("场景：时间相关的类型转换")

	// Duration 转换
	fmt.Println("  Duration 转换:")
	duration1 := eorm.Convert.ToDuration("5s")
	fmt.Printf("    String '5s' -> Duration: %v\n", duration1)

	duration2 := eorm.Convert.ToDuration("1m30s")
	fmt.Printf("    String '1m30s' -> Duration: %v\n", duration2)

	duration3 := eorm.Convert.ToDuration(1000000000) // 纳秒
	fmt.Printf("    Int 1000000000 -> Duration: %v\n", duration3)

	duration4 := eorm.Convert.ToDuration("invalid", 0)
	fmt.Printf("    String 'invalid' -> Duration (默认值): %v\n", duration4)

	// Time 转换
	fmt.Println("  Time 转换:")
	time1, err := eorm.Convert.ToTimeWithError("2024-01-15 14:30:00")
	if err != nil {
		fmt.Printf("    转换失败: %v\n", err)
	} else {
		fmt.Printf("    String '2024-01-15 14:30:00' -> Time: %s\n", time1.Format("2006-01-02 15:04:05"))
	}

	time2, err := eorm.Convert.ToTimeWithError("2024/01/15 14:30:00")
	if err != nil {
		fmt.Printf("    转换失败: %v\n", err)
	} else {
		fmt.Printf("    String '2024/01/15 14:30:00' -> Time: %s\n", time2.Format("2006-01-02 15:04:05"))
	}

	time3, err := eorm.Convert.ToTimeWithError("2024-01-15")
	if err != nil {
		fmt.Printf("    转换失败: %v\n", err)
	} else {
		fmt.Printf("    String '2024-01-15' -> Time: %s\n", time3.Format("2006-01-02"))
	}

	time4, err := eorm.Convert.ToTimeWithError(1705315800) // Unix 时间戳
	if err != nil {
		fmt.Printf("    转换失败: %v\n", err)
	} else {
		fmt.Printf("    Int 1705315800 -> Time: %s\n", time4.Format("2006-01-02 15:04:05"))
	}

	// 带默认值的 Time 转换
	now := time.Now()
	time5 := eorm.Convert.ToTime("invalid", now)
	fmt.Printf("    String 'invalid' -> Time (默认值): %s\n", time5.Format("2006-01-02 15:04:05"))

	// 空字符串处理
	time6, err := eorm.Convert.ToTimeWithError("")
	if err != nil {
		fmt.Printf("    空字符串转换失败: %v\n", err)
	} else {
		fmt.Printf("    空字符串 -> Time: %s\n", time6.Format("2006-01-02 15:04:05"))
	}
}

// demonstratePointerConversions 指针转换
func demonstratePointerConversions() {
	fmt.Println("场景：指针类型与值类型之间的转换")

	// 值转指针
	fmt.Println("  值转指针:")
	intPtr := eorm.Convert.ToIntPtr(123)
	fmt.Printf("    Int 123 -> *Int: %d\n", *intPtr)

	strPtr := eorm.Convert.ToStringPtr("hello")
	fmt.Printf("    String 'hello' -> *String: %s\n", *strPtr)

	boolPtr := eorm.Convert.ToBoolPtr(true)
	fmt.Printf("    Bool true -> *Bool: %t\n", *boolPtr)

	floatPtr := eorm.Convert.ToFloat64Ptr(3.14)
	fmt.Printf("    Float64 3.14 -> *Float64: %.2f\n", *floatPtr)

	// 指针转值
	fmt.Println("  指针转值:")
	intVal := eorm.Convert.IntPtrValue(intPtr)
	fmt.Printf("    *Int -> Int: %d\n", intVal)

	strVal := eorm.Convert.StringPtrValue(strPtr)
	fmt.Printf("    *String -> String: %s\n", strVal)

	boolVal := eorm.Convert.BoolPtrValue(boolPtr)
	fmt.Printf("    *Bool -> Bool: %t\n", boolVal)

	floatVal := eorm.Convert.Float64PtrValue(floatPtr)
	fmt.Printf("    *Float64 -> Float64: %.2f\n", floatVal)

	// nil 指针处理
	fmt.Println("  nil 指针处理:")
	var nilIntPtr *int = nil
	nilIntVal := eorm.Convert.IntPtrValue(nilIntPtr)
	fmt.Printf("    nil *Int -> Int (零值): %d\n", nilIntVal)

	nilIntValWithDefault := eorm.Convert.IntPtrValue(nilIntPtr, 999)
	fmt.Printf("    nil *Int -> Int (默认值): %d\n", nilIntValWithDefault)

	var nilStrPtr *string = nil
	nilStrVal := eorm.Convert.StringPtrValue(nilStrPtr, "default")
	fmt.Printf("    nil *String -> String (默认值): %s\n", nilStrVal)
}

// demonstrateDataConversions 数据转换函数
func demonstrateDataConversions() {
	fmt.Println("场景：Record 与结构体之间的转换")

	// 模拟从数据库查询的 Record
	record := eorm.NewRecord()
	record.Set("id", 1).
		Set("name", "张三").
		Set("email", "zhangsan@example.com").
		Set("age", 25).
		Set("active", true).
		Set("created_at", time.Now())

	// 定义结构体
	type User struct {
		ID        int       `json:"id"`
		Name      string    `json:"name"`
		Email     string    `json:"email"`
		Age       int       `json:"age"`
		Active    bool      `json:"active"`
		CreatedAt time.Time `json:"created_at"`
	}

	// Record 转结构体
	fmt.Println("  Record 转结构体:")
	var user User
	err := record.ToStruct(&user)
	if err != nil {
		fmt.Printf("    转换失败: %v\n", err)
	} else {
		fmt.Printf("    转换成功: %+v\n", user)
		fmt.Printf("    姓名: %s, 年龄: %d, 活跃: %t\n", user.Name, user.Age, user.Active)
	}

	// 使用全局函数转换
	fmt.Println("  使用全局函数转换:")
	var user2 User
	err = eorm.ToStruct(record, &user2)
	if err != nil {
		fmt.Printf("    转换失败: %v\n", err)
	} else {
		fmt.Printf("    转换成功: %+v\n", user2)
	}

	// 结构体转 Record
	fmt.Println("  结构体转 Record:")
	newRecord := eorm.ToRecord(user2)
	fmt.Printf("    转换成功: %s\n", newRecord.ToJson())

	// Record 转 JSON
	fmt.Println("  Record 转 JSON:")
	jsonStr := record.ToJson()
	fmt.Printf("    JSON: %s\n", jsonStr)

	// JSON 转 Record
	fmt.Println("  JSON 转 Record:")
	newRecordFromJson := eorm.NewRecord()
	err = newRecordFromJson.FromJson(jsonStr)
	if err != nil {
		fmt.Printf("    转换失败: %v\n", err)
	} else {
		fmt.Printf("    转换成功: 姓名=%s, 年龄=%d\n",
			newRecordFromJson.GetString("name"), newRecordFromJson.GetInt("age"))
	}
}

// demonstrateRealWorldUsage 实际应用场景
func demonstrateRealWorldUsage() {
	fmt.Println("场景：实际应用中的类型转换")

	// 模拟处理 HTTP 请求参数
	fmt.Println("  处理 HTTP 请求参数:")

	// 模拟从 URL 查询参数获取的字符串值
	pageStr := "2"
	sizeStr := "10"
	ageStr := "25"
	activeStr := "true"

	// 安全地转换参数
	page := eorm.Convert.ToInt(pageStr, 1)          // 默认第1页
	size := eorm.Convert.ToInt(sizeStr, 20)         // 默认每页20条
	age := eorm.Convert.ToInt(ageStr, 0)            // 默认年龄0
	active := eorm.Convert.ToBool(activeStr, false) // 默认不活跃

	fmt.Printf("    分页参数: 页码=%d, 每页=%d\n", page, size)
	fmt.Printf("    用户条件: 年龄=%d, 活跃=%t\n", age, active)

	// 模拟处理数据库查询结果
	fmt.Println("  处理数据库查询结果:")

	// 模拟数据库返回的 map[string]interface{}
	dbResult := map[string]interface{}{
		"user_id":  "123",                 // 数据库中可能是字符串
		"price":    "99.99",               // 价格可能是字符串
		"quantity": "5",                   // 数量可能是字符串
		"in_stock": "1",                   // 布尔值可能存储为字符串
		"created":  "2024-01-15 14:30:00", // 时间字符串
	}

	userID := eorm.Convert.ToInt64(dbResult["user_id"])
	price := eorm.Convert.ToFloat64(dbResult["price"])
	quantity := eorm.Convert.ToInt(dbResult["quantity"])
	inStock := eorm.Convert.ToBool(dbResult["in_stock"])
	created, _ := eorm.Convert.ToTimeWithError(dbResult["created"].(string))

	fmt.Printf("    用户ID: %d\n", userID)
	fmt.Printf("    价格: %.2f\n", price)
	fmt.Printf("    数量: %d\n", quantity)
	fmt.Printf("    有库存: %t\n", inStock)
	fmt.Printf("    创建时间: %s\n", created.Format("2006-01-02 15:04:05"))

	// 模拟配置文件处理
	fmt.Println("  处理配置文件:")

	// 模拟配置文件中的值
	config := map[string]interface{}{
		"max_connections": "100",
		"timeout":         "30s",
		"enable_debug":    "true",
		"retry_count":     "3",
	}

	maxConn := eorm.Convert.ToInt(config["max_connections"], 10)
	timeout := eorm.Convert.ToDuration(config["timeout"], 5*time.Second)
	enableDebug := eorm.Convert.ToBool(config["enable_debug"], false)
	retryCount := eorm.Convert.ToInt(config["retry_count"], 1)

	fmt.Printf("    最大连接数: %d\n", maxConn)
	fmt.Printf("    超时时间: %v\n", timeout)
	fmt.Printf("    调试模式: %t\n", enableDebug)
	fmt.Printf("    重试次数: %d\n", retryCount)
}

// init 函数：设置日志
func init() {
	// 开启调试模式
	eorm.SetDebugMode(true)

	// 初始化文件日志
	eorm.InitLoggerWithFile("debug", "convert_utils.log")
}
