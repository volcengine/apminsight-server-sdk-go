package drivers

import (
	"time"

	"github.com/volcengine/apminsight-server-sdk-go/trace/aiprofiler/common"
	"github.com/volcengine/apminsight-server-sdk-go/trace/internal/settings_fetcher/settings_models"
)

const (
	RecordTypeContinuous  = "continuous"
	RecordTypeTask        = "task" // alias 'timed', 'task' for forward compatibility
	RecordTypeConditional = "conditional"
)

const (
	Seconds60                = 60
	ContinuousPeriodTime     = time.Second * Seconds60 // continuous period
	ConditionalCheckInterval = time.Second * Seconds60 // time interval the conditions are checked
)

// conditional and continuous profiles use preset settings, user can only decide profileType
var presetProfileTypeConfigs = map[common.ProfileType]common.ProfileTypeConfig{
	common.ProfileTypeCPU: { // period
		ProfileType:     common.ProfileTypeCPU,
		DurationSeconds: Seconds60,
	},
	common.ProfileTypeHeap: { // delta
		ProfileType:            common.ProfileTypeHeap,
		DurationSeconds:        Seconds60,
		TargetDeltaSampleTypes: []common.SampleType{{Type: "alloc_objects", Unit: "count"}, {Type: "alloc_space", Unit: "bytes"}},
	},
	common.ProfileTypeBlock: { // delta
		ProfileType:     common.ProfileTypeBlock,
		DurationSeconds: Seconds60,
	},
	common.ProfileTypeMutex: { // delta
		ProfileType:     common.ProfileTypeMutex,
		DurationSeconds: Seconds60,
	},
	common.ProfileTypeGoroutine: { // snapshot
		ProfileType: common.ProfileTypeGoroutine,
		IsSnapshot:  true,
	},
}

type Driver interface {
	Name() string
	Handle([]*settings_models.Profile)
	Start()
	Stop()
}
