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

		// query with context. child of root
		{
			res, err := wrappedClient.WithContext(ctx).Set("key_1", "test.v6", 0).Result()
			fmt.Printf("set: %+v, %+v\n", res, err)
			res, err = wrappedClient.WithContext(ctx).Get("key_1").Result()
			fmt.Printf("get: %+v, %+v\n", res, err)
		}

		// pipeline. child of root
		{
			pipeline := wrappedClient.WithContext(ctx).Pipeline()
			for _, key := range []string{"foo", "bar", "key_1"} {
				pipeline.Get(key)
			}
			cmds, _ := pipeline.Exec()
			for i, c := range cmds {
				res, err := c.(*redis.StringCmd).Result()
				fmt.Printf("pipeline result [%d]: res=%+v, err=%+v\n", i, res, err)
			}
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
