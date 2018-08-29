package pool

import (
	"time"

	"wesure.cn/pool/goroutine"
)

var (
	// max memory 8GB (8KB/goroutine).
	_maxGoroutinesSize int32 = (1024 * 1024 * 8) / 8

	// maximum idle duration of a goroutine.
	_maxGoroutineIdleDuration = time.Second * 10

	// common block polling interval
	_commonBlockPollingInterval = 100 * time.Millisecond

	// new a common gopool.
	_gopool = goroutine.NewPool(_maxGoroutinesSize, _maxGoroutineIdleDuration, _commonBlockPollingInterval)
)

// SetGopool set or reset go pool config.
// Note: Make sure to call it before calling Go()
// If size<=0, will use default value.
// If expire<=0, will use default value.
func SetGopool(size int32, expire, interval time.Duration) {
	_maxGoroutinesSize, _maxGoroutineIdleDuration, _commonBlockPollingInterval := size, expire, interval
	if _gopool != nil {
		_gopool.Release()
	}
	_gopool = goroutine.NewPool(_maxGoroutinesSize, _maxGoroutineIdleDuration, _commonBlockPollingInterval)
}

// TryGo tries to execute the function via goroutine.
// If there are no concurrent resources, execute it synchronously.
func TryGo(fn func()) {
	_gopool.TryGo(fn)
}

// AnywayGo execute all tasks asynchronously,
// if pool is have no available worker, it will block waiting
func AnywayGo(fn func()) {
	_gopool.AnywayGo(fn)
}
