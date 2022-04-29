package runtime

import (
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/volcengine/apminsight-server-sdk-go/metrics"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/service_register/register_utils"
)

// 进程数据是server_agent采集的, 为了出服务的资源占用图, 需要将服务的数据从进程数据中fork一份写到自定义打点中
// 之前jvm数据采用了java_agent上报jvm结构体, performance_sink解析的方案, 存了一份进程jvm数据, 又存了一份服务jvm数据. 现在看来这二者是等价的.
// 完全可以通过只存一份服务的jvm数据, 然后同时支持进程tag的访问.
// 因此golang的runtime监控采用打自定义指标的方式. 后续jvm也替换成自定义指标.

// 通过metric客户端生成go的runtime打点
var logfunc func(string, ...interface{})

type Config struct {
	intervalSeconds int64 //time interval between two stats read
	tagsList        []map[string]string
}

func newDefaultConfig() *Config {
	return &Config{
		intervalSeconds: 30,
	}
}

type Option func(*Config)

// WithAdditionalTags add tags
func WithAdditionalTags(tags map[string]string) Option {
	return func(config *Config) {
		config.tagsList = append(config.tagsList, tags)
	}
}

func SetLogFunc(_logfunc func(string, ...interface{})) {
	logfunc = _logfunc
}

type stats struct {
	goRoutine int
	cgoCall   int64
	*runtime.MemStats
}

type Monitor struct {
	metricsClient *metrics.MetricsClient

	intervalSeconds int64
	tags            map[string]string

	serviceType string
	service     string
	instanceId  string
	pid         string
	createTime  string
	containerId string

	closeChan chan struct{}
	wg        sync.WaitGroup

	preStats *stats
}

func NewMonitor(serviceType, service string, mc *metrics.MetricsClient, opts ...Option) *Monitor {
	if mc == nil {
		mc = metrics.NewMetricClient()
	}

	cfg := newDefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	info, _ := register_utils.GetInfo()
	rm := &Monitor{
		metricsClient:   mc,
		intervalSeconds: cfg.intervalSeconds,

		serviceType: serviceType,
		service:     service,
		instanceId:  register_utils.GetInstanceID(),
		pid:         strconv.FormatInt(int64(info.Pid), 10),
		createTime:  strconv.FormatInt(info.StartTime, 10),
		containerId: info.ContainerId,
		closeChan:   make(chan struct{}),
	}

	// add tags
	tags := make(map[string]string)
	for _, ts := range cfg.tagsList {
		for k, v := range ts {
			tags[k] = v
		}
	}
	tags["service_type"] = serviceType
	tags["service"] = rm.service
	tags["instance_id"] = rm.instanceId // 需要开启服务注册. 将服务注册的代码从trace中移出
	//tags["pid"] = rm.pid        //in docker pid is actually NSPid. should be ignored.
	tags["create_time"] = rm.createTime
	tags["container_id"] = rm.containerId
	rm.tags = tags

	return rm
}

func (r *Monitor) Start() {
	r.metricsClient.Start() //能被重复调用么?
	r.wg.Add(1)
	go func() {
		tc := time.NewTicker(time.Duration(r.intervalSeconds) * time.Second)
		defer func() {
			tc.Stop()
			r.wg.Done()
		}()
		r.run()
		for {
			select {
			case <-tc.C:
				r.run()
			case <-r.closeChan:
				break
			}
		}
	}()
	if logfunc != nil {
		logfunc("[Monitor] Start success")
	}
}

func (r *Monitor) Close() {
	r.metricsClient.Close()
	close(r.closeChan)
	r.wg.Wait()

}

func (r *Monitor) readStats() *stats {
	ms := runtime.MemStats{} //mem
	runtime.ReadMemStats(&ms)
	return &stats{
		goRoutine: runtime.NumGoroutine(), //goroutines
		cgoCall:   runtime.NumCgoCall(),   //cgo调用
		MemStats:  &ms,
	}
}

func (r *Monitor) run() {
	curStats := r.readStats()
	preStats := &stats{
		goRoutine: 0,
		cgoCall:   0,
		MemStats:  &runtime.MemStats{},
	}
	if r.preStats != nil {
		preStats = r.preStats
	}

	defer func() {
		r.preStats = curStats // update preStats
	}()

	// emit metric
	if logfunc != nil && curStats != nil && curStats.MemStats != nil {
		logfunc("[Monitor] running. stats: goRoutine=%+v , cgoCall=%+v , MemStat=%+v", curStats.goRoutine, curStats.cgoCall, *curStats.MemStats)
	}

	// goroutine 数量
	_ = r.metricsClient.EmitGauge(metricGoRuntimeGoRoutineNum, float64(curStats.goRoutine), r.tags)

	// cgo调用次数. cumulative次数
	_ = r.metricsClient.EmitCounter(metricGoRuntimeCgoCallCount, float64(curStats.cgoCall-preStats.cgoCall), r.tags)

	// gc次数. cumulative次数
	_ = r.metricsClient.EmitCounter(metricGoRuntimeGcCount, float64(curStats.NumGC-preStats.NumGC), r.tags)
	// 累积gc时间. 采样间隔之间的gc时间=本次totalNs-上次totalNs, 得到两次采样点之间的gc耗时. 可以计算gc总耗时和gc时间占比
	_ = r.metricsClient.EmitCounter(metricGoRuntimeGcCostTotal, float64(curStats.PauseTotalNs-preStats.PauseTotalNs)/1000, r.tags)
	// 单次gc时间
	for _, gcPauseNs := range getGcPausesNs(curStats.PauseNs[:], preStats.NumGC, curStats.NumGC) {
		_ = r.metricsClient.EmitTimer(metricGoRuntimeGcCostDistribute, float64(gcPauseNs)/1000, r.tags)
	}

	// Heap
	_ = r.metricsClient.EmitGauge(metricGoRuntimeMemStatsHeapAlloc, float64(curStats.HeapAlloc), r.tags)
	_ = r.metricsClient.EmitGauge(metricGoRuntimeMemStatsHeapSys, float64(curStats.HeapSys), r.tags)
	_ = r.metricsClient.EmitGauge(metricGoRuntimeMemStatsHeapIdle, float64(curStats.HeapIdle), r.tags)
	_ = r.metricsClient.EmitGauge(metricGoRuntimeMemStatsHeapInuse, float64(curStats.HeapInuse), r.tags)
	_ = r.metricsClient.EmitGauge(metricGoRuntimeMemStatsHeapReleased, float64(curStats.HeapReleased), r.tags)
	_ = r.metricsClient.EmitGauge(metricGoRuntimeMemStatsHeapObjets, float64(curStats.HeapObjects), r.tags)

	// pointer lookups
	_ = r.metricsClient.EmitCounter(metricGoRuntimeMemStatsLookups, float64(curStats.Lookups-preStats.Lookups), r.tags)

	// Stack
	_ = r.metricsClient.EmitGauge(metricGoRuntimeMemStatsStackInuse, float64(curStats.StackInuse), r.tags)
	_ = r.metricsClient.EmitGauge(metricGoRuntimeMemStatsStackSys, float64(curStats.StackSys), r.tags)
	_ = r.metricsClient.EmitGauge(metricGoRuntimeMemStatsMSpanInuse, float64(curStats.MSpanInuse), r.tags)
	_ = r.metricsClient.EmitGauge(metricGoRuntimeMemStatsMSpanSys, float64(curStats.MSpanSys), r.tags)
	_ = r.metricsClient.EmitGauge(metricGoRuntimeMemStatsMCacheInuse, float64(curStats.MCacheInuse), r.tags)
	_ = r.metricsClient.EmitGauge(metricGoRuntimeMemStatsMCacheSys, float64(curStats.MCacheSys), r.tags)
	_ = r.metricsClient.EmitGauge(metricGoRuntimeMemStatsBuckHashSys, float64(curStats.BuckHashSys), r.tags)
	_ = r.metricsClient.EmitGauge(metricGoRuntimeMemStatsGcSys, float64(curStats.GCSys), r.tags)
	_ = r.metricsClient.EmitGauge(metricGoRuntimeMemStatsOtherSys, float64(curStats.OtherSys), r.tags)

	// gc指标
	_ = r.metricsClient.EmitGauge(metricGoRuntimeMemStatsNextGc, float64(curStats.NextGC), r.tags)
	_ = r.metricsClient.EmitCounter(metricGoRuntimeMemStatsNumForcedGc, float64(curStats.NumForcedGC-preStats.NumForcedGC), r.tags)
	_ = r.metricsClient.EmitGauge(metricGoRuntimeMemStatsGCCPUFraction, curStats.GCCPUFraction, r.tags)

	// derived metric
	_ = r.metricsClient.EmitGauge(metricGoRuntimeMemStatsLiveObjects, float64(curStats.Mallocs-curStats.Frees), r.tags)
	_ = r.metricsClient.EmitGauge(metricGoRuntimeMemStatsHeapRetained, float64(curStats.HeapIdle-curStats.HeapReleased), r.tags)
	_ = r.metricsClient.EmitGauge(metricGoRuntimeMemStatsHeapFragment, float64(curStats.HeapInuse-curStats.Alloc), r.tags)
}

func getGcPausesNs(pauseNs []uint64, preNumGc, curNumGc uint32) []uint64 {
	count := int(curNumGc) - int(preNumGc)
	if count <= 0 {
		return nil
	}
	if count >= len(pauseNs) {
		return pauseNs
	}

	length := uint32(len(pauseNs))
	i := preNumGc % length
	j := curNumGc % length
	if j < i {
		part := pauseNs[i:]
		part = append(part, pauseNs[:j]...)
		return part
	}
	return pauseNs[i:j]
}
