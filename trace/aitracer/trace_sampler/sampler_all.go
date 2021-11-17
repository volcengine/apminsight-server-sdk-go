package trace_sampler

type allSampler struct {
}

func (s *allSampler) Sample() (bool, int) {
	return true, 1
}
