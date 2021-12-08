package aitracer

import "context"

type registeredTracer struct {
	tracer       Tracer
	isRegistered bool
}

var (
	globalTracer registeredTracer
)

func SetGlobalTracer(tracer Tracer) {
	globalTracer = registeredTracer{tracer, true}
}

func GlobalTracer() Tracer {
	return globalTracer.tracer
}

func Extract(format interface{}, carrier interface{}) (SpanContext, error) {
	return globalTracer.tracer.Extract(format, carrier)
}

func Inject(sc SpanContext, format interface{}, carrier interface{}) error {
	return globalTracer.tracer.Inject(sc, format, carrier)
}

func StartServerSpan(operationName string, opts ...StartSpanOption) Span {
	return globalTracer.tracer.StartServerSpan(operationName, opts...)
}

func StartServerSpanFromContext(ctx context.Context, operationName string, opts ...StartSpanOption) (Span, context.Context) {
	return globalTracer.tracer.StartServerSpanFromContext(ctx, operationName, opts...)
}

func StartClientSpan(operationName string, opts ...StartSpanOption) Span {
	return globalTracer.tracer.StartClientSpan(operationName, opts...)
}

func StartClientSpanFromContext(ctx context.Context, operationName string, opts ...StartSpanOption) (Span, context.Context) {
	return globalTracer.tracer.StartClientSpanFromContext(ctx, operationName, opts...)
}

func StartSpan(operationName string, opts ...StartSpanOption) Span {
	return globalTracer.tracer.StartSpan(operationName, opts...)
}

func StartSpanFromContext(ctx context.Context, operationName string, opts ...StartSpanOption) (Span, context.Context) {
	return globalTracer.tracer.StartSpanFromContext(ctx, operationName, opts...)
}

func Log(ctx context.Context, data LogData) {
	globalTracer.tracer.Log(ctx, data)
}

func InitGlobalTracer(tracer Tracer) {
	SetGlobalTracer(tracer)
}

func IsGlobalTracerRegistered() bool {
	return globalTracer.isRegistered
}
