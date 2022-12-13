package aiprofiler

import (
	"encoding/json"
	"runtime"
	"sync"
	"time"

	"github.com/volcengine/apminsight-server-sdk-go/trace/aiprofiler/common"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aiprofiler/manager"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aiprofiler/p_runtime"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aiprofiler/profile_models"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aiprofiler/res_monitor"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aiprofiler/sender"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/logger"
	"github.com/volcengine/apminsight-server-sdk-go/trace/internal"
	"github.com/volcengine/apminsight-server-sdk-go/trace/internal/agentless_adapter"
	"github.com/volcengine/apminsight-server-sdk-go/trace/internal/service_register"
	"github.com/volcengine/apminsight-server-sdk-go/trace/internal/service_register/register_utils"
)

const (
	defaultQueuedTaskChanSize = 100
	defaultOutChanSize        = 100

	defaultBlockRate            = 100000000
	defaultDefaultMutexFraction = 100

	defaultBackoffInterval = time.Minute

	defaultSockAddr = "/var/run/apminsight/comm.sock"
)

type Config struct {
	TaskChanSize int
	OutChanSize  int

	SenderCfg   sender.Config
	SettingsCfg manager.SettingAddrConfig

	// service register
	ServiceRegisterCfg service_register.Config

	// pprof setting
	mutexFraction int
	blockRate     int

	Logger logger.Logger
}

type Profiler struct {
	serviceType string
	service     string

	taskChan chan *common.Task
	outChan  chan *profile_models.ProfileInfo

	resMonitor *res_monitor.Monitor

	serviceRegister *service_register.Register

	manager *manager.Manager

	sender sender.Sender

	wg sync.WaitGroup

	logger logger.Logger
}

type Option func(*Config)

// by default settings and profile data is transmitted via server-agent using local uds.
// WithHTTPEndPoint can set remote http endpoint thus bypass server-agent
func newDefaultConfig() *Config {
	return &Config{
		TaskChanSize: defaultQueuedTaskChanSize,
		OutChanSize:  defaultOutChanSize,
		ServiceRegisterCfg: service_register.Config{
			Sock:     defaultSockAddr,
			Interval: 30 * time.Second,
			Timeout:  500 * time.Millisecond,
		},
		SenderCfg: sender.Config{
			Sock:            defaultSockAddr,
			Timeout:         500 * time.Millisecond,
			BackoffInterval: defaultBackoffInterval,
		},
		SettingsCfg: manager.SettingAddrConfig{
			Sock:    defaultSockAddr,
			Timeout: 500 * time.Millisecond,
		},
	}
}

// WithHTTPEndPoint set http endpoint for both settings and profile data thus bypass server-agent,
// which is useful where server-agent is not installed
func WithHTTPEndPoint(schema, host string, timeout time.Duration) Option {
	return func(cfg *Config) {
		// service register endpoint
		cfg.ServiceRegisterCfg.Schema = schema
		cfg.ServiceRegisterCfg.Host = host
		cfg.ServiceRegisterCfg.Timeout = timeout

		// sender endpoint
		cfg.SenderCfg.Schema = schema
		cfg.SenderCfg.Host = host
		cfg.SenderCfg.Timeout = timeout

		// settings endpoint
		cfg.SettingsCfg.Schema = schema
		cfg.SettingsCfg.Host = host
		cfg.SettingsCfg.Timeout = timeout
	}
}

// WithBackoffInterval set wait interval between retries when data upload fail
func WithBackoffInterval(internal time.Duration) Option {
	return func(cfg *Config) {
		cfg.SenderCfg.BackoffInterval = internal
	}
}

// WithRetryCount set retry count when data upload fail
func WithRetryCount(count int) Option {
	return func(cfg *Config) {
		if count < 0 {
			count = 0
		}
		if count > 5 {
			count = 5
		}
		cfg.SenderCfg.RetryCount = count
	}
}

// WithLogger set logger used in profiler
func WithLogger(l logger.Logger) Option {
	return func(config *Config) {
		config.Logger = l
		config.SenderCfg.Logger = l
		config.ServiceRegisterCfg.Logger = l
	}
}

// WithBlockProfile enables blockProfile with rate, rate's unit is nanoseconds, which means 1 blocking event per rate nanoseconds is reported. see runtime.SetBlockProfileRate
// BlockProfile is disabled by default.
// In most cases, BlockProfile has low CPU overhead. However, please be aware that BlockProfile can cause CPU overhead under certain circumstance. (Setting rate to 10,000ns may cause up to 4% CPU overhead in some scenario according
// to document https://github.com/DataDog/go-profiler-notes/blob/main/guide/README.md#block-profiler-limitations).
// For safety defaultBlockRate is set to 100,000,000ns(100ms), meaning block events with duration longer than 100ms will be recorded,
// while block events with duration of 1ms has a 1% chance to be recorded.
func WithBlockProfile(rate int) Option {
	return func(config *Config) {
		config.blockRate = rate
	}
}

func WithBlockProfileDefault() Option {
	return func(config *Config) {
		config.blockRate = defaultBlockRate
	}
}

// WithMutexProfile enables mutexProfile with fraction, which means on average 1/fraction events are reported. see runtime.SetMutexProfileFraction
// MutexProfile is disabled by default.
// When using modest fraction (e.g. 100) MutexProfile has little impact on performance.
func WithMutexProfile(fraction int) Option {
	return func(config *Config) {
		config.mutexFraction = fraction
	}
}

func WithMutexProfileDefault() Option {
	return func(config *Config) {
		config.mutexFraction = defaultDefaultMutexFraction
	}
}

// NewProfiler fetch profileTasks from remoteConfig then profile and send pprof data to backend
func NewProfiler(serviceType, service string, opts ...Option) *Profiler {
	cfg := newDefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.Logger == nil {
		cfg.Logger = &logger.NoopLogger{}
	}

	{
		if cfg.blockRate > 0 {
			runtime.SetBlockProfileRate(cfg.blockRate)
		}
		if cfg.mutexFraction > 0 {
			runtime.SetMutexProfileFraction(cfg.mutexFraction)
		}
	}

	taskChan := make(chan *common.Task, cfg.TaskChanSize)              // manager -> fetch profileSettings -> drivers gen tasks -> taskChan
	outChan := make(chan *profile_models.ProfileInfo, cfg.OutChanSize) // taskChan -> profiler -> outChan -> sender -> backend

	p := Profiler{
		serviceType: serviceType,
		service:     service,
		taskChan:    taskChan,
		outChan:     outChan,
		logger:      cfg.Logger,
	}

	resMonitor := res_monitor.NewMonitor()
	p.resMonitor = resMonitor

	// register is singleton, so is safe to call it multiple time
	p.serviceRegister = service_register.GetRegister(serviceType, service, cfg.ServiceRegisterCfg)

	p.manager = manager.NewManager(service, cfg.SettingsCfg, taskChan, resMonitor, cfg.Logger)
	p.sender = sender.NewSender(cfg.SenderCfg, outChan)

	cfg.Logger.Info("[NewProfiler] init profiler success. config is %+v", cfg)

	return &p
}

func (p *Profiler) Start() {
	p.logger.Info("Staring profiler...")

	p.serviceRegister.Start()
	p.resMonitor.Start()
	p.manager.Start()
	p.sender.Start()
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		p.runLoop()
	}()

	p.logger.Info("profiler Started")

}

func (p *Profiler) Stop() {
	p.logger.Info("Stopping profiler...")

	p.manager.Stop()         // close manager first to avoid write to a closed chan
	close(p.taskChan)        // no task will be executed
	p.wg.Wait()              // wait runLoop/run to stop, which means no data will be sent
	p.resMonitor.Stop()      // res monitor is needed in task-run
	p.sender.Stop()          // close sender
	p.serviceRegister.Stop() // close register

	p.logger.Info("profiler stopped")
}

// DebugTasks create a debug task.
// DEBUG ONLY.
func (p *Profiler) DebugTasks(tasks []*common.Task) {
	for _, task := range tasks {
		p.taskChan <- task
	}
}

func (p *Profiler) runLoop() {
	for {
		select {
		case task, ok := <-p.taskChan: // tasks are processed serialized
			if !ok {
				return
			}
			p.run(task)
		}
	}
}

func (p *Profiler) run(task *common.Task) {
	task.StartTimeMilliSec = time.Now().Unix()*1e3 + int64(time.Now().Nanosecond())/1e6 // record the timestamp when task begin running

	wg := sync.WaitGroup{}
	l := sync.Mutex{}
	profiles := make([]*common.ProfileData, 0)

	for _, pt := range task.ProfileTypes {
		wg.Add(1)
		go func(n common.ProfileType) {
			defer wg.Done()
			pc := GetProfileCollector(n)
			if pc == nil {
				return
			}
			ptConfig, ok := task.FindConfig(n)
			if !ok {
				p.logger.Error("[run] ptConfig not found. profileType is %s", n)
				return
			}
			p.logger.Info("[run] start to collect %s data", n)
			profileData, err := pc.Collect(ptConfig.DurationSeconds, ptConfig.IsSnapshot, task.UseCache, ptConfig.TargetDeltaSampleTypes)
			if err != nil || profileData == nil {
				p.logger.Error("[run] collect profile data fail. profileType=%+v, err=%+v", n, err)
				return
			}
			l.Lock()
			defer l.Unlock()
			profiles = append(profiles, profileData)
		}(pt)
	}
	wg.Wait()

	task.EndTimeMilliSec = time.Now().Unix()*1e3 + int64(time.Now().Nanosecond())/1e6 // record the timestamp when task finished

	p.send(task, profiles) // send must complete before close outChan
}

func (p *Profiler) send(task *common.Task, batchProfileData []*common.ProfileData) {
	if len(batchProfileData) == 0 {
		p.logger.Info("send profileInfo.UploadInfo abort! empty data")
		return
	}
	info, _ := register_utils.GetInfo()
	duration := (task.EndTimeMilliSec - task.StartTimeMilliSec) / 1000
	ptConfigList, _ := json.Marshal(task.PtConfigs)
	profileInfo := profile_models.ProfileInfo{
		UploadInfo: &profile_models.UploadInfo{
			ProfileId:            task.ProfileID,
			UploadId:             task.UploadID,
			Hostname:             agentless_adapter.GetHostname(),
			ContainerId:          info.ContainerId,
			InstanceId:           register_utils.GetInstanceID(),
			RecordName:           "pprof",
			RuntimeType:          internal.RuntimeTypeGo,
			Format:               "pprof",
			StartTime:            task.StartTimeMilliSec,
			EndTime:              task.EndTimeMilliSec,
			ServiceName:          p.service,
			PtConfigList:         string(ptConfigList),
			GoRuntimeInfo:        p_runtime.GetRuntimeInfo(),
			CpuLimit:             int64(res_monitor.GetCPULimit()),
			ProcessCpuUsageRatio: p.resMonitor.GetPastCPURatio(duration),
			ProcessMemRssRatio:   p.resMonitor.GetPastMemRatio(duration),
		},
	}
	for _, profileData := range batchProfileData {
		if b, err := profileData.Marshal(); err == nil && len(b) != 0 {
			profileInfo.MultiData = append(profileInfo.MultiData, b)
		}
	}
	p.logger.Info("send profileInfo.UploadInfo=%+v, profileDataLen=%d", profileInfo.UploadInfo, len(profileInfo.MultiData))
	select {
	case p.outChan <- &profileInfo: //non-blocking
	default:
	}
}
