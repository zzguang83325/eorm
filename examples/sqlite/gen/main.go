package main

import (
	"log"

	"github.com/zzguang83325/eorm"
	_ "github.com/zzguang83325/eorm/drivers/sqlite"
)

func main() {
	// 1. 连接 SQLite 数据库
	dsn := "file:../test_multi.db?cache=shared&mode=rwc"
	_,err := eorm.OpenDatabaseWithDBName("sqlite", eorm.SQLite3, dsn, 10)
	if err != nil {
		log.Fatalf("SQLite数据库连接失败: %v", err)
	}

	// 2. 确保表存在
	setupTable()

	// 3. 生成模型
	err = eorm.Use("sqlite").GenerateDbModel("demo", "../models", "Demo")
	if err != nil {
		log.Fatalf("生成模型失败: %v", err)
	}

	log.Println("SQLite Demo 模型生成成功")
}

func setupTable() {
	sql := `
	CREATE TABLE IF NOT EXISTS demo (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		age INTEGER,
		salary REAL,
		is_active INTEGER,
		birthday TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		metadata TEXT
	)`
	_, err := eorm.Use("sqlite").Exec(sql)
	if err != nil {
		log.Fatalf("创建表失败: %v", err)
	}
}
