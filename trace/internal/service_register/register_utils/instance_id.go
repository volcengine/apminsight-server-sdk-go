package register_utils

import (
	"strings"
	"sync"

	"github.com/google/uuid"
)

var instanceId string // should be global

var once sync.Once

// GetInstanceID returns the unique id represent current process
func GetInstanceID() string {
	once.Do(func() {
		randUUID, _ := uuid.NewRandom()
		instanceId = strings.Replace(randUUID.String(), "-", "", -1)
	})
	return instanceId
}
