package sarama

import (
	"context"

	"github.com/Shopify/sarama"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
)

type asyncProducer struct {
	sarama.AsyncProducer

	outerInput     chan *sarama.ProducerMessage
	outerSuccesses chan *sarama.ProducerMessage
	outerErrors    chan *sarama.ProducerError

	closeChan      chan struct{}
	closeAsyncChan chan struct{}

	closeErrRet chan error
}

// WrapProducer wrap inner sarama.AsyncProducer to generate client span.
// due to sarama.AsyncProducer.Input() return a chan, we can not instrument Input method
// so a outerInput chan is provided, when put msg into outerInput chan, we can intercept and trace it
func WrapProducer(cfg *sarama.Config, p sarama.AsyncProducer, tracer aitracer.Tracer) sarama.AsyncProducer {
	if cfg == nil {
		panic("sarama config is nil")
	}
	if tracer == nil {
		panic("tracer is nil")
	}
	wrappedProducer := asyncProducer{
		AsyncProducer:  p,
		outerInput:     make(chan *sarama.ProducerMessage),
		outerSuccesses: make(chan *sarama.ProducerMessage),
		outerErrors:    make(chan *sarama.ProducerError),
		closeChan:      make(chan struct{}),
		closeAsyncChan: make(chan struct{}),
		closeErrRet:    make(chan error),
	}

	go func() {
		for {
			select {
			case <-wrappedProducer.closeChan:
				wrappedProducer.closeErrRet <- p.Close()
				return
			case <-wrappedProducer.closeAsyncChan:
				p.AsyncClose()
				return
			case msg, ok := <-wrappedProducer.outerInput:
				if !ok {
					continue // wait for closeChan/closeAsyncChan
				}

				// get ctx from metaData
				var (
					wrappedMeta metaDataWrapper
				)
				wrappedMeta, ok = msg.Metadata.(metaDataWrapper)
				if !ok { // if metadata not wrapped, we need store origin metadata and init ctx. if ok, means origin metadata has been stored and ctx has been passed in
					wrappedMeta.ctx = context.Background()
					wrappedMeta.originMetaData = msg.Metadata
				}

				// new client span
				clientSpan, ctxWithSpan := tracer.StartClientSpanFromContext(wrappedMeta.ctx, "kafka.produce", aitracer.ClientResourceAs(aitracer.Kafka, msg.Topic, "produce"))
				clientSpan.SetTagString("mq.type", "kafka")
				clientSpan.SetTagString("mq.topic", msg.Topic)
				clientSpan.SetTagString("kafka.version", cfg.Version.String())

				wrappedMeta.ctx = ctxWithSpan // update ctxWithSpan in wrappedMeta
				msg.Metadata = wrappedMeta    // set wrappedMeta into metadata, so we can finish it when return

				// inject client span into msg to propagate
				if cfg.Version.IsAtLeast(sarama.V0_11_0_0) {
					propagate(clientSpan, msg, tracer)
				}

				// if successes=false or errors=false, just finish
				// for example, if successes=true and errors=false, we never know when msg fails and span will never be closed
				if !cfg.Producer.Return.Successes || !cfg.Producer.Return.Errors {
					clientSpan.Finish()
				}

				p.Input() <- msg // real send
			}
		}
	}()

	go func() {
		defer func() {
			close(wrappedProducer.outerSuccesses)
		}()
		for msg := range p.Successes() {
			wrappedMeta, ok := msg.Metadata.(metaDataWrapper)
			if ok {
				if span := aitracer.GetSpanFromContext(wrappedMeta.ctx); span != nil {
					span.Finish()
				}
				msg.Metadata = wrappedMeta.originMetaData // restore metadata
			}
			wrappedProducer.outerSuccesses <- msg // send to outer chan so user can read
		}
	}()

	go func() {
		defer func() {
			close(wrappedProducer.outerErrors)
		}()
		for msg := range p.Errors() {
			wrappedMeta, ok := msg.Msg.Metadata.(metaDataWrapper)
			if ok {
				if span := aitracer.GetSpanFromContext(wrappedMeta.ctx); span != nil {
					span.SetStatus(aitracer.StatusCodeError)
					span.RecordError(msg.Err, aitracer.WithErrorKind(aitracer.ErrorKindMqError))
					span.Finish()
				}
				msg.Msg.Metadata = wrappedMeta.originMetaData // restore metadata
			}
			wrappedProducer.outerErrors <- msg // send to outer chan so user can read
		}
	}()
	return &wrappedProducer
}

type metaDataWrapper struct {
	ctx            context.Context
	originMetaData interface{}
}

// InjectCtx inject ctx into msg to generate clientSpan
func InjectCtx(ctx context.Context, msg *sarama.ProducerMessage) *sarama.ProducerMessage {
	if ctx == nil {
		ctx = context.Background()
	}
	newMeta := metaDataWrapper{
		ctx:            ctx,
		originMetaData: msg.Metadata,
	}
	msg.Metadata = newMeta
	return msg
}

// AsyncClose triggers a shutdown of the producer.
func (ap *asyncProducer) AsyncClose() {
	close(ap.outerInput)
	close(ap.closeAsyncChan)
}

// Close shuts down the producer and waits for any buffered messages to be
// flushed.
func (ap *asyncProducer) Close() error {
	close(ap.outerInput)
	close(ap.closeChan)
	return <-ap.closeErrRet
}

// Input returns the input channel.
func (ap *asyncProducer) Input() chan<- *sarama.ProducerMessage {
	return ap.outerInput
}

// Successes returns the successes channel.
func (ap *asyncProducer) Successes() <-chan *sarama.ProducerMessage {
	return ap.outerSuccesses
}

// Errors returns the errors channel.
func (ap *asyncProducer) Errors() <-chan *sarama.ProducerError {
	return ap.outerErrors
}

// propagate inject tracing info into message for propagation
func propagate(span aitracer.Span, msg *sarama.ProducerMessage, tracer aitracer.Tracer) {
	if tracer == nil || span == nil {
		return
	}
	m := make(map[string][]string)
	err := tracer.Inject(span.Context(), aitracer.HTTPHeaders, aitracer.HTTPHeadersCarrier(m))
	if err != nil {
		return
	}
	for k, vs := range m {
		key := []byte(k)
		for _, v := range vs {
			msg.Headers = append(msg.Headers, sarama.RecordHeader{
				Key:   key,
				Value: []byte(v),
			})
		}
	}
}
