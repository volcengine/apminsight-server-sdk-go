package redis_v8

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
)

func TestExample(t *testing.T) {
	opts := make([]aitracer.TracerOption, 0)
	opts = append(opts, aitracer.WithMetrics(true))
	opts = append(opts, aitracer.WithLogSender(true))
	opts = append(opts, aitracer.WithLogger(&logger{}))
	tracer := aitracer.NewTracer(
		aitracer.Http, "example_service", opts...,
	)
	tracer.Start()

	redisOpts := &redis.Options{Addr: "127.0.0.1:6379", Password: ""}
	client := redis.NewClient(redisOpts)
	client.AddHook(
		NewTracingHook(tracer, "127.0.0.1:6379", []Option{
			WithDB(redisOpts.DB),
		}...),
	)

	// root span
	span := tracer.StartServerSpan("root")
	ctx := aitracer.ContextWithSpan(context.Background(), span)

	// set
	res, err := client.Set(ctx, "key_2", "test.v8", time.Second*30).Result()
	fmt.Printf("set: %+v, %+v\n", res, err)

	// pipe get
	pipe := client.Pipeline()
	for _, key := range []string{"foo", "bar", "key_2"} {
		pipe.Get(ctx, key)
	}
	cmds, _ := pipe.Exec(ctx)
	for i, c := range cmds {
		res, err := c.(*redis.StringCmd).Result()
		fmt.Printf("pipeline result [%d]: res=%+v, err=%+v\n", i, res, err)
	}

	span.Finish() // must finish

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
