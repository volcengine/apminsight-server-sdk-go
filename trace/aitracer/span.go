package aitracer

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/volcengine/apminsight-server-sdk-go/trace/internal"
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

	errLock       sync.Mutex
	ErrorInfoList []*ErrorInfo

	startTime  time.Time
	finishTime time.Time
	duration   time.Duration

	tagsLock    sync.Mutex
	tagsString  map[string]string
	tagsInt64   map[string]int64
	tagsFloat64 map[string]float64

	spanContext spanContext

	finished  int64
	collected int64
}

type ErrorInfo struct {
	ErrorKind              ErrorKind
	ErrorMessage           string
	ErrorStack             []string
	ErrorOccurTimeMilliSec int64
	ErrorTags              map[string]string
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

// Finish todo: unify Finish and FinishWithOption. func (s *span) Finish(opt ...FinishSpanOption)
// Finish() and FinishWithOption() should be called directly by defer (but not be wrapped by a func) and recover() should be called directly in Finish() and FinishWithOption()
/*
	defer span.Finish()  // good

	defer func(){        // bad. can not capture panic
          span.Finish()
	}()

*/
// DO NOT place recover() in a func and call the func in Finish()

func (s *span) Finish() {
	if !atomic.CompareAndSwapInt64(&s.finished, 0, 1) {
		return
	}

	s.finishTime = time.Now()
	s.duration = s.finishTime.Sub(s.startTime)

	s.fillTag()
	//recordPanic.
	//if panic cause process crash directly, metric/trace may not have time to send out.
	//this will be resolved by globalPanicCapture
	if err := recover(); err != nil {
		defer panic(err)
		errorInfo := ErrorInfo{
			ErrorKind:              ErrorKindPanic,
			ErrorMessage:           fmt.Sprint(err),
			ErrorOccurTimeMilliSec: time.Now().Unix()*1e3 + int64(time.Now().Nanosecond())/1e6,
			ErrorTags:              map[string]string{internal.GoErrorType: getErrorType(err)},
		}
		errorInfo.ErrorStack = getStackTrace()
		s.errLock.Lock()
		s.ErrorInfoList = append(s.ErrorInfoList, &errorInfo)
		s.errLock.Unlock()
		s.status = 1
	}
	// emit metric
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
	//recordPanic
	if !opt.DisablePanicCapture {
		if err := recover(); err != nil {
			defer panic(err)
			errorInfo := ErrorInfo{
				ErrorKind:              ErrorKindPanic,
				ErrorMessage:           fmt.Sprint(err),
				ErrorOccurTimeMilliSec: time.Now().Unix()*1e3 + int64(time.Now().Nanosecond())/1e6,
				ErrorTags:              map[string]string{internal.GoErrorType: getErrorType(err)},
			}
			errorInfo.ErrorStack = getStackTrace()
			s.errLock.Lock()
			s.ErrorInfoList = append(s.ErrorInfoList, &errorInfo)
			s.errLock.Unlock()
			s.status = 1
		}
	}
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
	// must add sdk info to every span. this info is used to handler stack
	s.SetTagString(internal.SdkLanguage, internal.Go) //todo: add version
}

func (s *span) RecordError(err error, opts ...RecordOption) {
	if s == nil || err == nil || s.isFinished() {
		return
	}
	c := NewDefaultRecordConfig()
	for _, opt := range opts {
		opt(&c)
	}
	errorInfo := ErrorInfo{
		ErrorKind:              c.ErrorKind,
		ErrorMessage:           err.Error(),
		ErrorOccurTimeMilliSec: time.Now().Unix()*1e3 + int64(time.Now().Nanosecond())/1e6,
		ErrorTags:              map[string]string{internal.GoErrorType: getErrorType(err)},
	}
	if c.RecordStack && c.Stack == "" {
		errorInfo.ErrorStack = getStackTrace()
	} else if c.Stack != "" {
		errorInfo.ErrorStack = strings.Split(c.Stack, "\n")
	}
	s.errLock.Lock()
	defer s.errLock.Unlock()
	s.ErrorInfoList = append(s.ErrorInfoList, &errorInfo)
}

func (s *span) SetStatus(status int64) {
	if s == nil || s.isFinished() {
		return
	}
	s.status = status
}

func getStackTrace() []string {
	stackTrace := make([]byte, 2048)
	n := runtime.Stack(stackTrace, false)
	return strings.Split(string(stackTrace[0:n]), "\n")
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

func (s *span) isFinished() bool {
	return atomic.LoadInt64(&s.finished) == 1
}

func getErrorType(err interface{}) string {
	t := reflect.TypeOf(err)
	if t.PkgPath() == "" || t.Name() == "" {
		return t.String()
	}
	return fmt.Sprintf("%s.%s", t.PkgPath(), t.Name())
}

// -----------------------------
// sdk developers only. used to control trace/log emit
// by default controlEnable is false, means no control
const controlKey = "byteapm-tl-emit"

var (
	once          sync.Once
	controlEnable bool
)

func shouldEmit(s Span) bool {
	once.Do(func() {
		if os.Getenv("BYTEAPM_TL_EMIT_CONTROL") == "1" {
			controlEnable = true
		}
	})
	if !controlEnable {
		return true
	}
	if s == nil {
		return false
	}
	if flag := s.BaggageItem(controlKey); flag == "1" {
		return true
	}
	return false
}
