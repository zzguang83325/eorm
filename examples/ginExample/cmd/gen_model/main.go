package main

import (
	"log"

	"github.com/zzguang83325/eorm"
	"github.com/zzguang83325/eorm/examples/ginExample/config"
)

func main() {
	// 1. 初始化数据库
	dsn := "root:123456@tcp(localhost:3306)/test?charset=utf8mb4&parseTime=True&loc=Local"
	config.InitDB(dsn)

	// 强制重建表，避免旧表结构干扰
	log.Println("Recreating tables...")
	eorm.Exec("DROP TABLE IF EXISTS users")
	// 注意：point_logs 可能没被删除，这里主要是 users 表有问题
	eorm.Exec("DROP TABLE IF EXISTS point_logs")

	// 重新运行建表 SQL，这里我们简单地再次调用 initSchema 逻辑
	// 但由于 config 包不可导出 initSchema，我们在这里手动执行一次建表 SQL
	eorm.Exec(`
		CREATE TABLE users (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			username VARCHAR(50) NOT NULL UNIQUE,
			balance INT DEFAULT 0 COMMENT '积分余额',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)
	`)
	eorm.Exec(`
		CREATE TABLE point_logs (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			user_id BIGINT NOT NULL,
			amount INT NOT NULL COMMENT '变动金额',
			reason VARCHAR(100) NOT NULL COMMENT '变动原因',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_user_id (user_id)
		)
	`)

	log.Println("Starting model generation...")

	// 2. 生成 User 模型
	// outPath 指定生成的具体文件路径
	err := eorm.GenerateDbModel("users", "examples/realworld/user_service/internal/model/user.go", "User")
	if err != nil {
		log.Printf("Failed to generate User model: %v", err)
	} else {
		log.Println("Generated internal/model/user.go")
	}

	// 3. 生成 PointLog 模型
	err = eorm.GenerateDbModel("point_logs", "examples/realworld/user_service/internal/model/point_log.go", "PointLog")
	if err != nil {
		log.Printf("Failed to generate PointLog model: %v", err)
	} else {
		log.Println("Generated internal/model/point_log.go")
	}

	log.Println("Model generation complete.")
}
