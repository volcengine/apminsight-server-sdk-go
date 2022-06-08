package trace_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
)

func TestBinaryPropagation(t *testing.T) {
	tr := aitracer.NewTracer(aitracer.Http, "service.test")
	tr.Start()
	aitracer.SetGlobalTracer(tr)

	in := make([]aitracer.SpanContext, 0)
	res := make([]aitracer.SpanContext, 0)
	for i := 0; i < 10; i++ {
		span := aitracer.StartServerSpan("test")
		span.SetBaggageItem("target", "my_service")
		spanCtx := span.Context()
		in = append(in, spanCtx)

		// inject
		injector := aitracer.BinaryCarrierInjector{}
		carrier := bytes.NewBuffer(make([]byte, 0))
		injector.Inject(span.Context(), aitracer.BinaryCarrier(carrier))

		// extract
		extractor := aitracer.BinaryCarrierExtractor{}
		b := carrier.Bytes()
		exCarrier := bytes.NewBuffer(b)
		exSpanCtx, _ := extractor.Extract(exCarrier)
		res = append(res, exSpanCtx)

	}

	for i, exSpanCtx := range res {
		fmt.Printf("=====%d=====\n", i)
		spanCtx := in[i]
		baggage := make(map[string]string)
		spanCtx.ForeachBaggageItem(func(k string, v string) bool {
			baggage[k] = v
			return true
		})
		s, w := spanCtx.Sample()
		fmt.Printf("in: traceId=%s, spanId=%s, sample=%+v %+v, baggage=%+v \n", spanCtx.TraceID(), spanCtx.SpanID(), s, w, baggage)

		exBaggage := make(map[string]string)
		exSpanCtx.ForeachBaggageItem(func(k string, v string) bool {
			exBaggage[k] = v
			return true
		})
		exs, exw := exSpanCtx.Sample()
		fmt.Printf("ex: traceId=%s, spanId=%s, sample=%+v %+v, baggage=%+v \n", exSpanCtx.TraceID(), exSpanCtx.SpanID(), exs, exw, exBaggage)
	}
}
