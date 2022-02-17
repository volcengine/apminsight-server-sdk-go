package gin

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
)

func NewMiddleware(tracer aitracer.Tracer) gin.HandlerFunc {
	if tracer == nil {
		panic("tracer is nil")
	}
	return func(c *gin.Context) {
		resourceName := c.FullPath()
		if resourceName == "" {
			resourceName = "unknown"
		}

		chainSpanContext, _ := tracer.Extract(aitracer.HTTPHeaders, aitracer.HTTPHeadersCarrier(c.Request.Header))
		span := tracer.StartServerSpan("request", aitracer.ChildOf(chainSpanContext), aitracer.ServerResourceAs(resourceName))
		spanContext := span.Context()
		c.Request = c.Request.WithContext(aitracer.ContextWithSpan(c.Request.Context(), span))
		c.Writer.Header().Add("x-trace-id", spanContext.TraceID())

		span.SetTag(aitracer.HttpMethod, c.Request.Method)
		if c.Request.URL != nil {
			span.SetTag(aitracer.HttpScheme, c.Request.URL.Scheme)
			span.SetTag(aitracer.HttpHost, c.Request.URL.Host)
			span.SetTag(aitracer.HttpPath, c.Request.URL.Path)
		}

		isPanic := true
		defer func() {
			if isPanic {
				span.FinishWithOption(aitracer.FinishSpanOption{
					Status:     1,
					FinishTime: time.Now(),
				})
				return
			}
			status := c.Writer.Status()
			span.SetTag(aitracer.HttpStatusCode, status)
			if status == http.StatusOK {
				span.Finish()
			} else {
				span.FinishWithOption(aitracer.FinishSpanOption{
					Status:     int64(status),
					FinishTime: time.Now(),
				})
			}
		}()
		c.Next()
		isPanic = false

	}
}

// input: *gin.Context,  in order to use logrus/trace with gin.Context rather than gin.Context.Request.Context
func NewGinContextAdapter() func(context.Context) context.Context {
	return func(ctx context.Context) context.Context {
		if ctx == nil {
			return nil
		}
		if c, ok := ctx.(*gin.Context); ok {
			return c.Request.Context()
		}
		return ctx
	}
}
