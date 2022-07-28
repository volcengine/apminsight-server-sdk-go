package drivers

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/volcengine/apminsight-server-sdk-go/trace/aiprofiler/common"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aiprofiler/res_monitor"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aiprofiler/utils"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/logger"
	"github.com/volcengine/apminsight-server-sdk-go/trace/internal/settings_fetcher/settings_models"
)

const (
	ConditionKeyCPURatio     = "cpu_ratio"
	ConditionKeyMemRatio     = "mem_ratio"
	ConditionKeyGoroutineNum = "goroutine_num"

	ConditionCompareTypeThreshold = "threshold"
	ConditionCompareTypeWindow    = "windows"

	ConditionOpGt = ">" //only support gt

	executionLimitPerHour = 3 //as most 3 executions per hour for one profile
	limitExpireTime       = time.Hour
)

type count struct {
	cnt      int32
	expireAt time.Time
}

type ConditionalDriver struct {
	profiles atomic.Value     // map[int64]*settings_models.Profile
	cntLimit map[int64]*count // limit. avoid too many pprof when conditions are met
	taskChan chan *common.Task

	resMonitor *res_monitor.Monitor

	closeChan chan struct{}
	wg        sync.WaitGroup

	logger logger.Logger
}

func NewConditionalDriver(taskChan chan *common.Task, monitor *res_monitor.Monitor, l logger.Logger) Driver {
	if monitor == nil {
		monitor = res_monitor.NewMonitor()
		monitor.Start()
	}
	return &ConditionalDriver{
		cntLimit:   map[int64]*count{},
		taskChan:   taskChan,
		resMonitor: monitor,
		closeChan:  make(chan struct{}),
		logger:     l,
	}
}

func (d *ConditionalDriver) Name() string {
	return RecordTypeConditional
}

func (d *ConditionalDriver) Handle(ps []*settings_models.Profile) {
	if len(ps) == 0 {
		d.logger.Info("[ConditionalDriver.Handle] empty incoming! old profiles will be cleared")
	} else {
		d.logger.Info("[ConditionalDriver.Handle] incoming profiles len is %d.", len(ps))
	}
	tmp := make(map[int64]*settings_models.Profile)
	for _, p := range ps {
		tmp[p.ProfileId] = p
	}
	d.profiles.Store(tmp) //those not represent in the latest settings will be considered as disabled.
}

func (d *ConditionalDriver) Start() {
	tc := time.NewTicker(ConditionalCheckInterval)
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		for {
			select {
			case <-tc.C:
				// remove expired
				d.removeExpired()
				profiles, _ := d.profiles.Load().(map[int64]*settings_models.Profile)
				for _, p := range profiles {
					// check condition
					if !d.checkCondition(p) { // not meet
						d.logger.Info("[ConditionalDriver.runloop] condition not meet. profileId=%d", p.ProfileId)
						continue
					}
					// check execution limit
					if !d.checkExecutionLimit(p) { // exceed limit
						d.logger.Info("[ConditionalDriver.runloop] task has exceed limit. profileId=%d", p.ProfileId)
						continue
					}
					if task := d.profileToTask(p); task != nil {
						d.logger.Info("[ConditionalDriver.runloop] task is %+v", task)
						select {
						case d.taskChan <- task: //non-blocking.
						default:
						}
					}
				}
			case <-d.closeChan:
				return
			}
		}
	}()
}

func (d *ConditionalDriver) removeExpired() {
	for k, limit := range d.cntLimit {
		if limit.expireAt.Before(time.Now()) {
			delete(d.cntLimit, k)
		}
	}
}

func (d *ConditionalDriver) checkExecutionLimit(p *settings_models.Profile) bool {
	if limit, ok := d.cntLimit[p.ProfileId]; ok {
		if limit.cnt >= executionLimitPerHour {
			return false
		}
		limit.cnt++
		return true
	}
	//not executed before or has expired
	d.cntLimit[p.ProfileId] = &count{
		cnt:      1,
		expireAt: time.Now().Add(limitExpireTime),
	}
	return true
}

func (d *ConditionalDriver) Stop() {
	close(d.closeChan)
	d.wg.Wait()
}

func (d *ConditionalDriver) checkCondition(p *settings_models.Profile) bool {
	result := false
	for _, cond := range p.TriggerConditionList {
		switch cond.Key {
		case ConditionKeyCPURatio:
			result = d.meetCondition(cond, d.resMonitor.GetCPURatio(), d.resMonitor.GetCPURatioDelta()) // value in cond is something like 1%
			d.logger.Info("[ConditionalDriver.checkCondition] cond met? is %s. cond=%+v, cpuRatio=%+v, cpuDelta=%+v", result, cond, d.resMonitor.GetCPURatio(), d.resMonitor.GetCPURatioDelta())
		case ConditionKeyMemRatio:
			result = d.meetCondition(cond, d.resMonitor.GetMemRatio(), d.resMonitor.GetMemRatioDelta())
			d.logger.Info("[ConditionalDriver.checkCondition] cond met? is %s. cond=%+v, memRatio=%+v, memDelta=%+v", result, cond, d.resMonitor.GetMemRatio(), d.resMonitor.GetMemRatioDelta())
		case ConditionKeyGoroutineNum:
			result = d.meetCondition(cond, d.resMonitor.GetGoRoutineNum(), d.resMonitor.GetGoRoutineNumDelta())
			d.logger.Info("[ConditionalDriver.checkCondition] cond met? is %s. cond=%+v, goRoutine=%+v, goRoutineDelta=%+v", result, cond, d.resMonitor.GetGoRoutineNum(), d.resMonitor.GetGoRoutineNumDelta())
		}
		if result {
			break // if any cond is met then we profile
		}
	}
	return result
}

func (d *ConditionalDriver) meetCondition(cond *settings_models.TriggerCondition, curValue, deltaValue float64) bool {
	if cond.Compare == ConditionCompareTypeThreshold {
		switch cond.Op {
		case ConditionOpGt:
			return curValue > toDecimal(cond.Value, cond.Unit)
		}
	} else if cond.Compare == ConditionCompareTypeWindow {
		switch cond.Op {
		case ConditionOpGt:
			return deltaValue > toDecimal(cond.Value, cond.Unit)
		}
	} else {
		return false
	}
	return false
}

func (d *ConditionalDriver) profileToTask(p *settings_models.Profile) *common.Task {
	profileTypeList, ok := extractProfileTypes(p)
	if !ok {
		return nil
	}
	l := make([]common.ProfileTypeConfig, 0, len(profileTypeList))
	for _, pt := range profileTypeList {
		if ptCfg, found := presetProfileTypeConfigs[pt]; found {
			l = append(l, ptCfg)
		}
	}
	return &common.Task{
		Name:         "",
		ProfileID:    p.ProfileId,
		UploadID:     utils.NewRandID(),
		ProfileTypes: profileTypeList,
		PtConfigs:    l,
	}
}

func extractProfileTypes(p *settings_models.Profile) ([]common.ProfileType, bool) {
	profileTypeList := make([]common.ProfileType, 0, len(p.ProfileTypeList))
	for _, ptStr := range p.ProfileTypeList {
		if pt, ok := common.FromString(ptStr); ok {
			profileTypeList = append(profileTypeList, pt)
		}
	}
	if len(profileTypeList) == 0 {
		return nil, false
	}
	return profileTypeList, true
}

func toDecimal(f float64, unit string) float64 {
	if unit == "%" {
		return f / 100
	}
	return f
}
