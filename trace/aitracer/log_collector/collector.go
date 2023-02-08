package log_collector

import (
	"encoding/binary"
	"sync"
	"time"

	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/log_collector/log_models"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/logger"
	"github.com/volcengine/apminsight-server-sdk-go/trace/internal"
	"github.com/volcengine/apminsight-server-sdk-go/trace/internal/sendworker"
	"github.com/volcengine/apminsight-server-sdk-go/trace/internal/service_register/register_utils"
	"github.com/volcengine/apminsight-server-sdk-go/trace/internal/utils"
)

const (
	maxBatchBytes       = 16384
	streamMaxBatchBytes = 64 << 10 //64KB
)

type LogCollector struct {
	logger logger.Logger // setting logger in LogCollector is dangerous. we do not allow setting it, use debug option instead
	// For example, if you have hooked logrus with tracer and set logrus as logger,
	// recursive call will cause: logrus.Info -> trace.Logger.Info -> logrus.Info
	in chan *log_models.Log
	wg sync.WaitGroup

	bufferMaxSize int
	offset        int // send data by stream need more buf to prefix magicNum/dataLen
	flushInterval time.Duration

	tags []byte

	ws []sendworker.SendWorker
}

type LogCollectorConfig struct {
	Sock         string
	StreamSock   string
	ChanSize     int
	WorkerNumber int
	Debug        bool
}

func NewLogCollector(config LogCollectorConfig) *LogCollector {
	if config.Sock == "" {
		panic("socket address is empty")
	}
	if config.WorkerNumber <= 0 {
		panic("worker must be positive")
	}
	if config.ChanSize <= 0 {
		panic("channel size must be positive")
	}
	agentVersion := utils.GetAgentVersion()
	if utils.CompareVersion(agentVersion.Version, internal.AgentVersionSupportStreamSender) >= 0 {
		return newStreamLogCollector(config)
	}
	return newDatagramLogCollector(config)
}

func newDatagramLogCollector(config LogCollectorConfig) *LogCollector {
	var l logger.Logger
	if config.Debug {
		l = &logger.DebugLogger{}
	} else {
		l = &logger.NoopLogger{}
	}
	ws := make([]sendworker.SendWorker, 0, config.WorkerNumber)
	for i := 0; i < config.WorkerNumber; i++ {
		ws = append(ws, sendworker.NewDatagramWorker("log", config.Sock, l))
	}
	l.Info("newDatagramLogCollector success")
	return &LogCollector{
		logger:        l,
		in:            make(chan *log_models.Log, config.ChanSize),
		bufferMaxSize: maxBatchBytes, //16KB
		offset:        0,             // do not need
		flushInterval: time.Second,
		tags:          nil, // datagram sender do not support tags
		ws:            ws,
	}
}

func newStreamLogCollector(config LogCollectorConfig) *LogCollector {
	var l logger.Logger
	if config.Debug {
		l = &logger.DebugLogger{}
	} else {
		l = &logger.NoopLogger{}
	}
	ws := make([]sendworker.SendWorker, 0, config.WorkerNumber)
	for i := 0; i < config.WorkerNumber; i++ {
		ws = append(ws, sendworker.NewStreamWorker("log", config.StreamSock, l))
	}
	l.Info("newStreamLogCollector success")
	tagsBytes := sendworker.FormatMap(map[string]string{"instanceID": register_utils.GetInstanceID()})
	return &LogCollector{
		logger:        l,
		in:            make(chan *log_models.Log, config.ChanSize),
		bufferMaxSize: streamMaxBatchBytes,                   // 64KB
		offset:        sendworker.GetPrefixLen(1, tagsBytes), // preallocate prefix
		flushInterval: 5 * time.Second,
		tags:          tagsBytes,
		ws:            ws,
	}
}

func (s *LogCollector) Send(log *log_models.Log) {
	select {
	case s.in <- log:
	default:
		break
	}
}

func (s *LogCollector) Start() {
	for _, w := range s.ws {
		s.wg.Add(1)
		go func(iw sendworker.SendWorker) {
			defer func() {
				s.wg.Done()
			}()
			s.sendLoop(iw)
		}(w)
	}
}

func (s *LogCollector) Stop() {
	close(s.in)
	s.wg.Wait()
}

func (s *LogCollector) sendLoop(w sendworker.SendWorker) {
	defer func() {
		w.CloseConn()
	}()
	batchLog := make([]byte, s.offset, s.bufferMaxSize+s.offset)
	tc := time.NewTicker(s.flushInterval)
	defer func() {
		tc.Stop()
	}()
	for {
		select {
		case <-tc.C:
			if len(batchLog) > s.offset {
				w.BatchSend(batchLog, s.tags)
				batchLog = batchLog[:s.offset]
			}
		case item, ok := <-s.in:
			if !ok {
				if len(batchLog) > s.offset {
					w.BatchSend(batchLog, s.tags)
				}
				return
			}
			if item == nil {
				continue
			}
			size := item.Size()
			sizePrefixData := make([]byte, 4+size)
			binary.LittleEndian.PutUint32(sizePrefixData[0:4], uint32(size)) // Is's ok to cast a positive int to uint32
			_, err := item.MarshalTo(sizePrefixData[4:])
			if err != nil {
				s.logger.Error("send log marshal err %v", err)
				continue
			}

			s.logger.Debug("send logs %+v, len=%d", item, size)

			if len(batchLog)+len(sizePrefixData) <= s.bufferMaxSize+s.offset {
				batchLog = append(batchLog, sizePrefixData...)
			} else {
				if len(sizePrefixData) > s.bufferMaxSize+s.offset { // avoid grow batchLog
					var tmpBuf []byte
					if s.offset > 0 {
						tmpBuf = make([]byte, s.offset, s.offset+len(sizePrefixData)) // preAlloc. very low chance to enter this condition
						tmpBuf = append(tmpBuf, sizePrefixData...)
					} else {
						tmpBuf = sizePrefixData
					}
					w.BatchSend(tmpBuf, s.tags) // this will lead agent to read truncated data. should discard directly here?
				} else {
					w.BatchSend(batchLog, s.tags)
					batchLog = batchLog[:s.offset]
					batchLog = append(batchLog, sizePrefixData...)
				}
			}
		}
	}
}
