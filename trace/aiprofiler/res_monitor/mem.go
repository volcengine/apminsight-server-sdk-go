package res_monitor

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"strconv"

	"github.com/volcengine/apminsight-server-sdk-go/trace/internal/service_register/register_utils"
)

var pageSize = os.Getpagesize()

const (
	memCGroupPath = "/sys/fs/cgroup/memory" // current only v1 is supported
	memLimitFile  = "memory.limit_in_bytes"

	hostMeminfo = "/proc/meminfo"
)

type MemMonitor struct {
	memLimit int64
	memRss   int64
}

func NewMemMonitor() *MemMonitor {
	return &MemMonitor{
		memLimit: getMemLimit(),
	}
}

// GetMemRatio gets the current rssRatio. Be aware this method is not cached.
func (m *MemMonitor) GetMemRatio() float64 {
	rss := getRss()
	m.memRss = rss

	if m.memLimit <= 0 || m.memRss <= 0 {
		return 0
	}
	return math.Min(float64(m.memRss)/float64(m.memLimit), 1)
}

func getRss() int64 {
	data, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/statm", os.Getpid()))
	if err != nil {
		return 0
	}
	s := bytes.Fields(data)
	res, err := strconv.ParseInt(string(s[1]), 10, 64)
	if err != nil {
		return 0
	}
	return res * int64(pageSize)
}

func getMemLimit() int64 {
	hostMem := getLimitFromMeminfo()

	// not in a container
	info, _ := register_utils.GetInfo()
	if info.ContainerId == "" {
		return hostMem
	}

	// try to get cGroup limit
	memLimit, err := strconv.ParseInt(readFirstLine(memCGroupPath, memLimitFile), 10, 64)
	if err != nil || memLimit <= 0 {
		return hostMem
	}

	return memLimit
}

// memory unit is byte
func getLimitFromMeminfo() int64 {
	data, err := ioutil.ReadFile(hostMeminfo)
	if err == nil && len(data) != 0 {
		idx := bytes.Index(data, []byte("MemTotal"))
		line := bytes.Fields(data[idx:])[1]
		res, _ := strconv.ParseInt(string(line), 10, 64) // res's unit is KB
		return res * 1024
	}
	return 0
}
