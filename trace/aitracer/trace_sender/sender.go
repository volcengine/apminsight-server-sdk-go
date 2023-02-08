package trace_sender

import (
	"encoding/binary"
	"sync"
	"time"

	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/logger"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/trace_sender/trace_models"
	"github.com/volcengine/apminsight-server-sdk-go/trace/internal"
	"github.com/volcengine/apminsight-server-sdk-go/trace/internal/sendworker"
	"github.com/volcengine/apminsight-server-sdk-go/trace/internal/service_register/register_utils"
	"github.com/volcengine/apminsight-server-sdk-go/trace/internal/utils"
)

const (
	maxBatchBytes       = 16384    // should remain 16KB because old version of server-agent still use 16KB buffer
	streamMaxBatchBytes = 64 << 10 //64KB
)

func NewTraceSender(sock, streamSock string, in chan *trace_models.Trace, l logger.Logger) *TraceSender {
	agentVersion := utils.GetAgentVersion()
	if utils.CompareVersion(agentVersion.Version, internal.AgentVersionSupportStreamSender) >= 0 {
		return newStreamTraceSender(streamSock, in, l)
	}
	return newDatagramTraceSender(sock, in, l)
}

type TraceSender struct {
	logger logger.Logger

	in chan *trace_models.Trace
	wg sync.WaitGroup

	bufferMaxSize int
	offset        int // send data by stream need more buf to prefix magicNum/dataLen
	flushInterval time.Duration

	tags []byte

	w sendworker.SendWorker
}

func newDatagramTraceSender(sock string, in chan *trace_models.Trace, l logger.Logger) *TraceSender {
	if sock == "" {
		panic("sock address is empty")
	}
	if l == nil {
		l = &logger.NoopLogger{}
	}
	l.Info("newDatagramTraceSender success")
	return &TraceSender{
		logger: l,
		in:     in,

		bufferMaxSize: maxBatchBytes, //16KB
		offset:        0,             // do not need
		flushInterval: time.Second,
		tags:          nil, // datagram sender do not support tags

		w: sendworker.NewDatagramWorker("trace", sock, l),
	}
}

func newStreamTraceSender(sock string, in chan *trace_models.Trace, l logger.Logger) *TraceSender {
	if sock == "" {
		panic("sock address is empty")
	}
	if l == nil {
		l = &logger.NoopLogger{}
	}
	l.Info("newStreamTraceSender success")

	tagsBytes := sendworker.FormatMap(map[string]string{"instanceID": register_utils.GetInstanceID()})
	return &TraceSender{
		logger: l,
		in:     in,

		bufferMaxSize: streamMaxBatchBytes,                   //64KB
		offset:        sendworker.GetPrefixLen(1, tagsBytes), // preallocate prefix
		flushInterval: 5 * time.Second,
		tags:          tagsBytes,

		w: sendworker.NewStreamWorker("trace", sock, l),
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
		s.w.CloseConn()
	}()

	batchTrace := make([]byte, s.offset, s.bufferMaxSize+s.offset)
	tc := time.NewTicker(s.flushInterval)
	defer func() {
		tc.Stop()
	}()
	for {
		select {
		case <-tc.C:
			if len(batchTrace) > s.offset {
				s.w.BatchSend(batchTrace, s.tags)
				batchTrace = batchTrace[:s.offset]
			}
		case item, ok := <-s.in:
			if !ok {
				if len(batchTrace) > s.offset {
					s.w.BatchSend(batchTrace, s.tags)
				}
				return
			}
			if item == nil {
				continue
			}
			size := item.Size()
			sizePrefixData := make([]byte, 4+size)
			binary.LittleEndian.PutUint32(sizePrefixData[0:4], uint32(size)) // Is's ok to cast a positive int to uint32
			_, err := item.MarshalTo(sizePrefixData[4 : 4+size])
			if err != nil {
				s.logger.Error("send trace marshal err %v", err)
				continue
			}

			s.logger.Debug("send trace %+v, len=%d", item, size)

			if len(batchTrace)+len(sizePrefixData) <= s.bufferMaxSize+s.offset {
				batchTrace = append(batchTrace, sizePrefixData...)
			} else {
				if len(sizePrefixData) > s.bufferMaxSize+s.offset { // avoid grow batchTrace
					var tmpBuf []byte
					if s.offset > 0 {
						tmpBuf = make([]byte, s.offset, s.offset+len(sizePrefixData)) // preAlloc. very low chance to enter this condition
						tmpBuf = append(tmpBuf, sizePrefixData...)
					} else {
						tmpBuf = sizePrefixData
					}
					s.w.BatchSend(tmpBuf, s.tags) // this will lead agent to read truncated data. should discard directly here?
				} else {
					s.w.BatchSend(batchTrace, s.tags)
					batchTrace = batchTrace[:s.offset]
					batchTrace = append(batchTrace, sizePrefixData...)
				}
			}
		}
	}
}
