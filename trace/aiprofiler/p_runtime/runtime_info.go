package p_runtime

import (
	"encoding/json"
	"runtime"
	"strconv"
	"sync"

	"github.com/volcengine/apminsight-server-sdk-go/trace/aiprofiler/res_monitor"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/service_register/register_utils"
	"github.com/volcengine/apminsight-server-sdk-go/trace/internal"
	"github.com/volcengine/apminsight-server-sdk-go/trace/internal/agentless_adapter"
)

var (
	runtimeInfoStr string
	once           sync.Once
)

func GetRuntimeInfo() string {
	once.Do(func() {
		info, _ := register_utils.GetInfo()
		m := map[string]string{
			"runtime_type": internal.RuntimeTypeGo,
			"go_os":        runtime.GOOS,
			"go_arch":      runtime.GOARCH,
			"go_version":   runtime.Version(),
			"compiler":     runtime.Compiler,
			"cpu_num":      strconv.FormatInt(int64(runtime.NumCPU()), 10),
			"cpu_limit":    strconv.FormatInt(int64(res_monitor.GetCPULimit()), 10),
			"sdk_version":  internal.SDKVersionNum,
			"host":         agentless_adapter.GetHostname(),
			"instance_id":  register_utils.GetInstanceID(),
			"pid":          strconv.FormatInt(int64(info.Pid), 10),
			"container_id": info.ContainerId,
			"start_time":   strconv.FormatInt(info.StartTime, 10),
			"cmd_line":     info.Cmdline,
		}
		b, err := json.Marshal(m)
		if err != nil {
			return
		}
		runtimeInfoStr = string(b)
	})
	return runtimeInfoStr
}
