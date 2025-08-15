package worker_pool

import (
	"sync"
	"time"
)

var (
	globalConfig *Config
	once         sync.Once
)

// Config 线程池配置
type Config struct {
	// 核心工作者数量
	CoreWorkers int
	// 最大工作者数量
	MaxWorkers int
	// 全局任务队列大小
	QueueSize int
	// 本地任务队列大小
	LocalQueueSize int
	// 监控间隔
	MonitorInterval time.Duration
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		CoreWorkers:     5,
		MaxWorkers:      20,
		QueueSize:       100,
		LocalQueueSize:  10,
		MonitorInterval: 5 * time.Second,
	}
}

// initGlobalConfig 初始化全局配置（只执行一次）
func initGlobalConfig() {
	globalConfig = DefaultConfig()
}

// GetGlobalConfig 获取全局配置
func GetGlobalConfig() *Config {
	once.Do(initGlobalConfig)
	return globalConfig
}
