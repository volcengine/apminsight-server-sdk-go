idlpath=$(GOBYTESRC)/byteapm/server_proto

gen_trace_pb:
	protoc -I $(idlpath)/pb/trace  $(idlpath)/pb/trace/internal_trace.proto --gogofaster_out=./trace/aitracer/trace_sender/trace_models

gen_profile_pb:
	protoc -I $(idlpath)/pb/profile  $(idlpath)/pb/profile/profile.proto --gogofaster_out=./trace/aiprofiler/profile_models

gen_settings_pb:
	protoc -I $(idlpath)/pb/settings  $(idlpath)/pb/settings/settings.proto --gogofaster_out=./trace/internal/settings_fetcher/settings_models
