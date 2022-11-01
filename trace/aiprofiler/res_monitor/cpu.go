package res_monitor

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/volcengine/apminsight-server-sdk-go/trace/internal/service_register/register_utils"
)

const (
	defaultHz = 100
)

const (
	cpuCGroupPath      = "/sys/fs/cgroup/cpu,cpuacct" // current only v1 is supported
	cpuCFSPeriodUsFile = "cpu.cfs_period_us"
	cpuCFSQuotaUsFile  = "cpu.cfs_quota_us"
)

type CPUMonitor struct {
	cpuLimit float64
	hz       float64 // how many ticks per second
	time     time.Time
	tick     int64
}

func NewCPUMonitor() *CPUMonitor {
	tick, t := getTicks()
	return &CPUMonitor{
		cpuLimit: GetCPULimit(),
		hz:       float64(getHz()),
		tick:     tick,
		time:     t,
	}
}

// GetCPURatio gets the cpuRatio between current and last invoking. Be aware this method is not cached.
func (m *CPUMonitor) GetCPURatio() float64 {
	newTick, newTime := getTicks()

	et := float64(newTime.Sub(m.time).Milliseconds())
	ticks := newTick - m.tick // ticks process used between two invoking

	m.tick = newTick
	m.time = newTime

	if ticks <= 0 || et <= 0 {
		return 0
	}
	ticksAllCPUCore := m.cpuLimit * m.hz * et / 1000 // total ticks of all cores between two invoking. hz means how many ticks per sec, so needs to divide 1000

	return math.Min(float64(ticks)/ticksAllCPUCore, 1)
}

func getTicks() (int64, time.Time) {
	data, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/stat", os.Getpid()))
	if err != nil {
		return 0, time.Time{}
	}

	now := time.Now()
	s := bytes.Fields(data)
	utime, _ := strconv.ParseInt(string(s[13]), 10, 64)
	stime, _ := strconv.ParseInt(string(s[14]), 10, 64)
	cutime, _ := strconv.ParseInt(string(s[15]), 10, 64)
	cstime, _ := strconv.ParseInt(string(s[16]), 10, 64)

	return utime + stime + cutime + cstime, now
}

// GetCPULimit returns cpu cores that process can use
func GetCPULimit() (cpuLimit float64) {
	defer func() {
		// compare cpuLimit and GOMAXPROCS, return the lower
		cpuLimit = math.Min(cpuLimit, float64(runtime.GOMAXPROCS(0))) //set to 0 gets the current setting without changing it
	}()

	cpuLimit = float64(runtime.NumCPU())
	// not insider a container, return directly
	info, _ := register_utils.GetInfo()
	if info.ContainerId == "" {
		return
	}

	// insider a container, try to get cgroup limit
	periodUs, err := strconv.ParseInt(readFirstLine(cpuCGroupPath, cpuCFSPeriodUsFile), 10, 64)
	if err != nil || periodUs <= 0 {
		return
	}
	quotaUs, err := strconv.ParseInt(readFirstLine(cpuCGroupPath, cpuCFSQuotaUsFile), 10, 64)
	if err != nil || quotaUs <= 0 {
		return
	}
	if t := float64(quotaUs) / float64(periodUs); t > 0 { // what if cgroup cpuLimit is 4, and runtime.GOMAXPROCS(2) is set?
		cpuLimit = t
	}
	return
}

func readFirstLine(path, fileName string) string {
	file, err := os.Open(filepath.Join(path, fileName))
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		return scanner.Text()
	}
	if err := scanner.Err(); err != nil {
		return ""
	}
	return ""
}

func getHz() (hz int64) {
	clkTck, err := exec.Command("getconf", "CLK_TCK").Output()
	if err != nil {
		return defaultHz
	}
	if hz, err := strconv.ParseInt(strings.Trim(string(clkTck), "\n"), 10, 64); err == nil && hz != 0 {
		return hz
	}
	return defaultHz
}
