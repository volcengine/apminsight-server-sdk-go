package utils

import (
	"strings"

	"github.com/google/uuid"
)

func NewRandID() string {
	randUUID, _ := uuid.NewRandom()
	return strings.Replace(randUUID.String(), "-", "", -1)
}
