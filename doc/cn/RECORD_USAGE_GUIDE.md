# Record 对象使用指南

## 概述

Record 是 EORM 的核心数据结构，提供了灵活、高效的数据操作方式。它类似于 JFinal 的 ActiveRecord，但更加现代化和类型安全,它是一个非常灵活的通用数据容器，适合多种场景:

比如：

1. 需要灵活数据结构的场合 - 字段不固定，动态增减

2. 复杂嵌套数据结构 - 支持多层嵌套和路径访问

3. 数据转换和映射 - 方便在不同格式间转换

4. 中间件数据传递 - 在调用链中传递额外数据

5. 配置管理 - 支持多级配置和动态覆盖

6. API 响应处理 - 处理结构不固定的 JSON 数据

7. 临时数据存储 - 在处理流程中临时存储数据

8. 测试数据构造 - 方便构建复杂的测试数据

## Record 对象的便利性

### 1. 灵活的数据存储

Record 可以存储任意类型的数据，包括基本类型、嵌套对象、数组等。

```go
record := eorm.NewRecord()
record.Set("name", "张三")
record.Set("age", 25)
record.Set("active", true)
record.Set("score", 95.5)

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
// 从 JSON 创建包含嵌套数据的 Record
record := eorm.NewRecord().FromJson(`{
    "profile": {
        "city": "北京",
        "street": "朝阳路"
    },
    "contact": {
        "email": "zhangsan@example.com",
        "phone": "13800138000"
    }
}`)

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

### 1. 数据库应用

Record 非常适合数据库应用场景，特别是：

- **动态查询结果**：灵活地处理数据库查询返回的数据
- **批量数据处理**：高效处理大量数据库记录
- **数据转换**：在不同数据格式之间转换
- **缓存和临时存储**：存储查询结果和中间数据

**示例：处理数据库查询结果**

```go

// 从数据库查询用户列表，直接返回 Record 数组
records, err := eorm.Query("SELECT id, name, email, age FROM users WHERE status = ?", 1)
if err != nil {
    log.Fatal(err)
}


```

**示例：使用 Record 进行数据库操作**

```go
package main

import (
    "github.com/zzguang83325/eorm"
)

// 插入数据
func InsertUser() error {
    user := eorm.NewRecord().
        Set("name", "张三").
        Set("email", "zhangsan@example.com").
        Set("age", 25).
        Set("created_at", time.Now())
    
    _, err := eorm.InsertRecord("users", user)
    return err
}

// 批量插入
func BatchInsertUsers(users []map[string]interface{}) error {
    records := make([]*eorm.Record, len(users))
    for i, user := range users {
        records[i] = eorm.NewRecord().FromMap(user)
    }
    _, err := eorm.BatchInsertRecord("users", records)
    return err
}

// 更新数据
func UpdateUser(id int, updates map[string]interface{}) error {
    record := eorm.NewRecord().
        Set("id", id).
        FromMap(updates).
        Set("updated_at", time.Now())
    
    _, err := eorm.UpdateRecord("users", record)
    return err
}

// 查询并转换
func QueryUsersByAge(minAge int) ([]*eorm.Record, error) {
    // 从数据库查询，直接返回 Record 数组
    records, err := eorm.Query("SELECT * FROM users WHERE age >= ?", minAge)
    if err != nil {
        return nil, err
    }
 
    return records, nil
}

// 聚合查询
func GetUserStats() (int64, error) {
    // 查询总数
    totalRecord, _ := eorm.QueryFirst("SELECT COUNT(*) as total FROM users")
    total := totalRecord.GetInt("total")
    
 
}
```

**Record 在数据库应用中的优势：**

1. **灵活的数据处理**：动态添加、删除、修改字段
2. **批量操作支持**：高效处理大量数据库记录
3. **类型安全访问**：使用 GetString、GetInt 等方法，避免类型断言
4. **数据转换便捷**：在不同数据格式之间轻松转换

### 2. API 接口和 web 开发

Record 非常适合 API 接口和 web 开发场景：

- **响应包装**：统一包装 API 响应
- **中间件数据**：在中间件之间传递数据
- **错误处理**：统一处理和返回错误

**示例：API 接口响应包装**

```go
// 原始服务响应
serviceResponse := map[string]interface{}{
    "data": []interface{}{1, 2, 3},
    "total": 3,
}

// 统一包装响应
response := eorm.NewRecord()
response.Set("code", 200)
response.Set("message", "success")
response.Set("timestamp", time.Now().Unix())
response.Set("data", serviceResponse["data"])

// 返回给客户端
c.JSON(http.StatusOK, response)
```



**示例：使用 Gin 框架处理 HTTP 请求**

```go
package main

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/zzguang83325/eorm"
)

// 统一成功响应
func SuccessResponse(c *gin.Context, data interface{}) {
    c.JSON(http.StatusOK, eorm.NewRecord().
        Set("code", 0).
        Set("message", "ok").
        Set("data", data))
}

// 统一错误响应
func ErrorResponse(c *gin.Context, code int, message string) {
    c.JSON(code, eorm.NewRecord().
        Set("code", code).
        Set("message", message))
}

// 用户注册接口 - 无需定义 struct
func (h *UserHandler) Register(c *gin.Context) {
    // 直接使用 Record 接收前端 JSON 数据
    record := eorm.NewRecord()
    if err := c.ShouldBindJSON(&record); err != nil {
        ErrorResponse(c, http.StatusBadRequest, err.Error())
        return
    }

    // 数据验证
    if !record.Has("username") || !record.Has("password") {
        ErrorResponse(c, http.StatusBadRequest, "用户名和密码不能为空")
        return
    }

    // 检查用户名是否已存在
    exists, _ := h.CheckUserExists(record.GetString("username"))
    if exists {
        ErrorResponse(c, http.StatusConflict, "用户名已存在")
        return
    }

    // 保存到数据库
    record.Set("created_at", time.Now())
    if err := h.SaveUser(record); err != nil {
        ErrorResponse(c, http.StatusInternalServerError, "注册失败")
        return
    }

    // 返回用户信息（统一包装）
    SuccessResponse(c, record)
}

// 获取用户列表
func (h *UserHandler) ListUsers(c *gin.Context) {
    // 从数据库查询用户列表
    users, err := h.GetAllUsers()
    if err != nil {
        ErrorResponse(c, http.StatusInternalServerError, err.Error())
        return
    }

    // 直接返回 Record 数组（统一包装）
    SuccessResponse(c, users)
}

// 获取单个用户
func (h *UserHandler) GetUser(c *gin.Context) {
    userID := c.Param("id")

    // 查询用户
    user, err := h.GetUserByID(userID)
    if err != nil {
        ErrorResponse(c, http.StatusNotFound, "用户不存在")
        return
    }

    // 返回用户信息（统一包装）
    SuccessResponse(c, user)
}

// 更新用户信息
func (h *UserHandler) UpdateUser(c *gin.Context) {
    userID := c.Param("id")

    // 获取现有用户
    existingUser, err := h.GetUserByID(userID)
    if err != nil {
        ErrorResponse(c, http.StatusNotFound, "用户不存在")
        return
    }

    // 接收更新数据
    updateData := eorm.NewRecord()
    if err := c.ShouldBindJSON(&updateData); err != nil {
        ErrorResponse(c, http.StatusBadRequest, err.Error())
        return
    }

    // 合并更新数据
    existingUser.FromRecord(updateData)
    existingUser.Set("updated_at", time.Now())

    // 保存更新
    if err := h.UpdateUserRecord(existingUser); err != nil {
        ErrorResponse(c, http.StatusInternalServerError, "更新失败")
        return
    }

    // 返回更新后的用户信息（统一包装）
    SuccessResponse(c, existingUser)
}
```

**Record 在 Web 开发中的优势：**

1. **无需定义 struct**：直接接收前端 JSON 数据，减少代码量
2. **灵活的数据处理**：可以动态添加、删除、修改字段
3. **自动 JSON 序列化**：Record 自动转换为 JSON，前端可以直接使用
4. **支持数组输出**：可以直接返回 Record 数组，自动转换为 JSON 数组
5. **类型安全访问**：使用 GetString、GetInt 等方法，避免类型断言
6. **统一响应格式**：轻松实现统一的 API 响应格式

```go
// 前端发送的 JSON 请求
{
    "username": "zhangsan",
    "password": "123456",
    "email": "zhangsan@example.com",
    "profile": {
        "age": 25,
        "city": "北京"
    }
}

// 后端接收后，可以直接访问
username := record.GetString("username")
email := record.GetString("email")
age := record.GetIntByPath("profile.age")
city := record.GetStringByPath("profile.city")

// 返回给前端的 JSON 响应（统一包装格式）
{
    "code": 0,
    "message": "ok",
    "data": {
        "username": "zhangsan",
        "password": "123456",
        "email": "zhangsan@example.com",
        "profile": {
            "age": 25,
            "city": "北京"
        },
        "created_at": "2024-01-01T00:00:00Z"
    }
}

// 错误响应格式
{
    "code": 400,
    "message": "用户名和密码不能为空"
}
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
    // 创建 Mock 数据数组
    mockUsers := []*eorm.Record{
        eorm.NewRecord().
            Set("id", 1).
            Set("name", "张三").
            Set("email", "zhangsan@example.com").
            Set("age", 25),
        eorm.NewRecord().
            Set("id", 2).
            Set("name", "李四").
            Set("email", "lisi@example.com").
            Set("age", 30),
        eorm.NewRecord().
            Set("id", 3).
            Set("name", "王五").
            Set("email", "wangwu@example.com").
            Set("age", 28),
    }

    // 测试批量获取用户
    results := UserService.GetUsers([]int{1, 2, 3})

    // 断言验证
    assert.Equal(t, len(mockUsers), len(results))
    
    for i, mockUser := range mockUsers {
        assert.Equal(t, mockUser.GetInt("id"), results[i].GetInt("id"))
        assert.Equal(t, mockUser.GetString("name"), results[i].GetString("name"))
        assert.Equal(t, mockUser.GetString("email"), results[i].GetString("email"))
        assert.Equal(t, mockUser.GetInt("age"), results[i].GetInt("age"))
    }
}
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
    Set("profile", eorm.NewRecord().
        Set("city", "北京").
        Set("street", "朝阳路")).
    Set("tags", []string{"developer", "golang"})
```

### 2. Set

设置字段值，支持链式调用。

**简单示例：**

```go
record.Set("name", "张三")
record.Set("age", 25)
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
// 获取嵌套对象（推荐使用 GetRecord）
if profile, err := record.GetRecord("profile"); err == nil {
    city := profile.GetString("city")
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

```

### 7. Remove

删除字段。

**简单示例：**

```go
record.Remove("password")
record.Remove("token")
```



### 8. FromJson

从 JSON 字符串创建 Record，支持链式调用合并多个 JSON。

**简单示例：**

```go
jsonStr := `{"name": "张三", "age": 25}`
record := eorm.NewRecord().FromJson(jsonStr)
```

**复杂示例：合并多个 JSON**

```go
// 用户基本信息
userJson := `{
    "id": 1,
    "name": "张三",
    "age": 25
}`

// 用户扩展信息
profileJson := `{
    "email": "zhangsan@example.com",
    "phone": "13800138000",
    "address": "北京市朝阳区"
}`

// 用户设置
settingsJson := `{
    "theme": "dark",
    "language": "zh-CN",
    "notifications": true
}`

// 合并多个 JSON 到一个 Record
record := eorm.NewRecord().
    FromJson(userJson).
    FromJson(profileJson).
    FromJson(settingsJson)

// 访问合并后的数据
fmt.Println("姓名:", record.GetString("name"))
fmt.Println("年龄:", record.GetInt("age"))
fmt.Println("邮箱:", record.GetString("email"))
fmt.Println("电话:", record.GetString("phone"))
fmt.Println("地址:", record.GetString("address"))
fmt.Println("主题:", record.GetString("theme"))
fmt.Println("语言:", record.GetString("language"))

// 输出完整的 Record
fmt.Println("完整数据:", record.ToJson())
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

从 map 创建 Record，支持链式调用合并多个 map。

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
// 基础信息
baseInfo := map[string]interface{}{
    "id": 1,
    "name": "张三",
    "age": 25,
}

// 联系信息
contactInfo := map[string]interface{}{
    "email": "zhangsan@example.com",
    "phone": "13800138000",
    "address": "北京市朝阳区",
}

// 工作信息
workInfo := map[string]interface{}{
    "company": "ABC 公司",
    "position": "软件工程师",
    "department": "技术部",
}

// 设置信息
settingsInfo := map[string]interface{}{
    "theme": "dark",
    "language": "zh-CN",
    "notifications": true,
}

// 合并多个 map 到一个 Record
record := eorm.NewRecord().
    FromMap(baseInfo).
    FromMap(contactInfo).
    FromMap(workInfo).
    FromMap(settingsInfo)

// 访问合并后的数据
fmt.Println("ID:", record.GetInt("id"))
fmt.Println("姓名:", record.GetString("name"))
fmt.Println("年龄:", record.GetInt("age"))
fmt.Println("邮箱:", record.GetString("email"))
fmt.Println("电话:", record.GetString("phone"))
fmt.Println("地址:", record.GetString("address"))
fmt.Println("公司:", record.GetString("company"))
fmt.Println("职位:", record.GetString("position"))
fmt.Println("部门:", record.GetString("department"))
fmt.Println("主题:", record.GetString("theme"))
fmt.Println("语言:", record.GetString("language"))

// 输出完整的 Record
fmt.Println("完整数据:", record.ToJson())
```

### 11. FromStruct

从结构体创建 Record，支持链式调用合并多个结构体。

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

**复杂示例：合并多个结构体**

```go
// 基础信息结构体
type BaseInfo struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
    Age  int    `json:"age"`
}

// 联系信息结构体
type ContactInfo struct {
    Email   string `json:"email"`
    Phone   string `json:"phone"`
    Address string `json:"address"`
}

// 工作信息结构体
type WorkInfo struct {
    Company    string `json:"company"`
    Position   string `json:"position"`
    Department string `json:"department"`
}

// 设置信息结构体
type SettingsInfo struct {
    Theme         string `json:"theme"`
    Language      string `json:"language"`
    Notifications bool   `json:"notifications"`
}

// 创建多个结构体实例
base := BaseInfo{
    ID:   1,
    Name: "张三",
    Age:  25,
}

contact := ContactInfo{
    Email:   "zhangsan@example.com",
    Phone:   "13800138000",
    Address: "北京市朝阳区",
}

work := WorkInfo{
    Company:    "ABC 公司",
    Position:   "软件工程师",
    Department: "技术部",
}

settings := SettingsInfo{
    Theme:         "dark",
    Language:      "zh-CN",
    Notifications: true,
}

// 合并多个结构体到一个 Record
record := eorm.NewRecord().
    FromStruct(base).
    FromStruct(contact).
    FromStruct(work).
    FromStruct(settings)

// 访问合并后的数据
fmt.Println("ID:", record.GetInt("id"))
fmt.Println("姓名:", record.GetString("name"))
fmt.Println("年龄:", record.GetInt("age"))
fmt.Println("邮箱:", record.GetString("email"))
fmt.Println("电话:", record.GetString("phone"))
fmt.Println("地址:", record.GetString("address"))
fmt.Println("公司:", record.GetString("company"))
fmt.Println("职位:", record.GetString("position"))
fmt.Println("部门:", record.GetString("department"))
fmt.Println("主题:", record.GetString("theme"))
fmt.Println("语言:", record.GetString("language"))
fmt.Println("通知:", record.GetBool("notifications"))

// 输出完整的 Record
fmt.Println("完整数据:", record.ToJson())
```

### 12. FromRecord

从另一个 Record 填充当前 Record（浅拷贝），支持链式调用合并多个 Record。

**简单示例：**

```go
sourceRecord := eorm.NewRecord().
    Set("name", "张三").
    Set("age", 25)

record := eorm.NewRecord().FromRecord(sourceRecord)
```

**复杂示例：合并多个 Record**

```go
// 基础信息 Record
baseInfo := eorm.NewRecord().
    Set("id", 1).
    Set("name", "张三").
    Set("age", 25)

// 联系信息 Record
contactInfo := eorm.NewRecord().
    Set("email", "zhangsan@example.com").
    Set("phone", "13800138000").
    Set("address", "北京市朝阳区")

// 工作信息 Record
workInfo := eorm.NewRecord().
    Set("company", "ABC 公司").
    Set("position", "软件工程师").
    Set("department", "技术部")

// 设置信息 Record
settingsInfo := eorm.NewRecord().
    Set("theme", "dark").
    Set("language", "zh-CN").
    Set("notifications", true)

// 合并多个 Record 到一个 Record
record := eorm.NewRecord().
    FromRecord(baseInfo).
    FromRecord(contactInfo).
    FromRecord(workInfo).
    FromRecord(settingsInfo)

// 访问合并后的数据
fmt.Println("ID:", record.GetInt("id"))
fmt.Println("姓名:", record.GetString("name"))
fmt.Println("年龄:", record.GetInt("age"))
fmt.Println("邮箱:", record.GetString("email"))
fmt.Println("电话:", record.GetString("phone"))
fmt.Println("地址:", record.GetString("address"))
fmt.Println("公司:", record.GetString("company"))
fmt.Println("职位:", record.GetString("position"))
fmt.Println("部门:", record.GetString("department"))
fmt.Println("主题:", record.GetString("theme"))
fmt.Println("语言:", record.GetString("language"))
fmt.Println("通知:", record.GetBool("notifications"))

// 输出完整的 Record
fmt.Println("完整数据:", record.ToJson())
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


**浅拷贝说明：适合扁平化的基本数据类型**

```go
// 创建源 Record - 只包含扁平化的基本数据类型
source := eorm.NewRecord().
    Set("name", "张三").
    Set("age", 25).
    Set("email", "zhangsan@example.com").
    Set("phone", "13800138000").
    Set("active", true).
    Set("score", 95.5)

// 浅拷贝
copy := eorm.NewRecord().FromRecord(source)

// 修改基本类型 - 不会影响源 Record
copy.Set("name", "李四")
copy.Set("age", 30)
copy.Set("email", "lisi@example.com")
copy.Set("active", false)
copy.Set("score", 88.5)

fmt.Println("修改基本类型后:")
fmt.Println("源 Record name:", source.GetString("name"))      // 输出: 张三（未受影响）
fmt.Println("源 Record age:", source.GetInt("age"))          // 输出: 25（未受影响）
fmt.Println("源 Record email:", source.GetString("email"))    // 输出: zhangsan@example.com（未受影响）
fmt.Println("源 Record active:", source.GetBool("active"))      // 输出: true（未受影响）
fmt.Println("源 Record score:", source.GetFloat("score"))      // 输出: 95.5（未受影响）
fmt.Println()
fmt.Println("拷贝 Record name:", copy.GetString("name"))      // 输出: 李四
fmt.Println("拷贝 Record age:", copy.GetInt("age"))          // 输出: 30
fmt.Println("拷贝 Record email:", copy.GetString("email"))    // 输出: lisi@example.com
fmt.Println("拷贝 Record active:", copy.GetBool("active"))      // 输出: false
fmt.Println("拷贝 Record score:", copy.GetFloat("score"))      // 输出: 88.5

// 总结：
// - 浅拷贝适合扁平化的基本数据类型（string, int, bool, float64 等）
// - 修改基本类型字段，拷贝和源 Record 完全独立，互不影响
// - 如果 Record 包含嵌套对象（map, slice, *Record 等），建议使用 DeepClone()
// - DeepClone() 会递归复制所有嵌套对象，确保完全独立
```

### 14. DeepClone

创建 Record 的深拷贝。

**简单示例：**

```go
original := eorm.NewRecord().
    Set("name", "张三").
    Set("profile", eorm.NewRecord().
        Set("city", "北京").
        Set("age", 25))

cloned := original.DeepClone()
```

**复杂示例：独立修改**

```go
// 创建包含嵌套对象的 Record
original := eorm.NewRecord().
    Set("name", "张三").
    Set("profile", eorm.NewRecord().
        Set("city", "北京").
        Set("age", 25))

// 深拷贝
cloned := original.DeepClone()

// 修改克隆的嵌套对象
profile, _ := cloned.GetRecord("profile")
profile.Set("city", "上海")
profile.Set("age", 30)

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
    Set("profile", eorm.NewRecord().
        Set("theme", "light").
        Set("language", "zh-CN"))

// 创建用户特定配置
userConfig := eorm.NewRecord().
    FromRecordDeep(templateConfig).
    Set("user_id", 12345).
    Set("username", "zhangsan")

// 修改嵌套配置
profile, _ := userConfig.GetRecord("profile")
profile.Set("theme", "dark")

// 修改用户配置不影响模板
profile.Set("language", "en-US")

// 模板配置保持不变
fmt.Println("模板:", templateConfig.ToJson())
fmt.Println("用户:", userConfig.ToJson())
```

### 16. GetRecord

获取嵌套的 Record。

**简单示例：**

```go
record.Set("profile", eorm.NewRecord().
    Set("name", "张三").
    Set("age", 25))

profile, err := record.GetRecord("profile")
if err == nil {
    fmt.Println("姓名:", profile.GetString("name"))
    fmt.Println("年龄:", profile.GetInt("age"))
}
```

**复杂示例：多层嵌套访问**

```go
// 创建多层嵌套结构
record.Set("user", eorm.NewRecord().
    Set("profile", eorm.NewRecord().
        Set("basic", eorm.NewRecord().
            Set("name", "张三").
            Set("age", 25)).
        Set("contact", eorm.NewRecord().
            Set("email", "zhangsan@example.com").
            Set("phone", "13800138000"))))

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
record.Set("data", eorm.NewRecord().
    Set("user", eorm.NewRecord().
        Set("name", "张三").
        Set("age", 25)))

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
record.Set("user", eorm.NewRecord().
    Set("profile", eorm.NewRecord().
        Set("city", "北京")))

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

### 19. GetSlice

获取切片值，返回 []interface{}。

**简单示例：**

```go
record := eorm.NewRecord().FromJson(`{
    "hobbies": ["读书", "游泳", "旅游"],
    "scores": [85, 90, 95]
}`)

hobbies, err := record.GetSlice("hobbies")
if err == nil {
    fmt.Println("爱好:", hobbies)
}

scores, err := record.GetSlice("scores")
if err == nil {
    fmt.Println("分数:", scores)
}
```

**复杂示例：混合类型切片**

```go
record := eorm.NewRecord().FromJson(`{
    "mixed": ["string", 123, true, 45.67]
}`)

mixed, err := record.GetSlice("mixed")
if err == nil {
    for i, item := range mixed {
        fmt.Printf("[%d] %v (类型: %T)\n", i, item, item)
    }
}
```

### 20. GetStringSlice

获取字符串切片，自动转换为 []string。

**简单示例：**

```go
record := eorm.NewRecord().FromJson(`{
    "tags": ["developer", "golang", "database"],
    "hobbies": ["读书", "游泳", "旅游"]
}`)

tags, err := record.GetStringSlice("tags")
if err == nil {
    for i, tag := range tags {
        fmt.Printf("[%d] %s\n", i, tag)
    }
}
```

**复杂示例：字符串自动分割**

```go
record := eorm.NewRecord()

// 设置带分隔符的字符串
record.Set("comma_separated", "apple,banana,orange")
record.Set("semicolon_separated", "red;green;blue")
record.Set("pipe_separated", "cat|dog|bird")
record.Set("space_separated", "hello world go")

// 自动分割
commaSlice, _ := record.GetStringSlice("comma_separated")
fmt.Println("逗号分隔:", commaSlice)  // ["apple", "banana", "orange"]

semicolonSlice, _ := record.GetStringSlice("semicolon_separated")
fmt.Println("分号分隔:", semicolonSlice)  // ["red", "green", "blue"]

pipeSlice, _ := record.GetStringSlice("pipe_separated")
fmt.Println("竖线分隔:", pipeSlice)  // ["cat", "dog", "bird"]

spaceSlice, _ := record.GetStringSlice("space_separated")
fmt.Println("空格分隔:", spaceSlice)  // ["hello", "world", "go"]

// 单元素切片
record.Set("single_value", "hello")
singleSlice, _ := record.GetStringSlice("single_value")
fmt.Println("单元素切片:", singleSlice)  // ["hello"]
```

### 21. GetIntSlice

获取整数切片，自动转换为 []int。

**简单示例：**

```go
record := eorm.NewRecord().FromJson(`{
    "scores": [85, 90, 95],
    "ages": [25, 30, 35]
}`)

scores, err := record.GetIntSlice("scores")
if err == nil {
    for i, score := range scores {
        fmt.Printf("[%d] %d\n", i, score)
    }
}
```



### 22. GetSliceByPath

通过点分路径获取嵌套的切片。

**简单示例：**

```go
record := eorm.NewRecord().FromJson(`{
    "contact": {
        "phones": ["13800138000", "13900139000"],
        "emails": ["zhangsan@example.com", "zhangsan@work.com"]
    }
}`)

phones, err := record.GetSliceByPath("contact.phones")
if err == nil {
    fmt.Println("电话:", phones)
}

emails, err := record.GetSliceByPath("contact.emails")
if err == nil {
    fmt.Println("邮箱:", emails)
}
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
