package http

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
)

type logger struct{}

func (l *logger) Debug(format string, args ...interface{}) {
	fmt.Printf("[Debug] format %+v\n", args)
}
func (l *logger) Info(format string, args ...interface{}) {
	fmt.Printf("[Info] format %+v\n", args)
}
func (l *logger) Error(format string, args ...interface{}) {
	fmt.Printf("[Error] format %+v\n", args)
}

func Test_example(t *testing.T) {
	opts := make([]aitracer.TracerOption, 0)
	opts = append(opts, aitracer.WithMetrics(true))
	opts = append(opts, aitracer.WithLogSender(true))
	opts = append(opts, aitracer.WithLogger(&logger{}))

	tracer := aitracer.NewTracer(
		aitracer.Http, "example_service", opts...,
	)
	tracer.Start()
	defer func() {
		tracer.Stop()
	}()

	// server span
	ctx := context.Background()
	span, ctx := tracer.StartServerSpanFromContext(ctx, "go_http_example_test", aitracer.ServerResourceAs("go_http"))
	defer span.Finish()

	// client span
	{
		/*
				how to set remote service name(clientService) with your own getter.
			    example: maybe you want to set client_service in request context:
					// define your getter
					getter := func(req *http.Request) string{
					  return req.Context().Value("client_service_key")
					}
					// wrapClient with your getter
					hc = WrapClient(hc, tracer,  WithClientServiceGetter(getter))
					// set value by your getter
					ctx = context.WithValue(ctx, "client_service_key", "my_service")

				can be set in request Headers as well
		*/

		hc := &http.Client{
			Timeout: time.Second,
		}
		clientServiceGetter := func(req *http.Request) string {
			return req.Header.Get("X-client-service")
		}
		hc = WrapClient(hc, tracer, WithClientServiceGetter(clientServiceGetter))

		req, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1/user/1234/detail", nil)
		req.Header.Add("X-client-service", "my_service")

		// MUST: if ctx not passed in, span http_call will not be related to trace
		req = req.WithContext(ctx)

		res, _ := hc.Do(req)
		if res != nil {
			defer res.Body.Close()
		}
	}

}
