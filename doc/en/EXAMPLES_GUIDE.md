# eorm Examples Usage Guide

This document provides detailed explanations and usage methods for all eorm examples.

## Table of Contents

1. [Timestamp Example](#timestamp-example)
2. [Soft Delete Example](#soft-delete-example)
3. [Optimistic Lock Example](#optimistic-lock-example)
4. [MySQL Example](#mysql-example)
5. [PostgreSQL Example](#postgresql-example)

---

## Timestamp Example

**Location**: `examples/timestamp/main.go`

**Functionality**: Demonstrates auto timestamp functionality

### Core Concepts

- **created_at**: Record creation time, auto-filled on insert, never modified
- **updated_at**: Record last modification time, auto-filled on every update
- **Custom field names**: Support using different field names

### Main APIs

```go
// Enable timestamp functionality (global)
eorm.EnableTimestamps()

// Configure timestamps for a table
eorm.ConfigTimestamps("users")

// Use custom field names
eorm.ConfigTimestampsWithFields("orders", "create_time", "update_time")

// Configure only created_at
eorm.ConfigCreatedAt("logs", "log_time")

// Disable timestamp update
eorm.Table("users").Where("id = ?", id).WithoutTimestamps().Update(record)
```

### Use Cases

- Record data creation and modification time
- Audit logs
- Data version control
- Data recovery

### Example Code

```go
// 1. Auto-fill created_at on insert
record := eorm.NewRecord()
record.Set("name", "John Doe")
record.Set("email", "john@example.com")
id, _ := eorm.Insert("users", record)
// created_at will be automatically set to current time

// 2. Auto-fill updated_at on update
updateRecord := eorm.NewRecord()
updateRecord.Set("name", "John Updated")
eorm.Update("users", updateRecord, "id = ?", id)
// updated_at will be automatically set to current time

// 3. Disable timestamp update
eorm.Table("users").Where("id = ?", id).WithoutTimestamps().Update(record)
// updated_at will not be modified
```

---

## Soft Delete Example

**Location**: `examples/soft_delete/main.go`

**Functionality**: Demonstrates soft delete functionality

### Core Concepts

- **Soft delete**: Mark record as deleted but don't remove from database
- **deleted_at**: Delete marker field, NULL means active, has value means deleted
- **Restore**: Restore soft-deleted record to active state
- **Force delete**: Permanently delete record

### Main APIs

```go
// Enable soft delete functionality (global)
eorm.EnableSoftDelete()

// Configure soft delete for a table
eorm.ConfigSoftDelete("users", "deleted_at")

// Soft delete record
eorm.Delete("users", "id = ?", id)

// Restore soft-deleted record
eorm.Restore("users", "id = ?", id)

// Permanently delete record
eorm.ForceDelete("users", "id = ?", id)

// Query including deleted records
eorm.Table("users").WithTrashed().Find()

// Query only deleted records
eorm.Table("users").OnlyTrashed().Find()

// Query active records (default behavior)
eorm.Table("users").Find()
```

### Use Cases

- User deactivation but retain historical data
- Order cancellation but retain audit logs
- Data recovery and undo operations
- Compliance requirements (retain deleted records)

### Example Code

```go
// 1. Soft delete record
eorm.Delete("users", "id = ?", 2)
// Record's deleted_at is set to current time

// 2. Query active users (exclude deleted)
records, _ := eorm.Table("users").Find()
// Only returns records with deleted_at = NULL

// 3. Query all users (include deleted)
records, _ := eorm.Table("users").WithTrashed().Find()
// Returns all records

// 4. Restore deleted record
eorm.Restore("users", "id = ?", 2)
// Set deleted_at to NULL

// 5. Permanently delete record
eorm.ForceDelete("users", "id = ?", 3)
// Completely remove from database
```

---

## Optimistic Lock Example

**Location**: `examples/optimistic_lock/main.go`

**Functionality**: Demonstrates optimistic lock functionality

### Core Concepts

- **Optimistic lock**: Detect concurrent modifications via version number
- **Version number**: Each record has a version number, auto-incremented on update
- **Conflict detection**: Update fails when version number doesn't match
- **ErrVersionMismatch**: Version conflict error

### Main APIs

```go
// Enable optimistic lock functionality (global)
eorm.EnableOptimisticLock()

// Configure optimistic lock for a table
eorm.ConfigOptimisticLock("products")

// Use custom version field name
eorm.ConfigOptimisticLockWithField("orders", "revision")

// Check version conflict error
if errors.Is(err, eorm.ErrVersionMismatch) {
    // Handle version conflict
}
```

### Use Cases

- Prevent concurrent update conflicts
- Inventory management (prevent overselling)
- Order status updates
- Data consistency guarantee

### Example Code

```go
// 1. Auto-initialize version to 1 on insert
record := eorm.NewRecord()
record.Set("name", "Laptop")
record.Set("price", 999.99)
id, _ := eorm.Insert("products", record)
// version automatically set to 1

// 2. Update with correct version number
updateRecord := eorm.NewRecord()
updateRecord.Set("version", int64(1))  // Current version
updateRecord.Set("price", 899.99)
rows, err := eorm.Update("products", updateRecord, "id = ?", id)
// Update succeeds, version auto-incremented to 2

// 3. Update with stale version number (will fail)
staleRecord := eorm.NewRecord()
staleRecord.Set("version", int64(1))  // Stale version
staleRecord.Set("price", 799.99)
rows, err := eorm.Update("products", staleRecord, "id = ?", id)
// Returns ErrVersionMismatch error

// 4. Correct way to handle concurrent update
latestRecord, _ := eorm.Table("products").Where("id = ?", id).FindFirst()
currentVersion := latestRecord.GetInt("version")
updateRecord := eorm.NewRecord()
updateRecord.Set("version", currentVersion)
updateRecord.Set("price", 799.99)
rows, err := eorm.Update("products", updateRecord, "id = ?", id)
// Use latest version number, update succeeds
```

---

## MySQL Example

**Location**: `examples/mysql/main.go`

**Functionality**: Demonstrates various MySQL database operations

### Prerequisites

- MySQL database is running
- Database connection information is correct
- Have appropriate database and table permissions

### Connection String Format

```
user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local
```

Parameter explanation:
- `user`: Database username
- `password`: Database password
- `host`: Database host address
- `port`: Database port (default 3306)
- `dbname`: Database name
- `charset`: Character set (recommend utf8mb4)
- `parseTime`: Whether to parse time fields
- `loc`: Timezone setting

### Main APIs

#### Connection Management

```go
// Open database connection
eorm.OpenDatabase(eorm.MySQL, dsn, 25)

// Open connection with specified database name
eorm.OpenDatabaseWithDBName("mysql", eorm.MySQL, dsn, 25)

// Test connection
eorm.PingDB("mysql")

// Enable debug mode
eorm.SetDebugMode(true)
```

#### Record CRUD Operations

```go
// Insert record
record := eorm.NewRecord().Set("name", "John").Set("age", 30)
id, err := eorm.Use("mysql").Insert("users", record)

// Query multiple records
records, err := eorm.Use("mysql").Query("SELECT * FROM users WHERE age > ?", 25)

// Query single record
record, err := eorm.Use("mysql").QueryFirst("SELECT * FROM users WHERE id = ?", 1)

// Update record
updateRec := eorm.NewRecord().Set("age", 35)
affected, err := eorm.Use("mysql").Update("users", updateRec, "id = ?", 1)

// Save record (insert or update)
affected, err := eorm.Use("mysql").Save("users", record)

// Count records
count, err := eorm.Use("mysql").Count("users", "age > ?", 25)

// Check if record exists
exists, err := eorm.Use("mysql").Exists("users", "id = ?", 1)

// Delete record
affected, err := eorm.Use("mysql").Delete("users", "id = ?", 1)
```

#### Chain Queries

```go
// Basic chain query
records, err := eorm.Table("users").
    Where("age > ?", 25).
    OrderBy("age DESC").
    Limit(10).
    Find()

// Pagination query
page, err := eorm.Table("users").
    Where("age > ?", 25).
    Paginate(1, 10)

// Query single record
record, err := eorm.Table("users").
    Where("id = ?", 1).
    FindFirst()

// Count records
count, err := eorm.Table("users").
    Where("age > ?", 25).
    Count()
```

#### Batch Operations

```go
// Batch insert
records := make([]*eorm.Record, 0, 100)
for i := 1; i <= 100; i++ {
    record := eorm.NewRecord().Set("name", fmt.Sprintf("User_%d", i))
    records = append(records, record)
}
affected, err := eorm.Use("mysql").BatchInsert("users", records, 50)
```

#### Transaction Handling

```go
// Transaction operation
err := eorm.Transaction(func(tx *eorm.Tx) error {
    // Execute operations in transaction
    _, err := tx.Insert("users", record)
    if err != nil {
        return err
    }
    
    _, err = tx.Update("users", updateRec, "id = ?", 1)
    return err
})
```

#### Cache Operations

```go
// Query and cache result
records, err := eorm.Cache("user_cache").Query("SELECT * FROM users WHERE age > ?", 25)

// Pagination query and cache
page, err := eorm.Cache("user_page_cache").Paginate(1, 10, "SELECT * FROM users", "users", "", "")

// Count and cache
count, err := eorm.Cache("user_count_cache").Count("users", "age > ?", 25)
```

### Use Cases

- Basic CRUD operations
- Complex queries and filtering
- Data pagination
- Transaction handling
- Performance optimization (caching)

---

## PostgreSQL Example

**Location**: `examples/postgresql/main.go`

**Functionality**: Demonstrates various PostgreSQL database operations

### Prerequisites

- PostgreSQL database is running
- Database connection information is correct
- Have appropriate database and table permissions

### Connection String Format

```
user=postgres password=123456 host=127.0.0.1 port=5432 dbname=postgres sslmode=disable
```

Parameter explanation:
- `user`: Database username
- `password`: Database password
- `host`: Database host address
- `port`: Database port (default 5432)
- `dbname`: Database name
- `sslmode`: SSL mode (disable/require/prefer)

### Main APIs

APIs are basically the same as MySQL, main differences are:

1. **Connection string format is different**
2. **Parameter placeholders**: PostgreSQL uses `$1, $2` instead of `?`
3. **JSONB support**: PostgreSQL natively supports JSONB type
4. **Array types**: PostgreSQL supports array types

### Example Code

```go
// Connect to PostgreSQL
dsn := "user=postgres password=123456 host=127.0.0.1 port=5432 dbname=postgres sslmode=disable"
eorm.OpenDatabaseWithDBName("postgresql", eorm.PostgreSQL, dsn, 25)

// Other operations are the same as MySQL
records, err := eorm.Table("users").Where("age > ?", 25).Find()
```

---

## Quick Start

### 1. Timestamp Functionality

```bash
cd examples/timestamp
go run main.go
```

### 2. Soft Delete Functionality

```bash
cd examples/soft_delete
go run main.go
```

### 3. Optimistic Lock Functionality

```bash
cd examples/optimistic_lock
go run main.go
```

### 4. MySQL Comprehensive Test

```bash
cd examples/mysql
go run main.go
```

### 5. PostgreSQL Comprehensive Test

```bash
cd examples/postgresql
go run main.go
```

---

## FAQ

### Q: How do I choose between Record and DbModel?

**A**: 
- **Record**: Flexible, lightweight, suitable for dynamic data and rapid development
- **DbModel**: Type-safe, structured, suitable for large projects and complex business logic

### Q: How do I handle concurrent update conflicts?

**A**: Use optimistic lock:
1. Read the latest version number
2. Specify version number when updating
3. If version number doesn't match, retry operation

### Q: How do I improve query performance?

**A**: 
1. Use cache mechanism
2. Use pagination queries
3. Add appropriate database indexes
4. Use connection pool

### Q: How do I handle errors in transactions?

**A**: Return error in transaction callback function, eorm will auto-rollback:

```go
err := eorm.Transaction(func(tx *eorm.Tx) error {
    if err := doSomething(tx); err != nil {
        return err  // Auto-rollback
    }
    return nil  // Auto-commit
})
```

---

## More Resources

- [eorm Official Documentation](https://github.com/zzguang83325/eorm)
- [API Reference](../../api.md)
- [Best Practices](./BEST_PRACTICES.md)
