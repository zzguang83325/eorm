package eorm

import (
	"context"
	"sync"
	"time"
)

// DBPinger 定义数据库连接检查接口，便于测试
type DBPinger interface {
	Ping() error
	PingContext(ctx context.Context) error
}

// DBInfo 定义数据库信息接口
type DBInfo interface {
	GetName() string
}

// ConnectionMonitor 连接监控器
// 负责定时检查数据库连接状态，在连接断开时自动重连
type ConnectionMonitor struct {
	pinger         DBPinger      // 数据库连接检查器
	dbName         string        // 数据库名称
	normalInterval time.Duration // 正常检查间隔
	errorInterval  time.Duration // 故障检查间隔
	ticker         *time.Ticker  // 定时器
	stopCh         chan struct{} // 停止信号
	lastHealthy    bool          // 上次检查的健康状态（用于状态变化检测）
	mu             sync.RWMutex  // 读写锁
}

// 全局监控器管理
var (
	// monitors 存储所有数据库的监控器实例
	monitors = make(map[string]*ConnectionMonitor) // 数据库名 -> 监控器

	// monitorsMu 保护 monitors 映射的读写锁
	monitorsMu sync.RWMutex

	// globalLimitCh 全局并发限制信号量
	// 默认允许最多 5 个数据库同时进行 Ping 操作，避免慢库阻塞所有检查
	globalLimitCh = make(chan struct{}, 5)
)

// Stop 停止连接监控器
func (cm *ConnectionMonitor) Stop() {
	if cm == nil {
		return
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 发送停止信号
	select {
	case <-cm.stopCh:
		// 已经停止
		return
	default:
		close(cm.stopCh)
	}

	// 停止定时器
	if cm.ticker != nil {
		cm.ticker.Stop()
		cm.ticker = nil
	}
}

// checkConnection 检查数据库连接状态
func (cm *ConnectionMonitor) checkConnection() bool {
	// 获取限流许可
	select {
	case globalLimitCh <- struct{}{}:
		defer func() { <-globalLimitCh }()
	case <-cm.stopCh:
		return false
	default:
		// 如果限流已满，为了不阻塞监控协程，跳过本次检查
		// 等待下一个周期再检查，这样可以保证至少不会挂死
		return cm.lastHealthy
	}

	// 增加 Context 超时控制，防止 Ping 挂死
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var err error
	if pinger, ok := cm.pinger.(interface{ PingContext(context.Context) error }); ok {
		err = pinger.PingContext(ctx)
	} else {
		// 回边到不带 context 的 Ping
		done := make(chan error, 1)
		go func() {
			done <- cm.pinger.Ping()
		}()
		select {
		case err = <-done:
		case <-ctx.Done():
			err = ctx.Err()
		}
	}

	isHealthy := err == nil

	// 只在状态变化时记录日志
	cm.mu.Lock()
	if cm.lastHealthy != isHealthy {
		if isHealthy {
			LogConnectionRecovered(cm.dbName)
		} else {
			LogConnectionError(cm.dbName, err)
		}
		cm.lastHealthy = isHealthy
	}
	cm.mu.Unlock()

	return isHealthy
}

// LogConnectionError 记录连接错误日志（仅在检测到连接失败时记录）
func LogConnectionError(dbName string, err error) {
	LogError("数据库连接失败", NewRecord().
		Set("database", dbName).
		Set("error", err.Error()).
		Set("time", time.Now()))
}

// LogConnectionRecovered 记录连接恢复日志（仅在连接从失败状态恢复时记录）
func LogConnectionRecovered(dbName string) {
	LogInfo("数据库连接已恢复", NewRecord().
		Set("database", dbName).
		Set("time", time.Now()))
}

// run 监控器主循环
// 负责定时检查连接状态并动态调整检查频率
func (cm *ConnectionMonitor) run() {
	defer func() {
		cm.mu.Lock()
		if cm.ticker != nil {
			cm.ticker.Stop()
			cm.ticker = nil
		}
		cm.mu.Unlock()
	}()

	// 使用正常间隔开始检查
	currentInterval := cm.normalInterval

	cm.mu.Lock()
	cm.ticker = time.NewTicker(currentInterval)
	cm.mu.Unlock()

	for {
		cm.mu.RLock()
		ticker := cm.ticker
		cm.mu.RUnlock()

		if ticker == nil {
			return
		}

		select {
		case <-cm.stopCh:
			return
		case <-ticker.C:
			isHealthy := cm.checkConnection()

			// 根据连接状态调整检查间隔
			var newInterval time.Duration
			if isHealthy {
				newInterval = cm.normalInterval
			} else {
				newInterval = cm.errorInterval
			}

			// 如果间隔需要调整，重置定时器
			if newInterval != currentInterval {
				cm.mu.Lock()
				if cm.ticker != nil {
					cm.ticker.Stop()
					cm.ticker = time.NewTicker(newInterval)
					currentInterval = newInterval
				}
				cm.mu.Unlock()
			}
		}
	}
}

// Start 启动连接监控器
func (cm *ConnectionMonitor) Start() {
	if cm == nil {
		return
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 检查是否已经有 ticker 在运行
	if cm.ticker != nil {
		// 已经在运行，不需要重复启动
		return
	}

	// 确保 stopCh 是开放的
	select {
	case <-cm.stopCh:
		// stopCh 已关闭，需要重新创建
		cm.stopCh = make(chan struct{})
	default:
		// stopCh 是开放的，可以使用
	}

	// 启动监控 goroutine
	go cm.run()
}

// cleanupMonitor 清理指定数据库的监控器
func cleanupMonitor(dbName string) {
	monitorsMu.Lock()
	defer monitorsMu.Unlock()

	if monitor, exists := monitors[dbName]; exists {
		monitor.Stop()
		delete(monitors, dbName)
	}
}
