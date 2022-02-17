package aitracer

import (
	"context"
	"errors"
	"time"

	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/logger"
)

type StartSpanConfig struct {
	spanType spanType

	parentSpanContext SpanContext

	ServerResource string

	ClientResource    string
	ClientService     string
	ClientServiceType string
}

type StartSpanOption func(*StartSpanConfig)

type FinishSpanOption struct {
	FinishTime time.Time
	Status     int64
}

type Span interface {
	Finish()
	FinishWithOption(opt FinishSpanOption)
	Context() SpanContext

	SetBaggageItem(restrictedKey, value string) Span
	BaggageItem(restrictedKey string) string

	SetTag(key string, value interface{}) Span
	SetTagString(key string, value string) Span
	SetTagFloat64(key string, value float64) Span
	SetTagInt64(key string, value int64) Span
}

type SampleStrategy byte

const (
	SampleStrategyUnknown SampleStrategy = iota
	SampleStrategyNotSampled
	SampleStrategySampled
)

type SpanContext interface {
	TraceID() string
	SpanID() string
	Sample() (strategy SampleStrategy, weight int)
	ForeachBaggageItem(func(string, string) bool)
}

var (
	ErrUnsupportedFormat = errors.New("Unknown or unsupported Inject/Extract format")
	ErrInvalidCarrier    = errors.New("Invalid Inject/Extract carrier")
)

type Injector interface {
	Inject(sc SpanContext, carrier interface{}) error
}

type Extractor interface {
	Extract(carrier interface{}) (SpanContext, error)
}

type PropagatorConfig struct {
	Format    interface{}
	Injector  Injector
	Extractor Extractor
}

type TracerConfig struct {
	ServiceType string
	Service     string

	SenderChanSize int
	SenderSock     string
	SenderNumber   int

	Logger logger.Logger

	EnableMetric bool
	MetricSock   string

	EnableLogSender   bool
	LogSenderSock     string
	LogSenderNumber   int
	LogSenderChanSize int

	SettingsFetcherSock string

	PropagatorConfigs []PropagatorConfig

	ServerRegisterSock string

	ContextAdapter func(context.Context) context.Context
}

type TracerOption func(*TracerConfig)

func WithService(serviceType string, service string) TracerOption {
	return func(config *TracerConfig) {
		config.ServiceType = serviceType
		config.Service = service
	}
}

func WithSenderChanSize(chanSize int) TracerOption {
	return func(config *TracerConfig) {
		config.SenderChanSize = chanSize
	}
}

func WithSenderSock(senderSock string) TracerOption {
	return func(config *TracerConfig) {
		config.SenderSock = senderSock
	}
}

func WithSenderNumber(senderNumber int) TracerOption {
	return func(config *TracerConfig) {
		config.SenderNumber = senderNumber
	}
}

func WithLogger(logger logger.Logger) TracerOption {
	return func(config *TracerConfig) {
		config.Logger = logger
	}
}

func WithMetrics(enable bool) TracerOption {
	return func(config *TracerConfig) {
		config.EnableMetric = enable
	}
}

func WithMetricsAddress(metricAddress string) TracerOption {
	return func(config *TracerConfig) {
		config.MetricSock = metricAddress
	}
}

func WithLogSender(enable bool) TracerOption {
	return func(config *TracerConfig) {
		config.EnableLogSender = enable
	}
}

func WithPropagator(format interface{}, injector Injector, extractor Extractor) TracerOption {
	return func(config *TracerConfig) {
		config.PropagatorConfigs = append(config.PropagatorConfigs, PropagatorConfig{
			Format:    format,
			Injector:  injector,
			Extractor: extractor,
		})
	}
}

func WithContextAdapter(contextAdapter func(context.Context) context.Context) TracerOption {
	return func(config *TracerConfig) {
		config.ContextAdapter = contextAdapter
	}
}

type LogData struct {
	Message   []byte
	Timestamp time.Time
	FileName  string
	FileLine  int64
	LogLevel  string
	Source    string
}

type Tracer interface {
	Start()
	Extract(format interface{}, carrier interface{}) (SpanContext, error)
	Inject(sc SpanContext, format interface{}, carrier interface{}) error

	StartServerSpan(operationName string, opts ...StartSpanOption) Span
	StartServerSpanFromContext(ctx context.Context, operationName string, opts ...StartSpanOption) (Span, context.Context)

	StartClientSpan(operationName string, opts ...StartSpanOption) Span
	StartClientSpanFromContext(ctx context.Context, operationName string, opts ...StartSpanOption) (Span, context.Context)

	StartSpan(operationName string, opts ...StartSpanOption) Span
	StartSpanFromContext(ctx context.Context, operationName string, opts ...StartSpanOption) (Span, context.Context)

	Log(ctx context.Context, data LogData)
	Stop()
}

// 使用context传播

type spanContextKey struct{}

var (
	activeSpanContextKey spanContextKey
)

// Deprecated. use aitracer.GetSpanFromContext instead
func GetSpanFromContext(ctx context.Context) Span {
	if ctx == nil {
		return nil
	}
	s, _ := ctx.Value(activeSpanContextKey).(Span)
	return s
}

func ContextWithSpan(ctx context.Context, sc Span) context.Context {
	return context.WithValue(ctx, activeSpanContextKey, sc)
}

func ChildOf(sc SpanContext) StartSpanOption {
	return func(config *StartSpanConfig) {
		config.parentSpanContext = sc
	}
}

func ServerResourceAs(resource string) StartSpanOption {
	return func(config *StartSpanConfig) {
		config.ServerResource = resource
	}
}

func ClientResourceAs(clientServiceType string, clientService string, clientResource string) StartSpanOption {
	return func(config *StartSpanConfig) {
		config.ClientServiceType = clientServiceType
		config.ClientService = clientService
		config.ClientResource = clientResource
	}
}
