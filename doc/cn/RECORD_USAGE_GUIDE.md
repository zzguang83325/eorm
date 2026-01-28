# Record 对象使用指南

## 概述

Record 是 EORM 的核心数据结构，提供了灵活、高效的数据操作方式。它类似于 JFinal 的 ActiveRecord，但更加现代化和类型安全。

## Record 对象的便利性

### 1. 灵活的数据存储

Record 可以存储任意类型的数据，包括基本类型、嵌套对象、数组等。

```go
record := eorm.NewRecord()
record.Set("name", "张三")
record.Set("age", 25)
record.Set("active", true)
record.Set("score", 95.5)
record.Set("tags", []string{"developer", "golang"})
record.Set("profile", map[string]interface{}{
    "city": "北京",
    "street": "朝阳路",
})
```

### 2. 链式调用

Record 支持链式调用，让代码更加简洁。

```go
record := eorm.NewRecord().
    Set("name", "张三").
    Set("age", 25).
    Set("email", "zhangsan@example.com").
    Set("city", "北京")
```

### 3. 类型安全访问

提供类型安全的访问方法，避免类型断言的繁琐。

```go
name := record.GetString("name")      // 直接返回 string
age := record.GetInt("age")          // 直接返回 int
score := record.GetFloat("score")    // 直接返回 float64
active := record.GetBool("active")    // 直接返回 bool
```

### 4. JSON 序列化/反序列化

内置 JSON 支持，方便与外部系统交互。

```go
// 从 JSON 创建 Record
record := eorm.NewRecord().FromJson(`{"name": "张三", "age": 25}`)

// 转换为 JSON
jsonStr := record.ToJson()
```

### 5. 路径访问

支持点分路径访问嵌套数据。

```go
// 访问嵌套数据
city := record.GetStringByPath("profile.city")
email := record.GetStringByPath("contact.email")
```

### 6. 数据转换

支持从多种数据源创建 Record。

```go
// 从 map
record.FromMap(mapData)

// 从结构体
record.FromStruct(userStruct)

// 从 JSON
record.FromJson(jsonStr)

// 从另一个 Record
record.FromRecord(otherRecord)
```

## 适用环境

### 1. Web 应用开发

Record 非常适合 Web 应用开发，特别是：

- **API 响应处理**：灵活地构建和修改 API 响应
- **表单数据处理**：方便地处理表单提交的数据
- **配置管理**：存储和管理应用配置
- **缓存数据**：存储和传递缓存数据

**示例：处理 API 响应**

```go
// 从 API 响应创建 Record
response := eorm.NewRecord().FromJson(apiResponse)

// 访问嵌套数据
userName := response.GetStringByPath("data.user.name")
userEmail := response.GetStringByPath("data.user.email")

// 修改数据
response.Set("processed", true)
response.Set("timestamp", time.Now())

// 保存到数据库
eorm.SaveRecord("api_logs", response)
```

### 2. 数据处理和转换

Record 适合数据处理场景：

- **数据清洗**：方便地检查和修改数据
- **数据转换**：在不同格式之间转换
- **数据验证**：验证数据完整性
- **数据聚合**：合并多个数据源

**示例：数据清洗和转换**

```go
// 从数据库查询原始数据
rawData, _ := eorm.Query("SELECT * FROM raw_data WHERE id = ?", 1)

// 创建 Record 进行处理
record := eorm.NewRecord().FromMap(rawData)

// 数据清洗：确保必填字段
if !record.Has("email") {
    record.Set("email", "default@example.com")
}

// 数据转换：格式化日期
if rawDate, ok := record.Get("created_at"); ok {
    if t, err := time.Parse("2006-01-02", rawDate.(string)); err == nil {
        record.Set("formatted_date", t.Format("2006年01月02日"))
    }
}

// 数据验证：检查年龄范围
age := record.GetInt("age")
if age < 0 || age > 150 {
    record.Set("age", 0)  // 设置默认值
}

// 保存处理后的数据
eorm.SaveRecord("processed_data", record)
```

### 3. 配置管理

Record 非常适合配置管理：

- **多层级配置**：支持嵌套配置结构
- **动态配置**：运行时修改配置
- **配置继承**：从基础配置扩展
- **环境配置**：管理不同环境的配置

**示例：多环境配置管理**

```go
// 基础配置
baseConfig := eorm.NewRecord().FromJson(`{
    "database": {
        "host": "localhost",
        "port": 3306
    },
    "cache": {
        "enabled": true,
        "ttl": 3600
    }
}`)

// 开发环境配置
devConfig := baseConfig.DeepClone()
devConfig.SetByPath("database.host", "dev.example.com")
devConfig.SetByPath("database.name", "dev_db")

// 生产环境配置
prodConfig := baseConfig.DeepClone()
prodConfig.SetByPath("database.host", "prod.example.com")
prodConfig.SetByPath("database.name", "prod_db")

// 使用配置
dbHost := devConfig.GetStringByPath("database.host")
cacheEnabled := devConfig.GetBoolByPath("cache.enabled")
```

### 4. 测试和 Mock

Record 适合测试场景：

- **Mock 数据**：快速创建测试数据
- **断言验证**：方便地验证结果
- **数据隔离**：测试之间互不影响
- **灵活修改**：动态调整测试数据

**示例：单元测试**

```go
func TestUserService(t *testing.T) {
    // 创建 Mock 数据
    mockUser := eorm.NewRecord().
        Set("id", 1).
        Set("name", "张三").
        Set("email", "zhangsan@example.com").
        Set("age", 25)

    // 测试服务
    result := UserService.GetUser(1)

    // 断言验证
    assert.Equal(t, mockUser.GetString("name"), result.GetString("name"))
    assert.Equal(t, mockUser.GetString("email"), result.GetString("email"))
    assert.Equal(t, mockUser.GetInt("age"), result.GetInt("age"))
}
```

### 5. 日志和审计

Record 适合日志和审计场景：

- **结构化日志**：存储结构化的日志数据
- **审计追踪**：记录操作历史
- **错误报告**：收集错误信息
- **性能监控**：记录性能指标

**示例：审计日志**

```go
// 创建审计记录
auditLog := eorm.NewRecord().
    Set("user_id", userID).
    Set("action", "update_profile").
    Set("resource", "users").
    Set("timestamp", time.Now()).
    Set("ip_address", clientIP).
    Set("user_agent", userAgent).
    Set("changes", map[string]interface{}{
        "old": oldData,
        "new": newData,
    })

// 保存审计日志
eorm.InsertRecord("audit_logs", auditLog)
```

## API 详细用法

### 1. NewRecord

创建新的 Record 实例。

**简单示例：**

```go
record := eorm.NewRecord()
```

**复杂示例：链式调用创建并初始化**

```go
record := eorm.NewRecord().
    Set("id", 1).
    Set("name", "张三").
    Set("age", 25).
    Set("email", "zhangsan@example.com").
    Set("profile", map[string]interface{}{
        "city": "北京",
        "street": "朝阳路",
    }).
    Set("tags", []string{"developer", "golang"})
```

### 2. Set

设置字段值，支持链式调用。

**简单示例：**

```go
record.Set("name", "张三")
record.Set("age", 25)
```

**复杂示例：动态设置多个字段**

```go
// 从 map 动态设置字段
data := map[string]interface{}{
    "field1": "value1",
    "field2": 123,
    "field3": true,
}

for key, value := range data {
    record.Set(key, value)
}
```

**复杂示例：条件设置**

```go
// 根据条件设置字段
if user.IsAdmin {
    record.Set("role", "admin")
} else {
    record.Set("role", "user")
}

// 设置默认值
if !record.Has("email") {
    record.Set("email", "default@example.com")
}
```

### 3. Get

获取字段值。

**简单示例：**

```go
name := record.Get("name")
age := record.Get("age")
```

**复杂示例：类型断言和安全访问**

```go
// 获取嵌套对象
if profile, ok := record.Get("profile").(map[string]interface{}); ok {
    city := profile["city"]
    fmt.Println("城市:", city)
}

// 获取数组
if tags, ok := record.Get("tags").([]interface{}); ok {
    for _, tag := range tags {
        fmt.Println("标签:", tag)
    }
}
```

### 4. GetString / GetInt / GetFloat / GetBool

类型安全的获取方法。

**简单示例：**

```go
name := record.GetString("name")
age := record.GetInt("age")
score := record.GetFloat("score")
active := record.GetBool("active")
```

**复杂示例：带默认值的获取**

```go
// 获取字符串，带默认值
name := record.GetString("name")
if name == "" {
    name = "匿名用户"
}

// 获取数字，带范围检查
age := record.GetInt("age")
if age < 0 {
    age = 0
} else if age > 150 {
    age = 150
}

// 获取布尔值，带逻辑处理
isActive := record.GetBool("active")
if !isActive {
    fmt.Println("用户未激活")
}
```

### 5. Has

检查字段是否存在。

**简单示例：**

```go
if record.Has("email") {
    fmt.Println("邮箱:", record.GetString("email"))
}
```

**复杂示例：多字段检查**

```go
// 检查多个必填字段
requiredFields := []string{"name", "email", "age"}
missingFields := []string{}

for _, field := range requiredFields {
    if !record.Has(field) {
        missingFields = append(missingFields, field)
    }
}

if len(missingFields) > 0 {
    fmt.Println("缺少必填字段:", missingFields)
}
```

### 6. Keys / Columns

获取所有字段名。

**简单示例：**

```go
keys := record.Keys()
fmt.Println("字段列表:", keys)
```

**复杂示例：字段遍历和处理**

```go
// 遍历所有字段
for _, key := range record.Keys() {
    value := record.Get(key)
    fmt.Printf("%s: %v\n", key, value)
}

// 过滤特定字段
keys := record.Keys()
filteredKeys := []string{}
for _, key := range keys {
    if strings.HasPrefix(key, "user_") {
        filteredKeys = append(filteredKeys, key)
    }
}
```

### 7. Remove

删除字段。

**简单示例：**

```go
record.Remove("password")
record.Remove("token")
```

**复杂示例：批量删除敏感字段**

```go
// 定义敏感字段列表
sensitiveFields := []string{
    "password",
    "token",
    "secret",
    "api_key",
}

// 批量删除
for _, field := range sensitiveFields {
    record.Remove(field)
}

// 创建安全副本
safeRecord := record.DeepClone()
for _, field := range sensitiveFields {
    safeRecord.Remove(field)
}
```

### 8. FromJson

从 JSON 字符串创建 Record。

**简单示例：**

```go
jsonStr := `{"name": "张三", "age": 25}`
record := eorm.NewRecord().FromJson(jsonStr)
```

**复杂示例：处理嵌套 JSON**

```go
jsonStr := `{
    "user": {
        "name": "张三",
        "profile": {
            "age": 25,
            "city": "北京"
        }
    },
    "settings": {
        "theme": "dark",
        "language": "zh-CN"
    }
}`)

record := eorm.NewRecord().FromJson(jsonStr)

// 访问嵌套数据
userName := record.GetStringByPath("user.name")
userCity := record.GetStringByPath("user.profile.city")
theme := record.GetStringByPath("settings.theme")
```

### 9. ToJson

将 Record 转换为 JSON 字符串。

**简单示例：**

```go
jsonStr := record.ToJson()
fmt.Println(jsonStr)
```

**复杂示例：格式化输出**

```go
// 格式化 JSON 输出
jsonStr := record.ToJson()

// 缩进格式化
var buf bytes.Buffer
json.Indent(&buf, []byte(jsonStr), "", "  ")
formattedJSON := buf.String()

fmt.Println("格式化的 JSON:")
fmt.Println(formattedJSON)
```

### 10. FromMap

从 map 创建 Record。

**简单示例：**

```go
data := map[string]interface{}{
    "name": "张三",
    "age": 25,
}
record := eorm.NewRecord().FromMap(data)
```

**复杂示例：合并多个 map**

```go
// 基础数据
baseData := map[string]interface{}{
    "id": 1,
    "name": "张三",
}

// 扩展数据
extraData := map[string]interface{}{
    "email": "zhangsan@example.com",
    "age": 25,
}

// 创建 Record 并合并
record := eorm.NewRecord().FromMap(baseData)
for key, value := range extraData {
    record.Set(key, value)
}
```

### 11. FromStruct

从结构体创建 Record。

**简单示例：**

```go
type User struct {
    Name  string `json:"name"`
    Age   int    `json:"age"`
    Email string `json:"email"`
}

user := User{Name: "张三", Age: 25, Email: "zhangsan@example.com"}
record := eorm.NewRecord().FromStruct(user)
```

**复杂示例：嵌套结构体**

```go
type Address struct {
    City   string `json:"city"`
    Street string `json:"street"`
}

type Profile struct {
    Age    int     `json:"age"`
    Address Address `json:"address"`
}

type User struct {
    Name    string  `json:"name"`
    Profile Profile `json:"profile"`
}

user := User{
    Name: "张三",
    Profile: Profile{
        Age: 25,
        Address: Address{
            City:   "北京",
            Street: "朝阳路",
        },
    },
}

record := eorm.NewRecord().FromStruct(user)

// 访问嵌套数据
city := record.GetStringByPath("profile.address.city")
```

### 12. FromRecord

从另一个 Record 填充当前 Record（浅拷贝）。

**简单示例：**

```go
sourceRecord := eorm.NewRecord().
    Set("name", "张三").
    Set("age", 25)

record := eorm.NewRecord().FromRecord(sourceRecord)
```

**复杂示例：链式调用和扩展**

```go
// 基础配置
baseConfig := eorm.NewRecord().
    Set("timeout", 30).
    Set("retry", 3).
    Set("debug", false)

// 创建特定环境配置
devConfig := eorm.NewRecord().
    FromRecord(baseConfig).
    Set("host", "dev.example.com").
    Set("port", 8080).
    Set("log_level", "debug")

// 创建生产环境配置
prodConfig := eorm.NewRecord().
    FromRecord(baseConfig).
    Set("host", "prod.example.com").
    Set("port", 443).
    Set("log_level", "info")
```

### 13. Clone

创建 Record 的浅拷贝。

**简单示例：**

```go
original := eorm.NewRecord().
    Set("name", "张三").
    Set("age", 25)

cloned := original.Clone()
```

**复杂示例：批量克隆**

```go
// 原始数据
originalData := []*eorm.Record{
    eorm.NewRecord().Set("id", 1).Set("name", "张三"),
    eorm.NewRecord().Set("id", 2).Set("name", "李四"),
    eorm.NewRecord().Set("id", 3).Set("name", "王五"),
}

// 批量克隆
clonedData := make([]*eorm.Record, len(originalData))
for i, record := range originalData {
    clonedData[i] = record.Clone()
}
```

### 14. DeepClone

创建 Record 的深拷贝。

**简单示例：**

```go
original := eorm.NewRecord().
    Set("name", "张三").
    Set("profile", map[string]interface{}{
        "city": "北京",
        "age": 25,
    })

cloned := original.DeepClone()
```

**复杂示例：独立修改**

```go
// 创建包含嵌套对象的 Record
original := eorm.NewRecord().
    Set("name", "张三").
    Set("profile", map[string]interface{}{
        "city": "北京",
        "age": 25,
    })

// 深拷贝
cloned := original.DeepClone()

// 修改克隆的嵌套对象
if profile, ok := cloned.Get("profile").(map[string]interface{}); ok {
    profile["city"] = "上海"
    profile["age"] = 30
}

// 原始记录不受影响
fmt.Println("原始:", original.ToJson())
fmt.Println("克隆:", cloned.ToJson())
```

### 15. FromRecordDeep

从另一个 Record 深拷贝填充当前 Record。

**简单示例：**

```go
sourceRecord := eorm.NewRecord().
    Set("name", "张三").
    Set("age", 25)

record := eorm.NewRecord().FromRecordDeep(sourceRecord)
```

**复杂示例：链式调用和独立修改**

```go
// 模板配置
templateConfig := eorm.NewRecord().
    Set("timeout", 30).
    Set("retry", 3).
    Set("debug", false).
    Set("profile", map[string]interface{}{
        "theme": "light",
        "language": "zh-CN",
    })

// 创建用户特定配置
userConfig := eorm.NewRecord().
    FromRecordDeep(templateConfig).
    Set("user_id", 12345).
    Set("username", "zhangsan").
    SetByPath("profile.theme", "dark")

// 修改用户配置不影响模板
if profile, ok := userConfig.Get("profile").(map[string]interface{}); ok {
    profile["language"] = "en-US"
}

// 模板配置保持不变
fmt.Println("模板:", templateConfig.ToJson())
fmt.Println("用户:", userConfig.ToJson())
```

### 16. GetRecord

获取嵌套的 Record。

**简单示例：**

```go
record.Set("profile", map[string]interface{}{
    "name": "张三",
    "age": 25,
})

profile, err := record.GetRecord("profile")
if err == nil {
    fmt.Println("姓名:", profile.GetString("name"))
    fmt.Println("年龄:", profile.GetInt("age"))
}
```

**复杂示例：多层嵌套访问**

```go
// 创建多层嵌套结构
record.Set("user", map[string]interface{}{
    "profile": map[string]interface{}{
        "basic": map[string]interface{}{
            "name": "张三",
            "age": 25,
        },
        "contact": map[string]interface{}{
            "email": "zhangsan@example.com",
            "phone": "13800138000",
        },
    },
})

// 访问多层嵌套
user, _ := record.GetRecord("user")
profile, _ := user.GetRecord("profile")
basic, _ := profile.GetRecord("basic")
contact, _ := profile.GetRecord("contact")

fmt.Println("姓名:", basic.GetString("name"))
fmt.Println("邮箱:", contact.GetString("email"))
```

### 17. GetRecordByPath

通过点分路径获取嵌套 Record。

**简单示例：**

```go
record.Set("data", map[string]interface{}{
    "user": map[string]interface{}{
        "name": "张三",
        "age": 25,
    },
})

user, err := record.GetRecordByPath("data.user")
if err == nil {
    fmt.Println("姓名:", user.GetString("name"))
}
```

**复杂示例：动态路径访问**

```go
// 定义访问路径
paths := []string{
    "data.user.name",
    "data.user.profile.city",
    "data.settings.theme",
}

// 动态访问多个路径
for _, path := range paths {
    if value, err := record.GetStringByPath(path); err == nil {
        fmt.Printf("%s: %s\n", path, value)
    }
}
```

### 18. GetStringByPath

通过点分路径获取嵌套的字符串值。

**简单示例：**

```go
record.Set("user", map[string]interface{}{
    "profile": map[string]interface{}{
        "city": "北京",
    },
})

city, err := record.GetStringByPath("user.profile.city")
if err == nil {
    fmt.Println("城市:", city)
}
```

**复杂示例：配置路径访问**

```go
// 配置数据
config := eorm.NewRecord().FromJson(`{
    "database": {
        "host": "localhost",
        "port": 3306,
        "name": "mydb"
    },
    "cache": {
        "enabled": true,
        "ttl": 3600
    }
}`)

// 访问配置
dbHost, _ := config.GetStringByPath("database.host")
dbPort, _ := config.GetStringByPath("database.port")
dbName, _ := config.GetStringByPath("database.name")
cacheEnabled, _ := config.GetBoolByPath("cache.enabled")

fmt.Printf("数据库: %s:%d/%s\n", dbHost, dbPort, dbName)
fmt.Printf("缓存: %v\n", cacheEnabled)
```

## 最佳实践

### 1. 使用链式调用

链式调用可以让代码更加简洁和可读。

```go
// 推荐：使用链式调用
record := eorm.NewRecord().
    Set("name", "张三").
    Set("age", 25).
    Set("email", "zhangsan@example.com")

// 不推荐：多次调用
record := eorm.NewRecord()
record.Set("name", "张三")
record.Set("age", 25)
record.Set("email", "zhangsan@example.com")
```

### 2. 选择合适的拷贝方式

根据需求选择浅拷贝或深拷贝。

```go
// 只读场景：使用浅拷贝（性能更好）
readOnlyCopy := original.Clone()

// 需要修改：使用深拷贝（完全独立）
modifiableCopy := original.DeepClone()
```

### 3. 使用类型安全方法

优先使用类型安全的获取方法，避免类型断言。

```go
// 推荐：使用类型安全方法
name := record.GetString("name")
age := record.GetInt("age")

// 不推荐：手动类型断言
name, _ := record.Get("name").(string)
age, _ := record.Get("age").(int)
```

### 4. 错误处理

正确处理可能的错误。

```go
// 推荐：检查错误
if profile, err := record.GetRecord("profile"); err != nil {
    fmt.Printf("获取 profile 失败: %v\n", err)
    return
}

// 使用 profile
fmt.Println(profile.GetString("name"))
```

### 5. 性能优化

在性能敏感的场景下，选择合适的方法。

```go
// 批量操作：使用链式调用
record := eorm.NewRecord().
    Set("field1", "value1").
    Set("field2", "value2").
    Set("field3", "value3")

// 只读场景：使用浅拷贝
readOnlyCopy := original.Clone()

// 需要独立：使用深拷贝
independentCopy := original.DeepClone()
```

## 总结

Record 对象提供了灵活、高效的数据操作方式，适用于各种场景：

- ✅ **Web 应用开发**：API 响应、表单处理、配置管理
- ✅ **数据处理**：数据清洗、转换、验证、聚合
- ✅ **配置管理**：多层级、动态、环境配置
- ✅ **测试和 Mock**：快速创建测试数据、断言验证
- ✅ **日志和审计**：结构化日志、审计追踪、错误报告

通过合理使用 Record 对象的 API，可以大大提高开发效率和代码质量。
