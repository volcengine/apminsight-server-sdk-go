package utils

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

const versionfile = "/var/run/apminsight/version"

type AgentVersion struct {
	Version string `json:"version"`
}

var (
	agentVersion = AgentVersion{}
	once         sync.Once
)

func GetAgentVersion() AgentVersion {
	once.Do(func() {
		var lines []string
		f, err := os.Open(versionfile)
		if err != nil {
			fmt.Printf("read agent version info fail. err=%+v. server-agent maybe be not running or too old\n", err)
			return
		}
		s := bufio.NewScanner(f)
		for s.Scan() {
			lines = append(lines, s.Text())
		}
		// first line is version
		if len(lines) > 0 {
			if idx := strings.IndexRune(lines[0], '='); idx > 0 {
				version := lines[0][idx+1:]
				agentVersion.Version = version
			}
		}
	})
	return agentVersion
}

// CompareVersion compare versions. if v1<v2, return -1; if v1>v2, return 1; if v1=v2, return 0,
// Only consider release version
func CompareVersion(v1, v2 string) int {
	s1 := parseVersion(v1)
	s2 := parseVersion(v2)

	// s1 and s2 may have different len
	releaseVersionLen := 3
	for i := 0; i < releaseVersionLen; i++ {
		var d1, d2 int
		if i >= len(s1) {
			d1 = 0
		} else {
			d1 = s1[i]
		}
		if i >= len(s2) {
			d2 = 0
		} else {
			d2 = s2[i]
		}
		if d1 < d2 {
			return -1
		} else if d1 > d2 {
			return 1
		}
	}
	return 0
}

func parseVersion(v string) []int {
	is := make([]int, 0)
	v = strings.ReplaceAll(v, "-", ".")
	v = strings.ReplaceAll(v, "rc.", "") // "rc" is the only possible letter is version
	for _, s := range strings.Split(v, ".") {
		is = append(is, getDigit(s))
	}
	return is
}

func getDigit(s string) int {
	num, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return num
}
