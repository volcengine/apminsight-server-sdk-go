package http

import (
	"testing"

	"net/http"
	"time"

	"context"

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

	hc := &http.Client{
		Timeout: time.Second,
	}
	hc = WrapClient(hc, tracer)

	ctx := context.Background()
	span, ctx := tracer.StartServerSpanFromContext(ctx, "go_http_example_test", aitracer.ServerResourceAs("go_http"))
	defer span.Finish()
	{
		req, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1", nil)
		req = req.WithContext(ctx)
		_, _ = hc.Do(req)
	}

}
