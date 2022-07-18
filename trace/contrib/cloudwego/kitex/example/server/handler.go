package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
	"github.com/volcengine/apminsight-server-sdk-go/trace/contrib/cloudwego/kitex/example/server/kitex_gen/api"
	tracehttp "github.com/volcengine/apminsight-server-sdk-go/trace/contrib/net/http"
)

// HelloImpl implements the last service interface defined in the IDL.
type HelloImpl struct{}

// Echo implements the HelloImpl interface.
func (s *HelloImpl) Echo(ctx context.Context, req *api.Request) (resp *api.Response, err error) {
	// TODO: Your code here...
	if req == nil {
		return
	}
	fmt.Printf("incoming msg=%s\n", req.Message)

	resp = &api.Response{}

	// lets say kitex calls a http service
	resp.Message = CallRemote(ctx)

	if os.Getenv("TEST_ERROR") != "" {
		err = fmt.Errorf("this is an error")
	}
	{
		if os.Getenv("TEST_PANIC") != "" {
			panic("test panic capture")
		}
	}

	return
}

// CallRemote calls a remote service with trace. Be aware that span is held in context.Context
func CallRemote(ctx context.Context) string {
	// get global tracer
	tracer := aitracer.GlobalTracer()

	hc := &http.Client{
		Timeout: time.Second,
	}
	{
		// define clientService getter
		clientServiceGetter := func(req *http.Request) string {
			return req.Header.Get("X-client-service")
		}
		// wrap with getter
		hc = tracehttp.WrapClient(hc, tracer, tracehttp.WithClientServiceGetter(clientServiceGetter))
	}

	{
		// new request
		req, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1:5000", nil)
		// set clientService
		req.Header.Add("X-client-service", "downstream_service_name")
		// inject context and call
		req = req.WithContext(ctx)
		res, _ := hc.Do(req)
		if res != nil {
			defer res.Body.Close()
		}
	}

	return "Trace is on!"
}
