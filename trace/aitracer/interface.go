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
	FinishTime          time.Time
	Status              int64
	DisablePanicCapture bool // By default panic is captured, set True to disable
}

type RecordConfig struct {
	ErrorKind   ErrorKind
	RecordStack bool   // record stack is expensive and is disabled by default
	Stack       string // stack passed in. which is useful where RecordError being called is different from error occurred
}

func NewDefaultRecordConfig() RecordConfig {
	return RecordConfig{ErrorKind: ErrorKindBusinessError}
}

type ErrorKind int32

const (
	ErrorKindDbError ErrorKind = iota
	ErrorKindExternalServiceError
	ErrorKindHttpCodeError
	ErrorKindNoSqlError
	ErrorKindMqError
	ErrorKindUncaughtException
	ErrorKindBusinessError
	ErrorKindPanic
)

type RecordOption func(*RecordConfig)

func WithErrorKind(t ErrorKind) RecordOption {
	return func(cfg *RecordConfig) {
		cfg.ErrorKind = t
	}
}

func WithRecordStack(b bool) RecordOption {
	return func(cfg *RecordConfig) {
		cfg.RecordStack = b
	}
}

func WithStack(stack string) RecordOption {
	return func(cgf *RecordConfig) {
		cgf.Stack = stack
	}
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

	RecordError(err error, opt ...RecordOption)
	SetStatus(status int64)
}

type SampleStrategy byte

const (
	SampleStrategyUnknown SampleStrategy = iota
	SampleStrategyNotSampled
	SampleStrategySampled
)

// SampleFlags intersects with other tracing systems, should be explicitly defined.
type SampleFlags int32

const (
	SampleFlagsUnknown                SampleFlags = -1 // SampleFlags is determined by SampleStrategy and clientSampled
	SampleFlagsNotSampled             SampleFlags = 0
	SampleFlagsClientSampled          SampleFlags = 1
	SampleFlagsServerSampled          SampleFlags = 2
	SampleFlagsClientAndServerSampled SampleFlags = 3
)

func (s SampleFlags) ToString() string {
	switch s {
	case SampleFlagsNotSampled:
		return "0"
	case SampleFlagsClientSampled:
		return "1"
	case SampleFlagsServerSampled:
		return "2"
	case SampleFlagsClientAndServerSampled:
		return "3"
	default:
		return "-1"
	}
}

func SampleFlagsFromInt32(flag int32) SampleFlags {
	switch flag {
	case 0:
		return SampleFlagsNotSampled
	case 1:
		return SampleFlagsClientSampled
	case 2:
		return SampleFlagsServerSampled
	case 3:
		return SampleFlagsClientAndServerSampled
	default:
		return SampleFlagsUnknown
	}
}

func (s SampleFlags) Sampled() bool {
	return s > 0
}

type SpanContext interface {
	TraceID() string
	SpanID() string
	Sample() (strategy SampleStrategy, weight int)
	ClientSampled() bool
	SampleFlags() SampleFlags
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

	SenderChanSize   int
	SenderSock       string
	SenderStreamSock string
	SenderNumber     int

	Logger logger.Logger

	EnableMetric bool
	MetricSock   string

	EnableLogSender bool
	LogSenderDebug  bool // for safety, can not use incoming logger.Logger in LogCollector

	LogSenderSock       string
	LogSenderStreamSock string
	LogSenderNumber     int
	LogSenderChanSize   int

	EnableRuntimeMetric bool

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

func WithSenderStreamSock(senderStreamSock string) TracerOption {
	return func(config *TracerConfig) {
		config.SenderStreamSock = senderStreamSock
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

func WithLogSenderDebug(enable bool) TracerOption {
	return func(config *TracerConfig) {
		config.LogSenderDebug = enable
	}
}

func WithRuntimeMetric(enable bool) TracerOption {
	return func(config *TracerConfig) {
		config.EnableRuntimeMetric = enable
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
