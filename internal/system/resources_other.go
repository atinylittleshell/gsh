//go:build !linux && !darwin && !windows

package system

import "time"

func getResources() *Resources {
	return &Resources{
		Timestamp: time.Now(),
	}
}
