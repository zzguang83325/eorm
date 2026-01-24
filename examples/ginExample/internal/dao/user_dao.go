package dao

import (
	"database/sql"
	"time"

	"github.com/zzguang83325/eorm"
	"github.com/zzguang83325/eorm/examples/ginExample/internal/model"
)

type UserDAO struct{}

// NewUserDAO 创建 DAO 实例
func NewUserDAO() *UserDAO {
	return &UserDAO{}
}

// Executor 是一个接口，兼容 *eorm.DB 和 *eorm.Tx
// eorm 的 DB 和 Tx 都支持 Query, Exec 等方法，但 Tx 有 extra methods。
// 为了简单，我们这里使用具体的 *eorm.Tx 作为可选参数，或者利用 eorm 的链式能力。
// 最佳实践：DAO 方法应当无状态，接受一个执行上下文（tx or db）

// GetUserByUsername 查询用户
func (d *UserDAO) GetUserByUsername(username string) (*model.User, error) {
	// 使用 Stmt Cache 优化高频查询

	user := &model.User{}
	foundUser, err := user.Cache("user_by_username", 10*time.Second).FindFirst("username= ?", username)

	if err != nil {
		return nil, err
	}

	return foundUser, nil
}

// CreateUser 创建用户
func (d *UserDAO) CreateUser(username string) (int64, error) {
	record := eorm.NewRecord().Set("username", username).Set("balance", 0)
	return eorm.InsertRecord("users", record)
}

// UpdateBalance 更新余额 (支持事务)
// 如果 tx 不为 nil，使用事务执行；否则使用默认 DB
func (d *UserDAO) UpdateBalance(tx *eorm.Tx, userID int64, amount int) error {
	var executor interface {
		Exec(sql string, args ...interface{}) (sql.Result, error)
	}

	if tx != nil {
		executor = tx
	} else {
		executor = eorm.Use("default") // 或者 eorm.DefaultDB()
	}

	// 乐观锁更新：确保余额不为负（仅针对扣款）
	// UPDATE users SET balance = balance + amount WHERE id = ? AND balance + amount >= 0
	sql := "UPDATE users SET balance = balance + ? WHERE id = ?"
	if amount < 0 {
		sql += " AND balance + ? >= 0"
		_, err := executor.Exec(sql, amount, userID, amount)
		return err
	}

	_, err := executor.Exec(sql, amount, userID)
	return err
}

// AddPointLog 记录流水 (支持事务)
func (d *UserDAO) AddPointLog(tx *eorm.Tx, log *model.PointLog) error {
	record := eorm.NewRecord().
		Set("user_id", log.UserID).
		Set("amount", log.Amount).
		Set("reason", log.Reason)

	if tx != nil {
		_, err := tx.InsertRecord("point_logs", record)
		return err
	}

	_, err := eorm.InsertRecord("point_logs", record)
	return err
}

// GetAllUsers 获取所有用户列表
func (d *UserDAO) GetAllUsers() ([]*model.User, error) {
	// 使用原始 SQL 查询，避免 EORM 框架的查询问题
	// db := eorm.Use("default")

	// // 直接执行 SQL 查询
	// var users []*model.User

	// err := db.QueryToDbModel(&users, "SELECT id, username, balance, created_at, updated_at FROM users ORDER BY created_at DESC")
	// if err != nil {
	// 	fmt.Println(err)
	// 	return nil, err
	// }

	user := model.User{}

	return user.Find("", "")

}
