# EORM - Go Database Library

[English](README_EN.md) | [API 手册](api.md) | [API Reference](api_en.md) | [SQL 模板指南](doc/cn/SQL_TEMPLATE_GUIDE.md) | [SQL Template Guide](doc/en/SQL_TEMPLATE_GUIDE_EN.md) | [缓存使用指南](doc/cn/CACHE_ENHANCEMENT_GUIDE.md) | [Cache Usage Guide](doc/en/CACHE_ENHANCEMENT_GUIDE.md)

EORM (easy orm)是一个基于 Go 语言的高性能的数据库ORM框架。它提供了简洁、直观的 API和灵活的Record对象，无需定义Struct即可对数据库进行CRUD操作。 

**项目链接**：https://github.com/zzguang83325/eorm.git 

## 特性

- **数据库支持**: 支持 MySQL、PostgreSQL、SQLite、SQL Server、Oracle
- **多数据库管理**：支持同时连接多个数据库，并能轻松在它们之间切换。 
- **Record 对象**：摆脱繁琐的 Struct 定义，使用灵活的 `Record` 对数据进行 CRUD,灵感来源于Jfinal。
- **DbModel体验**:  在Record对象之外,可通过自动生成的DbModel对象，对数据CRUD。 
- **SQL 模板**: 支持 SQL 配置化管理，动态参数构建，支持可变参数 - [详细指南](doc/cn/SQL_TEMPLATE_GUIDE.md)
- **事务支持**:  提供简单易用的事务包装器及底层事务控制 
- **安全防护**: 内置 SQL 安全验证器，防御 SQL 注入、XSS 等攻击，支持 SELECT 语法白名单及危险模式检测
- **智能缓存**: 
  - **结果缓存**: 支持内存及 Redis 缓存，提供链式 API
  - **语句缓存**: 自动 LRU 语句缓存（Statement Cache），显著提升高并发下的查询性能
- **连接监控**: 自动监控数据库连接状态，支持故障检测与自动重连，确保服务高可用
- **分页查询**:  针对不同数据库优化的分页查询实现,一个函数即可查出记录数、总页数和当前页数据
- **日志记录**：内置 日志系统，支持详细的 SQL 执行耗时分析
- **自动时间戳**: 支持配置自动时间戳字段，插入和更新时自动填充 created_at 和 updated_at
- **软删除支持**: 支持配置软删除字段（时间戳/布尔值），自动过滤已删除记录，提供恢复和物理删除功能
- **乐观锁支持**: 支持配置版本字段，自动检测并发冲突，防止数据覆盖



## 安装

```
go get github.com/zzguang83325/eorm@latest
```

## 数据库驱动

eorm 支持以下数据库，你需要根据使用的数据库安装对应的驱动。

| 数据库     | 驱动包                                                                       |
| ---------- | -------------------------------- | --------------------------------------------------- |
| MySQL      | github.com/go-sql-driver/mysql    |
| PostgreSQL | github.com/jackc/pgx/v5/stdlib   |
| SQLite3    | github.com/mattn/go-sqlite3      |
| SQL Server | github.com/denisenkom/go-mssqldb |
| Oracle     | github.com/sijms/go-ora/v2       |

eorm 已经对上述驱动程序做了封装处理，在代码中直接导入即可使用：

```go
// MySQL
import _ "github.com/zzguang83325/eorm/drivers/mysql"

// PostgreSQL
import _ "github.com/zzguang83325/eorm/drivers/postgres"

// SQLite3
import _ "github.com/zzguang83325/eorm/drivers/sqlite"

// SQL Server
import _ "github.com/zzguang83325/eorm/drivers/sqlserver"

// Oracle
import _ "github.com/zzguang83325/eorm/drivers/oracle"
```



## 快速开始

```go
package main

import (
	"fmt"
	"log"
	"github.com/zzguang83325/eorm"
	_ "github.com/zzguang83325/eorm/drivers/mysql" // MySQL 驱动

)

func main() {
	// 初始化数据库连接, eorm支持多数据库,第一个打开的数据库是默认数据库
	db, err := eorm.OpenDatabase(eorm.MySQL, "root:password@tcp(localhost:3306)/test?charset=utf8mb4&parseTime=True&loc=Local", 10)
	if err != nil {
		log.Fatal(err)
	}
	defer eorm.Close()

	records, err := db.Query("SELECT * FROM users")
	if err != nil {
		log.Fatal(err)
	}

	// 测试连接
	if err := eorm.Ping(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("数据库连接成功")

	// 不指定db,直接操作的是默认数据库
	eorm.Exec(`CREATE TABLE IF NOT EXISTS users (
        id INT AUTO_INCREMENT PRIMARY KEY,
        name VARCHAR(100) NOT NULL,
        age INT NOT NULL,
        email VARCHAR(100) NOT NULL UNIQUE
    )`)

	// 创建Record, 并插入数据
	user := eorm.NewRecord().
		Set("name", "张三").
		Set("age", 25).
		Set("email", "zhangsan@example.com")

	id, err := eorm.SaveRecord("users", user) //根据主键,存在时执行update,不存在时执行insert
	// 或
	id, err := eorm.InsertRecord("users", user) // 执行insert 
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("插入成功，ID:", id)

	// 原生sql插入数据
	_, err = eorm.Exec("INSERT INTO orders (user_id, order_date, total_amount, status) VALUES (?, CURDATE(), ?, 'completed')", 1, 5999.00)
	if err != nil {
		log.Println("插入订单失败: %v", err)
	}

	// 查询数据,查出的数据直接放进Record对象,无需提前定义struct 
	users, err := eorm.Query("SELECT * FROM users where age > ?", 18)
    //下面一行是直接将查询结果缓存
    //users, err := eorm.Cache("user").Query("SELECT * FROM users where age > ?", 18)
	if err != nil {
		log.Fatal(err)
	}
	for _, u := range users {
		fmt.Printf("ID: %d, Name: %s, Age: %d, Email: %s\n",
			u.Int64("id"), u.Str("name"), u.Int("age"), u.Str("email"))
	}

	//  查询1条数据
	record, _ := eorm.QueryFirst("SELECT * FROM users WHERE id = ?", id)
	if record != nil {
		fmt.Printf("姓名: %s, 年龄: %d\n", record.GetString("name"), record.GetInt("age"))
	}

	// 更新数据
	record.Set("age", 18)
	//方法1
	eorm.SaveRecord("users", record) //Save方法,根据主键,存在时执行update,不存在时执行insert 

	//方法2
	_, err := eorm.UpdateRecord("users", record)
 

	// 删除数据
	//方法1
	eorm.DeleteRecord("users", record)
	//方法2
	rows, err = eorm.Delete("users", "id = ?", id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("删除成功，影响行数:", rows)

	// 分页查询,自动执行 select count 得到记录数和总页数等数据
	page := 1
	perPage := 10
	pageObj, err := eorm.Paginate(page, perPage, "SELECT * from tablename where status=?", "id ASC", 1)
	if err != nil {
		log.Printf("分页查询失败: %v", err)
	} else {
		fmt.Printf("  第%d页（共%d页），总条数: %d\n", pageObj.PageNumber, pageObj.TotalPage, pageObj.TotalRow)
		for i, d := range pageObj.List {
			fmt.Printf("    %d. %s (ID: %d)\n", i+1, d.GetString("name"), d.GetInt("id"))
		}
	}
}

   //下面是分页查询的缓存
eorm.Cache("cacheName").Paginate(page, perPage, "SELECT * from tablename where status=?", "id ASC", 1)

   //下面是分页时,只缓存count语句
eorm.WithCountCache(time.Minute*5).Paginate(page, perPage, "SELECT * from tablename where status=?", "id ASC", 1)

```



#### DbModel的基本使用

- Record的优点是太灵活,缺点是在编译期无法检查代码错误. 如果有需要结构体,可以先调用 GenerateDbModel 函数自动生成 结构体  ,然后对结构体进行CRUD

```go
//增
user := &models.User{
    Name: "张三",
    Age:  25,
}
id, err := user.Insert()  // user.Save()

//查
foundUser := &models.User{}
err := foundUser.FindFirst("id = ?", id)

//改
foundUser.Age = 31
foundUser.Update()   // foundUser.Save()

//删
foundUser.Delete()

//查询多条
users, err := user.Find("id>?","id desc",1)
for _, u := range users {
	fmt.Println(u.ToJson())
}

//分页查询
pageObj, err := foundUser.Paginate(1, 10, "select * from user where id>?",1)
if err != nil {
	return
}
fmt.Printf("  第%d页（共%d页），总条数: %d\n", pageObj.PageNumber, pageObj.TotalPage, pageObj.TotalRow)
for _, u := range pageObj.List {
	fmt.Println(u.ToJson())
}

//查询多条
var queryUsers []models.User
err = eorm.QueryToDbModel(&queryUsers, "SELECT * FROM users WHERE age > ?", 25)
// 或 
err = eorm.Table("users").QueryToDbModel(&queryUsers)
```



## 

## 📖 基本使用文档

### 1. 数据库初始化

#### 单数据库配置

```go
// 方式 1：快捷初始化
dsn:="root:123456@tcp(localhost:3306)/test?charset=utf8mb4&parseTime=True&loc=Local"
db, err := eorm.OpenDatabase(eorm.MySQL, dsn, 10)
if err != nil {
    log.Fatal(err)
}

// 方式 2：详细配置
config := &eorm.Config{
    Driver:          eorm.PostgreSQL,
    DSN:             "host=localhost port=5432 user=postgres dbname=test",
    MaxOpen:         50,
    MaxIdle:         25,
    ConnMaxLifetime: time.Hour,
    // 连接监控配置（可选，有默认值）
    MonitorNormalInterval: 60 * time.Second, // 正常检查间隔，默认60秒
    MonitorErrorInterval:  10 * time.Second, // 故障检查间隔，默认10秒
}
db, err = eorm.OpenDatabaseWithConfig(config)
if err != nil {
    log.Fatal(err)
}
```

#### 多数据库管理

```go
// 同时连接多个数据库,第一个注册的数据库为默认数据库
db1, err := eorm.OpenDatabaseWithDBName("main", eorm.MySQL, "root:123456@tcp(localhost:3306)/test?charset=utf8mb4&parseTime=True&loc=Local", 10)
if err != nil {
    log.Fatal(err)
}

db2, err := eorm.OpenDatabaseWithDBName("log_db", eorm.SQLite3, "file:./logs.db", 5)
if err != nil {
    log.Fatal(err)
}

db3, err := eorm.OpenDatabaseWithDBName("oracle", eorm.Oracle, "oracle://test:123456@127.0.0.1:1521/orcl", 25)
if err != nil {
    log.Fatal(err)
}

db4, err := eorm.OpenDatabaseWithDBName("sqlserver", eorm.SQLServer, "sqlserver://sa:123456@127.0.0.1:1433?database=test", 25)
if err != nil {
    log.Fatal(err)
}


// 使用默认数据库进行操作
eorm.Query("...")
// 使用 Use() 直接调用指定数据库并链式调用函数
eorm.Use("main").Query("...")
eorm.Use("oracle").Exec("...")
eorm.Use("sqlserver").SaveRecord("logs", record)

// 获取特定库
db := eorm.Use("main")
db.Query("...")
```

### 2. 查询操作

#### 基本查询

```go
// 操作默认数据库
users,_ := eorm.Query("SELECT * FROM users WHERE status = ?", "active")

// 返回第一条 Record 
user,_ := eorm.QueryFirst("SELECT * FROM users WHERE id = ?", 1)

// 返回 []map[string]interface{}
data,_ := eorm.QueryMap("SELECT name, age FROM users")

// 统计记录
count, _ := eorm.Count("users", "age > ?", 18)


//操作其它数据库用  eorm.Use("main").Query("...")
```

#### 分页查询 (Paginate)

eorm 提供了两种分页查询方式： `Paginate` 方法和 `PaginateBuilder` 方法。

##### 推荐方式：Paginate 方法

使用完整SQL语句进行分页查询，eorm会自动分析SQL并优化 `COUNT(*)` 查询以提高性能。

```go
// 方式 1：操作默认数据库
// 参数：页码, 每页数量, 完整SQL语句, 动态参数
// 返回：分页对象, 错误
pageObj, err := eorm.Paginate(1, 10, "SELECT id, name, age FROM users WHERE age > ? ORDER BY id DESC", 18)

fmt.Printf("  第%d页（共%d页），总条数: %d\n", pageObj.PageNumber, pageObj.TotalPage, pageObj.TotalRow)

// 方式 2：指定数据库
pageObj2, err := eorm.Use("oracle").Paginate(1, 10, "SELECT * FROM users WHERE age > ? ORDER BY id DESC", 18)
```

##### PaginateBuilder 方法

通过分别指定SELECT、表名、WHERE和ORDER BY子句进行分页查询。

```go

// 参数：页码, 每页数量, SELECT 部分, 表名, WHERE 部分, ORDER BY 部分, 动态参数
pageObj, err := eorm.PaginateBuilder(1, 10, "SELECT id, name, age", "users", "age > ?", "id DESC", 18)

// 指定数据库
pageObj2, err := eorm.Use("oracle").PaginateBuilder(1, 10, "SELECT *", "users", "age > ?", "id DESC", 18)
```



#### 链式查询

eorm 提供了一套流畅的链式查询 API，支持全局调用、多数据库调用以及事务内调用。

##### 基本用法

```go
// 查询 age > 18 且 status 为 active 的用户，按创建时间倒序排列，取前 10 条
users, err := eorm.Table("users").
    Where("age > ?", 18).
    Where("status = ?", "active").
    OrderBy("created_at DESC").
    Limit(10).
    Find()

// 查询单条记录
user, err := eorm.Table("users").Where("id = ?", 1).FindFirst()

// 分页查询 (第 1 页，每页 10 条)
page, err := eorm.Table("users").
    Where("age > ?", 18).
    OrderBy("id ASC").
    Paginate(1, 10)
```

##### 高级 WHERE 条件

```go
// OrWhere - OR 条件
orders, err := eorm.Table("orders").
    Where("status = ?", "active").
    OrWhere("priority = ?", "high").
    Find()
// 生成: WHERE (status = ?) OR priority = ?

// WhereInValues - 值列表 IN 查询
users, err := eorm.Table("users").
    WhereInValues("id", []interface{}{1, 2, 3, 4, 5}).
    Find()
// 生成: WHERE id IN (?, ?, ?, ?, ?)

// WhereNotInValues - 值列表 NOT IN 查询
orders, err := eorm.Table("orders").
    WhereNotInValues("status", []interface{}{"cancelled", "refunded"}).
    Find()

// WhereBetween - 范围查询
users, err := eorm.Table("users").
    WhereBetween("age", 18, 65).
    Find()
// 生成: WHERE age BETWEEN ? AND ?

// WhereNull / WhereNotNull - NULL 值检查
users, err := eorm.Table("users").
    WhereNull("deleted_at").
    WhereNotNull("email").
    Find()
// 生成: WHERE deleted_at IS NULL AND email IS NOT NULL
```

##### 分组和聚合

```go
// GroupBy + Having
stats, err := eorm.Table("orders").
    Select("user_id, COUNT(*) as order_count, SUM(total) as total_amount").
    GroupBy("user_id").
    Having("COUNT(*) > ?", 5).
    Find()
// 生成: SELECT ... GROUP BY user_id HAVING COUNT(*) > ?
```

##### 复杂查询示例

```go
// 组合多种条件的复杂查询
results, err := eorm.Table("orders").
    Select("status, COUNT(*) as cnt, SUM(total) as total_amount").
    Where("created_at > ?", "2024-01-01").
    Where("active = ?", 1).
    OrWhere("priority = ?", "high").
    WhereInValues("type", []interface{}{"A", "B", "C"}).
    WhereNotNull("customer_id").
    GroupBy("status").
    Having("COUNT(*) > ?", 10).
    OrderBy("total_amount DESC").
    Limit(20).
    Find()
```

##### 多数据库链式调用

```go
// 在名为 "db2" 的数据库上执行链式查询
logs, err := eorm.Use("db2").Table("logs").
    Where("level = ?", "ERROR").
    OrderBy("id DESC").
    Find()
```

##### 事务中的链式调用

```go
err := eorm.Transaction(func(tx *eorm.Tx) error {
    // 在事务中使用 Table
    user, err := tx.Table("users").Where("id = ?", 1).FindFirst()
    if err != nil {
        return err
    }
    
    // 执行删除
    _, err = tx.Table("logs").Where("user_id = ?", 1).Delete()
    return err
})
```

##### 支持的方法

| 方法 | 说明 |
|------|------|
| `Table(name)` | 指定查询的表名 |
| `Select(columns)` | 指定查询字段，默认为 `*` |
| `Where(condition, args...)` | 添加 WHERE 条件，多次调用使用 `AND` 连接 |
| `And(condition, args...)` | `Where` 的别名 |
| `OrWhere(condition, args...)` | 添加 OR 条件 |
| `WhereInValues(column, values)` | 值列表 IN 查询 |
| `WhereNotInValues(column, values)` | 值列表 NOT IN 查询 |
| `WhereBetween(column, min, max)` | 范围查询 BETWEEN |
| `WhereNotBetween(column, min, max)` | 排除范围 NOT BETWEEN |
| `WhereNull(column)` | IS NULL 检查 |
| `WhereNotNull(column)` | IS NOT NULL 检查 |
| `GroupBy(columns)` | GROUP BY 分组 |
| `Having(condition, args...)` | HAVING 过滤分组结果 |
| `OrderBy(orderBy)` | 指定排序规则 |
| `Limit(limit)` | 指定返回记录数 |
| `Offset(offset)` | 指定偏移量 |
| `Find() / Query()` | 执行查询并返回结果列表 |
| `FindFirst() / QueryFirst()` | 执行查询并返回第一条记录 |
| `Delete()` | 根据条件执行删除（必须带 `Where` 条件） |
| `Paginate(page, pageSize)` | 执行分页查询 |

### 3. 插入与更新

#### Save (自动识别插入或更新)
### `Save` 方法会自动识别主键（自动从数据库元数据获取主键名）。

- 如果 `Record` 中包含主键值且数据库中已存在该记录，则执行 `Update`。
- 如果不包含主键值或记录不存在，则执行 `Insert`。
- 实际生成的sql是upsert语句。

```go
// 情况 1：插入新记录（无主键）
user := eorm.NewRecord().Set("name", "张三").Set("age", 20)
id, err := eorm.SaveRecord("users", user)

// 情况 2：更新记录（带主键）
user.Set("id", 1).Set("name", "张三-已更新")
affected, err := eorm.SaveRecord("users", user)
```

#### Insert 
`执行 `INSERT` 语句，如果主键冲突会返回错误。

```go
user := eorm.NewRecord().Set("name", "李四")
id, err := eorm.InsertRecord("users", user)
```

#### Update 
```go
record := eorm.NewRecord().Set("age", 26)
affected, err := eorm.UpdateRecord("users", record, "id = ?", 1)
```

#### Delete (删除数据)
```go
rows, err := eorm.Delete("users", "id = ?", 10)
 或
eorm.DeleteRecord("users", userRecord)  // userRecord需要含有主键
```

#### 批量插入

```go
var records []*eorm.Record
// ... 填充 records

eorm.BatchInsertRecord("users", records, 500)
```

#### 批量更新

```go
// 根据主键批量更新（Record 中必须包含主键字段）
var records []*eorm.Record
for i := 1; i <= 100; i++ {
    record := eorm.NewRecord().
        Set("id", i).           // 主键
        Set("name", "updated"). // 要更新的字段
        Set("age", 30)
    records = append(records, record)
}
 
// 自定义每批数量
eorm.BatchUpdateRecord("users", records, 50)
```

#### 批量删除

```go
// 方式1：根据 Record 批量删除（Record 中必须包含主键字段）
var records []*eorm.Record
for i := 1; i <= 100; i++ {
    record := eorm.NewRecord().Set("id", i)
    records = append(records, record)
}
eorm.BatchDeleteRecord("users", records)

// 方式2：根据主键ID列表批量删除（仅支持单主键表）
ids := []interface{}{1, 2, 3, 4, 5}
eorm.BatchDeleteByIds("users", ids)


```

### 4. Record 对象详解

`Record` 是 eorm 的核心，它类似于一个增强版的 `map[string]interface{}`。不需要定义结构体即可操作数据库表,Record的字段名不区分大小写。

```go

// 创建 Record 对象
record := eorm.NewRecord().
    Set("name", "李四").
    Set("age", 30).
    Set("email", "lisi@example.com").
    Set("is_vip", true).
    Set("salary", 8000.50)

// 类型安全获取值
name := record.Str("name")       // 获取字符串
age := record.Int("age")         // 获取整数
email := record.Str("email")     // 获取字符串
isVIP := record.Bool("is_vip")   // 获取布尔值
salary := record.Float("salary") // 获取浮点数

// 检查字段是否存在
if record.Has("department") {
    department := record.Str("department")
}

// 获取所有键
keys := record.Keys() // []string{"name", "age", "email", "is_vip", "salary"}

// 转换为 map
recordMap := record.ToMap() // map[string]interface{}

// 转换为 JSON
jsonStr := record.ToJson() // 不返回错误，失败时返回 "{}"

// 从 JSON 创建 Record
newRecord := eorm.NewRecord()
newRecord.FromJson(jsonStr) 

// 删除字段
record.Remove("is_vip")

// 清空所有字段
record.Clear()

```



### 5.DbModel对象及代码生成

除了使用 `Record`，eorm 还支持直接自动生成Struct 进行增删改查。

eorm 提供了一个代码生成器，可以根据数据表结构自动生成结构体（实现IDbModel接口）。

```go
type IDbModel interface {
    TableName() string
    DatabaseName() string
}
```

#### 生成函数

```go
func GenerateDbModel(tablename, outPath, structName string) error
```

- `tablename`: 数据库中的表名。
- `outPath`: 生成的目标路径。
  - 如果以 `.go` 结尾，则视为完整文件路径。
  - 如果是目录路径，则自动以 `表名.go` 作为文件名。
  - 如果为空，默认在 `./models` 目录下生成。
- `structName`: 生成的结构体名称。如果为空，则根据表名自动转换（例如 `users` -> `User`）。

#### 示例

```go
// 1. 指定完整文件路径
eorm.GenerateDbModel("users", "./models/user.go", "User")

// 2. 仅指定目录，文件名将自动生成为 "products.go"
eorm.GenerateDbModel("products", "./models/", "Product")

// 3. 使用默认路径 (./models/orders.go)
eorm.GenerateDbModel("orders", "", "Order")
```

#### 生成内容示例

生成的代码结构如下：

```go

type User struct {
    ID        int64     `column:"id" json:"id"`
    Name      string    `column:"name" json:"name"`
    Age       int64     `column:"age" json:"age"`
    CreatedAt time.Time `column:"created_at" json:"created_at"`
}

// TableName returns the table name for User struct
func (m *User) TableName() string {
    return "users"
}

// DatabaseName returns the database name for User struct
func (m *User) DatabaseName() string {
    return "default"
}

// ToJson converts User to a JSON string
func (m *User) ToJson() string {
	return eorm.ToJson(m)
}

// Save saves the User record (insert or update)
func (m *User) Save() (int64, error) {
	return eorm.Use(m.DatabaseName()).SaveDbModel(m)
}

// ... 其他方法 (Insert, Update, Delete, FindFirst)
```

#### DbModel的使用

##### 1. 插入与保存 (Insert / Save)

- `InsertDbModel(model)`: 直接插入一条记录。
- `SaveDbModel(model)`: 智能插入或更新（如果存在主键冲突则更新）。

```go
user := &models.User{
    Name: "张三",
    Age:  25,
}
//DbModel自带方法
id, err := user.Insert()

//或 ，主键存在执行update， 主键不存在执行insert 
user.Save()   

// 或
id, err := eorm.InsertDbModel(user)

```

##### 2. 更新 (Update)

`UpdateDbModel(model)` 会根据 Struct 中主键字段的值自动更新记录。

```go
user.Age = 30

user.Update()

//或
user.Save()

//或
eorm.UpdateDbModel(user)
```

##### 3. 删除 (Delete)

```
user.Delete()
//或
eorm.DeleteDbModel(user)
```

##### 4. 查询单条 (FindFirst)

```go
user := &models.User{}
err := user.FindFirst("id = ?", 100)

// 或
err := eorm.FindFirstToDbModel(user, "id = ?", 100)

```

##### 5. 查询多条

`FindFirstToDbModel(model, where, args...)` 将查询结果的第一条直接映射到指定的 Struct 中。

```go
user := &models.User{}

//查询多条
users, err := user.Find("id>?","id desc",1)
for _, u := range users {
	fmt.Println(u.ToJson())
}
```

##### 6. 分页查询

```go
user := &models.User{}
pageObj, err := user.Paginate(1, 10, "select * from users where id>? order by id desc",1)
if err != nil {
	return
}

```



### 6. 事务处理

##### 自动事务

`Transaction` 函数会自动处理 `Commit` 和 `Rollback`。只要闭包返回 `error`，事务就会回滚。

```go
err := eorm.Transaction(func(tx *eorm.Tx) error {
    // 注意：在事务中必须使用 tx 对象的方法
    _, err := tx.Exec("UPDATE accounts SET balance = balance - 100 WHERE id = ?", 1)
    if err != nil {
        return err
    }
    
    record := eorm.NewRecord().Set("amount", 100).Set("from_id", 1)
    _, err = tx.Save("transfer_logs", record)
    return err
})
```

##### 手动控制

```go
tx, err := eorm.BeginTransaction()
// ... 执行操作
tx.Commit()   // 或 tx.Rollback()
```

### 日志配置 (Logging)

`eorm` 默认使用 `slog` 输出日志。也可以使用其它日志库。

#### 1. 输出日志到控制台
```go
//  直接开启 Debug 模式会输出 SQL 语句
	eorm.SetDebugMode(true)
```

#### 2. 使用 Zap 日志库

```go


type ZapAdapter struct {
	logger *zap.Logger
}

func (a *ZapAdapter) Log(level eorm.LogLevel, msg string, fields map[string]interface{}) {
	var zapFields []zap.Field
	if len(fields) > 0 {
		zapFields = make([]zap.Field, 0, len(fields))
		for k, v := range fields {
			zapFields = append(zapFields, zap.Any(k, v))
		}
	}

	switch level {
	case eorm.LevelDebug:
		a.logger.Debug(msg, zapFields...)
	case eorm.LevelInfo:
		a.logger.Info(msg, zapFields...)
	case eorm.LevelWarn:
		a.logger.Warn(msg, zapFields...)
	case eorm.LevelError:
		a.logger.Error(msg, zapFields...)
	}
}


func main() {
	// 1. 初始化 zap 日志，同时输出到控制台和文件
	cfg := zap.NewDevelopmentConfig()
	cfg.OutputPaths = []string{"stdout", "logfile.log"}

	zapLogger, _ := cfg.Build()
	defer zapLogger.Sync()

	// 2. 将 zap 集成到 eorm
	eorm.SetLogger(&ZapAdapter{logger: zapLogger})
	eorm.SetDebugMode(true) // 开启调试模式以查看 SQL 轨迹
}
```

#### 3. 使用zerolog
只需实现 `eorm.Logger` 接口即可：
```go
type ZerologAdapter struct {
	logger zerolog.Logger
}

func (a *ZerologAdapter) Log(level eorm.LogLevel, msg string, fields map[string]interface{}) {
	var event *zerolog.Event
	switch level {
	case eorm.LevelDebug:
		event = a.logger.Debug()
	case eorm.LevelInfo:
		event = a.logger.Info()
	case eorm.LevelWarn:
		event = a.logger.Warn()
	case eorm.LevelError:
		event = a.logger.Error()
	default:
		event = a.logger.Log()
	}

	if len(fields) > 0 {
		event.Fields(fields)
	}
	event.Msg(msg)
}

func main() {
// 1. 初始化 zerolog 日志
	// 打开日志文件
	logFile, _ := os.OpenFile("logfile.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer logFile.Close()

	// 2. 链式创建 Logger：同时输出到控制台和文件  
	logger := zerolog.New(zerolog.MultiLevelWriter(
		zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339},
		logFile,
	)).With().Timestamp().Logger()

	// 3. 将 zerolog 集成到 eorm
	eorm.SetLogger(&ZerologAdapter{logger: logger})
	eorm.SetDebugMode(true) // 开启调试模式以查看 SQL 
}
```

### 7. 连接池配置

eorm 自动管理数据库连接池，您可以通过 Config 结构体进行详细配置：

```go
config := &eorm.Config{
    Driver:          eorm.MySQL,
    DSN:             "root:password@tcp(127.0.0.1:3306)/test?charset=utf8mb4",
    MaxOpen:         50,    // 最大打开连接数
    MaxIdle:         25,    // 最大空闲连接数
    ConnMaxLifetime: time.Hour, // 连接最大生命周期
    QueryTimeout:    30 * time.Second, // 默认查询超时时间
    
    // 连接监控配置
    MonitorNormalInterval: 60 * time.Second, // 正常检查间隔（默认60秒）
    MonitorErrorInterval:  10 * time.Second, // 故障检查间隔（默认10秒）
}

db, err := eorm.OpenDatabaseWithConfig(config)
if err != nil {
    log.Fatal(err)
}
```

### 8. 数据库连接监控

eorm 提供数据库连接监控功能,以防止数据库因网络问题意外断开，默认启用，无需额外配置：

```go
// 默认配置，监控自动启用（60秒正常检查，10秒故障重试）
db, err := eorm.OpenDatabase(eorm.MySQL, "user:pass@tcp(localhost:3306)/db", 10)
if err != nil {
    log.Fatal(err)
}

// 自定义监控间隔
config := &eorm.Config{
    Driver:                eorm.MySQL,
    DSN:                   "user:pass@tcp(localhost:3306)/db",
    MaxOpen:               10,
    MonitorNormalInterval: 30 * time.Second, // 30秒正常检查
    MonitorErrorInterval:  5 * time.Second,  // 5秒故障重试
}
db, err = eorm.OpenDatabaseWithConfig(config)
if err != nil {
    log.Fatal(err)
}

// 禁用监控（设置为0）
config.MonitorNormalInterval = 0
```

**监控特点：**
- 自动启用，无需配置
- 智能频率调整：正常60秒，故障10秒
- 多数据库独立监控
- 全局锁避免并发检查
- 只在状态变化时记录日志
- 性能影响极小

### 9. 查询超时控制

eorm 支持全局和单次查询超时设置，使用 Go 标准库的 `context.Context` 实现，超时后自动取消查询。

#### 全局默认超时
```go
config := &eorm.Config{
    Driver:       eorm.MySQL,
    DSN:          "...",
    MaxOpen:      10,
    QueryTimeout: 30 * time.Second,  // 所有查询默认30秒超时
}
eorm.OpenDatabaseWithConfig(config)
```

#### 单次查询超时
```go
// 方式1：全局函数
users, err := eorm.Timeout(5 * time.Second).Query("SELECT * FROM users")

// 方式2：指定数据库
users, err := eorm.Use("mysqldb").Timeout(5 * time.Second).Query("SELECT * FROM users")

// 方式3：链式查询
users, err := eorm.Table("users").
    Where("age > ?", 18).
    Timeout(10 * time.Second).
    Find()
```

#### 事务中设置超时
```go
eorm.Transaction(func(tx *eorm.Tx) error {
    // 事务内的查询也支持超时
    _, err := tx.Timeout(5 * time.Second).Query("SELECT * FROM orders")
    return err
})
```

#### 超时错误处理
```go
import "context"

users, err := eorm.Timeout(1 * time.Second).Query("SELECT SLEEP(5)")
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        fmt.Println("查询超时")
    }
}
```

### 10. 连接池监控

eorm 提供连接池状态监控功能，可以实时查看连接池的使用情况。

#### 获取连接池统计
```go
// 获取默认数据库的连接池统计
stats := eorm.GetPoolStats()
fmt.Println(stats.String())
// 输出: PoolStats[default/mysql]: Open=5 (InUse=2, Idle=3), MaxOpen=10, WaitCount=0, WaitDuration=0s

// 获取指定数据库的连接池统计
stats := eorm.GetPoolStatsDB("postgresql")

// 获取所有数据库的连接池统计
allStats := eorm.AllPoolStats()
for name, stats := range allStats {
    fmt.Printf("%s: %s\n", name, stats.String())
}
```

#### PoolStats 结构体
```go
type PoolStats struct {
    DBName             string        // 数据库名称
    Driver             string        // 驱动类型
    MaxOpenConnections int           // 最大连接数（配置值）
    OpenConnections    int           // 当前打开的连接数
    InUse              int           // 正在使用的连接数
    Idle               int           // 空闲连接数
    WaitCount          int64         // 等待连接的总次数
    WaitDuration       time.Duration // 等待连接的总时长
    MaxIdleClosed      int64         // 因超过最大空闲数而关闭的连接数
    MaxLifetimeClosed  int64         // 因超过最大生命周期而关闭的连接数
}
```

#### 转换为 Map（便于 JSON 序列化）
```go
stats := eorm.GetPoolStats()
statsMap := stats.ToMap()
jsonBytes, _ := json.Marshal(statsMap)
fmt.Println(string(jsonBytes))
```

#### 
输出示例：
```
# HELP eorm_pool_max_open_connections Maximum number of open connections to the database.
# TYPE eorm_pool_max_open_connections gauge
eorm_pool_max_open_connections{db="default",driver="mysql"} 10

# HELP eorm_pool_open_connections The number of established connections both in use and idle.
# TYPE eorm_pool_open_connections gauge
eorm_pool_open_connections{db="default",driver="mysql"} 5

# HELP eorm_pool_in_use The number of connections currently in use.
# TYPE eorm_pool_in_use gauge
eorm_pool_in_use{db="default",driver="mysql"} 2

# HELP eorm_pool_idle The number of idle connections.
# TYPE eorm_pool_idle gauge
eorm_pool_idle{db="default",driver="mysql"} 3
```

### 11. 自动时间戳 (Auto Timestamps)

自动时间戳功能允许在插入和更新记录时自动填充时间戳字段，无需手动设置。

**注意**: eorm 默认关闭自动时间戳检查以获得最佳性能。如需使用此功能，请先启用：

```go
// 启用时间戳自动更新
eorm.EnableTimestampCheck()
```

#### 配置自动时间戳
```go
// 为表配置自动时间戳（使用默认字段名 created_at 和 updated_at）
eorm.ConfigTimestamps("users")

// 使用自定义字段名
eorm.ConfigTimestampsWithFields("orders", "create_time", "update_time")

// 仅配置 created_at
eorm.ConfigCreatedAt("logs", "log_time")

// 仅配置 updated_at
eorm.ConfigUpdatedAt("cache_data", "last_modified")

// 多数据库模式
eorm.Use("main").ConfigTimestamps("users")
```

#### 自动时间戳行为
```go
// 插入数据（created_at 自动填充为当前时间）
record := eorm.NewRecord()
record.Set("name", "John")
record.Set("email", "john@example.com")
eorm.InsertRecord("users", record)
// created_at 自动设置为当前时间

// 更新数据（updated_at 自动填充为当前时间）
updateRecord := eorm.NewRecord()
updateRecord.Set("name", "John Updated")
eorm.UpdateRecord("users", updateRecord)
// updated_at 自动设置为当前时间

// 手动指定 created_at（不会被覆盖）
customTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
record2 := eorm.NewRecord()
record2.Set("name", "Jane")
record2.Set("created_at", customTime)
eorm.InsertRecord("users", record2)
// created_at 保持为 2020-01-01

```

#### 
### 12. 软删除 (Soft Delete)

软删除允许删除记录时只标记为已删除而非物理删除，便于数据恢复和审计。

**注意**: eorm 默认关闭软删除检查以获得最佳性能。如需使用此功能，请先启用：

```go
// 启用软删除功能
eorm.EnableSoftDelete()
```

#### 配置软删除
```go
// 为表配置软删除（时间戳类型，字段为 deleted_at）
eorm.ConfigSoftDelete("users", "deleted_at")

// 使用布尔类型
eorm.ConfigSoftDeleteWithType("posts", "is_deleted", eorm.SoftDeleteBool)

// 多数据库模式
eorm.Use("main").ConfigSoftDelete("users", "deleted_at")
```

#### 软删除操作
```go
// 软删除（自动更新 deleted_at 字段）
eorm.Delete("users", "id = ?", 1)

// 普通查询（自动过滤已删除记录）
users, _ := eorm.Table("users").Find()

// 查询包含已删除记录
allUsers, _ := eorm.Table("users").WithTrashed().Find()

// 只查询已删除记录
deletedUsers, _ := eorm.Table("users").OnlyTrashed().Find()

// 恢复已删除记录
eorm.Restore("users", "id = ?", 1)

// 物理删除（真正删除数据）
eorm.ForceDelete("users", "id = ?", 1)
```

#### 原生 SQL 软删除过滤
eorm 提供了 `QueryWithOutTrashed` 和 `QueryFirstWithOutTrashed` 函数，可以对任意原生 SQL 查询自动添加软删除过滤条件：

```go
// 原生 SQL 查询自动过滤软删除数据
users, err := eorm.QueryWithOutTrashed("SELECT * FROM users WHERE age > ?", 18)
// 原始 SQL: SELECT * FROM users WHERE age > ?
// 自动转换为: SELECT * FROM users WHERE age > ? AND deleted_at IS NULL

// 查询第一条记录
user, err := eorm.QueryFirstWithOutTrashed("SELECT * FROM users WHERE email = ?", "test@example.com")

// 多表 JOIN 查询自动处理
posts, err := eorm.QueryWithOutTrashed(`
    SELECT p.*, u.name as author_name 
    FROM posts p 
    JOIN users u ON p.user_id = u.id 
    WHERE p.status = ?
`, "published")
// 自动为配置了软删除的表添加过滤条件

// 支持多数据库和事务
posts, err := eorm.Use("main").QueryWithOutTrashed("SELECT * FROM posts", )
err := eorm.Transaction(func(tx *eorm.Tx) error {
    users, err := tx.QueryWithOutTrashed("SELECT * FROM users")
    return err
})
```



#### 链式调用
```go
// 软删除
eorm.Table("users").Where("id = ?", 1).Delete()

// 恢复
eorm.Table("users").Where("id = ?", 1).Restore()

// 物理删除
eorm.Table("users").Where("id = ?", 1).ForceDelete()

// 统计（自动过滤已删除）
count, _ := eorm.Table("users").Count()

// 统计（包含已删除）
count, _ := eorm.Table("users").WithTrashed().Count()
```

#### DbModel 软删除
```go
// 生成的 DbModel 自动包含软删除方法
user.Delete()       // 软删除
user.ForceDelete()  // 物理删除
user.Restore()      // 恢复

// 查询方法
users, _ := user.FindWithTrashed("status = ?", "id DESC", "active")
deletedUsers, _ := user.FindOnlyTrashed("", "id DESC")
```

### 13. 乐观锁 (Optimistic Lock)

乐观锁通过版本号字段检测并发更新冲突，防止数据被意外覆盖。

#### 配置乐观锁
```go
// 为表配置乐观锁（默认字段名 version）
eorm.ConfigOptimisticLock("products")

// 使用自定义字段名
eorm.ConfigOptimisticLockWithField("orders", "revision")

// 多数据库模式
eorm.Use("main").ConfigOptimisticLock("products")
```

#### 乐观锁操作
```go
// 插入数据（version 自动初始化为 1）
record := eorm.NewRecord().Set("name", "Laptop").Set("price", 999.99)
eorm.InsertRecord("products", record)

// 更新数据（带版本号）
updateRecord := eorm.NewRecord()
updateRecord.Set("version", int64(1))  // 当前版本
updateRecord.Set("price", 899.99)
rows, err := eorm.UpdateRecord("products", updateRecord)
// 成功：version 自动递增为 2

// 并发冲突检测（使用过期版本）
staleRecord := eorm.NewRecord()
staleRecord.Set("version", int64(1))  // 过期版本！
staleRecord.Set("price", 799.99)
rows, err = eorm.UpdateRecord("products", staleRecord)
if errors.Is(err, eorm.ErrVersionMismatch) {
    fmt.Println("检测到并发冲突，记录已被其他事务修改")
}

// 正确处理并发：先读取最新版本
latestRecord, _ := eorm.Table("products").Where("id = ?", 1).FindFirst()
currentVersion := latestRecord.GetInt("version")

updateRecord2 := eorm.NewRecord()
updateRecord2.Set("version", currentVersion)
updateRecord2.Set("price", 799.99)
eorm.Update("products", updateRecord2, "id = ?", 1)
```

#### 事务中使用乐观锁
```go
eorm.Transaction(func(tx *eorm.Tx) error {
    rec, _ := tx.Table("products").Where("id = ?", 1).FindFirst()
    currentVersion := rec.GetInt("version")
    
    updateRec := eorm.NewRecord()
    updateRec.Set("version", currentVersion)
    updateRec.Set("stock", 80)
    _, err := tx.UpdateRecord("products", updateRec)
    return err  // 版本冲突时自动回滚
})
```

### 14. SQL 模板 (SQL Templates)

eorm 提供了强大的 SQL 模板功能，允许您将 SQL 语句配置化管理，支持动态参数、条件构建和多数据库执行。

📖 **[查看完整 SQL 模板使用指南](doc/cn/SQL_TEMPLATE_GUIDE.md)** - 包含详细的配置格式、参数类型、动态SQL构建、最佳实践等内容。

#### 配置文件结构

SQL 模板使用 JSON 格式的配置文件：

```json
{
  "version": "1.0",
  "description": "用户服务SQL配置",
  "namespace": "user_service",
  "sqls": [
    {
      "name": "findById",
      "description": "根据ID查找用户",
      "sql": "SELECT * FROM users WHERE id = ?",
      "type": "select"
    },
    {
      "name": "findByIdAndStatus",
      "description": "根据ID和状态查找用户",
      "sql": "SELECT * FROM users WHERE id = ? AND status = ?",
      "type": "select"
    },
    {
      "name": "updateUser",
      "description": "更新用户信息",
      "sql": "UPDATE users SET name = ?, email = ?, age = ? WHERE id = ?",
      "type": "update"
    }
  ]
}
```

#### 参数类型支持

eorm SQL 模板支持多种参数传递方式：

| 参数类型 | 适用场景 | SQL 占位符 | 示例 |
|---------|---------|-----------|------|
| `map[string]interface{}` | 命名参数 | `:name` | `map[string]interface{}{"id": 123}` |
| `[]interface{}` | 多个位置参数 | `?` | `[]interface{}{123, "John"}` |
| 单个简单类型 | 单个位置参数 | `?` | `123`, `"John"`, `true` |
| **🆕 可变参数** | **多个位置参数** | `?` | `SqlTemplate(name, 123, "John", true)` |

#### 配置加载

```go
// 加载单个配置文件
err := eorm.LoadSqlConfig("config/user_service.json")

// 加载多个配置文件
configPaths := []string{
    "config/user_service.json",
    "config/order_service.json",
}
err := eorm.LoadSqlConfigs(configPaths)

// 加载目录下所有 JSON 配置文件
err := eorm.LoadSqlConfigDir("config/")
```

#### SQL 模板执行

```go
// 1. 单个简单参数
user, err := eorm.SqlTemplate("user_service.findById", 123).QueryFirst()

// 2. 可变参数（推荐用于多参数查询）
users, err := eorm.SqlTemplate("user_service.findByIdAndStatus", 123, 1).Query()

// 3. 更新操作
result, err := eorm.SqlTemplate("user_service.updateUser", 
    "John Doe", "john@example.com", 30, 123).Exec()

// 4. 分页查询（新增功能）
pageObj, err := eorm.SqlTemplate("user_service.findActiveUsers", 1).Paginate(1, 10)
if err == nil {
    fmt.Printf("第%d页（共%d页），总条数: %d\n", 
        pageObj.PageNumber, pageObj.TotalPage, pageObj.TotalRow)
    for _, user := range pageObj.List {
        fmt.Printf("用户: %s\n", user.Str("name"))
    }
}

// 5. 命名参数（适用于复杂查询）
params := map[string]interface{}{
    "name": "John",
    "status": 1,
}
users, err := eorm.SqlTemplate("user_service.findByNamedParams", params).Query()

// 6. 位置参数数组（向后兼容）
users, err := eorm.SqlTemplate("user_service.findByIdAndStatus", 
    []interface{}{123, 1}).Query()
```

#### 多数据库和事务支持

```go
// 指定数据库执行
users, err := eorm.Use("mysql").SqlTemplate("findUsers", 123, 1).Query()

// 指定数据库执行分页查询
pageObj, err := eorm.Use("mysql").SqlTemplate("findUsers", 123, 1).Paginate(1, 20)

// 事务中使用
err := eorm.Transaction(func(tx *eorm.Tx) error {
    result, err := tx.SqlTemplate("insertUser", "John", "john@example.com", 25).Exec()
    return err
})

// 事务中使用分页查询
err := eorm.Transaction(func(tx *eorm.Tx) error {
    pageObj, err := tx.SqlTemplate("findOrders", userId).Paginate(1, 10)
    if err != nil {
        return err
    }
    // 处理分页结果...
    return nil
})

// 设置超时
users, err := eorm.SqlTemplate("findUsers", 123).
    Timeout(30 * time.Second).Query()

// 分页查询设置超时
pageObj, err := eorm.SqlTemplate("complexQuery", params).
    Timeout(30 * time.Second).
    Paginate(1, 50)
```

#### 参数数量验证

系统会自动验证参数数量与 SQL 占位符数量是否匹配：

```go
// ✅ 正确：2个参数匹配2个占位符
users, err := eorm.SqlTemplate("findByIdAndStatus", 123, 1).Query()

// ❌ 错误：参数不足
users, err := eorm.SqlTemplate("findByIdAndStatus", 123).Query()
// 返回: parameter count mismatch: SQL has 2 '?' placeholders but got 1 parameters

// ❌ 错误：参数过多
users, err := eorm.SqlTemplate("findByIdAndStatus", 123, 1, 2).Query()
// 返回: parameter count mismatch: SQL has 2 '?' placeholders but got 3 parameters
```

#### 动态 SQL 构建

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
      "sql": " AND status = ?"
    },
    {
      "name": "ageMin",
      "type": "int", 
      "desc": "最小年龄",
      "sql": " AND age >= ?"
    }
  ],
  "order": "created_at DESC"
}
```

```go
// 只传入部分参数，系统会自动构建相应的 SQL
params := map[string]interface{}{
    "status": 1,
    // ageMin 未提供，对应的条件不会被添加
}
users, err := eorm.SqlTemplate("searchUsers", params).Query()
// 生成的 SQL: SELECT * FROM users WHERE 1=1 AND status = ? ORDER BY created_at DESC
```

#### 最佳实践

1. **单参数查询** - 使用 `?` 占位符和简单参数
2. **多参数查询** - 使用可变参数或命名参数
3. **复杂查询** - 使用命名参数和动态 SQL
4. **参数验证** - 系统自动验证参数数量和类型
5. **错误处理** - 捕获并处理 `SqlConfigError` 类型的错误

### 缓存支持

eorm 提供灵活的缓存策略，支持本地缓存和 Redis 缓存，你可以根据场景自由选择。

#### 1. 三种缓存使用方式

```go
// 方式 1：显式使用本地缓存（速度最快，单实例）
user, _ := eorm.LocalCache("user_cache_store").QueryFirst("SELECT * FROM users WHERE id = ?", 1)

// 方式 2：显式使用 Redis 缓存（分布式共享）
order, _ := eorm.RedisCache("order_cache_store").Query("SELECT * FROM orders WHERE user_id = ?", userId)

// 方式 3：使用默认缓存（默认为本地缓存，可通过 SetDefaultCache 切换）
data, _ := eorm.Cache("default_cache_store").QueryFirst("SELECT * FROM configs WHERE key = ?", key)
```

#### 2. 初始化缓存

```go
// 本地缓存（已默认初始化，可选配置清理间隔）
eorm.InitLocalCache(1 * time.Minute)

// Redis 缓存（需要先引入 eorm/redis 子包）
import "github.com/zzguang83325/eorm/redis"

rc, err := redis.NewRedisCache("localhost:6379", "", "password", 0)
if err != nil {
    panic(err)
}
eorm.InitRedisCache(rc)

// 可选：切换默认缓存为 Redis
eorm.SetDefaultCache(rc)
```

#### 3. 使用场景

```go
// 场景 1：配置数据用本地缓存（快速访问，很少变化）
configs, _ := eorm.LocalCache("config_cache_store", 10*time.Minute).
    Query("SELECT * FROM configs")

// 场景 2：业务数据用 Redis 缓存（多实例共享）
orders, _ := eorm.RedisCache("order_cache_store", 5*time.Minute).
    Query("SELECT * FROM orders WHERE user_id = ?", userId)

// 场景 3：混合使用
func GetDashboardData(userID int) (*Dashboard, error) {
    // 配置用本地缓存
    configs, _ := eorm.LocalCache("configs_store").Query("SELECT * FROM configs")
    
    // 用户数据用 Redis
    user, _ := eorm.RedisCache("users_store").QueryFirst("SELECT * FROM users WHERE id = ?", userID)
    
    return &Dashboard{Configs: configs, User: user}, nil
}
```

#### 4. 手动缓存操作

eorm 提供三套缓存操作函数：

**默认缓存操作**（操作当前默认缓存）：

```go
// 存储缓存
eorm.CacheSet("my_store", "key1", "value1", 5*time.Minute)

// 获取缓存
val, ok := eorm.CacheGet("my_store", "key1")

// 删除指定键
eorm.CacheDelete("my_store", "key1")

// 清空指定存储库
eorm.CacheClearRepository("my_store")

// 查看状态
status := eorm.CacheStatus()
```

**本地缓存操作**（直接操作本地缓存）：
```go
// 存储到本地缓存
eorm.LocalCacheSet("config", "key1", "value1", 10*time.Minute)

// 从本地缓存获取
val, ok := eorm.LocalCacheGet("config", "key1")

// 删除本地缓存键
eorm.LocalCacheDelete("config", "key1")

// 清空本地缓存存储库
eorm.LocalCacheClearRepository("config")

// 查看本地缓存状态
status := eorm.LocalCacheStatus()
```

**Redis 缓存操作**（直接操作 Redis 缓存）：
```go
// 存储到 Redis
err := eorm.RedisCacheSet("session", "key1", "value1", 30*time.Minute)

// 从 Redis 获取
val, ok, err := eorm.RedisCacheGet("session", "key1")

// 删除 Redis 键
err = eorm.RedisCacheDelete("session", "key1")

// 清空 Redis 存储库
err = eorm.RedisCacheClearRepository("session")

// 查看 Redis 状态
status, err := eorm.RedisCacheStatus()
```

#### 5. 清空所有缓存

```go
// 清空本地缓存的所有存储库
eorm.LocalCacheClearAll()

// 清空 Redis 缓存的所有存储库
err := eorm.RedisCacheClearAll()
if err != nil {
    log.Printf("清空失败: %v", err)
}

// 清空默认缓存的所有存储库
eorm.ClearAllCaches()
```

#### 6. 查看缓存状态

```go
// 查看默认缓存状态
status := eorm.CacheStatus()
fmt.Printf("类型: %v\n", status["type"])
fmt.Printf("总项数: %v\n", status["total_items"])
fmt.Printf("内存: %v\n", status["estimated_memory_human"])

// 查看本地缓存状态
localStatus := eorm.LocalCacheStatus()
fmt.Printf("本地缓存项数: %v\n", localStatus["total_items"])

// 查看 Redis 缓存状态
redisStatus, err := eorm.RedisCacheStatus()
if err == nil {
    fmt.Printf("Redis 地址: %v\n", redisStatus["address"])
    fmt.Printf("数据库大小: %v\n", redisStatus["db_size"])
}
```

#### 7. 性能对比

| 缓存类型 | 延迟 | 吞吐量 | 分布式 | 使用场景 |
|---------|------|--------|--------|----------|
| 本地缓存 | ~1μs | 极高 | ✗ | 配置、字典、单实例 |
| Redis 缓存 | ~1ms | 高 | ✓ | 业务数据、多实例 |

更多示例请参考：[examples/cache_local_redis](examples/cache_local_redis)




## 🔗 项目链接

GitHub 仓库：[https://github.com/zzguang83325/eorm.git](https://github.com/zzguang83325/eorm.git)