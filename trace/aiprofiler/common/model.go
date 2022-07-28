package common

import "github.com/volcengine/apminsight-server-sdk-go/trace/aiprofiler/profile_models"

// ProfileData is profile result. raw data is []byte, wrapped in a struct for expansibility
type ProfileData = profile_models.DataWrapper

type SampleType struct {
	Type string `json:"type"`
	Unit string `json:"unit"`
}

type ProfileTypeConfig struct {
	ProfileType            ProfileType  `json:"profile_type"`
	DurationSeconds        int64        `json:"duration_seconds"`
	IsSnapshot             bool         `json:"is_snapshot"`
	TargetDeltaSampleTypes []SampleType `json:"target_delta_sample_types"` // calculate delta for these sampleTypes
}
