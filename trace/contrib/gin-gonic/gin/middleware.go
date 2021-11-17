package gin

import (
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
