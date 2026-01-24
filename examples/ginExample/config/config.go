package config

import (
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql" // 导入 MySQL 驱动
	"github.com/zzguang83325/eorm"
)

// InitDB 初始化数据库连接
// 在实际微服务中，这里通常会读取 file 或 env 配置
func InitDB(dsn string) error {
	config := &eorm.Config{
		Driver:                eorm.MySQL,
		DSN:                   dsn,
		MaxOpen:               50,
		MaxIdle:               10,
		ConnMaxLifetime:       30 * time.Minute,
		MonitorNormalInterval: 1 * time.Minute,
	}

	// 开启 Debug 模式以便观察 SQL
	eorm.SetDebugMode(true)

	_, err := eorm.OpenDatabaseWithConfig("default", config)
	if err != nil {
		return err
	}

	log.Println("Database connection established")

	// 自动迁移/初始化表结构（仅用于演示）
	initSchema()

	return nil
}

func initSchema() {
	// 用户表
	eorm.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			username VARCHAR(50) NOT NULL UNIQUE,
			balance INT DEFAULT 0 COMMENT '积分余额',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)
	`)

	// 积分流水表
	eorm.Exec(`
		CREATE TABLE IF NOT EXISTS point_logs (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			user_id BIGINT NOT NULL,
			amount INT NOT NULL COMMENT '变动金额',
			reason VARCHAR(100) NOT NULL COMMENT '变动原因',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_user_id (user_id)
		)
	`)
}
