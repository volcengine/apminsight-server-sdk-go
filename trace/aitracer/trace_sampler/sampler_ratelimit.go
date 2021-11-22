package trace_sampler

import (
	"sync"
	"time"
)

type ratelimitSampler struct {
	interval time.Duration

	mutex sync.Mutex

	last      time.Time
	noSampled int
}

func (s *ratelimitSampler) Sample() (bool, int) {
	s.mutex.Lock()
	defer func() {
		s.mutex.Unlock()
	}()
	now := time.Now()
	if now.Sub(s.last) < s.interval {
		s.noSampled++
		return false, 0
	}
	weight := s.noSampled + 1
	s.last = s.last.Add(now.Sub(s.last) / s.interval * s.interval)
	s.noSampled = 0
	return true, weight
}
