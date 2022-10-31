package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/Shopify/sarama"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
	tracesamara "github.com/volcengine/apminsight-server-sdk-go/trace/contrib/sarama"
)

func main() {
	opts := make([]aitracer.TracerOption, 0)
	opts = append(opts, aitracer.WithMetrics(true))
	opts = append(opts, aitracer.WithLogSender(true))
	opts = append(opts, aitracer.WithLogger(&logger{}))
	tracer := aitracer.NewTracer(
		aitracer.Http, "example_producer", opts...,
	)
	tracer.Start()
	aitracer.SetGlobalTracer(tracer)

	rootSpan := tracer.StartServerSpan("root")

	ctxWithSpan := aitracer.ContextWithSpan(context.Background(), rootSpan)

	{
		kafkaConf := sarama.NewConfig()
		kafkaConf.Producer.Return.Errors = true
		kafkaConf.Producer.Return.Successes = true

		//new producer
		producer, err := sarama.NewAsyncProducer([]string{"0.0.0.0:9092"}, kafkaConf)
		if err != nil {
			panic(err)
		}

		// warp with trace
		wrappedProducer := tracesamara.WrapProducer(kafkaConf, producer, tracer, tracesamara.WithAdditionalTags(map[string]string{"broker": "xxx"}))

		// consume successed and errors
		{
			go func() {
				for range wrappedProducer.Successes() {

				}
			}()

			go func() {
				for err := range wrappedProducer.Errors() {
					fmt.Printf("-------%+v\n", err.Err)
				}
			}()

		}

		if rand.Intn(100) < 50 {
			// case1: send msg via Input(). be aware to inject ctx.
			msg := sarama.ProducerMessage{
				Topic: "test",
				Value: sarama.StringEncoder(fmt.Sprintf("This is a test msg. send time is %s", time.Now().String())),
			}
			wrappedProducer.Input() <- tracesamara.InjectCtx(ctxWithSpan, &msg) // async produce.
		} else {
			// case2: for convenience, user should wrap their own CtxSend(ctx, msg) method. sarama do not provide CtxSend(ctx, msg) method, so we cannot inject ctx by override it.
			CtxSend(ctxWithSpan, wrappedProducer, "test", nil, []byte(fmt.Sprintf("This is a test msg. send time is %s", time.Now().String())))
		}
	}

	rootSpan.Finish() // msg produce is async, so rootSpan could finish before clientSpan end.

	time.Sleep(100 * time.Second)
}

func CtxSend(ctx context.Context, p sarama.AsyncProducer, topic string, key, value []byte) {
	msg := sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.ByteEncoder(key),
		Value: sarama.ByteEncoder(value),
	}
	p.Input() <- tracesamara.InjectCtx(ctx, &msg) // async produce.
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
