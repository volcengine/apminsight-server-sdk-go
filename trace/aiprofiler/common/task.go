package common

type Task struct {
	Name              string
	ProfileID         int64
	UploadID          string
	ProfileTypes      []ProfileType
	PtConfigs         []ProfileTypeConfig
	StartTimeMilliSec int64
	EndTimeMilliSec   int64
	UseCache          bool // use cache to compute delta. only continuous profile support this option
}

func (t *Task) FindConfig(pt ProfileType) (ProfileTypeConfig, bool) {
	for _, ptCfg := range t.PtConfigs {
		if ptCfg.ProfileType == pt {
			return ptCfg, true
		}
	}
	return ProfileTypeConfig{}, false
}
