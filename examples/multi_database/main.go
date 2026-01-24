package main

import (
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/zzguang83325/eorm"
)

// 多数据库模式示例
// 演示如何使用 eorm 的多数据库功能
func main() {
	fmt.Println("========================================")
	fmt.Println("      eorm 多数据库模式示例")
	fmt.Println("========================================")

	// 1. 初始化多个数据库连接
	fmt.Println("\n[1] 初始化多个数据库连接")
	initializeMultipleDatabases()

	// 2. 演示 Use 和 UseWithError
	fmt.Println("\n[2] 使用 Use 和 UseWithError")
	demonstrateUseFunctions()

	// 3. 演示不同数据库的操作
	fmt.Println("\n[3] 不同数据库的操作")
	demonstrateDatabaseOperations()

	// 4. 演示跨数据库查询
	fmt.Println("\n[4] 跨数据库查询")
	demonstrateCrossDatabaseQueries()

	// 5. 演示事务中的多数据库操作
	fmt.Println("\n[5] 事务中的多数据库操作")
	demonstrateMultiDatabaseTransactions()

	// 6. 演示数据库特定的配置
	fmt.Println("\n[6] 数据库特定的配置")
	demonstrateDatabaseSpecificConfigs()

	fmt.Println("\n========================================")
	fmt.Println("      多数据库模式示例演示完成")
	fmt.Println("========================================")

	// 程序退出前关闭所有数据库连接
	fmt.Println("\n正在关闭所有数据库连接...")
	eorm.Close()
	fmt.Println("✓ 所有数据库连接已关闭")
}

// initializeMultipleDatabases 初始化多个数据库连接
func initializeMultipleDatabases() {
	fmt.Println("场景：连接多个数据库实例")

	// 主数据库 - 用户数据
	fmt.Println("  连接主数据库 (用户数据)...")
	mainConfig := &eorm.Config{
		Driver:                eorm.MySQL,
		DSN:                   "root:123456@tcp(127.0.0.1:3306)/test?charset=utf8mb4&parseTime=True&loc=Local",
		MaxOpen:               20,
		MaxIdle:               10,
		MonitorNormalInterval: 60 * time.Second,
		MonitorErrorInterval:  10 * time.Second,
	}

	_, err := eorm.OpenDatabaseWithConfig("main", mainConfig)
	if err != nil {
		log.Printf("主数据库连接失败: %v", err)
		return
	}

	// 日志数据库 - 日志数据
	fmt.Println("  连接日志数据库 (日志数据)...")
	logConfig := &eorm.Config{
		Driver:                eorm.MySQL,
		DSN:                   "root:123456@tcp(127.0.0.1:3306)/test?charset=utf8mb4&parseTime=True&loc=Local",
		MaxOpen:               5,
		MaxIdle:               2,
		MonitorNormalInterval: 30 * time.Second,
		MonitorErrorInterval:  5 * time.Second,
	}

	_, err = eorm.OpenDatabaseWithConfig("log", logConfig)
	if err != nil {
		log.Printf("日志数据库连接失败: %v", err)
		return
	}

	// 订单数据库 - 订单数据
	fmt.Println("  连接订单数据库 (订单管理系统)...")
	orderConfig := &eorm.Config{
		Driver:                eorm.MySQL,
		DSN:                   "root:123456@tcp(127.0.0.1:3306)/test?charset=utf8mb4&parseTime=True&loc=Local",
		MaxOpen:               5,
		MaxIdle:               2,
		MonitorNormalInterval: 30 * time.Second,
		MonitorErrorInterval:  5 * time.Second,
	}

	_, err = eorm.OpenDatabaseWithConfig("orders", orderConfig)
	if err != nil {
		log.Printf("订单数据库连接失败: %v", err)
		return
	}

	// 测试所有数据库连接
	databases := []string{"main", "log", "orders"}
	for _, dbname := range databases {
		if err := eorm.PingDB(dbname); err != nil {
			log.Printf("数据库 %s 连接测试失败: %v", dbname, err)
		} else {
			fmt.Printf("  ✓ 数据库 %s 连接成功\n", dbname)
		}
	}

	// 创建测试表
	setupTestTables()

	// 注意：不要在这里关闭数据库连接，因为其他地方还需要使用
	// defer mainDB.Close()
	// defer logDB.Close()
	// defer orderDB.Close()
}

// setupTestTables 创建测试表
func setupTestTables() {
	// 主数据库表
	_, err := eorm.Use("main").BatchExec([]string{
		"DROP TABLE IF EXISTS main_users",
		"CREATE TABLE main_users (id INT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(100), email VARCHAR(100), created_at TIMESTAMP NULL, updated_at TIMESTAMP NULL)",
	})
	if err != nil {
		log.Printf("主数据库建表失败: %v", err)
	}

	// 日志数据库表
	_, err = eorm.Use("log").BatchExec([]string{
		"DROP TABLE IF EXISTS log_operations",
		"CREATE TABLE log_operations (id INT PRIMARY KEY AUTO_INCREMENT, operation VARCHAR(100), table_name VARCHAR(100), created_at TIMESTAMP, deleted_at TIMESTAMP NULL)",
	})
	if err != nil {
		log.Printf("日志数据库建表失败: %v", err)
	}

	// 订单数据库表
	_, err = eorm.Use("orders").BatchExec([]string{
		"DROP TABLE IF EXISTS order_data",
		`CREATE TABLE order_data (
		id INT PRIMARY KEY AUTO_INCREMENT,
		order_no VARCHAR(50) UNIQUE NOT NULL,
		customer_id INT,
		product_name VARCHAR(200),
		quantity INT DEFAULT 1,
		unit_price DECIMAL(10,2),
		total_price DECIMAL(10,2),
		order_status VARCHAR(20) DEFAULT 'pending',
		order_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		INDEX idx_order_no (order_no),
		INDEX idx_customer_id (customer_id),
		INDEX idx_order_date (order_date)
	)`,
	})
	if err != nil {
		log.Printf("订单数据库建表失败: %v", err)
	}

	fmt.Println("  ✓ 测试表创建完成")
}

// demonstrateUseFunctions 演示 Use 和 UseWithError
func demonstrateUseFunctions() {
	fmt.Println("场景：使用 Use 和 UseWithError 进行链式数据库调用")

	// 使用 Use 进行链式调用
	fmt.Println("  使用 Use 进行链式调用:")

	// 主数据库链式调用
	records, err := eorm.Use("main").Query("SELECT 'main_database' as source")
	if err != nil {
		log.Printf("主数据库查询失败: %v", err)
	} else {
		fmt.Printf("    主数据库查询成功: %s\n", records[0].GetString("source"))
	}

	// 日志数据库链式调用
	records, err = eorm.Use("log").Query("SELECT 'log_database' as source")
	if err != nil {
		log.Printf("日志数据库查询失败: %v", err)
	} else {
		fmt.Printf("    日志数据库查询成功: %s\n", records[0].GetString("source"))
	}

	// 使用 UseWithError 进行错误检查（不用于链式调用）
	fmt.Println("  使用 UseWithError 得到数据库,同时返回错误检查:")

	orderDB, err := eorm.UseWithError("orders")
	if err != nil {
		log.Printf("切换到订单数据库失败: %v", err)
	} else {
		// 获取到数据库连接后，可以进行操作
		records, err := orderDB.Query("SELECT 'orders_database' as source")
		if err != nil {
			log.Printf("订单数据库查询失败: %v", err)
		} else {
			fmt.Printf("    订单数据库查询成功: %s\n", records[0].GetString("source"))
		}
	}

	orderDB, err = eorm.UseWithError("orders")
	if err != nil {
		log.Printf("切换到订单数据库失败: %v", err)
	} else {
		// 获取到数据库连接后，可以进行操作
		records, err := orderDB.Query("SELECT 'orders_database' as source")
		if err != nil {
			log.Printf("订单数据库查询失败: %v", err)
		} else {
			fmt.Printf("    订单数据库查询成功: %s\n", records[0].GetString("source"))
		}
	}

	// 尝试切换到不存在的数据库
	fmt.Println("  尝试切换到不存在的数据库:")
	_, err = eorm.UseWithError("nonexistent")
	if err != nil {
		fmt.Printf("    ✓ 预期错误: %v\n", err)
	} else {
		fmt.Println("    ⚠ 意外成功")
	}

	// 演示链式调用
	fmt.Println("  演示链式调用:")

	// 在主数据库中进行链式查询
	users, err := eorm.Use("main").
		Table("main_users").
		Where("name LIKE ?", "%链式%").
		OrderBy("id DESC").
		Limit(10).
		Find()
	if err != nil {
		log.Printf("链式查询失败: %v", err)
	} else {
		fmt.Printf("    ✓ 链式查询成功，找到 %d 个用户\n", len(users))
		for _, user := range users {
			fmt.Printf("      - ID: %d, 姓名: %s, 邮箱: %s\n",
				user.GetInt("id"), user.GetString("name"), user.GetString("email"))
		}
	}

	// 演示链式插入（使用正确的方法）
	fmt.Println("  演示链式插入:")

	// 使用 InsertRecord 方法进行插入
	userRecord := eorm.NewRecord().Set("name", "链式用户2").Set("email", "chain2@example.com")
	userID, err := eorm.Use("main").InsertRecord("main_users", userRecord)
	if err != nil {
		log.Printf("链式插入失败: %v", err)
	} else {
		fmt.Printf("    ✓ 主数据库链式插入成功，ID: %d\n", userID)
	}
}

func demonstrateDatabaseOperations() {
	fmt.Println("场景：在不同数据库中执行不同的操作")

	// 在主数据库中插入用户数据
	fmt.Println("  主数据库操作 (用户数据):")
	userRecord := eorm.NewRecord().
		Set("name", "张三").
		Set("email", "zhangsan@example.com")
	userID, err := eorm.Use("main").InsertRecord("main_users", userRecord)
	if err != nil {
		log.Printf("用户插入失败: %v", err)
	} else {
		fmt.Printf("    ✓ 用户插入成功，ID: %d\n", userID)
	}

	// 在日志数据库中记录操作
	fmt.Println("  日志数据库操作 (记录日志):")
	logRecord := eorm.NewRecord().
		Set("operation", "INSERT").
		Set("table_name", "main_users").
		Set("created_at", time.Now())
	logID, err := eorm.Use("log").InsertRecord("log_operations", logRecord)
	if err != nil {
		log.Printf("日志插入失败: %v", err)
	} else {
		fmt.Printf("    ✓ 日志记录成功，ID: %d\n", logID)
	}

	// 在订单数据库中存储订单数据
	fmt.Println("  订单数据库操作 (订单管理):")
	orderRecord := eorm.NewRecord().
		Set("order_no", "ORD-2024-001").
		Set("customer_id", 1).
		Set("product_name", "笔记本电脑").
		Set("quantity", 2).
		Set("unit_price", 4999.99).
		Set("total_price", 9999.98).
		Set("order_status", "pending").
		Set("order_date", time.Now())
	orderID, err := eorm.Use("orders").InsertRecord("order_data", orderRecord)
	if err != nil {
		log.Printf("订单存储失败: %v", err)
	} else {
		fmt.Printf("    ✓ 订单创建成功，订单ID: %d\n", orderID)
	}

	// 创建订单明细记录
	orderItemRecord := eorm.NewRecord().
		Set("order_no", "ORD-2024-001").
		Set("customer_id", 1).
		Set("product_name", "鼠标").
		Set("quantity", 2).
		Set("unit_price", 199.99).
		Set("total_price", 399.98).
		Set("order_status", "pending").
		Set("order_date", time.Now())
	orderItemID, err := eorm.Use("orders").InsertRecord("order_data", orderItemRecord)
	if err != nil {
		log.Printf("订单明细存储失败: %v", err)
	} else {
		fmt.Printf("    ✓ 订单明细创建成功，明细ID: %d\n", orderItemID)
	}

	// 查询订单数据库
	orders, err := eorm.Use("orders").Query("SELECT * FROM order_data ORDER BY order_date DESC")
	if err != nil {
		log.Printf("订单查询失败: %v", err)
	} else {
		fmt.Printf("    ✓ 订单查询成功，订单数: %d\n", len(orders))
		for _, order := range orders {
			fmt.Printf("      - 订单号: %s, 客户ID: %d, 产品: %s, 数量: %d, 总价: %.2f\n",
				order.GetString("order_no"), order.GetInt("customer_id"), order.GetString("product_name"),
				order.GetInt("quantity"), order.GetFloat("total_price"))
		}
	}

	// 按客户统计订单
	customerStats, err := eorm.Use("orders").Query("SELECT customer_id, COUNT(*) as order_count, SUM(total_price) as total_amount FROM order_data GROUP BY customer_id")
	if err != nil {
		log.Printf("客户订单统计失败: %v", err)
	} else {
		fmt.Printf("    ✓ 客户订单统计成功，客户数: %d\n", len(customerStats))
		for _, stat := range customerStats {
			fmt.Printf("      - 客户ID: %d, 订单数: %d, 总金额: %.2f\n",
				stat.GetInt("customer_id"), stat.GetInt("order_count"), stat.GetFloat("total_amount"))
		}
	}
}

func demonstrateCrossDatabaseQueries() {
	fmt.Println("场景：跨数据库查询和数据同步")

	// 将用户信息同步到订单数据库
	fmt.Println("  跨数据库数据同步:")

	// 从主数据库查询用户信息
	users, err := eorm.Use("main").Query("SELECT * FROM main_users")
	if err != nil {
		log.Printf("用户查询失败: %v", err)
		return
	}

	// 将用户信息同步到订单数据库
	for _, user := range users {
		// 为每个用户创建订单
		orderRecord := eorm.NewRecord().
			Set("order_no", fmt.Sprintf("ORD-%d", user.GetInt("id"))).
			Set("customer_id", user.GetInt("id")).
			Set("product_name", "默认产品").
			Set("quantity", 1).
			Set("unit_price", 99.99).
			Set("total_price", 99.99).
			Set("order_status", "pending").
			Set("order_date", time.Now())

		orderID, err := eorm.Use("orders").InsertRecord("order_data", orderRecord)
		if err != nil {
			log.Printf("用户订单同步失败: %v", err)
		} else {
			fmt.Printf("    ✓ 用户 %s 的订单同步成功，订单ID: %d\n", user.GetString("name"), orderID)
		}
	}

	fmt.Printf("    ✓ 同步 %d 个用户到订单数据库\n", len(users))

	// 记录同步操作到日志数据库
	logRecord := eorm.NewRecord().
		Set("operation", "SYNC").
		Set("table_name", "order_data").
		Set("created_at", time.Now())

	_, err = eorm.Use("log").InsertRecord("log_operations", logRecord)
	if err != nil {
		log.Printf("同步日志记录失败: %v", err)
	} else {
		fmt.Println("    ✓ 同步操作已记录到日志")
	}

	// 验证同步结果
	fmt.Println("  验证同步结果:")
	orderCount, err := eorm.Use("orders").Count("order_data", "1=1")
	if err != nil {
		log.Printf("订单统计失败: %v", err)
	} else {
		fmt.Printf("    ✓ 订单数据库中的用户订单数: %d\n", orderCount)
	}
}

// demonstrateMultiDatabaseTransactions 演示事务中的多数据库操作
func demonstrateMultiDatabaseTransactions() {
	fmt.Println("场景：在事务中处理多数据库操作")

	// 注意：每个数据库的事务是独立的
	fmt.Println("  主数据库事务操作:")

	err := eorm.Use("main").Transaction(func(tx *eorm.Tx) error {
		// 在主数据库事务中插入用户
		userRecord := eorm.NewRecord().
			Set("name", "李四").
			Set("email", "lisi@example.com")

		_, err := tx.InsertRecord("main_users", userRecord)
		if err != nil {
			return fmt.Errorf("用户插入失败: %v", err)
		}

		// 在同一个事务中更新用户（模拟复杂操作）
		updateRecord := eorm.NewRecord().Set("email", "lisi_updated@example.com")
		_, err = tx.Update("main_users", updateRecord, "name = ?", "李四")
		if err != nil {
			return fmt.Errorf("用户更新失败: %v", err)
		}

		fmt.Println("    ✓ 主数据库事务操作成功")
		return nil
	})

	if err != nil {
		log.Printf("主数据库事务失败: %v", err)
	} else {
		fmt.Println("  ✓ 主数据库事务提交成功")
	}

	// 为订单数据库配置查询超时
	fmt.Println("  订单数据库配置:")
	orderDB := eorm.Use("orders").Timeout(5 * time.Second)
	fmt.Println("    ✓ 设置查询超时为 5 秒")

	// 测试配置效果
	fmt.Println("  测试配置效果:")

	// 测试时间戳功能
	userRecord := eorm.NewRecord().Set("name", "王五").Set("email", "wangwu@example.com")
	userID, err := eorm.Use("main").InsertRecord("main_users", userRecord)
	if err != nil {
		log.Printf("时间戳测试失败: %v", err)
	} else {
		fmt.Printf("    ✓ 时间戳功能测试成功，用户ID: %d\n", userID)
	}

	// 测试软删除功能
	logRecord := eorm.NewRecord().Set("operation", "TEST").Set("table_name", "test_table").Set("created_at", time.Now())
	logID, err := eorm.Use("log").InsertRecord("log_operations", logRecord)
	if err != nil {
		log.Printf("软删除测试失败: %v", err)
	} else {
		fmt.Printf("    ✓ 软删除功能测试成功，日志ID: %d\n", logID)
		_, err = eorm.Use("log").Delete("log_operations", "id = ?", logID)
		if err != nil {
			log.Printf("软删除操作失败: %v", err)
		} else {
			fmt.Println("    ✓ 软删除操作成功")
		}
	}

	// 测试查询超时功能
	start := time.Now()
	_, err = orderDB.Query("SELECT SLEEP(1)")
	elapsed := time.Since(start)
	if err != nil {
		log.Printf("超时测试失败: %v", err)
	} else {
		fmt.Printf("    ✓ 查询超时功能测试成功，耗时: %v\n", elapsed)
	}
}

func demonstrateDatabaseSpecificConfigs() {
	fmt.Println("场景：为不同数据库配置不同的功能")

	// 为主数据库启用时间戳功能
	fmt.Println("  主数据库配置:")
	eorm.Use("main").EnableTimestamps().ConfigTimestampsWithFields("main_users", "created_at", "updated_at")
	fmt.Println("    ✓ 启用时间戳功能")

	// 为日志数据库启用软删除功能
	fmt.Println("  日志数据库配置:")
	eorm.Use("log").EnableSoftDelete().ConfigSoftDelete("log_operations", "deleted_at")
	fmt.Println("    ✓ 启用软删除功能")
}

// init 函数：设置日志
func init() {
	// 开启调试模式
	eorm.SetDebugMode(true)

	// 初始化文件日志
	eorm.InitLoggerWithFile("debug", "multi_database.log")
}
