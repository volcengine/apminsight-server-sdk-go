package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
	tracesamara "github.com/volcengine/apminsight-server-sdk-go/trace/contrib/sarama"
)

func TestConsumerWrapper(t *testing.T) {
	opts := make([]aitracer.TracerOption, 0)
	opts = append(opts, aitracer.WithMetrics(true))
	opts = append(opts, aitracer.WithLogSender(true))
	opts = append(opts, aitracer.WithLogger(&logger{}))

	tracer := aitracer.NewTracer(
		aitracer.Consumer, "example_consumer", opts...,
	)
	tracer.Start()
	aitracer.SetGlobalTracer(tracer)

	msg := sarama.ConsumerMessage{
		Headers: []*sarama.RecordHeader{
			{Key: []byte("x-trace-id"), Value: []byte("2022100816085700f51897d026837150")},
			{Key: []byte("x-span-id"), Value: []byte("202210081610560014619739485898de")},
		},
		Value: []byte("this is test msg"),
	}

	h := tracesamara.WrapHandler(handleMsg, tracer) // equal to handleMsg(ctxWithSpan, msg.Value)
	h(&msg)

	time.Sleep(3 * time.Second)
}

func handleMsg(ctx context.Context, data []byte) {
	tracer := aitracer.GlobalTracer()
	span, _ := tracer.StartSpanFromContext(ctx, "handle")
	defer span.Finish()
	fmt.Println(string(data))
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
