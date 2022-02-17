package sirupsen

import (
	"context"
	"fmt"
	"net/http"

	gincontrib "github.com/volcengine/apminsight-server-sdk-go/trace/contrib/gin-gonic/gin"

	"github.com/gin-gonic/gin"

	//"github.com/gin-gonic/gin"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
	ailogrus "github.com/volcengine/apminsight-server-sdk-go/trace/contrib/sirupsen/logrus"
)

func Test_example(t *testing.T) {
	opts := make([]aitracer.TracerOption, 0)
	opts = append(opts, aitracer.WithLogSender(true))
	opts = append(opts, aitracer.WithContextAdapter(gincontrib.NewGinContextAdapter()))

	tracer := aitracer.NewTracer(
		aitracer.Http, "example_service", opts...,
	)
	tracer.Start()
	defer func() {
		tracer.Stop()
	}()

	logrus.SetLevel(logrus.TraceLevel)
	logrus.SetReportCaller(true)

	logrus.AddHook(ailogrus.NewHook(tracer, []logrus.Level{
		logrus.TraceLevel,
		logrus.DebugLevel,
		logrus.InfoLevel,
		logrus.WarnLevel,
		logrus.ErrorLevel,
		logrus.FatalLevel,
	}))

	ctx := context.Background()
	span, ctx := tracer.StartServerSpanFromContext(ctx, "logrus_example_test", aitracer.ServerResourceAs("log"))
	defer span.Finish()
	logrus.WithContext(ctx).Info(("start~~~"))
	logrus.WithContext(ctx).Info(("end~~~"))

	c := gin.Context{}
	c.Request, _ = http.NewRequest(http.MethodGet, "", nil)
	c.Request = c.Request.WithContext(ctx)
	logrus.WithContext(&c).Info("gin") //trace_id can be obtained if WithContextAdapter is set

}

func Test_depth(t *testing.T) {
	opts := make([]aitracer.TracerOption, 0)
	opts = append(opts, aitracer.WithLogSender(true))

	tracer := aitracer.NewTracer(
		aitracer.Http, "example_service", opts...,
	)
	tracer.Start()
	defer func() {
		tracer.Stop()
	}()

	logrus.SetLevel(logrus.TraceLevel)
	logrus.SetReportCaller(true)

	logrus.AddHook(ailogrus.NewHook(tracer, []logrus.Level{
		logrus.TraceLevel,
		logrus.DebugLevel,
		logrus.InfoLevel,
		logrus.WarnLevel,
		logrus.ErrorLevel,
		logrus.FatalLevel,
	}, ailogrus.WithDepth(1))) // depth = 1

	//logrus.Debug("debug message")
	//logrus.Info("info message") //will result in incorrect caller
	//logrus.Error("error message")
	//logrus.Fatal("fatal message")

	ctx := context.Background()
	LogWrapper(ctx) //correct caller

	fmt.Println()
}

func LogWrapper(c context.Context) {
	logrus.WithContext(c).Info("gin context")
}
