package aiprofiler

import (
	"bytes"
	"errors"
	"runtime/pprof"
	"time"

	"github.com/google/pprof/profile"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aiprofiler/common"
)

func GetProfileCollector(pt common.ProfileType) ProfileCollector {
	if pc, ok := collectorRegister[pt]; ok {
		return pc
	}
	return nil
}

var collectorRegister = map[common.ProfileType]ProfileCollector{
	common.ProfileTypeCPU:       &CPUProfileCollector{},
	common.ProfileTypeHeap:      &HeapProfileCollector{},
	common.ProfileTypeBlock:     &BlockProfileCollector{},
	common.ProfileTypeMutex:     &MutexProfileCollector{},
	common.ProfileTypeGoroutine: &GoroutineProfileCollector{},
}

var ErrorUnknownProfile = errors.New("unknown profile")

type ProfileCollector interface {
	Name() string
	FileName() string
	SupportDelta() bool //see go/src/net/http/pprof.go -> profileSupportsDelta
	GetPrevious() *profile.Profile
	SetPrevious(*profile.Profile)
	Collect(durationSeconds int64, isSnapshot, useCache bool, targetDeltaSampleTypes []common.SampleType) (*common.ProfileData, error)
}

type CPUProfileCollector struct{}

func (p *CPUProfileCollector) Name() string {
	return common.ProfileTypeCPU.ToString()
}

func (p *CPUProfileCollector) FileName() string {
	return common.ProfileTypeCPU.ToString() + ".pprof"
}

func (p *CPUProfileCollector) SupportDelta() bool {
	return false
}

func (p *CPUProfileCollector) GetPrevious() *profile.Profile {
	return nil
}

func (p *CPUProfileCollector) SetPrevious(*profile.Profile) {
}

func (p *CPUProfileCollector) Collect(durationSeconds int64, _, _ bool, _ []common.SampleType) (*common.ProfileData, error) { // CPU profile does not support snapshot or delta
	buf := bytes.NewBuffer(nil)
	if err := pprof.StartCPUProfile(buf); err != nil {
		return nil, err
	}
	sleep(time.Duration(durationSeconds) * time.Second)
	pprof.StopCPUProfile()
	return &common.ProfileData{
		Data:         buf.Bytes(),
		SampleMethod: "period",
		ProfileType:  p.Name(),
	}, nil
}

type HeapProfileCollector struct {
	previous *profile.Profile
}

func (p *HeapProfileCollector) Name() string {
	return common.ProfileTypeHeap.ToString()
}

func (p *HeapProfileCollector) FileName() string {
	return common.ProfileTypeHeap.ToString() + ".pprof"
}

func (p *HeapProfileCollector) SupportDelta() bool {
	return true
}

func (p *HeapProfileCollector) GetPrevious() *profile.Profile {
	return p.previous
}

func (p *HeapProfileCollector) SetPrevious(pre *profile.Profile) {
	p.previous = pre
}

func (p *HeapProfileCollector) Collect(durationSeconds int64, isSnapshot, useCache bool, targetDeltaSampleTypes []common.SampleType) (*common.ProfileData, error) {
	if durationSeconds > 0 && !isSnapshot { // delta sample
		return deltaProfile(p, durationSeconds, useCache, targetDeltaSampleTypes)
	}
	// snapshot
	return collectProfile(p.Name())
}

type BlockProfileCollector struct {
	previous *profile.Profile
}

func (p *BlockProfileCollector) Name() string {
	return common.ProfileTypeBlock.ToString()
}

func (p *BlockProfileCollector) FileName() string {
	return common.ProfileTypeBlock.ToString() + ".pprof"
}

func (p *BlockProfileCollector) SupportDelta() bool {
	return true
}

func (p *BlockProfileCollector) GetPrevious() *profile.Profile {
	return p.previous
}

func (p *BlockProfileCollector) SetPrevious(pre *profile.Profile) {
	p.previous = pre
}

func (p *BlockProfileCollector) Collect(durationSeconds int64, isSnapshot, useCache bool, targetDeltaSampleTypes []common.SampleType) (*common.ProfileData, error) {
	if durationSeconds > 0 && !isSnapshot { // delta sample
		return deltaProfile(p, durationSeconds, useCache, targetDeltaSampleTypes)
	}
	// snapshot
	return collectProfile(p.Name())
}

type MutexProfileCollector struct {
	previous *profile.Profile
}

func (p *MutexProfileCollector) Name() string {
	return common.ProfileTypeMutex.ToString()
}

func (p *MutexProfileCollector) FileName() string {
	return common.ProfileTypeMutex.ToString() + ".pprof"
}

func (p *MutexProfileCollector) SupportDelta() bool {
	return true
}

func (p *MutexProfileCollector) GetPrevious() *profile.Profile {
	return p.previous
}

func (p *MutexProfileCollector) SetPrevious(pre *profile.Profile) {
	p.previous = pre
}

func (p *MutexProfileCollector) Collect(durationSeconds int64, isSnapshot, useCache bool, targetDeltaSampleTypes []common.SampleType) (*common.ProfileData, error) {
	if durationSeconds > 0 && !isSnapshot { // delta sample
		return deltaProfile(p, durationSeconds, useCache, targetDeltaSampleTypes)
	}
	// snapshot
	return collectProfile(p.Name())
}

type GoroutineProfileCollector struct {
	previous *profile.Profile
}

func (p *GoroutineProfileCollector) Name() string {
	return common.ProfileTypeGoroutine.ToString()
}

func (p *GoroutineProfileCollector) FileName() string {
	return common.ProfileTypeGoroutine.ToString() + ".pprof"
}

func (p *GoroutineProfileCollector) SupportDelta() bool {
	return true
}

func (p *GoroutineProfileCollector) GetPrevious() *profile.Profile {
	return p.previous
}

func (p *GoroutineProfileCollector) SetPrevious(pre *profile.Profile) {
	p.previous = pre
}

func (p *GoroutineProfileCollector) Collect(durationSeconds int64, isSnapshot, useCache bool, targetDeltaSampleTypes []common.SampleType) (*common.ProfileData, error) {
	if durationSeconds > 0 && !isSnapshot { // delta sample
		return deltaProfile(p, durationSeconds, useCache, targetDeltaSampleTypes)
	}
	// snapshot
	return collectProfile(p.Name())
}

func sleep(d time.Duration) {
	select {
	case <-time.After(d):
	}
}

func collectProfile(name string) (*common.ProfileData, error) {
	p := pprof.Lookup(name)
	if p == nil {
		return nil, ErrorUnknownProfile
	}
	buf := bytes.NewBuffer(nil)
	if err := p.WriteTo(buf, 0); err != nil {
		return nil, err
	}
	return &common.ProfileData{
		Data:         buf.Bytes(),
		SampleMethod: "snapshot",
		ProfileType:  name,
	}, nil
}

func deltaProfile(pc ProfileCollector, durationSeconds int64, useCache bool, targetDeltaSampleTypes []common.SampleType) (*common.ProfileData, error) {
	var (
		pre, cur *profile.Profile
		err      error
	)
	if useCache && pc.GetPrevious() != nil { // cache
		pre = pc.GetPrevious()
		defer func() { //closure is necessary
			pc.SetPrevious(cur) //update pre
		}()
	} else { // nocache.  compute delta by  now+durationSeconds - now
		preRaw, err := collectProfile(pc.Name())
		if err != nil {
			return nil, err
		}
		pre, err = profile.ParseData(preRaw.Data)
		if err != nil {
			return nil, err
		}
		sleep(time.Duration(durationSeconds) * time.Second)
	}

	// get current data
	curRaw, err := collectProfile(pc.Name())
	if err != nil {
		return nil, err
	}
	cur, err = profile.ParseData(curRaw.Data)
	if err != nil {
		return nil, err
	}

	delta, sampleMethod, err := calculateDelta(pre, cur, targetDeltaSampleTypes)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(nil)
	if err := delta.Write(buf); err != nil {
		return nil, err
	}
	return &common.ProfileData{
		Data:         buf.Bytes(),
		ProfileType:  pc.Name(),
		SampleMethod: sampleMethod,
	}, nil
}

func calculateDelta(pre, cur *profile.Profile, targetDeltaSampleTypes []common.SampleType) (*profile.Profile, string, error) {
	if pre == nil || cur == nil {
		return nil, "", nil
	}

	sampleMethod := "delta"
	ts := cur.TimeNanos
	dur := cur.TimeNanos - pre.TimeNanos

	/*
		profile.Merge basically works like this:
		   res = cur + pre * ratio. set ratio to 0 will just keep cur and -1 means delta
	*/

	// only calculate delta for selected sampleTypes.
	// For example, we only calculate delta for alloc_bytes/alloc_objects when profiling heap
	count := 0
	ratios := make([]float64, len(pre.SampleType))
	for idx, st := range pre.SampleType {
		if len(targetDeltaSampleTypes) == 0 { // if targetDeltaSampleTypes is nil, calculate delta for every sampleType
			ratios[idx] = -1
		}
		for _, targetSt := range targetDeltaSampleTypes {
			if targetSt.Type == st.Type && targetSt.Unit == st.Unit {
				ratios[idx] = -1
				count++
			}
		}
	}
	if count > 0 && count < len(pre.SampleType) {
		sampleMethod = "mixed"
	}

	_ = pre.ScaleN(ratios)

	delta, err := profile.Merge([]*profile.Profile{pre, cur})
	if err != nil {
		return nil, sampleMethod, err
	}

	delta.TimeNanos = ts // set since we don't know what profile.Merge set for TimeNanos.
	delta.DurationNanos = dur

	return delta, sampleMethod, nil
}
