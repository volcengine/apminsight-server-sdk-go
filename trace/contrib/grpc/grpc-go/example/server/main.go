package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
	grpc_go "github.com/volcengine/apminsight-server-sdk-go/trace/contrib/grpc/grpc-go"
	"github.com/volcengine/apminsight-server-sdk-go/trace/contrib/grpc/grpc-go/example/hello"
	tracehttp "github.com/volcengine/apminsight-server-sdk-go/trace/contrib/net/http"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func main() {
	opts := make([]aitracer.TracerOption, 0)
	opts = append(opts, aitracer.WithMetrics(true))
	opts = append(opts, aitracer.WithLogSender(true))
	opts = append(opts, aitracer.WithLogger(&logger{}))

	tracer := aitracer.NewTracer(
		aitracer.Http, "example_grpc_server", opts...,
	)
	tracer.Start()
	aitracer.SetGlobalTracer(tracer)
	defer func() {
		tracer.Stop()
	}()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 18080))
	if err != nil {
		panic(err)
	}

	s := grpc.NewServer(
		grpc.MaxRecvMsgSize(10*1024*1024),
		grpc.UnaryInterceptor(grpc_go.NewUnaryServerInterceptor(tracer)),
	)
	hello.RegisterGreeterServer(s, helloServer)

	if err := s.Serve(lis); err != nil {
		fmt.Printf("grpc handler err %+v\n", err)
	}
}

type server struct {
	hello.UnimplementedGreeterServer
}

var helloServer = &server{}

func (s *server) SayHello(ctx context.Context, data *hello.HelloRequest) (*hello.HelloReply, error) {
	if data == nil {
		return &hello.HelloReply{}, nil
	}

	CallRemote(ctx)

	return &hello.HelloReply{
		Message: "Hello " + data.GetName(),
	}, status.Errorf(codes.Internal, "test")
}

// CallRemote calls a remote service with trace. Be aware that span is held in context.Context
func CallRemote(ctx context.Context) {
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

	// new request
	req, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1:5000", nil)
	// set clientService
	req.Header.Add("X-client-service", "downstream_service_name")
	// inject context and call
	req = req.WithContext(ctx)
	_, _ = hc.Do(req)
}

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
