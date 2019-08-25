package pool

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

var (
	n      int32 = 100
	curMem uint64
)

const (
	_   = 1 << (10 * iota)
	KiB // 1024
	MiB // 1048576
	GiB // 1073741824
	TiB // 1099511627776             (超过了int32的范围)
	PiB // 1125899906842624
	EiB // 1152921504606846976
	ZiB // 1180591620717411303424    (超过了int64的范围)
	YiB // 1208925819614629174706176
)

func TestNoPool(t *testing.T) {
	var wg sync.WaitGroup
	for i := int32(0); i < n; i++ {
		wg.Add(1)
		go func() {
			demoFunc()
			wg.Done()
		}()
	}

	wg.Wait()
	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	curMem = mem.TotalAlloc/MiB - curMem
	t.Logf("memory usage:%d MB", curMem)
}

func TestTryGoPool(t *testing.T) {
	goPool := NewPool(0, 0, 0) // use default config
	defer goPool.Release()

	var wg sync.WaitGroup
	for i := int32(0); i < n; i++ {
		wg.Add(1)
		goPool.TryGo(func() {
			demoFunc()
			wg.Done()
		})
	}

	wg.Wait()
	t.Logf("pool, capacity:%d", goPool.Cap())
	t.Logf("pool, running workers number:%d", goPool.Running())
	t.Logf("pool, free workers number:%d", goPool.Free())

	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	curMem = mem.TotalAlloc/MiB - curMem
	t.Logf("memory usage:%d MB", curMem)
}

func TestAnywayGoPool(t *testing.T) {
	goPool := NewPool(0, 0, 0) // use default config
	defer goPool.Release()

	var wg sync.WaitGroup
	for i := int32(0); i < n; i++ {
		wg.Add(1)
		goPool.AnywayGo(func() {
			demoFunc()
			wg.Done()
		})
	}

	wg.Wait()
	t.Logf("pool, capacity:%d", goPool.Cap())
	t.Logf("pool, running workers number:%d", goPool.Running())
	t.Logf("pool, free workers number:%d", goPool.Free())

	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	curMem = mem.TotalAlloc/MiB - curMem
	t.Logf("memory usage:%d MB", curMem)
}

func demoFunc() {
	time.Sleep(1 * time.Millisecond)
}
