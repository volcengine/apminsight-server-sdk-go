package gin

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
	ailogrus "github.com/volcengine/apminsight-server-sdk-go/trace/contrib/sirupsen/logrus"
)

type logger struct{}

func (l *logger) Debug(format string, args ...interface{}) {
	fmt.Printf("[Debug]"+format+"\n", args)
}
func (l *logger) Info(format string, args ...interface{}) {
	fmt.Printf("[Info]"+format+"\n", args)
}
func (l *logger) Error(format string, args ...interface{}) {
	fmt.Printf("[Error]"+format+"\n", args)
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

	r := gin.Default()
	r.ContextWithFallback = true // recommended. set ContextWithFallback=true enables propagation with gin.Context() rather than gin.Context.Request.Context()

	// you can define your own ignoreRequest func or pathNormalizer by using code below. In most case defaultPathNormalizer is recommended
	r.Use(
		NewMiddleware(tracer, []Option{
			WithIgnoreRequest(exampleIgnoreOptionsRequest),
			//WithPathNormalizer(func(escapedPath string) string {
			//	// your code here.
			//	return escapedPath
			//}),
			WithTagsExtractor(exampleTagsExtractor),
			WithResourceGetter(exampleResourceGetterHandlerName),
		}...),
	)

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

	r.GET("/error", func(context *gin.Context) { context.JSON(http.StatusNoContent, "error") })
	r.GET("/v1/error", func(context *gin.Context) { context.JSON(http.StatusOK, "v1") })

	r.GET("/ok", func(context *gin.Context) {
		logrus.WithContext(context).Infof("with gin context") // will not work if WithContextAdapter unset
		logrus.WithContext(context.Request.Context()).Infof("with gin.Context.Request.Context")
	})
	r.GET("/panic/", func(context *gin.Context) {
		panic("test")
	})
	r.GET("/h", handlerA())

	_ = r.Run("0.0.0.0:8912")
}

func handlerA() func(context *gin.Context) {
	return func(context *gin.Context) {
		context.JSON(http.StatusNoContent, "ok")
	}
}

func Test_path_normalizer(t *testing.T) {
	fmt.Println(defaultPathNormalizer(""))
	fmt.Println(defaultPathNormalizer("/"))
	fmt.Println(defaultPathNormalizer("/~//~/"))

	fmt.Println(defaultPathNormalizer("/v1"))
	fmt.Println(defaultPathNormalizer("/V1"))

	fmt.Println(defaultPathNormalizer("v1"))

	fmt.Println(defaultPathNormalizer("/v1/vv"))
	fmt.Println(defaultPathNormalizer("/v1/v2"))
	fmt.Println(defaultPathNormalizer("/v1/abc"))
	fmt.Println(defaultPathNormalizer("/v1/ab2"))

	fmt.Println(defaultPathNormalizer("/kkkk"))
	fmt.Println(defaultPathNormalizer("/v222"))

	fmt.Println(defaultPathNormalizer("/v2/ab1/kkkkkkkkkkkkkkk3/kkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkklllllll"))
	fmt.Println(defaultPathNormalizer("/v2/ab1/"))

	fmt.Println(defaultPathNormalizer("/v2/ab1/1/:222"))

	fmt.Println(defaultPathNormalizer("v2/ab1/1/:222"))

	fmt.Println(defaultPathNormalizer("/ABC/av-1/b_2/c.3/d4d/v5f/v699/7"))
}

func exampleIgnoreOptionsRequest(c *gin.Context) bool {
	if c.Request != nil && c.Request.Method == http.MethodOptions {
		return true
	}
	return false
}

func exampleTagsExtractor(c *gin.Context) map[string]string {
	tags := make(map[string]string)
	if c.Request != nil {
		tags["X-LOG-ID"] = c.Request.Header.Get("X-LOG-ID")
	}
	return tags
}

func exampleResourceGetterByURLQuery(c *gin.Context) string {
	return c.Query("Action") //use URL query 'Action' as resource
}

// Not Recommended
// could result in something like func1 if handler is wrapped
func exampleResourceGetterHandlerName(c *gin.Context) string {
	method := c.HandlerName()
	pos := strings.LastIndexByte(method, '.')
	if pos != -1 {
		method = c.HandlerName()[pos+1:]
	}
	return method
}
