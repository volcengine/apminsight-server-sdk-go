package metrics

import (
	"bytes"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type MetricsClient struct {
	monitor *monitor

	config  Config
	dataBuf chan *[]metricItem

	// batch to channel
	batchBuf  *[]metricItem
	batchLock sync.Mutex

	flusherStop chan struct{}
	flusherWg   sync.WaitGroup
	senderWg    sync.WaitGroup
}

func NewMetricClient(options ...ClientOption) *MetricsClient {
	config := Config{
		address: defaultAddress,
	}
	envAddress := os.Getenv("AI_METRICS_SOCK")
	if len(envAddress) != 0 {
		config.address = envAddress
	}
	for _, opt := range options {
		opt(&config)
	}
	mc := &MetricsClient{
		monitor:  newMonitor(),
		config:   config,
		dataBuf:  make(chan *[]metricItem, asyncChannelSize),
		batchBuf: getMetricItems(),

		flusherStop: make(chan struct{}),
	}
	return mc
}

func (mc *MetricsClient) Start() {
	mc.monitor.start()

	mc.flusherWg.Add(1)
	go func() {
		defer func() {
			mc.flusherWg.Done()
		}()
		mc.batchFlushLoop()
	}()
	for i := 0; i < asyncWokerNumber; i++ {
		mc.senderWg.Add(1)
		go func() {
			defer func() {
				mc.senderWg.Done()
			}()
			mc.sendLoop()
		}()
	}
}

func (mc *MetricsClient) Close() {
	close(mc.flusherStop)
	mc.flusherWg.Wait()

	close(mc.dataBuf)
	mc.senderWg.Wait()

	mc.monitor.stop()
}

func (mc *MetricsClient) EmitCounter(name string, value float64, tags map[string]string) error {
	return mc.emitMetric(mtCounter, name, value, tags)
}

func (mc *MetricsClient) EmitTimer(name string, value float64, tags map[string]string) error {
	return mc.emitMetric(mtTimer, name, value, tags)
}

func (mc *MetricsClient) EmitGauge(name string, value float64, tags map[string]string) error {
	return mc.emitMetric(mtGauge, name, value, tags)
}

func (mc *MetricsClient) emitMetric(mt uint8, name string, value float64, tags map[string]string) error {
	item := metricItem{
		mt:    mt,
		name:  name,
		value: value,
	}
	if len(tags) != 0 {
		item.tags = make([]t, 0, len(tags))
		for k, v := range tags {
			item.tags = append(item.tags, t{key: k, value: v})
		}
	}

	var flushBatch *[]metricItem
	mc.batchLock.Lock()
	*mc.batchBuf = append(*mc.batchBuf, item)
	if len(*mc.batchBuf) >= batchSize {
		flushBatch = mc.batchBuf
		mc.batchBuf = getMetricItems()
	}
	mc.batchLock.Unlock()
	if flushBatch != nil {
		select {
		case mc.dataBuf <- flushBatch:
		default:
			atomic.AddInt64(&mc.monitor.metricBufferFull, 1)
		}
	}
	return nil
}

func (mc *MetricsClient) batchFlushLoop() {
	ticker := time.Tick(time.Second)
	for {
		select {
		case <-ticker:
			mc.batchFlush()
		case <-mc.flusherStop:
			mc.batchFlush()
			return
		}
	}
}

func (mc *MetricsClient) batchFlush() {
	var flushBatch *[]metricItem
	mc.batchLock.Lock()
	if len(*mc.batchBuf) != 0 {
		flushBatch = mc.batchBuf
		mc.batchBuf = getMetricItems()
	}
	mc.batchLock.Unlock()
	if flushBatch != nil {
		mc.dataBuf <- flushBatch
	}
}

func (mc *MetricsClient) sendLoop() {
	sender := newSender(mc.config.address, mc.monitor)
	packetBuf := make([]byte, 0, maxPacketSize)
	itemBuf := bytes.NewBuffer(nil)

	prefix := mc.config.prefix
	for items := range mc.dataBuf {
		for _, item := range *items {
			itemBuf.Reset()
			err := formatCommon(itemBuf, item.mt, prefix, item.name, item.value, item.tags)
			if err != nil {
				atomic.AddInt64(&mc.monitor.formatError, 1)
				continue
			}
			data := itemBuf.Bytes()
			if len(packetBuf)+len(data) <= maxPacketSize {
				packetBuf = append(packetBuf, data...)
			} else {
				if len(data) > maxPacketSize {
					sender.SendPacket(data)
				} else {
					sender.SendPacket(packetBuf)
					packetBuf = packetBuf[:0]
					packetBuf = append(packetBuf, data...)
				}
			}
		}
		putMetricItems(items)
	}
}
