package models

import (
	"time"
	"github.com/zzguang83325/eorm"
)

// Product represents the products table
type Product struct {
	eorm.ModelCache `column:"-"`
	ID int64 `column:"id" json:"id"`
	Name string `column:"name" json:"name"`
	Price *float64 `column:"price" json:"price"`
	Stock *int64 `column:"stock" json:"stock"`
	CreatedAt *time.Time `column:"created_at" json:"created_at"`
}

// TableName returns the table name for Product struct
func (m *Product) TableName() string {
	return "products"
}

// DatabaseName returns the database name for Product struct
func (m *Product) DatabaseName() string {
	return "default"
}

// Cache sets the cache name and TTL for the next query
func (m *Product) Cache(cacheRepositoryName string, ttl ...time.Duration) *Product {
	m.SetCache(cacheRepositoryName, ttl...)
	return m
}

// WithCountCache 设置分页计数缓存时间，支持链式调用
func (m *Product) WithCountCache(ttl time.Duration) *Product {
	m.ModelCache.WithCountCache(ttl)
	return m
}

// ToJson converts Product to a JSON string
func (m *Product) ToJson() string {
	return eorm.ToJson(m)
}

// Save saves the Product record (insert or update)
func (m *Product) Save() (int64, error) {
	return eorm.SaveDbModel(m)
}

// Insert inserts the Product record
func (m *Product) Insert() (int64, error) {
	return eorm.InsertDbModel(m)
}

// Update updates the Product record based on its primary key
func (m *Product) Update() (int64, error) {
	return eorm.UpdateDbModel(m)
}

// Delete deletes the Product record based on its primary key
func (m *Product) Delete() (int64, error) {
	return eorm.DeleteDbModel(m)
}

// ForceDelete performs a physical delete, bypassing soft delete
func (m *Product) ForceDelete() (int64, error) {
	return eorm.ForceDeleteModel(m)
}

// Restore restores a soft-deleted record
func (m *Product) Restore() (int64, error) {
	return eorm.RestoreModel(m)
}

// FindFirst finds the first Product record based on conditions
func (m *Product) FindFirst(whereSql string, args ...interface{}) (*Product, error) {
	result := &Product{}
	return eorm.FindFirstModel(result, m.GetCache(), whereSql, args...)
}

// Find finds Product records based on conditions
func (m *Product) Find(whereSql string, orderBySql string, args ...interface{}) ([]*Product, error) {
	return eorm.FindModel[*Product](m, m.GetCache(), whereSql, orderBySql, args...)
}

// FindWithTrashed finds Product records including soft-deleted ones
func (m *Product) FindWithTrashed(whereSql string, orderBySql string, args ...interface{}) ([]*Product, error) {
	return eorm.FindModelWithTrashed[*Product](m, m.GetCache(), whereSql, orderBySql, args...)
}

// FindOnlyTrashed finds only soft-deleted Product records
func (m *Product) FindOnlyTrashed(whereSql string, orderBySql string, args ...interface{}) ([]*Product, error) {
	return eorm.FindModelOnlyTrashed[*Product](m, m.GetCache(), whereSql, orderBySql, args...)
}

// PaginateBuilder paginates Product records based on conditions (traditional method)
func (m *Product) PaginateBuilder(page int, pageSize int, whereSql string, orderBy string, args ...interface{}) (*eorm.Page[*Product], error) {
	return eorm.PaginateModel[*Product](m, m.GetCache(), page, pageSize, whereSql, orderBy, args...)
}

// Paginate paginates Product records using complete SQL statement (recommended)
// Uses complete SQL statement for pagination query, automatically parses SQL and generates corresponding pagination statements based on database type
func (m *Product) Paginate(page int, pageSize int, fullSQL string, args ...interface{}) (*eorm.Page[*Product], error) {
	return eorm.PaginateModel_FullSql[*Product](m, m.GetCache(), page, pageSize, fullSQL, args...)
}

