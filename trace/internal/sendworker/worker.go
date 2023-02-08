package sendworker

import (
	"net"
	"strings"

	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/logger"
)

type SendWorker interface {
	BatchSend([]byte, []byte)
	CloseConn()
}

type DatagramWorker struct {
	logger logger.Logger

	sock string
	conn net.Conn

	msgType string
}

func NewDatagramWorker(msgType, sock string, l logger.Logger) *DatagramWorker {
	if l == nil {
		l = &logger.NoopLogger{}
	}
	return &DatagramWorker{
		logger:  l,
		sock:    sock,
		msgType: msgType,
	}
}

func (w *DatagramWorker) BatchSend(data []byte, _ []byte) {
	if data == nil {
		return
	}
	w.logger.Debug("[DatagramWorker] send batch %s. data size %d bytes", w.msgType, len(data))

	if w.conn == nil {
		w.newConn()
		if w.conn == nil {
			return
		}
	}
	_, err := w.conn.Write(data)
	if err != nil {
		w.logger.Error("[DatagramWorker] send %s err %v", w.msgType, err)
		w.conn.Close()
		w.conn = nil
	}
}

func (w *DatagramWorker) newConn() {
	conn, err := net.Dial("unixgram", w.sock)
	if err != nil {
		w.logger.Error("[DatagramWorker] create conn %s err %v", w.sock, err)
		return
	}
	w.conn = conn
}

func (w *DatagramWorker) CloseConn() {
	if w.conn != nil {
		_ = w.conn.Close()
	}
}

type StreamWorker struct {
	logger logger.Logger

	sock string
	conn net.Conn

	msgType string
}

func NewStreamWorker(msgType, sock string, l logger.Logger) *StreamWorker {
	if l == nil {
		l = &logger.NoopLogger{}
	}
	return &StreamWorker{
		logger:  l,
		sock:    sock,
		msgType: msgType,
	}
}

func (w *StreamWorker) BatchSend(data []byte, tags []byte) {
	if len(data) == 0 {
		return
	}
	w.logger.Debug("[StreamWorker] send batch %s %d", w.msgType, len(data))

	payload := EncodePreAllocated(data, tags)

	if w.conn == nil {
		w.newConn()
		if w.conn == nil {
			return
		}
	}

	_, err := w.conn.Write(payload)
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "broken pipe") { // retry when server-agent has closed connection
		w.logger.Info("[StreamWorker] connection has been closed by remote. retrying send %s", w.msgType)
		w.CloseConn() // close current connection
		w.newConn()   // try to establish a new conn
		if w.conn == nil {
			return
		}
		_, err = w.conn.Write(payload) //retry once
	}
	if err != nil {
		w.logger.Error("[StreamWorker] send %s err %v", w.msgType, err)
		_ = w.conn.Close()
		w.conn = nil
	}
}

func (w *StreamWorker) newConn() {
	conn, err := net.Dial("unix", w.sock)
	if err != nil {
		w.logger.Error("[StreamWorker] create tcp conn %s err %v", w.sock, err)
		return
	}
	w.conn = conn
}

func (w *StreamWorker) CloseConn() {
	if w.conn != nil {
		_ = w.conn.Close()
	}
}
