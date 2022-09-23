package redis_v8

import (
	"testing"

	"github.com/go-redis/redis/v8"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
)

func TestExample(t *testing.T) {
	opts := make([]aitracer.TracerOption, 0)
	opts = append(opts, aitracer.WithMetrics(true))
	opts = append(opts, aitracer.WithLogSender(true))
	tracer := aitracer.NewTracer(
		aitracer.Http, "example_service", opts...,
	)

	redisOpts := &redis.Options{Addr: "127.0.0.1:6379", Password: ""}
	client := redis.NewClient(redisOpts)
	client.AddHook(
		NewTracingHook(tracer, "127.0.0.1:6379", []Option{
			WithTag("net.peer.ip", "127.0.0.1"),
			WithTag("net.peer.port", "6379"),
		}...),
	)
}
