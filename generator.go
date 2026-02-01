package eorm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// IDbModel represents a database model that provides its table name
type IDbModel interface {
	TableName() string
	DatabaseName() string
}

// ColumnInfo represents column metadata
type ColumnInfo struct {
	Name             string // 列名
	Type             string // 数据类型
	Nullable         bool   // 是否可空
	IsPK             bool   // 是否主键
	Comment          string // 列注释
	IsAutoIncr       bool   // 是否自增长
	IsIdentityAlways bool   // PostgreSQL: 是否为 GENERATED ALWAYS AS IDENTITY (新增)
}

// GenerateDbModel generates a Go struct for the specified table and saves it to a file
func GenerateDbModel(tablename, outPath, structName string) error {
	db, err := defaultDB()
	if err != nil {
		return err
	}
	return db.GenerateDbModel(tablename, outPath, structName)
}

// GenerateDbModel generates a Go struct for the specified table and saves it to a file
func (db *DB) GenerateDbModel(tablename, outPath, structName string) error {
	if db.lastErr != nil {
		return db.lastErr
	}

	// 安全校验：验证表名合法性
	if err := validateIdentifier(tablename); err != nil {
		return err
	}

	columns, err := db.dbMgr.getTableColumns(tablename)
	if err != nil {
		return err
	}

	if len(columns) == 0 {
		return fmt.Errorf("no columns found for table '%s'. please check if the table exists and you have access permissions", tablename)
	}

	// 1. Handle path and package name
	var pkgName string
	var finalPath string

	if outPath == "" {
		// If no path provided, generate models package in current directory
		pkgName = "models"
		// finalPath = filepath.Join("models", strings.ToLower(tablename)+".go")
		safeFileBase := strings.ReplaceAll(strings.ToLower(tablename), ".", "_")
		finalPath = filepath.Join("models", safeFileBase+".go")
	} else {
		// Check if outPath is a directory or file
		if strings.HasSuffix(outPath, ".go") {
			// Is file path
			finalPath = outPath
			dir := filepath.Dir(outPath)
			if dir == "." || dir == "/" {
				pkgName = "models"
			} else {
				pkgName = filepath.Base(dir)
			}
		} else {
			// Is directory path
			pkgName = filepath.Base(outPath)
			if pkgName == "." || pkgName == "/" {
				pkgName = "models"
			}
			// finalPath = filepath.Join(outPath, strings.ToLower(tablename)+".go")
			safeFileBase := strings.ReplaceAll(strings.ToLower(tablename), ".", "_")
			finalPath = filepath.Join(outPath, safeFileBase+".go")
		}
	}

	// 2. Determine struct name (if structName is empty, generate from table name)
	finalStructName := structName
	if finalStructName == "" {
		camelBase := strings.ReplaceAll(tablename, ".", "_")
		finalStructName = SnakeToCamel(camelBase)
	}

	// 3. Build code content
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("package %s\n\n", pkgName))

	// 检查是否需要 time 导入
	hasTime := false
	for _, col := range columns {
		if strings.Contains(dbTypeToGoType(col.Type, col.Nullable, col.IsPK), "time.Time") {
			hasTime = true
			break
		}
	}
	// Cache method always needs time.Duration, so always import time package
	hasTime = true

	// Generate import
	sb.WriteString("import (\n")
	if hasTime {
		sb.WriteString("\t\"time\"\n")
	}
	if pkgName != "eorm" {
		sb.WriteString("\t\"github.com/zzguang83325/eorm\"\n")
	}
	sb.WriteString(")\n\n")

	sb.WriteString(fmt.Sprintf("// %s represents the %s table\n", finalStructName, tablename))
	sb.WriteString(fmt.Sprintf("type %s struct {\n", finalStructName))
	// 嵌入 ModelCache 以支持缓存功能，添加 column:"-" 标签防止映射到数据库列
	sb.WriteString("\teorm.ModelCache `column:\"-\"`\n")

	for _, col := range columns {
		fieldName := SnakeToCamel(col.Name)
		goType := dbTypeToGoType(col.Type, col.Nullable, col.IsPK)

		// 跳过空字段名
		if fieldName == "" {
			continue
		}
		if goType == "" {
			goType = "interface{}"
		}

		tag := fmt.Sprintf("`column:\"%s\" json:\"%s\"`", col.Name, strings.ToLower(col.Name))

		line := fmt.Sprintf("\t%s %s %s", fieldName, goType, tag)
		if col.Comment != "" {
			line += " // " + col.Comment
		}
		sb.WriteString(line + "\n")
	}

	sb.WriteString("}\n\n")

	// Add TableName method
	sb.WriteString(fmt.Sprintf("// TableName returns the table name for %s struct\n", finalStructName))
	sb.WriteString(fmt.Sprintf("func (m *%s) TableName() string {\n", finalStructName))
	sb.WriteString(fmt.Sprintf("\treturn \"%s\"\n", tablename))
	sb.WriteString("}\n\n")

	// Add DatabaseName method
	sb.WriteString(fmt.Sprintf("// DatabaseName returns the database name for %s struct\n", finalStructName))
	sb.WriteString(fmt.Sprintf("func (m *%s) DatabaseName() string {\n", finalStructName))
	sb.WriteString(fmt.Sprintf("\treturn \"%s\"\n", db.dbMgr.name))
	sb.WriteString("}\n\n")

	// Add Cache method (returns self type to support method chaining)
	sb.WriteString(fmt.Sprintf("// Cache sets the cache name and TTL for the next query\n"))
	sb.WriteString(fmt.Sprintf("func (m *%s) Cache(cacheRepositoryName string, ttl ...time.Duration) *%s {\n", finalStructName, finalStructName))
	sb.WriteString("\tm.SetCache(cacheRepositoryName, ttl...)\n")
	sb.WriteString("\treturn m\n")
	sb.WriteString("}\n\n")

	// Add WithCountCache method (returns self type to support method chaining)
	sb.WriteString(fmt.Sprintf("// WithCountCache 设置分页计数缓存时间，支持链式调用\n"))
	sb.WriteString(fmt.Sprintf("func (m *%s) WithCountCache(ttl time.Duration) *%s {\n", finalStructName, finalStructName))
	sb.WriteString("\tm.ModelCache.WithCountCache(ttl)\n")
	sb.WriteString("\treturn m\n")
	sb.WriteString("}\n\n")

	// Add ToJson method
	sb.WriteString(fmt.Sprintf("// ToJson converts %s to a JSON string\n", finalStructName))
	sb.WriteString(fmt.Sprintf("func (m *%s) ToJson() string {\n", finalStructName))
	sb.WriteString("\treturn eorm.ToJson(m)\n")
	sb.WriteString("}\n\n")

	// Add ActiveRecord member methods (Save, Insert, Update, Delete)
	sb.WriteString(fmt.Sprintf("// Save saves the %s record (insert or update)\n", finalStructName))
	sb.WriteString(fmt.Sprintf("func (m *%s) Save() (int64, error) {\n", finalStructName))
	sb.WriteString("\treturn eorm.SaveDbModel(m)\n")
	sb.WriteString("}\n\n")

	sb.WriteString(fmt.Sprintf("// Insert inserts the %s record\n", finalStructName))
	sb.WriteString(fmt.Sprintf("func (m *%s) Insert() (int64, error) {\n", finalStructName))
	sb.WriteString("\treturn eorm.InsertDbModel(m)\n")
	sb.WriteString("}\n\n")

	sb.WriteString(fmt.Sprintf("// Update updates the %s record based on its primary key\n", finalStructName))
	sb.WriteString(fmt.Sprintf("func (m *%s) Update() (int64, error) {\n", finalStructName))
	sb.WriteString("\treturn eorm.UpdateDbModel(m)\n")
	sb.WriteString("}\n\n")

	sb.WriteString(fmt.Sprintf("// Delete deletes the %s record based on its primary key\n", finalStructName))
	sb.WriteString(fmt.Sprintf("func (m *%s) Delete() (int64, error) {\n", finalStructName))
	sb.WriteString("\treturn eorm.DeleteDbModel(m)\n")
	sb.WriteString("}\n\n")

	// Add ForceDelete method for soft delete support
	sb.WriteString(fmt.Sprintf("// ForceDelete performs a physical delete, bypassing soft delete\n"))
	sb.WriteString(fmt.Sprintf("func (m *%s) ForceDelete() (int64, error) {\n", finalStructName))
	sb.WriteString("\treturn eorm.ForceDeleteModel(m)\n")
	sb.WriteString("}\n\n")

	// Add Restore method for soft delete support
	sb.WriteString(fmt.Sprintf("// Restore restores a soft-deleted record\n"))
	sb.WriteString(fmt.Sprintf("func (m *%s) Restore() (int64, error) {\n", finalStructName))
	sb.WriteString("\treturn eorm.RestoreModel(m)\n")
	sb.WriteString("}\n\n")

	// Use generic function to simplify FindFirst
	sb.WriteString(fmt.Sprintf("// FindFirst finds the first %s record based on conditions\n", finalStructName))
	sb.WriteString(fmt.Sprintf("func (m *%s) FindFirst(whereSql string, args ...interface{}) (*%s, error) {\n", finalStructName, finalStructName))
	sb.WriteString(fmt.Sprintf("\tresult := &%s{}\n", finalStructName))
	sb.WriteString("\treturn eorm.FindFirstModel(result, m.GetCache(), whereSql, args...)\n")
	sb.WriteString("}\n\n")

	// Use generic function to simplify Find
	sb.WriteString(fmt.Sprintf("// Find finds %s records based on conditions\n", finalStructName))
	sb.WriteString(fmt.Sprintf("func (m *%s) Find(whereSql string, orderBySql string, args ...interface{}) ([]*%s, error) {\n", finalStructName, finalStructName))
	sb.WriteString(fmt.Sprintf("\treturn eorm.FindModel[*%s](m, m.GetCache(), whereSql, orderBySql, args...)\n", finalStructName))
	sb.WriteString("}\n\n")

	// Add FindWithTrashed for soft delete support
	sb.WriteString(fmt.Sprintf("// FindWithTrashed finds %s records including soft-deleted ones\n", finalStructName))
	sb.WriteString(fmt.Sprintf("func (m *%s) FindWithTrashed(whereSql string, orderBySql string, args ...interface{}) ([]*%s, error) {\n", finalStructName, finalStructName))
	sb.WriteString(fmt.Sprintf("\treturn eorm.FindModelWithTrashed[*%s](m, m.GetCache(), whereSql, orderBySql, args...)\n", finalStructName))
	sb.WriteString("}\n\n")

	// Add FindOnlyTrashed for soft delete support
	sb.WriteString(fmt.Sprintf("// FindOnlyTrashed finds only soft-deleted %s records\n", finalStructName))
	sb.WriteString(fmt.Sprintf("func (m *%s) FindOnlyTrashed(whereSql string, orderBySql string, args ...interface{}) ([]*%s, error) {\n", finalStructName, finalStructName))
	sb.WriteString(fmt.Sprintf("\treturn eorm.FindModelOnlyTrashed[*%s](m, m.GetCache(), whereSql, orderBySql, args...)\n", finalStructName))
	sb.WriteString("}\n\n")

	// Use generic function to simplify PaginateBuilder (traditional builder-style pagination)
	sb.WriteString(fmt.Sprintf("// PaginateBuilder paginates %s records based on conditions (traditional method)\n", finalStructName))
	sb.WriteString(fmt.Sprintf("func (m *%s) PaginateBuilder(page int, pageSize int, whereSql string, orderBy string, args ...interface{}) (*eorm.Page[*%s], error) {\n", finalStructName, finalStructName))
	sb.WriteString(fmt.Sprintf("\treturn eorm.PaginateModel[*%s](m, m.GetCache(), page, pageSize, whereSql, orderBy, args...)\n", finalStructName))
	sb.WriteString("}\n\n")

	// Add Paginate method (full SQL pagination - recommended)
	sb.WriteString(fmt.Sprintf("// Paginate paginates %s records using complete SQL statement (recommended)\n", finalStructName))
	sb.WriteString(fmt.Sprintf("// Uses complete SQL statement for pagination query, automatically parses SQL and generates corresponding pagination statements based on database type\n"))
	sb.WriteString(fmt.Sprintf("func (m *%s) Paginate(page int, pageSize int, fullSQL string, args ...interface{}) (*eorm.Page[*%s], error) {\n", finalStructName, finalStructName))
	sb.WriteString(fmt.Sprintf("\treturn eorm.PaginateModel_FullSql[*%s](m, m.GetCache(), page, pageSize, fullSQL, args...)\n", finalStructName))
	sb.WriteString("}\n\n")

	// 4. Write to file
	// Ensure directory exists
	dir := filepath.Dir(finalPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	if err := os.WriteFile(finalPath, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	return nil
}

// getTableColumns fetches column information for a table
// 返回列信息：数据类型、是否可空、是否主键、列注释、是否自增长
func (mgr *dbManager) getTableColumns(table string) ([]ColumnInfo, error) {
	// 安全校验：验证表名合法性
	if err := validateIdentifier(table); err != nil {
		return nil, err
	}

	// 1. 检查缓存（读锁）
	mgr.mu.RLock()
	if columns, ok := mgr.columnCache[table]; ok {
		mgr.mu.RUnlock()
		return columns, nil
	}
	mgr.mu.RUnlock()

	// 2. 缓存未命中，查询数据库
	var columns []ColumnInfo
	driver := mgr.config.Driver

	switch driver {
	case MySQL:
		// 使用公共查询构建函数
		query, args := mgr.buildColumnQuery(table)
		db, err := mgr.getDB()
		if err != nil {
			return nil, err
		}
		records, err := mgr.query(db, query, args...)
		if err != nil || len(records) == 0 {
			// If failed or empty, try simple SHOW COLUMNS
			query = fmt.Sprintf("SHOW COLUMNS FROM `%s`", table)
			records, err = mgr.query(db, query)
			if err != nil {
				return nil, err
			}
			for _, r := range records {
				columns = append(columns, ColumnInfo{
					Name:       r.GetString("Field"),
					Type:       r.GetString("Type"),
					Nullable:   r.GetString("Null") == "YES",
					IsPK:       r.GetString("Key") == "PRI",
					Comment:    "",
					IsAutoIncr: strings.Contains(strings.ToLower(r.GetString("Extra")), "auto_increment"),
				})
			}
		} else {
			for _, r := range records {
				columnName := fmt.Sprintf("%v", r.Get("COLUMN_NAME"))
				dataType := fmt.Sprintf("%v", r.Get("DATA_TYPE"))
				isNullable := fmt.Sprintf("%v", r.Get("IS_NULLABLE"))
				columnKey := fmt.Sprintf("%v", r.Get("COLUMN_KEY"))
				columnComment := fmt.Sprintf("%v", r.Get("COLUMN_COMMENT"))
				extra := fmt.Sprintf("%v", r.Get("EXTRA"))

				columns = append(columns, ColumnInfo{
					Name:       columnName,
					Type:       dataType,
					Nullable:   isNullable == "YES",
					IsPK:       columnKey == "PRI",
					Comment:    columnComment,
					IsAutoIncr: strings.Contains(strings.ToLower(extra), "auto_increment"),
				})
			}
		}

	case SQLite3:
		// 使用公共查询构建函数
		query, args := mgr.buildColumnQuery(table)
		db, err := mgr.getDB()
		if err != nil {
			return nil, err
		}
		records, err := mgr.query(db, query, args...)
		if err != nil {
			return nil, err
		}
		for _, r := range records {
			columnKey := r.GetString("column_key")
			extra := r.GetString("extra")
			isAutoIncr := strings.Contains(strings.ToLower(extra), "identity") ||
				strings.Contains(strings.ToLower(extra), "serial") ||
				strings.Contains(strings.ToLower(extra), "nextval")

			columns = append(columns, ColumnInfo{
				Name:             r.GetString("column_name"),
				Type:             r.GetString("data_type"),
				Nullable:         r.GetString("is_nullable") == "YES",
				IsPK:             columnKey == "PRI",
				Comment:          r.GetString("column_comment"),
				IsAutoIncr:       isAutoIncr,
				IsIdentityAlways: strings.Contains(strings.ToLower(extra), "always"),
			})
		}
		// // SQLite: 检查是否有 INTEGER PRIMARY KEY (自动自增)
		// // 需要查询 sqlite_master 获取建表语句
		// var createSQL string
		// sqlQuery := fmt.Sprintf("SELECT sql FROM sqlite_master WHERE type='table' AND name='%s'", table)
		// sqlRecords, sqlErr := mgr.query(db, sqlQuery)
		// if sqlErr == nil && len(sqlRecords) > 0 {
		// 	createSQL = strings.ToUpper(sqlRecords[0].GetString("sql"))
		// }

		// for _, r := range records {
		// 	colName := r.GetString("name")
		// 	colType := r.GetString("type")
		// 	isPK := r.GetInt("pk") > 0

		// 	// SQLite: INTEGER PRIMARY KEY 自动自增
		// 	isAutoIncr := false
		// 	if isPK && strings.ToUpper(colType) == "INTEGER" {
		// 		isAutoIncr = true
		// 	}
		// 	// 或者建表语句中明确指定了 AUTOINCREMENT
		// 	if createSQL != "" && strings.Contains(createSQL, strings.ToUpper(colName)) {
		// 		if strings.Contains(createSQL, "AUTOINCREMENT") {
		// 			isAutoIncr = true
		// 		}
		// 	}

		// 	columns = append(columns, ColumnInfo{
		// 		Name:       colName,
		// 		Type:       colType,
		// 		Nullable:   r.GetInt("notnull") == 0,
		// 		IsPK:       isPK,
		// 		Comment:    "", // SQLite 不支持列注释
		// 		IsAutoIncr: isAutoIncr,
		// 	})
		// }

	case PostgreSQL:
		// 使用公共查询构建函数
		query, args := mgr.buildColumnQuery(table)
		db, err := mgr.getDB()
		if err != nil {
			return nil, err
		}
		records, err := mgr.query(db, query, args...)
		if err != nil {
			return nil, err
		}
		for _, r := range records {
			columnKey := r.GetString("column_key")
			extra := r.GetString("extra")
			isAutoIncr := strings.Contains(strings.ToLower(extra), "identity") ||
				strings.Contains(strings.ToLower(extra), "serial") ||
				strings.Contains(strings.ToLower(extra), "nextval")

			columns = append(columns, ColumnInfo{
				Name:             r.GetString("column_name"),
				Type:             r.GetString("data_type"),
				Nullable:         r.GetString("is_nullable") == "YES",
				IsPK:             columnKey == "PRI",
				Comment:          r.GetString("column_comment"),
				IsAutoIncr:       isAutoIncr,
				IsIdentityAlways: strings.Contains(strings.ToLower(extra), "always"),
			})
		}

	case SQLServer:
		// 使用公共查询构建函数
		query, args := mgr.buildColumnQuery(table)
		db, err := mgr.getDB()
		if err != nil {
			return nil, err
		}
		records, err := mgr.query(db, query, args...)
		if err != nil {
			return nil, err
		}
		for _, r := range records {
			columnKey := r.GetString("COLUMN_KEY")
			extra := r.GetString("EXTRA")

			columns = append(columns, ColumnInfo{
				Name:       r.GetString("COLUMN_NAME"),
				Type:       r.GetString("DATA_TYPE"),
				Nullable:   r.GetString("IS_NULLABLE") == "YES",
				IsPK:       columnKey == "PRI",
				Comment:    r.GetString("COLUMN_COMMENT"),
				IsAutoIncr: strings.Contains(strings.ToLower(extra), "auto_increment"),
			})
		}

	case Oracle:
		// 使用公共查询构建函数
		query, args := mgr.buildColumnQuery(table)
		db, err := mgr.getDB()
		if err != nil {
			return nil, err
		}
		records, err := mgr.query(db, query, args...)
		if err != nil {
			return nil, err
		}
		for _, r := range records {
			columnKey := r.GetString("COLUMN_KEY")
			extra := r.GetString("EXTRA")
			nullable := r.GetString("NULLABLE")

			columns = append(columns, ColumnInfo{
				Name:       r.GetString("COLUMN_NAME"),
				Type:       r.GetString("DATA_TYPE"),
				Nullable:   nullable == "Y",
				IsPK:       columnKey == "PRI",
				Comment:    r.GetString("COLUMN_COMMENT"),
				IsAutoIncr: strings.Contains(strings.ToLower(extra), "auto_increment"),
			})
		}

	default:
		return nil, fmt.Errorf("unsupported driver: %s", driver)
	}

	// 3. 缓存结果（写锁）
	mgr.mu.Lock()
	if mgr.columnCache == nil {
		mgr.columnCache = make(map[string][]ColumnInfo)
	}
	mgr.columnCache[table] = columns
	mgr.mu.Unlock()

	return columns, nil
}

func SnakeToCamel(s string) string {
	s = strings.ToLower(s)
	parts := strings.Split(s, "_")
	for i := range parts {
		if len(parts[i]) > 0 {
			if strings.EqualFold(parts[i], "id") {
				parts[i] = "ID"
			} else {
				parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
			}
		}
	}
	return strings.Join(parts, "")
}

// dbTypeToGoType 将数据库类型转换为 Go 类型，支持可空字段
func dbTypeToGoType(dbType string, nullable bool, isPK bool) string {
	dbType = strings.ToLower(dbType)
	var goType string

	switch {
	case strings.Contains(dbType, "int") || strings.Contains(dbType, "integer") || strings.Contains(dbType, "bigint") || strings.Contains(dbType, "smallint") || strings.Contains(dbType, "tinyint"):
		if nullable && !isPK {
			goType = "*int64"
		} else {
			goType = "int64"
		}
	case strings.Contains(dbType, "char") || strings.Contains(dbType, "text") || strings.Contains(dbType, "string") || strings.Contains(dbType, "varchar"):
		if nullable && !isPK {
			goType = "*string"
		} else {
			goType = "string"
		}
	case strings.Contains(dbType, "float") || strings.Contains(dbType, "double") || strings.Contains(dbType, "decimal") || strings.Contains(dbType, "numeric") || strings.Contains(dbType, "number") || strings.Contains(dbType, "real"):
		if nullable && !isPK {
			goType = "*float64"
		} else {
			goType = "float64"
		}
	case strings.Contains(dbType, "date") || strings.Contains(dbType, "time") || strings.Contains(dbType, "timestamp"):
		// 时间类型：如果可空且不是主键，使用指针类型以支持 NULL 值和避免 MySQL 零值问题
		if nullable && !isPK {
			goType = "*time.Time"
		} else {
			goType = "time.Time"
		}
	case strings.Contains(dbType, "bool") || strings.Contains(dbType, "boolean"):
		if nullable && !isPK {
			goType = "*bool"
		} else {
			goType = "bool"
		}
	case strings.Contains(dbType, "json") || strings.Contains(dbType, "jsonb"):
		if nullable && !isPK {
			goType = "*string"
		} else {
			goType = "string"
		}
	case strings.Contains(dbType, "blob") || strings.Contains(dbType, "binary"):
		goType = "[]byte" // 二进制数据通常不使用指针
	default:
		goType = "interface{}"
	}

	return goType
}

// getAllTables 获取数据库中所有表名
func (mgr *dbManager) getAllTables() ([]string, error) {
	var tables []string
	driver := mgr.config.Driver

	db, err := mgr.getDB()
	if err != nil {
		return nil, err
	}

	switch driver {
	case MySQL:
		// MySQL: 查询当前数据库的所有表
		query := "SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_TYPE = 'BASE TABLE' ORDER BY TABLE_NAME"
		records, err := mgr.query(db, query)
		if err != nil {
			return nil, err
		}
		for _, r := range records {
			if tableName := r.GetString("TABLE_NAME"); tableName != "" {
				tables = append(tables, tableName)
			}
		}

	case PostgreSQL:
		// PostgreSQL: 查询当前 schema 的所有表
		query := "SELECT tablename FROM pg_catalog.pg_tables WHERE schemaname = current_schema() ORDER BY tablename"
		records, err := mgr.query(db, query)
		if err != nil {
			return nil, err
		}
		for _, r := range records {
			if tableName := r.GetString("tablename"); tableName != "" {
				tables = append(tables, tableName)
			}
		}

	case SQLite3:
		// SQLite3: 查询所有表（排除系统表）
		query := "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name"
		records, err := mgr.query(db, query)
		if err != nil {
			return nil, err
		}
		for _, r := range records {
			if tableName := r.GetString("name"); tableName != "" {
				tables = append(tables, tableName)
			}
		}

	case Oracle:
		// Oracle: 查询当前用户的所有表
		query := "SELECT TABLE_NAME FROM USER_TABLES ORDER BY TABLE_NAME"
		records, err := mgr.query(db, query)
		if err != nil {
			return nil, err
		}
		for _, r := range records {
			if tableName := r.GetString("TABLE_NAME"); tableName != "" {
				tables = append(tables, tableName)
			}
		}

	case SQLServer:
		// SQL Server: 查询当前数据库的所有表
		query := "SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_TYPE = 'BASE TABLE' ORDER BY TABLE_NAME"
		records, err := mgr.query(db, query)
		if err != nil {
			return nil, err
		}
		for _, r := range records {
			if tableName := r.GetString("TABLE_NAME"); tableName != "" {
				tables = append(tables, tableName)
			}
		}

	default:
		return nil, fmt.Errorf("unsupported driver: %s", driver)
	}

	return tables, nil
}

// GenerateAllDbModel 生成数据库中所有表的 Model 代码（全局函数）
// outPath: 输出目录路径，如果为空则使用 "models" 目录
// 返回生成的文件数量和错误信息
func GenerateAllDbModel(outPath string) (int, error) {
	db, err := defaultDB()
	if err != nil {
		return 0, err
	}
	return db.GenerateAllDbModel(outPath)
}

// GenerateAllDbModel 生成数据库中所有表的 Model 代码
// outPath: 输出目录路径，如果为空则使用 "models" 目录
// 返回生成的文件数量和错误信息
func (db *DB) GenerateAllDbModel(outPath string) (int, error) {
	if db.lastErr != nil {
		return 0, db.lastErr
	}

	// 1. 获取所有表名
	tables, err := db.dbMgr.getAllTables()
	if err != nil {
		return 0, fmt.Errorf("failed to get table list: %v", err)
	}

	if len(tables) == 0 {
		return 0, fmt.Errorf("no tables found in database")
	}

	// 2. 确定输出目录
	if outPath == "" {
		outPath = "models"
	}

	// 3. 创建输出目录
	if err := os.MkdirAll(outPath, 0755); err != nil {
		return 0, fmt.Errorf("failed to create output directory: %v", err)
	}

	// 4. 遍历所有表，生成 Model
	successCount := 0
	var errors []string

	for _, table := range tables {
		// 为每个表生成 Model，structName 为空表示自动生成
		err := db.GenerateDbModel(table, outPath, "")
		if err != nil {
			errors = append(errors, fmt.Sprintf("table '%s': %v", table, err))
		} else {
			successCount++
		}
	}

	// 5. 返回结果
	if len(errors) > 0 {
		return successCount, fmt.Errorf("generated %d/%d models successfully, errors: %s",
			successCount, len(tables), strings.Join(errors, "; "))
	}

	return successCount, nil
}
