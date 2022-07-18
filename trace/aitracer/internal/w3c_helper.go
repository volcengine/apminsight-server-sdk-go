package internal

import (
	"encoding/hex"
	"errors"
	"strings"
)

const (
	minLength       = 55
	parentDelimiter = "-"

	stateDelimiter = ","
	stateEqual     = "="
	maxMembers     = 32
)

const (
	keyAppID  = "app_id"
	keyOrigin = "origin"
)

var supportedVersions = map[int]struct{}{0: {}}

type TraceParent struct {
	Version      int
	TraceID      string
	ParentSpanID string
	TraceFlags   int
}

type TraceState struct {
	Members []Member
}

type Member struct {
	Key   string
	Value string
}

var DefaultSimpleW3CFormatParser = SimpleW3CFormatParser{}

// SimpleW3CFormatParser is a simple w3c format parser which ignores many format constraint
type SimpleW3CFormatParser struct{}

// ParseTraceParent parse format like 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01-suffix
func (w SimpleW3CFormatParser) ParseTraceParent(traceParent string) (TraceParent, error) {
	traceParentRet := TraceParent{}
	if len(traceParent) < minLength {
		return traceParentRet, errors.New("invalid trace parent: traceParend len is less than 55")
	}
	fields := strings.Split(traceParent, parentDelimiter)
	if len(fields) < 4 {
		return traceParentRet, errors.New("invalid trace parent: number of - is less than 3")
	}

	{
		versionStr := fields[0]
		if len(versionStr) != 2 {
			return traceParentRet, errors.New("invalid trace parent: len of versionStr is not 2")
		}
		versionBytes, err := hex.DecodeString(versionStr)
		if err != nil || len(versionBytes) == 0 {
			return traceParentRet, err
		}
		version := int(versionBytes[0])
		if _, ok := supportedVersions[version]; !ok {
			return traceParentRet, errors.New("invalid trace parent: version not supported")
		}
		traceParentRet.Version = version
	}
	{
		traceID := fields[1]
		if len(traceID) != 32 {
			return traceParentRet, errors.New("invalid trace parent: len of traceID is not 32")
		}
		if traceID == "00000000000000000000000000000000" {
			return traceParentRet, errors.New("invalid trace parent: all zero traceID is not allowed")
		}
		traceParentRet.TraceID = traceID
	}
	{
		spanID := fields[2]
		if len(spanID) != 16 {
			return traceParentRet, errors.New("invalid trace parent: len of spanID is not 16")
		}
		if spanID == "0000000000000000" {
			return traceParentRet, errors.New("invalid trace parent: all zero spanID is not allowed")
		}
		traceParentRet.ParentSpanID = spanID
	}
	{
		traceFlagsStr := fields[3]
		if len(traceFlagsStr) != 2 {
			return traceParentRet, errors.New("invalid trace parent: len of traceFlagsStr is not 2")
		}
		flagsBytes, err := hex.DecodeString(traceFlagsStr)
		if err != nil || len(flagsBytes) == 0 {
			return traceParentRet, err
		}
		traceParentRet.TraceFlags = int(flagsBytes[0])
	}
	return traceParentRet, nil
}

// ParseTraceState ignores key value format constraint. All k-v pair will be accepted
// By specification traceState should be parsed through
func (w SimpleW3CFormatParser) ParseTraceState(traceState string) (TraceState, error) {
	traceStateRet := TraceState{}
	if len(traceState) == 0 {
		return traceStateRet, nil
	}
	memberList := make([]Member, 0)
	keySet := make(map[string]struct{})
	for _, memberStr := range strings.Split(traceState, stateDelimiter) {
		if len(memberStr) == 0 {
			continue
		}
		if keyValue := strings.Split(memberStr, stateEqual); len(keyValue) == 2 {
			member := Member{
				Key:   keyValue[0],
				Value: keyValue[1],
			}
			if _, ok := keySet[member.Key]; ok {
				return traceStateRet, errors.New("invalid trace state: duplicated key")
			}
			keySet[member.Key] = struct{}{}
			memberList = append(memberList, member)
		}
	}
	if len(memberList) > maxMembers {
		return traceStateRet, errors.New("invalid trace state: too many members")
	}
	traceStateRet.Members = memberList
	return traceStateRet, nil
}

func (t *TraceState) GetAppID() string {
	for _, m := range t.Members {
		if m.Key == keyAppID {
			return m.Value
		}
	}
	return ""
}

func (t *TraceState) GetOrigin() string {
	for _, m := range t.Members {
		if m.Key == keyOrigin {
			return m.Value
		}
	}
	return ""
}
