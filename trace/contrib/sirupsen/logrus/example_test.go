package logrus

import (
	"testing"

	"context"

	"github.com/sirupsen/logrus"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
)

func Test_example(t *testing.T) {
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

	logrus.AddHook(NewHook(tracer, []logrus.Level{
		logrus.TraceLevel,
		logrus.DebugLevel,
		logrus.InfoLevel,
		logrus.WarnLevel,
		logrus.ErrorLevel,
		logrus.FatalLevel,
	}))

	logrus.Trace("trace message")
	logrus.Debug("debug message")
	logrus.Info("info message")
	logrus.Error("error message")
	//logrus.Fatal("fatal message")

	ctx := context.Background()
	span, ctx := tracer.StartServerSpanFromContext(ctx, "logrus_example_test", aitracer.ServerResourceAs("log"))
	defer span.Finish()
	logrus.WithContext(ctx).Info(("start~~~"))
	logrus.WithContext(ctx).Info(("end~~~"))
}
