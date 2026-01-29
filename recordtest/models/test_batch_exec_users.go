package models

import (
	"time"
	"github.com/zzguang83325/eorm"
)

// TestBatchExecUsers represents the test_batch_exec_users table
type TestBatchExecUsers struct {
	eorm.ModelCache `column:"-"`
	ID int64 `column:"id" json:"id"`
	Name string `column:"name" json:"name"`
	Age *int64 `column:"age" json:"age"`
	Email *string `column:"email" json:"email"`
	Status *string `column:"status" json:"status"`
}

// TableName returns the table name for TestBatchExecUsers struct
func (m *TestBatchExecUsers) TableName() string {
	return "test_batch_exec_users"
}

// DatabaseName returns the database name for TestBatchExecUsers struct
func (m *TestBatchExecUsers) DatabaseName() string {
	return "default"
}

// Cache sets the cache name and TTL for the next query
func (m *TestBatchExecUsers) Cache(cacheRepositoryName string, ttl ...time.Duration) *TestBatchExecUsers {
	m.SetCache(cacheRepositoryName, ttl...)
	return m
}

// WithCountCache 设置分页计数缓存时间，支持链式调用
func (m *TestBatchExecUsers) WithCountCache(ttl time.Duration) *TestBatchExecUsers {
	m.ModelCache.WithCountCache(ttl)
	return m
}

// ToJson converts TestBatchExecUsers to a JSON string
func (m *TestBatchExecUsers) ToJson() string {
	return eorm.ToJson(m)
}

// Save saves the TestBatchExecUsers record (insert or update)
func (m *TestBatchExecUsers) Save() (int64, error) {
	return eorm.SaveDbModel(m)
}

// Insert inserts the TestBatchExecUsers record
func (m *TestBatchExecUsers) Insert() (int64, error) {
	return eorm.InsertDbModel(m)
}

// Update updates the TestBatchExecUsers record based on its primary key
func (m *TestBatchExecUsers) Update() (int64, error) {
	return eorm.UpdateDbModel(m)
}

// Delete deletes the TestBatchExecUsers record based on its primary key
func (m *TestBatchExecUsers) Delete() (int64, error) {
	return eorm.DeleteDbModel(m)
}

// ForceDelete performs a physical delete, bypassing soft delete
func (m *TestBatchExecUsers) ForceDelete() (int64, error) {
	return eorm.ForceDeleteModel(m)
}

// Restore restores a soft-deleted record
func (m *TestBatchExecUsers) Restore() (int64, error) {
	return eorm.RestoreModel(m)
}

// FindFirst finds the first TestBatchExecUsers record based on conditions
func (m *TestBatchExecUsers) FindFirst(whereSql string, args ...interface{}) (*TestBatchExecUsers, error) {
	result := &TestBatchExecUsers{}
	return eorm.FindFirstModel(result, m.GetCache(), whereSql, args...)
}

// Find finds TestBatchExecUsers records based on conditions
func (m *TestBatchExecUsers) Find(whereSql string, orderBySql string, args ...interface{}) ([]*TestBatchExecUsers, error) {
	return eorm.FindModel[*TestBatchExecUsers](m, m.GetCache(), whereSql, orderBySql, args...)
}

// FindWithTrashed finds TestBatchExecUsers records including soft-deleted ones
func (m *TestBatchExecUsers) FindWithTrashed(whereSql string, orderBySql string, args ...interface{}) ([]*TestBatchExecUsers, error) {
	return eorm.FindModelWithTrashed[*TestBatchExecUsers](m, m.GetCache(), whereSql, orderBySql, args...)
}

// FindOnlyTrashed finds only soft-deleted TestBatchExecUsers records
func (m *TestBatchExecUsers) FindOnlyTrashed(whereSql string, orderBySql string, args ...interface{}) ([]*TestBatchExecUsers, error) {
	return eorm.FindModelOnlyTrashed[*TestBatchExecUsers](m, m.GetCache(), whereSql, orderBySql, args...)
}

// PaginateBuilder paginates TestBatchExecUsers records based on conditions (traditional method)
func (m *TestBatchExecUsers) PaginateBuilder(page int, pageSize int, whereSql string, orderBy string, args ...interface{}) (*eorm.Page[*TestBatchExecUsers], error) {
	return eorm.PaginateModel[*TestBatchExecUsers](m, m.GetCache(), page, pageSize, whereSql, orderBy, args...)
}

// Paginate paginates TestBatchExecUsers records using complete SQL statement (recommended)
// Uses complete SQL statement for pagination query, automatically parses SQL and generates corresponding pagination statements based on database type
func (m *TestBatchExecUsers) Paginate(page int, pageSize int, fullSQL string, args ...interface{}) (*eorm.Page[*TestBatchExecUsers], error) {
	return eorm.PaginateModel_FullSql[*TestBatchExecUsers](m, m.GetCache(), page, pageSize, fullSQL, args...)
}

