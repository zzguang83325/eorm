package main

import (
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/zzguang83325/eorm"

	"go.uber.org/zap"
)

// ZapAdapter 实现 eorm.Logger 接口，用于集成 zap 日志库
type ZapAdapter struct {
	logger *zap.Logger
}

func (a *ZapAdapter) Log(level eorm.LogLevel, msg string, fields map[string]interface{}) {
	// 将 map[string]interface{} 转换为 zap.Field 切片
	var zapFields []zap.Field
	if len(fields) > 0 {
		zapFields = make([]zap.Field, 0, len(fields))
		for k, v := range fields {
			zapFields = append(zapFields, zap.Any(k, v))
		}
	}

	switch level {
	case eorm.LevelDebug:
		a.logger.Debug(msg, zapFields...)
	case eorm.LevelInfo:
		a.logger.Info(msg, zapFields...)
	case eorm.LevelWarn:
		a.logger.Warn(msg, zapFields...)
	case eorm.LevelError:
		a.logger.Error(msg, zapFields...)
	}
}

func main() {
	// 1. 初始化 zap 日志，同时输出到控制台和 log.log 文件
	cfg := zap.NewDevelopmentConfig()
	cfg.OutputPaths = []string{"stdout", "logfile.log"}

	zapLogger, _ := cfg.Build()
	defer zapLogger.Sync()

	// 2. 将 zap 集成到 eorm
	eorm.SetLogger(&ZapAdapter{logger: zapLogger})
	eorm.SetDebugMode(true) // 开启调试模式以查看 SQL 轨迹

	// 3. 连接 SQLite 数据库
	dbPath := "test.db"
	// 确保开始前文件不存在
	os.Remove(dbPath)
	defer os.Remove(dbPath)

	_, err := eorm.OpenDatabase(eorm.SQLite3, dbPath, 10)
	if err != nil {
		log.Fatalf("无法连接数据库: %v", err)
	}
	defer eorm.Close()

	zapLogger.Info("=== 开始 Zap 日志集成测试 ===")

	// 4. 创建测试表
	_, err = eorm.Exec(`CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		age INTEGER
	)`)
	if err != nil {
		zapLogger.Error("创建表失败", zap.Error(err))
		return
	}

	// 5. 增 (Insert)
	user := eorm.NewRecord().Set("name", "张三").Set("age", 25)
	id, err := eorm.InsertRecord("users", user)
	if err != nil {
		zapLogger.Error("插入数据失败", zap.Error(err))
	} else {
		zapLogger.Info("数据插入成功", zap.Int64("id", id))
	}

	// 6. 查 (Query)
	row, err := eorm.QueryFirst("SELECT * FROM users WHERE id = ?", id)
	if err != nil {
		zapLogger.Error("查询数据失败", zap.Error(err))
	} else if row != nil {
		zapLogger.Info("查询到用户信息",
			zap.String("name", row.GetString("name")),
			zap.Int("age", row.GetInt("age")),
		)
	}

	// 7. 改 (Update)
	user.Set("age", 26)
	affected, err := eorm.Update("users", user, "id = ?", id)
	if err != nil {
		zapLogger.Error("更新数据失败", zap.Error(err))
	} else {
		zapLogger.Info("数据更新成功", zap.Int64("affected_rows", affected))
	}

	// 8. 删 (Delete)
	affected, err = eorm.Delete("users", "id = ?", id)
	if err != nil {
		zapLogger.Error("删除数据失败", zap.Error(err))
	} else {
		zapLogger.Info("数据删除成功", zap.Int64("affected_rows", affected))
	}

	zapLogger.Info("=== Zap 日志集成测试完成 ===")
}
