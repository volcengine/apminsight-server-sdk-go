package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/client/callopt"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
	"github.com/volcengine/apminsight-server-sdk-go/trace/contrib/cloudwego/kitex"
	"github.com/volcengine/apminsight-server-sdk-go/trace/contrib/cloudwego/kitex/example/server/kitex_gen/api"
	"github.com/volcengine/apminsight-server-sdk-go/trace/contrib/cloudwego/kitex/example/server/kitex_gen/api/hello"
)

func main() {
	opts := make([]aitracer.TracerOption, 0)
	opts = append(opts, aitracer.WithMetrics(true))
	opts = append(opts, aitracer.WithLogSender(true))
	opts = append(opts, aitracer.WithLogger(&logger{}))

	tracer := aitracer.NewTracer(
		aitracer.RPC, "kite_client_service", opts...,
	)
	tracer.Start()
	defer tracer.Stop()
	aitracer.SetGlobalTracer(tracer)

	// mock server span
	ctx := context.Background()
	span, ctxWithSpan := tracer.StartServerSpanFromContext(ctx, "client_test", aitracer.ServerResourceAs("caller.main"))
	defer span.Finish()

	// client span
	{
		c, err := hello.NewClient("example_service", //set destService. important
			client.WithHostPorts("0.0.0.0:8888"),
			client.WithSuite(kitex.NewClientSuite(tracer)))
		if err != nil {
			log.Fatal(err)
		}

		req := &api.Request{Message: "Is trace on?"}

		resp, err := c.Echo(ctxWithSpan, req, callopt.WithRPCTimeout(3*time.Second))

		if err != nil {
			log.Println(err)
		}
		log.Println(resp)

	}
}

type logger struct{}

func (l *logger) Debug(format string, args ...interface{}) {
	fmt.Printf("[Debug] format %+v\n", args)
}
func (l *logger) Info(format string, args ...interface{}) {
	fmt.Printf("[Info] format %+v\n", args)
}
func (l *logger) Error(format string, args ...interface{}) {
	fmt.Printf("[Error] format %+v\n", args)
}
