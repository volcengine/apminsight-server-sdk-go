package metrics

import (
	"net"
	"sync/atomic"
)

type sender struct {
	monitor *monitor
	address string
	conn    net.Conn
}

func newSender(address string, monitor *monitor) *sender {
	return &sender{
		monitor: monitor,
		address: address,
	}
}

func (s *sender) SendPacket(packet []byte) {
	if s.conn == nil {
		var err error
		s.conn, err = net.Dial("unixgram", s.address)
		if err != nil {
			atomic.AddInt64(&s.monitor.senderDialError, 1)
			if logfunc != nil {
				logfunc("dial address %s err %v", s.address, err)
			}
			return
		}
	}
	_, err := s.conn.Write(packet)
	if err != nil {
		s.conn = nil
		atomic.AddInt64(&s.monitor.senderWriteError, 1)
		if logfunc != nil {
			logfunc("write conn packet %d bytes err %v", len(packet), err)
		}
		return
	}

}
