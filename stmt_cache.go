package eorm

import (
	"container/list"
	"database/sql"
	"sync"
	"time"
)

// StmtCacheConfig 预编译语句缓存配置
type StmtCacheConfig struct {
	Enabled         bool          // 是否启用缓存（默认 true）
	MaxSize         int           // 最大缓存数量（默认 1000，0 表示无限制）
	BaseTTL         time.Duration // 基础 TTL（默认使用 ConnMaxLifetime * 0.8）
	Strategy        string        // 淘汰策略："lru" | "ttl"（默认 "lru"）
	CleanupInterval time.Duration // 定时清理间隔（0 表示禁用，仅惰性删除；> 0 启用后台清理）
}

// DefaultStmtCacheConfig 返回默认配置
func DefaultStmtCacheConfig() StmtCacheConfig {
	return StmtCacheConfig{
		Enabled:         true,
		MaxSize:         1000,
		Strategy:        "lru",
		CleanupInterval: 0, // 默认禁用定时清理，仅惰性删除
	}
}

// stmtCacheEntry 增强的缓存条目
type stmtCacheEntry struct {
	stmt        *sql.Stmt
	sql         string
	createdAt   time.Time
	lastUsedAt  time.Time
	accessCount int64
	listElement *list.Element // LRU 链表元素（用于快速删除）
}

// stmtCache LRU 缓存实现
type stmtCache struct {
	config  StmtCacheConfig
	mu      sync.RWMutex
	items   map[string]*stmtCacheEntry // key -> entry
	lruList *list.List                 // LRU 双向链表

	// 统计指标
	hits      int64
	misses    int64
	evictions int64
}

// newStmtCache 创建新的语句缓存
func newStmtCache(config StmtCacheConfig) *stmtCache {
	if config.MaxSize <= 0 {
		config.MaxSize = 1000 // 默认上限
	}
	if config.Strategy == "" {
		config.Strategy = "lru"
	}

	return &stmtCache{
		config:  config,
		items:   make(map[string]*stmtCacheEntry),
		lruList: list.New(),
	}
}

// Get 获取缓存的语句
func (c *stmtCache) Get(key string) (*sql.Stmt, bool) {
	if !c.config.Enabled {
		return nil, false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.items[key]
	if !exists {
		c.misses++
		return nil, false
	}

	// 检查是否过期（可选的安全网机制）
	// 注意：BaseTTL = 0 表示禁用 TTL，完全依赖 LRU 淘汰
	if c.config.BaseTTL > 0 && time.Since(entry.createdAt) > c.config.BaseTTL {
		// 过期，删除并关闭
		c.removeEntry(key, entry)
		c.misses++
		return nil, false
	}

	// 更新访问信息
	entry.lastUsedAt = time.Now()
	entry.accessCount++

	// 移动到 LRU 链表头部（最近使用）
	if c.config.Strategy == "lru" && entry.listElement != nil {
		c.lruList.MoveToFront(entry.listElement)
	}

	c.hits++
	return entry.stmt, true
}

// Set 添加或更新缓存
func (c *stmtCache) Set(key string, stmt *sql.Stmt, query string) {
	if !c.config.Enabled {
		return
	}

	if key == "" {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 如果已存在，先删除旧的
	if oldEntry, exists := c.items[key]; exists {
		c.removeEntry(key, oldEntry)
	}

	// 检查容量，必要时淘汰
	if c.config.MaxSize > 0 && len(c.items) >= c.config.MaxSize {
		c.evictOne()
	}

	// 创建新条目
	now := time.Now()
	entry := &stmtCacheEntry{
		stmt:        stmt,
		sql:         query,
		createdAt:   now,
		lastUsedAt:  now,
		accessCount: 0,
	}

	// 添加到 LRU 链表头部
	if c.config.Strategy == "lru" {
		entry.listElement = c.lruList.PushFront(key)
	}

	c.items[key] = entry
}

// Delete 删除指定缓存
func (c *stmtCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, exists := c.items[key]; exists {
		c.removeEntry(key, entry)
	}
}

// Clear 清空所有缓存
func (c *stmtCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 关闭所有语句
	for key, entry := range c.items {
		if entry.stmt != nil {
			entry.stmt.Close()
		}
		delete(c.items, key)
	}

	c.lruList.Init()
	c.evictions += int64(len(c.items))
}

// Stats 返回缓存统计信息
func (c *stmtCache) Stats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.hits + c.misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(c.hits) / float64(total) * 100
	}

	return map[string]interface{}{
		"enabled":   c.config.Enabled,
		"size":      len(c.items),
		"max_size":  c.config.MaxSize,
		"hits":      c.hits,
		"misses":    c.misses,
		"hit_rate":  hitRate,
		"evictions": c.evictions,
		"strategy":  c.config.Strategy,
	}
}

// evictOne 淘汰一个条目（内部方法，需持锁调用）
func (c *stmtCache) evictOne() {
	if c.config.Strategy == "lru" {
		// LRU：淘汰链表尾部（最久未使用）
		if elem := c.lruList.Back(); elem != nil {
			key := elem.Value.(string)
			if entry, exists := c.items[key]; exists {
				c.removeEntry(key, entry)
				c.evictions++
			}
		}
	} else {
		// TTL：淘汰最早创建的
		var oldestKey string
		var oldestTime time.Time

		for key, entry := range c.items {
			if oldestKey == "" || entry.createdAt.Before(oldestTime) {
				oldestKey = key
				oldestTime = entry.createdAt
			}
		}

		if oldestKey != "" {
			if entry, exists := c.items[oldestKey]; exists {
				c.removeEntry(oldestKey, entry)
				c.evictions++
			}
		}
	}
}

// removeEntry 移除条目（内部方法，需持锁调用）
func (c *stmtCache) removeEntry(key string, entry *stmtCacheEntry) {
	// 关闭语句
	if entry.stmt != nil {
		entry.stmt.Close()
	}

	// 从 LRU 链表移除
	if entry.listElement != nil {
		c.lruList.Remove(entry.listElement)
	}

	// 从 map 删除
	delete(c.items, key)
}
