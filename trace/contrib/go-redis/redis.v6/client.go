package redis_v6

import (
	"context"

	"github.com/go-redis/redis"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
)

type TraceClient struct {
	tracer aitracer.Tracer
	*redis.Client
}

// WrapClient create a wrapped redis.TraceClient with trace
func WrapClient(tracer aitracer.Tracer, client *redis.Client) *TraceClient {
	if tracer == nil {
		panic("tracer is nil")
	}
	return &TraceClient{
		tracer: tracer,
		Client: client,
	}
}

// WithContext is used to process redisCmd with trace. redisCmd should be executed by c2
func (c *TraceClient) WithContext(ctx context.Context) *redis.Client {
	c2 := c.Client.WithContext(ctx)
	c2.WrapProcess(process(ctx, c.tracer, c2.Options()))
	c2.WrapProcessPipeline(processPipeline(ctx, c.tracer, c2.Options()))
	return c2
}

func process(ctx context.Context, tracer aitracer.Tracer, opts *redis.Options) func(oldProcess func(cmd redis.Cmder) error) func(cmd redis.Cmder) error {
	return func(oldProcess func(cmd redis.Cmder) error) func(cmd redis.Cmder) error {
		return func(cmd redis.Cmder) error {
			addr := ""
			if opts != nil {
				addr = opts.Addr
			}
			span, _ := tracer.StartClientSpanFromContext(ctx, "redis.command",
				aitracer.ClientResourceAs(aitracer.Redis, "redis:"+addr, cmd.Name()))
			defer span.Finish()

			span.SetTag(aitracer.DbStatement, CmdString(cmd))

			err := oldProcess(cmd)
			if err != nil && err != redis.Nil {
				span.RecordError(err, aitracer.WithErrorKind(aitracer.ErrorKindDbError))
				span.SetStatus(aitracer.StatusCodeError)
			}
			return err
		}
	}
}

func processPipeline(ctx context.Context, tracer aitracer.Tracer, opts *redis.Options) func(oldProcess func(cmds []redis.Cmder) error) func(cmds []redis.Cmder) error {
	return func(oldProcess func(cmds []redis.Cmder) error) func(cmds []redis.Cmder) error {
		return func(cmds []redis.Cmder) error {
			addr := ""
			if opts != nil {
				addr = opts.Addr
			}
			span, _ := tracer.StartClientSpanFromContext(ctx, "redis.pipeline",
				aitracer.ClientResourceAs(aitracer.Redis, "redis:"+addr, "pipeline"))
			defer span.Finish()

			summary, cmdsString := CmdsString(cmds)
			span.SetTag(aitracer.DbStatement, cmdsString)
			span.SetTag("db.redis.pipe.summary", summary)
			span.SetTag("db.redis.pipe.cmds_num", len(cmds))

			err := oldProcess(cmds)
			if err != nil && err != redis.Nil {
				span.RecordError(err, aitracer.WithErrorKind(aitracer.ErrorKindDbError))
				span.SetStatus(aitracer.StatusCodeError)
			}
			return err
		}
	}
}
