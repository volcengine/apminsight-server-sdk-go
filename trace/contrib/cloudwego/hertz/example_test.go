package hertz

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/sirupsen/logrus"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
	tracehttp "github.com/volcengine/apminsight-server-sdk-go/trace/contrib/net/http"
	ailogrus "github.com/volcengine/apminsight-server-sdk-go/trace/contrib/sirupsen/logrus"
)

type logger struct{}

func (l *logger) Debug(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}
func (l *logger) Info(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}
func (l *logger) Error(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

func Test(t *testing.T) {
	h := server.Default(server.WithHostPorts("0.0.0.0:9000"))

	// service name
	h.Name = "my_service"

	// Init trace
	opts := make([]aitracer.TracerOption, 0)
	opts = append(opts, aitracer.WithMetrics(true))
	opts = append(opts, aitracer.WithLogSender(true))
	opts = append(opts, aitracer.WithLogger(&logger{}))
	tracer := aitracer.NewTracer(
		aitracer.Http, string(h.GetServerName()), opts...,
	)
	tracer.Start()
	aitracer.SetGlobalTracer(tracer)
	defer func() {
		tracer.Stop()
	}()

	// Init log
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetReportCaller(true)
	logrus.AddHook(ailogrus.NewHook(tracer, []logrus.Level{
		logrus.TraceLevel,
		logrus.DebugLevel,
		logrus.InfoLevel,
		logrus.WarnLevel,
		logrus.ErrorLevel,
		logrus.FatalLevel,
	}))

	h.Use(NewMiddleware(tracer))

	//  CallRemote shows how to call remote service with trace
	h.GET("/call_remote", CallRemote)

	h.Spin()
}

// CallRemote calls a remote service with trace. Be aware that span is held in context.Context
func CallRemote(ctx context.Context, reqCtx *app.RequestContext) {
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

	reqCtx.JSON(200, utils.H{
		"message": "success",
	})
}
