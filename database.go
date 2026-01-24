package eorm

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	// 推荐的数据库驱动（用户可根据需要导入）
	// _ "github.com/go-sql-driver/mysql"           // MySQL驱动
	// _ "github.com/jackc/pgx/v5/stdlib"          // PostgreSQL驱动（推荐，性能更好）
	// _ "github.com/lib/pq"                       // PostgreSQL驱动（传统，兼容性好）
	// _ "github.com/mattn/go-sqlite3"             // SQLite3驱动
	// _ "github.com/denisenkom/go-mssqldb"        // SQL Server驱动
	// _ "github.com/godror/godror"                // Oracle驱动（推荐）
	// _ "github.com/sijms/go-ora/v2"              // Oracle驱动（纯Go实现）
)

// DriverType represents the database driver type
type DriverType string

const (
	// MySQL database driver
	MySQL DriverType = "mysql"
	// PostgreSQL database driver
	PostgreSQL DriverType = "postgres"
	// SQLite3 database driver
	SQLite3 DriverType = "sqlite3"
	// Oracle database driver
	Oracle DriverType = "oracle"
	// SQL Server database driver
	SQLServer DriverType = "sqlserver"
)

// 预编译的正则表达式，用于 sanitizeArgs 函数
// 避免每次调用都重新编译，提升性能
var (
	postgresPlaceholderRe  = regexp.MustCompile(`\$(\d+)`)
	sqlserverPlaceholderRe = regexp.MustCompile(`@p(\d+)`)
	oraclePlaceholderRe    = regexp.MustCompile(`:(\d+)`)
)

// 预编译语句缓存相关常量已移至 constants.go
// 为了向后兼容，保留这些常量的别名
const (
	stmtCacheRepository = StmtCacheRepository // 内部使用的缓存名称
)

// Config holds the database configuration
type Config struct {
	Driver          DriverType    // Database driver type (mysql, postgres, sqlite3)
	DSN             string        // Data source name (connection string)
	MaxOpen         int           // Maximum number of open connections
	MaxIdle         int           // Maximum number of idle connections
	ConnMaxLifetime time.Duration // Maximum connection lifetime
	QueryTimeout    time.Duration // Default query timeout (0 means no timeout)

	// 连接监控配置（新增）
	MonitorNormalInterval time.Duration // 正常检查间隔（默认60秒，0表示禁用监控）
	MonitorErrorInterval  time.Duration // 故障检查间隔（默认10秒）
}

// SupportedDrivers returns a list of all supported database drivers
func SupportedDrivers() []DriverType {
	return []DriverType{MySQL, PostgreSQL, SQLite3, Oracle, SQLServer}
}

// IsValidDriver checks if the given driver is supported
func IsValidDriver(driver DriverType) bool {
	for _, d := range SupportedDrivers() {
		if d == driver {
			return true
		}
	}
	return false
}

// DB represents a database connection with chainable methods
type DB struct {
	dbMgr               *dbManager
	lastErr             error
	cacheRepositoryName string
	cacheTTL            time.Duration
	timeout             time.Duration // Query timeout for this instance
	cacheProvider       CacheProvider // 指定的缓存提供者（nil 表示使用默认缓存）
	countCacheTTL       time.Duration // 分页计数缓存时间（-1 表示不使用，0 表示不缓存，>0 表示使用指定时间）
}

// GetConfig returns the database configuration
func (db *DB) GetConfig() (*Config, error) {
	if db == nil || db.dbMgr == nil {
		return nil, fmt.Errorf("database or database manager is nil")
	}
	return db.dbMgr.GetConfig()
}

// getTimeout returns the effective timeout for this DB instance
func (db *DB) getTimeout() time.Duration {
	if db.timeout > 0 {
		return db.timeout
	}
	if db.dbMgr != nil && db.dbMgr.config != nil && db.dbMgr.config.QueryTimeout > 0 {
		return db.dbMgr.config.QueryTimeout
	}
	return 0
}

// getContext returns a context with timeout if configured
func (db *DB) getContext() (context.Context, context.CancelFunc) {
	timeout := db.getTimeout()
	if timeout > 0 {
		return context.WithTimeout(context.Background(), timeout)
	}
	return context.Background(), func() {}
}

// getEffectiveCache 获取当前有效的缓存提供者
// 如果 DB 实例指定了缓存提供者，则使用指定的；否则使用全局默认缓存
func (db *DB) getEffectiveCache() CacheProvider {
	if db.cacheProvider != nil {
		return db.cacheProvider
	}
	return GetCache()
}

// ClearCache clears the specified cache repository associated with this database
func (db *DB) ClearCache(repoName string) *DB {
	if db.dbMgr != nil {
		db.dbMgr.clearCache(repoName)
	}
	return db
}

// GetStmtCacheStats 获取预编译语句缓存的统计信息
// 返回包含命中率、大小、淘汰次数等指标的 map
func (db *DB) GetStmtCacheStats() map[string]interface{} {
	if db.dbMgr == nil || db.dbMgr.stmtCache == nil {
		return map[string]interface{}{
			"enabled": false,
			"error":   "cache not initialized",
		}
	}
	return db.dbMgr.stmtCache.Stats()
}

// Tx represents a database transaction with chainable methods
type Tx struct {
	tx                  *sql.Tx
	dbMgr               *dbManager
	cacheRepositoryName string
	cacheTTL            time.Duration
	timeout             time.Duration // Query timeout for this transaction
	cacheProvider       CacheProvider // 指定的缓存提供者（nil 表示使用默认缓存）
	countCacheTTL       time.Duration // 分页计数缓存时间（-1 表示不使用，0 表示不缓存，>0 表示使用指定时间）
}

// getEffectiveCache 获取当前有效的缓存提供者
// 如果 Tx 实例指定了缓存提供者，则使用指定的；否则使用全局默认缓存
func (tx *Tx) getEffectiveCache() CacheProvider {
	if tx.cacheProvider != nil {
		return tx.cacheProvider
	}
	return GetCache()
}

// ClearCache clears the specified cache repository associated with this database
func (tx *Tx) ClearCache(repoName string) *Tx {
	if tx.dbMgr != nil {
		tx.dbMgr.clearCache(repoName)
	}
	return tx
}

// sqlExecutor is an internal interface for executing SQL commands
type sqlExecutor interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
	Exec(query string, args ...interface{}) (sql.Result, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

// sqlExecutorContext is an internal interface for executing SQL commands with context
type sqlExecutorContext interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// StatementResult 单个 SQL 语句的执行结果
type StatementResult struct {
	Index  int           // 语句索引（从 0 开始）
	SQL    string        // SQL 语句
	Args   []interface{} // 参数列表
	Result sql.Result    // 执行结果（成功时有值，失败时为 nil）
	Error  error         // 错误信息（失败时有值，成功时为 nil）
}

// IsSuccess 判断该语句是否执行成功
func (r *StatementResult) IsSuccess() bool {
	return r.Error == nil
}

// RowsAffected 获取受影响的行数（成功时）
func (r *StatementResult) RowsAffected() (int64, error) {
	if r.Result == nil {
		return 0, fmt.Errorf("statement failed: %v", r.Error)
	}
	return r.Result.RowsAffected()
}

// LastInsertId 获取最后插入的 ID（成功时）
func (r *StatementResult) LastInsertId() (int64, error) {
	if r.Result == nil {
		return 0, fmt.Errorf("statement failed: %v", r.Error)
	}
	return r.Result.LastInsertId()
}

// BatchExecError 批量执行错误
type BatchExecError struct {
	FailedCount int    // 失败的语句数量
	Message     string // 错误摘要信息
}

// Error 实现 error 接口
func (e *BatchExecError) Error() string {
	return e.Message
}

// dbManager manages database connections
type dbManager struct {
	name            string
	config          *Config
	db              *sql.DB
	mu              sync.RWMutex
	initMu          sync.Mutex // 用于初始化数据库连接的独立锁
	drivers         map[string]bool
	pkCache         map[string][]string     // Table name -> PK column names
	identityCache   map[string]string       // Table name -> Identity column name
	columnCache     map[string][]ColumnInfo // Table name -> Column info list (新增：列信息缓存)
	softDeletes     *softDeleteRegistry     // Soft delete configurations
	timestamps      *timestampRegistry      // Auto timestamp configurations
	optimisticLocks *optimisticLockRegistry // Optimistic lock configurations
	stmtCacheTTL    time.Duration           // 已废弃：保留用于向后兼容
	stmtCache       *stmtCache              // 新的智能语句缓存
	// Feature flags
	enableTimestampCheck      bool // Enable auto timestamp check in Update (default: false)
	enableOptimisticLockCheck bool // Enable optimistic lock check in Update (default: false)
	enableSoftDeleteCheck     bool // Enable soft delete check in queries (default: false)

	// 连接监控相关（默认启用）
	monitor      *ConnectionMonitor // 连接监控器实例
	lastPingTime time.Time          // 最后一次 Ping 时间
	pingMu       sync.RWMutex       // Ping 操作锁
}

// clearCache clears the specified cache repository
func (mgr *dbManager) clearCache(repoName string) {
	if repoName == "" {
		return
	}
	cache := GetCache()
	if cache != nil {
		cache.CacheClearRepository(repoName)
	}
}

// MultiDBManager manages multiple database connections
type MultiDBManager struct {
	databases map[string]*dbManager
	currentDB string
	defaultDB string
	mu        sync.RWMutex
}

var (
	multiMgr *MultiDBManager
)

// init initializes the multi-database manager
func init() {
	multiMgr = &MultiDBManager{
		databases: make(map[string]*dbManager),
	}
}

// createDefaultConfig creates a Config with default settings
func createDefaultConfig(driver DriverType, dsn string, maxOpen int) *Config {
	return &Config{
		Driver:                driver,
		DSN:                   dsn,
		MaxOpen:               maxOpen,
		MaxIdle:               maxOpen / 2,
		ConnMaxLifetime:       time.Hour,
		MonitorNormalInterval: 60 * time.Second, // 默认60秒正常检查间隔
		MonitorErrorInterval:  10 * time.Second, // 默认10秒故障检查间隔
	}
}

// OpenDatabase opens a database connection with default settings
// This is equivalent to registering a database named "default"
// 使用默认配置打开数据库连接，返回DB实例用于链式调用
func OpenDatabase(driver DriverType, dsn string, maxOpen int) (*DB, error) {
	config := createDefaultConfig(driver, dsn, maxOpen)
	return OpenDatabaseWithConfig("default", config)
}

// OpenDatabaseWithDBName opens a database connection with a name (multi-database mode)
// 使用指定名称打开数据库连接，支持多数据库模式
func OpenDatabaseWithDBName(dbname string, driver DriverType, dsn string, maxOpen int) (*DB, error) {
	config := createDefaultConfig(driver, dsn, maxOpen)
	return OpenDatabaseWithConfig(dbname, config)
}

// RegisterDataBase open a database connection with a name (multi-database mode)
// 注册数据库连接，这是核心函数，其他函数都基于此实现
func OpenDatabaseWithConfig(dbname string, config *Config) (*DB, error) {
	dbMgr := &dbManager{
		name:          dbname,
		config:        config,
		pkCache:       make(map[string][]string),
		identityCache: make(map[string]string),
		columnCache:   make(map[string][]ColumnInfo), // 初始化列信息缓存
	}

	if err := dbMgr.initDB(); err != nil {
		return nil, err
	}

	multiMgr.mu.Lock()
	multiMgr.databases[dbname] = dbMgr
	// Set as default database if it's the first one
	if multiMgr.defaultDB == "" {
		multiMgr.defaultDB = dbname
		multiMgr.currentDB = dbname
	}
	multiMgr.mu.Unlock()

	// 返回新创建的DB实例
	return &DB{dbMgr: dbMgr}, nil
}

// Use switches to a different database by name and returns a DB object for chainable calls
// This is a convenience method that avoids panicking for fluent API usage.
// If the database is not found or another error occurs, the error is stored in the returned DB object
// and will be returned by subsequent operations.
func Use(dbname string) *DB {
	db, err := UseWithError(dbname)
	if err != nil {
		return &DB{lastErr: err}
	}
	return db
}

// UseWithError returns a DB object for the specified database by name
func UseWithError(dbname string) (*DB, error) {
	multiMgr.mu.RLock()
	dbMgr, exists := multiMgr.databases[dbname]
	multiMgr.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("database '%s' not found", dbname)
	}

	return &DB{dbMgr: dbMgr}, nil
}

var (
	// ErrNotInitialized is returned when an operation is performed on an uninitialized database
	ErrNotInitialized = fmt.Errorf("eorm: database not initialized. Please call eorm.OpenDatabase() before using eorm operations")
)

// defaultDB returns the default DB object (first registered database or single database mode)
func defaultDB() (*DB, error) {
	dbMgr, err := safeGetCurrentDB()
	if err != nil {
		return nil, err
	}
	return &DB{dbMgr: dbMgr}, nil
}

// --- Internal Helper Methods on dbManager to unify DB and Tx logic ---

func (mgr *dbManager) prepareQuerySQL(querySQL string, args ...interface{}) (string, []interface{}) {
	driver := mgr.config.Driver
	lowerSQL := strings.ToLower(querySQL)

	// 处理 Oracle 和 SQL Server 的 LIMIT 语法
	if driver == Oracle {
		if limitIndex := strings.LastIndex(lowerSQL, " limit "); limitIndex != -1 {
			limitStr := strings.TrimSpace(querySQL[limitIndex+6:])
			querySQL = fmt.Sprintf("SELECT * FROM (%s) WHERE ROWNUM <= %s", querySQL[:limitIndex], limitStr)
		}
	} else if driver == SQLServer {
		if limitIndex := strings.LastIndex(lowerSQL, " limit "); limitIndex != -1 {
			limitStr := strings.TrimSpace(querySQL[limitIndex+6:])
			sqlPart := querySQL[:limitIndex]
			if selectIndex := strings.Index(strings.ToLower(sqlPart), "select "); selectIndex != -1 {
				querySQL = fmt.Sprintf("SELECT TOP %s %s", limitStr, sqlPart[selectIndex+7:])
			}
		}
	}

	// Oracle: 为 time.Time 参数的占位符添加 TO_DATE 包装
	if driver == Oracle {
		querySQL = mgr.wrapOracleDatePlaceholders(querySQL, args)
	}

	querySQL = mgr.convertPlaceholder(querySQL, driver)
	args = mgr.sanitizeArgs(querySQL, args)
	return querySQL, args
}

// getOrPrepareStmt 获取或创建预编译语句（内部方法）
// 返回值：stmt, fromCache, error
func (mgr *dbManager) getOrPrepareStmt(sqlStr string) (*sql.Stmt, bool, error) {
	// 构造缓存键：数据库名称 + SQL 语句
	cacheKey := mgr.name + ":" + sqlStr

	// 1. 尝试从新的智能缓存获取
	if stmt, ok := mgr.stmtCache.Get(cacheKey); ok {
		return stmt, true, nil // 从缓存获取（自动更新访问时间和计数）
	}

	// 2. 缓存未命中，创建新的预编译语句
	stmt, err := mgr.db.Prepare(sqlStr)
	if err != nil {
		return nil, false, err
	}

	// 3. 存入智能缓存（自动处理容量限制和 LRU 淘汰）
	mgr.stmtCache.Set(cacheKey, stmt, sqlStr)

	return stmt, false, nil // 新创建的
}

// clearStmtCache 清空预编译语句缓存（内部方法，用于数据库关闭时）
func (mgr *dbManager) clearStmtCache() {
	if mgr.stmtCache != nil {
		mgr.stmtCache.Clear()
	}
}

// isStmtInvalidError 检查是否是语句失效错误
func isStmtInvalidError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "invalid connection") ||
		strings.Contains(errStr, "bad connection") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset")
}

func (mgr *dbManager) query(executor sqlExecutor, querySQL string, args ...interface{}) ([]*Record, error) {
	return mgr.queryWithContext(context.Background(), executor, querySQL, args...)
}

func (mgr *dbManager) queryWithContext(ctx context.Context, executor sqlExecutor, querySQL string, args ...interface{}) ([]*Record, error) {
	querySQL, args = mgr.prepareQuerySQL(querySQL, args...)
	start := time.Now()

	var rows *sql.Rows
	var err error

	// 只有当 executor 是 *sql.DB 时才使用预编译语句缓存
	// 事务（*sql.Tx）不使用缓存，因为事务有自己的生命周期
	if db, ok := executor.(*sql.DB); ok && db == mgr.db {
		// 使用缓存的预编译语句
		stmt, fromCache, stmtErr := mgr.getOrPrepareStmt(querySQL)
		if stmtErr != nil {
			mgr.logTrace(start, querySQL, args, stmtErr)
			return nil, stmtErr
		}

		// 执行查询（使用 context）
		rows, err = stmt.QueryContext(ctx, args...)

		// 如果执行失败且可能是语句失效，从缓存移除
		if err != nil && !fromCache {
			// 新创建的语句出错，不需要特殊处理
		} else if err != nil && isStmtInvalidError(err) {
			cacheKey := mgr.name + ":" + querySQL
			mgr.stmtCache.Delete(cacheKey) // 使用新的智能缓存删除
		}
	} else {
		// 事务或其他 executor，使用原有逻辑
		if execCtx, ok := executor.(sqlExecutorContext); ok {
			rows, err = execCtx.QueryContext(ctx, querySQL, args...)
		} else {
			rows, err = executor.Query(querySQL, args...)
		}
	}

	mgr.logTrace(start, querySQL, args, err)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results, err := scanRecords(rows, mgr.config.Driver)
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (mgr *dbManager) queryFirst(executor sqlExecutor, querySQL string, args ...interface{}) (*Record, error) {
	return mgr.queryFirstWithContext(context.Background(), executor, querySQL, args...)
}

func (mgr *dbManager) queryFirstWithContext(ctx context.Context, executor sqlExecutor, querySQL string, args ...interface{}) (*Record, error) {
	querySQL = mgr.addLimitOne(querySQL)
	return mgr.queryFirstInternalWithContext(ctx, executor, querySQL, args...)
}

func (mgr *dbManager) queryFirstInternal(executor sqlExecutor, querySQL string, args ...interface{}) (*Record, error) {
	return mgr.queryFirstInternalWithContext(context.Background(), executor, querySQL, args...)
}

func (mgr *dbManager) queryFirstInternalWithContext(ctx context.Context, executor sqlExecutor, querySQL string, args ...interface{}) (*Record, error) {
	records, err := mgr.queryWithContext(ctx, executor, querySQL, args...)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	return records[0], nil
}

func (mgr *dbManager) queryMap(executor sqlExecutor, querySQL string, args ...interface{}) ([]map[string]interface{}, error) {
	return mgr.queryMapWithContext(context.Background(), executor, querySQL, args...)
}

func (mgr *dbManager) queryMapWithContext(ctx context.Context, executor sqlExecutor, querySQL string, args ...interface{}) ([]map[string]interface{}, error) {
	querySQL, args = mgr.prepareQuerySQL(querySQL, args...)
	start := time.Now()

	var rows *sql.Rows
	var err error

	// 只有当 executor 是 *sql.DB 时才使用预编译语句缓存
	if db, ok := executor.(*sql.DB); ok && db == mgr.db {
		// 使用缓存的预编译语句
		stmt, fromCache, stmtErr := mgr.getOrPrepareStmt(querySQL)
		if stmtErr != nil {
			mgr.logTrace(start, querySQL, args, stmtErr)
			return nil, stmtErr
		}

		// 执行查询（使用 context）
		rows, err = stmt.QueryContext(ctx, args...)

		// 如果执行失败且可能是语句失效，从缓存移除
		if err != nil && !fromCache {
			// 新创建的语句出错，不需要特殊处理
		} else if err != nil && isStmtInvalidError(err) {
			cacheKey := mgr.name + ":" + querySQL
			mgr.stmtCache.Delete(cacheKey)
		}
	} else {
		// 事务或其他 executor，使用原有逻辑
		if execCtx, ok := executor.(sqlExecutorContext); ok {
			rows, err = execCtx.QueryContext(ctx, querySQL, args...)
		} else {
			rows, err = executor.Query(querySQL, args...)
		}
	}

	mgr.logTrace(start, querySQL, args, err)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results, err := scanMaps(rows, mgr.config.Driver)
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (mgr *dbManager) addLimitOne(querySQL string) string {
	driver := mgr.config.Driver
	lowerSQL := strings.ToLower(strings.TrimSpace(querySQL))

	// Check if already has limit
	if strings.Contains(lowerSQL, " limit ") ||
		strings.Contains(lowerSQL, " top ") ||
		strings.Contains(lowerSQL, " rownum ") ||
		strings.Contains(lowerSQL, " fetch first ") ||
		strings.Contains(lowerSQL, " fetch next ") {
		return querySQL
	}

	switch driver {
	case MySQL, PostgreSQL, SQLite3:
		return querySQL + " LIMIT 1"
	case Oracle:
		return fmt.Sprintf("SELECT * FROM (%s) WHERE ROWNUM <= 1", querySQL)
	case SQLServer:
		if strings.HasPrefix(lowerSQL, "select ") {
			// Basic SELECT TOP 1 implementation
			// Check for DISTINCT to avoid invalid syntax like "SELECT TOP 1 DISTINCT"
			if strings.HasPrefix(lowerSQL, "select distinct ") {
				return "SELECT DISTINCT TOP 1 " + querySQL[16:]
			}
			return "SELECT TOP 1 " + querySQL[7:]
		}
		return querySQL
	default:
		return querySQL
	}
}

func (mgr *dbManager) exec(executor sqlExecutor, querySQL string, args ...interface{}) (sql.Result, error) {
	return mgr.execWithContext(context.Background(), executor, querySQL, args...)
}

func (mgr *dbManager) execWithContext(ctx context.Context, executor sqlExecutor, querySQL string, args ...interface{}) (sql.Result, error) {
	querySQL = mgr.convertPlaceholder(querySQL, mgr.config.Driver)
	args = mgr.sanitizeArgs(querySQL, args)
	start := time.Now()

	var result sql.Result
	var err error

	// 只有当 executor 是 *sql.DB 时才使用预编译语句缓存
	if db, ok := executor.(*sql.DB); ok && db == mgr.db {
		// 使用缓存的预编译语句
		stmt, fromCache, stmtErr := mgr.getOrPrepareStmt(querySQL)
		if stmtErr != nil {
			mgr.logTrace(start, querySQL, args, stmtErr)
			return nil, stmtErr
		}

		// 执行命令（使用 context）
		result, err = stmt.ExecContext(ctx, args...)

		// 如果执行失败且可能是语句失效，从缓存移除
		if err != nil && !fromCache {
			// 新创建的语句出错，不需要特殊处理
		} else if err != nil && isStmtInvalidError(err) {
			cacheKey := mgr.name + ":" + querySQL
			mgr.stmtCache.Delete(cacheKey)
		}
	} else {
		// 事务或其他 executor，使用原有逻辑
		if execCtx, ok := executor.(sqlExecutorContext); ok {
			result, err = execCtx.ExecContext(ctx, querySQL, args...)
		} else {
			result, err = executor.Exec(querySQL, args...)
		}
	}

	mgr.logTrace(start, querySQL, args, err)

	if err != nil {
		return nil, err
	}
	return result, nil
}

// batchExecWithContext 批量执行多个 SQL 语句的核心实现（带 context）
// ctx: 上下文，用于超时控制
// executor: SQL 执行器（*sql.DB 或 *sql.Tx）
// sqls: SQL 语句列表
// args: 每个 SQL 语句对应的参数列表（可以为 nil）
// 返回: 每个语句的执行结果列表和错误
func (mgr *dbManager) batchExecWithContext(
	ctx context.Context,
	executor sqlExecutor,
	sqls []string,
	args [][]interface{},
) ([]StatementResult, error) {
	// 1. 参数验证
	if len(sqls) == 0 {
		return nil, fmt.Errorf("batch exec: SQL list is empty")
	}

	// 验证参数数量（如果提供了参数）
	if args != nil && len(args) != len(sqls) {
		return nil, fmt.Errorf("batch exec: args length (%d) does not match sqls length (%d)",
			len(args), len(sqls))
	}

	// 2. 判断是否在事务中
	isInTransaction := false
	if _, isTx := executor.(*sql.Tx); isTx {
		isInTransaction = true
	}

	// 3. 记录批量执行开始
	batchStart := time.Now()
	mgr.logTrace(batchStart, fmt.Sprintf("BatchExec: starting %d statements (transaction mode: %v)", len(sqls), isInTransaction), nil, nil)

	// 4. 初始化结果列表
	statementResults := make([]StatementResult, 0, len(sqls))
	failedCount := 0

	// 5. 循环执行 SQL 语句
	for i, sqlStr := range sqls {
		// 跳过空字符串
		if strings.TrimSpace(sqlStr) == "" {
			mgr.logTrace(time.Now(), fmt.Sprintf("BatchExec[%d]: skipping empty SQL", i), nil, nil)
			continue
		}

		// 获取参数
		var sqlArgs []interface{}
		if args != nil && i < len(args) {
			sqlArgs = args[i]
		}

		// 记录单个语句执行开始
		stmtStart := time.Now()

		// 执行 SQL
		result, err := mgr.execWithContext(ctx, executor, sqlStr, sqlArgs...)

		// 记录单个语句执行结果
		mgr.logTrace(stmtStart, fmt.Sprintf("BatchExec[%d]: %s", i, sqlStr), sqlArgs, err)

		if err != nil {
			// 记录失败结果（包含 SQL 和参数）
			statementResults = append(statementResults, StatementResult{
				Index:  i,
				SQL:    sqlStr,
				Args:   sqlArgs,
				Result: nil,
				Error:  err,
			})
			failedCount++

			// 事务模式：遇到错误立即停止
			if isInTransaction {
				mgr.logTrace(batchStart, fmt.Sprintf("BatchExec: stopped at statement %d due to error (transaction mode)", i), nil, err)
				break
			}

			// 非事务模式：继续执行后续语句
			continue
		}

		// 记录成功结果（也包含 SQL 和参数，方便查看）
		statementResults = append(statementResults, StatementResult{
			Index:  i,
			SQL:    sqlStr,
			Args:   sqlArgs,
			Result: result,
			Error:  nil,
		})
	}

	// 6. 记录批量执行完成
	mgr.logTrace(batchStart, fmt.Sprintf("BatchExec: completed %d/%d statements, %d failed", len(statementResults), len(sqls), failedCount), nil, nil)

	// 7. 检查是否有失败的语句
	if failedCount > 0 {
		return statementResults, &BatchExecError{
			FailedCount: failedCount,
			Message:     fmt.Sprintf("batch exec completed with %d error(s)", failedCount),
		}
	}

	// 全部成功
	return statementResults, nil
}

// batchExec 批量执行多个 SQL 语句（不带 context）
// executor: SQL 执行器（*sql.DB 或 *sql.Tx）
// sqls: SQL 语句列表
// args: 每个 SQL 语句对应的参数列表（可以为 nil）
// 返回: 每个语句的执行结果列表和错误
func (mgr *dbManager) batchExec(
	executor sqlExecutor,
	sqls []string,
	args [][]interface{},
) ([]StatementResult, error) {
	return mgr.batchExecWithContext(context.Background(), executor, sqls, args)
}

func (mgr *dbManager) getIdentityColumn(executor sqlExecutor, table string) string {
	// 1. 先检查旧缓存（向后兼容，快速路径）
	mgr.mu.RLock()
	if col, ok := mgr.identityCache[table]; ok {
		mgr.mu.RUnlock()
		return col
	}
	mgr.mu.RUnlock()

	// 2. 使用 getTableColumns 获取列信息（会自动缓存）
	columns, err := mgr.getTableColumns(table)
	if err != nil {
		return ""
	}

	// 3. 从列信息中提取自增列
	var identityCol string
	for _, col := range columns {
		if col.IsAutoIncr {
			identityCol = col.Name
			break // 通常只有一个自增列
		}
	}

	// 4. 同时更新旧缓存（保持兼容性）
	mgr.mu.Lock()
	if mgr.identityCache == nil {
		mgr.identityCache = make(map[string]string)
	}
	mgr.identityCache[table] = identityCol
	mgr.mu.Unlock()

	return identityCol
}

func (mgr *dbManager) getPrimaryKeys(executor sqlExecutor, table string) ([]string, error) {
	// 1. 先检查旧缓存（向后兼容，快速路径）
	mgr.mu.RLock()
	if pks, ok := mgr.pkCache[table]; ok {
		mgr.mu.RUnlock()
		return pks, nil
	}
	mgr.mu.RUnlock()

	// 2. 使用 getTableColumns 获取列信息（会自动缓存）
	columns, err := mgr.getTableColumns(table)
	if err != nil {
		return nil, err
	}

	// 3. 从列信息中提取主键列
	var pks []string
	for _, col := range columns {
		if col.IsPK {
			pks = append(pks, col.Name)
		}
	}

	// 4. 同时更新旧缓存（保持兼容性）
	mgr.mu.Lock()
	if mgr.pkCache == nil {
		mgr.pkCache = make(map[string][]string)
	}
	mgr.pkCache[table] = pks
	mgr.mu.Unlock()

	return pks, nil
}

func (mgr *dbManager) getRecordID(record *Record, pks []string) (int64, bool) {
	if len(pks) == 0 || record == nil {
		return 0, false
	}

	firstPK := pks[0]
	for k, v := range record.columns {
		if strings.EqualFold(k, firstPK) {
			// 尝试多种方式转换主键值为 int64
			// 优先复用 converter.go 的转换逻辑，保证行为一致
			val := v
			if b, ok := val.([]byte); ok {
				val = string(b)
			}
			if i, err := Convert.ToInt64WithError(val); err == nil {
				return i, true
			}
			// 兜底：尝试将任意值格式化为字符串再解析
			if i, err := strconv.ParseInt(fmt.Sprintf("%v", val), 10, 64); err == nil {
				return i, true
			}
			break
		}
	}
	return 0, false
}

func (mgr *dbManager) isInt64PrimaryKey(table string, pk string) bool {
	if table == "" || pk == "" {
		return false
	}
	columns, err := mgr.getTableColumns(table)
	if err != nil {
		return false
	}
	for _, col := range columns {
		if strings.EqualFold(col.Name, pk) {
			t := strings.ToLower(col.Type)
			if strings.Contains(t, "serial") || strings.Contains(t, "number") {
				return true
			}
			// 更严格的 int 判断：避免 interval 之类的误匹配
			if strings.Contains(t, "int") && !strings.Contains(t, "interval") {
				return true
			}
			return false
		}
	}
	return false
}

func (mgr *dbManager) saveRecord(executor sqlExecutor, table string, record *Record) (int64, error) {
	if err := validateIdentifier(table); err != nil {
		return 0, err
	}
	if record == nil || len(record.columns) == 0 {
		return 0, fmt.Errorf("record is empty")
	}

	pks, _ := mgr.getPrimaryKeys(executor, table)
	if len(pks) == 0 {
		// 没有主键，直接执行插入
		return mgr.insertRecord(executor, table, record)
	}

	// 检查 Record 中是否包含所有主键字段
	pkConditions := []string{}
	pkValues := []interface{}{}
	allPKsFound := true

	for _, pk := range pks {
		found := false
		var val interface{}
		// 尝试大小写敏感查找
		if v, ok := record.columns[pk]; ok {
			val = v
			found = true
		} else {
			// 尝试不区分大小写查找
			for k, v := range record.columns {
				if strings.EqualFold(k, pk) {
					val = v
					found = true
					break
				}
			}
		}

		if !found || val == nil {
			allPKsFound = false
			break
		}
		pkConditions = append(pkConditions, fmt.Sprintf("%s = ?", pk))
		pkValues = append(pkValues, val)
	}

	if allPKsFound {
		// Check if optimistic lock is configured and record has version field
		// If so, we need to use update instead of upsert to properly check version
		config := mgr.getOptimisticLockConfig(table)
		if config != nil && config.VersionField != "" {
			if _, hasVersion := mgr.getVersionFromRecord(table, record); hasVersion {
				// Record has version field, use update with version check
				where := strings.Join(pkConditions, " AND ")
				updateRecord := NewRecord()
				columns, _ := mgr.getOrderedColumns(record, table, executor)
				for _, k := range columns {
					v := record.columns[k]
					isPK := false
					for _, pk := range pks {
						if strings.EqualFold(k, pk) {
							isPK = true
							break
						}
					}
					if !isPK {
						updateRecord.Set(k, v)
					}
				}
				if len(updateRecord.columns) > 0 {
					return mgr.update(executor, table, updateRecord, where, pkValues...)
				}
				return 0, nil
			}
		}

		// 如果是 MySQL, PostgreSQL, SQLite, Oracle, SQLServer，使用原生的 Upsert 语法
		driver := mgr.config.Driver
		if driver == MySQL || driver == PostgreSQL || driver == SQLite3 || driver == Oracle || driver == SQLServer {
			return mgr.nativeUpsert(executor, table, record, pks)
		}

		// 所有主键字段都存在，检查记录是否存在
		where := strings.Join(pkConditions, " AND ")
		exists, err := mgr.exists(executor, table, where, pkValues...)
		if err == nil && exists {
			// 记录存在，执行更新
			updateRecord := NewRecord()
			columns, _ := mgr.getOrderedColumns(record, table, executor)
			for _, k := range columns {
				v := record.columns[k]
				isPK := false
				for _, pk := range pks {
					if strings.EqualFold(k, pk) {
						isPK = true
						break
					}
				}
				if !isPK {
					updateRecord.Set(k, v)
				}
			}
			// 如果除了主键还有其他字段，则执行更新
			if len(updateRecord.columns) > 0 {
				return mgr.update(executor, table, updateRecord, where, pkValues...)
			}
			return 0, nil // 只有主键且已存在，无需更新
		}
	}

	// 记录不存在或不包含完整主键，执行插入
	return mgr.insertRecord(executor, table, record)
}

// getOrderedColumns 返回 Record 中的列名和对应的值，根据数据库类型决定是否排除自增列
// 用于 UPDATE 操作
// SQL Server 和 Oracle: 排除自增列（这些数据库不允许更新自增列）
// MySQL/PostgreSQL/SQLite: 不排除自增列（这些数据库允许更新自增列）
// 注意：列不排序以提高性能，SQL 执行不受列顺序影响
func (mgr *dbManager) getOrderedColumns(record *Record, table string, executor sqlExecutor) ([]string, []interface{}) {
	if record == nil || len(record.columns) == 0 {
		return nil, nil
	}

	// 获取自增列名（利用缓存）
	identityCol := mgr.getIdentityColumn(executor, table)

	columns := make([]string, 0, len(record.columns))
	values := make([]interface{}, 0, len(record.columns))

	for col, val := range record.columns {
		// 只对 SQL Server 和 Oracle 排除自增列
		// 这两个数据库不允许更新 IDENTITY 列
		// MySQL/PostgreSQL/SQLite 允许更新自增列（虽然不推荐，但不会报错）
		if identityCol != "" && strings.EqualFold(col, identityCol) {
			driver := mgr.config.Driver
			if driver == SQLServer || driver == Oracle {
				continue // 跳过自增列
			}
		}

		columns = append(columns, col)
		values = append(values, val)
	}

	return columns, values
}

// getOrderedColumnsForInsert 返回 Record 中的列名和对应的值，排除 nil 值和零值的自增列
// 用于 INSERT 操作，避免插入 NULL 值和自增列的零值
// 智能检测逻辑：
// - nil 值：总是排除
// - 自增列的零值（0, 0.0, ""等）：排除（让数据库自动生成）
// - 自增列的非零值：保留（支持数据迁移场景）
func (mgr *dbManager) getOrderedColumnsForInsert(record *Record, table string, executor sqlExecutor) ([]string, []interface{}) {
	if record == nil || len(record.columns) == 0 {
		return nil, nil
	}

	// 获取自增列名（利用缓存）
	identityCol := mgr.getIdentityColumn(executor, table)

	columns := make([]string, 0, len(record.columns))
	values := make([]interface{}, 0, len(record.columns))

	for col, val := range record.columns {
		// 1. 排除 nil 值
		if val == nil {
			continue
		}

		// 2. 如果是自增列，检查是否为零值
		if identityCol != "" && strings.EqualFold(col, identityCol) {
			if isZeroValue(val) {
				// 零值：排除（让数据库自动生成）
				continue
			}
			// 非零值：保留（支持数据迁移）
		}

		columns = append(columns, col)
		values = append(values, val)
	}

	return columns, values
}

func (mgr *dbManager) nativeUpsert(executor sqlExecutor, table string, record *Record, pks []string) (int64, error) {
	driver := mgr.config.Driver

	// 如果是 Oracle 或 SQL Server，使用 MERGE 语句
	if driver == Oracle || driver == SQLServer {
		return mgr.mergeUpsert(executor, table, record, pks)
	}

	// Apply created_at timestamp for INSERT part of upsert
	// 为 upsert 的 INSERT 部分应用 created_at 时间戳
	if mgr.enableTimestampCheck {
		mgr.applyCreatedAtTimestamp(table, record, false)
	}

	// Apply version initialization for optimistic lock (for INSERT part of upsert)
	mgr.applyVersionInit(table, record)

	columns, values := mgr.getOrderedColumnsForInsert(record, table, executor)
	var placeholders []string
	for range columns {
		placeholders = append(placeholders, "?")
	}

	identityCol := mgr.getIdentityColumn(executor, table)
	_ = identityCol // 目前在 nativeUpsert 中仅作为保留，用于后续可能的扩展

	sqlStr := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, joinStrings(columns), joinStrings(placeholders))

	var updateClauses []string
	for _, col := range columns {
		isPK := false
		for _, pk := range pks {
			if strings.EqualFold(col, pk) {
				isPK = true
				break
			}
		}
		if !isPK {
			if driver == MySQL {
				updateClauses = append(updateClauses, fmt.Sprintf("%s = VALUES(%s)", col, col))
			} else { // PostgreSQL, SQLite
				updateClauses = append(updateClauses, fmt.Sprintf("%s = EXCLUDED.%s", col, col))
			}
		}
	}

	// 为 UPDATE 部分添加 updated_at 时间戳
	if mgr.enableTimestampCheck {
		config := mgr.getTimestampConfig(table)
		if config != nil && config.UpdatedAtField != "" {
			// 检查是否已经在 updateClauses 中
			found := false
			for _, clause := range updateClauses {
				if strings.Contains(strings.ToLower(clause), strings.ToLower(config.UpdatedAtField)) {
					found = true
					break
				}
			}
			if !found {
				if driver == MySQL {
					updateClauses = append(updateClauses, fmt.Sprintf("%s = NOW()", config.UpdatedAtField))
				} else { // PostgreSQL, SQLite
					updateClauses = append(updateClauses, fmt.Sprintf("%s = CURRENT_TIMESTAMP", config.UpdatedAtField))
				}
			}
		}
	}

	// 如果有 ON DUPLICATE/CONFLICT 子句，我们需要确保在插入部分正确处理自增列
	// 对于 MySQL/PG/SQLite 的 nativeUpsert，如果 record 中包含自增列，
	// 数据库通常会自动处理（如果为 null 或 0 则自增，如果提供了值则使用该值）。
	// 这与 MERGE 语法强制要求排除 IDENTITY 不同。
	// 因此这里保持现状，允许 INSERT 部分包含所有 Record 字段。

	if len(updateClauses) > 0 {
		if driver == MySQL {
			sqlStr += " ON DUPLICATE KEY UPDATE " + joinStrings(updateClauses)
		} else { // PostgreSQL, SQLite
			sqlStr += fmt.Sprintf(" ON CONFLICT (%s) DO UPDATE SET %s", joinStrings(pks), joinStrings(updateClauses))
		}
	} else {
		// 如果只有主键字段，执行一个无意义的更新以确保能返回 ID
		if driver == MySQL {
			sqlStr += fmt.Sprintf(" ON DUPLICATE KEY UPDATE %s = %s", pks[0], pks[0])
		} else {
			sqlStr += fmt.Sprintf(" ON CONFLICT (%s) DO UPDATE SET %s = EXCLUDED.%s", joinStrings(pks), pks[0], pks[0])
		}
	}

	sqlStr = mgr.convertPlaceholder(sqlStr, driver)
	values = mgr.sanitizeArgs(sqlStr, values)

	// 处理 PostgreSQL 的 ID 返回
	if driver == PostgreSQL {
		if len(pks) == 1 && strings.EqualFold(pks[0], "id") {
			sqlStr += " RETURNING id"
			var id int64
			start := time.Now()
			err := executor.QueryRow(sqlStr, values...).Scan(&id)
			mgr.logTrace(start, sqlStr, values, err)
			if err != nil {
				return 0, err
			}
			return id, nil
		}
	}

	start := time.Now()
	res, err := executor.Exec(sqlStr, values...)
	mgr.logTrace(start, sqlStr, values, err)
	if err != nil {
		return 0, err
	}

	// 1. 如果 Record 中已经包含了主键（通常是 Update 场景），优先返回它
	// 这样可以避免某些数据库（如 SQLite）在 Upsert 后 LastInsertId 返回不相关的值
	if id, ok := mgr.getRecordID(record, pks); ok {
		rows, _ := res.RowsAffected()
		if rows > 0 {
			return id, nil
		}
	}

	// 2. 否则对于 MySQL/SQLite 返回最后插入的 ID（通常是 Insert 场景）
	if driver == MySQL || driver == SQLite3 {
		id, _ := res.LastInsertId()
		if id > 0 {
			return id, nil
		}
	}

	return res.RowsAffected()
}

func (mgr *dbManager) mergeUpsert(executor sqlExecutor, table string, record *Record, pks []string) (int64, error) {
	driver := mgr.config.Driver

	// Apply created_at timestamp for INSERT part of merge
	// 为 merge 的 INSERT 部分应用 created_at 时间戳
	if mgr.enableTimestampCheck {
		mgr.applyCreatedAtTimestamp(table, record, false)
	}

	// Apply version initialization for optimistic lock (for INSERT part of merge)
	mgr.applyVersionInit(table, record)

	columns, values := mgr.getOrderedColumnsForInsert(record, table, executor)

	// 构造 USING 子句
	var selectCols []string
	for _, col := range columns {
		selectCols = append(selectCols, "? AS "+col)
	}

	usingSQL := "SELECT " + strings.Join(selectCols, ", ")
	if driver == Oracle {
		usingSQL += " FROM DUAL"
	}

	// 构造 ON 子句
	var onClauses []string
	for _, pk := range pks {
		onClauses = append(onClauses, fmt.Sprintf("t.%s = s.%s", pk, pk))
	}

	// 构造 UPDATE 子句
	var updateClauses []string
	for _, col := range columns {
		isPK := false
		for _, pk := range pks {
			if strings.EqualFold(col, pk) {
				isPK = true
				break
			}
		}
		if !isPK {
			updateClauses = append(updateClauses, fmt.Sprintf("t.%s = s.%s", col, col))
		}
	}

	// 为 UPDATE 部分添加 updated_at 时间戳
	if mgr.enableTimestampCheck {
		config := mgr.getTimestampConfig(table)
		if config != nil && config.UpdatedAtField != "" {
			// 检查是否已经在 updateClauses 中
			found := false
			for _, clause := range updateClauses {
				if strings.Contains(strings.ToLower(clause), strings.ToLower(config.UpdatedAtField)) {
					found = true
					break
				}
			}
			if !found {
				if driver == Oracle {
					updateClauses = append(updateClauses, fmt.Sprintf("t.%s = CURRENT_TIMESTAMP", config.UpdatedAtField))
				} else { // SQL Server
					updateClauses = append(updateClauses, fmt.Sprintf("t.%s = GETDATE()", config.UpdatedAtField))
				}
			}
		}
	}

	// 如果只有主键字段，执行一个无意义的更新以确保能触发更新逻辑
	if len(updateClauses) == 0 && len(pks) > 0 {
		updateClauses = append(updateClauses, fmt.Sprintf("t.%s = s.%s", pks[0], pks[0]))
	}

	sqlStr := fmt.Sprintf("MERGE INTO %s t USING (%s) s ON (%s)", table, usingSQL, strings.Join(onClauses, " AND "))

	if len(updateClauses) > 0 {
		sqlStr += " WHEN MATCHED THEN UPDATE SET " + strings.Join(updateClauses, ", ")
	}

	// 构造 INSERT 子句
	var insertCols []string
	var insertVals []string
	identityCol := mgr.getIdentityColumn(executor, table)

	for _, col := range columns {
		isIdentity := false
		// 对于支持 IDENTITY/自增的数据库，在 MERGE/Upsert 插入部分排除自增列
		// 这样数据库会自动生成值，或者避免违反 "GENERATED ALWAYS" 限制
		if identityCol != "" && strings.EqualFold(col, identityCol) {
			isIdentity = true
		}

		if !isIdentity {
			insertCols = append(insertCols, col)
			insertVals = append(insertVals, "s."+col)
		}
	}

	sqlStr += fmt.Sprintf(" WHEN NOT MATCHED THEN INSERT (%s) VALUES (%s)",
		strings.Join(insertCols, ", "),
		strings.Join(insertVals, ", "))

	if driver == SQLServer {
		sqlStr += ";" // SQL Server 的 MERGE 语句必须以分号结束
	}

	sqlStr = mgr.convertPlaceholder(sqlStr, driver)
	values = mgr.sanitizeArgs(sqlStr, values)

	// 对于 SQL Server，如果我们需要获取生成的 ID，可以使用 OUTPUT 子句
	// 但这会改变执行方式（从 Exec 变为 QueryRow），为了保持简单，我们先解决报错问题
	start := time.Now()
	res, err := executor.Exec(sqlStr, values...)
	mgr.logTrace(start, sqlStr, values, err)
	if err != nil {
		return 0, err
	}

	// 如果是 SQL Server 且执行的是 MERGE (Save)，RowsAffected 可能无法准确反映新生成的 ID
	// 但至少现在不会报错了。如果用户提供了主键值，我们返回它。
	if id, ok := mgr.getRecordID(record, pks); ok {
		return id, nil
	}

	return res.RowsAffected()
}

func isTimeValue(v interface{}) bool {
	if v == nil {
		return false
	}
	if _, ok := v.(time.Time); ok {
		return true
	}
	if tp, ok := v.(*time.Time); ok && tp != nil {
		return true
	}
	return false
}

func (mgr *dbManager) insertRecord(executor sqlExecutor, table string, record *Record) (int64, error) {
	return mgr.insertRecordWithOptions(executor, table, record, false)
}

func (mgr *dbManager) insertRecordWithOptions(executor sqlExecutor, table string, record *Record, skipTimestamps bool) (int64, error) {
	if err := validateIdentifier(table); err != nil {
		return 0, err
	}
	if record == nil || len(record.columns) == 0 {
		return 0, fmt.Errorf("record is empty")
	}

	// Apply created_at timestamp
	mgr.applyCreatedAtTimestamp(table, record, skipTimestamps)

	// Apply version initialization for optimistic lock
	mgr.applyVersionInit(table, record)

	driver := mgr.config.Driver
	columns, values := mgr.getOrderedColumnsForInsert(record, table, executor)
	var placeholders []string

	// Oracle: 为日期类型字段使用 TO_DATE 函数
	if driver == Oracle {
		for _, val := range values {
			if isTimeValue(val) {
				placeholders = append(placeholders, "TO_DATE(?, 'YYYY-MM-DD HH24:MI:SS')")
			} else {
				placeholders = append(placeholders, "?")
			}
		}
	} else {
		for range columns {
			placeholders = append(placeholders, "?")
		}
	}

	querySQL := fmt.Sprintf("INSERT INTO %s (%s)", table, joinStrings(columns))

	if driver == PostgreSQL {
		pks, _ := mgr.getPrimaryKeys(executor, table)
		// 只要存在单列主键就使用 RETURNING 获取自增 ID
		if len(pks) == 1 && mgr.isInt64PrimaryKey(table, pks[0]) {
			querySQLWithReturning := querySQL + fmt.Sprintf(" VALUES (%s) RETURNING %s", joinStrings(placeholders), pks[0])
			querySQLWithReturning = mgr.convertPlaceholder(querySQLWithReturning, driver)
			valuesForReturning := mgr.sanitizeArgs(querySQLWithReturning, values)
			var id int64
			start := time.Now()
			err := executor.QueryRow(querySQLWithReturning, valuesForReturning...).Scan(&id)
			mgr.logTrace(start, querySQLWithReturning, valuesForReturning, err)
			if err != nil {
				return 0, err
			}
			return id, nil
		}
		// 否则执行普通插入（此时 querySQL 未被修改）
		querySQL += fmt.Sprintf(" VALUES (%s)", joinStrings(placeholders))
		querySQL = mgr.convertPlaceholder(querySQL, driver)
		values = mgr.sanitizeArgs(querySQL, values)
		start := time.Now()
		res, err := executor.Exec(querySQL, values...)
		mgr.logTrace(start, querySQL, values, err)
		if err != nil {
			return 0, err
		}
		if len(pks) > 0 {
			if id, ok := mgr.getRecordID(record, pks); ok {
				return id, nil
			}
		}
		return res.RowsAffected()
	}

	if driver == SQLServer {
		pks, _ := mgr.getPrimaryKeys(executor, table)
		identityCol := mgr.getIdentityColumn(executor, table)
		// 只有当确定存在标识列且它是唯一主键时，才使用 SCOPE_IDENTITY
		if len(pks) == 1 && identityCol != "" && strings.EqualFold(pks[0], identityCol) {
			querySQLWithIdentity := querySQL + fmt.Sprintf(" VALUES (%s); SELECT SCOPE_IDENTITY()", joinStrings(placeholders))
			querySQLWithIdentity = mgr.convertPlaceholder(querySQLWithIdentity, driver)
			valuesForIdentity := mgr.sanitizeArgs(querySQLWithIdentity, values)
			var id int64
			start := time.Now()
			err := executor.QueryRow(querySQLWithIdentity, valuesForIdentity...).Scan(&id)
			mgr.logTrace(start, querySQLWithIdentity, valuesForIdentity, err)
			if err == nil {
				return id, nil
			}
		}

		// 如果 SCOPE_IDENTITY 路径失败或未执行，使用普通插入（此时 querySQL 未被修改）
		querySQL += fmt.Sprintf(" VALUES (%s)", joinStrings(placeholders))
		querySQL = mgr.convertPlaceholder(querySQL, driver)
		values = mgr.sanitizeArgs(querySQL, values)
		start := time.Now()
		res, err := executor.Exec(querySQL, values...)
		mgr.logTrace(start, querySQL, values, err)
		if err != nil {
			return 0, err
		}

		// 如果主键存在且非自增，尝试返回主键值
		if id, ok := mgr.getRecordID(record, pks); ok {
			return id, nil
		}
		return res.RowsAffected()
	}

	if driver == Oracle {
		pks, _ := mgr.getPrimaryKeys(executor, table)

		// 1. 如果 Record 中已经包含了主键，优先执行并返回该主键
		if id, ok := mgr.getRecordID(record, pks); ok {
			querySQL += fmt.Sprintf(" VALUES (%s)", joinStrings(placeholders))
			querySQL = mgr.convertPlaceholder(querySQL, driver)
			values = mgr.sanitizeArgs(querySQL, values)
			start := time.Now()
			_, err := executor.Exec(querySQL, values...)
			mgr.logTrace(start, querySQL, values, err)
			if err != nil {
				return 0, err
			}
			return id, nil
		}

		// 2. 否则尝试使用 RETURNING 获取新生成的 ID
		if len(pks) == 1 && mgr.isInt64PrimaryKey(table, pks[0]) {
			returningSql := querySQL + fmt.Sprintf(" VALUES (%s) RETURNING %s INTO ?", joinStrings(placeholders), pks[0])
			returningSql = mgr.convertPlaceholder(returningSql, driver)
			valuesForReturning := mgr.sanitizeArgs(returningSql, values)
			start := time.Now()

			var lastID int64
			argsWithOut := append(valuesForReturning, sql.Out{Dest: &lastID})
			_, err := executor.Exec(returningSql, argsWithOut...)
			mgr.logTrace(start, returningSql, valuesForReturning, err)
			if err == nil {
				return lastID, nil
			}
		}

		// 3. 最后退回到普通插入（此时 querySQL 未被修改）
		querySQL += fmt.Sprintf(" VALUES (%s)", joinStrings(placeholders))
		querySQL = mgr.convertPlaceholder(querySQL, driver)
		values = mgr.sanitizeArgs(querySQL, values)
		start := time.Now()
		res, err := executor.Exec(querySQL, values...)
		mgr.logTrace(start, querySQL, values, err)
		if err != nil {
			return 0, err
		}
		return res.RowsAffected()
	}

	querySQL += fmt.Sprintf(" VALUES (%s)", joinStrings(placeholders))
	querySQL = mgr.convertPlaceholder(querySQL, driver)
	values = mgr.sanitizeArgs(querySQL, values)
	start := time.Now()
	result, err := executor.Exec(querySQL, values...)
	mgr.logTrace(start, querySQL, values, err)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (mgr *dbManager) update(executor sqlExecutor, table string, record *Record, where string, whereArgs ...interface{}) (int64, error) {
	// If both feature checks are disabled, use fast path
	if !mgr.enableTimestampCheck && !mgr.enableOptimisticLockCheck {
		return mgr.updateRecordFast(executor, table, record, where, whereArgs...)
	}
	// Feature checks enabled, use full path
	return mgr.updateRecordWithOptions(executor, table, record, where, false, whereArgs...)
}

// updateFast is a lightweight update that skips timestamp and optimistic lock checks for better performance
func (mgr *dbManager) updateRecordFast(executor sqlExecutor, table string, record *Record, where string, whereArgs ...interface{}) (int64, error) {
	if err := validateIdentifier(table); err != nil {
		return 0, err
	}
	if record == nil || len(record.columns) == 0 {
		return 0, fmt.Errorf("record is empty")
	}

	columns, values := mgr.getOrderedColumns(record, table, executor)
	var setClauses []string

	// Oracle: 为日期类型字段使用 TO_DATE 函数
	if mgr.config.Driver == Oracle {
		for i, col := range columns {
			if isTimeValue(values[i]) {
				setClauses = append(setClauses, fmt.Sprintf("%s = TO_DATE(?, 'YYYY-MM-DD HH24:MI:SS')", col))
			} else {
				setClauses = append(setClauses, fmt.Sprintf("%s = ?", col))
			}
		}
	} else {
		for _, col := range columns {
			setClauses = append(setClauses, fmt.Sprintf("%s = ?", col))
		}
	}

	values = append(values, whereArgs...)

	var querySQL string
	if where != "" {
		querySQL = fmt.Sprintf("UPDATE %s SET %s WHERE %s", table, joinStrings(setClauses), where)
	} else {
		querySQL = fmt.Sprintf("UPDATE %s SET %s", table, joinStrings(setClauses))
	}

	querySQL = mgr.convertPlaceholder(querySQL, mgr.config.Driver)
	values = mgr.sanitizeArgs(querySQL, values)
	start := time.Now()
	result, err := executor.Exec(querySQL, values...)
	mgr.logTrace(start, querySQL, values, err)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func (mgr *dbManager) updateRecordWithOptions(executor sqlExecutor, table string, record *Record, where string, skipTimestamps bool, whereArgs ...interface{}) (int64, error) {
	if err := validateIdentifier(table); err != nil {
		return 0, err
	}
	if record == nil || len(record.columns) == 0 {
		return 0, fmt.Errorf("record is empty")
	}

	// Apply updated_at timestamp (only if feature is enabled)
	if mgr.enableTimestampCheck {
		mgr.applyUpdatedAtTimestamp(table, record, skipTimestamps)
	}

	// Check for optimistic lock (only if feature is enabled)
	versionChecked := false
	var currentVersion int64
	var config *OptimisticLockConfig
	if mgr.enableOptimisticLockCheck {
		config = mgr.getOptimisticLockConfig(table)
		if config != nil && config.VersionField != "" {
			if ver, ok := mgr.getVersionFromRecord(table, record); ok {
				currentVersion = ver
				versionChecked = true
				// Remove version from record so it's not in the regular SET clause
				// We'll add it separately with increment
				record.Remove(config.VersionField)
			}
		}
	}

	columns, values := mgr.getOrderedColumns(record, table, executor)
	var setClauses []string

	// Oracle: 为日期类型字段使用 TO_DATE 函数
	if mgr.config.Driver == Oracle {
		for i, col := range columns {
			colName := strings.ToUpper(col)
			if isTimeValue(values[i]) {
				setClauses = append(setClauses, fmt.Sprintf("%s = TO_DATE(?, 'YYYY-MM-DD HH24:MI:SS')", colName))
			} else {
				setClauses = append(setClauses, fmt.Sprintf("%s = ?", colName))
			}
		}
	} else {
		for _, col := range columns {
			// 根据数据库类型转换字段名大小写
			colName := col
			if mgr.config.Driver == Oracle {
				colName = strings.ToUpper(col)
			}
			setClauses = append(setClauses, fmt.Sprintf("%s = ?", colName))
		}
	}

	// Add version increment to SET clause if optimistic lock is enabled and version was found
	if versionChecked && config != nil {
		// 根据数据库类型转换字段名大小写
		versionFieldName := config.VersionField
		if mgr.config.Driver == Oracle {
			versionFieldName = strings.ToUpper(versionFieldName)
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", versionFieldName))
		values = append(values, currentVersion+1)
	}

	// Add version check to WHERE clause
	if versionChecked && config != nil {
		// 根据数据库类型转换字段名大小写
		versionFieldName := config.VersionField
		if mgr.config.Driver == Oracle {
			versionFieldName = strings.ToUpper(versionFieldName)
		}
		if where != "" {
			where = fmt.Sprintf("(%s) AND %s = ?", where, versionFieldName)
		} else {
			where = fmt.Sprintf("%s = ?", versionFieldName)
		}
		whereArgs = append(whereArgs, currentVersion)
	}

	values = append(values, whereArgs...)

	var querySQL string
	if where != "" {
		querySQL = fmt.Sprintf("UPDATE %s SET %s WHERE %s", table, joinStrings(setClauses), where)
	} else {
		querySQL = fmt.Sprintf("UPDATE %s SET %s", table, joinStrings(setClauses))
	}

	querySQL = mgr.convertPlaceholder(querySQL, mgr.config.Driver)
	values = mgr.sanitizeArgs(querySQL, values)
	start := time.Now()
	result, err := executor.Exec(querySQL, values...)
	mgr.logTrace(start, querySQL, values, err)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	// If version was checked and no rows were affected, it's a version mismatch
	if versionChecked && rowsAffected == 0 {
		return 0, ErrVersionMismatch
	}

	return rowsAffected, nil
}

func (mgr *dbManager) delete(executor sqlExecutor, table string, where string, whereArgs ...interface{}) (int64, error) {
	if err := validateIdentifier(table); err != nil {
		return 0, err
	}
	if where == "" {
		return 0, fmt.Errorf("where condition is required for delete")
	}

	// Check if soft delete is configured for this table
	if mgr.hasSoftDelete(table) {
		return mgr.softDelete(executor, table, where, whereArgs...)
	}

	querySQL := fmt.Sprintf("DELETE FROM %s WHERE %s", table, where)
	querySQL, whereArgs = mgr.prepareQuerySQL(querySQL, whereArgs...)

	start := time.Now()
	result, err := executor.Exec(querySQL, whereArgs...)
	mgr.logTrace(start, querySQL, whereArgs, err)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// deleteRecord 根据 Record 中的主键字段删除记录
// 支持软删除特性
func (mgr *dbManager) deleteRecord(executor sqlExecutor, table string, record *Record) (int64, error) {
	if err := validateIdentifier(table); err != nil {
		return 0, err
	}
	if record == nil || len(record.columns) == 0 {
		return 0, fmt.Errorf("record is empty")
	}

	// 获取表的主键
	pks, err := mgr.getPrimaryKeys(executor, table)
	if err != nil {
		return 0, fmt.Errorf("failed to get primary keys: %v", err)
	}
	if len(pks) == 0 {
		return 0, fmt.Errorf("table %s has no primary key, cannot use DeleteRecord", table)
	}

	// 从 Record 中提取主键值构建 WHERE 条件
	var whereClauses []string
	var whereArgs []interface{}
	for _, pk := range pks {
		if !record.Has(pk) {
			return 0, fmt.Errorf("primary key '%s' not found in record", pk)
		}
		val := record.Get(pk)
		whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", pk))
		whereArgs = append(whereArgs, val)
	}

	where := strings.Join(whereClauses, " AND ")
	// 使用支持软删除的 delete 方法
	return mgr.delete(executor, table, where, whereArgs...)
}

// updateRecord 根据 Record 中的主键字段更新记录
// 支持自动时间戳和乐观锁特性
// updateRecord 根据 Record 中的主键字段更新记录
// 支持自动时间戳和乐观锁特性
func (mgr *dbManager) updateRecord(executor sqlExecutor, table string, record *Record) (int64, error) {
	if err := validateIdentifier(table); err != nil {
		return 0, err
	}
	if record == nil || len(record.columns) == 0 {
		return 0, fmt.Errorf("record is empty")
	}

	// 获取表的主键
	pks, err := mgr.getPrimaryKeys(executor, table)
	if err != nil {
		return 0, fmt.Errorf("failed to get primary keys: %v", err)
	}
	if len(pks) == 0 {
		return 0, fmt.Errorf("table %s has no primary key, cannot use updateRecord", table)
	}

	// 第一步: 直接从 record 中提取主键值构建 WHERE 条件
	// 注意: 不依赖 getOrderedColumns,因为对于 SQL Server/Oracle 它会排除自增列
	var pkClauses []string
	var pkValues []interface{}
	pkMap := make(map[string]bool) // 记录哪些列是主键

	for _, pk := range pks {
		// 直接从 record 中查找主键值
		pkValue := record.Get(pk)
		if pkValue == nil {
			return 0, fmt.Errorf("primary key '%s' not found in record", pk)
		}

		pkClauses = append(pkClauses, fmt.Sprintf("%s = ?", pk))
		pkValues = append(pkValues, pkValue)
		pkMap[strings.ToLower(pk)] = true
	}

	// 第二步: 构建更新字段(排除主键,自增列已在 getOrderedColumns 中处理)
	updateRecord := NewRecord()
	columns, _ := mgr.getOrderedColumns(record, table, executor)

	for _, col := range columns {
		// 跳过主键列(主键不应该被更新)
		if pkMap[strings.ToLower(col)] {
			continue
		}
		updateRecord.Set(col, record.Get(col))
	}

	// 如果没有需要更新的字段,直接返回
	if len(updateRecord.columns) == 0 {
		return 0, nil
	}

	where := strings.Join(pkClauses, " AND ")

	// 如果特性检查都关闭，直接使用快速路径
	if !mgr.enableTimestampCheck && !mgr.enableOptimisticLockCheck {
		return mgr.updateRecordFast(executor, table, updateRecord, where, pkValues...)
	}
	return mgr.update(executor, table, updateRecord, where, pkValues...)
}

func (mgr *dbManager) count(executor sqlExecutor, table string, where string, whereArgs ...interface{}) (int64, error) {
	if err := validateIdentifier(table); err != nil {
		return 0, err
	}
	var querySQL string
	if where != "" {
		querySQL = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", table, where)
	} else {
		querySQL = fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
	}
	querySQL = mgr.convertPlaceholder(querySQL, mgr.config.Driver)
	whereArgs = mgr.sanitizeArgs(querySQL, whereArgs)

	var count int64
	start := time.Now()
	err := executor.QueryRow(querySQL, whereArgs...).Scan(&count)
	mgr.logTrace(start, querySQL, whereArgs, err)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (mgr *dbManager) exists(executor sqlExecutor, table string, where string, whereArgs ...interface{}) (bool, error) {
	count, err := mgr.count(executor, table, where, whereArgs...)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (mgr *dbManager) batchInsertRecord(executor sqlExecutor, table string, records []*Record, batchSize int) (int64, error) {
	if err := validateIdentifier(table); err != nil {
		return 0, err
	}
	if len(records) == 0 {
		return 0, nil
	}
	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}

	// 应用时间戳功能到每条记录
	for i := range records {
		mgr.applyCreatedAtTimestamp(table, records[i], false)
	}

	var totalAffected int64
	driver := mgr.config.Driver

	// Extract and sort columns once for all batches
	// 使用第一条记录获取列信息，并排除自增列的零值
	firstRecord := records[0]
	columns, _ := mgr.getOrderedColumnsForInsert(firstRecord, table, executor)

	numCols := len(columns)
	colNamesJoined := joinStrings(columns)

	// Pre-generate row placeholders for drivers that support multi-row INSERT
	var rowPlaceholder string
	if driver != PostgreSQL && driver != SQLServer && driver != Oracle {
		placeholders := make([]string, numCols)
		for i := range placeholders {
			placeholders[i] = "?"
		}
		rowPlaceholder = "(" + joinStrings(placeholders) + ")"
	}

	for i := 0; i < len(records); i += batchSize {
		end := i + batchSize
		if end > len(records) {
			end = len(records)
		}

		batch := records[i:end]
		batchLen := len(batch)

		var querySQL string
		var flatArgs []interface{}

		if driver == PostgreSQL {
			flatArgs = make([]interface{}, 0, batchLen*numCols)
			var sb strings.Builder
			sb.WriteString("INSERT INTO ")
			sb.WriteString(table)
			sb.WriteString(" (")
			sb.WriteString(colNamesJoined)
			sb.WriteString(") VALUES ")

			for rowIdx, record := range batch {
				if rowIdx > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString("(")
				record.mu.RLock()
				for colIdx, col := range columns {
					if colIdx > 0 {
						sb.WriteString(", ")
					}
					placeholderIdx := rowIdx*numCols + colIdx + 1
					sb.WriteString("$")
					sb.WriteString(strconv.Itoa(placeholderIdx))
					flatArgs = append(flatArgs, record.columns[col])
				}
				record.mu.RUnlock()
				sb.WriteString(")")
			}
			querySQL = sb.String()
		} else if driver == Oracle {
			// Oracle: 使用 INSERT ALL 语法实现真正的批量插入
			flatArgs = make([]interface{}, 0, batchLen*numCols)
			var sb strings.Builder
			sb.WriteString("INSERT ALL")

			// 根据第一条记录生成占位符模板（处理日期类型）
			var rowTemplate strings.Builder
			rowTemplate.WriteString(" INTO ")
			rowTemplate.WriteString(table)
			rowTemplate.WriteString(" (")
			rowTemplate.WriteString(colNamesJoined)
			rowTemplate.WriteString(") VALUES (")
			for j, col := range columns {
				if j > 0 {
					rowTemplate.WriteString(", ")
				}
				if isTimeValue(batch[0].getValue(col)) {
					rowTemplate.WriteString("TO_DATE(?, 'YYYY-MM-DD HH24:MI:SS')")
				} else {
					rowTemplate.WriteString("?")
				}
			}
			rowTemplate.WriteString(")")
			rowTemplateStr := rowTemplate.String()

			for _, record := range batch {
				sb.WriteString(rowTemplateStr)
				record.mu.RLock()
				for _, col := range columns {
					flatArgs = append(flatArgs, record.columns[col])
				}
				record.mu.RUnlock()
			}
			sb.WriteString(" SELECT 1 FROM DUAL")
			querySQL = sb.String()
		} else {
			// MySQL, SQLite, SQLServer: 使用 INSERT INTO ... VALUES (...), (...) 语法
			// SQLServer 2008+ 已支持多行 VALUES
			flatArgs = make([]interface{}, 0, batchLen*numCols)
			var sb strings.Builder
			sb.WriteString("INSERT INTO ")
			sb.WriteString(table)
			sb.WriteString(" (")
			sb.WriteString(colNamesJoined)
			sb.WriteString(") VALUES ")

			for rowIdx, record := range batch {
				if rowIdx > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(rowPlaceholder)
				record.mu.RLock()
				for _, col := range columns {
					flatArgs = append(flatArgs, record.columns[col])
				}
				record.mu.RUnlock()
			}
			querySQL = sb.String()
		}

		start := time.Now()
		result, err := executor.Exec(querySQL, flatArgs...)
		mgr.logTrace(start, querySQL, flatArgs, err)
		if err != nil {
			return totalAffected, err
		}
		affected, _ := result.RowsAffected()
		totalAffected += affected
	}
	return totalAffected, nil
}

// batchUpdate 批量更新记录（根据主键）
func (mgr *dbManager) batchUpdateRecord(executor sqlExecutor, table string, records []*Record, batchSize int) (int64, error) {
	if err := validateIdentifier(table); err != nil {
		return 0, err
	}
	if len(records) == 0 {
		return 0, nil
	}
	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}

	// 应用时间戳功能到每条记录
	for i := range records {
		mgr.applyUpdatedAtTimestamp(table, records[i], false)
	}

	// 获取表的主键
	pks, err := mgr.getPrimaryKeys(executor, table)
	if err != nil {
		return 0, fmt.Errorf("failed to get primary keys: %v", err)
	}
	if len(pks) == 0 {
		return 0, fmt.Errorf("table %s has no primary key, cannot use BatchUpdate", table)
	}

	// 检查是否启用了乐观锁
	optimisticConfig := mgr.getOptimisticLockConfig(table)
	hasOptimisticLock := optimisticConfig != nil && optimisticConfig.VersionField != ""

	// Extract update columns from all records (union of columns, excluding PKs)
	updateColsMap := make(map[string]bool)
	for _, record := range records {
		record.mu.RLock()
		for col := range record.columns {
			isPK := false
			for _, pk := range pks {
				if strings.EqualFold(col, pk) {
					isPK = true
					break
				}
			}
			if !isPK {
				updateColsMap[col] = true
			}
		}
		record.mu.RUnlock()
	}

	var updateCols []string
	for col := range updateColsMap {
		updateCols = append(updateCols, col)
	}

	if len(updateCols) == 0 {
		return 0, nil // Nothing to update besides PKs
	}

	// 如果启用了乐观锁，使用事务确保数据一致性
	if hasOptimisticLock {
		return mgr.batchUpdateWithOptimisticLockInTransaction(executor, table, records, batchSize, pks, updateCols, optimisticConfig)
	}

	// Build UPDATE SQL once
	var setClauses []string
	for _, col := range updateCols {
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", col))
	}
	var whereClauses []string
	for _, pk := range pks {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", pk))
	}

	querySQL := fmt.Sprintf("UPDATE %s SET %s WHERE %s",
		table, joinStrings(setClauses), joinStrings(whereClauses))
	querySQL = mgr.convertPlaceholder(querySQL, mgr.config.Driver)

	var totalAffected int64
	numUpdateCols := len(updateCols)
	numPKs := len(pks)
	numTotalArgs := numUpdateCols + numPKs

	// Try to use prepared statement for all batches
	var stmt *sql.Stmt
	if preparer, ok := executor.(interface {
		Prepare(query string) (*sql.Stmt, error)
	}); ok {
		stmt, _ = preparer.Prepare(querySQL)
	}
	if stmt != nil {
		defer stmt.Close()
	}

	// 分批处理
	for i := 0; i < len(records); i += batchSize {
		end := i + batchSize
		if end > len(records) {
			end = len(records)
		}
		batch := records[i:end]

		for _, record := range batch {
			values := make([]interface{}, numTotalArgs)
			record.mu.RLock()
			// SET values
			for j, col := range updateCols {
				values[j] = record.columns[col]
			}
			// WHERE values (PKs)
			for j, pk := range pks {
				values[numUpdateCols+j] = record.columns[pk]
			}
			record.mu.RUnlock()

			start := time.Now()
			var result sql.Result
			var err error
			if stmt != nil {
				result, err = stmt.Exec(values...)
			} else {
				sanitizedValues := mgr.sanitizeArgs(querySQL, values)
				result, err = executor.Exec(querySQL, sanitizedValues...)
			}
			mgr.logTrace(start, querySQL, values, err)
			if err != nil {
				return totalAffected, err
			}
			affected, _ := result.RowsAffected()
			totalAffected += affected
		}
	}

	return totalAffected, nil
}

// batchUpdateWithOptimisticLockInTransaction 在事务中进行带乐观锁的批量更新
// 如果任何一条记录更新失败，整个事务会回滚
func (mgr *dbManager) batchUpdateWithOptimisticLockInTransaction(executor sqlExecutor, table string, records []*Record, batchSize int, pks []string, updateCols []string, optimisticConfig *OptimisticLockConfig) (int64, error) {
	// 检查是否已经在事务中
	if _, isTransaction := executor.(*sql.Tx); isTransaction {
		// 已经在事务中，直接执行
		return mgr.batchUpdateWithOptimisticLock(executor, table, records, batchSize, pks, updateCols, optimisticConfig)
	}

	// 不在事务中，创建新事务
	db, ok := executor.(*sql.DB)
	if !ok {
		// 如果不是 *sql.DB，回退到原有逻辑
		return mgr.batchUpdateWithOptimisticLock(executor, table, records, batchSize, pks, updateCols, optimisticConfig)
	}

	// 开始事务
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %v", err)
	}

	// 在事务中执行批量更新
	totalAffected, err := mgr.batchUpdateWithOptimisticLock(tx, table, records, batchSize, pks, updateCols, optimisticConfig)

	if err != nil {
		// 更新失败，回滚事务
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			LogError("批量更新事务回滚失败", map[string]interface{}{
				"db":            mgr.name,
				"table":         table,
				"updateError":   err.Error(),
				"rollbackError": rollbackErr.Error(),
			})
			return 0, fmt.Errorf("update failed: %v, rollback failed: %v", err, rollbackErr)
		}

		LogInfo("批量更新失败，事务已回滚", map[string]interface{}{
			"db":      mgr.name,
			"table":   table,
			"records": len(records),
			"error":   err.Error(),
		})
		return 0, err
	}

	// 更新成功，提交事务
	if err := tx.Commit(); err != nil {
		LogError("批量更新事务提交失败", map[string]interface{}{
			"db":    mgr.name,
			"table": table,
			"error": err.Error(),
		})
		return 0, fmt.Errorf("failed to commit transaction: %v", err)
	}

	LogInfo("批量更新事务提交成功", map[string]interface{}{
		"db":            mgr.name,
		"table":         table,
		"records":       len(records),
		"totalAffected": totalAffected,
	})

	return totalAffected, nil
}

// batchUpdateWithOptimisticLock 执行带乐观锁检查的批量更新（不处理事务）
func (mgr *dbManager) batchUpdateWithOptimisticLock(executor sqlExecutor, table string, records []*Record, batchSize int, pks []string, updateCols []string, optimisticConfig *OptimisticLockConfig) (int64, error) {
	var totalAffected int64

	for i := 0; i < len(records); i += batchSize {
		end := i + batchSize
		if end > len(records) {
			end = len(records)
		}
		batch := records[i:end]

		for _, record := range batch {
			// 获取当前版本号
			currentVersion, hasVersion := mgr.getVersionFromRecord(table, record)
			if !hasVersion {
				return totalAffected, fmt.Errorf("record missing version field for optimistic lock")
			}

			// 构建更新条件，包含版本检查
			var whereClauses []string
			var whereValues []interface{}

			// 添加主键条件
			for _, pk := range pks {
				whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", pk))
				whereValues = append(whereValues, record.Get(pk))
			}

			// 添加版本条件
			whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", optimisticConfig.VersionField))
			whereValues = append(whereValues, currentVersion)

			// 构建SET子句
			var setClauses []string
			var setValues []interface{}
			for _, col := range updateCols {
				if strings.EqualFold(col, optimisticConfig.VersionField) {
					// 版本字段递增
					setClauses = append(setClauses, fmt.Sprintf("%s = ?", col))
					setValues = append(setValues, currentVersion+1)
				} else {
					setClauses = append(setClauses, fmt.Sprintf("%s = ?", col))
					setValues = append(setValues, record.Get(col))
				}
			}

			// 构建完整的UPDATE语句
			querySQL := fmt.Sprintf("UPDATE %s SET %s WHERE %s",
				table, strings.Join(setClauses, ", "), strings.Join(whereClauses, " AND "))
			querySQL = mgr.convertPlaceholder(querySQL, mgr.config.Driver)

			// 合并参数
			allValues := append(setValues, whereValues...)
			allValues = mgr.sanitizeArgs(querySQL, allValues)

			// 执行更新
			start := time.Now()
			result, err := executor.Exec(querySQL, allValues...)
			mgr.logTrace(start, querySQL, allValues, err)
			if err != nil {
				return totalAffected, err
			}

			affected, _ := result.RowsAffected()
			if affected == 0 {
				// 版本冲突，返回详细错误信息
				pkInfo := make(map[string]interface{})
				for _, pk := range pks {
					pkInfo[pk] = record.Get(pk)
				}
				LogWarn("乐观锁版本冲突", map[string]interface{}{
					"db":              mgr.name,
					"table":           table,
					"primaryKeys":     pkInfo,
					"expectedVersion": currentVersion,
					"versionField":    optimisticConfig.VersionField,
				})
				return totalAffected, ErrVersionMismatch
			}
			totalAffected += affected
		}
	}

	return totalAffected, nil
}

// batchDelete 批量删除记录（根据主键）
func (mgr *dbManager) batchDeleteRecord(executor sqlExecutor, table string, records []*Record, batchSize int) (int64, error) {
	if err := validateIdentifier(table); err != nil {
		return 0, err
	}
	if len(records) == 0 {
		return 0, fmt.Errorf("no records to delete")
	}
	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}

	// 检查是否配置了软删除
	softDeleteConfig := mgr.getSoftDeleteConfig(table)
	if softDeleteConfig != nil {
		// 使用软删除
		return mgr.batchSoftDeleteRecord(executor, table, records, batchSize, softDeleteConfig)
	}

	// 获取表的主键
	pks, err := mgr.getPrimaryKeys(executor, table)
	if err != nil {
		return 0, fmt.Errorf("failed to get primary keys: %v", err)
	}
	if len(pks) == 0 {
		return 0, fmt.Errorf("table %s has no primary key, cannot use BatchDelete", table)
	}

	var totalAffected int64
	driver := mgr.config.Driver

	// 分批处理
	for i := 0; i < len(records); i += batchSize {
		end := i + batchSize
		if end > len(records) {
			end = len(records)
		}

		batch := records[i:end]

		// 对于单主键，使用 IN 子句优化
		if len(pks) == 1 {
			pk := pks[0]
			var pkValues []interface{}
			var placeholders []string

			for idx, record := range batch {
				pkVal := record.Get(pk)
				if pkVal == nil {
					continue
				}
				pkValues = append(pkValues, pkVal)
				if driver == PostgreSQL {
					placeholders = append(placeholders, fmt.Sprintf("$%d", idx+1))
				} else if driver == SQLServer {
					placeholders = append(placeholders, fmt.Sprintf("@p%d", idx+1))
				} else if driver == Oracle {
					placeholders = append(placeholders, fmt.Sprintf(":%d", idx+1))
				} else {
					placeholders = append(placeholders, "?")
				}
			}

			if len(pkValues) == 0 {
				continue
			}

			querySQL := fmt.Sprintf("DELETE FROM %s WHERE %s IN (%s)",
				table, pk, strings.Join(placeholders, ", "))

			start := time.Now()
			result, err := executor.Exec(querySQL, pkValues...)
			mgr.logTrace(start, querySQL, pkValues, err)
			if err != nil {
				return totalAffected, err
			}
			affected, _ := result.RowsAffected()
			totalAffected += affected
		} else {
			// 复合主键，使用预处理语句逐条删除
			var whereClauses []string
			for _, pk := range pks {
				whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", pk))
			}

			querySQL := fmt.Sprintf("DELETE FROM %s WHERE %s",
				table, strings.Join(whereClauses, " AND "))
			querySQL = mgr.convertPlaceholder(querySQL, driver)

			// 尝试使用预处理语句
			if preparer, ok := executor.(interface {
				Prepare(query string) (*sql.Stmt, error)
			}); ok {
				stmt, err := preparer.Prepare(querySQL)
				if err == nil {
					defer stmt.Close()

					for _, record := range batch {
						var pkValues []interface{}
						for _, pk := range pks {
							pkValues = append(pkValues, record.Get(pk))
						}

						start := time.Now()
						result, err := stmt.Exec(pkValues...)
						mgr.logTrace(start, querySQL, pkValues, err)
						if err != nil {
							return totalAffected, err
						}
						affected, _ := result.RowsAffected()
						totalAffected += affected
					}
					continue
				}
			}

			// 回退到单条执行
			for _, record := range batch {
				var pkValues []interface{}
				for _, pk := range pks {
					pkValues = append(pkValues, record.Get(pk))
				}

				start := time.Now()
				result, err := executor.Exec(querySQL, pkValues...)
				mgr.logTrace(start, querySQL, pkValues, err)
				if err != nil {
					return totalAffected, err
				}
				affected, _ := result.RowsAffected()
				totalAffected += affected
			}
		}
	}

	return totalAffected, nil
}

// batchSoftDeleteRecord 批量软删除记录
func (mgr *dbManager) batchSoftDeleteRecord(executor sqlExecutor, table string, records []*Record, batchSize int, config *SoftDeleteConfig) (int64, error) {
	// 获取表的主键
	pks, err := mgr.getPrimaryKeys(executor, table)
	if err != nil {
		return 0, fmt.Errorf("failed to get primary keys: %v", err)
	}
	if len(pks) == 0 {
		return 0, fmt.Errorf("table %s has no primary key, cannot use BatchDelete", table)
	}

	var totalAffected int64
	driver := mgr.config.Driver

	// 准备软删除的SET子句
	var setValue string
	var setArgs []interface{}
	switch config.Type {
	case SoftDeleteTimestamp:
		setValue = fmt.Sprintf("%s = ?", config.Field)
		setArgs = []interface{}{time.Now()}
	case SoftDeleteBool:
		setValue = fmt.Sprintf("%s = ?", config.Field)
		setArgs = []interface{}{true}
	}

	// 分批处理
	for i := 0; i < len(records); i += batchSize {
		end := i + batchSize
		if end > len(records) {
			end = len(records)
		}

		batch := records[i:end]

		// 对于单主键，使用 IN 子句优化
		if len(pks) == 1 {
			pk := pks[0]
			var pkValues []interface{}
			var placeholders []string

			for idx, record := range batch {
				pkVal := record.Get(pk)
				if pkVal == nil {
					continue
				}
				pkValues = append(pkValues, pkVal)
				if driver == PostgreSQL {
					placeholders = append(placeholders, fmt.Sprintf("$%d", len(setArgs)+idx+1))
				} else if driver == SQLServer {
					placeholders = append(placeholders, fmt.Sprintf("@p%d", len(setArgs)+idx+1))
				} else if driver == Oracle {
					placeholders = append(placeholders, fmt.Sprintf(":%d", len(setArgs)+idx+1))
				} else {
					placeholders = append(placeholders, "?")
				}
			}

			if len(pkValues) == 0 {
				continue
			}

			querySQL := fmt.Sprintf("UPDATE %s SET %s WHERE %s IN (%s)",
				table, setValue, pk, strings.Join(placeholders, ", "))
			querySQL = mgr.convertPlaceholder(querySQL, driver)

			// 合并参数：SET参数 + WHERE参数
			allArgs := append(setArgs, pkValues...)
			allArgs = mgr.sanitizeArgs(querySQL, allArgs)

			start := time.Now()
			result, err := executor.Exec(querySQL, allArgs...)
			mgr.logTrace(start, querySQL, allArgs, err)
			if err != nil {
				return totalAffected, err
			}
			affected, _ := result.RowsAffected()
			totalAffected += affected
		} else {
			// 复合主键，逐条更新
			var whereClauses []string
			for _, pk := range pks {
				whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", pk))
			}

			querySQL := fmt.Sprintf("UPDATE %s SET %s WHERE %s",
				table, setValue, strings.Join(whereClauses, " AND "))
			querySQL = mgr.convertPlaceholder(querySQL, driver)

			for _, record := range batch {
				var pkValues []interface{}
				for _, pk := range pks {
					pkValues = append(pkValues, record.Get(pk))
				}

				// 合并参数：SET参数 + WHERE参数
				allArgs := append(setArgs, pkValues...)
				allArgs = mgr.sanitizeArgs(querySQL, allArgs)

				start := time.Now()
				result, err := executor.Exec(querySQL, allArgs...)
				mgr.logTrace(start, querySQL, allArgs, err)
				if err != nil {
					return totalAffected, err
				}
				affected, _ := result.RowsAffected()
				totalAffected += affected
			}
		}
	}

	return totalAffected, nil
}

// batchDeleteByIds 根据主键ID列表批量删除
func (mgr *dbManager) batchDeleteByIds(executor sqlExecutor, table string, ids []interface{}, batchSize int) (int64, error) {
	if err := validateIdentifier(table); err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, fmt.Errorf("no ids to delete")
	}
	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}

	// 获取表的主键
	pks, err := mgr.getPrimaryKeys(executor, table)
	if err != nil {
		return 0, fmt.Errorf("failed to get primary keys: %v", err)
	}
	if len(pks) == 0 {
		return 0, fmt.Errorf("table %s has no primary key", table)
	}
	if len(pks) > 1 {
		return 0, fmt.Errorf("BatchDeleteByIds only supports single primary key tables")
	}

	pk := pks[0]
	var totalAffected int64
	driver := mgr.config.Driver

	// 分批处理
	for i := 0; i < len(ids); i += batchSize {
		end := i + batchSize
		if end > len(ids) {
			end = len(ids)
		}

		batch := ids[i:end]
		var placeholders []string

		for idx := range batch {
			if driver == PostgreSQL {
				placeholders = append(placeholders, fmt.Sprintf("$%d", idx+1))
			} else if driver == SQLServer {
				placeholders = append(placeholders, fmt.Sprintf("@p%d", idx+1))
			} else if driver == Oracle {
				placeholders = append(placeholders, fmt.Sprintf(":%d", idx+1))
			} else {
				placeholders = append(placeholders, "?")
			}
		}

		querySQL := fmt.Sprintf("DELETE FROM %s WHERE %s IN (%s)",
			table, pk, strings.Join(placeholders, ", "))

		start := time.Now()
		result, err := executor.Exec(querySQL, batch...)
		mgr.logTrace(start, querySQL, batch, err)
		if err != nil {
			return totalAffected, err
		}
		affected, _ := result.RowsAffected()
		totalAffected += affected
	}

	return totalAffected, nil
}

func (mgr *dbManager) paginate(executor sqlExecutor, querySQL string, page, pageSize int, countCacheTTL time.Duration, args ...interface{}) ([]*Record, int64, error) {
	if page < 1 {
		page = DefaultPage
	}
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	// 限制最大页面大小，防止一次查询过多数据
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	driver := mgr.config.Driver
	baseSQL := querySQL
	if orderIdx := findKeywordIgnoringQuotes(querySQL, "ORDER BY", -1); orderIdx != -1 {
		baseSQL = querySQL[:orderIdx]
	}

	var countSQL string
	// 尝试优化 COUNT 语句
	if optimized, ok := optimizeCountSQL(baseSQL); ok {
		countSQL = optimized
	} else {
		// 如果无法优化（含有 DISTINCT, GROUP BY 等），则使用子查询
		if driver == Oracle {
			countSQL = fmt.Sprintf("SELECT COUNT(*) FROM (%s) sub", baseSQL)
		} else {
			countSQL = fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS sub", baseSQL)
		}
	}

	countSQL = mgr.convertPlaceholder(countSQL, driver)
	args = mgr.sanitizeArgs(countSQL, args)

	var total int64

	// 检查是否启用了计数缓存（countCacheTTL > 0 表示启用）
	if countCacheTTL > 0 {
		// 生成计数缓存键
		countCacheKey := GenerateCacheKey(mgr.name, "COUNT:"+countSQL, args...)
		countCacheRepo := "__eorm_count_cache__"

		// 尝试从缓存获取计数
		if cachedCount, ok := GetLocalCacheInstance().CacheGet(countCacheRepo, countCacheKey); ok {
			if count, ok := cachedCount.(int64); ok {
				total = count
			}
		} else {
			// 缓存未命中，执行 COUNT 查询
			startCount := time.Now()
			err := executor.QueryRow(countSQL, args...).Scan(&total)
			mgr.logTrace(startCount, countSQL, args, err)
			if err != nil {
				return nil, 0, err
			}

			// 将计数结果缓存
			GetLocalCacheInstance().CacheSet(countCacheRepo, countCacheKey, total, countCacheTTL)
		}
	} else {
		// 不使用缓存，直接执行 COUNT 查询
		startCount := time.Now()
		err := executor.QueryRow(countSQL, args...).Scan(&total)
		mgr.logTrace(startCount, countSQL, args, err)
		if err != nil {
			return nil, 0, err
		}
	}

	offset := (page - 1) * pageSize
	var paginatedSQL string
	hasOrderBy := findKeywordIgnoringQuotes(querySQL, "ORDER BY", -1) != -1
	if driver == SQLServer {
		if hasOrderBy {
			paginatedSQL = fmt.Sprintf("%s OFFSET %d ROWS FETCH NEXT %d ROWS ONLY", querySQL, offset, pageSize)
		} else {
			paginatedSQL = fmt.Sprintf("%s ORDER BY (SELECT NULL) OFFSET %d ROWS FETCH NEXT %d ROWS ONLY", querySQL, offset, pageSize)
		}
	} else if driver == Oracle {
		if hasOrderBy {
			paginatedSQL = fmt.Sprintf("SELECT a.* FROM (SELECT a.*, ROWNUM eorm_rn FROM (%s) a WHERE ROWNUM <= %d) a WHERE eorm_rn > %d", querySQL, offset+pageSize, offset)
		} else {
			paginatedSQL = fmt.Sprintf("SELECT a.* FROM (SELECT a.*, ROWNUM eorm_rn FROM (%s ORDER BY 1) a WHERE ROWNUM <= %d) a WHERE eorm_rn > %d", querySQL, offset+pageSize, offset)
		}
	} else {
		paginatedSQL = fmt.Sprintf("%s LIMIT %d OFFSET %d", querySQL, pageSize, offset)
	}

	paginatedSQL = mgr.convertPlaceholder(paginatedSQL, driver)

	startPaginate := time.Now()
	rows, err := executor.Query(paginatedSQL, args...)
	mgr.logTrace(startPaginate, paginatedSQL, args, err)
	if err != nil {
		return nil, total, err
	}
	defer rows.Close()

	results, err := scanRecords(rows, driver)
	if err != nil {
		return nil, total, err
	}
	return results, total, nil
}

// scanRows is a helper function to scan sql.Rows into a slice of maps
func scanRows(rows *sql.Rows) ([]map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	numCols := len(columns)
	var results []map[string]interface{}
	// Reuse slices for each row to reduce allocations
	values := make([]interface{}, numCols)
	valuePtrs := make([]interface{}, numCols)
	for i := range columns {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		entry := make(map[string]interface{}, numCols)
		for i, col := range columns {
			val := values[i]

			// Handle []byte conversion for numeric/decimal types
			if b, ok := val.([]byte); ok {
				dbType := strings.ToUpper(columnTypes[i].DatabaseTypeName())

				if isNumericType(dbType) {
					if s := string(b); s != "" {
						entry[col] = s
					} else {
						entry[col] = nil
					}
					continue
				}

				if !isBinaryType(dbType) {
					entry[col] = string(b)
					continue
				}
				// Keep as []byte for binary types, but we must copy it
				// because the underlying buffer might be reused by the driver
				bCopy := make([]byte, len(b))
				copy(bCopy, b)
				entry[col] = bCopy
				continue
			}

			entry[col] = val
		}
		results = append(results, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func isNumericType(dbType string) bool {
	numericTypes := []string{"DECIMAL", "NUMERIC", "NUMBER", "MONEY", "SMALLMONEY", "DEC", "FIXED"}
	for _, t := range numericTypes {
		if strings.Contains(dbType, t) {
			return true
		}
	}
	return false
}

func isBinaryType(dbType string) bool {
	binaryTypes := []string{"BLOB", "BINARY", "VARBINARY", "BYTEA", "IMAGE", "RAW"}
	for _, t := range binaryTypes {
		if strings.Contains(dbType, t) {
			return true
		}
	}
	return false
}

func findTopLevelKeywordIndex(sql, keyword string, start int) int {
	kwLen := len(keyword)
	if kwLen == 0 || len(sql) < kwLen {
		return -1
	}
	if start < 0 {
		start = 0
	}
	if start > len(sql)-kwLen {
		return -1
	}

	upperSQL := strings.ToUpper(sql)
	upperKW := strings.ToUpper(keyword)

	inSingleQuote := false
	inDoubleQuote := false
	inSingleLineComment := false
	inMultiLineComment := false
	escaped := false
	parenDepth := 0

	for i := start; i <= len(sql)-kwLen; i++ {
		char := sql[i]

		if !inSingleQuote && !inDoubleQuote {
			if !inSingleLineComment && !inMultiLineComment {
				if i+1 < len(sql) && sql[i:i+2] == "--" {
					inSingleLineComment = true
					i++
					continue
				}
				if i+1 < len(sql) && sql[i:i+2] == "/*" {
					inMultiLineComment = true
					i++
					continue
				}
			} else if inSingleLineComment {
				if char == '\n' {
					inSingleLineComment = false
				}
				continue
			} else if inMultiLineComment {
				if i+1 < len(sql) && sql[i:i+2] == "*/" {
					inMultiLineComment = false
					i++
				}
				continue
			}
		}

		if !inSingleLineComment && !inMultiLineComment {
			if char == '\'' && !inDoubleQuote && !escaped {
				if i+1 < len(sql) && sql[i+1] == '\'' {
					i++
				} else {
					inSingleQuote = !inSingleQuote
				}
			} else if char == '"' && !inSingleQuote && !escaped {
				inDoubleQuote = !inDoubleQuote
			}

			if char == '\\' {
				escaped = !escaped
			} else {
				escaped = false
			}
		}

		if !inSingleQuote && !inDoubleQuote && !inSingleLineComment && !inMultiLineComment {
			if char == '(' {
				parenDepth++
			} else if char == ')' {
				if parenDepth > 0 {
					parenDepth--
				}
			}
		}

		if parenDepth == 0 && !inSingleQuote && !inDoubleQuote && !inSingleLineComment && !inMultiLineComment {
			if upperSQL[i:i+kwLen] == upperKW {
				isStartBoundary := i == 0 || !isAlphaNum(sql[i-1])
				isEndBoundary := i+kwLen == len(sql) || !isAlphaNum(sql[i+kwLen])
				if isStartBoundary && isEndBoundary {
					return i
				}
			}
		}
	}

	return -1
}

func optimizeCountSQL(querySQL string) (string, bool) {
	// 如果包含以下关键字，不进行优化，使用子查询最安全
	if findKeywordIgnoringQuotes(querySQL, "DISTINCT", 1) != -1 ||
		findKeywordIgnoringQuotes(querySQL, "GROUP BY", 1) != -1 ||
		findKeywordIgnoringQuotes(querySQL, "UNION", 1) != -1 ||
		findKeywordIgnoringQuotes(querySQL, "HAVING", 1) != -1 ||
		findKeywordIgnoringQuotes(querySQL, "INTERSECT", 1) != -1 ||
		findKeywordIgnoringQuotes(querySQL, "EXCEPT", 1) != -1 {
		return "", false
	}

	// 寻找顶层第一个 FROM（避免误匹配子查询中的 FROM）
	fromIdx := findTopLevelKeywordIndex(querySQL, "FROM", 0)
	if fromIdx == -1 {
		return "", false
	}

	// 剥离顶层 ORDER BY / LIMIT（COUNT 查询不需要）
	endIdx := len(querySQL)
	if orderIdx := findTopLevelKeywordIndex(querySQL, "ORDER BY", fromIdx); orderIdx != -1 && orderIdx < endIdx {
		endIdx = orderIdx
	}
	if limitIdx := findTopLevelKeywordIndex(querySQL, "LIMIT", fromIdx); limitIdx != -1 && limitIdx < endIdx {
		endIdx = limitIdx
	}

	optimized := "SELECT COUNT(*) " + strings.TrimSpace(querySQL[fromIdx:endIdx])
	return optimized, true
}

// scanRecords_inefficiency 是原始的低效实现，保留作为对比参考
// 该实现存在以下性能问题：
// 1. 每次都创建新的Record对象，无法重用
// 2. 通过中间map转换，增加了一次内存分配
// 3. 没有利用已知的列数信息进行精确容量分配
func scanRecords_inefficiency(rows *sql.Rows, driver DriverType) ([]*Record, error) {
	maps, err := scanRows(rows)
	if err != nil {
		return nil, err
	}

	results := make([]*Record, len(maps))
	for i, m := range maps {
		// 直接构造 Record，避免 Set 方法的指针检查开销
		record := &Record{
			columns:     make(map[string]interface{}, len(m)),
			lowerKeyMap: make(map[string]string, len(m)),
		}
		// 使用 setDirect 直接设置，不需要加锁（record 是新创建的局部变量）
		for key, value := range m {
			record.setDirect(key, value)
		}
		results[i] = record
	}
	return results, nil
}

func processDBValue(val interface{}, dbType string) interface{} {
	if val == nil {
		return nil
	}

	if b, ok := val.([]byte); ok {
		if isNumericType(dbType) {
			if s := string(b); s != "" {
				return s
			}
			return nil
		}

		if !isBinaryType(dbType) {
			// 将字节数组转换为字符串，避免外部持有原始切片引用
			return string(b)
		}

		// 对于二进制类型，复制数据避免底层缓冲区重用问题
		bCopy := make([]byte, len(b))
		copy(bCopy, b)
		return bCopy
	}

	// 兜底防御：如果是指针类型且驱动没有处理，尝试解引用（可选扩展）
	// 目前保持通用逻辑，仅对 []byte 这一重灾区做特殊加固

	return val
}

// scanRecords 优化版本 - 直接扫描到Record，避免中间map转换
// 性能优化点：
// 1. 根据实际列数精确分配Record容量，避免Map扩容
// 2. 直接扫描到Record，避免中间map的内存分配
// 3. 重用扫描缓冲区，减少每行的内存分配
// 4. 零Map扩容，完全避免rehashing开销
// 注意：由于需要返回Record值而非指针，这里直接创建精确容量的Record
//
//	对象池更适合用于临时操作的场景
func scanRecords(rows *sql.Rows, driver DriverType) ([]*Record, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	numCols := len(columns)
	var results []*Record

	// 重用扫描缓冲区，避免每行都分配新的slice
	values := make([]interface{}, numCols)
	valuePtrs := make([]interface{}, numCols)
	for i := range columns {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		// 直接创建结果Record，使用精确容量（避免不必要的对象池操作）
		resultRecord := &Record{
			columns:     make(map[string]interface{}, numCols),
			lowerKeyMap: make(map[string]string, numCols),
		}

		for i, col := range columns {
			val := values[i]
			dbType := strings.ToUpper(columnTypes[i].DatabaseTypeName())

			// 使用专门的函数处理数据库值转换
			processedVal := processDBValue(val, dbType)

			// 使用 setDirect 直接设置，跳过 Set 方法的指针检查和加锁
			resultRecord.setDirect(col, processedVal)
		}

		results = append(results, resultRecord)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// scanMaps is a helper function to scan sql.Rows into a slice of map
func scanMaps(rows *sql.Rows, driver DriverType) ([]map[string]interface{}, error) {
	return scanRows(rows)
}

// GetDB returns the underlying database connection
func (db *DB) GetDB() (*sql.DB, error) {
	return db.dbMgr.getDB()
}

// Close closes the database connection
func (db *DB) Close() error {
	if db.dbMgr.db != nil {
		return db.dbMgr.db.Close()
	}
	return nil
}

// SetCurrentDB switches the global default database by name
func SetCurrentDB(dbname string) error {
	multiMgr.mu.RLock()
	_, exists := multiMgr.databases[dbname]
	multiMgr.mu.RUnlock()

	if !exists {
		return fmt.Errorf("database '%s' not found", dbname)
	}

	multiMgr.mu.Lock()
	multiMgr.currentDB = dbname
	multiMgr.mu.Unlock()

	return nil
}

// safeGetCurrentDB returns the current database manager without panicking
func safeGetCurrentDB() (*dbManager, error) {
	if multiMgr == nil {
		return nil, ErrNotInitialized
	}

	multiMgr.mu.RLock()
	currentDB := multiMgr.currentDB
	multiMgr.mu.RUnlock()

	if currentDB == "" {
		return nil, ErrNotInitialized
	}

	dbMgr := GetDatabase(currentDB)
	if dbMgr == nil {
		return nil, ErrNotInitialized
	}

	return dbMgr, nil
}

// GetCurrentDB returns the current database manager
func GetCurrentDB() (*dbManager, error) {
	dbMgr, err := safeGetCurrentDB()
	if err != nil {
		return nil, err
	}
	return dbMgr, nil
}

// GetConfig returns the database configuration
func (mgr *dbManager) GetConfig() (*Config, error) {
	if mgr == nil {
		return nil, fmt.Errorf("database manager is nil")
	}
	return mgr.config, nil
}

// GetDatabase returns the database manager by name
func GetDatabase(dbname string) *dbManager {
	if multiMgr == nil {
		return nil
	}

	multiMgr.mu.RLock()
	defer multiMgr.mu.RUnlock()

	return multiMgr.databases[dbname]
}

// GetDB returns the underlying database connection of current database
func GetDB() (*sql.DB, error) {

	mgr, err := GetCurrentDB()
	if err != nil {
		return nil, err
	}
	// *sql.DB, error
	db, err := mgr.getDB()
	if err != nil {
		return nil, err
	}
	return db, nil
	// return GetCurrentDB().getDB()
}

// GetDBByName returns the database connection by name
func GetDBByName(dbname string) (*sql.DB, error) {
	multiMgr.mu.RLock()
	dbMgr, exists := multiMgr.databases[dbname]
	multiMgr.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("database '%s' not found", dbname)
	}

	db, err := dbMgr.getDB()
	if err != nil {
		return nil, err
	}
	return db, nil
	// return dbMgr.getDB(), nil
}

// Close closes the database connection
// Close closes all database connections
func Close() error {
	if multiMgr == nil {
		return nil
	}

	multiMgr.mu.Lock()
	defer multiMgr.mu.Unlock()

	for _, dbMgr := range multiMgr.databases {
		// 停止连接监控
		cleanupMonitor(dbMgr.name)

		// 清理预编译语句缓存
		dbMgr.clearStmtCache()

		// 关闭数据库连接
		if dbMgr.db != nil {
			dbMgr.db.Close()
		}
	}
	multiMgr.databases = make(map[string]*dbManager)
	multiMgr.currentDB = ""
	multiMgr.defaultDB = ""

	return nil
}

// CloseDB closes a specific database connection by name
func CloseDB(dbname string) error {
	if multiMgr != nil {
		multiMgr.mu.Lock()
		defer multiMgr.mu.Unlock()

		if dbMgr, exists := multiMgr.databases[dbname]; exists {
			// 停止连接监控
			cleanupMonitor(dbname)

			// 清理预编译语句缓存
			dbMgr.clearStmtCache()

			// 关闭数据库连接
			if dbMgr.db != nil {
				dbMgr.db.Close()
				dbMgr.db = nil
			}
			delete(multiMgr.databases, dbname)

			if multiMgr.currentDB == dbname {
				if multiMgr.defaultDB != "" && multiMgr.defaultDB != dbname {
					multiMgr.currentDB = multiMgr.defaultDB
				} else {
					multiMgr.currentDB = ""
				}
			}

			if multiMgr.defaultDB == dbname {
				multiMgr.defaultDB = ""
				for name := range multiMgr.databases {
					multiMgr.defaultDB = name
					break
				}
			}
		}
	}

	return nil
}

// ListDatabases returns the list of registered database names
func ListDatabases() []string {
	var databases []string
	if multiMgr != nil {
		multiMgr.mu.RLock()
		for name := range multiMgr.databases {
			databases = append(databases, name)
		}
		multiMgr.mu.RUnlock()
	}
	return databases
}

// GetCurrentDBName returns the name of the current database
func GetCurrentDBName() string {
	if multiMgr == nil {
		return ""
	}

	multiMgr.mu.RLock()
	defer multiMgr.mu.RUnlock()

	return multiMgr.currentDB
}

// EnableTimestamps enables auto timestamp in Update operations.
// When enabled, Update will check and apply auto timestamp configurations.
// Default is false (disabled) for better performance.
func EnableTimestamps() {
	mgr, err := defaultDB()
	if err != nil {
		return
	}
	mgr.dbMgr.mu.Lock()
	defer mgr.dbMgr.mu.Unlock()
	mgr.dbMgr.enableTimestampCheck = true
}

// EnableTimestamps enables auto timestamp for this database instance.
func (db *DB) EnableTimestamps() *DB {
	if db.lastErr != nil {
		return db
	}
	db.dbMgr.mu.Lock()
	defer db.dbMgr.mu.Unlock()
	db.dbMgr.enableTimestampCheck = true
	return db
}

// Deprecated: Use EnableTimestamps() instead
func EnableTimestampCheck() {
	EnableTimestamps()
}

// Deprecated: Use EnableTimestamps() instead
func (db *DB) EnableTimestampCheck() *DB {
	return db.EnableTimestamps()
}

// EnableOptimisticLock enables optimistic lock in Update operations.
// When enabled, Update will check and apply optimistic lock configurations.
// Default is false (disabled) for better performance.
func EnableOptimisticLock() {
	mgr, err := defaultDB()
	if err != nil {
		return
	}
	mgr.dbMgr.mu.Lock()
	defer mgr.dbMgr.mu.Unlock()
	mgr.dbMgr.enableOptimisticLockCheck = true
}

// EnableOptimisticLock enables optimistic lock for this database instance.
func (db *DB) EnableOptimisticLock() *DB {
	if db.lastErr != nil {
		return db
	}
	db.dbMgr.mu.Lock()
	defer db.dbMgr.mu.Unlock()
	db.dbMgr.enableOptimisticLockCheck = true
	return db
}

// Deprecated: Use EnableOptimisticLock() instead
func EnableOptimisticLockCheck() {
	EnableOptimisticLock()
}

// Deprecated: Use EnableOptimisticLock() instead
func (db *DB) EnableOptimisticLockCheck() *DB {
	return db.EnableOptimisticLock()
}

// EnableSoftDelete enables soft delete in query operations.
// When enabled, queries will automatically filter out soft-deleted records.
// Default is false (disabled) for better performance.
func EnableSoftDelete() {
	mgr, err := defaultDB()
	if err != nil {
		return
	}
	mgr.dbMgr.mu.Lock()
	defer mgr.dbMgr.mu.Unlock()
	mgr.dbMgr.enableSoftDeleteCheck = true
}

// EnableSoftDelete enables soft delete for this database instance.
func (db *DB) EnableSoftDelete() *DB {
	if db.lastErr != nil {
		return db
	}
	db.dbMgr.mu.Lock()
	defer db.dbMgr.mu.Unlock()
	db.dbMgr.enableSoftDeleteCheck = true
	return db
}

// Deprecated: Use EnableSoftDelete() instead
func EnableSoftDeleteCheck() {
	EnableSoftDelete()
}

// Deprecated: Use EnableSoftDelete() instead
func (db *DB) EnableSoftDeleteCheck() *DB {
	return db.EnableSoftDelete()
}

// initDB initializes the database connection
func (mgr *dbManager) initDB() error {
	// 1. 第一层检查：使用 RLock 快速判断是否已初始化
	mgr.mu.RLock()
	if mgr.db != nil {
		mgr.mu.RUnlock()
		return nil
	}
	mgr.mu.RUnlock()

	// 2. 使用专门的初始化锁，防止多个请求同时执行 Ping 等耗时操作
	mgr.initMu.Lock()
	defer mgr.initMu.Unlock()

	// 3. 第二层检查：双重检查锁定 (Double-Checked Locking)
	if mgr.db != nil {
		return nil
	}

	db, err := sql.Open(string(mgr.config.Driver), mgr.config.DSN)
	if err != nil {
		return err
	}

	// Configure connection pool
	db.SetMaxOpenConns(mgr.config.MaxOpen)
	db.SetMaxIdleConns(mgr.config.MaxIdle)
	db.SetConnMaxLifetime(mgr.config.ConnMaxLifetime)

	// 初始化智能语句缓存
	cacheConfig := DefaultStmtCacheConfig()
	if mgr.config.ConnMaxLifetime > 0 {
		// 基础 TTL 设为连接生命周期的 80%（比之前的 50% 更激进）
		cacheConfig.BaseTTL = time.Duration(float64(mgr.config.ConnMaxLifetime) * 0.8)
		mgr.stmtCacheTTL = mgr.config.ConnMaxLifetime / 2 // 保留用于向后兼容
	} else {
		cacheConfig.BaseTTL = 0 // 永不过期
		mgr.stmtCacheTTL = 0
	}
	mgr.stmtCache = newStmtCache(cacheConfig)

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return err
	}

	// 4. 将成功初始化的连接赋值给 mgr.db (持锁以保证原子性)
	mgr.mu.Lock()
	mgr.db = db
	mgr.mu.Unlock()

	// 根据配置启用连接监控
	if mgr.config.MonitorNormalInterval > 0 {
		if err := mgr.startConnectionMonitoring(); err != nil {
			// 监控启动失败不影响数据库连接，只记录警告日志
			LogWarn("连接监控启动失败", map[string]interface{}{
				"database": mgr.name,
				"error":    err.Error(),
			})
		}
	}

	return nil
}

// getDB returns the database connection, initializing if necessary
func (mgr *dbManager) getDB() (*sql.DB, error) {
	if mgr == nil {
		return nil, fmt.Errorf("eorm: database manager is nil. Please call eorm.OpenDatabase()  before using eorm operations")
		// panic("eorm: database manager is nil. Please call eorm.OpenDatabase()  before using eorm operations")
	}
	if mgr.db == nil {
		if err := mgr.initDB(); err != nil {
			//panic(fmt.Sprintf("eorm: failed to initialize database: %v", err))
			return nil, fmt.Errorf("eorm: failed to initialize database: %w", err)
		}
	}
	return mgr.db, nil
}

// Ping checks if the database connection is alive
func (mgr *dbManager) Ping() error {
	return mgr.PingContext(context.Background())
}

// PingContext checks if the database connection is alive with context
func (mgr *dbManager) PingContext(ctx context.Context) error {
	if mgr == nil {
		return fmt.Errorf("database manager not initialized. Please call eorm.OpenDatabase() before using eorm operations")
	}
	if mgr.db == nil {
		return fmt.Errorf("database not initialized")
	}
	return mgr.db.PingContext(ctx)
}

// convertPlaceholder converts ? placeholders to $n for PostgreSQL, @param for SQL Server, or :n for Oracle
func (mgr *dbManager) convertPlaceholder(querySQL string, driver DriverType) string {
	return mgr.convertPlaceholderWithOffset(querySQL, driver, 0)
}

// convertPlaceholderWithOffset converts ? placeholders with an index offset
func (mgr *dbManager) convertPlaceholderWithOffset(querySQL string, driver DriverType, offset int) string {
	if driver == MySQL || driver == SQLite3 {
		return querySQL
	}

	var builder strings.Builder
	builder.Grow(len(querySQL) + 10)
	paramIndex := 1 + offset
	inSingleQuote := false
	inDoubleQuote := false
	inBacktick := false

	for i := 0; i < len(querySQL); i++ {
		char := querySQL[i]

		// Handle escaping (mostly for single quotes but good practice)
		if i+1 < len(querySQL) && char == '\\' {
			builder.WriteByte(char)
			builder.WriteByte(querySQL[i+1])
			i++
			continue
		}

		// Handle string literals and identifiers
		if char == '\'' && !inDoubleQuote && !inBacktick {
			if i+1 < len(querySQL) && querySQL[i+1] == '\'' {
				builder.WriteByte('\'')
				builder.WriteByte('\'')
				i++
				continue
			}
			inSingleQuote = !inSingleQuote
			builder.WriteByte('\'')
			continue
		}

		if char == '"' && !inSingleQuote && !inBacktick {
			inDoubleQuote = !inDoubleQuote
			builder.WriteByte('"')
			continue
		}

		if char == '`' && !inSingleQuote && !inDoubleQuote {
			inBacktick = !inBacktick
			builder.WriteByte('`')
			continue
		}

		if char == '?' && !inSingleQuote && !inDoubleQuote && !inBacktick {
			switch driver {
			case PostgreSQL:
				builder.WriteString(fmt.Sprintf("$%d", paramIndex))
			case SQLServer:
				builder.WriteString(fmt.Sprintf("@p%d", paramIndex))
			case Oracle:
				builder.WriteString(fmt.Sprintf(":%d", paramIndex))
			default:
				builder.WriteByte('?')
			}
			paramIndex++
		} else {
			builder.WriteByte(char)
		}
	}
	return builder.String()
}

// sanitizeArgs 自动清理不必要的参数。如果用户误传了参数，则根据 SQL 中的占位符数量进行截断或清理。
func (mgr *dbManager) sanitizeArgs(querySQL string, args []interface{}) []interface{} {
	if len(args) == 0 {
		return args
	}

	placeholderCount := 0
	switch mgr.config.Driver {
	case PostgreSQL:
		// 使用预编译正则精确匹配 $1, $2...，避免 $1 匹配到 $10
		matches := postgresPlaceholderRe.FindAllStringSubmatch(querySQL, -1)
		for _, match := range matches {
			if len(match) > 1 {
				idx, _ := strconv.Atoi(match[1])
				if idx > placeholderCount {
					placeholderCount = idx
				}
			}
		}
	case SQLServer:
		// 使用预编译正则精确匹配 @p1, @p2...
		matches := sqlserverPlaceholderRe.FindAllStringSubmatch(querySQL, -1)
		for _, match := range matches {
			if len(match) > 1 {
				idx, _ := strconv.Atoi(match[1])
				if idx > placeholderCount {
					placeholderCount = idx
				}
			}
		}
	case Oracle:
		// 使用预编译正则精确匹配 :1, :2...
		matches := oraclePlaceholderRe.FindAllStringSubmatch(querySQL, -1)
		for _, match := range matches {
			if len(match) > 1 {
				idx, _ := strconv.Atoi(match[1])
				if idx > placeholderCount {
					placeholderCount = idx
				}
			}
		}
	case MySQL, SQLite3:
		// 统计 ? 的数量，需要跳过字符串常量中的问号
		count := 0
		inString := false
		var quoteChar rune
		for i, char := range querySQL {
			if (char == '\'' || char == '"' || char == '`') && (i == 0 || querySQL[i-1] != '\\') {
				if !inString {
					inString = true
					quoteChar = char
				} else if char == quoteChar {
					inString = false
				}
			}
			if char == '?' && !inString {
				count++
			}
		}
		placeholderCount = count
	}

	if placeholderCount == 0 {
		return args
	}

	var cleanedArgs []interface{}
	maxArgs := placeholderCount
	if len(args) < placeholderCount {
		maxArgs = len(args)
	}

	// 处理参数，解引用指针类型
	// Oracle 数据库：将 time.Time 转换为字符串格式
	for i := 0; i < maxArgs; i++ {
		arg := args[i]
		if arg != nil {
			// 使用反射检查是否为指针类型
			v := reflect.ValueOf(arg)
			if v.Kind() == reflect.Ptr {
				if v.IsNil() {
					cleanedArgs = append(cleanedArgs, nil)
				} else {
					// 解引用指针，获取实际值
					actualValue := v.Elem().Interface()
					// Oracle: 转换 time.Time 为字符串
					if mgr.config.Driver == Oracle {
						if t, ok := actualValue.(time.Time); ok {
							cleanedArgs = append(cleanedArgs, mgr.formatOracleTime(t))
							continue
						}
					}
					cleanedArgs = append(cleanedArgs, actualValue)
				}
			} else {
				// Oracle: 转换 time.Time 为字符串
				if mgr.config.Driver == Oracle {
					if t, ok := arg.(time.Time); ok {
						cleanedArgs = append(cleanedArgs, mgr.formatOracleTime(t))
						continue
					}
				}
				// 保持原始类型
				cleanedArgs = append(cleanedArgs, arg)
			}
		} else {
			cleanedArgs = append(cleanedArgs, nil)
		}
	}

	return cleanedArgs
}

// logTrace 辅助函数，封装 SQL 日志记录逻辑
func (mgr *dbManager) logTrace(start time.Time, sql string, args []interface{}, err error) {
	duration := time.Since(start)
	cleanArgs := mgr.sanitizeArgs(sql, args)
	// 格式化参数用于日志显示
	displayArgs := formatArgsForLog(cleanArgs)
	if err != nil {
		LogSQLError(mgr.name, sql, displayArgs, duration, err)
	} else {
		LogSQL(mgr.name, sql, displayArgs, duration)
	}
}

// formatArgsForLog 格式化参数用于日志显示
// 将 time.Time 类型转换为字符串格式，便于阅读
func formatArgsForLog(args []interface{}) []interface{} {
	if len(args) == 0 {
		return args
	}

	formatted := make([]interface{}, len(args))
	for i, arg := range args {
		if arg == nil {
			formatted[i] = nil
		} else if t, ok := arg.(time.Time); ok {
			// 将 time.Time 格式化为字符串用于日志显示
			formatted[i] = t.Format("2006-01-02 15:04:05")
		} else {
			formatted[i] = arg
		}
	}
	return formatted
}

// buildColumnQuery 构建查询列信息的 SQL 语句
// 返回 query 和 args,查询表的所有列（包括：数据类型、是否可空、是否主键、列注释、是否自增长）
func (mgr *dbManager) buildColumnQuery(table string) (string, []interface{}) {
	var query string
	var args []interface{}

	switch mgr.config.Driver {
	case MySQL:
		// MySQL: 查询 EXTRA 字段以获取 auto_increment 信息
		query = `SELECT 
			COLUMN_NAME, 
			DATA_TYPE, 
			IS_NULLABLE, 
			COLUMN_COMMENT, 
			COLUMN_KEY,
			EXTRA
		FROM INFORMATION_SCHEMA.COLUMNS 
		WHERE LOWER(TABLE_NAME) = LOWER(?) 
		AND TABLE_SCHEMA = (SELECT DATABASE()) 
		ORDER BY ORDINAL_POSITION`
		args = []interface{}{table}

	case PostgreSQL:
		// PostgreSQL: 查询主键、注释和序列信息（自增）
		query = `SELECT 
			c.column_name, 
			c.data_type, 
			c.is_nullable,
			COALESCE(pgd.description, '') as column_comment,
			CASE WHEN pk.column_name IS NOT NULL THEN 'PRI' ELSE '' END as column_key,
			CASE WHEN c.column_default LIKE 'nextval%' THEN 'auto_increment' ELSE '' END as extra
		FROM information_schema.columns c
		LEFT JOIN pg_catalog.pg_statio_all_tables st ON c.table_schema = st.schemaname AND c.table_name = st.relname
		LEFT JOIN pg_catalog.pg_description pgd ON pgd.objoid = st.relid AND pgd.objsubid = c.ordinal_position
		LEFT JOIN (
			SELECT ku.column_name
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage ku ON tc.constraint_name = ku.constraint_name
			WHERE tc.constraint_type = 'PRIMARY KEY' 
			AND tc.table_schema = current_schema()
			AND LOWER(tc.table_name) = LOWER(?)
		) pk ON c.column_name = pk.column_name
		WHERE c.table_schema = current_schema() 
		AND LOWER(c.table_name) = LOWER(?) 
		ORDER BY c.ordinal_position`
		args = []interface{}{table, table}

	case SQLite3:
		// SQLite: PRAGMA table_info 返回 pk 字段（>0 表示主键）
		// SQLite 的自增通过 INTEGER PRIMARY KEY 自动实现，需要额外查询
		if err := validateIdentifier(table); err != nil {
			return "", nil
		}
		query = "PRAGMA table_info(" + table + ")"
		args = nil

	case SQLServer:
		// SQL Server: 查询主键、注释和 IDENTITY 信息（自增）
		query = `SELECT 
			c.COLUMN_NAME, 
			c.DATA_TYPE, 
			c.IS_NULLABLE,
			COALESCE(ep.value, '') as COLUMN_COMMENT,
			CASE WHEN pk.COLUMN_NAME IS NOT NULL THEN 'PRI' ELSE '' END as COLUMN_KEY,
			CASE WHEN COLUMNPROPERTY(OBJECT_ID(c.TABLE_SCHEMA + '.' + c.TABLE_NAME), c.COLUMN_NAME, 'IsIdentity') = 1 
				THEN 'auto_increment' ELSE '' END as EXTRA
		FROM INFORMATION_SCHEMA.COLUMNS c
		LEFT JOIN sys.extended_properties ep ON ep.major_id = OBJECT_ID(c.TABLE_SCHEMA + '.' + c.TABLE_NAME)
			AND ep.minor_id = COLUMNPROPERTY(OBJECT_ID(c.TABLE_SCHEMA + '.' + c.TABLE_NAME), c.COLUMN_NAME, 'ColumnId')
			AND ep.name = 'MS_Description'
		LEFT JOIN (
			SELECT ku.COLUMN_NAME
			FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc
			JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE ku ON tc.CONSTRAINT_NAME = ku.CONSTRAINT_NAME
			WHERE tc.CONSTRAINT_TYPE = 'PRIMARY KEY' 
			AND LOWER(tc.TABLE_NAME) = LOWER(?)
		) pk ON c.COLUMN_NAME = pk.COLUMN_NAME
		WHERE LOWER(c.TABLE_NAME) = LOWER(?) 
		ORDER BY c.ORDINAL_POSITION`
		args = []interface{}{table, table}

	case Oracle:
		// Oracle: 查询主键、注释（兼容 Oracle 11g，不查询 IDENTITY_COLUMN）
		query = `SELECT 
			c.COLUMN_NAME, 
			c.DATA_TYPE, 
			c.NULLABLE,
			COALESCE(cc.COMMENTS, '') as COLUMN_COMMENT,
			CASE WHEN pk.COLUMN_NAME IS NOT NULL THEN 'PRI' ELSE '' END as COLUMN_KEY,
			'' as EXTRA
		FROM USER_TAB_COLUMNS c
		LEFT JOIN USER_COL_COMMENTS cc ON c.TABLE_NAME = cc.TABLE_NAME AND c.COLUMN_NAME = cc.COLUMN_NAME
		LEFT JOIN (
			SELECT cols.COLUMN_NAME
			FROM USER_CONSTRAINTS cons
			JOIN USER_CONS_COLUMNS cols ON cons.CONSTRAINT_NAME = cols.CONSTRAINT_NAME
			WHERE cons.CONSTRAINT_TYPE = 'P' 
			AND cons.TABLE_NAME = ?
		) pk ON c.COLUMN_NAME = pk.COLUMN_NAME
		WHERE c.TABLE_NAME = ? 
		ORDER BY c.COLUMN_ID`
		args = []interface{}{strings.ToUpper(table), strings.ToUpper(table)}
	}

	return query, args
}

// checkTableColumn 检查表中是否存在指定字段
func (mgr *dbManager) checkTableColumn(table, column string) bool {
	if mgr.db == nil {
		return false
	}

	// 查询表的所有列,然后在代码中进行不区分大小写的比较
	query, args := mgr.buildColumnQuery(table)
	if query == "" {
		return false
	}

	// 统一使用 mgr.query() 方法查询所有列
	records, err := mgr.query(mgr.db, query, args...)
	if err != nil {
		return false
	}

	// 遍历所有列,查找匹配的列名(不区分大小写)
	for _, record := range records {
		var columnName interface{}

		// 根据不同数据库获取列名字段
		switch mgr.config.Driver {
		case SQLite3:
			// SQLite PRAGMA table_info 返回的列名字段是 "name"
			columnName = record.Get("name")
		case PostgreSQL:
			columnName = record.Get("column_name")
		case MySQL, SQLServer, Oracle:
			columnName = record.Get("COLUMN_NAME")
		}

		if columnName != nil {
			if columnNameStr, ok := columnName.(string); ok {
				// 不区分大小写比较
				if strings.EqualFold(columnNameStr, column) {
					return true
				}
			}
		}
	}
	return false
}

// PoolStats represents database connection pool statistics
type PoolStats struct {
	// Database name
	DBName string `json:"db_name"`
	// Driver type
	Driver string `json:"driver"`
	// Maximum number of open connections (configured)
	MaxOpenConnections int `json:"max_open_connections"`
	// Current number of open connections (in use + idle)
	OpenConnections int `json:"open_connections"`
	// Number of connections currently in use
	InUse int `json:"in_use"`
	// Number of idle connections
	Idle int `json:"idle"`
	// Total number of connections waited for
	WaitCount int64 `json:"wait_count"`
	// Total time blocked waiting for a new connection
	WaitDuration time.Duration `json:"wait_duration"`
	// Total number of connections closed due to MaxIdleTime
	MaxIdleClosed int64 `json:"max_idle_closed"`
	// Total number of connections closed due to MaxLifetime
	MaxLifetimeClosed int64 `json:"max_lifetime_closed"`
}

// PoolStats returns the connection pool statistics for the DB instance
func (db *DB) PoolStats() *PoolStats {
	if db.lastErr != nil || db.dbMgr == nil || db.dbMgr.db == nil {
		return nil
	}
	return db.dbMgr.poolStats()
}

// poolStats returns the connection pool statistics
func (mgr *dbManager) poolStats() *PoolStats {
	if mgr == nil || mgr.db == nil {
		return nil
	}

	stats := mgr.db.Stats()
	return &PoolStats{
		DBName:             mgr.name,
		Driver:             string(mgr.config.Driver),
		MaxOpenConnections: stats.MaxOpenConnections,
		OpenConnections:    stats.OpenConnections,
		InUse:              stats.InUse,
		Idle:               stats.Idle,
		WaitCount:          stats.WaitCount,
		WaitDuration:       stats.WaitDuration,
		MaxIdleClosed:      stats.MaxIdleClosed,
		MaxLifetimeClosed:  stats.MaxLifetimeClosed,
	}
}

// GetPoolStats returns the connection pool statistics for the default database
func GetPoolStats() *PoolStats {
	db, err := defaultDB()
	if err != nil {
		return nil
	}
	return db.PoolStats()
}

// GetPoolStatsDB returns the connection pool statistics for a specific database
func GetPoolStatsDB(dbname string) *PoolStats {
	return Use(dbname).PoolStats()
}

// AllPoolStats returns the connection pool statistics for all registered databases
func AllPoolStats() map[string]*PoolStats {
	result := make(map[string]*PoolStats)

	if multiMgr == nil {
		return result
	}

	multiMgr.mu.RLock()
	defer multiMgr.mu.RUnlock()

	for name, mgr := range multiMgr.databases {
		if mgr != nil && mgr.db != nil {
			result[name] = mgr.poolStats()
		}
	}

	return result
}

// PoolStatsMap returns the connection pool statistics as a map (for JSON serialization)
func (ps *PoolStats) ToMap() map[string]interface{} {
	if ps == nil {
		return nil
	}
	return map[string]interface{}{
		"db_name":              ps.DBName,
		"driver":               ps.Driver,
		"max_open_connections": ps.MaxOpenConnections,
		"open_connections":     ps.OpenConnections,
		"in_use":               ps.InUse,
		"idle":                 ps.Idle,
		"wait_count":           ps.WaitCount,
		"wait_duration_ms":     ps.WaitDuration.Milliseconds(),
		"max_idle_closed":      ps.MaxIdleClosed,
		"max_lifetime_closed":  ps.MaxLifetimeClosed,
	}
}

// String returns a human-readable string representation of the pool stats
func (ps *PoolStats) String() string {
	if ps == nil {
		return "PoolStats: nil"
	}
	return fmt.Sprintf(
		"PoolStats[%s/%s]: Open=%d (InUse=%d, Idle=%d), MaxOpen=%d, WaitCount=%d, WaitDuration=%v",
		ps.DBName, ps.Driver,
		ps.OpenConnections, ps.InUse, ps.Idle,
		ps.MaxOpenConnections, ps.WaitCount, ps.WaitDuration,
	)
}

// PrometheusMetrics returns Prometheus-compatible metrics string
func (ps *PoolStats) PrometheusMetrics() string {
	if ps == nil {
		return ""
	}

	dbLabel := fmt.Sprintf(`db="%s",driver="%s"`, ps.DBName, ps.Driver)

	return fmt.Sprintf(`# HELP eorm_pool_max_open_connections Maximum number of open connections to the database.
# TYPE eorm_pool_max_open_connections gauge
eorm_pool_max_open_connections{%s} %d

# HELP eorm_pool_open_connections The number of established connections both in use and idle.
# TYPE eorm_pool_open_connections gauge
eorm_pool_open_connections{%s} %d

# HELP eorm_pool_in_use The number of connections currently in use.
# TYPE eorm_pool_in_use gauge
eorm_pool_in_use{%s} %d

# HELP eorm_pool_idle The number of idle connections.
# TYPE eorm_pool_idle gauge
eorm_pool_idle{%s} %d

# HELP eorm_pool_wait_count_total The total number of connections waited for.
# TYPE eorm_pool_wait_count_total counter
eorm_pool_wait_count_total{%s} %d

# HELP eorm_pool_wait_duration_seconds_total The total time blocked waiting for a new connection.
# TYPE eorm_pool_wait_duration_seconds_total counter
eorm_pool_wait_duration_seconds_total{%s} %f

# HELP eorm_pool_max_idle_closed_total The total number of connections closed due to SetMaxIdleConns.
# TYPE eorm_pool_max_idle_closed_total counter
eorm_pool_max_idle_closed_total{%s} %d

# HELP eorm_pool_max_lifetime_closed_total The total number of connections closed due to SetConnMaxLifetime.
# TYPE eorm_pool_max_lifetime_closed_total counter
eorm_pool_max_lifetime_closed_total{%s} %d
`,
		dbLabel, ps.MaxOpenConnections,
		dbLabel, ps.OpenConnections,
		dbLabel, ps.InUse,
		dbLabel, ps.Idle,
		dbLabel, ps.WaitCount,
		dbLabel, ps.WaitDuration.Seconds(),
		dbLabel, ps.MaxIdleClosed,
		dbLabel, ps.MaxLifetimeClosed,
	)
}

// AllPrometheusMetrics returns Prometheus metrics for all databases
func AllPrometheusMetrics() string {
	allStats := AllPoolStats()
	var result strings.Builder

	for _, stats := range allStats {
		result.WriteString(stats.PrometheusMetrics())
		result.WriteString("\n")
	}

	return result.String()
}

// startConnectionMonitoring 启动连接监控
func (mgr *dbManager) startConnectionMonitoring() error {
	monitor := &ConnectionMonitor{
		pinger:         mgr, // dbManager 实现了 DBPinger 接口
		dbName:         mgr.name,
		normalInterval: mgr.config.MonitorNormalInterval,
		errorInterval:  mgr.config.MonitorErrorInterval,
		stopCh:         make(chan struct{}),
		lastHealthy:    true, // 假设初始状态为健康
	}

	monitorsMu.Lock()
	monitors[mgr.name] = monitor
	monitorsMu.Unlock()

	// 启动监控 goroutine
	go monitor.run()
	return nil
}

// joinStrings joins strings with commas
func joinStrings(strs []string) string {
	return strings.Join(strs, ", ")
}

// GetTableColumns 获取指定表的所有列信息
// 返回列名、字段类型、长度、是否可空、是否主键、备注信息等
func (db *DB) GetTableColumns(table string) ([]ColumnInfo, error) {
	if db.lastErr != nil {
		return nil, db.lastErr
	}
	return db.dbMgr.getTableColumns(table)
}

// GetAllTables 获取数据库中所有表名
// 根据不同的数据库类型使用相应的查询语句
func (db *DB) GetAllTables() ([]string, error) {
	if db.lastErr != nil {
		return nil, db.lastErr
	}
	return db.dbMgr.getAllTables()
}

// formatOracleTime 将 time.Time 格式化为 Oracle 可识别的字符串
// 使用标准 ISO 8601 格式: YYYY-MM-DD HH24:MI:SS
//
// 时区注意事项:
// - 格式化时保持 time.Time 的原始时区
// - Oracle TO_DATE 不包含时区信息,会按服务器时区解释
// - 建议: 应用和数据库使用相同时区,或统一使用 UTC
// - 使用 UTC: t.UTC().Format() 或创建时使用 time.UTC
func (mgr *dbManager) formatOracleTime(t time.Time) string {
	// 格式化为 ISO 8601 标准格式: YYYY-MM-DD HH24:MI:SS
	return t.Format("2006-01-02 15:04:05")
}

// wrapOracleDatePlaceholders 为 Oracle 的 time.Time 参数包装 TO_DATE 函数
func (mgr *dbManager) wrapOracleDatePlaceholders(querySQL string, args []interface{}) string {
	if len(args) == 0 {
		return querySQL
	}

	// 找到所有的 ? 占位符位置
	placeholderIndex := 0
	result := strings.Builder{}
	inString := false
	var quoteChar rune

	for i, char := range querySQL {
		// 处理字符串常量
		if (char == '\'' || char == '"' || char == '`') && (i == 0 || querySQL[i-1] != '\\') {
			if !inString {
				inString = true
				quoteChar = char
			} else if char == quoteChar {
				inString = false
			}
			result.WriteRune(char)
			continue
		}

		// 如果在字符串内,直接写入
		if inString {
			result.WriteRune(char)
			continue
		}

		// 处理占位符
		if char == '?' {
			if placeholderIndex < len(args) {
				// 检查对应的参数是否是 time.Time
				if _, ok := args[placeholderIndex].(time.Time); ok {
					// 替换为 TO_DATE(?, 'YYYY-MM-DD HH24:MI:SS')
					result.WriteString("TO_DATE(?, 'YYYY-MM-DD HH24:MI:SS')")
				} else {
					result.WriteRune('?')
				}
				placeholderIndex++
			} else {
				result.WriteRune('?')
			}
		} else {
			result.WriteRune(char)
		}
	}

	return result.String()
}
