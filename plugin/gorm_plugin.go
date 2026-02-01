package plugin

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/zzguang83325/eorm"
	"gorm.io/gorm"
)

var errSkipGorm = errors.New("eorm: skip gorm default callback")

// EormGormPlugin 适配 eorm.Record 的 GORM 插件
type EormGormPlugin struct {
	typeCache sync.Map
}

func (p *EormGormPlugin) Name() string {
	return "eorm:record"
}

// 定义标记常量
const (
	typeRecord          = "record"
	typeRecordPtr       = "record_ptr"
	typeRecordsSlice    = "records_slice"
	typeRecordsPtrSlice = "records_ptr_slice"
	eormSkipQueryKey    = "eorm:skip_query"
	eormSkipCreateKey   = "eorm:skip_create"
	eormSkipUpdateKey   = "eorm:skip_update"
	eormSkipDeleteKey   = "eorm:skip_delete"
	eormDestTypeKey     = "eorm:dest_type"
	eormBatchModeKey    = "eorm:batch_mode"
	eormOriginalSQLKey  = "eorm:original_sql"
	eormTxKey           = "eorm_tx"
	eormWhereCacheKey   = "eorm:where_cache"
)

// whereClauseCache 用于缓存 WHERE 子句构建结果
type whereClauseCache struct {
	sql  string
	args []interface{}
}

// Initialize 初始化插件
func (p *EormGormPlugin) Initialize(db *gorm.DB) error {
	// 拦截 Find 方法
	db.Callback().Query().Before("gorm:query").Register("eorm:before_query", p.beforeQuery)
	db.Callback().Query().After("gorm:query").Register("eorm:after_query", p.afterQuery)
	db.Callback().Query().Before("gorm:find").Register("eorm:before_find", p.beforeFind)
	db.Callback().Query().Before("gorm:first").Register("eorm:before_first", p.beforeFirst)
	db.Callback().Query().Before("gorm:last").Register("eorm:before_last", p.beforeLast)
	db.Callback().Query().Before("gorm:take").Register("eorm:before_take", p.beforeTake)

	// 拦截 Create/Update/Delete 方法
	db.Callback().Create().Before("gorm:create").Register("eorm:before_create", p.create)
	db.Callback().Update().Before("gorm:update").Register("eorm:before_update", p.update)
	db.Callback().Delete().Before("gorm:delete").Register("eorm:before_delete", p.delete)

	// 统一处理跳过逻辑
	p.replaceGormCallbacks(db)

	return nil
}

func (p *EormGormPlugin) replaceGormCallbacks(db *gorm.DB) {
	// 替换 Query
	queryCallback := db.Callback().Query().Get("gorm:query")
	db.Callback().Query().Replace("gorm:query", func(tx *gorm.DB) {
		if _, ok := tx.InstanceGet(eormSkipQueryKey); !ok {
			queryCallback(tx)
		}
	})

	// 替换 Create
	createCallback := db.Callback().Create().Get("gorm:create")
	db.Callback().Create().Replace("gorm:create", func(tx *gorm.DB) {
		if _, ok := tx.InstanceGet(eormSkipCreateKey); !ok {
			createCallback(tx)
		}
	})

	// 替换 Update
	updateCallback := db.Callback().Update().Get("gorm:update")
	db.Callback().Update().Replace("gorm:update", func(tx *gorm.DB) {
		if _, ok := tx.InstanceGet(eormSkipUpdateKey); !ok {
			updateCallback(tx)
		}
	})

	// 替换 Delete
	deleteCallback := db.Callback().Delete().Get("gorm:delete")
	db.Callback().Delete().Replace("gorm:delete", func(tx *gorm.DB) {
		if _, ok := tx.InstanceGet(eormSkipDeleteKey); !ok {
			deleteCallback(tx)
		}
	})
}

// detectDestType 检测目标类型是否为 eorm.Record 相关类型
func (p *EormGormPlugin) detectDestType(dest interface{}) string {
	if dest == nil {
		return ""
	}

	// 1. 尝试从缓存中获取
	rt := reflect.TypeOf(dest)
	if cached, ok := p.typeCache.Load(rt); ok {
		return cached.(string)
	}

	// 2. 快速路径：优先使用类型断言
	var result string
	switch dest.(type) {
	case *eorm.Record:
		result = typeRecord
	case **eorm.Record:
		result = typeRecordPtr
	case *[]eorm.Record:
		result = typeRecordsSlice
	case *[]*eorm.Record:
		result = typeRecordsPtrSlice
	default:
		// 3. 慢速路径：只有在类型断言失败时才使用反射（处理基于 Record 的自定义类型）
		result = p.detectDestTypeByReflect(dest)
	}

	// 4. 存入缓存
	if result != "" {
		p.typeCache.Store(rt, result)
	}

	return result
}

// detectDestTypeByReflect 使用反射检测自定义类型
func (p *EormGormPlugin) detectDestTypeByReflect(dest interface{}) string {
	rv := reflect.ValueOf(dest)
	if rv.Kind() != reflect.Ptr {
		return ""
	}

	rt := rv.Type()
	// 获取一级指针指向的类型
	elemType := rt.Elem()

	// 1. 处理结构体类型 (Record 或其别名)
	if elemType.Kind() == reflect.Struct {
		if p.isExactlyRecordType(elemType) {
			return typeRecord
		}
	}

	// 2. 处理指向指针的指针 (**Record)
	if elemType.Kind() == reflect.Ptr {
		innerElem := elemType.Elem()
		if innerElem.Kind() == reflect.Struct && p.isExactlyRecordType(innerElem) {
			return typeRecordPtr
		}
	}

	// 3. 处理切片类型 ([]Record 或 []*Record)
	if elemType.Kind() == reflect.Slice {
		sliceElem := elemType.Elem()
		if sliceElem.Kind() == reflect.Struct {
			if p.isExactlyRecordType(sliceElem) {
				return typeRecordsSlice
			}
		} else if sliceElem.Kind() == reflect.Ptr {
			ptrElem := sliceElem.Elem()
			if ptrElem.Kind() == reflect.Struct && p.isExactlyRecordType(ptrElem) {
				return typeRecordsPtrSlice
			}
		}
	}

	return ""
}

// isExactlyRecordType 检查是否为 eorm.Record 类型
func (p *EormGormPlugin) isExactlyRecordType(t reflect.Type) bool {
	// 更严格的检查：只接受完全相同的类型，不接受任何隐式转换
	return t == reflect.TypeOf(eorm.Record{})
}

// withTransactionContext 事务上下文处理
func (p *EormGormPlugin) withTransactionContext(db *gorm.DB, fn func(*gorm.DB) error) error {
	// 如果已经在事务中，直接使用当前 db
	// 如果不在事务中，直接执行，不开启额外事务，由 GORM 外部控制或直接执行 SQL
	return fn(db)
}

// withSavepoint 处理嵌套事务的保存点
func (p *EormGormPlugin) withSavepoint(db *gorm.DB, fn func(*gorm.DB) error) error {
	savepoint := fmt.Sprintf("sp_%d", time.Now().UnixNano())

	// 创建保存点
	if err := db.Exec("SAVEPOINT " + savepoint).Error; err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			db.Exec("ROLLBACK TO SAVEPOINT " + savepoint)
			panic(r)
		}
	}()

	err := fn(db)

	if err != nil {
		db.Exec("ROLLBACK TO SAVEPOINT " + savepoint)
	} else {
		db.Exec("RELEASE SAVEPOINT " + savepoint)
	}

	return err
}

// getTransactionFromContext 从上下文中获取事务对象
func (p *EormGormPlugin) getTransactionFromContext(db *gorm.DB) *gorm.DB {
	if db.Statement.Context != nil {
		if tx, ok := db.Statement.Context.Value(eormTxKey).(*gorm.DB); ok {
			if p.isValidTransaction(tx) {
				return tx
			}
		}
	}
	return nil
}

// isReadOnly 检测操作是否为只读
func (p *EormGormPlugin) isReadOnly(stmt *gorm.Statement) bool {
	// 简单的只读检测逻辑：如果是 Query 回调触发的通常是只读
	// 也可以根据具体业务需求扩展
	return false
}

// isInTransaction 检查是否已经在事务中
func (p *EormGormPlugin) isInTransaction(db *gorm.DB) bool {
	if db.Statement.ConnPool == nil {
		return false
	}
	// GORM 在事务中时，ConnPool 通常是 gorm.TxCommitter 类型
	_, ok := db.Statement.ConnPool.(gorm.TxCommitter)
	return ok
}

// isValidTransaction 检查事务对象是否有效
func (p *EormGormPlugin) isValidTransaction(tx *gorm.DB) bool {
	return tx != nil && tx.Error == nil && p.isInTransaction(tx)
}

// handleRecordError 统一错误处理并添加上下文信息
func (p *EormGormPlugin) handleRecordError(db *gorm.DB, operation string, err error) {
	if err == nil {
		return
	}

	// 添加操作上下文信息
	wrappedErr := fmt.Errorf("eorm plugin %s failed: %w", operation, err)

	// 如果有 SQL 语句，也记录下来方便排查
	if db.Statement.SQL.String() != "" {
		wrappedErr = fmt.Errorf("%w (SQL: %s)", wrappedErr, db.Statement.SQL.String())
	}

	db.AddError(wrappedErr)
}

// withRetry 执行带重试逻辑的操作
func (p *EormGormPlugin) withRetry(maxRetries int, fn func() error) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		err := fn()
		if err == nil {
			return nil
		}

		// 检查是否为可重试错误
		if p.shouldRetry(err) {
			lastErr = err
			// 指数退避策略
			time.Sleep(time.Duration(i*100) * time.Millisecond)
			continue
		}
		return err
	}
	return lastErr
}

// shouldRetry 判断错误是否可以重试
func (p *EormGormPlugin) shouldRetry(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	retryableErrors := []string{
		"deadlock",
		"timeout",
		"connection reset",
		"try again",
		"connection refused",
	}

	for _, pattern := range retryableErrors {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	return false
}

// shouldHandleRecordError 更精准的错误检测
func (p *EormGormPlugin) shouldHandleRecordError(db *gorm.DB) bool {
	if db.Error == nil {
		return false
	}

	// 优先使用 GORM 定义的错误类型进行匹配
	switch {
	case errors.Is(db.Error, gorm.ErrModelValueRequired),
		errors.Is(db.Error, gorm.ErrInvalidField):
		return true
	}

	errStr := db.Error.Error()
	errorPatterns := []string{
		"invalid memory address",
		"missing destination",
		"table not set",
		"schema not set",
	}

	for _, pattern := range errorPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	return false
}

// getExecutor 获取执行器
func (p *EormGormPlugin) getExecutor(db *gorm.DB) eorm.SqlExecutor {
	return &gormConnWrapper{
		db: db,
		pool: &sync.Pool{
			New: func() interface{} {
				return &bytes.Buffer{}
			},
		},
	}
}

type gormConnWrapper struct {
	db   *gorm.DB
	pool *sync.Pool
}

func (w *gormConnWrapper) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return w.db.Statement.ConnPool.QueryContext(w.db.Statement.Context, query, args...)
}

func (w *gormConnWrapper) Exec(query string, args ...interface{}) (sql.Result, error) {
	// 如果未来需要对 query 进行预处理，可以使用缓冲池
	buf := w.pool.Get().(*bytes.Buffer)
	buf.Reset()
	defer w.pool.Put(buf)

	return w.db.Statement.ConnPool.ExecContext(w.db.Statement.Context, query, args...)
}

func (w *gormConnWrapper) QueryRow(query string, args ...interface{}) *sql.Row {
	return w.db.Statement.ConnPool.QueryRowContext(w.db.Statement.Context, query, args...)
}

// execWithExecutor 执行带执行器的操作
func (p *EormGormPlugin) execWithExecutor(executor eorm.SqlExecutor, fn func(eorm.SqlExecutor) error) error {
	return fn(executor)
}

// beforeQuery 查询前的处理
func (p *EormGormPlugin) beforeQuery(db *gorm.DB) {
	dest := db.Statement.Dest
	if dest == nil {
		return
	}

	// 检测目标类型
	destType := p.detectDestType(dest)
	if destType == "" {
		return
	}

	// 设置标记
	db.InstanceSet(eormSkipQueryKey, true)
	db.InstanceSet(eormDestTypeKey, destType)
}

// clearStatementCache 清理 Statement 级别的缓存
func (p *EormGormPlugin) clearStatementCache(db *gorm.DB) {
	db.InstanceSet(eormWhereCacheKey, nil)
	db.InstanceSet(eormSkipQueryKey, nil)
	db.InstanceSet(eormDestTypeKey, nil)
}

// afterQuery 查询后的处理
func (p *EormGormPlugin) afterQuery(db *gorm.DB) {
	defer p.clearStatementCache(db)

	// 检查是否需要跳过
	if _, ok := db.InstanceGet(eormSkipQueryKey); !ok {
		return
	}

	// 如果 GORM 因为没有模型定义而报错，清除它
	if p.shouldHandleRecordError(db) {
		db.Error = nil
	}

	// 确保 SQL 已构建
	if db.Statement.SQL.String() == "" {
		db.Statement.Build(db.Statement.BuildClauses...)
	}

	sqlStr := db.Statement.SQL.String()
	table := db.Statement.Table
	if table == "" && db.Statement.Schema != nil {
		table = db.Statement.Schema.Table
	}

	// 1. 如果 SQL 为空或不包含 SELECT，手动构建基础查询
	upperSQL := strings.ToUpper(strings.TrimSpace(sqlStr))
	if sqlStr == "" || !strings.HasPrefix(upperSQL, "SELECT") {
		if table != "" {
			if sqlStr == "" {
				sqlStr = fmt.Sprintf("SELECT * FROM %s", table)
			} else {
				// 处理只有 WHERE/LIMIT 等子句的情况
				if !strings.Contains(upperSQL, "FROM") {
					sqlStr = fmt.Sprintf("SELECT * FROM %s %s", table, sqlStr)
				} else {
					sqlStr = fmt.Sprintf("SELECT * %s", sqlStr)
				}
			}
		}
	}

	// 2. 移除多余的空格并确保占位符与驱动匹配 (PostgreSQL 使用 $1, $2...)
	// GORM 的 Build 过程通常已经处理了占位符转换，但我们要确保万无一失
	db.Statement.SQL.Reset()
	db.Statement.SQL.WriteString(sqlStr)

	// 执行查询
	rows, err := db.Statement.ConnPool.QueryContext(db.Statement.Context, sqlStr, db.Statement.Vars...)
	if err != nil {
		p.handleRecordError(db, "query", err)
		return
	}
	defer rows.Close()

	// 获取目标类型
	destTypeVal, _ := db.InstanceGet(eormDestTypeKey)
	destType := destTypeVal.(string)

	// 根据目标类型处理结果
	switch destType {
	case typeRecordsSlice:
		p.handleRecordsSlice(db, rows)
	case typeRecordsPtrSlice:
		p.handleRecordPtrSlice(db, rows)
	case typeRecordPtr:
		p.handleRecordPtr(db, rows)
	case typeRecord:
		p.handleRecord(db, rows)
	}
}

// 处理 []eorm.Record
func (p *EormGormPlugin) handleRecordsSlice(db *gorm.DB, rows *sql.Rows) {
	records, err := eorm.ScanRecords(rows)
	if err != nil {
		db.AddError(err)
		return
	}

	dest := db.Statement.Dest.(*[]eorm.Record)
	*dest = make([]eorm.Record, len(records))
	for i, r := range records {
		(*dest)[i] = *r
	}
	db.RowsAffected = int64(len(records))
}

// 处理 []*eorm.Record
func (p *EormGormPlugin) handleRecordPtrSlice(db *gorm.DB, rows *sql.Rows) {
	records, err := eorm.ScanRecords(rows)
	if err != nil {
		db.AddError(err)
		return
	}

	dest := db.Statement.Dest.(*[]*eorm.Record)
	*dest = records
	db.RowsAffected = int64(len(records))
}

// 处理 **eorm.Record (用于 First/Last/Take)
func (p *EormGormPlugin) handleRecordPtr(db *gorm.DB, rows *sql.Rows) {
	records, err := eorm.ScanRecords(rows)
	if err != nil {
		db.AddError(err)
		return
	}

	if len(records) > 0 {
		dest := db.Statement.Dest.(**eorm.Record)
		*dest = records[0]
		db.RowsAffected = 1
	} else {
		db.AddError(gorm.ErrRecordNotFound)
	}
}

// 处理 *eorm.Record
func (p *EormGormPlugin) handleRecord(db *gorm.DB, rows *sql.Rows) {
	records, err := eorm.ScanRecords(rows)
	if err != nil {
		db.AddError(err)
		return
	}

	if len(records) > 0 {
		dest := db.Statement.Dest.(*eorm.Record)
		*dest = *records[0]
		db.RowsAffected = 1
	} else {
		db.AddError(gorm.ErrRecordNotFound)
	}
}

// beforeFind 拦截 Find 方法
func (p *EormGormPlugin) beforeFind(db *gorm.DB) {
	if p.detectDestType(db.Statement.Dest) != "" {
		db.InstanceSet(eormSkipQueryKey, true)
	}
}

// beforeFirst 拦截 First 方法
func (p *EormGormPlugin) beforeFirst(db *gorm.DB) {
	if p.detectDestType(db.Statement.Dest) != "" {
		db.InstanceSet(eormSkipQueryKey, true)
		// GORM 的 First() 会自动添加 Limit(1) 和 Order("id")
	}
}

// beforeLast 拦截 Last 方法
func (p *EormGormPlugin) beforeLast(db *gorm.DB) {
	if p.detectDestType(db.Statement.Dest) != "" {
		db.InstanceSet(eormSkipQueryKey, true)
		// GORM 的 Last() 会自动添加 Limit(1) 和 Order("id DESC")
	}
}

// beforeTake 拦截 Take 方法
func (p *EormGormPlugin) beforeTake(db *gorm.DB) {
	if p.detectDestType(db.Statement.Dest) != "" {
		db.InstanceSet(eormSkipQueryKey, true)
		// GORM 的 Take() 会自动添加 Limit(1)
	}
}

// create 拦截 gorm:create
func (p *EormGormPlugin) create(db *gorm.DB) {
	if p.detectDestType(db.Statement.Dest) == "" {
		return
	}

	db.InstanceSet(eormSkipCreateKey, true)

	table := db.Statement.Table
	if table == "" && db.Statement.Schema != nil {
		table = db.Statement.Schema.Table
	}

	if table == "" {
		db.AddError(errors.New("eorm: missing table name for create"))
		return
	}

	dest := db.Statement.Dest
	switch v := dest.(type) {
	case *eorm.Record:
		if err := p.withTransactionContext(db, func(txDB *gorm.DB) error {
			executor := p.getExecutor(txDB)
			return p.execWithExecutor(executor, func(e eorm.SqlExecutor) error {
				_, err := eorm.SaveRecordWithExecutor(e, table, v)
				if err != nil {
					return err
				}
				fmt.Printf("[DEBUG] after Create Record: %s\n", v.ToJson())
				db.RowsAffected = 1
				return nil
			})
		}); err != nil {
			p.handleRecordError(db, "create_single", err)
		} else {
			db.InstanceSet("eorm:handled", true)
		}
	case *[]eorm.Record:
		records := make([]*eorm.Record, len(*v))
		for i := range *v {
			records[i] = &(*v)[i]
		}
		if err := p.withTransactionContext(db, func(txDB *gorm.DB) error {
			executor := p.getExecutor(txDB)
			return p.execWithExecutor(executor, func(e eorm.SqlExecutor) error {
				affected, err := eorm.BatchInsertRecordWithExecutor(e, table, records)
				if err != nil {
					return err
				}
				db.RowsAffected = affected
				return nil
			})
		}); err != nil {
			p.handleRecordError(db, "create_batch_slice", err)
		}
	case *[]*eorm.Record:
		if err := p.withTransactionContext(db, func(txDB *gorm.DB) error {
			executor := p.getExecutor(txDB)
			return p.execWithExecutor(executor, func(e eorm.SqlExecutor) error {
				affected, err := eorm.BatchInsertRecordWithExecutor(e, table, *v)
				if err != nil {
					return err
				}
				db.RowsAffected = affected
				return nil
			})
		}); err != nil {
			p.handleRecordError(db, "create_batch_ptr_slice", err)
		}
	}
}

func (p *EormGormPlugin) afterCreate(db *gorm.DB) {}

func (p *EormGormPlugin) afterUpdate(db *gorm.DB) {}

func (p *EormGormPlugin) afterDelete(db *gorm.DB) {}

// afterOperation 统一的后续处理逻辑
func (p *EormGormPlugin) afterOperation(db *gorm.DB, skipKey string) {
	defer p.clearStatementCache(db)

	if _, ok := db.InstanceGet(skipKey); !ok {
		return
	}

	if p.shouldHandleRecordError(db) {
		db.Error = nil
	}
}

// cloneStatement 深度复制 Statement 副本
func (p *EormGormPlugin) cloneStatement(stmt *gorm.Statement) *gorm.Statement {
	clone := *stmt

	// 深度复制 Vars，防止切片共享导致的并发冲突
	if stmt.Vars != nil {
		clone.Vars = make([]interface{}, len(stmt.Vars))
		copy(clone.Vars, stmt.Vars)
	}

	// 复制 SQL 缓冲区
	clone.SQL = strings.Builder{}
	clone.SQL.WriteString(stmt.SQL.String())

	return &clone
}

// getWhereClause 获取并缓存 WHERE 子句
func (p *EormGormPlugin) getWhereClause(db *gorm.DB) (string, []interface{}) {
	// 1. 检查是否已有缓存的 WHERE
	if cache, ok := db.InstanceGet(eormWhereCacheKey); ok {
		if where, ok := cache.(whereClauseCache); ok {
			return where.sql, where.args
		}
	}

	// 2. 使用副本构建避免并发和状态污染问题
	stmtCopy := p.cloneStatement(db.Statement)

	// 重置副本状态以仅构建 WHERE 部分
	stmtCopy.Vars = make([]interface{}, 0)
	stmtCopy.SQL.Reset()
	stmtCopy.Build("WHERE")

	whereSql := stmtCopy.SQL.String()
	whereArgs := stmtCopy.Vars

	if strings.HasPrefix(strings.ToUpper(whereSql), " WHERE ") {
		whereSql = whereSql[7:]
	}

	// 3. 缓存结果
	result := whereClauseCache{
		sql:  whereSql,
		args: whereArgs,
	}
	db.InstanceSet(eormWhereCacheKey, result)

	return whereSql, whereArgs
}

// update 拦截 gorm:update
func (p *EormGormPlugin) update(db *gorm.DB) {
	destType := p.detectDestType(db.Statement.Dest)
	modelType := p.detectDestType(db.Statement.Model)

	if destType == "" && modelType == "" {
		return
	}

	db.InstanceSet(eormSkipUpdateKey, true)

	table := db.Statement.Table
	if table == "" && db.Statement.Schema != nil {
		table = db.Statement.Schema.Table
	}

	if table == "" {
		db.AddError(errors.New("eorm: missing table name for update"))
		return
	}

	// 获取 WHERE 部分和参数
	whereSql, whereArgs := p.getWhereClause(db)

	dest := db.Statement.Dest
	switch v := dest.(type) {
	case *eorm.Record:
		// 如果有 WHERE 条件，优先使用 eorm.Update
		if whereSql != "" {
			if err := p.withTransactionContext(db, func(txDB *gorm.DB) error {
				executor := p.getExecutor(txDB)
				return p.execWithExecutor(executor, func(e eorm.SqlExecutor) error {
					affected, err := eorm.UpdateWithExecutor(e, table, v, whereSql, whereArgs...)
					if err != nil {
						return err
					}
					db.RowsAffected = affected
					return nil
				})
			}); err != nil {
				db.AddError(err)
			}
		} else {
			// 否则使用 UpdateRecord (基于主键)
			if err := p.withTransactionContext(db, func(txDB *gorm.DB) error {
				executor := p.getExecutor(txDB)
				return p.execWithExecutor(executor, func(e eorm.SqlExecutor) error {
					affected, err := eorm.UpdateRecordWithExecutor(e, table, v)
					if err != nil {
						return err
					}
					db.RowsAffected = affected
					return nil
				})
			}); err != nil {
				db.AddError(err)
			}
		}
		return
	case *[]eorm.Record:
		records := make([]*eorm.Record, len(*v))
		for i := range *v {
			records[i] = &(*v)[i]
		}
		if err := p.withTransactionContext(db, func(txDB *gorm.DB) error {
			executor := p.getExecutor(txDB)
			return p.execWithExecutor(executor, func(e eorm.SqlExecutor) error {
				affected, err := eorm.BatchUpdateRecordWithExecutor(e, table, records)
				if err != nil {
					return err
				}
				db.RowsAffected = affected
				return nil
			})
		}); err != nil {
			db.AddError(err)
		}
		return
	case *[]*eorm.Record:
		if err := p.withTransactionContext(db, func(txDB *gorm.DB) error {
			executor := p.getExecutor(txDB)
			return p.execWithExecutor(executor, func(e eorm.SqlExecutor) error {
				affected, err := eorm.BatchUpdateRecordWithExecutor(e, table, *v)
				if err != nil {
					return err
				}
				db.RowsAffected = affected
				return nil
			})
		}); err != nil {
			db.AddError(err)
		}
		return
	case map[string]interface{}:
		// 处理 db.Table("xxx").Where(...).Updates(map[string]interface{}{...})
		// 将 map 转换为 Record
		record := eorm.NewRecord()
		for k, val := range v {
			record.Set(k, val)
		}

		// 如果 Model 是 Record，合并主键到 Record 中（如果 record 没设置的话）
		if modelRecord, ok := db.Statement.Model.(*eorm.Record); ok {
			// 尝试从 Model 中获取 ID，假设主键名为 id
			if id := modelRecord.Int64("id"); id > 0 && record.Int64("id") == 0 {
				record.Set("id", id)
			}
		}

		// 执行更新
		if err := p.withTransactionContext(db, func(txDB *gorm.DB) error {
			executor := p.getExecutor(txDB)
			return p.execWithExecutor(executor, func(e eorm.SqlExecutor) error {
				affected, err := eorm.UpdateWithExecutor(e, table, record, whereSql, whereArgs...)
				if err != nil {
					return err
				}
				db.RowsAffected = affected
				return nil
			})
		}); err != nil {
			db.AddError(err)
		}
		return
	}
}

// delete 拦截 gorm:delete
func (p *EormGormPlugin) delete(db *gorm.DB) {
	destType := p.detectDestType(db.Statement.Dest)
	modelType := p.detectDestType(db.Statement.Model)

	if destType == "" && modelType == "" {
		return
	}

	db.InstanceSet(eormSkipDeleteKey, true)

	table := db.Statement.Table
	if table == "" && db.Statement.Schema != nil {
		table = db.Statement.Schema.Table
	}

	if table == "" {
		db.AddError(errors.New("eorm: missing table name for delete"))
		return
	}

	// 获取 WHERE 部分和参数
	whereSql, whereArgs := p.getWhereClause(db)

	dest := db.Statement.Dest
	switch v := dest.(type) {
	case *eorm.Record:
		// 如果有 WHERE 条件，优先使用 eorm.Delete
		if whereSql != "" {
			if err := p.withTransactionContext(db, func(txDB *gorm.DB) error {
				executor := p.getExecutor(txDB)
				return p.execWithExecutor(executor, func(e eorm.SqlExecutor) error {
					affected, err := eorm.DeleteWithExecutor(e, table, whereSql, whereArgs...)
					if err != nil {
						return err
					}
					db.RowsAffected = affected
					return nil
				})
			}); err != nil {
				db.AddError(err)
			}
		} else {
			// 否则使用 DeleteRecord (基于主键)
			if err := p.withTransactionContext(db, func(txDB *gorm.DB) error {
				executor := p.getExecutor(txDB)
				return p.execWithExecutor(executor, func(e eorm.SqlExecutor) error {
					affected, err := eorm.DeleteRecordWithExecutor(e, table, v)
					if err != nil {
						return err
					}
					db.RowsAffected = affected
					return nil
				})
			}); err != nil {
				db.AddError(err)
			}
		}
		return
	case *[]eorm.Record:
		records := make([]*eorm.Record, len(*v))
		for i := range *v {
			records[i] = &(*v)[i]
		}
		if err := p.withTransactionContext(db, func(txDB *gorm.DB) error {
			executor := p.getExecutor(txDB)
			return p.execWithExecutor(executor, func(e eorm.SqlExecutor) error {
				affected, err := eorm.BatchDeleteRecordWithExecutor(e, table, records)
				if err != nil {
					return err
				}
				db.RowsAffected = affected
				return nil
			})
		}); err != nil {
			db.AddError(err)
		}
		return
	case *[]*eorm.Record:
		if err := p.withTransactionContext(db, func(txDB *gorm.DB) error {
			executor := p.getExecutor(txDB)
			return p.execWithExecutor(executor, func(e eorm.SqlExecutor) error {
				affected, err := eorm.BatchDeleteRecordWithExecutor(e, table, *v)
				if err != nil {
					return err
				}
				db.RowsAffected = affected
				return nil
			})
		}); err != nil {
			db.AddError(err)
		}
		return
	}

	// 如果 dest 是 nil 或者其他类型，但有 WHERE 条件，尝试直接执行删除
	if whereSql != "" {
		if err := p.withTransactionContext(db, func(txDB *gorm.DB) error {
			executor := p.getExecutor(txDB)
			return p.execWithExecutor(executor, func(e eorm.SqlExecutor) error {
				affected, err := eorm.DeleteWithExecutor(e, table, whereSql, whereArgs...)
				if err != nil {
					return err
				}
				db.RowsAffected = affected
				return nil
			})
		}); err != nil {
			db.AddError(err)
		}
		return
	}
}
