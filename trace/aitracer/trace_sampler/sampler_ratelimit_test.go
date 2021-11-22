package trace_sampler

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func Test00(t *testing.T) {
	s := &ratelimitSampler{interval: time.Second / 99}
	s.Sample()

	c := int64(0)
	w := int64(0)
	go func() {
		for range time.Tick(time.Millisecond) {
			ok, ww := s.Sample()
			//ok, ww := true, 1
			if ok {
				atomic.AddInt64(&c, int64(1))
				atomic.AddInt64(&w, int64(ww))
			}
		}

	}()
	for range time.Tick(time.Second) {
		cc := atomic.SwapInt64(&c, 0)
		ww := atomic.SwapInt64(&w, 0)
		fmt.Println(cc, ww)
	}
}
