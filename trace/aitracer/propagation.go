package aitracer

import (
	"net/http"
	"strconv"
	"strings"
)

type BuiltinFormat byte

const (
	HTTPHeaders BuiltinFormat = iota
)

type HTTPHeadersCarrier http.Header

func (c HTTPHeadersCarrier) Set(key, val string) {
	h := http.Header(c)
	h.Set(key, val)
}

func (c HTTPHeadersCarrier) ForeachKey(handler func(key, val string) error) error {
	for k, vals := range c {
		for _, v := range vals {
			if err := handler(k, v); err != nil {
				return err
			}
		}
	}
	return nil
}

type HTTPHeadersInjector struct {
}

var _ Injector = &HTTPHeadersInjector{}

func (injector *HTTPHeadersInjector) Inject(sc SpanContext, carrier interface{}) error {
	c, ok := carrier.(HTTPHeadersCarrier)
	if !ok {
		return ErrInvalidCarrier
	}
	c.Set(defaultTraceIDHeader, sc.TraceID())
	c.Set(defaultSpanIDHeader, sc.SpanID())
	strategy, weight := sc.Sample()
	switch strategy {
	case SampleStrategySampled:
		c.Set(defaultSampleHitHeader, "1")
		c.Set(defaultSampleWeightHeader, strconv.Itoa(weight))
	case SampleStrategyNotSampled:
		c.Set(defaultSampleHitHeader, "0")
		c.Set(defaultSampleWeightHeader, strconv.Itoa(weight))
	}
	return nil
}

const (
	defaultTraceIDHeader       = "x-trace-id"
	defaultSpanIDHeader        = "x-span-id"
	defaultSampleHitHeader     = "x-sample-hit"
	defaultSampleWeightHeader  = "x-sample-weight"
	defaultBaggageHeaderPrefix = "x-baggage-"
)

type HTTPHeadersExtractor struct {
}

var _ Extractor = &HTTPHeadersExtractor{}

type HTTPHeaderExtractSpanContext struct {
	traceID        string
	spanID         string
	sampleStrategy SampleStrategy
	sampleWeight   int
	baggage        map[string]string
}

func (sc *HTTPHeaderExtractSpanContext) TraceID() string {
	return sc.traceID
}

func (sc *HTTPHeaderExtractSpanContext) SpanID() string {
	return sc.spanID
}

func (sc *HTTPHeaderExtractSpanContext) Sample() (SampleStrategy, int) {
	return sc.sampleStrategy, sc.sampleWeight
}

func (sc *HTTPHeaderExtractSpanContext) ForeachBaggageItem(h func(string, string) bool) {
	for key, val := range sc.baggage {
		if !h(key, val) {
			return
		}
	}
}

func (extractor *HTTPHeadersExtractor) Extract(carrier interface{}) (SpanContext, error) {
	c, ok := carrier.(HTTPHeadersCarrier)
	if ok {
		ctx := HTTPHeaderExtractSpanContext{
			baggage: map[string]string{},
		}
		_ = c.ForeachKey(func(key, val string) error {
			lowerKey := strings.ToLower(key)
			switch lowerKey {
			case defaultTraceIDHeader:
				ctx.traceID = val
			case defaultSpanIDHeader:
				ctx.spanID = val
			case defaultSampleHitHeader:
				hit, _ := strconv.Atoi(val)
				switch hit {
				case 0:
					ctx.sampleStrategy = SampleStrategyNotSampled
				case 1:
					ctx.sampleStrategy = SampleStrategySampled
				}
			case defaultSampleWeightHeader:
				ctx.sampleWeight, _ = strconv.Atoi(val)
			default:
				if strings.HasPrefix(lowerKey, defaultBaggageHeaderPrefix) {
					ctx.baggage[lowerKey[len(defaultBaggageHeaderPrefix):]] = val
				}
			}
			return nil
		})
		if ctx.traceID == "" || ctx.spanID == "" {
			return nil, nil
		}
		return &ctx, nil
	}
	return nil, ErrInvalidCarrier
}
