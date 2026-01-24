# eorm Quick Reference

## Database Connection

```go
// Open connection
eorm.OpenDatabase(eorm.MySQL, "user:pass@tcp(localhost:3306)/db", 10)

// Use specified name
eorm.OpenDatabaseWithDBName("mysql", eorm.MySQL, dsn, 10)

// Test connection
eorm.PingDB("mysql")

// Enable debug
eorm.SetDebugMode(true)

// Close connection
eorm.Close()
```

## Record Operations

```go
// Create record
record := eorm.NewRecord()
record.Set("name", "John")
record.Set("age", 30)

// Insert
id, err := eorm.Insert("users", record)

// Query multiple
records, err := eorm.Query("SELECT * FROM users WHERE age > ?", 25)

// Query single
record, err := eorm.QueryFirst("SELECT * FROM users WHERE id = ?", 1)

// Update
affected, err := eorm.Update("users", record, "id = ?", 1)

// Save (insert or update)
affected, err := eorm.Save("users", record)

// Delete
affected, err := eorm.Delete("users", "id = ?", 1)

// Count
count, err := eorm.Count("users", "age > ?", 25)

// Check exists
exists, err := eorm.Exists("users", "id = ?", 1)
```

## Chain Queries (QueryBuilder)

```go
// Basic query
records, err := eorm.Table("users").
    Where("age > ?", 25).
    OrderBy("age DESC").
    Limit(10).
    Find()

// Query single
record, err := eorm.Table("users").
    Where("id = ?", 1).
    FindFirst()

// Pagination
page, err := eorm.Table("users").
    Where("age > ?", 25).
    Paginate(1, 10)

// Count
count, err := eorm.Table("users").
    Where("age > ?", 25).
    Count()

// Delete
affected, err := eorm.Table("users").
    Where("age < ?", 18).
    Delete()

// Update
affected, err := eorm.Table("users").
    Where("id = ?", 1).
    Update(eorm.NewRecord().Set("age", 35))
```

## Timestamp Functionality

```go
// Enable
eorm.EnableTimestamps()

// Configure table
eorm.ConfigTimestamps("users")

// Custom field names
eorm.ConfigTimestampsWithFields("orders", "create_time", "update_time")

// Only created_at
eorm.ConfigCreatedAt("logs", "log_time")

// Disable timestamp update
eorm.Table("users").Where("id = ?", 1).WithoutTimestamps().Update(record)
```

## Soft Delete Functionality

```go
// Enable
eorm.EnableSoftDelete()

// Configure table
eorm.ConfigSoftDelete("users", "deleted_at")

// Soft delete
eorm.Delete("users", "id = ?", 1)

// Restore
eorm.Restore("users", "id = ?", 1)

// Force delete
eorm.ForceDelete("users", "id = ?", 1)

// Query including deleted
records, err := eorm.Table("users").WithTrashed().Find()

// Query only deleted
records, err := eorm.Table("users").OnlyTrashed().Find()
```

## Optimistic Lock Functionality

```go
// Enable
eorm.EnableOptimisticLock()

// Configure table
eorm.ConfigOptimisticLock("products")

// Custom version field
eorm.ConfigOptimisticLockWithField("orders", "revision")

// Update with version
record := eorm.NewRecord()
record.Set("version", 1)
record.Set("price", 99.99)
affected, err := eorm.Update("products", record, "id = ?", 1)

// Check version conflict
if errors.Is(err, eorm.ErrVersionMismatch) {
    // Handle conflict
}
```

## Transaction Handling

```go
// Basic transaction
err := eorm.Transaction(func(tx *eorm.Tx) error {
    _, err := tx.Insert("users", record)
    if err != nil {
        return err  // Auto-rollback
    }
    return nil  // Auto-commit
})

// Query in transaction
err := eorm.Transaction(func(tx *eorm.Tx) error {
    records, err := tx.Query("SELECT * FROM users WHERE age > ?", 25)
    if err != nil {
        return err
    }
    return nil
})
```

## Cache Operations

```go
// Query and cache
records, err := eorm.Cache("user_cache").Query("SELECT * FROM users")

// Pagination and cache
page, err := eorm.Cache("user_page").Paginate(1, 10, "SELECT * FROM users", "users", "", "")

// Count and cache
count, err := eorm.Cache("user_count").Count("users", "age > ?", 25)

// Manual cache operations
eorm.CacheSet("store", "key", "value")
val, ok := eorm.CacheGet("store", "key")
eorm.CacheDelete("store", "key")
eorm.CacheClearRepository("store")

// Cache status
status := eorm.CacheStatus()
```

## Batch Operations

```go
// Batch insert
records := make([]*eorm.Record, 0, 100)
for i := 1; i <= 100; i++ {
    record := eorm.NewRecord().Set("name", fmt.Sprintf("User_%d", i))
    records = append(records, record)
}
affected, err := eorm.BatchInsert("users", records, 50)
```

## Database Selection

```go
// Use specified database
eorm.Use("mysql").Query("SELECT * FROM users")

// Use default database
eorm.Query("SELECT * FROM users")

// Chain call
eorm.Use("mysql").Table("users").Where("age > ?", 25).Find()
```

## Get Field Values

```go
// Get from Record
record.GetString("name")      // String
record.GetInt("age")          // Integer
record.GetInt64("id")         // 64-bit integer
record.GetFloat("salary")     // Float
record.GetBool("is_active")   // Boolean
record.Get("created_at")      // Raw value

// Set value
record.Set("name", "John")
record.Set("age", 30)
```

## Common WHERE Conditions

```go
// Basic condition
.Where("age > ?", 25)
.Where("name = ?", "John")

// Multiple conditions (AND)
.Where("age > ?", 25).Where("status = ?", "active")

// OR condition
.OrWhere("status = ?", "inactive")

// IN condition
.WhereInValues("status", []interface{}{"active", "pending"})

// NOT IN condition
.WhereNotInValues("status", []interface{}{"deleted", "banned"})

// BETWEEN condition
.WhereBetween("age", 20, 30)

// NOT BETWEEN condition
.WhereNotBetween("age", 20, 30)

// NULL condition
.WhereNull("deleted_at")
.WhereNotNull("email")
```

## Sorting and Pagination

```go
// Sort
.OrderBy("age DESC")
.OrderBy("age ASC")

// Limit
.Limit(10)

// Offset
.Offset(20)

// Pagination
.Paginate(pageNum, pageSize)
```

## JOIN Queries

```go
// LEFT JOIN
.LeftJoin("orders", "users.id", "orders.user_id")

// INNER JOIN
.InnerJoin("orders", "users.id", "orders.user_id")

// RIGHT JOIN
.RightJoin("orders", "users.id", "orders.user_id")

// Custom JOIN
.Join("orders", "users.id = orders.user_id")
```

## Subqueries

```go
// WHERE IN subquery
.WhereIn("id", eorm.Table("orders").
    Where("status = ?", "completed").
    Select("user_id"))

// WHERE NOT IN subquery
.WhereNotIn("id", eorm.Table("orders").
    Where("status = ?", "cancelled").
    Select("user_id"))
```

## Execute Raw SQL

```go
// Query
records, err := eorm.Query("SELECT * FROM users WHERE age > ?", 25)

// Query single
record, err := eorm.QueryFirst("SELECT * FROM users WHERE id = ?", 1)

// Execute
result, err := eorm.Exec("UPDATE users SET age = ? WHERE id = ?", 30, 1)

// Get affected rows
affected, err := result.RowsAffected()
```

## Error Handling

```go
// Check error
if err != nil {
    log.Printf("Error: %v", err)
}

// Check version conflict
if errors.Is(err, eorm.ErrVersionMismatch) {
    // Handle version conflict
}

// Check record not found
if errors.Is(err, eorm.ErrNoRecord) {
    // Handle record not found
}
```

## Database Types

```go
eorm.MySQL       // MySQL
eorm.PostgreSQL  // PostgreSQL
eorm.SQLite3     // SQLite
eorm.Oracle      // Oracle
eorm.SQLServer   // SQL Server
```

## Common Configuration

```go
// Set debug mode
eorm.SetDebugMode(true)

// Set default cache TTL
eorm.SetDefaultTtl(5 * time.Second)

// Create cache store
eorm.CreateCacheRepository("store_name", 10*time.Second)

// Initialize logger
eorm.InitLoggerWithFile("debug", "log.log")
```

## SQL Templates

```go
// Load configuration files
err := eorm.LoadSqlConfig("config/user_service.json")
err := eorm.LoadSqlConfigs([]string{"config/user1.json", "config/user2.json"})
err := eorm.LoadSqlConfigDir("config/")

// Single simple parameter
record, err := eorm.SqlTemplate("user_service.findById", 123).QueryFirst()

// Variadic parameters (recommended)
records, err := eorm.SqlTemplate("user_service.findByIdAndStatus", 123, 1).Query()

// Named parameters (Map)
params := map[string]interface{}{
    "name": "John",
    "status": 1,
}
records, err := eorm.SqlTemplate("user_service.findByParams", params).Query()

// Array parameters
records, err := eorm.SqlTemplate("user_service.insertUser", 
    []interface{}{"John", "john@example.com", 30}).Exec()

// Execute update
result, err := eorm.SqlTemplate("user_service.updateUser", 
    "Jane", "jane@example.com", 25, 123).Exec()

// Use in transaction
err := eorm.Transaction(func(tx *eorm.Tx) error {
    result, err := tx.SqlTemplate("user_service.insertUser", 
        "Bob", "bob@example.com", 28).Exec()
    return err
})

// Specify database
records, err := eorm.Use("mysql").SqlTemplate("findUsers", 123).Query()

// Set timeout
records, err := eorm.SqlTemplate("user_service.complexQuery", params).
    Timeout(30 * time.Second).Query()
```

### SQL Template Configuration Format

```json
{
  "version": "1.0",
  "description": "User service SQL configuration",
  "namespace": "user_service",
  "sqls": [
    {
      "name": "findById",
      "description": "Find user by ID",
      "sql": "SELECT * FROM users WHERE id = ?",
      "type": "select"
    },
    {
      "name": "updateUser",
      "description": "Update user information",
      "sql": "UPDATE users SET name = :name, email = :email WHERE id = :id",
      "type": "update"
    },
    {
      "name": "searchUsers",
      "description": "Dynamic user query",
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
          "desc": "Name fuzzy search",
          "sql": " AND name LIKE CONCAT('%', :name, '%')"
        }
      ]
    }
  ]
}
```

### SQL Template Parameter Types

| Parameter Type | Use Case | SQL Placeholder | Example |
|---------------|----------|-----------------|---------|
| Single simple type | Single positional parameter | `?` | `123`, `"John"`, `true` |
| Variadic parameters | Multiple positional parameters | `?` | `SqlTemplate(name, 123, "John")` |
| Map parameters | Named parameters | `:name` | `map[string]interface{}{"id": 123}` |
| Array parameters | Multiple positional parameters | `?` | `[]interface{}{123, "John"}` |

### SQL Template Error Handling

```go
result, err := eorm.SqlTemplate("user_service.findById", 123).QueryFirst()
if err != nil {
    if sqlErr, ok := err.(*eorm.SqlConfigError); ok {
        switch sqlErr.Type {
        case "NotFoundError":
            fmt.Printf("SQL template not found: %v\n", sqlErr.Message)
        case "ParameterError":
            fmt.Printf("Parameter error: %v\n", sqlErr.Message)
        case "ParameterTypeMismatch":
            fmt.Printf("Parameter type mismatch: %v\n", sqlErr.Message)
        }
    }
}
```
