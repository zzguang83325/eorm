# EORM - Go Database Library

[English](README_EN.md) | [API Manual](api.md) | [API Reference](api_en.md) | [SQL Template Guide](doc/cn/SQL_TEMPLATE_GUIDE.md) | [SQL Template Guide](doc/en/SQL_TEMPLATE_GUIDE_EN.md) | [Cache Usage Guide](doc/cn/CACHE_ENHANCEMENT_GUIDE.md) | [Cache Usage Guide](doc/en/CACHE_ENHANCEMENT_GUIDE.md)

EORM (easy orm) is a high-performance database ORM framework based on Go. It provides a simple and intuitive API and flexible Record objects, allowing you to perform CRUD operations on the database without defining Structs.

**Project Link**: https://github.com/zzguang83325/eorm.git

## Features

- **Database Support**: Supports MySQL, PostgreSQL, SQLite, SQL Server, Oracle
- **Multi-Database Management**: Supports connecting to multiple databases simultaneously and easily switching between them
- **Record Object**: Get rid of tedious Struct definitions, use flexible `Record` for CRUD operations, inspired by JFinal
- **DbModel Experience**: In addition to Record objects, you can perform CRUD operations through auto-generated DbModel objects
- **SQL Templates**: Supports SQL configuration management, dynamic parameter building, and variadic parameters - [Detailed Guide](doc/cn/SQL_TEMPLATE_GUIDE.md)
- **Transaction Support**: Provides simple and easy-to-use transaction wrappers and low-level transaction control
- **Security Protection**: Built-in SQL security validator to defend against SQL injection, XSS and other attacks, supports SELECT syntax whitelist and dangerous pattern detection
- **Smart Caching**: 
  - **Result Caching**: Supports memory and Redis caching, provides chain API
  - **Statement Caching**: Automatic LRU statement cache (Statement Cache), significantly improving query performance under high concurrency
- **Connection Monitoring**: Automatically monitors database connection status, supports fault detection and automatic reconnection, ensuring high service availability
- **Pagination Queries**: Pagination query implementation optimized for different databases, one function can get record count, total pages and current page data
- **Logging**: Built-in logging system, supports detailed SQL execution time analysis
- **Auto Timestamp**: Supports configuring auto timestamp fields, automatically fills created_at and updated_at when inserting and updating
- **Soft Delete Support**: Supports configuring soft delete fields (timestamp/boolean), automatically filters deleted records, provides restore and physical delete functions
- **Optimistic Lock Support**: Supports configuring version fields, automatically detects concurrent conflicts, prevents data overwriting

## Installation

```
go get github.com/zzguang83325/eorm@latest
```

## Database Drivers

eorm supports the following databases. You need to install the corresponding driver according to the database you use.

| Database     | Driver Package                                                                       |
| ---------- | -------------------------------- | --------------------------------------------------- |
| MySQL      | github.com/go-sql-driver/mysql    |
| PostgreSQL | github.com/jackc/pgx/v5/stdlib   |
| SQLite3    | github.com/mattn/go-sqlite3      |
| SQL Server | github.com/denisenkom/go-mssqldb |
| Oracle     | github.com/sijms/go-ora/v2       |

eorm has encapsulated the above drivers, you can directly import them in your code:

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

## Quick Start

```go
package main

import (
	"fmt"
	"log"
	"github.com/zzguang83325/eorm"
	_ "github.com/zzguang83325/eorm/drivers/mysql" // MySQL driver

)

func main() {
	// Initialize database connection, eorm supports multiple databases, the first opened database is the default database
	db, err := eorm.OpenDatabase(eorm.MySQL, "root:password@tcp(localhost:3306)/test?charset=utf8mb4&parseTime=True&loc=Local", 10)
	if err != nil {
		log.Fatal(err)
	}
	defer eorm.Close()

	records, err := db.Query("SELECT * FROM users")
	if err != nil {
		log.Fatal(err)
	}

	// Test connection
	if err := eorm.Ping(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Database connection successful")

	// Do not specify db, directly operate on the default database
	eorm.Exec(`CREATE TABLE IF NOT EXISTS users (
        id INT AUTO_INCREMENT PRIMARY KEY,
        name VARCHAR(100) NOT NULL,
        age INT NOT NULL,
        email VARCHAR(100) NOT NULL UNIQUE
    )`)

	// Create Record and insert data
	user := eorm.NewRecord().
		Set("name", "Zhang San").
		Set("age", 25).
		Set("email", "zhangsan@example.com")

	id, err := eorm.SaveRecord("users", user) // Based on primary key, execute update if exists, otherwise execute insert
	// or
	id, err := eorm.InsertRecord("users", user) // Execute insert
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Insert successful, ID:", id)

	// Native SQL insert data
	_, err = eorm.Exec("INSERT INTO orders (user_id, order_date, total_amount, status) VALUES (?, CURDATE(), ?, 'completed')", 1, 5999.00)
	if err != nil {
		log.Println("Insert order failed: %v", err)
	}

	// Query data, queried data is directly put into Record object, no need to define struct in advance
	users, err := eorm.Query("SELECT * FROM users where age > ?", 18)
    // The line below directly caches the query result
    //users, err := eorm.Cache("user").Query("SELECT * FROM users where age > ?", 18)
	if err != nil {
		log.Fatal(err)
	}
	for _, u := range users {
		fmt.Printf("ID: %d, Name: %s, Age: %d, Email: %s\n",
			u.Int64("id"), u.Str("name"), u.Int("age"), u.Str("email"))
	}

	// Query 1 record
	record, _ := eorm.QueryFirst("SELECT * FROM users WHERE id = ?", id)
	if record != nil {
		fmt.Printf("Name: %s, Age: %d\n", record.GetString("name"), record.GetInt("age"))
	}

	// Update data
	record.Set("age", 18)
	// Method 1
	eorm.SaveRecord("users", record) // Save method, based on primary key, execute update if exists, otherwise execute insert

	// Method 2
	_, err := eorm.UpdateRecord("users", record)

	// Delete data
	// Method 1
	eorm.DeleteRecord("users", record)
	// Method 2
	rows, err = eorm.Delete("users", "id = ?", id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Delete successful, affected rows:", rows)

	// Pagination query, automatically execute select count to get record count, total pages and other data
	page := 1
	perPage := 10
	pageObj, err := eorm.Paginate(page, perPage, "SELECT * from tablename where status=?", "id ASC", 1)
	if err != nil {
		log.Printf("Pagination query failed: %v", err)
	} else {
		fmt.Printf("  Page %d (Total %d pages), Total records: %d\n", pageObj.PageNumber, pageObj.TotalPage, pageObj.TotalRow)
		for i, d := range pageObj.List {
			fmt.Printf("    %d. %s (ID: %d)\n", i+1, d.GetString("name"), d.GetInt("id"))
		}
	}
}

   // Below is pagination query caching
eorm.Cache("cacheName").Paginate(page, perPage, "SELECT * from tablename where status=?", "id ASC", 1)

   // Below is pagination, only cache count statement
eorm.WithCountCache(time.Minute*5).Paginate(page, perPage, "SELECT * from tablename where status=?", "id ASC", 1)
```

#### Basic Usage of DbModel

- The advantage of Record is that it is too flexible, the disadvantage is that code errors cannot be checked at compile time. If you need structs, you can first call GenerateDbModel function to automatically generate structs, then perform CRUD on structs

```go
// Insert
user := &models.User{
    Name: "Zhang San",
    Age:  25,
}
id, err := user.Insert()  // user.Save()

// Query
foundUser := &models.User{}
err := foundUser.FindFirst("id = ?", id)

// Update
foundUser.Age = 31
foundUser.Update()   // foundUser.Save()

// Delete
foundUser.Delete()

// Query multiple records
users, err := user.Find("id>?","id desc",1)
for _, u := range users {
	fmt.Println(u.ToJson())
}

// Pagination query
pageObj, err := foundUser.Paginate(1, 10, "select * from user where id>?",1)
if err != nil {
	return
}
fmt.Printf("  Page %d (Total %d pages), Total records: %d\n", pageObj.PageNumber, pageObj.TotalPage, pageObj.TotalRow)
for _, u := range pageObj.List {
	fmt.Println(u.ToJson())
}

// Query multiple records
var queryUsers []models.User
err = eorm.QueryToDbModel(&queryUsers, "SELECT * FROM users WHERE age > ?", 25)
// or
err = eorm.Table("users").QueryToDbModel(&queryUsers)
```

## 

## 📖 Basic Usage Documentation

### 1. Database Initialization

#### Single Database Configuration

```go
// Method 1: Quick initialization
dsn:="root:123456@tcp(localhost:3306)/test?charset=utf8mb4&parseTime=True&loc=Local"
db, err := eorm.OpenDatabase(eorm.MySQL, dsn, 10)
if err != nil {
    log.Fatal(err)
}

// Method 2: Detailed configuration
config := &eorm.Config{
    Driver:          eorm.PostgreSQL,
    DSN:             "host=localhost port=5432 user=postgres dbname=test",
    MaxOpen:         50,
    MaxIdle:         25,
    ConnMaxLifetime: time.Hour,
    // Connection monitoring configuration (optional, has default values)
    MonitorNormalInterval: 60 * time.Second, // Normal check interval, default 60 seconds
    MonitorErrorInterval:  10 * time.Second, // Error check interval, default 10 seconds
}
db, err = eorm.OpenDatabaseWithConfig(config)
if err != nil {
    log.Fatal(err)
}
```

#### Multi-Database Management

```go
// Connect to multiple databases simultaneously, the first registered database is the default database
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

// Use default database for operations
eorm.Query("...")
// Use Use() to directly call specified database and chain functions
eorm.Use("main").Query("...")
eorm.Use("oracle").Exec("...")
eorm.Use("sqlserver").SaveRecord("logs", record)

// Get specific database
db := eorm.Use("main")
db.Query("...")
```

### 2. Query Operations

#### Basic Queries

```go
// Operate on default database
users,_ := eorm.Query("SELECT * FROM users WHERE status = ?", "active")

// Return first Record
user,_ := eorm.QueryFirst("SELECT * FROM users WHERE id = ?", 1)

// Return []map[string]interface{}
data,_ := eorm.QueryMap("SELECT name, age FROM users")

// Count records
count, _ := eorm.Count("users", "age > ?", 18)

// Operate on other databases using eorm.Use("main").Query("...")
```

#### Pagination Query (Paginate)

eorm provides two pagination query methods: `Paginate` method and `PaginateBuilder` method.

##### Recommended Method: Paginate Method

Use complete SQL statement for pagination query, eorm will automatically analyze SQL and optimize `COUNT(*)` query to improve performance.

```go
// Method 1: Operate on default database
// Parameters: page number, records per page, complete SQL statement, dynamic parameters
// Returns: pagination object, error
pageObj, err := eorm.Paginate(1, 10, "SELECT id, name, age FROM users WHERE age > ? ORDER BY id DESC", 18)

fmt.Printf("  Page %d (Total %d pages), Total records: %d\n", pageObj.PageNumber, pageObj.TotalPage, pageObj.TotalRow)

// Method 2: Specify database
pageObj2, err := eorm.Use("oracle").Paginate(1, 10, "SELECT * FROM users WHERE age > ? ORDER BY id DESC", 18)
```

##### PaginateBuilder Method

Perform pagination query by specifying SELECT, table name, WHERE and ORDER BY clauses separately.

```go
// Parameters: page number, records per page, SELECT part, table name, WHERE part, ORDER BY part, dynamic parameters
pageObj, err := eorm.PaginateBuilder(1, 10, "SELECT id, name, age", "users", "age > ?", "id DESC", 18)

// Specify database
pageObj2, err := eorm.Use("oracle").PaginateBuilder(1, 10, "SELECT *", "users", "age > ?", "id DESC", 18)
```

#### Chain Queries

eorm provides a fluent chain query API, supporting global calls, multi-database calls, and calls within transactions.

##### Basic Usage

```go
// Query users with age > 18 and status active, sorted by creation time in descending order, take first 10 records
users, err := eorm.Table("users").
    Where("age > ?", 18).
    Where("status = ?", "active").
    OrderBy("created_at DESC").
    Limit(10).
    Find()

// Query single record
user, err := eorm.Table("users").Where("id = ?", 1).FindFirst()

// Pagination query (page 1, 10 records per page)
page, err := eorm.Table("users").
    Where("age > ?", 18).
    OrderBy("id ASC").
    Paginate(1, 10)
```

##### Advanced WHERE Conditions

```go
// OrWhere - OR condition
orders, err := eorm.Table("orders").
    Where("status = ?", "active").
    OrWhere("priority = ?", "high").
    Find()
// Generates: WHERE (status = ?) OR priority = ?

// WhereInValues - Value list IN query
users, err := eorm.Table("users").
    WhereInValues("id", []interface{}{1, 2, 3, 4, 5}).
    Find()
// Generates: WHERE id IN (?, ?, ?, ?, ?)

// WhereNotInValues - Value list NOT IN query
orders, err := eorm.Table("orders").
    WhereNotInValues("status", []interface{}{"cancelled", "refunded"}).
    Find()

// WhereBetween - Range query
users, err := eorm.Table("users").
    WhereBetween("age", 18, 65).
    Find()
// Generates: WHERE age BETWEEN ? AND ?

// WhereNull / WhereNotNull - NULL value check
users, err := eorm.Table("users").
    WhereNull("deleted_at").
    WhereNotNull("email").
    Find()
// Generates: WHERE deleted_at IS NULL AND email IS NOT NULL
```

##### Grouping and Aggregation

```go
// GroupBy + Having
stats, err := eorm.Table("orders").
    Select("user_id, COUNT(*) as order_count, SUM(total) as total_amount").
    GroupBy("user_id").
    Having("COUNT(*) > ?", 5).
    Find()
// Generates: SELECT ... GROUP BY user_id HAVING COUNT(*) > ?
```

##### Complex Query Example

```go
// Complex query combining multiple conditions
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

##### Multi-Database Chain Calls

```go
// Execute chain query on database named "db2"
logs, err := eorm.Use("db2").Table("logs").
    Where("level = ?", "ERROR").
    OrderBy("id DESC").
    Find()
```

##### Chain Calls in Transaction

```go
err := eorm.Transaction(func(tx *eorm.Tx) error {
    // Use Table in transaction
    user, err := tx.Table("users").Where("id = ?", 1).FindFirst()
    if err != nil {
        return err
    }
    
    // Execute delete
    _, err = tx.Table("logs").Where("user_id = ?", 1).Delete()
    return err
})
```

##### Supported Methods

| Method | Description |
|------|------|
| `Table(name)` | Specify table name for query |
| `Select(columns)` | Specify query fields, default is `*` |
| `Where(condition, args...)` | Add WHERE condition, multiple calls use `AND` connection |
| `And(condition, args...)` | Alias of `Where` |
| `OrWhere(condition, args...)` | Add OR condition |
| `WhereInValues(column, values)` | Value list IN query |
| `WhereNotInValues(column, values)` | Value list NOT IN query |
| `WhereBetween(column, min, max)` | Range query BETWEEN |
| `WhereNotBetween(column, min, max)` | Exclude range NOT BETWEEN |
| `WhereNull(column)` | IS NULL check |
| `WhereNotNull(column)` | IS NOT NULL check |
| `GroupBy(columns)` | GROUP BY grouping |
| `Having(condition, args...)` | HAVING filter grouped results |
| `OrderBy(orderBy)` | Specify sort rules |
| `Limit(limit)` | Specify number of returned records |
| `Offset(offset)` | Specify offset |
| `Find() / Query()` | Execute query and return result list |
| `FindFirst() / QueryFirst()` | Execute query and return first record |
| `Delete()` | Execute delete based on condition (must have `Where` condition) |
| `Paginate(page, pageSize)` | Execute pagination query |

### 3. Insert and Update

#### Save (Auto-recognize Insert or Update)

The `Save` method automatically recognizes the primary key (automatically obtained from database metadata).

- If `Record` contains primary key value and the record exists in the database, execute `Update`.
- If it does not contain primary key value or the record does not exist, execute `Insert`.
- The actual generated SQL is an upsert statement.

```go
// Case 1: Insert new record (no primary key)
user := eorm.NewRecord().Set("name", "Zhang San").Set("age", 20)
id, err := eorm.SaveRecord("users", user)

// Case 2: Update record (with primary key)
user.Set("id", 1).Set("name", "Zhang San-Updated")
affected, err := eorm.SaveRecord("users", user)
```

#### Insert

Execute `INSERT` statement, returns error if primary key conflicts.

```go
user := eorm.NewRecord().Set("name", "Li Si")
id, err := eorm.InsertRecord("users", user)
```

#### Update

```go
record := eorm.NewRecord().Set("age", 26)
affected, err := eorm.UpdateRecord("users", record, "id = ?", 1)
```

#### Delete (Delete Data)

```go
rows, err := eorm.Delete("users", "id = ?", 10)
or
eorm.DeleteRecord("users", userRecord)  // userRecord needs to contain primary key
```

#### Batch Insert

```go
var records []*eorm.Record
// ... fill records

eorm.BatchInsertRecord("users", records, 500)
```

#### Batch Update

```go
// Batch update based on primary key (Record must contain primary key field)
var records []*eorm.Record
for i := 1; i <= 100; i++ {
    record := eorm.NewRecord().
        Set("id", i).           // Primary key
        Set("name", "updated"). // Fields to update
        Set("age", 30)
    records = append(records, record)
}

// Custom batch size
eorm.BatchUpdateRecord("users", records, 50)
```

#### Batch Delete

```go
// Method 1: Batch delete based on Record (Record must contain primary key field)
var records []*eorm.Record
for i := 1; i <= 100; i++ {
    record := eorm.NewRecord().Set("id", i)
    records = append(records, record)
}
eorm.BatchDeleteRecord("users", records)

// Method 2: Batch delete based on primary key ID list (only supports single primary key tables)
ids := []interface{}{1, 2, 3, 4, 5}
eorm.BatchDeleteByIds("users", ids)
```

### 4. Record Object 

`Record` is the core of eorm, it is like an enhanced version of `map[string]interface{}`. No need to define structs to operate on database tables, Record field names are case-insensitive.

```go
// Create Record object
record := eorm.NewRecord().
    Set("name", "Li Si").
    Set("age", 30).
    Set("email", "lisi@example.com").
    Set("is_vip", true).
    Set("salary", 8000.50)

// Type-safe value retrieval
name := record.Str("name")       // Get string
age := record.Int("age")         // Get integer
email := record.Str("email")     // Get string
isVIP := record.Bool("is_vip")   // Get boolean
salary := record.Float("salary") // Get float

// Check if field exists
if record.Has("department") {
    department := record.Str("department")
}

// Get all keys
keys := record.Keys() // []string{"name", "age", "email", "is_vip", "salary"}

// Convert to map
recordMap := record.ToMap() // map[string]interface{}

// Convert to JSON
jsonStr := record.ToJson() // Does not return error, returns "{}" on failure

// Create Record from JSON
newRecord := eorm.NewRecord()
newRecord.FromJson(jsonStr)

// Remove field
record.Remove("is_vip")

// Clear all fields
record.Clear()
```

### 5. DbModel Object and Code Generation

In addition to using `Record`, eorm also supports directly auto-generating Structs for CRUD operations.

eorm provides a code generator that can automatically generate structs (implementing IDbModel interface) based on database table structure.

```go
type IDbModel interface {
    TableName() string
    DatabaseName() string
}
```

#### Generation Function

```go
func GenerateDbModel(tablename, outPath, structName string) error
```

- `tablename`: Table name in the database.
- `outPath`: Target path for generation.
  - If ends with `.go`, it is treated as a complete file path.
  - If it is a directory path, automatically use `table_name.go` as the file name.
  - If empty, generate in `./models` directory by default.
