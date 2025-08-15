package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
	"worker_pool"
)

func main() {
	// 创建线程池
	pool := worker_pool.NewWorkerPool()

	// 启动线程池
	if err := pool.Start(); err != nil {
		fmt.Printf("Failed to start worker pool: %v\n", err)
		return
	}

	fmt.Println("Worker pool started")

	// 模拟任务提交
	var wg sync.WaitGroup

	// 阶段1：正常负载
	fmt.Println("\n=== Phase 1: Normal Load ===")
	for i := 0; i < 10; i++ {
		wg.Add(1)
		taskID := i
		task := worker_pool.NewFuncTask(func() {
			defer wg.Done()
			fmt.Printf("Executing normal task %d\n", taskID)
			time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
			fmt.Printf("Completed normal task %d\n", taskID)
		})

		if err := pool.Submit(task); err != nil {
			fmt.Printf("Failed to submit task %d: %v\n", taskID, err)
			wg.Done()
		}
		time.Sleep(100 * time.Millisecond)
	}

	// 打印统计信息
	time.Sleep(1 * time.Second)
	stats := pool.GetStats()
	fmt.Printf("Stats after normal load: %s\n", stats.String())
	time.Sleep(1 * time.Second)

	// 阶段2：高并发负载
	fmt.Println("\n=== Phase 2: High Concurrency Load ===")
	go func() {
		for i := 0; i < 260; i++ {
			wg.Add(1)
			taskID := i
			task := worker_pool.NewFuncTask(func() {
				defer wg.Done()
				fmt.Printf("Executing high-load task %d\n", taskID)
				time.Sleep(time.Duration(rand.Intn(5000)) * time.Millisecond)
				fmt.Printf("Completed high-load task %d\n", taskID)
			})

			if err := pool.Submit(task); err != nil {
				fmt.Printf("Failed to submit task %d: %v\n", taskID, err)
				wg.Done()
			}
		}
	}()

	go func() {
		for i := 0; i < 260; i++ {
			wg.Add(1)
			taskID := i + 260
			task := worker_pool.NewFuncTask(func() {
				defer wg.Done()
				fmt.Printf("Executing high-load task %d\n", taskID)
				time.Sleep(time.Duration(rand.Intn(5000)) * time.Millisecond)
				fmt.Printf("Completed high-load task %d\n", taskID)
			})

			if err := pool.Submit(task); err != nil {
				fmt.Printf("Failed to submit task %d: %v\n", taskID, err)
				wg.Done()
			}
		}
	}()

	// 打印统计信息
	time.Sleep(2 * time.Second)
	for i := 0; i < 30; i++ {
		stats = pool.GetStats()
		fmt.Printf("Stats during high load: %s\n", stats.String())
		time.Sleep(2 * time.Second)
	}

	// 等待所有任务完成
	wg.Wait()

	// 阶段3：等待缩容
	fmt.Println("\n=== Phase 3: Waiting for Scale Down ===")
	for i := 0; i < 10; i++ {
		time.Sleep(3 * time.Second)
		stats = pool.GetStats()
		fmt.Printf("Stats after %d seconds: %s\n", (i+1)*3, stats.String())
	}

	// 关闭线程池
	fmt.Println("\n=== Shutting Down ===")
	if err := pool.Shutdown(10 * time.Second); err != nil {
		fmt.Printf("Failed to shutdown worker pool: %v\n", err)
	} else {
		fmt.Println("Worker pool shutdown successfully")
	}
}
