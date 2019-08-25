package pool

import (
	"context"
	"errors"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// the default capacity for a default goroutine pool.
	defaultMaxGoroutinesSize = 512 * 1024

	// the default maximum idle duration of a goroutine.
	defaultMaxGoroutineIdleDuration = 10 * time.Second

	// the default block polling interval
	defaultBlockPollingInterval = 100 * time.Millisecond
)

var (
	ErrLack       = errors.New("lack of goroutines, because exceeded capacity limit.")
	ErrPoolClosed = errors.New("this pool has been closed")
)

type (
	sig struct{}

	// task execution func
	f func()
)

// Pool accept the tasks from client,it limits the total
// of goroutines to a given number by recycling goroutines.
type Pool struct {
	mu   sync.Mutex
	once sync.Once

	// capacity of the pool.
	capacity int32

	// number of tasks running goroutines.
	running int32

	// set the expired time (second) of every worker.
	expire time.Duration

	// set the block polling interval.
	interval time.Duration

	// is a slice that store the available workers.
	workers []*Worker

	// is used to notice the pool to closed itself.
	release chan sig
}

// clear expired workers periodically.
func (p *Pool) clean() {
	t := time.NewTicker(p.expire)
	for range t.C {
		currentTime := time.Now()
		p.mu.Lock()
		idleWorkers := p.workers
		if len(idleWorkers) == 0 && p.Running() == 0 && len(p.release) > 0 {
			p.mu.Unlock()
			return
		}
		n := 0
		for i, w := range idleWorkers {
			if currentTime.Sub(w.recycleTime) <= p.expire {
				break
			}
			n = i
			w.task <- nil
			idleWorkers[i] = nil
		}
		n++
		if n >= len(idleWorkers) {
			// all workers have expired
			p.workers = idleWorkers[:0]
		} else {
			// remove the expire workers
			p.workers = idleWorkers[n:]
		}
		p.mu.Unlock()
	}
}

// NewPool creates a new *Pool.
// If size<=0, will use default value.
// If expire<=0, will use default value.
// If interval<=0, will use default value.
func NewPool(size int32, expire, interval time.Duration) *Pool {
	p := &Pool{
		release: make(chan sig, 1),
	}
	if size <= 0 {
		p.capacity = defaultMaxGoroutinesSize
	} else {
		p.capacity = size
	}
	if expire <= 0 {
		p.expire = defaultMaxGoroutineIdleDuration
	} else {
		p.expire = expire
	}
	if interval <= 0 {
		p.interval = defaultBlockPollingInterval
	} else {
		p.interval = interval
	}

	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("clean err:%s\n", err)
			}
		}()

		p.clean()
	}()
	return p
}

// Create giroutine
func (p *Pool) Go(fn f) error {
	return p.submit(fn)
}

// TryGo tries to execute the function via goroutine.
// If there are no concurrent resources, execute it synchronously.
func (p *Pool) TryGo(fn f) {
	err := p.Go(fn)
	if err != nil {
		if err == ErrPoolClosed {
			log.Printf("%v\n", err)
		}
		fn()
	}
}

// AnywayGo block until the goroutine is obtained.
func (p *Pool) AnywayGo(fn f, ctx ...context.Context) error {
	if len(p.release) > 0 {
		return ErrPoolClosed
	}

	if len(ctx) == 0 {
		for p.Go(fn) != nil {
			runtime.Gosched()
		}
		return nil
	}
	c := ctx[0]
	for {
		select {
		case <-c.Done():
			return c.Err()
		default:
			if p.Go(fn) == nil {
				return nil
			}
			runtime.Gosched()
		}
	}
}

// Release closes this pool.
func (p *Pool) Release() error {
	p.once.Do(func() {
		p.release <- sig{}

		p.mu.Lock()
		idleWorkers := p.workers
		for i, w := range idleWorkers {
			w.task <- nil
			idleWorkers[i] = nil
		}
		p.workers = nil
		p.mu.Unlock()
	})
	return nil
}

// Running returns the number of the currently running goroutines.
func (p *Pool) Running() int {
	return int(atomic.LoadInt32(&p.running))
}

// Free returns the available goroutines to work.
func (p *Pool) Free() int {
	return int(atomic.LoadInt32(&p.capacity) - atomic.LoadInt32(&p.running))
}

// Cap returns the capacity of this pool.
func (p *Pool) Cap() int {
	return int(atomic.LoadInt32(&p.capacity))
}

// increases the number of the currently running goroutines.
func (p *Pool) incrRunning() {
	atomic.AddInt32(&p.running, 1)
}

// decreases the number of the currently running goroutines.
func (p *Pool) decrRunning() {
	atomic.AddInt32(&p.running, -1)
}

// submits a task to this pool.
func (p *Pool) submit(fn f) error {
	if len(p.release) > 0 {
		return ErrPoolClosed
	}

	w := p.getWorker()
	if w == nil {
		return ErrLack
	}

	w.task <- fn
	return nil
}

// returns a available worker to run the tasks.
func (p *Pool) getWorker() *Worker {
	var w *Worker
	// Flag indicating whether the number of currently running
	// workers has reached the capacity limit
	var isFull bool

	p.mu.Lock()
	idleWorkers := p.workers
	n := len(p.workers) - 1
	// no idle workers
	if n < 0 {
		isFull = p.Running() >= p.Cap()
	} else {
		w = p.workers[n]
		idleWorkers[n] = nil
		p.workers = idleWorkers[:n]
	}
	p.mu.Unlock()

	if !isFull && w == nil {
		// new worker to run
		w = &Worker{
			pool: p,
			task: make(chan f, 1),
		}
		w.run()
		p.incrRunning()
	}

	return w
}

// puts a worker back into free pool, recycling the goroutines.
func (p *Pool) putWorker(worker *Worker) {
	worker.recycleTime = time.Now()
	p.mu.Lock()
	p.workers = append(p.workers, worker)
	p.mu.Unlock()
}
