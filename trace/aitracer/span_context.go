package aitracer

import "sync"

type spanContext struct {
	spanID       string
	parentSpanID string

	traceContext *traceContext

	baggageLock sync.Mutex
	baggage     map[string]string
}

func (sc *spanContext) SpanID() string {
	return sc.spanID
}

func (sc *spanContext) TraceID() string {
	return sc.traceContext.traceID
}

func (sc *spanContext) Sample() (SampleStrategy, int) {
	return sc.traceContext.sampleStrategy, sc.traceContext.sampleWeight
}

func (sc *spanContext) ForeachBaggageItem(handler func(k, v string) bool) {
	sc.baggageLock.Lock()
	for k, v := range sc.baggage {
		if !handler(k, v) {
			break
		}
	}
	sc.baggageLock.Unlock()
}
