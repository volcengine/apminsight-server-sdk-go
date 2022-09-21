package res_monitor

import (
	"fmt"
	"math"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCPU(t *testing.T) {
	thread := 1
	cpu := 5

	exceptCpuRatio := math.Round(float64(thread) / float64(cpu) * 100)

	runtime.GOMAXPROCS(cpu)

	{
		for j := 0; j < thread; j++ {
			go func() {
				i := 0
				for {
					i++
				}
			}()
		}
	}

	{
		m := NewCPUMonitor()
		tc := time.NewTicker(5 * time.Second)
		for {
			select {
			case <-tc.C:
				r := m.GetCPURatio()
				fmt.Printf("m=%+v, expectRatio=%+v, reportRatio=%+v\n", m, exceptCpuRatio, r)
				assert.Equal(t, exceptCpuRatio, math.Round(r))
			}
		}
	}
}

var stone = make([]byte, 0)

func TestMem(t *testing.T) {
	var l sync.RWMutex
	mbPerTime := 10
	{
		go func() {
			for i := 1; i <= 20; i++ {
				l.Lock()
				stone = append(stone, make([]byte, 1024*1024*mbPerTime)...) //10MB
				l.Unlock()
				time.Sleep(5 * time.Second)
			}
		}()
	}

	{
		m := NewMemMonitor()
		tc := time.NewTicker(5 * time.Second)
		for {
			select {
			case <-tc.C:
				r := m.GetMemRatio()
				l.RLock()
				expectMem := int64(len(stone))
				l.RUnlock()
				fmt.Printf("m=%+v, expectMem=%+v, reportRatio=%+v, rss-expectMem=%+v \n", m, expectMem, r, m.memRss-expectMem)

				assert.Condition(t, func() (success bool) {
					return m.memRss >= expectMem
				})
			}
		}
	}
}

func TestGoRoutine(t *testing.T) {
	m := NewGoRoutineMonitor()
	tc := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-tc.C:
			r := m.GetGoRoutineNum()

			fmt.Printf("m=%+v, reportGoRoutine=%+v \n", m, r)

			assert.Condition(t, func() (success bool) {
				return r >= 1
			})
		}
	}
}

func TestResMonitor(t *testing.T) {
	resMonitor := NewMonitor()
	resMonitor.Start()

	{
		go func() {
			time.Sleep(50 * time.Second)
			for j := 0; j < 5; j++ {
				go func() {
					i := 0
					for {
						i++
					}
				}()
			}
		}()
	}
	for {
		fmt.Printf("cpuRatio=%+v, memRatio=%+v, num=%+v, cpuRatioDelta=%+v, memRatioDelta=%+v, numDelta=%+v \n",
			resMonitor.GetCPURatio(), resMonitor.GetMemRatio(), resMonitor.GetGoRoutineNum(),
			resMonitor.GetCPURatioDelta(), resMonitor.GetMemRatioDelta(), resMonitor.GetGoRoutineNumDelta())
		time.Sleep(resourceCheckPeriodTime)
	}

}
