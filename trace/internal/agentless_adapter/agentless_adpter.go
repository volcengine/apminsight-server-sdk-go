package agentless_adapter

import (
	"os"
	"sync"
)

const AppKey = "X-ByteAPM-AppKey"

var (
	appKey        string
	hostname      string
	runtimeBearer string

	onceAppKey        sync.Once
	onceHostname      sync.Once
	onceRuntimeBearer sync.Once
)

func GetAppKey() string {
	onceAppKey.Do(func() {
		appKey = os.Getenv("APMPLUS_APP_KEY")
	})
	return appKey
}

func GetHostname() string {
	onceHostname.Do(func() {
		hostname = os.Getenv("MY_NODE_NAME") // we do not know hostname when process is running inside container. set it via kubernetes env
		if hostname == "" {
			if h, err := os.Hostname(); err == nil {
				hostname = h
			}
		}
	})
	return hostname
}

func GetRuntimeBearer() string {
	onceRuntimeBearer.Do(func() {
		runtimeBearer = os.Getenv("MY_RUNTIME_BEARER") // set it via kubernetes env
	})
	return runtimeBearer
}
