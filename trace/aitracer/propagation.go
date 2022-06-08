package aitracer

import (
	"bytes"
	"encoding/binary"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type BuiltinFormat byte

const (
	HTTPHeaders BuiltinFormat = iota
	Binary
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

type BinaryExtractSpanContext struct {
	traceId        string
	spanId         string
	sampleStrategy SampleStrategy
	sampleWeight   int
	baggage        map[string]string
}

func (sc *BinaryExtractSpanContext) TraceID() string {
	return sc.traceId
}

func (sc *BinaryExtractSpanContext) SpanID() string {
	return sc.spanId
}

func (sc *BinaryExtractSpanContext) ForeachBaggageItem(f func(string, string) bool) {
	for k, v := range sc.baggage {
		if !f(k, v) {
			break
		}
	}
}

type BinaryCarrier = *bytes.Buffer

type BinaryCarrierInjector struct{}

func (i *BinaryCarrierInjector) Inject(sc SpanContext, carrier interface{}) error {
	ioCarrier, ok := carrier.(io.Writer)
	if !ok || ioCarrier == nil {
		return ErrInvalidCarrier
	}

	// traceId
	if err := writeString(ioCarrier, sc.TraceID()); err != nil {
		return err
	}

	// spanId
	if err := writeString(ioCarrier, sc.SpanID()); err != nil {
		return err
	}

	// sample
	strategy, weight := sc.Sample()
	switch strategy {
	case SampleStrategySampled:
		if err := binary.Write(ioCarrier, binary.LittleEndian, int32(1)); err != nil {
			return err
		}
		if err := binary.Write(ioCarrier, binary.LittleEndian, int32(weight)); err != nil {
			return err
		}
	case SampleStrategyNotSampled:
		if err := binary.Write(ioCarrier, binary.LittleEndian, int32(0)); err != nil {
			return err
		}
		if err := binary.Write(ioCarrier, binary.LittleEndian, int32(weight)); err != nil {
			return err
		}
	}

	// Baggage field
	cnt := int32(0)
	sc.ForeachBaggageItem(func(k string, v string) bool {
		cnt++
		return true
	})
	if err := binary.Write(ioCarrier, binary.LittleEndian, cnt); err != nil {
		return err
	}
	// write Baggage key-value
	sc.ForeachBaggageItem(func(k string, v string) bool {
		if err := writeString(ioCarrier, k); err != nil {
			return false
		}
		if err := writeString(ioCarrier, v); err != nil {
			return false
		}
		return true
	})
	return nil
}

type BinaryCarrierExtractor struct{}

func (e *BinaryCarrierExtractor) Extract(carrier interface{}) (SpanContext, error) {
	ioCarrier, ok := carrier.(io.Reader)
	if !ok {
		return nil, ErrInvalidCarrier
	}

	// traceId
	traceId, err := readString(ioCarrier)
	if err != nil {
		return nil, err
	}

	// spanId
	spanId, err := readString(ioCarrier)
	if err != nil {
		return nil, err
	}

	// sample filed
	sampleStrategy := SampleStrategyNotSampled
	sampleHit := int32(0)
	sampleWeight := int32(0)
	if err := binary.Read(ioCarrier, binary.LittleEndian, &sampleHit); err != nil {
		return nil, err
	}
	if sampleHit == 1 {
		sampleStrategy = SampleStrategySampled
	}
	if err := binary.Read(ioCarrier, binary.LittleEndian, &sampleWeight); err != nil {
		return nil, err
	}

	//  Baggage field
	cnt := int32(0)
	if err := binary.Read(ioCarrier, binary.LittleEndian, &cnt); err != nil {
		return nil, err
	}
	baggage := make(map[string]string)
	for i := 0; i < int(cnt); i++ {
		key, err := readString(ioCarrier)
		if err != nil {
			return nil, err
		}
		value, err := readString(ioCarrier)
		if err != nil {
			return nil, err
		}
		baggage[key] = value
	}

	spanContext := BinaryExtractSpanContext{
		traceId:        traceId,
		spanId:         spanId,
		sampleStrategy: sampleStrategy,
		sampleWeight:   int(sampleWeight),
		baggage:        baggage,
	}
	return &spanContext, nil
}

func (sc *BinaryExtractSpanContext) Sample() (SampleStrategy, int) {
	return sc.sampleStrategy, sc.sampleWeight
}

func writeString(writer io.Writer, s string) error {
	if err := binary.Write(writer, binary.LittleEndian, int32(len(s))); err != nil {
		return err
	}
	if n, err := io.WriteString(writer, s); err != nil || n != len(s) {
		return err
	}
	return nil
}

func readString(reader io.Reader) (string, error) {
	l := int32(0)
	if err := binary.Read(reader, binary.LittleEndian, &l); err != nil {
		return "", err
	}
	sb := strings.Builder{}
	sb.Grow(int(l))
	if n, err := io.CopyN(&sb, reader, int64(l)); err != nil || int32(n) != l {
		return "", err
	}
	return sb.String(), nil
}
