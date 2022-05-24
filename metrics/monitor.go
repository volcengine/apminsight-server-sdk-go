package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

type monitor struct {
	metricBufferFull int64
	senderDialError  int64
	senderWriteError int64
	formatError      int64

	stopChan chan struct{}
	wg       sync.WaitGroup
}

func newMonitor() *monitor {
	return &monitor{
		stopChan: make(chan struct{}),
	}
}

func (m *monitor) start() {
	ticker := time.NewTicker(time.Second)
	defer func() {
		ticker.Stop()
	}()

	m.wg.Add(1)
	go func() {
		defer func() {
			m.wg.Done()
		}()
		for {
			select {
			case <-ticker.C:
				m.showAndReset()
			case <-m.stopChan:
				m.showAndReset()
				return
			}
		}
	}()
}

func (m *monitor) stop() {
	close(m.stopChan)
}

func (m *monitor) showAndReset() {
	type item struct {
		v   *int64
		log string
	}
	for _, i := range []item{
		{v: &m.metricBufferFull, log: "metric buffer full"},
		{v: &m.senderWriteError, log: "sender write error"},
		{v: &m.senderDialError, log: "sender dial error"},
		{v: &m.formatError, log: "format error"},
	} {
		cv := atomic.SwapInt64(i.v, 0)
		if cv != 0 && logfunc != nil {
			logfunc("%s trigger %d times", i.log, cv)
		}
	}
}
