package aitracer

import "sync"

type traceContext struct {
	traceID  string
	resource string

	sampleStrategy SampleStrategy
	sampleWeight   int

	tracer *tracer

	spansLock   sync.Mutex
	finishCount int
	spans       []*span
}

func (tc *traceContext) addSpan(s *span) {
	tc.spansLock.Lock()
	tc.spans = append(tc.spans, s)
	tc.spansLock.Unlock()
}

func (tc *traceContext) finishSpan() {
	finished := false
	tc.spansLock.Lock()
	tc.finishCount++
	if tc.finishCount == len(tc.spans) {
		finished = true
	}
	tc.spansLock.Unlock()
	if finished {
		tc.tracer.collect(tc)
	}
}
