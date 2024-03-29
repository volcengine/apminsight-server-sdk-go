package aitracer

import (
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/logger"
)

const (
	defaultSenderSock       = "/var/run/apminsight/trace.sock"
	defaultSenderStreamSock = "/var/run/apminsight/trace_stream.sock"

	defaultSenderChanSize = 1024
	defaultSenderNumber   = 4

	defaultLogSenderSock       = "/var/run/apminsight/log.sock"
	defaultLogSenderStreamSock = "/var/run/apminsight/log_stream.sock"

	defaultLogSenderChanSize = 1024
	defaultLogSenderNumber   = 4

	defaultSettingsSock = "/var/run/apminsight/comm.sock"

	defaultServiceRegisterSock = "/var/run/apminsight/comm.sock"

	defaultMetricSock = "/var/run/apminsight/metrics.sock"
)

func newDefaultTracerConfig() TracerConfig {
	return TracerConfig{
		Logger: &logger.NoopLogger{},

		SenderChanSize:   defaultSenderChanSize,
		SenderSock:       defaultSenderSock,
		SenderStreamSock: defaultSenderStreamSock,
		SenderNumber:     defaultSenderNumber,

		EnableMetric: true,
		MetricSock:   defaultMetricSock,

		EnableLogSender:     true,
		LogSenderSock:       defaultLogSenderSock,
		LogSenderStreamSock: defaultLogSenderStreamSock,
		LogSenderNumber:     defaultLogSenderNumber,
		LogSenderChanSize:   defaultLogSenderChanSize,

		EnableRuntimeMetric: true,

		SettingsFetcherSock: defaultSettingsSock,

		ServerRegisterSock: defaultServiceRegisterSock,

		PropagatorConfigs: []PropagatorConfig{
			{
				Format:    HTTPHeaders,
				Injector:  &HTTPHeadersInjector{},
				Extractor: &HTTPHeadersExtractor{},
			},
			{
				Format:    Binary,
				Injector:  &BinaryCarrierInjector{},
				Extractor: &BinaryCarrierExtractor{},
			},
		},
	}
}
