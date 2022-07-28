package drivers

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/volcengine/apminsight-server-sdk-go/trace/aiprofiler/common"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aiprofiler/utils"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/logger"
	"github.com/volcengine/apminsight-server-sdk-go/trace/internal/settings_fetcher/settings_models"
)

const (
	TimedPeriodSec  = 10
	TimedPeriodTime = time.Second * TimedPeriodSec
)

type TimedDriver struct {
	profiles atomic.Value // []*settings_models.Profile

	handledSet map[int64]struct{} // if current time is smaller than target time, task will be ignored.
	// only single goroutine can access the map, not lock is needed

	taskChan chan *common.Task

	closeChan chan struct{}
	wg        sync.WaitGroup

	logger logger.Logger
}

func NewTimedDriver(taskChan chan *common.Task, l logger.Logger) Driver {
	return &TimedDriver{
		taskChan:   taskChan,
		handledSet: make(map[int64]struct{}, 0),
		closeChan:  make(chan struct{}),
		logger:     l,
	}
}

func (d *TimedDriver) Name() string {
	return RecordTypeTask
}

func (d *TimedDriver) Handle(ps []*settings_models.Profile) {
	if len(ps) == 0 {
		d.logger.Info("[TimedDriver.Handle] empty incoming! old profiles will be cleared")
	} else {
		d.logger.Info("[TimedDriver.Handle] incoming profiles len is %d.", len(ps))
	}
	tmp := make([]*settings_models.Profile, 0, len(ps))
	tmp = append(tmp, ps...)

	sort.SliceStable(tmp, func(i, j int) bool {
		return tmp[i].StartTime < tmp[j].StartTime //smaller start_time first
	})
	d.profiles.Store(tmp) //those not represent in the latest settings will be considered as disabled.
}

func (d *TimedDriver) Start() {
	tc := time.NewTicker(TimedPeriodTime)
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		for {
			select {
			case <-tc.C:
				now := time.Now().Unix()
				profiles, _ := d.profiles.Load().([]*settings_models.Profile)
				for _, p := range profiles {
					if p.StartTime > now { // not yet
						d.logger.Info("[TimedDriver.runloop] not yet. profile start_time %d > now %d. profile setting is %+v", p.StartTime, now, profiles)
						continue
					}
					if _, ok := d.handledSet[p.ProfileId]; ok { // already handled
						continue
					}
					if task := d.profileToTask(p); task != nil { // handle
						d.logger.Info("[TimedDriver.runloop] task is %+v", task)
						select {
						case d.taskChan <- task: //non-blocking
							d.handledSet[p.ProfileId] = struct{}{} // keep track of handled profile. can not run twice
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

func (d *TimedDriver) Stop() {
	close(d.closeChan)
	d.wg.Wait()
}

func (d *TimedDriver) profileToTask(p *settings_models.Profile) *common.Task {
	profileTypeList, ok := extractProfileTypes(p)
	if !ok {
		return nil
	}
	l := make([]common.ProfileTypeConfig, 0, len(p.PtConfigList))
	for _, cfg := range p.PtConfigList {
		if pt, ok := common.FromString(cfg.ProfileType); ok {
			ptc := common.ProfileTypeConfig{
				ProfileType:     pt,
				DurationSeconds: cfg.DurationSeconds,
				IsSnapshot:      cfg.IsSnapshot,
			}
			l = append(l, ptc)
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
