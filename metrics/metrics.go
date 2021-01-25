package metrics

var (
	logfunc func(string, ...interface{})

	defaultAddress      = "/var/run/apminsight/metrics.sock"
	defaultMetricClient *MetricsClient
)

type Config struct {
	prefix  string
	address string
}

type ClientOption func(config *Config)

func WithPrefix(prefix string) ClientOption {
	return func(config *Config) {
		config.prefix = prefix
	}
}

func WithAddress(address string) ClientOption {
	return func(config *Config) {
		config.address = address
	}
}

func Init(options ...ClientOption) {
	defaultMetricClient = NewMetricClient(options...)
	defaultMetricClient.Start()
}

func Close(clients ...*MetricsClient) {
	if defaultMetricClient != nil {
		defaultMetricClient.Close()
	}
	if len(clients) > 0 {
		for _, client := range clients {
			if client != nil {
				client.Close()
			}
		}
	}
}

func SetLogFunc(_logfunc func(string, ...interface{})) {
	logfunc = _logfunc
}

func EmitCounter(name string, value float64, tags map[string]string) error {
	return defaultMetricClient.EmitCounter(name, value, tags)
}

func EmitTimer(name string, value float64, tags map[string]string) error {
	return defaultMetricClient.EmitTimer(name, value, tags)
}

func EmitGauge(name string, value float64, tags map[string]string) error {
	return defaultMetricClient.EmitGauge(name, value, tags)
}
