package main

import (
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/zzguang83325/eorm"
)

// BatchExec 批量执行 SQL 语句示例
// 演示如何使用 BatchExec 功能进行批量 SQL 操作
func main() {
	fmt.Println("========================================")
	fmt.Println("      eorm BatchExec 批量执行示例")
	fmt.Println("========================================")

	// 1. 初始化数据库连接
	fmt.Println("\n[1] 初始化数据库连接...")
	dsn := "root:123456@tcp(127.0.0.1:3306)/test?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := eorm.OpenDatabase(eorm.MySQL, dsn, 10)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("数据库连接测试失败: %v", err)
	}
	fmt.Println("✓ 数据库连接成功")

	// 开启调试模式查看 SQL
	eorm.SetDebugMode(true)

	// 2. 准备测试环境
	fmt.Println("\n[2] 准备测试环境...")
	setupBatchExecTables()

	// 3. 演示 BatchExec 功能
	demonstrateBasicBatchExec()
	demonstrateBatchExecWithParams()
	demonstrateBatchExecInTransaction()
	demonstrateBatchExecErrorHandling()
	demonstrateBatchExecDDL()

	fmt.Println("\n========================================")
	fmt.Println("      BatchExec 示例演示完成")
	fmt.Println("========================================")
}

// setupBatchExecTables 准备测试表
func setupBatchExecTables() {
	sqls := []string{
		"DROP TABLE IF EXISTS batch_users",
		"DROP TABLE IF EXISTS batch_orders",
		"CREATE TABLE batch_users (id INT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(100), email VARCHAR(100), age INT)",
		"CREATE TABLE batch_orders (id INT PRIMARY KEY AUTO_INCREMENT, user_id INT, product VARCHAR(100), amount DECIMAL(10,2), created_at TIMESTAMP)",
	}

	results, err := eorm.BatchExec(sqls)
	if err != nil {
		log.Printf("准备表结构失败: %v", err)
		return
	}

	successCount := 0
	for _, r := range results {
		if r.IsSuccess() {
			successCount++
		}
	}
	fmt.Printf("✓ 测试表准备完成，成功执行 %d 条语句\n", successCount)
}

// demonstrateBasicBatchExec 基础批量执行（无参数）
func demonstrateBasicBatchExec() {
	fmt.Println("\n[3.1] 基础批量执行（无参数）")
	fmt.Println("场景：数据库初始化、表结构创建等")

	sqls := []string{
		"INSERT INTO batch_users (name, email, age) VALUES ('Alice', 'alice@example.com', 25)",
		"INSERT INTO batch_users (name, email, age) VALUES ('Bob', 'bob@example.com', 30)",
		"INSERT INTO batch_users (name, email, age) VALUES ('Charlie', 'charlie@example.com', 28)",
		"UPDATE batch_users SET age = age + 1 WHERE name = 'Alice'",
	}

	results, err := eorm.BatchExec(sqls)
	if err != nil {
		log.Printf("批量执行失败: %v", err)
		return
	}

	fmt.Printf("执行结果：\n")
	for _, r := range results {
		if r.IsSuccess() {
			affected, _ := r.RowsAffected()
			fmt.Printf("  ✓ 语句 %d: 成功，影响 %d 行\n", r.Index, affected)
		} else {
			fmt.Printf("  ✗ 语句 %d: 失败 - %v\n", r.Index, r.Error)
		}
	}
}

// demonstrateBatchExecWithParams 带参数的批量执行
func demonstrateBatchExecWithParams() {
	fmt.Println("\n[3.2] 带参数的批量执行")
	fmt.Println("场景：批量插入数据、批量更新等")

	sqls := []string{
		"INSERT INTO batch_users (name, email, age) VALUES (?, ?, ?)",
		"INSERT INTO batch_users (name, email, age) VALUES (?, ?, ?)",
		"INSERT INTO batch_orders (user_id, product, amount, created_at) VALUES (?, ?, ?, ?)",
		"INSERT INTO batch_orders (user_id, product, amount, created_at) VALUES (?, ?, ?, ?)",
	}

	args := [][]interface{}{
		{"David", "david@example.com", 32},
		{"Eva", "eva@example.com", 27},
		{1, "Laptop", 999.99, time.Now()},
		{2, "Mouse", 29.99, time.Now()},
	}

	results, err := eorm.BatchExec(sqls, args[0], args[1], args[2], args[3])
	if err != nil {
		log.Printf("带参数批量执行失败: %v", err)
		return
	}

	fmt.Printf("执行结果：\n")
	for _, r := range results {
		if r.IsSuccess() {
			affected, _ := r.RowsAffected()
			lastID, _ := r.LastInsertId()
			if lastID > 0 {
				fmt.Printf("  ✓ 语句 %d: 成功，影响 %d 行，插入 ID: %d\n", r.Index, affected, lastID)
			} else {
				fmt.Printf("  ✓ 语句 %d: 成功，影响 %d 行\n", r.Index, affected)
			}
		} else {
			fmt.Printf("  ✗ 语句 %d: 失败 - %v\n", r.Index, r.Error)
		}
	}
}

// demonstrateBatchExecInTransaction 事务中的批量执行
func demonstrateBatchExecInTransaction() {
	fmt.Println("\n[3.3] 事务中的批量执行")
	fmt.Println("场景：需要原子性保证的批量操作")

	err := eorm.Transaction(func(tx *eorm.Tx) error {
		sqls := []string{
			"INSERT INTO batch_users (name, email, age) VALUES (?, ?, ?)",
			"INSERT INTO batch_orders (user_id, product, amount, created_at) VALUES (?, ?, ?, ?)",
			"UPDATE batch_users SET age = age + 1 WHERE id = ?",
		}

		args := [][]interface{}{
			{"Frank", "frank@example.com", 35},
			{3, "Keyboard", 79.99, time.Now()},
			{3},
		}

		results, err := tx.BatchExec(sqls, args[0], args[1], args[2])
		if err != nil {
			return fmt.Errorf("事务中批量执行失败: %v", err)
		}

		// 检查所有语句是否成功
		for _, r := range results {
			if !r.IsSuccess() {
				return fmt.Errorf("语句 %d 执行失败: %v", r.Index, r.Error)
			}
		}

		fmt.Printf("  ✓ 事务中成功执行 %d 条语句\n", len(results))
		return nil
	})

	if err != nil {
		log.Printf("事务执行失败: %v", err)
	} else {
		fmt.Println("  ✓ 事务提交成功")
	}
}

// demonstrateBatchExecErrorHandling 错误处理演示
func demonstrateBatchExecErrorHandling() {
	fmt.Println("\n[3.4] 错误处理演示")
	fmt.Println("场景：演示错误发生时的处理方式")

	// 故意制造一个错误（主键冲突）
	sqls := []string{
		"INSERT INTO batch_users (id, name, email, age) VALUES (?, ?, ?, ?)",
		"INSERT INTO batch_users (id, name, email, age) VALUES (?, ?, ?, ?)", // 主键冲突
		"INSERT INTO batch_users (id, name, email, age) VALUES (?, ?, ?, ?)", // 不会执行
	}

	args := [][]interface{}{
		{100, "Grace", "grace@example.com", 29},
		{1, "Conflict", "conflict@example.com", 40}, // 与现有记录主键冲突
		{101, "Henry", "henry@example.com", 31},
	}

	results, err := eorm.BatchExec(sqls, args[0], args[1], args[2])
	if err != nil {
		fmt.Printf("  ⚠ 批量执行遇到错误: %v\n", err)
	}

	fmt.Printf("错误处理结果：\n")
	successCount := 0
	failedCount := 0

	for _, r := range results {
		if r.IsSuccess() {
			successCount++
			affected, _ := r.RowsAffected()
			fmt.Printf("  ✓ 语句 %d: 成功，影响 %d 行\n", r.Index, affected)
		} else {
			failedCount++
			fmt.Printf("  ✗ 语句 %d: 失败 - %v\n", r.Index, r.Error)
			fmt.Printf("    SQL: %s\n", r.SQL)
			fmt.Printf("    参数: %v\n", r.Args)
		}
	}

	fmt.Printf("  总结: 成功 %d 条，失败 %d 条\n", successCount, failedCount)
}

// demonstrateBatchExecDDL DDL 操作演示
func demonstrateBatchExecDDL() {
	fmt.Println("\n[3.5] DDL 操作演示")
	fmt.Println("场景：数据库结构变更、索引创建等")

	sqls := []string{
		"CREATE TABLE batch_test (id INT PRIMARY KEY, name VARCHAR(100))",
		"CREATE INDEX idx_name ON batch_test(name)",
		"INSERT INTO batch_test (id, name) VALUES (?, ?)",
		"INSERT INTO batch_test (id, name) VALUES (?, ?)",
		"ALTER TABLE batch_test ADD COLUMN email VARCHAR(100)",
		"UPDATE batch_test SET email = ? WHERE id = ?",
	}

	args := [][]interface{}{
		nil, // CREATE TABLE 不需要参数
		nil, // CREATE INDEX 不需要参数
		{1, "Test User 1"},
		{2, "Test User 2"},
		nil, // ALTER TABLE 不需要参数
		{"test@example.com", 1},
	}

	results, err := eorm.BatchExec(sqls, args[0], args[1], args[2], args[3], args[4], args[5])
	if err != nil {
		log.Printf("DDL 批量执行失败: %v", err)
		return
	}

	fmt.Printf("DDL 执行结果：\n")
	for _, r := range results {
		if r.IsSuccess() {
			affected, _ := r.RowsAffected()
			if affected > 0 {
				fmt.Printf("  ✓ 语句 %d: 成功，影响 %d 行\n", r.Index, affected)
			} else {
				fmt.Printf("  ✓ 语句 %d: 成功（DDL 操作）\n", r.Index)
			}
		} else {
			fmt.Printf("  ✗ 语句 %d: 失败 - %v\n", r.Index, r.Error)
		}
	}

	// 清理测试表
	eorm.Exec("DROP TABLE IF EXISTS batch_test")
	fmt.Println("  ✓ 清理测试表完成")
}
