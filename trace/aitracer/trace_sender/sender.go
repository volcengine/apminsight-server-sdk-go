package trace_sender

import (
	"encoding/binary"
	"net"
	"sync"
	"time"

	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/logger"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/trace_sender/trace_models"
)

const maxBatchBytes = 16384

type TraceSender struct {
	logger logger.Logger

	sock string
	in   chan *trace_models.Trace
	wg   sync.WaitGroup
	conn net.Conn
}

func NewSender(sock string, in chan *trace_models.Trace, l logger.Logger) *TraceSender {
	if sock == "" {
		panic("sock address is empty")
	}
	if l == nil {
		l = &logger.NoopLogger{}
	}
	return &TraceSender{
		logger: l,
		sock:   sock,
		in:     in,
	}
}

func (s *TraceSender) Start() {
	s.wg.Add(1)
	go func() {
		defer func() {
			s.wg.Done()
		}()
		s.sendLoop()
	}()
}

func (s *TraceSender) WaitStop() {
	s.wg.Wait()
}

func (s *TraceSender) sendLoop() {
	defer func() {
		if s.conn != nil {
			s.conn.Close()
		}
	}()
	batchTrace := make([]byte, 0, maxBatchBytes)
	tc := time.NewTicker(time.Second)
	for {
		select {
		case <-tc.C:
			if len(batchTrace) > 0 {
				s.batchSend(batchTrace)
				batchTrace = batchTrace[:0]
			}
		case item, ok := <-s.in:
			if !ok {
				if len(batchTrace) > 0 {
					s.batchSend(batchTrace)
				}
				return
			}
			if item == nil {
				continue
			}
			s.logger.Debug("send trace %+v", item)
			data, err := item.Marshal()
			if err != nil {
				s.logger.Error("send trace marshal err %v", err)
				continue
			}
			sizePrefixData := make([]byte, 4)
			binary.LittleEndian.PutUint32(sizePrefixData, uint32(len(data))) // Is's ok to cast a positive int to uint32
			sizePrefixData = append(sizePrefixData, data...)

			if len(batchTrace)+len(sizePrefixData) <= maxBatchBytes {
				batchTrace = append(batchTrace, sizePrefixData...)
			} else {
				if len(sizePrefixData) > maxBatchBytes {
					s.batchSend(sizePrefixData) // this will lead agent to read truncated data. should discard directly here?
				} else {
					s.batchSend(batchTrace)
					batchTrace = batchTrace[:0]
					batchTrace = append(batchTrace, sizePrefixData...)
				}
			}
		}
	}
}

func (s *TraceSender) batchSend(batchTrace []byte) {
	if batchTrace == nil {
		return
	}
	s.logger.Debug("send batch trace %d", len(batchTrace))

	if s.conn == nil {
		s.conn = s.newConn()
		if s.conn == nil {
			return
		}
	}
	_, err := s.conn.Write(batchTrace)
	if err != nil {
		s.logger.Error("send trace err %v", err)
		s.conn.Close()
		s.conn = nil
	}
}

func (s *TraceSender) newConn() net.Conn {
	conn, err := net.Dial("unixgram", s.sock)
	if err != nil {
		s.logger.Error("create conn %s err %v", s.sock, err)
		return nil
	}
	return conn
}
