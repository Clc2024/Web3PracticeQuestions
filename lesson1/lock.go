package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// 使用Mutex的协程安全计数器
type SafeCounter struct {
	mu    sync.Mutex
	count int64
}

// 创建一个Mutex的计数器 防止外部访问
func NewSafeCounter() *SafeCounter {
	return &SafeCounter{}
}

// 实现计数器自增加的方法
func (c *SafeCounter) Inc(m int) {
	c.mu.Lock()
	c.count++
	defer c.mu.Unlock()
	fmt.Printf("gorountine %d inc counter %d\n", m, c.count)
}

// 获取计数器值
func (c *SafeCounter) Get() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.count
}

func mutexDemo() {
	fmt.Println("=== mutexDemo ===")

	counter := NewSafeCounter()
	var wg sync.WaitGroup
	//启动多个goroutine并发增加计数器
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {

			defer wg.Done()
			for j := 0; j < 10; j++ {
				counter.Inc(j)
			}

		}(i)
	}

	wg.Wait()
	fmt.Printf("最终计数：%d(期望：100)\n", counter.Get())

}

// Counter 无锁原子计数器结构体（封装所有实现细节）
// 内部字段私有化，避免外部直接操作导致并发安全问题
type Counter struct {
	count int64
}

// NewCounter创建计数器实例（构造函数模式）
// 对外提供统一的初始化入口，可支持初始化配置
func NewCounter(initialValue int64) *Counter {
	return &Counter{count: initialValue}
}

// Increment计数器递增（对外暴露的核心方法）
// 封装atomic.AddInt64 ,用户无需关心原子操作细节
func (c *Counter) Increment(delta int64) {
	atomic.AddInt64(&c.count, delta)
}

// 获取计数器当前值（封装atomic.LoadInt64 ,保证读取的原子性）
func (c *Counter) Get() int64 {
	return atomic.LoadInt64(&c.count)
}

// Reset重置计数器为指定值（可选扩展方法）
func (c *Counter) Reset(newValue int64) {
	atomic.StoreInt64(&c.count, newValue)
}

// 无锁原子计数器
func atomicDemo() {
	Counter := NewCounter(0)
	const (
		gorountineNum = 10
		incrPerGorou  = 1000
	)

	var wg sync.WaitGroup
	wg.Add(gorountineNum)
	for i := 0; i < gorountineNum; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			//调用封装后的Increment方法，无需关系内部原子操作
			for j := 0; j < incrPerGorou; j++ {
				Counter.Increment(1)
			}
			fmt.Printf("goroutine %d finished, counter = %d\n", goroutineID, Counter.count)
		}(i)
	}
	wg.Wait()

	fmt.Printf("最终计数：%d(期望：%d)\n", Counter.Get(), gorountineNum*incrPerGorou)

	//可选：测试Reset方法
	Counter.Reset(0)
	fmt.Printf("计数器重置后的值：%d\n", Counter.Get())
}
func main() {
	/*
		1.题目 ：编写一个程序，使用 sync.Mutex 来保护一个共享的计数器。启动10个协程，每个协程对计数器进行1000次递增操作，最后输出计数器的值。
		考察点 ： sync.Mutex 的使用、并发数据安全。
		2.题目 ：使用原子操作（ sync/atomic 包）实现一个无锁的计数器。启动10个协程，每个协程对计数器进行1000次递增操作，最后输出计数器的值。
		考察点 ：原子操作、并发数据安全。
	*/
	mutexDemo()

	fmt.Println("=== atomicDemo ===")
	atomicDemo()

}
