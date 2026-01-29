# EORM API 文档

EORM 是一个基于 Go 的高性能 ORM 框架，采用类似 JFinal ActiveRecord 的设计模式，支持多数据库、缓存、事务、分页、SQL模板等功能。

## 目录

- [数据库连接](#数据库连接)
- [数据库连接监控](#数据库连接监控)
- [查询超时控制](#查询超时控制)
- [基础查询](#基础查询)
- [Record 对象相关操作](#record-对象相关操作)
- [DbModel 操作](#dbmodel-操作)
- [链式查询](#链式查询)
- [事务操作](#事务操作)
- [批量操作](#批量操作)
- [分页查询](#分页查询)
- [缓存功能](#缓存功能)
- [SQL 模板](#sql-模板)
- [日志配置](#日志配置)
- [增强功能配置](#增强功能配置)
- [功能支持说明](#功能支持说明)
- [工具函数](#工具函数)

---

## 数据库连接

### OpenDatabase
```go
func OpenDatabase(driver DriverType, dsn string, maxOpen int) (*DB, error)
```
使用默认配置打开数据库连接。

**示例：**
```go
db, err := eorm.OpenDatabase(eorm.MySQL, "root:password@tcp(localhost:3306)/test", 10)
if err != nil {
    log.Fatal(err)
}
```

### OpenDatabaseWithDBName
```go
func OpenDatabaseWithDBName(dbname string, driver DriverType, dsn string, maxOpen int) (*DB, error)
```
使用指定名称打开数据库连接（多数据库模式）。

**示例：**
```go
db1, err := eorm.OpenDatabaseWithDBName("db1", eorm.MySQL, "root:password@tcp(localhost:3306)/db1", 10)
db2, err := eorm.OpenDatabaseWithDBName("db2", eorm.PostgreSQL, "postgres://user:password@localhost/db2", 10)
```

### OpenDatabaseWithConfig
```go
func OpenDatabaseWithConfig(dbname string, config *Config) (*DB, error)
```
使用完整配置打开数据库连接。

**Config 结构体：**
```go
type Config struct {
    Driver          DriverType    // 数据库驱动类型
    DSN             string        // 数据源名称
    MaxOpen         int           // 最大打开连接数
    MaxIdle         int           // 最大空闲连接数
    ConnMaxLifetime time.Duration // 连接最大生命周期
    QueryTimeout    time.Duration // 默认查询超时时间（0表示不限制）
    
    // 连接监控配置
    MonitorNormalInterval time.Duration // 正常检查间隔（默认60秒，0表示禁用监控）
    MonitorErrorInterval  time.Duration // 故障检查间隔（默认10秒）
}
```

**示例：**
```go
config := &eorm.Config{
    Driver:          eorm.MySQL,
    DSN:             "root:password@tcp(localhost:3306)/test",
    MaxOpen:         20,
    MaxIdle:         10,
    ConnMaxLifetime: time.Hour,
    QueryTimeout:    30 * time.Second,
    MonitorNormalInterval: 60 * time.Second,
    MonitorErrorInterval:  10 * time.Second,
}
db, err := eorm.OpenDatabaseWithConfig("main", config)
```

### Use
```go
func Use(dbname string) *DB
```
切换到指定名称的数据库,适用于链式调用。

**示例：**
```go
// 使用不同数据库
eorm.Use("db1").Query("SELECT * FROM users")
eorm.Use("db2").Query("SELECT * FROM products")
```

### UseWithError
```go
func UseWithError(dbname string) (*DB, error)
```
切换到指定数据库，若有错误,返回错误信息。

---

## 基础查询

### Query
```go
func Query(querySQL string, args ...interface{}) ([]*Record, error)
func (db *DB) Query(querySQL string, args ...interface{}) ([]*Record, error)
func (tx *Tx) Query(querySQL string, args ...interface{}) ([]*Record, error)
```
执行 SQL 查询，返回 Record 切片。

**示例：**
```go
// 全局函数
records, err := eorm.Query("SELECT * FROM users WHERE age > ?", 18)

// DB 实例方法
records, err := db.Query("SELECT * FROM users WHERE age > ?", 18)

// 事务中查询
tx.Query("SELECT * FROM users WHERE age > ?", 18)
```

### QueryFirst
```go
func QueryFirst(querySQL string, args ...interface{}) (*Record, error)
func (db *DB) QueryFirst(querySQL string, args ...interface{}) (*Record, error)
func (tx *Tx) QueryFirst(querySQL string, args ...interface{}) (*Record, error)
```
执行 SQL 查询，返回第一条记录。

**示例：**
```go
record, err := eorm.QueryFirst("SELECT * FROM users WHERE id = ?", 1)
if record != nil {
    fmt.Println("用户名:", record.GetString("name"))
}
```

### QueryMap
```go
func QueryMap(querySQL string, args ...interface{}) ([]map[string]interface{}, error)
func (db *DB) QueryMap(querySQL string, args ...interface{}) ([]map[string]interface{}, error)
func (tx *Tx) QueryMap(querySQL string, args ...interface{}) ([]map[string]interface{}, error)
```
执行 SQL 查询，返回 map 切片。

**示例：**
```go
maps, err := eorm.QueryMap("SELECT name, age FROM users")
for _, m := range maps {
    fmt.Printf("姓名: %v, 年龄: %v\n", m["name"], m["age"])
}
```

### QueryToDbModel
```go
func QueryToDbModel(dest interface{}, querySQL string, args ...interface{}) error
func (db *DB) QueryToDbModel(dest interface{}, querySQL string, args ...interface{}) error
func (tx *Tx) QueryToDbModel(dest interface{}, querySQL string, args ...interface{}) error
```
执行查询并将结果映射到结构体切片。

**示例：**
```go
var users []User
err := eorm.QueryToDbModel(&users, "SELECT * FROM users WHERE age > ?", 18)
```

### QueryFirstToDbModel
```go
func QueryFirstToDbModel(dest interface{}, querySQL string, args ...interface{}) error
func (db *DB) QueryFirstToDbModel(dest interface{}, querySQL string, args ...interface{}) error
func (tx *Tx) QueryFirstToDbModel(dest interface{}, querySQL string, args ...interface{}) error
```
执行查询并将第一条记录映射到结构体。

**示例：**
```go
var user User
err := eorm.QueryFirstToDbModel(&user, "SELECT * FROM users WHERE id = ?", 1)
```

### QueryWithOutTrashed
```go
func QueryWithOutTrashed(querySQL string, args ...interface{}) ([]*Record, error)
func (db *DB) QueryWithOutTrashed(querySQL string, args ...interface{}) ([]*Record, error)
func (tx *Tx) QueryWithOutTrashed(querySQL string, args ...interface{}) ([]*Record, error)
```
执行查询并自动过滤软删除记录。

**示例：**
```go
// 自动过滤已软删除的用户
records, err := eorm.QueryWithOutTrashed("SELECT * FROM users")
```

### QueryFirstWithOutTrashed
```go
func QueryFirstWithOutTrashed(querySQL string, args ...interface{}) (*Record, error)
func (db *DB) QueryFirstWithOutTrashed(querySQL string, args ...interface{}) (*Record, error)
func (tx *Tx) QueryFirstWithOutTrashed(querySQL string, args ...interface{}) (*Record, error)
```
执行查询并返回第一条非软删除记录。

### Exec
```go
func Exec(querySQL string, args ...interface{}) (sql.Result, error)
func (db *DB) Exec(querySQL string, args ...interface{}) (sql.Result, error)
func (tx *Tx) Exec(querySQL string, args ...interface{}) (sql.Result, error)
```
执行 SQL 命令（INSERT、UPDATE、DELETE）。

**示例：**
```go
result, err := eorm.Exec("UPDATE users SET age = ? WHERE id = ?", 25, 1)
affected, _ := result.RowsAffected()
fmt.Printf("影响行数: %d\n", affected)
```

---

## Record 对象相关操作

### NewRecord
```go
func NewRecord() *Record
```
创建新的 Record 实例。

### NewRecordFromPool
```go
func NewRecordFromPool() *Record
```
从对象池获取 Record 实例。

**适用场景：**
- 高并发 Web API（减少 GC 压力）。
- 批量处理成千上万条记录（降低内存分配频率）。
- 循环内频繁创建/销毁 Record 的高性能路径。

**注意：** 使用完毕后必须调用 `Release()` 归还。

### Release
```go
func (r *Record) Release()
```
将 Record 归还到对象池。归还后严禁再次使用该对象。

**示例：**
```go
record := eorm.NewRecordFromPool()
defer record.Release() // 推荐使用 defer 确保归还

record.Set("name", "张三")
// ... 执行数据库操作
```

### SaveRecord
```go
func SaveRecord(table string, record *Record) (int64, error)
func (db *DB) SaveRecord(table string, record *Record) (int64, error)
func (tx *Tx) SaveRecord(table string, record *Record) (int64, error)
```
保存记录（主键存在则更新，不存在则插入）。

**功能支持：** ✅ 自动时间戳 | ✅ 乐观锁 | ❌ 软删除

**示例：**
```go
record := eorm.NewRecord()
record.Set("id", 1)
record.Set("name", "李四")
record.Set("age", 30)

id, err := eorm.SaveRecord("users", record)
```

### InsertRecord
```go
func InsertRecord(table string, record *Record) (int64, error)
func (db *DB) InsertRecord(table string, record *Record) (int64, error)
func (tx *Tx) InsertRecord(table string, record *Record) (int64, error)
```
插入新记录。

**示例：**
```go
record := eorm.NewRecord()
record.Set("name", "王五")
record.Set("age", 28)

id, err := eorm.InsertRecord("users", record)
```

### UpdateRecord
```go
func UpdateRecord(table string, record *Record) (int64, error)
func (db *DB) UpdateRecord(table string, record *Record) (int64, error)
func (tx *Tx) UpdateRecord(table string, record *Record) (int64, error)
```
根据主键更新记录。配置了自动时间戳会自动更新相应updated时间字段

**示例：**
```go
record := eorm.NewRecord()
record.Set("id", 1)
record.Set("name", "张三更新")
record.Set("age", 26)

affected, err := eorm.UpdateRecord("users", record)
```

### Update
```go
func Update(table string, record *Record, whereSql string, whereArgs ...interface{}) (int64, error)
func (db *DB) Update(table string, record *Record, whereSql string, whereArgs ...interface{}) (int64, error)
func (tx *Tx) Update(table string, record *Record, whereSql string, whereArgs ...interface{}) (int64, error)
```
根据条件更新记录。配置了自动时间戳会自动更新相应updated时间字段

**示例：**
```go
record := eorm.NewRecord()
record.Set("age", 30)

affected, err := eorm.Update("users", record, "name = ?", "张三")
```

### UpdateFast
```go
func UpdateFast(table string, record *Record, whereSql string, whereArgs ...interface{}) (int64, error)
func (db *DB) UpdateFast(table string, record *Record, whereSql string, whereArgs ...interface{}) (int64, error)
func (tx *Tx) UpdateFast(table string, record *Record, whereSql string, whereArgs ...interface{}) (int64, error)
```
快速更新，跳过时间戳和乐观锁检查。

**示例：**
```go
record := eorm.NewRecord()
record.Set("status", "active")

// 高性能更新，跳过时间戳和乐观锁检查
affected, err := eorm.UpdateFast("users", record, "id = ?", 1)
```

### DeleteRecord
```go
func DeleteRecord(table string, record *Record) (int64, error)
func (db *DB) DeleteRecord(table string, record *Record) (int64, error)
func (tx *Tx) DeleteRecord(table string, record *Record) (int64, error)
```
根据主键删除记录。配置软删除则是更新.

**示例：**
```go
record := eorm.NewRecord()
record.Set("id", 1)

affected, err := eorm.DeleteRecord("users", record)
```

### Delete
```go
func Delete(table string, whereSql string, whereArgs ...interface{}) (int64, error)
func (db *DB) Delete(table string, whereSql string, whereArgs ...interface{}) (int64, error)
func (tx *Tx) Delete(table string, whereSql string, whereArgs ...interface{}) (int64, error)
```
根据条件删除记录。配置软删除则是更新.

**示例：**
```go
// 软删除（如果配置了软删除）
affected, err := eorm.Delete("users", "age < ?", 18)
```

### Record 对象方法

#### Record.Set
```go
func (r *Record) Set(column string, value interface{}) *Record
```
设置字段值，支持链式调用。

**示例：**
```go
record := eorm.NewRecord()
record.Set("name", "张三").Set("age", 25).Set("email", "zhangsan@example.com")
```

#### Record.SetIf
```go
func (r *Record) SetIf(condition bool, column string, value interface{}) *Record
```
只有当 `condition` 为 `true` 时才设置字段。

#### Record.SetIfNotNil
```go
func (r *Record) SetIfNotNil(column string, value interface{}) *Record
```
只有当 `value` 不为 `nil` 时才设置字段。

#### Record.SetIfNotEmpty
```go
func (r *Record) SetIfNotEmpty(column, value string) *Record
```
只有当 `value` 不为空字符串时才设置字段。

#### Record.SetIfNil
```go
func (r *Record) SetIfNil(column string, value interface{}) *Record
```
只有当字段当前不存在或值为 `nil` 时才设置字段。

#### Record.SetIfEmpty
```go
func (r *Record) SetIfEmpty(column string, value string) *Record
```
只有当字段当前不存在或值为空字符串时才设置字段。

#### Record.Get
```go
func (r *Record) Get(column string) interface{}
```
获取字段值。

#### Record.GetValues
```go
func (r *Record) GetValues(columns ...string) []interface{}
```
批量获取多个字段的值。该方法是线程安全的，且比多次调用 `Get` 效率更高。

**示例：**
```go
values := record.GetValues("name", "age", "email")
// values[0] 为 name, values[1] 为 age, values[2] 为 email
```

#### 类型安全获取方法
```go
func (r *Record) GetString(column string) string
func (r *Record) GetInt(column string) int
func (r *Record) GetInt64(column string) int64
func (r *Record) GetInt32(column string) int32
func (r *Record) GetInt16(column string) int16
func (r *Record) GetUint(column string) uint
func (r *Record) GetFloat(column string) float64
func (r *Record) GetFloat32(column string) float32
func (r *Record) GetBytes(column string) []byte
func (r *Record) GetTime(column string) time.Time
func (r *Record) GetBool(column string) bool

// 简写方法（向后兼容，无错误返回）
func (r *Record) Str(column string) string
func (r *Record) Int(column string) int
func (r *Record) Int64(column string) int64
func (r *Record) Float(column string) float64
func (r *Record) Bool(column string) bool
```

**示例：**
```go
record, _ := eorm.QueryFirst("SELECT * FROM users WHERE id = ?", 1)

name := record.GetString("name")
age := record.GetInt("age")
email := record.Str("email")  // 简写方法
isActive := record.Bool("is_active")
```

#### Record.Has
```go
func (r *Record) Has(column string) bool
```
检查字段是否存在。

**示例：**
```go
if record.Has("email") {
    fmt.Println("邮箱:", record.GetString("email"))
}
```

#### Record.Keys
```go
func (r *Record) Keys() []string
```
获取所有字段名。

**示例：**
```go
keys := record.Keys()
fmt.Println("字段列表:", keys)
```

#### Record.Columns
```go
func (r *Record) Columns() []string
```
获取所有字段名（别名方法，与 Keys 相同）。

**示例：**
```go
columns := record.Columns()
fmt.Println("字段列表:", columns)
```

#### Record.Remove
```go
func (r *Record) Remove(column string)
```
删除字段。

**示例：**
```go
record.Remove("password")  // 删除敏感字段
```

#### Record.Clear
```go
func (r *Record) Clear()
```
清空所有字段。

**示例：**
```go
record.Clear()  // 清空记录
```

#### Record.Transform
```go
func (r *Record) Transform(fn func(key string, value interface{}) interface{}) *Record
```
批量转换 Record 中的数据（处理键和值）。

**示例：**
```go
record.Transform(func(key string, value interface{}) interface{} {
    if s, ok := value.(string); ok {
        return strings.TrimSpace(s) // 去除所有字符串字段的前后空格
    }
    return value
})
```

#### Record.TransformValues
```go
func (r *Record) TransformValues(fn func(value interface{}) interface{}) *Record
```
批量转换 Record 中的数据（只处理值）。

**示例：**
```go
record.TransformValues(func(value interface{}) interface{} {
    if s, ok := value.(string); ok {
        return strings.ToUpper(s) // 将所有字符串转换为大写
    }
    return value
})
```

#### Record.ToMap
```go
func (r *Record) ToMap() map[string]interface{}
```
转换为 map。

**示例：**
```go
dataMap := record.ToMap()
fmt.Printf("数据: %+v\n", dataMap)
```

#### Record.ToJson
```go
func (r *Record) ToJson() string
```
转换为 JSON 字符串。

**示例：**
```go
jsonStr := record.ToJson()
fmt.Println("JSON:", jsonStr)
```

#### Record.FromJson
```go
func (r *Record) FromJson(jsonStr string) *Record
```
从 JSON 字符串解析并合并到当前 Record。

**示例：**

```go
record := eorm.NewRecord()
record.FromJson(`{"name":"张三","age":25}`).FromJson(`{"address":"xxxxx","email":"xxxx@xxx.com"}`)
```

#### Record.ToStruct
```go
func (r *Record) ToStruct(dest interface{}) error
```
转换为结构体。

**示例：**
```go
type User struct {
    Name string `json:"name"`
    Age  int    `json:"age"`
}

var user User
err := record.ToStruct(&user)
if err != nil {
    log.Fatal(err)
}
```



#### Record.FromMap

```go
func (r *Record) FromMap(m map[string]interface{}) *Record
```

 将 map 中的数据填充到当前 Record，支持链式调用 。

**示例：**

```go
record := eorm.NewRecord()
record.FromMap(map[string]interface{}{
    "name": "张三",
    "age": 25,
}).FromMap(map[string]interface{}{
    "address": "xxxx",
    "email": "xxxx@xxx.com",
}).Set("extra", "value")
```



#### Record.FromStruct

```go
func (r *Record) FromStruct(src interface{}) *Record
```
从结构体填充。

**示例：**

```go
user := User{Name: "李四", Age: 30}
info := Info{Address: "xxxxx", Email: "xxxx@xxx.com"}
record := eorm.NewRecord()
record.FromStruct(user).FromStruct(info)

```

#### Record.FromRecord

```go
func (r *Record) FromRecord(src *Record) *Record
```
从另一个 Record 填充当前 Record，支持链式调用。使用浅拷贝复制数据。

**示例：**

```go
sourceRecord := eorm.NewRecord().
    FromJson(`{"id": 1, "name": "张三", "age": 25}`)

// 从 sourceRecord 复制数据到新 Record
record := eorm.NewRecord().
    FromRecord(sourceRecord).
    Set("email", "zhangsan@example.com").
    Set("active", true)

fmt.Println(record.ToJson())
// 输出: {"id":1,"name":"张三","age":25,"email":"zhangsan@example.com","active":true}
```

#### Record.GetRecord
```go
func (r *Record) GetRecord(column string) (*Record, error)
```
获取嵌套的 Record。

**示例：**
```go
record := eorm.NewRecord().FromJson(`{
    "user": {
        "name": "张三",
        "profile": {
            "age": 25
        }
    }
}`)

user, err := record.GetRecord("user")
if err != nil {
    // 处理错误
} else {
    fmt.Println("姓名:", user.GetString("name"))
    profile, _ := user.GetRecord("profile")
    fmt.Println("年龄:", profile.GetInt("age"))
}
```

#### Record.GetRecords
```go
func (r *Record) GetRecords(column string) ([]*Record, error)
```
获取嵌套的 Record 数组。

**示例：**
```go
record := eorm.NewRecord().FromJson(`{
    "users": [
        {"name": "张三", "age": 25},
        {"name": "李四", "age": 30}
    ]
}`)

users, err := record.GetRecords("users")
if err != nil {
    // 处理错误
} else {
    for _, user := range users {
        fmt.Printf("姓名: %s, 年龄: %d\n", user.GetString("name"), user.GetInt("age"))
    }
}
```

#### Record.GetRecordByPath
```go
func (r *Record) GetRecordByPath(path string) (*Record, error)
```
通过点分路径获取嵌套 Record。

**示例：**
```go
record := eorm.NewRecord().FromJson(`{
    "data": {
        "user": {
            "profile": {
                "name": "张三",
                "age": 25
            }
        }
    }
}`)

profile, err := record.GetRecordByPath("data.user.profile")
if err != nil {
    // 处理错误
} else {
    fmt.Printf("姓名: %s, 年龄: %d\n", profile.GetString("name"), profile.GetInt("age"))
}
```

#### Record.GetStringByPath
```go
func (r *Record) GetStringByPath(path string) (string, error)
```
通过点分路径获取嵌套的字符串值。

**示例：**
```go
record := eorm.NewRecord().FromJson(`{
    "user": {
        "name": "张三",
        "contact": {
            "email": "zhangsan@example.com",
            "phone": "13800138000"
        }
    }
}`)

email, err := record.GetStringByPath("user.contact.email")
if err != nil {
    // 处理错误
} else {
    fmt.Println("邮箱:", email)
}

// 如果路径指向 Record，会返回 JSON 字符串
contact, err := record.GetStringByPath("user.contact")
if err != nil {
    // 处理错误
} else {
    fmt.Println("联系方式:", contact)  // 以 JSON 格式输出
}
```



#### Record.GetSlice
```go
func (r *Record) GetSlice(column string) ([]interface{}, error)
```
获取切片值，返回 []interface{}。

**示例：**
```go
record := eorm.NewRecord().FromJson(`{
    "hobbies": ["读书", "游泳", "旅游"],
    "scores": [85, 90, 95]
}`)

hobbies, err := record.GetSlice("hobbies")
if err != nil {
    // 处理错误
} else {
    fmt.Println("爱好:", hobbies)
}
```

#### Record.GetStringSlice
```go
func (r *Record) GetStringSlice(column string) ([]string, error)
```
获取字符串切片，自动转换为 []string。

**示例：**
```go
record := eorm.NewRecord().FromJson(`{
    "tags": ["developer", "golang", "database"],
    "hobbies": ["读书", "游泳", "旅游"]
}`)

tags, err := record.GetStringSlice("tags")
if err != nil {
    // 处理错误
} else {
    for i, tag := range tags {
        fmt.Printf("[%d] %s\n", i, tag)
    }
}

// 字符串自动分割功能
record2 := eorm.NewRecord()
record2.Set("comma_separated", "apple,banana,orange")
commaSlice, _ := record2.GetStringSlice("comma_separated")
fmt.Println("逗号分隔:", commaSlice)  // ["apple", "banana", "orange"]
```

#### Record.GetIntSlice
```go
func (r *Record) GetIntSlice(column string) ([]int, error)
```
获取整数切片，自动转换为 []int。

**示例：**
```go
record := eorm.NewRecord().FromJson(`{
    "scores": [85, 90, 95],
    "ages": [25, 30, 35]
}`)

scores, err := record.GetIntSlice("scores")
if err != nil {
    // 处理错误
} else {
    for i, score := range scores {
        fmt.Printf("[%d] %d\n", i, score)
    }
    // 计算平均分
    total := 0
    for _, score := range scores {
        total += score
    }
    average := float64(total) / float64(len(scores))
    fmt.Printf("总分: %d, 平均分: %.2f\n", total, average)
}
```

#### Record.GetSliceByPath
```go
func (r *Record) GetSliceByPath(path string) ([]interface{}, error)
```
通过点分路径获取嵌套的切片。

**示例：**
```go
record := eorm.NewRecord().FromJson(`{
    "contact": {
        "phones": ["13800138000", "13900139000"],
        "emails": ["zhangsan@example.com", "zhangsan@work.com"]
    }
}`)

phones, err := record.GetSliceByPath("contact.phones")
if err != nil {
    // 处理错误
} else {
    fmt.Println("电话:", phones)
}

emails, err := record.GetSliceByPath("contact.emails")
if err != nil {
    // 处理错误
} else {
    fmt.Println("邮箱:", emails)
}
```

---

## DbModel 操作

### SaveDbModel
```go
func SaveDbModel(model IDbModel) (int64, error)
func (db *DB) SaveDbModel(model IDbModel) (int64, error)
func (tx *Tx) SaveDbModel(model IDbModel) (int64, error)
```
保存 DbModel 实例。

**示例：**
```go
user := &User{
    ID:   1,
    Name: "张三",
    Age:  25,
}

id, err := eorm.SaveDbModel(user)
// 或者
id, err := user.Save()
```

### InsertDbModel
```go
func InsertDbModel(model IDbModel) (int64, error)
func (db *DB) InsertDbModel(model IDbModel) (int64, error)
func (tx *Tx) InsertDbModel(model IDbModel) (int64, error)
```
插入 DbModel 实例。

**示例：**
```go
user := &User{
    Name: "李四",
    Age:  30,
}

id, err := eorm.InsertDbModel(user)
// 或者
id, err := user.Insert()
```

### UpdateDbModel
```go
func UpdateDbModel(model IDbModel) (int64, error)
func (db *DB) UpdateDbModel(model IDbModel) (int64, error)
func (tx *Tx) UpdateDbModel(model IDbModel) (int64, error)
```
更新 DbModel 实例。

**示例：**
```go
user.Age = 31
affected, err := eorm.UpdateDbModel(user)
// 或者
affected, err := user.Update()
```

### DeleteDbModel
```go
func DeleteDbModel(model IDbModel) (int64, error)
func (db *DB) DeleteDbModel(model IDbModel) (int64, error)
func (tx *Tx) DeleteDbModel(model IDbModel) (int64, error)
```
删除 DbModel 实例。

**示例：**
```go
affected, err := eorm.DeleteDbModel(user)
// 或者
affected, err := user.Delete()
```

### FindFirstToDbModel
```go
func FindFirstToDbModel(model IDbModel, whereSql string, whereArgs ...interface{}) error
func (db *DB) FindFirstToDbModel(model IDbModel, whereSql string, whereArgs ...interface{}) error
func (tx *Tx) FindFirstToDbModel(model IDbModel, whereSql string, whereArgs ...interface{}) error
```
查找第一条记录并映射到 DbModel。

**示例：**
```go
user := &User{}
err := eorm.FindFirstToDbModel(user, "name = ?", "张三")
```

### FindToDbModel
```go
func FindToDbModel(dest interface{}, table string, whereSql string, orderBySql string, whereArgs ...interface{}) error
func (db *DB) FindToDbModel(dest interface{}, table string, whereSql string, orderBySql string, whereArgs ...interface{}) error
func (tx *Tx) FindToDbModel(dest interface{}, table string, whereSql string, orderBySql string, whereArgs ...interface{}) error
```
查找多条记录并映射到结构体切片。

**示例：**
```go
var users []User
err := eorm.FindToDbModel(&users, "users", "age > ?", "age DESC", 18)
```

### GenerateDbModel
```go
func GenerateDbModel(tablename, outPath, structName string) error
func (db *DB) GenerateDbModel(tablename, outPath, structName string) error
```
根据数据表生成 Go 结构体代码。

**参数：**
- `tablename`: 表名
- `outPath`: 输出路径（目录或完整文件路径）
- `structName`: 结构体名称（空则自动生成）

**示例：**
```go
// 生成 User 结构体到指定文件
err := eorm.GenerateDbModel("users", "models/user.go", "User")

// 自动生成结构体名称
err := eorm.GenerateDbModel("products", "models/", "")
```

### GetTableColumns
```go
func GetTableColumns(table string) ([]ColumnInfo, error)
func (db *DB) GetTableColumns(table string) ([]ColumnInfo, error)
```
获取指定表的所有列信息。

**返回值：**
- `[]ColumnInfo`: 列信息切片，包含列名、类型、是否可空、是否主键、备注等信息

**ColumnInfo 结构体：**
```go
type ColumnInfo struct {
    Name     string // 列名
    Type     string // 数据类型
    Nullable bool   // 是否可空
    IsPK     bool   // 是否主键
    Comment  string // 列备注
}
```

**示例：**
```go
// 使用全局函数
columns, err := eorm.GetTableColumns("users")
if err != nil {
    log.Fatal(err)
}

for _, col := range columns {
    fmt.Printf("列名: %s, 类型: %s, 可空: %v, 主键: %v\n", 
        col.Name, col.Type, col.Nullable, col.IsPK)
    if col.Comment != "" {
        fmt.Printf("  备注: %s\n", col.Comment)
    }
}

// 使用 DB 实例方法
columns, err := db.GetTableColumns("orders")

// 多数据库模式
columns, err := eorm.Use("db2").GetTableColumns("products")
```

### GetAllTables
```go
func GetAllTables() ([]string, error)
func (db *DB) GetAllTables() ([]string, error)
```
获取数据库中所有表名。

**返回值：**
- `[]string`: 表名列表

**示例：**
```go
// 使用全局函数
tables, err := eorm.GetAllTables()
if err != nil {
    log.Fatal(err)
}

fmt.Printf("数据库中共有 %d 个表:\n", len(tables))
for i, table := range tables {
    fmt.Printf("%d. %s\n", i+1, table)
}

// 使用 DB 实例方法
tables, err := db.GetAllTables()

// 多数据库模式
tables, err := eorm.Use("db2").GetAllTables()
```

**支持的数据库：**
- MySQL: 使用 `INFORMATION_SCHEMA.TABLES`
- PostgreSQL: 使用 `pg_catalog.pg_tables`
- SQLite3: 使用 `sqlite_master`
- Oracle: 使用 `USER_TABLES`
- SQL Server: 使用 `INFORMATION_SCHEMA.TABLES`

### GenerateAllDbModel
```go
func GenerateAllDbModel(outPath string) (int, error)
func (db *DB) GenerateAllDbModel(outPath string) (int, error)
```
批量生成数据库中所有表的 Model 代码。

**参数：**
- `outPath`: 输出目录路径（空字符串则使用默认的 "models" 目录）

**返回值：**
- `int`: 成功生成的文件数量
- `error`: 错误信息（如果部分表生成失败，会返回包含详细错误信息的 error）

**示例：**
```go
// 使用全局函数，生成到 models 目录
count, err := eorm.GenerateAllDbModel("models")
if err != nil {
    log.Printf("生成过程中出现错误: %v\n", err)
}
fmt.Printf("成功生成 %d 个 Model 文件\n", count)

// 使用 DB 实例方法
count, err := db.GenerateAllDbModel("generated")

// 使用默认路径
count, err := eorm.GenerateAllDbModel("")

// 多数据库模式 - 为不同数据库生成到不同目录
db1, _ := eorm.OpenDatabase(eorm.MySQL, dsn1, 10)
db2, _ := eorm.OpenDatabaseWithDBName("db2", eorm.MySQL, dsn2, 10)

count1, _ := db1.GenerateAllDbModel("models/db1")
count2, _ := db2.GenerateAllDbModel("models/db2")

fmt.Printf("数据库1生成 %d 个文件\n", count1)
fmt.Printf("数据库2生成 %d 个文件\n", count2)
```

**特性：**
- 自动获取数据库中的所有表
- 为每个表生成独立的 Go 文件
- 结构体名称自动从表名转换（如 `user_info` → `UserInfo`）
- 文件名为表名小写（如 `user_info.go`）
- 容错处理：某个表生成失败不影响其他表
- 返回成功数量和详细错误信息

**生成的文件结构：**
```
models/
├── users.go      // User 结构体
├── orders.go     // Order 结构体
└── products.go   // Product 结构体
```

**错误处理：**
```go
count, err := eorm.GenerateAllDbModel("models")
if err != nil {
    // 即使有错误，count 仍然表示成功生成的数量
    fmt.Printf("部分成功: 生成了 %d 个文件\n", count)
    fmt.Printf("错误详情: %v\n", err)
} else {
    fmt.Printf("全部成功: 生成了 %d 个文件\n", count)
}
```

### IDbModel 接口
```go
type IDbModel interface {
    TableName() string
    DatabaseName() string
}
```

生成的 DbModel 结构体会自动实现此接口。

### 泛型辅助函数

#### FindModel
```go
func FindModel[T IDbModel](model T, cache *ModelCache, whereSql, orderBySql string, whereArgs ...interface{}) ([]T, error)
```
泛型查找多条记录。

#### FindFirstModel
```go
func FindFirstModel[T IDbModel](model T, cache *ModelCache, whereSql string, whereArgs ...interface{}) (T, error)
```
泛型查找第一条记录。

#### PaginateModel
```go
func PaginateModel[T IDbModel](model T, cache *ModelCache, page, pageSize int, whereSql, orderBySql string, whereArgs ...interface{}) (*Page[T], error)
```
泛型分页查询。

### ModelCache 结构体
```go
type ModelCache struct {
    CacheRepositoryName string        // 缓存仓库名称
    CacheTTL            time.Duration // 缓存过期时间
    CountCacheTTL       time.Duration // 分页计数缓存时间
}
```

**方法：**
```go
func (c *ModelCache) SetCache(cacheRepositoryName string, ttl ...time.Duration)  // 设置缓存配置
func (c *ModelCache) WithCountCache(ttl time.Duration) *ModelCache               // 启用分页计数缓存
func (c *ModelCache) GetCache() *ModelCache                                      // 获取缓存配置
```

**示例：**
```go
// 创建用户模型并使用链式调用设置缓存
user := &User{}

// 方式一：使用链式调用（推荐）
page, err := user.Cache("user_cache", 5*time.Minute).
    WithCountCache(5*time.Minute).
    PaginateBuilder(1, 10, "age > ?", "name ASC", 18)

// 方式二：使用 PaginateModel 函数
cache := &eorm.ModelCache{}
cache.SetCache("user_cache", 5*time.Minute)
cache.WithCountCache(5*time.Minute)
page, err := eorm.PaginateModel(user, cache, 1, 10, 
    "age > ?", "name ASC", 18)
```

---

## 链式查询

### Table
```go
func Table(name string) *QueryBuilder
func (db *DB) Table(name string) *QueryBuilder
func (tx *Tx) Table(name string) *QueryBuilder
```
开始链式查询。

**示例：**
```go
// 全局函数
records, err := eorm.Table("users").Where("age > ?", 18).Find()

// DB 实例
records, err := db.Table("users").Where("age > ?", 18).Find()

// 事务中
records, err := tx.Table("users").Where("age > ?", 18).Find()
```

### Select
```go
func (qb *QueryBuilder) Select(selectSql string) *QueryBuilder
```
设置 SELECT 字段。

**示例：**
```go
records, err := eorm.Table("users").
    Select("id, name, age").
    Where("age > ?", 18).
    Find()
```

### Where
```go
func (qb *QueryBuilder) Where(whereSql string, args ...interface{}) *QueryBuilder
```
添加 WHERE 条件。

**示例：**
```go
records, err := eorm.Table("users").
    Where("age > ?", 18).
    Where("status = ?", "active").
    Find()
```

### OrWhere
```go
func (qb *QueryBuilder) OrWhere(whereSql string, args ...interface{}) *QueryBuilder
```
添加 OR WHERE 条件。

**示例：**
```go
records, err := eorm.Table("users").
    Where("age > ?", 30).
    OrWhere("vip = ?", true).
    Find()
```

### OrderBy
```go
func (qb *QueryBuilder) OrderBy(orderBy string) *QueryBuilder
```
设置排序。

**示例：**
```go
records, err := eorm.Table("users").
    Where("age > ?", 18).
    OrderBy("age DESC, name ASC").
    Find()
```

### GroupBy
```go
func (qb *QueryBuilder) GroupBy(groupBy string) *QueryBuilder
```
设置分组。

**示例：**
```go
records, err := eorm.Table("orders").
    Select("user_id, COUNT(*) as order_count").
    GroupBy("user_id").
    Find()
```

### Having
```go
func (qb *QueryBuilder) Having(havingSql string, args ...interface{}) *QueryBuilder
```
添加 HAVING 条件。

**示例：**
```go
records, err := eorm.Table("orders").
    Select("user_id, COUNT(*) as order_count").
    GroupBy("user_id").
    Having("COUNT(*) > ?", 5).
    Find()
```

### Limit
```go
func (qb *QueryBuilder) Limit(limit int) *QueryBuilder
```
设置查询限制。

**示例：**
```go
records, err := eorm.Table("users").
    Where("age > ?", 18).
    OrderBy("created_at DESC").
    Limit(10).
    Find()
```

### Offset
```go
func (qb *QueryBuilder) Offset(offset int) *QueryBuilder
```
设置查询偏移。

**示例：**
```go
records, err := eorm.Table("users").
    Where("age > ?", 18).
    OrderBy("id").
    Limit(10).
    Offset(20).
    Find()
```

### Join
```go
func (qb *QueryBuilder) Join(table, condition string, args ...interface{}) *QueryBuilder
func (qb *QueryBuilder) LeftJoin(table, condition string, args ...interface{}) *QueryBuilder
func (qb *QueryBuilder) RightJoin(table, condition string, args ...interface{}) *QueryBuilder
func (qb *QueryBuilder) InnerJoin(table, condition string, args ...interface{}) *QueryBuilder
```
添加 JOIN 查询。

**示例：**
```go
records, err := eorm.Table("users").
    Select("users.name, orders.total").
    LeftJoin("orders", "users.id = orders.user_id").
    Where("users.age > ?", 18).
    Find()
```

### WithTrashed
```go
func (qb *QueryBuilder) WithTrashed() *QueryBuilder
```
包含软删除记录。

**示例：**
```go
// 查询包含软删除的记录
records, err := eorm.Table("users").WithTrashed().Find()
```

### OnlyTrashed
```go
func (qb *QueryBuilder) OnlyTrashed() *QueryBuilder
```
只查询软删除记录。

**示例：**
```go
// 只查询已软删除的记录
records, err := eorm.Table("users").OnlyTrashed().Find()
```

### ForceDelete
```go
func ForceDelete(table string, whereSql string, whereArgs ...interface{}) (int64, error)
func (db *DB) ForceDelete(table string, whereSql string, whereArgs ...interface{}) (int64, error)
func (tx *Tx) ForceDelete(table string, whereSql string, whereArgs ...interface{}) (int64, error)
func (qb *QueryBuilder) ForceDelete() (int64, error)
```
物理删除记录，绕过软删除配置。

**示例：**
```go
// 物理删除
eorm.ForceDelete("users", "id = ?", 1)

// 链式调用
eorm.Table("users").Where("id = ?", 1).ForceDelete()
```

### Restore
```go
func Restore(table string, whereSql string, whereArgs ...interface{}) (int64, error)
func (db *DB) Restore(table string, whereSql string, whereArgs ...interface{}) (int64, error)
func (tx *Tx) Restore(table string, whereSql string, whereArgs ...interface{}) (int64, error)
func (qb *QueryBuilder) Restore() (int64, error)
```
恢复已软删除的记录。

**示例：**
```go
// 恢复记录
eorm.Restore("users", "id = ?", 1)

// 链式调用
eorm.Table("users").Where("id = ?", 1).Restore()
```

### Find
```go
func (qb *QueryBuilder) Find() ([]*Record, error)
```
执行查询并返回所有记录。

**示例：**
```go
records, err := eorm.Table("users").
    Where("age > ?", 18).
    OrderBy("name").
    Find()
```

### FindFirst
```go
func (qb *QueryBuilder) FindFirst() (*Record, error)
```
执行查询并返回第一条记录。

**示例：**
```go
record, err := eorm.Table("users").
    Where("email = ?", "user@example.com").
    FindFirst()
```

### FindToDbModel
```go
func (qb *QueryBuilder) FindToDbModel(dest interface{}) error
```
执行查询并映射到结构体切片。

**示例：**
```go
var users []User
err := eorm.Table("users").
    Where("age > ?", 18).
    FindToDbModel(&users)
```

### FindFirstToDbModel
```go
func (qb *QueryBuilder) FindFirstToDbModel(dest interface{}) error
```
执行查询并映射第一条记录到结构体。

**示例：**
```go
var user User
err := eorm.Table("users").
    Where("id = ?", 1).
    FindFirstToDbModel(&user)
```

### Count
```go
func (qb *QueryBuilder) Count() (int64, error)
```
统计记录数量。

**示例：**
```go
count, err := eorm.Table("users").
    Where("age > ?", 18).
    Count()
```

### Update
```go
func (qb *QueryBuilder) Update(record *Record) (int64, error)
```
链式更新记录。

**示例：**
```go
record := eorm.NewRecord()
record.Set("status", "inactive")

affected, err := eorm.Table("users").
    Where("last_login < ?", time.Now().AddDate(0, -6, 0)).
    Update(record)
```

### Delete
```go
func (qb *QueryBuilder) Delete() (int64, error)
```
链式删除记录。

**示例：**
```go
affected, err := eorm.Table("users").
    Where("status = ?", "inactive").
    Delete()
```

### Paginate
```go
func (qb *QueryBuilder) Paginate(page, pageSize int) (*Page[*Record], error)
```
分页查询。

**功能支持：** ❌ 自动时间戳 | ❌ 乐观锁 | ✅ 软删除

**示例：**
```go
pageResult, err := eorm.Table("users").
    Where("age > ?", 18).
    OrderBy("created_at DESC").
    Paginate(1, 10)

fmt.Printf("总记录数: %d\n", pageResult.TotalRow)
fmt.Printf("总页数: %d\n", pageResult.TotalPage)
for _, record := range pageResult.List {
    fmt.Println("用户:", record.GetString("name"))
}
```

### 高级 WHERE 条件

#### OrWhere
```go
func (qb *QueryBuilder) OrWhere(condition string, args ...interface{}) *QueryBuilder
```
添加 OR 条件到查询。当与 Where 组合使用时，AND 条件会被括号包裹以保持正确的优先级。

**示例：**
```go
// 查询状态为 active 或 priority 为 high 的订单
orders, err := eorm.Table("orders").
    Where("status = ?", "active").
    OrWhere("priority = ?", "high").
    Find()
// SQL: SELECT * FROM orders WHERE (status = ?) OR priority = ?

// 多个 OR 条件
orders, err := eorm.Table("orders").
    OrWhere("status = ?", "pending").
    OrWhere("status = ?", "processing").
    OrWhere("status = ?", "shipped").
    Find()
// SQL: SELECT * FROM orders WHERE status = ? OR status = ? OR status = ?
```

#### WhereGroup / OrWhereGroup
```go
type WhereGroupFunc func(qb *QueryBuilder) *QueryBuilder

func (qb *QueryBuilder) WhereGroup(fn WhereGroupFunc) *QueryBuilder
func (qb *QueryBuilder) OrWhereGroup(fn WhereGroupFunc) *QueryBuilder
```
添加分组条件，支持嵌套括号。`WhereGroup` 使用 AND 连接，`OrWhereGroup` 使用 OR 连接。

**示例：**
```go
// OR 分组条件
records, err := eorm.Table("table").
    Where("a = ?", 1).
    OrWhereGroup(func(qb *eorm.QueryBuilder) *eorm.QueryBuilder {
        return qb.Where("b = ?", 1).OrWhere("c = ?", 1)
    }).
    Find()
// SQL: SELECT * FROM table WHERE (a = ?) OR (b = ? OR c = ?)

// AND 分组条件
records, err := eorm.Table("orders").
    Where("status = ?", "active").
    WhereGroup(func(qb *eorm.QueryBuilder) *eorm.QueryBuilder {
        return qb.Where("type = ?", "A").OrWhere("priority = ?", "high")
    }).
    Find()
// SQL: SELECT * FROM orders WHERE status = ? AND (type = ? OR priority = ?)
```

#### WhereInValues / WhereNotInValues
```go
func (qb *QueryBuilder) WhereInValues(column string, values []interface{}) *QueryBuilder
func (qb *QueryBuilder) WhereNotInValues(column string, values []interface{}) *QueryBuilder
```
使用值列表进行 IN/NOT IN 查询。

**示例：**
```go
// 查询指定 ID 的用户
users, err := eorm.Table("users").
    WhereInValues("id", []interface{}{1, 2, 3, 4, 5}).
    Find()
// SQL: SELECT * FROM users WHERE id IN (?, ?, ?, ?, ?)

// 排除指定状态的订单
orders, err := eorm.Table("orders").
    WhereNotInValues("status", []interface{}{"cancelled", "refunded"}).
    Find()
// SQL: SELECT * FROM orders WHERE status NOT IN (?, ?)
```

#### WhereBetween / WhereNotBetween
```go
func (qb *QueryBuilder) WhereBetween(column string, min, max interface{}) *QueryBuilder
func (qb *QueryBuilder) WhereNotBetween(column string, min, max interface{}) *QueryBuilder
```
范围查询。

**示例：**
```go
// 查询年龄在 18-65 之间的用户
users, err := eorm.Table("users").
    WhereBetween("age", 18, 65).
    Find()
// SQL: SELECT * FROM users WHERE age BETWEEN ? AND ?

// 查询价格不在 100-500 之间的产品
products, err := eorm.Table("products").
    WhereNotBetween("price", 100, 500).
    Find()
// SQL: SELECT * FROM products WHERE price NOT BETWEEN ? AND ?
```

#### WhereNull / WhereNotNull
```go
func (qb *QueryBuilder) WhereNull(column string) *QueryBuilder
func (qb *QueryBuilder) WhereNotNull(column string) *QueryBuilder
```
NULL 值检查。

**示例：**
```go
// 查询没有邮箱的用户
users, err := eorm.Table("users").
    WhereNull("email").
    Find()
// SQL: SELECT * FROM users WHERE email IS NULL

// 查询有手机号的用户
users, err := eorm.Table("users").
    WhereNotNull("phone").
    Find()
// SQL: SELECT * FROM users WHERE phone IS NOT NULL
```

### 子查询功能

#### NewSubquery
```go
func NewSubquery() *Subquery
```
创建新的子查询构建器。

#### Subquery 方法
```go
func (s *Subquery) Table(name string) *Subquery
func (s *Subquery) Select(columns string) *Subquery
func (s *Subquery) Where(condition string, args ...interface{}) *Subquery
func (s *Subquery) OrderBy(orderBy string) *Subquery
func (s *Subquery) Limit(limit int) *Subquery
func (s *Subquery) ToSQL() (string, []interface{})
```

#### WHERE IN 子查询
```go
func (qb *QueryBuilder) WhereIn(column string, sub *Subquery) *QueryBuilder
func (qb *QueryBuilder) WhereNotIn(column string, sub *Subquery) *QueryBuilder
```

**示例：**
```go
// 查询有已完成订单的用户
activeUsersSub := eorm.NewSubquery().
    Table("orders").
    Select("DISTINCT user_id").
    Where("status = ?", "completed")

users, err := eorm.Table("users").
    Select("*").
    WhereIn("id", activeUsersSub).
    Find()
// SQL: SELECT * FROM users WHERE id IN (SELECT DISTINCT user_id FROM orders WHERE status = ?)

// 查询没有被禁用的用户的订单
bannedUsersSub := eorm.NewSubquery().
    Table("users").
    Select("id").
    Where("status = ?", "banned")

orders, err := eorm.Table("orders").
    WhereNotIn("user_id", bannedUsersSub).
    Find()
// SQL: SELECT * FROM orders WHERE user_id NOT IN (SELECT id FROM users WHERE status = ?)
```

#### FROM 子查询
```go
func (qb *QueryBuilder) TableSubquery(sub *Subquery, alias string) *QueryBuilder
```
使用子查询作为 FROM 数据源（派生表）。

**示例：**
```go
// 从聚合子查询中查询
userTotalsSub := eorm.NewSubquery().
    Table("orders").
    Select("user_id, SUM(total) as total_spent")

records, err := (&eorm.QueryBuilder{}).
    TableSubquery(userTotalsSub, "user_totals").
    Select("user_id, total_spent").
    Where("total_spent > ?", 1000).
    Find()
// SQL: SELECT user_id, total_spent FROM (SELECT user_id, SUM(total) as total_spent FROM orders) AS user_totals WHERE total_spent > ?
```

#### SELECT 子查询
```go
func (qb *QueryBuilder) SelectSubquery(sub *Subquery, alias string) *QueryBuilder
```
在 SELECT 子句中添加子查询作为字段。

**示例：**
```go
// 为每个用户添加订单数量字段
orderCountSub := eorm.NewSubquery().
    Table("orders").
    Select("COUNT(*)").
    Where("orders.user_id = users.id")

users, err := eorm.Table("users").
    Select("users.id, users.name").
    SelectSubquery(orderCountSub, "order_count").
    Find()
// SQL: SELECT users.id, users.name, (SELECT COUNT(*) FROM orders WHERE orders.user_id = users.id) AS order_count FROM users
```

---

## 事务操作

### Transaction
```go
func Transaction(fn func(*Tx) error) error
func (db *DB) Transaction(fn func(*Tx) error) error
```
执行事务。

**示例：**
```go
err := eorm.Transaction(func(tx *eorm.Tx) error {
    // 插入用户
    user := eorm.NewRecord()
    user.Set("name", "张三")
    user.Set("email", "zhangsan@example.com")
    
    userID, err := tx.InsertRecord("users", user)
    if err != nil {
        return err
    }
    
    // 插入订单
    order := eorm.NewRecord()
    order.Set("user_id", userID)
    order.Set("total", 100.00)
    
    _, err = tx.InsertRecord("orders", order)
    return err
})
```

### BeginTransaction
```go
func BeginTransaction() (*Tx, error)
```
手动开始事务。

**示例：**
```go
tx, err := eorm.BeginTransaction()
if err != nil {
    log.Fatal(err)
}

defer func() {
    if r := recover(); r != nil {
        tx.Rollback()
        panic(r)
    }
}()

// 执行操作
_, err = tx.InsertRecord("users", userRecord)
if err != nil {
    tx.Rollback()
    return err
}

// 提交事务
err = tx.Commit()
```

### WithTransaction
```go
func WithTransaction(fn func(*Tx) error) error
```
事务包装器（Transaction 的别名）。

---

## 批量操作

### BatchInsertRecord
```go
func BatchInsertRecord(table string, records []*Record, batchSize ...int) (int64, error)
func (db *DB) BatchInsertRecord(table string, records []*Record, batchSize ...int) (int64, error)
func (tx *Tx) BatchInsertRecord(table string, records []*Record, batchSize ...int) (int64, error)
```
批量插入记录。

**示例：**
```go
var records []*eorm.Record
for i := 0; i < 1000; i++ {
    record := eorm.NewRecord()
    record.Set("name", fmt.Sprintf("用户%d", i))
    record.Set("age", 20+i%50)
    records = append(records, record)
}

// 使用默认批次大小 (100)
affected, err := eorm.BatchInsertRecord("users", records)

// 指定批次大小
affected, err := eorm.BatchInsertRecord("users", records, 50)
```

### BatchUpdateRecord
```go
func BatchUpdateRecord(table string, records []*Record, batchSize ...int) (int64, error)
func (db *DB) BatchUpdateRecord(table string, records []*Record, batchSize ...int) (int64, error)
func (tx *Tx) BatchUpdateRecord(table string, records []*Record, batchSize ...int) (int64, error)
```
批量更新记录（根据主键）。

**示例：**
```go
var records []*eorm.Record
for i := 1; i <= 100; i++ {
    record := eorm.NewRecord()
    record.Set("id", i)
    record.Set("status", "updated")
    records = append(records, record)
}

affected, err := eorm.BatchUpdateRecord("users", records, 20)
```

### BatchDeleteRecord
```go
func BatchDeleteRecord(table string, records []*Record, batchSize ...int) (int64, error)
func (db *DB) BatchDeleteRecord(table string, records []*Record, batchSize ...int) (int64, error)
func (tx *Tx) BatchDeleteRecord(table string, records []*Record, batchSize ...int) (int64, error)
```
批量删除记录（根据主键）。

**示例：**
```go
var records []*eorm.Record
for i := 1; i <= 50; i++ {
    record := eorm.NewRecord()
    record.Set("id", i)
    records = append(records, record)
}

affected, err := eorm.BatchDeleteRecord("users", records)
```

### BatchDeleteByIds
```go
func BatchDeleteByIds(table string, ids []interface{}, batchSize ...int) (int64, error)
func (db *DB) BatchDeleteByIds(table string, ids []interface{}, batchSize ...int) (int64, error)
func (tx *Tx) BatchDeleteByIds(table string, ids []interface{}, batchSize ...int) (int64, error)
```
根据 ID 批量删除记录。

**示例：**
```go
ids := []interface{}{1, 2, 3, 4, 5}
affected, err := eorm.BatchDeleteByIds("users", ids)
```

---

## 分页查询

### Paginate
```go
func Paginate(page int, pageSize int, querySQL string, args ...interface{}) (*Page[*Record], error)
func (db *DB) Paginate(page int, pageSize int, querySQL string, args ...interface{}) (*Page[*Record], error)
func (tx *Tx) Paginate(page int, pageSize int, querySQL string, args ...interface{}) (*Page[*Record], error)
```
使用完整 SQL 进行分页查询。

**功能支持：** ❌ 自动时间戳 | ❌ 乐观锁 | ❌ 软删除

**示例：**
```go
pageResult, err := eorm.Paginate(1, 10, "SELECT * FROM users WHERE age > ?", 18)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("当前页: %d/%d\n", pageResult.PageNumber, pageResult.TotalPage)
fmt.Printf("总记录数: %d\n", pageResult.TotalRow)

for _, record := range pageResult.List {
    fmt.Printf("用户: %s, 年龄: %d\n", 
        record.GetString("name"), 
        record.GetInt("age"))
}
```

### PaginateBuilder
```go
func PaginateBuilder(page int, pageSize int, selectSql string, table string, whereSql string, orderBySql string, args ...interface{}) (*Page[*Record], error)
func (db *DB) PaginateBuilder(page int, pageSize int, selectSql string, table string, whereSql string, orderBySql string, args ...interface{}) (*Page[*Record], error)
func (tx *Tx) PaginateBuilder(page int, pageSize int, selectSql string, table string, whereSql string, orderBySql string, args ...interface{}) (*Page[*Record], error)
```
使用构建器方式进行分页查询。

**功能支持：** ❌ 自动时间戳 | ❌ 乐观锁 | ❌ 软删除

**示例：**
```go
pageResult, err := eorm.PaginateBuilder(
    1, 10,                    // 第1页，每页10条
    "id, name, age",          // SELECT 字段
    "users",                  // 表名
    "age > ? AND status = ?", // WHERE 条件
    "age DESC",               // ORDER BY
    18, "active",             // WHERE 参数
)
```

---

## 缓存功能

### 缓存配置和初始化

eorm同时支持本机缓存和Redis缓存,程序启动后默认已开启本地缓存

#### NewRedisCache
```go
func NewRedisCache()
```
初始化 Redis 缓存实例。

**示例：**
```go
// 假设你有一个 Redis 缓存
	rc, err := redis.NewRedisCache("192.168.10.205:6379", "redisuser", "123456", 2)
	if err != nil {
		fmt.Printf("Redis 连接失败: %v\n", err)
		return
	}
	eorm.SetDefaultCache(rc) // 把redis设置为默认缓存
```

#### CreateCacheRepository
```go
func CreateCacheRepository(cacheRepositoryName string, ttl time.Duration)
```
创建缓存存储库,并指定过期时长。

**示例：**
```go
// 为用户缓存设置默认 TTL 为 30 分钟
eorm.CreateCacheRepository("user_cache", 30*time.Minute)

// 为产品缓存设置默认 TTL 为 1 小时
eorm.CreateCacheRepository("product_cache", time.Hour)
```

### 默认缓存操作(默认是 内存缓存,可通过SetDefaultCache切换为Redis

#### CacheSet
```go
func CacheSet(cacheRepositoryName, key string, value interface{}, ttl ...time.Duration)
```
在指定缓存存储库中存储值。最后的ttl参数可不填, 会自动根据CreateCacheRepository的ttl设置时长.

**示例：**
```go
// 使用默认 TTL
eorm.CacheSet("user_cache", "user:1", userObj)

// 指定 TTL
eorm.CacheSet("user_cache", "user:1", userObj, 10*time.Minute)
```

#### CacheGet
```go
func CacheGet(cacheRepositoryName, key string) (interface{}, bool)
```
从指定缓存存储库中获取值。

**示例：**
```go
if value, ok := eorm.CacheGet("user_cache", "user:1"); ok {
    user := value.(User)
    fmt.Printf("从缓存获取用户: %s\n", user.Name)
} else {
    fmt.Println("缓存中未找到用户")
}
```

#### CacheDelete
```go
func CacheDelete(cacheRepositoryName, key string)
```
从指定缓存存储库中删除指定键。

**示例：**
```go
// 删除特定用户的缓存
eorm.CacheDelete("user_cache", "user:1")
```

#### CacheClearRepository
```go
func CacheClearRepository(cacheRepositoryName string)
```
清空指定缓存存储库中的所有键。

**示例：**
```go
// 清空用户缓存存储库
eorm.CacheClearRepository("user_cache")
```

### 指定操作本机缓存

#### LocalCacheSet
```go
func LocalCacheSet(cacheRepositoryName, key string, value interface{}, ttl ...time.Duration)
```
在本地缓存中存储值。

**示例：**
```go
eorm.LocalCacheSet("session_cache", "session:abc123", sessionData, 30*time.Minute)
```

#### LocalCacheGet
```go
func LocalCacheGet(cacheRepositoryName, key string) (interface{}, bool)
```
从本地缓存中获取值。

**示例：**
```go
if session, ok := eorm.LocalCacheGet("session_cache", "session:abc123"); ok {
    fmt.Println("找到会话数据")
}
```

#### LocalCacheDelete
```go
func LocalCacheDelete(cacheRepositoryName, key string)
```
从本地缓存中删除指定键。

**示例：**
```go
eorm.LocalCacheDelete("session_cache", "session:abc123")
```

#### LocalCacheClearRepository
```go
func LocalCacheClearRepository(cacheRepositoryName string)
```
清空本地缓存中的指定存储库。

**示例：**
```go
eorm.LocalCacheClearRepository("session_cache")
```

#### LocalCacheClearAll
```go
func LocalCacheClearAll()
```
清空本地缓存中的所有存储库。

**示例：**
```go
// 清空所有本地缓存
eorm.LocalCacheClearAll()
```

### 指定操作Redis 缓存

#### RedisCacheSet
```go
func RedisCacheSet(cacheRepositoryName, key string, value interface{}, ttl ...time.Duration) error
```
在 Redis 缓存中存储值。

**示例：**
```go
err := eorm.RedisCacheSet("user_cache", "user:1", userObj, 15*time.Minute)
if err != nil {
    log.Printf("Redis 缓存设置失败: %v", err)
}
```

#### RedisCacheGet
```go
func RedisCacheGet(cacheRepositoryName, key string) (interface{}, bool, error)
```
从 Redis 缓存中获取值。

**示例：**
```go
if value, ok, err := eorm.RedisCacheGet("user_cache", "user:1"); err != nil {
    log.Printf("Redis 缓存获取失败: %v", err)
} else if ok {
    user := value.(User)
    fmt.Printf("从 Redis 获取用户: %s\n", user.Name)
}
```

#### RedisCacheDelete
```go
func RedisCacheDelete(cacheRepositoryName, key string) error
```
从 Redis 缓存中删除指定键。

**示例：**
```go
err := eorm.RedisCacheDelete("user_cache", "user:1")
if err != nil {
    log.Printf("Redis 缓存删除失败: %v", err)
}
```

#### RedisCacheClearRepository
```go
func RedisCacheClearRepository(cacheRepositoryName string) error
```
清空 Redis 缓存中的指定存储库。

**示例：**
```go
err := eorm.RedisCacheClearRepository("user_cache")
if err != nil {
    log.Printf("Redis 缓存清空失败: %v", err)
}
```

#### RedisCacheClearAll
```go
func RedisCacheClearAll() error
```
清空 Redis 缓存中的所有 eorm 相关缓存。

**示例：**
```go
err := eorm.RedisCacheClearAll()
if err != nil {
    log.Printf("Redis 缓存全部清空失败: %v", err)
}
```

### 查询缓存

### Cache
```go
func Cache(cacheRepositoryName string, ttl ...time.Duration) *DB
func (db *DB) Cache(cacheRepositoryName string, ttl ...time.Duration) *DB
func (tx *Tx) Cache(cacheRepositoryName string, ttl ...time.Duration) *DB
```
使用默认缓存。

**示例：**
```go
// 使用默认 TTL
records, err := eorm.Cache("user_cache").Query("SELECT * FROM users")

// 指定 TTL
records, err := eorm.Cache("user_cache", 5*time.Minute).Query("SELECT * FROM users")

// 链式调用
records, err := eorm.Cache("user_cache").
    Table("users").
    Where("age > ?", 18).
    Find()
```

### LocalCache
```go
func (db *DB) LocalCache(cacheRepositoryName string, ttl ...time.Duration) *DB
func (tx *Tx) LocalCache(cacheRepositoryName string, ttl ...time.Duration) *DB
```
使用本地内存缓存。

**示例：**
```go
records, err := db.LocalCache("user_cache", 10*time.Minute).
    Query("SELECT * FROM users WHERE status = ?", "active")
```

### RedisCache
```go
func (db *DB) RedisCache(cacheRepositoryName string, ttl ...time.Duration) *DB
func (tx *Tx) RedisCache(cacheRepositoryName string, ttl ...time.Duration) *DB
```
使用 Redis 缓存。

**示例：**
```go
records, err := db.RedisCache("user_cache", 30*time.Minute).
    Query("SELECT * FROM users WHERE vip = ?", true)
```

### WithCountCache
```go
func WithCountCache(ttl time.Duration) *DB
func (db *DB) WithCountCache(ttl time.Duration) *DB
func (tx *Tx) WithCountCache(ttl time.Duration) *DB
```
启用分页计数缓存。

**示例：**
```go
// 缓存 COUNT 查询结果 5 分钟
pageResult, err := eorm.Cache("user_cache").
    WithCountCache(5*time.Minute).
    Paginate(1, 10, "SELECT * FROM users WHERE age > ?", 18)
```

### 缓存使用完整示例

```go
package main

import (
    "fmt"
    "log"
    "time"
    
    "github.com/zzguang83325/eorm"
    _ "github.com/go-sql-driver/mysql"
)

func main() {
    // 1. 初始化数据库
    db, err := eorm.OpenDatabase(eorm.MySQL, "root:password@tcp(localhost:3306)/test", 10)
    if err != nil {
        log.Fatal(err)
    }
    
    // 2. 初始化缓存
    eorm.InitLocalCache(10 * time.Minute) // 每10分钟清理过期缓存
    
    // 3. 创建缓存存储库配置
    eorm.CreateCache("user_cache", 30*time.Minute)    // 用户缓存30分钟
    eorm.CreateCache("product_cache", time.Hour)      // 产品缓存1小时
    
    // 4. 手动缓存操作
    user := User{ID: 1, Name: "张三", Age: 25}
    
    // 存储到缓存
    eorm.CacheSet("user_cache", "user:1", user)
    
    // 从缓存获取
    if cachedUser, ok := eorm.CacheGet("user_cache", "user:1"); ok {
        fmt.Printf("从缓存获取用户: %+v\n", cachedUser)
    }
    
    // 5. 查询缓存集成
    // 第一次查询（从数据库，并缓存结果）
    records, err := eorm.Cache("user_cache", 15*time.Minute).
        Query("SELECT * FROM users WHERE age > ?", 18)
    
    // 第二次查询（从缓存获取）
    records, err = eorm.Cache("user_cache").
        Query("SELECT * FROM users WHERE age > ?", 18)
    
    // 6. 链式查询缓存
    pageResult, err := eorm.Cache("user_cache").
        WithCountCache(5*time.Minute).
        Table("users").
        Where("status = ?", "active").
        Paginate(1, 10)
    
    // 7. 缓存清理
    eorm.CacheDelete("user_cache", "user:1")           // 删除特定键
    eorm.CacheClearRepository("user_cache")            // 清空整个存储库
    
    // 8. 本地缓存操作
    eorm.LocalCacheSet("session_cache", "session:abc", sessionData, 30*time.Minute)
    if session, ok := eorm.LocalCacheGet("session_cache", "session:abc"); ok {
        fmt.Println("会话数据:", session)
    }
    
    // 9. Redis 缓存操作（如果配置了 Redis）
    err = eorm.RedisCacheSet("distributed_cache", "key1", "value1", time.Hour)
    if err != nil {
        log.Printf("Redis 缓存失败: %v", err)
    }
    
    if value, ok, err := eorm.RedisCacheGet("distributed_cache", "key1"); err == nil && ok {
        fmt.Println("Redis 缓存值:", value)
    }
}
```

---

## 增强功能

EORM 框架提供三大增强功能：**自动时间戳**、**乐观锁**、**软删除**。不同的函数对这些功能的支持程度不同。

### 增强功能详细说明

#### 🕒 自动时间戳功能

**支持的操作：**
- **INSERT 操作**：自动设置 `created_at` 字段为当前时间
- **UPDATE 操作**：自动设置 `updated_at` 字段为当前时间
- **UPSERT 操作**：INSERT 时设置 `created_at`，UPDATE 时设置 `updated_at`

**支持的函数：**
```go
// ✅ 完全支持
eorm.InsertRecord("users", record)     // 自动设置 created_at
eorm.UpdateRecord("users", record)     // 自动设置 updated_at  
eorm.Update("users", record, "id = ?", 1) // 自动设置 updated_at
eorm.SaveRecord("users", record)       // INSERT 时设置 created_at，UPDATE 时设置 updated_at

// ✅ DbModel 方法
eorm.InsertDbModel(user)               // 自动设置 created_at
eorm.UpdateDbModel(user)               // 自动设置 updated_at
eorm.SaveDbModel(user)                 // 根据操作类型设置相应时间戳

// ✅ 链式查询
eorm.Table("users").Where("id = ?", 1).Update(record) // 自动设置 updated_at

// ✅ 批量操作
eorm.BatchInsertRecord("users", records) // 为每条记录设置 created_at
eorm.BatchUpdateRecord("users", records) // 为每条记录设置 updated_at

// ❌ 不支持
eorm.Query("INSERT INTO users ...")    // 原始 SQL，不处理时间戳
eorm.UpdateFast("users", record, "id = ?", 1) // 明确跳过时间戳检查
```

**配置示例：**
```go
// 启用时间戳功能
eorm.EnableTimestamps()

// 配置表的时间戳字段
eorm.ConfigTimestampsWithFields("users", "created_at", "updated_at")
eorm.ConfigTimestampsWithFields("orders", "create_time", "update_time")
```

#### 🔒 乐观锁功能

**支持的操作：**
- **UPDATE 操作**：检查版本号，更新时自动递增版本号
- **INSERT 操作**：自动初始化版本号为 1
- **UPSERT 操作**：INSERT 时初始化版本号，UPDATE 时检查并递增版本号

**支持的函数：**
```go
// ✅ 完全支持
record := eorm.NewRecord()
record.Set("id", 1)
record.Set("name", "更新后的名称")
record.Set("version", 5) // 当前版本号

eorm.UpdateRecord("users", record)     // 检查版本号，成功后递增为 6
eorm.Update("users", record, "id = ?", 1) // 检查版本号，成功后递增为 6
eorm.SaveRecord("users", record)       // 根据操作检查或初始化版本号

// ✅ DbModel 方法
user.Version = 5
eorm.UpdateDbModel(user)               // 检查版本号，成功后递增为 6

// ✅ 链式查询
eorm.Table("users").Where("id = ?", 1).Update(record) // 检查版本号

// ❌ 不支持
eorm.UpdateFast("users", record, "id = ?", 1) // 明确跳过乐观锁检查
```

**版本冲突处理：**
```go
// 当版本号不匹配时，会返回 ErrVersionMismatch 错误
affected, err := eorm.UpdateRecord("users", record)
if errors.Is(err, eorm.ErrVersionMismatch) {
    fmt.Println("记录已被其他用户修改，请重新获取最新数据")
}
```

**配置示例：**
```go
// 启用乐观锁功能
eorm.EnableOptimisticLock()

// 配置表的版本字段
eorm.ConfigOptimisticLockWithField("users", "version")
eorm.ConfigOptimisticLockWithField("products", "revision")
```

#### 🗑️ 软删除功能

**支持的操作：**
- **DELETE 操作**：不物理删除记录，而是设置删除标记字段
- **查询操作**：自动过滤已软删除的记录
- **统计操作**：自动排除已软删除的记录

**支持的函数：**
```go
// ✅ 删除操作（软删除）
eorm.Delete("users", "id = ?", 1)      // 设置 deleted_at 字段
eorm.DeleteRecord("users", record)     // 设置 deleted_at 字段
eorm.DeleteDbModel(user)               // 设置 deleted_at 字段

// ✅ 链式删除
eorm.Table("users").Where("status = ?", "inactive").Delete() // 软删除

// ✅ 批量删除
eorm.BatchDeleteRecord("users", records) // 批量软删除
eorm.BatchDeleteByIds("users", ids)      // 批量软删除

// ✅ 查询时自动过滤
eorm.Table("users").Find()              // 自动排除软删除记录
eorm.Table("users").Count()             // 统计时排除软删除记录
eorm.Table("users").Exists()            // 检查时排除软删除记录
eorm.Table("users").Paginate(1, 10)     // 分页时排除软删除记录

// ✅ 特殊查询
eorm.QueryWithOutTrashed("SELECT * FROM users") // 明确过滤软删除
eorm.QueryFirstWithOutTrashed("SELECT * FROM users WHERE id = ?", 1)

// ✅ 包含软删除记录的查询
eorm.Table("users").WithTrashed().Find() // 包含软删除记录
eorm.Table("users").OnlyTrashed().Find() // 只查询软删除记录
eorm.Table("users").WithTrashed().Paginate(1, 10) // 分页包含软删除记录

// ❌ 不支持（原始 SQL）
eorm.Query("SELECT * FROM users")       // 不会自动过滤软删除
eorm.QueryFirst("SELECT * FROM users WHERE id = ?", 1) // 不会自动过滤
eorm.Paginate(1, 10, "SELECT * FROM users") // 原始SQL分页，不会自动过滤
eorm.PaginateBuilder(1, 10, "*", "users", "", "") // 不会自动过滤
```

**软删除类型：**
```go
// 时间戳类型（默认）
eorm.ConfigSoftDeleteWithType("users", "deleted_at", eorm.SoftDeleteTimestamp)

// 布尔类型
eorm.ConfigSoftDeleteWithType("products", "is_deleted", eorm.SoftDeleteBool)
```

**配置示例：**
```go
// 启用软删除功能
eorm.EnableSoftDelete()

// 配置表的软删除字段
eorm.ConfigSoftDelete("users", "deleted_at")           // 时间戳类型
eorm.ConfigSoftDelete("products", "is_deleted")        // 默认时间戳类型
eorm.ConfigSoftDeleteWithType("orders", "is_deleted", eorm.SoftDeleteBool) // 布尔类型
```

### 多数据库模式下的增强功能支持

每个数据库可以独立配置和启用功能：

```go
// 为不同数据库启用不同功能
eorm.Use("user_db").EnableTimestamps().ConfigTimestamps("users")
eorm.Use("product_db").EnableOptimisticLock().ConfigOptimisticLock("products")
eorm.Use("log_db").EnableSoftDelete().ConfigSoftDelete("logs")

// 使用时自动应用对应数据库的功能
eorm.Use("user_db").UpdateRecord("users", userRecord)     // 支持时间戳
eorm.Use("product_db").UpdateRecord("products", productRecord) // 支持乐观锁
eorm.Use("log_db").Delete("logs", "level = ?", "debug")   // 支持软删除
```

### 性能考虑

1. **功能检查开销**：启用功能会增加少量性能开销
2. **批量操作优化**：批量操作函数已优化功能处理，性能优于循环调用单条操作
3. **软删除查询**：软删除会在查询中添加 WHERE 条件，可能影响查询性能，建议在删除字段上建立索引

### 最佳实践

1. **统一启用**：建议在应用启动时统一启用需要的功能
2. **表级配置**：为每个需要功能的表单独配置字段名
3. **版本号处理**：更新记录前先查询获取最新版本号
4. **软删除索引**：为软删除字段建立数据库索引提高查询性能
5. **功能组合**：三大功能可以同时使用，互不冲突

---

### 增强功能配置

### 自动时间戳

#### EnableTimestamps
```go
func EnableTimestamps()
func (db *DB) EnableTimestamps() *DB
```
启用自动时间戳功能。

**示例：**
```go
// 全局启用
eorm.EnableTimestamps()

// 为特定数据库启用
eorm.Use("db1").EnableTimestamps()
```

#### ConfigTimestamps
```go
func ConfigTimestamps(table string)
func (db *DB) ConfigTimestamps(table string) *DB
```
为表配置默认时间戳字段（created_at, updated_at）。

**示例：**
```go
eorm.ConfigTimestamps("users")
```

#### ConfigTimestampsWithFields
```go
func ConfigTimestampsWithFields(table, createdAtField, updatedAtField string)
func (db *DB) ConfigTimestampsWithFields(table, createdAtField, updatedAtField string) *DB
```
为表配置自定义时间戳字段。

**示例：**
```go
eorm.ConfigTimestampsWithFields("users", "create_time", "update_time")
```

#### ConfigCreatedAt
```go
func ConfigCreatedAt(table, field string)
func (db *DB) ConfigCreatedAt(table, field string) *DB
```
只配置创建时间字段。

**示例：**
```go
eorm.ConfigCreatedAt("logs", "created_at")
```

#### ConfigUpdatedAt
```go
func ConfigUpdatedAt(table, field string)
func (db *DB) ConfigUpdatedAt(table, field string) *DB
```
只配置更新时间字段。

**示例：**
```go
eorm.ConfigUpdatedAt("users", "last_modified")
```

#### RemoveTimestamps
```go
func RemoveTimestamps(table string)
func (db *DB) RemoveTimestamps(table string) *DB
```
移除表的时间戳配置。

#### HasTimestamps
```go
func HasTimestamps(table string) bool
func (db *DB) HasTimestamps(table string) bool
```
检查表是否配置了时间戳。

### 乐观锁

#### EnableOptimisticLock
```go
func EnableOptimisticLock()
func (db *DB) EnableOptimisticLock() *DB
```
启用乐观锁功能。

**示例：**
```go
eorm.EnableOptimisticLock()
```

#### ConfigOptimisticLock
```go
func ConfigOptimisticLock(table string)
func (db *DB) ConfigOptimisticLock(table string) *DB
```
为表配置默认版本字段（version）。

**示例：**
```go
eorm.ConfigOptimisticLock("products")
```

#### ConfigOptimisticLockWithField
```go
func ConfigOptimisticLockWithField(table, versionField string)
func (db *DB) ConfigOptimisticLockWithField(table, versionField string) *DB
```
为表配置自定义版本字段。

**示例：**
```go
eorm.ConfigOptimisticLockWithField("products", "revision")
```

#### RemoveOptimisticLock
```go
func RemoveOptimisticLock(table string)
func (db *DB) RemoveOptimisticLock(table string) *DB
```
移除表的乐观锁配置。

#### HasOptimisticLock
```go
func HasOptimisticLock(table string) bool
func (db *DB) HasOptimisticLock(table string) bool
```
检查表是否配置了乐观锁。

### 软删除

#### EnableSoftDelete
```go
func EnableSoftDelete()
func (db *DB) EnableSoftDelete() *DB
```
启用软删除功能。

**示例：**
```go
eorm.EnableSoftDelete()
```

#### ConfigSoftDelete
```go
func ConfigSoftDelete(table string, field ...string)
func (db *DB) ConfigSoftDelete(table string, field ...string) *DB
```
为表配置软删除字段。

**示例：**
```go
// 使用默认字段 deleted_at
eorm.ConfigSoftDelete("users")

// 使用自定义字段
eorm.ConfigSoftDelete("users", "is_deleted")
```

#### ConfigSoftDeleteWithType
```go
func ConfigSoftDeleteWithType(table, field string, deleteType SoftDeleteType)
func (db *DB) ConfigSoftDeleteWithType(table, field string, deleteType SoftDeleteType) *DB
```
为表配置指定类型的软删除。

**示例：**
```go
// 时间戳类型软删除
eorm.ConfigSoftDeleteWithType("users", "deleted_at", eorm.SoftDeleteTimestamp)

// 布尔类型软删除
eorm.ConfigSoftDeleteWithType("products", "is_deleted", eorm.SoftDeleteBool)
```

#### RemoveSoftDelete
```go
func RemoveSoftDelete(table string)
func (db *DB) RemoveSoftDelete(table string) *DB
```
移除表的软删除配置。

#### HasSoftDelete
```go
func HasSoftDelete(table string) bool
func (db *DB) HasSoftDelete(table string) bool
```
检查表是否配置了软删除。

---

## SQL 模板

eorm 提供了强大的 SQL 模板功能，允许您将 SQL 语句配置化管理，支持动态参数、条件构建和多数据库执行。

### 配置文件结构

SQL 模板使用 JSON 格式的配置文件。以下是一个完整的配置文件格式模板：

#### 完整 JSON 格式模板

```json
{
  "version": "1.0",
  "description": "服务SQL配置文件描述",
  "namespace": "service_name",
  "sqls": [
    {
      "name": "sqlName",
      "description": "SQL语句描述",
      "sql": "SELECT * FROM table WHERE condition = :param",
      "type": "select",
      "order": "created_at DESC",
      "inparam": [
        {
          "name": "paramName",
          "type": "string",
          "desc": "参数描述",
          "sql": " AND column = :paramName"
        }
      ]
    }
  ]
}
```

#### 字段说明

**根级别字段：**
- `version` (string, 必需): 配置文件版本号
- `description` (string, 可选): 配置文件描述
- `namespace` (string, 可选): 命名空间，用于避免 SQL 名称冲突
- `sqls` (array, 必需): SQL 语句配置数组

**SQL 配置字段：**
- `name` (string, 必需): SQL 语句唯一标识符
- `description` (string, 可选): SQL 语句描述
- `sql` (string, 必需): SQL 语句模板
- `type` (string, 可选): SQL 类型 (`select`, `insert`, `update`, `delete`)
- `order` (string, 可选): 默认排序条件
- `inparam` (array, 可选): 输入参数定义（用于动态 SQL）

**输入参数字段 (inparam)：**
- `name` (string, 必需): 参数名称
- `type` (string, 必需): 参数类型
- `desc` (string, 可选): 参数描述
- `sql` (string, 必需): 当参数存在时追加的 SQL 片段

#### 实际配置示例

```json
{
  "version": "1.0",
  "description": "用户服务SQL配置",
  "namespace": "user_service",
  "sqls": [
    {
      "name": "findById",
      "description": "根据ID查找用户",
      "sql": "SELECT * FROM users WHERE id = :id",
      "type": "select"
    },
    {
      "name": "findUsers",
      "description": "动态查询用户列表",
      "sql": "SELECT * FROM users WHERE 1=1",
      "type": "select",
      "order": "created_at DESC",
      "inparam": [
        {
          "name": "status",
          "type": "int",
          "desc": "用户状态",
          "sql": " AND status = :status"
        },
        {
          "name": "name",
          "type": "string",
          "desc": "用户名模糊查询",
          "sql": " AND name LIKE CONCAT('%', :name, '%')"
        }
      ]
    }
  ]
}
```

### 参数类型支持

eorm SQL 模板支持多种参数传递方式，提供灵活的使用体验：

#### 支持的参数类型

| 参数类型 | 适用场景 | SQL 占位符 | 示例 |
|---------|---------|-----------|------|
| `map[string]interface{}` | 命名参数 | `:name` | `map[string]interface{}{"id": 123}` |
| `[]interface{}` | 多个位置参数 | `?` | `[]interface{}{123, "John"}` |
| **单个简单类型** | 单个位置参数 | `?` | `123`, `"John"`, `true` |
| **可变参数** | 多个位置参数 | `?` | `SqlTemplate(name, 123, "John", true)` |

#### 单个简单类型支持

支持直接传递简单类型参数，无需包装成 map 或 slice：

- `string` - 字符串
- `int`, `int8`, `int16`, `int32`, `int64` - 整数类型
- `uint`, `uint8`, `uint16`, `uint32`, `uint64` - 无符号整数
- `float32`, `float64` - 浮点数
- `bool` - 布尔值

#### 可变参数支持

支持 Go 风格的可变参数 (`...interface{}`)，提供最自然的参数传递方式：

```go
// 可变参数方式 - 最直观和简洁
records, err := eorm.SqlTemplate("findByIdAndStatus", 123, 1).Query()
records, err := eorm.SqlTemplate("updateUser", "John", "john@example.com", 25, 123).Exec()
records, err := eorm.SqlTemplate("findByAgeRange", 18, 65, 1).Query()
```

#### 参数匹配规则

| SQL 占位符 | 参数类型 | 结果 |
|-----------|---------|------|
| 单个 `?` | 单个简单类型 | ✅ 支持 |
| 单个 `?` | `map[string]interface{}` | ✅ 支持（向后兼容） |
| 单个 `?` | `[]interface{}{value}` | ✅ 支持（向后兼容） |
| 多个 `?` | `[]interface{}{v1, v2, ...}` | ✅ 支持 |
| 多个 `?` | **可变参数 `v1, v2, ...`** | ✅ 支持 |
| 多个 `?` | 单个简单类型 | ❌ 错误提示 |
| `:name` | `map[string]interface{}{"name": value}` | ✅ 支持 |
| `:name` | 单个简单类型 | ❌ 错误提示 |
| `:name` | 可变参数 | ❌ 错误提示 |

#### 参数数量验证

系统会自动验证参数数量与 SQL 占位符数量是否匹配：

```go
// 正确：2个参数匹配2个占位符
records, err := eorm.SqlTemplate("findByIdAndStatus", 123, 1).Query()

// 错误：参数不足
records, err := eorm.SqlTemplate("findByIdAndStatus", 123).Query()
// 返回错误: parameter count mismatch: SQL has 2 '?' placeholders but got 1 parameters

// 错误：参数过多  
records, err := eorm.SqlTemplate("findByIdAndStatus", 123, 1, 2).Query()
// 返回错误: parameter count mismatch: SQL has 2 '?' placeholders but got 3 parameters
```

#### 使用示例

```go
// 1. 单个简单参数（推荐用于单参数查询）
records, err := eorm.SqlTemplate("user_service.findById", 123).Query()
records, err := eorm.SqlTemplate("user_service.findByEmail", "user@example.com").Query()
records, err := eorm.SqlTemplate("user_service.findActive", true).Query()

// 2. 可变参数（推荐用于多参数查询）
records, err := eorm.SqlTemplate("user_service.findByIdAndStatus", 123, 1).Query()
records, err := eorm.SqlTemplate("user_service.updateUser", "John", "john@example.com", 25, 123).Exec()
records, err := eorm.SqlTemplate("user_service.findByAgeRange", 18, 65, 1).Query()

// 3. 命名参数（适用于复杂查询）
params := map[string]interface{}{
    "status": 1,
    "name": "John",
    "ageMin": 18,
}
records, err := eorm.SqlTemplate("user_service.findUsers", params).Query()

// 4. 位置参数（向后兼容）
records, err := eorm.SqlTemplate("user_service.findByIdAndStatus", 
    []interface{}{123, 1}).Query()
```

### 配置加载

#### LoadSqlConfig
```go
func LoadSqlConfig(configPath string) error
```
加载单个 SQL 配置文件。

**示例：**
```go
err := eorm.LoadSqlConfig("config/user_service.json")
```

#### LoadSqlConfigs
```go
func LoadSqlConfigs(configPaths []string) error
```
批量加载多个 SQL 配置文件。

**示例：**
```go
configPaths := []string{
    "config/user_service.json",
    "config/order_service.json",
}
err := eorm.LoadSqlConfigs(configPaths)
```

#### LoadSqlConfigDir
```go
func LoadSqlConfigDir(dirPath string) error
```
加载指定目录下的所有 JSON 配置文件。

**示例：**
```go
err := eorm.LoadSqlConfigDir("config/")
```

#### ReloadSqlConfig
```go
func ReloadSqlConfig(configPath string) error
```
重新加载指定的配置文件。

#### ReloadAllSqlConfigs
```go
func ReloadAllSqlConfigs() error
```
重新加载所有已加载的配置文件。

### 配置信息查询

#### GetSqlConfigInfo
```go
func GetSqlConfigInfo() []ConfigInfo
```
获取所有已加载配置文件的信息。

**ConfigInfo 结构体：**
```go
type ConfigInfo struct {
    FilePath    string `json:"filePath"`
    Namespace   string `json:"namespace"`
    Description string `json:"description"`
    SqlCount    int    `json:"sqlCount"`
}
```

#### ListSqlItems
```go
func ListSqlItems() map[string]*SqlItem
```
列出所有可用的 SQL 模板项。

### SQL 模板执行

#### SqlTemplate (全局)
```go
func SqlTemplate(name string, params ...interface{}) *SqlTemplateBuilder
```
创建 SQL 模板构建器，使用默认数据库连接。

**参数：**
- `name`: SQL 模板名称（支持命名空间，如 "user_service.findById"）
- `params`: 可变参数，支持以下类型：
  - `map[string]interface{}` - 命名参数（`:name`）
  - `[]interface{}` - 位置参数数组（`?`）
  - **单个简单类型** - 单个位置参数（`?`），支持 `string`、`int`、`float`、`bool` 等基本类型
  - **可变参数** - 多个位置参数（`?`），直接传递多个值

**示例：**
```go
// 使用命名参数
records, err := eorm.SqlTemplate("user_service.findById", 
    map[string]interface{}{"id": 123}).Query()

// 使用位置参数数组
records, err := eorm.SqlTemplate("user_service.findByIdAndStatus", 
    []interface{}{123, 1}).Query()

// 使用单个简单参数（推荐用于单参数查询）
records, err := eorm.SqlTemplate("user_service.findById", 123).Query()
records, err := eorm.SqlTemplate("user_service.findByEmail", "user@example.com").Query()

// 使用可变参数（推荐用于多参数查询）
records, err := eorm.SqlTemplate("user_service.findByIdAndStatus", 123, 1).Query()
records, err := eorm.SqlTemplate("user_service.updateUser", "John", "john@example.com", 25, 123).Exec()
records, err := eorm.SqlTemplate("user_service.findByAgeRange", 18, 65, 1).Query()
```

#### SqlTemplate (指定数据库)
```go
func (db *DB) SqlTemplate(name string, params ...interface{}) *SqlTemplateBuilder
```
在指定数据库上创建 SQL 模板构建器。

**示例：**
```go
// 传统方式
records, err := eorm.Use("mysql").SqlTemplate("user_service.findById", 
    map[string]interface{}{"id": 123}).Query()

// 单个简单参数（更简洁）
records, err := eorm.Use("mysql").SqlTemplate("user_service.findById", 123).Query()

// 可变参数（最简洁）
records, err := eorm.Use("mysql").SqlTemplate("user_service.findByIdAndStatus", 123, 1).Query()
```

#### SqlTemplate (事务)
```go
func (tx *Tx) SqlTemplate(name string, params ...interface{}) *SqlTemplateBuilder
```
在事务中使用 SQL 模板。

**示例：**
```go
err := eorm.Transaction(func(tx *eorm.Tx) error {
    // 使用可变参数
    result, err := tx.SqlTemplate("user_service.insertUser", "John", "john@example.com", 25).Exec()
    return err
})
```

### SqlTemplateBuilder 方法

#### Timeout
```go
func (b *SqlTemplateBuilder) Timeout(timeout time.Duration) *SqlTemplateBuilder
```
设置查询超时时间。

**示例：**
```go
records, err := eorm.SqlTemplate("user_service.findUsers", params).
    Timeout(30 * time.Second).Query()
```

#### WithCountCache
```go
func (b *SqlTemplateBuilder) WithCountCache(ttl time.Duration) *SqlTemplateBuilder
```
启用分页计数缓存。用于在分页查询时缓存 COUNT 查询结果，避免重复执行 COUNT 语句。

**参数：**
- `ttl`: 缓存时间，如果为 0 则不缓存，如果大于 0 则缓存指定时间

**示例：**
```go
// 启用计数缓存，缓存 5 分钟
pageObj, err := eorm.SqlTemplate("getUserList", params).
    Cache("user_cache").
    WithCountCache(5 * time.Minute).
    Paginate(1, 10)
```

#### Query
```go
func (b *SqlTemplateBuilder) Query() ([]*Record, error)
```
执行查询并返回多条记录。

#### QueryFirst
```go
func (b *SqlTemplateBuilder) QueryFirst() (*Record, error)
```
执行查询并返回第一条记录。

#### Exec
```go
func (b *SqlTemplateBuilder) Exec() (sql.Result, error)
```
执行 SQL 语句（INSERT、UPDATE、DELETE）。

#### Paginate
```go
func (b *SqlTemplateBuilder) Paginate(page int, pageSize int) (*Page[*Record], error)
```
执行 SQL 模板并返回分页结果。使用完整 SQL 语句进行分页查询，自动解析 SQL 并根据数据库类型生成相应的分页语句。

**参数：**
- `page`: 页码（从 1 开始）
- `pageSize`: 每页记录数

**返回：**
- `*Page[*Record]`: 分页结果对象
- `error`: 错误信息

**示例：**
```go
// 基本分页查询
pageObj, err := eorm.SqlTemplate("user_service.findActiveUsers", 1).
    Paginate(1, 10)

// 带参数的分页查询
pageObj, err := eorm.SqlTemplate("user_service.findByStatus", "active", 18).
    Paginate(2, 20)

// 带计数缓存的分页查询
pageObj, err := eorm.SqlTemplate("getUserList", params).
    Cache("user_cache").
    WithCountCache(5 * time.Minute).
    Paginate(1, 10)

// 在指定数据库上执行分页
pageObj, err := eorm.Use("mysql").SqlTemplate("findUsers", params).
    Paginate(1, 15)

// 事务中执行分页
err := eorm.Transaction(func(tx *eorm.Tx) error {
    pageObj, err := tx.SqlTemplate("findOrders", userId).Paginate(1, 10)
    // 处理分页结果...
    return err
})

// 带超时的分页查询
pageObj, err := eorm.SqlTemplate("complexQuery", params).
    Timeout(30 * time.Second).
    Paginate(1, 50)

// 访问分页结果
if err == nil {
    fmt.Printf("第%d页（共%d页），总条数: %d\n", 
        pageObj.PageNumber, pageObj.TotalPage, pageObj.TotalRow)
    
    for _, record := range pageObj.List {
        fmt.Printf("用户: %s, 年龄: %d\n", 
            record.Str("name"), record.Int("age"))
    }
}
```

### 动态 SQL 构建

通过 `inparam` 配置可以实现动态 SQL 条件构建：

```json
{
  "name": "searchUsers",
  "sql": "SELECT * FROM users WHERE 1=1",
  "inparam": [
    {
      "name": "status",
      "type": "int",
      "desc": "用户状态",
      "sql": " AND status = :status"
    },
    {
      "name": "ageMin",
      "type": "int", 
      "desc": "最小年龄",
      "sql": " AND age >= :ageMin"
    }
  ],
  "order": "created_at DESC"
}
```

**使用示例：**
```go
// 只传入部分参数，系统会自动构建相应的 SQL
params := map[string]interface{}{
    "status": 1,
    // ageMin 未提供，对应的条件不会被添加
}
records, err := eorm.SqlTemplate("searchUsers", params).Query()
// 生成的 SQL: SELECT * FROM users WHERE 1=1 AND status = ? ORDER BY created_at DESC
```

### 最佳实践

1. **命名规范**: 使用命名空间避免 SQL 名称冲突
2. **参数验证**: 系统会自动验证必需参数
3. **动态条件**: 使用 `inparam` 实现灵活的条件构建
4. **性能优化**: 配置文件在首次加载后会被缓存

**完整示例：**
```go
// 1. 加载配置
err := eorm.LoadSqlConfigDir("config/")
if err != nil {
    log.Fatal(err)
}

// 2. 执行查询
params := map[string]interface{}{
    "status": 1,
    "name": "张",
}

records, err := eorm.Use("mysql").
    SqlTemplate("user_service.findUsers", params).
    Timeout(30 * time.Second).
    Query()

if err != nil {
    log.Printf("执行错误: %v", err)
    return
}

// 3. 处理结果
for _, record := range records {
    fmt.Printf("用户: %s, 状态: %d\n", 
        record.GetString("name"), 
        record.GetInt("status"))
}
```

---

## 日志配置

eorm 提供了灵活的日志配置功能，支持调试模式、自定义日志记录器和文件日志。

### SetDebugMode
```go
func SetDebugMode(enabled bool)
```
开启/关闭调试模式（输出 SQL 语句）。

**示例：**
```go
// 启用调试模式，输出 SQL 语句
eorm.SetDebugMode(true)

// 关闭调试模式
eorm.SetDebugMode(false)
```

### SetLogger
```go
func SetLogger(l Logger)
```
设置自定义日志记录器。

**示例：**
```go
// 实现自定义日志记录器
type MyLogger struct{}

func (l *MyLogger) Log(level eorm.LogLevel, msg string, fields map[string]interface{}) {
    fmt.Printf("[%s] %s %v\n", level, msg, fields)
}

// 设置自定义日志记录器
eorm.SetLogger(&MyLogger{})
```

### InitLoggerWithFile
```go
func InitLoggerWithFile(level string, filePath string)
```
初始化文件日志。

**示例：**
```go
// 初始化文件日志，记录 INFO 级别及以上的日志到文件
eorm.InitLoggerWithFile("info", "logs/eorm.log")

// 记录所有级别的日志
eorm.InitLoggerWithFile("debug", "logs/debug.log")
```

### Logger 接口
```go
type Logger interface {
    Log(level LogLevel, msg string, fields map[string]interface{})
}
```

自定义日志记录器需要实现此接口。

### 日志级别
```go
const (
    LevelDebug LogLevel = "debug"
    LevelInfo  LogLevel = "info"
    LevelWarn  LogLevel = "warn"
    LevelError LogLevel = "error"
)
```

### 日志函数
```go
func LogDebug(msg string, fields map[string]interface{})
func LogInfo(msg string, fields map[string]interface{})
func LogWarn(msg string, fields map[string]interface{})
func LogError(msg string, fields map[string]interface{})
```

**示例：**
```go
// 记录调试信息
eorm.LogDebug("查询执行", map[string]interface{}{
    "sql": "SELECT * FROM users",
    "duration": "10ms",
})

// 记录错误信息
eorm.LogError("数据库连接失败", map[string]interface{}{
    "error": err.Error(),
    "dsn": "mysql://localhost:3306/test",
})
```

### 日志配置示例

```go
package main

import (
    "log"
    "os"
    
    "github.com/zzguang83325/eorm"
    _ "github.com/go-sql-driver/mysql"
)

// 自定义日志记录器
type CustomLogger struct {
    logger *log.Logger
}

func (l *CustomLogger) Log(level eorm.LogLevel, msg string, fields map[string]interface{}) {
    l.logger.Printf("[%s] %s %v", level, msg, fields)
}

func main() {
    // 方式一：使用调试模式
    eorm.SetDebugMode(true)
    
    // 方式二：使用文件日志
    eorm.InitLoggerWithFile("info", "logs/eorm.log")
    
    // 方式三：使用自定义日志记录器
    file, err := os.OpenFile("logs/custom.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()
    
    customLogger := &CustomLogger{
        logger: log.New(file, "", log.LstdFlags),
    }
    eorm.SetLogger(customLogger)
    
    // 初始化数据库
    db, err := eorm.OpenDatabase(eorm.MySQL, "root:password@tcp(localhost:3306)/test", 10)
    if err != nil {
        log.Fatal(err)
    }
    
    // 执行查询（会记录日志）
    records, err := eorm.Query("SELECT * FROM users")
    if err != nil {
        eorm.LogError("查询失败", map[string]interface{}{
            "error": err.Error(),
        })
    }
    
    // 手动记录日志
    eorm.LogInfo("应用启动完成", map[string]interface{}{
        "records_count": len(records),
    })
}
```

---

## 工具函数

### 类型转换工具

eorm 提供了一套完整的类型转换工具函数，通过 `eorm.Convert` 命名空间访问。所有函数都提供两个版本：
- 带 `WithError` 后缀：返回 `(value, error)`，适合需要错误处理的场景
- 不带后缀：返回值，转换失败时返回默认值（如果提供）或零值

#### Convert 命名空间

所有类型转换函数都通过 `eorm.Convert` 命名空间调用：

```go
// 带错误返回的版本
value, err := eorm.Convert.ToIntWithError("123")
if err != nil {
    // 处理错误
}

// 不带错误返回的版本
value := eorm.Convert.ToInt("123")           // 转换成功，返回 123
value := eorm.Convert.ToInt("abc", 999)      // 转换失败，返回默认值 999
value := eorm.Convert.ToInt("xyz")           // 转换失败，返回零值 0
```

#### 布尔类型转换

##### ToBoolWithError
```go
func (convertStruct) ToBoolWithError(a any) (bool, error)
```
将任意类型转换为 bool，转换失败返回错误。

**支持的类型：**
- 整数类型：非零为 true，零为 false
- 浮点数类型：非零为 true，零为 false
- 字符串："true", "1", "t", "T", "TRUE" 等

**示例：**
```go
value, err := eorm.Convert.ToBoolWithError("true")  // true, nil
value, err := eorm.Convert.ToBoolWithError(1)       // true, nil
value, err := eorm.Convert.ToBoolWithError("abc")   // false, error
```

##### ToBool
```go
func (convertStruct) ToBool(a any, defaultValue ...bool) bool
```
将任意类型转换为 bool，转换失败返回默认值。

**示例：**
```go
value := eorm.Convert.ToBool("true")        // true
value := eorm.Convert.ToBool("abc", false)  // false (使用默认值)
value := eorm.Convert.ToBool("xyz")         // false (零值)
```

#### 整数类型转换

##### ToIntWithError / ToInt
```go
func (convertStruct) ToIntWithError(a any) (int, error)
func (convertStruct) ToInt(a any, defaultValue ...int) int
```
转换为 int 类型。

**示例：**
```go
value, err := eorm.Convert.ToIntWithError("123")  // 123, nil
value := eorm.Convert.ToInt("456")                // 456
value := eorm.Convert.ToInt("abc", 999)           // 999
```

##### ToInt8WithError / ToInt8
```go
func (convertStruct) ToInt8WithError(a any) (int8, error)
func (convertStruct) ToInt8(a any, defaultValue ...int8) int8
```
转换为 int8 类型，自动检查溢出（-128 到 127）。

**示例：**
```go
value, err := eorm.Convert.ToInt8WithError(100)    // 100, nil
value, err := eorm.Convert.ToInt8WithError(1000)   // 0, error (溢出)
value := eorm.Convert.ToInt8(50, 0)                // 50
```

##### ToInt16WithError / ToInt16
```go
func (convertStruct) ToInt16WithError(a any) (int16, error)
func (convertStruct) ToInt16(a any, defaultValue ...int16) int16
```
转换为 int16 类型，自动检查溢出（-32768 到 32767）。

##### ToInt32WithError / ToInt32
```go
func (convertStruct) ToInt32WithError(a any) (int32, error)
func (convertStruct) ToInt32(a any, defaultValue ...int32) int32
```
转换为 int32 类型，自动检查溢出（-2147483648 到 2147483647）。

##### ToInt64WithError / ToInt64
```go
func (convertStruct) ToInt64WithError(a any) (int64, error)
func (convertStruct) ToInt64(a any, defaultValue ...int64) int64
```
转换为 int64 类型。

**示例：**
```go
value, err := eorm.Convert.ToInt64WithError("9876543210")  // 9876543210, nil
value := eorm.Convert.ToInt64("123", 0)                    // 123
```

#### 无符号整数类型转换

##### ToUintWithError / ToUint
```go
func (convertStruct) ToUintWithError(a any) (uint, error)
func (convertStruct) ToUint(a any, defaultValue ...uint) uint
```
转换为 uint 类型，自动检查负数。

**示例：**
```go
value, err := eorm.Convert.ToUintWithError(123)   // 123, nil
value, err := eorm.Convert.ToUintWithError(-100)  // 0, error (负数)
value := eorm.Convert.ToUint(456, 0)              // 456
```

##### ToUint8WithError / ToUint8
```go
func (convertStruct) ToUint8WithError(a any) (uint8, error)
func (convertStruct) ToUint8(a any, defaultValue ...uint8) uint8
```
转换为 uint8 类型，检查范围（0 到 255）。

##### ToUint16WithError / ToUint16
```go
func (convertStruct) ToUint16WithError(a any) (uint16, error)
func (convertStruct) ToUint16(a any, defaultValue ...uint16) uint16
```
转换为 uint16 类型，检查范围（0 到 65535）。

##### ToUint32WithError / ToUint32
```go
func (convertStruct) ToUint32WithError(a any) (uint32, error)
func (convertStruct) ToUint32(a any, defaultValue ...uint32) uint32
```
转换为 uint32 类型，检查范围（0 到 4294967295）。

##### ToUint64WithError / ToUint64
```go
func (convertStruct) ToUint64WithError(a any) (uint64, error)
func (convertStruct) ToUint64(a any, defaultValue ...uint64) uint64
```
转换为 uint64 类型，自动检查负数。

**示例：**
```go
value, err := eorm.Convert.ToUint64WithError("18446744073709551615")  // 最大值
value := eorm.Convert.ToUint64(123, 0)                                // 123
```

#### 浮点数类型转换

##### ToFloat32WithError / ToFloat32
```go
func (convertStruct) ToFloat32WithError(a any) (float32, error)
func (convertStruct) ToFloat32(a any, defaultValue ...float32) float32
```
转换为 float32 类型。

**示例：**
```go
value, err := eorm.Convert.ToFloat32WithError("3.14")  // 3.14, nil
value := eorm.Convert.ToFloat32("2.718", 0.0)          // 2.718
```

##### ToFloat64WithError / ToFloat64
```go
func (convertStruct) ToFloat64WithError(a any) (float64, error)
func (convertStruct) ToFloat64(a any, defaultValue ...float64) float64
```
转换为 float64 类型。

**示例：**
```go
value, err := eorm.Convert.ToFloat64WithError("3.14159")  // 3.14159, nil
value := eorm.Convert.ToFloat64("2.718", 0.0)             // 2.718
```

#### 字符串类型转换

##### ToStringWithError / ToString
```go
func (convertStruct) ToStringWithError(a any) (string, error)
func (convertStruct) ToString(a any, defaultValue ...string) string
```
将任意类型转换为字符串。

**支持的类型：**
- 基本类型：直接转换
- []byte：转换为字符串
- 结构体：序列化为 JSON

**示例：**
```go
value, err := eorm.Convert.ToStringWithError(123)           // "123", nil
value, err := eorm.Convert.ToStringWithError(3.14)          // "3.14", nil
value, err := eorm.Convert.ToStringWithError(true)          // "true", nil
value, err := eorm.Convert.ToStringWithError([]byte("hi"))  // "hi", nil

// 结构体转 JSON
type User struct {
    Name string `json:"name"`
    Age  int    `json:"age"`
}
user := User{Name: "张三", Age: 25}
value, err := eorm.Convert.ToStringWithError(user)  // {"name":"张三","age":25}

value := eorm.Convert.ToString(42, "")  // "42"
```

#### 时间类型转换

##### ToDurationWithError / ToDuration
```go
func (convertStruct) ToDurationWithError(a any) (time.Duration, error)
func (convertStruct) ToDuration(a any, defaultValue ...time.Duration) time.Duration
```
转换为 time.Duration 类型。

**支持的类型：**
- 整数类型：作为纳秒数
- 字符串：解析 Duration 格式（如 "5s", "1m", "2h"）

**示例：**
```go
value, err := eorm.Convert.ToDurationWithError("5s")           // 5秒
value, err := eorm.Convert.ToDurationWithError("1m30s")        // 1分30秒
value, err := eorm.Convert.ToDurationWithError(1000000000)     // 1秒（纳秒）

value := eorm.Convert.ToDuration("10s", 0)  // 10秒
```

##### ToTimeWithError / ToTime
```go
func (convertStruct) ToTimeWithError(a any) (time.Time, error)
func (convertStruct) ToTime(a any, defaultValue ...time.Time) time.Time
```
转换为 time.Time 类型。

**支持的类型：**
- time.Time：直接返回
- 字符串：支持多种日期时间格式（自动识别）
- int/int64/int32：Unix 时间戳（秒）
- []byte：先转为字符串再解析

**支持的字符串格式：**
- 标准格式：`2006-01-02 15:04:05`、`2006/01/02 15:04:05`
- 带纳秒：`2006-01-02 15:04:05.999999999`
- 日期格式：`2006-01-02`、`2006/01/02`
- ISO 8601：`2006-01-02T15:04:05Z`、`2006-01-02T15:04:05.999Z`
- 紧凑格式：`20060102`
- 时间格式：`15:04:05`、`15:04`
- 中文格式：`2006年01月02日 15:04:05`、`2006年01月02日`
- RFC 格式：RFC3339、RFC1123、RFC822 等

**示例：**
```go
// 标准日期时间格式
t, err := eorm.Convert.ToTimeWithError("2024-01-15 14:30:00")
t, err := eorm.Convert.ToTimeWithError("2024/01/15 14:30:00")

// 日期格式
t, err := eorm.Convert.ToTimeWithError("2024-01-15")

// ISO 8601 格式
t, err := eorm.Convert.ToTimeWithError("2024-01-15T14:30:00Z")

// Unix 时间戳
t, err := eorm.Convert.ToTimeWithError(1705315800)  // 秒级时间戳

// 中文格式
t, err := eorm.Convert.ToTimeWithError("2024年01月15日 14:30:00")

// 使用默认值
now := time.Now()
t := eorm.Convert.ToTime("invalid", now)  // 转换失败返回 now
t := eorm.Convert.ToTime("2024-01-15")    // 转换成功

// 空字符串返回零值
t, err := eorm.Convert.ToTimeWithError("")  // time.Time{}, nil
```

**性能优化：**
- 根据字符串长度和特征快速判断格式
- 最常用的格式优先匹配
- 避免不必要的格式尝试

#### 指针转换函数

##### 值转指针

将基本类型的值转换为对应的指针类型。

```go
func (convertStruct) ToIntPtr(v int) *int
func (convertStruct) ToInt8Ptr(v int8) *int8
func (convertStruct) ToInt16Ptr(v int16) *int16
func (convertStruct) ToInt32Ptr(v int32) *int32
func (convertStruct) ToInt64Ptr(v int64) *int64
func (convertStruct) ToUintPtr(v uint) *uint
func (convertStruct) ToUint8Ptr(v uint8) *uint8
func (convertStruct) ToUint16Ptr(v uint16) *uint16
func (convertStruct) ToUint32Ptr(v uint32) *uint32
func (convertStruct) ToUint64Ptr(v uint64) *uint64
func (convertStruct) ToFloat32Ptr(v float32) *float32
func (convertStruct) ToFloat64Ptr(v float64) *float64
func (convertStruct) ToBoolPtr(v bool) *bool
func (convertStruct) ToStringPtr(v string) *string
```

**示例：**
```go
// 创建指针
intPtr := eorm.Convert.ToIntPtr(123)
strPtr := eorm.Convert.ToStringPtr("hello")
boolPtr := eorm.Convert.ToBoolPtr(true)

// 常用于结构体字段
type User struct {
    Name  *string
    Age   *int
    Email *string
}

user := User{
    Name:  eorm.Convert.ToStringPtr("张三"),
    Age:   eorm.Convert.ToIntPtr(25),
    Email: eorm.Convert.ToStringPtr("zhangsan@example.com"),
}
```

##### 指针转值

将指针类型转换为对应的值类型，nil 指针返回默认值。

```go
func (convertStruct) IntPtrValue(ptr *int, defaultValue ...int) int
func (convertStruct) Int8PtrValue(ptr *int8, defaultValue ...int8) int8
func (convertStruct) Int16PtrValue(ptr *int16, defaultValue ...int16) int16
func (convertStruct) Int32PtrValue(ptr *int32, defaultValue ...int32) int32
func (convertStruct) Int64PtrValue(ptr *int64, defaultValue ...int64) int64
func (convertStruct) UintPtrValue(ptr *uint, defaultValue ...uint) uint
func (convertStruct) Uint8PtrValue(ptr *uint8, defaultValue ...uint8) uint8
func (convertStruct) Uint16PtrValue(ptr *uint16, defaultValue ...uint16) uint16
func (convertStruct) Uint32PtrValue(ptr *uint32, defaultValue ...uint32) uint32
func (convertStruct) Uint64PtrValue(ptr *uint64, defaultValue ...uint64) uint64
func (convertStruct) Float32PtrValue(ptr *float32, defaultValue ...float32) float32
func (convertStruct) Float64PtrValue(ptr *float64, defaultValue ...float64) float64
func (convertStruct) BoolPtrValue(ptr *bool, defaultValue ...bool) bool
func (convertStruct) StringPtrValue(ptr *string, defaultValue ...string) string
```

**示例：**
```go
// 从指针获取值
var intPtr *int = eorm.Convert.ToIntPtr(123)
value := eorm.Convert.IntPtrValue(intPtr)        // 123

// nil 指针返回零值
var nilPtr *int = nil
value := eorm.Convert.IntPtrValue(nilPtr)        // 0

// nil 指针返回默认值
value := eorm.Convert.IntPtrValue(nilPtr, 999)   // 999

// 处理可选字段
type User struct {
    Name  *string
    Age   *int
    Email *string
}

user := User{
    Name: eorm.Convert.ToStringPtr("李四"),
    Age:  nil,
}

name := eorm.Convert.StringPtrValue(user.Name, "未知")    // "李四"
age := eorm.Convert.IntPtrValue(user.Age, 18)            // 18 (使用默认值)
email := eorm.Convert.StringPtrValue(user.Email, "")     // "" (使用默认值)
```

**使用场景：**
- 数据库可空字段映射
- API 请求/响应中的可选字段
- 配置文件中的可选配置项
- 区分零值和未设置的情况

#### 转换规则

**转换为整数类型：**
- 其他整数类型：直接转换
- 浮点数：截断小数部分
- 布尔值：true=1, false=0
- 字符串：解析数字字符串

**转换为浮点数类型：**
- 整数类型：直接转换
- 其他浮点数类型：直接转换
- 布尔值：true=1.0, false=0.0
- 字符串：解析数字字符串

**转换为布尔类型：**
- 整数/浮点数：非零为 true，零为 false
- 字符串："true", "1", "t", "T", "TRUE" 等为 true

**错误情况：**
- nil 值：不能转换为基本类型
- 类型不兼容：无法转换的类型组合
- 溢出检查：值超出目标类型范围
- 解析失败：字符串格式不正确
- 负数转无符号：不能将负数转换为无符号整数

### 数据转换函数

#### ToStruct
```go
func ToStruct(r *Record, dest interface{}) error
func (r *Record) ToStruct(dest interface{}) error
```
将 Record 转换为结构体。

**示例：**
```go
record, err := eorm.QueryFirst("SELECT * FROM users WHERE id = ?", 1)
if err != nil {
    log.Fatal(err)
}

var user User
err = eorm.ToStruct(record, &user)
// 或者
err = record.ToStruct(&user)

fmt.Printf("用户: %s, 年龄: %d\n", user.Name, user.Age)
```

#### ToStructs
```go
func ToStructs(records []*Record, dest interface{}) error
```
将 Record 切片转换为结构体切片。

**示例：**
```go
records, err := eorm.Query("SELECT * FROM users WHERE age > ?", 18)
if err != nil {
    log.Fatal(err)
}

var users []User
err = eorm.ToStructs(records, &users)
if err != nil {
    log.Fatal(err)
}

for _, user := range users {
    fmt.Printf("用户: %s, 年龄: %d\n", user.Name, user.Age)
}
```

#### FromStruct
```go
func FromStruct(src interface{}, r *Record) error
func (r *Record) FromStruct(src interface{}) error
```
从结构体填充 Record。

**示例：**
```go
user := User{
    Name: "张三",
    Age:  25,
    Email: "zhangsan@example.com",
}

record := eorm.NewRecord()
err := eorm.FromStruct(user, record)
// 或者
err = record.FromStruct(user)

// 现在可以使用 record 进行数据库操作
id, err := eorm.InsertRecord("users", record)
```

#### ToRecord
```go
func ToRecord(src interface{}) *Record
```
将结构体转换为新的 Record。

**示例：**
```go
user := User{
    Name: "李四",
    Age:  30,
    Email: "lisi@example.com",
}

record := eorm.ToRecord(user)

// 直接使用转换后的 record
id, err := eorm.InsertRecord("users", record)
```

#### FromRecord
```go
func (r *Record) FromRecord(src *Record) *Record
```
从另一个 Record 填充当前 Record（浅拷贝）。使用浅拷贝复制数据，嵌套对象（如 map、slice、Record）会共享引用。

**特点：**
- ✅ 性能更好：只复制引用，不需要递归复制所有嵌套对象
- ✅ 内存占用更少：多个 Record 可以共享同一个嵌套对象
- ⚠️ 注意：修改嵌套对象会影响原始 Record

**适用场景：**
- 只读取数据，不修改嵌套对象
- 需要多个 Record 共享同一个配置对象
- 临时拷贝，不需要长期独立
- 性能敏感的场景

**示例：**
```go
sourceRecord := eorm.NewRecord().
    FromJson(`{"id": 1, "name": "张三", "profile": {"age": 25, "city": "北京"}}`)

// 浅拷贝：共享嵌套对象的引用
record := eorm.NewRecord().FromRecord(sourceRecord)

// 修改顶层字段不会影响原始 Record
record.Set("email", "zhangsan@example.com")

// 修改嵌套对象会影响原始 Record（因为共享引用）
if profile, _ := record.GetRecord("profile"); profile != nil {
    profile.Set("city", "上海")  // 这会同时修改 sourceRecord 的 profile
}

fmt.Println("原始:", sourceRecord.ToJson())
fmt.Println("新记录:", record.ToJson())
```

#### Clone
```go
func (r *Record) Clone() *Record
```
创建 Record 的浅拷贝，包括所有字段。与 FromRecord 类似，嵌套对象会共享引用。

**特点：**
- ✅ 性能更好：只复制引用
- ✅ 内存占用更少
- ⚠️ 注意：修改嵌套对象会影响原始 Record

**适用场景：**
- 只读取数据，不修改嵌套对象
- 需要多个 Record 共享同一个配置对象
- 临时拷贝，不需要长期独立

**示例：**
```go
original := eorm.NewRecord().
    FromJson(`{"id": 1, "name": "张三", "profile": {"age": 25, "city": "北京"}}`)

// 浅拷贝
cloned := original.Clone()

// 修改顶层字段不会影响原始
cloned.Set("name", "李四")

// 修改嵌套对象会影响原始
if profile, _ := cloned.GetRecord("profile"); profile != nil {
    profile.Set("city", "上海")  // 这会同时修改 original 的 profile
}

fmt.Println("原始:", original.ToJson())
fmt.Println("克隆:", cloned.ToJson())
```

#### DeepClone
```go
func (r *Record) DeepClone() *Record
```
创建 Record 的深拷贝，包括所有嵌套的对象。深拷贝会递归复制所有嵌套的 map、slice、Record 等对象，确保新 Record 与原 Record 完全独立。

**特点：**
- ✅ 完全独立：修改不会影响原始 Record
- ✅ 更安全：避免意外的副作用
- ⚠️ 性能相对较低：需要递归复制所有嵌套对象
- ⚠️ 内存占用更多：为嵌套对象创建新的副本

**适用场景：**
- 需要修改嵌套对象
- 需要长期独立的副本
- 需要传递给其他模块或线程
- 需要避免意外的副作用

**示例：**
```go
original := eorm.NewRecord().
    FromJson(`{"id": 1, "name": "张三", "profile": {"age": 25, "city": "北京"}}`)

// 深拷贝：创建完全独立的副本
cloned := original.DeepClone()

// 修改任何字段都不会影响原始
cloned.Set("name", "李四")
if profile, _ := cloned.GetRecord("profile"); profile != nil {
    profile.Set("city", "上海")  // 这不会影响 original 的 profile
}

fmt.Println("原始:", original.ToJson())
fmt.Println("克隆:", cloned.ToJson())
// 输出:
// 原始: {"id":1,"name":"张三","profile":{"age":25,"city":"北京"}}
// 克隆: {"id":1,"name":"李四","profile":{"age":25,"city":"上海"}}
```

#### FromRecordDeep
```go
func (r *Record) FromRecordDeep(src *Record) *Record
```
从另一个 Record 深拷贝填充当前 Record，支持链式调用。使用深拷贝复制数据，包括所有嵌套的对象，确保新 Record 与原 Record 完全独立。

**特点：**
- ✅ 完全独立：修改不会影响原始 Record
- ✅ 支持链式调用
- ⚠️ 性能相对较低：需要递归复制所有嵌套对象
- ⚠️ 内存占用更多：为嵌套对象创建新的副本

**适用场景：**
- 需要修改嵌套对象
- 需要长期独立的副本
- 需要传递给其他模块或线程
- 需要链式调用

**示例：**
```go
sourceRecord := eorm.NewRecord().
    FromJson(`{"id": 1, "name": "张三", "profile": {"age": 25, "city": "北京"}}`)

// 深拷贝填充当前 Record，支持链式调用
record := eorm.NewRecord().
    FromRecordDeep(sourceRecord).
    Set("email", "zhangsan@example.com")

// 修改嵌套对象不会影响原始 Record
if profile, _ := record.GetRecord("profile"); profile != nil {
    profile.Set("city", "上海")  // 这不会影响 sourceRecord 的 profile
}

fmt.Println("原始:", sourceRecord.ToJson())
fmt.Println("新记录:", record.ToJson())
// 输出:
// 原始: {"id":1,"name":"张三","profile":{"age":25,"city":"北京"}}
// 新记录: {"id":1,"name":"张三","profile":{"age":25,"city":"上海"},"email":"zhangsan@example.com"}
```

### 浅拷贝 vs 深拷贝对比

| 特性 | 浅拷贝 (Clone/FromRecord) | 深拷贝 (DeepClone/FromRecordDeep) |
|------|---------------------------|-----------------------------------|
| 性能 | ✅ 快 | ⚠️ 较慢 |
| 内存占用 | ✅ 少 | ⚠️ 多 |
| 嵌套对象 | ⚠️ 共享引用 | ✅ 完全独立 |
| 修改影响 | ⚠️ 修改嵌套对象会影响原始 | ✅ 修改不影响原始 |
| 适用场景 | 只读、共享引用、性能敏感 | 需要修改、需要独立、安全隔离 |

**选择建议：**
- **使用浅拷贝**：只读取数据、性能敏感、需要共享引用
- **使用深拷贝**：需要修改嵌套对象、需要独立、需要安全隔离
```

#### FromMap
​```go
func FromMap(m map[string]interface{}) *Record
```
从 map 创建新的 Record。

**示例：**
```go
// 常用于 JSON 解析后的数据
jsonMap := map[string]interface{}{
    "name": "张三",
    "age": 25,
    "email": "zhangsan@example.com",
}

record := eorm.FromMap(jsonMap)

// 直接使用创建的 record
id, err := eorm.InsertRecord("users", record)
```

### 数据库操作函数

#### Count
```go
func Count(table string, whereSql string, whereArgs ...interface{}) (int64, error)
func (db *DB) Count(table string, whereSql string, whereArgs ...interface{}) (int64, error)
func (tx *Tx) Count(table string, whereSql string, whereArgs ...interface{}) (int64, error)
```
统计记录数量。

**示例：**

```go
count, err := eorm.Count("users", "age > ?", 18)
```

### Exists
```go
func Exists(table string, whereSql string, whereArgs ...interface{}) (bool, error)
func (db *DB) Exists(table string, whereSql string, whereArgs ...interface{}) (bool, error)
func (tx *Tx) Exists(table string, whereSql string, whereArgs ...interface{}) (bool, error)
```
检查记录是否存在。

**示例：**
```go
exists, err := eorm.Exists("users", "email = ?", "user@example.com")
```

### FindAll
```go
func FindAll(table string) ([]*Record, error)
func (db *DB) FindAll(table string) ([]*Record, error)
func (tx *Tx) FindAll(table string) ([]*Record, error)
```
查找表中所有记录。

**示例：**
```go
records, err := eorm.FindAll("users")
```

### Ping
```go
func Ping() error
```
测试默认数据库连接。

**示例：**
```go
if err := eorm.Ping(); err != nil {
    log.Fatal("数据库连接失败:", err)
}
```

### PingDB
```go
func PingDB(dbname string) error
```
测试指定数据库连接。

**示例：**
```go
if err := eorm.PingDB("db1"); err != nil {
    log.Fatal("数据库 db1 连接失败:", err)
}
```

---

## 数据库连接监控

eorm 提供了自动的数据库连接监控功能，能够定时检查数据库连接状态，在连接断开时自动重连，确保应用程序的数据库连接稳定性。

### 功能特点

- **自动启用**：连接监控功能默认启用，用户无需额外配置
- **智能频率调整**：正常情况下每60秒检查一次，检测到连接错误时切换为每10秒快速重试
- **多数据库支持**：每个数据库独立监控，支持多数据库环境
- **并发控制**：使用全局锁确保同时只有一个数据库在进行连接检查，避免网络拥塞
- **简化日志**：只在连接状态变化时记录日志，减少日志噪音
- **性能优化**：使用轻量级 Ping 操作，CPU 和内存占用极低

### 配置连接监控

连接监控通过 `Config` 结构体中的两个字段进行配置：

#### 默认配置（推荐）

```go
// 使用默认配置，监控功能自动启用
db, err := eorm.OpenDatabase(eorm.MySQL, "user:pass@tcp(localhost:3306)/db", 10)
if err != nil {
    log.Fatal(err)
}

// 或使用 Config 结构体（监控字段使用默认值）
config := &eorm.Config{
    Driver:  eorm.MySQL,
    DSN:     "user:pass@tcp(localhost:3306)/db",
    MaxOpen: 10,
    // MonitorNormalInterval 和 MonitorErrorInterval 使用默认值
}
db, err = eorm.OpenDatabaseWithConfig("main", config)
```

#### 自定义监控间隔

```go
config := &eorm.Config{
    Driver:                eorm.MySQL,
    DSN:                   "user:pass@tcp(localhost:3306)/db",
    MaxOpen:               10,
    MonitorNormalInterval: 30 * time.Second, // 自定义正常检查间隔
    MonitorErrorInterval:  5 * time.Second,  // 自定义故障检查间隔
}
db, err := eorm.OpenDatabaseWithConfig("main", config)
```

#### 禁用连接监控

```go
config := &eorm.Config{
    Driver:                eorm.MySQL,
    DSN:                   "user:pass@tcp(localhost:3306)/db",
    MaxOpen:               10,
    MonitorNormalInterval: 0, // 设置为 0 禁用监控
}
db, err := eorm.OpenDatabaseWithConfig("main", config)
```

### 多数据库监控

```go
// 为不同数据库配置不同的监控策略
config1 := &eorm.Config{
    Driver:                eorm.MySQL,
    DSN:                   "user:pass@tcp(localhost:3306)/main_db",
    MaxOpen:               20,
    MonitorNormalInterval: 60 * time.Second,
    MonitorErrorInterval:  10 * time.Second,
}
db1, err := eorm.OpenDatabaseWithDBName("main", eorm.MySQL, "user:pass@tcp(localhost:3306)/main_db", 20)

config2 := &eorm.Config{
    Driver:                eorm.PostgreSQL,
    DSN:                   "host=localhost port=5432 user=postgres dbname=log_db",
    MaxOpen:               10,
    MonitorNormalInterval: 30 * time.Second, // 更频繁的检查
    MonitorErrorInterval:  5 * time.Second,
}
db2, err := eorm.OpenDatabaseWithDBName("logs", eorm.PostgreSQL, "host=localhost port=5432 user=postgres dbname=log_db", 10)

config3 := &eorm.Config{
    Driver:                eorm.SQLite3,
    DSN:                   "file:cache.db",
    MaxOpen:               5,
    MonitorNormalInterval: 0, // 禁用 SQLite 监控
}
db3, err := eorm.OpenDatabaseWithDBName("cache", eorm.SQLite3, "file:cache.db", 5)
```

### 监控工作原理

1. **定时检查**：监控器使用 `database.Ping()` 方法定时检查连接状态
2. **频率调整**：
   - 连接正常时：使用 `MonitorNormalInterval`（默认60秒）
   - 连接异常时：切换到 `MonitorErrorInterval`（默认10秒）快速重试
   - 连接恢复后：自动切换回正常间隔
3. **并发控制**：全局锁确保同时只有一个数据库在进行 Ping 操作
4. **日志记录**：只在连接状态变化时记录日志

### 性能影响

连接监控功能的性能影响极小：

- **CPU 占用**：每次检查约 6.94 纳秒，可忽略不计
- **内存占用**：每个监控器约几百字节
- **网络开销**：Ping 操作网络开销极小
- **并发控制**：全局锁避免网络突发流量

### 最佳实践

1. **生产环境**：使用默认配置（60秒/10秒）平衡性能和可靠性
2. **开发环境**：可以使用较短间隔（30秒/5秒）便于测试
3. **高可用环境**：适当缩短间隔提高响应速度
4. **本地数据库**：如 SQLite 可以考虑禁用监控
5. **多数据库**：根据数据库重要性配置不同的监控策略

---

## 查询超时控制

eorm 支持全局和单次查询超时设置，使用 Go 标准库的 `context.Context` 实现。

### 全局超时配置
在 Config 中设置 `QueryTimeout` 字段：
```go
config := &eorm.Config{
    Driver:       eorm.MySQL,
    DSN:          "root:password@tcp(localhost:3306)/test",
    MaxOpen:      10,
    QueryTimeout: 30 * time.Second,  // 所有查询默认30秒超时
}
db, err := eorm.OpenDatabaseWithConfig("main", config)
```

### Timeout (全局函数)
```go
func Timeout(d time.Duration) *DB
```
返回带有指定超时时间的 DB 实例。

**示例：**
```go
users, err := eorm.Timeout(5 * time.Second).Query("SELECT * FROM users")
```

### DB.Timeout
```go
func (db *DB) Timeout(d time.Duration) *DB
```
为 DB 实例设置查询超时时间。

**示例：**
```go
users, err := eorm.Use("default").Timeout(5 * time.Second).Query("SELECT * FROM users")
```

### Tx.Timeout
```go
func (tx *Tx) Timeout(d time.Duration) *Tx
```
为事务设置查询超时时间。

**示例：**
```go
eorm.Transaction(func(tx *eorm.Tx) error {
    _, err := tx.Timeout(5 * time.Second).Query("SELECT * FROM orders")
    return err
})
```

### QueryBuilder.Timeout
```go
func (qb *QueryBuilder) Timeout(d time.Duration) *QueryBuilder
```
为链式查询设置超时时间。

**示例：**
```go
users, err := eorm.Table("users").
    Where("age > ?", 18).
    Timeout(10 * time.Second).
    Find()
```

### 超时错误处理
超时后返回 `context.DeadlineExceeded` 错误：
```go
import "context"
import "errors"

users, err := eorm.Timeout(1 * time.Second).Query("SELECT SLEEP(5)")
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        fmt.Println("查询超时")
    }
}
```

---

### ToJson
```go
func ToJson(v interface{}) string
```
将对象转换为 JSON 字符串。

**示例：**
```go
record := eorm.NewRecord()
record.Set("name", "张三")
record.Set("age", 25)

jsonStr := eorm.ToJson(record)
fmt.Println(jsonStr)
```

---

## 常量和类型

### 数据库驱动类型
```go
const (
    MySQL      DriverType = "mysql"
    PostgreSQL DriverType = "postgres"
    SQLite3    DriverType = "sqlite3"
    Oracle     DriverType = "oracle"
    SQLServer  DriverType = "sqlserver"
)
```

### 软删除类型
```go
const (
    SoftDeleteTimestamp SoftDeleteType = "timestamp"
    SoftDeleteBool      SoftDeleteType = "bool"
)
```

### 
---

## 注意事项

1. **数据库驱动**：需要手动导入相应的数据库驱动
2. **功能启用**：时间戳、乐观锁、软删除功能需要先启用再配置
3. **事务处理**：长时间运行的事务可能导致锁等待，建议合理控制事务范围
4. **批量操作**：大量数据操作时建议使用批量函数提高性能
5. **缓存使用**：合理设置缓存 TTL，避免数据不一致
6. **SQL 安全**：框架内置 SQL 注入防护，但仍需注意参数化查询

---

## 版本信息

- **当前版本**：v1.0.0
- **Go 版本要求**：Go 1.21+
- **支持数据库**：MySQL, PostgreSQL, SQLite3, Oracle, SQL Server

---

## 许可证

本项目采用 MIT 许可证。详情请参阅 LICENSE 文件。

---

## 批量执行 SQL 语句

eorm 提供了 `BatchExec` 功能，允许一次性执行多个 SQL 语句。这在数据库初始化、批量 DDL 操作、数据迁移等场景中非常有用。

### 核心概念

**执行模式：**
- **非事务模式**：使用 `db.BatchExec()` 或全局 `BatchExec()`，遇到错误会继续执行后续语句，已执行的语句不会回滚
- **事务模式**：使用 `tx.BatchExec()`，遇到错误会停止执行后续语句，回滚事务

### StatementResult 结构体

每个 SQL 语句的执行结果都封装在 `StatementResult` 结构体中：

```go
type StatementResult struct {
    Index  int           // 语句索引（从 0 开始）
    SQL    string        // SQL 语句
    Args   []interface{} // 参数列表
    Result sql.Result    // 执行结果（成功时有值，失败时为 nil）
    Error  error         // 错误信息（失败时有值，成功时为 nil）
}
```

**辅助方法：**

```go
// IsSuccess 判断该语句是否执行成功
func (r *StatementResult) IsSuccess() bool

// RowsAffected 获取受影响的行数（成功时）
func (r *StatementResult) RowsAffected() (int64, error)

// LastInsertId 获取最后插入的 ID（成功时）
func (r *StatementResult) LastInsertId() (int64, error)
```

### BatchExec (全局函数)

```go
func BatchExec(sqls []string, args ...[]interface{}) ([]StatementResult, error)
```

批量执行多个 SQL 语句（非事务模式）。

**参数：**
- `sqls`: SQL 语句列表
- `args`: 每个 SQL 语句对应的参数列表（可选）

**返回值：**
- `[]StatementResult`: 每个语句的执行结果列表
- `error`: 如果有语句执行失败，返回错误信息

**示例：**

```go
// 示例 1: 不带参数的批量执行
sqls := []string{
    "CREATE TABLE IF NOT EXISTS users (id INT PRIMARY KEY, name VARCHAR(100))",
    "CREATE INDEX idx_name ON users(name)",
    "INSERT INTO users (id, name) VALUES (1, 'Alice')",
}

results, err := eorm.BatchExec(sqls)
if err != nil {
    log.Printf("批量执行出错: %v\n", err)
}

// 检查每个语句的执行结果
for _, r := range results {
    if r.IsSuccess() {
        affected, _ := r.RowsAffected()
        fmt.Printf("语句 %d 成功，影响 %d 行\n", r.Index, affected)
    } else {
        fmt.Printf("语句 %d 失败: %v\n", r.Index, r.Error)
    }
}

// 示例 2: 带参数的批量执行
sqls := []string{
    "INSERT INTO users (id, name) VALUES (?, ?)",
    "INSERT INTO users (id, name) VALUES (?, ?)",
    "UPDATE users SET name = ? WHERE id = ?",
}

args := [][]interface{}{
    {1, "Alice"},
    {2, "Bob"},
    {"Charlie", 1},
}

results, err := eorm.BatchExec(sqls, args)
if err != nil {
    log.Printf("批量执行出错: %v\n", err)
}

// 示例 3: 混合使用带参数和不带参数的语句
sqls := []string{
    "CREATE TABLE IF NOT EXISTS products (id INT PRIMARY KEY, name VARCHAR(100))",
    "INSERT INTO products (id, name) VALUES (?, ?)",
    "INSERT INTO products (id, name) VALUES (?, ?)",
}

args := [][]interface{}{
    nil,              // CREATE TABLE 不需要参数
    {1, "Product A"}, // INSERT 参数
    {2, "Product B"}, // INSERT 参数
}

results, err := eorm.BatchExec(sqls, args)
```

### BatchExec (DB 方法)

```go
func (db *DB) BatchExec(sqls []string, args ...[]interface{}) ([]StatementResult, error)
```

在指定数据库实例上批量执行多个 SQL 语句（非事务模式）。

**示例：**

```go
// 使用特定数据库实例
db, err := eorm.OpenDatabase(eorm.MySQL, dsn, 10)
if err != nil {
    log.Fatal(err)
}

sqls := []string{
    "CREATE TABLE IF NOT EXISTS orders (id INT PRIMARY KEY, total DECIMAL(10,2))",
    "INSERT INTO orders (id, total) VALUES (?, ?)",
    "INSERT INTO orders (id, total) VALUES (?, ?)",
}

args := [][]interface{}{
    nil,
    {1, 100.50},
    {2, 200.75},
}

results, err := db.BatchExec(sqls, args)
if err != nil {
    log.Printf("批量执行出错: %v\n", err)
}

// 多数据库模式
results, err := eorm.Use("db1").BatchExec(sqls, args)
```

### BatchExec (Tx 方法)

```go
func (tx *Tx) BatchExec(sqls []string, args ...[]interface{}) ([]StatementResult, error)
```

在事务中批量执行多个 SQL 语句。遇到错误会停止执行后续语句，由调用者决定是否回滚事务。

**示例：**

```go
// 示例 1: 事务模式 - 遇到错误立即停止
tx, err := eorm.BeginTransaction()
if err != nil {
    log.Fatal(err)
}

sqls := []string{
    "INSERT INTO users (id, name) VALUES (?, ?)",
    "INSERT INTO users (id, name) VALUES (?, ?)", // 假设这里主键冲突
    "INSERT INTO users (id, name) VALUES (?, ?)", // 不会执行
}

args := [][]interface{}{
    {1, "Alice"},
    {1, "Bob"},     // 主键冲突，会失败
    {2, "Charlie"}, // 不会执行
}

results, err := tx.BatchExec(sqls, args)
// results 包含 2 条结果（第 0 条成功，第 1 条失败）
// err != nil

// 检查结果并决定是否回滚
if err != nil {
    tx.Rollback() // 回滚所有操作
    log.Printf("事务回滚: %v\n", err)
} else {
    tx.Commit() // 提交事务
}

// 示例 2: 使用 Transaction 函数
err := eorm.Transaction(func(tx *eorm.Tx) error {
    sqls := []string{
        "INSERT INTO users (id, name) VALUES (?, ?)",
        "INSERT INTO orders (id, user_id, total) VALUES (?, ?, ?)",
    }
    
    args := [][]interface{}{
        {1, "Alice"},
        {1, 1, 100.00},
    }
    
    results, err := tx.BatchExec(sqls, args)
    if err != nil {
        return err // 自动回滚
    }
    
    // 检查结果
    for _, r := range results {
        if !r.IsSuccess() {
            return fmt.Errorf("语句 %d 执行失败: %v", r.Index, r.Error)
        }
    }
    
    return nil // 自动提交
})

if err != nil {
    log.Printf("事务执行失败: %v\n", err)
}
```

### 执行结果处理

```go
// 完整的结果处理示例
sqls := []string{
    "INSERT INTO users (id, name) VALUES (?, ?)",
    "INSERT INTO users (id, name) VALUES (?, ?)",
    "UPDATE users SET name = ? WHERE id = ?",
}

args := [][]interface{}{
    {1, "Alice"},
    {2, "Bob"},
    {"Charlie", 1},
}

results, err := eorm.BatchExec(sqls, args)

// 统计执行结果
successCount := 0
failedCount := 0

for _, r := range results {
    if r.IsSuccess() {
        successCount++
        
        // 获取受影响的行数
        affected, err := r.RowsAffected()
        if err == nil {
            fmt.Printf("语句 %d 成功，影响 %d 行\n", r.Index, affected)
        }
        
        // 对于 INSERT 语句，可以获取插入的 ID
        if r.Result != nil {
            lastID, err := r.LastInsertId()
            if err == nil && lastID > 0 {
                fmt.Printf("  插入的 ID: %d\n", lastID)
            }
        }
    } else {
        failedCount++
        fmt.Printf("语句 %d 失败\n", r.Index)
        fmt.Printf("  SQL: %s\n", r.SQL)
        fmt.Printf("  参数: %v\n", r.Args)
        fmt.Printf("  错误: %v\n", r.Error)
    }
}

fmt.Printf("\n总共: %d 条，成功: %d 条，失败: %d 条\n", 
    len(results), successCount, failedCount)

// 检查是否有失败的语句
if err != nil {
    log.Printf("批量执行完成，但有 %d 条语句失败\n", failedCount)
}
```

### 使用场景

#### 1. 数据库初始化

```go
// 数据库初始化脚本
sqls := []string{
    "CREATE TABLE IF NOT EXISTS users (id INT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(100), email VARCHAR(100))",
    "CREATE TABLE IF NOT EXISTS orders (id INT PRIMARY KEY AUTO_INCREMENT, user_id INT, total DECIMAL(10,2))",
    "CREATE INDEX idx_user_email ON users(email)",
    "CREATE INDEX idx_order_user ON orders(user_id)",
}

results, err := eorm.BatchExec(sqls)
if err != nil {
    log.Fatal("数据库初始化失败:", err)
}

fmt.Println("数据库初始化完成")
```

#### 2. 数据迁移

```go
// 数据迁移脚本
sqls := []string{
    "ALTER TABLE users ADD COLUMN phone VARCHAR(20)",
    "ALTER TABLE users ADD COLUMN address TEXT",
    "UPDATE users SET phone = '' WHERE phone IS NULL",
    "UPDATE users SET address = '' WHERE address IS NULL",
}

results, err := eorm.BatchExec(sqls)
if err != nil {
    log.Printf("数据迁移失败: %v\n", err)
    // 检查哪些语句失败了
    for _, r := range results {
        if !r.IsSuccess() {
            log.Printf("失败的语句: %s, 错误: %v\n", r.SQL, r.Error)
        }
    }
}
```

#### 3. 批量数据操作（事务）

```go
// 在事务中批量插入和更新
err := eorm.Transaction(func(tx *eorm.Tx) error {
    sqls := []string{
        "INSERT INTO users (id, name, email) VALUES (?, ?, ?)",
        "INSERT INTO users (id, name, email) VALUES (?, ?, ?)",
        "INSERT INTO users (id, name, email) VALUES (?, ?, ?)",
        "UPDATE users SET status = ? WHERE id IN (?, ?, ?)",
    }
    
    args := [][]interface{}{
        {1, "Alice", "alice@example.com"},
        {2, "Bob", "bob@example.com"},
        {3, "Charlie", "charlie@example.com"},
        {1, 1, 2, 3},
    }
    
    results, err := tx.BatchExec(sqls, args)
    if err != nil {
        return err
    }
    
    // 验证所有语句都成功
    for _, r := range results {
        if !r.IsSuccess() {
            return fmt.Errorf("语句 %d 执行失败: %v", r.Index, r.Error)
        }
    }
    
    return nil
})

if err != nil {
    log.Printf("批量操作失败: %v\n", err)
}
```

### 最佳实践

#### 1. 参数化查询

始终使用参数化查询，避免 SQL 注入：

```go
// ✅ 推荐：使用参数化查询
sqls := []string{
    "INSERT INTO users (name, email) VALUES (?, ?)",
    "UPDATE users SET status = ? WHERE id = ?",
}

args := [][]interface{}{
    {"Alice", "alice@example.com"},
    {1, 123},
}

results, err := eorm.BatchExec(sqls, args)

// ❌ 不推荐：字符串拼接（有 SQL 注入风险）
name := "Alice"
email := "alice@example.com"
sqls := []string{
    fmt.Sprintf("INSERT INTO users (name, email) VALUES ('%s', '%s')", name, email),
}
```

#### 2. 错误处理

始终检查执行结果和错误：

```go
results, err := eorm.BatchExec(sqls, args)

// 检查是否有错误
if err != nil {
    log.Printf("批量执行出错: %v\n", err)
}

// 检查每个语句的执行结果
for _, r := range results {
    if !r.IsSuccess() {
        log.Printf("语句 %d 失败: SQL=%s, 错误=%v\n", r.Index, r.SQL, r.Error)
    }
}
```

#### 3. 事务使用

对于需要原子性的操作，使用事务模式：

```go
// 需要原子性：使用事务
tx, err := eorm.BeginTransaction()
if err != nil {
    log.Fatal(err)
}

results, err := tx.BatchExec(sqls, args)
if err != nil {
    tx.Rollback() // 回滚
    log.Printf("执行失败，已回滚: %v\n", err)
} else {
    tx.Commit() // 提交
    log.Println("执行成功，已提交")
}
```

#### 4. 批量大小控制

对于大量 SQL 语句，建议分批执行：

```go
// 分批执行大量 SQL 语句
allSqls := []string{/* 1000 条 SQL 语句 */}
batchSize := 100

for i := 0; i < len(allSqls); i += batchSize {
    end := i + batchSize
    if end > len(allSqls) {
        end = len(allSqls)
    }
    
    batch := allSqls[i:end]
    results, err := eorm.BatchExec(batch)
    if err != nil {
        log.Printf("批次 %d-%d 执行失败: %v\n", i, end, err)
        break
    }
    
    fmt.Printf("批次 %d-%d 执行完成\n", i, end)
}
```

#### 5. 空字符串处理

BatchExec 会自动跳过空字符串：

```go
sqls := []string{
    "CREATE TABLE users (id INT PRIMARY KEY)",
    "",  // 空字符串会被跳过
    "INSERT INTO users (id) VALUES (1)",
    "  ", // 只包含空格的字符串也会被跳过
    "INSERT INTO users (id) VALUES (2)",
}

results, err := eorm.BatchExec(sqls)
// 只会执行 3 条有效的 SQL 语句
```

### 注意事项

1. **参数数量匹配**：如果提供了 `args` 参数，其长度必须与 `sqls` 长度一致
2. **占位符转换**：系统会自动将占位符转换为对应数据库的格式（MySQL 的 `?`、PostgreSQL 的 `$1`、Oracle 的 `:1` 等）
3. **错误停止**：遇到错误会立即停止执行后续语句
4. **事务回滚**：在事务模式下，如果执行失败，需要手动调用 `tx.Rollback()` 回滚事务
5. **连接复用**：所有语句使用同一个数据库连接执行
6. **超时控制**：支持查询超时配置，防止长时间阻塞

### 多数据库支持

BatchExec 支持所有 eorm 支持的数据库驱动：

```go
// MySQL
results, err := eorm.Use("mysql").BatchExec(sqls, args)

// PostgreSQL
results, err := eorm.Use("postgres").BatchExec(sqls, args)

// SQLite
results, err := eorm.Use("sqlite").BatchExec(sqls, args)

// Oracle
results, err := eorm.Use("oracle").BatchExec(sqls, args)

// SQL Server
results, err := eorm.Use("sqlserver").BatchExec(sqls, args)
```

每个数据库的占位符会自动转换为对应的格式。

### 性能考虑

1. **连接复用**：所有语句使用同一个数据库连接，避免频繁创建连接
2. **预编译缓存**：支持预编译语句缓存，提高重复执行的性能
3. **批量大小**：建议单次批量执行不超过 1000 条语句
4. **事务开销**：事务模式会有额外的事务管理开销，但提供原子性保证

### 与其他批量操作的对比

| 功能 | BatchExec | BatchInsertRecord | BatchUpdateRecord |
|------|-----------|-------------------|-------------------|
| 用途 | 执行任意 SQL 语句 | 批量插入记录 | 批量更新记录 |
| 灵活性 | 高（支持任意 SQL） | 中（仅 INSERT） | 中（仅 UPDATE） |
| 参数化 | 支持 | 自动处理 | 自动处理 |
| 事务支持 | 支持 | 支持 | 支持 |
| 错误处理 | 遇错停止 | 遇错停止 | 遇错停止 |
| 适用场景 | DDL、混合操作、迁移 | 批量插入数据 | 批量更新数据 |

### 完整示例

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/zzguang83325/eorm"
    _ "github.com/zzguang83325/eorm/drivers/mysql"
)

func main() {
    // 初始化数据库
    db, err := eorm.OpenDatabase(eorm.MySQL, 
        "root:123456@tcp(127.0.0.1:3306)/test?charset=utf8mb4&parseTime=True&loc=Local", 10)
    if err != nil {
        log.Fatal(err)
    }
    
    // 示例 1: 数据库初始化
    fmt.Println("=== 数据库初始化 ===")
    initSqls := []string{
        "DROP TABLE IF EXISTS batch_test_users",
        "CREATE TABLE batch_test_users (id INT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(100), email VARCHAR(100))",
        "CREATE INDEX idx_email ON batch_test_users(email)",
    }
    
    results, err := eorm.BatchExec(initSqls)
    if err != nil {
        log.Printf("初始化失败: %v\n", err)
    } else {
        fmt.Printf("初始化成功，执行了 %d 条语句\n", len(results))
    }
    
    // 示例 2: 批量插入（非事务模式）
    fmt.Println("\n=== 批量插入（非事务模式） ===")
    insertSqls := []string{
        "INSERT INTO batch_test_users (name, email) VALUES (?, ?)",
        "INSERT INTO batch_test_users (name, email) VALUES (?, ?)",
        "INSERT INTO batch_test_users (name, email) VALUES (?, ?)",
    }
    
    insertArgs := [][]interface{}{
        {"Alice", "alice@example.com"},
        {"Bob", "bob@example.com"},
        {"Charlie", "charlie@example.com"},
    }
    
    results, err = eorm.BatchExec(insertSqls, insertArgs)
    if err != nil {
        log.Printf("插入失败: %v\n", err)
    }
    
    for _, r := range results {
        if r.IsSuccess() {
            affected, _ := r.RowsAffected()
            lastID, _ := r.LastInsertId()
            fmt.Printf("语句 %d 成功: 影响 %d 行, 插入 ID=%d\n", r.Index, affected, lastID)
        } else {
            fmt.Printf("语句 %d 失败: %v\n", r.Index, r.Error)
        }
    }
    
    // 示例 3: 事务模式
    fmt.Println("\n=== 事务模式 ===")
    err = eorm.Transaction(func(tx *eorm.Tx) error {
        txSqls := []string{
            "INSERT INTO batch_test_users (name, email) VALUES (?, ?)",
            "UPDATE batch_test_users SET name = ? WHERE email = ?",
        }
        
        txArgs := [][]interface{}{
            {"David", "david@example.com"},
            {"Alice Updated", "alice@example.com"},
        }
        
        results, err := tx.BatchExec(txSqls, txArgs)
        if err != nil {
            return err
        }
        
        for _, r := range results {
            if !r.IsSuccess() {
                return fmt.Errorf("语句 %d 失败: %v", r.Index, r.Error)
            }
        }
        
        fmt.Println("事务执行成功")
        return nil
    })
    
    if err != nil {
        log.Printf("事务失败: %v\n", err)
    }
    
    // 查询结果
    fmt.Println("\n=== 查询结果 ===")
    records, err := eorm.Query("SELECT * FROM batch_test_users")
    if err != nil {
        log.Fatal(err)
    }
    
    for _, record := range records {
        fmt.Printf("ID: %d, Name: %s, Email: %s\n",
            record.GetInt("id"),
            record.GetString("name"),
            record.GetString("email"))
    }
}
```

