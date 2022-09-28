package aitracer

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/volcengine/apminsight-server-sdk-go/metrics"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/id_generator"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/log_collector"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/log_collector/log_models"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/logger"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/runtime"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/service_register"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/service_register/register_utils"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/trace_sampler"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/trace_sender"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/trace_sender/trace_models"
	"github.com/volcengine/apminsight-server-sdk-go/trace/internal/settings_fetcher"
	"github.com/volcengine/apminsight-server-sdk-go/trace/internal/settings_fetcher/settings_models"
)

var (
	_ Tracer = &tracer{}
)

type tracer struct {
	logger logger.Logger

	metricsClient *metrics.MetricsClient

	serviceType string
	service     string

	idGenerator *id_generator.IdGenerator

	logCollector *log_collector.LogCollector

	injects    map[interface{}]Injector
	extractors map[interface{}]Extractor

	traceChan chan *trace_models.Trace

	traceSenders []*trace_sender.TraceSender

	serviceRegister *service_register.Register

	traceSampler *trace_sampler.Sampler

	settingsFetcher *settings_fetcher.Fetcher

	runtimeMonitor *runtime.Monitor

	containerId string
	instanceId  string

	dynamicConfig atomic.Value

	contextAdapter func(context.Context) context.Context
}

func NewTracer(serviceType, service string, opts ...TracerOption) Tracer {
	config := newDefaultTracerConfig()
	for _, opt := range opts {
		opt(&config)
	}
	t := &tracer{
		serviceType: serviceType,
		service:     service,

		logger: config.Logger,

		idGenerator: id_generator.New(),

		traceSampler: trace_sampler.New(),

		traceChan: make(chan *trace_models.Trace, config.SenderChanSize),

		injects:    map[interface{}]Injector{},
		extractors: map[interface{}]Extractor{},
	}
	{
		t.instanceId = register_utils.GetInstanceID()
	}
	info, _ := register_utils.GetInfo()
	if len(info.ContainerId) != 0 {
		t.containerId = info.ContainerId
	}
	if config.EnableLogSender {
		config := log_collector.LogCollectorConfig{
			Sock:         config.LogSenderSock,
			WorkerNumber: config.LogSenderNumber,
			ChanSize:     config.LogSenderChanSize,
			Logger:       config.Logger,
		}
		t.logCollector = log_collector.NewLogCollector(config)
	}
	if config.EnableRuntimeMetric {
		var mc *metrics.MetricsClient
		if config.MetricSock != "" {
			mc = metrics.NewMetricClient(metrics.WithAddress(config.MetricSock))
		} else {
			mc = metrics.NewMetricClient()
		}
		t.runtimeMonitor = runtime.NewMonitor(serviceType, service, mc)
	}
	if config.EnableMetric {
		if config.MetricSock != "" {
			t.metricsClient = metrics.NewMetricClient(metrics.WithAddress(config.MetricSock))
		} else {
			t.metricsClient = metrics.NewMetricClient()
		}
	}
	t.serviceRegister = service_register.NewRegister(service, serviceType, t.instanceId, config.ServerRegisterSock, time.Second*30, t.logger)
	for _, p := range config.PropagatorConfigs {
		t.injects[p.Format] = p.Injector
		t.extractors[p.Format] = p.Extractor
	}
	for i := 0; i < config.SenderNumber; i++ {
		t.traceSenders = append(t.traceSenders, trace_sender.NewSender(config.SenderSock, t.traceChan, t.logger))
	}
	t.settingsFetcher = settings_fetcher.NewSettingsFetcher(settings_fetcher.SettingsFetcherConfig{
		Service: t.service,
		Logger:  t.logger,
		Sock:    config.SettingsFetcherSock,
		Notifier: []func(*settings_models.Settings){
			t.handleSettings,
		},
	})
	t.contextAdapter = config.ContextAdapter
	return t
}

func (t *tracer) Start() {
	t.settingsFetcher.Start()
	if t.logCollector != nil {
		t.logCollector.Start()
	}
	t.idGenerator.Start()
	if t.metricsClient != nil {
		t.metricsClient.Start()
	}
	for _, sender := range t.traceSenders {
		sender.Start()
	}
	t.serviceRegister.Start()
	t.runtimeMonitor.Start()
}

func (t *tracer) Stop() {
	t.serviceRegister.Stop()
	t.runtimeMonitor.Close()
	close(t.traceChan)
	for _, sender := range t.traceSenders {
		sender.WaitStop()
	}
	if t.metricsClient != nil {
		t.metricsClient.Close()
	}
	if t.logCollector != nil {
		t.logCollector.Stop()
	}
	t.settingsFetcher.Stop()
}

func (t *tracer) Extract(format interface{}, carrier interface{}) (SpanContext, error) {
	extractor, ok := t.extractors[format]
	if !ok {
		return nil, ErrUnsupportedFormat
	}
	return extractor.Extract(carrier)
}

func (t *tracer) Inject(sc SpanContext, format interface{}, carrier interface{}) error {
	injector, ok := t.injects[format]
	if !ok {
		return ErrUnsupportedFormat
	}
	return injector.Inject(sc, carrier)
}

func (t *tracer) Log(ctx context.Context, logData LogData) {
	if t.logCollector == nil {
		return
	}
	logItem := log_models.Log{}
	span := t.GetSpanFromContext(ctx)
	if span != nil {
		sc := span.Context()
		if sc != nil {
			logItem.TraceId = sc.TraceID()
		}
	}
	logItem.LogLevel = logData.LogLevel
	logItem.FileName = logData.FileName
	logItem.FileLine = logData.FileLine
	logItem.Source = logData.Source
	logItem.Timestamp = logData.Timestamp.Unix()*1e3 + int64(logData.Timestamp.Nanosecond()/1e6)
	logItem.Message = logData.Message
	logItem.Service = t.service
	logItem.ContainerId = t.containerId
	t.logCollector.Send(&logItem)
}

func (t *tracer) StartServerSpan(operationName string, opts ...StartSpanOption) Span {
	return t.startSpan(operationName, StartSpanConfig{spanType: serverSpanType}, opts...)
}

func (t *tracer) StartServerSpanFromContext(ctx context.Context, operationName string, opts ...StartSpanOption) (Span, context.Context) {
	return t.startSpanFromContext(ctx, operationName, StartSpanConfig{spanType: serverSpanType}, opts...)
}

func (t *tracer) StartClientSpan(operationName string, opts ...StartSpanOption) Span {
	return t.startSpan(operationName, StartSpanConfig{spanType: clientSpanType}, opts...)
}

func (t *tracer) StartClientSpanFromContext(ctx context.Context, operationName string, opts ...StartSpanOption) (Span, context.Context) {
	return t.startSpanFromContext(ctx, operationName, StartSpanConfig{spanType: clientSpanType}, opts...)
}

func (t *tracer) StartSpan(operationName string, opts ...StartSpanOption) Span {
	return t.startSpan(operationName, StartSpanConfig{spanType: commonSpanType}, opts...)
}

func (t *tracer) startSpan(operationName string, defaultConfig StartSpanConfig, opts ...StartSpanOption) Span {
	for _, opt := range opts {
		opt(&defaultConfig)
	}

	startTime := time.Now()
	spanId := t.idGenerator.GenId()
	parentSpanID := ""
	var (
		traceCtx        *traceContext
		incomingBaggage map[string]string
	)

	traceResource := defaultConfig.ServerResource
	if traceResource == "" {
		traceResource = "empty"
	}

	if defaultConfig.parentSpanContext != nil {
		parentSpanContext, ok := defaultConfig.parentSpanContext.(*spanContext)
		if ok && parentSpanContext != nil {
			// 内存
			parentSpanID = parentSpanContext.spanID
			traceCtx = parentSpanContext.traceContext
		} else {
			// 传播
			parentSpanID = defaultConfig.parentSpanContext.SpanID()
			traceCtx = &traceContext{
				traceID:       defaultConfig.parentSpanContext.TraceID(),
				resource:      traceResource,
				clientSampled: defaultConfig.parentSpanContext.ClientSampled(),
				sampleFlags:   defaultConfig.parentSpanContext.SampleFlags(),
				tracer:        t,
			}
			traceCtx.sampleStrategy, traceCtx.sampleWeight = defaultConfig.parentSpanContext.Sample()
		}

		// copy parent baggage to new span
		defaultConfig.parentSpanContext.ForeachBaggageItem(func(k string, v string) bool {
			if incomingBaggage == nil {
				incomingBaggage = make(map[string]string)
			}
			incomingBaggage[k] = v
			return true
		})
	} else {
		traceCtx = &traceContext{
			traceID:     t.idGenerator.GenId(),
			resource:    traceResource,
			sampleFlags: SampleFlagsUnknown,
			tracer:      t,
		}
	}
	if traceCtx.sampleStrategy == SampleStrategyUnknown {
		sampled, weight := t.traceSampler.Sample()
		if sampled {
			traceCtx.sampleStrategy = SampleStrategySampled
			traceCtx.sampleWeight = weight
		} else {
			traceCtx.sampleStrategy = SampleStrategyNotSampled
		}
	}

	// sampleStrategy has been determined so sampleFlags can be assigned
	if traceCtx.sampleFlags == SampleFlagsUnknown {
		if traceCtx.sampleStrategy == SampleStrategyNotSampled {
			if traceCtx.clientSampled {
				traceCtx.sampleFlags = SampleFlagsClientSampled
			} else {
				traceCtx.sampleFlags = SampleFlagsNotSampled
			}
		} else if traceCtx.sampleStrategy == SampleStrategySampled {
			if traceCtx.clientSampled {
				traceCtx.sampleFlags = SampleFlagsClientAndServerSampled
			} else {
				traceCtx.sampleFlags = SampleFlagsServerSampled
			}
		}
	}

	s := &span{
		spanType: defaultConfig.spanType,

		operationName: operationName,

		serverResource: defaultConfig.ServerResource,

		clientType:     defaultConfig.ClientServiceType,
		clientService:  defaultConfig.ClientService,
		clientResource: defaultConfig.ClientResource,

		startTime: startTime,
		spanContext: spanContext{
			spanID:       spanId,
			parentSpanID: parentSpanID,
			traceContext: traceCtx,
			baggage:      incomingBaggage,
		},
	}
	s.spanContext.traceContext.addSpan(s)
	return s
}

func (t *tracer) StartSpanFromContext(ctx context.Context, operationName string, opts ...StartSpanOption) (Span, context.Context) {
	return t.startSpanFromContext(ctx, operationName, StartSpanConfig{}, opts...)
}

func (t *tracer) startSpanFromContext(ctx context.Context, operationName string, defaultConfig StartSpanConfig, opts ...StartSpanOption) (Span, context.Context) {
	s := t.GetSpanFromContext(ctx)
	if s != nil {
		ChildOf(s.Context())(&defaultConfig)
	}
	span := t.startSpan(operationName, defaultConfig, opts...)
	return span, ContextWithSpan(ctx, span)
}

func (t *tracer) GetSpanFromContext(ctx context.Context) Span {
	if ctx == nil {
		return nil
	}
	if t.contextAdapter != nil {
		if c := t.contextAdapter(ctx); c != nil {
			s, _ := c.Value(activeSpanContextKey).(Span)
			return s
		}
	}
	s, _ := ctx.Value(activeSpanContextKey).(Span)
	return s
}

func (t *tracer) handleSettings(settings *settings_models.Settings) {
	t.handleSettingsForSampler(settings)
	t.handleSettingsForDynamicConfig(settings)
}

func (t *tracer) handleSettingsForSampler(settings *settings_models.Settings) {
	if settings == nil {
		return
	}
	if settings.Trace == nil {
		return
	}
	if settings.Trace.SampleConfig == nil {
		return
	}
	samplerConfig := trace_sampler.SamplerConfig{}
	switch settings.Trace.SampleConfig.Strategy {
	case settings_models.SampleStrategy_ALL:
		samplerConfig.Strategy = trace_sampler.SamplerStrategyAll
	case settings_models.SampleStrategy_SAMPLE_RATIO:
		samplerConfig.Strategy = trace_sampler.SamplerStrategyRatio
	case settings_models.SampleStrategy_RATE_LIMIT:
		samplerConfig.Strategy = trace_sampler.SamplerStrategyRateLimit
	default:
		return
	}
	samplerConfig.Value = settings.Trace.SampleConfig.Value
	t.traceSampler.RefreshConfig(samplerConfig)
}

type tracerDynamicConfig struct {
	DbSlowQuery time.Duration
}

func (t *tracer) getDynamicConfig() *tracerDynamicConfig {
	config, _ := t.dynamicConfig.Load().(*tracerDynamicConfig)
	return config
}

func (t *tracer) handleSettingsForDynamicConfig(settings *settings_models.Settings) {
	if settings == nil {
		return
	}
	newDynamicConfig := &tracerDynamicConfig{}
	if settings.Db != nil {
		newDynamicConfig.DbSlowQuery = time.Duration(settings.Db.SlowQueryMillseconds) * time.Millisecond
	}
	t.dynamicConfig.Store(newDynamicConfig)
}

// collect send trace via UDS
func (t *tracer) collect(tc *traceContext) {
	// 发送详情
	if tc.sampleStrategy == SampleStrategySampled || tc.sampleFlags.Sampled() {
		t.emitTrace(tc)
	}
}

func (t *tracer) emitTrace(tc *traceContext) {
	trace := trace_models.Trace{
		ServiceType: t.serviceType,
		Service:     t.service,
		ContainerId: t.containerId,
		InstanceId:  t.instanceId,
		TraceId:     tc.traceID,
	}
	if len(tc.spans) == 0 {
		return
	}
	serverResource := ""
	serverSpan := tc.spans[0]
	if serverSpan.spanType == serverSpanType {
		serverResource = serverSpan.serverResource
	}
	for _, span := range tc.spans {
		if span == nil {
			continue
		}
		if span.serverResource == "" {
			span.serverResource = serverResource
		}
		spanType := trace_models.SpanType_Common
		switch span.spanType {
		case commonSpanType:
		case serverSpanType:
			spanType = trace_models.SpanType_Server
		case clientSpanType:
			spanType = trace_models.SpanType_Client
		default:
			continue
		}
		errorInfoList := make([]*trace_models.ErrorInfo, 0, len(span.ErrorInfoList))
		for _, errorInfo := range span.ErrorInfoList {
			if errorInfo == nil {
				continue
			}
			errorType := trace_models.ErrorType_UncaughtException
			switch errorInfo.ErrorKind {
			case ErrorKindDbError:
				errorType = trace_models.ErrorType_DbError
			case ErrorKindExternalServiceError:
				errorType = trace_models.ErrorType_ExternalServiceError
			case ErrorKindHttpCodeError:
				errorType = trace_models.ErrorType_HttpCodeError
			case ErrorKindNoSqlError:
				errorType = trace_models.ErrorType_NoSqlError
			case ErrorKindMqError:
				errorType = trace_models.ErrorType_MqError
			case ErrorKindUncaughtException:
				errorType = trace_models.ErrorType_UncaughtException
			case ErrorKindBusinessError:
				errorType = trace_models.ErrorType_BusinessError
			case ErrorKindPanic:
				errorType = trace_models.ErrorType_Panic
			}
			errorInfoList = append(errorInfoList, &trace_models.ErrorInfo{
				ErrorKind:      errorType,
				ErrorMessage:   errorInfo.ErrorMessage,
				ErrorStack:     errorInfo.ErrorStack,
				ErrorOccurTime: errorInfo.ErrorOccurTimeMilliSec,
				ErrorTags:      errorInfo.ErrorTags,
			})
		}

		// set appId/origin in tags
		tagsStr := span.tagsString
		span.spanContext.ForeachBaggageItem(func(key, value string) bool {
			if key == defaultAppIDBaggageKey || key == defaultOriginBaggageKey {
				if tagsStr == nil {
					tagsStr = make(map[string]string)
				}
				tagsStr[key] = value
			}
			return true
		})

		trace.Spans = append(trace.Spans, &trace_models.Span{
			SpanType: spanType,

			SpanId:        span.spanContext.spanID,
			ParentSpanId:  span.spanContext.parentSpanID,
			OperationName: span.operationName,

			StartTimeMillisecond: span.startTime.Unix()*1e3 + int64(span.startTime.Nanosecond())/1e6,
			EndTimeMillisecond:   span.finishTime.Unix()*1e3 + int64(span.finishTime.Nanosecond())/1e6,
			DurationMicroseconds: span.duration.Microseconds(),

			StartTimeMicrosecond: span.startTime.Unix()*1e6 + int64(span.startTime.Nanosecond())/1e3,

			Status: span.status,

			ParamInt:    span.tagsInt64,
			ParamFloat:  span.tagsFloat64,
			ParamString: tagsStr,

			Resource: span.serverResource,

			CallServiceType: span.clientType,
			CallService:     span.clientService,
			CallResource:    span.clientResource,
			ErrorInfoList:   errorInfoList,
		})
	}
	select {
	case t.traceChan <- &trace: //non-blocking. otherwise span.Finish could be blocked.
	default:
	}
}
