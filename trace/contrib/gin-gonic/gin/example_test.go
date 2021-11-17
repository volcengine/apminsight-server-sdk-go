package gin

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
)

func Test_example(t *testing.T) {
	opts := make([]aitracer.TracerOption, 0)
	opts = append(opts, aitracer.WithMetrics(true))
	opts = append(opts, aitracer.WithLogSender(true))

	tracer := aitracer.NewTracer(
		aitracer.Http, "example_service", opts...,
	)
	tracer.Start()
	defer func() {
		tracer.Stop()
	}()

	r := gin.Default()
	r.Use(NewMiddleware(tracer))
	r.GET("/gin_path_1/:id/", func(context *gin.Context) {})
	r.GET("/gin_path_2/", func(context *gin.Context) {})

	_ = r.Run("0.0.0.0:8912")
}
