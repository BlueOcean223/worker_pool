package worker_pool

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// WorkerPool 线程池结构体
type WorkerPool struct {
	// 全局任务队列
	taskQueue chan Task
	// 工作者管理
	workers      map[int]*Worker
	workersMutex sync.RWMutex
	// 工作者ID生成器
	nextWorkerID atomic.Int32
	running      atomic.Int32
	activeJobs   atomic.Int32
	// 生命周期管理
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// 性能统计
	totalTasks     atomic.Int64
	completedTasks atomic.Int64
}

// NewWorkerPool 创建新的线程池
func NewWorkerPool() *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	config := GetGlobalConfig()

	pool := &WorkerPool{
		taskQueue: make(chan Task, config.QueueSize),
		workers:   make(map[int]*Worker),
		ctx:       ctx,
		cancel:    cancel,
	}

	return pool
}

// Start 启动线程池
func (p *WorkerPool) Start() error {
	if !p.running.CompareAndSwap(0, 1) {
		return fmt.Errorf("worker pool is already running")
	}

	config := GetGlobalConfig()
	// 启动核心工作者
	for i := 0; i < config.CoreWorkers; i++ {
		p.addWorker(true)
	}

	return nil
}

// Submit 提交任务
func (p *WorkerPool) Submit(task Task) error {
	if p.running.Load() == 0 {
		return fmt.Errorf("worker pool is not running")
	}

	p.totalTasks.Add(1)

	// 尝试随机分发到本地队列
	if p.distributeToLocalQueue(task) {
		p.activeJobs.Add(1)
		return nil
	}

	// 本地队列都满了，发送到全局队列
	select {
	case p.taskQueue <- task:
		p.activeJobs.Add(1)
		return nil
	case <-p.ctx.Done():
		return fmt.Errorf("worker pool is shutting down")
	default:
		// 全局队列满了，先尝试扩容
		if p.shouldExpand() {
			p.addWorker(false)
			// 重新尝试提交到全局队列或分发到其他工作者
			if p.distributeToLocalQueue(task) {
				p.activeJobs.Add(1)
				return nil
			}
			// 如果分发失败，再次尝试全局队列
			select {
			case p.taskQueue <- task:
				p.activeJobs.Add(1)
				return nil
			default:
				// 如果还是失败，丢弃任务
				p.totalTasks.Add(-1)
				return fmt.Errorf("task queue is full even after expansion")
			}
		}
		// 扩容失败，再次尝试提交到全局队列
		select {
		case p.taskQueue <- task:
			p.activeJobs.Add(1)
			return nil
		default:
			// 如果还是满了，则放弃该任务
			p.totalTasks.Add(-1)
			return fmt.Errorf("task queue is full and cannot expand")
		}
	}
}

// distributeToLocalQueue 随机分发任务到本地队列
func (p *WorkerPool) distributeToLocalQueue(task Task) bool {
	p.workersMutex.RLock()
	defer p.workersMutex.RUnlock()

	if len(p.workers) == 0 {
		return false
	}

	// 随机选择一个worker
	workerIDs := make([]int, 0, len(p.workers))
	for id := range p.workers {
		workerIDs = append(workerIDs, id)
	}

	// 随机打乱workerID列表
	for i := len(workerIDs) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		workerIDs[i], workerIDs[j] = workerIDs[j], workerIDs[i]
	}

	// 尝试分发到第一个有空闲位置且没有准备停止的worker
	for _, id := range workerIDs {
		if worker := p.workers[id]; worker != nil && !worker.stopped {
			if worker.addLocalTask(task) {
				return true
			}
		}
	}

	return false
}

// addWorker 添加核心工作者
func (p *WorkerPool) addWorker(isCore bool) {
	p.workersMutex.Lock()
	defer p.workersMutex.Unlock()

	config := GetGlobalConfig()

	// 工作线程已达到上限，返回
	if len(p.workers) >= config.MaxWorkers {
		return
	}

	// 创建并启动新的工作者
	workerID := int(p.nextWorkerID.Add(1))
	worker := NewWorker(workerID, isCore, p.taskQueue, p.ctx, p)
	p.workers[workerID] = worker

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		worker.Start(func() {
			p.activeJobs.Add(-1)
			p.completedTasks.Add(1)
		})
		// 工作者结束后从map中移除
		p.workersMutex.Lock()
		delete(p.workers, workerID)
		p.workersMutex.Unlock()
	}()
}

// shouldExpand 判断是否应该扩容
func (p *WorkerPool) shouldExpand() bool {
	p.workersMutex.RLock()
	defer p.workersMutex.RUnlock()

	config := GetGlobalConfig()
	currentWorkers := len(p.workers)

	// 已达到最大线程数
	if currentWorkers >= config.MaxWorkers {
		return false
	}

	return true
}

// GetStats 获取线程池统计信息
func (p *WorkerPool) GetStats() Stats {
	p.workersMutex.RLock()
	defer p.workersMutex.RUnlock()

	coreWorkers := 0
	nonCoreWorkers := 0
	totalLocalQueue := 0
	for _, worker := range p.workers {
		if worker.IsCore {
			coreWorkers++
		} else {
			nonCoreWorkers++
		}
		totalLocalQueue += worker.getLocalQueueLength()
	}

	return Stats{
		CoreWorkers:      coreWorkers,
		NonCoreWorkers:   nonCoreWorkers,
		TotalWorkers:     len(p.workers),
		QueueLength:      len(p.taskQueue),
		LocalQueueLength: totalLocalQueue,
		ActiveJobs:       int(p.activeJobs.Load()),
		TotalTasks:       int(p.totalTasks.Load()),
		CompletedTasks:   int(p.completedTasks.Load()),
		IsRunning:        p.running.Load() == 1,
	}
}

// Shutdown 关闭线程池
func (p *WorkerPool) Shutdown(timeout time.Duration) error {
	if !p.running.CompareAndSwap(1, 0) {
		return fmt.Errorf("worker pool is not running")
	}

	// 停止接收新任务
	p.cancel()

	// 关闭任务队列
	close(p.taskQueue)

	// 等待所有工作者完成
	done := make(chan interface{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("shutdown timeout")
	}
}
