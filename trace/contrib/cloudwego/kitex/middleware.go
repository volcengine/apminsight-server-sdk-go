package kitex

import (
	"bytes"
	"context"

	"github.com/bytedance/gopkg/cloud/metainfo"
	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/pkg/endpoint"
	"github.com/cloudwego/kitex/pkg/kerrors"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/pkg/transmeta"
	"github.com/cloudwego/kitex/server"
	"github.com/cloudwego/kitex/transport"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
)

const (
	SpanContextKey = "ApmPlusSpanContext"
)

// serverSuite is a set of options
type serverSuite struct {
	tracer aitracer.Tracer
}

func NewServerSuite(tr aitracer.Tracer) server.Suite {
	return &serverSuite{tracer: tr}
}

func (c *serverSuite) Options() []server.Option {
	var options []server.Option
	options = append(options, server.WithMiddleware(NewServerMiddleware(c.tracer)))
	options = append(options, server.WithMetaHandler(transmeta.ServerTTHeaderHandler))
	return options
}

// NewServerMiddleware return a Middleware that extract traceInfo from context
func NewServerMiddleware(tracer aitracer.Tracer) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req, resp interface{}) (err error) {
			ri := rpcinfo.GetRPCInfo(ctx)
			v, _ := metainfo.GetValue(ctx, SpanContextKey)
			chainSpanContext, _ := tracer.Extract(aitracer.Binary, aitracer.BinaryCarrier(bytes.NewBufferString(v)))
			span := tracer.StartServerSpan("rpc.called", aitracer.ChildOf(chainSpanContext), aitracer.ServerResourceAs(ri.To().Method()))
			defer span.Finish()

			// panics are recovered by kitex. so we need to get panic info from kitex stats.
			defer func() {
				if ok, panicErr := ri.Stats().Panicked(); ok && panicErr != nil {
					span.SetStatus(1)
					if pe, ok := err.(*kerrors.DetailedError); ok {
						span.RecordError(err, aitracer.WithErrorKind(aitracer.ErrorKindPanic), aitracer.WithStack(pe.Stack()))
					} else {
						span.RecordError(err, aitracer.WithErrorKind(aitracer.ErrorKindPanic))
					}
				}
			}()

			ctxWithSpan := aitracer.ContextWithSpan(ctx, span)
			err = next(ctxWithSpan, req, resp)
			if err != nil {
				span.SetStatus(1)
				span.RecordError(err)
			}
			return err
		}
	}
}

type clientSuite struct {
	tracer aitracer.Tracer
}

func NewClientSuite(tracer aitracer.Tracer) client.Suite {
	return &clientSuite{tracer}
}

func (c *clientSuite) Options() []client.Option {
	var options []client.Option
	options = append(options, client.WithMiddleware(NewClientMiddleware(c.tracer)))
	options = append(options, client.WithTransportProtocol(transport.TTHeader))
	options = append(options, client.WithMetaHandler(transmeta.ClientTTHeaderHandler))
	return options
}

// NewClientMiddleware return a Middleware that set meta info in RPCInfo
func NewClientMiddleware(tracer aitracer.Tracer) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req, resp interface{}) (err error) {
			ri := rpcinfo.GetRPCInfo(ctx)
			clientService := "empty"
			if svc := ri.To().ServiceName(); svc != "" {
				clientService = svc
			} else if addr := ri.To().Address().String(); addr != "" {
				clientService = addr
			}
			span, ctxWithSpan := tracer.StartClientSpanFromContext(ctx, "rpc.call",
				aitracer.ClientResourceAs(aitracer.RPC, clientService, ri.To().Method()))
			defer span.Finish()

			// panics are recovered by kitex. so we need to get panic info from kitex stats.
			defer func() {
				if ok, panicErr := ri.Stats().Panicked(); ok && panicErr != nil {
					span.SetStatus(1)
					if pe, ok := err.(*kerrors.DetailedError); ok {
						span.RecordError(err, aitracer.WithErrorKind(aitracer.ErrorKindPanic), aitracer.WithStack(pe.Stack()))
					} else {
						span.RecordError(err, aitracer.WithErrorKind(aitracer.ErrorKindPanic))
					}
				}
			}()

			// inject spanCtx in buf
			buf := bytes.NewBuffer(make([]byte, 0))
			err = tracer.Inject(span.Context(), aitracer.Binary, aitracer.BinaryCarrier(buf))
			// set buf in metainfo
			ctxWithSpan = metainfo.WithValue(ctxWithSpan, SpanContextKey, buf.String())

			err = next(ctxWithSpan, req, resp) // pass ctxWithSpan down in case client span has local child
			if err != nil {
				span.SetStatus(1)
				span.RecordError(err)
			}
			return err
		}
	}
}
