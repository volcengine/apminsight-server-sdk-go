package main

import (
	"context"
	"fmt"
	"sync"

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
		aitracer.Http, "example_consumer", opts...,
	)
	tracer.Start()
	aitracer.SetGlobalTracer(tracer)

	// wrap handler
	wrapperHandler := tracesamara.WrapHandler(handler, tracer, tracesamara.WithAdditionalTags(map[string]string{"broker": "xxx"}))

	// new consumerGroupHandler
	consumerGroupHandler := NewConsumerGroupHandler(wrapperHandler)

	// new consumer group
	config := sarama.NewConfig()
	config.Version = sarama.V1_0_0_0
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	consumerGroup, err := sarama.NewConsumerGroup([]string{"0.0.0.0:9092"}, "example", config)
	if err != nil {
		panic(err)
	}

	// consume
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			err = consumerGroup.Consume(context.Background(), []string{"test"}, consumerGroupHandler)
			if err != nil {
				fmt.Printf("consume fail. err=%+v\n", err)
			}
		}
	}()
	wg.Wait()
}

func handler(ctx context.Context, data []byte) {
	// do your process
	span, _ := aitracer.GlobalTracer().StartSpanFromContext(ctx, "handler")
	defer span.Finish()

	fmt.Printf("incoming data is %+v\n", string(data))
}

func NewConsumerGroupHandler(h func(message *sarama.ConsumerMessage)) sarama.ConsumerGroupHandler {
	if h == nil {
		panic("handler is nil")
	}
	return &consumerGH{
		handler: h,
	}
}

// consumerGH represents a Sarama consumer group consumer.
type consumerGH struct {
	handler func(message *sarama.ConsumerMessage)
}

// Setup is run at the beginning of a new session, before ConsumeClaim.
func (c *consumerGH) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited.
func (c *consumerGH) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (c *consumerGH) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	// NOTE:
	// Do not move the code below to a goroutine.
	// The `ConsumeClaim` itself is called within a goroutine, see:
	// https://github.com/Shopify/sarama/blob/master/consumer_group.go#L27-L29
	for message := range claim.Messages() {
		c.handler(message)
		session.MarkMessage(message, "")
	}
	return nil
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
