package runtime

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// TestMonitor should run on a host where server-agent is active
func TestMonitor(t *testing.T) {
	SetLogFunc(logrus.Infof)
	runtimeMonitor := NewMonitor("server_runtime", "http", nil)
	runtimeMonitor.Start()

	time.Sleep(10 * time.Minute)
}
