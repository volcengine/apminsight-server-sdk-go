package log_collector

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/log_collector/log_models"
)

const (
	maxBatchBytes = 16384
)

type LogCollector struct {
	// logger logger.Logger   // setting logger in LogCollector is dangerous. disable for safety
	// For example, if you have hooked logrus with tracer and set logrus as logger,
	// recursive call will cause: logrus.Info -> trace.Logger.Info -> logrus.Info

	sock string

	workerNumber int

	in chan *log_models.Log
	wg sync.WaitGroup

	debug bool
}

type LogCollectorConfig struct {
	Sock         string
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
	return &LogCollector{
		sock:         config.Sock,
		workerNumber: config.WorkerNumber,
		in:           make(chan *log_models.Log, config.ChanSize),
		debug:        config.Debug,
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
	for i := 0; i < s.workerNumber; i++ {
		s.wg.Add(1)
		go func() {
			defer func() {
				s.wg.Done()
			}()
			s.sendLoop()
		}()
	}
}

func (s *LogCollector) Stop() {
	close(s.in)
	s.wg.Wait()
}

func (s *LogCollector) sendLoop() {
	sw := &senderWorker{
		sock: s.sock,
	}
	defer func() {
		sw.closeConn()
	}()
	batchLog := make([]byte, 0, maxBatchBytes)
	tc := time.NewTicker(time.Second * 5)
	defer func() {
		tc.Stop()
	}()
	for {
		select {
		case <-tc.C:
			if len(batchLog) > 0 {
				sw.batchSend(batchLog)
				batchLog = batchLog[:0]
			}
		case item, ok := <-s.in:
			if !ok {
				if len(batchLog) > 0 {
					sw.batchSend(batchLog)
				}
				return
			}
			if item == nil {
				continue
			}
			if s.debug {
				fmt.Printf("[Debug] send logs %+v\n", item)
			}
			data, err := item.Marshal()
			if err != nil {
				fmt.Printf("[Error] send log marshal err %v\n", err)
				continue
			}
			sizePrefixData := make([]byte, 4)
			binary.LittleEndian.PutUint32(sizePrefixData, uint32(len(data))) // Is's ok to cast a positive int to uint32
			sizePrefixData = append(sizePrefixData, data...)

			if len(batchLog)+len(sizePrefixData) <= maxBatchBytes {
				batchLog = append(batchLog, sizePrefixData...)
			} else {
				if len(sizePrefixData) > maxBatchBytes {
					sw.batchSend(sizePrefixData) // this will lead agent to read truncated data. should discard directly here?
				} else {
					sw.batchSend(batchLog)
					batchLog = batchLog[:0]
					batchLog = append(batchLog, sizePrefixData...)
				}
			}
		}
	}
}

type senderWorker struct {
	sock string
	conn net.Conn
}

func (s *senderWorker) closeConn() {
	if s.conn != nil {
		s.conn.Close()
	}
}

func (s *senderWorker) batchSend(batchLog []byte) {
	if batchLog == nil {
		return
	}
	if s.conn == nil {
		s.conn = s.newConn()
		if s.conn == nil {
			return
		}
	}
	_, err := s.conn.Write(batchLog)
	if err != nil {
		fmt.Printf("[Error] send batch logs err %v\n", err)
		s.conn.Close()
		s.conn = nil
	}
}

func (s *senderWorker) newConn() net.Conn {
	conn, err := net.Dial("unixgram", s.sock)
	if err != nil {
		fmt.Printf("[Error] create conn %s err %v\n", s.sock, err)
		return nil
	}
	return conn
}
