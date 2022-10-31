package sarama

import (
	"context"

	"github.com/Shopify/sarama"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
)

// WrapHandler wrap func(ctx context.Context, data []byte) for sarama.ConsumerGroupHandler in order to extract tracing info from msg and generate serverSpan
// handler take sarama.ConsumerMessage.Value as param, and ctx contains serverSpan
func WrapHandler(handler func(ctx context.Context, data []byte), tracer aitracer.Tracer, opts ...Option) func(msg *sarama.ConsumerMessage) {
	return func(msg *sarama.ConsumerMessage) {
		if tracer == nil {
			panic("tracer is nil")
		}

		cfg := newDefaultConfig()
		for _, opt := range opts {
			opt(cfg)
		}

		// get tracing from msg header
		m := make(map[string][]string)
		for _, h := range msg.Headers {
			k := string(h.Key)
			v := string(h.Value)
			m[k] = append(m[k], v)
		}

		parentSpanContext, _ := tracer.Extract(aitracer.HTTPHeaders, aitracer.HTTPHeadersCarrier(m))

		span := tracer.StartServerSpan("kafka.consume", aitracer.ChildOf(parentSpanContext), aitracer.ServerResourceAs("consume"))
		defer span.Finish()

		span.SetTagString("mq.type", "kafka")
		span.SetTagString("mq.topic", msg.Topic)

		// extra tag
		for k, v := range cfg.additionalTags {
			span.SetTagString(k, v)
		}

		// in this case, we regard mq as consumer's upstream service
		span.SetTagString("from_service_type", "kafka")
		span.SetTagString("from_service", msg.Topic)

		// set span in context
		ctxWithSpan := aitracer.ContextWithSpan(context.Background(), span)

		handler(ctxWithSpan, msg.Value)
	}
}
