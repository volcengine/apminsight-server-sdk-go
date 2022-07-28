package drivers

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/volcengine/apminsight-server-sdk-go/trace/aiprofiler/common"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aiprofiler/utils"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/logger"
	"github.com/volcengine/apminsight-server-sdk-go/trace/internal/settings_fetcher/settings_models"
)

type ContinuousDriver struct {
	profiles atomic.Value //map[int64]*settings_models.Profile
	taskChan chan *common.Task

	closeChan chan struct{}
	wg        sync.WaitGroup

	logger logger.Logger
}

func NewContinuousDriver(taskChan chan *common.Task, l logger.Logger) Driver {
	return &ContinuousDriver{
		taskChan:  taskChan,
		closeChan: make(chan struct{}),
		logger:    l,
	}
}

func (d *ContinuousDriver) Name() string {
	return RecordTypeContinuous
}

func (d *ContinuousDriver) Handle(ps []*settings_models.Profile) {
	if len(ps) == 0 {
		d.logger.Info("[ContinuousDriver.Handle] empty incoming! old profiles will be cleared")
	} else {
		d.logger.Info("[ContinuousDriver.Handle] incoming profiles len is %d.", len(ps))
	}

	tmp := make(map[int64]*settings_models.Profile)
	for _, p := range ps {
		tmp[p.ProfileId] = p
	}
	d.profiles.Store(tmp) // those not represent in the latest settings will be considered as disabled.
}

func (d *ContinuousDriver) Start() {
	tc := time.NewTicker(ContinuousPeriodTime)
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		for {
			select {
			case <-tc.C:
				profiles, _ := d.profiles.Load().(map[int64]*settings_models.Profile)
				for _, p := range profiles {
					if task := d.profileToTask(p); task != nil {
						d.logger.Info("[ContinuousDriver.runloop] task is %+v", task)
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

func (d *ContinuousDriver) Stop() {
	close(d.closeChan)
	d.wg.Wait()
}

func (d *ContinuousDriver) profileToTask(p *settings_models.Profile) *common.Task {
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
		UseCache:     true, // for continuous, cache can be used to calculate delta
	}
}
