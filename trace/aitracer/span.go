package aitracer

import (
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type spanType int

const (
	commonSpanType spanType = iota
	serverSpanType
	clientSpanType
)

type span struct {
	spanType spanType

	status int64

	operationName string

	serverResource string

	clientType     string
	clientService  string
	clientResource string

	startTime  time.Time
	finishTime time.Time
	duration   time.Duration

	tagsLock    sync.Mutex
	tagsString  map[string]string
	tagsInt64   map[string]int64
	tagsFloat64 map[string]float64

	spanContext spanContext

	finished int64
}

const (
	aiCalledThroughput = "apminsight.service.trace.called.throughput"
	aiCalledLatency    = "apminsight.service.trace.called.latency.us"
	aiCallThroughput   = "apminsight.service.trace.call.throughput"
	aiCallLatency      = "apminsight.service.trace.call.latency.us"
)

func (s *span) emitMetric() {
	if s.spanType == commonSpanType {
		return
	}
	tc := s.spanContext.traceContext
	t := tc.tracer
	mc := t.metricsClient
	if mc == nil {
		return
	}
	if s.spanType == serverSpanType {
		tags := map[string]string{}
		tags["service_type"] = t.serviceType
		tags["service"] = t.service
		tags["resource"] = tc.resource
		tags["status"] = strconv.FormatInt(s.status, 10)
		tags["instance_id"] = t.instanceId

		_ = mc.EmitCounter(aiCalledThroughput, 1, tags)
		_ = mc.EmitTimer(aiCalledLatency, float64(s.duration.Microseconds()), tags)
	} else {
		tags := map[string]string{}
		tags["service_type"] = t.serviceType
		tags["service"] = t.service
		tags["resource"] = tc.resource
		tags["status"] = strconv.FormatInt(s.status, 10)
		tags["instance_id"] = t.instanceId
		tags["call_service_type"] = s.clientType
		tags["call_service"] = s.clientService
		tags["call_resource"] = s.clientResource

		// add extra tag
		slowQuery, ok := s.GetTagString("db.slow_query")
		if ok {
			tags["db.slow_query"] = slowQuery
		}

		_ = mc.EmitCounter(aiCallThroughput, 1, tags)
		_ = mc.EmitTimer(aiCallLatency, float64(s.duration.Microseconds()), tags)
	}
}

func (s *span) Finish() {
	if !atomic.CompareAndSwapInt64(&s.finished, 0, 1) {
		return
	}

	s.finishTime = time.Now()
	s.duration = s.finishTime.Sub(s.startTime)
	// emit metric
	s.fillTag()
	s.emitMetric()
	s.spanContext.traceContext.finishSpan()
}

func (s *span) FinishWithOption(opt FinishSpanOption) {
	if !atomic.CompareAndSwapInt64(&s.finished, 0, 1) {
		return
	}

	s.status = opt.Status
	if opt.FinishTime.IsZero() {
		s.finishTime = time.Now()
	} else {
		s.finishTime = opt.FinishTime
	}
	s.duration = s.finishTime.Sub(s.startTime)
	s.fillTag()
	s.emitMetric()
	s.spanContext.traceContext.finishSpan()
}

func (s *span) fillTag() {
	switch s.clientType {
	case MySQL:
		isSlow := "0"
		dynamicConfig := s.spanContext.traceContext.tracer.getDynamicConfig()
		if dynamicConfig != nil {
			if s.duration > dynamicConfig.DbSlowQuery {
				isSlow = "1"
			}
		}
		s.SetTagString("db.slow_query", isSlow)
	}
}

func (s *span) Context() SpanContext {
	return &s.spanContext
}

func (s *span) SetTag(key string, value interface{}) Span {
	switch v := value.(type) {
	case string:
		s.SetTagString(key, v)
	case int:
		s.SetTagInt64(key, int64(v))
	case int8:
		s.SetTagInt64(key, int64(v))
	case int16:
		s.SetTagInt64(key, int64(v))
	case int32:
		s.SetTagInt64(key, int64(v))
	case int64:
		s.SetTagInt64(key, v)
	case uint:
		s.SetTagInt64(key, int64(v))
	case uint8:
		s.SetTagInt64(key, int64(v))
	case uint16:
		s.SetTagInt64(key, int64(v))
	case uint32:
		s.SetTagInt64(key, int64(v))
	case float32:
		s.SetTagFloat64(key, float64(v))
	case float64:
		s.SetTagFloat64(key, v)
	}
	return s
}

func (s *span) GetTagString(key string) (string, bool) {
	s.tagsLock.Lock()
	if s.tagsString == nil {
		return "", false
	}
	value, ok := s.tagsString[key]
	s.tagsLock.Unlock()
	return value, ok

}

func (s *span) SetTagString(key string, value string) Span {
	s.tagsLock.Lock()
	if s.tagsString == nil {
		s.tagsString = map[string]string{}
	}
	s.tagsString[key] = value
	s.tagsLock.Unlock()
	return s

}

func (s *span) GetTagFloat64(key string) (float64, bool) {
	s.tagsLock.Lock()
	if s.tagsFloat64 == nil {
		return 0, false
	}
	value, ok := s.tagsFloat64[key]
	s.tagsLock.Unlock()
	return value, ok
}

func (s *span) SetTagFloat64(key string, value float64) Span {
	s.tagsLock.Lock()
	if s.tagsFloat64 == nil {
		s.tagsFloat64 = map[string]float64{}
	}
	s.tagsFloat64[key] = value
	s.tagsLock.Unlock()
	return s
}

func (s *span) GetTagInt64(key string) (int64, bool) {
	s.tagsLock.Lock()
	if s.tagsInt64 == nil {
		return 0, false
	}
	value, ok := s.tagsInt64[key]
	s.tagsLock.Unlock()
	return value, ok
}

func (s *span) SetTagInt64(key string, value int64) Span {
	s.tagsLock.Lock()
	if s.tagsInt64 == nil {
		s.tagsInt64 = map[string]int64{}
	}
	s.tagsInt64[key] = value
	s.tagsLock.Unlock()
	return s
}

func (s *span) SetBaggageItem(restrictedKey, value string) Span {
	s.spanContext.baggageLock.Lock()
	if s.spanContext.baggage == nil {
		s.spanContext.baggage = map[string]string{}
	}
	s.spanContext.baggage[restrictedKey] = value
	s.spanContext.baggageLock.Unlock()
	return s
}

func (s *span) BaggageItem(restrictedKey string) string {
	s.spanContext.baggageLock.Lock()
	var ret string
	if s.spanContext.baggage != nil {
		ret = s.spanContext.baggage[restrictedKey]
	}
	s.spanContext.baggageLock.Unlock()
	return ret
}
