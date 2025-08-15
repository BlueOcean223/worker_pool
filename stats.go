package worker_pool

import "fmt"

// Stats 线程池统计信息
type Stats struct {
	// 核心工作者数量
	CoreWorkers int
	// 非核心工作者数量
	NonCoreWorkers int
	// 总工作者数量
	TotalWorkers int
	// 全局队列长度
	QueueLength int
	// 本地队列总长度
	LocalQueueLength int
	// 活跃任务数
	ActiveJobs int
	// 总任务数
	TotalTasks int
	// 已完成任务数
	CompletedTasks int
	// 是否运行中
	IsRunning bool
}

// String 返回统计信息的字符串表示
func (s Stats) String() string {
	completionRate := float64(0)
	if s.TotalTasks > 0 {
		completionRate = float64(s.CompletedTasks) / float64(s.TotalTasks) * 100
	}

	return fmt.Sprintf(
		"Stats{CoreWorkers: %d, NonCoreWorkers: %d, TotalWorkers: %d, QueueLength: %d, LocalQueueLength: %d, ActiveJobs: %d, TotalTasks: %d, CompletedTasks: %d, CompletionRate: %.1f%%, IsRunning: %t}",
		s.CoreWorkers, s.NonCoreWorkers, s.TotalWorkers, s.QueueLength, s.LocalQueueLength, s.ActiveJobs, s.TotalTasks, s.CompletedTasks, completionRate, s.IsRunning,
	)
}
