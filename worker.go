package worker_pool

import (
	"context"
	"time"
)

// Worker 工作者结构体
type Worker struct {
	// 工作者ID
	ID int
	// 是否为核心工作者
	IsCore bool
	// 全局任务管道
	taskChan <-chan Task
	// 本地任务队列
	localQueue chan Task
	// 用于管理生命周期的上下文
	ctx context.Context
	// 停止
	stopped bool
	// 所属线程池
	pool *WorkerPool
}

// NewWorker 创建新的工作者
func NewWorker(id int, isCore bool, taskChan <-chan Task, ctx context.Context, pool *WorkerPool) *Worker {
	return &Worker{
		ID:         id,
		IsCore:     isCore,
		taskChan:   taskChan,
		localQueue: make(chan Task, GetGlobalConfig().LocalQueueSize),
		ctx:        ctx,
		pool:       pool,
	}
}

// Start 启动工作者
func (w *Worker) Start(onTaskComplete func()) {
	idleCheck := time.NewTicker(time.Second * 30)
	defer idleCheck.Stop()

	for {
		select {
		case task := <-w.localQueue:
			// 处理本地任务
			task.Execute()
			if onTaskComplete != nil {
				onTaskComplete()
			}

		case task, ok := <-w.taskChan:
			if !ok {
				// 任务通道已关闭
				// 退出前再次检查本地队列，防止任务丢失
				w.drainLocalQueue(onTaskComplete)
				return
			}
			// 直接执行全局任务
			task.Execute()
			if onTaskComplete != nil {
				onTaskComplete()
			}

		case <-w.ctx.Done():
			// 线程池正在关闭
			return

		case <-idleCheck.C:
			// 检查空闲状态
			if w.shouldExit() {
				// 先停止接收新任务
				w.Stop()
				// 确保本地队列中的所有任务都被处理
				w.drainLocalQueue(onTaskComplete)
				return
			}
		}
	}
}

// Stop 停止工作者
func (w *Worker) Stop() {
	w.stopped = true
}

// addLocalTask 添加任务到本地队列
func (w *Worker) addLocalTask(task Task) bool {
	select {
	case w.localQueue <- task:
		// 成功添加到本地队列
		return true
	default:
		// 本地队列满了，不阻塞，任务将由池的其他方式处理
		return false
	}
}

// getLocalQueueLength 获取本地队列长度
func (w *Worker) getLocalQueueLength() int {
	return len(w.localQueue)
}

// 检查工作者是否空闲超过一定时间
func (w *Worker) shouldExit() bool {
	// 核心线程不退出
	if w.IsCore {
		return false
	}

	// 本地队列不为空，不退出
	if w.getLocalQueueLength() > 0 {
		return false
	}

	// 全局任务数量超过上限的一半，不退出
	if len(w.taskChan) >= GetGlobalConfig().QueueSize/2 {
		return false
	}

	return true
}

// 退出前检查是否有剩余任务
func (w *Worker) drainLocalQueue(onTaskComplete func()) {
	for w.getLocalQueueLength() > 0 {
		task := <-w.localQueue
		select {
		case w.pool.taskQueue <- task:
			// 将任务分发到全局队列
		default:
			// 全局任务已满，立即执行任务
			task.Execute()
			if onTaskComplete != nil {
				onTaskComplete()
			}
		}
	}
}
