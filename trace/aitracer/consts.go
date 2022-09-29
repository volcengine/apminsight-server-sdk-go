package aitracer

const (
	Web     = "web"
	Http    = "http"
	RPC     = "rpc"
	GRPC    = "grpc"
	MySQL   = "mysql"
	Redis   = "redis"
	Kafka   = "kafka"
	Mongodb = "mongodb"
)

// log level
const (
	LogLevelTrace  = "trace"
	LogLevelDebug  = "debug"
	LogLevelInfo   = "info"
	LogLevelNotice = "notice"
	LogLevelWarn   = "warn"
	LogLevelError  = "error"
	LogLevelFatal  = "fatal"
)

// http field
const (
	HttpScheme     = "http.scheme"
	HttpMethod     = "http.method"
	HttpHost       = "http.host"
	HttpPath       = "http.path"
	HttpStatusCode = "http.status_code"
)

const (
	DbStatement = "db.statement"
)

const (
	StatusCodeOK    = 0
	StatusCodeError = 1
)
