package main

import (
	"log"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/cloudwego/kitex/server"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
	tracekitex "github.com/volcengine/apminsight-server-sdk-go/trace/contrib/cloudwego/kitex"
	api "github.com/volcengine/apminsight-server-sdk-go/trace/contrib/cloudwego/kitex/example/server/kitex_gen/api/hello"
)

func main() {
	// test panic capture flag
	//os.Setenv("TEST_PANIC", "1")

	opts := make([]aitracer.TracerOption, 0)
	opts = append(opts, aitracer.WithMetrics(true))
	opts = append(opts, aitracer.WithLogSender(true))
	opts = append(opts, aitracer.WithLogger(&logger{}))

	tracer := aitracer.NewTracer(
		aitracer.RPC, "example_service", opts...,
	)
	tracer.Start()
	defer tracer.Stop()
	aitracer.SetGlobalTracer(tracer)

	klog.SetLevel(klog.LevelDebug)
	svr := api.NewServer(new(HelloImpl), server.WithSuite(tracekitex.NewServerSuite(tracer)))

	err := svr.Run()

	if err != nil {
		log.Println(err.Error())
	}
}

type logger struct{}

func (l *logger) Debug(format string, args ...interface{}) {
	klog.Debugf(format, args)
}
func (l *logger) Info(format string, args ...interface{}) {
	klog.Infof(format, args)
}
func (l *logger) Error(format string, args ...interface{}) {
	klog.Errorf(format, args)
}
