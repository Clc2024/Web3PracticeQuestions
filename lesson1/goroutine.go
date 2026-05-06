package main

import (
	"fmt"
	"sync"
	"time"
)

// printOdd 打印1到10的奇数
func printOdd(wg *sync.WaitGroup) {
	defer wg.Done() //协程结束时通知WaitGroup计数-1
	fmt.Println("=== 奇数写成开始执行===")
	for i := 1; i <= 10; i += 2 {
		fmt.Printf("奇数：%d\n", i)

	}
	fmt.Printf("=== 奇数协程执行完成 ===")

}

// printEven 打印2到10的偶数
func printEven(wg *sync.WaitGroup) {
	defer wg.Done() //协程结束时通知WaitGroup计数-1
	fmt.Println("=== 偶数写成开始执行===")
	for i := 2; i <= 10; i += 2 {
		fmt.Printf("偶数：%d\n", i)

	}
	fmt.Printf("=== 偶数协程执行完成 ===")

}

// Task定义任务类型：接受空参数，返回空（通用任务函数签名）
type Task func()

// TaskResult 存储单个任务的执行结果（包含执行时间）
type TaskResult struct {
	TaskID    int
	StartTime time.Time
	EndTime   time.Time
	Elapsed   time.Duration //执行耗时
}

// Scheduler任务调度器结构体：封装调度逻辑和结果
type Scheduler struct {
	tasks   []Task
	results []TaskResult
	wg      sync.WaitGroup //同步协程，等待所有任务完成
	mu      sync.Mutex     //保护results的并发写入（避免数据竞争）
}

// NewScheduler创建调度器实例
func NewScheduler(tasks []Task) *Scheduler {
	return &Scheduler{
		tasks:   tasks,
		results: make([]TaskResult, 0, len(tasks)), //预分配切片容量，提升性能
	}

}

// Run 启动调度器：并发执行所有任务，统计执行时间
func (s *Scheduler) Run() {
	//遍历任务列表，为每个任务启动协程
	for taskID, task := range s.tasks {
		s.wg.Add(1) //每启动一个任务，waitGroup计数+1
		//启动协程执行任务（注意：循环变量要传参，避免捕获共享变量）
		go func(id int, t Task) {
			defer s.wg.Done()
			//1.统计任务执行时间
			startTime := time.Now()
			t() //执行任务
			endTime := time.Now()
			elapsed := endTime.Sub(startTime)
			//2.安全存储任务结果（加锁避免并发写入数据竞争）
			s.mu.Lock()
			s.results = append(s.results, TaskResult{
				TaskID:    id,
				StartTime: startTime,
				EndTime:   endTime,
				Elapsed:   elapsed,
			})
			s.mu.Unlock()

		}(taskID, task)
	}

	//等待所有任务执行完成
	s.wg.Wait()
}

// PrintResults打印所有任务的执行结果
func (s *Scheduler) PrintResults() {
	fmt.Println("=== 任务执行结果 ===")
	for _, res := range s.results {
		fmt.Printf("任务%d:开始时间=%s,结束时间=%s,执行耗时=%s\n",
			res.TaskID,
			res.StartTime.Format("2006-01-02 15:04:05.000"),
			res.EndTime.Format("2006-01-02 15:04:05.000"),
			res.Elapsed,
		)
	}

}
func main() {
	/*
		1.题目 ：编写一个程序，使用 go 关键字启动两个协程，一个协程打印从1到10的奇数，另一个协程打印从2到10的偶数。
		考察点 ： go 关键字的使用、协程的并发执行。
		2.题目 ：设计一个任务调度器，接收一组任务（可以用函数表示），并使用协程并发执行这些任务，同时统计每个任务的执行时间。
		考察点 ：协程原理、并发任务调度。
	*/

	//定义waitGroup,用于等待两个协程执行完成（避免著协诚提前退出）
	var wg sync.WaitGroup
	//声明要等待2个协程（必须再启动协程前调用）
	wg.Add(2)

	//使用go关键字启动协程：并发执行printOdd和printEven
	go printOdd(&wg)
	go printEven(&wg)
	//主协程阻塞，直到两个子协程都调用Done()
	wg.Wait()
	fmt.Println("\n所有协程执行完成，程序退出")

	//1.定义测试任务（模拟不同耗时的业务逻辑）
	tasks := []Task{
		func() {
			fmt.Println("任务1开始执行：模拟数据处理")
			time.Sleep(500 * time.Millisecond)
			fmt.Println("任务1执行执行完成")
		},
		func() {
			fmt.Println("任务2开始执行：模拟数据处理")
			time.Sleep(1500 * time.Millisecond)
			fmt.Println("任务3执行执行完成")
		},
		func() {
			fmt.Println("任务3开始执行：模拟数据处理")
			time.Sleep(300 * time.Millisecond)
			fmt.Println("任务3执行执行完成")
		},
	}
	//2.创建调度器并执行任务
	scheduler := NewScheduler(tasks)
	fmt.Println("开始执行所有任务...")
	scheduler.Run()
	//3.打印执行结果
	scheduler.PrintResults()
}
