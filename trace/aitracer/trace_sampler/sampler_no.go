package trace_sampler

type noSampler struct {
}

func (s *noSampler) Sample() (bool, int) {
	return false, 0
}
