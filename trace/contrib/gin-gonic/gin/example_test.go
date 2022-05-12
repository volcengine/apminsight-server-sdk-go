package gin

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	ailogrus "github.com/volcengine/apminsight-server-sdk-go/trace/contrib/sirupsen/logrus"

	"github.com/gin-gonic/gin"
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
	opts = append(opts, aitracer.WithContextAdapter(NewGinContextAdapter()))
	opts = append(opts, aitracer.WithLogger(&logger{}))

	tracer := aitracer.NewTracer(
		aitracer.Http, "example_service", opts...,
	)
	tracer.Start()
	defer func() {
		tracer.Stop()
	}()

	r := gin.Default()
	r.Use(NewMiddleware(tracer))

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

	r.GET("/error", func(context *gin.Context) { context.JSON(http.StatusBadRequest, "error") })
	r.GET("/ok", func(context *gin.Context) {
		logrus.WithContext(context).Infof("with gin context") // will not work if WithContextAdapter unset
		logrus.WithContext(context.Request.Context()).Infof("with gin.Context.Request.Context")
	})
	r.GET("/panic/", func(context *gin.Context) {
		panic("test")
	})

	_ = r.Run("0.0.0.0:8912")
}
