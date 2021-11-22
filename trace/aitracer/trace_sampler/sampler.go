package trace_sampler

import (
	"sync"
	"time"
)

type SamplerStrategy int

const (
	SamplerStrategyAll = iota
	SamplerStrategyRatio
	SamplerStrategyRateLimit
)

type SamplerConfig struct {
	Strategy int
	Value    float64
}

type Sampler struct {
	rwlock          sync.RWMutex
	internalSampler internalSampler
}

func New() *Sampler {
	s := &Sampler{}
	s.RefreshConfig(SamplerConfig{
		Strategy: SamplerStrategyAll,
	})
	return s
}

func (s *Sampler) Sample() (bool, int) {
	s.rwlock.RLock()
	defer func() {
		s.rwlock.RUnlock()
	}()
	return s.internalSampler.Sample()
}

func (s *Sampler) RefreshConfig(config SamplerConfig) {
	s.rwlock.Lock()
	switch config.Strategy {
	case SamplerStrategyAll:
		s.internalSampler = &allSampler{}
	case SamplerStrategyRatio:
		if config.Value <= 0 {
			s.internalSampler = &noSampler{}
		} else {
			s.internalSampler = &ratioSampler{
				permil: int(config.Value * 1000),
				weight: int(1 / config.Value),
			}
		}
	case SamplerStrategyRateLimit:
		if config.Value <= 0 {
			s.internalSampler = &noSampler{}
		} else {
			s.internalSampler = &ratelimitSampler{
				interval: time.Second / time.Duration(config.Value),
			}
		}
	}
	s.rwlock.Unlock()
}

type internalSampler interface {
	Sample() (bool, int)
}
