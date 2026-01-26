package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// redisCache implements CacheProvider using Redis
type redisCache struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisCache 创建一个新的 Redis 缓存提供者
// 参数说明：
//   - addr: Redis 服务器地址，格式 "host:port"
//   - username: 用户名（Redis 6.0+），为空则不使用
//   - password: 密码，为空则不使用
//   - db: 数据库编号（0-15）
//   - maxConnections: 可选参数，最大连接数。不传或传 0 则使用默认值（10 * CPU核心数）
//
// 使用示例：
//
//	// 使用默认连接池大小（适合中等并发）
//	cache, err := NewRedisCache("192.168.10.205:6379", "user", "pass", 2)
//
//	// 自定义连接池大小（适合高并发场景）
//	cache, err := NewRedisCache("192.168.10.205:6379", "user", "pass", 2, 200)
func NewRedisCache(addr, username, password string, db int, maxConnections ...int) (*redisCache, error) {
	opts := &redis.Options{
		Addr:     addr,
		Username: username,
		Password: password,
		DB:       db,
	}

	// 如果指定了最大连接数，则使用自定义值
	if len(maxConnections) > 0 && maxConnections[0] > 0 {
		poolSize := maxConnections[0]
		opts.PoolSize = poolSize
		// 设置合理的最小空闲连接数（约为最大连接数的 10-20%）
		opts.MinIdleConns = poolSize / 10
		if opts.MinIdleConns < 5 {
			opts.MinIdleConns = 5
		}
		// 设置获取连接超时时间（高并发场景建议设置更长的超时）
		opts.PoolTimeout = 5 * time.Second
	}

	return NewRedisCacheWithOptions(opts)
}

// NewRedisCacheWithOptions 创建一个新的 Redis 缓存提供者（支持自定义连接池配置）
// 适合高并发场景，可以自定义连接池大小、超时等参数
//
// 推荐的高并发配置示例：
//
//	opts := &redis.Options{
//	    Addr:         "192.168.10.205:6379",
//	    Password:     "123456",
//	    DB:           2,
//	    PoolSize:     200,              // 最大连接数（根据并发量调整）
//	    MinIdleConns: 20,               // 最小空闲连接
//	    PoolTimeout:  5 * time.Second,  // 获取连接超时
//	    IdleTimeout:  10 * time.Minute, // 空闲连接超时
//	}
//	cache, err := NewRedisCacheWithOptions(opts)
func NewRedisCacheWithOptions(opts *redis.Options) (*redisCache, error) {
	client := redis.NewClient(opts)

	rc := &redisCache{
		client: client,
		ctx:    context.Background(),
	}

	// 测试连接
	if err := rc.client.Ping(rc.ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %v", err)
	}

	return rc, nil
}

// CacheGet 从 Redis 获取缓存值
// 优化：直接返回 JSON 字节数组，避免字符串转换开销
func (r *redisCache) CacheGet(cacheRepositoryName, key string) (interface{}, bool) {
	fullKey := fmt.Sprintf("eorm:%s:%s", cacheRepositoryName, key)

	// 使用 Bytes() 而不是 Result()，避免字符串转换
	val, err := r.client.Get(r.ctx, fullKey).Bytes()
	if err == redis.Nil {
		return nil, false
	} else if err != nil {
		return nil, false
	}

	// 直接返回字节数组，让 convertCacheValue 处理反序列化
	// 这样可以避免双重序列化：string → []byte → object
	return val, true
}

func (r *redisCache) CacheSet(cacheRepositoryName, key string, value interface{}, ttl time.Duration) {
	fullKey := fmt.Sprintf("eorm:%s:%s", cacheRepositoryName, key)

	var data interface{} = value
	switch value.(type) {
	case string, []byte:
		data = value
	default:
		jsonData, err := json.Marshal(value)
		if err != nil {
			// 序列化失败，记录日志并跳过存储
			fmt.Printf("eorm: redis cache marshal failed, key=%s, error=%v\n", fullKey, err)
			return
		}
		data = jsonData
	}

	r.client.Set(r.ctx, fullKey, data, ttl)
}

func (r *redisCache) CacheDelete(cacheRepositoryName, key string) {
	if cacheRepositoryName == "" || key == "" {
		return // 忽略空参数
	}
	fullKey := fmt.Sprintf("eorm:%s:%s", cacheRepositoryName, key)
	r.client.Del(r.ctx, fullKey)
}

// CacheClearRepository 清空指定存储库的所有缓存
func (r *redisCache) CacheClearRepository(cacheRepositoryName string) {
	if cacheRepositoryName == "" {
		return // 忽略空字符串，避免误删除所有缓存
	}
	pattern := fmt.Sprintf("eorm:%s:*", cacheRepositoryName)
	iter := r.client.Scan(r.ctx, 0, pattern, 0).Iterator()
	for iter.Next(r.ctx) {
		r.client.Del(r.ctx, iter.Val())
	}
}

// ClearAll 清空所有 eorm 相关的缓存
func (r *redisCache) ClearAll() {
	pattern := "eorm:*"
	iter := r.client.Scan(r.ctx, 0, pattern, 0).Iterator()
	for iter.Next(r.ctx) {
		r.client.Del(r.ctx, iter.Val())
	}
}

func (r *redisCache) Status() map[string]interface{} {
	stats := make(map[string]interface{})
	stats["type"] = "RedisCache"
	stats["address"] = r.client.Options().Addr

	// 获取连接池统计信息（重要：用于监控高并发场景）
	poolStats := r.client.PoolStats()
	stats["pool_hits"] = poolStats.Hits              // 连接池命中次数
	stats["pool_misses"] = poolStats.Misses          // 连接池未命中次数
	stats["pool_timeouts"] = poolStats.Timeouts      // 获取连接超时次数
	stats["pool_total_conns"] = poolStats.TotalConns // 总连接数
	stats["pool_idle_conns"] = poolStats.IdleConns   // 空闲连接数
	stats["pool_stale_conns"] = poolStats.StaleConns // 过期连接数

	// 连接池配置信息
	opts := r.client.Options()
	stats["pool_size"] = opts.PoolSize                // 最大连接数
	stats["min_idle_conns"] = opts.MinIdleConns       // 最小空闲连接
	stats["pool_timeout"] = opts.PoolTimeout.String() // 获取连接超时

	info, err := r.client.Info(r.ctx, "memory").Result()
	if err == nil {
		stats["redis_info_memory"] = info
	}

	dbSize, err := r.client.DBSize(r.ctx).Result()
	if err == nil {
		stats["db_size"] = dbSize
	}

	return stats
}

// GetPoolStats 获取连接池统计信息（用于监控）
// 返回值说明：
//   - Hits: 从连接池成功获取连接的次数
//   - Misses: 需要创建新连接的次数
//   - Timeouts: 获取连接超时的次数（重要：如果这个值很高，说明需要增加 PoolSize）
//   - TotalConns: 当前总连接数
//   - IdleConns: 当前空闲连接数
//   - StaleConns: 过期连接数
func (r *redisCache) GetPoolStats() *redis.PoolStats {
	return r.client.PoolStats()
}
