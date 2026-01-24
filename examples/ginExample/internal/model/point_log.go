package model

import (
	"time"
	"github.com/zzguang83325/eorm"
)

// PointLog represents the point_logs table
type PointLog struct {
	eorm.ModelCache `column:"-"`
	ID int64 `column:"id" json:"id"`
	UserID int64 `column:"user_id" json:"user_id"`
	Amount int64 `column:"amount" json:"amount"` // 变动金额
	Reason string `column:"reason" json:"reason"` // 变动原因
	CreatedAt *time.Time `column:"created_at" json:"created_at"`
}

// TableName returns the table name for PointLog struct
func (m *PointLog) TableName() string {
	return "point_logs"
}

// DatabaseName returns the database name for PointLog struct
func (m *PointLog) DatabaseName() string {
	return "default"
}

// Cache sets the cache name and TTL for the next query
func (m *PointLog) Cache(cacheRepositoryName string, ttl ...time.Duration) *PointLog {
	m.SetCache(cacheRepositoryName, ttl...)
	return m
}

// WithCountCache 设置分页计数缓存时间，支持链式调用
func (m *PointLog) WithCountCache(ttl time.Duration) *PointLog {
	m.ModelCache.WithCountCache(ttl)
	return m
}

// ToJson converts PointLog to a JSON string
func (m *PointLog) ToJson() string {
	return eorm.ToJson(m)
}

// Save saves the PointLog record (insert or update)
func (m *PointLog) Save() (int64, error) {
	return eorm.SaveDbModel(m)
}

// Insert inserts the PointLog record
func (m *PointLog) Insert() (int64, error) {
	return eorm.InsertDbModel(m)
}

// Update updates the PointLog record based on its primary key
func (m *PointLog) Update() (int64, error) {
	return eorm.UpdateDbModel(m)
}

// Delete deletes the PointLog record based on its primary key
func (m *PointLog) Delete() (int64, error) {
	return eorm.DeleteDbModel(m)
}

// ForceDelete performs a physical delete, bypassing soft delete
func (m *PointLog) ForceDelete() (int64, error) {
	return eorm.ForceDeleteModel(m)
}

// Restore restores a soft-deleted record
func (m *PointLog) Restore() (int64, error) {
	return eorm.RestoreModel(m)
}

// FindFirst finds the first PointLog record based on conditions
func (m *PointLog) FindFirst(whereSql string, args ...interface{}) (*PointLog, error) {
	result := &PointLog{}
	return eorm.FindFirstModel(result, m.GetCache(), whereSql, args...)
}

// Find finds PointLog records based on conditions
func (m *PointLog) Find(whereSql string, orderBySql string, args ...interface{}) ([]*PointLog, error) {
	return eorm.FindModel[*PointLog](m, m.GetCache(), whereSql, orderBySql, args...)
}

// FindWithTrashed finds PointLog records including soft-deleted ones
func (m *PointLog) FindWithTrashed(whereSql string, orderBySql string, args ...interface{}) ([]*PointLog, error) {
	return eorm.FindModelWithTrashed[*PointLog](m, m.GetCache(), whereSql, orderBySql, args...)
}

// FindOnlyTrashed finds only soft-deleted PointLog records
func (m *PointLog) FindOnlyTrashed(whereSql string, orderBySql string, args ...interface{}) ([]*PointLog, error) {
	return eorm.FindModelOnlyTrashed[*PointLog](m, m.GetCache(), whereSql, orderBySql, args...)
}

// PaginateBuilder paginates PointLog records based on conditions (traditional method)
func (m *PointLog) PaginateBuilder(page int, pageSize int, whereSql string, orderBy string, args ...interface{}) (*eorm.Page[*PointLog], error) {
	return eorm.PaginateModel[*PointLog](m, m.GetCache(), page, pageSize, whereSql, orderBy, args...)
}

// Paginate paginates PointLog records using complete SQL statement (recommended)
// Uses complete SQL statement for pagination query, automatically parses SQL and generates corresponding pagination statements based on database type
func (m *PointLog) Paginate(page int, pageSize int, fullSQL string, args ...interface{}) (*eorm.Page[*PointLog], error) {
	return eorm.PaginateModel_FullSql[*PointLog](m, m.GetCache(), page, pageSize, fullSQL, args...)
}

