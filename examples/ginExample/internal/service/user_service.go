package service

import (
	"errors"
	"fmt"

	"github.com/zzguang83325/eorm"
	"github.com/zzguang83325/eorm/examples/ginExample/internal/dao"
	"github.com/zzguang83325/eorm/examples/ginExample/internal/model"
)

type UserService struct {
	userDAO *dao.UserDAO
}

func NewUserService() *UserService {
	return &UserService{
		userDAO: dao.NewUserDAO(),
	}
}

// Register 模拟用户注册
func (s *UserService) Register(username string) (int64, error) {
	// 简单的检查是否存在
	existing, _ := s.userDAO.GetUserByUsername(username)
	if existing != nil {
		return 0, errors.New("username already exists")
	}
	return s.userDAO.CreateUser(username)
}

// UserCheckIn 用户签到 (核心业务)
// 签到成功后增加积分，并记录流水
// 这是一个原子操作，必须在事务中完成
func (s *UserService) UserCheckIn(userID int64) error {
	// 1. 定义奖励积分
	rewardPoints := 10

	// 2. 开启事务
	// eorm.Transaction 自动处理了 Begin / Commit / Rollback
	return eorm.Transaction(func(tx *eorm.Tx) error {
		// 2.1 更新用户余额
		// 直接复用 DAO 方法，传入 tx
		err := s.userDAO.UpdateBalance(tx, userID, rewardPoints)
		if err != nil {
			return fmt.Errorf("failed to update balance: %v", err)
		}

		// 2.2 记录积分流水
		log := &model.PointLog{
			UserID: userID,
			Amount: int64(rewardPoints),
			Reason: "Daily Check-in Reward",
		}
		err = s.userDAO.AddPointLog(tx, log)
		if err != nil {
			return fmt.Errorf("failed to log points: %v", err)
		}

		// 返回 nil 则自动 Commit，返回 error 则自动 Rollback
		return nil
	})
}

// GetUserInfo 获取用户详情
func (s *UserService) GetUserInfo(username string) (*model.User, error) {
	return s.userDAO.GetUserByUsername(username)
}

// GetAllUsers 获取所有用户列表
func (s *UserService) GetAllUsers() ([]*model.User, error) {
	return s.userDAO.GetAllUsers()
}
