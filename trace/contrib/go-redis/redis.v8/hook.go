package redis_v8

import (
	"context"

	"github.com/go-redis/redis/extra/rediscmd/v8"
	"github.com/go-redis/redis/v8"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
)

type TracingHook struct {
	tracer aitracer.Tracer

	addr string
	tags map[string]string

	// cache
	callService string
}

type config struct {
	tags map[string]string
}

func newDefaultConfig() *config {
	return &config{}
}

type Option func(*config)

func WithTag(key, value string) Option {
	return func(cfg *config) {
		if cfg.tags == nil {
			cfg.tags = make(map[string]string)
		}
		cfg.tags[key] = value
	}
}

// NewTracingHook return a redis monitor hook.
func NewTracingHook(tracer aitracer.Tracer, addr string, opts ...Option) *TracingHook {
	cfg := newDefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return &TracingHook{tracer: tracer, addr: addr, tags: cfg.tags}
}

func (th *TracingHook) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {
	span, ctxWithSpan := th.tracer.StartClientSpanFromContext(ctx, "redis.command",
		aitracer.ClientResourceAs(aitracer.Redis, th.getCallService(), cmd.Name()))
	span.SetTag(aitracer.DbStatement, rediscmd.CmdString(cmd))
	return ctxWithSpan, nil
}

func (th *TracingHook) AfterProcess(ctx context.Context, cmd redis.Cmder) error {
	span := aitracer.GetSpanFromContext(ctx)
	if err := cmd.Err(); err != nil && err != redis.Nil {
		span.RecordError(err, aitracer.WithErrorKind(aitracer.ErrorKindDbError))
		span.SetStatus(aitracer.StatusCodeError)
	}
	span.Finish()
	return nil
}

func (th *TracingHook) BeforeProcessPipeline(ctx context.Context, cmds []redis.Cmder) (context.Context, error) {
	summary, cmdsString := rediscmd.CmdsString(cmds)
	span, ctxWithSpan := th.tracer.StartClientSpanFromContext(ctx, "redis.pipeline",
		aitracer.ClientResourceAs(aitracer.Redis, th.getCallService(), "pipeline"))
	span.SetTag(aitracer.DbStatement, cmdsString)
	span.SetTag("db.redis.pipe.summary", summary)
	span.SetTag("db.redis.pipe.cmds_num", len(cmds))
	return ctxWithSpan, nil
}

func (th *TracingHook) AfterProcessPipeline(ctx context.Context, cmds []redis.Cmder) error {
	span := aitracer.GetSpanFromContext(ctx)
	if len(cmds) > 0 {
		if err := cmds[0].Err(); err != nil && err != redis.Nil {
			span.RecordError(err, aitracer.WithErrorKind(aitracer.ErrorKindDbError))
			span.SetStatus(aitracer.StatusCodeError)
		}
	}
	span.Finish()
	return nil
}

func (th *TracingHook) getCallService() string {
	if len(th.callService) != 0 {
		return th.callService
	}
	th.callService = "redis:" + th.addr
	return th.callService
}
