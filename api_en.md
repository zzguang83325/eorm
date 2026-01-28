# EORM API Documentation

EORM is a high-performance ORM framework based on Go, using a design pattern similar to JFinal ActiveRecord. It supports multiple databases, caching, transactions, pagination, SQL templates, and other features.

## Table of Contents

- [Database Connection](#database-connection)
- [Database Connection Monitoring](#database-connection-monitoring)
- [Query Timeout Control](#query-timeout-control)
- [Basic Queries](#basic-queries)
- [Record Object Operations](#record-object-operations)
- [DbModel Operations](#dbmodel-operations)
- [Chain Queries](#chain-queries)
- [Transaction Operations](#transaction-operations)
- [Batch Operations](#batch-operations)
- [Pagination Queries](#pagination-queries)
- [Cache Functions](#cache-functions)
- [SQL Templates](#sql-templates)
- [Log Configuration](#log-configuration)
- [Enhanced Features Configuration](#enhanced-features-configuration)
- [Feature Support Description](#feature-support-description)
- [Utility Functions](#utility-functions)

---

## Database Connection

### OpenDatabase
```go
func OpenDatabase(driver DriverType, dsn string, maxOpen int) (*DB, error)
```
Open a database connection with default configuration.

**Example:**
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
Open a database connection with a specified name (multi-database mode).

**Example:**
```go
db1, err := eorm.OpenDatabaseWithDBName("db1", eorm.MySQL, "root:password@tcp(localhost:3306)/db1", 10)
db2, err := eorm.OpenDatabaseWithDBName("db2", eorm.PostgreSQL, "postgres://user:password@localhost/db2", 10)
```

### OpenDatabaseWithConfig
```go
func OpenDatabaseWithConfig(dbname string, config *Config) (*DB, error)
```
Open a database connection with full configuration.

**Config struct:**
```go
type Config struct {
    Driver          DriverType    // Database driver type
    DSN             string        // Data source name
    MaxOpen         int           // Maximum open connections
    MaxIdle         int           // Maximum idle connections
    ConnMaxLifetime time.Duration // Connection maximum lifetime
    QueryTimeout    time.Duration // Default query timeout (0 means no limit)
    
    // Connection monitoring configuration
    MonitorNormalInterval time.Duration // Normal check interval (default 60 seconds, 0 means disable monitoring)
    MonitorErrorInterval  time.Duration // Error check interval (default 10 seconds)
}
```

**Example:**
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
Switch to the database with the specified name, suitable for chain calls.

**Example:**
```go
// Use different databases
eorm.Use("db1").Query("SELECT * FROM users")
eorm.Use("db2").Query("SELECT * FROM products")
```

### UseWithError
```go
func UseWithError(dbname string) (*DB, error)
```
Switch to the specified database, return error information if there is an error.

---

## Basic Queries

### Query
```go
func Query(querySQL string, args ...interface{}) ([]*Record, error)
func (db *DB) Query(querySQL string, args ...interface{}) ([]*Record, error)
func (tx *Tx) Query(querySQL string, args ...interface{}) ([]*Record, error)
```
Execute SQL query and return Record slice.

**Example:**
```go
// Global function
records, err := eorm.Query("SELECT * FROM users WHERE age > ?", 18)

// DB instance method
records, err := db.Query("SELECT * FROM users WHERE age > ?", 18)

// Query in transaction
tx.Query("SELECT * FROM users WHERE age > ?", 18)
```

### QueryFirst
```go
func QueryFirst(querySQL string, args ...interface{}) (*Record, error)
func (db *DB) QueryFirst(querySQL string, args ...interface{}) (*Record, error)
func (tx *Tx) QueryFirst(querySQL string, args ...interface{}) (*Record, error)
```
Execute SQL query and return the first record.

**Example:**
```go
record, err := eorm.QueryFirst("SELECT * FROM users WHERE id = ?", 1)
if record != nil {
    fmt.Println("Username:", record.GetString("name"))
}
```

### QueryMap
```go
func QueryMap(querySQL string, args ...interface{}) ([]map[string]interface{}, error)
func (db *DB) QueryMap(querySQL string, args ...interface{}) ([]map[string]interface{}, error)
func (tx *Tx) QueryMap(querySQL string, args ...interface{}) ([]map[string]interface{}, error)
```
Execute SQL query and return map slice.

**Example:**
```go
maps, err := eorm.QueryMap("SELECT name, age FROM users")
for _, m := range maps {
    fmt.Printf("Name: %v, Age: %v\n", m["name"], m["age"])
}
```

### QueryToDbModel
```go
func QueryToDbModel(dest interface{}, querySQL string, args ...interface{}) error
func (db *DB) QueryToDbModel(dest interface{}, querySQL string, args ...interface{}) error
func (tx *Tx) QueryToDbModel(dest interface{}, querySQL string, args ...interface{}) error
```
Execute query and map results to struct slice.

**Example:**
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
Execute query and map the first record to struct.

**Example:**
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
Execute query and automatically filter soft-deleted records.

**Example:**
```go
// Automatically filter soft-deleted users
records, err := eorm.QueryWithOutTrashed("SELECT * FROM users")
```

### QueryFirstWithOutTrashed
```go
func QueryFirstWithOutTrashed(querySQL string, args ...interface{}) (*Record, error)
func (db *DB) QueryFirstWithOutTrashed(querySQL string, args ...interface{}) (*Record, error)
func (tx *Tx) QueryFirstWithOutTrashed(querySQL string, args ...interface{}) (*Record, error)
```
Execute query and return the first non-soft-deleted record.

### Exec
```go
func Exec(querySQL string, args ...interface{}) (sql.Result, error)
func (db *DB) Exec(querySQL string, args ...interface{}) (sql.Result, error)
func (tx *Tx) Exec(querySQL string, args ...interface{}) (sql.Result, error)
```
Execute SQL commands (INSERT, UPDATE, DELETE).

**Example:**
```go
result, err := eorm.Exec("UPDATE users SET age = ? WHERE id = ?", 25, 1)
affected, _ := result.RowsAffected()
fmt.Printf("Affected rows: %d\n", affected)
```

---

## Record Object Operations

### NewRecord
```go
func NewRecord() *Record
```
Create a new Record instance.

**Example:**
```go
record := eorm.NewRecord()
record.Set("name", "Zhang San")
record.Set("age", 25)
record.Set("email", "zhangsan@example.com")
```

### SaveRecord
```go
func SaveRecord(table string, record *Record) (int64, error)
func (db *DB) SaveRecord(table string, record *Record) (int64, error)
func (tx *Tx) SaveRecord(table string, record *Record) (int64, error)
```
Save record (update if primary key exists, otherwise insert).

**Feature Support:** ✅ Auto Timestamp | ✅ Optimistic Lock | ❌ Soft Delete

**Example:**
```go
record := eorm.NewRecord()
record.Set("id", 1)
record.Set("name", "Li Si")
record.Set("age", 30)

id, err := eorm.SaveRecord("users", record)
```

### InsertRecord
```go
func InsertRecord(table string, record *Record) (int64, error)
func (db *DB) InsertRecord(table string, record *Record) (int64, error)
func (tx *Tx) InsertRecord(table string, record *Record) (int64, error)
```
Insert a new record.

**Example:**
```go
record := eorm.NewRecord()
record.Set("name", "Wang Wu")
record.Set("age", 28)

id, err := eorm.InsertRecord("users", record)
```

### UpdateRecord
```go
func UpdateRecord(table string, record *Record) (int64, error)
func (db *DB) UpdateRecord(table string, record *Record) (int64, error)
func (tx *Tx) UpdateRecord(table string, record *Record) (int64, error)
```
Update record by primary key. If auto-timestamp is configured, it will automatically update the corresponding updated time field.

**Example:**
```go
record := eorm.NewRecord()
record.Set("id", 1)
record.Set("name", "Zhang San Updated")
record.Set("age", 26)

affected, err := eorm.UpdateRecord("users", record)
```

### Update
```go
func Update(table string, record *Record, whereSql string, whereArgs ...interface{}) (int64, error)
func (db *DB) Update(table string, record *Record, whereSql string, whereArgs ...interface{}) (int64, error)
func (tx *Tx) Update(table string, record *Record, whereSql string, whereArgs ...interface{}) (int64, error)
```
Update records by condition. If auto-timestamp is configured, it will automatically update the corresponding updated time field.

**Example:**
```go
record := eorm.NewRecord()
record.Set("age", 30)

affected, err := eorm.Update("users", record, "name = ?", "Zhang San")
```

### UpdateFast
```go
func UpdateFast(table string, record *Record, whereSql string, whereArgs ...interface{}) (int64, error)
func (db *DB) UpdateFast(table string, record *Record, whereSql string, whereArgs ...interface{}) (int64, error)
func (tx *Tx) UpdateFast(table string, record *Record, whereSql string, whereArgs ...interface{}) (int64, error)
```
Fast update, skip timestamp and optimistic lock checks.

**Example:**
```go
record := eorm.NewRecord()
record.Set("status", "active")

// High-performance update, skip timestamp and optimistic lock checks
affected, err := eorm.UpdateFast("users", record, "id = ?", 1)
```

### DeleteRecord
```go
func DeleteRecord(table string, record *Record) (int64, error)
func (db *DB) DeleteRecord(table string, record *Record) (int64, error)
func (tx *Tx) DeleteRecord(table string, record *Record) (int64, error)
```
Delete record by primary key. If soft delete is configured, it performs an update.

**Example:**
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
Delete records by condition. If soft delete is configured, it performs an update.

**Example:**
```go
// Soft delete (if soft delete is configured)
affected, err := eorm.Delete("users", "age < ?", 18)
```

### Record Object Methods

#### Record.Set
```go
func (r *Record) Set(column string, value interface{}) *Record
```
Set field value, supports chain calls.

**Example:**
```go
record := eorm.NewRecord()
record.Set("name", "Zhang San").Set("age", 25).Set("email", "zhangsan@example.com")
```

#### Record.Get
```go
func (r *Record) Get(column string) interface{}
```
Get field value.

#### Type-safe Get Methods
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

// Shorthand methods (backward compatible, no error return)
func (r *Record) Str(column string) string
func (r *Record) Int(column string) int
func (r *Record) Int64(column string) int64
func (r *Record) Float(column string) float64
func (r *Record) Bool(column string) bool
```

**Example:**
```go
record, _ := eorm.QueryFirst("SELECT * FROM users WHERE id = ?", 1)

name := record.GetString("name")
age := record.GetInt("age")
email := record.Str("email")  // Shorthand method
isActive := record.Bool("is_active")
```

#### Record.Has
```go
func (r *Record) Has(column string) bool
```
Check if field exists.

**Example:**
```go
if record.Has("email") {
    fmt.Println("Email:", record.GetString("email"))
}
```

#### Record.Keys
```go
func (r *Record) Keys() []string
```
Get all field names.

**Example:**
```go
keys := record.Keys()
fmt.Println("Field list:", keys)
```

#### Record.Columns
```go
func (r *Record) Columns() []string
```
Get all field names (alias method, same as Keys).

**Example:**
```go
columns := record.Columns()
fmt.Println("Field list:", columns)
```

#### Record.Remove
```go
func (r *Record) Remove(column string)
```
Remove field.

**Example:**
```go
record.Remove("password")  // Remove sensitive field
```

#### Record.Clear
```go
func (r *Record) Clear()
```
Clear all fields.

**Example:**
```go
record.Clear()  // Clear record
```

#### Record.ToMap
```go
func (r *Record) ToMap() map[string]interface{}
```
Convert to map.

**Example:**
```go
dataMap := record.ToMap()
fmt.Printf("Data: %+v\n", dataMap)
```

#### Record.ToJson
```go
func (r *Record) ToJson() string
```
Convert to JSON string.

**Example:**
```go
jsonStr := record.ToJson()
fmt.Println("JSON:", jsonStr)
```

#### Record.FromJson
```go
func (r *Record) FromJson(jsonStr string) *Record
```
Parse from JSON string and merge into current Record.

**Example:**
```go
record := eorm.NewRecord()
record.FromJson(`{"name":"Zhang San","age":25}`).FromJson(`{"address":"Beijing","email":"zhangsan@example.com"}`)
```

#### Record.ToStruct
```go
func (r *Record) ToStruct(dest interface{}) error
```
Convert to struct.

**Example:**
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
Fill from struct.

**Example:**
```go
user := User{Name: "Li Si", Age: 30}
info := Info{Address: "Beijing", Email: "lisi@example.com"}
record := eorm.NewRecord()
record.FromStruct(user).FromStruct(info)
```

#### Record.FromRecord
```go
func (r *Record) FromRecord(src *Record) *Record
```
Populate current Record from another Record, supports chain calls. Uses deep copy to ensure nested objects are completely copied.

**Example:**
```go
sourceRecord := eorm.NewRecord().
    FromJson(`{"id": 1, "name": "Zhang San", "age": 25}`)

// Copy data from sourceRecord to new Record
record := eorm.NewRecord().
    FromRecord(sourceRecord).
    Set("email", "zhangsan@example.com").
    Set("active", true)

fmt.Println(record.ToJson())
// Output: {"id":1,"name":"Zhang San","age":25,"email":"zhangsan@example.com","active":true}
```

#### Record.GetRecord
```go
func (r *Record) GetRecord(column string) (*Record, error)
```
Get nested Record.

**Example:**
```go
record := eorm.NewRecord().FromJson(`{
    "user": {
        "name": "Zhang San",
        "profile": {
            "age": 25
        }
    }
}`)

user, err := record.GetRecord("user")
if err != nil {
    // Handle error
} else {
    fmt.Println("Name:", user.GetString("name"))
    profile, _ := user.GetRecord("profile")
    fmt.Println("Age:", profile.GetInt("age"))
}
```

#### Record.GetRecords
```go
func (r *Record) GetRecords(column string) ([]*Record, error)
```
Get nested Record array.

**Example:**
```go
record := eorm.NewRecord().FromJson(`{
    "users": [
        {"name": "Zhang San", "age": 25},
        {"name": "Li Si", "age": 30}
    ]
}`)

users, err := record.GetRecords("users")
if err != nil {
    // Handle error
} else {
    for _, user := range users {
        fmt.Printf("Name: %s, Age: %d\n", user.GetString("name"), user.GetInt("age"))
    }
}
```

#### Record.GetRecordByPath
```go
func (r *Record) GetRecordByPath(path string) (*Record, error)
```
Get nested Record by dot-separated path.

**Example:**
```go
record := eorm.NewRecord().FromJson(`{
    "data": {
        "user": {
            "profile": {
                "name": "Zhang San",
                "age": 25
            }
        }
    }
}`)

profile, err := record.GetRecordByPath("data.user.profile")
if err != nil {
    // Handle error
} else {
    fmt.Printf("Name: %s, Age: %d\n", profile.GetString("name"), profile.GetInt("age"))
}
```

#### Record.GetStringByPath
```go
func (r *Record) GetStringByPath(path string) (string, error)
```
Get nested string value by dot-separated path.

**Example:**
```go
record := eorm.NewRecord().FromJson(`{
    "user": {
        "name": "Zhang San",
        "contact": {
            "email": "zhangsan@example.com",
            "phone": "13800138000"
        }
    }
}`)

email, err := record.GetStringByPath("user.contact.email")
if err != nil {
    // Handle error
} else {
    fmt.Println("Email:", email)
}

// If path points to a Record, it will return JSON string
contact, err := record.GetStringByPath("user.contact")
if err != nil {
    // Handle error
} else {
    fmt.Println("Contact:", contact)  // Output in JSON format
}
```

#### Record.String
```go
func (r *Record) String() string
```
Implement Stringer interface, returns JSON format string.

**Example:**
```go
record := eorm.NewRecord().Set("name", "Zhang San").Set("age", 25)
fmt.Println(record)  // Directly output JSON format
fmt.Printf("%v\n", record)  // Using %v will also call String() method
```

---

## DbModel Operations

### SaveDbModel
```go
func SaveDbModel(model IDbModel) (int64, error)
func (db *DB) SaveDbModel(model IDbModel) (int64, error)
func (tx *Tx) SaveDbModel(model IDbModel) (int64, error)
```
Save DbModel instance.

**Example:**
```go
user := &User{
    ID:   1,
    Name: "Zhang San",
    Age:  25,
}

id, err := eorm.SaveDbModel(user)
// or
id, err := user.Save()
```

### InsertDbModel
```go
func InsertDbModel(model IDbModel) (int64, error)
func (db *DB) InsertDbModel(model IDbModel) (int64, error)
func (tx *Tx) InsertDbModel(model IDbModel) (int64, error)
```
Insert DbModel instance.

**Example:**
```go
user := &User{
    Name: "Li Si",
    Age:  30,
}

id, err := eorm.InsertDbModel(user)
// or
id, err := user.Insert()
```

### UpdateDbModel
```go
func UpdateDbModel(model IDbModel) (int64, error)
func (db *DB) UpdateDbModel(model IDbModel) (int64, error)
func (tx *Tx) UpdateDbModel(model IDbModel) (int64, error)
```
Update DbModel instance.

**Example:**
```go
user.Age = 31
affected, err := eorm.UpdateDbModel(user)
// or
affected, err := user.Update()
```

### DeleteDbModel
```go
func DeleteDbModel(model IDbModel) (int64, error)
func (db *DB) DeleteDbModel(model IDbModel) (int64, error)
func (tx *Tx) DeleteDbModel(model IDbModel) (int64, error)
```
Delete DbModel instance.

**Example:**
```go
affected, err := eorm.DeleteDbModel(user)
// or
affected, err := user.Delete()
```

### FindFirstToDbModel
```go
func FindFirstToDbModel(model IDbModel, whereSql string, whereArgs ...interface{}) error
func (db *DB) FindFirstToDbModel(model IDbModel, whereSql string, whereArgs ...interface{}) error
func (tx *Tx) FindFirstToDbModel(model IDbModel, whereSql string, whereArgs ...interface{}) error
```
Find the first record and map to DbModel.

**Example:**
```go
user := &User{}
err := eorm.FindFirstToDbModel(user, "name = ?", "Zhang San")
```

### FindToDbModel
```go
func FindToDbModel(dest interface{}, table string, whereSql string, orderBySql string, whereArgs ...interface{}) error
func (db *DB) FindToDbModel(dest interface{}, table string, whereSql string, orderBySql string, whereArgs ...interface{}) error
func (tx *Tx) FindToDbModel(dest interface{}, table string, whereSql string, orderBySql string, whereArgs ...interface{}) error
```
Find multiple records and map to struct slice.

**Example:**
```go
var users []User
err := eorm.FindToDbModel(&users, "users", "age > ?", "age DESC", 18)
```

### GenerateDbModel
```go
func GenerateDbModel(tablename, outPath, structName string) error
func (db *DB) GenerateDbModel(tablename, outPath, structName string) error
```
Generate Go struct code based on database table.

**Parameters:**
- `tablename`: Table name
- `outPath`: Output path (directory or full file path)
- `structName`: Struct name (empty means auto-generate)

**Example:**
```go
// Generate User struct to specified file
err := eorm.GenerateDbModel("users", "models/user.go", "User")

// Auto-generate struct name
err := eorm.GenerateDbModel("products", "models/", "")
```

### GetTableColumns
```go
func GetTableColumns(table string) ([]ColumnInfo, error)
func (db *DB) GetTableColumns(table string) ([]ColumnInfo, error)
```
Get all column information for the specified table.

**Return value:**
- `[]ColumnInfo`: Column information slice, containing column name, type, nullable, primary key, comment, etc.

**ColumnInfo struct:**
```go
type ColumnInfo struct {
    Name     string // Column name
    Type     string // Data type
    Nullable bool   // Is nullable
    IsPK     bool   // Is primary key
    Comment  string // Column comment
}
```

**Example:**
```go
// Use global function
columns, err := eorm.GetTableColumns("users")
if err != nil {
    log.Fatal(err)
}

for _, col := range columns {
    fmt.Printf("Column: %s, Type: %s, Nullable: %v, Primary Key: %v\n", 
        col.Name, col.Type, col.Nullable, col.IsPK)
    if col.Comment != "" {
        fmt.Printf("  Comment: %s\n", col.Comment)
    }
}

// Use DB instance method
columns, err := db.GetTableColumns("orders")

// Multi-database mode
columns, err := eorm.Use("db2").GetTableColumns("products")
```

### GetAllTables
```go
func GetAllTables() ([]string, error)
func (db *DB) GetAllTables() ([]string, error)
```
Get all table names in the database.

**Return value:**
- `[]string`: List of table names

**Example:**
```go
// Use global function
tables, err := eorm.GetAllTables()
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Database has %d tables:\n", len(tables))
for i, table := range tables {
    fmt.Printf("%d. %s\n", i+1, table)
}

// Use DB instance method
tables, err := db.GetAllTables()

// Multi-database mode
tables, err := eorm.Use("db2").GetAllTables()
```

**Supported databases:**
- MySQL: Use `INFORMATION_SCHEMA.TABLES`
- PostgreSQL: Use `pg_catalog.pg_tables`
- SQLite3: Use `sqlite_master`
- Oracle: Use `USER_TABLES`
- SQL Server: Use `INFORMATION_SCHEMA.TABLES`

### GenerateAllDbModel
```go
func GenerateAllDbModel(outPath string) (int, error)
func (db *DB) GenerateAllDbModel(outPath string) (int, error)
```
Batch generate Model code for all tables in the database.

**Parameters:**
- `outPath`: Output directory path (empty string uses default "models" directory)

**Return value:**
- `int`: Number of successfully generated files
- `error`: Error information (if some tables fail to generate, returns error with detailed error information)

**Example:**
```go
// Use global function, generate to models directory
count, err := eorm.GenerateAllDbModel("models")
if err != nil {
    log.Printf("Error occurred during generation: %v\n", err)
}
fmt.Printf("Successfully generated %d Model files\n", count)

// Use DB instance method
count, err := db.GenerateAllDbModel("generated")

// Use default path
count, err := eorm.GenerateAllDbModel("")

// Multi-database mode - generate to different directories for different databases
db1, _ := eorm.OpenDatabase(eorm.MySQL, dsn1, 10)
db2, _ := eorm.OpenDatabaseWithDBName("db2", eorm.MySQL, dsn2, 10)

count1, _ := db1.GenerateAllDbModel("models/db1")
count2, _ := db2.GenerateAllDbModel("models/db2")

fmt.Printf("Database 1 generated %d files\n", count1)
fmt.Printf("Database 2 generated %d files\n", count2)
```

**Features:**
- Automatically get all tables in the database
- Generate independent Go file for each table
- Struct name automatically converted from table name (e.g., `user_info` → `UserInfo`)
- File name is lowercase table name (e.g., `user_info.go`)
- Fault tolerance: failure to generate one table does not affect other tables
- Return success count and detailed error information

**Generated file structure:**
```
models/
├── users.go      // User struct
├── orders.go     // Order struct
└── products.go   // Product struct
```

**Error handling:**
```go
count, err := eorm.GenerateAllDbModel("models")
if err != nil {
    // Even if some tables fail, err contains detailed error information
    // count is the number of successfully generated files
    log.Printf("Generation completed, %d files succeeded, some errors: %v\n", count, err)
}
```

---

## Enhanced Features

EORM framework provides three major enhanced features: **Auto Timestamp**, **Optimistic Lock**, **Soft Delete**. Different functions have different levels of support for these features.

### Detailed Description of Enhanced Features

#### 🕒 Auto Timestamp Feature

**Supported operations:**
- **INSERT operation**: Automatically set `created_at` field to current time
- **UPDATE operation**: Automatically set `updated_at` field to current time
- **UPSERT operation**: Set `created_at` on INSERT, set `updated_at` on UPDATE

**Supported functions:**
```go
// ✅ Fully supported
eorm.InsertRecord("users", record)     // Automatically set created_at
eorm.UpdateRecord("users", record)     // Automatically set updated_at  
eorm.Update("users", record, "id = ?", 1) // Automatically set updated_at
eorm.SaveRecord("users", record)       // Set created_at on INSERT, set updated_at on UPDATE

// ✅ DbModel methods
eorm.InsertDbModel(user)               // Automatically set created_at
eorm.UpdateDbModel(user)               // Automatically set updated_at
eorm.SaveDbModel(user)                 // Set corresponding timestamp based on operation type

// ✅ Chain queries
eorm.Table("users").Where("id = ?", 1).Update(record) // Automatically set updated_at

// ✅ Batch operations
eorm.BatchInsertRecord("users", records) // Set created_at for each record
eorm.BatchUpdateRecord("users", records) // Set updated_at for each record

// ❌ Not supported
eorm.Query("INSERT INTO users ...")    // Raw SQL, does not handle timestamps
eorm.UpdateFast("users", record, "id = ?", 1) // Explicitly skip timestamp check
```

**Configuration example:**
```go
// Enable timestamp feature
eorm.EnableTimestamps()

// Configure table timestamp fields
eorm.ConfigTimestampsWithFields("users", "created_at", "updated_at")
eorm.ConfigTimestampsWithFields("orders", "create_time", "update_time")
```

#### 🔒 Optimistic Lock Feature

**Supported operations:**
- **UPDATE operation**: Check version number, automatically increment version number on update
- **INSERT operation**: Automatically initialize version number to 1
- **UPSERT operation**: Initialize version number on INSERT, check and increment version number on UPDATE

**Supported functions:**
```go
// ✅ Fully supported
record := eorm.NewRecord()
record.Set("id", 1)
record.Set("name", "Updated name")
record.Set("version", 5) // Current version number

eorm.UpdateRecord("users", record)     // Check version number, increment to 6 on success
eorm.Update("users", record, "id = ?", 1) // Check version number, increment to 6 on success
eorm.SaveRecord("users", record)       // Check or initialize version number based on operation

// ✅ DbModel methods
user.Version = 5
eorm.UpdateDbModel(user)               // Check version number, increment to 6 on success

// ✅ Chain queries
eorm.Table("users").Where("id = ?", 1).Update(record) // Check version number

// ❌ Not supported
eorm.UpdateFast("users", record, "id = ?", 1) // Explicitly skip optimistic lock check
```

**Version conflict handling:**
```go
// When version number does not match, returns ErrVersionMismatch error
affected, err := eorm.UpdateRecord("users", record)
if errors.Is(err, eorm.ErrVersionMismatch) {
    fmt.Println("Record has been modified by another user, please fetch latest data")
}
```

**Configuration example:**
```go
// Enable optimistic lock feature
eorm.EnableOptimisticLock()

// Configure table version field
eorm.ConfigOptimisticLockWithField("users", "version")
eorm.ConfigOptimisticLockWithField("products", "revision")
```

#### 🗑️ Soft Delete Feature

**Supported operations:**
- **DELETE operation**: Do not physically delete records, but set delete marker field
- **Query operation**: Automatically filter soft-deleted records
- **Count operation**: Automatically exclude soft-deleted records

**Supported functions:**
```go
// ✅ Delete operations (soft delete)
eorm.Delete("users", "id = ?", 1)      // Set deleted_at field
eorm.DeleteRecord("users", record)     // Set deleted_at field
eorm.DeleteDbModel(user)               // Set deleted_at field

// ✅ Chain delete
eorm.Table("users").Where("status = ?", "inactive").Delete() // Soft delete

// ✅ Batch delete
eorm.BatchDeleteRecord("users", records) // Batch soft delete
eorm.BatchDeleteByIds("users", ids)      // Batch soft delete

// ✅ Automatically filter when querying
eorm.Table("users").Find()              // Automatically exclude soft-deleted records
eorm.Table("users").Count()             // Exclude soft-deleted records when counting
eorm.Table("users").Exists()            // Exclude soft-deleted records when checking
eorm.Table("users").Paginate(1, 10)     // Exclude soft-deleted records when paginating

// ✅ Special queries
eorm.QueryWithOutTrashed("SELECT * FROM users") // Explicitly filter soft-deleted records
eorm.QueryFirstWithOutTrashed("SELECT * FROM users WHERE id = ?", 1)

// ✅ Queries including soft-deleted records
eorm.Table("users").WithTrashed().Find() // Include soft-deleted records
eorm.Table("users").OnlyTrashed().Find() // Only query soft-deleted records
eorm.Table("users").WithTrashed().Paginate(1, 10) // Paginate including soft-deleted records

// ❌ Not supported (raw SQL)
eorm.Query("SELECT * FROM users")       // Does not automatically filter soft-deleted records
eorm.QueryFirst("SELECT * FROM users WHERE id = ?", 1) // Does not automatically filter
eorm.Paginate(1, 10, "SELECT * FROM users") // Raw SQL pagination, does not automatically filter
eorm.PaginateBuilder(1, 10, "*", "users", "", "") // Does not automatically filter
```

**Soft delete types:**
```go
// Timestamp type (default)
eorm.ConfigSoftDeleteWithType("users", "deleted_at", eorm.SoftDeleteTimestamp)

// Boolean type
eorm.ConfigSoftDeleteWithType("products", "is_deleted", eorm.SoftDeleteBool)
```

**Configuration example:**
```go
// Enable soft delete feature
eorm.EnableSoftDelete()

// Configure table soft delete field
eorm.ConfigSoftDelete("users", "deleted_at")           // Timestamp type
eorm.ConfigSoftDelete("products", "is_deleted")        // Default timestamp type
eorm.ConfigSoftDeleteWithType("orders", "is_deleted", eorm.SoftDeleteBool) // Boolean type
```

### Enhanced Features Support in Multi-Database Mode

Each database can be independently configured and enabled with features:

```go
// Enable different features for different databases
eorm.Use("user_db").EnableTimestamps().ConfigTimestamps("users")
eorm.Use("product_db").EnableOptimisticLock().ConfigOptimisticLock("products")
eorm.Use("log_db").EnableSoftDelete().ConfigSoftDelete("logs")

// Automatically apply corresponding database features when using
eorm.Use("user_db").UpdateRecord("users", userRecord)     // Supports timestamp
eorm.Use("product_db").UpdateRecord("products", productRecord) // Supports optimistic lock
eorm.Use("log_db").Delete("logs", "level = ?", "debug")   // Supports soft delete
```

### Performance Considerations

1. **Feature check overhead**: Enabling features adds a small performance overhead
2. **Batch operation optimization**: Batch operation functions have optimized feature processing, performance is better than loop calling single operations
3. **Soft delete query**: Soft delete adds WHERE condition in queries, may affect query performance, recommend building index on delete field

### Best Practices

1. **Unified enable**: Recommend enabling required features uniformly when application starts
2. **Table-level configuration**: Configure field names separately for each table that needs features
3. **Version number handling**: Query to get latest version number before updating records
4. **Soft delete index**: Build database index on soft delete field to improve query performance
5. **Feature combination**: Three features can be used simultaneously without conflict

---

### Enhanced Features Configuration

### Auto Timestamp

#### EnableTimestamps
```go
func EnableTimestamps()
func (db *DB) EnableTimestamps() *DB
```
Enable auto timestamp feature.

**Example:**
```go
// Global enable
eorm.EnableTimestamps()

// Enable for specific database
eorm.Use("db1").EnableTimestamps()
```

#### ConfigTimestamps
```go
func ConfigTimestamps(table string)
func (db *DB) ConfigTimestamps(table string) *DB
```
Configure default timestamp fields for table (created_at, updated_at).

**Example:**
```go
eorm.ConfigTimestamps("users")
```

#### ConfigTimestampsWithFields
```go
func ConfigTimestampsWithFields(table, createdAtField, updatedAtField string)
func (db *DB) ConfigTimestampsWithFields(table, createdAtField, updatedAtField string) *DB
```
Configure custom timestamp fields for table.

**Example:**
```go
eorm.ConfigTimestampsWithFields("users", "create_time", "update_time")
```

#### ConfigCreatedAt
```go
func ConfigCreatedAt(table, field string)
func (db *DB) ConfigCreatedAt(table, field string) *DB
```
Configure only create time field.

**Example:**
```go
eorm.ConfigCreatedAt("logs", "created_at")
```

#### ConfigUpdatedAt
```go
func ConfigUpdatedAt(table, field string)
func (db *DB) ConfigUpdatedAt(table, field string) *DB
```
Configure only update time field.

**Example:**
```go
eorm.ConfigUpdatedAt("users", "last_modified")
```

#### RemoveTimestamps
```go
func RemoveTimestamps(table string)
func (db *DB) RemoveTimestamps(table string) *DB
```
Remove table timestamp configuration.

#### HasTimestamps
```go
func HasTimestamps(table string) bool
func (db *DB) HasTimestamps(table string) bool
```
Check if table has timestamp configured.

### Optimistic Lock

#### EnableOptimisticLock
```go
func EnableOptimisticLock()
func (db *DB) EnableOptimisticLock() *DB
```
Enable optimistic lock feature.

**Example:**
```go
eorm.EnableOptimisticLock()
```

#### ConfigOptimisticLock
```go
func ConfigOptimisticLock(table string)
func (db *DB) ConfigOptimisticLock(table string) *DB
```
Configure default version field for table (version).

**Example:**
```go
eorm.ConfigOptimisticLock("products")
```

#### ConfigOptimisticLockWithField
```go
func ConfigOptimisticLockWithField(table, versionField string)
func (db *DB) ConfigOptimisticLockWithField(table, versionField string) *DB
```
Configure custom version field for table.

**Example:**
```go
eorm.ConfigOptimisticLockWithField("products", "revision")
```

#### RemoveOptimisticLock
```go
func RemoveOptimisticLock(table string)
func (db *DB) RemoveOptimisticLock(table string) *DB
```
Remove table optimistic lock configuration.

#### HasOptimisticLock
```go
func HasOptimisticLock(table string) bool
func (db *DB) HasOptimisticLock(table string) bool
```
Check if table has optimistic lock configured.

### Soft Delete

#### EnableSoftDelete
```go
func EnableSoftDelete()
func (db *DB) EnableSoftDelete() *DB
```
Enable soft delete feature.

**Example:**
```go
eorm.EnableSoftDelete()
```

#### ConfigSoftDelete
```go
func ConfigSoftDelete(table string, field ...string)
func (db *DB) ConfigSoftDelete(table string, field ...string) *DB
```
Configure soft delete field for table.

**Example:**
```go
// Use default field deleted_at
eorm.ConfigSoftDelete("users")

// Use custom field
eorm.ConfigSoftDelete("users", "is_deleted")
```

#### ConfigSoftDeleteWithType
```go
func ConfigSoftDeleteWithType(table, field string, deleteType SoftDeleteType)
func (db *DB) ConfigSoftDeleteWithType(table, field string, deleteType SoftDeleteType) *DB
```
Configure specified type of soft delete for table.

**Example:**
```go
// Timestamp type soft delete
eorm.ConfigSoftDeleteWithType("users", "deleted_at", eorm.SoftDeleteTimestamp)

// Boolean type soft delete
eorm.ConfigSoftDeleteWithType("products", "is_deleted", eorm.SoftDeleteBool)
```

#### RemoveSoftDelete
```go
func RemoveSoftDelete(table string)
func (db *DB) RemoveSoftDelete(table string) *DB
```
Remove table soft delete configuration.

#### HasSoftDelete
```go
func HasSoftDelete(table string) bool
func (db *DB) HasSoftDelete(table string) bool
```
Check if table has soft delete configured.

---

## SQL Templates

eorm provides powerful SQL template functionality, allowing you to manage SQL statements in a configuration file, supporting dynamic parameters, condition building, and multi-database execution.

### Configuration File Structure

SQL templates use JSON format configuration files. Here is a complete configuration file format template:

#### Complete JSON Format Template

```json
{
  "version": "1.0",
  "description": "Service SQL configuration file description",
  "namespace": "service_name",
  "sqls": [
    {
      "name": "sqlName",
      "description": "SQL statement description",
      "sql": "SELECT * FROM table WHERE condition = :param",
      "type": "select",
      "order": "created_at DESC",
      "inparam": [
        {
          "name": "paramName",
          "type": "string",
          "desc": "Parameter description",
          "sql": " AND column = :paramName"
        }
      ]
    }
  ]
}
```

#### Field Description

**Root level fields:**
- `version` (string, required): Configuration file version number
- `description` (string, optional): Configuration file description
- `namespace` (string, optional): Namespace, used to avoid SQL name conflicts
- `sqls` (array, required): SQL statement configuration array

**SQL configuration fields:**
- `name` (string, required): SQL statement unique identifier
- `description` (string, optional): SQL statement description
- `sql` (string, required): SQL statement template
- `type` (string, optional): SQL type (`select`, `insert`, `update`, `delete`)
- `order` (string, optional): Default sort condition
- `inparam` (array, optional): Input parameter definition (for dynamic SQL)

**Input parameter fields (inparam):**
- `name` (string, required): Parameter name
- `type` (string, required): Parameter type
- `desc` (string, optional): Parameter description
- `sql` (string, required): SQL fragment to append when parameter exists

#### Actual Configuration Example

```json
{
  "version": "1.0",
  "description": "User service SQL configuration",
  "namespace": "user_service",
  "sqls": [
    {
      "name": "findById",
      "description": "Find user by ID",
      "sql": "SELECT * FROM users WHERE id = :id",
      "type": "select"
    },
    {
      "name": "findUsers",
      "description": "Dynamic query user list",
      "sql": "SELECT * FROM users WHERE 1=1",
      "type": "select",
      "order": "created_at DESC",
      "inparam": [
        {
          "name": "status",
          "type": "int",
          "desc": "User status",
          "sql": " AND status = :status"
        },
        {
          "name": "name",
          "type": "string",
          "desc": "Username fuzzy query",
          "sql": " AND name LIKE CONCAT('%', :name, '%')"
        }
      ]
    }
  ]
}
```

### Parameter Type Support

eorm SQL templates support multiple parameter passing methods, providing flexible usage experience:

#### Supported Parameter Types

| Parameter Type | Use Case | SQL Placeholder | Example |
|---------|---------|-----------|------|
| `map[string]interface{}` | Named parameters | `:name` | `map[string]interface{}{"id": 123}` |
| `[]interface{}` | Multiple positional parameters | `?` | `[]interface{}{123, "John"}` |
| **Single simple type** | Single positional parameter | `?` | `123`, `"John"`, `true` |
| **Variadic parameters** | Multiple positional parameters | `?` | `SqlTemplate(name, 123, "John", true)` |

#### Single Simple Type Support

Supports directly passing simple type parameters without wrapping in map or slice:

- `string` - String
- `int`, `int8`, `int16`, `int32`, `int64` - Integer types
- `uint`, `uint8`, `uint16`, `uint32`, `uint64` - Unsigned integers
- `float32`, `float64` - Floating point numbers
- `bool` - Boolean value

#### Variadic Parameter Support

Supports Go-style variadic parameters (`...interface{}`), providing the most natural parameter passing method:

```go
// Variadic parameter method - most intuitive and concise
records, err := eorm.SqlTemplate("findByIdAndStatus", 123, 1).Query()
records, err := eorm.SqlTemplate("updateUser", "John", "john@example.com", 25, 123).Exec()
records, err := eorm.SqlTemplate("findByAgeRange", 18, 65, 1).Query()
```

#### Parameter Matching Rules

| SQL Placeholder | Parameter Type | Result |
|-----------|---------|------|
| `:name` | `map[string]interface{}` | Named parameter |
| `?` | `[]interface{}` | Positional parameter |
| `?` | Single simple type | Single positional parameter |
| `?` | Variadic parameters | Multiple positional parameters |

---

## Utility Functions

### FromRecord
```go
func FromRecord(record *Record) *Query
```
Create query from Record.

### FromJson
```go
func FromJson(jsonStr string) *Record
```
Create Record from JSON.

### FromMap
```go
func FromMap(m map[string]interface{}) *Record
```
Create Record from map.

**Example:**
```go
// Commonly used for data after JSON parsing
jsonMap := map[string]interface{}{
    "name": "Zhang San",
    "age": 25,
    "email": "zhangsan@example.com",
}

record := eorm.FromMap(jsonMap)

// Directly use the created record
id, err := eorm.InsertRecord("users", record)
```

### FromRecord
```go
func FromRecord(src *Record) *Record
```
Create a new Record from another Record (deep copy). Uses deep copy to ensure nested objects are completely copied.

**Example:**
```go
sourceRecord := eorm.NewRecord().
    FromJson(`{"id": 1, "name": "Zhang San", "age": 25}`)

// Create new Record from sourceRecord
record := eorm.FromRecord(sourceRecord)

// Modifying the new Record won't affect the original Record
record.Set("email", "zhangsan@example.com")

fmt.Println("Original:", sourceRecord.ToJson())
fmt.Println("New record:", record.ToJson())
// Output:
// Original: {"id":1,"name":"Zhang San","age":25}
// New record: {"id":1,"name":"Zhang San","age":25,"email":"zhangsan@example.com"}
```

### FromStruct
```go
func FromStruct(src interface{}) *Record
```
Create Record from struct.

### ToRecord
```go
func ToRecord(src interface{}) *Record
```
Convert any type to Record.

### ToStruct
```go
func ToStruct(r *Record, dest interface{}) error
func (r *Record) ToStruct(dest interface{}) error
```
Convert Record to struct.

### ToStructs
```go
func ToStructs(records []*Record, dest interface{}) error
```
Convert Record slice to struct slice.
