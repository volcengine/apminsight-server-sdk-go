package grpc_go

import (
	"context"
	"net/http"

	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// currently only supported unaryInterceptor

type Config struct {
	targetServiceType string
	baggageSetter     func(ctx context.Context) map[string]string // get key-value pair from ctx and set in span baggage
}

func newDefaultConfig() *Config {
	return &Config{
		targetServiceType: "grpc",
	}
}

type Option func(*Config)

func WithTargetServiceType(tst string) Option {
	return func(cfg *Config) {
		cfg.targetServiceType = tst
	}
}

func WithBaggageSetter(f func(ctx context.Context) map[string]string) Option {
	return func(cfg *Config) {
		if f != nil {
			cfg.baggageSetter = f
		}
	}
}

func NewUnaryServerInterceptor(tracer aitracer.Tracer, opts ...Option) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{},
		info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		cfg := newDefaultConfig()
		for _, opt := range opts {
			opt(cfg)
		}

		// get metaInfo from context
		meta, _ := metadata.FromIncomingContext(ctx)
		metaCopy := meta.Copy()

		// extract spanContext from metaInfo
		parentSpanContext, _ := tracer.Extract(aitracer.HTTPHeaders, aitracer.HTTPHeadersCarrier(metaCopy))

		// start server span from parentSpanContext
		span := tracer.StartServerSpan("grpc.called", aitracer.ChildOf(parentSpanContext), aitracer.ServerResourceAs(info.FullMethod))
		defer span.Finish()

		if cfg.baggageSetter != nil {
			for k, v := range cfg.baggageSetter(ctx) {
				span.SetBaggageItem(k, v)
			}
		}

		// set span in context
		ctxWithSpan := aitracer.ContextWithSpan(ctx, span)

		// use ctxWithSpan to handler request
		resp, err = handler(ctxWithSpan, req)
		if err != nil {
			s, _ := status.FromError(err)
			span.SetStatus(aitracer.StatusCodeError)
			span.SetTagInt64("grpc.status_code", int64(s.Code()))
			span.RecordError(err, aitracer.WithErrorKind(aitracer.ErrorKindBusinessError))
		} else {
			span.SetTagInt64("grpc.status_code", int64(codes.OK))
		}
		return
	}
}

func NewUnaryClientInterceptor(tracer aitracer.Tracer, opts ...Option) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker, callOpts ...grpc.CallOption) error {
		cfg := newDefaultConfig()
		for _, opt := range opts {
			opt(cfg)
		}

		span, ctxWithSpan := tracer.StartClientSpanFromContext(ctx, "grpc.call", aitracer.ClientResourceAs(cfg.targetServiceType, cc.Target(), method))
		defer span.Finish()

		// propagate
		meta, _ := metadata.FromOutgoingContext(ctx)
		metaCopy := meta.Copy()
		// format spanCtx to header
		header := make(http.Header)
		_ = tracer.Inject(span.Context(), aitracer.HTTPHeaders, aitracer.HTTPHeadersCarrier(header))
		// set buf in metainfo
		for k, v := range header {
			metaCopy.Set(k, v...)
		}
		// set metainfo to context
		ctxWithSpan = metadata.NewOutgoingContext(ctxWithSpan, metaCopy)

		// call remote service with ctxWithSpan
		err := invoker(ctxWithSpan, method, req, reply, cc, callOpts...)
		if err != nil {
			s, _ := status.FromError(err)
			span.SetStatus(aitracer.StatusCodeError)
			span.SetTagInt64("grpc.status_code", int64(s.Code()))
			span.RecordError(err, aitracer.WithErrorKind(aitracer.ErrorKindExternalServiceError))
		} else {
			span.SetTagInt64("grpc.status_code", int64(codes.OK))
		}
		return err
	}
}
