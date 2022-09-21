package aiprofiler

import (
	"fmt"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/volcengine/apminsight-server-sdk-go/trace/aiprofiler/common"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aiprofiler/utils"
)

func TestProfiler(t *testing.T) {
	os.Setenv("APMPLUS_APP_KEY", "{YOUR_APP_KEY}")

	makeProcessBusy()

	opts := make([]Option, 0)
	opts = append(opts, WithLogger(&l{}))
	opts = append(opts, WithHTTPEndPoint("http", "0.0.0.0:8080", 5*time.Second))

	profiler := NewProfiler("http", "server_a", opts...)

	profiler.Start()

	go func() {
		time.Sleep(80 * time.Second)
		profiler.Stop()
	}()

	//// local debug
	//localTest(profiler)

	time.Sleep(1000 * time.Second)

}

func localTest(profiler *Profiler) {
	for {
		profiler.DebugTasks([]*common.Task{{
			Name:         "local test",
			ProfileID:    1,
			UploadID:     utils.NewRandID(),
			ProfileTypes: []common.ProfileType{common.ProfileTypeCPU},
			PtConfigs: []common.ProfileTypeConfig{
				{
					ProfileType:     common.ProfileTypeCPU,
					DurationSeconds: 10,
				},
			},
		}})
		time.Sleep(15 * time.Second)
	}
}

type l struct{}

func (l *l) Debug(format string, args ...interface{}) {
	fmt.Printf("[Debug]"+format+"\n", args...)
}
func (l *l) Info(format string, args ...interface{}) {
	fmt.Printf("[Info]"+format+"\n", args...)
}
func (l *l) Error(format string, args ...interface{}) {
	fmt.Printf("[Error]"+format+"\n", args...)
}

const (
	networkTime = 70 * time.Millisecond
	cpuTime     = 30 * time.Millisecond
)

func makeProcessBusy() {
	// cpu and heap
	go func() {
		for {
			cpuIntensiveWorkload()
			idleWorkload()
			memoryAllocHeap()

		}
	}()
	// mutex
	go func() {
		var l sync.Mutex
		var m = make(map[int]struct{})
		for {
			for i := 0; i < 1000; i++ {
				go func(i int) {
					l.Lock()
					defer l.Unlock()
					m[i] = struct{}{}
				}(i)
			}
		}
	}()
	// block
	go func() {
		c := make(chan int, 0)
		for {
			for i := 0; i < 10; i++ {
				c <- 1
			}
		}
	}()
}

func cpuIntensiveWorkload() {
	st := time.Now()
	d := data{}
	for time.Since(st) < cpuTime {
		d.cpuIntensiveWorkloadCore()
	}
}

type data struct {
}

func (d data) cpuIntensiveWorkloadCore() {
	for i := 0; i < 10000; i++ {
		_ = i
	}
}

func idleWorkload() {
	time.Sleep(networkTime)
}

func memoryAllocHeap() []byte {
	a := make([]byte, rand.Intn(20000)) //10kb
	return a
}
