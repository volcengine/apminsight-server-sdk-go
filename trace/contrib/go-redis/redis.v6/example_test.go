package redis_v6

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-redis/redis"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
)

func TestExample(t *testing.T) {
	opts := make([]aitracer.TracerOption, 0)
	opts = append(opts, aitracer.WithLogger(&logger{}))
	tracer := aitracer.NewTracer(
		aitracer.Http, "example_service", opts...,
	)
	tracer.Start()

	{
		// root span
		span := tracer.StartServerSpan("root")

		ctx := aitracer.ContextWithSpan(context.Background(), span)

		// new redis.TraceClient
		redisOpts := &redis.Options{Addr: "127.0.0.1:6379", Password: "", DB: 0}
		client := redis.NewClient(redisOpts)

		// wrap redis client
		wrappedClient := WrapClient(tracer, client)

		// query WithContext
		{
			// get/set are child of root span
			wrappedClient.WithContext(ctx).Get("child_root")
			wrappedClient.WithContext(ctx).Set("child_root_2", "test", 0)
		}

		{
			// doSomething is child of root span
			doSomethingSpan, ctxWithSpan := tracer.StartSpanFromContext(ctx, "doSomething")
			// child of doSomething
			wrappedClient.WithContext(ctxWithSpan).Get("child_doSomething")
			doSomethingSpan.Finish()
		}
		{
			// pipeline. child fo root span
			pipeline := wrappedClient.WithContext(ctx).Pipeline()
			for _, key := range []string{"foo", "bar"} {
				pipeline.Get(key)
			}
			_, _ = pipeline.Exec()
		}

		span.Finish() // must finish
	}

	time.Sleep(1 * time.Second) // wait to print log

}

type logger struct{}

func (l *logger) Debug(format string, args ...interface{}) {
	fmt.Printf("[Debug]"+format+"\n", args)
}
func (l *logger) Info(format string, args ...interface{}) {
	fmt.Printf("[Info]"+format+"\n", args)
}
func (l *logger) Error(format string, args ...interface{}) {
	fmt.Printf("[Error]"+format+"\n", args)
}
