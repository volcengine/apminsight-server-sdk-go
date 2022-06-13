// Code generated by Kitex v0.3.2. DO NOT EDIT.
package hello

import (
	"github.com/cloudwego/kitex/server"
	"github.com/volcengine/apminsight-server-sdk-go/trace/contrib/cloudwego/kitex/example/server/kitex_gen/api"
)

// NewServer creates a server.Server with the given handler and options.
func NewServer(handler api.Hello, opts ...server.Option) server.Server {
	var options []server.Option

	options = append(options, opts...)

	svr := server.NewServer(options...)
	if err := svr.RegisterService(serviceInfo(), handler); err != nil {
		panic(err)
	}
	return svr
}
