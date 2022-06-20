package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
	tracegin "github.com/volcengine/apminsight-server-sdk-go/trace/contrib/gin-gonic/gin"
	ailogrus "github.com/volcengine/apminsight-server-sdk-go/trace/contrib/sirupsen/logrus"
)

type logger struct{}

func (l *logger) Debug(format string, args ...interface{}) {
	fmt.Printf("[Debug] "+format+" %+v\n", args)
}
func (l *logger) Info(format string, args ...interface{}) {
	fmt.Printf("[Info] "+format+" %+v\n", args)
}
func (l *logger) Error(format string, args ...interface{}) {
	fmt.Printf("[Error] "+format+" %+v\n", args)
}

func main() {
	opts := make([]aitracer.TracerOption, 0)
	opts = append(opts, aitracer.WithMetrics(true))
	opts = append(opts, aitracer.WithLogSender(true))
	opts = append(opts, aitracer.WithLogger(&logger{}))

	tracer := aitracer.NewTracer(
		aitracer.Http, "example_gin_service", opts...,
	)
	tracer.Start()
	aitracer.SetGlobalTracer(tracer) // must be done
	defer func() {
		tracer.Stop()
	}()

	r := gin.Default()

	//  if gin.Version >= "1.8.1", then engin.ContextWithFallback is available,
	//  set ContextWithFallback=true so that gin.Context() can be directly used by trace,
	//  if ContextWithFallback=false then gin.Context.Request.Context() must be used
	r.ContextWithFallback = true

	r.Use(tracegin.NewMiddleware(tracer))

	logrus.SetLevel(logrus.TraceLevel)
	logrus.AddHook(ailogrus.NewHook(tracer, []logrus.Level{
		logrus.TraceLevel,
		logrus.DebugLevel,
		logrus.InfoLevel,
		logrus.WarnLevel,
		logrus.ErrorLevel,
		logrus.FatalLevel,
	}))

	r.GET("/call_rpc", caller)

	_ = r.Run("0.0.0.0:8912")
}
