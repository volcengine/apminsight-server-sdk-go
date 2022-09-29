package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
	grpc_go "github.com/volcengine/apminsight-server-sdk-go/trace/contrib/grpc/grpc-go"
	"github.com/volcengine/apminsight-server-sdk-go/trace/contrib/grpc/grpc-go/example/hello"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	opts := make([]aitracer.TracerOption, 0)
	opts = append(opts, aitracer.WithMetrics(true))
	opts = append(opts, aitracer.WithLogSender(true))
	opts = append(opts, aitracer.WithLogger(&logger{}))

	tracer := aitracer.NewTracer(
		aitracer.Http, "example_grpc_client", opts...,
	)
	tracer.Start()

	conn, err := grpc.Dial("0.0.0.0:18080",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(grpc_go.NewUnaryClientInterceptor(tracer)),
	)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
		return
	}
	defer conn.Close()
	client := hello.NewGreeterClient(conn)
	req := hello.HelloRequest{
		Name: "byteapm-grpc-example",
	}

	resp, err := client.SayHello(context.Background(), &req)
	logrus.Infof("resp=%+v, err=%+v", resp, err)

	time.Sleep(1 * time.Second) //wait to print log
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
