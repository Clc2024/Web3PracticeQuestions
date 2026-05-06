package main

import (
	"fmt"
	"sync"
)

func runUnbufferedChannelExample() {
	var wg = sync.WaitGroup{}
	ch := make(chan int)

	wg.Add(2)
	//生产者协程
	go func() {
		defer wg.Done()
		defer close(ch)
		defer func() {
			if r := recover(); r != nil {
				fmt.Print("Producer panic:%v\n", r)
			}
		}()
		for i := 1; i <= 10; i++ {
			ch <- i
		}

	}()
	//消费者协程
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				fmt.Print("Consumer panic:%v\n", r)
			}
		}()
		for i := range ch {
			fmt.Println(i)
		}
	}()
	wg.Wait()
	fmt.Println("over\n")

}

func runBufferedChannelExample() {
	var wg = sync.WaitGroup{}
	ch := make(chan int, 10)
	wg.Add(2)
	//生产者协程
	go func() {
		defer wg.Done()
		defer close(ch)
		defer func() {
			if r := recover(); r != nil {
				fmt.Print("Producer panic:%v\n", r)
			}
		}()
		for i := 1; i <= 10; i++ {
			ch <- i
		}

	}()
	//消费者协程
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				fmt.Print("Consumer panic:%v\n", r)
			}
		}()
		for i := range ch {
			fmt.Println(i)
		}
	}()
	wg.Wait()
	fmt.Println("over\n")
}
func main() {
	/*
		1.题目 ：编写一个程序，使用通道实现两个协程之间的通信。一个协程生成从1到10的整数，并将这些整数发送到通道中，另一个协程从通道中接收这些整数并打印出来。
		考察点 ：通道的基本使用、协程间通信。
		2.题目 ：实现一个带有缓冲的通道，生产者协程向通道中发送100个整数，消费者协程从通道中接收这些整数并打印。
		考察点 ：通道的缓冲机制。
	*/
	fmt.Println("runUnbufferedChannelExample")
	runUnbufferedChannelExample()
	fmt.Println("runBufferedChannelExample")
	runBufferedChannelExample()
}
