## 协程池设计文档

### 1. 为什么要有协程池？

> 1.Golang没有限制goroutine的生成数量，虽然goroutine的创建很轻量(约8kb)，但是无限制大规模的创建还是可能会压爆内存；以创建一百万的goroutine来算，创建的内存大约是`8000000/(1024*1024)=7.6G`；协程池可以很好的控制goroutine数量.    
> 2.当同时存在大量的goroutine在执行任务时，runtime调度和GC的性能大大降低，甚至可能会出现问题.      
> 3.请求并发过高时，每次都会创建goroutine，做不到goroutine复用，浪费大量资源；协程池的goroutine可以进行复用节约资源.

### 2. 设计思路

```
	启动服务之时先初始化一个Goroutine Pool池(维护了一个类似栈的FILO队列)，里面存放负责处理task的Worker，然后client提交task到pool中：
	1.检查当前Worker队列中是否有空闲的Worker，如果有，取出执行当前的task；
	2.没有空闲Worker，判断当前在运行的Worker是否已超过该Pool的容量，是则阻塞等待直至有Worker被放回Pool；否则新开一个Worker(goroutine)处理;
	3.每个Worker执行完task之后，放回Pool的队列中等待；
	4.每个Worker都有设置了超时，启动pool时回单独启动一个goroutine去clean超时的worker.
```

### 3. 测试性能

_100w并发测试_

```
	无协程池
		耗时(1.32s), 耗用内存(132M)
		
	有协程池
		耗时(1.38s), 耗用内存(69M) 
```

_1000W并发测试_

```
	无协程池
		耗时(19.58s), 耗用内存(1373M) 
		
	有协程池
		耗时(15.89s), 耗用内存(574M) 
```

### 4. 使用方法

_pool使用默认配置_

```Golang
package main

import (
	"wesure.cn/pool"
)

func main() {
	// 使用自定义配置【也可选择不设置：使用默认配置】
	pool.SetGopool(100, 10*time.Second(), 10*time.Second())

	// 如果协程池所有goroutine处于运行状态，则同步执行.
	pool.TryGo(func(){
		// func body
	})
	
	// 无论如何都会异步执行，如果无可用goroutine，则会阻塞等待.
	pool.AnywayGo(func() {
		// func body
	})
}

```

