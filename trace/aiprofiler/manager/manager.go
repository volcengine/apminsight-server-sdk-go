package manager

import (
	"time"

	"github.com/volcengine/apminsight-server-sdk-go/trace/aiprofiler/common"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aiprofiler/drivers"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aiprofiler/res_monitor"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/logger"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/service_register/register_utils"
	"github.com/volcengine/apminsight-server-sdk-go/trace/internal/settings_fetcher"
	"github.com/volcengine/apminsight-server-sdk-go/trace/internal/settings_fetcher/settings_models"
)

// Manager fetch settings and dispatch profile tasks to drivers
type Manager struct {
	service         string
	instanceID      string
	settingsFetcher *settings_fetcher.Fetcher

	// drivers keep track of task status
	continuousDriver  drivers.Driver
	timedDriver       drivers.Driver
	conditionalDriver drivers.Driver

	logger logger.Logger
}

type SettingAddrConfig struct {
	Sock string

	Schema  string
	Host    string
	Timeout time.Duration

	Logger logger.Logger
}

func NewManager(service string, settingAddrCfg SettingAddrConfig, taskChan chan *common.Task, monitor *res_monitor.Monitor, l logger.Logger) *Manager {
	if l == nil {
		l = &logger.NoopLogger{}
	}

	m := &Manager{
		service:           service,
		instanceID:        register_utils.GetInstanceID(),
		continuousDriver:  drivers.NewContinuousDriver(taskChan, l),
		timedDriver:       drivers.NewTimedDriver(taskChan, l),
		conditionalDriver: drivers.NewConditionalDriver(taskChan, monitor, l),
		logger:            l,
	}

	settingCfg := settings_fetcher.SettingsFetcherConfig{
		Service:  service,
		Logger:   l,
		Sock:     settingAddrCfg.Sock,
		Schema:   settingAddrCfg.Schema,
		Host:     settingAddrCfg.Host,
		Timeout:  settingAddrCfg.Timeout,
		Notifier: []func(*settings_models.Settings){m.HandlerProfileTasks},
	}
	m.settingsFetcher = settings_fetcher.NewSettingsFetcher(settingCfg)

	return m
}

func (m *Manager) HandlerProfileTasks(s *settings_models.Settings) {
	var (
		continuousProfiles  []*settings_models.Profile
		timedProfiles       []*settings_models.Profile
		conditionalProfiles []*settings_models.Profile
	)
	defer func() {
		// Handle will always be invoked to make sure that old profiles get cleared
		m.continuousDriver.Handle(continuousProfiles)
		m.timedDriver.Handle(timedProfiles)
		m.conditionalDriver.Handle(conditionalProfiles)
	}()

	// s is settings of this service
	if s == nil || s.ProfileSettings == nil {
		m.logger.Info("[HandlerProfileTasks] get empty Settings/ProfileSettings. all old profiles will be cleared") //can not return here, or old profiles will not be cleared
		return
	}

	// instanceID filter
	instanceIDMatched := func(p *settings_models.Profile) bool {
		if len(p.InstanceIdList) == 0 {
			return true
		}
		for _, instanceID := range p.InstanceIdList {
			if instanceID == m.instanceID {
				return true
			}
		}
		return false
	}

	m.logger.Info("[HandlerProfileTasks] get profile settings success. ProfileSettings is %+v", s.ProfileSettings)

	// dispatch
	for _, p := range s.ProfileSettings.Profile {
		if !instanceIDMatched(p) {
			m.logger.Debug("[HandlerProfileTasks] instanceID not match. profile is %+v", p)
			continue
		}
		switch p.RecordType {
		case drivers.RecordTypeContinuous:
			continuousProfiles = append(continuousProfiles, p)
		case drivers.RecordTypeTask:
			timedProfiles = append(timedProfiles, p)
		case drivers.RecordTypeConditional:
			conditionalProfiles = append(conditionalProfiles, p)
		default:
			m.logger.Error("[HandlerProfileTasks] invalid RecordType %s", p.RecordType)
		}
	}
}

func (m *Manager) Start() {
	m.continuousDriver.Start()
	m.timedDriver.Start()
	m.conditionalDriver.Start()
	m.settingsFetcher.Start()
}

func (m *Manager) Stop() {
	m.settingsFetcher.Stop()
	m.continuousDriver.Stop()
	m.timedDriver.Stop()
	m.conditionalDriver.Stop()
}
