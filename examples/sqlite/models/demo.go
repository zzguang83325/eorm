package models

import (
	"time"
	"github.com/zzguang83325/eorm"
)

// Demo represents the demo table
type Demo struct {
	eorm.ModelCache `column:"-"`
	ID int64 `column:"id" json:"id"`
	Name *string `column:"name" json:"name"`
	Age *int64 `column:"age" json:"age"`
	Salary *float64 `column:"salary" json:"salary"`
	IsActive *int64 `column:"is_active" json:"is_active"`
	Birthday *string `column:"birthday" json:"birthday"`
	CreatedAt *time.Time `column:"created_at" json:"created_at"`
	Metadata *string `column:"metadata" json:"metadata"`
}

// TableName returns the table name for Demo struct
func (m *Demo) TableName() string {
	return "demo"
}

// DatabaseName returns the database name for Demo struct
func (m *Demo) DatabaseName() string {
	return "sqlite"
}

// Cache sets the cache name and TTL for the next query
func (m *Demo) Cache(cacheRepositoryName string, ttl ...time.Duration) *Demo {
	m.SetCache(cacheRepositoryName, ttl...)
	return m
}

// WithCountCache 设置分页计数缓存时间，支持链式调用
func (m *Demo) WithCountCache(ttl time.Duration) *Demo {
	m.ModelCache.WithCountCache(ttl)
	return m
}

// ToJson converts Demo to a JSON string
func (m *Demo) ToJson() string {
	return eorm.ToJson(m)
}

// Save saves the Demo record (insert or update)
func (m *Demo) Save() (int64, error) {
	return eorm.SaveDbModel(m)
}

// Insert inserts the Demo record
func (m *Demo) Insert() (int64, error) {
	return eorm.InsertDbModel(m)
}

// Update updates the Demo record based on its primary key
func (m *Demo) Update() (int64, error) {
	return eorm.UpdateDbModel(m)
}

// Delete deletes the Demo record based on its primary key
func (m *Demo) Delete() (int64, error) {
	return eorm.DeleteDbModel(m)
}

// ForceDelete performs a physical delete, bypassing soft delete
func (m *Demo) ForceDelete() (int64, error) {
	return eorm.ForceDeleteModel(m)
}

// Restore restores a soft-deleted record
func (m *Demo) Restore() (int64, error) {
	return eorm.RestoreModel(m)
}

// FindFirst finds the first Demo record based on conditions
func (m *Demo) FindFirst(whereSql string, args ...interface{}) (*Demo, error) {
	result := &Demo{}
	return eorm.FindFirstModel(result, m.GetCache(), whereSql, args...)
}

// Find finds Demo records based on conditions
func (m *Demo) Find(whereSql string, orderBySql string, args ...interface{}) ([]*Demo, error) {
	return eorm.FindModel[*Demo](m, m.GetCache(), whereSql, orderBySql, args...)
}

// FindWithTrashed finds Demo records including soft-deleted ones
func (m *Demo) FindWithTrashed(whereSql string, orderBySql string, args ...interface{}) ([]*Demo, error) {
	return eorm.FindModelWithTrashed[*Demo](m, m.GetCache(), whereSql, orderBySql, args...)
}

// FindOnlyTrashed finds only soft-deleted Demo records
func (m *Demo) FindOnlyTrashed(whereSql string, orderBySql string, args ...interface{}) ([]*Demo, error) {
	return eorm.FindModelOnlyTrashed[*Demo](m, m.GetCache(), whereSql, orderBySql, args...)
}

// PaginateBuilder paginates Demo records based on conditions (traditional method)
func (m *Demo) PaginateBuilder(page int, pageSize int, whereSql string, orderBy string, args ...interface{}) (*eorm.Page[*Demo], error) {
	return eorm.PaginateModel[*Demo](m, m.GetCache(), page, pageSize, whereSql, orderBy, args...)
}

// Paginate paginates Demo records using complete SQL statement (recommended)
// Uses complete SQL statement for pagination query, automatically parses SQL and generates corresponding pagination statements based on database type
func (m *Demo) Paginate(page int, pageSize int, fullSQL string, args ...interface{}) (*eorm.Page[*Demo], error) {
	return eorm.PaginateModel_FullSql[*Demo](m, m.GetCache(), page, pageSize, fullSQL, args...)
}

