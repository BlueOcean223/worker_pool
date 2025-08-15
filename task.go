package worker_pool

import (
	"fmt"
	"time"
)

// Task 任务接口
type Task interface {
	Execute()
}

// FuncTask 函数任务
type FuncTask struct {
	Fn func()
}

// Execute 执行函数任务
func (ft *FuncTask) Execute() {
	if ft.Fn != nil {
		ft.Fn()
	}
}

// NewFuncTask 创建函数任务
func NewFuncTask(fn func()) Task {
	return &FuncTask{Fn: fn}
}

// TimedTask 带超时的任务
type TimedTask struct {
	task    Task
	timeout time.Duration
}

// Execute 执行带超时的任务
func (tt *TimedTask) Execute() {
	done := make(chan struct{})
	go func() {
		tt.task.Execute()
		close(done)
	}()

	select {
	case <-done:
		// 任务正常完成
		fmt.Println("Task completed successfully")
	case <-time.After(tt.timeout):
		// 任务超时
		fmt.Printf("Task timeout after %v\n", tt.timeout)
	}
}

// NewTimedTask 创建带超时的任务
func NewTimedTask(task Task, timeout time.Duration) Task {
	return &TimedTask{
		task:    task,
		timeout: timeout,
	}
}
