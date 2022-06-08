package hertz

import (
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
)

func NewMiddleware(tracer aitracer.Tracer) app.HandlerFunc {
	if tracer == nil {
		panic("tracer is nil")
	}
	return func(ctx context.Context, reqCtx *app.RequestContext) {
		resourceName := string(reqCtx.Path())
		if resourceName == "" {
			resourceName = "unknown"
		}

		chainSpanContext, _ := tracer.Extract(aitracer.HTTPHeaders, aitracer.HTTPHeadersCarrier(hertzHeaderToHttpHeader(&reqCtx.Request.Header)))
		span := tracer.StartServerSpan("request", aitracer.ChildOf(chainSpanContext), aitracer.ServerResourceAs(resourceName))

		// set span to Context
		ctx = aitracer.ContextWithSpan(ctx, span)

		// set trace-id in response header
		spanContext := span.Context()
		reqCtx.Response.Header.Add("x-trace-id", spanContext.TraceID())

		// set additional tags
		span.SetTag(aitracer.HttpMethod, string(reqCtx.Request.Method()))
		if uri := reqCtx.Request.URI(); uri != nil {
			span.SetTag(aitracer.HttpScheme, string(uri.Scheme()))
			span.SetTag(aitracer.HttpHost, string(uri.Host()))
			span.SetTag(aitracer.HttpPath, string(uri.Path()))
		}

		// Finish should be called directly by defer
		defer span.Finish()

		isPanic := true
		defer func() {
			status := reqCtx.Response.StatusCode()
			if isPanic {
				status = http.StatusInternalServerError
			}
			span.SetTag(aitracer.HttpStatusCode, status)
			if status != http.StatusOK {
				span.SetStatus(int64(status))
			}
		}()

		reqCtx.Next(ctx)
		isPanic = false
	}
}

// hertzHeaderToHttpHeader trans hertz header to stander http.Header
func hertzHeaderToHttpHeader(hertzHeader *protocol.RequestHeader) http.Header {
	h := http.Header{}
	if hertzHeader == nil {
		return h
	}
	hertzHeader.VisitAll(func(key, value []byte) {
		h.Set(string(key), string(value))
	})
	return h
}
