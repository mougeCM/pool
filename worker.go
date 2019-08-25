package pool

import (
	"time"
)

// Worker responsible for performing tasks,
// use with goroutine.
type Worker struct {
	// who owns this worker.
	pool *Pool

	// is a job should be done.
	task chan f

	// will be update when putting a worker back into queue.
	recycleTime time.Time
}

func (w *Worker) run() {
	go func() {
		// listening task list, Once the task is taken out, run it.
		for fn := range w.task {
			// the task was release
			if fn == nil {
				w.pool.decrRunning()
				return
			}
			fn()
			// after finish f, recycling worker
			w.pool.putWorker(w)
		}
	}()
}
