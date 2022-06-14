package aitracer

const (
	Web   = "web"
	Http  = "http"
	RPC   = "rpc"
	GRPC  = "grpc"
	MySQL = "mysql"
	Redis = "redis"
	Kafka = "kafka"
)

// log level
const (
	LogLevelTrace = "trace"
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
	LogLevelFatal = "fatal"
)

// http
const (
	HttpScheme     = "http.scheme"
	HttpMethod     = "http.method"
	HttpHost       = "http.host"
	HttpPath       = "http.path"
	HttpStatusCode = "http.status_code"
)

const (
	Go = "Go" // equal to runtime_type "Go" in service_register
)

const (
	SdkLanguage = "sdk.language" // alias of runtime_type,  values of sdk.language are always equal to runtime_type, but more comprehensible for user
	SdkVersion  = "sdk.version"
)

const (
	GoErrorType = "go.error_type"
)
