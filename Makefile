idlpath=$(GOBYTESRC)/byteapm/server_proto

gen_trace_pb:
	protoc -I $(idlpath)/pb/trace  $(idlpath)/pb/trace/internal_trace.proto --gogofaster_out=./trace/aitracer/trace_sender/trace_models
