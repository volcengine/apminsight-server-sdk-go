package common

type ProfileType string

const (
	ProfileTypeCPU       ProfileType = "cpu" //type are identical to defines in runtime/pprof/profiles
	ProfileTypeHeap      ProfileType = "heap"
	ProfileTypeBlock     ProfileType = "block"
	ProfileTypeMutex     ProfileType = "mutex"
	ProfileTypeGoroutine ProfileType = "goroutine"
)

var validProfileTypes = map[ProfileType]struct{}{
	ProfileTypeCPU: {}, ProfileTypeHeap: {}, ProfileTypeBlock: {}, ProfileTypeMutex: {}, ProfileTypeGoroutine: {},
}

func (pt ProfileType) ToString() string {
	return string(pt)
}

func FromString(s string) (ProfileType, bool) {
	pt := ProfileType(s)
	if _, ok := validProfileTypes[pt]; ok {
		return pt, true
	}
	return "", false
}
