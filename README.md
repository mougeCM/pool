## A Go-based open source pool toolkit

### 1. Why do you have a coroutine pool？

> 1.Golang does not limit the number of goroutines generated. Although the creation of goroutines is very lightweight (about 8kb), unlimited large-scale creation may squash memory; to create a million goroutine, the memory created is about `8000000/(1024*1024)=7.6G`; the coroutine pool can control the number of goroutines very well. 
> 2.When there are a large number of goroutines at the same time, the performance of the runtime scheduling and GC is greatly reduced, and even problems may occur.     
> 3.When the request is too high, each time a goroutine is created, goroutine reuse is not done, and a lot of resources are wasted; the goroutine of the coroutine pool can be reused to save resources.

### 2. Design ideas

```
	Initialize a Goroutine Pool when starting the service (maintain a stack-like FILO queue)
，Inside is the Worker responsible for processing the task, and then the client submits the task to the pool:
	1.Check whether there is a free worker in the current Worker queue, and if so, take out the current task;
	
	2.There is no idle worker, it is judged whether the currently running worker has exceeded the capacity of the pool, and then the device waits until the worker is put back to the pool; otherwise, a new worker (goroutine) process is opened;
	
	3.After each worker executes the task, it is put back into the pool queue to wait;
	
	4.Each worker has a timeout set. When the pool is started, a goroutine is started separately to clean the timeout worker.
```

### 3. Test performance

_100w concurrent test_

```
	No pool
		cost(1.32s), Memory(132M)
		
	Goroutine pool
		cost(1.38s), Memory(69M) 
```

_1000w concurrent test_

```
	No pool
		cost(19.58s), Memory(1373M) 
		
	Goroutine pool
		cost(15.89s), Memory(574M) 
```

### 4. Instructions

```Golang
package main

import (
	"wesure.cn/pool"
)

func main() {
	// Use custom configuration【Also choose not to set: use default configuration】
	pool.SetGopool(100, 10*time.Second(), 10*time.Second())

	// If all the goroutines in the coroutine pool are running, they are executed synchronously.
	pool.TryGo(func(){
		// func body
	})
	
	// It will be executed asynchronously anyway, and if no goroutine is available, it will block waiting.
	pool.AnywayGo(func() {
		// func body
	})
}

```

