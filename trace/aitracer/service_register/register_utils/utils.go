package register_utils

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

var (
	cacheSelfInfo = cacheRegisterInfo{}
)

type cacheRegisterInfo struct {
	RegisterInfo
	err error
}

type RegisterInfo struct {
	Pid         int // could be pid or nspid
	StartTime   int64
	ContainerId string
	Cmdline     string
}

func GetInfo() (RegisterInfo, error) {
	if cacheSelfInfo.Pid != 0 {
		return cacheSelfInfo.RegisterInfo, cacheSelfInfo.err
	}
	cacheSelfInfo = getInfo()
	return cacheSelfInfo.RegisterInfo, cacheSelfInfo.err
}

const (
	clockTicks = 100 // C.sysconf(C._SC_CLK_TCK)
)

func getInfo() (info cacheRegisterInfo) {
	info.Pid = os.Getpid()
	{
		line, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/stat", info.Pid))
		if err != nil {
			info.err = err
			return
		}
		fields := strings.Fields(string(line))

		i := 1
		for !strings.HasSuffix(fields[i], ")") {
			i++
		}
		if i+20 >= len(fields) {
			info.err = errors.New("no uptime field")
			return
		}
		upTime, err := strconv.ParseUint(fields[i+20], 10, 64)
		if err != nil {
			info.err = err
			return
		}

		bootTime, err := getBootTime()
		if err != nil {
			info.err = err
			return
		}

		info.StartTime = int64(upTime/uint64(clockTicks) + uint64(bootTime))
	}
	{
		containerId, err := getSelfDockerId()
		if err != nil {
			info.err = err
			return
		}
		info.ContainerId = containerId
	}
	{
		cmdline, err := getCmdline(info.Pid)
		if err != nil {
			info.err = err
			return
		}
		info.Cmdline = cmdline
	}
	return
}

func readLines(filePath string) ([]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return []string{""}, err
	}
	defer f.Close()

	var ret []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		ret = append(ret, scanner.Text())
	}
	return ret, scanner.Err()
}

func getBootTime() (uint64, error) {
	lines, err := readLines("/proc/stat")
	if err != nil {
		return 0, err
	}
	for _, line := range lines {
		if strings.HasPrefix(line, "btime") {
			fields := strings.Fields(line)
			if len(fields) != 2 {
				return 0, fmt.Errorf("invalid btime")
			}
			btime, err := strconv.ParseInt(fields[1], 10, 64)
			if err != nil {
				return 0, err
			}
			return uint64(btime), nil
		}
	}
	return 0, fmt.Errorf("cannot get btime from /proc/stat")
}

func getSelfDockerId() (string, error) {
	data, err := ioutil.ReadFile("/proc/1/cpuset")
	if err != nil {
		return "", err
	}
	parts := bytes.Split(data, []byte("/"))
	if len(parts) == 0 {
		return "", nil
	}
	dockerId := string(bytes.TrimSpace(parts[len(parts)-1]))
	dockerId = strings.TrimPrefix(dockerId, "docker-")
	dockerId = strings.TrimSuffix(dockerId, ".scope")
	return dockerId, nil
}

func getCmdline(pid int) (string, error) {
	cmdInfo, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return "", err
	}
	if len(cmdInfo) == 0 {
		return "", nil
	}
	if cmdInfo[len(cmdInfo)-1] == 0 {
		cmdInfo = cmdInfo[:len(cmdInfo)-1]
	}
	fields := bytes.Split(cmdInfo, []byte{0})
	var cmdArgs []string
	for _, f := range fields {
		cmdArgs = append(cmdArgs, string(f))
	}
	return genMd5(strings.Join(cmdArgs, " ")), nil
}

func genMd5(str string) string {
	hash := md5.Sum([]byte(str))
	return hex.EncodeToString(hash[:])
}
