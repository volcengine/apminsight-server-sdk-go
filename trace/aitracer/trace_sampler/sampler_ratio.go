package trace_sampler

import "math/rand"

type ratioSampler struct {
	permil int
	weight int
}

func (s *ratioSampler) Sample() (bool, int) {
	if rand.Intn(1000) < s.permil {
		return true, s.weight
	}
	return false, 0
}
