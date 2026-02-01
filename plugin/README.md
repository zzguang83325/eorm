# Eorm GORM 插件使用指南

## 简介
`EormGormPlugin` 是一个为 GORM 深度定制的插件，旨在让 GORM 能够原生支持 `eorm.Record` 类型。通过该插件，你可以直接使用 GORM 的 API 来操作动态结构的 `Record` 对象，而无需预定义静态的 Go 结构体。

## 为什么选择 eorm.Record？

在传统的 GORM 开发中，每个数据库表通常需要对应一个静态的 Go `struct`。这种“一表一结构”的模式在处理简单业务时很清晰，但在中大型或动态业务中，其**繁琐性与缺点**也十分明显：

- **维护成本高**：数据库表每增加、删除或修改一个字段，都必须同步修改 Go 代码中的 `struct` 定义，并重新编译。在表结构频繁变动的开发初期，这会浪费大量时间。
- **应对动态需求吃力**：面对动态表单、自定义报表、或需要处理成百上千张临时表的场景，预定义 `struct` 几乎是不可能的任务。
- **代码冗余严重**：为了处理局部更新（Partial Update）或特定字段的查询，开发者往往需要定义大量的临时 `struct` 或使用 `map[string]interface{}`，导致项目代码臃肿且难以维护。
- **强耦合风险**：静态结构体将业务逻辑与数据库 Schema 深度绑定，降低了系统的灵活性，尤其在处理多租户、动态分表等场景时，代码逻辑会变得异常复杂。

相比之下，`eorm.Record` 提供了更强大的灵活性：

- **动态 Schema**：无需预先定义结构体，支持在运行时处理任意结构的表或查询结果。
- **减少冗余代码**：对于字段频繁变动的业务，或者需要处理大量临时表、分析型查询的场景，免去了频繁修改和维护 `struct` 的痛苦。
- **无缝集成与嵌套支持**：完美支持 JSON 序列化（`ToJson()`），且 `Record` 内部可以集成各种数据类型（包括基础类型、`Record` 对象、甚至 `Record` 切片）。这种强大的**嵌套组合能力**，使其能够极其方便地构建并返回给前端复杂的、多层级的查询结果。
- **类型安全转换**：内置了健壮的类型转换机制，能够安全地处理数据库类型与 Go 类型之间的转换。

## 核心特性
- **动态 Schema 支持**：直接使用 `eorm.Record` 进行增删改查。
- **连接池共享**：`eorm` 与 GORM 完美共享同一个数据库连接池，无需建立多余连接，节省系统资源。
- **双 API 协同**：你既可以继续使用 GORM 原生的 Struct 操作，也可以在同一事务或连接中使用 `eorm` 的原生 API，两者互不干扰。
- **智能事务感知**：自动识别事务环境，已处于事务中时自动复用；支持配置是否在非事务环境下自动为写操作开启事务。
- **高性能类型缓存**：内置反射类型缓存，并具备自动清理机制防止内存溢出。
- **详细的错误上下文**：报错时会自动附带表名、SQL 语句等调试信息。
- **原生 API 兼容**：支持 `First`, `Find`, `Create`, `Save`, `Update`, `Delete` 等标准 GORM 链式调用。
- **无侵入设计**：插件仅在识别到 `eorm.Record` 相关类型时介入，完全不影响 GORM 对普通 Go 结构体（Struct）的原生增删改查操作。

## Record 基础操作指南

`Record` 是 eorm 的核心，它类似于一个增强版的 `map[string]interface{}`，其字段名不区分大小写。

### 1. 创建与赋值
```go
// 创建 Record 对象并链式赋值
record := eorm.NewRecord().
    Set("name", "张三").
    Set("age", 30).
    Set("is_vip", true)
```

### 2. 类型安全获取值
```go
name := record.Str("name")       // 获取字符串
age := record.Int("age")         // 获取整数
isVIP := record.Bool("is_vip")   // 获取布尔值
salary := record.Float("salary") // 获取浮点数
```

### 3. JSON 转换与嵌套支持
```go
// 1. 嵌套组合：Record 中可以包含 Record 切片
subRecords := []eorm.Record{
    *eorm.NewRecord().Set("item", "A"),
    *eorm.NewRecord().Set("item", "B"),
}
mainRecord := eorm.NewRecord().
    Set("title", "订单列表").
    Set("items", subRecords) // 极其方便返回给前端复杂结构

// 2. 转换为 JSON 字符串
jsonStr := mainRecord.ToJson()

// 3. 从 JSON 字符串填充 Record
newRecord := eorm.NewRecord()
newRecord.FromJson(jsonStr)
```

### 4. 常用 API 示例 (通过 GORM 插件)

#### 查询 (Read)
```go
// 查询单条并转换为 Record
var user eorm.Record
db.Table("users").Where("id = ?", 1).First(&user)

// 查询多条并转换为切片
var users []eorm.Record
db.Table("users").Where("age > ?", 18).Find(&users)
```

#### 插入与更新 (Create/Save/Update)
```go
// 插入新记录
db.Table("users").Create(record)

// 根据主键保存 (存在则更新，不存在则插入)
db.Table("users").Save(record)

// 指定条件更新
db.Table("users").Where("id = ?", 1).Updates(record)
```

#### 删除 (Delete)
```go
// 根据 Record 中的主键删除
db.Table("users").Delete(record)

// 根据条件删除
db.Table("users").Where("id = ?", 1).Delete(&eorm.Record{})
```

## 工作原理

`EormGormPlugin` 作为一个中间层，通过拦截 GORM 的回调（Callbacks）来决定执行路径：

1. **eorm 接管路径**：
   当 GORM 的 `Dest` 为 `eorm.Record` 相关类型时，插件会拦截原生执行逻辑，并调用 `eorm` 的核心 API：
   - **查询**：通过 `eorm.ScanRecords` 处理结果集。
   - **创建**：调用 `eorm.SaveRecordWithExecutor` 或 `eorm.BatchInsertRecordWithExecutor`。
   - **更新/保存**：调用 `eorm.SaveRecordWithExecutor`（根据主键自动判断）。
   - **删除**：调用 `eorm.DeleteRecordWithExecutor`。

2. **GORM 原生路径**：
   当操作对象为普通 Go 结构体或 Map 时，插件会自动跳过（Pass-through），由 GORM 原生引擎完全负责执行。

## 配置参数说明

`EormGormPlugin` 接收一个 `Config` 结构体用于行为控制，各参数详细说明如下：

| 参数名 | 类型 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- |
| `EnableCache` | `bool` | `true` | 是否启用反射类型缓存。开启后可显著提升重复查询的性能。 |
| `MaxCacheSize` | `int` | `1000` | 缓存的最大条目数。超过此限制后，插件会自动执行清理以防止内存溢出。 |
| `EnableAutoTransaction` | `bool` | `false` | 是否在写操作（Create/Update/Delete）时自动开启事务。设为 `false` 则由用户显式控制事务。 |

## 快速开始

### 1. 注册插件
在初始化 GORM `db` 对象后，使用 `Use` 方法注册插件：

```go
import (
    "github.com/zzguang83325/eorm/plugin"
    "gorm.io/gorm"
)

db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{})

// 注册插件
db.Use(&plugin.EormGormPlugin{})
```

### 2. 基本操作示例

#### 连接池共享与双 API 协同

插件最强大的地方在于它让 `eorm` 和 `GORM` 站在了同一条战线上：

```go
// 1. GORM 原生操作 (使用 Struct)
var userStruct User
db.First(&userStruct, 1)

// 2. 插件增强操作 (使用 Record)
var userRecord eorm.Record
db.Table("users").First(&userRecord, 1)

// 3. eorm 原生操作 (共享 GORM 的连接池)
// 你可以直接从 db 对象中提取 *sql.DB 给 eorm 使用，或者通过插件提供的执行器
records, _ := eorm.Query("SELECT * FROM users LIMIT 5")
```
```go
user := eorm.NewRecord()
user.Set("name", "张三")
user.Set("age", 25)

// 直接将 Record 传递给 GORM
db.Table("users").Create(user)
```

#### 查询数据 (Find/First)
```go
// 查询单条
var record eorm.Record
db.Table("users").Where("id = ?", 1).First(&record)

// 查询多条
var records []eorm.Record
db.Table("users").Limit(10).Find(&records)
```

#### 更新数据 (Update/Save)
```go
record.Set("age", 26)
db.Table("users").Save(&record) // Save 会根据主键自动判断 Insert 或 Update
```

#### 删除数据 (Delete)
```go
db.Table("users").Delete(&record)
```

## 高级配置

你可以在初始化插件时自定义配置：

```go
plugin := &plugin.EormGormPlugin{
    Config: plugin.Config{
        EnableCache:           true,  // 是否启用类型缓存（默认开启）
        MaxCacheSize:          2000,  // 缓存项上限（默认 1000）
        EnableAutoTransaction: false, // 写操作是否自动开启事务（默认 false）
    },
}
db.Use(plugin)
```


## 调试与日志

由于该插件是连接 GORM 和 eorm 的桥梁，你可以通过以下两种方式查看执行细节：

### 1. 开启 GORM 日志
查看插件交给 GORM 执行的最终 SQL 语句：
```go
import "gorm.io/gorm/logger"

db.Config.Logger = logger.Default.LogMode(logger.Info)
```

### 2. 开启 eorm 调试模式
查看 eorm 内部的 SQL 构建逻辑和执行详情：
```go
import "github.com/zzguang83325/eorm"

eorm.SetDebugMode(true) // 开启 eorm 的 SQL 日志
```
