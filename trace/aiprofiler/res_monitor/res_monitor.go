package res_monitor

import (
	"math"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

const (
	compareCount            = 5
	reserveCount            = 12
	resourceCheckPeriodSec  = 10
	resourceCheckPeriodTime = time.Second * resourceCheckPeriodSec
)

type Monitor struct {
	cpuMonitor       *CPUMonitor
	memMonitor       *MemMonitor
	goRoutineMonitor *GoRoutineMonitor

	//current
	cpuRatio     float64 // is decimal, not percent
	memRatio     float64 // is decimal, not percent
	goroutineNum float64 // is decimal, not percent

	//delta
	cpuRatioDelta     float64 // is decimal, not percent
	memRatioDelta     float64 // is decimal, not percent
	goroutineNumDelta float64 // is decimal, not percent

	// previous reserve
	previousCPURatio     [reserveCount]float64
	previousMemRatio     [reserveCount]float64
	previousGoroutineNum [reserveCount]float64

	cnt int64

	l sync.RWMutex

	closeChan chan struct{}
	wg        sync.WaitGroup
}

func NewMonitor() *Monitor {
	return &Monitor{
		cpuMonitor:       NewCPUMonitor(),
		memMonitor:       NewMemMonitor(),
		goRoutineMonitor: NewGoRoutineMonitor(),

		closeChan: make(chan struct{}),
	}
}

func (r *Monitor) Start() {
	r.update()
	tc := time.NewTicker(resourceCheckPeriodTime)
	r.wg.Add(1)
	go func() {
		defer func() {
			tc.Stop()
			r.wg.Done()
		}()
		for {
			select {
			case <-tc.C:
				r.update()
			case <-r.closeChan:
				return
			}
		}
	}()
}

func (r *Monitor) Stop() {
	close(r.closeChan)
	r.wg.Wait()
}

func (r *Monitor) update() {
	r.cnt++

	// get current data
	cpuRatio := r.cpuMonitor.GetCPURatio()
	memRatio := r.memMonitor.GetMemRatio()
	goroutineNum := float64(r.goRoutineMonitor.GetGoRoutineNum())

	// assign
	atomicStoreFloat64(&r.cpuRatio, cpuRatio)
	atomicStoreFloat64(&r.memRatio, memRatio)
	atomicStoreFloat64(&r.goroutineNum, goroutineNum)

	// update delta
	if r.cnt >= reserveCount { //cold start
		atomicStoreFloat64(&r.cpuRatioDelta, calDelta(r.cpuRatio, r.previousCPURatio[:compareCount]))
		atomicStoreFloat64(&r.memRatioDelta, calDelta(r.memRatio, r.previousMemRatio[:compareCount]))
		atomicStoreFloat64(&r.goroutineNumDelta, calDelta(r.goroutineNum, r.previousGoroutineNum[:compareCount]))
	}

	r.l.Lock()
	defer r.l.Unlock()
	// store previous data
	for idx := reserveCount - 1; idx >= 1; idx-- {
		r.previousCPURatio[idx] = r.previousCPURatio[idx-1] //better to use a link
		r.previousMemRatio[idx] = r.previousMemRatio[idx-1]
		r.previousGoroutineNum[idx] = r.previousGoroutineNum[idx-1]
	}
	r.previousCPURatio[0] = r.cpuRatio
	r.previousMemRatio[0] = r.memRatio
	r.previousGoroutineNum[0] = r.goroutineNum
}

func (r *Monitor) GetCPURatio() float64 {
	return atomicLoadFloat64(&r.cpuRatio)
}

func (r *Monitor) GetMemRatio() float64 {
	return atomicLoadFloat64(&r.memRatio)
}

func (r *Monitor) GetGoRoutineNum() float64 {
	return atomicLoadFloat64(&r.goroutineNum)
}

// GetCPURatioDelta compare with last window
func (r *Monitor) GetCPURatioDelta() float64 {
	return atomicLoadFloat64(&r.cpuRatioDelta)
}

// GetMemRatioDelta compare with last window
func (r *Monitor) GetMemRatioDelta() float64 {
	return atomicLoadFloat64(&r.memRatioDelta)
}

// GetGoRoutineNumDelta compare with last window
func (r *Monitor) GetGoRoutineNumDelta() float64 {
	return atomicLoadFloat64(&r.goroutineNumDelta)
}

// GetPastCPURatio get past avg cpu_ratio in last n seconds
func (r *Monitor) GetPastCPURatio(sec int64) float64 {
	rightIdx := getRightIndex(sec)
	r.l.RLock()
	defer r.l.RUnlock()
	return avg(r.previousCPURatio[0:rightIdx])
}

// GetPastMemRatio get past avg mem_ratio in last n seconds
func (r *Monitor) GetPastMemRatio(sec int64) float64 {
	rightIdx := getRightIndex(sec)
	r.l.RLock()
	defer r.l.RUnlock()
	return avg(r.previousMemRatio[0:rightIdx])
}

func getRightIndex(sec int64) int {
	cnt := math.Round(float64(sec) / float64(resourceCheckPeriodSec))
	if cnt <= 0 {
		cnt = 1
	} else if cnt > reserveCount {
		cnt = reserveCount
	}
	return int(cnt)
}

func avg(fs []float64) float64 {
	if len(fs) == 0 {
		return 0
	}
	// pre avg
	sum := float64(0)
	for _, pre := range fs {
		sum += pre
	}
	return sum / float64(len(fs))
}

func calDelta(cur float64, preList []float64) float64 {
	preAvg := avg(preList)
	//delta
	if preAvg == 0 { // preAvg is 0, means data collection is fail
		return 0
	}
	return (cur - preAvg) / preAvg
}

func atomicLoadFloat64(x *float64) float64 {
	return math.Float64frombits(atomic.LoadUint64((*uint64)(unsafe.Pointer(x))))
}

func atomicStoreFloat64(x *float64, f float64) {
	atomic.StoreUint64((*uint64)(unsafe.Pointer(x)), math.Float64bits(f))
}
