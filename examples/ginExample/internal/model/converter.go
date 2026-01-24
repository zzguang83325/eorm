package model

import (
	"github.com/zzguang83325/eorm"
)

func ToUser(r *eorm.Record) *User {
	if r == nil {
		return nil
	}

	balance := int64(r.Int("balance"))
	createdAt := r.Time("created_at")
	updatedAt := r.Time("updated_at")

	// 注意：自动生成的 User 结构体使用了指针
	return &User{
		ID:        r.Int64("id"),
		Username:  r.Str("username"),
		Balance:   &balance,
		CreatedAt: &createdAt,
		UpdatedAt: &updatedAt,
	}
}
