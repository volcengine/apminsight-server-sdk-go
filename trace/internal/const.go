package internal

const (
	RuntimeTypeGo = "Go"
	Go            = "Go" // sdk language. equal to runtime_type "Go" used in service_register
)

const (
	SdkLanguage = "sdk.language" // alias of runtime_type,  values of sdk.language are always equal to runtime_type, but more comprehensible for user
	SdkVersion  = "sdk.version"
)

const (
	GoErrorType = "go.error_type"
)

const (
	AgentVersionSupportStreamSender = "1.0.28"
)
